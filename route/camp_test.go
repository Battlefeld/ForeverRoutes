package route

import (
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

func TestSkillQuestNeedsTheProfessions(t *testing.T) {
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Skip()
	}
	w, _ := NewWorld(d, Character{Race: "Tauren", Class: "Warrior"}, Options{})
	q := w.ByID[97936] // Camping 101: Skinning, Bloodhoof
	if q == nil {
		t.Fatalf("excluded: %s", w.Excluded[97936])
	}
	s := w.NewState()
	for w.Level(s) < 10 {
		s.XP += 100
	}
	for _, p := range append(append([]int(nil), q.PreAll...), q.PreAny...) {
		if pq := w.ByID[p]; pq != nil {
			s.Status[pq.Index] = done
		}
	}
	if err := w.Apply(s, Action{Op: "accept", Quest: q.ID, At: q.StartPos}); err != nil {
		t.Fatal(err)
	}
	if err := w.Apply(s.clone(), Action{Op: "work", Quest: q.ID, Obj: 0, At: q.Objectives[0].Sources[0].Site.Center}); err == nil {
		t.Fatal("skill done without the professions")
	}
	s.Prof = true
	if err := w.Apply(s, Action{Op: "work", Quest: q.ID, Obj: 0, At: q.Objectives[0].Sources[0].Site.Center}); err != nil {
		t.Fatal(err)
	}
	if err := w.Apply(s, Action{Op: "turnin", Quest: q.ID, At: q.EndPos}); err != nil {
		t.Fatal(err)
	}
}
