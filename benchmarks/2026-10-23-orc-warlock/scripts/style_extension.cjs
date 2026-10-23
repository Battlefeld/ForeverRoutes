'use strict';
// Generated continuation from the shared WORLD, never another guide's sequence.
// Call after each yielded group has been replayed, so eligibility uses live state.
function* continuation(source,run,ev,{target,missingCounts}){
 const original=new Set(source.groups.flatMap(g=>(g.events||[]).map(e=>Number(e.q)).filter(Boolean)));
 const attempted=new Set();
 let previousZone=null;
 while(ev.level<target){
  const candidates=[];
  for(let i=0;i<run.Q;i++){
   const q=run.quests[i];
   if(original.has(q.id)||attempted.has(q.id)||ev.done[i]||ev.active[i]||missingCounts.has(q.id)||!(q.xp>0)||q.minLevel>=target)continue;
   if(!ev.prereqOk(i)||ev.level<q.minLevel||ev.level+ev.p.dangerGap<q.level)continue;
   if(ev.open>=run.logCap)continue;
   let work=0;
   for(const task of q.tasks||[])for(const w of task.work||[])work+=w[0]==='kill'?w[2]*20:w[0]==='use'?w[1]*8:60;
   const a=run.qGiver[i],b=ev.pos;
   const distance=b>=0?Math.hypot(run.pX[a]-run.pX[b],run.pY[a]-run.pY[b]):0;
   const sameZone=q.zone===previousZone;
   // Distinct, documented heuristics, with a stable ID tie-break.
   const score=source.id==='rxp'?(work+distance/7+12)/q.xp:(sameZone?0:100000)+Math.abs(q.level-ev.level)*1000+distance+work;
   if(Number.isFinite(score))candidates.push({q,score});
  }
  candidates.sort((a,b)=>a.score-b.score||a.q.id-b.q.id);
  if(!candidates.length)return;
  const q=candidates[0].q;
  attempted.add(q.id);previousZone=q.zone;
  yield {guide:'GENERATED_'+source.id+'_CONTINUATION',generated:true,style:source.id==='rxp'?'xp-per-work-and-distance':'level-and-hub-continuity',events:['accept','complete','turnin'].map(t=>({t,q:q.id,generated:true}))};
 }
}
module.exports={continuation};
