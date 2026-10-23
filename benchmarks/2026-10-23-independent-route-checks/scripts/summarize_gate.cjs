'use strict';
// Turn raw checks into an honest start report without combining or editing guides.
const fs=require('node:fs'),path=require('node:path');
const root=path.resolve(__dirname,'..');
const runRoot=path.resolve(root,'../2026-10-23-orc-warlock');
const read=f=>JSON.parse(fs.readFileSync(f,'utf8'));
function summarize(reports,comparison){
 if(!comparison || !Array.isArray(comparison.rows) || comparison.rows.length!==4)throw Error('Expected all four independent diagnostic replay outputs');
 const ids=['rxp','journey','wisp','polestar'];
 if(new Set(comparison.rows.map(x=>x.id)).size!==4||!ids.every(x=>comparison.rows.some(r=>r.id===x)))throw Error('Missing or duplicated route');
 const expected=134;
 for(const r of reports){
  if(!r.independentRoutes||!r.routeOrderPreserved||r.sourceGuideCount!==expected)throw Error('Guide preservation/coverage check failed');
  if(r.selectedGuideCount!==expected)throw Error('Preflight did not inspect all guide chapters: '+r.selectedGuideCount);
 }
 const guideRows=reports.flatMap(r=>r.rows);
 const compareRows=comparison.rows.map(r=>({id:r.id,label:r.label,coverage:r.declaredCoverage,diagnosticLevel:r.endLevel,diagnosticSeconds:r.endSeconds,timeTo:r.timeTo,blockers:Object.fromEntries(Object.entries(r.blockers||{}).map(([k,v])=>[k,Array.isArray(v)?v.length:v]))}));
 const compatibleMilestones=[6,12,20,30].map(level=>{
  const present=compareRows.filter(x=>x.coverage[1]>=level&&x.timeTo?.[String(level)]!=null);
  // Only compare where each guide's declared coverage includes the level AND
  // each replay has zero unresolved blockers. A diagnostic time is not a ranking.
  const common=present.length===4;
  const eligible=common&&comparison.rows.every(r=>r.status==='provisional_model_complete')&&reports.every(r=>(r.statusCounts||{}).blocked===0 && (r.statusCounts||{}).invalid_contract===0);
  return {level,routesWithDiagnosticCheckpoint:present.map(x=>x.id),sharedFourRouteComparison:common,rankingEligible:eligible,rankingPublished:false};
 });
 return {
  schema:'foreverroutes.simulation_start_gate.v1',
  generatedAt:new Date().toISOString(),
  sourceRevision:comparison.engine,
  importedGuides:expected,
  routesRemainIndependent:true,
  originalsEdited:false,
  zeroXpPolicy:'explicit rewards and 160 user-approved provisional assumptions; other unknown XP stays blocked',
  reports:reports.map(r=>({faction:r.profile.faction,race:r.profile.race,class:r.profile.class,statusCounts:r.statusCounts,issueCounts:r.issueCounts,worldAuditAvailable:r.worldAuditAvailable})),
  diagnostic:compareRows,
  milestones:compatibleMilestones,
  rankingPublished:false,
  readyForFullComparativeRanking:compatibleMilestones.every(x=>x.rankingEligible),
  blockersNext:'unknown quest locations, objective counts, nonquest travel/inventory/condition semantics and missing guide coverage cannot be priced as zero'
 };
}
if(require.main===module){
 const input=process.argv[2]||path.join(root,'reports');
 const reports=['replayed-horde.json','replayed-alliance.json'].map(n=>read(path.join(input,n)));
 const comp=read(path.join(runRoot,'.comparison/results/comparison.json'));
 const result=summarize(reports,comp);
 const output=process.argv[3]||path.join(input,'simulation-start-gate.json');
 fs.writeFileSync(output,JSON.stringify(result,null,2)+'\n');
 console.log(JSON.stringify({output,chapters:result.importedGuides,rankingPublished:result.rankingPublished,milestones:result.milestones,blockerTypes:result.reports.map(x=>x.issueCounts)}));
}
module.exports={summarize};
