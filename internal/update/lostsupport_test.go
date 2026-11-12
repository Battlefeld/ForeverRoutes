package update

import (
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// New quests of a new game build that no source describes yet are not a loss; a quest that worked and no longer does is.
func TestLostSupport(t *testing.T) {
	prev := &dataset.Dataset{Quests: []dataset.Quest{{ID: 1}, {ID: 2}, {ID: 3, Unsupported: "x"}}}
	next := &dataset.Dataset{Quests: []dataset.Quest{{ID: 1}, {ID: 2, Unsupported: "broke"}, {ID: 3, Unsupported: "x"}, {ID: 4, Unsupported: "new"}, {ID: 5, Unsupported: "new"}}}
	if n := lostSupport(prev, next); n != 1 {
		t.Fatalf("lost %d, want 1", n)
	}
}
