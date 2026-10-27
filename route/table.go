package route

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"sync"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// KillProfileSchema is the file ForeverCore writes (job "kill-profile"): MythicSim with ForeverCore's level overlay.
const KillProfileSchema = "foreverroutes.killprofile.v1"

// TableProfile takes the fight from MythicSim: for every character level the damage dealt and the mana and health
// spent after so many seconds of fighting a mob of the same level and one two levels above. The fight lasts until the
// damage reaches the mob's health. The rest after it is eating and drinking back what the fight cost: the mob's melee
// damage (cmangos' average hit, one swing per 2 s, for the share of the fight the mob reaches the character) plus the
// health the character spends itself, and the mana spent, at the rates of the Classic vendor food and water of the
// level; plus 4 s for looting and pulling the next mob, as in DefaultProfile. Levels the table does not cover fall
// back to DefaultProfile.
type TableProfile struct {
	CurveWarnings []string       `json:"curveWarnings,omitempty"`
	Schema        string         `json:"schema"`
	Source        string         `json:"source"`
	Class         string         `json:"class"`
	Race          string         `json:"race"`
	Rotation      string         `json:"rotation"`
	ClassOptions  map[string]any `json:"classOptions"`
	Gear          string         `json:"gear"`
	Talents       string         `json:"talents"`
	Levels        []TableLevel   `json:"levels"`

	// MeleeShare is the share of the fight the mob hits the character. Without a value: 0.25 when a voidwalker
	// tanks (it takes the mob after the first hits), else 1 minus the time the mob needs to run to a caster.
	MeleeShare float64        `json:"-"`
	Fallback   DefaultProfile `json:"-"`
	byLevel    map[int]*TableLevel
	edgeOnce   sync.Once
	edge       [2][2]float64 // fight and rest factors of the fallback at the table's first and last level
}

// TableLevel is one character level.
type TableLevel struct {
	Level   int                     `json:"level"`
	Health  float64                 `json:"health"`
	Mana    float64                 `json:"mana"`
	Targets map[string][]TablePoint `json:"targets"` // key: target level minus character level
}

// TablePoint is one fight length.
type TablePoint struct {
	Seconds     float64 `json:"seconds"`
	Damage      float64 `json:"damage"`
	ManaSpent   float64 `json:"manaSpent"`
	HealthSpent float64 `json:"healthSpent"`
}

// LoadTableProfile reads and checks a kill profile.
func LoadTableProfile(r io.Reader) (*TableProfile, error) {
	var p TableProfile
	if err := json.NewDecoder(r).Decode(&p); err != nil {
		return nil, err
	}
	if p.Schema != KillProfileSchema {
		return nil, fmt.Errorf("kill profile: schema %q, want %q", p.Schema, KillProfileSchema)
	}
	known := false
	for c := range ClassBits {
		known = known || strings.EqualFold(c, p.Class)
	}
	if !known {
		return nil, fmt.Errorf("kill profile: unknown class %q", p.Class)
	}
	p.byLevel = map[int]*TableLevel{}
	for i := range p.Levels {
		l := &p.Levels[i]
		if l.Level < 1 || l.Level > 60 || p.byLevel[l.Level] != nil || l.Health <= 0 || l.Mana < 0 || math.IsNaN(l.Health+l.Mana) || math.IsInf(l.Health+l.Mana, 0) {
			return nil, fmt.Errorf("kill profile: invalid or duplicate level %d", l.Level)
		}
		for _, off := range []string{"0", "2"} {
			pts := l.Targets[off]
			sort.Slice(pts, func(a, b int) bool { return pts[a].Seconds < pts[b].Seconds })
			if len(pts) == 0 || pts[0].Seconds <= 0 || pts[len(pts)-1].Damage <= 0 {
				return nil, fmt.Errorf("kill profile: level %d has no fight data for target %+s", l.Level, off)
			}
			previous := TablePoint{}
			for _, point := range pts {
				if math.IsNaN(point.Seconds+point.Damage+point.ManaSpent+point.HealthSpent) || math.IsInf(point.Seconds+point.Damage+point.ManaSpent+point.HealthSpent, 0) || point.Seconds <= previous.Seconds || point.Damage < 0 {
					return nil, fmt.Errorf("kill profile: invalid/non-monotone curve at level %d target %s", l.Level, off)
				}
				previous = point
			}
			// Separate duration simulations can report less cumulative damage later.
			// Use the conservative monotone minorant: lower earlier damage, never raise it.
			adjusted := false
			for j := len(pts) - 2; j >= 0; j-- {
				if pts[j].Damage > pts[j+1].Damage {
					pts[j].Damage = pts[j+1].Damage
					adjusted = true
				}
			}
			if adjusted {
				p.CurveWarnings = append(p.CurveWarnings, fmt.Sprintf("level %d target %s: conservative monotone damage envelope", l.Level, off))
			}
			if len(pts) > 1 && pts[len(pts)-1].Damage <= pts[len(pts)-2].Damage {
				return nil, fmt.Errorf("kill profile: level %d target %s has no terminal damage rate", l.Level, off)
			}
		}
		p.byLevel[l.Level] = l
	}
	if len(p.byLevel) == 0 {
		return nil, fmt.Errorf("kill profile without levels")
	}
	return &p, nil
}

