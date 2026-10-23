'use strict';
const fs=require('node:fs'),path=require('node:path'),zlib=require('node:zlib');
const root=path.resolve(__dirname,'..');
const independent=path.resolve(root,'../2026-10-23-independent-route-checks');
const world=JSON.parse(fs.readFileSync(path.join(root,'.comparison/cache/world-horde.json'),'utf8'));
const questData=JSON.parse(zlib.gunzipSync(fs.readFileSync(path.join(independent,'data/quests.json.gz'))));
const facts=JSON.parse(fs.readFileSync(path.join(independent,'data/objective-facts.json'),'utf8'));
const last=JSON.parse(fs.readFileSync(path.join(root,'.comparison/results/comparison.json'),'utf8'));
const ids=new Set(last.rows.flatMap(r=>r.blockers.unknownObjectiveCounts));
for(const id of [...ids].sort((a,b)=>a-b)){
 const q=world.quests.find(q=>q.id===id),rec=questData[id];
 console.log('OBJECTIVE_INSPECT '+JSON.stringify({id,name:q?.title||rec?.name,objectives:rec?.objectives,rawKeys:rec?Object.keys(rec):null,worldTasks:q?.tasks?.map(t=>t.work),factualObjectives:facts.filled.filter(f=>f.quest===id)}));
}
