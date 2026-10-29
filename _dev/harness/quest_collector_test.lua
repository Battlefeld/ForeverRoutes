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
