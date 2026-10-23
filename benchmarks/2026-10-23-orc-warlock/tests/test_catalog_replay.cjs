'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const fs=require('node:fs'),path=require('node:path');
const adapter=require('../scripts/catalog_replay.cjs');
const {recoverAccepts}=require('../scripts/quest_recovery.cjs');
const {reconcileWorldGaps}=require('../scripts/world_gap_overlay.cjs');
const root=path.resolve(__dirname,'..');
const read=p=>JSON.parse(fs.readFileSync(p));
const sources=adapter.loadCatalogSources(['rxp','journey','wisp'].map(id=>read(path.join(root,'.comparison/events',id+'.json'))),path.resolve(root,'../2026-10-23-independent-route-checks/events/catalog.json.gz'));
const qstate=()=>0,run={},ev={level:14,xp:0};
test('the Tauren Shaman quest guard does not suppress the Warlock Plainstrider acceptance',()=>{
 const g=sources[0].groups.find(g=>g.guide==='12-17 The Barrens'&&g.step===42);
 assert.equal(adapter.groupAllowed(g,run,ev,qstate,[]),true);
 assert.ok(g.events.some(e=>e.t==='accept'&&e.q===845));
});
test('RXP XP skipstep thresholds run before the entire quest step',()=>{
 const e={args:'<15,1'};
 assert.equal(adapter.levelGuard(e,{level:14,xp:0},[]),false);
 assert.equal(adapter.levelGuard(e,{level:15,xp:0},[]),true);
 assert.equal(adapter.levelGuard({args:'>15,1'},{level:14,xp:0},[]),true);
 assert.equal(adapter.levelGuard({args:'>15,1'},{level:15,xp:0},[]),false);
});
test('nested Journey expansion conditions exclude TBC-only quests on Forever',()=>{
 const e=sources[1].groups.flatMap(g=>g.events).find(e=>e.q===9431&&e.t==='accept');
 assert.ok(e);assert.equal(adapter.allowed(e,run,ev,qstate),false);
 assert.equal(adapter.allowed({conditions:['not HC and not INGROUP'],condition_format:'journey'},run,ev,qstate),true);
});
test('an optional Shaman variant does not make the mandatory Warlock quest optional',()=>{
 assert.equal(sources[1].optionalQuests.includes(819),false);
 const e=sources[1].groups.flatMap(g=>g.events).find(e=>e.q===819&&e.t==='accept'&&!e.optional);
 assert.equal(adapter.optional(e,sources[1]),false);
});
test('Wisp optional profession and dungeon chains keep their source metadata',()=>{
 for(const id of [97905,5728]){
  const e=sources[2].groups.flatMap(g=>g.events).find(e=>e.q===id&&e.t==='accept');
  assert.equal(e.optional,true);assert.equal(adapter.optional(e,sources[2]),true);
  assert.equal(adapter.optional(e,sources[2],true),false);
 }
 const dungeon=sources[2].groups.flatMap(g=>g.events).find(e=>e.q===5728);assert.equal(dungeon.trip.groupSize,5);
});
test('deferred accepts recover only at a real giver visit, with satisfied prerequisites and charged work',()=>{
 const failure={quest:3090},pending=[{event:{t:'accept',q:3090},failure}];
 const r={qIndex:new Map([[3090,0]]),qGiver:[3],qMinLevel:[1],qLevel:[1],logCap:20};
 const e={pos:1,done:[0],active:[0],open:0,level:2,p:{dangerGap:5},time:0,prereqOk:()=>false,step(){this.active[0]=1;this.time+=6;}};
 const compile=()=>({kind:[0],q:[0],task:[-1],pt:[3]});
 assert.deepEqual(recoverAccepts(pending,r,e,compile),[]);
 e.pos=3;assert.deepEqual(recoverAccepts(pending,r,e,compile),[]);
 e.prereqOk=()=>true;const out=recoverAccepts(pending,r,e,compile);
 assert.equal(out.length,1);assert.equal(out[0].failure,failure);assert.equal(e.time,6);assert.equal(e.active[0],1);
});
test('real Test quests are restored from pinned sources without mutating the world snapshot',()=>{
 const world=read(path.join(root,'.comparison/cache/world-horde.json')),pack=read(path.join(root,'data/world-gap-sources.json'));
 const before=JSON.stringify(world),out=reconcileWorldGaps(world,pack);
 assert.deepEqual(out.report.added,[1149,1150]);assert.equal(JSON.stringify(world),before);
 assert.equal(out.world.quests.find(q=>q.id===1150).tasks[0].work[0][2],1,'sourced 100% drop, one claw');
 assert.deepEqual(out.world.quests.find(q=>q.id===1150).pre,[1149]);
 assert.ok(out.report.unvalidated.some(x=>x.quest===1149),'scripted jump duration must stay unvalidated');
 assert.ok(out.world.excluded.some(q=>q.id===1),'unused quests remain excluded');
});
test('an unknown inventory guard blocks the whole RXP step instead of attempting its turn-in',()=>{
 const g=sources[0].groups.find(g=>g.guide==='12-14 Silverpine Forest'&&g.step===110);
 assert.equal(adapter.groupAllowed(g,run,ev,qstate,[]),null);
 assert.equal(adapter.groupAllowed(g,run,{...ev,inventory:{210177:0}},qstate,[]),false);
 assert.equal(adapter.groupAllowed(g,run,{...ev,inventory:{210177:1}},qstate,[]),true);
});
