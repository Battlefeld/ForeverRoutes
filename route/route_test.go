package route

import (
	"math"
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// tiny is a small Kalimdor: the Orc start NPC with an inn next to it, a boar field east (10 spawns) and a smaller
// boar field west, two flight masters, and three quests.
func tiny() *dataset.Dataset {
	pt := func(x, y float64) dataset.Point { return dataset.Point{Map: 1, X: x, Y: y} }
	var east, west []dataset.Point
	for i := 0; i < 10; i++ {
		east = append(east, pt(600+float64(i%5)*30, float64(i/5)*30))
	}
	for i := 0; i < 4; i++ {
		west = append(west, pt(-300-float64(i)*30, 0))
	}
	return &dataset.Dataset{
		Schema:  dataset.Schema,
		LevelXP: []int{400, 900, 1400, 2100, 2800, 3600},
		NPCs: []dataset.NPC{
			{ID: StartNPC["Orc"], Name: "Giver", MinLevel: 5, MaxLevel: 5, Hostile: "A", Spawns: []dataset.Point{pt(0, 0)}},
			{ID: 100, Name: "Boar", MinLevel: 1, MaxLevel: 2, HP: 50, Respawn: 600, Spawns: append(append([]dataset.Point{}, east...), west...)},
			{ID: 200, Name: "Elite", MinLevel: 2, MaxLevel: 2, HP: 500, Rank: 1, Spawns: []dataset.Point{pt(100, 100)}},
			{ID: 300, Name: "Innkeeper", Spawns: []dataset.Point{pt(10, 0)}},
		},
		Quests: []dataset.Quest{
			{ID: 1, Name: "Boars", Level: 1, MinLevel: 1, XP: 300, Start: []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}},
				End: []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}}, Objectives: []dataset.Objective{{Kind: "kill", ID: 100, Count: 5}}},
			{ID: 2, Name: "Tusks", Level: 2, MinLevel: 1, XP: 400, Start: []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}},
				End: []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}}, Objectives: []dataset.Objective{{Kind: "item", ID: 9, Count: 3, Drops: []dataset.Drop{{NPC: 100, Chance: 50}}}}},
			{ID: 3, Name: "Follow-up", Level: 2, MinLevel: 1, XP: 250, PreAll: []int{1}, Start: []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}},
				End: []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}}},
			{ID: 4, Name: "Alliance only", Level: 1, MinLevel: 1, XP: 300, Races: RaceBits["Human"], Start: []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}},
				End: []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}}},
			{ID: 6, Name: "Found a tusk", Level: 2, MinLevel: 1, XP: 200, Start: []dataset.Ref{{Kind: "item", ID: 9}},
				StartDrops: []dataset.Drop{{NPC: 100, Chance: 25}}, End: []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}}},
			{ID: 7, Name: "Pelts", Level: 2, MinLevel: 1, XP: 300, Start: []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}},
				End: []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}}, Objectives: []dataset.Objective{{Kind: "item", ID: 11, Count: 2, Drops: []dataset.Drop{{NPC: 100, Unknown: true}}}}},
			{ID: 5, Name: "Elite", Level: 2, MinLevel: 1, XP: 900, Start: []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}},
				End: []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}}, Objectives: []dataset.Objective{{Kind: "kill", ID: 200, Count: 1}}},
		},
		Travel: dataset.Travel{
			FlightMasters: []dataset.Node{{NPC: 400, Faction: "H", At: pt(0, 40)}, {NPC: 401, Faction: "H", At: pt(5000, 0)}},
			Inns:          []dataset.Node{{NPC: 300, Faction: "H", At: pt(10, 0)}},
		},
	}
}

func tinyWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(tiny(), Character{Race: "Orc", Class: "Warlock"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestWorldExcludesWhatTheCharacterCannotDo(t *testing.T) {
	w := tinyWorld(t)
	if w.Excluded[4] != "other race" {
		t.Errorf("quest 4: %q", w.Excluded[4])
	}
	if w.ByID[5] == nil {
		t.Errorf("a warlock's pet tanks a plain elite: %s", w.Excluded[5])
	}
	mage, err := NewWorld(tiny(), Character{Race: "Orc", Class: "Mage"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if mage.Excluded[5] == "" {
		t.Error("a quest that needs an elite kill must be excluded for a mage")
	}
	for _, id := range []int{1, 2, 3} {
		if w.ByID[id] == nil {
			t.Errorf("quest %d should be usable: %s", id, w.Excluded[id])
		}
	}
}

func TestClassRules(t *testing.T) {
	warrior, err := NewWorld(tiny(), Character{Race: "Orc", Class: "Warrior"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if warrior.Opt.MaxMobAbove != 1 || warrior.Opt.QuestAtLevel || warrior.Opt.SoloElites {
		t.Errorf("warrior: %+v", warrior.Opt)
	}
	// the boars are level 1-2: a level 1 warrior may fight them, the level 2 elite not at all
	s := warrior.NewState()
	if warrior.tooDangerous(s, warrior.ByID[1].Objectives[0].Sources[0]) {
		t.Error("a group of level 1-2 suits a level 1 warrior")
	}
	if warrior.ByID[5] != nil {
		t.Error("a warrior does not fight elites alone")
	}
}

func TestSitesSplitSpawnGroups(t *testing.T) {
	w := tinyWorld(t)
	src := w.ByID[1].Objectives[0].Sources
	if len(src) != 2 {
		t.Fatalf("boars should have two spawn groups, got %d", len(src))
	}
	if src[0].Site.Spawns != 10 || src[1].Site.Spawns != 4 {
		t.Errorf("groups: %d and %d spawns", src[0].Site.Spawns, src[1].Site.Spawns)
	}
	// from the start the west group is closer, so a short objective (4 kills, as many as it has boars) is worked there
	s := w.NewState()
	if err := w.Apply(s, Action{Op: "accept", Quest: 1, At: w.ByID[1].StartPos}); err != nil {
		t.Fatal(err)
	}
	q := w.ByID[1]
	s.Prog[q.Index][0] = 1
	a, ok := w.bestSource(s, q, 0)
	if !ok || a.At.X > 0 {
		t.Errorf("expected the west group, got %+v", a.At)
	}
	// five kills there would mean waiting for a respawn: the larger east group is faster
	s.Prog[q.Index][0] = 0
	if a, _ := w.bestSource(s, q, 0); a.At.X < 0 {
		t.Errorf("expected the east group for more kills than the west group has, got %+v", a.At)
	}
}

func TestKillsCountForEveryQuestInTheLog(t *testing.T) {
	w := tinyWorld(t)
	s := w.NewState()
	for _, id := range []int{1, 2} {
		if err := w.Apply(s, Action{Op: "accept", Quest: id, At: w.ByID[id].StartPos}); err != nil {
			t.Fatal(err)
		}
	}
	a, _ := w.bestSource(s, w.ByID[1], 0)
	if err := w.Apply(s, a); err != nil {
		t.Fatal(err)
	}
	if got := s.Prog[w.ByID[2].Index][0]; math.Abs(got-2.5) > 1e-9 {
		t.Errorf("5 boar kills at 50 %% should give 2.5 tusks, got %v", got)
	}
}

func TestRespawnsAreWaitedFor(t *testing.T) {
	w := tinyWorld(t)
	s := w.NewState()
	site := w.ByID[1].Objectives[0].Sources[1].Site // 4 spawns, 600 s respawn
	w.kill(s, w.NPC[100], 0, site, 8)
	if s.Wait <= 0 {
		t.Fatal("8 kills at 4 spawns must wait for respawns")
	}
	// coming back right away: the group is still empty
	before := s.Wait
	w.kill(s, w.NPC[100], 0, site, 2)
	if s.Wait <= before {
		t.Error("a group killed a moment ago must be waited for again")
	}
}

func TestHearthstoneAndBinding(t *testing.T) {
	w := tinyWorld(t)
	s := w.NewState()
	if err := w.Apply(s, Action{Op: "accept", Quest: 1, At: w.ByID[1].StartPos}); err != nil {
		t.Fatal(err)
	}
	if s.Bind != w.inns[0] {
		t.Fatalf("accepting next to the inn should bind there, bind %+v", s.Bind)
	}
	s.Pos = dataset.Point{Map: 1, X: 3000, Y: 0}
	t0 := s.Time
	if err := w.move(s, w.Start); err != nil {
		t.Fatal(err)
	}
	if dt := s.Time - t0; dt > hearthCast+10 {
		t.Errorf("the hearthstone should bring the character home in about %v s, took %v", hearthCast, dt)
	}
	if s.Hearth != t0+hearthCooldown {
		t.Errorf("cooldown not started: %v", s.Hearth)
	}
	// on cooldown: walking
	s.Pos = dataset.Point{Map: 1, X: 3000, Y: 0}
	if _, how := w.travel(s, s.Pos, w.Start); how == byHearth {
		t.Error("the hearthstone is on cooldown")
	}
}

func TestFlightMastersAreFoundOnTheWay(t *testing.T) {
	w := tinyWorld(t)
	s := w.NewState()
	s.Pos = dataset.Point{Map: 1, X: -500, Y: 30}
	s.Hearth = 1e9 // walk
	if err := w.move(s, dataset.Point{Map: 1, X: 500, Y: 30}); err != nil {
		t.Fatal(err)
	}
	if s.Known&1 == 0 {
		t.Error("walking past the flight master at (0,40) should find it")
	}
}

func TestPlanReachesTheTargetAndReplays(t *testing.T) {
	w := tinyWorld(t)
	res, err := w.Plan(PlanOptions{TargetLevel: 4, Restarts: 2})
	if err != nil {
		t.Fatal(err)
	}
	if res.Level < 4 {
		t.Fatalf("level %d", res.Level)
	}
	if res.Quests < 3 {
		t.Errorf("all three quests are worth doing, did %d", res.Quests)
	}
	actions := make([]Action, len(res.Steps))
	for i, st := range res.Steps {
		actions[i] = st.Action
	}
	again, err := w.Replay(actions)
	if err != nil {
		t.Fatal(err)
	}
	if again.Seconds != res.Seconds || again.XP != res.XP {
		t.Errorf("replay differs: %v s / %d XP against %v s / %d XP", again.Seconds, again.XP, res.Seconds, res.XP)
	}
}

func TestReplayRejectsImpossibleSteps(t *testing.T) {
	w := tinyWorld(t)
	if _, err := w.Replay([]Action{{Op: "turnin", Quest: 1}}); err == nil {
		t.Error("turning in a quest that was never taken must fail")
	}
	if _, err := w.Replay([]Action{{Op: "accept", Quest: 3}}); err == nil {
		t.Error("a follow-up must not be accepted before its prerequisite")
	}
}

func TestFollowGuide(t *testing.T) {
	w := tinyWorld(t)
	res, err := w.FollowGuide([]GuideEvent{
		{Op: "accept", Quest: 1}, {Op: "accept", Quest: 4}, {Op: "complete", Quest: 1}, {Op: "turnin", Quest: 1},
		{Op: "accept", Quest: 3}, {Op: "turnin", Quest: 3},
	}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if res.Quests != 2 || res.Skipped["quest not available"] != 1 {
		t.Errorf("quests %d, skipped %v", res.Quests, res.Skipped)
	}
	if res.Level < 3 || res.Grind <= 0 {
		t.Errorf("should grind to the target: level %d, grind %v", res.Level, res.Grind)
	}
}

// A guide that sends a melee character to mobs more than one level up: it levels first, then does the quest.
func TestFollowerLevelsBeforeMobsAbove(t *testing.T) {
	d := tiny()
	var wolves []dataset.Point
	for i := 0; i < 8; i++ {
		wolves = append(wolves, dataset.Point{Map: 1, X: -200, Y: float64(i) * 30})
	}
	d.NPCs = append(d.NPCs, dataset.NPC{ID: 500, Name: "Wolf", MinLevel: 3, MaxLevel: 3, HP: 80, Respawn: 300, Spawns: wolves})
	d.Quests = append(d.Quests, dataset.Quest{ID: 8, Name: "Wolves", Level: 3, MinLevel: 1, XP: 500, Start: []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}},
		End: []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}}, Objectives: []dataset.Objective{{Kind: "kill", ID: 500, Count: 4}}})
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Warrior"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	res, err := w.FollowGuide([]GuideEvent{{Op: "accept", Quest: 8}, {Op: "turnin", Quest: 8}}, 4)
	if err != nil {
		t.Fatal(err)
	}
	if res.Quests != 1 {
		t.Fatalf("the wolves must be done after levelling: %v", res.Missed)
	}
	for _, st := range res.Steps {
		if st.Op == "work" && st.Quest == 8 && st.Level < 2 {
			t.Errorf("level 3 wolves fought at level %d", st.Level)
		}
	}
}

func TestDefaultProfile(t *testing.T) {
	p := DefaultProfile{}
	even := &dataset.NPC{MaxLevel: 10, HP: int(p.hpAtForTest(10))}
	f10, r10 := p.KillSeconds(Character{}, 10, even)
	if math.Abs(f10-10.5) > 0.01 {
		t.Errorf("even mob at 10: %v s fight", f10)
	}
	high := &dataset.NPC{MaxLevel: 12, HP: even.HP}
	if f, _ := p.KillSeconds(Character{}, 10, high); f <= f10*1.4 {
		t.Errorf("two levels higher should take 50 %% longer: %v", f)
	}
	weak := &dataset.NPC{MaxLevel: 10, HP: even.HP / 10}
	if f, _ := p.KillSeconds(Character{}, 10, weak); f < f10*0.6-1e-9 {
		t.Errorf("weak mobs still take 60 %% of a fight: %v", f)
	}
	if r10 <= 4 {
		t.Errorf("rest %v", r10)
	}
}

func (p DefaultProfile) hpAtForTest(l int) float64 {
	return typical(p.EvenHP, l, 30+12*float64(l)+0.7*float64(l*l))
}

func TestQuestStartedByAnItem(t *testing.T) {
	w := tinyWorld(t)
	q := w.ByID[6]
	if q == nil || len(q.StartSources) != 2 {
		t.Fatalf("item-started quest: %v %s", q, w.Excluded[6])
	}
	s := w.NewState()
	at := w.startAt(s, q)
	if at.X > 0 {
		t.Errorf("the closer (west) boars should be chosen, got %+v", at)
	}
	xp0 := s.XP
	if err := w.Apply(s, Action{Op: "accept", Quest: 6, At: at}); err != nil {
		t.Fatal(err)
	}
	if s.Status[q.Index] != active || s.XP <= xp0 || s.Combat <= 0 {
		t.Errorf("accepting should kill about 4 boars first: status %d, xp %d, combat %v", s.Status[q.Index], s.XP, s.Combat)
	}
}

func TestUnknownDropChanceIsAssumedAndReported(t *testing.T) {
	w := tinyWorld(t)
	q := w.ByID[7]
	if q == nil {
		t.Fatalf("quest with an unknown drop chance should be usable: %s", w.Excluded[7])
	}
	src := q.Objectives[0].Sources[0]
	if !src.Assumed || src.Chance != w.Opt.AssumedDrop {
		t.Fatalf("assumed %v chance %v", src.Assumed, src.Chance)
	}
	res, err := w.Replay([]Action{{Op: "accept", Quest: 7, At: q.StartPos}, {Op: "work", Quest: 7, Obj: 0, NPC: 100, At: src.Site.Center}, {Op: "turnin", Quest: 7, At: q.EndPos}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Assumed != 1 || res.Quests != 1 {
		t.Errorf("assumed steps %d, quests %d", res.Assumed, res.Quests)
	}
}

// A druid from level 10 teleports to Moonglade instead of walking there.
func TestDruidTeleportsToMoonglade(t *testing.T) {
	d := tiny()
	d.LevelXP = []int{400, 900, 1400, 2100, 2800, 3600, 4500, 5400, 6500, 7600, 8800, 10100}
	d.NPCs = append(d.NPCs, dataset.NPC{ID: 501, Name: "Spirit", Spawns: []dataset.Point{{Map: 1, X: moonglade.X + 50, Y: moonglade.Y}}})
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Druid"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	s := w.NewState()
	far := dataset.Point{Map: 1, X: moonglade.X + 50, Y: moonglade.Y}
	if _, how := w.travel(s, s.Pos, far); how == byTeleport {
		t.Error("no teleport before level 10")
	}
	s.XP = 100000
	if tt, how := w.travel(s, s.Pos, far); how != byTeleport || tt > teleportCast+60 {
		t.Errorf("teleport: how %d, %.0f s", how, tt)
	}
}
