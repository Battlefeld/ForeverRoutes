package route

import (
	"math"
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// a 3 x 3 grid of 100-yard cells with a wall: the middle column is blocked except at the bottom row.
func testWalks() *dataset.Walks {
	m := dataset.WalkMap{Map: 0, X0: 0, Y0: 0, Cell: 100, NI: 3, NJ: 3}
	open := func(i, j int) bool { return !(i == 1 && j > 0) }
	moves := map[[2]int]int{}
	for k, mv := range dataset.WalkMoves {
		moves[mv] = k
	}
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			if !open(i, j) {
				continue
			}
			m.Cells = append(m.Cells, float64(i), float64(j), float64(i)*100+50, float64(j)*100+50)
			var costs [16]int
			for _, mv := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				if a, b := i+mv[0], j+mv[1]; a >= 0 && b >= 0 && a < 3 && b < 3 && open(a, b) {
					costs[moves[mv]] = 100
				}
			}
			m.Costs = append(m.Costs, costs[:]...)
		}
	}
	return &dataset.Walks{Schema: dataset.WalksSchema, Maps: []dataset.WalkMap{m}}
}

func TestWalkGridDetour(t *testing.T) {
	g := newWalks(testWalks())[0]
	a, b := dataset.Point{X: 50, Y: 250}, dataset.Point{X: 250, Y: 250}
	d := distance(a, b)
	got, ok := g.walkTime(a, b, d)
	if !ok {
		t.Fatal("no answer")
	}
	// around the wall: 6 steps of 100 yards for 200 yards straight
	if want := d * 3 * navFactor; math.Abs(got-want) > 1e-6 {
		t.Fatalf("got %.1f, want %.1f", got, want)
	}
	if _, ok := g.walkTime(a, dataset.Point{X: 60, Y: 260}, 14); ok {
		t.Fatal("same cell answered")
	}
	// a point in the blocked cell snaps to the nearest open one
	if c := g.cellOf(dataset.Point{X: 150, Y: 250}); c < 0 {
		t.Fatal("no snap")
	}
}

func TestWalkUsesGrid(t *testing.T) {
	w := &World{Opt: Options{Walks: testWalks()}.withDefaults()}
	w.walkGrids = newWalks(w.Opt.Walks)
	a, b := dataset.Point{X: 20, Y: 280}, dataset.Point{X: 280, Y: 280}
	straight := distance(a, b) * w.Opt.PathFactor / w.Opt.RunSpeed
	if got := w.walk(a, b); got <= straight {
		t.Fatalf("walk %.1f s not longer than the straight line %.1f s", got, straight)
	}
	near := dataset.Point{X: 50, Y: 150}
	if got, want := w.walk(a, near), distance(a, near)*w.Opt.PathFactor/w.Opt.RunSpeed; got != want {
		t.Fatalf("short walk %.2f, want %.2f", got, want)
	}
}
