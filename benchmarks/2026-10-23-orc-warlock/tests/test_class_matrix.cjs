'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const fs=require('node:fs'),path=require('node:path'),os=require('node:os'),{spawnSync}=require('node:child_process');
const {CLASSES,resolveProfile}=require('../scripts/class_profiles.cjs');
const {planProfileSources}=require('../scripts/class_sources.cjs');
const {loadCatalogSources,allowed}=require('../scripts/catalog_replay.cjs');
const {accounting,rankingEligibility}=require('../scripts/time_evidence.cjs');
const ledger=require('../scripts/time_ledger.cjs');
const root=path.resolve(__dirname,'..'),catalog=path.resolve(root,'../2026-10-23-independent-route-checks/events/catalog.json.gz');
test('all nine reference classes have playable, faction-consistent profiles',()=>{
 assert.equal(CLASSES.length,9);assert.equal(new Set(CLASSES.map(c=>resolveProfile({class:c}).classBit)).size,9);
 assert.equal(resolveProfile({class:'Paladin'}).faction,'Alliance');
 assert.equal(resolveProfile({class:'Druid'}).race,'Tauren');
 assert.throws(()=>resolveProfile({class:'Paladin',race:'Orc'}));
 assert.throws(()=>resolveProfile({class:'Mage',race:'Troll',faction:'Alliance'}));
 assert.throws(()=>resolveProfile({class:'DeathKnight'}));
});
test('class-specific RXP next chains are selected instead of rescaling a Warlock chain',()=>{
 const select=cls=>planProfileSources(catalog,resolveProfile({class:cls})).find(s=>s.id==='rxp');
 assert.ok(select('Warlock').groups.some(g=>g.guide==='12-14 Silverpine Forest'));
 for(const cls of ['Hunter','Shaman'])assert.ok(!select(cls).groups.some(g=>g.guide==='12-14 Silverpine Forest'));
 assert.ok(select('Paladin').groups[0].guide.includes('Northshire'));
 assert.ok(select('Druid').groups[0].guide.includes('Mulgore'));
});
test('route selection keeps missing original 20-30 chapters explicit for every class',()=>{
 for(const cls of CLASSES){
  const sources=planProfileSources(catalog,resolveProfile({class:cls}));
  assert.deepEqual(sources.map(s=>s.id),['rxp','journey','wisp','polestar']);
  for(const id of ['rxp','wisp'])assert.ok(sources.find(s=>s.id===id).selectionIssues.length>0);
  for(const id of ['journey','polestar'])assert.equal(sources.find(s=>s.id===id).selectionIssues.length,0);
 }
});
test('Polestar chapter boundary does not truncate a target-level replay early',()=>{
 const s=planProfileSources(catalog,resolveProfile({class:'Warrior'}),20).find(s=>s.id==='polestar');
 assert.ok(s.groups.some(g=>g.guide==='forever/horde/04-the-barrens'));
});
test('nested quest and movement scopes use the selected class; absent guide flags are known false',()=>{
 const shaman=resolveProfile({class:'Shaman'}),warlock=resolveProfile({class:'Warlock'}),ev={level:12},qstate=()=>0;
 const event={t:'move',only:[{classes:['Shaman']}]};
 assert.equal(allowed(event,{},ev,qstate,shaman),true);assert.equal(allowed(event,{},ev,qstate,warlock),false);
 const silverpine={conditions:['H12SF'],condition_format:'journey'};
 assert.equal(allowed(silverpine,{},ev,qstate,warlock),false);
 assert.equal(allowed(silverpine,{},ev,qstate,resolveProfile({class:'Warlock',race:'Undead'})),true);
 const source=planProfileSources(catalog,shaman).find(s=>s.id==='polestar');
 const full=loadCatalogSources([source],catalog,shaman)[0];assert.equal(full.catalog,true);
 assert.ok(full.groups.some(g=>g.events.length>0));
});
test('time ledger accounts for source and generated costs without discounts and detects missing costs',()=>{
 const book=ledger.ledger();
 ledger.record(book,{seconds:0,travelSeconds:0,grindSeconds:0},{seconds:100,travelSeconds:40,grindSeconds:10},'complete');
 ledger.record(book,{seconds:100,travelSeconds:40,grindSeconds:10},{seconds:130,travelSeconds:50,grindSeconds:10},'turnin',true);
 const row={endSeconds:130,travelSeconds:50,grindSeconds:10,costLedger:book};
 assert.equal(accounting(row).closed,true);assert.equal(book.byOrigin.generated.seconds,30);
 assert.equal(accounting({...row,endSeconds:129}).closed,false);
 assert.equal(accounting({...row,travelSeconds:150}).closed,false);
 assert.equal(accounting({...row,endSeconds:NaN}).closed,false);
});
test('finite times, an asserted fit or an incomplete comparison cannot manufacture a fastest route',()=>{
 const p=resolveProfile({class:'Hunter'}),book=ledger.ledger();
 ledger.record(book,{seconds:0,travelSeconds:0,grindSeconds:0},{seconds:100,travelSeconds:20,grindSeconds:10},'turnin');
 const rows=['rxp','journey','wisp','polestar'].map(id=>({id,endLevel:30,timeTo:{30:100},originalCoverage:[1,30],generatedContinuation:[],blockers:{},sourceProfile:{...p},endSeconds:100,travelSeconds:20,grindSeconds:10,costLedger:book}));
 const report={rows,profile:p,timeModel:{fit:'self-asserted',calibrated:true}};
 const gate=rankingEligibility(report);assert.equal(gate.winner,null);assert.equal(gate.eligible,false);
 assert.ok(gate.reasons.includes('class_combat_and_recovery_measurements_not_integrated'));
 const stateful=rankingEligibility({...report,timeModel:{kind:'isolated_foreversim_stateful_diagnostic'}});
 assert.equal(stateful.winner,null);assert.ok(stateful.reasons.includes('stateful_combat_not_calibrated_for_1_30'));
 const generated=structuredClone(report);generated.rows[0].generatedContinuation=[{q:1}];
 assert.ok(rankingEligibility(generated).reasons.includes('rxp:generated_or_incomplete_source'));
 const mismatch=structuredClone(report);mismatch.rows[0].sourceProfile.class='Warlock';
 assert.ok(rankingEligibility(mismatch).reasons.includes('rxp:profile_mismatch'));
 assert.ok(rankingEligibility({...report,rows:rows.slice(1)}).reasons.includes('four_unique_routes_required'));
});
test('all 36 diagnostic combinations reach 30, retain blockers and refuse an unvalidated winner',()=>{
 const out=fs.mkdtempSync(path.join(os.tmpdir(),'forever-class-matrix-'));
 try{
  const result=spawnSync(process.execPath,[path.join(root,'scripts/run_class_matrix.cjs'),'--output-dir',out,'--diagnostic-continuations','--require-validated'],{encoding:'utf8',timeout:120000,maxBuffer:4*1024*1024});
  assert.equal(result.status,2,result.stderr);
  const m=JSON.parse(fs.readFileSync(path.join(out,'matrix.json')));
  assert.equal(m.actualCells,36);assert.equal(m.executionFailures,0);assert.equal(m.targetReachedCells,36);assert.equal(m.validatedWinnerCount,0);
  for(const p of m.profiles){
   assert.equal(p.cells.length,4);assert.equal(p.ranking.winner,null);assert.equal(p.timeModel.kind,'uncalibrated_class_priors');
   for(const c of p.cells){assert.equal(c.endLevel,30);assert.ok(Number.isFinite(c.timeToTarget));assert.equal(c.accounting.closed,true,JSON.stringify(c.accounting));assert.equal(c.generatedActionFailures,0,p.profile.class+':'+c.id);}
  }
 }finally{fs.rmSync(out,{recursive:true,force:true});}
});
