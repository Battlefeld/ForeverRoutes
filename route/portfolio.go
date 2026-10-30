package route

import (
	"math"
	"math/rand"
	"slices"
	"sort"
	"time"
)

type OptimizationSeed struct {
	Events       []GuideEvent     `json:"events"`
	Preparations *PreparationPlan `json:"preparations,omitempty"`
	WandDetour   float64          `json:"wandDetour,omitempty"`
}

func needsDungeon(q *Quest) bool {
	if q == nil || q.Dungeon == nil {
		return false
	}
	if q.StartInside || q.EndInside {
		return true
	}
	for _, o := range q.Objectives {
		if o.Inside {
			return true
		}
	}
	return false
}
func (w *World) filterDungeons(events []GuideEvent, allowed map[string]bool) []GuideEvent {
	return without(events, func(e GuideEvent) bool {
		if e.Op == "dungeon" {
			return !allowed[e.Dungeon]
		}
		q := w.ByID[e.Quest]
		return needsDungeon(q) && !allowed[q.Dungeon.Key]
	})
}
func actualDungeonKeys(r *GuideResult) []string {
	var keys []string
	for _, s := range r.Steps {
		if s.Op == "dungeon" {
			keys = append(keys, s.Dungeon)
		}
	}
	return keys
}
func (w *World) dungeonSubset(seed []GuideEvent, chosen map[string]bool, target int, deadline time.Time) []GuideEvent {
	events := w.filterDungeons(seed, chosen)
	for _, d := range w.Dungeons {
		if !chosen[d.Key] || slices.Contains(RunKeys(events), d.Key) {
			continue
		}
		base, err := w.walkFast(events, target, math.Inf(1))
		if err != nil {
			return nil
		}
		var best []GuideEvent
		secs := math.Inf(1)
		// All feasible level windows are eligible; sample positions evenly to keep this coarse phase bounded.
		var at []int
		for i := 0; i <= len(events); i++ {
			l := target
			if i < len(base.levels) {
				l = base.levels[i]
			}
			if l >= d.Min && l <= d.Max {
				at = append(at, i)
			}
		}
		if len(at) == 0 {
			at = []int{len(events)}
		}
		for k := 0; k < min(20, len(at)); k++ {
			if time.Now().After(deadline) {
				break
			}
			i := at[k*len(at)/min(20, len(at))]
			cand := slices.Insert(slices.Clone(events), i, GuideEvent{Op: "dungeon", Dungeon: d.Key})
			r, err := w.walkFast(cand, target, secs)
			if err != nil || r.Level < target || !slices.Contains(actualDungeonKeys(r), d.Key) || !w.tankFirst(r) {
				continue
			}
			if r.Seconds < secs {
				best, secs = cand, r.Seconds
			}
		}
		if best == nil {
			return nil
		}
		events = best
	}
	return events
}

// destroyRepair changes complete quest chains, including preparation journeys, instead of only moving individual rows.
func (w *World) destroyRepair(events []GuideEvent, rng *rand.Rand, target int) []GuideEvent {
	if len(events) < 2 {
		return nil
	}
	first := rng.Intn(len(events))
	last := min(len(events), first+5+rng.Intn(25))
	ids := map[int]bool{}
	for _, e := range events[first:last] {
		if e.Quest != 0 {
			ids[e.Quest] = true
		}
	}
	if len(ids) == 0 {
		return nil
	}
	out := without(events, func(e GuideEvent) bool { return ids[e.Quest] })
	var quests []*Quest
	for id := range ids {
		if q := w.ByID[id]; q != nil {
			quests = append(quests, q)
		}
	}
	sort.Slice(quests, func(i, j int) bool { return quests[i].ID < quests[j].ID })
	rng.Shuffle(len(quests), func(i, j int) { quests[i], quests[j] = quests[j], quests[i] })
	for _, q := range quests {
		// A ruin can replace an optional quest; required chains and selected goals are enforced by full replay.
		if rng.Intn(5) == 0 && !w.keepClassQuest(q.ID, target) {
			continue
		}
		i, _ := w.closestStep(out, q.StartPos)
		if len(out) == 0 {
			i = -1
		}
		if cand := w.insertQuestChain(out, q, i+1); cand != nil {
			out = cand
		}
	}
	return out
}

// moveVisit groups accept/turn-in actions of one hub while keeping objectives at their original places.
func (w *World) moveVisit(events []GuideEvent, rng *rand.Rand) []GuideEvent {
	if len(events) < 2 {
		return nil
	}
	i := rng.Intn(len(events))
	p, ok := w.eventPos(events[i])
	if !ok {
		return nil
	}
	var block, rest []GuideEvent
	original := 0
	for k, e := range events {
		at, placed := w.eventPos(e)
		if placed && (e.Op == "accept" || e.Op == "turnin") && distance(p, at) <= visitRadius {
			block = append(block, e)
			if k < i {
				original++
			}
		} else {
			rest = append(rest, e)
		}
	}
	if len(block) < 2 {
		return nil
	}
	var places []int
	for k, e := range rest {
		if at, ok := w.eventPos(e); ok && distance(p, at) <= visitRadius {
			places = append(places, k)
		}
	}
	if len(places) == 0 {
		places = []int{max(0, min(len(rest), i-original))}
	}
	at := places[rng.Intn(len(places))]
	return slices.Insert(rest, at, block...)
}

func (w *World) bagRests(events []GuideEvent, stacks, target int) []GuideEvent {
	out := without(events, func(e GuideEvent) bool { return e.Op == "sleep" })
	if stacks == 0 {
		return out
	}
	r, err := w.walkFast(out, target, math.Inf(1))
	if err != nil {
		return out
	}
	cost := w.Opt.SleepingBag.PlacementSeconds + float64(stacks)*w.Opt.SleepingBag.StackSeconds
	// Conservative estimate of the break-even point; full replay, not this estimate, decides whether a policy wins.
	payback := cost / (float64(stacks) * w.Opt.SleepingBag.PerStack)
	owned := false
	next := 0.0
	var result []GuideEvent
	for i, e := range out {
		at := r.eventTimes[i]
		if owned && at >= next && r.Seconds-at > payback {
			result = append(result, GuideEvent{Op: "sleep", Stacks: stacks})
			next = at + math.Max(w.Opt.SleepingBag.Duration, w.Opt.SleepingBag.Cooldown)
		}
		result = append(result, e)
		if e.Op == "turnin" && e.Quest == 79976 {
			owned = true
		}
	}
	return result
}
