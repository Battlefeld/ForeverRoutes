'use strict';
// Extend the compiled run's geometry before constructing its evaluator.
// No synthetic quests, extra XP, network nodes or engine source changes.
function addMovementPoints(run,source,frames,{nearK,islands}){
 const points=new Map(),world=run.world;
 if(source.id!=='wisp')return points;
 const old=run.ptList.length,extra=[];
 for(const g of source.groups)for(const e of g.events||[]){
  if(e.t!=='move'||!Number.isFinite(e.map)||!Number.isFinite(e.x)||!Number.isFinite(e.y)||e.x<0||e.x>100||e.y<0||e.y>100)continue;
  const f=frames.forever?.[e.map]||frames.era?.[e.map];if(!f)continue;
  const p=[f.c,f.c*10+(islands[e.map]||0),Math.round((f.top+(f.bottom-f.top)*e.y/100)*10)/10,Math.round((f.left+(f.right-f.left)*e.x/100)*10)/10,e.map];
  let w=world.points.findIndex(v=>v.every((x,i)=>x===p[i]));
  if(w<0){w=world.points.length;world.points.push(p);}
  let at=run.ptIndex.get(w);
  if(at==null){at=run.ptList.length;run.ptIndex.set(w,at);run.ptList.push(w);extra.push({at,p});}
  points.set(e,at);
 }
 const size=run.ptList.length;
 if(size===old)return points;
 const extend=(name,n,fill=0)=>{const a=run[name],b=new a.constructor(n);if(fill)b.fill(fill);b.set(a);run[name]=b;};
 for(const key of ['pC','pR','pX','pY','pMap','pZone','pCell'])extend(key,size);
 for(const key of ['pFp','pInn'])extend(key,size,-1);
 extend('pNear',size*nearK,-1);extend('pNearD',size*nearK);
 const zoneIds=new Map(),cellIds=new Map();
 for(let i=0;i<old;i++){zoneIds.set(run.pMap[i],run.pZone[i]);cellIds.set(`${run.pR[i]}:${Math.floor(run.pX[i]/250)}:${Math.floor(run.pY[i]/250)}`,run.pCell[i]);}
 for(const {at:i,p} of extra){
  for(const [key,k] of [['pC',0],['pR',1],['pX',2],['pY',3],['pMap',4]])run[key][i]=p[k];
  if(!zoneIds.has(p[4]))zoneIds.set(p[4],run.zoneCount++);run.pZone[i]=zoneIds.get(p[4]);
  const cell=`${p[1]}:${Math.floor(p[2]/250)}:${Math.floor(p[3]/250)}`;
  if(!cellIds.has(cell))cellIds.set(cell,run.cellCount++);run.pCell[i]=cellIds.get(cell);
  const near=[];
  for(let n=0;n<run.N;n++)if(run.nodeR[n]===p[1])near.push([Math.hypot(run.nodeX[n]-p[2],run.nodeY[n]-p[3]),n]);
  near.sort((a,b)=>a[0]-b[0]||a[1]-b[1]);
  for(let k=0;k<Math.min(nearK,near.length);k++){run.pNear[i*nearK+k]=near[k][1];run.pNearD[i*nearK+k]=near[k][0];}
  const fp=near.find(([d,n])=>d<=200&&run.nodes[n].kind==='fp');if(fp)run.pFp[i]=fp[1];
  let distance=200;
  for(let n=0;n<run.innPt.length;n++){
   const at=run.innPt[n];if(run.pR[at]!==p[1])continue;const d=Math.hypot(run.pX[at]-p[2],run.pY[at]-p[3]);
   if(d<=distance){distance=d;run.pInn[i]=n;}
  }
 }
 // Saved snapshots currently have no routed walk matrix. Refuse silent loss
 // if a caller later supplies one with the old dimensions.
 if(run.walkRun)throw Error('Movement-point extension needs a rebuilt routed walk matrix');
 return points;
}
module.exports={addMovementPoints};
