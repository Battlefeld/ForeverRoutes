'use strict';
// Reintroduce source-defined quests misclassified merely for the word "Test".
// Coordinates and work stay sourced; unmeasured scripted duration stays a blocker.
function reconcileWorldGaps(world,sources){
 const copy=structuredClone(world),added=[],unvalidated=[];
 const addPoint=(map,x,y)=>{
  const f=sources.frames[map];if(!f)throw Error('Missing sourced map frame '+map);
  const p=[f.c,f.c*10,Math.round((f.top+(f.bottom-f.top)*y/100)*10)/10,Math.round((f.left+(f.right-f.left)*x/100)*10)/10,Number(map)];
  const found=copy.points.findIndex(v=>v.every((x,i)=>x===p[i]));if(found>=0)return found;
  copy.points.push(p);return copy.points.length-1;
 };
 for(const raw of Object.values(sources.quests)){
  if(copy.quests.some(q=>q.id===raw.id))continue;
  const excluded=copy.excluded.find(q=>q.id===raw.id);
  if(excluded?.reason!=='unused/test quest')continue;
  if(!raw.start?.npc?.length||!raw.end?.npc?.length||!Number.isFinite(raw.xp))throw Error('Incomplete sourced quest '+raw.id);
  const giver=sources.npcs[raw.start.npc[0]],ender=sources.npcs[raw.end.npc[0]],map=raw.zone;
  const gp=giver?.spawns?.[raw.area]?.[0],ep=ender?.spawns?.[raw.area]?.[0];
  if(!gp||!ep)throw Error('Missing sourced giver/ender '+raw.id);
  const tasks=[];
  for(const objective of raw.objectives||[]){
   if(objective.type==='event'&&objective.map&&Number.isFinite(objective.x)&&Number.isFinite(objective.y)){
    tasks.push({pt:addPoint(objective.map,objective.x,objective.y),work:[['use',1]]});
    unvalidated.push({quest:raw.id,reason:'scripted_event_duration_uses_interaction_prior',event:objective.id});
   }else if(objective.type==='item'&&Number.isInteger(objective.count)&&objective.count>0){
    const item=sources.items[objective.id],mobId=item?.npcDrops?.[0],mob=sources.npcs[mobId],p=mob?.spawns?.[raw.area]?.[0];
    if(!p)throw Error('Missing sourced item drop site '+raw.id);
    const rate=sources.dropRates?.[objective.id+':'+mobId];
    if(!rate||!(rate.chance>0))throw Error('Missing sourced drop probability '+raw.id);
    tasks.push({pt:addPoint(map,p[0],p[1]),work:[['kill',mob.minLevel,objective.count/(rate.chance/100),mob.rank===1?1:0,0,mobId]]});
   }else throw Error('Unsupported sourced objective '+raw.id);
  }
  copy.quests.push({id:raw.id,title:raw.title,level:raw.level,minLevel:raw.minLevel,races:raw.races,classes:raw.classes,pre:raw.preAny||raw.pre||[],preAll:raw.preAll||[],excl:raw.excl||[],next:raw.next,xp:raw.xp,noXp:raw.xp===0,giver:addPoint(map,gp[0],gp[1]),ender:addPoint(map,ep[0],ep[1]),accept:null,tasks,zone:map,breadcrumb:false,escort:false,elite:false,gName:giver.name,eName:ender.name,gPos:{map,x:gp[0],y:gp[1]},ePos:{map,x:ep[0],y:ep[1]},modelUnvalidated:unvalidated.some(x=>x.quest===raw.id),tier:5,preRule:'questie_source_overlay'});
  copy.excluded=copy.excluded.filter(q=>q.id!==raw.id);copy.excludedIds=copy.excludedIds.filter(id=>id!==raw.id);added.push(raw.id);
 }
 return {world:copy,report:{added,unvalidated,provenance:sources.provenance}};
}
module.exports={reconcileWorldGaps};
