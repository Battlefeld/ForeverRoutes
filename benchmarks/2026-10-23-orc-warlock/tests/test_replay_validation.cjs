'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const {actionFailure,acceptance,rejection}=require('../scripts/replay_validation.cjs');
const {continuation}=require('../scripts/style_extension.cjs');
function fixture(){
 const run={Q:3,qIndex:new Map([[10,0],[20,1],[30,2]]),quests:[{id:10,xp:100,level:1,minLevel:1,zone:1,tasks:[]},{id:20,xp:200,level:1,minLevel:1,zone:2,tasks:[]},{id:30,xp:50,level:1,minLevel:1,zone:1,tasks:[]}],qTask0:[0,0,0],qTask1:[0,0,0],qParent:[-1,-1,-1],qPre0:[0,0,1],qPre1:[0,1,1],preList:[0],qAll0:[0,0,0],qAll1:[0,0,0],allList:[],qExcl0:[0,0,0],qExcl1:[0,0,0],exclList:[],qMinLevel:[1,1,1],qLevel:[1,1,1],qGiver:[0,1,2],pX:[0,0,100],pY:[0,0,100],logCap:20};
 const ev={level:2,done:[0,0,0],active:[0,0,0],taskDone:[],p:{dangerGap:3},open:0,pos:0,prereqOk(i){return i!==1||!!this.done[0];}};
 return {run,ev};
}
test('failed complete and turnin are detected, not just failed accepts',()=>{
 const {run,ev}=fixture();
 for(const t of ['accept','complete','turnin'])assert.equal(actionFailure(run,ev,{t,q:10}).type,'failed_'+t);
 ev.active[0]=1;
 assert.equal(actionFailure(run,ev,{t:'complete',q:10}),null);
 assert.equal(actionFailure(run,ev,{t:'turnin',q:10}).type,'failed_turnin');
 ev.done[0]=1;assert.equal(actionFailure(run,ev,{t:'turnin',q:10}),null);
});
test('rejection names the actual unmet engine prerequisite',()=>{
 const {run,ev}=fixture();
 assert.deepEqual(rejection(run,ev,20).missingAny,[10]);
 assert.equal(rejection(run,ev,20).reason,'prerequisites_not_turned_in');
 ev.done[0]=1;assert.deepEqual(rejection(run,ev,20).missingAny,[]);
});
test('acceptance requires target, coverage, milestone, XP and every event',()=>{
 const row={id:'x',endLevel:30,declaredCoverage:[1,30],timeTo:{30:100},blockers:{eventFailures:[],unknownQuestIds:[],unknownXpIds:[],unreachable:0}};
 assert.equal(acceptance(row,30).passed,true);
 for(const key of ['eventFailures','unknownXpIds']){
  row.blockers[key].push(123);assert.equal(acceptance(row,30).passed,false);row.blockers[key].pop();
 }
 row.endLevel=29;assert.ok(acceptance(row,30).reasons.includes('target_not_reached'));
 row.endLevel=30;row.declaredCoverage=[1,22];assert.ok(acceptance(row,30).reasons.includes('source_coverage_short'));
 row.declaredCoverage=[1,30];row.timeTo[30]=null;assert.ok(acceptance(row,30).reasons.includes('milestone_time_missing'));
});
test('continuation checks live prerequisites and never mutates original groups',()=>{
 const {run,ev}=fixture(),source={id:'rxp',groups:[]},before=JSON.stringify(source);
 const iter=continuation(source,run,ev,{target:30,missingCounts:new Set([30])});
 assert.equal(iter.next().value.events[0].q,10);
 ev.done[0]=1;
 assert.equal(iter.next().value.events[0].q,20);
 ev.done[1]=1;
 assert.equal(iter.next().done,true);
 assert.equal(JSON.stringify(source),before);
});
test('continuation never duplicates an original quest or manufactures level XP',()=>{
 const {run,ev}=fixture(),source={id:'wisp',groups:[{events:[{t:'accept',q:10}]}]};
 const groups=[...continuation(source,run,ev,{target:30,missingCounts:new Set()})];
 assert.deepEqual(groups.map(g=>g.events[0].q),[30]);
 assert.ok(groups.every(g=>g.events.every(e=>['accept','complete','turnin'].includes(e.t))));
});
