'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const C=require('../scripts/lib/route_contract.cjs');
const make=(events,extra={})=>({schema:'foreverroutes.guide_events.v2',id:'test:route',source:'test',coverage:[1,30],groups:[{step:'original-1',events}],...extra});
test('Class, race and faction are selected rather than fixed to Orc Warlock',()=>{
 const p=C.profile({race:'Human',class:'Mage',faction:'Alliance'}),ctx=C.context(p);
 assert.equal(C.expression('MAGE and HUMAN and ALLIANCE',ctx),true);
 assert.equal(C.expression('WARLOCK or HORDE',ctx),false);
 assert.throws(()=>C.profile({race:'Human',faction:'Horde'}));
});
test('Skyborne race masks retain the bit above 32 bits',()=>{
 const p=C.profile({race:'Skyborne',class:'Mage',faction:'Horde'});
 assert.equal(C.compatible({races:2**33,classes:128},p),true);
 assert.equal(C.compatible({races:2**32,classes:128},p),false);
});
test('Nested TBC guards do not become missing Forever quests',()=>{
 const p=C.profile(),doc=make([{t:'accept',q:9431,condition_format:'journey',conditions:['TBC or WOTLK','HUNTER']}]);
 const r=C.check(doc,p,{});assert.equal(r.issues.length,0);assert.equal(r.counts.selectedEvents,0);
});
test('Optional acceptance excludes the same quest objectives and reward, without excluding required work',()=>{
 const d=make([{t:'accept',q:99,optional:true},{t:'complete',q:99},{t:'turnin',q:99},{t:'accept',q:788}]);
 const r=C.check(d,C.profile(),{788:{xp:170,objectives:[]}});
 assert.equal(r.counts.optionalSkipped,3);assert.equal(r.counts.selectedEvents,1);assert.equal(r.issues.length,0);
});
test('Known zero XP is valid; an unknown reward remains a blocker',()=>{
 const d=make([{t:'turnin',q:1},{t:'turnin',q:2}]);
 const r=C.check(d,C.profile(),{1:{xp:0,objectives:[]},2:{xp:null,objectives:[]}});
 assert.deepEqual(r.issues.filter(x=>x.code==='quest_xp_missing').map(x=>x.quest),[2]);
});
test('Inventory and quest state conditions use the supplied runtime state',()=>{
 const ctx=C.context(C.profile(),{level:12,inventory:{3713:10},questState:q=>q===788?3:0});
 assert.equal(C.expression('ITEM[3713] >= 10 and QUEST[788] == TURNEDIN',ctx),true);
 assert.equal(C.expression('ITEM[999] >= 1',ctx),undefined);
 assert.equal(C.expression('MAGE and ITEM[999] >= 1',ctx),false);
 assert.equal(C.expression('not (TBC or WOTLK)',ctx),true);
});
test('Unresolved route conditions block replay until a state-aware adapter exists',()=>{
 const d=make([{t:'turnin',q:788,conditions:['ITEM[999] >= 1']}]);
 const r=C.check(d,C.profile(),{788:{xp:170,objectives:[]}});
 assert.equal(r.status,'blocked');assert.equal(r.counts.deferredConditions,1);
 assert.ok(r.issues.some(x=>x.code==='dynamic_condition_execution_missing'));
 const negative=C.check(make([{t:'turnin',q:788,skipIfMissing:true}]),C.profile(),{788:{xp:170,objectives:[]}});
 assert.equal(negative.status,'blocked');
});
test('Guide-level version guards are retained as well as step-level guards',()=>{
 const r=C.check(make([{t:'accept',q:9431}],{condition:'TBC or WOTLK'}),C.profile(),{});
 assert.equal(r.status,'not_applicable');assert.equal(r.issues.length,0);
});
test('Friendly dialogue does not require an invented kill count',()=>{
 const d=make([{t:'complete',q:10}]),q={10:{xp:40,objectives:[{type:'kill',id:123,count:null}]}};
 const w={faction:'Horde',excluded:[],quests:[{id:10,tasks:[{work:[['use',1]]}]}]};
 assert.equal(C.check(d,C.profile(),q,w).issues.filter(x=>x.code==='objective_count_missing').length,0);
});
test('Group content is distinguished from absent quest definitions',()=>{
 const d=make([{t:'accept',q:10}]),q={10:{xp:40,objectives:[]}},w={faction:'Horde',quests:[],excluded:[{id:10,category:'dungeon',reason:'instance'}]};
 const r=C.check(d,C.profile(),q,w);assert.ok(r.issues.some(x=>x.code==='content_outside_profile'));assert.ok(!r.issues.some(x=>x.code==='quest_definition_missing'));
});
test('A required unsupported route action is retained and blocks timing',()=>{
 const d=make([{t:'hearth',origin:{step:'original-hearth'}},{t:'unsupported',directive:'new_source_command',origin:{step:'original-new'}}]);
 const r=C.check(d,C.profile(),{});assert.equal(r.status,'blocked');assert.equal(r.timeTo,null);assert.equal(r.issues.length,2);
});
test('Two different routes keep their own quest order and identity',()=>{
 const a=make([{t:'accept',q:788},{t:'accept',q:4641}],{id:'a:original'}),b=make([{t:'accept',q:4641},{t:'accept',q:788}],{id:'b:original'});
 const before=JSON.stringify([a,b]);const q={788:{xp:170},4641:{xp:40}};
 C.check(a,C.profile(),q);C.check(b,C.profile(),q);
 assert.equal(JSON.stringify([a,b]),before);
 assert.deepEqual(a.groups[0].events.map(e=>e.q),[788,4641]);assert.deepEqual(b.groups[0].events.map(e=>e.q),[4641,788]);
});
test('Invalid event and objective references are reported instead of silently dropped',()=>{
 const errors=C.validate(make([{t:'magic_teleport'},{t:'complete',q:788,obj:0}]));
 assert.ok(errors.some(e=>e.code==='event_type'));assert.ok(errors.some(e=>e.code==='objective_index'));
});
