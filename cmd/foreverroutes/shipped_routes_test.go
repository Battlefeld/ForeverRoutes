package main

import (
	"encoding/json"
	"github.com/Battlefeld/ForeverRoutes/route"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The release gate replays every shipped route using its own character, combat profile and policy.
// Updating code/data without regenerating stale route metadata must fail, rather than ship old advertised hours.
func TestShippedRoutesMatchCurrentModel(t *testing.T) {
	if testing.Short() {
		t.Skip("full 112-route release check")
	}
	dir := "../../data/routes"
	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 112 {
		t.Fatalf("%d routes, expected 112", len(files))
	}
	expected := map[string]bool{}
	for _, c := range route.SupportedCharacters() {
		expected[c.RouteKey()] = true
		expected[c.RouteKey()+"-dungeons"] = true
	}
	for _, path := range files {
		key := strings.TrimSuffix(filepath.Base(path), ".json")
		if !expected[key] {
			t.Fatalf("unexpected route %s", key)
		}
		delete(expected, key)
		t.Run(key, func(t *testing.T) {
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			var r struct {
				routeFile
				Revision string  `json:"modelRevision"`
				Seconds  float64 `json:"seconds"`
			}
			if err = json.Unmarshal(b, &r); err != nil {
				t.Fatal(err)
			}
			if r.Revision != route.TimelineModelRevision || r.Target != 20 || r.Preparations == nil || r.SleepingBagModel == nil || r.Seconds <= 0 {
				t.Fatal("stale or incomplete route model metadata")
			}
			base := r.Character.RouteKey()
			if strings.TrimSuffix(key, "-dungeons") != base {
				t.Fatal("route file points to another character")
			}
			w, err := loadWorld("../../data/routes-data.json.gz", r.Character.Race, r.Character.Class, r.Character.Faction, "../../data/profiles/"+strings.ToLower(r.Character.Class)+".json", strings.HasSuffix(key, "-dungeons"))
			if err != nil {
				t.Fatal(err)
			}
			if err = r.RouteModel.apply(w); err != nil {
				t.Fatal(err)
			}
			w.SetWandDetour(r.WandDetour)
			g, err := w.FollowGuide(r.Events, 20)
			if err != nil {
				t.Fatal(err)
			}
			if g.Level < 20 || len(g.Skipped) > 0 || math.Abs(g.Seconds-r.Seconds) > .11 {
				t.Fatalf("stale/invalid route: level %d seconds %.1f advertised %.1f skipped %v", g.Level, g.Seconds, r.Seconds, g.Skipped)
			}
			if g.Quests != r.Quests {
				t.Fatal("advertised quest count differs from replay")
			}
			if math.Abs(g.Travel+g.Combat+g.Talk-g.Seconds) > .21 {
				t.Fatal("time ledger does not add up")
			}
			done := map[int]bool{}
			runs := map[string]bool{}
			lastTime := 0.0
			lastXP := 0
			lastLevel := 1
			var actions []route.Action
			for _, s := range g.Steps {
				if s.Time < lastTime || s.XP < lastXP || math.IsNaN(s.Time+s.At.X+s.At.Y) || math.IsInf(s.Time+s.At.X+s.At.Y, 0) {
					t.Fatal("invalid timeline")
				}
				if s.Op == "turnin" {
					done[s.Quest] = true
				}
				if s.Op == "dungeon" {
					if runs[s.Dungeon] || !strings.HasSuffix(key, "-dungeons") {
						t.Fatal("unexpected or repeated dungeon")
					}
					runs[s.Dungeon] = true
					found := false
					for _, d := range w.Dungeons {
						if d.Key == s.Dungeon {
							found = true
							if lastLevel < d.Min || lastLevel > d.Max {
								t.Fatalf("dungeon %s started at invalid level %d", d.Key, lastLevel)
							}
						}
					}
					if !found {
						t.Fatal("unknown dungeon")
					}
				}
				lastTime, lastXP, lastLevel = s.Time, s.XP, s.Level
				actions = append(actions, s.Action)
			}
			for _, c := range w.ClassChains(20) {
				if c.Tier == "required" && !done[c.Goal.ID] {
					t.Fatalf("missing required class goal %d", c.Goal.ID)
				}
			}
			replay, err := w.Replay(actions)
			if err != nil || replay.XP != g.XP || math.Abs(replay.Seconds-g.Seconds) > .11 {
				t.Fatalf("action replay differs: %v", err)
			}
		})
	}
	if len(expected) != 0 {
		t.Fatal("missing roster files", expected)
	}
}
