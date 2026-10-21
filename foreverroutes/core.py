"""Offline route evaluation. No Blizzard API calls, no imported guide code.

The compact state graph and Dijkstra solver are correct for the supplied small
world; 1-60 needs decomposition / heuristics and proper measured world data.
"""
from __future__ import annotations
from dataclasses import dataclass
from functools import lru_cache
import heapq
import json
from pathlib import Path
from typing import NamedTuple


class WorldError(ValueError):
    pass


class RouteError(ValueError):
    pass


@dataclass(frozen=True)
class Objective:
    kind: str  # kill, collect
    target: str
    location: str
    count: int
    drop_chance: float = 1.0


@dataclass(frozen=True)
class Quest:
    id: int
    giver: str
    turnin: str
    level: int
    xp: int
    prereqs: tuple[int, ...]
    faction: str | None
    classes: tuple[str, ...]
    races: tuple[str, ...]
    objectives: tuple[Objective, ...]
    excl: tuple[int, ...] = ()


@dataclass(frozen=True)
class Mob:
    hp: float
    xp: float
    kill_seconds: float | None = None
    downtime: float = 0.0
    level: int | None = None  # optional mob-level gate


class World:
    def __init__(self, doc: dict):
        self.metadata = {k:doc.get(k) for k in ('synthetic','source','revision','description')}
        self.nodes = set(doc['nodes'])
        self.graph = {node: [] for node in self.nodes}
        for a, b, sec in doc['edges']:
            if a not in self.nodes or b not in self.nodes or not isinstance(sec, (int,float)) or sec < 0:
                raise WorldError(f'Invalid travel edge {a!r}->{b!r}')
            self.graph[a].append((b, float(sec)))
            self.graph[b].append((a, float(sec)))
        self.level_xp = {int(k): int(v) for k,v in doc['level_xp'].items()}
        if not self.level_xp or any(x<=0 for x in self.level_xp.values()):
            raise WorldError('XP-to-next thresholds must be positive')
        self.mobs = {key: Mob(**val) for key,val in doc['mobs'].items()}
        for name, m in self.mobs.items():
            if m.hp <= 0 or m.xp < 0 or m.downtime < 0 or (m.level is not None and m.level < 1):
                raise WorldError(f'Invalid mob: {name}')
        self.max_open = int(doc.get('max_open', 40))
        self.dps = float(doc['dps'])
        if self.dps <= 0 or self.max_open < 1: raise WorldError('Invalid DPS or max_open')
        qs = []
        for raw in doc['quests']:
            objs = tuple(Objective(**o) for o in raw['objectives'])
            qs.append(Quest(id=int(raw['id']),giver=raw['giver'],turnin=raw['turnin'],
                level=int(raw['level']),xp=int(raw['xp']),prereqs=tuple(map(int,raw.get('prereqs',[]))),
                faction=raw.get('faction'),classes=tuple(raw.get('classes',[])),
                races=tuple(raw.get('races',[])), objectives=objs,
                excl=tuple(map(int,raw.get('excl',[])))))
        self.quests = tuple(sorted(qs,key=lambda x:x.id))
        self.qix = {q.id:i for i,q in enumerate(self.quests)}
        if len(self.qix)!=len(qs):raise WorldError('Duplicate quest ID')
        for q in qs:
            if q.giver not in self.nodes or q.turnin not in self.nodes:
                raise WorldError(f'Unknown giver/turn-in for {q.id}')
            if q.xp<0 or q.level<1:raise WorldError(f'Invalid quest {q.id}')
            for req in q.prereqs:
                if req not in self.qix:raise WorldError(f'Unknown prerequisite {req}')
            for obj in q.objectives:
                if obj.count<1 or obj.location not in self.nodes or obj.target not in self.mobs:
                    raise WorldError(f'Invalid objective {q.id} {obj.target}')
                if obj.kind not in ('kill','collect') or not 0<obj.drop_chance<=1:
                    raise WorldError(f'Invalid objective kind or drop chance for {q.id}')
        self.farms = tuple((f['at'],f['mob']) for f in doc.get('farms',[]))
        if any(loc not in self.nodes or mob not in self.mobs for loc,mob in self.farms):
            raise WorldError('Invalid grind farm')
        self._travel_cache = {}

    @classmethod
    def load(cls, name: str | Path):
        return cls(json.loads(Path(name).read_text(encoding='utf8')))

    def travel(self,a:str,b:str)->float:
        if a==b:return 0.0
        key=(a,b)
        if key in self._travel_cache:return self._travel_cache[key]
        heap=[(0.0,a)];best={a:0.0}
        while heap:
            t,node=heapq.heappop(heap)
            if t!=best[node]:continue
            if node==b:
                self._travel_cache[(a,b)]=self._travel_cache[(b,a)]=t
                return t
            for nxt,sec in self.graph[node]:
                alt=t+sec
                if alt<best.get(nxt,float('inf')):
                    best[nxt]=alt;heapq.heappush(heap,(alt,nxt))
        return float('inf')  # forbidden rather than an invented straight line


