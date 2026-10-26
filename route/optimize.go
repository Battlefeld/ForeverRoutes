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
	cost := func(evs []GuideEvent) (float64, *GuideResult) {
		r, err := w.FollowGuide(evs, target)
		if err != nil || r.Level < target || tank && !w.tankFirst(r) || runs && Runs(evs) == 0 {
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
	curC, best := cost(cur)
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
	deadline := time.Now().Add(o.Budget)
	for k := 0; time.Now().Before(deadline); k++ {
		wandBefore := w.wandDetour
		cand := w.edit(cur, rng, pool, target, wand)
		if cand == nil {
			continue
		}
		c, r := cost(cand)
		if v := k % hist; c <= curC || c <= late[v] {
			cur, curC = cand, c
			if c < bestC-1 {
				bestE, bestC, best, bestWand = cand, c, r, w.wandDetour
			}
			late[v] = curC
		} else {
			w.wandDetour = wandBefore
		}
	}
	w.wandDetour = bestWand
	return bestE, best, nil
}

// edit: one random change of the route, nil when the one drawn does not apply.
func (w *World) edit(cur []GuideEvent, rng *rand.Rand, pool []*Quest, target int, wand bool) []GuideEvent {
	n := len(cur)
	if n < 2 {
		return nil
	}
	span := func() int { return 1 + int(math.Abs(rng.NormFloat64())*30) } // mostly near, now and then far
	switch p := rng.Intn(100); {
	case p < 30: // one step elsewhere
		i := rng.Intn(n)
		j := i + span()*(1-2*rng.Intn(2))
		if j < 0 || j >= n || j == i {
			return nil
		}
		out := slices.Delete(slices.Clone(cur), i, i+1)
		return slices.Insert(out, j, cur[i])
	case p < 50: // a quest's steps together, earlier or later
		q := cur[rng.Intn(n)].Quest
		if q == 0 {
			return nil
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
		return rest
	case p < 65: // a stretch of the route elsewhere (a stay in a zone before or after another)
		l := 2 + rng.Intn(min(60, n-1))
		i := rng.Intn(n - l + 1)
		seg := slices.Clone(cur[i : i+l])
		rest := slices.Delete(slices.Clone(cur), i, i+l)
		j := rng.Intn(len(rest) + 1)
		if j == i {
			return nil
		}
		return slices.Insert(rest, j, seg...)
	case p < 75: // a quest left out
		q := cur[rng.Intn(n)].Quest
		if q == 0 || w.keepClassQuest(q, target) {
			return nil
		}
		return without(cur, func(e GuideEvent) bool { return e.Quest == q })
	case p < 88: // a quest put in where the route passes its giver
		if len(pool) == 0 {
			return nil
		}
		q := pool[rng.Intn(len(pool))]
		if slices.ContainsFunc(cur, func(e GuideEvent) bool { return e.Quest == q.ID }) {
			return nil
		}
		i, d := w.closestStep(cur, q.StartPos)
		if d > optimizeReach {
			return nil
		}
		block := []GuideEvent{{Op: "accept", Quest: q.ID}}
		if q.Dungeon == nil {
			block = append(block, GuideEvent{Op: "complete", Quest: q.ID}, GuideEvent{Op: "turnin", Quest: q.ID})
		}
		return slices.Insert(slices.Clone(cur), i+1, block...)
	case p < 94: // a dungeon run put in or left out
		if !w.Opt.Dungeons || len(w.Dungeons) == 0 {
			return nil
		}
		d := w.Dungeons[rng.Intn(len(w.Dungeons))]
		if at := slices.IndexFunc(cur, func(e GuideEvent) bool { return e.Op == "dungeon" && e.Dungeon == d.Key }); at >= 0 {
			return slices.Delete(slices.Clone(cur), at, at+1)
		}
		return slices.Insert(slices.Clone(cur), rng.Intn(n+1), GuideEvent{Op: "dungeon", Dungeon: d.Key})
	case p < 97: // a level step left out
		i := rng.Intn(n)
		if cur[i].Op != "level" {
			return nil
		}
		return slices.Delete(slices.Clone(cur), i, i+1)
	default: // the detour for the wands
		if !wand {
			return nil
		}
		w.wandDetour = WandDetours[rng.Intn(len(WandDetours))]
		return slices.Clone(cur)
	}
}
