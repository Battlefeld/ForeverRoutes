'use strict';
const fs=require('node:fs'),path=require('node:path');
const f=path.resolve(__dirname,'../.comparison/results/comparison-estimated.json');
const r=JSON.parse(fs.readFileSync(f,'utf8'));
const rows=r.modelComparison||[];
console.log('## ForeverRoutes – estimates (NOT a validated speed ranking)');
console.log('');
console.log('Original RestedXP, Journey, ForeverWisp and Polestar guide events are separate and unchanged. Shared data is applied in documented priority order. Blank means that guide does not cover the milestone.');
console.log('');
console.log('| Milestone | RestedXP | Journey | ForeverWisp | Polestar |');
console.log('|---|---:|---:|---:|---:|');
const fmt=n=>!Number.isFinite(n)?'–':(n/3600).toFixed(2)+' h';
for(const m of rows){const time=id=>m.entries.find(x=>x.id===id)?.seconds;console.log('| '+m.level+' | '+['rxp','journey','wisp','polestar'].map(id=>fmt(time(id))).join(' | ')+' |');}
console.log('');
console.log('### Source precedence');
console.log('ForeverRoutes → QuestieDB → pfQuest → CMaNGOS → Wago/DB2 → ForeverDungeonJournal → WarcraftDB → ForeverDB → Wowhead Forever → estimated. Only available locally merged source snapshots were actually used; listing a source does not mean a live import took place.');
console.log('');
console.log('### Uncertainty accounting');
console.log('XP model: '+JSON.stringify(r.modelXpPolicy?.counts||{}));
for(const x of r.rows){console.log('- '+x.label+': '+x.blockers.unknownQuestIds.length+' world-unmapped quests (proxy estimates), '+x.blockers.unknownObjectiveCounts.length+' quests with fallback target counts, '+x.blockers.skippedAccepts.length+' game-prerequisite accept failures, '+x.blockers.unresolvedConditions.length+' uncertain conditions');}
console.log('');
console.log('**Warning:** Unmodelled decisions, location guesses, source gaps and partial route coverage make these exploratory estimates, not verified /played fastest-route results.');
