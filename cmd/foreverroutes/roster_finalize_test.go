package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/route"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Revalidate every saved scenario against the final compiled model, including resumed searches.
// Refresh descriptive counts; a changed time, missing goal, skip or invalid option is a failure, never silently repaired.
func TestFinalizeRosterEvidence(t *testing.T) {
	dir := os.Getenv("FR_ROSTER_FINALIZE")
	if dir == "" {
		t.Skip("explicit final audit of complete matrix")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]json.RawMessage
	if err = json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	if err = json.Unmarshal(doc["results"], &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 516 {
		t.Fatalf("%d scenarios, expected 516", len(rows))
	}
	d, err := dataset.Load("../../data/routes-data.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	walks, err := dataset.LoadWalks("../../data/walks.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	profiles := map[string]*route.TableProfile{}
	count := 0
	for _, row := range rows {
		if row["projectionOnly"] == true {
			continue
		}
		b, err := json.Marshal(row)
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
		}
		if err = json.Unmarshal(b, &saved); err != nil {
			t.Fatal(err)
		}
		key := saved.Character.RouteKey() + " " + saved.Scenario
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
		w, err := route.NewWorld(d, saved.Character, route.Options{Profile: profile, Walks: walks, Dungeons: strings.HasPrefix(saved.Scenario, "dungeons_"), Preparations: saved.Constraints, SleepingBag: saved.Bag})
		if err != nil {
			t.Fatal(err)
		}
		if saved.Preparations == nil {
			t.Fatal(key, "missing policy")
		}
		if err = w.SetPreparations(*saved.Preparations); err != nil {
			t.Fatal(err)
		}
		w.SetWandDetour(saved.WandDetour)
		r, err := w.FollowGuide(saved.Events, 20)
		if err != nil {
			t.Fatal(key, err)
		}
		if r.Level < 20 || len(r.Skipped) > 0 || math.Abs(r.Seconds-saved.Seconds) > .11 {
			t.Fatalf("%s differs under final model: level %d, %.1f vs %.1f, skipped %v", key, r.Level, r.Seconds, saved.Seconds, r.Skipped)
		}
		var actions []route.Action
		done := map[int]bool{}
		runs := 0
		wands := 0
		prevTime := 0.
		prevXP := 0
		for _, step := range r.Steps {
			actions = append(actions, step.Action)
			if step.Op == "turnin" {
				done[step.Quest] = true
			}
			if step.Op == "dungeon" {
				runs++
			}
			if step.Op == "wand" {
				wands++
			}
			if step.Time < prevTime || step.XP < prevXP {
				t.Fatal(key, "nonmonotone timeline")
			}
			prevTime, prevXP = step.Time, step.XP
		}
		for _, c := range w.ClassChains(20) {
			if c.Tier == "required" && !done[c.Goal.ID] {
				t.Fatal(key, "missing required goal", c.Goal.ID)
			}
		}
		if saved.Constraints.Wands == "on" && wands == 0 || saved.Constraints.Wands == "off" && wands > 0 || saved.Constraints.SleepingBag == "on" && !done[79976] || saved.Constraints.SleepingBag == "off" && done[79976] || strings.HasPrefix(saved.Scenario, "solo_") && runs > 0 {
			t.Fatal(key, "selected option mismatch")
		}
		replay, err := w.Replay(actions)
		if err != nil || replay.XP != r.XP || math.Abs(replay.Seconds-r.Seconds) > .11 {
			t.Fatal(key, "action replay mismatch", err)
		}
		if math.Abs(r.Travel+r.Combat+r.Talk-r.Seconds) > .21 {
			t.Fatal(key, "time ledger mismatch")
		}
		row["quests"], row["finalModelReplayVerified"], row["finalXP"], row["actionCount"], row["modelRevision"] = r.Quests, true, r.XP, len(actions), route.TimelineModelRevision
		if strings.HasSuffix(saved.Scenario, "_auto") {
			name := saved.Character.RouteKey()
			if strings.HasPrefix(saved.Scenario, "dungeons_") {
				name += "-dungeons"
			}
			out, err := json.MarshalIndent(row, "", " ")
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(dir, name+".json"), append(out, '\n'), 0644); err != nil {
				t.Fatal(err)
			}
		}
		count++
	}
	if count != 404 {
		t.Fatal("missing optimized scenarios", count)
	}
	best := map[string]float64{}
	auto := map[string]float64{}
	for _, row := range rows {
		if row["projectionOnly"] == true {
			continue
		}
		key := row["key"].(string)
		scenario := row["scenario"].(string)
		group := "solo"
		if strings.HasPrefix(scenario, "dungeons_") {
			group = "dungeons"
		}
		k := key + " " + group
		seconds := row["seconds"].(float64)
		if best[k] == 0 || seconds < best[k] {
			best[k] = seconds
		}
		if strings.HasSuffix(scenario, "_auto") {
			auto[k] = seconds
		}
	}
	for k, seconds := range auto {
		if seconds > best[k]+.11 {
			t.Fatal(k, "free choice discarded a known faster fixed-policy result")
		}
	}
	for _, c := range route.SupportedCharacters() {
		key := c.RouteKey()
		if auto[key+" dungeons"] > auto[key+" solo"]+.11 {
			t.Fatal(key, "allowing optional dungeons discarded the faster solo incumbent")
		}
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	doc["results"] = payload
	bytes, err := os.ReadFile("../../data/routes-data.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(bytes)
	doc["datasetSHA256"], _ = json.Marshal(fmt.Sprintf("%x", sum))
	doc["finalModelReplayCount"], _ = json.Marshal(count)
	final, err := json.MarshalIndent(doc, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "summary.json"), append(final, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("Final model: all %d optimized scenarios reverified with exact action replay and time ledger", count)
}
