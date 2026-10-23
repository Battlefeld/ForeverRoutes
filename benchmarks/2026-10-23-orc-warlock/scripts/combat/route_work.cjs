'use strict';
// Opt-in adapter for the pinned Polestar Evaluator. No external repo is edited.
const bridge=require('./bridge.cjs');
function gap(reason,detail={}){const e=new Error(reason);e.code='COMBAT_INPUT_GAP';e.detail={reason,...detail};throw e;}
function attach(ev,run,config,{engine,profile,target,seed=1,lua}={}){
 if(!Number.isInteger(seed)||seed<0||seed>0xffffffff)throw Error('Explicit uint32 route seed required');
 if(config.schema!=='foreverroutes.combat_input.v1')throw Error('Invalid route combat input schema');
 if(config.profile?.class!==profile.class||config.profile?.race!==profile.race)throw Error('Combat input profile mismatch');
 const {MOBL,W_KILL,W_USE,W_ESCORT,W_BUY}=engine.run,{LEVEL_XP}=engine.xp;
 const book={kind:'isolated_foreversim_stateful_diagnostic',source:bridge.verifySnapshot(),seed,fights:0,combatSeconds:0,recoverySeconds:0,spacingSeconds:0,roundedExpectedKills:[],samples:[],calibrated:false,limitations:['Levels 1-4 unsupported by pinned source','Levels 5-9 use approximated class data','Training, inventory, pet health/threat and observed-time calibration remain required']};
 let state=null,previousFight=null,pendingIdle=0,lastClock=ev.time;
 function battle(mob,L,elite,walk){
  const record=config.levels?.[L],enemy=config.enemies?.[mob];
  if(L<5)gap('source_combat_level_unsupported',{level:L,mob});
  if(!record)gap('missing_recorded_character_profile',{level:L,mob});
  if(!enemy||!enemy.source||!record.source)gap('missing_sourced_enemy_or_character',{level:L,mob});
  if(enemy.elite!==elite)gap('enemy_rank_mismatch',{level:L,mob});
  if(!Number.isFinite(enemy.health)||enemy.health<=0)gap('missing_enemy_health',{mob});
  pendingIdle+=Math.max(0,ev.time-lastClock);
  const player=structuredClone(record.player);
  if(player.level!==L||player.class!==profile.class)gap('character_level_or_class_mismatch',{level:L});
  player.encounter={...enemy.encounter,levelDifference:enemy.level-L};
  const fight={kind:'fight',player,enemy:{id:mob,level:enemy.level,health:enemy.health},choice:record.choice,seed:(seed+Math.imul(book.fights+1,2654435761))>>>0,duration:config.combatHorizonSeconds||300};
  const events=[];
  if(pendingIdle+walk>0)events.push({kind:'idle',seconds:pendingIdle+walk});
  // Recovery is an explicit, paid policy, with finite supplied rates and duration.
  // It is never inferred from a desired completion time or silently inserted.
  if(state&&record.recoveryBeforeFight){
   if(!record.recoveryBeforeFight.source)gap('missing_recovery_evidence',{level:L});
   events.push({kind:'idle',seconds:record.recoveryBeforeFight.seconds,recovery:record.recoveryBeforeFight.rates});
  }
  events.push(fight);
  let result;try{result=bridge.run({events,initialState:state,previousFight},{lua});}catch(e){gap('combat_adapter_rejected_input',{level:L,mob,message:e.message});}
  const row=result.rows.findLast(r=>r.kind==='fight'),duration=row.result.killTime??row.result.deathTime??row.result.duration;
  const recovery=state&&record.recoveryBeforeFight?record.recoveryBeforeFight.seconds:0;
  ev.time+=walk+recovery+duration;
  book.fights++;book.combatSeconds+=duration;book.recoverySeconds+=recovery;book.spacingSeconds+=walk;
  state=result.state;previousFight=fight;pendingIdle=0;lastClock=ev.time;
  if(book.samples.length<100)book.samples.push({mob,level:L,seconds:duration,start:row.startState,end:{mana:row.result.manaEnd,health:row.result.healthEnd},killed:row.result.killed});
  if(result.stopped)gap(result.stopped,{level:L,mob,seconds:duration});
 }
 ev.doWork=function(task){
  const r=this.run,p=this.p,t=this.t;
  for(let w=r.taskW0[task];w<r.taskW1[task]&&!this.finished&&this.level<target;w++){
   const kind=r.wKind[w];
   if(kind===W_KILL){
    let n=r.wCount[w]*(1+p.extraKill);if(r.wDrop[w])n/=p.dropMul;
    const had=this.killed[w],base=had>0||r.wShare0[w]<r.wShare1[w]?this.killsFor(w):0;
    if(had>0){if(had>=base-1e-9)continue;n*=(base-had)/base;}
    if(!r.wMob[w])gap('unknown_enemy_identity',{task,work:w});
    if(Math.abs(n-Math.round(n))>1e-8)book.roundedExpectedKills.push({mob:r.wMob[w],expected:n,charged:Math.ceil(n)});
    if(r.wShare0[w]<r.wShare1[w])this.shareKills(w,base-had);
    const M=r.wLevel[w],elite=r.wElite[w]===1,walk=r.wSpacing[w]*p.spacing/p.runSpeed;
    for(let k=0;k<Math.ceil(n-1e-9)&&!this.finished&&this.level<target;k++){
     const L=this.level;const fact=config.enemies?.[r.wMob[w]];
     if(fact&&fact.level!==M)gap('enemy_level_mismatch',{mob:r.wMob[w],worldLevel:M,inputLevel:fact.level});
     battle(r.wMob[w],L,elite,walk);
     const xp=t.killX[L*MOBL+M]*(elite?2:1);this.xpKill+=xp;this.gain(xp);
    }
   }else if(kind===W_USE){let n=r.wCount[w];if(r.wDrop[w])n/=p.dropMul;this.time+=n*p.interactSec;}
   else if(kind===W_ESCORT)this.time+=p.escortSec;
   else if(kind===W_BUY)this.time+=p.talkSec;
  }
 };
 ev.grindTo=function(level,xpInto=0){if(this.level<level||(this.level===level&&this.xp<xpInto))gap('sourced_grind_location_and_enemy_required',{level:this.level,target:level});};
 return book;
}
module.exports={attach,gap};
