package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Battlefeld/ForeverRoutes/route"
)

func TestRouteModelSurvivesFileRoundTrip(t *testing.T) {
	w, err := loadWorld("../../data/routes-data.json.gz", "Undead", "Warlock", "", "", true)
	if err != nil {
		t.Fatal(err)
	}
	var file routeFileData
	raw := `{"events":[{"op":"sleep","stacks":2}],"preparations":{"camping":false,"wands":false,"professionDetour":180,"professionLevel":8,"bagStacks":2},"sleepingBagModel":{"perStack":0.01,"stackSeconds":60,"placementSeconds":10,"duration":3600,"cooldown":3600,"source":"sensitivity profile"}}`
	if err := json.Unmarshal([]byte(raw), &file); err != nil {
		t.Fatal(err)
	}
	if err := file.RouteModel.apply(w); err != nil {
		t.Fatal(err)
	}
	if w.Preparations().Camping || w.Preparations().Wands || w.Opt.SleepingBag.Duration != 3600 || file.Events[0].Stacks != 2 {
		t.Fatal("selected model was not restored")
	}
	b, err := json.Marshal(file)
	if err != nil {
		t.Fatal(err)
	}
	var second routeFileData
	if err := json.Unmarshal(b, &second); err != nil {
		t.Fatal(err)
	}
	if *second.Preparations != *file.Preparations || *second.SleepingBagModel != *file.SleepingBagModel {
		t.Fatal("route metadata was lost")
	}
}
func TestSleepingBagExportIncludesUseAndRestDuration(t *testing.T) {
	w, err := loadWorld("../../data/routes-data.json.gz", "Undead", "Warlock", "", "", true)
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	writeStep(&b, w, &route.GuideResult{}, route.Step{Action: route.Action{Op: "sleep", Stacks: 2}, Name: "Sleep", Level: 18})
	if !strings.Contains(b.String(), `op = "sleep"`) || !strings.Contains(b.String(), "seconds = 130") || !strings.Contains(b.String(), "it = 211527") {
		t.Fatalf("missing user instruction: %s", b.String())
	}
}
