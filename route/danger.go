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

// at is the danger level of a place (0: nothing known).
func (d dangerMap) at(p dataset.Point) int {
	return d[[3]int{p.Map, int(math.Floor(p.X / dangerCell)), int(math.Floor(p.Y / dangerCell))}]
}

// tooDangerous tells whether the character cannot work at a source yet: a creature above its fighting range, or a place
// among such creatures.
func (w *World) tooDangerous(s *State, src Source) bool {
	lvl := w.Level(s) + w.Opt.MaxMobAbove
	if src.Talk {
		return false
	}
	if src.NPC != nil {
		return mobLevel(src.NPC) > lvl
	}
	return w.danger.at(src.Site.Center) > lvl
}
