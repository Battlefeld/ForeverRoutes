package route

import (
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

func TestSoloElitePolicyAtActualMobMaximum(t *testing.T) {
	for _, class := range []string{"Warrior", "Paladin", "Hunter", "Rogue", "Priest", "Shaman", "Mage", "Warlock", "Druid"} {
		t.Run(class, func(t *testing.T) {
			w, err := NewWorld(tiny(), Character{Race: "Orc", Class: class}, Options{})
			if err != nil {
				t.Fatal(err)
			}
			src := Source{NPC: &dataset.NPC{Rank: 1, MinLevel: 17, MaxLevel: 20}}
			if !w.dangerousAt(17, src) {
				t.Fatal("level 17 character sent to level 20 elite")
			}
			pet := class == "Hunter" || class == "Warlock"
			if w.dangerousAt(20, src) == pet {
				t.Fatal("elite permission ignored class policy")
			}
			src.Class = true
			if !w.dangerousAt(10, src) {
				t.Fatal("class quest bypassed combat level limit")
			}
		})
	}
}

func TestEliteDropsCannotBecomeGroundPickups(t *testing.T) {
	d := tiny()
	d.Quests = append(d.Quests, dataset.Quest{ID: 999, Name: "Elite head", Level: 2, MinLevel: 1, XP: 100, Start: d.Quests[0].Start, End: d.Quests[0].End, Objectives: []dataset.Objective{{Kind: "item", ID: 999, Count: 1, Drops: []dataset.Drop{{NPC: 200, Chance: 100}}, Places: []dataset.Point{{Map: 1, X: 100, Y: 100}}}}})
	for i := range d.NPCs {
		if d.NPCs[i].ID == 200 {
			d.NPCs[i].Rank = 3
		}
	}
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Shaman"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if w.ByID[999] != nil {
		t.Fatal("boss drop became a safe pickup at fallback waypoint")
	}
}

func TestUnknownCombatLevelIsNotAssumedToBeOne(t *testing.T) {
	w := tinyWorld(t)
	src := Source{NPC: &dataset.NPC{ID: 999}, Chance: 1, Site: Site{Spawns: 1}}
	if len(w.soloSources([]Source{src}, false)) != 0 || !w.dangerousAt(20, src) {
		t.Fatal("unknown combat level was treated as safe")
	}
	src.Talk = true
	if len(w.soloSources([]Source{src}, false)) != 1 || w.dangerousAt(20, src) {
		t.Fatal("non-combat dialogue was blocked by missing combat metadata")
	}
}
