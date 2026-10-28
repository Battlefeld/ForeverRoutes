package build

import (
	"github.com/Battlefeld/ForeverRoutes/dataset"
	"testing"
)

func TestSourcedQuestMinimumSurvivesRebuild(t *testing.T) {
	fixes, err := readQuestFixes("../../data/sources/quest-fixes.json")
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{stats: map[string]int{}, questFixes: fixes, out: []dataset.Quest{{ID: 971, Level: 23, MinLevel: 18, XP: 6550, Sources: map[string]string{}}}}
	b.applyQuestFixes()
	q := b.out[0]
	if q.MinLevel != 10 || q.Level != 23 || q.XP != 6550 || q.Sources["minLevel"] != "https://foreverchanges.pro/quest/971" {
		t.Fatalf("sourced minimum did not survive rebuild: %+v", q)
	}
}
