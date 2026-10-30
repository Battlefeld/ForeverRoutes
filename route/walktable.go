package route

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math"
	"sort"
	"sync"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// The walk table (dataset.WalkTable): walks between the places routes go to (World.Places), measured once with
// Shortest Path Forever's path finder at the places' heights, as the yards a player runs. A walk between two places
// of the table takes its measured yards; any other walk the walk grid (walkgrid.go).

// tableSnap: a point this close to a place of the table is at it (the places are kept one per 2 yards).
const tableSnap = 3.0

type tableMap struct {
	x, y  []float32
	cells map[[2]int32][]int32 // tableSnap cells -> places
	off   []int32              // place i's walks to places after it: to[off[i]:off[i+1]], sorted
	to    []int32
	yards []uint16
}

type walkTable map[int]*tableMap

// walkTables: the decoded tables by the file read, so the worlds of one run share one.
var walkTables sync.Map

func newWalkTable(t *dataset.WalkTable) walkTable {
	if t == nil {
		return nil
	}
	if wt, ok := walkTables.Load(t); ok {
		return wt.(walkTable)
	}
	out := walkTable{}
	for _, m := range t.Maps {
		tm := &tableMap{x: m.X, y: m.Y, cells: map[[2]int32][]int32{}, off: make([]int32, len(m.X)+1)}
		for i := range m.X {
			c := tableCell(float64(m.X[i]), float64(m.Y[i]))
			tm.cells[c] = append(tm.cells[c], int32(i))
		}
		type walk struct{ i, j, y int32 }
		walks := make([]walk, 0, len(m.Pairs)/3)
		i, j := int32(0), int32(0)
		for k := 0; k+2 < len(m.Pairs); k += 3 {
			di, dj := m.Pairs[k], m.Pairs[k+1]
			i += di
			if di == 0 && k > 0 {
				j += dj
			} else {
				j = i + dj
			}
			if i >= 0 && j > i && int(j) < len(m.X) {
				walks = append(walks, walk{i, j, m.Pairs[k+2]})
			}
		}
		sort.Slice(walks, func(a, b int) bool {
			if walks[a].i != walks[b].i {
				return walks[a].i < walks[b].i
			}
			return walks[a].j < walks[b].j
		})
		tm.to, tm.yards = make([]int32, len(walks)), make([]uint16, len(walks))
		for k, wk := range walks {
			tm.off[wk.i+1]++
			tm.to[k], tm.yards[k] = wk.j, uint16(min(wk.y, math.MaxUint16))
		}
		for k := 1; k < len(tm.off); k++ {
			tm.off[k] += tm.off[k-1]
		}
		out[m.Map] = tm
	}
	walkTables.Store(t, out)
	return out
}

func tableCell(x, y float64) [2]int32 {
	return [2]int32{int32(math.Floor(x / tableSnap)), int32(math.Floor(y / tableSnap))}
}

// place: the table's place nearest p within tableSnap and how far it is, or -1.
func (tm *tableMap) place(p dataset.Point) (int32, float64) {
	best, bd := int32(-1), tableSnap
	c := tableCell(p.X, p.Y)
	for di := int32(-1); di <= 1; di++ {
		for dj := int32(-1); dj <= 1; dj++ {
			for _, i := range tm.cells[[2]int32{c[0] + di, c[1] + dj}] {
				if d := math.Hypot(float64(tm.x[i])-p.X, float64(tm.y[i])-p.Y); d <= bd {
					best, bd = i, d
				}
			}
		}
	}
	return best, bd
}

// yards: the measured walk from a to b in running yards, false when the table does not have it.
func (t walkTable) yards(a, b dataset.Point) (float64, bool) {
	tm := t[a.Map]
	if tm == nil || a.Map != b.Map {
		return 0, false
	}
	i, da := tm.place(a)
	j, db := tm.place(b)
	if i < 0 || j < 0 {
		return 0, false
	}
	if i == j {
		return math.Hypot(a.X-b.X, a.Y-b.Y), true
	}
	if i > j {
		i, j = j, i
	}
	to := tm.to[tm.off[i]:tm.off[i+1]]
	k := sort.Search(len(to), func(k int) bool { return to[k] >= j })
	if k == len(to) || to[k] != j {
		return 0, false
	}
	return float64(tm.yards[int(tm.off[i])+k]) + da + db, true
}

// digest names the table's content for a checkpoint's context ("" without one): a route found with other walks is
// another route.
func (t walkTable) digest() string {
	if len(t) == 0 {
		return ""
	}
	h := sha256.New()
	maps := make([]int, 0, len(t))
	for m := range t {
		maps = append(maps, m)
	}
	sort.Ints(maps)
	for _, m := range maps {
		tm := t[m]
		binary.Write(h, binary.LittleEndian, int64(m))
		binary.Write(h, binary.LittleEndian, tm.x)
		binary.Write(h, binary.LittleEndian, tm.y)
		binary.Write(h, binary.LittleEndian, tm.to)
		binary.Write(h, binary.LittleEndian, tm.yards)
	}
	return hex.EncodeToString(h.Sum(nil))
}
