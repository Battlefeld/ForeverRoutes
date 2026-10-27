package route

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

func TestTimelineCanDropTheLastDungeon(t *testing.T) {
	d := tinyDungeon()
	d.Dungeons[0].Minutes = 1000
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Warlock"}, Options{Dungeons: true})
	if err != nil {
		t.Fatal(err)
	}
	guide := []GuideEvent{{Op: "level", Level: 2}, {Op: "dungeon", Dungeon: "cave"}}
	initial, err := w.FollowGuide(guide, 3)
	if err != nil {
		t.Fatal(err)
	}
	stats := &OptimizeStats{}
	e, r, err := w.Optimize(guide, OptimizeOptions{Target: 3, Budget: time.Second, MaxIterations: 30, Stats: stats})
	if err != nil {
		t.Fatal(err)
	}
	if len(actualDungeonKeys(r)) != 0 || r.Seconds >= initial.Seconds || !stats.DungeonEnumerationComplete {
		t.Fatalf("last dungeon retained: %v %.1f / %.1f stats=%+v", RunKeys(e), r.Seconds, initial.Seconds, stats)
	}
	_, required, err := w.Optimize(guide, OptimizeOptions{Target: 3, Budget: time.Second, MaxIterations: 20, RequireDungeons: []string{"cave"}})
	if err != nil || !slices.Contains(actualDungeonKeys(required), "cave") {
		t.Fatalf("required dungeon lost: %v", err)
	}
}

func TestTimelineEvaluatesEightDungeonSubsets(t *testing.T) {
	d := tinyDungeon()
	base := d.Dungeons[0]
	d.Dungeons = nil
	for _, key := range []string{"one", "two", "three"} {
		q := base
		q.Key = key
		q.Quests = nil
		q.Minutes = 999
		d.Dungeons = append(d.Dungeons, q)
	}
	// No quest binds to the old cave; empty expensive runs are still legitimate candidates.
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Warlock"}, Options{Dungeons: true})
	if err != nil {
		t.Fatal(err)
	}
	stats := &OptimizeStats{}
	_, _, err = w.Optimize([]GuideEvent{{Op: "level", Level: 2}}, OptimizeOptions{Target: 3, Budget: time.Second, MaxIterations: 1, Stats: stats})
	if err != nil {
		t.Fatal(err)
	}
	if stats.DungeonCombinationsTotal != 8 || stats.DungeonCombinationsEvaluated != 8 || !stats.DungeonEnumerationComplete {
		t.Fatalf("not all subsets including zero: %+v", stats)
	}
}

func TestTimelineUsesIndependentSeedsAndKeepsRequiredGoals(t *testing.T) {
	w := tinyWorld(t)
	slow := []GuideEvent{{Op: "accept", Quest: 1}, {Op: "complete", Quest: 1}, {Op: "turnin", Quest: 1}}
	fast := []GuideEvent{{Op: "accept", Quest: 2}, {Op: "complete", Quest: 2}, {Op: "turnin", Quest: 2}}
	_, _ = w.FollowGuide(slow, 3)
	b, _ := w.FollowGuide(fast, 3)
	_, r, err := w.Optimize(slow, OptimizeOptions{Target: 3, Seeds: []OptimizationSeed{{Events: fast}}, Keep: []int{2}})
	if err != nil {
		t.Fatal(err)
	}
	if !handsIn(r, 2) || r.Seconds > b.Seconds {
		t.Fatalf("valid alternative seed not selected: %.1f", r.Seconds)
	}
	_, _, err = w.Optimize(slow, OptimizeOptions{Target: 3, Keep: []int{99999}})
	if err == nil {
		t.Fatal("missing mandatory goal silently skipped")
	}
}

