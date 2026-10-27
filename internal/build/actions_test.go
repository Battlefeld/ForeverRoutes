package build

import (
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/guides"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/polestar"
)

func TestClientObjectivesReplaceStaleClassicButPreserveObserved(t *testing.T) {
	for _, observed := range []bool{false, true} {
		b := &builder{stats: map[string]int{}, npcWanted: map[int]bool{}, w: &world{}, pdb: &polestar.DB{
			Quests:     map[int]*polestar.Quest{5727: {ID: 5727, Level: 12, MinLevel: 9}},
			Objectives: map[int][]polestar.Objective{5727: {{N: 1, Kind: "kill", Target: 3216, Count: 1}}},
		}}
		source := "cmangos"
		if observed {
			source = "observed"
		}
		q := dataset.Quest{ID: 5727, Sources: map[string]string{"objectives": source}, Objectives: []dataset.Objective{{Kind: "kill", ID: 999, Count: 3}}}
		b.applyPolestar(&q)
		want := 3216
		if observed {
			want = 999
		}
		if len(q.Objectives) != 1 || q.Objectives[0].ID != want {
			t.Fatalf("observed=%v: %+v", observed, q.Objectives)
		}
	}
}

func TestMissingObjectiveCountIsNotInvented(t *testing.T) {
	b := &builder{stats: map[string]int{}, npcWanted: map[int]bool{}, w: &world{}, pdb: &polestar.DB{
		Objectives: map[int][]polestar.Objective{1: {{N: 1, Kind: "kill", Target: 100}}},
	}}
	q := dataset.Quest{ID: 1, Sources: map[string]string{}}
	b.polestarObjectives(&q)
	if q.Unsupported == "" || len(q.Objectives) > 0 {
		t.Fatal("missing count became a free/single kill")
	}
	q.Unsupported = ""
	b.polestarObjectives(&q, []dataset.Objective{{Kind: "kill", ID: 100, Count: 5}})
	if q.Unsupported != "" || len(q.Objectives) != 1 || q.Objectives[0].Count != 5 {
		t.Fatal("known matching count was lost")
	}
}

func TestActionHintsRequireTheExactObjectiveTarget(t *testing.T) {
	b := &builder{stats: map[string]int{}, w: &world{}, npcOut: []dataset.NPC{{ID: 100, Name: "Hostile"}, {ID: 3216, Name: "Neeru Fireblade"}},
		out: []dataset.Quest{{ID: 1, Sources: map[string]string{}, Objectives: []dataset.Objective{{Kind: "kill", ID: 100, Count: 5}, {Kind: "kill", ID: 3216, Count: 1}}}},
		gdx: &guideIndex{byQuest: map[int][]guides.Fact{1: {{Complete: true, Obj: 1, Action: "talk", Targets: []int{3216}}, {Complete: true, Obj: 2, Action: "talk", Targets: []int{3216}}}}},
	}
	b.objectiveActions()
	if b.out[0].Objectives[0].Kind != "kill" || b.out[0].Objectives[1].Kind != "talk" {
		t.Fatalf("wrong action attribution: %+v", b.out[0].Objectives)
	}
}

func TestUnknownDeliverySourceIsNotSuppliedByAssumption(t *testing.T) {
	b := &builder{pdb: &polestar.DB{Starts: map[int][]polestar.Ref{1: {{Kind: "npc", ID: 10}}}, Ends: map[int][]polestar.Ref{1: {{Kind: "npc", ID: 20}}}}}
	if b.delivered(1, polestar.Objective{Kind: "item", Target: 500, Count: 1}) {
		t.Fatal("different NPCs are not evidence of a supplied item")
	}
	for _, why := range []string{"objective item 5: count unknown in client and references", "profession's quest"} {
		if dungeonPositionGap(why) {
			t.Fatalf("run cleared non-position gap: %s", why)
		}
	}
}
