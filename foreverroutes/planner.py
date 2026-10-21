"""Quest/grind route search. New independent implementation, not copied from guide addons.

This is a stochastic multi-start, greedy macro search: efficient for real 1-30
worlds with tens to hundreds of quest candidates, but NOT a proof of global
optimality. Every returned route is replayed with the same deterministic model.
"""
from __future__ import annotations

import math
import random
from dataclasses import dataclass
from .core import World, State, Action, RouteError, action_candidates, apply, initial, level, evaluate


@dataclass(frozen=True)
class Character:
    race: str = 'Orc'
    cls: str = 'Warlock'
    faction: str = 'Horde'
    start: str = 'Spawn'
    target: int = 30


@dataclass
class Candidate:
    actions: list[Action]
    state: State
    seconds: float
    xp: int
    source: str
    value: float = 0.0  # heuristic look-ahead XP, never added to character XP


def xp_until_next(world: World, xp: int) -> int:
    remaining = xp
    n = 1
    while n in world.level_xp and remaining >= world.level_xp[n]:
        remaining -= world.level_xp[n]
        n += 1
    return max(0, world.level_xp.get(n, 0) - remaining)


def _run_action(world: World, state: State, action: Action, c: Character):
    return apply(world, state, action, c.race, c.cls, c.faction)


def _quest_actions(world: World, state: State, quest_id: int, c: Character,
                   sibling_pickups: bool = True) -> Candidate | None:
    """Complete one quest, with other eligible pickups at its giver when useful.

    Greedy objective order is recomputed each move, minimizing walking+work.
    Neighboring unfinished quest objectives can share kill credit. Only a
    *local* order is selected here; multi-start search chooses the next quest.
    """
    if quest_id not in world.qix:
        return None
    i = world.qix[quest_id]
    quest = world.quests[i]
    if state.status[i] == 2:
        return None
    actions: list[Action] = []
    work_state = state
    total = 0.0
    first = Action('accept', quest_id, at=quest.giver)
    allowed = set(action_candidates(world, work_state, c.race, c.cls, c.faction))
    if work_state.status[i] == 0:
        if first not in allowed:
            return None
        work_state, cost = _run_action(world, work_state, first, c)
        total += cost.seconds
        actions.append(first)
        # Accept useful co-located, overlapping quests before doing shared work.
        if sibling_pickups:
            own = {(o.location, o.target) for o in quest.objectives}
            siblings = []
            for a in action_candidates(world, work_state, c.race, c.cls, c.faction):
                if a.op != 'accept' or a.at != quest.giver or a.quest == quest_id:
                    continue
                q = world.quests[world.qix[a.quest]]
                shared = own.intersection((o.location, o.target) for o in q.objectives)
                if shared:
                    siblings.append(a)
            for a in sorted(siblings, key=lambda a: a.quest):
                try:
                    work_state, cost = _run_action(world, work_state, a, c)
                    total += cost.seconds
                    actions.append(a)
                except RouteError:
                    pass
    # One work operation per kill or expected collection unit; bounded by
    # objectives counts. Avoid an entire-zone state-space expansion.
    while True:
        unfinished = []
        for k, obj in enumerate(quest.objectives):
            if work_state.progress[i][k] >= obj.count:
                continue
            a = Action('work', quest.id, k, at=obj.location, target=obj.target)
            if world.travel(work_state.loc, a.at) == float('inf'):
                return None
            cost = world.travel(work_state.loc, a.at)
            mob = world.mobs[obj.target]
            cost += ((mob.kill_seconds if mob.kill_seconds is not None else mob.hp/world.dps) + mob.downtime) / (obj.drop_chance if obj.kind == 'collect' else 1)
            unfinished.append((cost, k, a))
        if not unfinished:
            break
        _, _, a = min(unfinished)
        work_state, cost = _run_action(world, work_state, a, c)
        actions.append(a)
        total += cost.seconds
    a = Action('turnin', quest.id, at=quest.turnin)
    if a not in set(action_candidates(world, work_state, c.race, c.cls, c.faction)):
        return None
    work_state, cost = _run_action(world, work_state, a, c)
    total += cost.seconds
    actions.append(a)
    # Turn in simultaneously completed quests only when standing at same NPC.
    while True:
        extras = [a for a in action_candidates(world, work_state, c.race, c.cls, c.faction)
                  if a.op == 'turnin' and a.at == work_state.loc]
        if not extras:
            break
        for a in sorted(extras, key=lambda a: a.quest):
            work_state, cost = _run_action(world, work_state, a, c)
            total += cost.seconds
            actions.append(a)
    gained = work_state.xp-state.xp
    future = 0.0
    # A zero-XP breadcrumb may unlock a high-XP quest (including chains).
    if gained == 0:
        for future_q in world.quests:
            if quest_id in future_q.prereqs and future_q.xp > 0:
                future = max(future, future_q.xp*0.35)
    return Candidate(actions, work_state, total, gained, f'quest:{quest_id}', max(gained, future))


