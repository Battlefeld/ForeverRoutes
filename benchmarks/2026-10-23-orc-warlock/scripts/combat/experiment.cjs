'use strict';
const {run}=require('./bridge.cjs');
function quantile(values,p){const a=[...values].sort((x,y)=>x-y);if(!a.length)return null;const index=(a.length-1)*p,lo=Math.floor(index),hi=Math.ceil(index);return a[lo]+(a[hi]-a[lo])*(index-lo);}
function compare(plan,{lua,seeds=[1,2,3,4,5]}={}){
 if(!Array.isArray(plan.choices)||!plan.choices.length||new Set(plan.choices.map(c=>c.id)).size!==plan.choices.length)throw Error('Distinct choice IDs required');
 if(!Array.isArray(seeds)||!seeds.length||seeds.some(x=>!Number.isInteger(x)||x<0||x>0xffffffff))throw Error('Explicit uint32 seeds required');
 const rows=[];
 for(const choice of plan.choices){
  const samples=[],failures=[];
  for(const seed of seeds){
   const job=structuredClone(plan.job);let ix=0;
   for(const event of job.events)if(event.kind==='fight'){event.choice=choice.rotation;event.seed=(seed+Math.imul(ix++,2654435761))>>>0;}
   try{const result=run(job,{lua});if(result.stopped)failures.push({seed,reason:result.stopped,paidSeconds:result.totalSeconds});else samples.push({seed,seconds:result.totalSeconds,combatSeconds:result.combatSeconds,recoverySeconds:result.idleSeconds});}
   catch(e){failures.push({seed,reason:e.message});}
  }
  const times=samples.map(s=>s.seconds);
  rows.push({id:choice.id,samples,failures,complete:failures.length===0&&samples.length===seeds.length,medianSeconds:quantile(times,.5),p10Seconds:quantile(times,.1),p90Seconds:quantile(times,.9)});
 }
 const complete=rows.every(r=>r.complete),ordered=complete?[...rows].sort((a,b)=>a.medianSeconds-b.medianSeconds):[];
 const best=ordered[0],tie=best&&ordered[1]&&Math.abs(best.medianSeconds-ordered[1].medianSeconds)<1e-9;
 return {schema:'foreverroutes.combat_experiment.v1',seeds,rows,complete,diagnosticFastestChoice:complete&&!tie?best.id:null,validated:false,rankingEligible:false,routeWinner:null,meaning:'Fastest tested rotation within the supplied synthetic or recorded encounter sequence; not a validated route ranking'};
}
module.exports={compare,quantile};
