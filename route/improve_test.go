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
