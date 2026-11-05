package route

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// The live window re-orders the next quest steps from a live state that left the route's order, never slower and
// always a plan that replays.
func TestImproveLiveFromDeviatedState(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: plans on the shipped dataset")
	}
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Skip("dataset not available:", err)
	}
	raw, err := os.ReadFile("../data/routes/troll-warlock.json")
	if err != nil {
		t.Skip("route not available:", err)
	}
	var file struct {
		Events []GuideEvent `json:"events"`
	}
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	w, err := NewWorld(d, Character{Race: "Troll", Class: "Warlock"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	full, err := w.FollowGuide(file.Events, 20)
	if err != nil {
		t.Fatal(err)
	}
	gains := 0
	for _, cut := range []int{30, 60, 90, 120} {
		// play the route up to cut, but leave out every third hand-in: the player did those quests later
		s := w.NewState()
		var acts []Action
		for i, st := range full.Steps[:cut] {
			if st.Op == "turnin" && i%3 == 0 {
				continue
			}
			if w.Apply(s, st.Action) == nil {
				acts = append(acts, st.Action)
			}
		}
		live, err := w.FollowGuideFrom(s, file.Events, 20, 3600)
		if err != nil {
			t.Fatal(err)
		}
		t0 := time.Now()
		better, changed := w.ImproveLive(s, live.Result, 3*time.Second)
		took := time.Since(t0)
		if better.Seconds > live.Seconds+0.01 {
			t.Fatalf("cut %d: slower %.0f > %.0f", cut, better.Seconds, live.Seconds)
		}
		if took > 5*time.Second {
			t.Errorf("cut %d: took %s", cut, took)
		}
		if changed {
			gains++
			check := make([]Action, len(better.Steps))
			for i, st := range better.Steps {
				check[i] = st.Action
			}
			if _, err := w.replayFrom(s, check); err != nil {
				t.Fatalf("cut %d: improved plan does not replay: %v", cut, err)
			}
		}
		t.Logf("cut %d: %.0f s -> %.0f s (%s, changed %v)", cut, live.Seconds, better.Seconds, took.Round(time.Millisecond), changed)
	}
	_ = gains
}
