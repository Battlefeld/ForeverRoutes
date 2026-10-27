package route

import (
	"math"
	"math/rand"
	"slices"
	"time"
)

// OptimizeOptions: what Optimize keeps and how long it searches.
type OptimizeOptions struct {
	Target  int
	Budget  time.Duration
	Seed    int64
	Keep    []int // quests the route must hand in: the goal of an option switched on (the sleeping bag's)
	History int   // late acceptance: an edit is kept when no slower than the route this many edits ago (400)
	Stats   *OptimizeStats
}

// OptimizeStats: how the search went, per kind of edit.
type OptimizeStats struct {
	Walks, Kept, Better [9]int // by kind: step, quest, stretch, drop, add, dungeon, level, wand, visit
}

// optimizeReach: a quest put in goes where the route comes this close to its giver (yards).
const optimizeReach = 400.0

// Optimize searches the route as a timeline: every step can move, whole quests and stretches of the route (a stay in
// a zone) move, quests are left out or put in where the route passes their givers, dungeon runs are put in, moved
// and left out, and a caster's detour for its wands changes, so the quests, the dungeon quests and runs, the options
// and the wands find the order that reaches the target soonest. Camps and rests follow from the route (the walk makes
// them when the buff runs out). Random edits with late acceptance: a slower route is kept for a while when it is not
// slower than the route some edits ago, so the search gets past orders where every single edit costs. The required
// class chains, the quests to keep, the tank form before the first run and, for a route with dungeons, one run stay.
func (w *World) Optimize(events []GuideEvent, o OptimizeOptions) ([]GuideEvent, *GuideResult, error) {
	target := o.Target
	events = w.withClassQuests(events, target)
	deadline := time.Now().Add(o.Budget)
	events = w.improveDungeonChains(events, target, deadline)
	start, err := w.FollowGuide(events, target)
	if err != nil {
		return nil, nil, err
	}
	var goals []int
	for _, c := range w.ClassChains(target) {
		if c.Tier == "required" && handsIn(start, c.Goal.ID) {
			goals = append(goals, c.Goal.ID)
		}
	}
	goals = append(goals, o.Keep...)
	tank := w.tankFirst(start)
	runs := Runs(events) > 0 && w.Opt.Dungeons
	cost := func(evs []GuideEvent, limit float64) (float64, *GuideResult) {
		r, err := w.walkFast(evs, target, limit+1)
		if err != nil || r.Level < target || tank && !w.tankFirst(r) || runs && Runs(evs) == 0 || w.runAbove(evs, target) {
			return math.Inf(1), nil
		}
		for _, g := range goals {
			if !handsIn(r, g) {
				return math.Inf(1), nil
			}
		}
		return r.Seconds, r
	}
	var pool []*Quest // quests that may be put in
	for _, q := range w.Quests {
		if q.MinLevel <= target && q.Level <= target+3 && !q.Repeatable {
			pool = append(pool, q)
		}
	}
	rng := rand.New(rand.NewSource(o.Seed))
	hist := o.History
	if hist <= 0 {
		hist = 400
	}
	cur := events
	curC, best := cost(cur, math.Inf(1))
	if best == nil {
		best = start // the route given breaks a rule already: searched from it anyway
		curC = start.Seconds
	}
	bestE, bestC := cur, curC
	bestWand := w.wandDetour
	late := make([]float64, hist)
	for i := range late {
		late[i] = curC
	}
	wand := wandClasses[w.Char.Class]
	for k := 0; time.Now().Before(deadline); k++ {
		wandBefore := w.wandDetour
		cand, kind := w.edit(cur, rng, pool, target, wand)
		if cand == nil {
			continue
		}
		v := k % hist
		c, r := cost(cand, math.Max(curC, late[v])) // a walk slower than what is kept stops early
		if o.Stats != nil {
			o.Stats.Walks[kind]++
		}
		if c <= curC || c <= late[v] {
			if o.Stats != nil {
				o.Stats.Kept[kind]++
			}
			cur, curC = cand, c
			if c < bestC-1 {
				if o.Stats != nil {
					o.Stats.Better[kind]++
				}
				bestE, bestC, best, bestWand = cand, c, r, w.wandDetour
			}
			late[v] = curC
		} else {
			w.wandDetour = wandBefore
		}
	}
	w.wandDetour = bestWand
	// what the search left behind and the walk does not need (a step it skips, a run after the target): out
	for i := len(bestE) - 1; i >= 0; i-- {
		cand := slices.Delete(slices.Clone(bestE), i, i+1)
		if c, r := cost(cand, bestC); c <= bestC {
			bestE, bestC, best = cand, c, r
		}
	}
	full, err := w.FollowGuide(bestE, target) // the time line, for the report
	if err != nil {
		return nil, nil, err
	}
	return bestE, full, nil
}

