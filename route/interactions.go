package route

import "math"

type interactionKey struct{ Quest, Objective, Site int }

// nonCombatWork prices a conversation, item use, healing/emote action, or duel. Creature credit never grants kill XP.
// Repeated interaction goals need distinct targets: a single peon/guard cannot satisfy five uses in one visit.
func (w *World) nonCombatWork(s *State, q *Quest, oi int, src Source) (units, seconds float64, ok bool) {
	o := q.Objectives[oi]
	left := math.Max(0, float64(o.Count)-s.Prog[q.Index][oi])
	if left <= 0 {
		return 0, 0, false
	}
	units = left
	if src.Duel {
		return units, w.mobTime(s, src.NPC, src.Site) * units, true
	}
	if o.Kind == "talk" || o.Kind == "interact" {
		available := float64(max(1, src.Site.Spawns)) - s.Interacted[interactionKey{q.ID, oi, src.Site.Key}]
		units = math.Min(left, available)
		if units <= 0 {
			return 0, 0, false
		}
		return units, units*w.Opt.InteractSecs + math.Max(0, units-1)*src.Site.Spacing*w.Opt.PathFactor/w.Opt.RunSpeed, true
	}
	// A purchase or skill check is one interaction, regardless of item count or the required skill rank.
	return units, w.Opt.InteractSecs, true
}
