package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	profile := fs.String("profile", "", "kill profile from ForeverCore; empty = default estimate")
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
	if *profile == "" && *routeFile == "auto" && isEngineProfile(file.Profile) {
		candidate := filepath.Join(filepath.Dir(*path), "profiles", strings.ToLower(req.Class)+".json")
		if fileExists(candidate) {
			*profile = candidate
		}
	}
	w, err := loadWorldWithClientQuests(*path, req.Race, req.Class, req.Faction, *profile, *observations, *group || hasDungeons(events))
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
	if len(events) > 0 {
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
	if *window > 0 && mode != "plan" {
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
