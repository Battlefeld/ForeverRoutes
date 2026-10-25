package route

import (
	"testing"
	"time"
)

// Improve never returns a guide slower than the one it starts from, and puts in a quest the guide passes by when
// that is faster than grinding.
func TestImproveNeverSlower(t *testing.T) {
	w := tinyWorld(t)
	guide := []GuideEvent{{Op: "accept", Quest: 1}, {Op: "complete", Quest: 1}, {Op: "turnin", Quest: 1}}
	start, err := w.FollowGuide(guide, 3)
	if err != nil {
		t.Fatal(err)
	}
	events, res, err := w.Improve(guide, 3, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if res.Seconds > start.Seconds || res.Level < 3 {
		t.Fatalf("improved %.0f s (level %d), guide %.0f s", res.Seconds, res.Level, start.Seconds)
	}
	again, _ := w.FollowGuide(events, 3)
	if again.Seconds != res.Seconds {
		t.Errorf("the improved guide walks in %.0f s, reported %.0f s", again.Seconds, res.Seconds)
	}
	if len(EventsOf(start.Steps)) == 0 {
		t.Error("a walked guide gives no steps back")
	}
}

// The planner's grinding stays in the guide steps as a level step; back-to-back grinds are one step.
func TestEventsOfKeepsGrinding(t *testing.T) {
	steps := []Step{
		{Action: Action{Op: "accept", Quest: 1}, Level: 3},
		{Action: Action{Op: "grind"}, Level: 3},
		{Action: Action{Op: "grind"}, Level: 4},
		{Action: Action{Op: "work", Quest: 1}, Level: 4},
		{Action: Action{Op: "work", Quest: 1}, Level: 4},
		{Action: Action{Op: "turnin", Quest: 1}, Level: 5},
	}
	got := EventsOf(steps)
	want := []GuideEvent{{Op: "accept", Quest: 1}, {Op: "level", Level: 4}, {Op: "complete", Quest: 1}, {Op: "turnin", Quest: 1}}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
