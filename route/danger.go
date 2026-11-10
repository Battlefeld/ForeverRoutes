package route

import (
	"math"
	"sort"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Danger: what a place costs a character of a level is the creatures around it, not only the one it fights. Each cell
// of dangerCell yards gets the level most of the hostile creatures spawning in it reach (the 70th percentile of their
// levels, from dangerMin spawns on); a place whose cell is more than MaxMobAbove levels above the character waits until
// the character is strong enough: gathering, using an object, an event, or an item that starts a quest there. Fighting
// a creature already has its own check (its level), and talking to an NPC is never refused: givers stand in camps.

const (
	dangerCell = 200.0
	dangerMin  = 3
)

type dangerMap map[[3]int]int // map, i, j -> level

func newDanger(npcs []dataset.NPC, faction string) dangerMap {
	levels := map[[3]int][]int{}
	for i := range npcs {
		n := &npcs[i]
		if n.Rank != 0 || n.MaxLevel < 1 || n.Hostile == "" || n.Hostile == "-" || (n.Hostile != "AH" && n.Hostile != faction) {
			continue
		}
		for _, p := range n.Spawns {
			k := [3]int{p.Map, int(math.Floor(p.X / dangerCell)), int(math.Floor(p.Y / dangerCell))}
			levels[k] = append(levels[k], mobLevel(n))
		}
	}
	d := dangerMap{}
	for k, l := range levels {
		if len(l) < dangerMin {
			continue
		}
		sort.Ints(l)
		d[k] = l[int(0.7*float64(len(l)-1))]
	}
	return d
}

// eliteMin: elite spawns in a cell from which the cell is an elite place (the outer caves of dungeons, ogre camps);
// a lone patrolling elite does not make one.
const eliteMin = 3

// newElites: cells with eliteMin or more spawns of elites that may attack this faction, with the highest level among
// them. Solo, such a place is only for a warlock or hunter of that level (the pet tanks).
func newElites(npcs []dataset.NPC, faction string) dangerMap {
	n, top := map[[3]int]int{}, map[[3]int]int{}
	for i := range npcs {
		c := &npcs[i]
		if c.Rank < 1 || c.Rank > 3 || c.Lone || c.Hostile == "-" || (c.Hostile != "" && c.Hostile != "AH" && c.Hostile != faction) {
			continue
		}
		for _, p := range c.Spawns {
			k := [3]int{p.Map, int(math.Floor(p.X / dangerCell)), int(math.Floor(p.Y / dangerCell))}
			n[k]++
			top[k] = max(top[k], mobLevel(c))
		}
	}
	d := dangerMap{}
	for k, c := range n {
		if c >= eliteMin {
			d[k] = top[k]
		}
	}
	return d
}

type eliteSpawn struct {
	at    dataset.Point
	level int
}

// newEliteSpawns: the spawns newElites counts, kept by cell for the radius rule (Options.EliteRadius).
func newEliteSpawns(npcs []dataset.NPC, faction string) map[[3]int][]eliteSpawn {
	out := map[[3]int][]eliteSpawn{}
	for i := range npcs {
		c := &npcs[i]
		if c.Rank < 1 || c.Rank > 3 || c.Lone || c.Hostile == "-" || (c.Hostile != "" && c.Hostile != "AH" && c.Hostile != faction) {
			continue
		}
		for _, p := range c.Spawns {
			k := [3]int{p.Map, int(math.Floor(p.X / dangerCell)), int(math.Floor(p.Y / dangerCell))}
			out[k] = append(out[k], eliteSpawn{p, mobLevel(c)})
		}
	}
	return out
}

// eliteAt is the highest elite level at a place: by the cell, or with Options.EliteRadius by the elite spawns within
// that radius (0: no elite place).
func (w *World) eliteAt(p dataset.Point) int {
	r := w.Opt.EliteRadius
	if r <= 0 || w.eliteSpawns == nil {
		return w.elites.at(p)
	}
	cx, cy := int(math.Floor(p.X/dangerCell)), int(math.Floor(p.Y/dangerCell))
	n, top := 0, 0
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			for _, e := range w.eliteSpawns[[3]int{p.Map, cx + dx, cy + dy}] {
				if math.Hypot(e.at.X-p.X, e.at.Y-p.Y) <= r {
					n++
					top = max(top, e.level)
				}
			}
		}
	}
	if n < eliteMin {
		return 0
	}
	return top
}

// at is the danger level of a place (0: nothing known).
func (d dangerMap) at(p dataset.Point) int {
	return d[[3]int{p.Map, int(math.Floor(p.X / dangerCell)), int(math.Floor(p.Y / dangerCell))}]
}

// tooDangerous tells whether the character cannot work at a source yet: a creature above its fighting range, or a place
// among such creatures.
func (w *World) tooDangerous(s *State, src Source) bool {
	return w.dangerousAt(w.Level(s), src)
}

// dangerousAt: tooDangerous for a character of the given level.
func (w *World) dangerousAt(level int, src Source) bool {
	lvl := level + w.Opt.MaxMobAbove
	if src.Talk {
		return false
	}
	if src.NPC != nil && src.NPC.MaxLevel < 1 {
		return true // unknown combat difficulty is not a level-one target
	}
	if src.Class {
		return src.NPC != nil && max(src.NPC.MaxLevel, mobLevel(src.NPC)) > lvl // a class quest's creature: by its level only
	}
	if e := w.eliteAt(src.Site.Center); e > 0 && !(w.Opt.SoloElites && e <= level) {
		return true // among elites: not alone
	}
	if src.NPC != nil {
		if src.NPC.Rank == 1 {
			// Only the explicit pet-tanking policy permits ordinary elites, never above the character's level.
			return !w.Opt.SoloElites || src.NPC.MaxLevel > level
		}
		return max(src.NPC.MaxLevel, mobLevel(src.NPC)) > lvl
	}
	return w.danger.at(src.Site.Center) > lvl
}
