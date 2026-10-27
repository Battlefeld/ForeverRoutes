package route

import (
	"testing"
	"time"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Improve never returns a guide slower than the one it starts from, and puts in a quest the guide passes by when
// that is faster than grinding.
func TestImproveNeverSlower(t *testing.T) {
	w := tinyWorld(t)
	guide := []GuideEvent{{Op: "accept", Quest: 1}, {Op: "complete", Quest: 1}, {Op: "turnin", Quest: 1}}
	start, err := w.FollowGuide(guide, 3)
	if err != nil {
		t.Fatal(err)
	}
	events, res, err := w.Improve(guide, 3, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if res.Seconds > start.Seconds || res.Level < 3 {
		t.Fatalf("improved %.0f s (level %d), guide %.0f s", res.Seconds, res.Level, start.Seconds)
	}
	again, _ := w.FollowGuide(events, 3)
	if again.Seconds != res.Seconds {
		t.Errorf("the improved guide walks in %.0f s, reported %.0f s", again.Seconds, res.Seconds)
	}
	if len(EventsOf(start.Steps)) == 0 {
		t.Error("a walked guide gives no steps back")
	}
}

// The planner's grinding stays in the guide steps as a level step; back-to-back grinds are one step.
func TestEventsOfKeepsGrinding(t *testing.T) {
	steps := []Step{
		{Action: Action{Op: "accept", Quest: 1}, Level: 3},
		{Action: Action{Op: "grind"}, Level: 3},
		{Action: Action{Op: "grind"}, Level: 4},
		{Action: Action{Op: "work", Quest: 1}, Level: 4},
		{Action: Action{Op: "work", Quest: 1}, Level: 4},
		{Action: Action{Op: "turnin", Quest: 1}, Level: 5},
	}
	got := EventsOf(steps)
	want := []GuideEvent{{Op: "accept", Quest: 1}, {Op: "level", Level: 4}, {Op: "complete", Quest: 1, Objective: 1}, {Op: "turnin", Quest: 1}}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

// A required class quest (the warlock's imp) goes into a route that lacks it, and an improvement keeps it.
func TestRequiredClassQuestsGoIn(t *testing.T) {
	d := tiny()
	giver := []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}}
	d.Quests = append(d.Quests, dataset.Quest{ID: 1470, Name: "Piercing the Veil", Level: 2, MinLevel: 1, XP: 200, Classes: 256,
		Start: giver, End: giver, Objectives: []dataset.Objective{{Kind: "kill", ID: 100, Count: 2}}})
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Warlock"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	chains := w.ClassChains(5)
	if len(chains) != 1 || chains[0].Tier != "required" || chains[0].Reward != "Wichtel" {
		t.Fatalf("chains %+v", chains)
	}
	guide := []GuideEvent{{Op: "accept", Quest: 1}, {Op: "complete", Quest: 1}, {Op: "turnin", Quest: 1}}
	events, _, err := w.Improve(guide, 3, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range events {
		found = found || e.Quest == 1470 && e.Op == "turnin"
	}
	if !found {
		t.Error("the imp quest must be in the route")
	}
	rep := w.ClassReport(events, 3)
	if len(rep) != 1 || !rep[0].InRoute {
		t.Errorf("report %+v", rep)
	}
}

// A prerequisite of another race is none for this character.
func TestOtherRacesPrerequisiteIgnored(t *testing.T) {
	d := tiny()
	giver := []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}}
	d.Quests = append(d.Quests, dataset.Quest{ID: 60, Name: "Copied", Level: 1, MinLevel: 1, XP: 100, PreAll: []int{4}, Start: giver, End: giver})
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Warlock"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if q := w.ByID[60]; q == nil || len(q.PreAll) != 0 {
		t.Fatalf("quest 60: %v %s", q, w.Excluded[60])
	}
	if len(d.Quests[len(d.Quests)-1].PreAll) != 1 {
		t.Error("the dataset itself must stay as it is")
	}
}

func TestZephrasQuestsOnlyForSkyborne(t *testing.T) {
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Skip("dataset not available:", err)
	}
	w, err := NewWorld(d, Character{Race: "Tauren", Class: "Shaman"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range w.Quests {
		if q.StartPos.Map == zephrasMap {
			t.Errorf("a tauren takes %d %s on Zephras Isle", q.ID, q.Name)
		}
	}
}
