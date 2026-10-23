'use strict';
// Shared deterministic route replay. Numeric output is diagnostic when blockers remain.
const fs=require('node:fs'),path=require('node:path'),zlib=require('node:zlib'),crypto=require('node:crypto');
const {resolveProfile}=require('./class_profiles.cjs');
const arg=name=>{const i=process.argv.indexOf(name);if(i<0)return undefined;if(!process.argv[i+1]||process.argv[i+1].startsWith('--'))throw Error(name+' needs a value');return process.argv[i+1];};
const selectedProfile=resolveProfile({class:arg('--class'),race:arg('--race'),faction:arg('--faction'),targetLevel:Number(arg('--target-level')||30)});
const profileSelection=process.argv.includes('--class')||process.argv.includes('--race')||process.argv.includes('--faction');
const {planProfileSources}=require('./class_sources.cjs');
const root=path.resolve(__dirname,'..'),external=path.resolve(process.env.POLESTAR_ROOT || path.join(root,'..','polestar'));
const req=p=>require(path.join(external,'tools/lib',p));
const {getWorld,ISLANDS}=req('sim/world'),{paramsFor}=req('sim/index');
const {compileRun,makeTables,NEAR_K}=req('sim/run'),{Evaluator}=req('sim/evaluate');
const {addMovementPoints}=require('./movement_replay.cjs');
const {prepareObjectiveSites,compileEvent}=require('./objective_sequence.cjs');
const {stepEvent}=require('./engine_step.cjs');
const {compileSequence}=req('sim/sequence'),{K_GRIND,K_TASK,K_ACCEPT}=req('sim/run');
const {loadPolestarRoutes,requiredEvents}=req('sim/baselines/polestar');
const {reconcile,loadInputs}=require('./objective_count_overlay.cjs');
const {awardXpPolicy,estimateAction,estimateNonQuest}=require('./estimate_model.cjs');
const {actionFailure,acceptance}=require('./replay_validation.cjs');
const {continuation}=require('./style_extension.cjs');
const catalogAdapter=require('./catalog_replay.cjs');
const timeLedger=require('./time_ledger.cjs');
const routeCombat=require('./combat/route_work.cjs');
const combatRequests=require('./combat/requirements.cjs');
let routeCombatConfig=null;
const {LEVEL_XP}=req('game/xp');
const {recoverAccepts}=require('./quest_recovery.cjs');
const {reconcileWorldGaps}=require('./world_gap_overlay.cjs');
const targetArg=process.argv.indexOf('--target-level');
const targetLevel=targetArg<0?30:Number(process.argv[targetArg+1]);
if(!Number.isInteger(targetLevel)||targetLevel<2||targetLevel>30)throw Error('--target-level must be an integer from 2 to 30');
const dataDir=path.join(root,'.comparison/data'),simDir=path.join(root,'.comparison/cache'),outDir=process.argv.includes('--output-dir')?path.resolve(process.argv[process.argv.indexOf('--output-dir')+1]):path.join(root,'.comparison/results');
const read=p=>{const bytes=fs.readFileSync(p);return JSON.parse(p.endsWith('.gz')?zlib.gunzipSync(bytes):bytes.toString('utf8'));};
const save=(n,d)=>fs.writeFileSync(path.join(outDir,n),JSON.stringify(d,null,2)+'\n');
// Keep every original route immutable. Apply the common, explicitly sourced XP
// overlays only in memory, in the same order for all four route replays.
const rawQuestFacts=read(path.join(dataDir,'quests.json'));
const assumedXpFile=path.resolve(root,'../../sources/quest-xp-provisional-zero-fallback.json');
const verifiedXpFile=path.resolve(root,'../../sources/warcraftdb-verified-xp-20261023.json');
function effectiveQuestXp(records, assumed, verified) {
 const out={...records},applied={verified:[],assumedZero:[]};
 const override=(id,xp,label)=>{
  const existing=out[id];
  // Never replace a known Forever reward with a weaker Classic reference.
  if(!existing || Number.isFinite(existing.xp) || !Number.isFinite(xp))return;
  out[id]={...existing,xp,xp_status:label};
  applied[label==='assumed_zero_missing_source'?'assumedZero':'verified'].push(Number(id));
 };
 for(const [id,record] of Object.entries((verified && verified.quests)||{}))
  override(id,record.fields && record.fields.xp,'verified_reference');
 for(const entry of (assumed && assumed.entries)||[])
  if(entry.xp===0)override(entry.quest_id,0,'assumed_zero_missing_source');
 return {records:out,applied};
}
const replayXpReferences=read(path.join(root,'data/replay-xp-references.json'));
const previousVerified=fs.existsSync(verifiedXpFile)?read(verifiedXpFile):{quests:{}};
const combinedVerified={quests:{...replayXpReferences.quests,...previousVerified.quests}};
const xpOverlay=effectiveQuestXp(rawQuestFacts,
 fs.existsSync(assumedXpFile)?read(assumedXpFile):null,
 combinedVerified);
