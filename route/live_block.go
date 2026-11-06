package route

import (
	"math"
	"math/rand"
	"time"
)

// Live block: the next quests planned for the most XP per hour of this character, from where it stands.
//
// A block is BlockQuests quests, each taken, done and handed in (or grinding where that brings more): the planner's
// blocks (one quest with its neighbours, a hub, a hand-in round, grinding) are each worth what they bring beyond
// grinding for the same time, so a quest that brings less than grinding at that place is not done just because the
// character stands next to it. Several runs, the first greedy and the others picking among the best few blocks at
// each decision, are compared by the XP per hour of the block and a short stretch after it (a block that ends in a
// corner with nothing worth doing is worse). The best run is returned in full; the addon plays its first BlockQuests
// quests before it asks for the next block.

// BlockQuests is how many quests (taken, done and handed in) make a live block.
const BlockQuests = 10

// blockTail: seconds of play after the block that count for its rating (where the block leaves the character).
const blockTail = 900.0

// PlanBlock plans from s: the best of several runs within budget, nil when nothing can be planned.
func (w *World) PlanBlock(s *State, po PlanOptions, budget time.Duration) *Result {
	po = po.withDefaults()
	deadline := time.Now().Add(budget)
	rng := rand.New(rand.NewSource(po.Seed + 1))
	var best []Action
	bestRate := math.Inf(-1)
	for run := 0; run < 64 && (run == 0 || time.Now().Before(deadline)); run++ {
		cur := s.clone()
		var acts []Action
		var blockEnd *State
		handed := 0
		for step := 0; step < 400 && w.Level(cur) < po.TargetLevel; step++ {
			if blockEnd != nil && cur.Time-blockEnd.Time >= blockTail {
				break
			}
			options := w.options(cur, po.TargetLevel)
			if len(options) == 0 {
				break
			}
			pick := w.pick(cur, options, po, run, rng)
			acts = append(acts, pick.actions...)
			for _, a := range pick.actions {
				if a.Op == "turnin" {
					handed++
				}
			}
			cur = pick.state
			if blockEnd == nil && handed >= BlockQuests {
				blockEnd = cur
			}
		}
		if len(acts) == 0 || cur.Time <= s.Time {
			continue
		}
		rate := float64(cur.XP-s.XP) / (cur.Time - s.Time)
		if rate > bestRate {
			best, bestRate = acts, rate
		}
	}
	if best == nil {
		return nil
	}
	res, err := w.replayFrom(s, best)
	if err != nil {
		return nil
	}
	return res
}

// XPRate is the XP per second a plan brings from s.
func XPRate(s *State, r *Result) float64 {
	if r == nil || r.Seconds <= 0 {
		return 0
	}
	return float64(r.XP-s.XP) / r.Seconds
}
