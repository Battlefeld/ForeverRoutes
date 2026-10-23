'use strict';
// Offline rebuild from pinned Questie entities, existing audited quest facts,
// and the same published geometry/transport data for BOTH factions.
const fs=require('node:fs'),path=require('node:path'),zlib=require('node:zlib');
const {execFileSync}=require('node:child_process');
const root=path.resolve(__dirname,'..');
const engine=process.env.POLESTAR_ROOT,questie=process.env.QUESTIEDB_ROOT;
if(!engine||!questie)throw Error('Set POLESTAR_ROOT and QUESTIEDB_ROOT to pinned checkouts');
if(execFileSync('git',['-C',engine,'rev-parse','HEAD'],{encoding:'utf8'}).trim()!=='eb7b47840f032bf711a3c8d7b1df9f3f6551b990')throw Error('Unexpected Polestar engine revision');
const req=p=>require(path.join(engine,'tools/lib',p));
const {readQuestie}=req('data/questie'),{applyQuestie}=req('data/questie-merge'),{buildWorld}=req('sim/world'),{luaToJson}=req('lua');
const qz=readQuestie(questie);
if(qz.info.buildCommit!=='1f538335cfbb240c14a97c340f48c555fdc8a52b')throw Error('Unexpected Questie source revision');
const frames=JSON.parse(fs.readFileSync(path.join(root,'data/replay-map-frames.json')));
const questFacts=JSON.parse(zlib.gunzipSync(fs.readFileSync(path.resolve(root,'../2026-10-23-independent-route-checks/data/quests.json.gz'))));
const areaToUiMap=area=>qz.areaToUiMap.get(Number(area))||qz.areaToUiMap.get(qz.subZoneToParent.get(Number(area)))||null;
const changed=new Set(Object.keys(frames.forever||{}).filter(id=>!frames.era[id]||['left','right','top','bottom','c'].some(k=>frames.forever[id][k]!==frames.era[id][k])).map(Number));
const toWorld=(map,f,x,y)=>{const fr=(f==='l'&&frames.forever[map])||frames.era[map]||frames.forever[map];return fr?{c:fr.c,wx:fr.top+(fr.bottom-fr.top)*y/100,wy:fr.left+(fr.right-fr.left)*x/100}:null;};
const data={quests:{},npcs:{},objects:{},items:{}};
const stats=applyQuestie(data,qz,{areaToUiMap,liveMaps:{has:m=>changed.has(m)||!frames.era[m]},spawnOptions:req('data/spawns').DEFAULTS,toWorld});
data.quests=questFacts;
data['map-frames']=frames;
data.zones=Object.fromEntries(Object.entries(frames.forever).map(([id,fr])=>[id,{id:Number(id),name:fr.name||id,areaId:qz.uiMapToArea.get(Number(id))||null,type:3,parent:null}]));
const travelFile=path.join(engine,'addon/Polestar/Nav/TravelData.lua');
const raw=luaToJson({prelude:'local travel; local ns={Nav=setmetatable({RegisterTravelData=function(x) travel=x end,RegisterCrossings=function(x) if x.complete then travel.zoneFrames=x.zones end end},{__index=function() return function() end end})}; assert(loadfile('+JSON.stringify(travelFile)+'))("Polestar",ns)',expr:'return travel'});
const point=p=>({name:p[0],c:p[1],wx:p[2],wy:p[3],map:p[4],x:p[5],y:p[6],f:p[7]||'e'});
const taxi=(raw.taxi||[]).map(r=>({id:r[0],c:r[1],wx:r[2],wy:r[3],map:r[4],x:r[5],y:r[6],faction:r[7],name:r[8],short:r[8]}));
const legs=[];for(let i=0;i<(raw.flights||[]).length;i+=3)legs.push({from:raw.flights[i],to:raw.flights[i+1],cost:raw.flights[i+2]/32});
data.travel={taxi:{nodes:taxi,legs},transports:(raw.transports||[]).map(t=>({...t,a:Array.isArray(t.a)?point(t.a):t.a,b:Array.isArray(t.b)?point(t.b):t.b})),inns:(raw.inns||[]).map(t=>({...t,name:t.names?.[0]||t.name}))};
const out=path.join(root,'.comparison/matrix-worlds');fs.mkdirSync(out,{recursive:true});
for(const faction of ['Horde','Alliance']){
 const world=buildWorld(data,{faction});world.questXp=Array.from({length:61},()=>Array(10).fill(0));
 world.inputProvenance={questie:qz.info,questFacts:'existing audited quest snapshot',entityStats:stats,engine:'fitzforge/polestar@eb7b47840f032bf711a3c8d7b1df9f3f6551b990',geometry:frames.provenance};
 const file=path.join(out,'world-'+faction.toLowerCase()+'.json.gz');fs.writeFileSync(file,zlib.gzipSync(JSON.stringify(world)));
 console.log(JSON.stringify({faction,file,quests:world.quests.length,starts:Object.keys(world.starts)}));
}
