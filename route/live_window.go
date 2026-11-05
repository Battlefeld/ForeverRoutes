package route

import (
	"math"
	"time"
)

// Live window: the route's next steps re-ordered from the character's real state. The shipped route is the best order
// for a character that plays it from level 1; a live character is often elsewhere (ahead in level, quests in another
// order, some skipped), and the route's order then sends it back and forth. ImproveLive re-orders the first quest
// steps (accept, work, hand in) up to the first other step (grind, dungeon, camp, ...) so that they take the least
// time from where the character is, and keeps the rest of the plan as it is.

// liveWindow is how many quest steps at the front of a live plan are re-ordered.
const liveWindow = 30

func reorderable(op string) bool { return op == "accept" || op == "work" || op == "turnin" }

// ImproveLive returns the plan with its first quest steps in the fastest order found within budget, and whether the
// order changed. The plan is replayed in full from s; an order that breaks a later step is not taken.
func (w *World) ImproveLive(s *State, plan *Result, budget time.Duration) (*Result, bool) {
	if plan == nil || len(plan.Steps) < 3 {
		return plan, false
	}
	acts := make([]Action, len(plan.Steps))
	for i, st := range plan.Steps {
		acts[i] = st.Action
	}
	end := 0
	for end < len(acts) && end < liveWindow && reorderable(acts[end].Op) {
		end++
	}
	if end < 3 {
		return plan, false
	}
	// cost of an order of the window: the time it takes from s, infinite when a step is not possible there
	cost := func(seq []Action) float64 {
		c := s.clone()
		for _, a := range seq {
			if w.Apply(c, a) != nil {
				return math.Inf(1)
			}
		}
		return c.Time
	}
	best := append([]Action(nil), acts[:end]...)
	bestCost := cost(best)
	if math.IsInf(bestCost, 1) {
		return plan, false
	}
	deadline := time.Now().Add(budget)
	trial := make([]Action, end)
	for improved := true; improved && time.Now().Before(deadline); {
		improved = false
		// move one step (and the run of steps at the same place after it) to another position
		for i := 0; i < end && time.Now().Before(deadline); i++ {
			for j := 0; j < end; j++ {
				if j == i {
					continue
				}
				moveTo(trial, best, i, j)
				if c := cost(trial); c < bestCost-1 {
					copy(best, trial)
					bestCost, improved = c, true
				}
			}
		}
	}
	full := append(append([]Action(nil), best...), acts[end:]...)
	res, err := w.replayFrom(s, full)
	if err != nil || res.Seconds >= plan.Seconds-1 {
		return plan, false
	}
	return res, true
}

// moveTo writes src with the element at i moved to position j into dst.
func moveTo(dst, src []Action, i, j int) {
	a := src[i]
	k := 0
	for n := range src {
		if n == i {
			continue
		}
		if k == j {
			dst[k] = a
			k++
		}
		dst[k] = src[n]
		k++
	}
	if k == j {
		dst[k] = a
	}
}
