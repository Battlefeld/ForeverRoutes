package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Battlefeld/ForeverRoutes/fallback"
	"github.com/Battlefeld/ForeverRoutes/route"
)

// liveRequest is what ForeverCore hands over: who the character is and what the game shows of it.
type liveRequest struct {
	Race    string     `json:"race"`
	Class   string     `json:"class"`
	Faction string     `json:"faction,omitempty"` // A or H (Alliance, Horde): needed for the Skyborne
	Live    route.Live `json:"live"`
	Options []int      `json:"options,omitempty"` // the route's options switched on in the addon (their goal quests)
}

// nextReply is the next stretch of route.
type nextReply struct {
	Schema    string        `json:"schema"`
	Mode      string        `json:"mode"` // route: following the character's best route; plan: planned freely
	Level     int           `json:"level"`
	Horizon   float64       `json:"horizon"`
	Seconds   float64       `json:"seconds"`
	ToLevel   int           `json:"toLevel"`
	Profile   string        `json:"killProfile"`
	Assumed   int           `json:"assumedDropSteps"`
	Notes     []string      `json:"notes,omitempty"`
	Stages    []route.Stage `json:"stages,omitempty"` // the legs of the next hours: zones, cities, dungeon runs
	Steps     []route.Step  `json:"steps"`
	PlannedMS int64         `json:"plannedMs"`
}

