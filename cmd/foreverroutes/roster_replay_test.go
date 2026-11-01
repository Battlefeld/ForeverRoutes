package main

import (
	"encoding/json"
	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/route"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Opt-in: replays every scenario file a TestRebuildRoster run wrote into FR_ROSTER_REPLAY the way
// TestFinalizeRosterEvidence does, and lists the ones whose time moves. It needs no summary.json, so a rebuild of
// one context (go test -run '^TestRebuildRoster/skyborne-a-mage$') can be checked on its own.
func TestReplayRosterScenarios(t *testing.T) {
	dir := os.Getenv("FR_ROSTER_REPLAY")
	if dir == "" {
		t.Skip("opt-in replay of rebuilt scenario files")
	}
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := dataset.Load("../../data/routes-data.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	walks, err := dataset.LoadWalks("../../data/walks.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	walkTable, err := dataset.LoadWalkTable("../../data/walktable.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	profiles := map[string]*route.TableProfile{}
	checked := 0
	for _, path := range files {
		if filepath.Base(path) == "summary.json" {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var saved struct {
			route.OptimizationSeed
			Character   route.Character              `json:"character"`
			Constraints route.PreparationConstraints `json:"constraints"`
			Bag         route.SleepingBagModel       `json:"sleepingBagModel"`
			Seconds     float64                      `json:"seconds"`
			Scenario    string                       `json:"scenario"`
			Error       string                       `json:"error"`
		}
		if err = json.Unmarshal(b, &saved); err != nil {
			t.Fatal(err)
		}
		if saved.Error != "" || saved.Preparations == nil {
			continue
		}
		profile := profiles[saved.Character.Class]
		if profile == nil {
			f, err := os.Open("../../data/profiles/" + strings.ToLower(saved.Character.Class) + ".json")
			if err != nil {
				t.Fatal(err)
			}
			profile, err = route.LoadTableProfile(f)
			f.Close()
			if err != nil {
				t.Fatal(err)
			}
			profiles[saved.Character.Class] = profile
		}
		w, err := route.NewWorld(d, saved.Character, route.Options{Profile: profile, Walks: walks, WalkTable: walkTable, Dungeons: strings.HasPrefix(saved.Scenario, "dungeons_"), Preparations: saved.Constraints, SleepingBag: saved.Bag})
		if err != nil {
			t.Fatal(err)
		}
		if err = w.SetPreparations(*saved.Preparations); err != nil {
			t.Fatal(err)
		}
		w.SetWandDetour(saved.WandDetour)
		r, err := w.FollowGuide(saved.Events, 20)
		checked++
		if err != nil || r.Level < 20 || len(r.Skipped) > 0 || math.Abs(r.Seconds-saved.Seconds) > .11 {
			t.Errorf("%s %s: replay %.1f vs saved %.1f (level %d, skipped %v, err %v)", saved.Character.RouteKey(), saved.Scenario, r.Seconds, saved.Seconds, r.Level, r.Skipped, err)
		}
	}
	t.Logf("%d scenarios replayed", checked)
}
