package route

import (
	"maps"
	"slices"
	"sort"
	"time"
)

// questChain makes prerequisites and their reward one candidate. Otherwise a useful dungeon chain can never be
// discovered: its setup alone costs time and is rejected before the dungeon quest can repay it.
func (w *World) questChain(q *Quest, events []GuideEvent) []*Quest {
	handed := map[int]bool{}
	for _, e := range events {
		if e.Op == "turnin" {
			handed[e.Quest] = true
		}
	}
	seen, visiting := map[int]bool{}, map[int]bool{}
	var chain []*Quest
	var add func(*Quest) bool
	add = func(cur *Quest) bool {
		if cur == nil || visiting[cur.ID] {
			return false
		}
		if seen[cur.ID] {
			return true
		}
		visiting[cur.ID] = true
		defer delete(visiting, cur.ID)
		for _, id := range cur.PreAll {
			if !add(w.ByID[id]) {
				return false
			}
		}
		if len(cur.PreAny) > 0 {
			var candidates []*Quest
			for _, id := range cur.PreAny {
				p := w.ByID[id]
				if p == nil {
					continue
				}
				candidates = append(candidates, p)
			}
			sort.SliceStable(candidates, func(i, j int) bool {
				a, b := candidates[i], candidates[j]
				if handed[a.ID] != handed[b.ID] {
					return handed[a.ID]
				}
				return distance(a.EndPos, cur.StartPos) < distance(b.EndPos, cur.StartPos)
			})
			found := false
			for _, p := range candidates {
				previous, n := maps.Clone(seen), len(chain)
				if add(p) {
					found = true
					break
				}
				seen, chain = previous, chain[:n]
			}
			if !found {
				return false
			}
		}
		visiting[cur.ID] = false
		seen[cur.ID] = true
		chain = append(chain, cur)
		return true
	}
	if !add(q) {
		return nil
	}
	return chain
}

func (w *World) insertQuestChain(events []GuideEvent, q *Quest, at int) []GuideEvent {
	chain := w.questChain(q, events)
	if len(chain) == 0 {
		return nil
	}
	ids := map[int]bool{}
	for _, p := range chain {
		ids[p.ID] = true
	}
	var block []GuideEvent
	for _, p := range chain {
		if p.StartInside {
			continue
		}
		block = append(block, GuideEvent{Op: "accept", Quest: p.ID})
		if p.Dungeon == nil {
			block = append(block, GuideEvent{Op: "complete", Quest: p.ID}, GuideEvent{Op: "turnin", Quest: p.ID})
		}
	}
	// Steps of this same chain move as a whole. All unrelated route steps retain their order.
	before := without(events[:at], func(e GuideEvent) bool { return ids[e.Quest] })
	after := without(events[at:], func(e GuideEvent) bool { return ids[e.Quest] })
	out := append(append(slices.Clone(before), block...), after...)
	if q.Dungeon != nil && !slices.ContainsFunc(out, func(e GuideEvent) bool { return e.Op == "dungeon" && e.Dungeon == q.Dungeon.Key }) {
		out = slices.Insert(out, len(before)+len(block), GuideEvent{Op: "dungeon", Dungeon: q.Dungeon.Key})
	}
	return out
}

// improveDungeonChains deliberately tests every unfinished external quest of a dungeon already in the route,
// including its whole prerequisite chain, before random timeline edits. It only keeps a verified time saving.
func (w *World) improveDungeonChains(events []GuideEvent, target int, deadline time.Time) []GuideEvent {
	best, err := w.FollowGuide(events, target)
	if err != nil {
		return events
	}
	for _, d := range w.Dungeons {
		run := slices.IndexFunc(events, func(e GuideEvent) bool { return e.Op == "dungeon" && e.Dungeon == d.Key })
		if run < 0 {
			continue
		}
		for _, q := range d.Quests {
			if time.Now().After(deadline) {
				return events
			}
			if q.StartInside || handsIn(best, q.ID) {
				continue
			}
			type visit struct {
				at       int
				distance float64
			}
			var visits []visit
			for i := 0; i <= run; i++ {
				if i < len(best.levels) && best.levels[i]+2 < q.MinLevel {
					continue
				}
				if i == run {
					visits = append(visits, visit{i, 0})
					continue
				}
				if p, ok := w.eventPos(events[i]); ok {
					visits = append(visits, visit{i, distance(p, q.StartPos)})
				}
			}
			sort.SliceStable(visits, func(i, j int) bool { return visits[i].distance < visits[j].distance })
			var pick []GuideEvent
			pickR := best
			for _, v := range visits[:min(12, len(visits))] {
				if time.Now().After(deadline) {
					return events
				}
				cand := w.insertQuestChain(events, q, v.at)
				if cand == nil {
					continue
				}
				r, err := w.FollowGuide(cand, target)
				if err != nil || r.Level < target || !handsIn(r, q.ID) || r.Seconds >= pickR.Seconds-1 {
					continue
				}
				valid := true
				for _, c := range w.ClassChains(target) {
					if c.Tier == "required" && handsIn(best, c.Goal.ID) && !handsIn(r, c.Goal.ID) {
						valid = false
						break
					}
				}
				if valid && w.tankFirst(r) {
					pick, pickR = cand, r
				}
			}
			if pick != nil {
				events, best = pick, pickR
				run = slices.IndexFunc(events, func(e GuideEvent) bool { return e.Op == "dungeon" && e.Dungeon == d.Key })
			}
		}
	}
	return events
}
