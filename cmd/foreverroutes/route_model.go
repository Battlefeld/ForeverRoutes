package main

import "github.com/Battlefeld/ForeverRoutes/route"

// RouteModel travels with optimized events so every consumer replays the selected policy and buff assumptions.
// Legacy route files omit these fields and keep the legacy defaults.
type RouteModel struct {
	Preparations     *route.PreparationPlan  `json:"preparations,omitempty"`
	SleepingBagModel *route.SleepingBagModel `json:"sleepingBagModel,omitempty"`
}

func (m RouteModel) apply(w *route.World) error {
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
