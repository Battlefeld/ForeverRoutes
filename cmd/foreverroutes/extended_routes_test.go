package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// The routes continued past 20 (data/routes-ext) replay to their target under the current data and model, like the
// shipped routes (the name keeps it in the shipped-route check of the workflows: after a data fix the update
// continues them again).
func TestShippedRoutesMatchCurrentModelExtended(t *testing.T) {
	files, _ := filepath.Glob("../../data/routes-ext/*.json")
	if len(files) == 0 {
		t.Skip("no continued routes")
	}
	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			r, err := readRoute(f)
			if err != nil {
				t.Fatal(err)
			}
			c := r.Character
			prof := filepath.Join("../../data/profiles", strings.ToLower(c.Class)+".json")
			if !fileExists(prof) {
				prof = ""
			}
			w, err := loadWorld("../../data/routes-data.json.gz", c.Race, c.Class, c.Faction, prof, hasDungeons(r.Events))
			if err != nil {
				t.Fatal(err)
			}
			if err := r.RouteModel.apply(w); err != nil {
				t.Fatal(err)
			}
			w.SetWandDetour(r.WandDetour)
			g, err := w.FollowGuide(r.Events, r.Target)
			if err != nil {
				t.Fatal(err)
			}
			if g.Level < r.Target || len(g.Skipped) > 0 {
				t.Errorf("stale/invalid route: level %d of %d, skipped %v %+v", g.Level, r.Target, g.Skipped, g.Missed)
			}
		})
	}
}
