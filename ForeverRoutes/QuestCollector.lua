-- Client observations, partitioned by build and locale. No account or character identifiers.
local C = {}; ForeverRoutes.QuestCollector = C
local queue, queued, requested, pending = {}, {}, {}, nil
local dirty, sent = {}, {}
local function call(fn, ...)
 if type(fn) ~= 'function' then return nil end
 local ok, value = pcall(fn, ...); if ok then return value end
end
local function identity()
 local version, build = GetBuildInfo()
 if not tostring(version):match('^1%.60%.') then return nil end
 return version .. '.' .. build, GetLocale(), call(GetCurrentRegion) or 0
end
local function db()
 local build, locale, region = identity(); if not build then return nil end
 ForeverRoutesQuestDB = ForeverRoutesQuestDB or {schema=1, captures={}}
 local key = build .. '/' .. region .. '/' .. locale
 local captures = ForeverRoutesQuestDB.captures
 captures[key] = captures[key] or {build=build, locale=locale, region=region, quests={}, turnins={}}
 return captures[key]
end
local function position()
 local map = C_Map and call(C_Map.GetBestMapForUnit, 'player')
 local pos = map and call(C_Map.GetPlayerMapPosition, map, 'player')
 if pos and pos.x and pos.y then return {uiMapID=map, x=pos.x, y=pos.y} end
