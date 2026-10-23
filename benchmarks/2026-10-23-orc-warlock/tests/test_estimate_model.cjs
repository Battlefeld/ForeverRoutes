'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const {awardXpPolicy,baseQuestXp,estimateAction,estimateNonQuest}=require('../scripts/estimate_model.cjs');
test('ours wins and a verified zero does not become estimated',()=>{
 const world={quests:[{id:1,level:10,xp:700},{id:2,level:8,xp:null},{id:3,level:8,xp:100}]};
 const p=awardXpPolicy(world,{'1':{xp:0},'3':{xp:250}},{estimateMissing:true});
 assert.equal(world.quests[0].xp,0);assert.equal(world.quests[0].noXp,true);
 assert.equal(world.quests[1].xp,baseQuestXp(8));assert.equal(world.quests[1].noXp,false);
 assert.equal(world.quests[2].xp,250);
 assert.deepEqual(p.counts,{foreverroutes:1,otherSource:0,zero:1,estimated:1,strictUnknown:0});
});
test('unknown targets remain zero-reward in strict mode',()=>{
 const world={quests:[{id:9,level:12,xp:null}]};
 const p=awardXpPolicy(world,{}, {estimateMissing:false});
 assert.equal(world.quests[0].noXp,true);assert.equal(p.counts.strictUnknown,1);
});
test('unmapped quest stages are priced exactly once and XP gained at turn-in',()=>{
 const ev={time:0,xpQuest:0,questsDone:0,level:10,xp:0,gain(x){this.xp+=x;}},states=new Map(),trace=[];
 const p={talkSec:6,interactSec:8,killSec1:11,killSec60:30};
 const q={level:10,xp:600,objectives:[{type:'kill',id:1,count:4}]};
 const ctx=event=>estimateAction({event,ev,quest:q,states,params:p,trace});
 ctx({t:'accept',q:93});ctx({t:'complete',q:93});ctx({t:'turnin',q:93});
 const t=ev.time;ctx({t:'turnin',q:93});
 assert.ok(t>200);assert.equal(ev.time,t);
 assert.equal(ev.xp,600);assert.equal(ev.questsDone,1);assert.equal(states.get(93),3);
});
test('non-quest cost estimates do not fabricate XP',()=>{
 const ev={time:0,xp:0};
 assert.equal(estimateNonQuest({t:'hearth'},ev,{hearthCast:10},[]),true);
 assert.equal(ev.time,10);assert.equal(ev.xp,0);
 assert.equal(estimateNonQuest({t:'guide'},ev,{},[]),false);
});
