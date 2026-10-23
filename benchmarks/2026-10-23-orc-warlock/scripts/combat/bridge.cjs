'use strict';
const fs=require('node:fs'),path=require('node:path'),crypto=require('node:crypto'),os=require('node:os'),zlib=require('node:zlib'),{spawnSync}=require('node:child_process');
const snapshot=path.resolve(__dirname,'../../../../vendor/foreversim-2083b589');
const EXPECTED_COMMIT='2083b58935fc17dfae86fb4b188a4659bd0c845c';
const BLOBS={'CombatEngine.lua':'724b0a9d04b4308f077864b3f8f5dc0e0129b020','Classes/warlock.lua':'9b904724c4b4d8c0f7031625d413793c54e302e9','UniversalRules.lua':'5a86e1a702729ac791c7f4353f1782e16b5dc744','Optimizers.lua':'4b8f3b5659d31ca83285c4dbc726913b31a5034d','THIRD_PARTY_LICENSES.txt':'9d236ab2b413dc4be62822ec8a4e3c7743ec2c52'};
function sources(dir){return JSON.parse(zlib.gunzipSync(fs.readFileSync(path.join(dir,'sources.json.gz'))));}
function verifySnapshot(dir=snapshot){
 const manifest=JSON.parse(fs.readFileSync(path.join(dir,'manifest.json')));
 if(manifest.commit!==EXPECTED_COMMIT)throw Error('Unexpected ForeverSim snapshot commit');
 const content=sources(dir);
 if(Object.keys(content).length!==Object.keys(BLOBS).length||Object.keys(content).some(k=>!BLOBS[k]))throw Error('Unexpected snapshot file set');
 if(manifest.files.length!==Object.keys(BLOBS).length||new Set(manifest.files.map(f=>f.file)).size!==Object.keys(BLOBS).length)throw Error('Incomplete ForeverSim snapshot');
 for(const f of manifest.files){
  if(!BLOBS[f.file]||BLOBS[f.file]!==f.gitBlob)throw Error('Unexpected source identity');
  const bytes=Buffer.from(content[f.file]);
  if(crypto.createHash('sha256').update(bytes).digest('hex')!==f.sha256)throw Error('ForeverSim snapshot changed: '+f.file);
  const git=crypto.createHash('sha1').update('blob '+bytes.length+'\0').update(bytes).digest('hex');
  if(git!==f.gitBlob)throw Error('Pinned source blob mismatch: '+f.file);
 }
 return manifest;
}
function literal(value){
 if(value===null)return 'nil';if(typeof value==='boolean')return String(value);
 if(typeof value==='number'){if(!Number.isFinite(value))throw Error('Nonfinite request');return String(value);}
 if(typeof value==='string')return '"'+value.replace(/[\\"\x00-\x1f]/g,c=>'\\'+c.charCodeAt(0).toString().padStart(3,'0'))+'"';
 if(Array.isArray(value))return '{'+value.map(literal).join(',')+'}';
 if(value&&typeof value==='object')return '{'+Object.entries(value).filter(([,v])=>v!==undefined).map(([k,v])=>'['+literal(k)+']='+literal(v)).join(',')+'}';
 throw Error('Unsupported request value');
}
function validate(job){
 if(!Array.isArray(job.events)||!job.events.length)throw Error('Combat events required');
 if(job.previousFight)validate({events:[job.previousFight]});
 if(job.initialState&&(!Number.isFinite(job.initialState.mana)||job.initialState.mana<0||!Number.isFinite(job.initialState.health)||job.initialState.health<=0))throw Error('Invalid persisted player resources');
 for(const e of job.events){
  if(e.kind==='idle'){
   if(!Number.isFinite(e.seconds)||e.seconds<0)throw Error('Invalid idle duration');
   for(const [key,v] of Object.entries(e.recovery||{}))if(!['manaPerSecond','healthPerSecond'].includes(key)||!Number.isFinite(v)||v<0)throw Error('Invalid recovery rate');
   continue;
  }
  if(e.kind!=='fight')throw Error('Unknown combat event');
  if(e.player?.class!=='Warlock')throw Error('Initial isolated adapter supports Warlock only');
  if(!Number.isInteger(e.player.level)||e.player.level<5||e.player.level>30)throw Error('Supported diagnostic level window is 5-30; 1-4 remain blocked');
  if(!e.player.hasSpellbook||!e.player.knownSpellRanks)throw Error('An explicit learned spellbook is required');
  for(const key of ['health','mana','baseMana','intellect','spirit','hastePct','hitPct','critPct','spellPower','manaRegen'])if(!Number.isFinite(e.player.stats?.[key])||e.player.stats[key]<0)throw Error('Explicit captured character stat required: '+key);
  if(e.player.stats.health<=0||e.player.stats.mana<=0)throw Error('Positive character resource maxima required');
  if(e.player.petActive){
   const pet=e.player.pet;
   if(!pet||!Number.isInteger(pet.level)||!pet.hasSpellbook||!pet.knownSpellRanks||!Number.isFinite(pet.mana)||pet.mana<0||!Number.isFinite(pet.manaRegen)||pet.manaRegen<0)throw Error('Explicit pet level, resource maxima, regeneration and spellbook required');
  }
  if(!Number.isFinite(e.enemy?.health)||e.enemy.health<=0||!Number.isInteger(e.enemy.level))throw Error('Sourced enemy health and level required');
  if(!e.choice?.policies?.length)throw Error('Explicit rotation required');
  if(e.duration!==undefined&&(!Number.isFinite(e.duration)||e.duration<=0||e.duration>600))throw Error('Combat horizon must be 0-600 seconds');
  const encounter=e.player.encounter;
  if(!encounter||encounter.levelDifference!==e.enemy.level-e.player.level||!Number.isFinite(encounter.armor)||encounter.armor<0||!Number.isFinite(encounter.incomingDamage)||encounter.incomingDamage<0||!(encounter.hitEvery>0))throw Error('Explicit enemy armor, incoming damage and swing interval required');
  if(encounter.levelResistance!==false||!encounter.targetResist||Object.values(encounter.targetResist).some(v=>!Number.isFinite(v)||v<0))throw Error('Explicit target resistance profile required; implicit boss resistance is disabled');
  if(!Number.isInteger(e.seed)||e.seed<0||e.seed>0xffffffff)throw Error('Explicit uint32 seed required');
 }
}
function auditInput(config){
 const issues=[];
 for(let level=1;level<=30;level++){
  if(level<5)issues.push({level,reason:'source_level_unsupported'});
  else if(!config.levels?.[level])issues.push({level,reason:'missing_character_profile'});
  else if(level<10)issues.push({level,reason:'source_level_data_approximation'});
 }
 return {complete:false,validated:false,rankingEligible:false,issues,remaining:['Recorded character, enemy and recovery evidence','Pet health, threat and tanking','Training and equipment progression','Recorded route-time validation']};
}
function run(job,{lua=process.env.LUA||'lua',timeout=30000}={}){
 const manifest=verifySnapshot();validate(job);
 const dir=fs.mkdtempSync(path.join(os.tmpdir(),'foreverroutes-combat-'));
 let p;try{
  for(const [name,text] of Object.entries(sources(snapshot))){const file=path.join(dir,name);fs.mkdirSync(path.dirname(file),{recursive:true});fs.writeFileSync(file,text,{mode:0o444});}
  p=spawnSync(lua,[path.join(__dirname,'worker.lua'),dir,__dirname],{input:literal(job),encoding:'utf8',timeout,maxBuffer:16*1024*1024});
 }finally{fs.rmSync(dir,{recursive:true,force:true});}
 if(p.error||p.status!==0)throw Error('Isolated ForeverSim run failed: '+(p.error?.message||p.stderr));
 const result=JSON.parse(p.stdout);result.source={repository:manifest.repository,commit:manifest.commit,filesVerified:true};
 result.validated=false;result.rankingEligible=false;
 result.approximateLevels=[...new Set(result.rows.filter(r=>r.approximateLevel).map(r=>r.level))];
 return result;
}
module.exports={verifySnapshot,literal,validate,auditInput,run,snapshot};
