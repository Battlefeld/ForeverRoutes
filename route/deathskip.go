package route

import (
	"math"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Deathskips: die on purpose, release and take the Spirit Healer's resurrection at the graveyard the server gives for
// the place of death (dataset.DeathGrid), then go on from there. The death can be where the character stands or a short
// walk away, where the server gives another graveyard. It costs deathSecs (being killed, releasing, the healer) and,
// from level 11 on, resurrection sickness: Forever gives none through level 10; above, a minute per level over 10, ten
// at most, as in Classic. The character can travel while sick but not fight well, so the part of the sickness still
// left on arrival is paid as waiting. A deathskip is taken only when it saves deathMargin over the other ways.

const (
	deathSecs    = 60.0
	deathMargin  = 30.0
	deathMinLeg  = 120.0 // only legs longer than this are worth checking
	deathReach   = 2     // cells around the start where the death may be
	sickFreeUpTo = 10
)

type deaths struct {
	cell   float64
	grave  map[[3]int][2]int32 // map, i, j -> graveyard index + 1 for Alliance, Horde
	graves []dataset.Node
}

func newDeaths(t dataset.Travel) *deaths {
	if t.Deaths == nil || len(t.Graveyards) == 0 {
		return nil
	}
	d := &deaths{cell: t.Deaths.Cell, grave: map[[3]int][2]int32{}, graves: t.Graveyards}
	c := t.Deaths.Cells
	for k := 0; k+4 < len(c); k += 5 {
		d.grave[[3]int{c[k], c[k+1], c[k+2]}] = [2]int32{int32(c[k+3]), int32(c[k+4])}
	}
	return d
}

// sickness is how long resurrection sickness lasts at a level.
func sickness(level int) float64 {
	if level <= sickFreeUpTo {
		return 0
	}
	return math.Min(600, float64(level-sickFreeUpTo)*60)
}

// deathskip is the fastest deathskip from a to b: its time and the graveyard (-1: none).
func (w *World) deathskip(s *State, a, b dataset.Point) (float64, int) {
	d := w.deaths
	if d == nil {
		return math.Inf(1), -1
	}
	side := 0
	if w.Faction == "H" {
		side = 1
	}
	sick := sickness(w.Level(s))
	best, bestG := math.Inf(1), -1
	tried := map[int32]bool{}
	ci, cj := int(math.Floor(a.X/d.cell)), int(math.Floor(a.Y/d.cell))
	try := func(g int32, die float64) {
		if g <= 0 || tried[g] {
			return
		}
		tried[g] = true
		at := d.graves[g-1].At
		if at.Map != b.Map {
			return
		}
		after, _ := w.overland(s, at, b)
		if t := die + deathSecs + after + math.Max(0, sick-after); t < best {
			best, bestG = t, int(g-1)
		}
	}
	if gs, ok := d.grave[[3]int{a.Map, ci, cj}]; ok {
		try(gs[side], 0)
	}
	for di := -deathReach; di <= deathReach; di++ {
		for dj := -deathReach; dj <= deathReach; dj++ {
			if gs, ok := d.grave[[3]int{a.Map, ci + di, cj + dj}]; ok && !tried[gs[side]] {
				centre := dataset.Point{Map: a.Map, X: (float64(ci+di) + 0.5) * d.cell, Y: (float64(cj+dj) + 0.5) * d.cell}
				try(gs[side], w.walk(a, centre))
			}
		}
	}
	return best, bestG
}
