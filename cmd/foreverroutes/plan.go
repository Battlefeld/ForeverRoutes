package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/route"
)

// loadWorld opens the dataset; profile is a kill profile written by ForeverCore ("" = the default estimate).
func loadWorld(path, race, class, faction, profile string) (*route.World, error) {
	d, err := dataset.Load(path)
	if err != nil {
		return nil, err
	}
	c := route.Character{Race: race, Class: class, Faction: faction}
	var opt route.Options
	if profile != "" {
		f, err := os.Open(profile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		p, err := route.LoadTableProfile(f)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", profile, err)
		}
		if !p.Fits(c) {
			return nil, fmt.Errorf("%s is a %s profile, not %s", profile, p.Class, class)
		}
		opt.Profile = p
	}
	return route.NewWorld(d, c, opt)
}

func cmdPlan(args []string) error {
	fs := newFlags("plan")
	path := fs.String("data", "data/routes-data.json.gz", "dataset")
	race := fs.String("race", "Orc", "race")
	class := fs.String("class", "Warlock", "class")
	faction := fs.String("faction", "", "A or H (only for Skyborne)")
	profile := fs.String("profile", "", "kill profile from ForeverCore (job kill-profile); empty = default estimate")
	target := fs.Int("target", 30, "target level")
	restarts := fs.Int("restarts", 4, "search runs")
	seed := fs.Int64("seed", 1, "seed")
	out := fs.String("out", "", "write the route (JSON) here")
	fs.Parse(args)
	w, err := loadWorld(*path, *race, *class, *faction, *profile)
	if err != nil {
		return err
	}
	t0 := time.Now()
	res, err := w.Plan(route.PlanOptions{TargetLevel: *target, Restarts: *restarts, Seed: *seed})
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "%s %s [%s]: level %d in %.1f h (travel %.1f h, combat %.1f h), %d quests, %d steps, planned in %s\n",
		*race, *class, res.Profile, res.Level, res.Seconds/3600, res.Travel/3600, res.Combat/3600, res.Quests, len(res.Steps), time.Since(t0).Round(time.Millisecond))
	if *out != "" {
		b, _ := json.MarshalIndent(res, "", " ")
		return os.WriteFile(*out, b, 0o644)
	}
	return nil
}

func cmdReplay(args []string) error {
	fs := newFlags("replay")
	path := fs.String("data", "data/routes-data.json.gz", "dataset")
	race := fs.String("race", "Orc", "race")
	class := fs.String("class", "Warlock", "class")
	faction := fs.String("faction", "", "A or H (only for Skyborne)")
	profile := fs.String("profile", "", "kill profile from ForeverCore (job kill-profile); empty = default estimate")
	file := fs.String("route", "", "route file (output of plan)")
	fs.Parse(args)
	w, err := loadWorld(*path, *race, *class, *faction, *profile)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	var r route.Result
	if err := json.Unmarshal(raw, &r); err != nil {
		return err
	}
	actions := make([]route.Action, len(r.Steps))
	for i, s := range r.Steps {
		actions[i] = s.Action
	}
	res, err := w.Replay(actions)
	if err != nil {
		return err
	}
	res.Steps = nil
	return printJSON(res)
}

func cmdGuide(args []string) error {
	fs := newFlags("guide")
	path := fs.String("data", "data/routes-data.json.gz", "dataset")
	race := fs.String("race", "Orc", "race")
	class := fs.String("class", "Warlock", "class")
	faction := fs.String("faction", "", "A or H (only for Skyborne)")
	profile := fs.String("profile", "", "kill profile from ForeverCore (job kill-profile); empty = default estimate")
	target := fs.Int("target", 30, "target level")
	file := fs.String("events", "", `guide events: JSON list of {"op":"accept|complete|turnin|level","quest":id,"level":n}`)
	out := fs.String("out", "", "write the walked route (JSON) here")
	fs.Parse(args)
	w, err := loadWorld(*path, *race, *class, *faction, *profile)
	if err != nil {
		return err
	}
	raw, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	var events []route.GuideEvent
	if err := json.Unmarshal(raw, &events); err != nil {
		return err
	}
	res, err := w.FollowGuide(events, *target)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "%s %s [%s] following the guide: level %d in %.1f h (travel %.1f h, combat %.1f h, of it grinding %.1f h), %d guide quests, skipped %v\n",
		*race, *class, res.Profile, res.Level, res.Seconds/3600, res.Travel/3600, res.Combat/3600, res.Grind/3600, res.Quests, res.Skipped)
	if *out != "" {
		b, _ := json.MarshalIndent(res, "", " ")
		return os.WriteFile(*out, b, 0o644)
	}
	return nil
}
