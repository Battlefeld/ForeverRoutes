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

// A fix with xp 0 is a real value: the quest gives no XP in Forever, so foreverOnly must not estimate one for it.
func TestQuestFixZeroXP(t *testing.T) {
	fixes, err := readQuestFixes("../../data/sources/quest-fixes.json")
	if err != nil {
		t.Fatal(err)
	}
	b := &builder{stats: map[string]int{}, questFixes: fixes}
	b.out = append(b.out, dataset.Quest{ID: 2951, Level: 30, XP: 2450, Sources: map[string]string{"xp": "cmangos"}})
	for i := 0; i < 3; i++ {
		b.out = append(b.out, dataset.Quest{ID: 100 + i, Level: 30, XP: 2450, Sources: map[string]string{"xp": "wowhead_forever"}})
	}
	b.applyQuestFixes()
	b.foreverOnly()
	if b.out[0].XP != 0 {
		t.Fatalf("Sparklematic XP = %d, want 0", b.out[0].XP)
	}
}