class State(NamedTuple):
    loc:str
    xp:int
    status:tuple[int,...] # 0 unaccepted, 1 active, 2 turned in
    progress:tuple[tuple[int,...],...]


@dataclass(frozen=True)
class Action:
    op:str
    quest:int|None = None
    obj:int|None = None
    at:str|None = None
    target:str|None = None

    def as_dict(self):
        return {'op':self.op,'quest':self.quest,'obj':self.obj,'at':self.at,'target':self.target}


@dataclass(frozen=True)
class StepCost:
    seconds:float
    travel:float=0
    combat:float=0
    interaction:float=0
    xp_gained:float=0


def initial(world:World, loc:str)->State:
    if loc not in world.nodes:raise WorldError('Unknown starting position')
    return State(loc,0,tuple(0 for q in world.quests),tuple(tuple(0 for o in q.objectives) for q in world.quests))


def level(world:World,xp:int)->int:
    lvl=1
    remaining=xp
    while lvl in world.level_xp and remaining>=world.level_xp[lvl]:
        remaining-=world.level_xp[lvl]
        lvl+=1
    return lvl


def _modify(tup,at,value):
    new=list(tup);new[at]=value;return tuple(new)


def action_candidates(world:World,state:State,race:str,cls:str,faction:str):
    # Only quests in range, with prerequisites; don't assume unavailable actions can be done.
    L=level(world,state.xp)
    for i,q in enumerate(world.quests):
        if q.faction and q.faction != faction:continue
        if q.classes and cls not in q.classes:continue
        if q.races and race not in q.races:continue
        if state.status[i]==0 and L>=q.level:
            if all(state.status[world.qix[req]]==2 for req in q.prereqs):
                if not any(state.status[world.qix[ex]]>0 for ex in q.excl if ex in world.qix):
                    if sum(s==1 for s in state.status)<world.max_open:
                        yield Action('accept',q.id,at=q.giver)
        if state.status[i]==1:
            if all(v>=o.count for v,o in zip(state.progress[i],q.objectives)):
                yield Action('turnin',q.id,at=q.turnin)
            else:
                for ix,obj in enumerate(q.objectives):
                    if state.progress[i][ix]<obj.count:
                        yield Action('work',q.id,ix,at=obj.location,target=obj.target)
    for at,mob in world.farms:
        ml = world.mobs[mob].level
        if ml is None or ml <= L + 3:
            yield Action('grind',at=at,target=mob)


