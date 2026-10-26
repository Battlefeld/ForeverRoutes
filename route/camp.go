package route

import (
	"math"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Camping. Every character learns Skinning, Cooking and Leatherworking on the way (casters at the capital they fetch
// their wands in: Enchanting for the wands, unlearned again for Leatherworking). Skinning the beasts it kills brings
// Leatherworking to 20, for the Camp Tent: once an hour at a campfire (Cooking 1) the tent gives rested experience,
// 5 % of a level (conquestcapped.com/guides/wow-forever/wow-forever-camping/), which doubles kill XP until it is used
// up: resting 30 seconds by the tent raises the rested experience to that much when it is lower (Field Seated Rest
// Bonus, spell 1229487, an hour's cooldown aura 1229451; kovarel.com/wow-forever/features/camping). It costs the camp
// (fire, tent, a minute's rest) each hour and skinning on every kill.
const (
	campLevel    = 10    // Leatherworking 20 from the leather skinned by then (an estimate)
	campSecs     = 90.0  // fire, tent, a minute's rest
	campCooldown = 3600. // all camp objects share an hour's cooldown
	tentShare    = 0.05  // rested experience per tent, of the level's XP
	profSecs     = 240.0 // at the trainers: Skinning, Cooking, Leatherworking; the leather kits for Leatherworking 20
	skinSecs     = 1.0   // skinning, per kill on average (about half the kills are beasts, two seconds each)
	profDetour   = 180.0 // the most the trainers may add to the way to the next step
)

// CampLevel is the level the hourly camps start at.
const CampLevel = campLevel

// profSites: where Leatherworking and Cooking trainers stand, by side (cmangos spawns): the capitals, Goldshire and
// Bloodhoof Village.
var profSites = map[string][]dataset.Point{
	"A": {{Map: 0, X: -9376, Y: -74}, {Map: 0, X: -8857, Y: 805}, {Map: 0, X: -4750, Y: -1030}, {Map: 1, X: 10120, Y: 2280}},
	"H": {{Map: 1, X: -2255, Y: -287}, {Map: 1, X: 1861, Y: -4562}, {Map: 1, X: -1150, Y: 80}, {Map: 0, X: 1510, Y: 200}},
}

// profAction: the visit to the trainers on the way to next when it adds at most profDetour, else nil.
func (w *World) profAction(s *State, next dataset.Point) *Action {
	if s.Prof {
		return nil
	}
	direct, _ := w.travel(s, s.Pos, next)
	var best *dataset.Point
	bt := profDetour
	for i, p := range profSites[w.Faction] {
		if p.Map != s.Pos.Map && p.Map != next.Map {
			continue
		}
		there, _ := w.travel(s, s.Pos, p)
		on, _ := w.travel(s, p, next)
		if extra := there + on - direct; extra <= bt {
			best, bt = &profSites[w.Faction][i], extra
		}
	}
	if best == nil {
		return nil
	}
	return &Action{Op: "prof", At: *best}
}

// camp: once an hour from campLevel, a camp before the next fight: the tent's rested experience.
func (w *World) camp(s *State) {
	if !s.Prof || w.Level(s) < campLevel || s.Time < s.Camp {
		return
	}
	l := w.Level(s)
	if l-1 >= len(w.Data.LevelXP) {
		return
	}
	s.Time += campSecs
	s.Talk += campSecs
	s.Rested = math.Max(s.Rested, tentShare*float64(w.Data.LevelXP[l-1])) // raised to 5 % of the level, not added
	s.Camp = s.Time + campCooldown
	s.Camps++
}

// restedGain: the extra XP of a kill worth gain from the rested pool (used up).
func restedGain(s *State, gain int) int {
	b := math.Min(float64(gain), s.Rested)
	s.Rested -= b
	return int(math.Round(b))
}
