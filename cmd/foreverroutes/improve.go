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
	out := fs.String("out", "", "write the result here")
	fs.Parse(args)
	w, err := loadWorld(*path, *race, *class, *faction, *profile)
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
	rep := bestReply{Character: w.Char, Target: *target, Hours: map[string]float64{"guide": g0.Seconds / 3600, "guideImproved": g1.Seconds / 3600,
		"plan": plan.Seconds / 3600, "planImproved": p1.Seconds / 3600}, SearchSecs: time.Since(t0).Seconds()}
	if p1.Seconds < g1.Seconds {
		rep.Winner, rep.Events, rep.Quests, rep.Profile, rep.Assumed = "plan", pe, p1.Quests, p1.Profile, p1.Assumed
	} else {
		rep.Winner, rep.Events, rep.Quests, rep.Profile, rep.Assumed = "guide", ge, g1.Quests, g1.Profile, g1.Assumed
	}
	fmt.Fprintf(os.Stderr, "%s %s: guide %.1f h -> %.1f h, plan %.1f h -> %.1f h, winner %s (%.0f s)\n", *race, *class,
		rep.Hours["guide"], rep.Hours["guideImproved"], rep.Hours["plan"], rep.Hours["planImproved"], rep.Winner, rep.SearchSecs)
	if *out == "" {
		return printJSON(rep)
	}
	b, _ := json.MarshalIndent(rep, "", " ")
	return os.WriteFile(*out, b, 0o644)
}
