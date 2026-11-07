package route

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

func TestContinuationKeepsBaselineAndReplaysFromProgress(t *testing.T) {
	w := tinyWorld(t)
	s := w.NewState()
	// Begin with an accepted quest: none of the trials may lose its progress
	// or accept it twice when starting a continuation.
	if err := w.Apply(s, Action{Op: "accept", Quest: 1, At: w.Start}); err != nil {
		t.Fatal(err)
	}
	before := s.clone()
	base, err := w.PlanAhead(s, PlanOptions{TargetLevel: 3}, 1e9)
	if err != nil {
		t.Fatal(err)
	}
	best, trials, err := w.PlanContinuation(s, 3)
	if err != nil {
		t.Fatal(err)
	}
	if best.Level < 3 || best.Seconds > base.Seconds || len(trials) != 4 {
		t.Fatalf("invalid portfolio: best=%+v baseline=%v trials=%+v", best, base.Seconds, trials)
	}
	if trials[0].Name != "baseline" || trials[0].Seconds != base.Seconds {
		t.Fatal("portfolio changed the baseline policy")
	}
	if !reflect.DeepEqual(s.clone(), before) {
		t.Fatal("planning mutated the starting state")
	}
	var actions []Action
	for _, st := range best.Steps {
		actions = append(actions, st.Action)
	}
	replayed, err := w.replayFrom(s, actions)
	if err != nil || replayed.Seconds != best.Seconds || replayed.XP != best.XP {
		t.Fatalf("continuation did not replay: %v", err)
	}
}

func TestContinuationRejectsInvalidTarget(t *testing.T) {
	w := tinyWorld(t)
	if _, trials, err := w.PlanContinuation(w.NewState(), 99); err == nil || len(trials) != 4 {
		t.Fatal("invalid target accepted")
	}
}

func TestContinuationRejectsLostClassMilestone(t *testing.T) {
	d := tiny()
	d.NPCs = append(d.NPCs, dataset.NPC{ID: 501, Name: "Too high", MinLevel: 6, MaxLevel: 6, HP: 500, Spawns: d.NPCs[1].Spawns})
	d.Quests = append(d.Quests, dataset.Quest{ID: 1470, Name: "Required imp", Classes: ClassBits["Warlock"], Level: 1, MinLevel: 1, XP: 1,
		Start: d.Quests[0].Start, End: d.Quests[0].End,
		Objectives: []dataset.Objective{{Kind: "kill", ID: 501, Count: 1}}})
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Warlock"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	base, err := w.PlanAhead(w.NewState(), PlanOptions{TargetLevel: 2}, 1e9)
	if err != nil || base.Level < 2 {
		t.Fatalf("fixture needs a reachable XP target: %v", err)
	}
	_, trials, err := w.PlanContinuation(w.NewState(), 2)
	if err == nil {
		t.Fatal("accepted XP target without required class milestone")
	}
	for _, trial := range trials {
		if !strings.Contains(trial.Error, "required class quest 1470") {
			t.Fatalf("wrong rejection: %+v", trial)
		}
	}
}
