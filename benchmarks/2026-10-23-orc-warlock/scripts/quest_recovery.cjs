'use strict';
// Retry a deferred acceptance ONLY on a later real visit to that giver.
// Prerequisites, minimum levels, log space and travel checks stay in the engine.
function recoverAccepts(pending,run,ev,compileSequence){
 const recovered=[];
 for(const item of pending){
  const i=run.qIndex.get(Number(item.event.q));
  if(i==null||ev.done[i]||ev.active[i])continue;
  if(run.qGiver[i]!==ev.pos||!ev.prereqOk(i)||ev.open>=run.logCap||ev.level<run.qMinLevel[i]||ev.level+ev.p.dangerGap<run.qLevel[i])continue;
  const seq=compileSequence(run,[item.event]);
  for(let j=0;j<seq.kind.length;j++)ev.step(seq.kind[j],seq.q[j],seq.task[j],seq.pt[j]);
  if(ev.active[i]||ev.done[i])recovered.push({...item,time:ev.time,level:ev.level});
 }
 return recovered;
}
module.exports={recoverAccepts};
