package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

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
	Schema    string       `json:"schema"`
	Mode      string       `json:"mode"` // route: following the character's best route; plan: planned freely
	Level     int          `json:"level"`
	Horizon   float64      `json:"horizon"`
	Seconds   float64      `json:"seconds"`
	ToLevel   int          `json:"toLevel"`
	Profile   string       `json:"killProfile"`
	Assumed   int          `json:"assumedDropSteps"`
	Notes     []string     `json:"notes,omitempty"`
	Steps     []route.Step `json:"steps"`
	PlannedMS int64        `json:"plannedMs"`
}

func cmdNext(args []string) error {
	fs := newFlags("next")
	path := fs.String("data", "data/routes-data.json.gz", "dataset")
	profile := fs.String("profile", "", "kill profile from ForeverCore; empty = default estimate")
	live := fs.String("live", "-", "live state (JSON {race, class, live}); - = standard input")
	horizon := fs.Float64("horizon", 3600, "seconds of play to plan ahead")
	routeFile := fs.String("route", "", "the character's best route (output of improve, or a list of guide steps): followed from the live state; empty = plan freely")
	out := fs.String("out", "", "write the reply here instead of standard output")
	group := fs.Bool("dungeons", false, "route with dungeons: dungeon runs with their quests may be planned (also when the route runs dungeons)")
	feedbackPolicy := fs.String("feedback-policy", "", "optional locally approved post-release ForeverSim feedback settings; disabled by default")
	fs.Parse(args)
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
	if *routeFile != "" {
		if file, err = readRoute(*routeFile); err != nil {
			return err
		}
		events = file.Events
	}
	w, err := loadWorld(*path, req.Race, req.Class, req.Faction, *profile, *group || hasDungeons(events))
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
	if err := file.RouteModel.apply(w); err != nil {
		return err
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
	}
	s, notes, err := w.StateFrom(req.Live)
	if err != nil {
		return err
	}
	var res *route.Result
	mode := "plan"
	if len(events) > 0 {
		// the best route known for this character; when nothing of it is left, plan freely
		if g, err := w.FollowGuideFrom(s, events, 60, *horizon); err == nil {
			res, mode = g.Result, "route"
		}
	}
	if res == nil {
		if res, err = w.PlanAhead(s, route.PlanOptions{TargetLevel: 60}, *horizon); err != nil {
			return err
		}
	}
	reply := nextReply{Schema: "foreverroutes.next.v1", Mode: mode, Level: req.Live.Level, Horizon: *horizon, Seconds: res.Seconds, ToLevel: res.Level,
		Profile: res.Profile, Assumed: res.Assumed, Notes: notes, Steps: res.Steps, PlannedMS: time.Since(t0).Milliseconds()}
	if *out == "" {
		return printJSON(reply)
	}
	b, _ := json.MarshalIndent(reply, "", " ")
	return os.WriteFile(*out, b, 0o644)
}

// routeFileData is what next takes from a route file: its events, its options and its wand detour.
type routeFileData struct {
	RouteModel
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
