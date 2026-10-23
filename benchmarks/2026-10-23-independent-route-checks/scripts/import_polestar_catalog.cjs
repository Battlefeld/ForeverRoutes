'use strict';
const fs=require('node:fs'),path=require('node:path');
const root=path.resolve(__dirname,'..');
const external=path.resolve(process.env.POLESTAR_ROOT || path.join(root,'../polestar'));
const {compileProject}=require(path.join(external,'tools/lib/dsl/project'));
const p=compileProject({packDir:path.join(external,'guides'),dataDir:path.join(root,'.comparison/data')});
const file=path.join(root,'.routecheck/events/catalog.json');
const catalog=JSON.parse(fs.readFileSync(file,'utf8'));
catalog.guides=catalog.guides.filter(g=>g.source!=='polestar');
const docs=[];
for(const g of p.guides.filter(g=>g.category==='Leveling')){
 const groups=[];
 for(const [i,s] of (g.steps || []).entries()){
  const events=[];
  for(const goal of s.goals || []){
   if(['note','text'].includes(goal.t))continue;
   const e={t:goal.quest!=null?(goal.t==='accept'||goal.t==='turnin'?goal.t:'complete'):({goto:'move',fly:'transport',sethome:'bind',getfp:'learn_flight',vendor:'vendor',hearth:'hearth',train:'train',level:'level',grind:'grind',skill:'profession',use:'use_item',buy:'vendor',talk:'service',learn:'train'}[goal.t] || 'unsupported'),directive:goal.t,optional:!!goal.optional,only:[s.only,goal.only].filter(Boolean),origin:{path:g.id+'.guide',step:i+1}};
   if(goal.quest!=null)e.q=goal.quest;
   if(goal.oi!=null)e.obj=goal.oi;
   if(goal.t==='level')Object.assign(e,{level:goal.level,xp:0});
   for(const k of ['npc','object','map','x','y','count','spell','item'])if(goal[k]!=null)e[k]=goal[k];
   events.push(e);
  }
  if(events.length)groups.push({guide:g.id,step:i+1,events});
 }
 docs.push({schema:'foreverroutes.guide_events.v2',id:'polestar:'+g.id,source:'polestar',guideId:g.id,label:g.title,faction:g.faction,races:g.races || null,classes:g.classes || null,coverage:[g.minLevel,g.maxLevel],nextGuide:g.next,nextChoices:g.nextChoices,groups,sourceRevision:'eb7b47840f032bf711a3c8d7b1df9f3f6551b990'});
}
catalog.guides.push(...docs);
if(new Set(catalog.guides.map(g=>g.id)).size!==catalog.guides.length)throw Error('Duplicate guide ids');
fs.writeFileSync(file,JSON.stringify(catalog)+'\n');
console.log(JSON.stringify({polestarGuides:docs.length,totalGuides:catalog.guides.length}));
