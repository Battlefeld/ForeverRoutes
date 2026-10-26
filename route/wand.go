package route

import (
	"math"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Wands. Mages, priests and warlocks finish their mobs with a wand, which costs no mana: fewer rests. The wands are
// crafted with Enchanting, whose trainers and suppliers stand in the capitals (cmangos): the Novice's Practice Wand
// (new in Forever, Enchanting 1, Simple Wood and a Mote of Magic from the supplier; 5-8 arcane, speed 1.5) from level
// 1, the Lesser Magic Wand (Enchanting 10, reached with practice wands; Simple Wood and a Lesser Magic Essence from the
// auction house; 12-22 arcane, speed 1.5; foreverchanges.pro/item/11287), worn from level 5. One visit crafts both:
// the practice wand until level 5, the Lesser Magic Wand from then on. A route goes for them when a capital is close
// enough (wandDetour); later a quest's wand may be better (cmangos quest rewards: the wands of quests up to level 22,
// damage per second).
const (
	practiceWandDPS = 4.3
	lesserWandLevel = 5
	lesserWandDPS   = 11.3
	// wandCraft: time at the trainer, supplier and auction house: learning Enchanting, practice wands for the skill to
	// 10, the Lesser Magic Wand
	wandCraft = 300.0
)

// WandDetours: the most time, in seconds, a visit to a capital for the wands may add to the way to the route's next
// step, tried in turn by ChooseWandDetour: only where a capital is on the way (on leaving the start area), up to a
// way back and forth (the hearthstone home).
var WandDetours = []float64{60, 180, 420, 900}

// wandVisitLevel: the first level a caster goes for its wands (before it, it has not the money for the essence).
const wandVisitLevel = lesserWandLevel - 1

// defaultWandDetour: the detour when none is chosen.
const defaultWandDetour = 420.0

// SetWandDetour sets the detour for the wands a route was made with (0: defaultWandDetour).
func (w *World) SetWandDetour(d float64) { w.wandDetour = d }

// WandDetour is the detour for the wands the route is walked with.
func (w *World) WandDetour() float64 { return w.wandDetour }

// ChooseWandDetour sets the detour for the wands (WandDetours) under which the guide reaches the target soonest.
func (w *World) ChooseWandDetour(events []GuideEvent, target int) float64 {
	if !wandClasses[w.Char.Class] {
		return 0
	}
	best, bt := defaultWandDetour, math.Inf(1)
	for _, d := range WandDetours {
		w.wandDetour = d
		if r, err := w.FollowGuide(events, target); err == nil && r.Seconds < bt {
			best, bt = d, r.Seconds
		}
	}
	w.wandDetour = best
	return best
}

// wandSites: the Enchanting trainer and supplier of each capital (cmangos spawns), by side; their capitals have an
// auction house.
var wandSites = map[string][]dataset.Point{
	"A": {{Map: 0, X: -8857, Y: 805}, {Map: 0, X: -4805, Y: -1191}, {Map: 1, X: 10140, Y: 2319}}, // Stormwind, Ironforge, Darnassus
	"H": {{Map: 1, X: 1915, Y: -4436}, {Map: 1, X: -1115, Y: 48}, {Map: 0, X: 1485, Y: 275}},     // Orgrimmar, Thunder Bluff, Undercity
}

var wandClasses = map[string]bool{"Mage": true, "Priest": true, "Warlock": true}

// WandClass: whether the class finishes its mobs with a wand.
func WandClass(class string) bool { return wandClasses[class] }

// LesserWandItem is the Lesser Magic Wand's item.
const LesserWandItem = 11287

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
	if !wandClasses[w.Char.Class] {
		return 0
	}
	best := 0.0
	switch {
	case s.Wand && w.Level(s) >= lesserWandLevel:
		best = lesserWandDPS
	case s.Wand:
		best = practiceWandDPS
	}
	for id, dps := range wandQuests {
		if q := w.ByID[id]; q != nil && dps > best && s.Status[q.Index] == done {
			best = dps
		}
	}
	return best
}

// wandAction: the visit to a capital for the wands on the way to next (the route's next step) when it adds at most
// the detour chosen (ChooseWandDetour), else nil.
func (w *World) wandAction(s *State, next dataset.Point) *Action {
	if s.Wand || !wandClasses[w.Char.Class] || w.Level(s) < wandVisitLevel {
		return nil
	}
	direct, _ := w.travel(s, s.Pos, next)
	var best *dataset.Point
	bt := w.wandDetour
	if bt == 0 {
		bt = defaultWandDetour
	}
	for i, p := range wandSites[w.Faction] {
		if p.Map != s.Pos.Map && p.Map != next.Map {
			continue
		}
		there, _ := w.travel(s, s.Pos, p)
		on, _ := w.travel(s, p, next)
		if extra := there + on - direct; extra <= bt {
			best, bt = &wandSites[w.Faction][i], extra
		}
	}
	if best == nil {
		return nil
	}
	return &Action{Op: "wand", At: *best}
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