def apply(world:World,state:State,act:Action,race:str,cls:str,faction:str):
    """Raises RouteError for an illegal action. Returns new state and cost."""
    if act not in set(action_candidates(world,state,race,cls,faction)):
        raise RouteError(f'Unavailable: {act}; level {level(world,state.xp)} at {state.loc}')
    dest=act.at
    move=world.travel(state.loc,dest)
    if move==float('inf'):raise RouteError(f'Unreachable {state.loc}->{dest}')
    status=state.status;progress=state.progress;xp=state.xp
    combat=0.0;interaction=0.0;xp_gain=0.0
    if act.op=='accept':
        status=_modify(status,world.qix[act.quest],1)
        interaction=6.0
    elif act.op=='turnin':
        i=world.qix[act.quest];q=world.quests[i]
        status=_modify(status,i,2)
        xp_gain=q.xp;interaction=6.0
    else:
        mob=world.mobs[act.target]
        if mob.level is not None and mob.level > level(world, xp) + 3:
            raise RouteError(f'Mob {act.target} is too high level')
        once=mob.kill_seconds if mob.kill_seconds is not None else mob.hp/world.dps
        if not (once>0):raise WorldError('Invalid mob time')
        if act.op=='grind':
            combat=once+mob.downtime
            xp_gain=mob.xp if mob.level is None or mob.level >= level(world, xp) - 7 else 0
        else:
            i=world.qix[act.quest];obj=world.quests[i].objectives[act.obj]
            kills=1/obj.drop_chance if obj.kind=='collect' else 1
            combat=(once+mob.downtime)*kills
            xp_gain=(mob.xp if mob.level is None or mob.level >= level(world, xp) - 7 else 0)*kills
            # Same kill advances all *already accepted* matching objectives.
            # Collection sharing and multi-item droprates require a richer model.
            progress2=list(progress)
            for j,q in enumerate(world.quests):
                if status[j]!=1:continue
                pr=list(progress2[j]);changed=False
                for k,o in enumerate(q.objectives):
                    if o.kind=='kill' and obj.kind=='kill' and o.target==obj.target and o.location==dest and pr[k]<o.count:
                        pr[k]=min(o.count,pr[k]+max(1,int(kills)));changed=True
                    elif j==i and k==act.obj and o.kind=='collect':
                        pr[k]+=1;changed=True
                if changed:progress2[j]=tuple(pr)
            progress=tuple(progress2)
    xp+=round(xp_gain)
    cost=StepCost(move+combat+interaction,travel=move,combat=combat,interaction=interaction,xp_gained=xp_gain)
    return State(dest,xp,status,progress),cost


def evaluate(world:World,route:list[dict]|list[Action],start:str,race:str,cls:str,faction:str,target_level:int):
    state=initial(world,start)
    totals={'seconds':0.0,'travel':0.0,'combat':0.0,'interaction':0.0,'xp':0.0}
    steps=[]
    for n,raw in enumerate(route,1):
        act=raw if isinstance(raw,Action) else Action(**raw)
        state,cost=apply(world,state,act,race,cls,faction)
        for k in ['seconds','travel','combat','interaction']:
            totals[k]+=getattr(cost,k)
        totals['xp']+=cost.xp_gained
        steps.append({'step':n,**act.as_dict(),'elapsed_seconds':round(totals['seconds'],3),'level':level(world,state.xp),'xp':state.xp})
    totals['level']=level(world,state.xp)
    totals['reached_target']=totals['level']>=target_level
    totals['completed_quests']=[q.id for i,q in enumerate(world.quests) if state.status[i]==2]
    totals['steps']=steps
    return totals


def optimize(world:World,start:str,race:str,cls:str,faction:str,target_level:int,max_states:int=50000):
    """Dijkstra, provably shortest in *this supplied deterministic graph*.

    NEVER claim optimality when max_states is reached or model data incomplete.
    """
    first=initial(world,start)
    if level(world,first.xp)>=target_level:
        return evaluate(world,[],start,race,cls,faction,target_level)|{'search_states':0,'optimal_in_model':True}
    pq=[(0.0,0,first)]; best={first:0.0}; parents={};tick=0;explored=0
    while pq and explored<max_states:
        score,_,state=heapq.heappop(pq)
        if score>best[state]+1e-8:continue
        explored+=1
        if level(world,state.xp)>=target_level:
            route=[];cur=state
            while cur!=first:
                prev,a=parents[cur];route.append(a);cur=prev
            route.reverse()
            result=evaluate(world,route,start,race,cls,faction,target_level)
            result['optimal_in_model']=True
            result['search_states']=explored
            return result
        for a in action_candidates(world,state,race,cls,faction):
            try:nxt,cost=apply(world,state,a,race,cls,faction)
            except RouteError:continue
            ns=score+cost.seconds
            if ns+1e-8<best.get(nxt,float('inf')):
                best[nxt]=ns;parents[nxt]=(state,a);tick+=1;heapq.heappush(pq,(ns,tick,nxt))
    return {'optimal_in_model':False,'search_states':explored,'reason':'state_limit' if pq else 'no_feasible_route',
            'target_level':target_level,'best_seconds':None}