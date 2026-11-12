package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Battlefeld/ForeverRoutes/route"
)

// cmdExtend continues a route (data/routes) to a higher level: the route is walked to its end and the planner goes on
// from the state it leaves (quests done, place, flight masters), solo or with dungeon runs. The planned part is
// improved by local edits like the routes to 20.
func cmdExtend(args []string) error {
	fs := newFlags("extend")
	path := fs.String("data", "data/routes-data.json.gz", "dataset")
	file := fs.String("route", "", "route file to continue (data/routes/<race>-<class>.json)")
	profiles := fs.String("profiles", "data/profiles", "kill profiles (<class>.json)")
	target := fs.Int("target", 40, "target level")
	group := fs.Bool("dungeons", false, "route with dungeons: dungeon runs with their quests are part of the planned part")
	budget := fs.Duration("budget", 60*time.Second, "search time for improving the planned part")
	restarts := fs.Int("restarts", 2, "search runs of the planner")
	out := fs.String("out", "", "write the continued route here")
	legsFile := fs.String("legs", "", "a laid-out route (JSON list of legs: {zones, until} or {dungeon}) the planner fills in")
	from := fs.Int("from", 0, "continue from this level of the route (default: its end)")
	liveFile := fs.String("live", "", "plan from this character's state instead of the route's (JSON with live as next reads it: level, XP, place, finished quests, quest log); the route then starts there")
	chosen := fs.String("dungeon-keys", "", "only these dungeons (keys, comma separated; \"none\": no dungeon): the player's choice, kept in the route")
	stages := fs.Duration("stages", 0, "plan by zones (the live stages, 4 hours at a time) with this search time per stretch instead of the free planner")
	grind := fs.Float64("grind-share", 0.8, "share of its XP grinding counts with against quests (1: all; below: quest unless grinding is clearly better)")
	fs.Parse(args)
	raw, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	var r routeFile
	if err := json.Unmarshal(raw, &r); err != nil {
		return err
	}
	prof := filepath.Join(*profiles, strings.ToLower(r.Character.Class)+".json")
	if !fileExists(prof) {
		prof = ""
	}
	if *chosen != "" {
		keys := []string{}
		if *chosen != "none" {
			for _, k := range strings.Split(*chosen, ",") {
				if k = strings.TrimSpace(k); k != "" {
					keys = append(keys, k)
				}
			}
		}
		r.RouteModel.DungeonKeys = &keys
	}
	dungeonKeys := r.RouteModel.DungeonKeys
	w, err := withModel(r.RouteModel, func() (*route.World, error) {
		return loadWorld(*path, r.Character.Race, r.Character.Class, r.Character.Faction, prof, *group || hasDungeons(r.Events))
	})
	if err != nil {
		return err
	}
	if err := r.RouteModel.apply(w); err != nil {
		return err
	}
	w.SetWandDetour(r.WandDetour)
	w.Opt.GrindShare = *grind
	t0 := time.Now()
	baseTarget := r.Target
	if *from > 0 && *from < baseTarget {
		baseTarget = *from
	}
	var base *route.GuideResult
	var s *route.State
	if *liveFile != "" {
		// the character's own state: what it has done is done, its quest log is open, it stands where it is
		raw, err := os.ReadFile(*liveFile)
		if err != nil {
			return err
		}
		var req struct {
			Live route.Live `json:"live"`
		}
		if err := json.Unmarshal(raw, &req); err != nil {
			return fmt.Errorf("%s: %w", *liveFile, err)
		}
		if s, _, err = w.StateFrom(req.Live); err != nil {
			return err
		}
		start := req.Live
		r.Start, r.Events = &start, nil
		base = &route.GuideResult{Result: &route.Result{}}
	} else {
		if base, err = w.FollowGuide(r.Events, baseTarget); err != nil {
			return err
		}
		if baseTarget < r.Target {
			r.Events = route.EventsOf(base.Steps) // the route up to -from; the legs go on from there
		}
		if s, err = w.StateAfter(base.Steps); err != nil {
			return err
		}
	}
	follow := func(events []route.GuideEvent) (*route.GuideResult, error) {
		if r.Start != nil {
			return w.FollowGuideFromState(s, events, *target)
		}
		return w.FollowGuide(events, *target)
	}
	var best *route.Result
	if *legsFile != "" {
		raw, err := os.ReadFile(*legsFile)
		if err != nil {
			return err
		}
		var legs []route.Leg
		if err := json.Unmarshal(raw, &legs); err != nil {
			return fmt.Errorf("%s: %w", *legsFile, err)
		}
		res, notes, err := w.PlanLegs(s, legs, *target)
		if err != nil {
			return err
		}
		for _, n := range notes {
			fmt.Fprintln(os.Stderr, "  "+n)
		}
		best = res
	} else if *stages > 0 {
		// stretch after stretch from the end of the last: each zone worked off before the next
		cur := s
		var acts []route.Step
		for w.Level(cur) < *target {
			// the dungeons the player chose are run, each once it is due (the planner alone runs one only where it
			// pays in time)
			if *chosen != "" && dungeonKeys != nil {
				ran := false
				for _, key := range *dungeonKeys {
					run, err := w.RunChosenDungeon(cur, key)
					if err != nil || run == nil || len(run.Steps) == 0 {
						continue
					}
					next, err := w.StateAfterFrom(cur, run.Steps)
					if err != nil || !w.DungeonRun(next, key) {
						continue
					}
					acts = append(acts, run.Steps...)
					fmt.Fprintf(os.Stderr, "  chosen dungeon %s at level %d (quest log %d)\n", key, w.Level(cur), cur.Active)
					cur, ran = next, true
				}
				if ran {
					continue
				}
			}
			// plan only up to the level the next chosen dungeon is due at, so it is run on time
			until := *target
			if *chosen != "" && dungeonKeys != nil {
				for _, key := range *dungeonKeys {
					if due := w.DungeonDueLevel(key); due > w.Level(cur) && due < until && !w.DungeonRun(cur, key) {
						until = due
					}
				}
			}
			res, _ := w.PlanStages(cur, route.PlanOptions{TargetLevel: until}, *stages)
			if res == nil || len(res.Steps) == 0 {
				return fmt.Errorf("stages: nothing to plan at level %d", w.Level(cur))
			}
			acts = append(acts, res.Steps...)
			next, err := w.StateAfterFrom(cur, res.Steps)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "  stretch to level %d (%.1f h)\n", w.Level(next), (next.Time-s.Time)/3600)
			cur = next
		}
		best = &route.Result{Steps: acts, Level: w.Level(cur)}
	}
	for run := 0; best == nil && run < *restarts; run++ {
		res, err := w.PlanAhead(s, route.PlanOptions{TargetLevel: *target, Seed: int64(run + 1)}, 1e9)
		if err == nil && res.Level >= *target && (best == nil || res.Seconds < best.Seconds) {
			best = res
		}
	}
	if best == nil {
		return fmt.Errorf("no route from level %d to %d with this data", base.Level, *target)
	}
	planned := route.EventsOf(best.Steps)
	events := append(append([]route.GuideEvent(nil), r.Events...), planned...)
	full, err := follow(events)
	if err != nil {
		return err
	}
	// the stages may take a quest the route follower refuses (an item start that takes long to farm): its events
	// go, with those of quests depending on it, until the route replays as planned
	for try := 0; try < 5 && len(full.Missed) > 0; try++ {
		drop := map[int]bool{}
		for _, m := range full.Missed {
			if !drop[m.Quest] {
				fmt.Fprintf(os.Stderr, "  quest %d left out: %s %s (%s)\n", m.Quest, m.Op, m.Reason, m.Detail)
			}
			drop[m.Quest] = true
		}
		var kept []route.GuideEvent
		for _, e := range events {
			if !drop[e.Quest] {
				kept = append(kept, e)
			}
		}
		events = kept
		if full, err = follow(events); err != nil {
			return err
		}
	}
	improved, fullBest := events, full
	if *budget > 0 && r.Start == nil {
		if improved, fullBest, err = w.Improve(events, *target, *budget); err != nil {
			return err
		}
	}
	runs := 0
	for _, st := range fullBest.Steps {
		if st.Op == "dungeon" {
			runs++
		}
	}
	fmt.Fprintf(os.Stderr, "%s %s: to %d %.1f h, then to %d %.1f h planned, %.1f h improved (%d dungeon runs, %s)\n",
		r.Character.Race, r.Character.Class, r.Target, base.Seconds/3600, *target, full.Seconds/3600, fullBest.Seconds/3600, runs, time.Since(t0).Round(time.Second))
	r.Target, r.Events = *target, improved
	if *chosen != "" && dungeonKeys != nil {
		// say which chosen dungeons the route could not take (past their levels at the start, or no way in)
		ran := map[string]bool{}
		for _, st := range fullBest.Steps {
			if st.Op == "dungeon" {
				ran[st.Dungeon] = true
			}
		}
		r.DungeonsLeftOut = nil
		for _, key := range *dungeonKeys {
			if !ran[key] {
				r.DungeonsLeftOut = append(r.DungeonsLeftOut, key)
				fmt.Fprintf(os.Stderr, "  chosen dungeon %s not in the route\n", key)
			}
		}
	}
	r.RouteModel = RouteModel{Preparations: &fullBest.Preparations, SleepingBagModel: &fullBest.BuffModel, DungeonKeys: dungeonKeys}
	r.Hours = map[string]float64{"base": round2(base.Seconds / 3600), "planned": round2(full.Seconds / 3600), "improved": round2(fullBest.Seconds / 3600)}
	r.Quests, r.Assumed, r.Dungeons = fullBest.Quests, fullBest.Assumed, runs
	if *out == "" {
		return printJSON(r)
	}
	b, _ := json.MarshalIndent(r, "", " ")
	return os.WriteFile(*out, b, 0o644)
}

// routeFile is a route of data/routes.
type routeFile struct {
	RouteModel
	Schema          string             `json:"schema"`
	Character       route.Character    `json:"character"`
	Target          int                `json:"target"`
	Winner          string             `json:"winner"`
	Hours           map[string]float64 `json:"hours"`
	Quests          int                `json:"quests"`
	Profile         string             `json:"killProfile"`
	Assumed         int                `json:"assumedDropSteps"`
	Dungeons        int                `json:"dungeonRuns,omitempty"`
	Basis           string             `json:"basis"`
	Events          []route.GuideEvent `json:"events"`
	WandDetour      float64            `json:"wandDetour,omitempty"`
	DungeonsLeftOut []string           `json:"dungeonsLeftOut,omitempty"` // chosen dungeons the route could not take
	Start           *route.Live        `json:"start,omitempty"`           // planned from this character's state (extend -live): the events go on from there
}

// hasDungeons: the route runs dungeons (a route with dungeons).
func hasDungeons(events []route.GuideEvent) bool {
	for _, e := range events {
		if e.Op == "dungeon" {
			return true
		}
	}
	return false
}

func round2(v float64) float64 { return float64(int(v*100+0.5)) / 100 }
