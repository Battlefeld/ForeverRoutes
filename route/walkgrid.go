package route

import (
	"container/heap"
	"math"
	"sync"

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
	index map[[2]int]int32 // (i, j) -> cell
	n     int
	mu    sync.Mutex
	from  map[int32]map[int32]float32 // source cell -> reached cell -> cost
}

func newWalkGrid(m dataset.WalkMap) *walkGrid {
	g := &walkGrid{m: m, index: map[[2]int]int32{}, n: len(m.Cells) / 4, from: map[int32]map[int32]float32{}}
	for c := 0; c < g.n; c++ {
		g.index[[2]int{int(m.Cells[4*c]), int(m.Cells[4*c+1])}] = int32(c)
	}
	return g
}

func (g *walkGrid) point(c int32) (float64, float64) { return g.m.Cells[4*c+2], g.m.Cells[4*c+3] }

// cellOf is the walkable cell a point lies in, else the nearest within walkSnap cells (-1: none).
func (g *walkGrid) cellOf(p dataset.Point) int32 {
	i := int(math.Floor((p.X - g.m.X0) / g.m.Cell))
	j := int(math.Floor((p.Y - g.m.Y0) / g.m.Cell))
	if c, ok := g.index[[2]int{i, j}]; ok {
		return c
	}
	best, bestD := int32(-1), math.Inf(1)
	for di := -walkSnap; di <= walkSnap; di++ {
		for dj := -walkSnap; dj <= walkSnap; dj++ {
			if c, ok := g.index[[2]int{i + di, j + dj}]; ok {
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

// costs from a source cell to every cell within walkReach, searched once and kept.
func (g *walkGrid) costs(src int32) map[int32]float32 {
	g.mu.Lock()
	if m, ok := g.from[src]; ok {
		g.mu.Unlock()
		return m
	}
	g.mu.Unlock()
	dist := map[int32]float32{src: 0}
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
			nb, ok := g.index[[2]int{i + mv[0], j + mv[1]}]
			if !ok {
				continue
			}
			nd := it.d + float32(cost)
			if nd > walkReach {
				continue
			}
			if old, seen := dist[nb]; !seen || nd < old {
				dist[nb] = nd
				heap.Push(h, walkItem{nb, nd})
			}
		}
	}
	g.mu.Lock()
	g.from[src] = dist
	g.mu.Unlock()
	return dist
}

// walkTime prices a walk on the grid; ok is false when the grid has no answer.
func (g *walkGrid) walkTime(a, b dataset.Point, straight float64) (float64, bool) {
	ca, cb := g.cellOf(a), g.cellOf(b)
	if ca < 0 || cb < 0 || ca == cb {
		return 0, false
	}
	cost, ok := g.costs(ca)[cb]
	if !ok {
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
