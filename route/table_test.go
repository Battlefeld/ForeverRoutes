package route

import (
	"math"
	"os"
	"strings"
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

const tableJSON = `{"schema":"foreverroutes.killprofile.v1","class":"mage","rotation":"fire","classOptions":{},"gear":"none","talents":"none",
"levels":[{"level":10,"health":200,"mana":400,"targets":{
 "0":[{"seconds":10,"damage":200,"manaSpent":200},{"seconds":5,"damage":100,"manaSpent":100}],
 "2":[{"seconds":5,"damage":50,"manaSpent":100},{"seconds":10,"damage":100,"manaSpent":200}]}}]}`

func TestTableProfile(t *testing.T) {
	p, err := LoadTableProfile(strings.NewReader(tableJSON))
	if err != nil {
		t.Fatal(err)
	}
	if !p.Fits(Character{Class: "Mage"}) || p.Fits(Character{Class: "Warlock"}) {
		t.Error("Fits")
	}
	c := Character{Race: "Human", Class: "Mage"}
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
	// same level, 150 health: halfway between 5 s and 10 s; 150 mana back with level 10 water (151 over 18 s)
	f, r := p.KillSeconds(c, 10, &dataset.NPC{HP: 150, MaxLevel: 10})
	if !near(f, 7.5) || !near(r, 150/(151.0/18)+4) {
		t.Errorf("even: %v %v", f, r)
	}
	// two above: the +2 curve, extrapolated past its last point
	if f, _ := p.KillSeconds(c, 10, &dataset.NPC{HP: 150, MaxLevel: 12}); !near(f, 15) {
		t.Errorf("+2: %v", f)
	}
	// one above: halfway between both curves
	if f, _ := p.KillSeconds(c, 10, &dataset.NPC{HP: 100, MaxLevel: 11}); !near(f, 7.5) {
		t.Errorf("+1: %v", f)
	}
	// melee damage: 3 s of 7.5 s are spent before the mob arrives
	_, r = p.KillSeconds(c, 10, &dataset.NPC{HP: 150, MaxLevel: 10, Dmg: 20})
	if want := math.Max(10*4.5/(61.0/18), 150/(151.0/18)) + 4; !near(r, want) {
		t.Errorf("melee: %v, want %v", r, want)
	}
	// outside the table: the default estimate, scaled so that it agrees with the table at the table's first level
	d := DefaultProfile{}
	ref := &dataset.NPC{MinLevel: 10, MaxLevel: 10}
	tf, tr := p.KillSeconds(c, 10, ref)
	ef, er := d.KillSeconds(c, 10, ref)
	f, r = p.KillSeconds(c, 5, &dataset.NPC{HP: 100, MaxLevel: 5})
	if df, dr := d.KillSeconds(c, 5, &dataset.NPC{HP: 100, MaxLevel: 5}); !near(f, df*tf/ef) || !near(r, dr*tr/er) {
		t.Errorf("fallback: %v %v vs %v %v scaled by %v %v", f, r, df, dr, tf/ef, tr/er)
	}
	if _, err := LoadTableProfile(strings.NewReader(`{"schema":"x"}`)); err == nil {
		t.Error("wrong schema accepted")
	}
}

// The profiles in data/profiles load and give plausible fights for a normal mob of the character's level.
func TestShippedProfiles(t *testing.T) {
	for _, f := range []string{"warrior", "paladin", "hunter", "rogue", "priest", "shaman", "mage", "warlock", "druid"} {
		r, err := os.Open("../data/profiles/" + f + ".json")
		if err != nil {
			t.Fatal(err)
		}
		p, err := LoadTableProfile(r)
		r.Close()
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		if lo, hi := p.LevelRange(); lo != 10 || hi != 30 {
			t.Errorf("%s: levels %d-%d", f, lo, hi)
		}
		fight, rest := p.KillSeconds(Character{Class: p.Class}, 20, &dataset.NPC{HP: 600, MaxLevel: 20, Dmg: 25})
		if fight < 5 || fight > 90 || rest < 4 || rest > 90 {
			t.Errorf("%s: level 20 mob %.1f s fight, %.1f s rest", f, fight, rest)
		}
		t.Logf("%s: level 20 mob (600 HP) %.1f s fight, %.1f s rest", f, fight, rest)
	}
}