func TestCheckpointResumesAndNeverOverwritesFasterRoute(t *testing.T) {
	w := tinyWorld(t)
	path := filepath.Join(t.TempDir(), "best.json")
	guide := []GuideEvent{{Op: "accept", Quest: 1}, {Op: "complete", Quest: 1}, {Op: "turnin", Quest: 1}}
	opts := OptimizeOptions{Target: 3, Budget: time.Second, MaxIterations: 40, Checkpoint: path, Seed: 9}
	_, first, err := w.Optimize(guide, opts)
	if err != nil {
		t.Fatal(err)
	}
	stats := &OptimizeStats{}
	opts.Budget = 0
	opts.Stats = stats
	_, next, err := w.Optimize([]GuideEvent{{Op: "level", Level: 2}}, opts)
	if err != nil {
		t.Fatal(err)
	}
	if !stats.CheckpointLoaded || next.Seconds > first.Seconds {
		t.Fatal("checkpoint was discarded or worsened")
	}
	cp, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved OptimizationCheckpoint
	if err = json.Unmarshal(cp, &saved); err != nil {
		t.Fatal(err)
	}
	// Corrupt claimed timing: loading must recompute the route, not trust a miraculous zero-second result.
	saved.Seconds = 0
	raw, _ := json.Marshal(saved)
	if err = os.WriteFile(path, raw, 0644); err != nil {
		t.Fatal(err)
	}
	_, verified, err := w.Optimize(guide, opts)
	if err != nil {
		t.Fatal(err)
	}
	if verified.Seconds <= 0 {
		t.Fatal("trusted checkpoint time instead of replay")
	}
	repaired, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(repaired, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Seconds != verified.Seconds {
		t.Fatal("checkpoint retained a corrupted recorded time")
	}
	w.Opt.RunSpeed += 1
	if _, _, err = w.Optimize(guide, opts); err == nil {
		t.Fatal("reused incompatible model context")
	}
}

func bagWorld(t *testing.T) *World {
	t.Helper()
	d := tiny()
	giver := []dataset.Ref{{Kind: "npc", ID: StartNPC["Orc"]}}
	d.Quests = append(d.Quests, dataset.Quest{ID: 79976, Name: "Bag", Level: 1, MinLevel: 1, Start: giver, End: giver})
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Warlock"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	return w
}
func TestSleepingBagChargesRestAndExpiresForQuestAndKillXP(t *testing.T) {
	w := bagWorld(t)
	s := w.NewState()
	original := *s
	if w.Apply(s, Action{Op: "sleep", Stacks: 3}) == nil || !reflect.DeepEqual(*s, original) {
		t.Fatal("free sleep without a bag")
	}
	s.Status[w.ByID[79976].Index] = done
	s.Pos.X = 300
	if err := w.Apply(s, Action{Op: "sleep", Stacks: 3}); err != nil {
		t.Fatal(err)
	}
	if s.Time != 190 || s.BagUntil != 7390 || s.BagReady != 3600 {
		t.Fatalf("rest/cooldown/duration not charged: %+v", s)
	}
	if w.bonusXP(s, 100, s.Time) != 103 || w.bonusXP(s, 100, s.BagUntil) != 100 {
		t.Fatal("wrong bonus or expiry")
	}
	before := *s
	if w.Apply(s, Action{Op: "sleep", Stacks: 1}) == nil || !reflect.DeepEqual(*s, before) {
		t.Fatal("cooldown failure changed state")
	}
	s.XP = 0
	s.Rested = 0
	s.BagUntil = 10
	s.BagStacks = 3
	w.awardKillXP(s, 1000, 0, 20)
	if s.XP != 1015 {
		t.Fatalf("buff wrongly covered whole dungeon: %d", s.XP)
	}
	s.Time = 20
	s.XP = 0
	w.awardKillXP(s, 1000, 20, 30)
	if s.XP != 1000 {
		t.Fatal("expired buff still active")
	}
}

func TestPreparationSwitchesChangeRealCraftingCost(t *testing.T) {
	w := tinyWorld(t)
	plan := w.Preparations()
	plan.Camping = false
	if err := w.SetPreparations(plan); err != nil {
		t.Fatal(err)
	}
	s := w.NewState()
	s.XP = 900
	if err := w.Apply(s, Action{Op: "wand", At: wandSites[w.Faction][0]}); err != nil {
		t.Fatal(err)
	}
	if s.Prof || round1(s.Time-s.Travel) != wandCraft {
		t.Fatal("wand forced unrelated profession preparation")
	}
	if w.profAction(s, w.Start) != nil {
		t.Fatal("disabled profession preparation generated an action")
	}
	plan.Wands = false
	w.SetPreparations(plan)
	s = w.NewState()
	s.Wand = true
	if w.wandDPS(s) != 0 || w.wandAction(s, w.Start) != nil {
		t.Fatal("disabled wand policy still active")
	}
}

func TestFeedbackRequiresPostReleaseApprovalAndMatchesEquipment(t *testing.T) {
	w := tinyWorld(t)
	f := CombatFeedback{GameBuild: "released-client", Schema: "foreverroutes.combat-feedback.v1", Character: w.Char, EquipmentRevision: "gear-one", BuildRevision: "build-one", Source: "foreversim", Kills: []KillObservation{{NPC: 100, PlayerLevel: 1, FightSeconds: 1, RestSeconds: 0}}}
	if w.AcceptFeedback(f) == nil {
		t.Fatal("feedback enabled before approval")
	}
	policy := FeedbackSettings{Enabled: true, ReleaseConfirmed: true, UserConfirmedAfterRelease: true, ReleaseBuild: "released-client", ConfirmedAt: time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)}
	for _, gate := range []string{"enabled", "release", "user", "build", "time"} {
		p := policy
		switch gate {
		case "enabled":
			p.Enabled = false
		case "release":
			p.ReleaseConfirmed = false
		case "user":
			p.UserConfirmedAfterRelease = false
		case "build":
			p.ReleaseBuild = ""
		case "time":
			p.ConfirmedAt = ""
		}
		w.Opt.Feedback = p
		if w.AcceptFeedback(f) == nil {
			t.Fatalf("missing gate accepted: %s", gate)
		}
	}
	w.Opt.Feedback = policy
	if err := w.AcceptFeedback(f); err != nil {
		t.Fatal(err)
	}
	s := w.NewState()
	s.EquipmentRevision = "gear-one"
	s.BuildRevision = "build-one"
	base := w.ByID[1].Objectives[0].Sources[0]
	if _, _, ok := w.feedbackSeconds(s, base.NPC); !ok {
		t.Fatal("approved matching feedback not used")
	}
	s.EquipmentRevision = "gear-two"
	if _, _, ok := w.feedbackSeconds(s, base.NPC); ok {
		t.Fatal("old equipment timing reused")
	}
	f.EquipmentRevision = "gear-two"
	f.Kills[0].FightSeconds = .5
	if err := w.AcceptFeedback(f); err != nil {
		t.Fatal(err)
	}
	if fight, _, ok := w.feedbackSeconds(s, base.NPC); !ok || fight != .5 {
		t.Fatal("new equipment did not update kill timing")
	}
	if err := w.Apply(s, Action{Op: "wand", At: wandSites[w.Faction][0]}); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := w.feedbackSeconds(s, base.NPC); ok {
		t.Fatal("planned equipment upgrade reused old measured timings")
	}
	w.Opt.Feedback.Enabled = false
	if _, _, ok := w.feedbackSeconds(s, base.NPC); ok {
		t.Fatal("feedback continued after disabling")
	}
}

