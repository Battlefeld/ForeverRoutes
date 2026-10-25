package route

import (
	"math"
	"strings"
	"sync"
	"sync/atomic"

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
	known  []dataset.Deathskip // the guides' deathskips this character may use
	memo   sync.Map            // skipKey -> skipResult: the planner asks the same ways again and again
	memoN  atomic.Int64
}

type skipKey struct {
	a, b  dataset.Point
	known uint64 // flight masters known (the way on from the graveyard)
	level int    // resurrection sickness
}

type skipResult struct {
	t  float64
	g  int
	at dataset.Point
}

// memoMax bounds the memo; it is cleared when full.
const memoMax = 1 << 21

func newDeaths(t dataset.Travel, char Character, faction string) *deaths {
	if t.Deaths == nil || len(t.Graveyards) == 0 {
		return nil
	}
	d := &deaths{cell: t.Deaths.Cell, grave: map[[3]int][2]int32{}, graves: t.Graveyards}
	for _, k := range t.Deathskips {
		if k.Grave >= 0 && k.Grave < len(t.Graveyards) && filterFits(k.Filter, char, faction) {
			d.known = append(d.known, k)
		}
	}
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
func (w *World) deathskip(s *State, a, b dataset.Point, limit float64) (float64, int) {
	d := w.deaths
	k := skipKey{a: a, b: b, known: s.Known, level: w.Level(s)}
	if r, ok := d.memo.Load(k); ok {
		return r.(skipResult).t, r.(skipResult).g
	}
	t, g, at := w.deathskipAt(s, a, b, math.Inf(1))
	if d.memoN.Add(1) > memoMax {
		d.memo.Clear()
		d.memoN.Store(0)
	}
	d.memo.Store(k, skipResult{t, g, at})
	return t, g
}

// deathskipAt is deathskip with the place to die. Ways that cannot beat limit are not followed (the result is then
// Inf or a time at or above limit).
func (w *World) deathskipAt(s *State, a, b dataset.Point, limit float64) (float64, int, dataset.Point) {
	d := w.deaths
	if d == nil {
		return math.Inf(1), -1, a
	}
	at := a
	side := 0
	if w.Faction == "H" {
		side = 1
	}
	sick := sickness(w.Level(s))
	best, bestG := limit, -1
	tried := map[int32]bool{}
	ci, cj := int(math.Floor(a.X/d.cell)), int(math.Floor(a.Y/d.cell))
	try := func(g int32, die float64, where dataset.Point) {
		if g <= 0 || tried[g] {
			return
		}
		tried[g] = true
		gp := d.graves[g-1].At
		if gp.Map != b.Map || die+deathSecs >= best {
			return
		}
		after, _ := w.overland(s, gp, b)
		if t := die + deathSecs + after + math.Max(0, sick-after); t < best {
			best, bestG, at = t, int(g-1), where
		}
	}
	if gs, ok := d.grave[[3]int{a.Map, ci, cj}]; ok {
		try(gs[side], 0, a)
	}
	for di := -deathReach; di <= deathReach; di++ {
		for dj := -deathReach; dj <= deathReach; dj++ {
			if gs, ok := d.grave[[3]int{a.Map, ci + di, cj + dj}]; ok && gs[side] > 0 && !tried[gs[side]] {
				centre := dataset.Point{Map: a.Map, X: (float64(ci+di) + 0.5) * d.cell, Y: (float64(cj+dj) + 0.5) * d.cell}
				if distance(a, centre)/w.Opt.RunSpeed+deathSecs >= best {
					continue // walking there alone takes longer than the best way
				}
				try(gs[side], w.walk(a, centre), centre)
			}
		}
	}
	// the guides' deathskips: walk to where they die
	for _, k := range d.known {
		if k.At.Map != a.Map || d.graves[k.Grave].At.Map != b.Map || distance(a, k.At)/w.Opt.RunSpeed+deathSecs >= best {
			continue
		}
		die := w.walk(a, k.At)
		if die+deathSecs >= best {
			continue
		}
		g := d.graves[k.Grave].At
		after, _ := w.overland(s, g, b)
		if t := die + deathSecs + after + math.Max(0, sick-after); t < best {
			best, bestG, at = t, k.Grave, k.At
		}
	}
	if bestG < 0 {
		return math.Inf(1), -1, a
	}
	return best, bestG, at
}

// filterFits reads a guide step's filter: alternatives split by "/", each a list of words that must all fit (a race,
// a class, Alliance or Horde; "!" negates). Words it does not know fit.
func filterFits(filter string, c Character, faction string) bool {
	if strings.TrimSpace(filter) == "" {
		return true
	}
	race := c.Race
	if race == "Undead" {
		race = "Scourge"
	}
	side := map[string]string{"A": "Alliance", "H": "Horde"}[faction]
	word := func(w string) bool {
		switch {
		case strings.EqualFold(w, c.Class), strings.EqualFold(w, side), strings.EqualFold(w, race), strings.EqualFold(w, c.Race):
			return true
		case ClassBits[w] != 0, RaceBits[w] != 0, w == "Alliance", w == "Horde", w == "Skyborne", w == "Scourge", w == "Undead":
			return false
		}
		return true
	}
	for _, alt := range strings.Split(filter, "/") {
		ok := true
		for _, w := range strings.Fields(alt) {
			if neg := strings.HasPrefix(w, "!"); neg {
				if word(w[1:]) && known(w[1:]) {
					ok = false
				}
			} else if !word(w) {
				ok = false
			}
		}
		if ok {
			return true
		}
	}
	return false
}

func known(w string) bool {
	return ClassBits[w] != 0 || RaceBits[w] != 0 || w == "Alliance" || w == "Horde" || w == "Skyborne" || w == "Scourge" || w == "Undead"
}
