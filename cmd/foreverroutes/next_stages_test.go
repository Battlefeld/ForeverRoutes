package main

import (
	"testing"
	"time"

	"github.com/Battlefeld/ForeverRoutes/route"
)

// From the middle of a real route (a Troll Shaman with dungeons at level 14 in the Barrens) the stages plan the next
// hours by zones: legs that differ from one another, a plan that replays, a head that ends after a block of
// hand-ins, and at least the XP per hour of following the route over the same stretch.
func TestStagesFromTheMiddleOfARoute(t *testing.T) {
	if testing.Short() {
		t.Skip("plans for seconds")
	}
	file, err := readRoute("../../data/routes/troll-shaman-dungeons.json")
	if err != nil {
		t.Skip("route not available:", err)
	}
	c := file.Character
	w, err := loadWorld("../../data/routes-data.json.gz", c.Race, c.Class, c.Faction, "../../data/profiles/shaman.json", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.RouteModel.apply(w); err != nil {
		t.Fatal(err)
	}
	g, err := w.FollowGuide(file.Events, 23)
	if err != nil {
		t.Fatal(err)
	}
	at := 0
	for i, st := range g.Steps {
		if st.Level >= 14 {
			at = i
			break
		}
	}
	s, err := w.StateAfter(g.Steps[:at])
	if err != nil {
		t.Fatal(err)
	}
	res, stages := w.PlanStages(s, route.PlanOptions{TargetLevel: 60}, 6*time.Second)
	if res == nil || len(stages) < 2 {
		t.Fatalf("no stages: %v", stages)
	}
	for i := 1; i < len(stages); i++ {
		if stages[i].Zone == stages[i-1].Zone && stages[i].Dungeon == stages[i-1].Dungeon {
			t.Errorf("legs %d and %d are one zone: %+v", i-1, i, stages)
		}
	}
	if res.Level <= w.Level(s) {
		t.Errorf("the stages bring no level: %d", res.Level)
	}
	head := res.Head(3600, route.BlockQuests)
	turnins := 0
	for _, st := range head.Steps {
		if st.Op == "turnin" {
			turnins++
		}
	}
	if len(head.Steps) == 0 || len(head.Steps) > len(res.Steps) || turnins < min(route.BlockQuests, res.Quests) && len(head.Steps) < len(res.Steps) {
		t.Errorf("head: %d steps, %d hand-ins", len(head.Steps), turnins)
	}
	ref, err := w.FollowGuideFrom(s, file.Events, 60, route.StageHorizon)
	if err != nil {
		t.Fatal(err)
	}
	if rs, rr := route.XPRate(s, res), route.XPRate(s, ref.Result); rs < rr {
		t.Errorf("stages %.0f XP/h, the route %.0f XP/h", rs*3600, rr*3600)
	}
	for _, st := range stages {
		t.Logf("%-30s %-20s L%d-%d %3.0f min %d quests", st.Zone, st.Dungeon, st.FromLevel, st.ToLevel, st.Minutes, st.Quests)
	}
}
