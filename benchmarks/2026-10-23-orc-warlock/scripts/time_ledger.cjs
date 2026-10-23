'use strict';
// Cost attribution only: never adds, discounts, or estimates engine time.
function clock(ev){return {seconds:ev.time,travelSeconds:ev.travelSec,grindSeconds:ev.grindSec};}
function ledger(){return {schema:'foreverroutes.time_ledger.v1',byAction:{},byOrigin:{}};}
function record(book,before,after,action,generated=false){
 const delta=Object.fromEntries(Object.keys(before).map(k=>[k,after[k]-before[k]]));
 for(const bucket of [book.byAction[action]??=empty(),book.byOrigin[generated?'generated':'source']??=empty()]){
  bucket.events++;for(const k of Object.keys(delta))bucket[k]+=delta[k];
 }
}
function empty(){return {events:0,seconds:0,travelSeconds:0,grindSeconds:0};}
function validate(book,total){
 const issues=[];
 for(const [axis,groups] of Object.entries({action:book.byAction,origin:book.byOrigin})){
  for(const [key,value] of Object.entries(total)){
   const sum=Object.values(groups).reduce((n,g)=>n+g[key],0);
   if(!Number.isFinite(sum)||!Number.isFinite(value)||Math.abs(sum-value)>Math.max(1e-6,Math.abs(value)*1e-10))issues.push(axis+':'+key+':unaccounted');
  }
  for(const g of Object.values(groups))if(Object.values(g).some(v=>!Number.isFinite(v)||v < -1e-7))issues.push(axis+':invalid_component');
 }
 return {closed:issues.length===0,issues};
}
module.exports={clock,ledger,record,validate};
