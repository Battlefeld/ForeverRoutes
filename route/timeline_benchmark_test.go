package route

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Opt-in integration proof on the shipped data. Each scenario owns its World and its checkpoint.
func TestTimelineShippedBenchmark(t *testing.T) {
	scenario := os.Getenv("FR_TIMELINE_SCENARIO")
	if scenario == "" {
		t.Skip("opt-in level-20 timeline benchmark")
	}
	modes := map[string]PreparationConstraints{
		"auto":     {Camping: "auto", Wands: "auto", SleepingBag: "auto"},
		"no_wand":  {Camping: "auto", Wands: "off", SleepingBag: "off"},
		"wand":     {Camping: "auto", Wands: "on", SleepingBag: "off"},
		"bag":      {Camping: "auto", Wands: "off", SleepingBag: "on"},
		"wand_bag": {Camping: "auto", Wands: "on", SleepingBag: "on"},
	}
	name := strings.TrimSuffix(scenario, "_1h")
	policy, ok := modes[name]
	if !ok {
		t.Fatal("unknown scenario")
	}
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	walks, err := dataset.LoadWalks("../data/walks.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	walkTable, err := dataset.LoadWalkTable("../data/walktable.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("../data/profiles/warlock.json")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := LoadTableProfile(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	opt := Options{Dungeons: true, Walks: walks, WalkTable: walkTable, Profile: profile, Preparations: policy}
	if strings.HasSuffix(scenario, "_1h") {
		opt.SleepingBag = DefaultSleepingBagModel()
		opt.SleepingBag.Duration = 3600
	}
	var seeds []OptimizationSeed
	for _, path := range []string{"../data/routes/undead-warlock.json", "../data/routes/undead-warlock-dungeons.json", "../data/audits/no-wc-seeds/dungeon.json", "../data/audits/no-wc-seeds/dungeon_wand.json", "../data/audits/no-wc-seeds/dungeon_bag.json", "../data/audits/no-wc-seeds/dungeon_wand_bag.json"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var s OptimizationSeed
		if err = json.Unmarshal(raw, &s); err != nil {
			t.Fatal(err)
		}
		seeds = append(seeds, s)
	}
	ref, err := NewWorld(d, Character{Race: "Undead", Class: "Warlock"}, opt)
	if err != nil {
		t.Fatal(err)
	}
	baselineEvents, baseline, err := ref.Optimize(seeds[0].Events, OptimizeOptions{Target: 20, Seeds: seeds[1:]})
	if err != nil {
		t.Fatal(err)
	}
	plan := ref.Preparations()
	seeds = append(seeds, OptimizationSeed{baselineEvents, &plan, ref.WandDetour()})
	w, err := NewWorld(d, ref.Char, opt)
	if err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("FR_TIMELINE_OUTPUT")
	if dir == "" {
		t.Fatal("FR_TIMELINE_OUTPUT required")
	}
	if err = os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	stats := &OptimizeStats{}
	events, r, err := w.Optimize(seeds[0].Events, OptimizeOptions{Target: 20, Seeds: seeds[1:], Budget: 180 * time.Second, Seed: 20261027, Stats: stats, Checkpoint: filepath.Join(dir, scenario+".best.json")})
	if err != nil {
		t.Fatal(err)
	}
	if r.Level < 20 || r.Seconds > baseline.Seconds+.11 {
		t.Fatalf("incumbent worsened: %.1f > %.1f", r.Seconds, baseline.Seconds)
	}
	var acts []Action
	wandSteps, restSteps := 0, 0
	for _, s := range r.Steps {
		acts = append(acts, s.Action)
		if s.Op == "wand" {
			wandSteps++
		}
		if s.Op == "sleep" {
			restSteps++
		}
	}
	replay, err := w.Replay(acts)
	if err != nil {
		t.Fatal(err)
	}
	if replay.Level < 20 || math.Abs(replay.Seconds-r.Seconds) > .11 {
		t.Fatal("actual-action replay differs")
	}
	if policy.Wands == "on" && wandSteps == 0 || policy.Wands == "off" && wandSteps != 0 {
		t.Fatal("wand choice lost")
	}
	if policy.SleepingBag == "on" && !handsIn(r, 79976) || policy.SleepingBag == "off" && handsIn(r, 79976) {
		t.Fatal("bag choice lost")
	}
	for _, c := range w.ClassChains(20) {
		if c.Tier == "required" && !handsIn(r, c.Goal.ID) {
			t.Fatalf("missing class chain %d", c.Goal.ID)
		}
	}
	beforeResume := r.Seconds
	resumeStats := &OptimizeStats{}
	_, resumed, err := w.Optimize(events, OptimizeOptions{Target: 20, Seed: 20261027, Stats: resumeStats, Checkpoint: filepath.Join(dir, scenario+".best.json")})
	if err != nil || !resumeStats.CheckpointLoaded || resumed.Seconds > beforeResume+.11 {
		t.Fatalf("resume worsened incumbent: %v", err)
	}
	out := map[string]any{"scenario": scenario, "modelRevision": timelineModelRevision, "character": w.Char, "target": 20, "baselineSeconds": baseline.Seconds, "seconds": r.Seconds, "resumedSeconds": resumed.Seconds, "preparations": r.Preparations, "sleepingBagModel": w.Opt.SleepingBag, "dungeonRuns": actualDungeonKeys(r), "wandSteps": wandSteps, "sleepSteps": restSteps, "sleepingBagFinished": handsIn(r, 79976), "replayVerified": true, "checkpointResumeVerified": true, "feedbackActive": w.Opt.Feedback.Active(), "search": stats, "events": events, "wandDetour": w.WandDetour(), "result": r}
	raw, err := json.MarshalIndent(out, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, scenario+".json"), raw, 0644); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("TIMELINE %s baseline=%.1f best=%.1f resume=%.1f runs=%v plan=%+v\n", scenario, baseline.Seconds, r.Seconds, resumed.Seconds, actualDungeonKeys(r), r.Preparations)
}

// Pool feasible winners under the free constraints. A free search must retain a known compatible fixed-policy winner.
// This also isolates the duration assumption by replaying identical events, rather than comparing different searches.
func TestTimelinePoolAndDurationSensitivity(t *testing.T) {
	dir := os.Getenv("FR_TIMELINE_POOL")
	if dir == "" {
		t.Skip("opt-in pooling of completed scenario results")
	}
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	walks, err := dataset.LoadWalks("../data/walks.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	walkTable, err := dataset.LoadWalkTable("../data/walktable.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open("../data/profiles/warlock.json")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := LoadTableProfile(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	type saved struct {
		Events       []GuideEvent    `json:"events"`
		Preparations PreparationPlan `json:"preparations"`
		WandDetour   float64         `json:"wandDetour"`
		Seconds      float64         `json:"seconds"`
	}
	var seeds []OptimizationSeed
	bestKnown := math.Inf(1)
	independentAuto := 0.0
	for _, name := range []string{"auto", "no_wand", "wand", "bag", "wand_bag"} {
		raw, err := os.ReadFile(filepath.Join(dir, name+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var s saved
		if err = json.Unmarshal(raw, &s); err != nil {
			t.Fatal(err)
		}
		p := s.Preparations
		seeds = append(seeds, OptimizationSeed{s.Events, &p, s.WandDetour})
		bestKnown = math.Min(bestKnown, s.Seconds)
		if name == "auto" {
			independentAuto = s.Seconds
		}
	}
	opt := Options{Dungeons: true, Walks: walks, WalkTable: walkTable, Profile: profile, Preparations: PreparationConstraints{Camping: "auto", Wands: "auto", SleepingBag: "auto"}}
	w, err := NewWorld(d, Character{Race: "Undead", Class: "Warlock"}, opt)
	if err != nil {
		t.Fatal(err)
	}
	w.SetPreparations(*seeds[0].Preparations)
	w.SetWandDetour(seeds[0].WandDetour)
	stats := &OptimizeStats{}
	events, r, err := w.Optimize(seeds[0].Events, OptimizeOptions{Target: 20, Seeds: seeds[1:], Stats: stats, Checkpoint: filepath.Join(dir, "auto.best.json")})
	if err != nil {
		t.Fatal(err)
	}
	if r.Seconds > bestKnown+.11 {
		t.Fatalf("free constraints lost a known feasible winner: %.1f > %.1f", r.Seconds, bestKnown)
	}
	var actions []Action
	for _, s := range r.Steps {
		actions = append(actions, s.Action)
	}
	replay, err := w.Replay(actions)
	if err != nil || math.Abs(replay.Seconds-r.Seconds) > .11 {
		t.Fatal("pooled winner failed replay")
	}
	pooled := map[string]any{"schema": "foreverroutes.timeline-pooled.v1", "modelRevision": timelineModelRevision, "scenario": "auto_pooled", "independentAutoSeconds": independentAuto, "bestKnownFixedSeconds": bestKnown, "seconds": r.Seconds, "target": 20, "character": w.Char, "preparations": w.Preparations(), "sleepingBagModel": w.Opt.SleepingBag, "events": events, "wandDetour": w.WandDetour(), "dungeonRuns": actualDungeonKeys(r), "replayVerified": true, "feedbackActive": false, "search": stats, "result": r}
	raw, err := json.MarshalIndent(pooled, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "auto_pooled.json"), raw, 0644); err != nil {
		t.Fatal(err)
	}
	var sensitivity []map[string]any
	for _, name := range []string{"auto", "bag", "wand_bag"} {
		b, err := os.ReadFile(filepath.Join(dir, name+".json"))
		if err != nil {
			t.Fatal(err)
		}
		var s saved
		if err = json.Unmarshal(b, &s); err != nil {
			t.Fatal(err)
		}
		row := map[string]any{"scenario": name, "method": "identical events, preparations and wand detour; only buff duration changes"}
		for _, duration := range []float64{3600, 7200} {
			p := opt
			p.SleepingBag = DefaultSleepingBagModel()
			p.SleepingBag.Duration = duration
			ww, err := NewWorld(d, w.Char, p)
			if err != nil {
				t.Fatal(err)
			}
			ww.SetPreparations(s.Preparations)
			ww.SetWandDetour(s.WandDetour)
			rr, err := ww.FollowGuide(s.Events, 20)
			if err != nil {
				t.Fatal(err)
			}
			var acts []Action
			for _, step := range rr.Steps {
				acts = append(acts, step.Action)
			}
			proof, err := ww.Replay(acts)
			if err != nil || proof.Level < 20 || math.Abs(proof.Seconds-rr.Seconds) > .11 {
				t.Fatal("duration sensitivity failed replay")
			}
			row[fmt.Sprintf("seconds_%dh", int(duration/3600))] = rr.Seconds
		}
		sensitivity = append(sensitivity, row)
	}
	b, err := json.MarshalIndent(sensitivity, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "duration-sensitivity.json"), b, 0644); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("POOLED auto %.1f <= known %.1f; duration sensitivity replayed\n", r.Seconds, bestKnown)
}
