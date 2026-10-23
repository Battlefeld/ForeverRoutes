package build

import (
	"math"
	"sort"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// The merged database stores positions as percent of a zone map (Questie: zone id, EverythingQuests: uiMapId). The
// route engine needs one metric space: world yards per continent. For each map the linear relation is fitted from
// creatures whose position is known in both systems (cmangos world coordinates); a map without a reliable fit is not
// converted.

type pair struct {
	px, py float64 // percent
	m      int     // continent
	wx, wy float64 // world yards
}

type transform struct {
	m         int
	ax, bx    float64 // world x = ax + bx*py
	ay, by    float64 // world y = ay + by*px
	median, n float64
}

func (t transform) apply(px, py float64) dataset.Point {
	return dataset.Point{Map: t.m, X: round1(t.ax + t.bx*py), Y: round1(t.ay + t.by*px)}
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func linFit(xs, ys []float64) (a, b float64, ok bool) {
	n := float64(len(xs))
	if n < 2 {
		return 0, 0, false
	}
	var mx, my float64
	for i := range xs {
		mx += xs[i]
		my += ys[i]
	}
	mx /= n
	my /= n
	var sxx, sxy float64
	for i := range xs {
		sxx += (xs[i] - mx) * (xs[i] - mx)
		sxy += (xs[i] - mx) * (ys[i] - my)
	}
	if sxx == 0 {
		return 0, 0, false
	}
	b = sxy / sxx
	return my - b*mx, b, true
}

// fitTransform fits one map robustly: the main continent only, then the pairs that disagree with the fit (creatures
// Forever moved) are dropped and the fit is repeated.
func fitTransform(ps []pair) (transform, bool) {
	if len(ps) < 3 {
		return transform{}, false
	}
	counts := map[int]int{}
	for _, p := range ps {
		counts[p.m]++
	}
	best, bm := -1, 0
	for m, c := range counts {
		if c > bm || c == bm && m < best {
			best, bm = m, c
		}
	}
	var use []pair
	for _, p := range ps {
		if p.m == best {
			use = append(use, p)
		}
	}
	var t transform
	for round := 0; round < 5; round++ {
		if len(use) < 3 {
			return transform{}, false
		}
		xs, ys1, ys2, xs2 := []float64{}, []float64{}, []float64{}, []float64{}
		for _, p := range use {
			xs = append(xs, p.py)
			ys1 = append(ys1, p.wx)
			xs2 = append(xs2, p.px)
			ys2 = append(ys2, p.wy)
		}
		ax, bx, ok1 := linFit(xs, ys1)
		ay, by, ok2 := linFit(xs2, ys2)
		if !ok1 || !ok2 {
			return transform{}, false
		}
		t = transform{m: best, ax: ax, bx: bx, ay: ay, by: by}
		res := make([]float64, len(use))
		for i, p := range use {
			q := t.apply(p.px, p.py)
			res[i] = math.Hypot(q.X-p.wx, q.Y-p.wy)
		}
		sorted := append([]float64(nil), res...)
		sort.Float64s(sorted)
		t.median, t.n = sorted[len(sorted)/2], float64(len(use))
		// a single far-off pair pulls the first fit; the cut follows the median error until the fit settles at 30 yards
		cut := math.Max(30, 2.5*t.median)
		var keep []pair
		for i, p := range use {
			if res[i] <= cut {
				keep = append(keep, p)
			}
		}
		if len(keep) == len(use) {
			break
		}
		use = keep
	}
	// a percent map is 100 units wide; a scale near zero or a poor fit means the pairs do not describe one map
	if t.median > 15 || math.Abs(t.bx) < 1 || math.Abs(t.by) < 1 {
		return transform{}, false
	}
	return t, true
}
