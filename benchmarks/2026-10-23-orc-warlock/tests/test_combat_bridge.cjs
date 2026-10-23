'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const fs=require('node:fs'),path=require('node:path'),os=require('node:os'),zlib=require('node:zlib'),{spawnSync}=require('node:child_process');
const bridge=require('../scripts/combat/bridge.cjs'),{compare}=require('../scripts/combat/experiment.cjs');
const root=path.resolve(__dirname,'..');
function fight(level=10){return {kind:'fight',player:{class:'Warlock',level,hasSpellbook:true,knownSpellRanks:{shadowBolt:[1]},stats:{health:300,mana:400,baseMana:200,intellect:30,spirit:30,hastePct:0,hitPct:0,critPct:0,spellPower:0,manaRegen:0},equipped:[],talents:[],petActive:false,encounter:{levelDifference:0,armor:0,targetResist:{shadow:0},levelResistance:false,incomingDamage:2,hitEvery:2}},choice:{policies:[{key:'shadowBolt'}]},enemy:{id:1,health:100,level},seed:7,duration:60};}
test('pinned source content matches original ForeverSim Git blobs; tampering is rejected',()=>{
 const before=bridge.verifySnapshot();assert.equal(before.files.length,5);
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'forever-source-integrity-'));
 try{
  fs.copyFileSync(path.join(bridge.snapshot,'manifest.json'),path.join(dir,'manifest.json'));
  const source=JSON.parse(zlib.gunzipSync(fs.readFileSync(path.join(bridge.snapshot,'sources.json.gz'))));
  source['CombatEngine.lua']+='\n-- changed';fs.writeFileSync(path.join(dir,'sources.json.gz'),zlib.gzipSync(JSON.stringify(source)));
  assert.throws(()=>bridge.verifySnapshot(dir),/snapshot changed/);
 }finally{fs.rmSync(dir,{recursive:true,force:true});}
 assert.deepEqual(bridge.verifySnapshot(),before);
});
test('isolated fresh-state results exactly match the original engine for levels 10-30 and paired seeds',()=>{
 for(const level of [10,15,20,25,30])for(const seed of [1,7,991]){
  const event=fight(level);event.seed=seed;
  const result=bridge.run({compareBaseline:true,events:[event]});
  assert.equal(result.rows[0].result.killed,true);assert.equal(result.validated,false);assert.equal(result.rankingEligible,false);
 }
});
test('mana and health persist between targets without shrinking their maxima',()=>{
 const r=bridge.run({events:[fight(),fight()]});
 const [a,b]=r.rows;
 assert.equal(b.startState.mana,a.result.manaEnd);assert.equal(b.startState.health,a.result.healthEnd);
 assert.equal(b.maximum.mana,400);assert.equal(b.maximum.health,300);
 assert.ok(b.result.manaEnd<a.result.manaEnd);assert.ok(b.result.healthEnd<a.result.healthEnd);
});
test('cross-process batches preserve the same resources and fight results as one uninterrupted batch',()=>{
 const both=bridge.run({events:[fight(),fight()]}),first=bridge.run({events:[fight()]});
 const second=bridge.run({initialState:first.state,previousFight:fight(),events:[fight()]});
 assert.deepEqual(second.rows[0].result,both.rows[1].result);
});
test('running out of mana produces a paid failed fight instead of a free refill or successful kill',()=>{
 const r=bridge.run({events:[fight(),fight(),fight()]});
 assert.equal(r.stopped,'target_not_killed');assert.equal(r.rows[2].result.killed,false);assert.equal(r.rows[2].result.manaEnd,0);
 assert.equal(r.rows[2].result.duration,60);assert.ok(r.totalSeconds>=60);
});
test('explicit recovery restores resources and its entire duration remains in the total',()=>{
 const r=bridge.run({events:[fight(),{kind:'idle',seconds:10,recovery:{manaPerSecond:20,healthPerSecond:2}},fight()]});
 assert.equal(r.idleSeconds,10);assert.equal(r.totalSeconds,r.combatSeconds+10);
 assert.equal(r.rows[2].startState.mana,400);assert.equal(r.rows[2].startState.health,300);
});
test('pet mana is carried to the next target and does not get its initial pool for free',()=>{
 const e=fight();e.player.petActive=true;e.player.pet={type:'imp',level:10,mana:200,manaRegen:0,spellPower:0,hasSpellbook:true,knownSpellRanks:{firebolt:[1]}};
 const r=bridge.run({compareBaseline:true,events:[e,e]});
 const a=r.rows[0].result.routeState.actors.pet,b=r.rows[1].result.routeState.actors.pet;
 assert.equal(a.maxMana,200);assert.ok(a.mana<200);assert.ok(b.mana<a.mana);
});
test('cooldowns and the global cooldown survive switching targets',()=>{
 const e=fight(20);e.player.knownSpellRanks.shadowburn=[1];e.player.soulShards=5;
 e.player.talents=[{entries:[{classKey:'shadowburn',name:'Shadowburn',rank:1}]}];
 e.choice={policies:[{key:'shadowburn'},{key:'shadowBolt'}]};e.enemy.health=20;
 const r=bridge.run({events:[e,e]});
 const a=r.rows[0].result,b=r.rows[1].result;
 assert.ok(a.routeState.cooldowns.shadowburn>0);assert.ok(a.routeState.ready>0);
 assert.equal(a.bySpell.shadowburn.casts,1);assert.ok(!b.bySpell.shadowburn||!b.bySpell.shadowburn.casts);
 assert.ok(b.events.find(x=>x.kind==='start').t>=a.routeState.ready);
});
test('unlearned ranks, missing pet data, unknown enemies and invalid recovery never fall back to class factors',()=>{
 const unlearned=fight();unlearned.player.knownSpellRanks={shadowBolt:[99]};assert.throws(()=>bridge.run({events:[unlearned]}),/Nicht erlernter/);
 const pet=fight();pet.player.petActive=true;assert.throws(()=>bridge.run({events:[pet]}),/pet level/);
 const enemy=fight();delete enemy.enemy.health;assert.throws(()=>bridge.run({events:[enemy]}),/enemy health/);
 assert.throws(()=>bridge.run({events:[{kind:'idle',seconds:1,recovery:{manaPerSecond:-1}}]}),/recovery rate/);
 const stats=fight();delete stats.player.stats.baseMana;assert.throws(()=>bridge.run({events:[stats]}),/baseMana/);
});
test('source level gates remain enforced; levels 5-9 are explicitly approximate',()=>{
 assert.throws(()=>bridge.run({events:[fight(1)]}),/1-4 remain blocked/);
 const r=bridge.run({events:[fight(5)]});assert.deepEqual(r.approximateLevels,[5]);assert.equal(r.validated,false);
 const audit=bridge.auditInput({levels:{10:{}}});assert.equal(audit.complete,false);assert.equal(audit.issues.filter(i=>i.reason==='source_level_unsupported').length,4);
});
test('paired rotation experiments retain failed samples and cannot produce a route winner',()=>{
 const plan={job:{events:[fight()]},choices:[{id:'learned',rotation:{policies:[{key:'shadowBolt'}]}},{id:'unlearned',rotation:{policies:[{key:'soulFire'}]}}]};
 const r=compare(plan,{seeds:[1,7]});assert.equal(r.complete,false);assert.equal(r.rows[1].failures.length,2);assert.equal(r.diagnosticFastestChoice,null);assert.equal(r.routeWinner,null);
 const one=compare({...plan,choices:plan.choices.slice(0,1)},{seeds:[1,7]});
 assert.equal(one.complete,true);assert.equal(one.diagnosticFastestChoice,'learned');assert.equal(one.validated,false);
});
test('opt-in route replay stops on missing level-one combat support and preserves partial accounting',()=>{
 const out=fs.mkdtempSync(path.join(os.tmpdir(),'forever-route-combat-'));
 try{
  const input=path.join(out,'input.json');fs.writeFileSync(input,JSON.stringify({schema:'foreverroutes.combat_input.v1',profile:{class:'Warlock',race:'Orc'},levels:{},enemies:{}}));
  const r=spawnSync(process.execPath,[path.join(root,'scripts/run_route_comparison.cjs'),'--saved-inputs','--class','Warlock','--combat-input',input,'--target-level','6','--require-complete','--output-dir',out],{encoding:'utf8',timeout:30000,maxBuffer:4*1024*1024});
  assert.equal(r.status,2,r.stderr);
  const report=JSON.parse(fs.readFileSync(path.join(out,'comparison.json')));
  assert.equal(report.timeModel.kind,'isolated_foreversim_stateful_diagnostic');assert.equal(report.rankingPublished,false);
  for(const row of report.rows){assert.equal(row.endLevel,1);assert.equal(row.timeTo[6],null);assert.equal(row.blockers.combatInputGaps[0].reason,'source_combat_level_unsupported');assert.equal(row.costLedger.validation.closed,true);}
 }finally{fs.rmSync(out,{recursive:true,force:true});}
});
test('the real route work path uses combat results and charges spacing and recovery exactly once',()=>{
 const external=process.env.POLESTAR_ROOT;assert.ok(external,'POLESTAR_ROOT is required');
 const req=p=>require(path.join(external,'tools/lib',p));
 const engine={run:req('sim/run'),xp:req('game/xp')};
 const {Evaluator}=req('sim/evaluate'),{paramsFor}=req('sim/index');
 const world=JSON.parse(zlib.gunzipSync(fs.readFileSync(path.join(root,'.comparison/matrix-worlds/world-horde.json.gz'))));
 const run=engine.run.compileRun(world,{race:'Orc',cls:'Warlock',family:'solo'});
 const params={...paramsFor(path.join(root,'.comparison/cache'),'Warlock').mean,extraKill:0,dropMul:1};
 const tables=engine.run.makeTables(run,params),ev=new Evaluator(run,tables);ev.reset();ev.level=10;
 let task=-1,w=-1;
 for(let i=0;i<run.taskW0.length;i++)if(run.taskW1[i]-run.taskW0[i]===1){
  const k=run.taskW0[i];if(run.wKind[k]===engine.run.W_KILL&&run.wMob[k]&&run.wLevel[k]>=8&&run.wLevel[k]<=10&&run.wCount[k]>=1&&run.wCount[k]<=2&&run.wCount[k]%1===0){task=i;w=k;break;}
 }
 assert.ok(task>=0);tables.killT.fill(NaN);tables.grindRate.fill(NaN);
 const f=fight(),enemy={source:'synthetic software fixture, not measured',health:20,level:run.wLevel[w],elite:run.wElite[w]===1,encounter:f.player.encounter};
 const config={schema:'foreverroutes.combat_input.v1',profile:{class:'Warlock',race:'Orc'},levels:{10:{source:'synthetic software fixture, not measured',player:f.player,choice:f.choice,recoveryBeforeFight:{source:'synthetic software fixture',seconds:2,rates:{manaPerSecond:20,healthPerSecond:2}}}},enemies:{[run.wMob[w]]:enemy}};
 const book=require('../scripts/combat/route_work.cjs').attach(ev,run,config,{engine,profile:config.profile,target:30,seed:7});
 ev.doWork(task);
 assert.ok(Number.isFinite(ev.time));assert.equal(book.fights,run.wCount[w]);
 assert.ok(Math.abs(ev.time-book.combatSeconds-book.spacingSeconds-book.recoverySeconds)<1e-8);
 assert.equal(book.recoverySeconds,Math.max(0,book.fights-1)*2);assert.ok(ev.xpKill>0);
 assert.throws(()=>ev.grindTo(20),e=>e.code==='COMBAT_INPUT_GAP'&&e.detail.reason==='sourced_grind_location_and_enemy_required');
});
test('a player death remains a failed, paid encounter',()=>{
 const e=fight();e.player.stats.health=1;e.player.encounter.incomingDamage=1000;e.player.encounter.hitEvery=.1;
 const r=bridge.run({events:[e,e]});assert.equal(r.stopped,'player_died');assert.equal(r.rows.length,1);assert.ok(r.combatSeconds>0);assert.equal(r.rows[0].result.killed,false);
});
test('collecting combat input requirements leaves route times, XP and failures unchanged',()=>{
 const out=fs.mkdtempSync(path.join(os.tmpdir(),'forever-combat-requests-'));
 try{
  const reports=[];
  for(const collect of [false,true]){
   const dir=path.join(out,String(collect));
   const args=[path.join(root,'scripts/run_route_comparison.cjs'),'--saved-inputs','--class','Warlock','--target-level','6','--output-dir',dir];
   if(collect)args.push('--collect-combat-requirements');
   const r=spawnSync(process.execPath,args,{encoding:'utf8',timeout:30000,maxBuffer:4*1024*1024});assert.equal(r.status,0,r.stderr);
   reports.push(JSON.parse(fs.readFileSync(path.join(dir,'comparison.json'))));
  }
  for(let i=0;i<4;i++){
   const [a,b]=reports.map(r=>r.rows[i]);assert.equal(a.endSeconds,b.endSeconds);assert.equal(a.endXp,b.endXp);assert.deepEqual(a.blockers,b.blockers);
   assert.ok(b.combatRequirements.work.length>0);assert.ok(b.combatRequirements.work.every(w=>w.playerLevelRange[0]>=1));
  }
 }finally{fs.rmSync(out,{recursive:true,force:true});}
});
