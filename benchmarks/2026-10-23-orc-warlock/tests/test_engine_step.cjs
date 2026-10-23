'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const {stepEvent}=require('../scripts/engine_step.cjs');
test('accepting a legal red quest does not grind to its displayed quest level',()=>{
 const p={dangerGap:5},run={qAccept:[-1],qMinLevel:[26],qLevel:[37]};
 const ev={level:27,p,step(){assert.equal(this.p.dangerGap,10);}};
 stepEvent(ev,run,1,0,-1,0,1);assert.equal(ev.p,p);
});
test('minimum level and quest-starting-item combat retain the original engine rules',()=>{
 const p={dangerGap:5},run={qAccept:[0],qMinLevel:[26],qLevel:[37]};
 const ev={level:27,p,step(){assert.equal(this.p,p);}};
 stepEvent(ev,run,1,0,0,0,1);
 run.qAccept[0]=-1;ev.level=25;stepEvent(ev,run,1,0,-1,0,1);
});
