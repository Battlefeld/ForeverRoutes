package route

import (
	"math"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Wands. Mages, priests and warlocks finish their mobs with a wand, which costs no mana: fewer rests. From level 5
// they craft the Lesser Magic Wand with Enchanting (12-22 arcane, speed 1.5; foreverchanges.pro/item/11287); later a
// quest's wand may be better (cmangos quest rewards: the wands of quests up to level 22, damage per second).
const (
	lesserWandLevel = 5
	lesserWandDPS   = 11.3
	// wandCraft: learning Enchanting and crafting the Lesser Magic Wand (Simple Wood, Lesser Magic Essence from the
	// enchanting vendor), once at level 5 (an estimate; done where the character is)
	wandCraft = 600.0
)

var wandClasses = map[string]bool{"Mage": true, "Priest": true, "Warlock": true}

var wandQuests = map[int]float64{
	14:   12.8, // The People's Militia: Spark of the People's Militia
	99:   11.8, // Arugal's Folly: Cinder Wand
	104:  15.8, // The Coastal Menace: Torchlight Wand
	297:  13.6, // Gathering Idols: Dwarven Flamestick
	516:  15.7, // Beren's Peril: Wand of Decay
	567:  22.2, // Dangerous!: Charred Wand
	863:  13.7, // The Escape: Flaring Baton
	873:  21.6, // Isha Awak: Branding Rod
	957:  9.4,  // Bashal'Aran: Elven Wand
	1078: 20.3, // Retrieval for Mauren: Spellcrafter Wand
	1200: 29.0, // Blackfathom Villainy: Gravestone Scepter
	1487: 17.6, // Deviate Eradication (Wailing Caverns): Sizzle Stick
	2040: 16.9, // Underground Assault (Gnomeregan): Sable Wand
	4763: 13.6, // The Blackwood Corrupted: Moonstone Wand
	6561: 29.0, // Blackfathom Villainy: Gravestone Scepter
}

// WandProfile is a kill profile that knows how fights go when the character finishes mobs with a wand.
type WandProfile interface {
	KillSecondsWand(c Character, playerLevel int, mob *dataset.NPC, wandDPS float64) (fight, rest float64)
}

// wandDPS is the best wand this character has in s, 0 for none.
func (w *World) wandDPS(s *State) float64 {
	if !wandClasses[w.Char.Class] || w.Level(s) < lesserWandLevel {
		return 0
	}
	best := lesserWandDPS
	for id, dps := range wandQuests {
		if q := w.ByID[id]; q != nil && dps > best && s.Status[q.Index] == done {
			best = dps
		}
	}
	return best
}

// craftWand: a caster crafts the Lesser Magic Wand before its first fight from level 5.
func (w *World) craftWand(s *State) {
	if s.Wand || !wandClasses[w.Char.Class] || w.Level(s) < lesserWandLevel {
		return
	}
	s.Wand = true
	s.Time += wandCraft
	s.Talk += wandCraft
}

// KillSecondsWand: the fastest way to kill the mob with the wand at hand, rest included: spells only, or spells for a
// share of its health and the wand for the rest (no mana). A wand misses 5 % and 5 % more per level the mob is above.
func (p *TableProfile) KillSecondsWand(c Character, pl int, m *dataset.NPC, dps float64) (float64, float64) {
	f, r := p.KillSeconds(c, pl, m)
	l := p.byLevel[pl]
	hp := float64(m.HP)
	if hp <= 0 {
		hp = typical(p.Fallback.EvenHP, m.MaxLevel, 30+12*float64(pl)+0.7*float64(pl*pl))
	}
	eff := dps * math.Max(0.5, 0.95-0.05*math.Max(0, float64(m.MaxLevel-pl)))
	if l == nil {
		// below the table: half by spells, half by wand, half the mana
		wf := f/2 + hp/2/eff
		if wf+r/2 < f+r {
			return wf, r/2 + 2
		}
		return f, r
	}
	wl := math.Max(0, float64(m.MaxLevel-pl)) / 2
	food, drink := restRates(pl)
	best, bf, br := f+r, f, r
	for _, share := range []float64{0, 0.25, 0.5, 0.75} {
		fight, mana, taken := 0.0, 0.0, 0.0
		if share > 0 {
			t0, mana0, self0 := solve(l.Targets["0"], hp*share)
			t2, mana2, self2 := solve(l.Targets["2"], hp*share)
			fight = t0 + wl*(t2-t0)
			mana = math.Min(math.Max(0, mana0+wl*(mana2-mana0)), l.Mana)
			taken = math.Max(0, self0+wl*(self2-self0))
		}
		fight += hp * (1 - share) / eff
		if m.Dmg > 0 {
			taken += m.Dmg / 2 * fight * p.meleeShare(fight)
		}
		taken = math.Min(taken, l.Health)
		rest := taken / food
		if l.Mana > 0 {
			rest = math.Max(rest, mana/drink)
		}
		rest += 4
		if fight+rest < best {
			best, bf, br = fight+rest, fight, rest
		}
	}
	return bf, br
}
