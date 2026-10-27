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

func cmdTimeline(args []string) error {
	fs := newFlags("timeline")
	data := fs.String("data", "data/routes-data.json.gz", "dataset")
	race := fs.String("race", "Undead", "race")
	class := fs.String("class", "Warlock", "class")
	faction := fs.String("faction", "", "A or H for Skyborne")
	profile := fs.String("profile", "", "fixed combat profile")
	paths := fs.String("seeds", "", "comma-separated independent route files or event lists")
	target := fs.Int("target", 20, "target level; elapsed time is minimized, not fixed")
	budget := fs.Duration("budget", 3*time.Minute, "total search budget")
	seed := fs.Int64("seed", 20261027, "search seed")
	iterations := fs.Int("iterations", 0, "optional deterministic cap on random search iterations")
	dungeons := fs.Bool("dungeons", true, "allow dungeons, including the choice of none")
	require := fs.String("require-dungeons", "", "comma-separated dungeon keys that must actually be run")
	exclude := fs.String("exclude-dungeons", "", "comma-separated dungeon keys forbidden in the final route")
	camping := fs.String("camping", "auto", "auto/on/off")
	wands := fs.String("wands", "auto", "auto/on/off; on requires crafting a wand")
	bag := fs.String("sleeping-bag", "auto", "auto/on/off; on requires acquisition, resting is optimized")
	duration := fs.Duration("bag-duration", 2*time.Hour, "bag buff duration assumption (compare 1h and 2h)")
	checkpoint := fs.String("checkpoint", "", "persistent incumbent; defaults beside output or in data/checkpoints")
	out := fs.String("out", "", "write events, chosen preparations, full replay and search diagnostics")
	fs.Parse(args)
	if *paths == "" {
		return fmt.Errorf("-seeds must name at least one route file")
	}
	base, err := loadWorld(*data, *race, *class, *faction, *profile, *dungeons)
	if err != nil {
		return err
	}
	opt := base.Opt
	opt.Preparations = route.PreparationConstraints{Camping: *camping, Wands: *wands, SleepingBag: *bag}
	opt.SleepingBag.Duration = duration.Seconds()
	w, err := route.NewWorld(base.Data, base.Char, opt)
	if err != nil {
		return err
	}
	var seeds []route.OptimizationSeed
	for _, path := range strings.Split(*paths, ",") {
		raw, err := os.ReadFile(strings.TrimSpace(path))
		if err != nil {
			return err
		}
		var s route.OptimizationSeed
		var identity struct {
			Character route.Character `json:"character"`
		}
		if json.Unmarshal(raw, &identity) == nil && identity.Character.Race != "" && !route.SameCharacter(identity.Character, w.Char) {
			return fmt.Errorf("seed %s belongs to another character", path)
		}
		if err = json.Unmarshal(raw, &s); err != nil {
			if err = json.Unmarshal(raw, &s.Events); err != nil {
				return fmt.Errorf("seed %s: %w", path, err)
			}
		}
		if len(s.Events) == 0 {
			return fmt.Errorf("seed %s has no events", path)
		}
		seeds = append(seeds, s)
	}
	split := func(s string) []string {
		var out []string
		for _, k := range strings.Split(s, ",") {
			if k = strings.TrimSpace(k); k != "" {
				out = append(out, k)
			}
		}
		return out
	}
	cp := *checkpoint
	if cp == "" {
		if *out != "" {
			cp = *out + ".best.json"
		} else {
			cp = filepath.Join("data", "checkpoints", strings.ToLower(*race+"-"+*class)+fmt.Sprintf("-%d.best.json", *target))
		}
	}
	if seeds[0].Preparations != nil {
		if err := w.SetPreparations(*seeds[0].Preparations); err != nil {
			return err
		}
	}
	w.SetWandDetour(seeds[0].WandDetour)
	stats := &route.OptimizeStats{}
	e, r, err := w.Optimize(seeds[0].Events, route.OptimizeOptions{Target: *target, Budget: *budget, Seed: *seed, Keep: nil, Stats: stats, Seeds: seeds[1:], RequireDungeons: split(*require), ExcludeDungeons: split(*exclude), Checkpoint: cp, MaxIterations: *iterations})
	if err != nil {
		return err
	}
	reply := map[string]any{
		"schema": "foreverroutes.timeline.v1", "modelRevision": route.TimelineModelRevision,
		"character": w.Char, "target": *target, "winner": "timeline",
		"basis":       "ForeverRoutes optimization of independent input routes; original guides remain unchanged",
		"sourceSeeds": strings.Split(*paths, ","), "events": e, "preparations": w.Preparations(),
		"wandDetour": w.WandDetour(), "sleepingBagModel": w.Opt.SleepingBag, "constraints": w.Opt.Preparations,
		"requireDungeons": split(*require), "excludeDungeons": split(*exclude),
		"result": r, "search": stats, "checkpoint": cp, "killProfile": r.Profile,
		"classQuests": w.ClassReport(e, *target),
		"hours":       map[string]float64{"optimized": r.Seconds / 3600}, "quests": r.Quests,
	}

	b, err := json.MarshalIndent(reply, "", " ")
	if err != nil {
		return err
	}
	if *out == "" {
		fmt.Println(string(b))
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0755); err != nil {
		return err
	}
	return os.WriteFile(*out, append(b, '\n'), 0644)
}
