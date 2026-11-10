local frames, clock, wall = {}, 0, 1000
function CreateFrame() local f={}; function f:RegisterEvent() end; function f:SetScript(n,fn) self[n]=fn end; frames[#frames+1]=f; return f end
function GetBuildInfo() return '1.60.1','70685' end
function GetLocale() return 'enUS' end
function GetCurrentRegion() return 3 end
function GetTime() return clock end
function time() return wall end
local level, xp, xpMax = 29, 100, 30000
function UnitLevel() return level end
function UnitXP() return xp end
function UnitXPMax() return xpMax end
function UnitPosition() return -5459.4, -2440.2, 0, 1 end
local finished = false
ForeverRoutes={}; SlashCmdList={}
C_QuestLog={
 GetNumQuestLogEntries=function() return 1 end,
 GetInfo=function() return {questID=4841} end,
 GetQuestObjectives=function() return {{text='Galak Scout slain',type='monster',numRequired=12,finished=finished}} end,
 GetTitleForQuestID=function() return 'Pacify the Centaur' end,
 RequestLoadQuestByID=function() end,
}
dofile('ForeverRoutes/QuestCollector.lua')
dofile('ForeverRoutes/Timing.lua')
local timing=frames[#frames]
local function fire(...) timing.OnEvent(timing, ...) end
fire('PLAYER_LOGIN')
local d=ForeverRoutesQuestDB.captures['1.60.1.70685/3/enUS']
assert(#d.timeline==1 and d.timeline[1].e=='login' and d.timeline[1].lvl==29 and d.timeline[1].x==-5459 and d.timeline[1].m==1,'login entry')
wall=1010; fire('QUEST_ACCEPTED',4841)
-- two fights, 20 and 15 seconds, two kills with XP
clock=100; fire('PLAYER_REGEN_DISABLED'); clock=120; fire('PLAYER_REGEN_ENABLED'); xp=xp+210; fire('PLAYER_XP_UPDATE')
clock=130; fire('PLAYER_REGEN_DISABLED'); clock=145; fire('PLAYER_REGEN_ENABLED'); xp=xp+190; fire('PLAYER_XP_UPDATE')
fire('QUEST_LOG_UPDATE'); assert(#d.timeline==2,'an unfinished objective is no entry')
finished=true; wall=1200; fire('QUEST_LOG_UPDATE'); fire('QUEST_LOG_UPDATE')
local o=d.timeline[3]
assert(#d.timeline==3 and o.e=='objective' and o.q==4841 and o.o==1,'objective finished once')
assert(o.c==35 and o.f==2 and o.k==2 and o.kx==400,'fights and kills since the entry before: '..o.c..' '..o.f..' '..o.k..' '..o.kx)
-- the hand-in: its XP is the quest's, not a kill's
clock=300; wall=1300; fire('QUEST_TURNED_IN',4841,2100); xp=xp+2100; fire('PLAYER_XP_UPDATE')
local t=d.timeline[4]
assert(t.e=='turnin' and t.q==4841 and t.qx==2100 and t.k==0 and t.c==0,'hand-in entry')
clock=310; fire('PLAYER_REGEN_DISABLED'); clock=318; wall=1400; fire('PLAYER_LEVEL_UP',30)
assert(d.timeline[5].e=='level' and d.timeline[5].to==30 and d.timeline[5].c==8 and d.timeline[5].k==0,'time in a running fight counts; quest XP is no kill')
-- the route's step
ForeverRoutes.index=3; ForeverRoutes.Steps=function() return {{},{},{op='w',q=4841,o=1}} end
timing.OnUpdate(timing,1); timing.OnUpdate(timing,1)
assert(#d.timeline==6 and d.timeline[6].e=='step' and d.timeline[6].i==3 and d.timeline[6].q==4841,'step entered once')
assert(ForeverRoutes.QuestCollector.Export():find('"timeline"',1,true),'timeline missing in the export')
assert(not ForeverRoutes.QuestCollector.Export():find('name',1,true),'no names in the export')
print('Timing: quest events, fights and kills between them, level and route steps are recorded')
