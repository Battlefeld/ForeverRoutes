package route

import (
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// The table's walks are found both ways and near their places; a walk it lacks is not.
func TestWalkTable(t *testing.T) {
	// places 0 (0,0), 1 (100,0), 2 (0,100); walks 0-1 (150 yd), 0-2 (300 yd), 1-2 (500 yd), delta-coded
	tab := &dataset.WalkTable{Schema: dataset.WalkTableSchema, Maps: []dataset.WalkTableMap{{
		Map: 1, X: []float32{0, 100, 0}, Y: []float32{0, 0, 100},
		Pairs: []int32{0, 1, 150, 0, 1, 300, 1, 1, 500},
	}}}
	wt := newWalkTable(tab)
	p := func(x, y float64) dataset.Point { return dataset.Point{Map: 1, X: x, Y: y} }
	for _, c := range []struct {
		a, b dataset.Point
		want float64
		ok   bool
	}{
		{p(0, 0), p(100, 0), 150, true},
		{p(100, 0), p(0, 0), 150, true},
		{p(0, 0), p(0, 100), 300, true},
		{p(100, 0), p(0, 100), 500, true},
		{p(1, 0), p(100, 0), 151, true}, // a yard off a place
		{p(0, 0), p(50, 50), 0, false},  // not a place
		{dataset.Point{Map: 0}, p(0, 0), 0, false},
	} {
		got, ok := wt.yards(c.a, c.b)
		if ok != c.ok || ok && got != c.want {
			t.Errorf("%v -> %v: %.0f %v, want %.0f %v", c.a, c.b, got, ok, c.want, c.ok)
		}
	}
}
