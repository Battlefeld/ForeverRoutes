'use strict';
const fs=require('node:fs'),path=require('node:path'),zlib=require('node:zlib'),crypto=require('node:crypto');
const contract=require('../../2026-10-23-independent-route-checks/scripts/lib/route_contract.cjs');
const profile=contract.profile({race:'Orc',class:'Warlock',faction:'Horde',guideFlags:['H12B'],variables:{HORDETB20:false}});
function loadCatalogSources(sources,file){
 const bytes=fs.readFileSync(file),catalog=JSON.parse(zlib.gunzipSync(bytes));
 return sources.map(source=>{
  if(source.id==='polestar')return source;
  const chain=[...new Set(source.groups.map(g=>g.guide).filter(Boolean))];
  const groups=[],chapters=[];
  for(const id of chain){
   const matches=catalog.guides.filter(g=>g.source===source.id&&g.guideId===id);
   if(matches.length!==1)throw Error('Expected one catalog chapter for '+source.id+':'+id);
   const doc=matches[0];chapters.push({id:doc.id,sourceSha256:doc.sourceSha256});
   for(const g of doc.groups){const copy=structuredClone(g);if((copy.flags||[]).some(f=>/^completewith(?: |$)/.test(f)))copy.flags.push('parallel');groups.push({...copy,catalog:true});}
  }
  const accepts=new Map(),staticContext=contract.context(profile);
  for(const e of groups.flatMap(g=>g.events||[])){if(e.t!=='accept'||contract.scope(e,profile,staticContext)===false)continue;if(!accepts.has(e.q))accepts.set(e.q,[]);accepts.get(e.q).push(e);}
  const optionalQuests=[...accepts].filter(([,events])=>events.every(e=>e.optional)).map(([id])=>id);
  return {...source,groups,catalog:true,optionalQuests,inputProvenance:{kind:'full_guide_catalog',file:path.basename(file),sha256:crypto.createHash('sha256').update(bytes).digest('hex'),chapters}};
 });
}
function context(run,ev,qstate){
 const ctx=contract.context(profile,{level:ev.level,questState:id=>qstate(ev,run,id),inventory:ev.inventory||{},variables:ev.variables||{HORDETB20:false}});
 const lookup=ctx.lookup;ctx.lookup=t=>{if(t.toUpperCase()==='HC')return profile.hardcore;if(t.toUpperCase()==='INGROUP')return profile.partySize>1;return lookup(t);};return ctx;
}
function allowed(value,run,ev,qstate){
 const result=contract.scope(value,profile,context(run,ev,qstate));
 return result===undefined?null:result;
}
function questGuard(event,run,ev,qstate){
 const ids=String(event.args||'').split(',').map(Number).filter(x=>Number.isInteger(x)&&x>0);
 if(!ids.length)return null;
 const states=ids.map(id=>qstate(ev,run,id));
 switch(event.directive){
  case 'isOnQuest':case 'isQuestActive':return states.some(x=>x===1||x===2);
  case 'isNotOnQuest':return !states.some(x=>x===1||x===2);
  case 'isQuestComplete':return states.some(x=>x===2);
  case 'isNotQuestComplete':case 'isQuestNotComplete':return !states.some(x=>x===2);
  case 'isQuestTurnedIn':return states.some(x=>x===3);
  case 'isNotQuestTurnedIn':return !states.some(x=>x===3);
  case 'isQuestAvailable':case 'isQuestNotAvailable':{
   let available=false;for(const id of ids){const i=run.qIndex.get(id);if(i==null)return null;available ||= !ev.done[i]&&!ev.active[i]&&ev.level>=run.qMinLevel[i]&&ev.prereqOk(i);}
   return event.directive==='isQuestAvailable'?available:!available;
  }
  default:return null;
 }
}
// RXP .xp with skipstep applies to the WHOLE step, and retains its own
// class condition. Verified against RXPGuides.functions.xp at the pinned SHA.
function levelGuard(event,ev,levelXp){
 const m=/^([<>]?)(\d+)([+.-]?\d*)\s*,\s*(-?\d+)(?:,.*)?$/.exec(event.args||'');
 if(!m)return null;
 let level=Number(m[2]),xp=Number(m[3]||0);
 if(xp<0){level--;xp=(levelXp[level]||0)+xp;}
 else if(xp<1)xp*=levelXp[level]||0;
 const reached=ev.level>level||(ev.level===level&&ev.xp>=xp);
 const reverse=m[1]==='<'||Number(m[4])<0;
 return reached!==!reverse; // true means the step remains enabled
}
function groupAllowed(g,run,ev,qstate,levelXp){
 let answer=allowed(g,run,ev,qstate);
 if(answer===false)return false;
 for(const e of g.events||[]){
  if(!['quest_guard','level_guard','inventory_guard'].includes(e.t))continue;
  const ec=allowed(e,run,ev,qstate);if(ec===false)continue;
  const value=ec===null?null:e.t==='quest_guard'?questGuard(e,run,ev,qstate):e.t==='inventory_guard'?inventoryGuard(e,ev):levelGuard(e,ev,levelXp);
  if(value===false)return false;if(value===null)answer=null;
 }
 return answer;
}
function inventoryGuard(e,ev){
 const m=/^(\d+)\s*,\s*([<>]?)(=?)(\d+)(?:,.*)?$/.exec(e.args||'');
 if(!m||!ev.inventory||!Object.hasOwn(ev.inventory,m[1]))return null;
 const actual=ev.inventory[m[1]],target=Number(m[4]);
 if(m[2]==='<')return m[3]?actual<=target:actual<target;
 if(m[2]==='>')return m[3]?actual>=target:actual>target;
 return m[3]?actual===target:actual>=target;
}
function optional(event,source,withOptional=false){return !withOptional&&(!!event.optional||(source.optionalQuests||[]).includes(event.q));}
module.exports={loadCatalogSources,allowed,groupAllowed,questGuard,levelGuard,inventoryGuard,optional};
