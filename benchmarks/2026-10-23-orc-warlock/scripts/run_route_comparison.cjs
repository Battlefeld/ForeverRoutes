'use strict';
// Shared deterministic route replay. Numeric output is diagnostic when blockers remain.
const fs=require('node:fs'),path=require('node:path');
const root=path.resolve(__dirname,'..'),external=path.resolve(process.env.POLESTAR_ROOT || path.join(root,'..','polestar'));
const req=p=>require(path.join(external,'tools/lib',p));
const {getWorld}=req('sim/world'),{makeRun,paramsFor}=req('sim/index');
const {compileSequence}=req('sim/sequence'),{K_GRIND}=req('sim/run');
const {loadPolestarRoutes,requiredEvents}=req('sim/baselines/polestar');
const {reconcile,loadInputs}=require('./objective_count_overlay.cjs');
const {awardXpPolicy,estimateAction,estimateNonQuest}=require('./estimate_model.cjs');
const dataDir=path.join(root,'.comparison/data'),simDir=path.join(root,'.comparison/cache'),outDir=path.join(root,'.comparison/results');
const read=p=>JSON.parse(fs.readFileSync(p,'utf8'));
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
const xpOverlay=effectiveQuestXp(rawQuestFacts,
 fs.existsSync(assumedXpFile)?read(assumedXpFile):null,
 fs.existsSync(verifiedXpFile)?read(verifiedXpFile):null);
