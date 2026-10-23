'use strict';
// Build common, source-tagged model inputs without requiring a live Wago request.
// Geometry and transport tables come from Polestar's pinned generated addon data.
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const root = path.resolve(__dirname, '..');
const external = path.resolve(process.env.POLESTAR_ROOT || path.join(root, '..', 'polestar'));
const qdir = path.resolve(process.env.QUESTIEDB_ROOT || path.join(root, '..', 'QuestieDB'));
const req = p => require(path.join(external, 'tools/lib', p));
const { luaToJson } = req('lua');
const { readQuestie } = req('data/questie');
const { loadPfQuest } = req('data/pfquest');
const { normalizePfQuest } = req('data/normalize');
const { applyQuestie } = req('data/questie-merge');
const { loadQuestCounts } = req('travel/cmangos');
const outDir = path.join(root, '.comparison/data');
const json = p => JSON.parse(fs.readFileSync(p, 'utf8'));
const save = (name, obj) => fs.writeFileSync(path.join(outDir, name + '.json'), JSON.stringify(obj) + '\n');

async function main() {
 fs.mkdirSync(outDir, { recursive: true });
 const qz = readQuestie(qdir, { log: console.log });
 console.log('Questie loaded', qz.quests.size, Object.keys(qz));
 const framesFile = path.join(external, 'addon/Polestar_Guides/MapFrames.lua');
 const frames = luaToJson({ files: [framesFile], prelude: 'local frames; Polestar={RegisterMapFrames=function(x) frames=x end}', expr: 'return frames' });
 const travelFile = path.join(external, 'addon/Polestar/Nav/TravelData.lua');
 const travelRaw = luaToJson({ prelude: 'local travel; local ns={Nav=setmetatable({RegisterTravelData=function(x) travel=x end,RegisterCrossings=function(x) if x.complete then travel.zoneFrames=x.zones end end},{__index=function() return function() end end})}; assert(loadfile(' + JSON.stringify(travelFile) + '))("Polestar",ns)', expr: 'return travel' });
 frames.forever = travelRaw.zoneFrames;
 console.log('Published geometry loaded', Object.keys(frames), 'travel', Object.keys(travelRaw));
 const changed = new Set(Object.keys(frames.forever || {}).filter(id => !frames.era[id] || ['left','right','top','bottom','c'].some(k => Math.abs(frames.forever[id][k] - frames.era[id][k]) > 1)).map(Number));
 const liveMaps = { changed, has: map => changed.has(map) || !frames.era[map] };
 const areaToUiMap = area => {
  const m = qz.areaToUiMap.get(Number(area));
  if (m > 0) return m;
  const parent = qz.subZoneToParent && qz.subZoneToParent.get(Number(area));
  return parent ? qz.areaToUiMap.get(Number(parent)) || null : null;
 };
 const toWorld = (map, f, x, y) => {
  const fr = (f === 'l' && frames.forever && frames.forever[map]) || frames.era[map] || (frames.forever && frames.forever[map]);
  return fr ? { c: fr.c, wx: fr.top + (fr.bottom-fr.top)*y/100, wy: fr.left+(fr.right-fr.left)*x/100 } : null;
 };
 const pf = loadPfQuest(path.join(external, 'data/sources/pfQuest'));
 const spawnOptions = req('data/spawns').DEFAULTS;
 const norm = normalizePfQuest(pf, { resolveArea: areaToUiMap, spawnOptions });
 const counts = await loadQuestCounts({sourcesDir: path.join(external,'data/sources'), offline:true});
 const stats = applyQuestie(norm, qz, { areaToUiMap, liveMaps, spawnOptions, toWorld, questCounts:counts.counts });
 const corrections = req('data/corrections.json');
 for (const [id,c] of Object.entries(corrections.quests || {})) if(norm.quests[id]) Object.assign(norm.quests[id], c.set);
 for (const [id,c] of Object.entries(corrections.items || {})) {
  const it=norm.items[id]; if(!it)continue;
  for(const kind of ['npc','object'])for(const k of (c.removeDrops || {})[kind] || [])delete it.drops[kind][k];
  if(c.removeVendors)it.vendors=it.vendors.filter(k=>!c.removeVendors.includes(k));
 }
 for(const kind of ['npcs','objects'])for(const [id,c] of Object.entries(corrections[kind] || {})) {
  const e=norm[kind][id];if(!e)continue;
  if(c.set)Object.assign(e,c.set);
  if(c.addSpawns)e.spawns={...e.spawns,...c.addSpawns};
 }
 const db=json(path.join(root,'.inputs/unified/quests-1-30.json'));
 const xpSources={};const zeroXp=[];const variableObserved=[];
 for(const [id,entry] of Object.entries(db.quests)) {
  const q=norm.quests[id];if(!q)continue;
  const f=entry.fields || {};let rec=f.xp;
  if(f.xp_observed_by_player_level) {
   const vals=[...new Set(Object.values(f.xp_observed_by_player_level.value))];
   if(vals.length===1)rec={...f.xp_observed_by_player_level,value:vals[0]};
   else variableObserved.push({id,observation:f.xp_observed_by_player_level});
  }
  if(rec && Number.isFinite(rec.value)) {q.xp=rec.value;xpSources[id]=rec;if(rec.value===0)zeroXp.push(Number(id));}
 }
 const zones={};
 for(const [id,fr] of Object.entries(travelRaw.zoneFrames || {})) zones[id]={id:Number(id),name:fr.name || String(id),areaId:qz.uiMapToArea.get(Number(id)) || null,type:3,parent:null};
 const point = p => ({name:p[0],c:p[1],wx:p[2],wy:p[3],map:p[4],x:p[5],y:p[6],f:p[7] || 'e'});
 const taxi=(travelRaw.taxi || []).map(r=>({id:r[0],c:r[1],wx:r[2],wy:r[3],map:r[4],x:r[5],y:r[6],faction:r[7],name:r[8],short:r[8]}));
 const legs=[];
 // Shipped flight lengths use the same 32 yd/s prior as Polestar's travel model.
 for(let i=0;i<(travelRaw.flights || []).length;i+=3)legs.push({from:travelRaw.flights[i],to:travelRaw.flights[i+1],cost:travelRaw.flights[i+2]/32});
 const transports=(travelRaw.transports || []).map(t=>({...t,a:Array.isArray(t.a)?point(t.a):t.a,b:Array.isArray(t.b)?point(t.b):t.b}));
 const inns=(travelRaw.inns || []).map(t=>({...t,name:(t.names || [])[0] || t.name}));
 const travel={taxi:{nodes:taxi,legs},transports,inns};
 const missingCounts=Object.values(norm.quests).filter(q=>(q.objectives || []).some(o=>['kill','item','object'].includes(o.type) && !(o.count>0))).map(q=>q.id);
 for(const k of ['quests','npcs','objects','items'])save(k,norm[k]);
 save('zones',zones);save('map-frames',frames);save('travel',travel);
 save('xp-provenance',xpSources);save('zero-xp',zeroXp);save('missing-counts',missingCounts);
 const provenance={builtAt:new Date().toISOString(),engine:'fitzforge/polestar@eb7b47840f032bf711a3c8d7b1df9f3f6551b990',questie:qz.info,stats,xpOverlays:Object.keys(xpSources).length,zeroXp:zeroXp.length,variableObserved,missingCounts:missingCounts.length,geometry:'Pinned generated MapFrames.lua and TravelData.lua; live Wago request returned HTTP 403',classicObjectiveCounts:counts.rev,limits:['Provisional model; not calibrated player /played','Unrouted walks use the common Polestar detour prior','Flight lengths converted at the common 32 yd/s travel prior','Unknown item drop chances keep the explicit Polestar 30% / 60% priors','Unknown objective counts are recorded as blockers for a definitive speed ranking']};
 save('meta',provenance);
 console.log(JSON.stringify({outDir,quests:Object.keys(norm.quests).length,npcs:Object.keys(norm.npcs).length,frames:Object.keys(zones).length,transports:transports.length,zeroXp:zeroXp.length,missingCounts:missingCounts.length}));
}
main().catch(e=>{console.error(e);process.exitCode=1});