func cmdNext(args []string) error {
	fs := newFlags("next")
	path := fs.String("data", "data/routes-data.json.gz", "dataset")
	observations := fs.String("client-quests", "", "optional local build-scoped client quest observations")
	profile := fs.String("profile", automaticProfile, combatProfileHelp)
	live := fs.String("live", "-", "live state (JSON {race, class, live}); - = standard input")
	horizon := fs.Float64("horizon", 3600, "seconds of play to plan ahead")
	routeFile := fs.String("route", "auto", "auto = shipped route with RXP fallback; a route path; empty = plan freely")
	routesDir := fs.String("routes", "", "shipped route directory; default beside dataset")
	fallbackFile := fs.String("fallbacks", "", "ordinary RXP fallback catalog; default beside dataset")
	out := fs.String("out", "", "write the reply here instead of standard output")
	group := fs.Bool("dungeons", false, "route with dungeons: dungeon runs with their quests may be planned (also when the route runs dungeons)")
	window := fs.Duration("optimize", 3*time.Second, "time to re-order the plan's next quest steps from the live state (0: keep the route's order)")
	blockBudget := fs.Duration("block", 0, "time to plan the next block of quests for XP per hour alone (0: off; the stages plan further)")
	stagesBudget := fs.Duration("stages", 6*time.Second, "time to plan the next hours by zones for XP per hour (0: follow the route only)")
	feedbackPolicy := fs.String("feedback-policy", "", "optional locally approved post-release ForeverSim feedback settings; disabled by default")
	fs.Parse(args)
	if *routesDir == "" {
		*routesDir = filepath.Join(filepath.Dir(*path), "routes")
	}
	if *fallbackFile == "" {
		*fallbackFile = filepath.Join(filepath.Dir(*path), "fallbacks", "rxp.json.gz")
	}
	var raw []byte
	var err error
	if *live == "-" {
		raw, err = io.ReadAll(os.Stdin)
	} else {
		raw, err = os.ReadFile(*live)
	}
	if err != nil {
		return err
	}
	var req liveRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return fmt.Errorf("live state: %w", err)
	}
	t0 := time.Now()
	var events []route.GuideEvent
	var file routeFileData
	var selectionNotes []string
	fallbackMode := false
	character := route.Character{Race: req.Race, Class: req.Class, Faction: req.Faction}
	loadFallback := func() error {
		cat, err := fallback.Load(*fallbackFile)
		if err != nil {
			return err
		}
		r, err := cat.For(character)
		if err != nil {
			return err
		}
		file = routeFileData{Events: r.Events, Character: r.Character, Target: r.Target}
		events = r.Events
		fallbackMode = true
		selectionNotes = append(selectionNotes, "ordinary RXP fallback selected; CLI uses a quest-workload projection; original chapter instructions are preserved for the addon")
		return nil
	}
	if *routeFile == "auto" {
		key := character.RouteKey()
		if *group {
			key += "-dungeons"
		}
		selected := filepath.Join(*routesDir, key+".json")
		file, err = readRoute(selected)
		if err != nil || file.Character.Race == "" || !route.SameCharacter(file.Character, character) || file.Revision != route.TimelineModelRevision {
			selectionNotes = append(selectionNotes, "optimized route missing, invalid or for another character; using RXP fallback")
			if err = loadFallback(); err != nil {
				return fmt.Errorf("no usable optimized route or RXP fallback: %w", err)
			}
		}
		events = file.Events
	} else {
		if *routeFile != "" {
			if file, err = readRoute(*routeFile); err != nil {
				return err
			}
			events = file.Events
		}
	}
	if file.Character.Race != "" && !route.SameCharacter(file.Character, character) {
		return fmt.Errorf("route character does not match live character")
	}
	// a stretch planned by zones for this character and level (routes-tests beside the routes): planned anew from the
	// live state with its measured model, so the route goes on from the quests the character has (solo: also when
	// the route with dungeons is selected, the stretch has none)
	var stretch *stretchFile
	if *routeFile == "auto" {
		stretch = findStretch(filepath.Join(filepath.Dir(filepath.Clean(*routesDir)), "routes-tests"), character, req.Live.Level)
		if stretch != nil {
			file = stretch.routeFileData
			events = nil
			selectionNotes = append(selectionNotes, fmt.Sprintf("stretch %q: planned by zones from the live state to %d", stretch.TestZone, stretch.Target))
		}
	}
	load := func() (*route.World, error) {
		return loadWorldWithClientQuests(*path, req.Race, req.Class, req.Faction, *profile, *observations, *group || hasDungeons(events))
	}
	if stretch != nil {
		load = func() (*route.World, error) {
			return withModel(stretch.RouteModel, func() (*route.World, error) {
				return loadWorldWithClientQuests(*path, req.Race, req.Class, req.Faction, *profile, *observations, false)
			})
		}
	}
	w, err := load()
	if err != nil {
		return err
	}
	if *feedbackPolicy != "" {
		raw, err := os.ReadFile(*feedbackPolicy)
		if err != nil {
			return err
		}
		var policy route.FeedbackSettings
		if err = json.Unmarshal(raw, &policy); err != nil {
			return err
		}
		if !policy.Active() {
			return fmt.Errorf("feedback policy requires release confirmation and explicit post-release user approval")
		}
		w.Opt.Feedback = policy
	}
	previousPolicy, previousBag := w.Preparations(), w.Opt.SleepingBag
	if err := file.RouteModel.apply(w); err != nil {
		if *routeFile != "auto" || fallbackMode {
			return err
		}
		selectionNotes = append(selectionNotes, "invalid optimized route model: "+err.Error())
		if err = loadFallback(); err != nil {
			return err
		}
		if err = w.SetPreparations(previousPolicy); err != nil {
			return err
		}
		if err = w.SetSleepingBagModel(previousBag); err != nil {
			return err
		}
	}
	w.SetWandDetour(file.WandDetour) // as the route was made
	if len(req.Options) > 0 && len(events) > 0 {
		// the options switched on, put in as the addon shows them
		var opts [][]route.GuideEvent
		var qs []map[int]bool
		for _, id := range req.Options {
			for _, c := range file.Options {
				if c.Quest == id && len(c.Events) > 0 {
					set := map[int]bool{}
					for _, q := range c.Quests {
						set[q] = true
					}
					opts, qs = append(opts, c.Events), append(qs, set)
				}
			}
		}
		events = route.WithOptions(events, opts, qs)
		var goals [][]int
		for _, id := range req.Options {
			for _, c := range file.Options {
				if c.Quest == id && len(c.Events) > 0 {
					goals = append(goals, c.Quests)
				}
			}
		}
		w.SetLiveGoals(goals) // the stages take these chains along too, where they cost least
	}
	if fb := req.Live.CombatFeedback; fb != nil && fb.Source == "measured" {
		fb.Character = w.Char // the bridge measures this character; it names race and class in the game's terms
	}
	s, notes, err := w.StateFrom(req.Live)
	if err != nil {
		return err
	}
	notes = append(notes, selectionNotes...)
	if left := w.GoalLeft(s); left > 0 {
		notes = append(notes, fmt.Sprintf("goals switched on: %.0f min of travel left", left/60))
	}
	if pf, pr := w.Pace(); pf != 1 || pr != 1 {
		notes = append(notes, fmt.Sprintf("measured pace: fights x%.2f, rests x%.2f of the model", pf, pr))
	}
	var res *route.Result
	mode := "plan"
	if stretch != nil {
		// no trip to the profession trainers or the camping vendor on a stretch: it was planned and measured without
		p := w.Preparations()
		if p.Camping || p.ForceProfessions {
			p.Camping, p.ForceProfessions = false, false
			if err := w.SetPreparations(p); err != nil {
				return err
			}
			s.Preparation.Camping, s.Preparation.ForceProfessions = false, false // the state took them already
		}
		// Steady: the stretch's plan is kept and followed from the live state while it still fits (done quests drop out,
		// the order stays); planned anew by zones only when the character left it (a step of it cannot be followed any
		// more) or there is none yet. Planned anew after every quest change, the order jumped between near equal plans.
		cache := stretchCache(character, stretch, *path)
		set := map[int]bool{}
		for _, z := range stretch.Zones {
			for id := range zoneQuests(w, s, z, stretch.Target+3, nil, map[int]bool{}) {
				set[id] = true
			}
		}
		var fresh *route.Result
		var freshEvents []route.GuideEvent
		if sw, _, err := w.Sweep(set, route.SweepOptions{Target: stretch.Target, Near: 120, Out: 900, Reach: 500, Zones: true, From: s}); err == nil {
			fresh, freshEvents = sw.Head(*horizon, route.BlockQuests+2), route.EventsOf(sw.Steps)
		} else {
			notes = append(notes, "stretch could not be planned from the live state: "+err.Error())
		}
		if old := readStretchPlan(cache); len(old) > 0 {
			if g, err := w.FollowGuideFrom(s, old, stretch.Target, *horizon); err == nil && len(g.Missed) == 0 && len(g.Steps) > 0 {
				// the new plan takes over only when it is clearly better (15 % more XP per hour over the next hours)
				if fresh == nil || route.XPRate(s, fresh) < stretchSwitch*route.XPRate(s, g.Result) {
					res, mode = g.Result, "stretch"
					notes = append(notes, "stretch: the plan so far, followed from the live state")
				} else {
					notes = append(notes, fmt.Sprintf("stretch: a new plan is %.0f %% faster, it takes over", (route.XPRate(s, fresh)/route.XPRate(s, g.Result)-1)*100))
				}
			} else if err == nil && len(g.Missed) > 0 {
				notes = append(notes, fmt.Sprintf("stretch: the plan so far does not fit any more (%s %d: %s), planned anew", g.Missed[0].Op, g.Missed[0].Quest, g.Missed[0].Reason))
			}
		}
		if res == nil && fresh != nil {
			res, mode = fresh, "stretch"
			writeStretchPlan(cache, freshEvents)
			notes = append(notes, "stretch: planned anew by zones from the live state")
		}
	}
	if res == nil && len(events) > 0 {
		// the best route known for this character; when nothing of it is left, plan freely
		target := file.Target
		if target == 0 {
			target = 60
		}
		if g, err := w.FollowGuideFrom(s, events, target, *horizon); err == nil {
			res, mode = g.Result, "route"
			if fallbackMode {
				mode = "rxp-fallback"
			}
		} else if *routeFile == "auto" && !fallbackMode && req.Live.Level < 20 {
			notes = append(notes, "optimized route failed from live state: "+err.Error())
			if err := loadFallback(); err != nil {
				return err
			}
			if g, err := w.FollowGuideFrom(s, events, file.Target, *horizon); err == nil {
				res, mode = g.Result, "rxp-fallback"
			}
		}
	}
	if res == nil {
		if res, err = w.PlanAhead(s, route.PlanOptions{TargetLevel: 60}, *horizon); err != nil {
			return err
		}
	}
	if *window > 0 && mode != "plan" && mode != "stretch" {
		if better, changed := w.ImproveLive(s, res, *window); changed {
			notes = append(notes, fmt.Sprintf("next steps re-ordered from the live state: %.0f s faster", res.Seconds-better.Seconds))
			res = better
		}
	}
	// The next block of quests for the most XP per hour from where the character stands, against the route's way on.
	// From level 6 on (before, the class quests of the start lead) and while the route's next steps hold no
	// preparation (trainers, wands, sleeping bag, a dungeon run): those the free block does not plan.
	if *blockBudget > 0 && mode == "route" && req.Live.Level >= 6 && !preparationAhead(res, 60) {
		if b := w.PlanBlock(s, route.PlanOptions{TargetLevel: 20}, *blockBudget); b != nil {
			rb, rr := route.XPRate(s, b), route.XPRate(s, res)
			if rb > rr*1.03 {
				notes = append(notes, fmt.Sprintf("next block planned for XP per hour: %.0f instead of %.0f XP/h", rb*3600, rr*3600))
				res, mode = b, "block"
			}
		}
	}
	// The next hours by zones (live_stages.go): where to go and in which order, each zone worked off before the next,
	// against the route's way on over the same stretch. Not while the route's next steps hold a trainer or the wand
	// (those the stages do not plan); dungeon runs and the sleeping bag they plan themselves.
	full := res
	if *stagesBudget > 0 && (mode == "route" || mode == "rxp-fallback") && req.Live.Level >= 6 && !trainingAhead(res, 60) {
		if st, _ := w.PlanStages(s, route.PlanOptions{TargetLevel: 60}, *stagesBudget); st != nil {
			ref := res
			if g, err := w.FollowGuideFrom(s, events, cmp.Or(file.Target, 60), route.StageHorizon); err == nil {
				ref = g.Result
			}
			if rs, rr := route.XPRate(s, st), route.XPRate(s, ref); rs >= stagesPreferred*rr {
				notes = append(notes, fmt.Sprintf("next hours planned by zones: %.0f instead of %.0f XP/h", rs*3600, rr*3600))
				res, full, mode = st.Head(*horizon, route.BlockQuests+2), st, "stages"
			} else {
				full = ref
			}
		}
	}
	reply := nextReply{Schema: "foreverroutes.next.v1", Mode: mode, Level: req.Live.Level, Horizon: *horizon, Seconds: res.Seconds, ToLevel: res.Level,
		Profile: res.Profile, Assumed: res.Assumed, Notes: notes, Stages: w.Stages(s, full), Steps: res.Steps, PlannedMS: time.Since(t0).Milliseconds()}
	if *out == "" {
		return printJSON(reply)
	}
	b, _ := json.MarshalIndent(reply, "", " ")
	return os.WriteFile(*out, b, 0o644)
}