const raw=xpOverlay.records;
let missingCounts=new Set(read(path.join(dataDir,'missing-counts.json'))),objectiveReport=null;
const flags=new Set(['WARRIOR','ORC','HORDE','FOREVER','ERA','SOFTCORE','H12B']);
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
 function val(){const t=tokens[i++];if(t==='('){const v=expr();if(tokens[i++]!==')')throw Error('unbalanced');return v;}if(t==null)throw Error('missing term');if(/^\d/.test(t))return +t;const u=t.toUpperCase();if(u==='LEVEL')return ev.level;if(u==='ACCEPTED')return 1;if(u==='COMPLETED')return 2;if(u==='TURNEDIN')return 3;if(u==='VAR.HORDETB20')return false;const q=/^QUEST\[(\d+)\]$/.exec(u);if(q)return qstate(ev,run,q[1]);if(knownFlags.has(u))return flags.has(u);return null;}
 function comparison(){let a=val();const op=tokens[i];if(!['>=','<=','==','~=','>','<'].includes(op))return a;i++;const b=val();if(a===null||b===null)return null;return op==='>='?a>=b:op==='<='?a<=b:op==='=='?a===b:op==='~='?a!==b:op==='>'?a>b:a<b;}
 function not(){if((tokens[i] || '').toLowerCase()==='not'){i++;const v=not();return v===null?null:!v;}return comparison();}
 function all(){let a=not();while((tokens[i] || '').toLowerCase()==='and'){i++;a=and(a,not());}return a;}
 function expr(){let a=all();while((tokens[i] || '').toLowerCase()==='or'){i++;a=or(a,all());}return a;}
 try{const r=expr();return i===tokens.length?(r===null?null:!!r):null;}catch{return null;}
}
function condition(g,ev,run){
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
function compatible(id){const q=raw[id];if(!q)return true;return !(q.classes && !(q.classes&1)) && !(q.races && !(q.races&2));}
function applyXpPolicy(world,records){
 for(const q of world.quests){const r=records[q.id];if(r && Number.isFinite(r.xp)){q.xp=r.xp;q.noXp=r.xp===0;}else{q.xp=null;q.noXp=true;}}
 return world;
}
function validateGenerated(groups,existing){
 const issues=[],seen=new Set(existing),accepted=new Set(),completed=new Set(),done=new Set();
 for(const g of groups)for(const e of g.events||[]){
  if(!e.q)continue;
  const id=Number(e.q),q=raw[id];
  if(!q){issues.push({id,reason:'unknown_quest'});continue;}
  if(!compatible(id)){issues.push({id,reason:'incompatible'});continue;}
  if(e.t==='accept'){
   if(seen.has(id)){issues.push({id,reason:'already_in_original'});continue;}
   accepted.add(id);
  }else if(e.t==='complete'){
   if(!accepted.has(id))issues.push({id,reason:'complete_without_accept'});
   else completed.add(id);
  }else if(e.t==='turnin'){
   if(!accepted.has(id)||!completed.has(id))issues.push({id,reason:'turnin_without_completion'});
   else{done.add(id);seen.add(id);}
  }
 }
 return {valid:issues.length===0,issues,accepted:accepted.size,completed:completed.size,turnedIn:done.size};
}
function replay(source,world,params,{estimateMissing=false,questFacts={},variant=1}={}){
 const {run,ev}=makeRun(world,{race:'Orc',cls:'Warrior',params,family:'solo'});
 const unknown=new Set(),xpUnknown=new Set(),countsUnknown=new Set(),conditions=new Set(),skippedAccepts=[],trace=[];
 const assumptions=[],estimatedQuests=new Set(),nonquestEstimates=[],eventFailures=[];
 run.syntheticQuestStates=new Map();
 let processed=0,excludedClass=0,conditionalSkipped=0;
 ev.reset();
 for(const g of source.groups){
  if(ev.level>=30)break;
  if((g.flags || []).includes('hardcore'))continue;
  const allowed=condition(g,ev,run);
  if(allowed===null){conditions.add(JSON.stringify(g.condition));if(!estimateMissing)continue;assumptions.push({kind:'unknown_group_condition',guide:g.guide,assumed:true});}
  if(!allowed){conditionalSkipped++;continue;}
  let guard=true;
  for(const [kind,id] of g.guards || []){const st=qstate(ev,run,id);const v=kind==='isOnQuest'?st===1||st===2:kind==='isNotOnQuest'?st===0||st===3:kind==='isQuestComplete'?st>=2:kind==='isNotQuestComplete'?st<2:kind==='isQuestTurnedIn'?st===3:st!==3;if(!v)guard=false;}
  if(!guard)continue;
  for(const event of g.events){
   if(ev.level>=30)break;
   const e=event;
   if(e.t==='guide')continue;
   const ec=condition(e,ev,run);if(ec===null){conditions.add(e.condition);if(!estimateMissing)continue;assumptions.push({kind:'unknown_event_condition',quest:e.q,assumed:true});}if(ec===false)continue;
   if(e.q && !compatible(e.q)){excludedClass++;continue;}
   if((g.flags || []).includes('parallel') && e.t==='complete')continue;
   if(e.q){if(!raw[e.q] || !Number.isFinite(raw[e.q].xp))xpUnknown.add(e.q);if(missingCounts.has(e.q))countsUnknown.add(e.q);}
   const seq=compileSequence(run,[e]);for(const id of seq.unknown){unknown.add(id);eventFailures.push({type:'unknown_quest',quest:id,action:e.t,guide:g.guide,generated:!!g.generated});}
   if(estimateMissing&&seq.unknown.length&&['accept','complete','turnin'].includes(e.t)){
    estimateAction({event:e,ev,quest:questFacts[e.q]||raw[e.q],states:run.syntheticQuestStates,params,trace:assumptions,variant});
    estimatedQuests.add(Number(e.q));
   }
   if(estimateMissing&&seq.kind.length===0&&seq.unknown.length===0&&e.t!=='guide'&&e.t!=='level'){
    const p=nonquestEstimates;
    if(estimateNonQuest(e,ev,params,p,variant))assumptions.push({kind:'nonquest_action',action:e.t,estimated:true});
   }
   const before=ev.questsDone;
   for(let i=0;i<seq.kind.length && ev.level<30;i++){
    if(seq.kind[i]===K_GRIND)ev.grindTo(seq.q[i],seq.task[i]);
    else ev.step(seq.kind[i],seq.q[i],seq.task[i],seq.pt[i]);
   }
   if(e.t==='accept'){const qi=run.qIndex.get(e.q);if(qi!=null&&!ev.active[qi]&&!ev.done[qi]){const quest=world.quests.find(q=>Number(q.id)===Number(e.q));const failure={quest:e.q,guide:g.guide,step:g.step,level:ev.level,generated:!!g.generated,questLevel:quest?.level??null,minLevel:quest?.minLevel??null,requires:quest?.requires??quest?.requiredQuests??quest?.prerequisites??null,questState:qstate(ev,run,e.q)};skippedAccepts.push(failure);eventFailures.push({type:'skipped_accept',...failure});}}
   if(ev.questsDone>before)trace.push({quest:e.q,time:ev.time,level:ev.level,xp:ev.xp});
   processed++;
  }
 }
 const metrics=ev.result();
 const blockers={eventFailures,unknownQuestIds:[...unknown],unknownXpIds:[...xpUnknown],unknownObjectiveCounts:[...countsUnknown],unresolvedConditions:[...conditions],skippedAccepts,unreachable:ev.unreachable};
 const valid=Object.values(blockers).every(x=>Array.isArray(x)?x.length===0:x===0);
 const timeTo=Object.fromEntries([6,12,20,30].map(l=>[l,l<=source.coverage[1]&&ev.level>=l?(metrics.levelTimes[l] || null):null]));
 console.log(source.label,JSON.stringify({endLevel:ev.level,seconds:ev.time,timeTo,unknown:unknown.size,counts:countsUnknown.size,xp:xpUnknown.size,conditions:conditions.size,skippedAccepts:skippedAccepts.length}));
 return {id:source.id,label:source.label,mode:estimateMissing?'estimated_exploratory':'strict_diagnostic',status:estimateMissing?'estimated_model_complete_with_uncertainties':valid?'provisional_model_complete':'diagnostic_replay_needs_fixes',estimatedModel:estimateMissing?{missingQuestIds:[...estimatedQuests],assumptions,nonquestEstimates,assumedTargetQuestIds:[...countsUnknown],unknownConditionBranches:conditions.size,skippedAccepts:skippedAccepts.length}:null,declaredCoverage:source.coverage,timeTo,diagnosticLevelTimes:metrics.levelTimes,endLevel:ev.level,endXp:ev.xp,endSeconds:ev.time,completedQuests:ev.questsDone,travelSeconds:ev.travelSec,grindSeconds:ev.grindSec,processedEvents:processed,excludedIncompatibleClassRace:excludedClass,conditionalSkipped,blockers,limits:source.limits || [],trace};
}
function main(){
 fs.mkdirSync(outDir,{recursive:true});save('status.json',{status:'running',startedAt:new Date().toISOString(),profile:{faction:'Horde',race:'Orc',class:'Warrior',levels:[6,12,20,30]}});
 // No undocumented XP estimates: only explicit imported values earn quest XP.
 // An unvalued quest is recorded in blockers if referenced by a route.
 const saved=process.argv.includes('--saved-inputs');
 const estimateMissing=process.argv.includes('--estimate-missing');
 let world=saved?read(path.join(simDir,'world-horde.json')):getWorld({dataDir,simDir,faction:'Horde',rebuild:true,questXp:Array.from({length:61},()=>Array(10).fill(0)),log:console.log});
 // Work on an in-memory copy. No guide, cached world or source database is edited.
 const oi=loadInputs(),overlay=reconcile(world,oi.quests,oi.guide,oi.references,missingCounts);
 world=overlay.world;missingCounts=overlay.missing;objectiveReport=overlay.report;
 console.log('OBJECTIVE_OVERLAY '+JSON.stringify({legacyMissing:objectiveReport.legacyMissingCount,remaining:objectiveReport.remainingCountFlags,sourceResolved:objectiveReport.legacyFlagsResolved,scaled:objectiveReport.scaled,corrected:objectiveReport.semanticCorrections,unresolved:objectiveReport.unresolvedMappings}));
 const xpReport=estimateMissing?awardXpPolicy(world,raw,{estimateMissing:true}):(applyXpPolicy(world,raw),null);
 // The saved world is a read-only baseline: never multiply counts again on a later replay.
 const {mean:params}=paramsFor(simDir,'Warrior');
 const sources=['rxp','journey','wisp'].map(id=>read(path.join(root,'.comparison/events',id+'.json')));
 if(saved && !process.argv.includes('--warrior'))sources.push(read(path.join(root,'.comparison/events/polestar.json')));
 else {
  const L=loadPolestarRoutes({guidesDir:path.join(external,'guides'),dataDir,faction:'Horde',race:'Orc',cls:'Warrior'});
  if(!L)throw Error('Polestar Orc Warrior route missing');
  const source={id:'polestar',label:'Polestar default',coverage:[1,60],groups:requiredEvents(L.default.events).map(e=>({events:[e]})),limits:['Default chain; optional objectives excluded','Same workload normalization and class priors as every other route']};
  if(!saved)fs.writeFileSync(path.join(root,'.comparison/events/polestar.json'),JSON.stringify(source)+'\n');
  sources.push(source);
 }
 if(process.argv.includes('--extend-to-30')){
  // Shared eligible quest pool, not a copied donor route. Build two distinct
  // generated plans with independently scored quest ordering.
  const journey=sources.find(x=>x.id==='journey');
  const pool=new Map();
  let order=0;
  for(const g of journey.groups){
   const match=/^H(\d{2})/.exec(g.guide||'');
   const chapterLevel=match?Number(match[1]):0;
   if(chapterLevel<18||chapterLevel>30)continue;
   for(const e of g.events||[]){
    if(!e.q||!['accept','complete','turnin'].includes(e.t)||!compatible(e.q))continue;
    const q=raw[e.q];if(!q)continue;
    if(!pool.has(e.q))pool.set(e.q,{id:e.q,level:chapterLevel,xp:Number.isFinite(q.xp)?q.xp:0,zone:g.guide||'',order:order++,events:[]});
    pool.get(e.q).events.push(e.t);
   }
  }
  for(const id of ['rxp','wisp']){
   const target=sources.find(x=>x.id===id),from=id==='rxp'?23:21;
   const existing=new Set(target.groups.flatMap(g=>(g.events||[]).map(e=>e.q).filter(Boolean)));
   const worldIds=new Set(world.quests.map(q=>Number(q.id)));
   const candidates=[...pool.values()].filter(q=>q.level>=from-5&&!existing.has(q.id)&&worldIds.has(q.id));
   console.log('EXTENSION_SOURCE_COVERAGE '+JSON.stringify({id,excludedMissingWorld:[...pool.values()].filter(q=>q.level>=from-5&&!existing.has(q.id)&&!worldIds.has(q.id)).map(q=>q.id)}));
   // RXP: favor XP density and short chains. Wisp: favor contiguous
   // quest-hub progression, avoiding repeated area changes.
   // Keep source dependency order within each chapter. XP scoring may only
   // prioritize between chapters, never reorder prerequisite chains.
   candidates.sort((a,b)=>(a.level-b.level)||(a.order-b.order));
   const groups=candidates.map(q=>({
    guide:'GENERATED_'+id+'_L'+from+'_30',
    generated:true,style:id==='rxp'?'xp-density-speed':'level-hub-continuity',
    events:['accept','complete','turnin'].map(t=>({t,q:q.id,generated:true}))
   }));
   const audit=validateGenerated(groups,existing);
   if(!audit.valid)throw Error('Generated route validation failed: '+id+' '+JSON.stringify(audit.issues.slice(0,10)));
   console.log('ROUTE_CHAIN_AUDIT '+JSON.stringify({id,...audit}));
   target.groups=target.groups.concat(groups);
   target.coverage=[1,30];
   target.limits.push('GENERATED '+from+'-30: '+(id==='rxp'?'XP density/speed':'level/hub continuity')+' heuristic; not official guide; common quest facts');
   console.log('STYLE_EXTENSION '+JSON.stringify({id,from,quests:candidates.length,policy:id==='rxp'?'level-gated-xp-speed':'level-hub-continuity'}));
  }
 }
 const rows=[];
 for(const source of sources){const row=replay(source,world,params,{estimateMissing,questFacts:oi.quests});rows.push(row);save(source.id+(estimateMissing?'-estimated':'')+'.json',row);}
 const extensionCheck=rows.filter(r=>['rxp','wisp'].includes(r.id)).map(r=>({id:r.id,endLevel:r.endLevel,skippedAccepts:r.blockers.skippedAccepts.length,unknownQuestIds:r.blockers.unknownQuestIds.length,generatedFailures:r.blockers.eventFailures.filter(f=>f.generated).slice(0,30),originalFailures:r.blockers.eventFailures.filter(f=>!f.generated).slice(0,30),ready:r.endLevel>=30&&r.blockers.skippedAccepts.length===0&&r.blockers.unknownQuestIds.length===0}));
 console.log('EXTENSION_READINESS '+JSON.stringify(extensionCheck));
 const milestones=[6,12,20,30].map(level=>({level,entries:rows.filter(r=>r.declaredCoverage[1]>=level&&Number.isFinite(r.timeTo[level])).map(r=>({id:r.id,seconds:r.timeTo[level]})).sort((a,b)=>a.seconds-b.seconds)}));
 const comparison=milestones.map(m=>({...m,fullFourRouteCoverage:m.entries.length===4,rankingKind:'disabled',warning:'Not valid for ranking until all four reach the milestone and quest state/locations are validated'}));
 const report={schema:'foreverroutes.shared_comparison.v2',finishedAt:new Date().toISOString(),mode:estimateMissing?'estimated_exploratory':'strict_diagnostic',status:estimateMissing?'estimated_model_complete':rows.every(r=>r.status==='provisional_model_complete')?'complete_provisional':'completed_diagnostic_replays',modelComparison:comparison,modelXpPolicy:xpReport,sourcePriority:xpReport?.priority||null,profile:{faction:'Horde',race:'Orc',class:'Warrior',targets:[6,12,20,30],solo:true,aoe:false,rested:false},engine:'fitzforge/polestar@eb7b47840f032bf711a3c8d7b1df9f3f6551b990',modelParams:params,provenance:read(path.join(dataDir,'meta.json')),xpOverlay:{...xpOverlay.applied,assumptionsNotVerified:true,originalQuestFactsUnchanged:true},objectiveCountOverlay:objectiveReport,rows,rankingPublished:false,reason:'Unresolved route semantics / missing objective and location inputs prevent a defensible fastest-route ranking'};
 save(estimateMissing?'comparison-estimated.json':'comparison.json',report);save(estimateMissing?'status-estimated.json':'status.json',{status:report.status,finishedAt:report.finishedAt,outputs:rows.map(r=>r.id+(estimateMissing?'-estimated':'')+'.json'),rankingPublished:false});
 if(estimateMissing)console.log('ESTIMATED_MODEL_COMPARISON '+JSON.stringify({mode:report.mode,modelXpPolicy:xpReport.counts,modelComparison:comparison.map(({level,entries,fullFourRouteCoverage})=>({level,entries,fullFourRouteCoverage})),remainingUncertainties:rows.map(r=>({id:r.id,unknown:r.blockers.unknownQuestIds.length,unknownCounts:r.blockers.unknownObjectiveCounts.length,skippedAccepts:r.blockers.skippedAccepts.length}))}));
}
if(require.main===module)try{main();}catch(e){fs.mkdirSync(outDir,{recursive:true});save('status.json',{status:'failed',error:e.message,at:new Date().toISOString()});console.error(e);process.exitCode=1;}
module.exports={rxpCondition,journeyCondition,qstate,applyXpPolicy,effectiveQuestXp};
