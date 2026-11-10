local frame, clock, requests = {}, 0, 0
function CreateFrame() return frame end
function frame:RegisterEvent() end
function frame:SetScript(name,fn) self[name]=fn end
function GetBuildInfo() return '1.60.1','70685' end
function GetLocale() return 'enUS' end
function GetCurrentRegion() return 3 end
function GetTime() return clock end
function time() return 123 end
function UnitLevel() return 20 end
ForeverRoutes={}; SlashCmdList={}
C_QuestLog={
 GetTitleForQuestID=function(id) if id==971 then return 'Knowledge "in" the Deeps' end end,
 GetQuestObjectives=function() return {{text='Items 1/2',type='item',numRequired=2,numFulfilled=1,finished=false}} end,
 RequestLoadQuestByID=function(id)
  requests=requests+1
  frame.OnEvent(frame,'QUEST_DATA_LOAD_RESULT',id,id==971)
 end,
 GetNumQuestLogEntries=function()return 0 end,
}
dofile('ForeverRoutes/QuestCollector.lua')
local c=ForeverRoutes.QuestCollector
c.Enqueue(971); c.Enqueue(971); frame.OnUpdate(frame,.5)
assert(requests==1,'duplicate queued query')
c.Enqueue(971); frame.OnUpdate(frame,.5); assert(requests==1,'completed request repeated')
local d=ForeverRoutesQuestDB.captures['1.60.1.70685/3/enUS']
assert(d.quests['971'].title=='Knowledge "in" the Deeps')
assert(d.quests['971'].objectives[1].numFulfilled==nil,'character progress leaked')
frame.OnEvent(frame,'QUEST_TURNED_IN',971,999,123)
assert(d.turnins[1].observedXP==999 and d.quests['971'].xp==nil,'observed XP became base XP')
assert(c.Export():find('Knowledge \\"in\\" the Deeps',1,true),'JSON escaping failed')
C_QuestLog.RequestLoadQuestByID=function()requests=requests+1 end
c.Enqueue(972); c.Enqueue(973); frame.OnUpdate(frame,.5)
clock=16; frame.OnUpdate(frame,.5)
assert(requests==3,'timeout did not release request slot')
assert(d.quests['972']==nil,'timeout became a quest fact')
print('Quest collector: synchronous responses, deduplication, timeout and contextual XP passed')

local sentPayload, callback
ForeverCoreClient={clientDataSync=true,queue={},Request=function(kind,payload,fn)
 assert(kind=='quest-observations');sentPayload=payload;callback=fn;return true
end}
c.FlushLive();assert(sentPayload and sentPayload.capture:find('Knowledge',1,true),'live export missing')
callback({imported=1});sentPayload=nil;c.FlushLive();assert(not sentPayload,'acknowledged title sent again')
print('Live quest export uses an idle bridge and waits for acknowledgement')

-- A planned quest its giver does not offer is noted once, with what the giver offers; an offered one is not.
local px,py=100,200
function UnitPosition() return px,py,0,1 end
function UnitGUID() return 'Creature-0-4444-1-55-3143-0000ABCDEF' end
ForeverRoutes.index=1
ForeverRoutes.Steps=function() return {{op='a',q=844,n='Plainstrider Menace',m=1,x=100,y=210},{op='a',q=870,n='The Forgotten Pools',m=1,x=110,y=205},{op='w',q=844}} end
C_QuestLog.IsOnQuest=function() return false end
C_QuestLog.IsQuestFlaggedCompleted=function() return false end
C_GossipInfo={GetAvailableQuests=function() return {{questID=870,title='The Forgotten Pools'},{questID=5041,title='Supplies'}} end}
local printed=0; local realPrint=print; print=function() printed=printed+1 end
frame.OnEvent(frame,'GOSSIP_SHOW'); frame.OnEvent(frame,'GOSSIP_SHOW')
print=realPrint
assert(#d.notOffered==1 and d.notOffered[1].id==844 and d.notOffered[1].npc==3143,'missing quest not noted once')
assert(d.notOffered[1].offered[1]==870 and d.notOffered[1].offered[2]==5041 and d.notOffered[1].playerLevel==20,'offer list or level missing')
assert(printed==1,'the player is told once')
-- far from the step's place, with a quest list the game does not give, or with the quest in the log: nothing
px=900; ForeverRoutes.Steps=function() return {{op='a',q=871,n='Disrupt the Attacks',m=1,x=100,y=210}} end
frame.OnEvent(frame,'GOSSIP_SHOW'); assert(#d.notOffered==1,'noted far from the giver')
px=100; C_GossipInfo.GetAvailableQuests=function() return nil end
frame.OnEvent(frame,'GOSSIP_SHOW'); assert(#d.notOffered==1,'noted without a quest list')
C_GossipInfo.GetAvailableQuests=function() return {} end; C_QuestLog.IsOnQuest=function(q) return q==871 end
frame.OnEvent(frame,'GOSSIP_SHOW'); assert(#d.notOffered==1,'noted a quest in the log')
-- the greeting window: ids from GetAvailableQuestInfo
C_QuestLog.IsOnQuest=function() return false end
function GetNumAvailableQuests() return 1 end
function GetAvailableQuestInfo() return false,0,false,false,872 end
frame.OnEvent(frame,'QUEST_GREETING'); assert(#d.notOffered==2 and d.notOffered[2].id==871,'greeting window not read')
assert(c.Export():find('"notOffered"',1,true),'not offered quests missing in the export')
print('Quests a giver does not offer are noted once, only at the giver and only with a readable offer list')
