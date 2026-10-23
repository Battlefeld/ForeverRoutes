'use strict';
const fs=require('node:fs'),path=require('node:path'),{spawnSync}=require('node:child_process');
const {CLASSES,resolveProfile}=require('./class_profiles.cjs');
const {accounting,rankingEligibility,SOURCES}=require('./time_evidence.cjs');
const root=path.resolve(__dirname,'..'),script=path.join(__dirname,'run_route_comparison.cjs');
const index=process.argv.indexOf('--output-dir'),out=index<0?path.join(root,'.comparison/class-matrix'):path.resolve(process.argv[index+1]);
const targetIndex=process.argv.indexOf('--target-level'),target=targetIndex<0?30:Number(process.argv[targetIndex+1]);
if(!Number.isInteger(target)||target<2||target>30)throw Error('Invalid matrix target level');
fs.mkdirSync(out,{recursive:true});
const matrix={schema:'foreverroutes.class_matrix.v1',target,classes:CLASSES.length,sourceCount:4,requestedCells:CLASSES.length*4,diagnosticContinuations:process.argv.includes('--diagnostic-continuations'),profiles:[],winnerPolicy:'No winner from incomplete, generated, uncalibrated or differently modeled workloads',rankingPublished:false};
for(const cls of CLASSES){
 const profile=resolveProfile({class:cls,targetLevel:target}),dir=path.join(out,cls.toLowerCase());
 const world=path.join(root,'.comparison/matrix-worlds','world-'+profile.faction.toLowerCase()+'.json.gz');
 const args=[script,'--saved-inputs','--class',cls,'--race',profile.race,'--faction',profile.faction,'--world-file',world,'--target-level',String(target),'--output-dir',dir];
 if(process.argv.includes('--diagnostic-continuations'))args.push('--extend-to-30');
 const result=spawnSync(process.execPath,args,{encoding:'utf8',timeout:120000,maxBuffer:4*1024*1024});
 let entry;
 const file=path.join(dir,'comparison.json');
 if(result.status!==0||!fs.existsSync(file)){
  const message=result.error?.message||result.stderr.trim().split('\n').slice(0,4).join('\n')||'Replay process failed';
  entry={profile,status:'execution_failed',reason:message,cells:SOURCES.map(id=>({id,status:'not_measured',timeToTarget:null})),ranking:{eligible:false,winner:null,reasons:['execution_failed']}};
 }else{
  const report=JSON.parse(fs.readFileSync(file));
  entry={profile,status:report.acceptancePassed?'replay_complete_time_unvalidated':'replay_incomplete',timeModel:report.timeModel,worldInput:report.worldInput,cells:report.rows.map(row=>({id:row.id,status:row.status,originalCoverage:row.originalCoverage,endLevel:row.endLevel,timeToTarget:row.timeTo[target]??null,generatedQuestCount:row.generatedContinuation.length,generatedActionFailures:row.blockers.eventFailures.filter(e=>e.generated).length,blockerCounts:Object.fromEntries(Object.entries(row.blockers).map(([key,v])=>[key,Array.isArray(v)?v.length:v])),accounting:accounting(row),costLedger:row.costLedger,inputProvenance:row.inputProvenance})),ranking:rankingEligibility(report,{target})};
 }
 matrix.profiles.push(entry);fs.writeFileSync(path.join(out,'matrix.json'),JSON.stringify(matrix,null,2)+'\n');
 console.log(JSON.stringify({class:cls,race:profile.race,faction:profile.faction,status:entry.status,cells:entry.cells.length,winner:null}));
}
matrix.finishedAt=new Date().toISOString();matrix.actualCells=matrix.profiles.reduce((n,p)=>n+p.cells.length,0);
matrix.executionFailures=matrix.profiles.filter(p=>p.status==='execution_failed').length;
matrix.targetReachedCells=matrix.profiles.flatMap(p=>p.cells).filter(c=>c.endLevel>=target&&Number.isFinite(c.timeToTarget)).length;
matrix.validReplayCells=matrix.profiles.flatMap(p=>p.cells).filter(c=>c.status==='provisional_model_complete').length;
matrix.validatedWinnerCount=matrix.profiles.filter(p=>p.ranking.eligible).length;
matrix.status=matrix.executionFailures?'execution_failed':matrix.validatedWinnerCount===CLASSES.length?'validated':matrix.targetReachedCells===matrix.requestedCells?'diagnostic_target_reached_unvalidated':'diagnostic_target_incomplete';
fs.writeFileSync(path.join(out,'matrix.json'),JSON.stringify(matrix,null,2)+'\n');
if(matrix.executionFailures)process.exitCode=1;
else if((process.argv.includes('--require-target')&&matrix.targetReachedCells!==matrix.requestedCells)||(process.argv.includes('--require-validated')&&!matrix.profiles.every(p=>p.ranking.eligible)))process.exitCode=2;
