'use strict';
// Preserve objective indices instead of turning every indexed instruction into
// a full-quest completion. Only source/order-preserving mappings are accepted.
function prepareObjectiveSites(world,facts,dropRefs={}){
 for(const q of world.quests){
  const objectives=facts[q.id]?.objectives||[],tasks=q.tasks||[];
  if(!objectives.length)continue;
  const ordered=(tasks.length===1&&tasks[0].work.length===objectives.length)||(tasks.length===objectives.length&&tasks.every(t=>t.work.length===1));
  q.tasks=tasks.flatMap(t=>t.work.map(work=>({pt:t.pt,work:[work]})));
  const assigned=new Set();
  q.objectiveTasks=objectives.map((o,index)=>{
   if(ordered){assigned.add(index);return index;}
   const mobIds=o.type==='kill'?[o.id]:o.type==='item'?(dropRefs[o.id]?.npcDrops||[]):[];
   const matches=q.tasks.map((t,i)=>t.work.some(w=>w[0]==='kill'&&mobIds.includes(w[5]))?i:-1).filter(i=>i>=0);
   const point=dropRefs[o.id]?.singleObjectPoint;
   if(!matches.length&&point){
    // The pinned builder groups in source-objective order into the FIRST
    // existing site within SITE_R=70 yards. Reproduce that exact rule only
    // for a single sourced object at a single sourced spawn.
    const site=tasks.find(t=>{const p=world.points[t.pt];return p[1]===point[1]&&Math.hypot(p[2]-point[2],p[3]-point[3])<=70.15;});
    if(site){const relative=q.tasks.findIndex((t,i)=>t.pt===site.pt&&t.work[0][0]==='use'&&!assigned.has(i));if(relative>=0){assigned.add(relative);return relative;}}
   }
   if(matches.length===1)assigned.add(matches[0]);
   return matches.length===1&&q.tasks[matches[0]].work.length===1?matches[0]:null;
  });
 }
 return world;
}
function compileEvent(run,event,compileSequence,taskKind){
 if(event.t!=='complete'||event.obj==null)return compileSequence(run,[event]);
 const qi=run.qIndex.get(Number(event.q));
 if(qi==null)return compileSequence(run,[event]);
 const relative=run.quests[qi].objectiveTasks?.[event.obj-1];
 if(!Number.isInteger(relative))return {kind:[],q:[],task:[],pt:[],unknown:[],unmappedObjective:true};
 const task=run.qTask0[qi]+relative;
 return {kind:[taskKind],q:[qi],task:[task],pt:[run.taskPt[task]],unknown:[],indexedObjective:true};
}
module.exports={prepareObjectiveSites,compileEvent};
