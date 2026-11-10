package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/route"
)

// dungeonAssessment is one dungeon at one level: its quests with what leads to them, taken from where most of them
// are given, the run, and the hand-ins, against grinding there.
type dungeonAssessment struct {
	Dungeon   string   `json:"dungeon"`
	Name      string   `json:"name"`
	Level     int      `json:"level"`
	Start     string   `json:"start"`             // where the block begins (the place most of its quests are given)
	Quests    []int    `json:"quests"`            // the dungeon's quests this character can do
	Prereqs   []int    `json:"prereqs"`           // quests outside that lead to them
	QuestXP   int      `json:"questXP"`           // full XP of the dungeon's quests (at their level)
	Ran       bool     `json:"ran"`               // the block ran the dungeon
	TurnedIn  int      `json:"turnedIn"`          // dungeon quests handed in
	PreDone   int      `json:"prereqsDone"`       // quests before them handed in
	XP        int      `json:"xp"`                // gained in the block
	Minutes   float64  `json:"minutes"`           // the block, travel and the run included
	Walk      float64  `json:"walkShare"`         // share of the time travelling
	XPPerHour float64  `json:"xpPerHour"`         // the block
	GrindXPH  float64  `json:"grindXPPerHour"`    // grinding at the start for a level
	Missing   []string `json:"missing,omitempty"` // quests the data cannot plan (no source, no place)
	Far       []string `json:"far,omitempty"`     // quests given or handed in far from the block: taken along when the route is there anyway
	Verdict   string   `json:"verdict"`
}

// farTravel: seconds from the block's home (or the entrance) within which a dungeon quest's giver and turn-in belong
// to the block
const farTravel = 600

// cmdAssessDungeons rates dungeons for levelling: each dungeon's quests and the quests leading to them as one block
// from the place most of them are given, with the run, against grinding there, with the same measured model as
// assess-zones.
func cmdAssessDungeons(args []string) error {
	fs := newFlags("assess-dungeons")
	path := fs.String("data", "data/routes-data.json.gz", "dataset")
	modelFile := fs.String("model", "", "a route file or model file whose measured calibration and rules are used")
	race := fs.String("race", "Orc", "race")
	class := fs.String("class", "Shaman", "class")
	profile := fs.String("profile", automaticProfile, combatProfileHelp)
	list := fs.String("dungeons", "", "key:level pairs, comma separated: ragefire-chasm:15,wailing-caverns:19")
	out := fs.String("out", "", "write the table (JSON) here")
	steps := fs.Bool("steps", false, "print each block's steps")
	fs.Parse(args)

	var m RouteModel
	if *modelFile != "" {
		raw, err := os.ReadFile(*modelFile)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &m); err != nil {
			return err
		}
	}
	prof, err := resolveCombatProfile(*path, *class, *profile)
	if err != nil {
		return err
	}
	w, err := withModel(m, func() (*route.World, error) { return loadWorld(*path, *race, *class, "", prof, true) })
	if err != nil {
		return err
	}
	if err := m.apply(w); err != nil {
		return err
	}
	byKey := map[string]*route.Dungeon{}
	for _, d := range w.Dungeons {
		byKey[d.Key] = d
	}
	var rows []dungeonAssessment
	for _, part := range strings.Split(*list, ",") {
		kl := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(kl) != 2 {
			return fmt.Errorf("-dungeons: %q is not key:level", part)
		}
		level, err := strconv.Atoi(kl[1])
		if err != nil {
			return fmt.Errorf("-dungeons: %q", part)
		}
		row := dungeonAssessment{Dungeon: kl[0], Level: level}
		d := byKey[kl[0]]
		if d == nil {
			row.Verdict = "not a dungeon this character reaches in the data"
			rows = append(rows, row)
			continue
		}
		row.Name = d.Name
		if len(d.Quests) == 0 {
			for _, id := range d.Dungeon.Quests {
				if why, ok := w.Excluded[id]; ok {
					row.Missing = append(row.Missing, fmt.Sprintf("%d: %s", id, why))
				}
			}
			row.Verdict = "useless: no quest of it this character can do"
			rows = append(rows, row)
			report(row)
			continue
		}
		// the block's home: where most of the dungeon's quests (and those before them) are given
		var starts []dataset.Point
		for _, q := range d.Quests {
			if !q.StartInside {
				starts = append(starts, q.StartPos)
			}
			for _, p := range w.ChainBefore(q) {
				starts = append(starts, p.StartPos)
			}
		}
		at := d.Entrance
		if p, n := cluster(starts, 300); n > 0 {
			at = p
		}
		s, err := dungeonState(w, level, at, d, nil)
		if err != nil {
			return err
		}
		// a quest whose giver, turn-in or a quest before it lies more than farTravel from home and from the entrance is
		// not the block's: it is taken along when the route is there anyway (a capital's dungeon quests)
		near := func(p dataset.Point) bool {
			return w.TravelTime(s, at, p) <= farTravel || w.TravelTime(s, d.Entrance, p) <= farTravel
		}
		set := map[int]bool{}
		for _, q := range d.Quests {
			ok := (q.StartInside || near(q.StartPos)) && (q.EndInside || near(q.EndPos))
			chain := w.ChainBefore(q)
			for _, p := range chain {
				ok = ok && near(p.StartPos) && near(p.EndPos)
			}
			if !ok {
				row.Far = append(row.Far, fmt.Sprintf("%d %s (%s)", q.ID, q.Name, w.ZoneName(q.StartPos)))
				continue
			}
			row.Quests = append(row.Quests, q.ID)
			row.QuestXP += q.XP
			set[q.ID] = true
			for _, p := range chain {
				if !set[p.ID] {
					set[p.ID] = true
					row.Prereqs = append(row.Prereqs, p.ID)
				}
			}
		}
		for _, id := range d.Dungeon.Quests {
			if why, ok := w.Excluded[id]; ok && w.ByID[id] == nil {
				row.Missing = append(row.Missing, fmt.Sprintf("%d: %s", id, why))
			}
		}
		if s, err = dungeonState(w, level, at, d, set); err != nil {
			return err
		}
		row.Start = fmt.Sprintf("%s %.0f,%.0f", w.ZoneName(at), at.X, at.Y)
		if g, _, err := w.Sweep(map[int]bool{}, route.SweepOptions{Target: level + 1, From: s}); err == nil && g.Seconds > s.Time {
			row.GrindXPH = float64(g.XP-s.XP) / (g.Seconds - s.Time) * 3600
		}
		res, _, err := w.Sweep(set, route.SweepOptions{Target: level + 8, Near: 120, Out: 900, Reach: 500, StopIdle: true,
			Dungeons: true, DungeonKeys: []string{d.Key}, KeepSet: true, Below: 8, From: s})
		if err != nil {
			row.Verdict = "nothing to do: " + err.Error()
			rows = append(rows, row)
			report(row)
			continue
		}
		if *steps {
			prev := s.Time
			for _, st := range res.Steps {
				fmt.Fprintf(os.Stderr, "   %5.1f min  %-7s %5d %-40.40s %s %s\n", (st.Time-prev)/60, st.Op, st.Quest, st.Name, st.Zone, st.Via)
				prev = st.Time
			}
		}
		in := map[int]bool{}
		for _, q := range d.Quests {
			in[q.ID] = true
		}
		for _, st := range res.Steps {
			switch {
			case st.Op == "dungeon":
				row.Ran = true
			case st.Op == "turnin" && in[st.Quest]:
				row.TurnedIn++
			case st.Op == "turnin":
				row.PreDone++
			}
		}
		dt := res.Seconds - s.Time
		row.XP, row.Minutes = res.XP-s.XP, dt/60
		if dt > 0 {
			row.XPPerHour = float64(row.XP) / dt * 3600
			row.Walk = res.Travel / dt
		}
		switch {
		case !row.Ran:
			row.Verdict = "useless: the block does not get to the run"
		case row.TurnedIn == 0:
			row.Verdict = "useless: no quest of it handed in"
		case row.GrindXPH > 0 && row.XPPerHour < 0.75*row.GrindXPH:
			row.Verdict = "useless: far slower than grinding"
		case row.GrindXPH > 0 && row.XPPerHour < 0.9*row.GrindXPH:
			row.Verdict = "ok: a little slower than grinding"
		default:
			row.Verdict = "good"
		}
		rows = append(rows, row)
		report(row)
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Level < rows[j].Level })
	if *out != "" {
		b, _ := json.MarshalIndent(rows, "", " ")
		return os.WriteFile(*out, append(b, '\n'), 0o644)
	}
	return nil
}

