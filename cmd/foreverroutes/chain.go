package main

import (
	"fmt"
	"os"
	"sort"

	"github.com/Battlefeld/ForeverRoutes/route"
)

// A chain of zones: from the character's state, every zone's block (its quests, worked off until nothing of it is
// left to do at that level) is planned, and the one with the most XP per hour, the way there included, is taken;
// grinding a level is one more candidate. Then again from where that block ends, until the target level. A zone can
// come back later, when its higher quests open. The route reads like a guide: one zone at a time.

// chainLink is one block of a chain.
type chainLink struct {
	Zone     string  `json:"zone"` // "" for grinding
	Start    float64 `json:"startMinute"`
	End      float64 `json:"endMinute"`
	FromLvl  int     `json:"fromLevel"`
	ToLvl    int     `json:"toLevel"`
	XP       int     `json:"xp"`
	TurnIns  int     `json:"turnIns"`
	XPPerHr  float64 `json:"xpPerHour"`
	Rejected string  `json:"others,omitempty"` // the next best candidates, for the record
}

// zoneQuests: the quests of one zone a chain block may take (taken and handed in there, not a dungeon's, not elite,
// no objective of an impossible count, not left out by the caller), with what is in the log and handed in there.
func zoneQuests(w *route.World, s *route.State, zone string, target int, skipped map[int]bool, inLog map[int]bool) map[int]bool {
	out := map[int]bool{}
	for _, q := range w.Quests {
		if skipped[q.ID] || hugeObjective(q) || w.ZoneName(q.EndPos) != zone {
			continue
		}
		if inLog[q.ID] || q.MinLevel < target && q.Dungeon == nil && q.Type == "" && !q.Repeatable && w.ZoneName(q.StartPos) == zone {
			out[q.ID] = true
		}
	}
	return out
}

// chainRoute plans the chain from s to target over the zones.
func chainRoute(w *route.World, s *route.State, zones []string, target int, prefer map[int][]int, skipped map[int]bool) ([]route.Step, []chainLink, error) {
	var steps []route.Step
	var links []chainLink
	t0 := s.Time
	for round := 0; round < 60 && w.Level(s) < target; round++ {
		type cand struct {
			zone string
			res  *route.Result
			gain int
			dt   float64
			ti   int
		}
		var cands []cand
		inLog := map[int]bool{}
		for _, q := range w.Quests {
			if s.Status[q.Index] == 1 { // taken, not handed in (route: notTaken 0, active 1, done 2)
				inLog[q.ID] = true
			}
		}
		for _, z := range zones {
			set := zoneQuests(w, s, z, target, skipped, inLog)
			if len(set) == 0 {
				continue
			}
			o := route.SweepOptions{Target: target, Near: 120, Out: 900, Reach: 500, Zones: true, StopIdle: true, Prefer: prefer, From: s}
			res, _, err := w.Sweep(set, o)
			if err != nil {
				continue
			}
			ti := 0
			for _, st := range res.Steps {
				if st.Op == "turnin" {
					ti++
				}
			}
			dt := res.Seconds - s.Time
			if ti < 2 || dt < 300 {
				continue // a hand-in or two on the way is no block of its own
			}
			cands = append(cands, cand{z, res, res.XP - s.XP, dt, ti})
		}
		// grinding one level where the character is (or near) is always a candidate
		if g, _, err := w.Sweep(map[int]bool{}, route.SweepOptions{Target: w.Level(s) + 1, From: s}); err == nil && g.Seconds > s.Time {
			cands = append(cands, cand{"", g, g.XP - s.XP, g.Seconds - s.Time, 0})
		}
		if len(cands) == 0 {
			return steps, links, fmt.Errorf("level %d: nothing to do and no grinding", w.Level(s))
		}
		sort.SliceStable(cands, func(i, j int) bool {
			return float64(cands[i].gain)/cands[i].dt > float64(cands[j].gain)/cands[j].dt
		})
		c := cands[0]
		others := ""
		for i := 1; i < len(cands) && i < 4; i++ {
			name := cands[i].zone
			if name == "" {
				name = "grind"
			}
			others += fmt.Sprintf("%s %.0f XP/h; ", name, float64(cands[i].gain)/cands[i].dt*3600)
		}
		next, err := w.StateAfterFrom(s, c.res.Steps)
		if err != nil {
			return steps, links, err
		}
		links = append(links, chainLink{Zone: c.zone, Start: (s.Time - t0) / 60, End: (next.Time - t0) / 60, FromLvl: w.Level(s), ToLvl: w.Level(next),
			XP: c.gain, TurnIns: c.ti, XPPerHr: float64(c.gain) / c.dt * 3600, Rejected: others})
		name := c.zone
		if name == "" {
			name = "grind"
		}
		fmt.Fprintf(os.Stderr, "chain %2d: %-22s level %d->%d, %3.0f min, %2d hand-ins, %6.0f XP/h (%s)\n", round+1, name, w.Level(s), w.Level(next), c.dt/60, c.ti, float64(c.gain)/c.dt*3600, others)
		steps = append(steps, c.res.Steps...)
		s = next
	}
	return steps, links, nil
}
