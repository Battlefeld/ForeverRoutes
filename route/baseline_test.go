package route

import (
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// TestBaselineCharacters plans the two reference characters on the shipped dataset: a Horde Orc Warlock and an
// Alliance Human Mage from level 1 to 30. It guards that the data and the engine keep producing complete, replayable
// routes; the hours depend on the uncalibrated default kill profile and are only checked for a sane range.
func TestBaselineCharacters(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: plans two full routes")
	}
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Skip("dataset not available:", err)
	}
	for _, c := range []Character{{Race: "Orc", Class: "Warlock"}, {Race: "Human", Class: "Mage"}} {
		t.Run(c.Race+c.Class, func(t *testing.T) {
			t.Parallel()
			w, err := NewWorld(d, c, Options{})
			if err != nil {
				t.Fatal(err)
			}
			res, err := w.Plan(PlanOptions{TargetLevel: 30, Restarts: 1, Lookahead: 3, Depth: 2})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%s %s: level %d in %.1f h (travel %.1f h, combat %.1f h), %d quests, %d usable quests",
				c.Race, c.Class, res.Level, res.Seconds/3600, res.Travel/3600, res.Combat/3600, res.Quests, len(w.Quests))
			if res.Level < 30 {
				t.Errorf("level %d", res.Level)
			}
			if len(w.Quests) < 300 {
				t.Errorf("only %d usable quests: data lost?", len(w.Quests))
			}
			if res.Quests < 25 {
				t.Errorf("only %d quests on the route", res.Quests)
			}
			if h := res.Seconds / 3600; h < 8 || h > 40 {
				t.Errorf("%.1f h is outside the sane range", h)
			}
		})
	}
}
