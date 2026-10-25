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
			if try(without(cur, func(e GuideEvent) bool { return e.Quest == q })) {
				improved = true
			}
		}
		for i := 0; i < len(cur) && time.Now().Before(deadline); i++ {
			if cur[i].Op == "level" || cur[i].Op == "dungeon" {
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

// EventsOf turns a planned route into guide steps, so planned routes and guides are improved alike.
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
		}
	}
	return out
}
