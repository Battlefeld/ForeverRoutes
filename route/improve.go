package route

import (
	"math"
	"time"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Improve makes a guide faster under the model, the way Polestar improves its shipped orders: it starts from the
// guide's own steps and keeps only local edits that reach the target level sooner when the whole guide is walked
// again (FollowGuide): leaving out a quest or a level step, or putting in a quest the guide passes by (taken, done
// and turned in where the guide comes closest to its giver). So the result is never slower than the guide it starts
// from. The budget bounds the time spent searching.
func (w *World) Improve(events []GuideEvent, target int, budget time.Duration) ([]GuideEvent, *GuideResult, error) {
	events = w.withClassQuests(events)
	best, err := w.FollowGuide(events, target)
	if err != nil {
		return nil, nil, err
	}
	cur := append([]GuideEvent(nil), events...)
	deadline := time.Now().Add(budget)
	try := func(cand []GuideEvent) bool {
		r, err := w.FollowGuide(cand, target)
		if err != nil || r.Level < target || r.Seconds >= best.Seconds-1 {
			return false
		}
		cur, best = cand, r
		return true
	}
	for pass := 0; time.Now().Before(deadline); pass++ {
		improved := false
		// leave out: each quest the guide does, each level step
		for _, q := range questsOf(cur) {
			if time.Now().After(deadline) {
				break
			}
			if w.keepClassQuest(q) {
				continue // the first class quests teach abilities: never left out for time
			}
			if try(without(cur, func(e GuideEvent) bool { return e.Quest == q })) {
				improved = true
			}
		}
		for i := 0; i < len(cur) && time.Now().Before(deadline); i++ {
			if cur[i].Op == "level" || (cur[i].Op == "dungeon" && Runs(cur) > 1) {
				// a route with dungeons keeps at least one run: whoever asks for dungeons runs the one that costs least
				if try(append(append([]GuideEvent(nil), cur[:i]...), cur[i+1:]...)) {
					improved = true
					i--
				}
			}
		}
		// put in: usable quests the guide does not do, at the step whose place is closest to the giver
		in := map[int]bool{}
		for _, q := range questsOf(cur) {
			in[q] = true
		}
		for _, q := range w.Quests {
			if time.Now().After(deadline) {
				break
			}
			if in[q.ID] || q.MinLevel > target || q.Level > target+3 || q.Repeatable {
				continue
			}
			i, d := w.closestStep(cur, q.StartPos)
			if d > 250 {
				continue
			}
			block := []GuideEvent{{Op: "accept", Quest: q.ID}, {Op: "complete", Quest: q.ID}, {Op: "turnin", Quest: q.ID}}
			cand := append(append(append([]GuideEvent(nil), cur[:i+1]...), block...), cur[i+1:]...)
			if try(cand) {
				improved = true
				in[q.ID] = true
			}
		}
		if !improved {
			break
		}
	}
	return cur, best, nil
}

// WithDungeons puts dungeon runs into a route (a guide has none): each dungeon at the one of twenty places spread over
// the route that brings it fastest to the target, when one does. When none does, the run that costs least goes in:
// a route with dungeons runs at least one. A run takes the dungeon's quests on the way and hands them in after, as
// the planner's dungeon block.
func (w *World) WithDungeons(events []GuideEvent, target int) []GuideEvent {
	best, err := w.FollowGuide(events, target)
	if err != nil {
		return events
	}
	cur := events
	var least []GuideEvent
	leastT := math.Inf(1)
	for _, d := range w.Dungeons {
		have := false
		for _, e := range cur {
			have = have || (e.Op == "dungeon" && e.Dungeon == d.Key)
		}
		if have || d.Min > target {
			continue
		}
		var pick []GuideEvent
		for k := 1; k <= 20; k++ {
			i := len(cur) * k / 21
			cand := append(append(append([]GuideEvent(nil), cur[:i]...), GuideEvent{Op: "dungeon", Dungeon: d.Key}), cur[i:]...)
			r, err := w.FollowGuide(cand, target)
			if err != nil || r.Level < target {
				continue
			}
			if r.Seconds < best.Seconds-1 {
				best, pick = r, cand
			}
			if r.Seconds < leastT {
				least, leastT = cand, r.Seconds
			}
		}
		if pick != nil {
			cur = pick
		}
	}
	if Runs(cur) == 0 && least != nil {
		return least // no run made it faster: until then cur is the route given, least one run put into it
	}
	return cur
}

// Runs counts the dungeon runs of a route.
func Runs(events []GuideEvent) int {
	n := 0
	for _, e := range events {
		if e.Op == "dungeon" {
			n++
		}
	}
	return n
}

// keepClassQuest: a class quest a character takes by level 10.
func (w *World) keepClassQuest(id int) bool {
	q := w.ByID[id]
	return q != nil && q.Classes != 0 && q.MinLevel <= 10
}

// withClassQuests puts in the class quests of the start (given by level 6 near the start) the route lacks, each at the
// step whose place is closest to its giver, chain after chain in order.
func (w *World) withClassQuests(events []GuideEvent) []GuideEvent {
	in := map[int]bool{}
	for _, q := range questsOf(events) {
		in[q] = true
	}
	out := append([]GuideEvent(nil), events...)
	for _, q := range w.Quests {
		if in[q.ID] || q.Classes == 0 || q.MinLevel > 6 || distance(q.StartPos, w.Start) > startArea {
			continue
		}
		i, _ := w.closestStep(out, q.StartPos)
		// a follow-up goes after its prerequisite's turn-in
		for _, p := range append(append([]int(nil), q.PreAll...), q.PreAny...) {
			for j, e := range out {
				if e.Quest == p && e.Op == "turnin" && j > i {
					i = j
				}
			}
		}
		block := []GuideEvent{{Op: "accept", Quest: q.ID}, {Op: "complete", Quest: q.ID}, {Op: "turnin", Quest: q.ID}}
		out = append(append(append([]GuideEvent(nil), out[:i+1]...), block...), out[i+1:]...)
		in[q.ID] = true
	}
	return out
}

// questsOf lists the quests a guide names, in first-seen order.
func questsOf(events []GuideEvent) []int {
	seen := map[int]bool{}
	var out []int
	for _, e := range events {
		if e.Quest != 0 && !seen[e.Quest] {
			seen[e.Quest] = true
			out = append(out, e.Quest)
		}
	}
	return out
}

func without(events []GuideEvent, drop func(GuideEvent) bool) []GuideEvent {
	out := make([]GuideEvent, 0, len(events))
	for _, e := range events {
		if !drop(e) {
			out = append(out, e)
		}
	}
	return out
}

// closestStep is the index of the guide step whose place (the giver or ender of its quest) is closest to p.
func (w *World) closestStep(events []GuideEvent, p dataset.Point) (int, float64) {
	bi, bd := 0, math.Inf(1)
	for i, e := range events {
		q := w.ByID[e.Quest]
		if q == nil {
			continue
		}
		at := q.StartPos
		if e.Op == "turnin" {
			at = q.EndPos
		}
		if d := distance(at, p); d < bd {
			bi, bd = i, d
		}
	}
	return bi, bd
}

// EventsOf turns a planned route into guide steps, so planned routes and guides are improved alike. The planner's
// grinding becomes a level step (grind to the level it reached there): left out, the follower would do the next
// quests a level lower and grind at the end.
func EventsOf(steps []Step) []GuideEvent {
	var out []GuideEvent
	done := map[int]bool{}
	for _, s := range steps {
		switch s.Op {
		case "accept", "turnin":
			out = append(out, GuideEvent{Op: s.Op, Quest: s.Quest})
		case "work":
			if !done[s.Quest] {
				done[s.Quest] = true
				out = append(out, GuideEvent{Op: "complete", Quest: s.Quest})
			}
		case "dungeon":
			out = append(out, GuideEvent{Op: "dungeon", Dungeon: s.Dungeon})
		case "grind":
			if n := len(out); n > 0 && out[n-1].Op == "level" {
				out[n-1].Level = s.Level
			} else {
				out = append(out, GuideEvent{Op: "level", Level: s.Level})
			}
		}
	}
	return out
}
