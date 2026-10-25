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
	group := fs.Bool("dungeons", false, "group route: dungeon runs with their quests are part of the planned part")
	budget := fs.Duration("budget", 60*time.Second, "search time for improving the planned part")
	restarts := fs.Int("restarts", 2, "search runs of the planner")
	out := fs.String("out", "", "write the continued route here")
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
	w, err := loadWorld(*path, r.Character.Race, r.Character.Class, r.Character.Faction, prof, *group || hasDungeons(r.Events))
	if err != nil {
		return err
	}
	t0 := time.Now()
	base, err := w.FollowGuide(r.Events, r.Target)
	if err != nil {
		return err
	}
	s, err := w.StateAfter(base.Steps)
	if err != nil {
		return err
	}
	var best *route.Result
	for run := 0; run < *restarts; run++ {
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
	full, err := w.FollowGuide(events, *target)
	if err != nil {
		return err
	}
	improved, fullBest, err := w.Improve(events, *target, *budget)
	if err != nil {
		return err
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
	Schema    string             `json:"schema"`
	Character route.Character    `json:"character"`
	Target    int                `json:"target"`
	Winner    string             `json:"winner"`
	Hours     map[string]float64 `json:"hours"`
	Quests    int                `json:"quests"`
	Profile   string             `json:"killProfile"`
	Assumed   int                `json:"assumedDropSteps"`
	Dungeons  int                `json:"dungeonRuns,omitempty"`
	Basis     string             `json:"basis"`
	Events    []route.GuideEvent `json:"events"`
}

// hasDungeons: the route runs dungeons (a group route).
func hasDungeons(events []route.GuideEvent) bool {
	for _, e := range events {
		if e.Op == "dungeon" {
			return true
		}
	}
	return false
}

func round2(v float64) float64 { return float64(int(v*100+0.5)) / 100 }
