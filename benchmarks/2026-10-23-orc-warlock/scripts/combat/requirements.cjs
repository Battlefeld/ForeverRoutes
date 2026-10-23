'use strict';
// Observe the legacy replay to collect an input request, without changing time or XP.
function collect(ev,run,W_KILL){
 const book={schema:'foreverroutes.combat_requirements.v1',basis:'legacy diagnostic replay; level ranges are conservative, not observed play',origin:'source',work:[],grind:[],requiredPlayerFields:['level','class','learned spell ranks','talents','equipment','health','mana','baseMana','intellect','spirit','spellPower','hastePct','hitPct','critPct','manaRegen','pet stats and spellbook','source evidence'],requiredEnemyFields:['id','level','elite','health','armor','resistances','incoming damage','swing interval','source evidence']};
 const nativeWork=ev.doWork,nativeGrind=ev.grindTo;
 ev.doWork=function(task){
  const before=this.level;const result=nativeWork.call(this,task);
  for(let w=run.taskW0[task];w<run.taskW1[task];w++)if(run.wKind[w]===W_KILL)book.work.push({origin:book.origin,task,mob:run.wMob[w]||null,mobLevel:run.wLevel[w],elite:run.wElite[w]===1,playerLevelRange:[before,this.level],expectedBaseKills:run.wCount[w],dropWork:!!run.wDrop[w]});
  return result;
 };
 ev.grindTo=function(level,xpInto=0){
  const before=this.level,xp=this.xp,result=nativeGrind.call(this,level,xpInto);
  if(this.level!==before||this.xp!==xp)book.grind.push({origin:book.origin,position:this.pos,from:before,to:this.level,request:{level,xpInto},needs:'sourced reachable grind site and enemy profile'});
  return result;
 };
 return book;
}
module.exports={collect};
