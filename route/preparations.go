package route

import (
	"fmt"
	"github.com/Battlefeld/ForeverRoutes/dataset"
	"math"
)

// PreparationPlan is part of a route's reproducible model configuration, not an implicit world-wide switch.
// Training and material procurement still use the existing estimated cost model; they are not inventory simulation.
type PreparationPlan struct {
	ForceWand        bool    `json:"forceWand"`
	ForceProfessions bool    `json:"forceProfessions"`
	Camping          bool    `json:"camping"`
	Wands            bool    `json:"wands"`
	ProfessionDetour float64 `json:"professionDetour"`
	ProfessionLevel  int     `json:"professionLevel"`
	BagStacks        int     `json:"bagStacks"`
}

// PreparationConstraints pins a user choice; empty/auto lets Optimize compare alternatives.
type PreparationConstraints struct {
	Camping     string `json:"camping,omitempty"`
	Wands       string `json:"wands,omitempty"`
	SleepingBag string `json:"sleepingBag,omitempty"`
}

// SleepingBagModel keeps disputed beta values visible and replaceable. Placement is an estimate.
// Tooltip: https://www.wowhead.com/forever/item=211527 ; duration: /forever/spell=429959 .
// The guide also reports one hour, so published comparisons must identify this profile explicitly.
type SleepingBagModel struct {
	PerStack         float64 `json:"perStack"`
	StackSeconds     float64 `json:"stackSeconds"`
	PlacementSeconds float64 `json:"placementSeconds"`
	Duration         float64 `json:"duration"`
	Cooldown         float64 `json:"cooldown"`
	Source           string  `json:"source"`
}

func DefaultSleepingBagModel() SleepingBagModel {
	return SleepingBagModel{.01, 60, 10, 7200, 3600, "Forever item 211527 / spell 429959 tooltip; duration disputed (3600/7200s); placement estimated"}
}
func (w *World) Preparations() PreparationPlan { return w.preparation }
func (w *World) SetPreparations(p PreparationPlan) error {
	if p.Wands && !WandClass(w.Char.Class) {
		return fmt.Errorf("class cannot use the selected wand preparation")
	}
	if p.BagStacks < 0 || p.BagStacks > 3 || p.ProfessionLevel < 1 || p.ProfessionLevel > 60 || p.ProfessionDetour < 0 || math.IsNaN(p.ProfessionDetour) || math.IsInf(p.ProfessionDetour, 0) {
		return fmt.Errorf("invalid preparation plan")
	}
	w.preparation = p
	return nil
}
func validMode(s string) bool { return s == "" || s == "auto" || s == "on" || s == "off" }
func (w *World) planAllowed(p PreparationPlan) bool {
	c := w.Opt.Preparations
	return !(c.Camping == "on" && !p.Camping || c.Camping == "off" && p.Camping || c.Wands == "on" && !p.Wands || c.Wands == "off" && p.Wands || c.SleepingBag == "off" && p.BagStacks != 0)
}
func (w *World) bagOwned(s *State) bool {
	if s.BagOwnershipKnown {
		return s.OwnedBag
	}
	q := w.ByID[79976]
	return q != nil && s.Status[q.Index] == done
}
func (w *World) bonusXP(s *State, base int, at float64) int {
	if base <= 0 {
		return base
	}
	if s.BagUntil > at && s.BagStacks > 0 {
		return int(math.Round(float64(base) * (1 + float64(s.BagStacks)*w.Opt.SleepingBag.PerStack)))
	}
	return base
}

// awardKillXP spreads an aggregate dungeon award uniformly across its modelled kill interval. Actual mob order is unknown.
func (w *World) awardKillXP(s *State, base int, from, to float64) {
	if base <= 0 {
		return
	}
	active := 0
	if s.BagStacks > 0 && s.BagUntil > from {
		fraction := 1.0
		if to > from {
			fraction = math.Max(0, math.Min(1, (s.BagUntil-from)/(to-from)))
		}
		active = int(math.Round(float64(base) * fraction))
	}
	gain := base - active + w.bonusXP(s, active, from)
	s.XP += gain + restedGain(s, gain)
}

func (w *World) SetSleepingBagModel(m SleepingBagModel) error {
	for _, v := range []float64{m.PerStack, m.StackSeconds, m.PlacementSeconds, m.Duration, m.Cooldown} {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("non-finite sleeping bag model")
		}
	}
	if m.PerStack < 0 || m.PerStack > .1 || m.StackSeconds <= 0 || m.PlacementSeconds < 0 || m.Duration <= 0 || m.Cooldown <= 0 {
		return fmt.Errorf("invalid sleeping bag model")
	}
	w.Opt.SleepingBag = m
	return nil
}

// Approximate placement checks until live map-area flags are supplied: do not rest in a known inn/capital or
// a known hostile/elite camp. This is conservative geometry, not a claim that every valid terrain point is safe.
func (w *World) canSleep(s *State) bool {
	if s.RestAreaAt != nil && distance(s.Pos, *s.RestAreaAt) < 50 {
		return false
	}
	for _, p := range w.inns {
		if distance(s.Pos, p) < 150 {
			return false
		}
	}
	for _, p := range homeCapital {
		if distance(s.Pos, p) < 1000 {
			return false
		}
	}
	if w.danger.at(s.Pos) > 0 || w.elites.at(s.Pos) > 0 {
		return false
	}
	return true
}

func atTrainerHub(at dataset.Point, hubs []dataset.Point) bool {
	for _, p := range hubs {
		if distance(at, p) <= 150 {
			return true
		}
	}
	return false
}
