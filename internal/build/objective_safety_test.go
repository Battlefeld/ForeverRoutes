package build

import (
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/foreverchanges"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/polestar"
)

func TestEliteClientTypeIsPreserved(t *testing.T) {
	b := &builder{stats: map[string]int{}, pdb: &polestar.DB{Quests: map[int]*polestar.Quest{97003: {ID: 97003, Type: 1}}}}
	q := dataset.Quest{ID: 97003, Sources: map[string]string{}}
	b.applyPolestar(&q)
	if q.Type != "elite" {
		t.Fatalf("elite quest became %q", q.Type)
	}
}

func TestResolvedObjectiveCannotEraseEliteClassification(t *testing.T) {
	b := &builder{stats: map[string]int{}, fch: &foreverchanges.Overlay{Quests: map[int]foreverchanges.Quest{1: {ID: 1, Kind: "elite", URL: "https://foreverchanges.pro/quest/1"}}}}
	q := dataset.Quest{ID: 1, Sources: map[string]string{}, Objectives: []dataset.Objective{{Kind: "item", ID: 1, Count: 1, Drops: []dataset.Drop{{NPC: 2, Chance: 100}}}}}
	b.classify(&q)
	if q.Type != "elite" || q.Unsupported != "" {
		t.Fatalf("group classification confused with objective completeness: %+v", q)
	}
}

func TestMissingClientTypeCannotEraseKnownGroupRestriction(t *testing.T) {
	b := &builder{stats: map[string]int{}, pdb: &polestar.DB{Quests: map[int]*polestar.Quest{1: {ID: 1}}}}
	q := dataset.Quest{ID: 1, Type: "elite", Sources: map[string]string{}}
	b.applyPolestar(&q)
	if q.Type != "elite" {
		t.Fatal("missing client type erased known group restriction")
	}
}

func TestItemWaypointsDoNotProveAcquisition(t *testing.T) {
	for _, tc := range []struct {
		name string
		o    dataset.Objective
		safe bool
	}{
		{"waypoint", dataset.Objective{Kind: "item", ID: 1, Places: []dataset.Point{{Map: 1, X: 10}}}, false},
		{"ground", dataset.Objective{Kind: "item", ID: 1, Ground: true, Places: []dataset.Point{{Map: 1, X: 10}}}, true},
		{"drop", dataset.Objective{Kind: "item", ID: 1, Drops: []dataset.Drop{{NPC: 2, Unknown: true}}}, true},
		{"vendor", dataset.Objective{Kind: "item", ID: 1, Vendors: []int{2}}, true},
		{"dungeon", dataset.Objective{Kind: "item", ID: 1, Inside: true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := &builder{stats: map[string]int{}, out: []dataset.Quest{{ID: 1, Sources: map[string]string{}, Objectives: []dataset.Objective{tc.o}}}}
			b.objectiveSourceSafety()
			if (b.out[0].Unsupported == "") != tc.safe {
				t.Fatalf("safe=%v: %s", tc.safe, b.out[0].Unsupported)
			}
		})
	}
}

func TestClientCombatMetadataOverridesClassicReference(t *testing.T) {
	rank := 1
	b := &builder{stats: map[string]int{}, pdb: &polestar.DB{NPCs: map[int]*polestar.NPC{2: {ID: 2, Rank: &rank, LevelMin: 20, LevelMax: 20}}}}
	n := dataset.NPC{ID: 2, Rank: 0, MinLevel: 17, MaxLevel: 17, Spawns: []dataset.Point{{Map: 1}}}
	b.polestarNPC(&n, true)
	if n.Rank != 1 || n.MaxLevel != 20 || n.MinLevel != 20 {
		t.Fatalf("stale combat metadata: %+v", n)
	}
}