// routeFileData is what next takes from a route file: its events, its options and its wand detour.
type routeFileData struct {
	RouteModel
	Character  route.Character    `json:"character"`
	Target     int                `json:"target"`
	Profile    string             `json:"killProfile"`
	Revision   string             `json:"modelRevision"`
	Events     []route.GuideEvent `json:"events"`
	Options    []route.ClassCost  `json:"classQuests"`
	WandDetour float64            `json:"wandDetour"`
}

// stretchFile is a test stretch (data/routes-tests): a route from a character's state over a few levels, with the
// zones it was planned over.
type stretchFile struct {
	routeFileData
	TestZone string      `json:"testZone"`
	Zones    []string    `json:"zones"`
	Start    *route.Live `json:"start"`
}

// stretchSwitch: how much more XP per hour a new plan of a stretch needs to replace the one being followed
const stretchSwitch = 1.15

// stretchCache is where the plan of a stretch is kept between calls (the user's cache folder): per character, stretch
// and dataset, so a new dataset or stretch starts a new plan.
func stretchCache(c route.Character, st *stretchFile, dataPath string) string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	stamp := ""
	if fi, err := os.Stat(dataPath); err == nil {
		stamp = fmt.Sprintf("%d", fi.Size())
	}
	name := strings.NewReplacer(" ", "_", "/", "_").Replace(fmt.Sprintf("%s-%s-%s.json", c.RouteKey(), st.TestZone, stamp))
	return filepath.Join(dir, "ForeverRoutes", "stretch", name)
}

