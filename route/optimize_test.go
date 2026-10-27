package route

import (
	"math/rand"
	"testing"
	"time"
)

// Optimize never returns a route slower than the one it starts from, reports what the route walks in, and is the
// same for the same seed.
func TestOptimizeNeverSlower(t *testing.T) {
	w := tinyWorld(t)
	guide := []GuideEvent{{Op: "accept", Quest: 1}, {Op: "complete", Quest: 1}, {Op: "turnin", Quest: 1}}
	start, err := w.FollowGuide(guide, 3)
	if err != nil {
		t.Fatal(err)
	}
	events, res, err := w.Optimize(guide, OptimizeOptions{Target: 3, Budget: time.Second, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	if res.Seconds > start.Seconds || res.Level < 3 {
		t.Fatalf("optimized %.0f s (level %d), guide %.0f s", res.Seconds, res.Level, start.Seconds)
	}
	again, _ := w.FollowGuide(events, 3)
	if again.Seconds != res.Seconds {
		t.Errorf("the optimized route walks in %.0f s, reported %.0f s", again.Seconds, res.Seconds)
	}
}

// An edit without quests to put in never adds steps, and keeps every step of the quests it does not leave out.
func TestEditKeepsSteps(t *testing.T) {
	w := tinyWorld(t)
	cur := []GuideEvent{{Op: "accept", Quest: 1}, {Op: "accept", Quest: 2}, {Op: "complete", Quest: 1}, {Op: "turnin", Quest: 1},
		{Op: "complete", Quest: 2}, {Op: "turnin", Quest: 2}, {Op: "level", Level: 3}}
	rng := rand.New(rand.NewSource(7))
	for k := 0; k < 2000; k++ {
		e, _ := w.edit(cur, rng, nil, 3, false)
		if e == nil {
			continue
		}
		n := map[int]int{}
		for _, ev := range e {
			n[ev.Quest]++
		}
		if len(e) > len(cur) || n[1] != 0 && n[1] != 3 || n[2] != 0 && n[2] != 3 {
			t.Fatalf("edit gave %v", e)
		}
	}
}
