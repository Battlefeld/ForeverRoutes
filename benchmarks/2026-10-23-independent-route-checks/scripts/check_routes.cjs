'use strict';
// Validate every route independently; never turn unsupported instructions into free time.
const fs=require('node:fs'),path=require('node:path'),crypto=require('node:crypto'),zlib=require('node:zlib');
const {profile,check}=require('./lib/route_contract.cjs');
const root=path.resolve(__dirname,'..');
function args(argv){const o={},valued=new Set(['--catalog','--quests','--world','--facts','--scenario','--race','--class','--faction','--to','--source','--guides','--output']);for(let i=0;i<argv.length;i++){const key=argv[i];if(['--with-optional','--hardcore'].includes(key)){o[key.slice(2)]=true;continue;}if(!valued.has(key))throw Error('Unexpected argument '+key);if(i+1===argv.length||argv[i+1].startsWith('--'))throw Error('Value required for '+key);o[key.slice(2)]=argv[++i];}return o;}
const load=p=>JSON.parse(p.endsWith('.gz')?zlib.gunzipSync(fs.readFileSync(p)):fs.readFileSync(p,'utf8'));
const digest=x=>crypto.createHash('sha256').update(JSON.stringify(x)).digest('hex');
function run(o){
 const catalogFile=path.resolve(o.catalog || path.join(root,'.routecheck/events/catalog.json'));
 const catalog=load(catalogFile);
 if(catalog.schema!=='foreverroutes.guide_events.v2'||!Array.isArray(catalog.guides))throw Error('Not a v2 guide catalog');
 if(new Set(catalog.guides.map(g=>g.id)).size!==catalog.guides.length)throw Error('Duplicate guide ids');
 const scenario=o.scenario?load(path.resolve(o.scenario)):{};
 const p=profile({...scenario,race:o.race || scenario.race,class:o.class || scenario.class,faction:o.faction || scenario.faction,targetLevel:Number(o.to || scenario.targetLevel || 30),withOptional:!!(o['with-optional'] || scenario.withOptional),hardcore:!!(o.hardcore || scenario.hardcore)});
 const quests=load(path.resolve(o.quests || path.join(root,'.routecheck/data/quests.json')));
 const worldFile=path.resolve(o.world || path.join(root,'.comparison/cache/world-horde.json'));
 const world=fs.existsSync(worldFile)?load(worldFile):null;
 const factFile=path.resolve(o.facts || path.join(root,'.routecheck/data/objective-facts.json'));
 const facts=fs.existsSync(factFile)?load(factFile):{};
 const ids=o.guides?new Set(o.guides.split(',')):null;
 const docs=catalog.guides.filter(g=>(!o.source||g.source===o.source)&&(!ids||ids.has(g.id)||ids.has(g.guideId)));
 if(!docs.length)throw Error('No selected guides');
 const rows=docs.map(g=>{const before=digest(g),r=check(g,p,quests,world,facts);if(before!==digest(g))throw Error('Route was mutated: '+g.id);return {...r,routeFingerprint:before,routeOrderPreserved:true};});
 const counts={};const issueCounts={};for(const r of rows){counts[r.status]=(counts[r.status] || 0)+1;for(const i of r.issues)issueCounts[i.code]=(issueCounts[i.code] || 0)+i.occurrences;}
 const report={schema:'foreverroutes.route_check.v2',generatedAt:new Date().toISOString(),profile:p,catalogSha256:digest(catalog),sourceGuideCount:catalog.guides.length,selectedGuideCount:rows.length,sources:[...new Set(rows.map(r=>r.source))],routeOrderPreserved:rows.every(r=>r.routeOrderPreserved),independentRoutes:true,rankingPublished:false,worldAuditAvailable:!!world&&world.faction===p.faction,statusCounts:counts,issueCounts,sharedFacts:{filledCount:facts.filled_count,conflicts:facts.conflicts},backendCapabilities:'Quest replay exists; required source-native travel, inventory, profession and objective-control actions still require execution adapters',rows};
 const output=path.resolve(o.output || path.join(root,'.routecheck/reports/check-'+p.faction.toLowerCase()+'-'+p.race.toLowerCase()+'-'+p.class.toLowerCase()+'.json'));
 fs.mkdirSync(path.dirname(output),{recursive:true});fs.writeFileSync(output,JSON.stringify(report,null,1)+'\n');
 console.log(JSON.stringify({output,profile:{race:p.race,class:p.class,faction:p.faction},guides:rows.length,status:counts,orderPreserved:report.routeOrderPreserved,issues:issueCounts}));
 return report;
}
if(require.main===module)try{run(args(process.argv.slice(2)));}catch(e){console.error(e.message);process.exitCode=1;}
module.exports={args,run};
