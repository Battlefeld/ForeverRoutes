package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/fallback"
	"github.com/Battlefeld/ForeverRoutes/route"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// Opt-in full rebuild, using frozen independent inputs rather than last run's outputs.
// FR_ROSTER_REBUILD=/absolute/output/directory go test ./cmd/foreverroutes -run '^TestRebuildRoster$' -timeout 60m -v
func TestRebuildRoster(t *testing.T) {
	dir := os.Getenv("FR_ROSTER_REBUILD")
	if dir == "" {
		t.Skip("opt-in complete roster recalculation")
	}
	budget := 12 * time.Second
	if s := os.Getenv("FR_ROSTER_BUDGET"); s != "" {
		var err error
		budget, err = time.ParseDuration(s)
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("../../data/audits/roster-inputs.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	var inputs struct {
		Routes []struct {
			Path      string          `json:"path"`
			Character route.Character `json:"character"`
			route.OptimizationSeed
		}
	}
	err = json.NewDecoder(z).Decode(&inputs)
	z.Close()
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	cat, err := fallback.Load("../../data/fallbacks/rxp.json.gz")
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
	var mu sync.Mutex
	var rows []map[string]any
	write := func(path string, v any) {
		b, err := json.MarshalIndent(v, "", " ")
		if err != nil {
			t.Error(err)
			return
		}
		if err = os.WriteFile(filepath.Join(dir, path), append(b, '\n'), 0644); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(func() {
		slices.SortFunc(rows, func(a, b map[string]any) int {
			return strings.Compare(a["key"].(string)+a["scenario"].(string), b["key"].(string)+b["scenario"].(string))
		})
		write("summary.json", map[string]any{"schema": "foreverroutes.roster-review.v1", "modelRevision": route.TimelineModelRevision, "target": 20, "contexts": len(route.SupportedCharacters()), "budgetPerSearch": budget.String(), "results": rows})
	})
	for _, ch := range route.SupportedCharacters() {
		t.Run(ch.RouteKey(), func(t *testing.T) {
			t.Parallel()
			var seeds []route.OptimizationSeed
			for _, r := range inputs.Routes {
				if route.SameCharacter(r.Character, ch) {
					seeds = append(seeds, r.OptimizationSeed)
				}
			}
			if len(seeds) != 2 {
				t.Fatalf("expected independent solo/dungeon inputs, got %d", len(seeds))
			}
			if ch.Race == "Undead" && ch.Class == "Warlock" {
				for _, pattern := range []string{"../../data/audits/timeline-v2-winners/*.json", "../../data/audits/no-wc-seeds/*.json"} {
					paths, err := filepath.Glob(pattern)
					if err != nil {
						t.Fatal(err)
					}
					for _, path := range paths {
						b, err := os.ReadFile(path)
						if err != nil {
							t.Fatal(err)
						}
						var s route.OptimizationSeed
						if err = json.Unmarshal(b, &s); err != nil {
							t.Fatal(err)
						}
						seeds = append(seeds, s)
					}
				}
			}
			rxp, err := cat.For(ch)
			if err != nil {
				t.Fatal(err)
			}
			pf, err := os.Open("../../data/profiles/" + strings.ToLower(ch.Class) + ".json")
			if err != nil {
				t.Fatal(err)
			}
			profile, err := route.LoadTableProfile(pf)
			pf.Close()
			if err != nil {
				t.Fatal(err)
			}
			crossGroupSeeds := slices.Clone(seeds)
			for _, group := range []bool{false, true} {
				label := "solo"
				if group {
					label = "dungeons"
				}
				makeWorld := func(policy route.PreparationConstraints) *route.World {
					w, err := route.NewWorld(d, ch, route.Options{Walks: walks, WalkTable: walkTable, Profile: profile, Dungeons: group, Preparations: policy})
					if err != nil {
						t.Fatal(err)
					}
					return w
				}
				pool := slices.Clone(crossGroupSeeds)
				pool = append(pool, route.OptimizationSeed{Events: rxp.Events})
				modes := []string{"no_wand", "bag"}
				if route.WandClass(ch.Class) {
					modes = append(modes, "wand", "wand_bag")
				}
				modes = append(modes, "auto")
				var bestSeconds = math.Inf(1)
				for _, mode := range modes {
					policy := route.PreparationConstraints{Camping: "auto", Wands: "off", SleepingBag: "off"}
					if strings.Contains(mode, "wand") && mode != "no_wand" {
						policy.Wands = "on"
					}
					if strings.Contains(mode, "bag") {
						policy.SleepingBag = "on"
					}
					if mode == "auto" {
						policy = route.PreparationConstraints{Camping: "auto", Wands: "auto", SleepingBag: "auto"}
					}
					w := makeWorld(policy)
					stats := &route.OptimizeStats{}
					start := time.Now()
					var e []route.GuideEvent
					var r *route.GuideResult
					var err error
					resumed := false
					if os.Getenv("FR_ROSTER_RESUME") != "" {
						path := filepath.Join(dir, ch.RouteKey()+"-"+label+"-"+mode+".json")
						if b, issue := os.ReadFile(path); issue == nil {
							var saved struct {
								route.OptimizationSeed
								Character   route.Character              `json:"character"`
								Seconds     float64                      `json:"seconds"`
								Constraints route.PreparationConstraints `json:"constraints"`
								Search      *route.OptimizeStats         `json:"search"`
								Error       string                       `json:"error"`
							}
							if json.Unmarshal(b, &saved) == nil && saved.Error == "" && saved.Constraints == policy && route.SameCharacter(ch, saved.Character) && saved.Preparations != nil {
								if err = w.SetPreparations(*saved.Preparations); err != nil {
									t.Fatal(err)
								}
								w.SetWandDetour(saved.WandDetour)
								if replay, issue := w.FollowGuide(saved.Events, 20); issue == nil && replay.Level >= 20 && len(replay.Skipped) == 0 && math.Abs(replay.Seconds-saved.Seconds) < .11 {
									e, r, resumed = saved.Events, replay, true
									if saved.Search != nil {
										stats = saved.Search
									}
								}
							}
						}
					}
					if !resumed {
						e, r, err = w.Optimize(pool[0].Events, route.OptimizeOptions{Target: 20, Seeds: pool[1:], Budget: budget, Seed: 20261027, Stats: stats})
					}
					row := map[string]any{"key": ch.RouteKey(), "character": ch, "scenario": label + "_" + mode, "searchSeconds": time.Since(start).Seconds(), "constraints": policy}
					row["resumedAndReverified"], row["modelRevision"] = resumed, route.TimelineModelRevision
					record := func() {
						mu.Lock()
						rows = append(rows, row)
						mu.Unlock()
						write(ch.RouteKey()+"-"+label+"-"+mode+".json", row)
					}
					if err != nil {
						row["error"] = err.Error()
						record()
						t.Errorf("%s %s: %v", label, mode, err)
						continue
					}
					// Publish an executable event list. Source files remain frozen independently.
					// Re-project actual actions until no stale/duplicate event remains.
					for pass := 0; pass < 4; pass++ {
						clean := route.EventsOf(r.Steps)
						next, err := w.FollowGuide(clean, 20)
						if err != nil || next.Level < 20 {
							break
						}
						e, r = clean, next
						if len(r.Skipped) == 0 {
							break
						}
					}
					{
						chosen := w.Preparations()
						detour := w.WandDetour()
						for _, candidate := range pool {
							if candidate.Preparations == nil {
								continue
							}
							p := *candidate.Preparations
							if policy.Wands == "off" && p.Wands || policy.Wands == "on" && !p.Wands || policy.SleepingBag == "off" && p.BagStacks != 0 {
								continue
							}
							if err := w.SetPreparations(*candidate.Preparations); err != nil {
								t.Fatal(err)
							}
							w.SetWandDetour(candidate.WandDetour)
							if replay, err := w.FollowGuide(candidate.Events, 20); err == nil && replay.Level >= 20 && len(replay.Skipped) == 0 && replay.Seconds < r.Seconds {
								bag := slices.ContainsFunc(replay.Steps, func(s route.Step) bool { return s.Op == "turnin" && s.Quest == 79976 })
								if policy.SleepingBag == "on" && !bag || policy.SleepingBag == "off" && bag || policy.Wands == "on" && !replay.CraftedWand {
									continue
								}
								good := true
								for _, c := range w.ClassChains(20) {
									if c.Tier == "required" && !slices.ContainsFunc(replay.Steps, func(s route.Step) bool { return s.Op == "turnin" && s.Quest == c.Goal.ID }) {
										good = false
									}
								}
								if !good {
									continue
								}
								row["compatibleSeedWinnerAdopted"] = true
								e, r = candidate.Events, replay
								chosen = w.Preparations()
								detour = w.WandDetour()
							}
						}
						if err := w.SetPreparations(chosen); err != nil {
							t.Fatal(err)
						}
						w.SetWandDetour(detour)
					}
					var acts []route.Action
					for _, step := range r.Steps {
						acts = append(acts, step.Action)
					}
					replay, err := w.Replay(acts)
					if err != nil || replay.Level < 20 || math.Abs(replay.Seconds-r.Seconds) > .11 {
						t.Errorf("%s %s: action replay mismatch: %v", label, mode, err)
						row["error"] = "action replay mismatch"
					}
					var goals []int
					for _, c := range w.ClassChains(20) {
						if c.Tier == "required" {
							goals = append(goals, c.Goal.ID)
							if !slices.ContainsFunc(r.Steps, func(s route.Step) bool { return s.Op == "turnin" && s.Quest == c.Goal.ID }) {
								t.Errorf("%s %s: missing required quest %d", label, mode, c.Goal.ID)
								row["error"] = "missing required class quest"
							}
						}
					}
					if len(r.Skipped) > 0 {
						t.Errorf("%s %s has skipped events: %v", label, mode, r.Skipped)
						row["error"] = "skipped events remain"
					}
					var runs []string
					var sleeps, wands int
					var bag bool
					prev := 0.0
					for _, step := range r.Steps {
						if step.Time < prev || math.IsNaN(step.Time) || math.IsInf(step.Time, 0) {
							t.Error("non-monotone/non-finite timeline")
						}
						prev = step.Time
						if step.Op == "dungeon" {
							runs = append(runs, step.Dungeon)
						}
						if step.Op == "sleep" {
							sleeps++
						}
						if step.Op == "wand" {
							wands++
						}
						if step.Op == "turnin" && step.Quest == 79976 {
							bag = true
						}
					}
					if !group && len(runs) > 0 || policy.Wands == "on" && wands == 0 || policy.Wands == "off" && wands > 0 || policy.SleepingBag == "on" && !bag || policy.SleepingBag == "off" && bag {
						t.Errorf("%s %s: selected option lost", label, mode)
						row["error"] = "selected option lost"
					}
					if mode == "auto" && r.Seconds > bestSeconds+.11 {
						t.Errorf("free search lost fixed-policy winner %.1f > %.1f", r.Seconds, bestSeconds)
						row["error"] = "free search worsened known feasible winner"
					}
					if r.Seconds < bestSeconds {
						bestSeconds = r.Seconds
					}
					prep := w.Preparations()
					pool = append(pool, route.OptimizationSeed{Events: e, Preparations: &prep, WandDetour: w.WandDetour()})
					row["seconds"], row["level"], row["quests"], row["events"], row["preparations"], row["wandDetour"] = r.Seconds, r.Level, r.Quests, e, prep, w.WandDetour()
					row["sleepingBagModel"], row["requiredClassGoals"], row["runs"], row["search"], row["replayVerified"], row["skipped"], row["sleepSteps"], row["wandSteps"], row["bagFinished"] = w.Opt.SleepingBag, goals, runs, stats, true, r.Skipped, sleeps, wands, bag
					if mode == "auto" {
						row["schema"], row["modelRevision"], row["target"], row["winner"], row["basis"], row["hours"], row["killProfile"] = "foreverroutes.route.v1", route.TimelineModelRevision, 20, "timeline", "ForeverRoutes; independent frozen inputs and ordinary RXP quest projection", map[string]float64{"optimized": r.Seconds / 3600}, r.Profile
						row["classQuests"] = w.ClassReportWithBudget(e, 20, 0)
						name := ch.RouteKey()
						if group {
							name += "-dungeons"
						}
						write(name+".json", row)
					}
					record()
					t.Logf("%s %s: %.1fs, %d quests, runs=%v", label, mode, r.Seconds, r.Quests, runs)
				}
				crossGroupSeeds = slices.Clone(pool)
				// Original RXP stays unchanged: all omissions are evidence, never silently called an exact guide run.
				rw := makeWorld(route.PreparationConstraints{Camping: "off", Wands: "off", SleepingBag: "off"})
				rr, err := rw.FollowGuide(rxp.Events, 20)
				row := map[string]any{"key": ch.RouteKey(), "character": ch, "scenario": label + "_rxp_projection", "projectionOnly": true, "chapters": rxp.Chapters}
				if err != nil {
					row["error"] = err.Error()
					t.Error(err)
				} else {
					row["seconds"], row["level"], row["skipped"], row["missed"] = rr.Seconds, rr.Level, rr.Skipped, rr.Missed
				}
				mu.Lock()
				rows = append(rows, row)
				mu.Unlock()
			}
		})
	}
	fmt.Println("All roster cases scheduled; each owns its World and outputs.")
}
