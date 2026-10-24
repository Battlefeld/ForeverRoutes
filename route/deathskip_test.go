package route

import (
	"math"
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

func TestSickness(t *testing.T) {
	for lvl, want := range map[int]float64{1: 0, 10: 0, 11: 60, 15: 300, 20: 600, 40: 600} {
		if got := sickness(lvl); got != want {
			t.Errorf("level %d: %v, want %v", lvl, got, want)
		}
	}
}

// a character deep in a cave far from the turn-in: dying gives the graveyard beside it.
func TestDeathskip(t *testing.T) {
	d := &dataset.Dataset{}
	for i := 0; i < 30; i++ {
		d.LevelXP = append(d.LevelXP, 100) // level 1 + XP/100
	}
	d.Travel.Graveyards = []dataset.Node{{NPC: 1, Name: "Village", At: dataset.Point{Map: 0, X: 100, Y: 100}}}
	// the cave at x 2100: its cell (10, 0) gives the village graveyard to both sides
	d.Travel.Deaths = &dataset.DeathGrid{Cell: 200, Cells: []int{0, 10, 0, 1, 1}}
	w := &World{Data: d, Faction: "A", Opt: Options{}.withDefaults()}
	w.deaths, w.net = newDeaths(d.Travel, Character{Race: "Dwarf", Class: "Warrior"}, "A"), &network{}
	cave, village := dataset.Point{Map: 0, X: 2100, Y: 100}, dataset.Point{Map: 0, X: 150, Y: 100}
	s := &State{Hearth: math.Inf(1)} // hearthstone on cooldown
	walk := w.walk(cave, village)
	got, how := w.travel(s, cave, village)
	if how != byDeath {
		t.Fatalf("walked (%.0f s) instead of a deathskip", got)
	}
	if want := deathSecs + w.walk(d.Travel.Graveyards[0].At, village); math.Abs(got-want) > 1e-6 || got >= walk {
		t.Fatalf("deathskip %.1f s, want %.1f (walk %.1f)", got, want, walk)
	}
	// at level 20 ten minutes of sickness make walking the better choice
	s.XP = 2000 // level 21
	if _, how := w.travel(s, cave, village); how == byDeath {
		t.Fatal("deathskip taken with ten minutes of sickness")
	}
	if err := w.move(s, village); err != nil || s.Deathskips != 0 {
		t.Fatalf("move: %v, %d deathskips", err, s.Deathskips)
	}
	s2 := &State{Pos: cave, Hearth: math.Inf(1)}
	if err := w.move(s2, village); err != nil {
		t.Fatal(err)
	}
	if s2.Deathskips != 1 || s2.via != "deathskip: Village" {
		t.Fatalf("move: %d deathskips, via %q", s2.Deathskips, s2.via)
	}
}

func TestFilterFits(t *testing.T) {
	dw := Character{Race: "Dwarf", Class: "Warrior"}
	ud := Character{Race: "Undead", Class: "Mage"}
	for _, c := range []struct {
		f    string
		c    Character
		side string
		want bool
	}{
		{"", dw, "A", true}, {"Horde Shaman", dw, "A", false}, {"!Warlock", dw, "A", true}, {"Warlock", dw, "A", false},
		{"Human Warrior/Dwarf Warrior", dw, "A", true}, {"!Paladin !Warlock !Hunter !Shaman", dw, "A", true},
		{"Alliance", dw, "A", true}, {"Scourge", ud, "H", true}, {"!Tauren", ud, "H", true}, {"Rogue/Druid", ud, "H", false},
		{"NightElf Hunter", dw, "A", false},
	} {
		if got := filterFits(c.f, c.c, c.side); got != c.want {
			t.Errorf("%q for %s %s: %v", c.f, c.c.Race, c.c.Class, got)
		}
	}
}

func TestLift(t *testing.T) {
	d := &dataset.Dataset{}
	d.Travel.Lifts = []dataset.Lift{{Name: "Mesa", At: dataset.Point{Map: 1, X: 0, Y: 0}, Radius: 300, Seconds: 18}}
	w := &World{Data: d, Opt: Options{}.withDefaults(), lifts: d.Travel.Lifts}
	top, top2, ground := dataset.Point{Map: 1, X: 100, Y: 0}, dataset.Point{Map: 1, X: -100, Y: 50}, dataset.Point{Map: 1, X: 600, Y: 0}
	if got := w.walk(top, ground) - w.walkFlat(top, ground); got != 18 {
		t.Fatalf("down the lift: +%.1f s", got)
	}
	if got := w.walk(top, top2) - w.walkFlat(top, top2); got != 0 {
		t.Fatalf("on the mesa: +%.1f s", got)
	}
	if got := w.walk(ground, dataset.Point{Map: 0, X: 0, Y: 0}); !math.IsInf(got, 1) {
		t.Fatalf("other map: %.1f", got)
	}
}
