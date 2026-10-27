package route

import (
	"encoding/json"
	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/xp"
	"math"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestGuideKeepsPartialObjectivesAndXPThreshold(t *testing.T) {
	d := tiny()
	d.Quests[0].Objectives = append(d.Quests[0].Objectives, dataset.Objective{Kind: "kill", ID: 200, Count: 1})
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Warlock"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	r, err := w.FollowGuide([]GuideEvent{{Op: "accept", Quest: 1}, {Op: "complete", Quest: 1, Objective: 1}, {Op: "level", Level: 2, XP: 150}}, 3)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range r.Steps {
		if s.Op == "work" && s.Obj == 1 {
			t.Fatal("partial complete worked another objective")
		}
	}
	if len(r.Skipped) != 0 {
		t.Fatal(r.Skipped)
	}
	// Inspect the prefix before the final automatic grind.
	s := w.NewState()
	if err = w.Apply(s, Action{Op: "accept", Quest: 1}); err != nil {
		t.Fatal(err)
	}
	a, ok := w.bestSource(s, w.ByID[1], 0)
	if !ok {
		t.Fatal("source missing")
	}
	if err = w.Apply(s, a); err != nil {
		t.Fatal(err)
	}
	if s.Prog[w.ByID[1].Index][1] != 0 {
		t.Fatal("other objective credited")
	}
	e := EventsOf([]Step{{Action: Action{Op: "grind"}, Level: 2, XP: xp.Total(d.LevelXP, 2) + 150}})
	if e[0].TotalXP != 550 {
		t.Fatal(e)
	}
	r, err = w.FollowGuide([]GuideEvent{{Op: "level", Level: 2, XP: 150}, {Op: "accept", Quest: 2}}, 3)
	if err != nil {
		t.Fatal(err)
	}
	seen := false
	for _, step := range r.Steps {
		if step.Op == "accept" && step.Quest == 2 {
			seen = true
			if step.XP < 550 {
				t.Fatal("XP threshold lost", step.XP)
			}
		}
	}
	if !seen {
		t.Fatal("threshold route did not accept quest")
	}
}

func TestLiveRejectsInconsistentFacts(t *testing.T) {
	w := tinyWorld(t)
	p := w.Start
	for _, l := range []Live{
		{Level: 2, XP: 900, Pos: &p}, {Level: 2, XP: -1, Pos: &p}, {Level: 2, Pos: &p, HearthIn: math.NaN()},
		{Level: 2, Pos: &p, Active: []LiveQuest{{ID: 1}, {ID: 1}}},
		{Level: 2, Pos: &p, Completed: []int{1}, Active: []LiveQuest{{ID: 1}}},
		{Level: 2, Pos: &p, Active: []LiveQuest{{ID: 1, Objectives: []LiveObjective{{Have: -1, Need: 5}}}}},
	} {
		if _, _, err := w.StateFrom(l); err == nil {
			t.Errorf("accepted inconsistent live facts: %+v", l)
		}
	}
	if _, err := NewWorld(tiny(), Character{Race: "Orc", Class: "Warlock", Faction: "A"}, Options{}); err == nil {
		t.Fatal("wrong faction accepted")
	}
	if _, err := NewWorld(tiny(), Character{Race: "Orc", Class: "Warlock"}, Options{RunSpeed: -1}); err == nil {
		t.Fatal("negative speed accepted")
	}
}

func TestSharedProfileHasIndependentFallbackCaches(t *testing.T) {
	p, err := LoadTableProfile(strings.NewReader(tableJSON))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewWorld(tiny(), Character{Race: "Orc", Class: "Warlock"}, Options{Profile: p}); err == nil {
		t.Fatal("library accepted another class's combat profile")
	}
	if _, err := NewWorld(tiny(), Character{Race: "Orc", Class: "Mage"}, Options{Profile: &TableProfile{Class: "mage"}}); err == nil {
		t.Fatal("uninitialized table profile accepted")
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w, err := NewWorld(tiny(), Character{Race: "Orc", Class: "Mage"}, Options{Profile: p})
			if err != nil {
				t.Error(err)
				return
			}
			w.Opt.Profile.KillSeconds(w.Char, 2, &dataset.NPC{HP: 60, MaxLevel: 2})
		}()
	}
	wg.Wait()
	if p.Fallback.EvenHP != nil {
		t.Fatal("NewWorld mutated caller's profile")
	}
}

func TestConservativeCurveNeverAddsDamage(t *testing.T) {
	raw := strings.Replace(tableJSON, `"seconds":10,"damage":200`, `"seconds":10,"damage":50`, 1)
	raw = strings.Replace(raw, `"0":[`, `"0":[{"seconds":15,"damage":300,"manaSpent":250},`, 1)
	p, err := LoadTableProfile(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.CurveWarnings) != 1 {
		t.Fatal(p.CurveWarnings)
	}
	if p.Levels[0].Targets["0"][0].Damage != 50 {
		t.Fatal("earlier damage was not reduced conservatively")
	}
	for _, s := range []string{strings.Replace(tableJSON, `"health":200`, `"health":0`, 1), strings.Replace(tableJSON, `"seconds":10,"damage":200`, `"seconds":5,"damage":200`, 1)} {
		if _, err := LoadTableProfile(strings.NewReader(s)); err == nil {
			t.Fatal("invalid profile accepted")
		}
	}
}

func TestAllShippedProfileTimesAreFinite(t *testing.T) {
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	for _, class := range []string{"warrior", "paladin", "hunter", "rogue", "priest", "shaman", "mage", "warlock", "druid"} {
		f, err := os.Open("../data/profiles/" + class + ".json")
		if err != nil {
			t.Fatal(err)
		}
		p, err := LoadTableProfile(f)
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		for level := 1; level <= 60; level++ {
			for i := range d.NPCs {
				n := &d.NPCs[i]
				if n.HP <= 0 || n.MaxLevel < level-4 || n.MaxLevel > level+3 {
					continue
				}
				fight, rest := p.KillSeconds(Character{Class: p.Class}, level, n)
				if fight <= 0 || rest < 0 || math.IsNaN(fight+rest) || math.IsInf(fight+rest, 0) {
					t.Fatalf("%s level %d NPC %d invalid timing %v %v", class, level, n.ID, fight, rest)
				}
			}
		}
	}
}

func TestRosterKeysAndFactionAliases(t *testing.T) {
	chars := SupportedCharacters()
	if len(chars) != 56 {
		t.Fatal(len(chars))
	}
	seen := map[string]bool{}
	for _, c := range chars {
		if seen[c.RouteKey()] {
			t.Fatal(c)
		}
		seen[c.RouteKey()] = true
	}
	if !SameCharacter(Character{Race: "Skyborne", Class: "Mage", Faction: "Alliance"}, Character{Race: "Skyborne", Class: "Mage", Faction: "A"}) {
		t.Fatal("faction aliases differ")
	}
	if SameCharacter(Character{Race: "Human", Class: "Mage", Faction: "H"}, Character{Race: "Human", Class: "Mage"}) {
		t.Fatal("invalid faction accepted")
	}
	if _, err := json.Marshal(chars); err != nil {
		t.Fatal(err)
	}
}
