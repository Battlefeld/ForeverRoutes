// Package route plans and replays quest routes for one character on the route dataset: which quests to accept, where
// to kill what, when to turn in and when grinding is faster. Every route is a list of plain actions that Replay checks
// step by step with the same rules, so a planned route and a replayed one always agree.
package route

import (
	"math"
	"sort"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Race and class bits of the Classic client (quest masks use them). Forever's Skyborne play on either side and have a
// bit per side past the classic races (SkyborneBits); Character.Faction picks it.
var RaceBits = map[string]int{"Human": 1, "Orc": 2, "Dwarf": 4, "NightElf": 8, "Undead": 16, "Scourge": 16, "Tauren": 32, "Gnome": 64, "Troll": 128}
var SkyborneBits = map[string]int{"A": 1 << 32, "H": 1 << 33}
var ClassBits = map[string]int{"Warrior": 1, "Paladin": 2, "Hunter": 4, "Rogue": 8, "Priest": 16, "Shaman": 64, "Mage": 128, "Warlock": 256, "Druid": 1024}

// FactionOf is the faction of a race (A or H); the Skyborne have none of their own (Character.Faction).
func FactionOf(race string) string {
	switch race {
	case "Orc", "Undead", "Scourge", "Tauren", "Troll":
		return "H"
	}
	return "A"
}

// StartNPC is the first quest giver of each start area (where a new character stands). The Skyborne start in Thendal
// Village on Zephras Isle (Ailee Farheart).
var StartNPC = map[string]int{"Human": 823, "Orc": 10176, "Troll": 10176, "Dwarf": 658, "Gnome": 658, "NightElf": 2079, "Undead": 1569, "Scourge": 1569, "Tauren": 2981, "Skyborne": 251362}

// Character is who walks the route.
type Character struct {
	Race    string `json:"race"`
	Class   string `json:"class"`
	Faction string `json:"faction,omitempty"` // A or H (Alliance, Horde also read); needed for the Skyborne only
}

// side is the character's faction (A or H) and its race bit; ok is false for an unknown race or a Skyborne without a
// faction.
func (c Character) side() (faction string, race int, ok bool) {
	if c.Race == "Skyborne" {
		f := map[string]string{"A": "A", "Alliance": "A", "H": "H", "Horde": "H"}[c.Faction]
		return f, SkyborneBits[f], f != ""
	}
	race = RaceBits[c.Race]
	if c.Faction != "" {
		f := map[string]string{"A": "A", "Alliance": "A", "H": "H", "Horde": "H"}[c.Faction]
		if f == "" || f != FactionOf(c.Race) {
			return "", 0, false
		}
	}
	return FactionOf(c.Race), race, race != 0
}

// KillProfile tells how long the character needs for one mob. ForeverCore supplies MythicSim's numbers; without it the
// default profile below is used, which is a rough estimate and marked as such in every result.
type KillProfile interface {
	// KillSeconds is the fight time and the rest time after it for one mob.
	KillSeconds(c Character, playerLevel int, mob *dataset.NPC) (fight, rest float64)
	Name() string
}

// DefaultProfile is a stand-in until ForeverCore supplies MythicSim's numbers. A normal mob of the character's own
// level takes 8 s at level 1 up to 15.5 s at level 30; other mobs scale with their health against the typical health
// of a mob of the character's level, but never below 60 % of that (pulling and casting take their time), and mobs above
// the character take 25 % longer per level (more misses and resists, more damage taken). Resting, looting and pulling
// the next one adds 45 % of the fight plus 4 s, scaled by how hard the mob hits against a typical mob of the
// character's level. This gives about 20k XP per hour grinding even-level mobs at level 20, in line with common
// experience; it is not measured. The typical values (EvenHP, EvenDmg) are filled from the dataset by NewWorld.
type DefaultProfile struct {
	EvenHP  []float64 // EvenHP[l] = typical health of a normal mob of level l
	EvenDmg []float64 // EvenDmg[l] = typical melee hit of a normal mob of level l
}

func (DefaultProfile) Name() string { return "default (uncalibrated)" }
func (p DefaultProfile) KillSeconds(c Character, pl int, m *dataset.NPC) (float64, float64) {
	even := 8 + 0.25*float64(pl)
	hp, ref := float64(m.HP), typical(p.EvenHP, pl, 30+12*float64(pl)+0.7*float64(pl*pl))
	if hp <= 0 {
		hp = typical(p.EvenHP, m.MaxLevel, ref)
	}
	f := even * math.Max(0.6, hp/ref)
	d := float64(m.MaxLevel - pl)
	if d > 0 {
		f *= 1 + 0.25*d
	}
	hit := 1.0
	if ref := typical(p.EvenDmg, pl, 0); m.Dmg > 0 && ref > 0 {
		hit = math.Min(2.5, math.Max(0.6, m.Dmg/ref))
	}
	return f, (f*0.45 + 4) * hit
}

// typical is v[l] when known, else def.
func typical(v []float64, l int, def float64) float64 {
	if l < 1 {
		l = 1
	}
	if l < len(v) && v[l] > 0 {
		return v[l]
	}
	return def
}

// medianByLevel is the median of a value of normal creatures per level (levels with at least 5 creatures).
func medianByLevel(d *dataset.Dataset, val func(*dataset.NPC) float64) []float64 {
	by := map[int][]float64{}
	maxL := 0
	for i := range d.NPCs {
		n := &d.NPCs[i]
		if v := val(n); n.Rank == 0 && v > 0 && n.MaxLevel > 0 && n.MaxLevel <= 63 {
			by[n.MaxLevel] = append(by[n.MaxLevel], v)
			maxL = max(maxL, n.MaxLevel)
		}
	}
	out := make([]float64, maxL+1)
	for l, vs := range by {
		if len(vs) >= 5 {
			sort.Float64s(vs)
			out[l] = vs[len(vs)/2]
		}
	}
	return out
}

// Options of the model; zero values take the defaults.
type Options struct {
	Preparations PreparationConstraints
	SleepingBag  SleepingBagModel
	Feedback     FeedbackSettings
	RunSpeed     float64            // yards per second (7 = Classic run speed)
	PathFactor   float64            // real paths are longer than straight lines (1.2)
	QuestLogCap  int                // 20 in Classic
	InteractSecs float64            // talking to an NPC (accept / turn in)
	Profile      KillProfile        // nil = DefaultProfile
	MaxMobAbove  int                // mobs more than this many levels above the character are not attacked (3; melee and shamans 1)
	SoloElites   bool               // plain elites of the character's level or below are fought alone (warlocks and hunters: the pet tanks)
	QuestAtLevel bool               // a quest's objectives are worked only from the quest's level on (off by default)
	NoClassRules bool               // keep the options as given instead of the class's rules
	AssumedDrop  float64            // chance per kill assumed where a source names the creature but no chance (0.3); reported in the result
	Walks        *dataset.Walks     // walking costs measured on navigation maps (nil: straight lines times PathFactor)
	WalkTable    *dataset.WalkTable `json:"-"` // walks measured between the places routes go to (nil: the walk grid for all)
	Dungeons     bool               // dungeon runs are allowed; the optimizer may choose none
	GroupSearch  float64            // seconds to find a group for a dungeon run (600; tanks and healers 60)
}

func (o Options) withDefaults() Options {
	if o.SleepingBag == (SleepingBagModel{}) {
		o.SleepingBag = DefaultSleepingBagModel()
	}
	if o.RunSpeed == 0 {
		o.RunSpeed = 7
	}
	if o.PathFactor == 0 {
		o.PathFactor = 1.2
	}
	if o.QuestLogCap == 0 {
		o.QuestLogCap = 20
	}
	if o.InteractSecs == 0 {
		o.InteractSecs = 5
	}
	if o.Profile == nil {
		o.Profile = DefaultProfile{}
	}
	if o.MaxMobAbove == 0 {
		o.MaxMobAbove = 3
	}
	if o.AssumedDrop == 0 {
		o.AssumedDrop = 0.3
	}
	if o.GroupSearch == 0 {
		o.GroupSearch = groupSearch
	}
	return o
}

// distance in yards; positions on different continents are unreachable (Inf) in this version.
func distance(a, b dataset.Point) float64 {
	if a.Map != b.Map {
		return math.Inf(1)
	}
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}

// classRules sets what a class does alone: warlocks and hunters fight plain elites of their level with the pet as
// tank; melee classes and shamans fight at most one level up, so their route reaches an area when its mobs are at
// their level (kills of higher mobs take longer, so the planner prefers even or lower ones), which keeps fights short
// and rests rare. The
// classes that can tank or heal find a group for a dungeon almost at once.
func (w *World) classRules() {
	if w.Opt.NoClassRules {
		return
	}
	switch w.Char.Class {
	case "Warlock", "Hunter":
		w.Opt.SoloElites = true
	case "Warrior", "Rogue", "Paladin", "Shaman":
		w.Opt.MaxMobAbove = 1
	}
	switch w.Char.Class {
	case "Warrior", "Paladin", "Shaman", "Priest", "Druid":
		w.Opt.GroupSearch = groupSearchRole
	}
}
