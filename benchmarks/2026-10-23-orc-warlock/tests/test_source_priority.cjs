'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const {ORDER,resolveField,resolveRecord}=require('../scripts/source_priority.cjs');
test('requested exact precedence',()=>assert.deepEqual(ORDER,['foreverroutes','questiedb','pfquest','cmangos','wago_db2','foreverdungeonjournal','warcraftdb','foreverdb','wowhead_forever','estimated']));
test('zero is a real value, not a reason to estimate',()=>{const r=resolveField('xp',[{source:'foreverroutes',value:0},{source:'questiedb',value:150}],{estimate:()=>200});assert.equal(r.value,0);assert.equal(r.source,'foreverroutes');assert.equal(r.conflicts.length,1)});
test('field by field source selection, with explicit estimation',()=>{const r=resolveRecord(['xp','spawn','count'],{xp:[{source:'foreverroutes',value:200}],spawn:[{source:'foreverdb',value:[1,2]}],count:[]},{count:()=>3});assert.deepEqual(r.values,{xp:200,spawn:[1,2],count:3});assert.equal(r.provenance.count.estimated,true)});
test('missing estimate remains visibly unresolved',()=>assert.equal(resolveField('location',[],{}).unresolved,true));
test('lower priority never overwrites a higher priority fact',()=>assert.equal(resolveField('count',[{source:'wowhead_forever',value:4},{source:'questiedb',value:8}]).value,8));
