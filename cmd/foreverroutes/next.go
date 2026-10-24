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
	Race  string     `json:"race"`
	Class string     `json:"class"`
	Live  route.Live `json:"live"`
}

// nextReply is the next stretch of route.
type nextReply struct {
	Schema    string       `json:"schema"`
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
	out := fs.String("out", "", "write the reply here instead of standard output")
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
	w, err := loadWorld(*path, req.Race, req.Class, *profile)
	if err != nil {
		return err
	}
	s, notes, err := w.StateFrom(req.Live)
	if err != nil {
		return err
	}
	res, err := w.PlanAhead(s, route.PlanOptions{TargetLevel: 60}, *horizon)
	if err != nil {
		return err
	}
	reply := nextReply{Schema: "foreverroutes.next.v1", Level: req.Live.Level, Horizon: *horizon, Seconds: res.Seconds, ToLevel: res.Level,
		Profile: res.Profile, Assumed: res.Assumed, Notes: notes, Steps: res.Steps, PlannedMS: time.Since(t0).Milliseconds()}
	if *out == "" {
		return printJSON(reply)
	}
	b, _ := json.MarshalIndent(reply, "", " ")
	return os.WriteFile(*out, b, 0o644)
}
