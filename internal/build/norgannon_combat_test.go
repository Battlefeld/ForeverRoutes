package build

import (
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/norgannon"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/polestar"
)

func TestSourcedEliteRanksDoNotOverrideExplicitClientNormalRank(t *testing.T) {
	zero, elite := 0, 1
	b := &builder{stats: map[string]int{}, npcOut: []dataset.NPC{{ID: 1}, {ID: 2}, {ID: 3, Rank: 3}},
		ngn: &norgannon.Data{NPCs: map[int]norgannon.CombatMetadata{1: {Rank: &elite, MinLevel: 20, MaxLevel: 20}, 2: {Rank: &elite}, 3: {Rank: &elite}}},
		pdb: &polestar.DB{NPCs: map[int]*polestar.NPC{2: {ID: 2, Rank: &zero}}}}
	b.norgannonCombat()
	if b.npcOut[0].Rank != 1 || b.npcOut[0].MaxLevel != 20 {
		t.Fatal("missing elite metadata was not filled")
	}
	if b.npcOut[1].Rank != 0 {
		t.Fatal("explicit current client normal rank was overwritten")
	}
	if b.npcOut[2].Rank != 3 {
		t.Fatal("known boss was downgraded by a fallback source")
	}
}
