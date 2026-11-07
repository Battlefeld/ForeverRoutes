package route

import (
	"fmt"
	"slices"
)

// Legs: a route the player lays out (zones and dungeon runs, each to a level), the planner fills in. In each zone leg
// it takes the quests of the zones named, nearest and best first (a hub before single quests: all taken at once,
// worked off, handed in together), and hand-ins in any capital on the way; a dungeon leg collects the run's quests
// (givers near the character or near the portal), runs it and hands them in. Where a leg's zones run out of quests
// before its level, it grinds there and says so: the player sees which part of the plan is thin.

// Leg is one part of a laid-out route: zones to quest in until a level, or a dungeon run.
type Leg struct {
	Zones   []string `json:"zones,omitempty"`
	Dungeon string   `json:"dungeon,omitempty"`
	Until   int      `json:"until,omitempty"` // zone leg: the level it goes to
}

// PlanLegs plays the legs from s; notes say where a leg did not go as laid out.
func (w *World) PlanLegs(s *State, legs []Leg, target int) (*Result, []string, error) {
	w.liveLocal, w.legPlan = true, true
	ss := &stageSearch{w: w, s0: s, po: PlanOptions{TargetLevel: target}.withDefaults()}
	cur := s.clone()
	var acts []Action
	var notes []string
	for li, leg := range legs {
		if leg.Dungeon != "" {
			d := w.dungeonByKey(leg.Dungeon)
			if d == nil {
				notes = append(notes, fmt.Sprintf("leg %d: dungeon %s not in the data (or not for this character)", li+1, leg.Dungeon))
				continue
			}
			m := w.dungeonMacro(cur, d)
			if m == nil {
				notes = append(notes, fmt.Sprintf("leg %d: %s not run (level %d, levels %d-%d, or run already)", li+1, d.Name, w.Level(cur), d.Min, d.Max))
				continue
			}
			acts = append(acts, m.actions...)
			cur = m.state
			continue
		}
		need := w.legQuests(leg.Zones)
		start, ground := cur.Time, 0.0
		for steps := 0; steps < 3000 && w.Level(cur) < min(leg.Until, target); steps++ {
			options := withoutPickups(w.options(cur, target))
			var best *macro
			bestRate := -1.0
			for i := range options {
				m := &options[i]
				if m.kind == "dungeon" || !w.inLeg(m, leg.Zones, need) {
					continue
				}
				r := ss.rateOf(cur, m, nearSpan)
				if m.kind == "hub" {
					r *= hubPreferred
				}
				if m.kind == "grind" {
					r *= max(w.Opt.GrindShare, 0.5) // quests while they are about as good: the player came for them
				}
				if r > bestRate {
					best, bestRate = m, r
				}
			}
			if best == nil {
				notes = append(notes, fmt.Sprintf("leg %d (%v): nothing left to do at level %d (until %d)", li+1, leg.Zones, w.Level(cur), leg.Until))
				break
			}
			if best.kind == "grind" {
				ground += best.state.Time - cur.Time
			}
			acts = append(acts, best.actions...)
			cur = best.state
		}
		if ground > 600 {
			notes = append(notes, fmt.Sprintf("leg %d (%v): %.0f of %.0f minutes grinding (the zones run short of quests)", li+1, leg.Zones, ground/60, (cur.Time-start)/60))
		}
	}
	res, err := w.replayFrom(s, w.takeAlong(s, acts))
	if err != nil {
		return nil, notes, err
	}
	return res, notes, nil
}

// legQuests: the quests given in the leg's zones and the quests before them (a chain's first quest is often given in a
// capital or the zone before: without it the whole chain stays closed).
func (w *World) legQuests(zones []string) map[int]bool {
	need := map[int]bool{}
	var add func(id int)
	add = func(id int) {
		q := w.ByID[id]
		if q == nil || need[id] {
			return
		}
		need[id] = true
		for _, p := range q.PreAll {
			add(p)
		}
		for _, p := range q.PreAny {
			add(p)
		}
	}
	for _, q := range w.Quests {
		if q.Dungeon == nil && slices.Contains(zones, w.zoneOf(q.StartPos)) {
			add(q.ID)
		}
	}
	return need
}

// inLeg: the block keeps to the leg's zones, or takes only quests of the leg (its zones' quests and the ones before
// them, wherever they are given), or only hands in or talks (an errand) in the leg's zones and the capitals.
func (w *World) inLeg(m *macro, zones []string, need map[int]bool) bool {
	in := func(z string) bool { return slices.Contains(zones, z) }
	all, errandOK, ours := true, errand(w, m), m.kind != "grind"
	for _, a := range m.actions {
		z := w.zoneOf(a.At)
		if !in(z) {
			all = false
		}
		if a.Quest != 0 && !need[a.Quest] {
			ours = false
		}
		if a.Quest == 0 && !in(z) {
			ours = false // grinding or travel of its own outside the zones
		}
	}
	if all || ours {
		return true
	}
	if !errandOK {
		return false
	}
	for _, a := range m.actions {
		if z := w.zoneOf(a.At); !in(z) {
			if _, capital := capitals[z]; !capital {
				return false
			}
		}
	}
	return true
}
