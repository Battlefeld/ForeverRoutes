'use strict';
// Observe engine state; never mark a quest done or weaken its prerequisites.
function state(run,ev,id){
 const q=run.qIndex.get(Number(id));
 if(q==null)return null;
 if(ev.done[q])return 3;
 if(!ev.active[q])return 0;
 for(let t=run.qTask0[q];t<run.qTask1[q];t++)if(!ev.taskDone[t])return 1;
 return 2;
}
function rejection(run,ev,id){
 const q=run.qIndex.get(Number(id));
 if(q==null){
  const fact=run.world?.quests?.find(x=>x.id===Number(id));
  const excluded=run.world?.excluded?.find(x=>x.id===Number(id));
  return {reason:excluded?.reason||'not_in_character_world',worldQuestPresent:!!fact};
 }
 const ids=(list,a,b)=>Array.from(list.slice(a,b),i=>run.quests[i].id);
 const any=ids(run.preList,run.qPre0[q],run.qPre1[q]);
 const all=ids(run.allList,run.qAll0[q],run.qAll1[q]);
 const exclusive=ids(run.exclList,run.qExcl0[q],run.qExcl1[q]).filter(id=>state(run,ev,id)>0);
 const missingAll=all.filter(id=>state(run,ev,id)!==3);
 const missingAny=any.length&&!any.some(id=>state(run,ev,id)===3)?any:[];
 const parent=run.qParent[q]>=0?run.quests[run.qParent[q]].id:null;
 const parentMissing=parent!=null&&![1,2].includes(state(run,ev,parent));
 const reason=exclusive.length?'exclusive_quest':parentMissing?'parent_not_active':missingAll.length||missingAny.length?'prerequisites_not_turned_in':ev.level<run.qMinLevel[q]||ev.level+(ev.p?.dangerGap||0)<run.qLevel[q]?'level_gate':ev.open>=run.logCap?'quest_log_full':'engine_action_rejected';
 return {reason,missingAny,missingAll,exclusive,parentMissing:parentMissing?parent:null,minLevel:run.qMinLevel[q],questLevel:run.qLevel[q]};
}
function actionFailure(run,ev,event){
 if(!['accept','complete','turnin'].includes(event.t))return null;
 const actual=state(run,ev,event.q),required={accept:1,complete:2,turnin:3}[event.t];
 if(actual!=null&&actual>=required)return null;
 return {type:actual==null?'unknown_quest':'failed_'+event.t,quest:Number(event.q),action:event.t,actualState:actual,requiredState:required,...rejection(run,ev,event.q)};
}
function acceptance(row,target){
 const blockers=row.blockers;
 const reasons=[];
 if(row.endLevel<target)reasons.push('target_not_reached');
 if(row.declaredCoverage[1]<target)reasons.push('source_coverage_short');
 if(!Number.isFinite(row.timeTo[target])||row.timeTo[target]<=0)reasons.push('milestone_time_missing');
 for(const [key,value] of Object.entries(blockers))if(Array.isArray(value)?value.length>0:value!==0)reasons.push(key);
 return {id:row.id,target,passed:reasons.length===0,reasons};
}
module.exports={state,rejection,actionFailure,acceptance};
