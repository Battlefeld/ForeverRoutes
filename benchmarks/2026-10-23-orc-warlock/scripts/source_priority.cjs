'use strict';
const ORDER=['foreverroutes','questiedb','pfquest','cmangos','wago_db2','foreverdungeonjournal','warcraftdb','foreverdb','wowhead_forever','estimated'];
function resolveField(field,candidates,{estimate}={}){
 const ranked=(candidates||[]).filter(x=>x&&x.value!==null&&x.value!==undefined&&!(typeof x.value==='number'&&!Number.isFinite(x.value))).map(x=>({...x,priority:ORDER.indexOf(x.source)})).filter(x=>x.priority>=0&&x.source!=='estimated').sort((a,b)=>a.priority-b.priority);
 if(ranked.length)return {field,value:ranked[0].value,source:ranked[0].source,estimated:false,conflicts:ranked.slice(1).filter(x=>JSON.stringify(x.value)!==JSON.stringify(ranked[0].value)).map(x=>({source:x.source,value:x.value}))};
 if(typeof estimate==='function'){const value=estimate();if(value!==undefined&&value!==null&&!(typeof value==='number'&&!Number.isFinite(value)))return {field,value,source:'estimated',estimated:true,conflicts:[]};}
 return {field,value:null,source:null,estimated:false,unresolved:true,conflicts:[]};
}
function resolveRecord(fields,candidatesByField,estimators={}){
 const values={},provenance={},unresolved=[];
 for(const field of fields){const r=resolveField(field,candidatesByField[field],{estimate:estimators[field]});values[field]=r.value;provenance[field]={source:r.source,estimated:r.estimated,conflicts:r.conflicts};if(r.unresolved)unresolved.push(field);}
 return {values,provenance,unresolved};
}
module.exports={ORDER,resolveField,resolveRecord};
