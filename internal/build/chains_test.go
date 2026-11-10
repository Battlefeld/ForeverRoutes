package build

import (
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Of ForeverDB's links only "Complete first" is a prerequisite; "Offered after" is a soft one and "Chain link" none.
func TestForeverChainsByRelation(t *testing.T) {
	src := func() map[string]string { return map[string]string{} }
	b := &builder{stats: map[string]int{}, in: Inputs{ForeverChains: map[int][]ChainLink{
		844:  {{Pre: 860, Relation: "Chain link"}, {Pre: 842, Relation: "Offered after"}}, // Plainstrider Menace
		845:  {{Pre: 844, Relation: "Complete first"}},
		848:  {{Pre: 1358, Relation: "Offered after"}},
		6062: {{Pre: 6068, Relation: "Complete first"}, {Pre: 6069, Relation: "Complete first"}},
		903:  {{Pre: 844, Relation: "Offered after"}, {Pre: 999999, Relation: "Complete first"}}, // unknown quest: ignored
		5041: {{Pre: 871, Relation: "Offered after"}},
	}}, out: []dataset.Quest{
		{ID: 842, Sources: src()}, {ID: 844, Sources: src()}, {ID: 845, Sources: src()}, {ID: 848, Sources: src()},
		{ID: 860, Sources: src()}, {ID: 1358, Sources: src()}, {ID: 6062, Sources: src()}, {ID: 6068, Sources: src()},
		{ID: 6069, Sources: src()}, {ID: 903, Sources: src()}, {ID: 871, Sources: src()},
		{ID: 5041, PreAll: []int{871}, Sources: src()}, // another source's prerequisite stays, and is not soft as well
	}}
	b.foreverChains()
	q := map[int]dataset.Quest{}
	for _, x := range b.out {
		q[x.ID] = x
	}
	if len(q[844].PreAll)+len(q[844].PreAny) != 0 || len(q[844].PreSoft) != 1 || q[844].PreSoft[0] != 842 {
		t.Errorf("844: %v %v soft %v", q[844].PreAll, q[844].PreAny, q[844].PreSoft)
	}
	if len(q[845].PreAll) != 1 || q[845].PreAll[0] != 844 || q[845].Sources["prerequisites"] != "foreverdb_chain" {
		t.Errorf("845: %v %v", q[845].PreAll, q[845].Sources)
	}
	if len(q[848].PreAll)+len(q[848].PreAny) != 0 || len(q[848].PreSoft) != 1 {
		t.Errorf("848: %v %v soft %v", q[848].PreAll, q[848].PreAny, q[848].PreSoft)
	}
	if len(q[6062].PreAny) != 2 || len(q[6062].PreAll) != 0 {
		t.Errorf("6062: %v %v", q[6062].PreAll, q[6062].PreAny)
	}
	if len(q[903].PreAll)+len(q[903].PreAny) != 0 || len(q[903].PreSoft) != 1 {
		t.Errorf("903: %v %v soft %v", q[903].PreAll, q[903].PreAny, q[903].PreSoft)
	}
	if len(q[5041].PreAll) != 1 || len(q[5041].PreSoft) != 0 {
		t.Errorf("5041: %v soft %v", q[5041].PreAll, q[5041].PreSoft)
	}
	if b.stats["prerequisites_from_foreverdb_chains"] != 2 || b.stats["soft_prerequisites_from_foreverdb_chains"] != 3 {
		t.Errorf("stats %v", b.stats)
	}
}
