package route

import (
	"container/heap"
	"math"
	"sync/atomic"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Walking on the walk grid (dataset.Walks): costs measured on navigation maps between neighbouring cells, chained by
// Dijkstra from the cell a walk starts in over the whole map. A walk between two points takes the chained cost between
// their cells' points, scaled to the points' own distance. Two cells the grid does not connect are walkMax times their
// straight line apart (a way exists in game only where the maps miss it); a walk off the grid or a short one is a
// straight line times PathFactor as before.

const (
	walkReach = 1e9   // a search from one cell covers the whole map
	walkShort = 250.0 // below this a walk is a straight line times PathFactor
	walkSnap  = 2     // cells searched around a point outside the walkable cells
	walkMax   = 6.0   // a walk is never priced above this many times its straight line
	navFactor = 1.05  // players run a little longer than the finder's path (the chained cells are within 2 % of it)
)

type walkGrid struct {
	m     dataset.WalkMap
	index []int32 // i*NJ + j -> cell, -1 for none
	n     int
	from  []atomic.Pointer[[]float32] // source cell -> cost to every cell (+Inf beyond walkReach), searched once
}

func newWalkGrid(m dataset.WalkMap) *walkGrid {
	g := &walkGrid{m: m, index: make([]int32, m.NI*m.NJ), n: len(m.Cells) / 4}
	for k := range g.index {
		g.index[k] = -1
	}
	for c := 0; c < g.n; c++ {
		if i, j := int(m.Cells[4*c]), int(m.Cells[4*c+1]); i >= 0 && j >= 0 && i < m.NI && j < m.NJ {
			g.index[i*m.NJ+j] = int32(c)
		}
	}
	g.from = make([]atomic.Pointer[[]float32], g.n)
	return g
}

func (g *walkGrid) at(i, j int) int32 {
	if i < 0 || j < 0 || i >= g.m.NI || j >= g.m.NJ {
		return -1
	}
	return g.index[i*g.m.NJ+j]
}

func (g *walkGrid) point(c int32) (float64, float64) { return g.m.Cells[4*c+2], g.m.Cells[4*c+3] }

// cellOf is the walkable cell a point lies in, else the nearest within walkSnap cells (-1: none).
func (g *walkGrid) cellOf(p dataset.Point) int32 {
	i := int(math.Floor((p.X - g.m.X0) / g.m.Cell))
	j := int(math.Floor((p.Y - g.m.Y0) / g.m.Cell))
	if c := g.at(i, j); c >= 0 {
		return c
	}
	best, bestD := int32(-1), math.Inf(1)
	for di := -walkSnap; di <= walkSnap; di++ {
		for dj := -walkSnap; dj <= walkSnap; dj++ {
			if c := g.at(i+di, j+dj); c >= 0 {
				x, y := g.point(c)
				if d := math.Hypot(x-p.X, y-p.Y); d < bestD {
					best, bestD = c, d
				}
			}
		}
	}
	return best
}

type walkItem struct {
	c int32
	d float32
}
type walkHeap []walkItem

func (h walkHeap) Len() int           { return len(h) }
func (h walkHeap) Less(a, b int) bool { return h[a].d < h[b].d }
func (h walkHeap) Swap(a, b int)      { h[a], h[b] = h[b], h[a] }
func (h *walkHeap) Push(x any)        { *h = append(*h, x.(walkItem)) }
func (h *walkHeap) Pop() any {
	old := *h
	it := old[len(old)-1]
	*h = old[:len(old)-1]
	return it
}

// costs from a source cell to every cell within walkReach (+Inf beyond), searched once and kept.
func (g *walkGrid) costs(src int32) []float32 {
	if p := g.from[src].Load(); p != nil {
		return *p
	}
	inf := float32(math.Inf(1))
	dist := make([]float32, g.n)
	for k := range dist {
		dist[k] = inf
	}
	dist[src] = 0
	h := &walkHeap{{src, 0}}
	for h.Len() > 0 {
		it := heap.Pop(h).(walkItem)
		if it.d > dist[it.c] {
			continue
		}
		i, j := int(g.m.Cells[4*it.c]), int(g.m.Cells[4*it.c+1])
		for k, mv := range dataset.WalkMoves {
			cost := g.m.Costs[16*int(it.c)+k]
			if cost <= 0 {
				continue
			}
			nb := g.at(i+mv[0], j+mv[1])
			if nb < 0 {
				continue
			}
			if nd := it.d + float32(cost); nd <= walkReach && nd < dist[nb] {
				dist[nb] = nd
				heap.Push(h, walkItem{nb, nd})
			}
		}
	}
	g.from[src].Store(&dist)
	return dist
}

// walkTime prices a walk on the grid; ok is false when the grid has no answer.
func (g *walkGrid) walkTime(a, b dataset.Point, straight float64) (float64, bool) {
	ca, cb := g.cellOf(a), g.cellOf(b)
	if ca < 0 || cb < 0 || ca == cb {
		return 0, false
	}
	cost := g.costs(ca)[cb]
	if math.IsInf(float64(cost), 1) {
		// a point on a pocket of cells the grid does not join to the rest (the top of a dam, a ledge): from or to the
		// nearest cell around it that is joined, the rest a straight line
		if y, ok := g.viaPocket(a, b, ca, cb); ok {
			return math.Min(y, straight*walkMax), true
		}
		return straight * walkMax, true
	}
	ax, ay := g.point(ca)
	bx, by := g.point(cb)
	base := math.Hypot(ax-bx, ay-by)
	if base <= 0 {
		return 0, false
	}
	f := math.Max(1, math.Min(walkMax, float64(cost)/base))
	return straight * f * navFactor, true
}

// pocketReach: cells around a point on a pocket searched for a joined one.
const pocketReach = 3

// viaPocket prices a walk whose ends the grid does not join: the cheapest pair of cells around them (a's own first)
// that are joined, with straight lines (times pocketFactor) to and from them, in yards.
func (g *walkGrid) viaPocket(a, b dataset.Point, ca, cb int32) (float64, bool) {
	around := func(p dataset.Point, c int32) []int32 {
		out := []int32{c}
		i := int(math.Floor((p.X - g.m.X0) / g.m.Cell))
		j := int(math.Floor((p.Y - g.m.Y0) / g.m.Cell))
		for di := -pocketReach; di <= pocketReach; di++ {
			for dj := -pocketReach; dj <= pocketReach; dj++ {
				if n := g.at(i+di, j+dj); n >= 0 && n != c {
					out = append(out, n)
				}
			}
		}
		return out
	}
	off := func(p dataset.Point, c int32) float64 {
		x, y := g.point(c)
		return math.Hypot(x-p.X, y-p.Y) * pocketFactor
	}
	best := math.Inf(1)
	for k, sa := range around(a, ca) {
		if k > 0 && !math.IsInf(best, 1) {
			break // a's own cell reaches one around b: no need to leave it
		}
		cs := g.costs(sa)
		for _, sb := range around(b, cb) {
			if c := cs[sb]; !math.IsInf(float64(c), 1) {
				best = math.Min(best, off(a, sa)+float64(c)*navFactor+off(b, sb))
			}
		}
	}
	return best, !math.IsInf(best, 1)
}

// pocketFactor: the straight line to or from a pocket, as PathFactor.
const pocketFactor = 1.2

// walks are the walk grids by map.
type walks map[int]*walkGrid

func newWalks(w *dataset.Walks) walks {
	if w == nil {
		return nil
	}
	out := walks{}
	for _, m := range w.Maps {
		out[m.Map] = newWalkGrid(m)
	}
	return out
}
