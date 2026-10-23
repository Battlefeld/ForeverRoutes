'use strict';
// Source-independent input contract. Unknown state never silently becomes false.
const CLASS_BITS={Warrior:1,Paladin:2,Hunter:4,Rogue:8,Priest:16,Shaman:64,Mage:128,Warlock:256,Druid:1024};
const RACES={Human:['Alliance',1],Orc:['Horde',2],Dwarf:['Alliance',4],NightElf:['Alliance',8],Undead:['Horde',16],Tauren:['Horde',32],Gnome:['Alliance',64],Troll:['Horde',128],Skyborne:[null,null]};
const KIND=new Set(['guide','accept','complete','turnin','level','move','bind','hearth','learn_flight','transport','deathskip','resurrect','transition','inventory_goal','inventory_guard','destroy','set_variable','bank_deposit','bank_withdraw','track_objective','choice','service','callback','manual','arrival','vendor','train','level_guard','use_item','unsupported','quest_guard','money_guard','profession','cast','gossip','grind']);
const REPLAY_SUPPORTED=new Set(['guide','accept','complete','turnin','level','transition']);
const STATIC=new Set([...Object.keys(CLASS_BITS).map(x=>x.toUpperCase()),...Object.keys(RACES).map(x=>x.toUpperCase()),'SCOURGE','NIGHTELF','BLOODELF','DRAENEI','ALLIANCE','HORDE','FOREVER','ERA','SOM1','SOM2','SOD','SOD2','TBC','WOTLK','CATA','JOYOUS','GROUPS','AH','HARDCORE','SOFTCORE','SKIP']);
const key=s=>String(s).toLowerCase().replace(/[ _-]/g,'');
function named(s,names){return Object.keys(names).find(n=>key(n)===key(s));}
function profile(o={}){
 const race=named(o.race || 'Orc',RACES),cls=named(o.class || 'Warlock',CLASS_BITS),faction=named(o.faction || (race && RACES[race][0]) || 'Horde',{Alliance:1,Horde:1});
 if(!race||!cls||!faction)throw Error('Unknown race, class or faction');
 if(RACES[race][0] && RACES[race][0]!==faction)throw Error('Race does not belong to the selected faction');
 const level=o.level || 1,targetLevel=o.targetLevel || 30;
 if(!Number.isInteger(level)||level<1||!Number.isInteger(targetLevel)||targetLevel<level||targetLevel>60)throw Error('Invalid level window');
 return {race,class:cls,faction,level,targetLevel,partySize:o.partySize || 1,withOptional:!!o.withOptional,hardcore:!!o.hardcore,variables:o.variables || {},inventory:o.inventory || {},guideFlags:o.guideFlags || [],classBit:CLASS_BITS[cls],raceBit:race==='Skyborne'?2**(faction==='Alliance'?32:33):RACES[race][1]};
}
function hasBit(mask,bit){return Math.floor(Number(mask)/bit)%2>=1;}
function compatible(q,p){return !q || ((!q.classes || hasBit(q.classes,p.classBit))&&(!q.races || hasBit(q.races,p.raceBit)));}
function truth(x){return x===undefined?undefined:x!==false&&x!==null;}
function and(a,b){a=truth(a);b=truth(b);return a===false||b===false?false:a===undefined||b===undefined?undefined:true;}
function or(a,b){a=truth(a);b=truth(b);return a===true||b===true?true:a===undefined||b===undefined?undefined:false;}
function context(p,state={}){
 const flags=new Set([p.race.toUpperCase(),p.class.toUpperCase(),p.faction.toUpperCase(),'ERA','FOREVER',p.hardcore?'HARDCORE':'SOFTCORE',...p.guideFlags]);
 if(p.race==='Undead')flags.add('SCOURGE');if(p.partySize>1)flags.add('GROUPS');
 const variables={...p.variables,...state.variables};
 return {profile:p,state,lookup(t){
  const u=t.toUpperCase();
  if(u==='TRUE')return true;if(u==='FALSE')return false;if(u==='NIL')return null;
  if(u==='GAMEVERSION')return 16001;if(u==='LEVEL')return state.level;
  if(u==='NONE')return 0;if(u==='ACCEPTED')return 1;if(u==='COMPLETED')return 2;if(u==='TURNEDIN')return 3;
  if(u.startsWith('VAR.'))return variables[t.slice(4)];
  const m=/^([A-Z_]+)\[(\d+)\](?:\[(\d+)\])?$/.exec(u);
  if(m){const id=+m[2],ix=m[3]?+m[3]:null;
   if(m[1]==='QUEST')return state.questState?state.questState(id):undefined;
   if(m[1]==='ITEM')return Object.hasOwn(state.inventory || p.inventory,id)?(state.inventory || p.inventory)[id]:undefined;
   if(m[1]==='OBJECTIVE')return state.objectiveState?state.objectiveState(id,ix):undefined;
   if(m[1]==='OBJECTIVECOUNT')return state.objectiveCount?state.objectiveCount(id,ix):undefined;
   if(m[1]==='WITHIN')return state.within?state.within(id):undefined;
   return undefined;
  }
  if(flags.has(u))return true;if(STATIC.has(u))return false;
  if(Object.hasOwn(variables,t))return variables[t];
  if(state.knownGuideFlags && state.knownGuideFlags.has(u))return flags.has(u);
  return undefined;
 }};
}
function expression(s,ctx){
 if(!s)return true;
 const tokens=String(s).match(/>=|<=|==|~=|>|<|\(|\)|\d+(?:\.\d+)?|[A-Za-z_][A-Za-z_0-9.]*(?:\[\d+\]){0,2}/g) || [];
 if(tokens.join('').toLowerCase()!==s.replace(/\s/g,'').toLowerCase())return undefined;
 let i=0;
 function value(){const t=tokens[i++];if(t==='('){const x=expr();if(tokens[i++]!==')')throw Error('unclosed condition');return x;}if(t==null)throw Error('missing condition term');return /^\d/.test(t)?+t:ctx.lookup(t);}
 function compare(){const a=value(),op=tokens[i];if(!['>=','<=','==','~=','>','<'].includes(op))return a;i++;const b=value();if(a===undefined||b===undefined)return undefined;return op==='>='?a>=b:op==='<='?a<=b:op==='=='?a===b:op==='~='?a!==b:op==='>'?a>b:a<b;}
 function negate(){if((tokens[i] || '').toLowerCase()==='not'){i++;const a=negate();return a===undefined?undefined:!truth(a);}return compare();}
 function all(){let a=negate();while((tokens[i] || '').toLowerCase()==='and'){i++;a=and(a,negate());}return a;}
 function expr(){let a=all();while((tokens[i] || '').toLowerCase()==='or'){i++;a=or(a,all());}return a;}
 try{const a=expr();return i===tokens.length?truth(a):undefined;}catch{return undefined;}
}
function rxp(s,ctx){
 if(!s)return true;let answer=false;
 for(const clause of String(s).split('/')){let a=true;for(const token of clause.trim().split(/\s+/)){const neg=token.startsWith('!'),b=ctx.lookup(token.replace(/^!/,''));a=and(a,b===undefined?undefined:neg?!truth(b):b);}answer=or(answer,a);}
 return answer;
}
function wisp(c,ctx){
 if(!c || !Object.keys(c).length)return true;
 let result=true;
 for(const [k,v] of Object.entries(c)){
  if(k==='level')result=and(result,ctx.state.level===undefined?undefined:ctx.state.level>=v);
  else if(k==='quests'){for(const id of v) {const state=ctx.lookup('QUEST['+id+']');result=and(result,state===undefined?undefined:state>=3);}}
  else if(k==='anyQuests'){let any=false;for(const q of v){const state=ctx.lookup('QUEST['+q.questId+']');any=or(any,state===undefined?undefined:state>=(q.state==='turned-in'?3:q.state==='objectives'?2:1));}result=and(result,any);}
  else result=and(result,undefined);
 }
 return result;
}
function onlyHolds(o,p){
 if(!o)return true;
 for(const [k,v] of Object.entries(o)){
  if(k==='classes' && !v.some(x=>key(x)===key(p.class)))return false;
  if(k==='races' && !v.some(x=>key(x)===key(p.race)||(p.race==='Undead'&&key(x)==='scourge')))return false;
  if(k==='factions' && !v.some(x=>key(x)===key(p.faction)))return false;
  if(!['classes','races','factions'].includes(k))return undefined;
 }
 return true;
}
function scope(e,p,ctx){
 if(!onlyHolds({classes:e.classes || [p.class],races:e.races || [p.race]},p))return false;
 if((e.excludedClasses || []).some(x=>key(x)===key(p.class)))return false;
 let allowed=true;
 for(const only of e.only || [])allowed=and(allowed,onlyHolds(only,p));
 const conditions=e.conditions || (e.condition?[e.condition]:[]);
 for(const s of conditions)allowed=and(allowed,e.condition_format==='rxp'?rxp(s,ctx):e.condition_format==='wisp'?wisp(s,ctx):expression(s,ctx));
 return allowed;
}
function validate(doc){
 const errors=[];
 if(doc.schema!=='foreverroutes.guide_events.v2')errors.push({code:'schema',message:'Expected guide_events.v2'});
 if(!doc.id || !doc.source)errors.push({code:'identity',message:'Guide id and source required'});
 if(!Array.isArray(doc.coverage)||doc.coverage.length!==2||doc.coverage.some(x=>!Number.isInteger(x))||doc.coverage[0]<1||doc.coverage[1]<doc.coverage[0])errors.push({code:'coverage',message:'Valid level range required'});
 if(!Array.isArray(doc.groups)){errors.push({code:'groups',message:'Groups array required'});return errors;}
 for(const [gi,g] of doc.groups.entries())for(const [ei,e] of (g.events || []).entries()){
  const loc={guide:doc.id,group:gi,event:ei,origin:e.origin || g.origin};
  if(!KIND.has(e.t))errors.push({...loc,code:'event_type',message:'Unrecognized event '+e.t});
  if(['accept','complete','turnin'].includes(e.t)&&(!Number.isInteger(e.q)||e.q<=0))errors.push({...loc,code:'quest_id',message:'Positive quest id required'});
  if(e.t==='complete'&&e.obj!=null&&(!Number.isInteger(e.obj)||e.obj<1))errors.push({...loc,code:'objective_index',message:'Objective index must be one based'});
  if(e.t==='level'&&(!Number.isInteger(e.level)||e.level<1||e.level>60||!Number.isFinite(e.xp || 0)||(e.xp || 0)<0))errors.push({...loc,code:'level_event',message:'Valid level and XP required'});
 }
 return errors;
}
function check(doc,p,quests,world=null,sharedFacts={}){
 const issues=new Map(),schemaErrors=validate(doc),ctx=context(p),rows=[];
 const add=(code,e={},extra={})=>{const id=JSON.stringify([code,e.q,e.obj,e.t,e.directive,extra]);let issue=issues.get(id);if(!issue){issue={code,...(e.q?{quest:e.q}:{}),...(e.obj?{objective:e.obj}:{}),event:e.t,directive:e.directive,...extra,occurrences:0,origins:[]};issues.set(id,issue);}issue.occurrences++;if(issue.origins.length<8&&e.origin)issue.origins.push(e.origin);};
 const top=and(onlyHolds({factions:doc.faction?[doc.faction]:[p.faction],races:doc.races || [p.race],classes:doc.classes || [p.class]},p),scope({condition:doc.condition,condition_format:doc.source==='rxp'?'rxp':doc.source==='wisp'?'wisp':'journey'},p,ctx));
 if(top===false || (doc.coverage && doc.coverage[0]>p.targetLevel))return {id:doc.id,source:doc.source,status:'not_applicable',schemaErrors,issues:[],counts:{}};
 if(doc.source==='rxp' && doc.defaultFor && doc.defaultFor.length && doc.coverage[0]===1 && !doc.defaultFor.some(c=>rxp(c,ctx)!==false))return {id:doc.id,source:doc.source,status:'not_applicable',schemaErrors,issues:[],counts:{}};
 let optional=0,profileSkipped=0,deferred=0;const optionalQuests=new Set();
 for(const g of doc.groups)for(const e of g.events || [])if(e.t==='accept'&&e.optional&&scope(e,p,ctx)!==false)optionalQuests.add(e.q);
 const excludes=new Map(world && world.faction===p.faction?(world.excluded || []).map(q=>[q.id,q]):[]),worldQuests=new Map(world && world.faction===p.faction?(world.quests || []).map(q=>[q.id,q]):[]);
 for(const g of doc.groups){
  if((g.flags || []).some(f=>(f==='hardcore'&&!p.hardcore)||(f==='softcore'&&p.hardcore)))continue;
  let gs=and(top,scope(g,p,ctx));if(gs===false)continue;
  for(const e of g.events || []){
   if((!p.withOptional&&(e.optional || optionalQuests.has(e.q)))||e.recoveryOnly||e.advisory){optional++;continue;}
   const allowed=and(gs,scope(e,p,ctx));if(allowed===false||!compatible(quests[e.q],p)){profileSkipped++;continue;}
   if(allowed===undefined){deferred++;add('dynamic_condition_execution_missing',e);}
   if(e.skipIfMissing||e.skipIfIncomplete)add('dynamic_condition_execution_missing',e,{skipIfMissing:!!e.skipIfMissing,skipIfIncomplete:!!e.skipIfIncomplete});
   rows.push(e);
   for(const flag of g.flags || [])if(/^(completewith|sticky|loop|requiresquest)(?:\s|$)/.test(flag))add('route_control_execution_missing',e,{flag});
   if(e.t==='unsupported')add('unsupported_source_directive',e);
   else if(!REPLAY_SUPPORTED.has(e.t))add('event_execution_missing',e);
   if(e.t==='complete'&&e.obj!=null)add('objective_execution_mapping_missing',e);
   if(!e.q)continue;
   if((sharedFacts.conflicts || []).some(c=>c.quest===e.q))add('shared_fact_conflict',e);
   const q=quests[e.q];
   if(!q){add('quest_definition_missing',e);continue;}
   const ex=excludes.get(e.q);if(ex)add(ex.category?'content_outside_profile':'quest_location_or_work_missing',e,{reason:ex.reason,category:ex.category || null});
   if(e.t==='turnin'&&!Number.isFinite(q.xp))add('quest_xp_missing',e);
   if(['complete','turnin'].includes(e.t)){
    const indices=e.t==='complete'&&e.obj!=null?[e.obj-1]:(q.objectives || []).map((_,i)=>i);
    for(const i of indices){const obj=(q.objectives || [])[i];if(!obj){add('objective_reference_missing',e,{index:i+1});continue;}
     if(obj.count>0)continue;
     // The backend already represents a friendly conversation as one use;
     // it requires no invented kill quantity in the input audit.
     const w=worldQuests.get(e.q),work=w?(w.tasks || []).flatMap(t=>t.work || []):[];
     if(obj.type==='kill'&&work.length&&work.every(x=>x[0]==='use'))continue;
     add('objective_count_missing',e,{index:i+1,target:obj.id,type:obj.type});
    }
   }
  }
 }
 return {id:doc.id,source:doc.source,label:doc.label,coverage:doc.coverage,status:schemaErrors.length?'invalid_contract':issues.size?'blocked':'ready_for_replay',timeTo:null,rankingEligible:false,counts:{selectedEvents:rows.length,optionalSkipped:optional,profileSkipped,deferredConditions:deferred},schemaErrors,issues:[...issues.values()]};
}
module.exports={profile,compatible,hasBit,context,expression,rxp,wisp,scope,validate,check,KIND,REPLAY_SUPPORTED,CLASS_BITS,RACES};
