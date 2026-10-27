package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/internal/sources/guides"
	"github.com/Battlefeld/ForeverRoutes/route"
)

// check-routes replays every route file and reports what a player would trip over: steps that take long (farming, a
// rare, a long walk), quests no guide does, steps resting on an assumed drop chance, deathskips paid with sickness,
// and events the replay had to skip.
func cmdCheckRoutes(args []string) error {
	fs := newFlags("check-routes")
	path := fs.String("data", "data/routes-data.json.gz", "dataset")
	dir := fs.String("routes", "data/routes", "route files")
	profiles := fs.String("profiles", "data/profiles", "kill profiles (<class>.json)")
	rxp := fs.String("rxp", "", "RestedXP Guides folder (quests a guide does)")
	polestar := fs.String("polestar", "", "Polestar guides/forever folder")
	long := fs.Float64("long", 900, "a step longer than this many seconds is reported")
	out := fs.String("out", "", "write the report (Markdown) here")
	strict := fs.Bool("strict", true, "fail when a route is malformed, incomplete, skips events or misses required class goals")
	fs.Parse(args)

	guided := map[int]bool{}
	accept := regexp.MustCompile(`(?m)^\s*\.?accept\s+(\d+)`)
	for _, src := range []struct{ dir, suffix string }{{*rxp, ".lua"}, {*polestar, ".guide"}} {
		if src.dir == "" {
			continue
		}
		files, err := guides.ReadDir(src.dir, src.suffix)
		if err != nil {
			return err
		}
		for _, body := range files {
			for _, m := range accept.FindAllStringSubmatch(body, -1) {
				var id int
				fmt.Sscan(m[1], &id)
				guided[id] = true
			}
		}
	}
	files, err := filepath.Glob(filepath.Join(*dir, "*.json"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("no route files in %s", *dir)
	}
	var problems []string
	var b strings.Builder
	fmt.Fprintf(&b, "# Route check\n\n%d routes, steps over %.0f min reported", len(files), *long/60)
	if len(guided) > 0 {
		fmt.Fprintf(&b, ", %d quests known from the guides", len(guided))
	}
	b.WriteString(".\n\n| Route | Hours | Long steps | Not in a guide | Assumed drops | Deathskips (sick) | Skipped | Options (not finished) |\n|---|---|---|---|---|---|---|---|\n")
	var details strings.Builder
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		var r struct {
			RouteModel
			Character route.Character    `json:"character"`
			Target    int                `json:"target"`
			Events    []route.GuideEvent `json:"events"`
			Options   []route.ClassCost  `json:"classQuests"`
			Wand      float64            `json:"wandDetour"`
		}
		if err := json.Unmarshal(raw, &r); err != nil || len(r.Events) == 0 {
			return fmt.Errorf("%s: malformed or empty route", f)
		}
		prof := filepath.Join(*profiles, strings.ToLower(r.Character.Class)+".json")
		if !fileExists(prof) {
			prof = ""
		}
		w, err := loadWorld(*path, r.Character.Race, r.Character.Class, r.Character.Faction, prof, hasDungeons(r.Events))
		if err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
		if err := r.RouteModel.apply(w); err != nil {
			return err
		}
		w.SetWandDetour(r.Wand)
		res, err := w.FollowGuide(r.Events, r.Target)
		if err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
		name := strings.TrimSuffix(filepath.Base(f), ".json")
		var longSteps, unguided, sick []string
		seen := map[int]bool{}
		prev := 0.0
		for _, st := range res.Steps {
			if d := st.Time - prev; d > *long {
				longSteps = append(longSteps, fmt.Sprintf("L%d %s %s (%.0f min)", st.Level, st.Op, st.Name, d/60))
			}
			prev = st.Time
			if st.Op == "accept" && len(guided) > 0 && !guided[st.Quest] && !seen[st.Quest] {
				unguided = append(unguided, fmt.Sprintf("%d %s", st.Quest, st.Name))
			}
			seen[st.Quest] = true
			if strings.HasPrefix(st.Via, "deathskip") && st.Level > 10 {
				sick = append(sick, fmt.Sprintf("L%d %s", st.Level, strings.TrimPrefix(st.Via, "deathskip: ")))
			}
		}
		// every option switched on: its last quest handed in before the target
		var options, unfinished []string
		for _, c := range r.Options {
			if len(c.Events) == 0 {
				continue
			}
			options = append(options, c.Name)
			with, err := w.FollowGuide(c.Events, r.Target)
			handed := false
			if err == nil {
				for _, st := range with.Steps {
					handed = handed || (st.Op == "turnin" && st.Quest == c.Quest)
				}
			}
			if err != nil || !handed {
				unfinished = append(unfinished, fmt.Sprintf("%d %s", c.Quest, c.Name))
			}
		}
		skipped := 0
		for _, n := range res.Skipped {
			skipped += n
		}
		if res.Level < r.Target || skipped > 0 {
			problems = append(problems, name+": incomplete or skipped events")
		}
		done:=map[int]bool{};for _,step:=range res.Steps{if step.Op=="turnin"{done[step.Quest]=true}}
		for _, c := range w.ClassChains(r.Target) {
			if c.Tier == "required" && !done[c.Goal.ID] {
				problems = append(problems, fmt.Sprintf("%s: missing required quest %d", name, c.Goal.ID))
			}
		}
		if len(unfinished) > 0 {
			problems = append(problems, name+": unfinished option routes")
		}
		fmt.Fprintf(&b, "| %s | %.1f | %d | %d | %d | %d (%d) | %d | %d (%d) |\n", name, res.Seconds/3600, len(longSteps), len(unguided), res.Assumed, res.Deaths, len(sick), skipped, len(options), len(unfinished))
		if len(longSteps)+len(unguided)+len(sick)+len(res.Missed)+len(unfinished) > 0 {
			fmt.Fprintf(&details, "\n## %s\n", name)
			for _, l := range [][2]any{{"Long steps", longSteps}, {"Not in a guide", unguided}, {"Deathskips with sickness", sick}, {"Options not finished", unfinished}} {
				if items := l[1].([]string); len(items) > 0 {
					fmt.Fprintf(&details, "\n%s:\n", l[0])
					for _, it := range items {
						fmt.Fprintf(&details, "- %s\n", it)
					}
				}
			}
			if len(res.Missed) > 0 {
				details.WriteString("\nSkipped:\n")
				for _, m := range res.Missed {
					fmt.Fprintf(&details, "- %s %d: %s %s\n", m.Op, m.Quest, m.Reason, m.Detail)
				}
			}
		}
		fmt.Fprintf(os.Stderr, "%s: %.1f h\n", name, res.Seconds/3600)
	}
	b.WriteString(details.String())
	if *out == "" {
		fmt.Print(b.String())
	} else if err := os.WriteFile(*out, []byte(b.String()), 0o644); err != nil {
		return err
	}
	if *strict && len(problems) > 0 {
		return fmt.Errorf("route validation failed: %s", strings.Join(problems, "; "))
	}
	return nil
}
