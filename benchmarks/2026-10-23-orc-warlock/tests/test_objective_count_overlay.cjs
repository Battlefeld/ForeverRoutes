'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const {reconcile,loadInputs}=require('../scripts/objective_count_overlay.cjs');
const src={97242:{objectives:[{type:'item',id:277278,count:2,countProvenance:{source:'guide_reference'}},{type:'item',id:277279,count:4,countProvenance:{source:'guide_reference'}}]},96655:{objectives:[{type:'kill',id:3191,count:null}]}};
const world={quests:[{id:97242,tasks:[{pt:1,work:[['use',5/3],['use',5/3]]}]},{id:96655,tasks:[{pt:1,work:[['use',1]]}]}]};
const reference={entries:[{quest:96655,objective:1,type:'kill',target:3191,count:1,url:'https://www.wowhead.com/forever/quest=96655'}]};
test('Source counts scale expected item interaction work and do not touch world or guide inputs',()=>{
 const original=JSON.stringify(world),q=JSON.stringify(src);
 const {world:merged,missing,report}=reconcile(world,src,{filled:[]},reference,new Set([97242,96655]));
 assert.ok(Math.abs(merged.quests[0].tasks[0].work[0][1]-10/3)<1e-9);
 assert.ok(Math.abs(merged.quests[0].tasks[0].work[1][1]-20/3)<1e-9);
 assert.equal(report.scaled.length,2);
 assert.equal(report.newEvidence.length,1);
 assert.equal(missing.size,0);
 assert.equal(JSON.stringify(src),q);assert.equal(JSON.stringify(world),original);
});
test('Ambiguous grouping cannot be priced as a made-up count',()=>{
 const awkward={quests:[{id:97242,tasks:[{pt:1,work:[['use',5/3]]},{pt:2,work:[['use',5/3],['use',1]]}]}]};
 const r=reconcile(awkward,src,{filled:[]},reference,new Set([97242]));
 assert.ok(r.missing.has(97242));
 assert.equal(r.report.scaled.length,0);
});
test('Contradictory or wrong-target references are rejected instead of replacing values',()=>{
 const ref={entries:[{quest:97242,objective:1,type:'item',target:277278,count:8,url:'https://www.wowhead.com/forever/quest=97242'},{quest:96655,objective:1,type:'kill',target:999,count:1,url:'https://www.wowhead.com/forever/quest=96655'}]};
 const r=reconcile(world,src,{filled:[]},ref,new Set([96655]));
 assert.equal(r.report.conflicts.length,2);assert.ok(r.missing.has(96655));
});
test('Dialogue target is never simulated as an NPC kill',()=>{
 const q={5727:{objectives:[{type:'kill',id:3216,count:null}]}};
 const w={quests:[{id:5727,tasks:[{pt:10,work:[['kill',37,1,0,0,3216]]}]}]};
 const ref={entries:[{quest:5727,objective:1,type:'kill',target:3216,count:1,url:'https://www.wowhead.com/classic/quest=5727',action:'dialogue'}]};
 const r=reconcile(w,q,{filled:[]},ref,new Set([5727]));
 assert.deepEqual(r.world.quests[0].tasks[0].work[0],['use',1]);
 assert.equal(r.missing.size,0);
 assert.equal(r.report.semanticCorrections[0].quest,5727);
});
test('Recorded original counts, positive or missing, match the original real quest facts',()=>{
 const {quests,guide,references}=loadInputs();
 assert.equal(quests[97242].objectives[0].count,2);
 assert.equal(quests[97242].objectives[1].count,4);
 assert.ok(guide.filled.some(x=>x.quest===97242&&x.count===4));
 assert.ok(references.entries.some(x=>x.quest===96604&&x.objective===2));
});
