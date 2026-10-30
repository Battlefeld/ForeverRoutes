package route

import (
	"encoding/json"
	"github.com/Battlefeld/ForeverRoutes/dataset"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewExistingRoster(t *testing.T) {
	if os.Getenv("FR_REVIEW_EXISTING") == "" {
		t.Skip("explicit diagnostic of legacy routes")
	}
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	walks, err := dataset.LoadWalks("../data/walks.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	files, _ := filepath.Glob("../data/routes/*.json")
	var rows []map[string]any
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var f struct {
			Character    Character         `json:"character"`
			Target       int               `json:"target"`
			Events       []GuideEvent      `json:"events"`
			Wand         float64           `json:"wandDetour"`
			Preparations *PreparationPlan  `json:"preparations"`
			Bag          *SleepingBagModel `json:"sleepingBagModel"`
		}
		if err = json.Unmarshal(raw, &f); err != nil {
			t.Fatal(path, err)
		}
		pf, err := os.Open("../data/profiles/" + strings.ToLower(f.Character.Class) + ".json")
		if err != nil {
			t.Fatal(err)
		}
		p, err := LoadTableProfile(pf)
		pf.Close()
		if err != nil {
			t.Fatal(err)
		}
		w, err := NewWorld(d, f.Character, Options{Profile: p, Walks: walks, Dungeons: strings.Contains(path, "-dungeons")})
		if err != nil {
			t.Fatal(err)
		}
		if f.Preparations != nil {
			if err = w.SetPreparations(*f.Preparations); err != nil {
				t.Fatal(err)
			}
		}
		if f.Bag != nil {
			if err = w.SetSleepingBagModel(*f.Bag); err != nil {
				t.Fatal(err)
			}
		}
		w.SetWandDetour(f.Wand)
		row := map[string]any{"file": filepath.Base(path), "character": f.Character}
		r, err := w.FollowGuide(f.Events, f.Target)
		if err != nil {
			row["error"] = err.Error()
		} else {
			var missing []int
			for _, c := range w.ClassChains(f.Target) {
				if c.Tier == "required" && !handsIn(r, c.Goal.ID) {
					missing = append(missing, c.Goal.ID)
				}
			}
			row["seconds"], row["level"], row["skipped"], row["missed"], row["missingClassGoals"] = r.Seconds, r.Level, r.Skipped, r.Missed, missing
		}
		rows = append(rows, row)
	}
	b, _ := json.MarshalIndent(rows, "", " ")
	if err = os.WriteFile(os.Getenv("FR_REVIEW_EXISTING"), b, 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("reviewed %d routes", len(rows))
}
