package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/route"
)

// Opt-in, between TestRebuildRoster (run with FR_ROSTER_TOLERATE) and TestFinalizeRosterEvidence:
// FR_ROSTER_REPAIR=<rebuild output> replays every optimized scenario and replaces each one that failed or does not
// replay with a verified route of the same character and group that keeps the scenario's options: the fastest such
// scenario of this run, else the scenario's route of the last complete audit if it still replays under the current
// model. A scenario with neither stays broken and fails here, so the update stops as before. Every replacement is
// listed in repairs.txt and marked in the row ("repairedFrom").
func TestRepairRosterScenarios(t *testing.T) {
	dir := os.Getenv("FR_ROSTER_REPAIR")
	if dir == "" {
		t.Skip("opt-in repair of a tolerant roster rebuild")
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
	profile := func(class string) *route.TableProfile {
		if p := profiles[class]; p != nil {
			return p
		}
		f, err := os.Open("../../data/profiles/" + strings.ToLower(class) + ".json")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		p, err := route.LoadTableProfile(f)
		if err != nil {
			t.Fatal(err)
		}
		profiles[class] = p
		return p
	}
	type replayed struct {
		world *route.World
		res   *route.GuideResult
	}
	// replay a row's events the way the final audit does; nil when the row is not a verified route
	replay := func(row map[string]any, scenario string, constraints route.PreparationConstraints) *replayed {
		b, err := json.Marshal(row)
		if err != nil {
			t.Fatal(err)
		}
		var saved struct {
			route.OptimizationSeed
			Character route.Character        `json:"character"`
			Bag       route.SleepingBagModel `json:"sleepingBagModel"`
			Seconds   float64                `json:"seconds"`
			Error     string                 `json:"error"`
		}
		if json.Unmarshal(b, &saved) != nil || saved.Error != "" || saved.Preparations == nil || len(saved.Events) == 0 {
			return nil
		}
		w, err := route.NewWorld(d, saved.Character, route.Options{Profile: profile(saved.Character.Class), Walks: walks, WalkTable: walkTable, Dungeons: strings.HasPrefix(scenario, "dungeons_"), Preparations: constraints, SleepingBag: saved.Bag})
		if err != nil || w.SetPreparations(*saved.Preparations) != nil {
			return nil
		}
		w.SetWandDetour(saved.WandDetour)
		r, err := w.FollowGuide(saved.Events, 20)
		if err != nil || r.Level < 20 || len(r.Skipped) > 0 || math.Abs(r.Seconds-saved.Seconds) > .11 {
			return nil
		}
		for _, c := range w.ClassChains(20) {
			if c.Tier == "required" && !handedIn(r, c.Goal.ID) {
				return nil
			}
		}
		return &replayed{w, r}
	}
	constraintsOf := func(row map[string]any) route.PreparationConstraints {
		var c route.PreparationConstraints
		b, _ := json.Marshal(row["constraints"])
		_ = json.Unmarshal(b, &c)
		return c
	}
	// whether a verified route keeps a scenario's options (wand crafted or not, sleeping bag or not, no run when solo)
	keeps := func(res *route.GuideResult, scenario string, c route.PreparationConstraints) bool {
		wands, runs := 0, 0
		for _, s := range res.Steps {
			if s.Op == "wand" {
				wands++
			}
			if s.Op == "dungeon" {
				runs++
			}
		}
		bag := handedIn(res, 79976)
		return !(c.Wands == "on" && wands == 0 || c.Wands == "off" && wands > 0 || c.SleepingBag == "on" && !bag || c.SleepingBag == "off" && bag || strings.HasPrefix(scenario, "solo_") && runs > 0)
	}
	group := func(row map[string]any) string {
		return row["key"].(string) + " " + strings.SplitN(row["scenario"].(string), "_", 2)[0]
	}
	previous := map[string]map[string]any{}
	if f, err := os.Open("../../data/audits/" + lastAudit(t) + ".json.gz"); err == nil {
		if z, err := gzip.NewReader(f); err == nil {
			var old struct {
				Results []map[string]any `json:"results"`
			}
			if json.NewDecoder(z).Decode(&old) == nil {
				for _, row := range old.Results {
					if row["projectionOnly"] != true {
						previous[row["key"].(string)+" "+row["scenario"].(string)] = row
					}
				}
			}
			z.Close()
		}
		f.Close()
	}
	valid := map[int]*replayed{}
	var broken []int
	for i, row := range rows {
		if row["projectionOnly"] == true {
			continue
		}
		if r := replay(row, row["scenario"].(string), constraintsOf(row)); r != nil {
			valid[i] = r
		} else {
			broken = append(broken, i)
		}
	}
	var repairs, unrepaired []string
	for _, i := range broken {
		row := rows[i]
		scenario := row["scenario"].(string)
		c := constraintsOf(row)
		var source map[string]any
		from := ""
		var best *replayed
		var order []int
		for j := range valid {
			order = append(order, j)
		}
		sort.Ints(order)
		for _, j := range order {
			r := valid[j]
			if group(rows[j]) != group(row) || !keeps(r.res, scenario, c) {
				continue
			}
			// the candidate's own world may allow options this scenario's world does not; replay it in this one
			if again := replay(rows[j], scenario, c); again == nil || !keeps(again.res, scenario, c) {
				continue
			} else if best == nil || again.res.Seconds < best.res.Seconds {
				best, source, from = again, rows[j], rows[j]["scenario"].(string)
			}
		}
		if best == nil {
			if old := previous[row["key"].(string)+" "+scenario]; old != nil {
				if r := replay(old, scenario, c); r != nil && keeps(r.res, scenario, c) {
					best, source, from = r, old, "previous audit"
				}
			}
		}
		if best == nil {
			unrepaired = append(unrepaired, fmt.Sprintf("%s %s: %v", row["key"], scenario, row["error"]))
			continue
		}
		for _, k := range []string{"events", "preparations", "wandDetour", "sleepingBagModel", "requiredClassGoals", "runs", "sleepSteps", "wandSteps", "bagFinished"} {
			if v, ok := source[k]; ok {
				row[k] = v
			}
		}
		reason := fmt.Sprint(row["error"])
		if reason == "<nil>" {
			reason = "does not replay under the final model"
		}
		delete(row, "error")
		row["seconds"], row["level"], row["quests"], row["skipped"], row["replayVerified"] = best.res.Seconds, best.res.Level, best.res.Quests, best.res.Skipped, true
		row["repairedFrom"], row["repairReason"] = from, reason
		if strings.HasSuffix(scenario, "_auto") {
			var events []route.GuideEvent
			b, _ := json.Marshal(row["events"])
			_ = json.Unmarshal(b, &events)
			row["hours"], row["killProfile"], row["classQuests"] = map[string]float64{"optimized": best.res.Seconds / 3600}, best.res.Profile, best.world.ClassReportWithBudget(events, 20, 0)
			name := row["key"].(string)
			if strings.HasPrefix(scenario, "dungeons_") {
				name += "-dungeons"
			}
			writeJSON(t, filepath.Join(dir, name+".json"), row)
		}
		writeJSON(t, filepath.Join(dir, row["key"].(string)+"-"+strings.Replace(scenario, "_", "-", 1)+".json"), row)
		repairs = append(repairs, fmt.Sprintf("%s %s: replaced by %s (%.1f s)", row["key"], scenario, from, best.res.Seconds))
	}
	payload, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	doc["results"] = payload
	writeJSON(t, filepath.Join(dir, "summary.json"), doc)
	report := strings.Join(repairs, "\n")
	if len(unrepaired) > 0 {
		report += "\nnot repairable:\n" + strings.Join(unrepaired, "\n")
	}
	if err = os.WriteFile(filepath.Join(dir, "repairs.txt"), []byte(report+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("%d scenarios verified, %d repaired", len(valid), len(repairs))
	for _, line := range repairs {
		t.Log(line)
	}
	if len(unrepaired) > 0 {
		t.Fatalf("scenarios without a verified replacement:\n%s", strings.Join(unrepaired, "\n"))
	}
	// many replacements mean something broke for everyone, not a rare bad search: then the update stops
	if limit := os.Getenv("FR_ROSTER_REPAIR_LIMIT"); limit != "" {
		var max int
		if _, err := fmt.Sscan(limit, &max); err == nil && len(repairs) > max {
			t.Fatalf("%d scenarios needed a replacement, more than the limit of %d", len(repairs), max)
		}
	}
}

func handedIn(r *route.GuideResult, quest int) bool {
	for _, s := range r.Steps {
		if s.Op == "turnin" && s.Quest == quest {
			return true
		}
	}
	return false
}

// lastAudit names the newest complete roster audit, roster-vNN.json.gz.
func lastAudit(t *testing.T) string {
	paths, err := filepath.Glob("../../data/audits/roster-v*.json.gz")
	if err != nil || len(paths) == 0 {
		return ""
	}
	sort.Strings(paths)
	return strings.TrimSuffix(filepath.Base(paths[len(paths)-1]), ".json.gz")
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, append(b, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
}
