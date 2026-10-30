package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/Battlefeld/ForeverRoutes/clientquests"
	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/route"
)

// loadWorld opens the dataset; profile is a kill profile written by ForeverCore ("" = the default estimate).
func loadWorld(path, race, class, faction, profile string, dungeons ...bool) (*route.World, error) {
	return loadWorldWithClientQuests(path, race, class, faction, profile, "", dungeons...)
}

func loadWorldWithClientQuests(path, race, class, faction, profile, observations string, dungeons ...bool) (*route.World, error) {
	d, err := dataset.Load(path)
	if err != nil {
		return nil, err
	}
	if observations != "" {
		if _, err := clientquests.Apply(observations, d); err != nil {
			return nil, fmt.Errorf("client quest observations: %w", err)
		}
	}
	c := route.Character{Race: race, Class: class, Faction: faction}
	var opt route.Options
	opt.Dungeons = len(dungeons) > 0 && dungeons[0]
	// the walk grid beside the dataset, when there is one
	if wk := filepath.Join(filepath.Dir(path), "walks.json.gz"); fileExists(wk) {
		if opt.Walks, err = dataset.LoadWalks(wk); err != nil {
			return nil, err
		}
	}
	// the walk table beside it: the measured walks between the places routes go to
	if wt := filepath.Join(filepath.Dir(path), "walktable.json.gz"); fileExists(wt) {
		if opt.WalkTable, err = loadWalkTable(wt); err != nil {
			return nil, err
		}
	}
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

// walkTables: the walk tables read, by path: the worlds of one run (improve's workers) share one.
var walkTables sync.Map

func loadWalkTable(path string) (*dataset.WalkTable, error) {
	if t, ok := walkTables.Load(path); ok {
		return t.(*dataset.WalkTable), nil
	}
	t, err := dataset.LoadWalkTable(path)
	if err != nil {
		return nil, err
	}
	walkTables.Store(path, t)
	return t, nil
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
	group := fs.Bool("dungeons", false, "route with dungeons: dungeon runs with their quests are part of the route")
	cpu := fs.String("cpuprofile", "", "write a CPU profile here")
	fs.Parse(args)
	if *cpu != "" {
		f, err := os.Create(*cpu)
		if err != nil {
			return err
		}
		defer f.Close()
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	w, err := loadWorld(*path, *race, *class, *faction, *profile, *group)
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
	raw, err := os.ReadFile(*file)
	if err != nil {
		return err
	}
	var saved struct {
		RouteModel
		Character  route.Character    `json:"character"`
		Target     int                `json:"target"`
		Profile    string             `json:"killProfile"`
		WandDetour float64            `json:"wandDetour"`
		Steps      []route.Step       `json:"steps"`
		Events     []route.GuideEvent `json:"events"`
		Result     *route.Result      `json:"result"`
	}
	if err = json.Unmarshal(raw, &saved); err != nil {
		return err
	}
	if saved.Character.Race != "" {
		supplied := map[string]bool{}
		fs.Visit(func(f *flag.Flag) { supplied[f.Name] = true })
		if !supplied["race"] {
			*race = saved.Character.Race
		}
		if !supplied["class"] {
			*class = saved.Character.Class
		}
		if !supplied["faction"] {
			*faction = saved.Character.Faction
		}
		if !route.SameCharacter(saved.Character, route.Character{Race: *race, Class: *class, Faction: *faction}) {
			return fmt.Errorf("replay character does not match saved route")
		}
	}
	if saved.Result != nil && len(saved.Steps) == 0 {
		saved.Steps = saved.Result.Steps
	}
	if *profile == "" && strings.HasPrefix(saved.Profile, "MythicSim ") {
		candidate := filepath.Join(filepath.Dir(*path), "profiles", strings.ToLower(*class)+".json")
		if fileExists(candidate) {
			*profile = candidate
		}
	}
	group := hasDungeons(saved.Events)
	for _, st := range saved.Steps {
		group = group || st.Op == "dungeon"
	}
	w, err := loadWorld(*path, *race, *class, *faction, *profile, group)
	if err != nil {
		return err
	}
	if err = saved.RouteModel.apply(w); err != nil {
		return err
	}
	w.SetWandDetour(saved.WandDetour)
	var res *route.Result
	if len(saved.Events) > 0 {
		if saved.Target < 2 {
			return fmt.Errorf("saved event route has no valid target")
		}
		g, err := w.FollowGuide(saved.Events, saved.Target)
		if err != nil {
			return err
		}
		if g.Level < saved.Target || len(g.Skipped) > 0 {
			return fmt.Errorf("saved event route incomplete or contains skipped events")
		}
		res = g.Result
	} else {
		if len(saved.Steps) == 0 {
			return fmt.Errorf("saved route has no events or steps")
		}
		actions := make([]route.Action, len(saved.Steps))
		for i, st := range saved.Steps {
			actions[i] = st.Action
		}
		res, err = w.Replay(actions)
		if err != nil {
			return err
		}
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

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}
