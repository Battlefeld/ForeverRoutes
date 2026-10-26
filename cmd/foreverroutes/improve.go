package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
	Class      []route.ClassCost  `json:"classQuests"`          // what each class quest chain costs the route
	WandDetour float64            `json:"wandDetour,omitempty"` // a caster: the most a visit for its wands adds to the way (s)
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
	seed := fs.String("seed", "", "with -dungeons: routes (improve results, comma-separated) to start from too: the solo route (kept when no run pays), the routes with dungeons of the race's other classes")
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
	wand := w.ChooseWandDetour(guide, *target) // a caster: how far it goes for its wands
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
		"plan": plan.Seconds / 3600, "planImproved": p1.Seconds / 3600}, SearchSecs: time.Since(t0).Seconds(), WandDetour: wand}
	// a route that does not reach the target never wins
	reached := func(r *route.GuideResult) bool { return r != nil && r.Level >= *target }
	type cand struct {
		winner string
		events []route.GuideEvent
		res    *route.GuideResult
	}
	set := func(c cand) {
		rep.Winner, rep.Events, rep.Quests, rep.Profile, rep.Assumed = c.winner, c.events, c.res.Quests, c.res.Profile, c.res.Assumed
	}
	// without dungeons: the guide unless ours is clearly faster
	if reached(p1) && (!reached(g1) || p1.Seconds < g1.Seconds*(1-*margin)) {
		set(cand{"plan", pe, p1})
	} else {
		set(cand{"guide", ge, g1})
	}
	if *group {
		// with dungeons: runs put into both routes (the improved guide and ours) and both improved again; the fastest
		// that runs a dungeon wins, unless no run pays on the way to the target (then the route without them)
		best := cand{rep.Winner, rep.Events, map[bool]*route.GuideResult{true: p1, false: g1}[rep.Winner == "plan"]}
		var grouped *cand
		try := func(winner string, events []route.GuideEvent) {
			if e, r, err := w.Improve(w.WithDungeons(events, *target), *target, *budget); err == nil && route.Runs(e) > 0 && reached(r) &&
				(grouped == nil || r.Seconds < grouped.res.Seconds) {
				grouped = &cand{winner, e, r}
			}
		}
		try("guide", ge)
		try("plan", pe)
		for i, path := range strings.Split(*seed, ",") {
			if path == "" {
				continue
			}
			f, err := readRoute(path)
			if err != nil {
				if i == 0 {
					return err
				}
				continue // another class's route not there yet
			}
			var sw struct {
				Winner    string          `json:"winner"`
				Character route.Character `json:"character"`
			}
			raw, _ := os.ReadFile(path)
			json.Unmarshal(raw, &sw)
			events := f.Events
			if sw.Character.Class != w.Char.Class {
				// another class's route: this class's own class quests put in, the other's left out by the follower
				e, _, err := w.Improve(events, *target, *budget/2)
				if err != nil {
					continue
				}
				events = e
				if r, err := w.FollowGuide(events, *target); err == nil && route.Runs(events) > 0 && reached(r) &&
					(grouped == nil || r.Seconds < grouped.res.Seconds) {
					grouped = &cand{sw.Winner, events, r}
				}
			} else if sr, err := w.FollowGuide(events, *target); err == nil && reached(sr) && sr.Seconds < best.res.Seconds {
				best = cand{sw.Winner, events, sr} // the solo route, made without dungeons: kept when no run pays
			}
			try(sw.Winner, events)
		}
		if grouped != nil {
			rep.Hours[map[string]string{"guide": "guideImproved", "plan": "planImproved"}[grouped.winner]] = grouped.res.Seconds / 3600
			if grouped.res.Seconds < best.res.Seconds {
				best = *grouped
			}
		}
		set(best)
	}
	rep.Events = w.WithoutExtraOptions(rep.Events, *target) // an option's chain is the addon's to switch on
	rep.Class = w.ClassReport(rep.Events, *target)
	fmt.Fprintf(os.Stderr, "%s %s: guide %.1f h -> %.1f h, plan %.1f h -> %.1f h, winner %s (%.0f s)\n", *race, *class,
		rep.Hours["guide"], rep.Hours["guideImproved"], rep.Hours["plan"], rep.Hours["planImproved"], rep.Winner, rep.SearchSecs)
	if *out == "" {
		return printJSON(rep)
	}
	b, _ := json.MarshalIndent(rep, "", " ")
	return os.WriteFile(*out, b, 0o644)
}
