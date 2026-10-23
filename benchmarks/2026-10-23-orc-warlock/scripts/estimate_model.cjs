'use strict';
// Modelled mode: same ordered events, same fixed cost priors for all guides.
// Our fully sourced values win; guesses live only in replay memory and
// are enumerated in the result. This DOES NOT alter original guide files.
const {resolveField,ORDER}=require('./source_priority.cjs');
const ANCHORS=[[1,100],[5,400],[10,800],[15,1200],[20,1700],[25,2200],[30,2700],[35,3250],[40,3800],[45,4400],[50,5100],[55,5800],[60,6600]];
function baseQuestXp(level){
 const x=Math.max(1,Math.min(60,Number(level)||1));
 for(let i=1;i<ANCHORS.length;i++)if(x<=ANCHORS[i][0]){
  const [l,a]=ANCHORS[i-1],[r,b]=ANCHORS[i];
  return Math.round(a+(b-a)*(x-l)/(r-l));
 }
 return 6600;
}
function awardXpPolicy(world,records,{estimateMissing=false,provenance={}}={}){
 const report={priority:ORDER,counts:{foreverroutes:0,otherSource:0,zero:0,estimated:0,strictUnknown:0},estimated:[],nonrewarding:[]};
 for(const q of world.quests||[]){
  const r=records[q.id],source=provenance[q.id]?.source||'foreverroutes';
  // A known zero must never become guessed XP.
  const known=Number.isFinite(r?.xp)&&r.xp>=0;
  const result=resolveField('xp',[
   ...(known?[{source:'foreverroutes',value:r.xp}]:[]),
   ...(q.xp>0?[{source:'questiedb',value:q.xp}]:[])
  ],{estimate:estimateMissing?()=>q.noXp?0:baseQuestXp(q.level||q.minLevel):undefined});
  if(result.value!==null){
   q.xp=result.value;q.noXp=result.value===0;
   if(result.source==='estimated'){report.counts.estimated++;report.estimated.push({quest:q.id,xp:q.xp,source:'polestar_classic_xp_anchors'});}
   else if(result.value===0){report.counts.zero++;report.nonrewarding.push({quest:q.id,confirmedZero:true});}
   else report.counts[result.source==='foreverroutes'?'foreverroutes':'otherSource']++;
  }else if(q.noXp){q.xp=0;report.counts.zero++;}
  else{q.xp=null;q.noXp=true;report.counts.strictUnknown++;}
 }
 return report;
}
function targetCount(o){const n=Number(o?.count);return n>0&&Number.isFinite(n)?n:1;}
function estimateAction({event,ev,quest,states,params,trace,variant=1}){
 const type=event.t,id=Number(event.q);
 const stage=states.get(id)||0;
 const level=Math.max(1,Math.min(60,quest?.level||ev.level));
 const talk=Math.max(4,params.talkSec||6);
 const interact=Math.max(5,params.interactSec||8);
 const killSec=Math.max(8,(params.killSec1||11)+(level-1)*((params.killSec60||30)-(params.killSec1||11))/59);
 const adds=(seconds,reason)=>{const sec=Math.max(0,seconds*variant);ev.time+=sec;trace.push({quest:id,action:type,reason,seconds:sec,source:'estimated'});};
 if(type==='accept'){
  if(stage<1){adds(65+talk,'unknown_quest_giver_location');states.set(id,1);}
  return;
 }
 if(type==='complete'||type==='turnin'){
  if(stage<1){adds(65+talk,'implicit_accept_missing_quest');states.set(id,1);}
  if((states.get(id)||0)<2){
   let work=0,objectiveCount=0;
   for(const o of quest?.objectives||[]){
    objectiveCount++;
    const n=targetCount(o);
    if(o.type==='kill')work+=n*killSec;
    else if(o.type==='item')work+=n*killSec/(o.questItem?0.6:0.3);
    else if(o.type==='object')work+=n*interact;
    else work+=interact;
   }
   if(!objectiveCount)work=interact;
   adds(110+work,'unknown_objective_site_or_dungeon');
   states.set(id,2);
  }
  if(type==='turnin'&&(states.get(id)||0)<3){
   adds(65+talk,'unknown_quest_ender_location');
   const reward=Number.isFinite(quest?.xp)&&quest.xp>=0?quest.xp:baseQuestXp(level);
   // A guestimated reward is not a verified game value.
   const actualXp=Math.round(reward*variant);
   if(actualXp>0){ev.xpQuest+=actualXp;ev.gain(actualXp);}
   ev.questsDone++;
   trace.push({quest:id,action:'turnin',xp:actualXp,source:Number.isFinite(quest?.xp)?'cached_or_merged':'estimated_quest_xp'});
   states.set(id,3);
  }
 }
}
function estimateNonQuest(event,ev,params,trace,variant=1){
 const costs={hearth:params.hearthCast||10,deathskip:120,trainer:45,vendor:35,flight:90,taxi:90,boat:120,transport:120,walk:90,travel:90,repair:35};
 const key=String(event.t).toLowerCase();
 if(!Object.prototype.hasOwnProperty.call(costs,key))return false;
 const s=Math.max(0,costs[key]*variant);
 ev.time+=s;trace.push({action:key,seconds:s,source:'estimated_nonquest_action'});
 return true;
}
module.exports={baseQuestXp,awardXpPolicy,targetCount,estimateAction,estimateNonQuest};
