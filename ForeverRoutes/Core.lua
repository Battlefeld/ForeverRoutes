-- ForeverRoutes: the leveling route of this race and class (Data/Routes.lua, made by ForeverRoutes' export-addon),
-- followed step by step from what the quest log says. No ForeverCore needed: the route is in the addon. When the
-- ForeverCore bridge plans live, its steps take the place of the stored ones (FR.SetLive).
local FR={}
ForeverRoutes=FR
-- every return value kept, nils in between too (an aura's spell id comes after several nils)
local function pass(ok,...)if not ok then return nil end;return ... end
local function call(fn,...)if type(fn)~='function'then return nil end;return pass(pcall(fn,...))end

-- Game API across clients: Forever's client has the C_Item/C_Container/C_UnitAuras functions; the old globals
-- (GetItemCount, GetItemInfo, UnitBuff, ...) may be missing there. Each helper takes whichever exists.
local function api(ns,name)local t=_G[ns];return(type(t)=='table'and type(t[name])=='function')and t[name]or _G[name]end
function FR.ItemCount(id)
 return tonumber(call(api('C_Item','GetItemCount'),id,false,false))or 0 -- bags/equipment, never inaccessible bank stock
end
function FR.ItemInfo(id)
 local name,link,_,_,_,_,_,_,_,icon=call(api('C_Item','GetItemInfo'),id)
 if not name then name=call(C_Item and C_Item.GetItemNameByID,id)end
 return name,link,icon
end
-- the player's auras: {name, spell, duration, expires}
function FR.Auras()
 local out={}
 if C_UnitAuras and C_UnitAuras.GetAuraDataByIndex then
  for _,filter in ipairs({'HELPFUL','HARMFUL'})do
   for i=1,40 do
    local a=call(C_UnitAuras.GetAuraDataByIndex,'player',i,filter)
    if type(a)~='table'then break end
    out[#out+1]={name=a.name,spell=a.spellId,duration=a.duration,expires=a.expirationTime}
   end
  end
  return out
 end
 for _,filter in ipairs({'HELPFUL','HARMFUL'})do
  for i=1,40 do
   local name,_,_,_,duration,expires,_,_,_,spell=call(UnitAura,'player',i,filter)
   if not name then break end
   out[#out+1]={name=name,spell=spell,duration=duration,expires=expires}
  end
 end
 return out
end

-- The route's key: race and class as the route files name them; the Skyborne by side.
local RACE={Scourge='undead',Undead='undead',NightElf='nightelf',Human='human',Dwarf='dwarf',Gnome='gnome',Orc='orc',Troll='troll',Tauren='tauren'}
function FR.Key()
 local _,race=call(UnitRace,'player');local _,class=call(UnitClass,'player')
 if not race or not class then return nil end
 local r=RACE[race]
 if not r and tostring(race):find('Skyborne')then
  local side=call(UnitFactionGroup,'player');if side~='Horde'and side~='Alliance'then return nil end;r=(side=='Horde')and'skyborne-h'or'skyborne-a'
 end
 if not r then r=tostring(race):lower()end
 return r..'-'..tostring(class):lower()
end

local function charDB()
 ForeverRoutesCharDB=ForeverRoutesCharDB or{}
 local c=ForeverRoutesCharDB;c.skipped=c.skipped or{};c.confirmed=c.confirmed or{};c.index=math.max(1,tonumber(c.index)or 1)
 return c
end

-- The route of this character: "Solo mit Dungeon" (with dungeon runs) when chosen and there is one, else "Solo ohne
-- Dungeon".
function FR.Route()
 local key=FR.Key();local data=ForeverRoutesData
 if not key or not data then return nil,key end
 local c=charDB();local selected=key..((c.dungeons or c.group)and'-dungeons'or'')
 local revision=data.routes and data.routes[selected]and data.routes[selected].revision or data.build
 if c.routeKey~=selected or c.dataRevision~=revision then
  c.routeKey=selected;c.dataRevision=revision;c.index=1;c.skipped={};c.confirmed={};FR.pinned=nil
 end
 local r=data.routes and data.routes[selected]
 if r and r.s and #r.s>0 then return r,selected,'optimized'end
 local solo=data.routes and data.routes[key]
 if solo and solo.s and #solo.s>0 then return solo,key,'optimized'end
 return data.fallbacks and data.fallbacks[key],selected,'rxp-fallback'
end

-- With or without dungeons: the other route starts over from its first step that is not done yet.
function FR.SetDungeons(on)
 local c=charDB();c.dungeons=on and true or nil;c.group=nil;c.index=1;c.skipped={};c.confirmed={};FR.live=nil;FR.pinned=nil
 return FR.Advance()
end

-- Quest state from the game.
local function completed(q)
 if C_QuestLog and C_QuestLog.IsQuestFlaggedCompleted then return call(C_QuestLog.IsQuestFlaggedCompleted,q)==true end
 return call(IsQuestFlaggedCompleted,q)==true
end
local function onQuest(q)
 if C_QuestLog and C_QuestLog.IsOnQuest then return call(C_QuestLog.IsOnQuest,q)==true end
 return false
end
local function questDone(q)
 return C_QuestLog and C_QuestLog.IsComplete and call(C_QuestLog.IsComplete,q)==true
end
local TYPE={kill='monster',item='item',object='object',event='event',talk='monster'}
-- An objective step is done when the game says its objective is finished: the game's objective of the same kind and
-- count (the route numbers objectives its own way), the one naming the step's target first when two are alike, or
-- every objective when none matches.
local function objectiveDone(step)
 if questDone(step.q)then return true end
 local list=C_QuestLog and call(C_QuestLog.GetQuestObjectives,step.q)
 if type(list)~='table'or #list==0 then return false end
 local want=TYPE[step.k]
 local alike={}
 for _,o in ipairs(list)do
  if(not want or o.type==want)and(not step.c or tonumber(o.numRequired)==step.c)then alike[#alike+1]=o end
 end
 if #alike>1 and type(step.tg)=='string'and step.tg~=''then
  for _,o in ipairs(alike)do
   if type(o.text)=='string'and o.text:lower():find(step.tg:lower(),1,true)then return o.finished==true end
  end
 end
 if #alike==1 then return alike[1].finished==true end
 -- A localized or ambiguous target must not be confused with another completed objective.
 if #alike>1 then return false end
 for _,o in ipairs(list)do if not o.finished then return false end end
 return true
end

-- The instance the character is in (its map id), nil outside.
local function instanceID()
 if call(IsInInstance)~=true then return nil end
 local r={call(GetInstanceInfo)}
 return r[8]or 0
end
-- Whether the character is inside the dungeon of a dungeon step (any dungeon when the step does not name one).
local function inDungeonOf(step)
 local id=instanceID()
 return id~=nil and(not step.ii or step.ii==id)
end

-- A step's identity, the same for the same step in the stored route and in a new live plan.
function FR.StepKey(step)
 if not step then return''end
 return table.concat({tostring(step.op),tostring(step.q or''),tostring(step.o or''),tostring(step.n or''),tostring(step.x or''),tostring(step.y or''),tostring(step.l or''),tostring(step.xp or''),tostring(step.id or''),tostring(step.stacks or'')},':')
end

-- Whether a step is done: taken, handed in, objective finished, level reached.
function FR.Done(step)
 if not step then return true end
 if charDB().confirmed[FR.StepKey(step)]then return true end
 if step.op=='a'then return onQuest(step.q)or completed(step.q)end
 if step.op=='t'then return completed(step.q)end
 if step.op=='w'then return completed(step.q)or(onQuest(step.q)and objectiveDone(step))end
 if step.op=='g'then
  local l=call(UnitLevel,'player')or 0
  if not step.l then return false end
  return l>step.l or(l==step.l and(step.xp==nil or(call(UnitXP,'player')or 0)>=step.xp))
 end
 if step.op=='z'then return FR.ItemCount(step.it)>0 or call(api('C_Item','IsEquippedItem'),step.it)==true end
 if step.op=='p'then return FR.HasProfessions()end
 -- Dungeon, sleep and other unobservable tasks require explicit confirmation in the tracker.

 return false
end

-- Skinning and Leatherworking learned (by the skill lines, English or German).
local SKILL={skin={Skinning=true,['Kürschnerei']=true},leather={Leatherworking=true,Lederverarbeitung=true},cook={Cooking=true,Kochkunst=true}}
-- the profession skill lines by id (also the newer API without GetSkillLineInfo): Skinning, Leatherworking, Cooking
local SKILL_LINE={[393]='skin',[165]='leather',[185]='cook'}
function FR.Professions()
 local have={}
 local function add(kind,rank)if kind then have[kind]=math.max(have[kind]or 0,tonumber(rank)or 0)end end
 local n=call(GetNumSkillLines)or 0
 for i=1,n do
  local name,_,_,rank=call(GetSkillLineInfo,i)
  for kind,names in pairs(SKILL)do if names[name]then add(kind,rank)end end
 end
 if GetProfessions and GetProfessionInfo then
  local list={call(GetProfessions)}
  for _,idx in pairs(list)do
   if idx then
    local name,_,rank,_,_,_,line=call(GetProfessionInfo,idx)
    add(SKILL_LINE[line],rank)
    for kind,names in pairs(SKILL)do if names[name]then add(kind,rank)end end
   end
  end
 end
 return have
end
function FR.HasProfessions()
 local p=FR.Professions()
 return p.skin~=nil and(p.leather or 0)>=20 and p.cook~=nil
end

-- The camp, coupled to the camp buff (an hour; all camp objects share that cooldown). The buff is found among the
-- character's auras by its name (CAMP_AURAS); without it the time since the last camp (/fr lager) stands in.
-- 10 minutes before it runs out: is there leather for another tent? If not, "farm leather" beside the route.
-- 5 minutes before: the buff ends soon, with what the camp still lacks. Run out: make camp. All of it only when the
-- rested experience is used up (or will be): with rested experience left the tent raises nothing.
-- The hour is the camp cooldown aura (spell 1229451) or the Leatherworking camp buff "Well Rested Shelter"
-- (kovarel.com/wow-forever/features/camping); the tent's rest bonus raises rested XP to 5 % of a level.
FR.CAMP_AURA_IDS={[1229451]=true}
FR.CAMP_AURAS={'Well Rested Shelter'}
FR.LIGHT_LEATHER=2318;FR.TENT_LEATHER=5            -- Camp Tent (item 279978): 5 Light Leather
FR.TENT_ITEM=279978
FR.FIRE_KIT=279981;FR.FIRE={{4470,1},{4471,1}}     -- Basic Campfire Kit (item 279981): Simple Wood, Flint and Tinder
local function campAura()
 local now=call(GetTime)or 0
 for _,a in ipairs(FR.Auras())do
  local ours=FR.CAMP_AURA_IDS[a.spell]
  for _,k in ipairs(FR.CAMP_AURAS)do ours=ours or tostring(a.name)==k end
  if ours and(a.duration or 0)>0 and a.expires then return a.expires-now end
 end
 return nil
end
-- Seconds the camp buff has left (0: none or run out); nil before the camps start.
function FR.CampLeft()
 local c=charDB()
 -- the camps start with the professions, however they were learned (also off the route)
 if not c.campLevel and FR.HasProfessions()then c.campLevel=FR.CAMP_LEVEL end
 if not c.campLevel or(call(UnitLevel,'player')or 0)<c.campLevel then return nil end
 local left=campAura()
 if left then return math.max(0,left)end
 return math.max(0,3600-((call(time)or 0)-(c.camp or -1e9)))
end
-- The bags by item name (the client's English names): the camp items are found by name, not only by their ids,
-- since Forever's ids are not all known. Each entry: name, id, count.
FR.CAMP_NAMES={tent={'tent','zelt'},fire={'campfire','lagerfeuer','firewood','kindling','brennholz'}}
function FR.BagItems()
 local out={}
 local slots=api('C_Container','GetContainerNumSlots')
 local idOf=api('C_Container','GetContainerItemID')
 if not slots or not idOf then return out end
 for bag=0,NUM_BAG_SLOTS or 4 do
  for slot=1,tonumber(call(slots,bag))or 0 do
   local id=call(idOf,bag,slot)
   if id then
    local name=FR.ItemInfo(id)
    if type(name)=='string'then out[#out+1]={name=name,id=id,count=FR.ItemCount(id)}end
   end
  end
 end
 return out
end
local function bagHas(kind)
 for _,it in ipairs(FR.BagItems())do
  local n=it.name:lower()
  for _,w in ipairs(FR.CAMP_NAMES[kind])do if n:find(w,1,true)then return it end end
 end
end
-- What a camp needs that the bags lack: tent (or leather for one), fire.
function FR.CampMaterials()
 local leather=FR.ItemCount(FR.LIGHT_LEATHER)
 local tent=FR.ItemCount(FR.TENT_ITEM)>0 or leather>=FR.TENT_LEATHER or bagHas('tent')~=nil
 local fire=FR.ItemCount(FR.FIRE_KIT)>0 or bagHas('fire')~=nil
 if not fire then
  fire=true
  for _,r in ipairs(FR.FIRE)do if FR.ItemCount(r[1])<r[2]then fire=false end end
 end
 return{tent=tent,leather=leather,fire=fire}
end
-- /fr camp: what the addon sees of the camp (for a report when it is not recognized)
function FR.CampReport()
 local m=FR.CampMaterials();local left=FR.CampLeft()
 print(string.format('ForeverRoutes Lager: Zelt %s, Lagerfeuer %s, Leichtes Leder %d, Lager-Buff %s, Rested %.1f %%',
  m.tent and'ja'or'nein',m.fire and'ja'or'nein',math.floor(m.leather),left and string.format('%.0f s',left)or'(Lager noch nicht aktiv)',FR.RestedShare()*100))
 for _,it in ipairs(FR.BagItems())do
  local n=it.name:lower()
  if n:find('tent',1,true)or n:find('camp',1,true)or n:find('fire',1,true)or n:find('wood',1,true)or n:find('flint',1,true)or n:find('leather',1,true)then
   print(string.format('  Tasche: %s (Item %d) x%d',it.name,it.id,it.count))
  end
 end
 for _,a in ipairs(FR.Auras())do
  print(string.format('  Aura: %s (Zauber %s, %s s)',tostring(a.name),tostring(a.spell),tostring(a.duration)))
 end
 local p=FR.Professions()
 print(string.format('  Berufe: Kürschnern %s, Lederverarbeitung %s, Kochkunst %s',tostring(p.skin or'-'),tostring(p.leather or'-'),tostring(p.cook or'-')))
 print(string.format('  Taschen gelesen: %d Gegenstände; Stufe %s, Lager ab Stufe %s',#FR.BagItems(),tostring(call(UnitLevel,'player')),tostring(charDB().campLevel)))
end
-- Rested experience left, as a share of the level (the tent raises it to 5 %: with more left a camp brings nothing).
function FR.RestedShare()
 local max=call(UnitXPMax,'player')or 0
 if max<=0 then return 0 end
 return(call(GetXPExhaustion)or 0)/max
end
FR.CAMP_EMPTY=0.01   -- make camp once the rested experience is used up (below 1 % of the level)
FR.CAMP_SOON=0.025   -- warn ahead only when it will be used up by then (below half a tent)
-- The camp's word for the window: a hint for the route line and a task beside the route (nil when nothing).
function FR.CampStatus()
 local left=FR.CampLeft();if not left then return nil,nil end
 local rested=FR.RestedShare()
 if rested>=FR.CAMP_SOON or(left<=0 and rested>=FR.CAMP_EMPTY)then return nil,nil end -- rested experience left: no camp yet
 local m=FR.CampMaterials()
 local farm=not m.tent and string.format('Nebenbei: Leder farmen - Wildtiere kürschnern (%d/%d Leichtes Leder für ein Lagerzelt)',m.leather,FR.TENT_LEATHER)or nil
 if left<=0 then
  return'Lager aufschlagen: Lagerfeuer, Lagerzelt, 1 Minute rasten (/fr lager)',farm
 end
 if left<=300 then
  return string.format('Lager-Buff endet in %d Min - Zelt: %s, Lagerfeuer: %s',math.ceil(left/60),m.tent and'bereit'or'fehlt',m.fire and'bereit'or'fehlt'),farm
 end
 if left<=600 then return nil,farm end
 return nil,nil
end
function FR.Camped()local c=charDB();c.camp=call(time)or 0 end

-- Paused: the character is inside a dungeon the route does not run now (a second run). Nothing is
-- blocked: the route waits and goes on from its first open step once the character is out (steps done meanwhile,
-- levels reached meanwhile, are passed by themselves).
function FR.Paused(step)
 return instanceID()~=nil and not(step and step.op=='d'and inDungeonOf(step))
end

-- Options (route.o): goals the route leaves out that can be switched on, each with its steps and for each the step
-- of the route it comes after (af, 0 before the first). On, its steps go in there; the route's grind steps are then
-- done earlier by themselves (a grind step is done at its level).
function FR.Options()
 local r=FR.Route();return r and r.o or{}
end
local function optionOn(o)local c=charDB();return c.opts and c.opts[o.id]==true end
-- How many options are switched on.
function FR.OptionsOn()
 local n=0;for _,o in ipairs(FR.Options())do if optionOn(o)then n=n+1 end end;return n
end
-- An option by its number in the list or its id; nil when there is none.
function FR.Option(n)
 n=tonumber(n);if not n then return nil end
 local list=FR.Options()
 if list[n]then return list[n]end
 for _,o in ipairs(list)do if o.id==n then return o end end
 return nil
end
-- Switches an option on or off (nil: the other way); the route goes on from its first step not done yet.
function FR.SetOption(n,on)
 local o=FR.Option(n);if not o then return nil end
 local c=charDB();c.opts=c.opts or{}
 if on==nil then on=not optionOn(o)end
 c.opts[o.id]=on and true or nil;c.index=1;c.skipped={};c.confirmed={};FR.live=nil;FR.pinned=nil
 FR.Advance()
 return o,on
end
-- The stored route with the options switched on in it (kept until the route or the options change).
local merged
local function withOptions(r,key)
 local on={}
 for _,o in ipairs(r.o or{})do if optionOn(o)then on[#on+1]=o end end
 if #on==0 then return r.s or{}end
 local ids={};for _,o in ipairs(on)do ids[#ids+1]=o.id end
 local tag=key..':'..table.concat(ids,',')
 if merged and merged.tag==tag then return merged.s end
 local at={}
 for _,o in ipairs(on)do
  for _,st in ipairs(o.s or{})do
   st.opt=o.n;local a=st.af or 0;at[a]=at[a]or{};local l=at[a];l[#l+1]=st
  end
 end
 local out={}
 for i=0,#(r.s or{})do
  if i>0 then out[#out+1]=r.s[i]end
  for _,st in ipairs(at[i]or{})do out[#out+1]=st end
 end
 merged={tag=tag,s=out}
 return out
end

-- The steps to show: the live plan when there is one, else the stored route (with the options switched on).
function FR.Steps()
 if FR.live then return FR.live,true end
 local r,key=FR.Route();return r and r.s and withOptions(r,key)or{},false
end

-- Moves past what is done or skipped. Returns the current step.
function FR.Advance()
 local steps,live=FR.Steps();local c=charDB()
 local i=live and 1 or c.index
 while steps[i]and(FR.Done(steps[i])or(not live and c.skipped[i]))do
  if(steps[i].op=='p'or steps[i].op=='z')and steps[i].cl then c.campLevel=steps[i].cl end -- the professions: camps from then on
  i=i+1
 end
 if not live then c.index=i end
 FR.index=i
 return steps[i],i
end

-- A skipped quest stays skipped: every live plan that comes after it leaves its steps out (skipQuests), until the
-- player takes the skip back or restores the skipped steps.
local function skipQuest(c,q,on)
 if not q then return end
 c.skipQuests=c.skipQuests or{};c.skipQuests[q]=on or nil
end
function FR.Skip()
 local st,i=FR.Advance();local _,live=FR.Steps();local c=charDB();if not st then return end
 FR.undo={step=st,index=i,live=live,removed=live}
 if st.op=='a'then skipQuest(c,st.q,true)end
 if live then table.remove(FR.live,i);if FR.liveBlock and i<=FR.liveBlock then FR.liveBlock=FR.liveBlock-1 end else c.skipped[i]=true;c.index=i+1 end
 return FR.Advance()
end
function FR.Back()
 local c=charDB();if FR.undo then local u=FR.undo;FR.undo=nil;if u.step and u.step.op=='a'then skipQuest(c,u.step.q,false)end;if u.live and FR.live then if u.removed then table.insert(FR.live,u.index,u.step)end;c.confirmed[FR.StepKey(u.step)]=nil else c.skipped[u.index]=nil;c.confirmed[FR.StepKey(u.step)]=nil;c.index=u.index end;return FR.Advance()end;if FR.live then return FR.Advance()end
 local i=math.max(1,c.index-1);c.skipped[i]=nil;c.index=i;local st=FR.Steps()[i];if st then c.confirmed[FR.StepKey(st)]=nil end
 return FR.Steps()[i],i
end
function FR.Reset()
 local c=charDB()
 FR.undo=nil
 ForeverRoutesCharDB={dungeons=c.dungeons,opts=c.opts};return FR.Advance()
end

-- The live plan from the ForeverCore bridge (its route-next result): steps in its own form, turned into ours.
local OP={sleep="sleep",accept='a',turnin='t',work='w',grind='g',dungeon='d',wand='z',prof='p'}
FR.LESSER_WAND=11287;FR.CAMP_LEVEL=10   -- a live wand step: done with this wand; live professions: camps from this level
-- A live dungeon step names the dungeon only: its instance and the quests taken inside come from the stored routes.
local function storedDungeon(name)
 local data=ForeverRoutesData and ForeverRoutesData.routes
 local r=FR.Route()
 for _,route in ipairs({r or{},data and data[(FR.Key()or'')..'-dungeons']or{}})do
  for _,st in ipairs(route.s or{})do if st.op=='d'and st.n==name then return st end end
 end
 return nil
end
function FR.SetLive(result)
 if type(result)~='table'or type(result.steps)~='table'then FR.live=nil;FR.nextLive=nil;return end
 local out={}
 for _,s in ipairs(result.steps)do
  local at=type(s.at)=='table'and s.at or{}
  local via=type(s.via)=='string'and s.via or''
  local name=type(s.name)=='string'and s.name:gsub(' %(dungeon run%)$','')or s.name
  out[#out+1]={op=OP[s.op]or s.op,q=s.quest,n=name,m=at.m,x=at.x,y=at.y,l=s.level,o=s.obj and(s.obj+1),npc=s.npc,object=s.object,items=s.items,stacks=s.stacks,kills=tonumber(s.kills),
   v=(via:find('^deathskip')and'd')or(via=='hearth'and'h')or(via=='teleport'and't')or nil,gy=via:match('^deathskip: (.+)$'),
   dm=s.viaAt and s.viaAt.m,dx=s.viaAt and s.viaAt.x,dy=s.viaAt and s.viaAt.y,live=true}
  local sb=type(s.startBy)=='table'and s.startBy or nil
  if sb and sb.name then local st=out[#out];st.sk,st.sl,st.se,st.si=sb.name,sb.level,sb.elite==true or nil,sb.item end
  local bind=type(s.bind)=='table'and s.bind or nil
  if bind and type(bind.at)=='table'then local st=out[#out];st.bn,st.bm,st.bx,st.by,st.bu=bind.name,bind.at.m,bind.at.x,bind.at.y,s.bindUsed==true or nil end
  local st=out[#out]
  if st.q and(charDB().skipQuests or{})[st.q]then out[#out]=nil;st={}end
  if st.op=='d'then local known=storedDungeon(st.n);if known then st.ii,st.qi=known.ii,known.qi end end
  if st.op=='g'and type(st.n)=='string'then st.tg=st.n:match('^%d+ × (.+)$')or st.n end
  if st.op=='z'then st.it,st.cl=FR.LESSER_WAND,FR.CAMP_LEVEL end
  if st.op=='p'then st.cl=FR.CAMP_LEVEL end
 end
 FR.nextLive=out
 if FR.KeepBlock()then return end -- the block being played stays; this plan waits until it is (nearly) done
 FR.AdoptLive(out)
end

-- A live plan is played in blocks: its first BLOCK steps stay as they are until BLOCK_LEFT of them are left, however
-- often ForeverCore plans again meanwhile (after every bit of quest progress). Then the newest plan takes over, made
-- from where the character is by then. A block that no longer fits (a quest of it left the log, a step is blocked)
-- or "Live-Plan neu berechnen" lets the newest plan take over at once.
FR.BLOCK,FR.BLOCK_LEFT=10,2
function FR.BlockLeft()
 if not FR.live then return 0 end
 local n=0
 for i=1,math.min(FR.liveBlock or 0,#FR.live)do if not FR.Done(FR.live[i])then n=n+1 end end
 return n
end
local function blockFits()
 local taking={}
 for i=1,math.min(FR.liveBlock or 0,#FR.live)do
  local s=FR.live[i]
  if not FR.Done(s)then
   if s.op=='a'and s.q then taking[s.q]=true end
   if(s.op=='w'or s.op=='t')and s.q and not taking[s.q]and not FR.OnQuest(s.q)then return false end
   if s.op=='a'and FR.Blocked and FR.Blocked(s)then return false end
  end
 end
 return true
end
function FR.KeepBlock()
 return FR.live~=nil and not FR.forceReplan and FR.BlockLeft()>FR.BLOCK_LEFT and blockFits()
end
-- Take the newest plan when the block is (nearly) done or no longer fits; called on every refresh.
function FR.CheckBlock()
 local nextPlan=FR.nextLive
 if FR.live and nextPlan and nextPlan~=FR.live and not FR.KeepBlock()then FR.AdoptLive(nextPlan)end
end
function FR.ReplanNow()
 FR.forceReplan=true
 local r=ForeverCoreClient and ForeverCoreClient.Route
 if r and r.Request then call(r.Request,false)end
 FR.CheckBlock()
end
function FR.AdoptLive(out)
 -- Stay on the step being done: a new plan (one comes after every bit of quest progress) may put another step first,
 -- often an accept the character walks past. The current step stays first until it is done; an accept the new plan
 -- wants right here becomes a hint beside it (FR.nearbyAccept).
 FR.nearbyAccept=nil
 local old=FR.live and FR.Advance()
 if old and not FR.Done(old)then
  for i=2,#out do
   local s=out[i]
   if s.op==old.op and s.q==old.q and s.o==old.o and(s.q or s.n==old.n)then
    local first=out[1]
    table.remove(out,i);table.insert(out,1,s)
    if first.op=='a'and not first.sk then
     local x,y,_,m=call(UnitPosition,'player')
     if m==first.m and type(x)=='number'and type(first.x)=='number'and(x-first.x)^2+(y-first.y)^2<=80^2 then FR.nearbyAccept=first end
    end
    break
   end
  end
 end
 FR.live=out;FR.undo=nil;FR.pinned=nil;FR.forceReplan=nil;FR.nextLive=out
 FR.liveBlock=math.min(FR.BLOCK,#out)
end

-- Wording.
local function questName(step)
 local t=C_QuestLog and call(C_QuestLog.GetTitleForQuestID,step.q)
 return(type(t)=='string'and t~='')and t or step.n or('Quest '..tostring(step.q))
end
function FR.Text(step)
 if not step then return'Route geschafft'end
 local n=questName(step)
 if step.op=='a'and step.sk then
  -- an item starts the quest: the step is the kill
  local item=step.si and FR.ItemInfo(step.si)
  return string.format('Töten: %s (%sStufe %d) - droppt %s, damit %s annehmen',step.sk,step.se and'Elite, 'or'',step.sl or 0,item or'den Gegenstand',n)
 end
 if step.op=='a'then return'Annehmen: '..n end
 if step.op=='t'then return'Abgeben: '..n end
 if step.op=='sleep'then return'Rasten: Schlafsack verwenden und den vorgesehenen Buff aufbauen'end
 if step.op=='p'then return'Berufe: Kürschnern, Kochkunst und Lederverarbeitung lernen - Wildtiere kürschnern, Lederverarbeitung mit dem Leder auf 20 (Lagerzelt)'end
 if step.op=='z'then
  local t="Zauberstäbe: Verzauberkunst lernen, Novice's Practice Wand herstellen (Einfaches Holz, Mote of Magic vom Händler), mit Übungszauberstäben auf 10, Geringe Magische Essenz im Auktionshaus, Geringen Zauberstab herstellen"
  if(step.l or 0)<5 then t=t..' - den Übungszauberstab tragen, ab Level 5 den Geringen'end
  if step.cl then t=t..'. Danach Verzauberkunst verlernen; Kürschnern, Kochkunst und Lederverarbeitung lernen (Lagerzelt)'end
  return t
 end
 if step.op=='g'then return string.format('Grinden bis Level %d%s%s',step.l or 0,step.xp and(' + '..step.xp..' XP')or'',step.tg and(': '..step.tg)or'')end
 if step.op=='d'then
  local t='Dungeon laufen: '..(step.n or'?')..' (Gruppe suchen, alle Quests im Log erledigen)'
  if type(step.qi)=='table'and #step.qi>0 then t=t..' - im Dungeon annehmen: '..table.concat(step.qi,', ')end
  return t
 end
 if step.op=='w'then
  local c,tg=step.c or 1,step.tg
  if step.k=='kill'then return string.format('%s: töte %d x %s',n,c,tg or'?')end
  if step.k=='item'then return string.format('%s: sammle %d%s',n,c,tg and(' von '..tg)or'')end
  if step.k=='object'then return string.format('%s: benutze %s',n,tg or'das Objekt')end
  if step.k=='talk'then return string.format('%s: sprich mit %s',n,tg or'?')end
  return n..': Ziel erledigen'
 end
 return tostring(step.op)
end
-- What to do first to get there: a deathskip, the hearthstone or a druid's teleport to Moonglade.
function FR.Hint(step)
 if not step then return nil end
 local h=FR.TravelHint(step)
 if step.opt then h='Option: '..step.opt..(h and(' · '..h)or'')end
 local camp=FR.CampStatus()
 if camp then h=(h and(h..' · ')or'')..camp end
 return h
end
function FR.TravelHint(step)
 if step.v=='d'then return'Death Skip: hier sterben, beim Geistheiler wiederbeleben'..(step.gy and(' ('..step.gy..')')or'')end
 if step.v=='h'then local to=FR.HearthTarget and FR.HearthTarget(step);return'Ruhestein benutzen'..(to and(' (nach '..to..')')or'')end
 if step.v=='t'then return'Teleportieren: Mondlichtung'end
 return nil
end

-- Where the arrow points: the place to die before a deathskip (until the character has died for this step), else the
-- step.
function FR.Target(step)
 if not step then return nil end
 local died=charDB().died
 if step.v=='d'and step.dx and not call(UnitIsDeadOrGhost,'player')and died~=FR.StepKey(step)then return step.dm,step.dx,step.dy,true end
 return step.m,step.x,step.y,false
end
-- The character died: the deathskip of the current step is done.
function FR.Died()
 local steps=FR.Steps()
 charDB().died=FR.StepKey(steps[FR.index or 1])
end


-- Explicit completion is distinct from skipping; quest steps always follow the game quest log.
-- Every step can be marked done by hand ("Erledigt"): the route may know a step the game does not offer (a quest
-- behind a profession, a changed quest). An "accept" marked done for a quest not in the log takes the whole quest out
-- of this and every later live plan, like a skip.
function FR.CanConfirm(step)
 return step~=nil and not FR.Paused(step)
end
function FR.Confirm()
 local step=FR.Advance()
 if not FR.CanConfirm(step)then return false end
 FR.undo={step=step,index=FR.index or 1,live=FR.live~=nil}
 local c=charDB()
 c.confirmed[FR.StepKey(step)]=true
 if step.op=='a'and step.q and not onQuest(step.q)then c.skipQuests=c.skipQuests or{};c.skipQuests[step.q]=true end
 if FR.live and step.q and(c.skipQuests or{})[step.q]then
  for i=#FR.live,1,-1 do if FR.live[i].q==step.q then table.remove(FR.live,i)end end
 end
 FR.Advance();return true
end
function FR.Objectives(step)
 if not step or not step.q then return{}end
 local list=C_QuestLog and call(C_QuestLog.GetQuestObjectives,step.q)
 return type(list)=='table'and list or{}
end
function FR.RestoreSkipped()
 local c=charDB();c.skipped={};c.skipQuests=nil;c.index=1;FR.pinned=nil;return FR.Advance()
end

FR.Call=call
FR.CharDB=charDB
FR.OnQuest=onQuest
FR.Completed=completed
FR.QuestDone=questDone


-- Grind progress. Kills are counted from the XP (no combat log: it is closed to addons in this client): every XP gain
-- that does not come with a quest turned in counts as one kill. Discovering an area gives XP too and is counted as
-- well; which creature died is not known, the grind step is about its creature anyway.
local xpLast,levelLast,maxLast,turnedIn
local function readXP()return call(UnitXP,'player'),call(UnitLevel,'player'),call(UnitXPMax,'player')end
function FR.OnXP()
 local xp,level,max=readXP()
 if not(xp and level and max)then return end
 if xpLast and levelLast then
  local gain=level>levelLast and(maxLast or 0)-xpLast+xp or xp-xpLast
  local now=GetTime and GetTime()or 0
  if gain>0 and not(turnedIn and now-turnedIn<2)then
   local c=charDB();c.xpKills=(c.xpKills or 0)+1;c.lastKillXP=gain
  end
 end
 xpLast,levelLast,maxLast=xp,level,max
end
local xpFrame=CreateFrame('Frame')
for _,e in ipairs({'PLAYER_ENTERING_WORLD','PLAYER_XP_UPDATE','QUEST_TURNED_IN'})do pcall(xpFrame.RegisterEvent,xpFrame,e)end
xpFrame:SetScript('OnEvent',function(_,event)
 if event=='QUEST_TURNED_IN'then turnedIn=GetTime and GetTime()or 0;return end
 if event=='PLAYER_ENTERING_WORLD'then xpLast,levelLast,maxLast=readXP();return end
 FR.OnXP()
end)
-- The lines a grind step shows: the kills against the planned number ("Mottled Boar: 3/16") and the XP.
function FR.GrindProgress(step)
 if not step or step.op~='g'then return nil end
 local c=charDB();c.grindFrom=c.grindFrom or{}
 local lines={}
 local key=FR.StepKey(step)
 if c.grindFrom[key]==nil then
  local n=0;for _ in pairs(c.grindFrom)do n=n+1 end;if n>200 then c.grindFrom={} end
  c.grindFrom[key]=c.xpKills or 0
 end
 local done=math.max(0,(c.xpKills or 0)-c.grindFrom[key])
 local who=step.tg or'Kills'
 local last=c.lastKillXP and done>0 and string.format(' (zuletzt +%d XP)',c.lastKillXP)or''
 lines[#lines+1]=step.kills and string.format('%s: %d/%d%s',who,done,step.kills,last)or string.format('%s: %d getötet%s',who,done,last)
 local level,xp,max=call(UnitLevel,'player'),call(UnitXP,'player'),call(UnitXPMax,'player')
 if level and xp and max and max>0 then
  if step.l and level<step.l then
   lines[#lines+1]=string.format('XP: %d/%d (Level %d, Ziel Level %d)',xp,max,level,step.l)
  else
   lines[#lines+1]=string.format('XP: %d/%d (Level %d)',xp,max,level)
  end
 end
 return lines
end

