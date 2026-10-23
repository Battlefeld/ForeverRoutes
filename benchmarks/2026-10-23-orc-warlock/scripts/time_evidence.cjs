'use strict';
const SOURCES=['rxp','journey','wisp','polestar'];
const timeLedger=require('./time_ledger.cjs');
function accounting(row){
 const total=row.endSeconds,travel=row.travelSeconds,grind=row.grindSeconds;
 const workAndRecovery=total-travel-grind;
 const issues=[];
 if(![total,travel,grind,workAndRecovery].every(Number.isFinite))issues.push('nonfinite_time');
 if([total,travel,grind,workAndRecovery].some(x=>x < -1e-7))issues.push('negative_time_component');
 if(!row.costLedger)issues.push('action_ledger_missing');
 else issues.push(...timeLedger.validate(row.costLedger,{seconds:total,travelSeconds:travel,grindSeconds:grind}).issues);
 const workDetail=row.combatBook?{combatSeconds:row.combatBook.combatSeconds,recoverySeconds:row.combatBook.recoverySeconds,betweenEnemiesSeconds:row.combatBook.spacingSeconds,alreadyIncludedInTotal:true}:null;
 return {totalSeconds:total,travelSeconds:travel,grindSeconds:grind,questWorkDialogueRecoveryAndOverheadSeconds:workAndRecovery,workDetail,componentsSumSeconds:travel+grind+workAndRecovery,closed:issues.length===0,issues,limits:[row.combatBook?'Stateful combat and paid recovery are separated, but not calibrated against recorded play':'Combat and recovery are combined in the current engine; no separate measured mana/health recovery ledger','Trainer/vendor/repair costs still include a generic level-up overhead where an explicit adapter is absent','Travel uses the common detour prior; recorded movement and automatic quest travel must not be compared as equivalent workloads']};
}
function rankingEligibility(report,{target=30}={}){
 const reasons=[],rows=report.rows||[];
 if(rows.length!==4||new Set(rows.map(r=>r.id)).size!==4||SOURCES.some(id=>!rows.some(r=>r.id===id)))reasons.push('four_unique_routes_required');
 if(report.mode==='estimated_exploratory')reasons.push('guessed_missing_quest_work');
 for(const row of rows){
  if(row.endLevel<target||!Number.isFinite(row.timeTo?.[target])||row.timeTo[target]<=0)reasons.push(row.id+':target_not_measured');
  if(!row.originalCoverage||row.originalCoverage[1]<target||row.generatedContinuation?.length)reasons.push(row.id+':generated_or_incomplete_source');
  if(Object.values(row.blockers||{}).some(v=>Array.isArray(v)?v.length>0:v!==0))reasons.push(row.id+':replay_incomplete');
  if(!accounting(row).closed)reasons.push(row.id+':time_accounting_invalid');
  const actual=row.sourceProfile;
  if(!actual||actual.class!==report.profile?.class||actual.race!==report.profile?.race||actual.faction!==report.profile?.faction)reasons.push(row.id+':profile_mismatch');
 }
 // A fitted parameter file alone does not prove a class was validated to 30.
 // Only a supported measurement adapter can supply this evidence. The current
 // prior-based engine must not self-certify it from a test passing or a label.
 reasons.push(report.timeModel?.kind==='isolated_foreversim_stateful_diagnostic'?'stateful_combat_not_calibrated_for_1_30':'class_combat_and_recovery_measurements_not_integrated');
 return {eligible:false,reasons:[...new Set(reasons)],winner:null,rankingPublished:false,target,profile:report.profile,provisionalTimesOnly:true};
}
module.exports={accounting,rankingEligibility,SOURCES};
