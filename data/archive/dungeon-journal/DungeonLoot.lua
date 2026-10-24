local _,ns=...
local L={};ns.DungeonLoot=L
local requested={}
local proficiency={[0]=196,[1]=197,[2]=264,[3]=266,[4]=198,[5]=199,[6]=200,[7]=201,[8]=202,[10]=227,[13]=15590,[15]=1180,[16]=2567,[18]=5011,[19]=5009}
local function call(fn,...)
 if type(fn)~='function'then return end
 local ok,a,b,c,d,e,f,g,h,i,j,k,l,m=pcall(fn,...);if ok then return a,b,c,d,e,f,g,h,i,j,k,l,m end
end
-- A class recommendation requires actual usable equipment and live stats.
-- Static item types identify candidates; they do not establish an upgrade.
function L.Usable(id,state)
 local name,link,quality,level,required,_,_,_,location,_,_,class,subclass=call(C_Item and C_Item.GetItemInfo or GetItemInfo,id)
 if not name or not link then
  if not requested[id]then requested[id]=true;call(C_Item and C_Item.RequestLoadItemDataByID,id)end;return nil,'Itemdaten werden geladen'
 end
 if (required or 0)>state.level then return false end
 -- IsUsableItem describes item actions/inventory availability, not proficiency
 -- for a hypothetical boss drop. Inspect current tooltip requirements instead.
 local tooltip=call(C_TooltipInfo and C_TooltipInfo.GetItemByID,id)
 local inspected=tooltip and tooltip.lines and #tooltip.lines>0
 if inspected then
  for _,line in ipairs(tooltip.lines)do
   local c=line.leftColor
   if c and c.r and c.g and c.b and c.r>.9 and c.g<.25 and c.b<.25 then return false end
  end
 end
 if not inspected and class==4 and subclass~=0 and subclass>(ns.GearDB and ns.GearDB.armor[state.class] or 1)then return false end
 if class==2 and not inspected then
  local known=proficiency[subclass] and call(IsPlayerSpell or IsSpellKnown,proficiency[subclass])
  if known==false then return false end
  if known~=true then return nil,'Waffenproficiency noch unbekannt' end
 end
 if not (ns.GearDB and ns.GearDB.slots[location])then return false end
 return true,link,location,name
end
function L.Value(link,location,state)
 local stats=call(C_Item and C_Item.GetItemStats or GetItemStats,link)
 local weights=ns.ClassDB[state.class] and ns.ClassDB[state.class].gear
 if not stats or not weights then return end
 local score=0
 local keys={strength='ITEM_MOD_STRENGTH_SHORT',agility='ITEM_MOD_AGILITY_SHORT',stamina='ITEM_MOD_STAMINA_SHORT',intellect='ITEM_MOD_INTELLECT_SHORT',spirit='ITEM_MOD_SPIRIT_SHORT'}
 for key,stat in pairs(keys)do score=score+(stats[stat] or 0)*(weights[key] or 0)end
 local ranged=location=='INVTYPE_RANGED' or location=='INVTYPE_RANGEDRIGHT'
 local caster=state.class=='MAGE' or state.class=='PRIEST' or state.class=='WARLOCK'
 local dps=(state.class=='HUNTER' and ranged or caster and location=='INVTYPE_RANGEDRIGHT' or not caster and state.class~='HUNTER' and state.class~='DRUID') and (weights.dps or 0) or 0
 score=score+(stats.ITEM_MOD_DAMAGE_PER_SECOND_SHORT or 0)*dps
 if caster or state.class=='DRUID' or state.class=='SHAMAN'then
  score=score+(stats.ITEM_MOD_SPELL_POWER_SHORT or stats.ITEM_MOD_SPELL_DAMAGE_DONE_SHORT or 0)*2
  score=score+(stats.ITEM_MOD_HEALING_DONE_SHORT or 0)*.5
 end
 return score