func readStretchPlan(path string) []route.GuideEvent {
	if path == "" {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var ev []route.GuideEvent
	if json.Unmarshal(raw, &ev) != nil {
		return nil
	}
	return ev
}

func writeStretchPlan(path string, ev []route.GuideEvent) {
	if path == "" {
		return
	}
	if os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	b, _ := json.Marshal(ev)
	_ = os.WriteFile(path, b, 0o644)
}

// findStretch: the stretch for this character whose levels hold the live level (from its start to its target).
func findStretch(dir string, c route.Character, level int) *stretchFile {
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	sort.Strings(files)
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var st stretchFile
		if json.Unmarshal(raw, &st) != nil || st.Start == nil || len(st.Zones) == 0 || !route.SameCharacter(st.Character, c) {
			continue
		}
		if level >= st.Start.Level && level < st.Target {
			return &st
		}
	}
	return nil
}

// readRoute reads guide steps: a plain list, or a route file (the output of improve).
func readRoute(path string) (routeFileData, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return routeFileData{}, err
	}
	var events []route.GuideEvent
	if json.Unmarshal(raw, &events) == nil {
		return routeFileData{Events: events}, nil
	}
	var f routeFileData
	if err := json.Unmarshal(raw, &f); err != nil || len(f.Events) == 0 {
		return routeFileData{}, fmt.Errorf("%s: not a route", path)
	}
	return f, nil
}

// trainingAhead: the plan's first n steps go to the profession trainers or make the wand.
func trainingAhead(r *route.Result, n int) bool {
	for i, st := range r.Steps {
		if i >= n {
			break
		}
		if st.Op == "prof" || st.Op == "wand" {
			return true
		}
	}
	return false
}

// preparationAhead: the plan's first n steps hold a step the free block planner does not plan (the profession
// trainers, the wands, the sleeping bag, a dungeon run).
func preparationAhead(r *route.Result, n int) bool {
	for i, st := range r.Steps {
		if i >= n {
			break
		}
		switch st.Op {
		case "prof", "wand", "sleep", "dungeon":
			return true
		}
	}
	return false
}

// stagesPreferred: the stages are taken while they bring at least this share of the route's XP per hour over the same
// stretch: following the route from a state it was not made for sends the player back and forth, the stages do not.
const stagesPreferred = 0.9
