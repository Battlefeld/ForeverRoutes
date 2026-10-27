package route

import (
	"crypto/sha256"
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

func TestUndeadQuestAuditBenchmark(t *testing.T) {
	scenario := os.Getenv("FR_BENCH_SCENARIO")
	if scenario == "" {
		t.Skip("explicit one-off benchmark")
	}
	dungeons := strings.Contains(scenario, "dungeon")
	wands := strings.Contains(scenario, "wand")
	bag := strings.Contains(scenario, "bag")
	wandClasses["Warlock"] = wands
	data, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	walks, err := dataset.LoadWalks("../data/walks.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	pf, err := os.Open("../data/profiles/warlock.json")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := LoadTableProfile(pf)
	pf.Close()
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWorld(data, Character{Race: "Undead", Class: "Warlock"}, Options{Dungeons: dungeons, Walks: walks, Profile: profile})
	if err != nil {
		t.Fatal(err)
	}
	var bagChain ClassChain
	for _, c := range w.ClassChains(20) {
		if c.Goal.ID == 79976 {
			bagChain = c
		}
	}
	if bagChain.Goal == nil {
		t.Fatal("sleeping bag chain unavailable")
	}
	forbidden := map[int]bool{}
	if !bag {
		for _, q := range bagChain.Quests {
			forbidden[q.ID] = true
			delete(w.ByID, q.ID)
		}
	}
	var bestE []GuideEvent
	var best *GuideResult
	bestWand := 0.0
	for _, name := range []string{"undead-warlock.json", "undead-warlock-dungeons.json"} {
		var src struct {
			Events []GuideEvent `json:"events"`
			Wand   float64      `json:"wandDetour"`
		}
		raw, err := os.ReadFile(filepath.Join("../data/routes", name))
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(raw, &src); err != nil {
			t.Fatal(err)
		}
		ev := without(src.Events, func(e GuideEvent) bool {
			return forbidden[e.Quest] || (!dungeons && (e.Op == "dungeon" || w.ByID[e.Quest] == nil))
		})
		if dungeons && Runs(ev) == 0 {
			continue
		}
		w.SetWandDetour(src.Wand)
		if wands {
			w.ChooseWandDetour(ev, 20)
		}
		if bag {
			ev = w.WeaveIn(ev, bagChain, 20)
			w.setClassOverride(79976, "on")
		}
		base, err := w.FollowGuide(ev, 20)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("CHECKPOINT %s seed=%s seconds=%.1f level=%d bag=%v\n", scenario, name, base.Seconds, base.Level, handsIn(base, 79976))
		if bag && !handsIn(base, 79976) {
			t.Fatal("sleeping bag seed did not finish")
		}
		e, r, err := w.Improve(ev, 20, 30*time.Second)
		if err != nil {
			t.Fatal(err)
		}
		if best == nil || r.Seconds < best.Seconds {
			bestE, best, bestWand = e, r, w.WandDetour()
		}
	}
	if best == nil {
		t.Fatal("no usable starting route")
	}
	baseline := best.Seconds
	w.SetWandDetour(bestWand)
	stats := &OptimizeStats{}
	e, r, err := w.Optimize(bestE, OptimizeOptions{Target: 20, Budget: 180 * time.Second, Seed: 20261027, Keep: func() []int {
		if bag {
			return []int{79976}
		}
		return nil
	}(), Stats: stats})
	if err != nil {
		t.Fatal(err)
	}
	bestE, best = e, r
	// Confirm every reported action with the unmodified replay engine.
	actions := make([]Action, len(best.Steps))
	dungeonKeys := []string{}
	wandSteps := 0
	for i, st := range best.Steps {
		actions[i] = st.Action
		if st.Op == "dungeon" {
			dungeonKeys = append(dungeonKeys, st.Dungeon)
		}
		if st.Op == "wand" {
			wandSteps++
		}
	}
	replay, err := w.Replay(actions)
	if err != nil {
		t.Fatal(err)
	}
	if replay.Level < 20 || math.Abs(replay.Seconds-best.Seconds) > 0.11 {
		t.Fatalf("replay mismatch: %v", replay)
	}
	if dungeons != (len(dungeonKeys) > 0) {
		t.Fatalf("wrong dungeon variant: %v", dungeonKeys)
	}
	if wands != (wandSteps > 0) {
		t.Fatalf("wrong wand variant: %d", wandSteps)
	}
	if bag != handsIn(best, 79976) {
		t.Fatalf("wrong sleeping bag variant")
	}
	required := map[int]string{}
	for _, c := range w.ClassChains(20) {
		if c.Tier == "required" {
			required[c.Goal.ID] = c.Reward
			if !handsIn(best, c.Goal.ID) {
				t.Fatalf("missing required chain %d %s", c.Goal.ID, c.Reward)
			}
		}
	}
	dataBytes, err := os.ReadFile("../data/routes-data.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]any{
		"datasetSHA256": fmt.Sprintf("%x", sha256.Sum256(dataBytes)),
		"scenario":      scenario, "baseCommit": "dd2c371f03d44b2d1bcdda7311219137c808430d",
		"character": w.Char, "target": 20, "dungeons": dungeons, "wands": wands, "sleepingBag": bag,
		"seconds": best.Seconds, "hours": best.Seconds / 3600, "beforeTimelineOptimization": baseline,
		"wandDetour": w.WandDetour(), "wandSteps": wandSteps, "dungeonRuns": dungeonKeys,
		"sleepingBagFinished": handsIn(best, 79976), "requiredChains": required,
		"replayVerified": true, "optimizerStats": stats, "events": bestE, "result": best,
	}
	dir := os.Getenv("FR_BENCH_OUTPUT")
	if err = os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.MarshalIndent(out, "", "  ")
	if err = os.WriteFile(filepath.Join(dir, scenario+".json"), raw, 0644); err != nil {
		t.Fatal(err)
	}
	fmt.Printf("BENCHMARK_RESULT %s seconds=%.1f hours=%.6f level=%d dungeonRuns=%v wandSteps=%d bag=%v replay=true\n", scenario, best.Seconds, best.Seconds/3600, best.Level, dungeonKeys, wandSteps, bag)
}
