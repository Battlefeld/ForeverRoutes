package main

import (
	"fmt"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/route"
)

// RouteModel travels with optimized events so every consumer replays the selected policy and buff assumptions.
// Legacy route files omit these fields and keep the legacy defaults.
type RouteModel struct {
	Preparations     *route.PreparationPlan  `json:"preparations,omitempty"`
	SleepingBagModel *route.SleepingBagModel `json:"sleepingBagModel,omitempty"`
	HearthMinSave    float64                 `json:"hearthMinSave,omitempty"`
	// Calibration scales the kill profile's fight and rest times by what was measured in the game (Timing.lua).
	Calibration *Calibration `json:"calibration,omitempty"`
	// Measured in the game: elite quests this player did alone (their creatures still go by the danger rules), plain
	// elites of the player's level or below fought alone, and how many levels above the player a creature may be.
	SoloQuests []int `json:"soloQuests,omitempty"`
	SoloElites bool  `json:"soloElites,omitempty"`
	MobAbove   int   `json:"mobAbove,omitempty"`
	// EliteRadius: an elite place is one with elites within this many yards (route.Options.EliteRadius)
	EliteRadius float64 `json:"eliteRadius,omitempty"`
}

// edit: the dataset as this model sees it (before the world is made).
func (m RouteModel) edit(d *dataset.Dataset) {
	solo := map[int]bool{}
	for _, id := range m.SoloQuests {
		solo[id] = true
	}
	for i := range d.Quests {
		if solo[d.Quests[i].ID] && d.Quests[i].Type == "elite" {
			d.Quests[i].Type = ""
		}
	}
}

// options: the world's options as this model sees them (before the world is made).
func (m RouteModel) options(o *route.Options) {
	if m.SoloElites {
		o.SoloElites = true
	}
}

// withModel runs load with the model's dataset and option rules in force (and the ones already set).
func withModel(m RouteModel, load func() (*route.World, error)) (*route.World, error) {
	pd, po := datasetEdit, optionsEdit
	defer func() { datasetEdit, optionsEdit = pd, po }()
	datasetEdit = func(d *dataset.Dataset) {
		if pd != nil {
			pd(d)
		}
		m.edit(d)
	}
	optionsEdit = func(o *route.Options) {
		if po != nil {
			po(o)
		}
		m.options(o)
	}
	return load()
}

// Calibration is a measured correction of the kill profile: fight and rest seconds are multiplied by these.
type Calibration struct {
	Fight  float64 `json:"fight"`
	Rest   float64 `json:"rest"`
	Source string  `json:"source"`
}

// calibratedProfile is a kill profile with measured factors on its fight and rest times.
type calibratedProfile struct {
	route.KillProfile
	c Calibration
}

func (p calibratedProfile) KillSeconds(c route.Character, level int, mob *dataset.NPC) (float64, float64) {
	f, r := p.KillProfile.KillSeconds(c, level, mob)
	return f * p.c.Fight, r * p.c.Rest
}

func (p calibratedProfile) Name() string {
	return fmt.Sprintf("%s (calibrated: fights x%.2f, rest x%.2f)", p.KillProfile.Name(), p.c.Fight, p.c.Rest)
}

func (m RouteModel) apply(w *route.World) error {
	if m.MobAbove > 0 {
		w.Opt.MaxMobAbove = m.MobAbove
	}
	if m.SoloElites {
		w.Opt.SoloElites = true
	}
	if m.EliteRadius > 0 {
		w.Opt.EliteRadius = m.EliteRadius
	}
	if c := m.Calibration; c != nil {
		if c.Fight <= 0 || c.Rest < 0 || c.Fight > 5 || c.Rest > 5 {
			return fmt.Errorf("calibration out of range: fights x%v, rest x%v", c.Fight, c.Rest)
		}
		if _, done := w.Opt.Profile.(calibratedProfile); !done {
			w.Opt.Profile = calibratedProfile{w.Opt.Profile, *c}
		}
	}
	if m.HearthMinSave > 0 {
		w.Opt.HearthMinSave = m.HearthMinSave
	}
	if m.Preparations != nil {
		if err := w.SetPreparations(*m.Preparations); err != nil {
			return err
		}
	}
	if m.SleepingBagModel != nil {
		if err := w.SetSleepingBagModel(*m.SleepingBagModel); err != nil {
			return err
		}
	}
	return nil
}
