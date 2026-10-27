package route

import (
	"fmt"
	"math"
	"time"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// FeedbackSettings is deliberately inactive by default. Neither a level-up, a date, nor an incoming FS payload
// activates it. Release confirmation AND the user's explicit post-release approval must be recorded by ForeverCore.
type FeedbackSettings struct {
	Enabled                   bool   `json:"enabled"`
	ReleaseConfirmed          bool   `json:"releaseConfirmed"`
	UserConfirmedAfterRelease bool   `json:"userConfirmedAfterRelease"`
	ReleaseBuild              string `json:"releaseBuild"`
	ConfirmedAt               string `json:"confirmedAt"`
}

func (f FeedbackSettings) Active() bool {
	if !f.Enabled || !f.ReleaseConfirmed || !f.UserConfirmedAfterRelease || f.ReleaseBuild == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, f.ConfirmedAt)
	return f.Enabled && f.ReleaseConfirmed && f.UserConfirmedAfterRelease && f.ReleaseBuild != "" && err == nil && !t.IsZero() && !t.After(time.Now())
}

// CombatFeedback is the versioned read-only handoff from ForeverCore/ForeverSim. No FS code or data is modified.
// Gear changes invalidate the previous equipment revision; an old measurement never scales unrelated enemies.
type CombatFeedback struct {
	GameBuild         string            `json:"gameBuild"`
	Schema            string            `json:"schema"`
	Character         Character         `json:"character"`
	EquipmentRevision string            `json:"equipmentRevision"`
	BuildRevision     string            `json:"buildRevision"`
	Source            string            `json:"source"` // foreversim or measured
	Kills             []KillObservation `json:"kills"`
}
type KillObservation struct {
	NPC          int     `json:"npc"`
	PlayerLevel  int     `json:"playerLevel"`
	FightSeconds float64 `json:"fightSeconds"`
	RestSeconds  float64 `json:"restSeconds"`
	Samples      int     `json:"samples"`
}

func (w *World) AcceptFeedback(f CombatFeedback) error {
	if !w.Opt.Feedback.Active() {
		return fmt.Errorf("ForeverSim live feedback disabled: post-release user confirmation required")
	}
	if f.Schema != "foreverroutes.combat-feedback.v1" || !SameCharacter(f.Character, w.Char) || f.EquipmentRevision == "" || f.BuildRevision == "" || f.GameBuild != w.Opt.Feedback.ReleaseBuild || (f.Source != "foreversim" && f.Source != "measured") {
		return fmt.Errorf("invalid or incompatible combat feedback")
	}
	for _, k := range f.Kills {
		if k.NPC <= 0 || k.PlayerLevel < 1 || k.PlayerLevel > 60 || k.FightSeconds <= 0 || k.RestSeconds < 0 || math.IsNaN(k.FightSeconds+k.RestSeconds) || math.IsInf(k.FightSeconds+k.RestSeconds, 0) || (f.Source == "measured" && k.Samples < 3) {
			return fmt.Errorf("invalid or insufficient kill observation")
		}
	}
	c := f
	c.Kills = append([]KillObservation(nil), f.Kills...)
	w.feedback = &c
	return nil
}
func (w *World) feedbackSeconds(s *State, n *dataset.NPC) (float64, float64, bool) {
	f := w.feedback
	if !w.Opt.Feedback.Active() || f == nil || s.EquipmentRevision == "" || s.EquipmentRevision != f.EquipmentRevision || s.BuildRevision != f.BuildRevision {
		return 0, 0, false
	}
	for _, k := range f.Kills {
		if k.NPC == n.ID && k.PlayerLevel == w.Level(s) {
			return k.FightSeconds, k.RestSeconds, true
		}
	}
	return 0, 0, false
}
