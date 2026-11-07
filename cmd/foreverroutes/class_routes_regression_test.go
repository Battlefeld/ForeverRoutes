package main

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Battlefeld/ForeverRoutes/route"
)

// Reprice the previous published winners with today's model. A corrected combat
// model may legitimately increase advertised hours; losing an old route that is
// still faster under that same corrected model is a genuine search regression.
func TestShippedClassRoutesRetainPreviousFeasibleWinner(t *testing.T) {
	if testing.Short() {
		t.Skip("full previous-winner comparison across 112 shipped routes")
	}
	f, err := os.Open("../../data/audits/roster-v23.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	var previous struct {
		Results []struct {
			RouteModel
			Character      route.Character    `json:"character"`
			Scenario       string             `json:"scenario"`
			Events         []route.GuideEvent `json:"events"`
			WandDetour     float64            `json:"wandDetour"`
			ProjectionOnly bool               `json:"projectionOnly"`
			Error          string             `json:"error"`
		} `json:"results"`
	}
	if err = json.NewDecoder(z).Decode(&previous); err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, old := range previous.Results {
		if old.ProjectionOnly || old.Error != "" || !strings.HasSuffix(old.Scenario, "_auto") {
			continue
		}
		checked++
		group := strings.HasPrefix(old.Scenario, "dungeons_")
		key := old.Character.RouteKey()
		if group {
			key += "-dungeons"
		}
		t.Run(key, func(t *testing.T) {
			w, err := loadWorld("../../data/routes-data.json.gz", old.Character.Race, old.Character.Class, old.Character.Faction, automaticProfile, group)
			if err != nil {
				t.Fatal(err)
			}
			if err = old.RouteModel.apply(w); err != nil {
				t.Fatal(err)
			}
			w.SetWandDetour(old.WandDetour)
			incumbent, err := w.FollowGuide(old.Events, 20)
			if err != nil || incumbent.Level < 20 || len(incumbent.Skipped) != 0 {
				t.Log("old winner is no longer executable under the corrected model")
				return
			}
			for _, chain := range w.ClassChains(20) {
				if chain.Tier == "required" && !slices.ContainsFunc(incumbent.Steps, func(step route.Step) bool { return step.Op == "turnin" && step.Quest == chain.Goal.ID }) {
					t.Log("old winner lacks a currently required class goal")
					return
				}
			}
			current, err := readRoute(filepath.Join("../../data/routes", key+".json"))
			if err != nil {
				t.Fatal(err)
			}
			if err = current.RouteModel.apply(w); err != nil {
				t.Fatal(err)
			}
			w.SetWandDetour(current.WandDetour)
			result, err := w.FollowGuide(current.Events, 20)
			if err != nil || result.Level < 20 || len(result.Skipped) != 0 {
				t.Fatalf("new class route is not executable: %v", err)
			}
			if result.Seconds > incumbent.Seconds+.11 {
				t.Fatalf("lost known feasible winner: new %.2fs, previous repriced %.2fs", result.Seconds, incumbent.Seconds)
			}
		})
	}
	if checked != 112 {
		t.Fatalf("checked %d previous winners, expected 112", checked)
	}
}
