package route

import (
	"reflect"
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

func TestInteractionNeedsDistinctTargetsAndGivesNoKillCredit(t *testing.T) {
	d := tiny()
	d.Quests[0].Objectives = []dataset.Objective{{Kind: "interact", ID: 100, Count: 5}}
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Warlock"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := w.NewState()
	for _, id := range []int{1, 2} {
		if err := w.Apply(s, Action{Op: "accept", Quest: id}); err != nil {
			t.Fatal(err)
		}
	}
	q := w.ByID[1]
	src := q.Objectives[0].Sources[1] // four boars west, ten east
	a := Action{Op: "work", Quest: 1, NPC: 100, At: src.Site.Center}
	if err := w.Apply(s, a); err != nil {
		t.Fatal(err)
	}
	if s.Prog[q.Index][0] != 4 || s.XP != 0 || s.Combat != 0 || len(s.Dead) != 0 || s.Prog[w.ByID[2].Index][0] != 0 {
		t.Fatalf("interaction awarded kill credit or reused targets: %+v", s)
	}
	before := s.clone()
	if err := w.Apply(s, a); err == nil {
		t.Fatal("reused the same four targets")
	}
	if !reflect.DeepEqual(s, before) {
		t.Fatal("failed interaction mutated state")
	}
	a, ok := w.bestSource(s, q, 0)
	if !ok || a.At.X < 0 {
		t.Fatal("planner did not move to unused targets")
	}
	if err := w.Apply(s, a); err != nil {
		t.Fatal(err)
	}
	if !w.complete(s, q) || s.Prog[q.Index][0] != 5 {
		t.Fatal("did not complete exactly five interactions")
	}
	clone := s.clone()
	clone.Interacted[interactionKey{q.ID, 0, src.Site.Key}] = 99
	if reflect.DeepEqual(s.Interacted, clone.Interacted) {
		t.Fatal("clones share interaction state")
	}
}

func TestDuelCostsCombatWithoutKillXPOrLoot(t *testing.T) {
	d := tiny()
	d.Quests[0].Objectives = []dataset.Objective{{Kind: "duel", ID: 100, Count: 3}}
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Warlock"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := w.NewState()
	for _, id := range []int{1, 2} {
		if err := w.Apply(s, Action{Op: "accept", Quest: id}); err != nil {
			t.Fatal(err)
		}
	}
	a, ok := w.bestSource(s, w.ByID[1], 0)
	if !ok {
		t.Fatal("duel unavailable")
	}
	if err := w.Apply(s, a); err != nil {
		t.Fatal(err)
	}
	if s.Combat <= 0 || s.XP != 0 || len(s.Dead) != 0 || s.Prog[w.ByID[2].Index][0] != 0 || !w.complete(s, w.ByID[1]) {
		t.Fatalf("wrong duel credit: %+v", s)
	}
}

func TestRejectedActionDoesNotMakeCamp(t *testing.T) {
	d := tiny()
	d.LevelXP = make([]int, 20)
	for i := range d.LevelXP {
		d.LevelXP[i] = 100
	}
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Warlock"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := w.NewState()
	s.XP = 900
	s.Prof = true
	before := *s
	if err := w.Apply(s, Action{Op: "work", Quest: 999999}); err == nil {
		t.Fatal("invalid quest worked")
	}
	if !reflect.DeepEqual(*s, before) {
		t.Fatal("failed work charged a camp or rested XP")
	}
}

func TestUnknownDungeonDataCannotBeCompletedByRun(t *testing.T) {
	for _, scenario := range []string{"unsupported", "unknown-kind", "zero-count", "outside-source"} {
		t.Run(scenario, func(t *testing.T) {
			d := tinyDungeon()
			q := &d.Quests[len(d.Quests)-2]
			switch scenario {
			case "unsupported":
				q.Unsupported = "objective count unknown"
			case "unknown-kind":
				q.Objectives[0].Kind = "mystery"
			case "zero-count":
				q.Objectives[0].Count = 0
			case "outside-source":
				q.Objectives[0].Inside = false
				q.Objectives[0].ID = 999999
			}
			w, err := NewWorld(d, Character{Race: "Orc", Class: "Warlock"}, Options{Dungeons: true})
			if err != nil {
				t.Fatal(err)
			}
			if w.ByID[50] != nil {
				t.Fatal("unknown objective became automatic dungeon credit")
			}
		})
	}
}

func TestRepeatablePrerequisiteCanBeDoneOnce(t *testing.T) {
	d := tiny()
	d.Quests[0].Repeatable = true
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Warlock"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	q := w.ByID[1]
	if q == nil || w.ByID[3] == nil {
		t.Fatal("repeatable prerequisite blocked follow-up")
	}
	s := w.NewState()
	if err := w.Apply(s, Action{Op: "accept", Quest: 1}); err != nil {
		t.Fatal(err)
	}
	a, _ := w.bestSource(s, q, 0)
	if err := w.Apply(s, a); err != nil {
		t.Fatal(err)
	}
	if err := w.Apply(s, Action{Op: "turnin", Quest: 1}); err != nil {
		t.Fatal(err)
	}
	if w.CanAccept(s, q) == nil {
		t.Fatal("repeatable quest created an XP loop")
	}
	if w.CanAccept(s, w.ByID[3]) != nil {
		t.Fatal("follow-up is blocked after prerequisite")
	}
}
