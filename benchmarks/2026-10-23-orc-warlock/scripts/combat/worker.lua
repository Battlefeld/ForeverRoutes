local root,adapter=assert(arg[1]),assert(arg[2])
local input=assert(loadstring('return '..io.read('*a'),'serialized route request'))
setfenv(input,{})
local job=input()
dofile(root..'/Classes/warlock.lua');dofile(root..'/CombatEngine.lua')
dofile(root..'/UniversalRules.lua');dofile(root..'/Optimizers.lua')
dofile(adapter..'/route_overlay.lua')
local f=assert(io.open(root..'/CombatEngine.lua','rb'));local source=f:read('*a');f:close()
FUSLive.installRouteRunner(source)
local P=FUSLive
local function json(v)
 local t=type(v)
 if t=='nil'then return 'null'elseif t=='boolean'then return tostring(v)
 elseif t=='number'then assert(v==v and math.abs(v)<math.huge,'Nonfinite output');return string.format('%.17g',v)
 elseif t=='string'then return '"'..v:gsub('[%z\1-\31\\"]',function(c)local map={['"']='\\"',['\\']='\\\\',['\n']='\\n',['\r']='\\r',['\t']='\\t'};return map[c]or string.format('\\u%04x',c:byte())end)..'"'
 elseif t=='table'then
  local keys,count,array={},0,true
  for k in pairs(v)do count=count+1;keys[#keys+1]=k;if type(k)~='number'or k<1 or k%1~=0 then array=false end end
  if count==0 then return '{}'end
  if array and #v==count then local parts={}for i=1,count do parts[i]=json(v[i])end;return '['..table.concat(parts,',')..']'end
  table.sort(keys,function(a,b)return tostring(a)<tostring(b)end);local parts={}
  for _,k in ipairs(keys)do parts[#parts+1]=json(tostring(k))..':'..json(v[k])end;return '{'..table.concat(parts,',')..'}'
 end
 error('Unsupported result type '..t)
end
local function compile(fight)
 local p=P.copy(fight.player);p.trace=true
 assert(p.level>=5,'Pinned ForeverSim product gate: levels 1-4 unsupported')
 assert(not p.minSimLevel or p.minSimLevel>=5,'Cannot bypass source level gate')
 local data=FUSLiveData.warlock
 local spec=P.compile(data,p,P.combine(p.equipped or {}),fight.choice,fight.duration or 300,fight.seed or 1)
 spec.targetHP=fight.enemy.health
 return spec
end
local result={engineVersion=P.version,rows={},combatSeconds=0,idleSeconds=0,limitations={'No validation against recorded leveling sessions','Pet health, threat and tanking are not modeled','Buff-specific out-of-combat regeneration requires validation','Player weapon swing timers are not carried across targets','Ammunition, consumable stock, trainer unlocks and equipment progression require route evidence'}}
local state=job.initialState
local lastSpec=job.previousFight and compile(job.previousFight)or nil
for _,event in ipairs(job.events)do
 if event.kind=='idle'then
  if state then assert(lastSpec,'Missing previous resource profile');state=P.routeIdle(state,lastSpec,event.seconds,event.recovery)end
  result.idleSeconds=result.idleSeconds+event.seconds
  result.rows[#result.rows+1]={kind='idle',seconds=event.seconds,state=state and P.copy(state)}
 else
  local spec=compile(event)
  if job.compareBaseline then
   local base=P.run(P.copy(spec));local adapted=P.routeRun(P.copy(spec))
   adapted.routeState=nil;assert(P.canonical(base)==P.canonical(adapted),'Fresh-state adapter differs from original engine')
  end
  spec.routeState=state
  local fight=P.routeRun(spec)
  local seconds=fight.killTime or fight.deathTime or fight.duration
  result.combatSeconds=result.combatSeconds+seconds
  result.rows[#result.rows+1]={kind='fight',level=event.player.level,mob=event.enemy.id,result=fight,maximum=spec.initial,approximateLevel=event.player.level<10,startState=state and P.copy(state)or {mana=spec.initial.mana,health=spec.initial.health}}
  state=fight.routeState;lastSpec=spec
  if not fight.killed then result.stopped=fight.deathTime and 'player_died'or'target_not_killed';break end
 end
end
result.state=state;result.totalSeconds=result.combatSeconds+result.idleSeconds
io.write(json(result),'\n')
