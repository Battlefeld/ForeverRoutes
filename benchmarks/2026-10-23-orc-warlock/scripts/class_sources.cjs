'use strict';
const fs=require('node:fs'),zlib=require('node:zlib');
const contract=require('../../2026-10-23-independent-route-checks/scripts/lib/route_contract.cjs');
const raceKey=s=>String(s).replace(/^Scourge$/,'Undead');
function planProfileSources(file,profile,target=30){
 const catalog=JSON.parse(zlib.gunzipSync(fs.readFileSync(file))).guides;
 const ctx=contract.context(profile,{level:1,questState:()=>0});
 const relevant=d=>(d.faction===profile.faction||d.faction==='Both')&&(!d.races||d.races.some(r=>raceKey(r)===profile.race));
 const source=(id,docs,issues=[])=>({id,label:id+' '+profile.race+' '+profile.class,coverage:[1,docs.length?Math.max(...docs.map(d=>d.coverage[1])):0],groups:docs.map(d=>({guide:d.guideId,events:[]})),useCatalog:true,selectionIssues:issues,limits:[],selectionProfile:{race:profile.race,class:profile.class,faction:profile.faction}});
 function chained(id,first,next){
  const docs=[],issues=[],seen=new Set();let d=first;
  while(d){if(seen.has(d.guideId)){issues.push({reason:'guide_cycle',guide:d.guideId});break;}if(id==='polestar'&&docs.length&&d.coverage[0]>=target)break;seen.add(d.guideId);docs.push(d);if(id!=='polestar'&&d.coverage[1]>=target)break;
   const result=next(d);if(result.issue)issues.push(result.issue);d=result.doc;
  }
  if(!first)issues.push({reason:'starter_missing',race:profile.race});
  return source(id,docs,issues);
 }
 const rxps=catalog.filter(d=>d.source==='rxp'&&relevant(d));
 const starts=rxps.filter(d=>d.coverage[0]===1&&(d.defaultFor||[]).some(c=>contract.rxp(c,ctx)===true));
 const rxp=chained('rxp',starts.length===1?starts[0]:null,d=>{
  const candidates=[],missing=[];
  for(const value of d.nextChoices||[])for(const part of value.split(';')){
   const [name,condition]=part.split('<<').map(x=>x.trim());if(condition&&contract.rxp(condition,ctx)!==true)continue;
   const title=name.split('\\').pop(),found=rxps.find(x=>x.guideId===title);
   if(!found){missing.push(title);continue;}
   if(!found.defaultFor?.length||found.defaultFor.some(c=>contract.rxp(c,ctx)===true))candidates.push(found);
  }
  const unique=[...new Map(candidates.map(x=>[x.guideId,x])).values()];
  if(unique.length===1)return {doc:unique[0],issue:missing.length?{reason:'unavailable_source_alternatives',guide:d.guideId,missing}:null};
  return {doc:null,issue:{reason:unique.length?'ambiguous_source_next':'next_chapter_missing',guide:d.guideId,candidates:unique.map(x=>x.guideId),missing}};
 });
 const journeyStarts={Orc:['H01D_FVR','H06D_FVR'],Troll:['H01D_FVR','H06D_FVR'],Tauren:['H01M_FVR','H06M_FVR'],Undead:['H01TG_FVR','H06TG_FVR'],Human:['A01EF_FVR','A06EF_FVR'],Dwarf:['A01DM_FVR','A06DM_FVR'],Gnome:['A01DM_FVR','A06DM_FVR'],NightElf:['A01T_FVR','A06T_FVR']};
 const journeyTail=profile.faction==='Horde'?[profile.race==='Undead'?'H12SF_FVR':'H12B_FVR','H15S_FVR','H16B_FVR','H20SM_FVR','H21A_FVR','H22SB_FVR','H23S_FVR','H25SB_FVR','H25TN_FVR','H26A_FVR','H27SM_FVR','H28TN_FVR','H29HF_FVR']:['A12D_FVR','A17LM_FVR','A18RM_FVR','A20D_FVR','A21A_FVR','A22SM_FVR','A23D_FVR','A24A_FVR','A24W_FVR','A27RM_FVR','A28D_FVR'];
 const journeyDocs=[],journeyIssues=[];
 for(const id of [...(journeyStarts[profile.race]||[]),...journeyTail]){
  const d=catalog.find(d=>d.source==='journey'&&d.guideId===id&&relevant(d));
  if(!d){journeyIssues.push({reason:'chapter_missing',guide:id});break;}journeyDocs.push(d);if(d.coverage[1]>=target)break;
 }
 const journey=source('journey',journeyDocs,journeyIssues);
 const wisps=catalog.filter(d=>d.source==='wisp'&&relevant(d));
 const wisp=chained('wisp',wisps.find(d=>d.coverage[0]===1),d=>({doc:wisps.find(x=>x.guideId===d.nextGuide),issue:wisps.some(x=>x.guideId===d.nextGuide)?null:{reason:'next_chapter_missing',guide:d.nextGuide}}));
 const stars=catalog.filter(d=>d.source==='polestar'&&relevant(d));
 const polestar=chained('polestar',stars.find(d=>d.coverage[0]===1&&d.races?.some(r=>raceKey(r)===profile.race)),d=>({doc:stars.find(x=>x.guideId===d.nextGuide),issue:stars.some(x=>x.guideId===d.nextGuide)?null:{reason:'next_chapter_missing',guide:d.nextGuide}}));
 return [rxp,journey,wisp,polestar];
}
module.exports={planProfileSources};