// edit: one random change of the route and its kind, nil when the one drawn does not apply.
func (w *World) edit(cur []GuideEvent, rng *rand.Rand, pool []*Quest, target int, wand bool) (out []GuideEvent, kind int) {
	n := len(cur)
	if n < 2 {
		return nil, kind
	}
	span := func() int { return 1 + int(math.Abs(rng.NormFloat64())*30) } // mostly near, now and then far
	switch p := rng.Intn(100); {
	case p < 20: // a step to another visit of its place: one trip to a town instead of two (a turn-in waits for the
		// quest there that can be taken later)
		kind = 8
		i := rng.Intn(n)
		pi, ok := w.eventPos(cur[i])
		if !ok {
			return nil, kind
		}
		var js []int
		for j, e := range cur {
			if pj, ok := w.eventPos(e); ok && (j < i-3 || j > i+3) && distance(pi, pj) <= visitRadius {
				js = append(js, j)
			}
		}
		if len(js) == 0 {
			return nil, kind
		}
		j := js[rng.Intn(len(js))]
		out := slices.Delete(slices.Clone(cur), i, i+1)
		if j > i {
			j-- // after the deletion
		}
		if i > j {
			j++ // after the visit's step
		}
		return slices.Insert(out, j, cur[i]), kind
	case p < 35: // one step elsewhere
		kind = 0
		i := rng.Intn(n)
		j := i + span()*(1-2*rng.Intn(2))
		if j < 0 || j >= n || j == i {
			return nil, kind
		}
		out := slices.Delete(slices.Clone(cur), i, i+1)
		return slices.Insert(out, j, cur[i]), kind
	case p < 50: // a quest's steps together, earlier or later
		kind = 1
		q := cur[rng.Intn(n)].Quest
		if q == 0 {
			return nil, kind
		}
		d := span() * (1 - 2*rng.Intn(2))
		var rest, steps []GuideEvent
		var at []int
		for i, e := range cur {
			if e.Quest == q && e.Op != "level" {
				steps, at = append(steps, e), append(at, i)
			} else {
				rest = append(rest, e)
			}
		}
		base := len(rest)
		for k, e := range steps {
			// its place among the others, moved; the quest's steps stay in order (at only grows)
			j := min(max(at[k]-k+d, 0), base) + k
			rest = slices.Insert(rest, min(j, len(rest)), e)
		}
		return rest, kind
	case p < 65: // a stretch of the route elsewhere (a stay in a zone before or after another)
		kind = 2
		l := 2 + rng.Intn(min(60, n-1))
		i := rng.Intn(n - l + 1)
		seg := slices.Clone(cur[i : i+l])
		rest := slices.Delete(slices.Clone(cur), i, i+l)
		j := rng.Intn(len(rest) + 1)
		if j == i {
			return nil, kind
		}
		return slices.Insert(rest, j, seg...), kind
	case p < 75: // a quest left out
		kind = 3
		q := cur[rng.Intn(n)].Quest
		if q == 0 || w.keepClassQuest(q, target) {
			return nil, kind
		}
		return without(cur, func(e GuideEvent) bool { return e.Quest == q }), kind
	case p < 88: // a quest put in where the route passes its giver
		kind = 4
		if len(pool) == 0 {
			return nil, kind
		}
		q := pool[rng.Intn(len(pool))]
		if slices.ContainsFunc(cur, func(e GuideEvent) bool { return e.Quest == q.ID }) {
			return nil, kind
		}
		i, d := w.closestStep(cur, q.StartPos)
		if d > optimizeReach {
			return nil, kind
		}
		return w.insertQuestChain(cur, q, i+1), kind
	case p < 94: // a dungeon run put in or left out
		kind = 5
		if !w.Opt.Dungeons || len(w.Dungeons) == 0 {
			return nil, kind
		}
		d := w.Dungeons[rng.Intn(len(w.Dungeons))]
		if at := slices.IndexFunc(cur, func(e GuideEvent) bool { return e.Op == "dungeon" && e.Dungeon == d.Key }); at >= 0 {
			return slices.Delete(slices.Clone(cur), at, at+1), kind
		}
		return slices.Insert(slices.Clone(cur), rng.Intn(n+1), GuideEvent{Op: "dungeon", Dungeon: d.Key}), kind
	case p < 97: // a level step left out
		kind = 6
		i := rng.Intn(n)
		if cur[i].Op != "level" {
			return nil, kind
		}
		return slices.Delete(slices.Clone(cur), i, i+1), kind
	default: // the detour for the wands
		kind = 7
		if !wand {
			return nil, kind
		}
		w.wandDetour = WandDetours[rng.Intn(len(WandDetours))]
		return slices.Clone(cur), kind
	}
}

// runAbove: a route runs a dungeon that is for characters above the target level.
func (w *World) runAbove(events []GuideEvent, target int) bool {
	for _, e := range events {
		if d := w.dungeonOf[e.Dungeon]; e.Op == "dungeon" && d != nil && d.Min > target {
			return true
		}
	}
	return false
}