end
function L.Upgrade(id,state)
 local usable,link,location,name=L.Usable(id,state);if not usable then return nil,usable==nil end
 local value=L.Value(link,location,state);if value==nil then return nil,true end
 if not GetInventoryItemLink then return nil,true end
 local main=call(GetInventoryItemLink,'player',16)
 local mainLocation=main and select(9,call(C_Item and C_Item.GetItemInfo or GetItemInfo,main))
 if ns.GearDB.slots[location][1]==17 and mainLocation=='INVTYPE_2HWEAPON'then return nil end
 local current
 for _,slot in ipairs(ns.GearDB.slots[location])do
  local equipped=call(GetInventoryItemLink,'player',slot)
  local score=equipped and L.Value(equipped,location,state) or 0
  if score==nil then return nil,true end
  if not current or score<current then current=score end
 end
 if location=='INVTYPE_2HWEAPON'then
  local offhand=call(GetInventoryItemLink,'player',17)
  if offhand then
   local offLocation=select(9,call(C_Item and C_Item.GetItemInfo or GetItemInfo,offhand))
   local offValue=L.Value(offhand,offLocation,state);if offValue==nil then return nil,true end
   current=current+offValue
  end
 end
 if value>current and value>0 then return {id=id,name=name,gain=value-current,relativeGain=(value-current)/math.max(1,current),location=location}end
end
function L.Evaluate(package,state)
 local result={bossUpgrades={},questUpgrades={},unknown=0,classQuest=false,reason='Kein belegtes Ausrüstungsupgrade'}
 local seen={}
 for _,id in ipairs(package.definition and package.definition.bossLoot or {})do
  if not seen[id]then seen[id]=true;local u,unknown=L.Upgrade(id,state);if u then result.bossUpgrades[#result.bossUpgrades+1]=u elseif unknown then result.unknown=result.unknown+1 end end
 end
 for _,id in ipairs(package.quests or {})do
  local q=ns.DataProvider.GetQuest(id)
  if q and (q.minLevel or 1)<=state.level and not (state.completed and state.completed[id])then
   if q.classMask and q.classMask~=0 then result.classQuest=true end
   local best
   for _,item in ipairs(q.rewards or {})do local u,unknown=L.Upgrade(item,state);if u and (not best or u.gain>best.gain)then best=u elseif unknown then result.unknown=result.unknown+1 end end
   if best then best.questID=id;result.questUpgrades[#result.questUpgrades+1]=best end
  end
 end
 result.count=#result.bossUpgrades+#result.questUpgrades
 local best
 for _,list in ipairs({result.bossUpgrades,result.questUpgrades})do for _,u in ipairs(list)do if not best or u.relativeGain>best.relativeGain then best=u end end end
 result.best=best
 if best then result.reason=string.format('%s: nutzbares %s-Upgrade (+%.1f; %.0f%% Modellwert)',best.name,state.class,best.gain,best.relativeGain*100)
 elseif result.classQuest then result.reason='Passende Klassenquest im Dungeon'
 elseif result.unknown>0 then result.reason='Lootdaten/Nutzbarkeit noch unvollständig; kein belegtes Upgrade' end
 return result
end
function L.Balanced(package,state,analysis)
 if state.instance and ns.DungeonManager.Match(package.definition,state.instance)then return true,'Bereits im Dungeon' end
 if analysis.accepted>0 then return true,'Dungeonquests bereits angenommen' end
 local result=L.Evaluate(package,state)
 local travel=0;local d=package.definition
 if d.map and state.map and d.map~=state.map and ns.TravelCost then travel=ns.TravelCost(state.map,d.map,state) or 1800 end
 local cost=#analysis.prerequisites
 if result.classQuest and cost<=8 and travel<=900 then return true,result.reason,result end
 if result.best and result.best.relativeGain>=.15 and cost<=4 and travel<=600 then return true,result.reason,result end
 if result.best and result.best.relativeGain>=.35 and cost<=8 and travel<=1200 then return true,result.reason,result end
 -- A ready nearby quest run with a full group can pay off through quest XP.
 if state.group and (state.partySize or 0)>=5 and cost==0 and travel<=300 and #analysis.missingPickups>=3 then return true,'Drei verfügbare Questbelohnungen, Gruppe vorhanden, kurzer Weg',result end
 return false,result.reason,result
end
L.Source='Forever Dungeon Journal boss items + live client item stats/tooltip requirements/proficiency; class stat weights are a leveling model, not BiS or drop probabilities'
