-- Play times for calibrating the routes: when a quest was taken, an objective finished, a quest handed in, with the
-- fighting and the kills between. Written beside the quest observations (ForeverRoutesQuestDB, per build and locale);
-- no account or character names. Read-only: nothing here acts in the game.
local T = {}; ForeverRoutes.Timing = T
local LIMIT = 20000
local combatSince, combat, fights, kills, killXP = nil, 0, 0, 0, 0
local lastXP, lastMax, lastTurnin, lastStep = nil, nil, -10, nil
local objectives = {} -- quest -> objective -> finished
local session = {started = nil, entries = 0, combat = 0, fights = 0, kills = 0}
local function now() return type(GetTime) == 'function' and GetTime() or 0 end
local function place()
 if type(UnitPosition) ~= 'function' then return nil end
 local ok, x, y, _, m = pcall(UnitPosition, 'player')
 if ok and type(x) == 'number' and type(y) == 'number' then return math.floor(x + 0.5), math.floor(y + 0.5), m end
end
local function record(event, fields)
 local C = ForeverRoutes.QuestCollector
 local d = C and C.DB and C.DB(); if not d then return end
 d.timeline = d.timeline or {}
 if #d.timeline >= LIMIT then table.remove(d.timeline, 1) end
 local e = fields or {}
 e.e, e.t = event, time()
 e.lvl = UnitLevel and UnitLevel('player') or nil
 e.xp = UnitXP and UnitXP('player') or nil
 e.x, e.y, e.m = place()
 if combatSince then combat = combat + (now() - combatSince); combatSince = now() end
 -- what happened since the entry before: seconds in combat, fights, kills that gave XP and their XP
 e.c, e.f, e.k, e.kx = math.floor(combat + 0.5), fights, kills, killXP
 session.combat, session.fights, session.kills = session.combat + combat, session.fights + fights, session.kills + kills
 combat, fights, kills, killXP = 0, 0, 0, 0
 d.timeline[#d.timeline+1] = e
 session.entries = session.entries + 1
end
T.Record = record
local function scanObjectives()
 if not C_QuestLog or type(C_QuestLog.GetNumQuestLogEntries) ~= 'function' then return end
 local ok, n = pcall(C_QuestLog.GetNumQuestLogEntries); if not ok or type(n) ~= 'number' then return end
 for i = 1, n do
  local okI, info = pcall(C_QuestLog.GetInfo, i)
  local q = okI and type(info) == 'table' and info.questID
  if q then
   local okO, list = pcall(C_QuestLog.GetQuestObjectives, q)
   if okO and type(list) == 'table' then
    local known = objectives[q]
    objectives[q] = objectives[q] or {}
    for o, ob in ipairs(list) do
     local fin = ob.finished == true
     -- only a change seen while playing counts: what was finished at the first look has no time
     if known and fin and objectives[q][o] == false then record('objective', {q = q, o = o}) end
     objectives[q][o] = fin
    end
   end
  end
 end
end
local function step()
 local FR = ForeverRoutes
 if type(FR.Steps) ~= 'function' or type(FR.index) ~= 'number' then return end
 local s = FR.Steps()[FR.index]
 local key = s and (tostring(FR.index) .. ':' .. tostring(s.op) .. ':' .. tostring(s.q or s.tg or '') .. ':' .. tostring(s.o or '')) or 'end'
 if key ~= lastStep then
  lastStep = key
  record('step', {i = FR.index, op = s and s.op or nil, q = s and s.q or nil, o = s and s.o or nil})
 end
end
local frame = CreateFrame('Frame')
for _, event in ipairs({'PLAYER_LOGIN', 'PLAYER_LOGOUT', 'QUEST_ACCEPTED', 'QUEST_TURNED_IN', 'QUEST_LOG_UPDATE', 'PLAYER_XP_UPDATE',
 'PLAYER_LEVEL_UP', 'PLAYER_REGEN_DISABLED', 'PLAYER_REGEN_ENABLED', 'PLAYER_FLAGS_CHANGED', 'PLAYER_DEAD'}) do frame:RegisterEvent(event) end
local afk = false
frame:SetScript('OnEvent', function(_, event, a, b)
 if event == 'PLAYER_LOGIN' then
  session.started = time()
  lastXP, lastMax = UnitXP and UnitXP('player') or nil, UnitXPMax and UnitXPMax('player') or nil
  scanObjectives(); record('login')
 elseif event == 'PLAYER_LOGOUT' then record('logout')
 elseif event == 'QUEST_ACCEPTED' then record('accept', {q = a})
 elseif event == 'QUEST_TURNED_IN' then lastTurnin = now(); objectives[a] = nil; record('turnin', {q = a, qx = b})
 elseif event == 'QUEST_LOG_UPDATE' then scanObjectives()
 elseif event == 'PLAYER_REGEN_DISABLED' then combatSince = now(); fights = fights + 1
 elseif event == 'PLAYER_REGEN_ENABLED' then
  if combatSince then combat = combat + (now() - combatSince) end
  combatSince = nil
 elseif event == 'PLAYER_XP_UPDATE' then
  local xp, max = UnitXP and UnitXP('player') or nil, UnitXPMax and UnitXPMax('player') or nil
  if type(xp) == 'number' and type(lastXP) == 'number' then
   local gain = xp - lastXP
   if gain < 0 and type(lastMax) == 'number' then gain = gain + lastMax end -- a level was reached
   -- XP within a second of a hand-in is the quest's; the rest comes from kills
   if gain > 0 and now() - lastTurnin > 1 then kills = kills + 1; killXP = killXP + gain end
  end
  lastXP, lastMax = xp, max
 elseif event == 'PLAYER_LEVEL_UP' then record('level', {to = a})
 elseif event == 'PLAYER_DEAD' then record('death')
 elseif event == 'PLAYER_FLAGS_CHANGED' and a == 'player' and type(UnitIsAFK) == 'function' then
  local away = UnitIsAFK('player') == true
  if away ~= afk then afk = away; record(away and 'afk' or 'back') end
 end
end)
local elapsed = 0
frame:SetScript('OnUpdate', function(_, delta) elapsed = elapsed + delta; if elapsed >= 1 then elapsed = 0; step() end end)
SLASH_FOREVERROUTESTIME1 = '/frtime'
SlashCmdList.FOREVERROUTESTIME = function()
 local minutes = session.started and math.floor((time() - session.started) / 60) or 0
 print(string.format('ForeverRoutes Zeitmessung: %d Minuten in dieser Sitzung, %d Einträge, %d Kämpfe, %d Minuten im Kampf, %d Kills mit XP. Wird beim Ausloggen gespeichert.',
  minutes, session.entries, session.fights + fights, math.floor((session.combat + combat) / 60), session.kills + kills))
end
