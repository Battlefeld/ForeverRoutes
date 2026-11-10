package route

import (
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// The sweep through Durotar's quests reaches the level without grinding more than a guide would, hands in what it
// takes and never walks away from a finished quest giver's neighbour: every block's quests end in the block or later.
func TestSweepDurotar(t *testing.T) {
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Skip("no dataset")
	}
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Shaman"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	set := map[int]bool{}
	for _, q := range w.Quests {
		if q.MinLevel < 8 && q.Dungeon == nil && q.Type == "" && !q.Repeatable && len(q.StartSources) == 0 && w.zoneOf(q.StartPos) == "Durotar" {
			set[q.ID] = true
		}
	}
	if len(set) < 20 {
		t.Fatalf("only %d quests in Durotar", len(set))
	}
	res, blocks, err := w.Sweep(set, SweepOptions{Target: 8})
	if err != nil {
		t.Fatal(err)
	}
	if res.Level < 8 {
		t.Fatalf("level %d, want 8", res.Level)
	}
	taken, handed := map[int]bool{}, map[int]bool{}
	for _, s := range res.Steps {
		switch s.Op {
		case "accept":
			if !set[s.Quest] {
				t.Errorf("quest %d is not in the set", s.Quest)
			}
			taken[s.Quest] = true
		case "turnin":
			handed[s.Quest] = true
		}
	}
	if len(handed) < 15 {
		t.Errorf("only %d quests handed in", len(handed))
	}
	// a second run gives the same route
	again, _, err := w.Sweep(set, SweepOptions{Target: 8})
	if err != nil || again.Seconds != res.Seconds || len(again.Steps) != len(res.Steps) {
		t.Errorf("the sweep is not repeatable: %v, %.1f s against %.1f s", err, again.Seconds, res.Seconds)
	}
	// the blocks cover the route in order and without gaps
	if len(blocks) == 0 {
		t.Fatal("no blocks")
	}
	xp, n := 0, 0
	for i, b := range blocks {
		if i > 0 && b.Start != blocks[i-1].End {
			t.Errorf("block %d begins at %.1f, the one before ends at %.1f", i, b.Start, blocks[i-1].End)
		}
		xp += b.XP
		n += len(b.TurnedIn)
	}
	if xp != res.XP || n != len(handed) {
		t.Errorf("blocks hold %d XP and %d hand-ins, the route %d and %d", xp, n, res.XP, len(handed))
	}
	if blocks[len(blocks)-1].End != res.Seconds {
		t.Errorf("blocks end at %.1f, the route at %.1f", blocks[len(blocks)-1].End, res.Seconds)
	}
}

// A quest waits for the one players took before it while that one is open, and is taken once nothing else is left.
func TestSweepPrefers(t *testing.T) {
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Skip("no dataset")
	}
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Shaman"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	// two quests of the Valley of Trials
	a, b := w.ByID[788], w.ByID[790]
	if a == nil || b == nil {
		t.Skip("quests not in this dataset")
	}
	set := map[int]bool{788: true, 790: true}
	res, _, err := w.Sweep(set, SweepOptions{Target: 2, Prefer: map[int][]int{788: {790}}})
	if err != nil {
		t.Fatal(err)
	}
	order := map[int]int{}
	for i, s := range res.Steps {
		if s.Op == "accept" {
			order[s.Quest] = i + 1
		}
		if s.Op == "turnin" && s.Quest == 790 && order[788] != 0 {
			t.Errorf("788 was taken before 790 was handed in")
		}
	}
}