const raw=xpOverlay.records;
let missingCounts=new Set(read(path.join(dataDir,'missing-counts.json'))),objectiveReport=null,worldGapReport=null;
const flags=new Set([selectedProfile.class.toUpperCase(),selectedProfile.race.toUpperCase(),selectedProfile.faction.toUpperCase(),'FOREVER','ERA','SOFTCORE',...selectedProfile.guideFlags]);
if(selectedProfile.race==='Undead')flags.add('SCOURGE');
const knownFlags=new Set(['WARRIOR','PALADIN','HUNTER','ROGUE','PRIEST','SHAMAN','MAGE','WARLOCK','DRUID','ORC','TROLL','TAUREN','SCOURGE','UNDEAD','HUMAN','DWARF','GNOME','NIGHTELF','SKYBORNE','BLOODELF','DRAENEI','HORDE','ALLIANCE','FOREVER','ERA','SOM1','SOM2','SOD','SOD2','TBC','WOTLK','JOYOUS','GROUPS','AH','HARDCORE','SOFTCORE','SKIP','H12B']);
const and=(a,b)=>a===false||b===false?false:a===null||b===null?null:!!a&&!!b;
const or=(a,b)=>a===true||b===true?true:a===null||b===null?null:!!a||!!b;
function qstate(ev,run,id){const i=run.qIndex.get(+id);if(i==null)return run.syntheticQuestStates?.get(+id)||0;if(ev.done[i])return 3;if(!ev.active[i])return 0;for(let t=run.qTask0[i];t<run.qTask1[i];t++)if(!ev.taskDone[t])return 1;return 2;}
function rxpCondition(s){
 if(!s)return true;
 let answer=false;
 for(const clause of s.trim().split('/')){let value=true;for(const t0 of clause.trim().split(/\s+/)){const neg=t0.startsWith('!'),t=t0.replace(/^!/, '').toUpperCase();const v=knownFlags.has(t)?flags.has(t):null;value=and(value,v===null?null:neg?!v:v);}answer=or(answer,value);}
 return answer;
}
function journeyCondition(s,ev,run){
 if(!s)return true;
 const tokens=s.match(/>=|<=|==|~=|>|<|\(|\)|\d+(?:\.\d+)?|[A-Za-z_][A-Za-z_0-9.]*(?:\[\d+\]){0,2}/g) || [];
 if(tokens.join('').toLowerCase()!==s.replace(/\s/g,'').toLowerCase())return null;
 let i=0;
 function val(){const t=tokens[i++];if(t==='('){const v=expr();if(tokens[i++]!==')')throw Error('unbalanced');return v;}if(t==null)throw Error('missing term');if(/^\d/.test(t))return +t;const u=t.toUpperCase();if(u==='LEVEL')return ev.level;if(u==='NONE')return 0;if(u==='ACCEPTED')return 1;if(u==='COMPLETED')return 2;if(u==='TURNEDIN')return 3;if(u==='VAR.HORDETB20')return false;const q=/^QUEST\[(\d+)\]$/.exec(u);if(q)return qstate(ev,run,q[1]);if(knownFlags.has(u))return flags.has(u);return null;}
 function comparison(){let a=val();const op=tokens[i];if(!['>=','<=','==','~=','>','<'].includes(op))return a;i++;const b=val();if(a===null||b===null)return null;return op==='>='?a>=b:op==='<='?a<=b:op==='=='?a===b:op==='~='?a!==b:op==='>'?a>b:a<b;}
 function not(){if((tokens[i] || '').toLowerCase()==='not'){i++;const v=not();return v===null?null:!v;}return comparison();}
 function all(){let a=not();while((tokens[i] || '').toLowerCase()==='and'){i++;a=and(a,not());}return a;}
 function expr(){let a=all();while((tokens[i] || '').toLowerCase()==='or'){i++;a=or(a,all());}return a;}
 try{const r=expr();return i===tokens.length?(r===null?null:!!r):null;}catch{return null;}
}
function condition(g,ev,run){
 if(g.conditions)return catalogAdapter.allowed(g,run,ev,qstate,selectedProfile);
 if(g.condition_format==='rxp')return rxpCondition(g.condition);
 if(g.condition_format==='journey')return journeyCondition(g.condition,ev,run);
 if(g.condition_format==='wisp'){
  const c=g.condition || {};
  // Level labels are suggestions in this export; actual quest-level gates are enforced by the shared engine.
  if(c.quests && !c.quests.every(q=>qstate(ev,run,q)>=3))return false;
  if(c.anyQuests && !c.anyQuests.some(q=>qstate(ev,run,q.questId)>=(q.state==='turned-in'?3:1)))return false;
 }
 return true;
}
function compatible(id){const q=raw[id];if(!q)return true;const hasBit=(mask,bit)=>Math.floor(Number(mask)/bit)%2===1;return (!q.classes||hasBit(q.classes,selectedProfile.classBit))&&(!q.races||hasBit(q.races,selectedProfile.raceBit));}
function applyXpPolicy(world,records){
 for(const q of world.quests){const r=records[q.id];if(r && Number.isFinite(r.xp)){q.xp=r.xp;q.noXp=r.xp===0;}else{q.xp=null;q.noXp=true;}}
 return world;
}
function replay(source,world,params,{estimateMissing=false,questFacts={},variant=1,extend=false}={}){
 const run=compileRun(structuredClone(world),{race:selectedProfile.race,cls:selectedProfile.class,family:'solo'});
 const movementPoints=addMovementPoints(run,source,read(path.join(root,'data/replay-map-frames.json')),{nearK:NEAR_K,islands:ISLANDS});
 const tables=makeTables(run,params),ev=new Evaluator(run,tables);
 const unknown=new Set(),xpUnknown=new Set(),countsUnknown=new Set(),conditions=new Set(),skippedAccepts=[],trace=[];
 const assumptions=[],estimatedQuests=new Set(),nonquestEstimates=[],eventFailures=[];
 run.syntheticQuestStates=new Map();
 let processed=0,excludedClass=0,conditionalSkipped=0,optionalExcluded=0;
 const omittedSourceActions={};
 const pendingAccepts=[],recoveredAccepts=[];
 const generatedGroups=[];
 const costLedger=timeLedger.ledger();
 const combatInputGaps=[];
 function* groups(){
  yield* source.groups;
  if(extend&&source.coverage[1]<targetLevel)for(const g of continuation(source,run,ev,{target:targetLevel,missingCounts})){generatedGroups.push(g);yield g;}
 }
 ev.reset();
 const combatBook=routeCombatConfig?routeCombat.attach(ev,run,routeCombatConfig,{engine:{run:req('sim/run'),xp:req('game/xp')},profile:selectedProfile,target:targetLevel,seed:Number(arg('--seed')||1)}):null;
 const combatRequirements=process.argv.includes('--collect-combat-requirements')?combatRequests.collect(ev,run,req('sim/run').W_KILL):null;
 routeLoop:for(const g of groups()){
  if(combatRequirements)combatRequirements.origin=g.generated?'generated':'source';
  if(ev.level>=targetLevel)break;
  if((g.flags || []).includes('hardcore'))continue;
  const allowed=g.catalog?catalogAdapter.groupAllowed(g,run,ev,qstate,LEVEL_XP,selectedProfile):condition(g,ev,run);
  if(allowed===null){conditions.add(JSON.stringify({guide:g.guide,step:g.step,condition:g.condition,guards:g.events?.filter(e=>['quest_guard','level_guard','inventory_guard'].includes(e.t))}));if(!estimateMissing)continue;assumptions.push({kind:'unknown_group_condition',guide:g.guide,assumed:true});}
  if(allowed===false){conditionalSkipped++;continue;}
  let guard=true;
  for(const [kind,id] of g.guards || []){const st=qstate(ev,run,id);const v=kind==='isOnQuest'?st===1||st===2:kind==='isNotOnQuest'?st===0||st===3:kind==='isQuestComplete'?st>=2:kind==='isNotQuestComplete'?st<2:kind==='isQuestTurnedIn'?st===3:st!==3;if(!v)guard=false;}
  if(!guard)continue;
  for(const event of g.events){
   if(ev.level>=targetLevel)break;
   const e=event;
   const eventClock=timeLedger.clock(ev);
   try{
   if(e.t==='guide')continue;
   if(catalogAdapter.optional(e,source,process.argv.includes('--with-optional'))||e.recoveryOnly||e.advisory){optionalExcluded++;continue;}
   if(e.skipIfMissing&&qstate(ev,run,e.q)===0){conditionalSkipped++;continue;}
   if(e.skipIfIncomplete&&qstate(ev,run,e.q)!==2){conditionalSkipped++;continue;}
   const ec=condition(e,ev,run);if(ec===null){conditions.add(JSON.stringify({guide:g.guide,step:g.step,conditions:e.conditions||[e.condition]}));if(!estimateMissing)continue;assumptions.push({kind:'unknown_event_condition',quest:e.q,assumed:true});}if(ec===false)continue;
   if(e.q && !compatible(e.q)){excludedClass++;continue;}
   if(g.catalog&&e.t==='move'&&movementPoints.has(e)){if(!ev.moveTo(movementPoints.get(e)))ev.unreachable++;processed++;continue;}
   if(g.catalog&&!['accept','complete','turnin','level'].includes(e.t)){if(!['quest_guard','level_guard','inventory_guard'].includes(e.t))omittedSourceActions[e.t]=(omittedSourceActions[e.t]||0)+1;continue;}
   if((g.flags || []).includes('parallel') && e.t==='complete')continue;
   if(e.q){if(!raw[e.q] || !Number.isFinite(raw[e.q].xp))xpUnknown.add(e.q);if(missingCounts.has(e.q))countsUnknown.add(e.q);}
   if(!['accept','complete','turnin','level'].includes(e.t)){if(estimateMissing)estimateNonQuest(e,ev,params,nonquestEstimates,variant);eventFailures.push({type:'unsupported_action',action:e.t,guide:g.guide,step:g.step,generated:!!g.generated});processed++;continue;}
   const seq=compileEvent(run,e,compileSequence,K_TASK);
   if(seq.unmappedObjective){eventFailures.push({type:'unmapped_objective',quest:e.q,objective:e.obj,guide:g.guide,step:g.step,origin:e.origin});processed++;continue;}
   for(const id of seq.unknown){unknown.add(id);eventFailures.push({...actionFailure(run,ev,{...e,q:id}),guide:g.guide,step:g.step,generated:!!g.generated});}
   if(estimateMissing&&seq.unknown.length&&['accept','complete','turnin'].includes(e.t)){
    estimateAction({event:e,ev,quest:questFacts[e.q]||raw[e.q],states:run.syntheticQuestStates,params,trace:assumptions,variant});
    estimatedQuests.add(Number(e.q));
   }
   if(estimateMissing&&seq.kind.length===0&&seq.unknown.length===0&&e.t!=='guide'&&e.t!=='level'){
    const p=nonquestEstimates;
    if(estimateNonQuest(e,ev,params,p,variant))assumptions.push({kind:'nonquest_action',action:e.t,estimated:true});
   }
   const before=ev.questsDone;
   for(let i=0;i<seq.kind.length && ev.level<targetLevel;i++){
    if(seq.kind[i]===K_GRIND)ev.grindTo(seq.q[i],seq.task[i]);
    else stepEvent(ev,run,seq.kind[i],seq.q[i],seq.task[i],seq.pt[i],K_ACCEPT);
   }
   if(ev.level<targetLevel){const failure=seq.indexedObjective?(seq.task.every(t=>ev.taskDone[t])?null:{type:'failed_objective',quest:e.q,objective:e.obj}):actionFailure(run,ev,e);if(failure&&failure.type!=='unknown_quest'){const detail={...failure,guide:g.guide,step:g.step,level:ev.level,generated:!!g.generated};eventFailures.push(detail);if(e.t==='accept'){skippedAccepts.push(detail);if(detail.reason==='prerequisites_not_turned_in')pendingAccepts.push({event:e,failure:detail,guide:g.guide,step:g.step});}}}
   if(ev.questsDone>before&&ev.level<targetLevel){
    for(const item of recoverAccepts(pendingAccepts,run,ev,compileSequence)){
     eventFailures.splice(eventFailures.indexOf(item.failure),1);skippedAccepts.splice(skippedAccepts.indexOf(item.failure),1);pendingAccepts.splice(pendingAccepts.findIndex(p=>p.failure===item.failure),1);recoveredAccepts.push(item);
    }
   }
   if(ev.questsDone>before)trace.push({quest:e.q,time:ev.time,level:ev.level,xp:ev.xp});
   processed++;
   }catch(error){if(error.code!=='COMBAT_INPUT_GAP')throw error;combatInputGaps.push(error.detail);break routeLoop;}
   finally{timeLedger.record(costLedger,eventClock,timeLedger.clock(ev),e.t,!!g.generated);}
  }
 }
 costLedger.validation=timeLedger.validate(costLedger,timeLedger.clock(ev));
 const metrics=ev.result();
 const blockers={combatInputGaps,invalidTimeAccounting:costLedger.validation.issues,sourceSelectionIssues:source.selectionIssues||[],sourceActionsNotReplayed:Object.entries(omittedSourceActions).map(([action,count])=>({action,count})),unvalidatedMechanics:[...(worldGapReport?.unvalidated||[]).filter(x=>trace.some(t=>t.quest===x.quest)),...(combatBook?combatBook.limitations.map(reason=>({mechanic:'stateful_combat',reason})):[])],eventFailures,unknownQuestIds:[...unknown],unknownXpIds:[...xpUnknown],unknownObjectiveCounts:[...countsUnknown],unresolvedConditions:[...conditions],skippedAccepts,unreachable:ev.unreachable};
 const coverage=generatedGroups.length?[source.coverage[0],Math.max(source.coverage[1],ev.level)]:source.coverage;
 const valid=ev.level>=targetLevel&&coverage[1]>=targetLevel&&Object.values(blockers).every(x=>Array.isArray(x)?x.length===0:x===0);
 const timeTo=Object.fromEntries([...new Set([6,12,20,targetLevel])].filter(l=>l<=targetLevel).map(l=>[l,l<=coverage[1]&&ev.level>=l?(metrics.levelTimes[l] || null):null]));
 console.log(source.label,JSON.stringify({endLevel:ev.level,seconds:ev.time,timeTo,unknown:unknown.size,counts:countsUnknown.size,xp:xpUnknown.size,conditions:conditions.size,skippedAccepts:skippedAccepts.length}));
 return {id:source.id,label:source.label,mode:estimateMissing?'estimated_exploratory':'strict_diagnostic',status:estimateMissing?(ev.level>=targetLevel?'estimated_model_complete_with_uncertainties':'estimated_model_incomplete'):valid?'provisional_model_complete':'diagnostic_replay_needs_fixes',estimatedModel:estimateMissing?{missingQuestIds:[...estimatedQuests],assumptions,nonquestEstimates,assumedTargetQuestIds:[...countsUnknown],unknownConditionBranches:conditions.size,skippedAccepts:skippedAccepts.length}:null,declaredCoverage:coverage,originalCoverage:source.coverage,generatedContinuation:generatedGroups,sourceProfile:{race:selectedProfile.race,class:selectedProfile.class,faction:selectedProfile.faction},timeTo,diagnosticLevelTimes:metrics.levelTimes,endLevel:ev.level,endXp:ev.xp,endSeconds:ev.time,costLedger,combatBook,combatRequirements,completedQuests:ev.questsDone,travelSeconds:ev.travelSec,grindSeconds:ev.grindSec,processedEvents:processed,excludedIncompatibleClassRace:excludedClass,conditionalSkipped,blockers,optionalExcluded,omittedSourceActions,recoveredAccepts,inputProvenance:source.inputProvenance||{kind:'saved_event_export'},limits:[...(source.limits || []),...(generatedGroups.length?['Generated continuation is a model heuristic, not an official source guide; original event order is preserved.']:[])],trace};
}
function main(){
 const estimateMissing=process.argv.includes('--estimate-missing');
 const combatInputFile=arg('--combat-input');
 if(combatInputFile){if(estimateMissing)throw Error('Stateful combat cannot be mixed with guessed quest work');routeCombatConfig=read(path.resolve(combatInputFile));}
 fs.mkdirSync(outDir,{recursive:true});save(estimateMissing?'status-estimated.json':'status.json',{status:'running',startedAt:new Date().toISOString(),profile:{faction:selectedProfile.faction,race:selectedProfile.race,class:selectedProfile.class,levels:[6,12,20,30]}});
 // No undocumented XP estimates: only explicit imported values earn quest XP.
 // An unvalued quest is recorded in blockers if referenced by a route.
 if(process.argv.includes('--warrior'))throw Error('Saved source chain is Orc Warlock; import the Warrior #next chain before requesting a Warrior comparison');
 const saved=process.argv.includes('--saved-inputs');
 const worldFile=arg('--world-file')||(profileSelection?path.join(root,'.comparison/matrix-worlds','world-'+selectedProfile.faction.toLowerCase()+'.json.gz'):path.join(simDir,'world-'+selectedProfile.faction.toLowerCase()+'.json'));
 if(saved&&!fs.existsSync(worldFile))throw Error('World snapshot missing for '+selectedProfile.faction+': '+worldFile);
 let world=saved?read(worldFile):getWorld({dataDir,simDir,faction:selectedProfile.faction,rebuild:true,questXp:Array.from({length:61},()=>Array(10).fill(0)),log:console.log});
 if(world.faction!==selectedProfile.faction)throw Error('World faction does not match selected profile');
 // Work on an in-memory copy. No guide, cached world or source database is edited.
 const gap=reconcileWorldGaps(world,read(path.join(root,'data/world-gap-sources.json')));world=gap.world;worldGapReport=gap.report;
 const oi=loadInputs(),overlay=reconcile(world,oi.quests,oi.guide,oi.references,missingCounts);
 world=prepareObjectiveSites(overlay.world,oi.quests,read(path.join(root,'data/objective-drop-refs.json')).items);missingCounts=overlay.missing;objectiveReport=overlay.report;
 console.log('OBJECTIVE_OVERLAY '+JSON.stringify({legacyMissing:objectiveReport.legacyMissingCount,remaining:objectiveReport.remainingCountFlags,sourceResolved:objectiveReport.legacyFlagsResolved,scaled:objectiveReport.scaled,corrected:objectiveReport.semanticCorrections,unresolved:objectiveReport.unresolvedMappings}));
 const xpReport=estimateMissing?awardXpPolicy(world,raw,{estimateMissing:true}):(applyXpPolicy(world,raw),null);
 // The saved world is a read-only baseline: never multiply counts again on a later replay.
 const parameterSet=paramsFor(simDir,selectedProfile.class);
 const params=parameterSet.mean;
 let sources=['rxp','journey','wisp'].map(id=>read(path.join(root,'.comparison/events',id+'.json')));
 if(saved)sources.push(read(path.join(root,'.comparison/events/polestar.json')));
 else {
  const L=loadPolestarRoutes({guidesDir:path.join(external,'guides'),dataDir,faction:selectedProfile.faction,race:selectedProfile.race,cls:selectedProfile.class});
  if(!L)throw Error('Polestar Orc Warlock route missing');
  const source={id:'polestar',label:'Polestar default',coverage:[1,60],groups:requiredEvents(L.default.events).map(e=>({events:[e]})),limits:['Default chain; optional objectives excluded','Same workload normalization and class priors as every other route']};

  sources.push(source);
 }

 const catalogFile=path.resolve(root,'../2026-10-23-independent-route-checks/events/catalog.json.gz');
 if(profileSelection)sources=planProfileSources(catalogFile,selectedProfile,targetLevel);
 sources=catalogAdapter.loadCatalogSources(sources,catalogFile,selectedProfile);
 const rows=[];
 for(const source of sources){const row=replay(source,world,params,{estimateMissing,questFacts:oi.quests,extend:process.argv.includes('--extend-to-30')});rows.push(row);save(source.id+(estimateMissing?'-estimated':'')+'.json',row);}

 const acceptanceRows=rows.map(r=>acceptance(r,targetLevel));
 save('acceptance-gate'+(estimateMissing?'-estimated':'')+'.json',{passed:acceptanceRows.every(x=>x.passed),routes:acceptanceRows});
 console.log('ACCEPTANCE_GATE '+JSON.stringify(acceptanceRows));
 const milestones=[...new Set([6,12,20,targetLevel])].filter(l=>l<=targetLevel).map(level=>({level,entries:rows.filter(r=>r.declaredCoverage[1]>=level&&Number.isFinite(r.timeTo[level])).map(r=>({id:r.id,seconds:r.timeTo[level]})).sort((a,b)=>a.seconds-b.seconds)}));
 const comparison=milestones.map(m=>({...m,fullFourRouteCoverage:m.entries.length===4,rankingKind:'disabled',warning:'Not valid for ranking until all four reach the milestone and quest state/locations are validated'}));
 const report={schema:'foreverroutes.shared_comparison.v3',finishedAt:new Date().toISOString(),mode:estimateMissing?'estimated_exploratory':'strict_diagnostic',status:estimateMissing?(rows.every(r=>r.endLevel>=targetLevel)?'estimated_model_complete_with_uncertainties':'estimated_model_incomplete'):rows.every(r=>r.status==='provisional_model_complete')?'complete_provisional':'completed_diagnostic_replays',modelComparison:comparison,modelXpPolicy:xpReport,sourcePriority:xpReport?.priority||null,profile:{faction:selectedProfile.faction,race:selectedProfile.race,class:selectedProfile.class,targets:[...new Set([6,12,20,targetLevel])].filter(l=>l<=targetLevel),solo:true,aoe:false,rested:false},engine:'fitzforge/polestar@eb7b47840f032bf711a3c8d7b1df9f3f6551b990',modelParams:params,combatInput:combatInputFile?{file:path.resolve(combatInputFile),sha256:crypto.createHash('sha256').update(fs.readFileSync(combatInputFile)).digest('hex'),diagnosticOnly:true}:null,timeModel:{kind:routeCombatConfig?'isolated_foreversim_stateful_diagnostic':parameterSet.doc.fit?'parameter_fit_requires_class_validation':'uncalibrated_class_priors',fit:parameterSet.doc.fit||null,classFactor:params.classFactor,combatIncludesRecovery:true,levelOverheadIsGeneric:true},worldInput:{file:worldFile,sha256:fs.existsSync(worldFile)?crypto.createHash('sha256').update(fs.readFileSync(worldFile)).digest('hex'):null},provenance:read(path.join(dataDir,'meta.json')),xpOverlay:{...xpOverlay.applied,replayReferenceSources:replayXpReferences,assumptionsNotVerified:true,originalQuestFactsUnchanged:true},objectiveCountOverlay:objectiveReport,worldGapOverlay:worldGapReport,rows,rankingPublished:false,acceptancePassed:acceptanceRows.every(x=>x.passed),reason:'Unresolved route semantics / missing objective and location inputs prevent a defensible fastest-route ranking'};
 if(process.argv.includes('--require-complete')&&!report.acceptancePassed)process.exitCode=2;
 save(estimateMissing?'comparison-estimated.json':'comparison.json',report);save(estimateMissing?'status-estimated.json':'status.json',{status:report.status,finishedAt:report.finishedAt,outputs:rows.map(r=>r.id+(estimateMissing?'-estimated':'')+'.json'),rankingPublished:false});
 if(estimateMissing)console.log('ESTIMATED_MODEL_COMPARISON '+JSON.stringify({mode:report.mode,modelXpPolicy:xpReport.counts,modelComparison:comparison.map(({level,entries,fullFourRouteCoverage})=>({level,entries,fullFourRouteCoverage})),remainingUncertainties:rows.map(r=>({id:r.id,unknown:r.blockers.unknownQuestIds.length,unknownCounts:r.blockers.unknownObjectiveCounts.length,skippedAccepts:r.blockers.skippedAccepts.length}))}));
}
if(require.main===module)try{main();}catch(e){fs.mkdirSync(outDir,{recursive:true});save(process.argv.includes('--estimate-missing')?'status-estimated.json':'status.json',{status:'failed',error:e.message,at:new Date().toISOString()});console.error(e);process.exitCode=1;}
module.exports={rxpCondition,journeyCondition,qstate,applyXpPolicy,effectiveQuestXp};

