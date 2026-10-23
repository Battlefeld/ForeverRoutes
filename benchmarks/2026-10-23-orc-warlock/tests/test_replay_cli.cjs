'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const fs=require('node:fs'),path=require('node:path'),os=require('node:os'),crypto=require('node:crypto');
const {spawnSync}=require('node:child_process');
const root=path.resolve(__dirname,'..'),script=path.join(root,'scripts/run_route_comparison.cjs');
const digest=f=>crypto.createHash('sha256').update(fs.readFileSync(f)).digest('hex');
function invoke(args){
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'foreverroutes-test-'));
 const result=spawnSync(process.execPath,[script,'--saved-inputs','--output-dir',dir,...args],{encoding:'utf8',timeout:60000});
 return {dir,result,read(name){return JSON.parse(fs.readFileSync(path.join(dir,name)));},close(){fs.rmSync(dir,{recursive:true,force:true});}};
}
test('four deterministic model continuations reach 30 without changing any source export',()=>{
 const files=['rxp','journey','wisp','polestar'].map(id=>path.join(root,'.comparison/events',id+'.json'));
 const before=files.map(digest),a=invoke(['--extend-to-30','--require-complete']);
 try{
  assert.equal(a.result.status,2,'unresolved source actions must fail the acceptance gate');
  const d=a.read('comparison.json');assert.equal(d.profile.class,'Warlock');assert.equal(d.rows.length,4);
  for(const row of d.rows){assert.ok(row.endLevel>=30);assert.ok(row.timeTo[30]>0);assert.equal(row.blockers.eventFailures.filter(f=>f.generated).length,0);}
  assert.equal(d.acceptancePassed,false);assert.equal(d.rankingPublished,false);
  const wisp=d.rows.find(r=>r.id==='wisp'),polestar=d.rows.find(r=>r.id==='polestar');
  assert.equal(wisp.blockers.eventFailures.length,0);assert.equal(wisp.blockers.sourceActionsNotReplayed.length,0);assert.equal(wisp.blockers.unreachable,0);
  assert.equal(polestar.blockers.eventFailures.length,0);assert.equal(polestar.blockers.unknownQuestIds.length,0);assert.equal(polestar.blockers.unknownXpIds.length,0);
  assert.ok(polestar.blockers.unvalidatedMechanics.length>0,'scripted events must not silently pass');
  assert.deepEqual(files.map(digest),before);
  const b=invoke(['--extend-to-30']);
  try{assert.equal(b.result.status,0);assert.deepEqual(b.read('comparison.json').rows.map(r=>r.timeTo),d.rows.map(r=>r.timeTo));}finally{b.close();}
 }finally{a.close();}
});
test('an unsupported Warrior profile cannot silently reuse the Warlock chain',()=>{
 const a=invoke(['--warrior']);try{assert.equal(a.result.status,1);assert.match(a.read('status.json').error,/Warlock/);}finally{a.close();}
});
test('target-level 20 emits only reached milestones and never generates an unnecessary extension',()=>{
 const a=invoke(['--target-level','20','--extend-to-30']);
 try{assert.equal(a.result.status,0);for(const r of a.read('comparison.json').rows){assert.ok(r.endLevel>=20);assert.equal(r.timeTo[30],undefined);assert.equal(r.generatedContinuation.length,0);}}finally{a.close();}
});