func TestGuideIntermediateLevelDoesNotGrindStraightToTarget(t *testing.T) {
	w := tinyWorld(t)
	e := []GuideEvent{{Op: "level", Level: 2}, {Op: "accept", Quest: 1}, {Op: "complete", Quest: 1}, {Op: "turnin", Quest: 1}}
	r, err := w.FollowGuide(e, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !handsIn(r, 1) {
		t.Fatal("fixed 30-kill grinding skipped the intermediate quest")
	}
	for _, s := range r.Steps {
		if s.Op == "accept" {
			if s.Level != 2 {
				t.Fatalf("intermediate level overshot: %d", s.Level)
			}
			break
		}
	}
}

func TestTimelineCanConstructRequiredDungeonFromSoloSeed(t *testing.T) {
	w, err := NewWorld(tinyDungeon(), Character{Race: "Orc", Class: "Warlock"}, Options{Dungeons: true})
	if err != nil {
		t.Fatal(err)
	}
	_, r, err := w.Optimize([]GuideEvent{{Op: "level", Level: 2}}, OptimizeOptions{Target: 3, RequireDungeons: []string{"cave"}})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(actualDungeonKeys(r), "cave") {
		t.Fatal("required dungeon was not constructed")
	}
}
func TestTimelineIterationCapIsReproducible(t *testing.T) {
	guide := []GuideEvent{{Op: "accept", Quest: 1}, {Op: "complete", Quest: 1}, {Op: "turnin", Quest: 1}}
	run := func() ([]GuideEvent, float64) {
		w := tinyWorld(t)
		e, r, err := w.Optimize(guide, OptimizeOptions{Target: 3, Budget: 5 * time.Second, MaxIterations: 80, Seed: 17})
		if err != nil {
			t.Fatal(err)
		}
		return e, r.Seconds
	}
	a, at := run()
	b, bt := run()
	if at != bt || !slices.Equal(a, b) {
		t.Fatal("same iteration cap and seed produced different routes")
	}
}

func TestLiveDoesNotInventGearOrProfessionsAndKeepsRestedXP(t *testing.T) {
	w := tinyWorld(t)
	s, notes, err := w.StateFrom(Live{Level: 5, Pos: &w.Start, RestedXP: 75})
	if err != nil {
		t.Fatal(err)
	}
	if s.Wand || s.Prof || s.Rested != 75 || len(notes) == 0 {
		t.Fatal("invented equipment or lost observed rested XP")
	}
}

func TestPreparationPolicyReplaysIndependentlyOfModeLabel(t *testing.T) {
	a, err := NewWorld(tiny(), Character{Race: "Orc", Class: "Warlock"}, Options{Preparations: PreparationConstraints{Wands: "auto"}})
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewWorld(tiny(), a.Char, Options{Preparations: PreparationConstraints{Wands: "on"}})
	if err != nil {
		t.Fatal(err)
	}
	p := a.Preparations()
	p.ForceWand = true
	p.ProfessionLevel = 8
	for _, w := range []*World{a, b} {
		if err := w.SetPreparations(p); err != nil {
			t.Fatal(err)
		}
	}
	makeState := func(w *World) *State { s := w.NewState(); s.XP = 2700; s.Pos.X = 5000; return s }
	sa, sb := makeState(a), makeState(b)
	aa, ab := a.wandAction(sa, sa.Pos), b.wandAction(sb, sb.Pos)
	if aa == nil || ab == nil || *aa != *ab {
		t.Fatal("fixed and free searches have different physical preparation policies")
	}
	if err := a.Apply(sa, *aa); err != nil {
		t.Fatal(err)
	}
	if err := b.Apply(sb, *ab); err != nil {
		t.Fatal(err)
	}
	if sa.Time != sb.Time || sa.Prof || sb.Prof {
		t.Fatal("mode label changed time or ignored delayed professions")
	}
	p.ForceWand = false
	a.SetPreparations(p)
	b.SetPreparations(p)
	sa, sb = makeState(a), makeState(b)
	if a.wandAction(sa, sa.Pos) != nil || b.wandAction(sb, sb.Pos) != nil {
		t.Fatal("constraint label forced an unrecorded crafting policy")
	}
}

func TestCheckpointIgnoresAnEmptyOverrideCache(t *testing.T) {
	w := tinyWorld(t)
	a, err := w.optimizationKey(OptimizeOptions{Target: 3}, nil)
	if err != nil {
		t.Fatal(err)
	}
	w.classOverride = map[int]string{}
	b, err := w.optimizationKey(OptimizeOptions{Target: 3}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatal("incidental empty override cache invalidated the checkpoint")
	}
}