func report(r dungeonAssessment) {
	fmt.Fprintf(os.Stderr, "%-30s %2d  %2d quests (%2d before)  %6d quest XP  run %-5v  in %2d/%2d  %5.0f min  walk %3.0f%%  %6.0f XP/h  grind %6.0f XP/h  %s\n",
		r.Name, r.Level, len(r.Quests), len(r.Prereqs), r.QuestXP, r.Ran, r.TurnedIn, len(r.Quests), r.Minutes, r.Walk*100, r.XPPerHour, r.GrindXPH, r.Verdict)
}

// cluster: the point with the most others within r yards (same map), and how many.
func cluster(ps []dataset.Point, r float64) (dataset.Point, int) {
	best, pos := 0, dataset.Point{}
	for _, a := range ps {
		n := 0
		for _, b := range ps {
			if a.Map == b.Map && math.Hypot(a.X-b.X, a.Y-b.Y) <= r {
				n++
			}
		}
		if n > best {
			best, pos = n, a
		}
	}
	return pos, best
}

// dungeonState: a character of the level at p, every quest more than two levels below done, except the block's and
// the dungeon's own (one of them done would count the dungeon as run).
func dungeonState(w *route.World, level int, p dataset.Point, d *route.Dungeon, keep map[int]bool) (*route.State, error) {
	l := route.Live{Level: level, Pos: &p, Bind: &p, Completed: []int{}, Active: []route.LiveQuest{}}
	for _, q := range w.Quests {
		if q.Level < level-2 && !keep[q.ID] && q.Dungeon != d {
			l.Completed = append(l.Completed, q.ID)
		}
	}
	s, _, err := w.StateFrom(l)
	return s, err
}
