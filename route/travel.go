package route

import (
	"math"
	"sort"
	"sync"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Travel model: walking on straight lines times PathFactor; flights between flight masters the character has found
// (flying ~30 yards per second plus 20 s for take-off and landing, on a slightly longer line than straight); zeppelins
// and boats as fixed connections; the hearthstone (10 s cast, one hour cooldown) back to where it is bound. A flight
// master is found by walking past it within 60 yards. The hearthstone starts bound at the start position and is bound
// at an inn whenever the character accepts or turns in a quest within 40 yards of an innkeeper.

const (
	flightSpeed    = 30.0
	flightOverhead = 20.0
	flightDetour   = 1.15
	discoverRadius = 60.0
	hearthCast     = 10.0
	hearthCooldown = 3600.0
	bindRadius     = 40.0
)

// how a move is made
const (
	byWalk = iota
	byNetwork
	byHearth
)

type network struct {
	nodes []dataset.Point
	fm    []bool // node is a flight master (bit index = node index)
	links map[[2]int]float64
	byNPC map[int]int // flight master or dock NPC -> node
	mu    sync.Mutex // the planner looks ahead in parallel
	cache map[uint64][][]float64
}

func (w *World) buildNetwork() {
	n := &network{links: map[[2]int]float64{}, cache: map[uint64][][]float64{}}
	idx := map[int]int{}
	for _, f := range w.Data.Travel.FlightMasters {
		if f.Faction != w.Faction || len(n.nodes) >= 64 {
			continue
		}
		idx[f.NPC] = len(n.nodes)
		n.nodes = append(n.nodes, f.At)
		n.fm = append(n.fm, true)
	}
	for _, d := range w.Data.Travel.Docks {
		if (d.Faction != "" && d.Faction != w.Faction) || len(n.nodes) >= 64 {
			continue
		}
		idx[d.NPC] = len(n.nodes)
		n.nodes = append(n.nodes, d.At)
		n.fm = append(n.fm, false)
	}
	for _, l := range w.Data.Travel.Links {
		a, okA := idx[l.From]
		b, okB := idx[l.To]
		if okA && okB {
			n.links[[2]int{a, b}] = l.Seconds
		}
	}
	n.byNPC = idx
	w.net = n
	for _, in := range w.Data.Travel.Inns {
		if in.Faction == "" || in.Faction == w.Faction {
			w.inns = append(w.inns, in.At)
		}
	}
}

// reachableMaps: the start continent, and every continent a transport of this faction reaches.
func (w *World) reachableMaps() map[int]bool {
	maps := map[int]bool{w.Start.Map: true}
	for k := range w.net.links {
		maps[w.net.nodes[k[0]].Map] = true
		maps[w.net.nodes[k[1]].Map] = true
	}
	return maps
}

func (w *World) walk(a, b dataset.Point) float64 {
	return distance(a, b) * w.Opt.PathFactor / w.Opt.RunSpeed
}

// table is the fastest time between every two network nodes for a set of known flight masters.
func (w *World) table(known uint64) [][]float64 {
	n := w.net
	n.mu.Lock()
	defer n.mu.Unlock()
	if t, ok := n.cache[known]; ok {
		return t
	}
	k := len(n.nodes)
	d := make([][]float64, k)
	for i := range d {
		d[i] = make([]float64, k)
		for j := range d[i] {
			if i == j {
				continue
			}
			d[i][j] = w.walk(n.nodes[i], n.nodes[j])
			if n.fm[i] && n.fm[j] && known&(1<<i) != 0 && known&(1<<j) != 0 && n.nodes[i].Map == n.nodes[j].Map {
				fly := distance(n.nodes[i], n.nodes[j])*flightDetour/flightSpeed + flightOverhead
				d[i][j] = math.Min(d[i][j], fly)
			}
			if s, ok := n.links[[2]int{i, j}]; ok {
				d[i][j] = math.Min(d[i][j], s)
			}
		}
	}
	for m := 0; m < k; m++ {
		for i := 0; i < k; i++ {
			for j := 0; j < k; j++ {
				if v := d[i][m] + d[m][j]; v < d[i][j] {
					d[i][j] = v
				}
			}
		}
	}
	n.cache[known] = d
	return d
}

// nearestNodes: the few network nodes closest to p on its continent.
func (w *World) nearestNodes(p dataset.Point, max int) []int {
	type c struct {
		i int
		d float64
	}
	var cs []c
	for i, q := range w.net.nodes {
		if q.Map == p.Map {
			cs = append(cs, c{i, distance(p, q)})
		}
	}
	sort.Slice(cs, func(a, b int) bool { return cs[a].d < cs[b].d })
	var out []int
	for i := 0; i < len(cs) && i < max; i++ {
		out = append(out, cs[i].i)
	}
	return out
}

// TravelTime is the fastest way from a to b in state s: walking, the network of known flight masters and transports,
// or the hearthstone when it is ready.
func (w *World) TravelTime(s *State, a, b dataset.Point) float64 {
	t, _ := w.travel(s, a, b)
	return t
}

func (w *World) travel(s *State, a, b dataset.Point) (float64, int) {
	best, how := w.overland(s, a, b)
	if best > hearthCast+30 && s.Time >= s.Hearth && w.walk(s.Bind, a) > 60 {
		if h, _ := w.overland(s, s.Bind, b); hearthCast+h < best {
			best, how = hearthCast+h, byHearth
		}
	}
	return best, how
}

// overland is the fastest way without the hearthstone.
func (w *World) overland(s *State, a, b dataset.Point) (float64, int) {
	best := w.walk(a, b)
	if w.net == nil || len(w.net.nodes) == 0 {
		return best, byWalk
	}
	// only worth checking when walking is long or impossible
	if best < 120 && a.Map == b.Map {
		return best, byWalk
	}
	how := byWalk
	d := w.table(s.Known)
	for _, i := range w.nearestNodes(a, 4) {
		wi := w.walk(a, w.net.nodes[i])
		for _, j := range w.nearestNodes(b, 4) {
			if v := wi + d[i][j] + w.walk(w.net.nodes[j], b); v < best {
				best, how = v, byNetwork
			}
		}
	}
	return best, how
}

// discoverAlong finds the flight masters within reach of the straight way from a to b.
func (w *World) discoverAlong(s *State, a, b dataset.Point) {
	if a.Map != b.Map {
		return
	}
	dx, dy := b.X-a.X, b.Y-a.Y
	l2 := dx*dx + dy*dy
	for i, p := range w.net.nodes {
		if !w.net.fm[i] || s.Known&(1<<i) != 0 || p.Map != a.Map {
			continue
		}
		t := 0.0
		if l2 > 0 {
			t = math.Max(0, math.Min(1, ((p.X-a.X)*dx+(p.Y-a.Y)*dy)/l2))
		}
		if math.Hypot(a.X+t*dx-p.X, a.Y+t*dy-p.Y) <= discoverRadius {
			s.Known |= 1 << i
		}
	}
}

func (w *World) discover(s *State) {
	for i, p := range w.net.nodes {
		if w.net.fm[i] && s.Known&(1<<i) == 0 && distance(s.Pos, p) <= discoverRadius {
			s.Known |= 1 << i
		}
	}
}
