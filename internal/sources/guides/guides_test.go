package guides

import "testing"

func TestParseRXP(t *testing.T) {
	src := `RXPGuides.RegisterGuide([[
#forever
step
    .goto 1429,48.314,36.487,45,0
    .goto 1429,49.070,36.438,45,0
    >>Kill |cRXP_ENEMY_Kobold Vermins|r. Loot them for the [|cRXP_LOOT_Nibbled-On Book|r]
    .use 247834 >> Use it
    .collect 247834,1,91741,1 -- Nibbled-On Book (1)
    .accept 91741 >> Accept Nibbled-On Book
    .complete 7,1 --Kill Kobold Vermin (x10)
    .mob Kobold Vermin
step
    .goto 1411/1,-4260.000,-404.200
    >>Loot |cRXP_PICK_Abandoned Training Weapons|r on the ground
    .complete 97279,1 --|6/6 Abandoned Training Weapon
step
    >>Kill |cRXP_ENEMY_Pygmy Surf Crawlers|r
    .complete 818,2 --Crawler Mucus (8)
    .mob +Pygmy Surf Crawler
    .complete 818,1 --Intact Makrura Eye (4)
    .mob +Makrura Shellhide << Warrior
]])`
	facts := ParseRXP(map[string]string{"g.lua": src})
	by := map[[2]int]Fact{}
	for _, f := range facts {
		by[[2]int{f.Quest, f.Obj}] = f
	}
	start := by[[2]int{91741, 0}]
	if !start.Start || start.Item != 247834 || len(start.Places) != 2 || start.Places[0].UIMap != 1429 || start.Places[0].X != 48.314 {
		t.Errorf("start item: %+v", start)
	}
	if f := by[[2]int{7, 1}]; len(f.Mobs) != 1 || f.Mobs[0] != "Kobold Vermin" {
		t.Errorf("kill: %+v", f)
	}
	ground := by[[2]int{97279, 1}]
	if !ground.Ground || len(ground.Places) != 1 || !ground.Places[0].World || ground.Places[0].Map != 1 || ground.Places[0].X != -404.2 || ground.Places[0].Y != -4260 {
		t.Errorf("ground: %+v", ground)
	}
	if a, b := by[[2]int{818, 2}], by[[2]int{818, 1}]; len(a.Mobs) != 1 || a.Mobs[0] != "Pygmy Surf Crawler" || len(b.Mobs) != 1 || b.Mobs[0] != "Makrura Shellhide" {
		t.Errorf("mobs per objective: %+v %+v", a, b)
	}
}

func TestParsePolestar(t *testing.T) {
	src := "guide forever/horde/orc-troll/01-durotar\nzone Durotar\n\nstep Training weapons and boars\n  tip The weapons lie around the Den.\n  collect 277653 quest 97279.1\n  kill 3098 quest 788.1\n  goto 42.4,68.6 \"The Den\"\n\nstep Isle\n  zone Thunder Bluff\n  collect 251918 quest 92473.1 @41.4,26.4\n  interact 460001 quest 92598.1 \"Cast Skysight\"   -- note\n"
	facts := ParsePolestar(map[string]string{"g.guide": src})
	if len(facts) != 4 {
		t.Fatalf("%d facts: %+v", len(facts), facts)
	}
	if f := facts[0]; f.Item != 277653 || f.Quest != 97279 || f.Obj != 1 || len(f.Places) != 1 || f.Places[0].Zone != "Durotar" || f.Places[0].X != 42.4 {
		t.Errorf("collect: %+v", f)
	}
	if f := facts[1]; len(f.NPCs) != 1 || f.NPCs[0] != 3098 {
		t.Errorf("kill: %+v", f)
	}
	if f := facts[2]; f.Places[0].Zone != "Thunder Bluff" || f.Places[0].X != 41.4 {
		t.Errorf("@ place in the step's zone: %+v", f)
	}
	if f := facts[3]; len(f.Objects) != 1 || f.Objects[0] != 460001 {
		t.Errorf("interact: %+v", f)
	}
}

func TestPolestarWithoutPlaces(t *testing.T) {
	facts := ParsePolestar(map[string]string{"e.guide": `zone Elwynn Forest

step Crystal Lake and the wolves
  tip Skinners: Gray Forest Wolves roam the woods near the Tower of Azora.
  collect 247877 quest 91751 optional         -- Rough Wolf Pelts

step Moonbrook
  kill 121 quest 14
  collect 254676 quest 92747.1 only Dwarf -- from the crates
`})
	if len(facts) != 3 {
		t.Fatalf("facts: %+v", facts)
	}
	pelts := facts[0]
	if pelts.Quest != 91751 || pelts.Obj != 0 || pelts.Item != 247877 || pelts.Zone != "Elwynn Forest" || len(pelts.Near) != 0 {
		t.Fatalf("pelts: %+v", pelts)
	}
	found := false
	for _, m := range pelts.Mobs {
		found = found || m == "Gray Forest Wolf"
	}
	if !found {
		t.Fatalf("tip names: %v", pelts.Mobs)
	}
	if crates := facts[2]; crates.Item != 254676 || crates.Obj != 1 || len(crates.Mobs) != 0 || len(crates.Near) != 1 || crates.Near[0] != 121 {
		t.Fatalf("crates: %+v", crates)
	}
}

func TestRXPDeathskips(t *testing.T) {
	skips := RXPDeathskips(map[string]string{"g.lua": `
step << Horde Shaman
    .goto 2521,41.06,22.32
    .deathskip >>Talk to the Spirit Healer
step
    .goto 1413/1,-2672.76,-544.77
    .turnin 1 >> x
step << skip
    .goto 1411/1,-5002.81,-774.08
    .deathskip >> Die
step
    .goto 1411/1,-4948.88,-768.79
`})
	if len(skips) != 1 {
		t.Fatalf("%+v", skips)
	}
	s := skips[0]
	if s.Filter != "Horde Shaman" || s.Die.UIMap != 2521 || s.Die.X != 41.06 || !s.Rise.World || s.Rise.Map != 1 || s.Rise.X != -544.77 || s.Rise.Y != -2672.76 {
		t.Fatalf("%+v", s)
	}
}
