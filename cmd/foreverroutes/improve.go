package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/Battlefeld/ForeverRoutes/route"
)

// bestReply is the faster of the improved guide and the improved planned route for one character.
type bestReply struct {
	RouteModel
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
	splice := fs.String("splice", "", "with -dungeons: routes of the side's other races (comma-separated) this route may switch to at a level (-splice-levels), with dungeons put in")
	spliceLevels := fs.String("splice-levels", "6,8,10,12,14", "the levels a route may switch at")
	spliceKeep := fs.Int("splice-keep", 4, "the switched routes improved: the fastest by a first walk with dungeons")
	optimize := fs.Duration("optimize", 0, "then search the route as a timeline (route.Optimize) this long, on -workers cores with their own seeds")
	workers := fs.Int("workers", 4, "with -optimize: searches in parallel")
	rounds := fs.Int("rounds", 4, "with -optimize: rounds, each from the best route of the one before")
	keep := fs.String("keep", "", "with -optimize: quests the route must hand in (comma-separated): an option switched on")
	checkpoint := fs.String("checkpoint", "", "persistent best route for timeline optimization (defaults beside output)")
	fs.Parse(args)
	if *workers < 1 || *rounds < 1 || *budget < 0 || *optimize < 0 {
		return fmt.Errorf("workers/rounds must be positive and budgets nonnegative")
	}
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
		rep.RouteModel = RouteModel{Preparations: &c.res.Preparations, SleepingBagModel: &c.res.BuffModel}
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
			e, r, err := w.Improve(w.WithDungeons(events, *target), *target, *budget)
			if err != nil {
				return
			}
			if m := w.MergeTrips(e, *target); len(m) > 0 {
				if mr, err := w.FollowGuide(m, *target); err == nil && mr.Seconds < r.Seconds {
					e, r = m, mr
				}
			}
			fmt.Fprintf(os.Stderr, "with dungeons from %s: %.2f h, %s\n", winner, r.Seconds/3600, strings.Join(route.RunKeys(e), " "))
			if route.Runs(e) > 0 && reached(r) && (grouped == nil || r.Seconds < grouped.res.Seconds) {
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
			try(path, events)
		}
		if *splice != "" {
			// another race's route from a level on, dungeons put in: walked once, the fastest improved
			type sc struct {
				name   string
				events []route.GuideEvent
				secs   float64
			}
			var cs []sc
			for _, path := range strings.Split(*splice, ",") {
				f, err := readRoute(path)
				if err != nil {
					continue
				}
				for _, l := range strings.Split(*spliceLevels, ",") {
					var lvl int
					fmt.Sscan(l, &lvl)
					s, err := w.Splice(best.events, f.Events, lvl, *target)
					if err != nil {
						continue
					}
					s = w.WithDungeons(s, *target)
					if r, err := w.FollowGuide(s, *target); err == nil && reached(r) {
						cs = append(cs, sc{fmt.Sprintf("%s@%d", path, lvl), s, r.Seconds})
					}
				}
			}
			sort.Slice(cs, func(i, j int) bool { return cs[i].secs < cs[j].secs })
			for i, c := range cs {
				if i >= *spliceKeep {
					break
				}
				if e, r, err := w.Improve(c.events, *target, *budget/2); err == nil && route.Runs(e) > 0 && reached(r) {
					fmt.Fprintf(os.Stderr, "splice %s: %.2f h -> %.2f h\n", c.name, c.secs/3600, r.Seconds/3600)
					if grouped == nil || r.Seconds < grouped.res.Seconds {
						grouped = &cand{"plan", e, r}
					}
				}
			}
		}
		if grouped != nil {
			rep.Hours[map[string]string{"guide": "guideImproved", "plan": "planImproved"}[grouped.winner]] = grouped.res.Seconds / 3600
			if grouped.res.Seconds < best.res.Seconds {
				best = *grouped
			}
		}
		set(best)
	}
	var keepIDs []int
	for _, k := range strings.Split(*keep, ",") {
		var id int
		if _, err := fmt.Sscan(k, &id); err == nil {
			keepIDs = append(keepIDs, id)
		}
	}
	if len(keepIDs) > 0 {
		// an option switched on: its chain woven in first, then searched with the route
		for _, c := range w.ClassChains(*target) {
			if slices.Contains(keepIDs, c.Goal.ID) {
				rep.Events = w.WeaveIn(rep.Events, c, *target)
			}
		}
	}
	if *optimize > 0 {
		checkpointPath := *checkpoint
		if checkpointPath == "" {
			if *out != "" {
				checkpointPath = *out + ".best.json"
			} else {
				checkpointPath = filepath.Join("data", "checkpoints", strings.ToLower(*race+"-"+*class)+fmt.Sprintf("-%d-improve.best.json", *target))
			}
		}
		type found struct {
			err    error
			events []route.GuideEvent
			res    *route.GuideResult
			wand   float64
			prep   route.PreparationPlan
		}
		before, _ := w.FollowGuide(rep.Events, *target)
		worlds := make([]*route.World, *workers)
		for k := range worlds {
			ww, err := loadWorld(*path, *race, *class, *faction, *profile, *group)
			if err != nil {
				return err
			}
			worlds[k] = ww
		}
		// rounds: the workers search from the best route so far, each with its own seed
		var top *found
		events, wd := rep.Events, wand
		prep := w.Preparations()
		for round := 0; round < *rounds; round++ {
			out := make(chan found, *workers)
			for k, ww := range worlds {
				go func(ww *route.World, seed int64) {
					ww.SetWandDetour(wd)
					ww.SetPreparations(prep)
					e, r, err := ww.Optimize(events, route.OptimizeOptions{Target: *target, Budget: *optimize / time.Duration(*rounds), Seed: seed, Keep: keepIDs, Checkpoint: checkpointPath})
					if err != nil {
						out <- found{err: err}
						return
					}
					out <- found{nil, e, r, ww.WandDetour(), ww.Preparations()}
				}(ww, int64(round**workers+k+1))
			}
			var lastError error
			successes := 0
			for range worlds {
				f := <-out
				if f.err != nil {
					lastError = f.err
					fmt.Fprintf(os.Stderr, "optimizer worker: %v\n", f.err)
				} else {
					successes++
				}
				if f.res != nil && (top == nil || f.res.Seconds < top.res.Seconds) {
					top = &f
				}
			}
			if successes == 0 {
				return fmt.Errorf("all optimizer workers failed: %w", lastError)
			}
			if top != nil {
				events, wd = top.events, top.wand
				prep = top.prep
				fmt.Fprintf(os.Stderr, "optimize round %d: %.2f h\n", round+1, top.res.Seconds/3600)
			}
		}
		if top != nil && before != nil && top.res.Seconds < before.Seconds {
			fmt.Fprintf(os.Stderr, "optimized: %.2f h -> %.2f h, %s\n", before.Seconds/3600, top.res.Seconds/3600, strings.Join(route.RunKeys(top.events), " "))
			rep.Events, rep.Quests, rep.Profile, rep.Assumed, rep.WandDetour = top.events, top.res.Quests, top.res.Profile, top.res.Assumed, top.wand
			w.SetWandDetour(top.wand)
			w.SetPreparations(top.prep)
			rep.RouteModel = RouteModel{Preparations: &top.res.Preparations, SleepingBagModel: &top.res.BuffModel}
			rep.Hours["optimized"] = top.res.Seconds / 3600
		}
	}
	if len(keepIDs) == 0 && *optimize == 0 {
		rep.Events = w.WithoutExtraOptions(rep.Events, *target) // an option's chain is the addon's to switch on
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