// Fits tells whether the profile was made for this class.
func (p *TableProfile) Fits(c Character) bool { return strings.EqualFold(p.Class, c.Class) }

func (p *TableProfile) Name() string {
	s := fmt.Sprintf("MythicSim %s %s (gear %s, talents %s; outside the table: %s scaled to the table's edge)", p.Class, p.Rotation, p.Gear, p.Talents, p.Fallback.Name())
	if len(p.CurveWarnings) > 0 {
		s += fmt.Sprintf("; %d conservative curve corrections", len(p.CurveWarnings))
	}
	return s
}

// forWorld creates an independent fallback and edge cache. A supplied profile can be shared by callers safely.
func (p *TableProfile) forWorld(d *dataset.Dataset) *TableProfile {
	q := &TableProfile{Schema: p.Schema, Source: p.Source, Class: p.Class, Race: p.Race, Rotation: p.Rotation, ClassOptions: p.ClassOptions, Gear: p.Gear, Talents: p.Talents, Levels: p.Levels, MeleeShare: p.MeleeShare, Fallback: p.Fallback, byLevel: p.byLevel, CurveWarnings: p.CurveWarnings}
	if q.Fallback.EvenHP == nil {
		q.Fallback.EvenHP = medianByLevel(d, func(n *dataset.NPC) float64 { return float64(n.HP) })
		q.Fallback.EvenDmg = medianByLevel(d, func(n *dataset.NPC) float64 { return n.Dmg })
	}
	return q
}

func (p *TableProfile) KillSeconds(c Character, pl int, m *dataset.NPC) (float64, float64) {
	l := p.byLevel[pl]
	if l == nil {
		return p.outside(c, pl, m)
	}
	hp := float64(m.HP)
	if hp <= 0 {
		hp = typical(p.Fallback.EvenHP, m.MaxLevel, 30+12*float64(pl)+0.7*float64(pl*pl))
	}
	// Between the same-level and the +2 curve by the mob's level; three above extrapolates, lower mobs take the
	// same-level curve.
	w := math.Max(0, float64(m.MaxLevel-pl)) / 2
	t0, mana0, self0 := solve(l.Targets["0"], hp)
	t2, mana2, self2 := solve(l.Targets["2"], hp)
	fight := t0 + w*(t2-t0)
	mana := math.Min(math.Max(0, mana0+w*(mana2-mana0)), l.Mana)
	taken := math.Max(0, self0+w*(self2-self0))
	if m.Dmg > 0 {
		taken += m.Dmg / 2 * fight * p.meleeShare(fight)
	}
	taken = math.Min(taken, l.Health)
	food, drink := restRates(pl)
	rest := taken / food
	if l.Mana > 0 {
		rest = math.Max(rest, mana/drink)
	}
	return fight, rest + 4
}

