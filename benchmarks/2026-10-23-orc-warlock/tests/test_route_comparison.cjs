'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const fs=require('node:fs'),path=require('node:path');
const {rxpCondition,journeyCondition,applyXpPolicy,effectiveQuestXp}=require('../scripts/run_route_comparison.cjs');
test('RXP class/race gates keep the Orc Warlock branch and reject the other classes',()=>{
 assert.equal(rxpCondition('Warrior/Shaman/Warlock/Mage/Priest'),true);
 assert.equal(rxpCondition('!Warrior !Rogue !Shaman'),true);
 assert.equal(rxpCondition('!Warlock'),false);
 assert.equal(rxpCondition('Orc Hunter/Troll Hunter'),false);
 assert.equal(rxpCondition('NotImplementedGate'),null);
});
test('Journey conditions use quest state and do not guess unknown inventory',()=>{
 const run={qIndex:new Map([[788,0]]),qTask0:[0],qTask1:[1]};
 const ev={done:[0],active:[1],taskDone:[0],level:4};
 assert.equal(journeyCondition('WARLOCK and QUEST[788] >= ACCEPTED',ev,run),true);
 assert.equal(journeyCondition('QUEST[788] >= COMPLETED',ev,run),false);
 ev.taskDone[0]=1;
 assert.equal(journeyCondition('QUEST[788] >= COMPLETED',ev,run),true);
 assert.equal(journeyCondition('not ROGUE and H12B',ev,run),true);
 assert.equal(journeyCondition('ITEM[3713] >= 10',ev,run),null);
 assert.equal(journeyCondition('MAGE and ITEM[3713] >= 10',ev,run),false);
});
test('Explicit zero XP becomes the engine no-reward sentinel; positive rewards survive',()=>{
 const w={quests:[{id:172,xp:null},{id:788,xp:999},{id:999999,xp:999}]};
 applyXpPolicy(w,{172:{xp:0},788:{xp:170}});
 assert.deepEqual(w.quests.map(q=>[q.xp,q.noXp]),[[0,true],[170,false],[null,true]]);
});
test('All imported zero-XP quests actually award zero in the shared combat/travel model',()=>{
 const root=path.resolve(__dirname,'..'),engine=path.resolve(process.env.POLESTAR_ROOT || path.resolve(root,'../polestar'));
 const w=JSON.parse(fs.readFileSync(path.join(root,'.comparison/cache/world-horde.json')));
 const raw=JSON.parse(fs.readFileSync(path.join(root,'.comparison/data/quests.json')));
 applyXpPolicy(w,raw);
 const sim=require(path.join(engine,'tools/lib/sim'));
 const params=sim.paramsFor(path.join(root,'.comparison/cache'),'Warlock').mean;
 const {run,tables}=sim.makeRun(w,{race:'Orc',cls:'Warlock',params,family:'solo'});
 let checked=0;
 for(let i=0;i<run.Q;i++)if(raw[run.quests[i].id].xp===0){assert.equal(tables.qFull[i],0);checked++;}
 assert.ok(checked>10,'expected multiple real zero-XP quests in the playable Horde world');
});
test('Diagnostic comparisons never invent a 1-30 time for guides that end earlier',()=>{
 const d=JSON.parse(fs.readFileSync(path.resolve(__dirname,'../.comparison/results/comparison.json')));
 assert.equal(d.rankingPublished,false);assert.equal(d.rows.length,4);
 for(const id of ['rxp','wisp'])assert.equal(d.rows.find(r=>r.id===id).timeTo['30'],null);
 for(const r of d.rows){assert.match(r.status,/diagnostic/);assert.ok(r.trace.length>20);for(const t of r.trace)assert.ok(t.time<=r.endSeconds);}
});

test('Source-verified and user-approved zero XP fill only missing rewards for every route',()=>{
 const base={172:{id:172,xp:null},308:{id:308,xp:null},788:{id:788,xp:170},901:{id:901,xp:0},456:{id:456,xp:null}};
 const fallback={entries:[{quest_id:172,xp:0},{quest_id:308,xp:0},{quest_id:788,xp:0},{quest_id:901,xp:0},{quest_id:999,xp:0}]};
 const verified={quests:{308:{fields:{xp:15}},456:{fields:{xp:0}},788:{fields:{xp:99}}}};
 const {records,applied}=effectiveQuestXp(base,fallback,verified);
 assert.equal(records[172].xp,0);
 assert.equal(records[172].xp_status,'assumed_zero_missing_source');
 assert.equal(records[308].xp,15);
 assert.equal(records[308].xp_status,'verified_reference');
 assert.equal(records[456].xp,0);
 assert.equal(records[788].xp,170,'known Forever XP outranks references');
 assert.equal(records[901].xp,0);
 assert.equal(records[999],undefined,'do not manufacture missing quests');
 assert.equal(base[172].xp,null,'never modify source records');
 assert.deepEqual(applied.verified.sort((a,b)=>a-b),[308,456]);
 assert.deepEqual(applied.assumedZero,[172]);
});
