package build

import (
	"math"
	"testing"
)

func TestFitTransformDropsMovedCreatures(t *testing.T) {
	// a zone 1000 x 800 yards: world x = 500 - 10*py, world y = 2000 - 8*px
	var ps []pair
	for i := 0; i < 12; i++ {
		px, py := float64(5+i*7), float64(90-i*6)
		ps = append(ps, pair{px: px, py: py, m: 1, wx: 500 - 10*py, wy: 2000 - 8*px})
	}
	ps = append(ps, pair{px: 50, py: 50, m: 1, wx: 9999, wy: 9999}) // moved on Forever
	ps = append(ps, pair{px: 50, py: 50, m: 0, wx: 1, wy: 1})       // other continent
	tr, ok := fitTransform(ps)
	if !ok {
		t.Fatal("no fit")
	}
	p := tr.apply(40, 20)
	if p.Map != 1 || math.Abs(p.X-300) > 0.5 || math.Abs(p.Y-1680) > 0.5 {
		t.Errorf("apply(40,20) = %+v, want (1, 300, 1680)", p)
	}
}

func TestFitTransformNeedsEnoughPairs(t *testing.T) {
	if _, ok := fitTransform([]pair{{px: 1, py: 1, wx: 1, wy: 1}, {px: 2, py: 2, wx: 2, wy: 2}}); ok {
		t.Error("two pairs are not enough")
	}
}
