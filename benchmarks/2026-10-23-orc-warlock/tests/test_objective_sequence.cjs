'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const fs=require('node:fs'),path=require('node:path'),zlib=require('node:zlib');
const {prepareObjectiveSites,compileEvent}=require('../scripts/objective_sequence.cjs');
const root=path.resolve(__dirname,'..');
const world=JSON.parse(fs.readFileSync(path.join(root,'.comparison/cache/world-horde.json')));
const facts=JSON.parse(zlib.gunzipSync(fs.readFileSync(path.resolve(root,'../2026-10-23-independent-route-checks/data/quests.json.gz'))));
const refs=JSON.parse(fs.readFileSync(path.join(root,'data/objective-drop-refs.json'))).items;
prepareObjectiveSites(world,facts,refs);
test('mixed-site kill objectives select one original workload, not the whole quest',()=>{
 const quest=world.quests.find(q=>q.id===784);
 assert.deepEqual(quest.objectiveTasks,[0,1,2]);
 const run={qIndex:new Map([[784,0]]),quests:[quest],qTask0:[50],taskPt:Array.from({length:60},(_,i)=>i+100)};
 const seq=compileEvent(run,{t:'complete',q:784,obj:2},()=>{throw Error('must not compile whole quest');},7);
 assert.deepEqual(seq.task,[51]);assert.deepEqual(seq.kind,[7]);assert.equal(quest.tasks[1].work[0][5],3129);
});
test('object objectives recover the pinned builder site grouping with source-objective order',()=>{
 const q=world.quests.find(q=>q.id===99049);
 assert.deepEqual(q.objectiveTasks,[0,2,1]);
 assert.equal(q.tasks[q.objectiveTasks[0]].pt,q.tasks[q.objectiveTasks[2]].pt);
 assert.notEqual(q.tasks[q.objectiveTasks[0]].pt,q.tasks[q.objectiveTasks[1]].pt);
});
test('three item objectives from the same mob keep their distinct ordered work entries',()=>{
 const q=world.quests.find(q=>q.id===846);
 assert.deepEqual(q.objectiveTasks,[0,1,2]);assert.equal(q.tasks.length,3);
});
test('an absent objective definition remains an explicit mapping failure',()=>{
 const q=world.quests.find(q=>q.id===97891);
 const seq=compileEvent({qIndex:new Map([[q.id,0]]),quests:[q]},{t:'complete',q:q.id,obj:1},()=>{},7);
 assert.equal(seq.unmappedObjective,true);assert.equal(seq.kind.length,0);
});
