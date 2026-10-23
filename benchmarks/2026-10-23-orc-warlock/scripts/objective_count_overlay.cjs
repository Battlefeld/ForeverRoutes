'use strict';
// The cached world used a silent count=1 for missing objectives. Reconcile it
// with verified numeric facts without changing any imported guide events.
const fs=require('node:fs'),path=require('node:path'),zlib=require('node:zlib');
const base=path.resolve(__dirname,'..'),audit=path.resolve(base,'../2026-10-23-independent-route-checks');
function read(p){return JSON.parse(fs.readFileSync(p,'utf8'));}
function loadInputs(){
 const quests=JSON.parse(zlib.gunzipSync(fs.readFileSync(path.join(audit,'data/quests.json.gz'))));
 const guide=read(path.join(audit,'data/objective-facts.json'));
 const references=read(path.join(base,'data/wowhead-objective-evidence-20261023.json'));
 return {quests,guide,references};
}
const isCount=n=>Number.isInteger(n)&&n>0;
function reconcile(world,sourceQuests,guideFacts,referenceFacts,legacyMissing){
 const quests=structuredClone(sourceQuests),copy=structuredClone(world);
 const newEvidence=[],conflicts=[],unresolvedMappings=[],scaled=[],special=[],semanticCorrections=[];
 const worldById=new Map((copy.quests||[]).map(q=>[q.id,q]));
 const confirmedGuide=new Set((guideFacts.filled||[]).map(x=>x.quest+':'+x.objective));
 // Preserve existing confirmed guide overlays, but add a distinct source layer for
 // newly checked Wowhead facts. Never overwrite known counts with conflicts.
 for(const item of referenceFacts.entries||[]){
  const {quest,objective,type,target,count,url,action}=item;
  const q=quests[quest],o=q?.objectives?.[objective-1];
  if(!q||!o){unresolvedMappings.push({quest,objective,reason:'objective_absent'});continue;}
  if(!isCount(count)||!url||type!==o.type||(target!=null&&Number(target)!==Number(o.id))){
   conflicts.push({quest,objective,reason:'evidence_does_not_match_objective'});continue;
  }
  if(isCount(o.count)){
   if(o.count!==count)conflicts.push({quest,objective,old:o.count,verified:count,reason:'count_conflict'});
  }else{
   o.count=count;
   o.countProvenance={count,evidence:[{source:item.source,url,quality:'page_reference',verifiedAt:item.verifiedAt}]};
   newEvidence.push({quest,objective,count,url});
  }
  if(action)special.push({quest,objective,action,url});
 }
 // A cached quest work item may represent a drop probability, so multiply
 // expected kills/uses by the proven item quantity; never substitute 1 kill.
 // Only map when original objective order is provably retained (one site with
 // all work items, or one work per site). Grouped mixed sites are ambiguous.
 for(const [id,q] of Object.entries(quests)){
  const changed=(q.objectives||[]).map((o,i)=>({o,i})).filter(x=>x.o&&isCount(x.o.count)&&x.o.count>1&&x.o.countProvenance);
  if(!changed.length)continue;
  const w=worldById.get(Number(id));
  if(!w){unresolvedMappings.push({quest:Number(id),reason:'not_in_playable_world',objectives:changed.map(x=>x.i+1)});continue;}
  const objs=q.objectives;
  let work=null;
  if((w.tasks||[]).length===1&&w.tasks[0].work.length===objs.length)work=w.tasks[0].work;
  else if(w.tasks.length===objs.length&&w.tasks.every(t=>t.work.length===1))work=w.tasks.map(t=>t.work[0]);
  if(!work){unresolvedMappings.push({quest:Number(id),reason:'ambiguous_objective_site_grouping',objectives:changed.map(x=>x.i+1)});continue;}
  for(const {o,i} of changed){
   const item=work[i],kind=item?.[0];
   const supported=o.type==='kill'?kind==='kill'&&item[5]===Number(o.id):
      o.type==='object'?kind==='use':
      o.type==='item'?kind==='kill'||kind==='use':false;
   if(!supported){unresolvedMappings.push({quest:Number(id),objective:i+1,reason:'work_kind_or_target_mismatch',kind});continue;}
   const col=kind==='kill'?2:1,prior=item[col];
   if(!(prior>0&&Number.isFinite(prior))){unresolvedMappings.push({quest:Number(id),objective:i+1,reason:'invalid_original_work'});continue;}
   item[col]=prior*o.count;
   scaled.push({quest:Number(id),objective:i+1,kind,oldWork:prior,newWork:item[col],quantity:o.count});
  }
 }
 // Quest 5727 is a dialogue with Neeru Fireblade, not a kill of this NPC.
 // The quest text (Classic as the explicit reference) confirms this; match
 // its exact NPC so no other objective can ever be silently rewritten.
 const n=worldById.get(5727);
 if(n){
  const work=(n.tasks||[]).flatMap(t=>t.work||[]);
  if(work.length===1&&work[0][0]==='kill'&&work[0][5]===3216){
   n.tasks[0].work[0]=['use',1];
   semanticCorrections.push({quest:5727,from:'kill',to:'use',target:3216});
  }else if(work.length===1&&work[0][0]==='use'){
   semanticCorrections.push({quest:5727,alreadyCorrect:true});
  }else unresolvedMappings.push({quest:5727,reason:'dialogue_work_not_unambiguously_identified'});
 }
 // Other semantic actions are not yet a timed escort/skill simulator. Mark
 // them, do not claim a valid comparison merely because quantity is known.
 const unsupportedActions=special.filter(x=>['escort','skill_gate'].includes(x.action));
 const missing=new Set([...legacyMissing].filter(id=>{
  const q=quests[id];
  if(!q?.objectives?.length)return true;
  return q.objectives.some(o=>o && !isCount(o.count));
 }));
 const resolved=[...legacyMissing].filter(id=>!missing.has(id));
 // An unresolvable rescale is a blocker even when the numeric count is known.
 for(const x of unresolvedMappings)if((x.objective!=null||x.objectives?.length)&&x.reason!=='objective_absent')missing.add(x.quest);
 return {world:copy,quests,missing,report:{schema:'foreverroutes.objective_overlay.v1',legacyMissingCount:legacyMissing.size,newEvidence,sourceGuideFacts:confirmedGuide.size,legacyFlagsResolved:resolved.length,resolvedQuestIds:resolved,scaled,semanticCorrections,unsupportedActions,unresolvedMappings,conflicts,remainingCountFlags:missing.size}};
}
module.exports={reconcile,loadInputs};
