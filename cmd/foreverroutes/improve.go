package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Battlefeld/ForeverRoutes/route"
)

// bestReply is the faster of the improved guide and the improved planned route for one character.
type bestReply struct {
	Character  route.Character    `json:"character"`
	Target     int                `json:"target"`
	Winner     string             `json:"winner"` // guide | plan
	Hours      map[string]float64 `json:"hours"`  // guide, guideImproved, plan, planImproved
	Quests     int                `json:"quests"`
	Events     []route.GuideEvent `json:"events"`
	Profile    string             `json:"killProfile"`
	Assumed    int                `json:"assumedDropSteps"`
	SearchSecs float64            `json:"searchSeconds"`
	Class      []route.ClassCost  `json:"classQuests"` // what each class quest chain costs the route
}

// cmdImprove: the guide improved by local edits, our planned route improved the same way; the faster wins.
func cmdImprove(args []string) error {
	fs := newFlags("improve")
	path := fs.String("data", "data/routes-data.json.gz", "dataset")
	race := fs.String("race", "Orc", "race")
	class := fs.String("class", "Warlock", "class")
	faction := fs.String("faction", "", "A or H (only for Skyborne)")
	profile := fs.String("profile", "", "kill profile from ForeverCore")
	target := fs.Int("target", 20, "target level")
	file := fs.String("events", "", "guide events (JSON list), the starting point")
	budget := fs.Duration("budget", 60*time.Second, "search time per route")
	margin := fs.Float64("margin", 0.03, "our planned route wins only when it is this much faster than the improved guide (the guide's route is what players know and the model is not calibrated yet)")
	out := fs.String("out", "", "write the result here")
	group := fs.Bool("dungeons", false, "route with dungeons: dungeon runs with their quests may be part of the planned route")
	fs.Parse(args)
	w, err := loadWorld(*path, *race, *class, *faction, *profile, *group)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	var guide []route.GuideEvent
	if err := json.Unmarshal(raw, &guide); err != nil {
		return err
	}
	t0 := time.Now()
	g0, err := w.FollowGuide(guide, *target)
	if err != nil {
		return err
	}
	ge, g1, err := w.Improve(guide, *target, *budget)
	if err != nil {
		return err
	}
	plan, err := w.Plan(route.PlanOptions{TargetLevel: *target, Restarts: 1})
	if err != nil {
		return err
	}
	pe, p1, err := w.Improve(route.EventsOf(plan.Steps), *target, *budget)
	if err != nil {
		return err
	}
	soloEvents, solo := ge, g1 // the guide improved without runs
	if *group {
		// with dungeons: the guide with dungeon runs put in competes with the planned route
		if ge2, g2, err := w.Improve(w.WithDungeons(ge, *target), *target, *budget); err == nil && route.Runs(ge2) > 0 && g2.Level >= *target {
			ge, g1 = ge2, g2
		}
	}
	rep := bestReply{Character: w.Char, Target: *target, Hours: map[string]float64{"guide": g0.Seconds / 3600, "guideImproved": g1.Seconds / 3600,
		"plan": plan.Seconds / 3600, "planImproved": p1.Seconds / 3600}, SearchSecs: time.Since(t0).Seconds()}
	// a route that does not reach the target never wins
	reached := func(r *route.GuideResult) bool { return r != nil && r.Level >= *target }
	// with dungeons the faster wins, as long as it runs a dungeon; without, the guide unless ours is clearly faster
	planWins := reached(p1) && (!reached(g1) || p1.Seconds < g1.Seconds*(1-*margin))
	if *group {
		planWins = reached(p1) && route.Runs(pe) > 0 && (!reached(g1) || route.Runs(ge) == 0 || p1.Seconds <= g1.Seconds)
	}
	if planWins {
		rep.Winner, rep.Events, rep.Quests, rep.Profile, rep.Assumed = "plan", pe, p1.Quests, p1.Profile, p1.Assumed
	} else {
		rep.Winner, rep.Events, rep.Quests, rep.Profile, rep.Assumed = "guide", ge, g1.Quests, g1.Profile, g1.Assumed
	}
	if win := map[bool]*route.GuideResult{true: p1, false: g1}[planWins]; *group && reached(solo) && solo.Seconds < win.Seconds {
		// no run pays on the way to the target (the runs cost more than they bring): the route with dungeons is the
		// one without
		rep.Winner, rep.Events, rep.Quests, rep.Profile, rep.Assumed = "guide", soloEvents, solo.Quests, solo.Profile, solo.Assumed
		rep.Hours["guideImproved"] = solo.Seconds / 3600
	}
	rep.Class = w.ClassReport(rep.Events, *target)
	fmt.Fprintf(os.Stderr, "%s %s: guide %.1f h -> %.1f h, plan %.1f h -> %.1f h, winner %s (%.0f s)\n", *race, *class,
		rep.Hours["guide"], rep.Hours["guideImproved"], rep.Hours["plan"], rep.Hours["planImproved"], rep.Winner, rep.SearchSecs)
	if *out == "" {
		return printJSON(rep)
	}
	b, _ := json.MarshalIndent(rep, "", " ")
	return os.WriteFile(*out, b, 0o644)
}
