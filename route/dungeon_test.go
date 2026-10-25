package route

import (
	"math"
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// tinyDungeon: the tiny world with a dungeon 400 yards away; one quest is given outside and its boss is inside, one is
// given and handed in inside.
func tinyDungeon() *dataset.Dataset {
	d := tiny()
	door := dataset.Point{Map: 1, X: 0, Y: 400}
	giver := []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}}
	d.NPCs = append(d.NPCs, dataset.NPC{ID: 900, Name: "Boss", MinLevel: 3, MaxLevel: 3, Rank: 1})
	d.Quests = append(d.Quests,
		dataset.Quest{ID: 50, Name: "Kill the boss", Level: 3, MinLevel: 1, XP: 2000, Type: "dungeon", Dungeon: "cave", Start: giver, End: giver,
			Objectives: []dataset.Objective{{Kind: "kill", ID: 900, Count: 1, Inside: true}}},
		dataset.Quest{ID: 51, Name: "Ghost's plea", Level: 3, MinLevel: 1, XP: 1000, Type: "dungeon", Dungeon: "cave",
			Start: []dataset.Ref{{Kind: "npc", ID: 901}}, End: []dataset.Ref{{Kind: "npc", ID: 901}},
			Objectives: []dataset.Objective{{Kind: "run", ID: 1, Count: 1, Inside: true}}},
	)
	d.Dungeons = []dataset.Dungeon{{Key: "cave", Name: "The Cave", Min: 2, Max: 6, Minutes: 20, Entrance: door,
		Mobs: []dataset.DungeonMob{{Level: 3, Elite: true, N: 40}}, Quests: []int{50, 51}}}
	return d
}

func TestDungeonQuestsOnlyInGroupRoutes(t *testing.T) {
	solo, err := NewWorld(tinyDungeon(), Character{Race: "Orc", Class: "Warlock"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if solo.ByID[50] != nil || len(solo.Dungeons) != 0 {
		t.Fatal("a solo route must not run dungeons")
	}
	w, err := NewWorld(tinyDungeon(), Character{Race: "Orc", Class: "Warlock"}, Options{Dungeons: true})
	if err != nil {
		t.Fatal(err)
	}
	q, g := w.ByID[50], w.ByID[51]
	if q == nil || g == nil || len(w.Dungeons) != 1 || len(w.Dungeons[0].Quests) != 2 {
		t.Fatalf("dungeon quests not in the group world: %v", w.Excluded)
	}
	if q.StartInside || !g.StartInside || !g.EndInside {
		t.Error("the ghost gives and takes its quest inside, the boss quest is given outside")
	}
	if _, err := w.Replay([]Action{{Op: "accept", Quest: 51}}); err == nil {
		t.Error("a quest given inside must not be taken outside")
	}
	// the run needs the level, finishes the boss, takes and hands in the ghost's quest and gives the trash XP
	if _, err := w.Replay([]Action{{Op: "dungeon", Dungeon: "cave"}}); err == nil {
		t.Error("the dungeon needs level 2")
	}
	s := w.NewState()
	s.XP = 400
	for _, a := range []Action{{Op: "accept", Quest: 50, At: q.StartPos}, {Op: "dungeon", Dungeon: "cave"}} {
		if err := w.Apply(s, a); err != nil {
			t.Fatal(err)
		}
	}
	if !w.complete(s, q) || s.Status[g.Index] != done {
		t.Fatal("the run must finish the boss and the ghost's quest")
	}
	if s.XP <= 400+1000 || s.Time < 20*60 {
		t.Errorf("run gave %d XP in %.0f s", s.XP-400, s.Time)
	}
	if err := w.Apply(s, Action{Op: "dungeon", Dungeon: "cave"}); err == nil {
		t.Error("a dungeon is run once")
	}
}

func TestCollectionQuestsCostTheGroupMoreKills(t *testing.T) {
	d := tinyDungeon()
	giver := []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}}
	d.Quests = append(d.Quests, dataset.Quest{ID: 52, Name: "Hides", Level: 3, MinLevel: 1, XP: 500, Type: "dungeon", Dungeon: "cave", Start: giver, End: giver,
		Objectives: []dataset.Objective{{Kind: "item", ID: 77, Count: 5, Inside: true, Drops: []dataset.Drop{{NPC: 900, Chance: 50}}}}})
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Warlock"}, Options{Dungeons: true})
	if err != nil {
		t.Fatal(err)
	}
	base := w.NewState()
	base.XP = 400
	with := base.clone()
	if err := w.Apply(with, Action{Op: "accept", Quest: 52, At: w.ByID[52].StartPos}); err != nil {
		t.Fatal(err)
	}
	t0 := with.Time
	without := base.clone()
	without.Time = t0
	for _, s := range []*State{with, without} {
		if err := w.Apply(s, Action{Op: "dungeon", Dungeon: "cave"}); err != nil {
			t.Fatal(err)
		}
	}
	// four more shares of 5 items at 50 %: 40 kills of 15 s
	if got := with.Time - without.Time; math.Abs(got-600) > 1 {
		t.Errorf("collection quest: %.0f s more, want 600", got)
	}
}

func TestPlanRunsAWorthwhileDungeon(t *testing.T) {
	w, err := NewWorld(tinyDungeon(), Character{Race: "Orc", Class: "Warlock"}, Options{Dungeons: true})
	if err != nil {
		t.Fatal(err)
	}
	res, err := w.Plan(PlanOptions{TargetLevel: 5, Restarts: 1})
	if err != nil {
		t.Fatal(err)
	}
	ran := false
	for _, st := range res.Steps {
		ran = ran || st.Op == "dungeon"
	}
	if !ran {
		t.Error("3000 quest XP in 20 minutes must be worth a run")
	}
}
