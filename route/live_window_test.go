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

// The player's measured kills set the pace of every fight: twice as slow as the model, every kill takes longer.
func TestMeasuredPace(t *testing.T) {
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Skip("dataset not available:", err)
	}
	w, err := NewWorld(d, Character{Race: "Troll", Class: "Shaman"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	n := w.NPC[3125] // Clattering Scorpid
	if n == nil {
		t.Skip("creature not in the dataset")
	}
	s := w.NewState()
	s.XP = 2000
	site := Site{Center: n.Spawns[0], Spawns: 1}
	before := w.mobTime(s, n, site)
	f, r := w.Opt.Profile.KillSeconds(w.Char, 5, n)
	fb := CombatFeedback{Schema: "foreverroutes.combat-feedback.v1", Source: "measured", Character: w.Char,
		Kills: []KillObservation{{NPC: n.ID, PlayerLevel: 5, FightSeconds: 2 * f, RestSeconds: 2*r + 1, Samples: 6}}}
	if err := w.AcceptFeedback(fb); err != nil {
		t.Fatal(err)
	}
	if pf, _ := w.Pace(); pf < 1.9 || pf > 2.1 {
		t.Fatalf("pace %.2f, want 2", pf)
	}
	other := w.NPC[3098] // Mottled Boar: not measured, scaled by the pace
	if other != nil {
		of, _ := w.Opt.Profile.KillSeconds(w.Char, w.Level(s), other)
		if got := w.mobTime(s, other, Site{Center: other.Spawns[0], Spawns: 1}); got < 1.8*of {
			t.Fatalf("unmeasured creature not scaled: %.1f s for a %.1f s model fight", got, of)
		}
	}
	if after := w.mobTime(s, n, site); after <= before {
		t.Fatalf("measured creature not slower: %.1f -> %.1f", before, after)
	}
}
