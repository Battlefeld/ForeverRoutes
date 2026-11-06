unpack=unpack or table.unpack
math.atan2=math.atan2 or function(y,x)return math.atan(y,x)end
local M=dofile('_dev/harness/wowmock.lua')
dofile('_dev/harness/gameapi.lua')
local count=0
local function check(ok,msg)assert(ok,msg);count=count+1 end
local accepted,complete,objectives={},{},{}
C_QuestLog={IsOnQuest=function(q)return accepted[q]or false end,IsComplete=function()return false end,
 IsQuestFlaggedCompleted=function(q)return complete[q]or false end,
 GetQuestObjectives=function(q)return objectives[q]end,GetTitleForQuestID=function(q)return 'Quest '..q end}
local instance=false
function IsInInstance()return instance end
function GetInstanceInfo()return nil,nil,nil,nil,nil,nil,nil,999 end
local px,py,pm=0,0,0
function UnitPosition()return px,py,0,pm end
function GetPlayerFacing()return 0 end
local bags=0
function GetItemCount(id)return bags end
C_Item=C_Item or{};C_Item.GetItemCount=GetItemCount
function GetItemInfo(id)return 'Item '..id,'item:'..id,nil,nil,nil,nil,nil,nil,nil,'icon' end
function GetNumSkillLines()return 0 end
function InCombatLockdown()return false end
C_Container={GetContainerNumFreeSlots=function()return 8,0 end}
local pin,added,removed=nil,0,0
C_Map={GetBestMapForUnit=function()return 1 end,GetMapPosFromWorldPos=function()return 1,{x=.5,y=.5}end,
 SetUserWaypoint=function(v)pin=v end,ClearUserWaypoint=function()pin=nil end}
UiMapPoint={CreateFromCoordinates=function(map,x,y)return{uiMapID=map,position={x=x,y=y}}end}
local function data(steps)
 ForeverRoutesData={build='test',routes={['gnome-warlock']={revision='r1',s=steps,opts={}}}}
 ForeverRoutesCharDB={};ForeverRoutesDB={};ForeverRoutes.live=nil;ForeverRoutes.undo=nil
