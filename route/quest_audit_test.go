package route

import (
	"slices"
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

func TestShippedQuestAuditCoversEveryQuestThrough20(t *testing.T) {
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	a, err := AuditQuests(d, 20)
	if err != nil {
		t.Fatal(err)
	}
	want := 0
	for _, q := range d.Quests {
		if q.MinLevel <= 20 {
			want++
		}
	}
	if a.Examined != want || a.CharacterContexts != 56 {
		t.Fatalf("coverage %d/%d, contexts %d", a.Examined, want, a.CharacterContexts)
	}
	seen := map[int]bool{}
	for _, q := range a.Quests {
		if seen[q.Quest] || q.Status == "" {
			t.Fatalf("missing or duplicate classification: %+v", q)
		}
		seen[q.Quest] = true
	}
	w, err := NewWorld(d, Character{Race: "Undead", Class: "Warlock"}, Options{Dungeons: true})
	if err != nil {
		t.Fatal(err)
	}
	q := w.ByID[5727]
	if q == nil || len(q.Objectives) != 1 || q.Objectives[0].Kind != "talk" {
		t.Fatal("Neeru is still a kill objective")
	}
	s := w.NewState()
	s.XP = 100000
	for _, id := range []int{5726} {
		p := w.ByID[id]
		if p == nil {
			t.Fatal("missing prerequisite")
		}
		s.Status[p.Index] = done
	}
	if err := w.Apply(s, Action{Op: "accept", Quest: q.ID}); err != nil {
		t.Fatal(err)
	}
	xpBefore, combatBefore := s.XP, s.Combat
	act, ok := w.bestSource(s, q, 0)
	if !ok || act.NPC != 3216 {
		t.Fatal("Neeru conversation unavailable by level 20")
	}
	if err := w.Apply(s, act); err != nil {
		t.Fatal(err)
	}
	if s.XP != xpBefore || s.Combat != combatBefore {
		t.Fatal("Neeru awarded combat XP")
	}
	if err := w.Apply(s, Action{Op: "turnin", Quest: q.ID}); err != nil {
		t.Fatal(err)
	}
	if w.ByID[5728] == nil || w.CanAccept(s, w.ByID[5728]) != nil {
		t.Fatal("dungeon follow-up not unlocked")
	}
	chain := w.questChain(w.ByID[5728], nil)
	ids := []int{}
	for _, p := range chain {
		ids = append(ids, p.ID)
	}
	if !slices.Equal(ids, []int{5726, 5727, 5728}) {
		t.Fatalf("incomplete optimization candidate: %v", ids)
	}
	for _, tc := range []struct {
		race  string
		quest int
	}{{"NightElf", 6001}, {"Tauren", 6002}} {
		druid, err := NewWorld(d, Character{Race: tc.race, Class: "Druid"}, Options{})
		if err != nil {
			t.Fatal(err)
		}
		body := druid.ByID[tc.quest]
		if body == nil || len(body.Objectives) != 2 || body.Objectives[0].Kind != "kill" || body.Objectives[1].Kind != "talk" {
			t.Fatalf("%s lost Lunaclaw fight or spirit: %s", tc.race, druid.Excluded[tc.quest])
		}
		fight, spirit := body.Objectives[0].Sources[0].Site.Center, body.Objectives[1].Sources[0].Site.Center
		if distance(fight, spirit) > 10 {
			t.Fatalf("%s spirit placed in another race's zone", tc.race)
		}
	}
	spar := w.ByID[99108]
	if spar == nil || spar.Objectives[0].Kind != "duel" || spar.Objectives[0].Count != 3 {
		t.Fatal("Sparring Match must require three duels")
	}
}

func TestChainInsertionPreservesUnrelatedSteps(t *testing.T) {
	w := tinyWorld(t)
	events := []GuideEvent{{Op: "accept", Quest: 2}, {Op: "accept", Quest: 1}, {Op: "turnin", Quest: 1}, {Op: "turnin", Quest: 2}}
	got := w.insertQuestChain(events, w.ByID[3], 3)
	want := []GuideEvent{{Op: "accept", Quest: 2}, {Op: "accept", Quest: 1}, {Op: "complete", Quest: 1}, {Op: "turnin", Quest: 1}, {Op: "accept", Quest: 3}, {Op: "complete", Quest: 3}, {Op: "turnin", Quest: 3}, {Op: "turnin", Quest: 2}}
	if !slices.Equal(got, want) {
		t.Fatalf("chain insertion: %+v", got)
	}
}

func TestChainTriesAnotherPrerequisiteWhenNearestCycles(t *testing.T) {
	w := tinyWorld(t)
	w.ByID[3].PreAll = nil
	w.ByID[3].PreAny = []int{3, 1}
	chain := w.questChain(w.ByID[3], nil)
	if len(chain) != 2 || chain[0].ID != 1 || chain[1].ID != 3 {
		t.Fatalf("valid OR alternative was lost: %+v", chain)
	}
}
