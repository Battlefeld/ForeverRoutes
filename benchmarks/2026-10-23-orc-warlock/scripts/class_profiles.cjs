'use strict';
const {profile}=require('../../2026-10-23-independent-route-checks/scripts/lib/route_contract.cjs');
const CLASSES=['Warrior','Paladin','Hunter','Rogue','Priest','Shaman','Mage','Warlock','Druid'];
// Conservative, playable reference profiles. Compare routes WITHIN each profile,
// never a Human Paladin's route time against a Tauren Druid's route time.
const REFERENCE_RACES={Warrior:'Orc',Paladin:'Human',Hunter:'Orc',Rogue:'Orc',Priest:'Troll',Shaman:'Orc',Mage:'Troll',Warlock:'Orc',Druid:'Tauren'};
const CLASS_RACES={Human:['Warrior','Paladin','Rogue','Priest','Mage','Warlock'],Orc:['Warrior','Hunter','Rogue','Shaman','Warlock'],Troll:['Warrior','Hunter','Rogue','Priest','Shaman','Mage'],Tauren:['Warrior','Hunter','Shaman','Druid'],Undead:['Warrior','Rogue','Priest','Mage','Warlock'],Dwarf:['Warrior','Paladin','Hunter','Rogue','Priest'],Gnome:['Warrior','Rogue','Mage','Warlock'],NightElf:['Warrior','Hunter','Rogue','Priest','Druid']};
function resolveProfile(input={}){
 const cls=CLASSES.find(c=>c.toLowerCase()===String(input.class||'Warlock').toLowerCase());
 if(!cls)throw Error('Unknown class: '+input.class);
 const race=input.race||REFERENCE_RACES[cls];
 const p=profile({...input,class:cls,race,guideFlags:input.guideFlags||[],variables:input.variables||{HORDETB20:false}});
 if(!CLASS_RACES[p.race]?.includes(p.class))throw Error('Race/class combination lacks a supported reference profile: '+p.race+' '+p.class);
 if(!input.guideFlags&&p.faction==='Horde')p.guideFlags=[p.race==='Undead'?'H12SF':'H12B'];
 return p;
}
module.exports={CLASSES,REFERENCE_RACES,CLASS_RACES,resolveProfile};
