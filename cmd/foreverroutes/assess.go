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

// assess-zones: how much a zone is worth for levelling at a level. A character of that level stands at the zone's
// hub, with every quest below a few levels done elsewhere; the zone's quests (taken and handed in there) are worked
// off until none is left to do at that level. Its XP per hour, the time and the walking are set against grinding at
// that level, all with the same model (a route file's measured calibration and rules).

type zoneAssessment struct {
	Zone      string  `json:"zone"`
	Level     int     `json:"level"`
	Hub       string  `json:"hub"`
	Quests    int     `json:"quests"` // handed in
	QuestXP   int     `json:"questXP"`
	XP        int     `json:"xp"` // quests and kills
	Minutes   float64 `json:"minutes"`
	Walk      float64 `json:"walkShare"`
	XPPerHour float64 `json:"xpPerHour"`
	GrindXPH  float64 `json:"grindXPPerHour"`
	Verdict   string  `json:"verdict"`
}

// zoneHub: the quest giver place of the zone with the most quests for that level around it.
func zoneHub(w *route.World, zone string, level int) (dataset.Point, int, bool) {
	var starts []dataset.Point
	for _, q := range w.Quests {
		if w.ZoneName(q.StartPos) == zone && q.MinLevel <= level && q.Level >= level-3 && q.Dungeon == nil {
			starts = append(starts, q.StartPos)
		}
	}
	best, pos := -1, dataset.Point{}
	for _, a := range starts {
		n := 0
		for _, b := range starts {
			if a.Map == b.Map && math.Hypot(a.X-b.X, a.Y-b.Y) <= 60 {
				n++
			}
		}
		if n > best {
			best, pos = n, a
		}
	}
	return pos, best, best > 0
}

// levelState: a character of the level at p, every quest below doneBelow done (none of the zone's).
func levelState(w *route.World, level, doneBelow int, p dataset.Point, zone string) (*route.State, error) {
	l := route.Live{Level: level, Pos: &p, Bind: &p, Completed: []int{}, Active: []route.LiveQuest{}}
	for _, q := range w.Quests {
		if q.Level < doneBelow && w.ZoneName(q.StartPos) != zone {
			l.Completed = append(l.Completed, q.ID)
		}
	}
	s, _, err := w.StateFrom(l)
	return s, err
}

func cmdAssessZones(args []string) error {
	fs := newFlags("assess-zones")
	path := fs.String("data", "data/routes-data.json.gz", "dataset")
	modelFile := fs.String("model", "", "a route file whose measured calibration and rules are used (calibration, soloQuests, mobAbove, ...)")
	race := fs.String("race", "Orc", "race")
	class := fs.String("class", "Shaman", "class")
	profile := fs.String("profile", automaticProfile, combatProfileHelp)
	zones := fs.String("zones", "", "zone:level[/level...] pairs, comma separated: Durotar:1,The Barrens:12/16/20")
	out := fs.String("out", "", "write the table (JSON) here")
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
	w, err := withModel(m, func() (*route.World, error) { return loadWorld(*path, *race, *class, "", prof, false) })
	if err != nil {
		return err
	}
	if err := m.apply(w); err != nil {
		return err
	}
	var rows []zoneAssessment
	for _, part := range strings.Split(*zones, ",") {
		zl := strings.SplitN(strings.TrimSpace(part), ":", 2)
		if len(zl) != 2 {
			return fmt.Errorf("-zones: %q is not zone:level", part)
		}
		zone := zl[0]
		for _, ls := range strings.Split(zl[1], "/") {
			level, err := strconv.Atoi(ls)
			if err != nil {
				return fmt.Errorf("-zones: %q", part)
			}
			row := zoneAssessment{Zone: zone, Level: level}
			hub, n, ok := zoneHub(w, zone, level)
			if !ok {
				row.Verdict = "no quest of this level starts here"
				rows = append(rows, row)
				continue
			}
			row.Hub = fmt.Sprintf("%.0f,%.0f (%d quests within 60 yd)", hub.X, hub.Y, n)
			s, err := levelState(w, level, level-2, hub, zone)
			if err != nil {
				return err
			}
			if g, _, err := w.Sweep(map[int]bool{}, route.SweepOptions{Target: level + 1, From: s}); err == nil && g.Seconds > s.Time {
				row.GrindXPH = float64(g.XP-s.XP) / (g.Seconds - s.Time) * 3600
			}
			set := zoneQuests(w, s, zone, level+8, nil, map[int]bool{})
			res, _, err := w.Sweep(set, route.SweepOptions{Target: level + 8, Near: 120, Out: 900, Reach: 500, Zones: true, StopIdle: true, From: s})
			if err != nil {
				row.Verdict = "nothing to do: " + err.Error()
				rows = append(rows, row)
				continue
			}
			for _, st := range res.Steps {
				if st.Op == "turnin" {
					row.Quests++
					if q := w.ByID[st.Quest]; q != nil {
						row.QuestXP += q.XP
					}
				}
			}
			dt := res.Seconds - s.Time
			row.XP, row.Minutes = res.XP-s.XP, dt/60
			if dt > 0 {
				row.XPPerHour = float64(row.XP) / dt * 3600
				row.Walk = res.Travel / dt
			}
			switch {
			case row.Quests < 5:
				row.Verdict = "useless: fewer than 5 quests"
			case row.GrindXPH > 0 && row.XPPerHour < 0.75*row.GrindXPH:
				row.Verdict = "useless: far slower than grinding"
			case row.Walk > 0.5:
				row.Verdict = "weak: more than half the time walking"
			case row.GrindXPH > 0 && row.XPPerHour < 0.9*row.GrindXPH:
				// the grinding model counts dense camps and no deaths: questing a little under it is still worth it
				// (gear, money, quests that open later ones)
				row.Verdict = "ok: a little slower than grinding"
			default:
				row.Verdict = "good"
			}
			rows = append(rows, row)
			fmt.Fprintf(os.Stderr, "%-22s %2d  %3d quests  %6d quest XP  %5.0f min  walk %3.0f%%  %6.0f XP/h  grind %6.0f XP/h  %s\n",
				zone, level, row.Quests, row.QuestXP, row.Minutes, row.Walk*100, row.XPPerHour, row.GrindXPH, row.Verdict)
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Level < rows[j].Level })
	if *out != "" {
		b, _ := json.MarshalIndent(rows, "", " ")
		return os.WriteFile(*out, append(b, '\n'), 0o644)
	}
	return nil
}