end
function C.Capture(id)
 id = tonumber(id); local d = db()
 if not d or not id or id <= 0 or not C_QuestLog then return end
 local title = call(C_QuestLog.GetTitleForQuestID, id)
 if type(title) ~= 'string' or title == '' then return end
 local q = d.quests[tostring(id)] or {id=id}
 q.title, q.seenAt = title, time()
 local objectives = call(C_QuestLog.GetQuestObjectives, id)
 if type(objectives) == 'table' then
  q.objectives = {}
  for _, o in ipairs(objectives) do
   -- Progress and completion are character state, not static quest requirements.
   q.objectives[#q.objectives+1] = {type=o.type, text=o.text, numRequired=o.numRequired}
  end
 end
 d.quests[tostring(id)] = q
 if sent[id]~=title then dirty[id]=q end
end
function C.Enqueue(id)
 id = tonumber(id)
 if id and id > 0 and id == math.floor(id) and not queued[id] and not requested[id] then
  queued[id] = true; requested[id] = true; queue[#queue+1] = id
 end
end
local function scan()
 local count = C_QuestLog and call(C_QuestLog.GetNumQuestLogEntries) or 0
 for i=1,count do
  local info = call(C_QuestLog.GetInfo, i)
  if info and info.questID then C.Enqueue(info.questID); C.Capture(info.questID) end
 end
end
local function tick()
 if not db() or not C_QuestLog then return end
 if pending then
  if GetTime() - pending.started < 15 then return end
  queued[pending.id] = nil; pending = nil -- A timeout is unknown, never a deleted quest.
 end
 local id = table.remove(queue, 1)
 if id then
  pending = {id=id, started=GetTime()} -- Set before requesting: cached responses can be synchronous.
  call(C_QuestLog.RequestLoadQuestByID, id)
 end
end
local function escape(s)
 return '"' .. s:gsub('[%z\1-\31\\"]', function(c)
  if c == '"' then return '\\"' end
  if c == '\\' then return '\\\\' end
  return string.format('\\u%04x', string.byte(c))
 end) .. '"'
end
local function json(v)
 if type(v)=='string' then return escape(v) end
 if type(v)=='number' then return tostring(v) end
 if type(v)=='boolean' then return v and 'true' or 'false' end
 if type(v)~='table' then return 'null' end
 local keys={}; for k in pairs(v) do keys[#keys+1]=k end
 table.sort(keys,function(a,b)return tostring(a)<tostring(b)end)
 local out={}
 -- Always objects; numeric keys are stringified and importer normalizes lists.
 for _,k in ipairs(keys) do out[#out+1]=escape(tostring(k))..':'..json(v[k]) end
 return '{'..table.concat(out,',')..'}'
end
function C.Export()
 ForeverRoutesQuestExport = json(ForeverRoutesQuestDB or {schema=1,captures={}})
 return ForeverRoutesQuestExport
end
-- Send only public quest titles through an idle bridge; objective/turn-in observations
-- also persist through SavedVariables. No character snapshot is attached to this job.
local transmitting=false
function C.FlushLive()
 local client=ForeverCoreClient
 if transmitting or not client or not client.clientDataSync or not client.Request or client.pending or #(client.queue or {})>0 then return end
 if InCombatLockdown and InCombatLockdown() then return end
 local d=db();if not d then return end
 local batch, titles, count={}, {}, 0
 for id,q in pairs(dirty) do
  batch[tostring(id)]={id=id,title=q.title,seenAt=q.seenAt};titles[id]=q.title;count=count+1
  if count>=24 then break end
 end
 if count==0 then return end
 local key=d.build..'/'..d.region..'/'..d.locale
 local document={schema=1,captures={[key]={build=d.build,region=d.region,locale=d.locale,quests=batch,turnins={}}}}
 local raw=json(document);if #raw>20000 then return end
 transmitting=true
 local queued=client.Request('quest-observations',{capture=raw},function(result)
  transmitting=false
  if not result then return end
  for id,title in pairs(titles) do sent[id]=title;if dirty[id] and dirty[id].title==title then dirty[id]=nil end end
  if client.ApplyDataStatus then client.ApplyDataStatus(result.dataStatus)end
 end)
 if not queued then transmitting=false end
end
local frame=CreateFrame('Frame')
for _,event in ipairs({'PLAYER_LOGIN','QUEST_LOG_UPDATE','QUEST_DATA_LOAD_RESULT','QUEST_DETAIL','QUEST_ACCEPTED','QUEST_TURNED_IN','PLAYER_LOGOUT'}) do frame:RegisterEvent(event) end
frame:SetScript('OnEvent',function(_,event,id,xp,money)
 if event=='PLAYER_LOGIN' then
  if not db() then return end
  scan()
  -- Existing route IDs are candidate queries, not proof of availability.
  local data=ForeverRoutesData or {}
  for _,group in ipairs({data.routes or {},data.fallbacks or {}}) do
   for _,route in pairs(group) do for _,step in ipairs(route.s or {}) do if step.q then C.Enqueue(step.q) end end end
  end
 elseif event=='QUEST_LOG_UPDATE' then scan()
 elseif event=='QUEST_DATA_LOAD_RESULT' then
  if xp then C.Capture(id) end
  queued[id]=nil; if pending and pending.id==id then pending=nil end
 elseif event=='QUEST_DETAIL' then C.Capture(call(GetQuestID))
 elseif event=='QUEST_ACCEPTED' then C.Enqueue(id); C.Capture(id)
 elseif event=='QUEST_TURNED_IN' then
  local d=db(); if not d then return end
  C.Capture(id)
  if #d.turnins>=5000 then table.remove(d.turnins,1) end
  d.turnins[#d.turnins+1]={id=id,observedXP=xp,observedMoney=money,playerLevel=UnitLevel('player'),seenAt=time(),position=position()}
 elseif event=='PLAYER_LOGOUT' then C.Export() end
end)
local elapsed, liveElapsed=0, 0
frame:SetScript('OnUpdate',function(_,delta) elapsed=elapsed+delta;liveElapsed=liveElapsed+delta; if elapsed>=0.5 then elapsed=0; tick() end;if liveElapsed>=15 then liveElapsed=0;C.FlushLive()end end)
SLASH_FOREVERROUTESCAPTURE1='/frcapture'
SlashCmdList.FOREVERROUTESCAPTURE=function(text)
 local id=tonumber(text)
 if id then requested[id]=nil; C.Enqueue(id); print('ForeverRoutes: Quest-Abfrage vorgemerkt: '..id)
 else
  C.Export(); local d=db(); local n=0; if d then for _ in pairs(d.quests) do n=n+1 end end
  print('ForeverRoutes: '..n..' Questdatensätze. Export wird beim Ausloggen in SavedVariables gespeichert.')
 end
end