end
for _,file in ipairs({'Core','Tasks','ArtData','Art','Navigation','Tooltips','UI'})do dofile('ForeverRoutes/'..file..'.lua')end
local FR=ForeverRoutes
for _,style in ipairs({'panel','button','note'})do check(FR.Art[FR.UI.STYLE[style][1]]~=nil,'packaged art exists for '..style)end
local steps={{op='a',q=1,n='First',m=0,x=100,y=0},{op='w',q=1,n='First',k='kill',c=2,tg='Wolf',m=0,x=100,y=0},{op='t',q=1,n='First',m=0,x=0,y=0},{op='note',n='Read this'}}
data(steps);FR.Show();check(FR.TaskView().step.op=='a','accept first')
accepted[1]=true;objectives[1]={{text='Wolves: 1/2',type='monster',numFulfilled=1,numRequired=2,finished=false}}
FR.Refresh();check(FR.TaskView().step.op=='w','accepted quest advances')
check(FR.frame.current.text:GetText():find('töte',1,true),'current task visible')
check(FR.frame.rows[1].text:GetText():find('1/2',1,true),'real objective progress visible')
check(FR.CanConfirm(FR.TaskView().step),'any step can be marked done by hand (Erledigt)')
FR.Skip();check(FR.TaskView().step.op=='t','skip work')
FR.Back();check(FR.TaskView().step.op=='w','undo skip work')
objectives[1][1].finished=true;FR.Refresh();check(FR.TaskView().step.op=='t','finished objective advances')
complete[1]=true;FR.Refresh();check(FR.TaskView().step.op=='note','turnin advances')
check(FR.Confirm(),'manual note confirm');check(FR.TaskView().step==nil,'route ends')
FR.Back();check(FR.TaskView().step.op=='note','undo manual confirm')
FR.Config().compact=true;FR.Refresh();check(not FR.frame.nextLabel:IsShown(),'compact hides preview')
-- Arrival and navigation failure do not complete an active quest.
local st={op='w',q=2,k='kill',c=1,m=0,x=0,y=0};accepted[2]=true
objectives[2]={{text='Mob: 0/1',type='monster',numRequired=1,finished=false}}
check(FR.Navigation(st).arrived,'arrival recognized');check(not FR.Done(st),'arrival does not finish quest')
check(FR.Navigation({op='w',m=1,x=0,y=0}).distance==nil,'cross continent has no false distance')
C_Map.GetMapPosFromWorldPos=function()return 1,{x=1.1,y=.4}end
check(not FR.Waypoint(0,1,1,'test'),'invalid converted pin rejected')
C_Map.GetMapPosFromWorldPos=function()return 1,{x=.5,y=.5}end
FR.UpdateWaypoint(st,FR.Navigation(st));check(pin~=nil,'fallback map pin installed')
FR.UpdateWaypoint(nil,nil);check(pin==nil and FR.pinned==nil,'stale pin cleared')
-- TomTom only replaces waypoints when target changes.
TomTom={AddWaypoint=function()added=added+1;return added end,RemoveWaypoint=function()removed=removed+1 end}
FR.UpdateWaypoint(st,FR.Navigation(st));FR.UpdateWaypoint(st,FR.Navigation(st));check(added==1,'no TomTom churn')
FR.ClearWaypoint();check(removed==1,'TomTom waypoint removed')
TomTom=nil
-- Completed leading steps in a live plan must not make Skip remove step 1.
FR.live={{op='a',q=1},{op='note',n='live note'},{op='note',n='live second'}}
FR.CharDB().confirmed={};check(FR.Advance().n=='live note','live completed prefix skipped')
FR.Skip();check(#FR.live==2 and FR.live[1].q==1 and FR.Advance().n=='live second','live skip removes actual current index')
FR.Back();check(#FR.live==3 and FR.Advance().n=='live note','live undo inserts once at correct index')
FR.Confirm();FR.Back();check(#FR.live==3 and FR.Advance().n=='live note','live undo confirmation does not duplicate')
FR.SetLive({steps={{op='accept',quest=1},{op='work',quest=2,obj=0,at={m=0,x=0,y=0}}}})
check(FR.live[2].o==1 and FR.undo==nil,'new plan carries objective and discards undo')
instance=true;check(FR.Navigation(FR.live[2])==nil,'no navigation in dungeon');FR.Refresh();check(not FR.arrow:IsShown(),'arrow hides in dungeon')
instance=false
check(FR.Done({op='g',l=20}),'level-only grind completes at target level')
check(not FR.Done({op='g',l=20,xp=6000}),'grind waits for target xp')
-- A live grind step counts kills from the XP: each gain without a quest turned in is one kill.
FR.SetLive({steps={{op='grind',name='16 × Mottled Boar',npc=3098,kills=16,level=3,at={m=1,x=0,y=0}}}})
local g=FR.live[1];check(g.op=='g'and g.kills==16 and g.tg=='Mottled Boar','live grind keeps kills and mob')
local lines=FR.GrindProgress(g);check(lines[1]=='Mottled Boar: 0/16','grind starts at 0/16: '..tostring(lines[1]))
check(lines[2]:find('XP: 5000/14000',1,true)~=nil,'grind shows XP: '..tostring(lines[2]))
local xpNow,levelNow=5000,2;UnitXP=function()return xpNow end;UnitLevel=function()return levelNow end
M.Fire('PLAYER_ENTERING_WORLD')
xpNow=5045;M.Fire('PLAYER_XP_UPDATE');xpNow=5090;M.Fire('PLAYER_XP_UPDATE')
check(FR.GrindProgress(g)[1]=='Mottled Boar: 2/16 (zuletzt +45 XP)','two kills from XP: '..tostring(FR.GrindProgress(g)[1]))
M.Fire('QUEST_TURNED_IN',1);xpNow=5500;M.Fire('PLAYER_XP_UPDATE')
check(FR.GrindProgress(g)[1]:find('2/16',1,true)~=nil,'quest XP is no kill')
xpNow,levelNow=30,3;M.Advance(3);M.Fire('PLAYER_XP_UPDATE')
check(FR.GrindProgress(g)[1]:find('3/16',1,true)~=nil,'a kill across the level up counts')
UnitXP=function()return 5000 end;UnitLevel=function()return 2 end
-- In combat the map pin and the quest log stay untouched (Blizzard blocks them there); after combat the pin follows.
do
 local sets=0;local old=C_Map.SetUserWaypoint;C_Map.SetUserWaypoint=function(...)sets=sets+1;if old then return old(...)end end
 FR.ClearWaypoint();local step={op='note',n='combat pin',m=1,x=100,y=100}
 InCombatLockdown=function()return true end
 FR.UpdateWaypoint(step,FR.Navigation(step));FR.OpenQuest(1)
 check(sets==0,'no map pin set in combat')
 InCombatLockdown=function()return false end
 C_Map.SetUserWaypoint=old
end
-- A quest skipped in a live plan stays out of the next live plans, until the skip is taken back.
FR.live=nil;FR.CharDB().skipQuests=nil
FR.SetLive({steps={{op='accept',quest=96873,at={m=1,x=0,y=0}},{op='accept',quest=4641,at={m=1,x=0,y=0}}}})
check(FR.Advance().q==96873,'live plan starts with the quest')
FR.Skip();check(FR.Advance().q==4641,'skip moves on')
FR.SetLive({steps={{op='accept',quest=96873,at={m=1,x=0,y=0}},{op='work',quest=96873,obj=0,at={m=1,x=0,y=0}},{op='accept',quest=4641,at={m=1,x=0,y=0}}}})
check(#FR.live==1 and FR.live[1].q==4641,'replanned live plan leaves the skipped quest out')
FR.RestoreSkipped();FR.SetLive({steps={{op='accept',quest=96873,at={m=1,x=0,y=0}}}})
check(#FR.live==1 and FR.live[1].q==96873,'restoring skipped steps lets the quest in again')
FR.Skip();FR.Back();FR.SetLive({steps={{op='accept',quest=96873,at={m=1,x=0,y=0}}}})
check(#FR.live==1 and FR.live[1].q==96873,'taking the skip back lets the quest in again')
FR.live=nil;FR.CharDB().skipQuests=nil
-- "Erledigt" on an accept of a quest not in the log takes the quest out of the live plan and later ones.
FR.live=nil;FR.CharDB().skipQuests=nil;FR.CharDB().confirmed={}
FR.SetLive({steps={{op='accept',quest=96873,at={m=1,x=0,y=0}},{op='work',quest=96873,obj=0,at={m=1,x=0,y=0}},{op='accept',quest=4641,at={m=1,x=0,y=0}}}})
check(FR.Confirm()and FR.Advance().q==4641,'done accept of a missing quest moves past all its steps')
FR.SetLive({steps={{op='accept',quest=96873,at={m=1,x=0,y=0}},{op='accept',quest=4641,at={m=1,x=0,y=0}}}})
check(#FR.live==1 and FR.live[1].q==4641,'later live plans leave it out')
FR.live=nil;FR.CharDB().skipQuests=nil;FR.CharDB().confirmed={}
-- Data-driven shopping never invents item quantities.
local shop={op='buy',items={{id=9001,count=3}}};bags=2
check(not FR.Done(shop),'shopping awaits all items');bags=3;check(FR.Done(shop),'shopping completes from bags')
check(not FR.Done({op='buy'}),'missing shopping data is not completion')
-- Tracker hooks preserve native behavior and clear recycled quest identities.
local native=0;local block=CreateFrame('Button',nil,UIParent);block.id=2;block.Header=block;block:SetScript('OnEnter',function()native=native+1 end)
QuestObjectiveTracker={usedBlocks={[2]=block}};FR.ScanQuestTracker();M.script(block,'OnEnter')
check(native==1 and ForeverRoutesQuestTooltip:IsShown(),'native mouseover preserved and extended')
QuestObjectiveTracker.usedBlocks={};FR.ScanQuestTracker();FR.HideQuestTooltip();M.script(block,'OnEnter')
check(native==2 and not ForeverRoutesQuestTooltip:IsShown(),'recycled native block drops old identity')
-- Ambiguous equal-count objectives never invent an NPC/model.
ForeverRoutesData.quests={[2]={objectives={{kind='kill',id=12,count=1},{kind='kill',id=13,count=1}}}}
check(FR.ObjectiveMeta(2,1)==nil,'ambiguous objective has no target')
-- One quest display: the window lists the quest log with each quest's place in the route and hides the game's tracker.
do
 data(steps);complete[1]=nil;accepted[1]=true;accepted[7]=true;objectives[1]={{text='Wolves: 1/2',numFulfilled=1,numRequired=2,finished=false}}
 objectives[7]={{text='Boars: 0/5',numFulfilled=0,numRequired=5,finished=false},{text='Done thing',finished=true}}
 local log={{questID=0,title='Elwynn',isHeader=true},{questID=1,title='First'},{questID=7,title='Side quest'}}
 C_QuestLog.GetNumQuestLogEntries=function()return #log end;C_QuestLog.GetInfo=function(i)return log[i]end
 GetQuestLogSpecialItemInfo=function(i)if i==3 then return '|cffffffff|Hitem:555::|h[Horn]|h|r','horn'end end
 local tracker=CreateFrame('Frame','ObjectiveTrackerFrame',UIParent);tracker:Show()
 FR.Config().blizzTracker=nil;FR.Config().compact=nil;FR.Show();FR.Refresh()
 check(not tracker:IsShown(),'game quest tracker hidden while our window shows the quests')
 local texts={};for _,r in ipairs(FR.frame.questRows)do if r:IsShown()then texts[#texts+1]=r.text:GetText()end end
 local all=table.concat(texts,'\n')
 check(FR.frame.questLabel:GetText()=='QUESTS (2)','quest list counts quests, not headers')
 check(all:find('First',1,true)and all:find('jetzt',1,true),'route quest marked as current')
 check(all:find('Side quest',1,true)and all:find('nicht in der Route',1,true),'quest outside the route marked')
 check(all:find('Boars: 0/5',1,true)and not all:find('Done thing',1,true),'open objectives listed, finished ones not')
 check(not all:find('Wolves',1,true)or FR.frame.rows[1].text:GetText():find('Wolves',1,true),'current quest objectives not repeated in the list')
 local b=ForeverRoutesQuestItem1
 check(b and b:IsShown()and b:GetAttribute('item')=='item:555'and b:GetAttribute('type')=='item','quest item button set up')
 check(b:GetParent()==ForeverRoutesQuestItems and ForeverRoutesQuestItems:GetParent()==UIParent,'item button not anchored to the window')
 tracker:Show();check(not tracker:IsShown(),'tracker shown by the game is hidden again')
 InCombatLockdown=function()return true end;tracker:Show();check(tracker:IsShown(),'tracker left alone in combat')
 InCombatLockdown=function()return false end;FR.Refresh();check(not tracker:IsShown(),'tracker hidden after combat')
 SlashCmdList.FOREVERROUTES('tracker');check(tracker:IsShown()and not FR.frame.questLabel:IsShown(),'/fr tracker brings the game tracker back')
 SlashCmdList.FOREVERROUTES('tracker');check(not tracker:IsShown()and FR.frame.questLabel:IsShown(),'/fr tracker hides it again')
 SlashCmdList.FOREVERROUTES('hide');check(tracker:IsShown()and not ForeverRoutesQuestItems:IsShown(),'hiding our window gives the quests back to the game')
 SlashCmdList.FOREVERROUTES('show')
 C_QuestLog.GetNumQuestLogEntries=nil;GetQuestLogSpecialItemInfo=nil;accepted[7]=nil;FR.Refresh()
end
-- The hearthstone: the window says where to bind it when a later hearthstone trip of the route needs that bind.
do
 local hs={{op='a',q=31,n='At the inn',m=0,x=0,y=0,bn='Innkeeper Farley',bm=0,bx=10,by=10,bu=true},{op='a',q=32,n='Far away',m=0,x=900,y=900},{op='t',q=31,n='Back at the inn',m=0,x=0,y=0,v='h'}}
 data(hs);px,py,pm=0,0,0;FR.Show();FR.Refresh()
 local line,now=FR.BindLine(FR.TaskView())
 check(line and line:find('hier binden: Innkeeper Farley',1,true)and now,'bind asked at the inn step')
 accepted[31]=true;FR.Refresh();line=FR.BindLine(FR.TaskView())
 check(line and line:find('später per Ruhestein',1,true),'bind still asked after the step while a hearthstone trip needs it')
 local found=false;for _,r in ipairs(FR.frame.rows)do if r:IsShown()and r.text:GetText():find('Innkeeper Farley',1,true)then found=true end end
 check(found,'bind line shown in the window')
 GetBindLocation=function()return'Goldshire'end;FR.HearthBound()
 check(FR.BindLine(FR.TaskView())==nil,'no bind line once bound there')
 check(FR.TravelHint(hs[3]):find('(nach Innkeeper Farley)',1,true),'hearthstone step says where it leads')
 px,py=500,500;FR.HearthBound();check(FR.BindLine(FR.TaskView())~=nil,'bound elsewhere: asked again')
 FR.CharDB().bound=nil;accepted[31]=nil;px,py=0,0;GetBindLocation=nil
end
-- The camp items are found by their names in the bags, whatever their ids.
do
 local bag={[1]=500001,[2]=500002}
 local names={[500001]='Camp Tent',[500002]='Basic Campfire Kit'}
 local oldInfo,oldCount=GetItemInfo,GetItemCount
 C_Container.GetContainerNumSlots=function(b)return b==0 and 2 or 0 end
 C_Container.GetContainerItemID=function(b,s)return b==0 and bag[s]or nil end
 GetItemInfo=function(id)return names[id]or('Item '..id)end
 GetItemCount=function(id)return names[id]and 1 or 0 end
 local m=FR.CampMaterials()
 check(m.tent and m.fire,'tent and campfire recognized by name')
 FR.CampReport()
 GetItemInfo,GetItemCount=oldInfo,oldCount;C_Container.GetContainerNumSlots=nil;C_Container.GetContainerItemID=nil
end
-- Forever's client: only the C_ functions (no GetItemCount, GetItemInfo, UnitBuff, GetSkillLineInfo).
do
 local oldCount,oldInfo,oldBuff,oldLines=GetItemCount,GetItemInfo,UnitBuff,GetNumSkillLines
 GetItemCount,GetItemInfo,UnitBuff,UnitDebuff,UnitAura,GetNumSkillLines=nil,nil,nil,nil,nil,nil
 local oldC=C_Item.GetItemCount
 C_Item.GetItemCount=function(id)return({[279978]=1,[279981]=2,[2318]=7})[id]or 0 end
 C_UnitAuras={GetAuraDataByIndex=function(u,i,f)if f=='HELPFUL'and i==1 then return{name='Well Rested Shelter',spellId=1,duration=3600,expirationTime=GetTime()+1200}end end}
 GetProfessions=function()return 1,2,nil,nil,5 end
 GetProfessionInfo=function(i)return({'Skinning','Leatherworking',nil,nil,'Cooking'})[i],nil,({30,25,nil,nil,5})[i],75,0,0,({393,165,nil,nil,185})[i]end
 local m=FR.CampMaterials()
 check(m.tent and m.fire and m.leather==7,'camp items counted with C_Item only')
 check(FR.HasProfessions(),'professions from GetProfessions')
 FR.CharDB().campLevel=nil;UnitLevel=function()return 12 end
 local left=FR.CampLeft();check(left and left>1000 and left<=1201,'camp buff read from C_UnitAuras')
 FR.CampReport()
 GetItemCount,GetItemInfo,UnitBuff,GetNumSkillLines=oldCount,oldInfo,oldBuff,oldLines;C_Item.GetItemCount=oldC
 C_UnitAuras=nil;GetProfessions=nil;GetProfessionInfo=nil;UnitLevel=function()return 2 end;FR.CharDB().campLevel=nil
end
-- A quest an item starts reads as the kill it is.
do
 local st={op='a',q=96877,n="Halikor's Hoof",sk='Halikor',sl=12,se=true,si=275723,m=1,x=0,y=0}
 local t=FR.Text(st)
 check(t:find('Töten: Halikor',1,true)and t:find('Elite',1,true)and t:find('annehmen',1,true),'item-started accept reads as a kill: '..t)
 check(FR.StepIcon(st):find('DualWield',1,true),'with the kill icon')
 FR.SetLive({steps={{op='accept',quest=96877,name="Halikor's Hoof",at={m=1,x=0,y=0},startBy={npc=266852,name='Halikor',level=12,elite=true,item=275723}}}})
 check(FR.live[1].sk=='Halikor'and FR.live[1].se,'live step carries the creature');FR.live=nil
end
-- A new live plan keeps the step being done first; an accept right here becomes a hint.
do
 FR.live=nil;FR.CharDB().skipQuests=nil;accepted[501]=true;objectives[501]={{text='Boars: 2/8',type='monster',numFulfilled=2,numRequired=8,finished=false}}
 px,py,pm=0,0,1
 FR.SetLive({steps={{op='work',quest=501,obj=0,at={m=1,x=300,y=0}},{op='accept',quest=502,at={m=1,x=10,y=0}}}})
 check(FR.Advance().q==501,'working on the boars')
 FR.SetLive({steps={{op='accept',quest=502,at={m=1,x=10,y=0}},{op='work',quest=501,obj=0,at={m=1,x=300,y=0}}}})
 check(FR.Advance().q==501,'replanned with an accept first: the boars stay the step')
 local hint=false;for _,c in ipairs(FR.Callouts(FR.TaskView()))do if c.text:find('Nebenbei hier annehmen',1,true)then hint=true end end
 check(hint,'the accept here shows as a hint')
 objectives[501][1].finished=true;objectives[501][1].numFulfilled=8
 FR.SetLive({steps={{op='accept',quest=502,at={m=1,x=10,y=0}},{op='turnin',quest=501,at={m=1,x=0,y=0}}}})
 check(FR.Advance().op=='a','a finished step does not hold the plan')
 FR.live=nil;FR.nearbyAccept=nil;accepted[501]=nil;objectives[501]=nil;pm=0
end
-- A live plan is played in blocks: a new plan waits until the block is nearly done or no longer fits.
do
 FR.live=nil;FR.nextLive=nil;FR.CharDB().skipQuests=nil
 local function plan(ids)local st={};for _,q in ipairs(ids)do st[#st+1]={op='accept',quest=q,at={m=1,x=q,y=0}}end;return{steps=st}end
 FR.SetLive(plan({601,602,603,604,605,606}))
 FR.SetLive(plan({606,605,604,603,602,601}))
 check(FR.Advance().q==601,'a new plan does not replace the block being played')
 check(FR.BlockLeft()==6,'six steps left in the block')
 accepted[601],accepted[602],accepted[603]=true,true,true;FR.CheckBlock()
 check(FR.Advance().q==604,'block kept while more than two are left')
 accepted[604]=true;FR.CheckBlock()
 check(FR.live[1].q==605 and FR.live[2].q==606,'with two left the newest plan takes over (the step being done stays first)')
 FR.SetLive(plan({611,612,613,614}));FR.ReplanNow();check(FR.live[1].q==611,'replan now takes the newest plan at once')
 for q=601,614 do accepted[q]=nil end;FR.live=nil;FR.nextLive=nil
end
-- Exercise events/timers, option window, slash commands and saved visibility.
FR.ToggleMenu();FR.ToggleMenu()
for _,cmd in ipairs({'compact','scale 1.1','lock','position','arrow','show','hide','show'})do SlashCmdList.FOREVERROUTES(cmd)end
M.Fire('QUEST_LOG_UPDATE');M.Advance(2)
check(FR.frame:IsShown()and not FR.Config().hidden,'show state preserved')
check(#M.errors==0,'UI/event errors: '..(#M.errors>0 and M.errors[1].msg or''))
local unknown={};for k in pairs(M.unknownMethods)do unknown[#unknown+1]=k end
check(#unknown==0,'unknown widget methods: '..table.concat(unknown,', '))
local realData=os.getenv('ROUTES_DATA')
if realData then
 dofile(realData);local originalKey=FR.Key;local routeCount=0
 for key,r in pairs(ForeverRoutesData.routes)do
  local base=key:gsub('%-dungeons$','')
  FR.Key=function()return base end;FR.live=nil;FR.undo=nil
  for _,idx in ipairs({1,math.max(1,math.floor(#r.s/2)),#r.s})do
   ForeverRoutesCharDB={index=idx,routeKey=key,dataRevision=r.revision,dungeons=key:find('%-dungeons$')~=nil}
   FR.Refresh()
  end
  routeCount=routeCount+1
 end
 FR.Key=originalKey;check(#M.errors==0,'real route event errors')
 io.write('PASS actual route data: '..routeCount..' variants, start/middle/end HUD refresh\n')
end
io.write('PASS '..count..' checks; no mocked UI/event errors\n')

