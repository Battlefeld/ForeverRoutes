-- Route-only entry point, loaded in a separate process. Never edits the snapshot.
-- Damage, ranks, casts, hits and procs still execute the pinned original code.
local P=FUSLive
local function once(text,old,new)
 local a,b=text:find(old,1,true)
 assert(a and not text:find(old,b+1,true),'Pinned combat anchor changed: '..old:sub(1,50))
 return text:sub(1,a-1)..new..text:sub(b+1)
end
function P.installRouteRunner(source)
 local a=assert(source:find('function P.run(spec)',1,true))
 local b=assert(source:find('function P.candidates(data,p)',a,true))
 local code=source:sub(a,b-1)
 code=once(code,'function P.run(spec)','function P.routeRun(spec)')
 code=once(code,'local energy,rage,combo=',[[if spec.routeState then
  local s=spec.routeState
  mana=math.min(spec.initial.mana,s.mana);health=math.min(spec.initial.health,s.health)
  assert(mana>=0 and health>0,'Invalid resumed resources')
  cds=copy(s.cooldowns or {});buffs=copy(s.buffs or {});lastSpend=s.lastSpend or -1e9
  shards=s.shards or shards
 end
 local routeReady=0
 local energy,rage,combo=]])
 code=once(code,'local procReady={}','local procReady=copy(spec.routeState and spec.routeState.procReady or {})')
 code=once(code,'-- A cast may start before the boss dies',"routeReady=gcdEnd\n    -- A cast may start before the boss dies")
 code=once(code,'local function refreshLead(a,policy)',[[local resumedReady=spec.routeState and spec.routeState.ready or 0
 if resumedReady>0 then advance(math.min(duration,resumedReady))end
 local function refreshLead(a,policy)]])
 code=once(code,'a.remainingMana=x.mana;a.lastRegen=0;',[[a.remainingMana=x.mana;a.lastRegen=0;
 local resumed=spec.routeState and (spec.routeState.actors or {})[a.key]
 if resumed then a.remainingMana=math.min(a.mana,resumed.mana);a.cooldowns=copy(resumed.cooldowns or {}) end;]])
 code=once(code,"for i,a in ipairs(actorStates)do if a.generic then schedule(0,2,'actorAct','pet',{actor=i})end;if a.spell then schedule(0,2,'actorAct',a.spell.key,{actor=i})end;if a.melee then schedule(0,0,'actorSwing','pet:melee',{actor=i});a.nextSwing=q[#q]end end",[[for i,a in ipairs(actorStates)do
 local resumed=spec.routeState and (spec.routeState.actors or {})[a.key]
 local ready=resumed and resumed.ready or 0;local swing=resumed and resumed.swing or 0
 if a.generic then schedule(ready,2,'actorAct','pet',{actor=i})end
 if a.spell then schedule(ready,2,'actorAct',a.spell.key,{actor=i})end
 if a.melee then schedule(swing,0,'actorSwing','pet:melee',{actor=i});a.nextSwing=q[#q]end
 end]])
 code=once(code,'return {dps=total/',[[local elapsed=killTime or deathTime or duration
 local function remaining(values)local out={}for k,v in pairs(values or {})do if v>elapsed then out[k]=v-elapsed end end return out end
 local state={mana=mana,health=health,shards=shards,ready=math.max(0,routeReady-elapsed),lastSpend=lastSpend-elapsed,cooldowns=remaining(cds),procReady=remaining(procReady),buffs={},actors={}}
 for k,v in pairs(buffs)do if v.expires>elapsed and v.charges>0 then state.buffs[k]=copy(v);state.buffs[k].expires=v.expires-elapsed end end
 for i,actor in ipairs(actorStates)do
  local ready,swing=nil,nil
  for _,event in ipairs(q)do if event.actor==i and event.t>=elapsed then
   if event.kind=='actorAct' then ready=math.min(ready or math.huge,event.t-elapsed)end
   if event.kind=='actorSwing' then swing=math.min(swing or math.huge,event.t-elapsed)end
  end end
  local regen=math.max(0,elapsed-actor.lastRegen)*(actor.manaRegen or 0)
  state.actors[actor.key]={mana=math.min(actor.mana,actor.remainingMana+regen),maxMana=actor.mana,regen=actor.manaRegen or 0,cooldowns=remaining(actor.cooldowns),ready=ready or 0,swing=swing or 0}
 end
 return {routeState=state,dps=total/]])
 local has=function(a,k)if not a then return true end for _,x in ipairs(a)do if x==k then return true end end return false end
 local schoolMatches=function(schools,b)if not schools then return true end for _,k in ipairs(b.schools or {b.school})do if has(schools,k)then return true end end return false end
 local prefix='local copy,clamp,has,schoolMatches=...;local P=FUSLive;'
 assert(loadstring(prefix..code,'ForeverRoutes isolated resource adapter'))(P.copy,function(x,l,h)return math.max(l,math.min(h,x))end,has,schoolMatches)
end
function P.routeIdle(state,spec,seconds,recovery)
 assert(seconds>=0,'Negative idle duration')
 local s=P.copy(state);local outside=math.max(0,seconds-math.max(0,s.lastSpend+5))
 s.mana=math.min(spec.initial.mana,s.mana+(spec.mp5 or 0)/5*seconds+(spec.castingRegen or 0)*(seconds-outside)+(spec.naturalRegen or 0)*outside)
 local r=recovery or {}
 s.mana=math.min(spec.initial.mana,s.mana+(r.manaPerSecond or 0)*seconds)
 s.health=math.min(spec.initial.health,s.health+(r.healthPerSecond or 0)*seconds)
 s.lastSpend=s.lastSpend-seconds
 s.ready=math.max(0,(s.ready or 0)-seconds)
 for _,name in ipairs({'cooldowns','procReady'})do for k,v in pairs(s[name])do s[name][k]=math.max(0,v-seconds)end end
 for k,v in pairs(s.buffs)do v.expires=v.expires-seconds;if v.expires<=0 then s.buffs[k]=nil end end
 for _,actor in pairs(s.actors)do
  actor.mana=math.min(actor.maxMana,actor.mana+actor.regen*seconds)
  actor.ready=math.max(0,actor.ready-seconds);actor.swing=math.max(0,actor.swing-seconds)
  for k,v in pairs(actor.cooldowns)do actor.cooldowns[k]=math.max(0,v-seconds)end
 end
 return s
end
