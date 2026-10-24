package route

import (
	"container/heap"
	"math"
	"sync/atomic"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Walking on the walk grid (dataset.Walks): costs measured on navigation maps between neighbouring cells, chained by
// Dijkstra from the cell a walk starts in, out to walkReach running yards. A walk between two points takes the
// chained cost between their cells' points, scaled to the points' own distance; a walk the grid cannot answer (off the
// grid, too long, short) is a straight line times PathFactor as before.

const (
	walkReach = 6000.0 // running yards a search from one cell covers
	walkShort = 250.0  // below this a walk is a straight line times PathFactor
	walkSnap  = 2      // cells searched around a point outside the walkable cells
	walkMax   = 6.0    // a walk is never priced above this many times its straight line
	navFactor = 1.05   // players run a little longer than the finder's path (the chained cells are within 2 % of it)
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
		return 0, false
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