def _grind_actions(world: World, state: State, c: Character):
    """One grind candidate at each reachable farm; batch to next level, up to 80 kills."""
    missing = xp_until_next(world, state.xp)
    for at, mob in world.farms:
        if world.travel(state.loc, at) == float('inf'):
            continue
        m = world.mobs[mob]
        if m.xp <= 0 or (m.level is not None and (m.level > level(world, state.xp) + 3 or m.level < level(world,state.xp)-7)):
            continue
        hits = min(80, max(1, math.ceil(missing / m.xp)))
        work_state = state
        acts: list[Action] = []
        total = 0.0
        # Compact one-action-per-kill implementation, with future optimization
        # of event transport separate from model semantics.
        a = Action('grind', at=at, target=mob)
        for _ in range(hits):
            work_state, cost = _run_action(world, work_state, a, c)
            acts.append(a)
            total += cost.seconds
            if level(world, work_state.xp) >= c.target:
                break
        yield Candidate(acts, work_state, total, work_state.xp-state.xp, f'grind:{at}/{mob}', work_state.xp-state.xp)


def _options(world: World, state: State, c: Character, max_candidates: int = 160):
    # Active quests must be considered even if no new ones are available.
    ids = [a.quest for a in action_candidates(world, state, c.race, c.cls, c.faction)
           if a.op == 'accept' or a.op == 'turnin' or a.op == 'work']
    for i,q in enumerate(world.quests):
        if state.status[i] == 1:
            ids.append(q.id)
    seen = set()
    for quest_id in ids:
        if quest_id in seen:
            continue
        seen.add(quest_id)
        if len(seen) > max_candidates:
            break
        option = _quest_actions(world, state, quest_id, c)
        if option and option.value > 0 and option.seconds > 0:
            yield option
    yield from _grind_actions(world, state, c)


def search(world: World, character: Character, *, restarts: int = 12,
           seed: int = 1, max_decisions: int = 1800, pool: int = 7) -> dict:
    """Find the fastest tested quest/grind policy, never assert global optimality.

    Every restart explores a ranked randomized choice from the best local
    plans, so different quest sequences and grinding alternatives compete.
    Verified results replay all selected atomic actions through World.apply.
    """
    if character.target < 2 or character.target > 30:
        raise ValueError('Phase 1 supports target levels 2..30')
    if any(k not in world.level_xp for k in range(1, character.target)):
        raise ValueError('XP thresholds missing before target; cannot score result')
    if not world.farms:
        raise ValueError('No grind farms configured; a 1-30 planner needs a fallback')
    if restarts < 1 or pool < 1 or max_decisions < 1:
        raise ValueError('restarts, pool, max_decisions must be positive')
    rng = random.Random(seed)
    best = None
    summaries = []
    for run_id in range(restarts):
        state = initial(world, character.start)
        route: list[Action] = []
        decisions = 0
        seen_stall = 0
        # Policy temperatures differ by restart; deterministic from seed.
        select_n = 1 if run_id == 0 else min(pool, 2 + run_id % max(1,pool-1))
        while level(world, state.xp) < character.target and decisions < max_decisions:
            options = [p for p in _options(world, state, character) if p.value > 0 and p.seconds > 0]
            if not options:
                break
            # Cost per XP with nearest tasks favored only by *actual* modeled seconds.
            options.sort(key=lambda p: (p.seconds/p.value, p.seconds, p.source))
            pick = options[0] if run_id == 0 else options[rng.randrange(min(len(options), select_n))]
            route.extend(pick.actions)
            if pick.state.xp <= state.xp:
                seen_stall += 1
            else:
                seen_stall = 0
            state = pick.state
            decisions += 1
            if seen_stall >= 12:
                break
        reached = level(world, state.xp) >= character.target
        if not reached:
            summaries.append({'restart':run_id,'reached_target':False,'decisions':decisions,'level':level(world,state.xp)})
            continue
        result = evaluate(world, route, character.start, character.race,
                          character.cls, character.faction, character.target)
        if not result['reached_target']:
            raise AssertionError('Optimizer result did not replay to target')
        result['search_method'] = 'multistart_quest_grind_greedy'
        result['optimal_in_model'] = False
        result['restarts'] = restarts
        result['seed'] = seed
        result['decisions'] = decisions
        summaries.append({'restart':run_id,'reached_target':True,'seconds':result['seconds'], 'decisions':decisions})
        if best is None or result['seconds'] < best['seconds']:
            best = result
    if best is None:
        return {'reached_target':False,'optimal_in_model':False,'method':'multistart','runs':summaries,
                'reason':'no feasible route found: audit input data and reachable farms'}
    best['trials'] = summaries
    return best