// outside: a level the table does not have (the engine starts at level 10). The fallback estimate gives the shape
// over the levels; it is scaled so that it agrees with the table at the table's nearest level, for an even-level mob of
// typical health. Without the scaling the fallback rates grinding at low levels about twice as high as the table.
func (p *TableProfile) outside(c Character, pl int, m *dataset.NPC) (float64, float64) {
	lo, hi := p.LevelRange()
	p.edgeOnce.Do(func() {
		for i, e := range [2]int{lo, hi} {
			ref := &dataset.NPC{MinLevel: e, MaxLevel: e}
			tf, tr := p.KillSeconds(c, e, ref)
			df, dr := p.Fallback.KillSeconds(c, e, ref)
			p.edge[i] = [2]float64{1, 1}
			if df > 0 && dr > 0 {
				p.edge[i] = [2]float64{tf / df, tr / dr}
			}
		}
	})
	k := p.edge[0]
	if pl > hi {
		k = p.edge[1]
	}
	f, r := p.Fallback.KillSeconds(c, pl, m)
	return f * k[0], r * k[1]
}

func (p *TableProfile) meleeShare(fight float64) float64 {
	if p.MeleeShare > 0 {
		return p.MeleeShare
	}
	if v, _ := p.ClassOptions["summon"].(float64); strings.EqualFold(p.Class, "warlock") && v == 2 {
		return 0.25
	}
	// A caster pulls from about 25 yd; the mob needs about 3 s to arrive.
	return math.Max(0, fight-3) / fight
}

// solve is the time at which the curve's damage reaches hp, and the mana and health spent until then. Before the first
// point the curve starts at zero; after the last it goes on with the rate between the last two points.
func solve(pts []TablePoint, hp float64) (t, mana, self float64) {
	prev := TablePoint{}
	for i, p := range pts {
		if p.Damage >= hp || i == len(pts)-1 {
			f := 1.0
			if d := p.Damage - prev.Damage; d > 0 {
				f = (hp - prev.Damage) / d
			}
			lerp := func(a, b float64) float64 { return a + f*(b-a) }
			return lerp(prev.Seconds, p.Seconds), lerp(prev.ManaSpent, p.ManaSpent), lerp(prev.HealthSpent, p.HealthSpent)
		}
		prev = p
	}
	return 0, 0, 0
}

// restRates is health and mana per second from the Classic vendor food and water a character of the level can use:
// Tough Hunk of Bread / Refreshing Spring Water (61 health, 151 mana over 18 s), from level 15 Freshly Baked Bread /
// Ice Cold Milk (243, 437 over 21 s), 25 Moist Cornbread / Melon Juice (552, 835 over 24 s), 35 Mulgore Spice Bread /
// Sweet Nectar (874, 1344 over 27 s), 45 Soft Banana Bread / Moonberry Juice (1392, 1992 over 30 s), 55 Homemade
// Cherry Pie / Morning Glory Dew (2148, 2934 over 30 s).
func restRates(level int) (health, mana float64) {
	tiers := []struct {
		level            int
		health, mana, in float64
	}{{55, 2148, 2934, 30}, {45, 1392, 1992, 30}, {35, 874, 1344, 27}, {25, 552, 835, 24}, {15, 243, 437, 21}, {0, 61, 151, 18}}
	for _, t := range tiers {
		if level >= t.level {
			return t.health / t.in, t.mana / t.in
		}
	}
	return 61.0 / 18, 151.0 / 18
}

// LevelRange is the first and last level of the table.
func (p *TableProfile) LevelRange() (int, int) {
	lo, hi := math.MaxInt, 0
	for l := range p.byLevel {
		lo, hi = min(lo, l), max(hi, l)
	}
	return lo, hi
}
