package route

import (
	"testing"
	"time"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// liveOf reads a planning state back the way the addon would report it: level and XP into the level, position,
// finished quests, and the log with each objective's progress in the game's terms.
func liveOf(w *World, s *State) Live {
	l := Live{Level: w.Level(s), Pos: &s.Pos, Bind: &s.Bind}
	l.XP = s.XP - totalXP(w, l.Level)
	types := map[string]string{"kill": "monster", "item": "item", "object": "object"}
	for _, q := range w.Quests {
		switch s.Status[q.Index] {
		case done:
			l.Completed = append(l.Completed, q.ID)
		case active:
			lq := LiveQuest{ID: q.ID, Ready: w.complete(s, q)}
			// the game lists objectives in its own order: reversed here
			for i := len(q.Objectives) - 1; i >= 0; i-- {
				o := q.Objectives[i]
				lq.Objectives = append(lq.Objectives, LiveObjective{Type: types[o.Kind], Have: int(s.Prog[q.Index][i]), Need: o.Count})
			}
			l.Active = append(l.Active, lq)
		}
	}
	for npc, i := range w.net.byNPC {
		if w.net.fm[i] && s.Known&(1<<i) != 0 {
			l.FlightMasters = append(l.FlightMasters, npc)
		}
	}
	return l
}

func totalXP(w *World, level int) int {
	sum := 0
	for i := 0; i < level-1; i++ {
		sum += w.Data.LevelXP[i]
	}
	return sum
}

func TestLivePlanning(t *testing.T) {
	if testing.Short() {
		t.Skip("slow: plans on the shipped dataset")
	}
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Skip("dataset not available:", err)
	}
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Warlock"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	// play the first two hours of a planned route, then hand the character over as the game would see it
	first, err := w.PlanAhead(w.NewState(), PlanOptions{}, 7200)
	if err != nil {
		t.Fatal(err)
	}
	s := w.NewState()
	for _, st := range first.Steps {
		if err := w.Apply(s, st.Action); err != nil {
			t.Fatal(err)
		}
	}
	got, notes, err := w.StateFrom(liveOf(w, s))
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) > 0 {
		t.Logf("notes: %v", notes)
	}
	if got.XP != s.XP || got.Active != s.Active || got.Known != s.Known || got.Pos != s.Pos {
		t.Fatalf("state differs: xp %d/%d log %d/%d known %b/%b", got.XP, s.XP, got.Active, s.Active, got.Known, s.Known)
	}
	for _, q := range w.Quests {
		if got.Status[q.Index] != s.Status[q.Index] {
			t.Fatalf("%s: status %d, want %d", q.Name, got.Status[q.Index], s.Status[q.Index])
		}
		if s.Status[q.Index] == active && w.complete(got, q) != w.complete(s, q) {
			t.Errorf("%s: completion not carried over", q.Name)
		}
	}
	// the next hour from there, quickly enough to run while playing
	t0 := time.Now()
	next, err := w.PlanAhead(got, PlanOptions{}, 3600)
	if err != nil {
		t.Fatal(err)
	}
	took := time.Since(t0)
	t.Logf("level %d: next hour %d steps, %.0f s of play to level %d, planned in %s", w.Level(got), len(next.Steps), next.Seconds, next.Level, took.Round(time.Millisecond))
	if next.Seconds < 3600 || next.Steps[0].Time > 1800 {
		t.Errorf("plan covers %.0f s, first step at %.0f s", next.Seconds, next.Steps[0].Time)
	}
	if took > 30*time.Second {
		t.Errorf("planning one hour took %s", took)
	}
	// a quest the planner does not know still takes a log slot
	l := liveOf(w, s)
	l.Active = append(l.Active, LiveQuest{ID: 999999})
	if got, notes, _ := w.StateFrom(l); got.Active != s.Active+1 || len(notes) == 0 {
		t.Errorf("unknown quest: log %d, notes %v", got.Active, notes)
	}
	if _, _, err := w.StateFrom(Live{Level: 5}); err == nil {
		t.Error("missing position accepted")
	}
}
