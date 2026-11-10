package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/blocks"
	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/fallback"
	"github.com/Battlefeld/ForeverRoutes/route"
)

// blocks: a route built from area blocks (route.Sweep) against the RestedXP chapters of the same character, both
// replayed with the same model to the same level.

// pacedProfile scales the fight and rest times of a profile: a slower or faster player.
type pacedProfile struct {
	route.KillProfile
	pace float64
}

func (p pacedProfile) KillSeconds(c route.Character, level int, mob *dataset.NPC) (float64, float64) {
	f, r := p.KillProfile.KillSeconds(c, level, mob)
	return f * p.pace, r * p.pace
}

func (p pacedProfile) Name() string {
	return fmt.Sprintf("%s (fights x%.1f)", p.KillProfile.Name(), p.pace)
}

// blocksWorld opens the world as loadWorld does, with the dataset changed first (the chain rules) and the fights
// scaled by pace.
func blocksWorld(path, race, class, faction, profile string, pace float64, edit func(*dataset.Dataset)) (*route.World, error) {
	profile, err := resolveCombatProfile(path, class, profile)
	if err != nil {
		return nil, err
	}
	d, err := dataset.Load(path)
	if err != nil {
		return nil, err
	}
	if edit != nil {
		edit(d)
	}
	c := route.Character{Race: race, Class: class, Faction: faction}
	var opt route.Options
	if wk := filepath.Join(filepath.Dir(path), "walks.json.gz"); fileExists(wk) {
		if opt.Walks, err = dataset.LoadWalks(wk); err != nil {
			return nil, err
		}
	}
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
	w, err := route.NewWorld(d, c, opt)
	if err != nil {
		return nil, err
	}
	if pace != 1 {
		w.Opt.Profile = pacedProfile{w.Opt.Profile, pace}
	}
	return w, nil
}

// blocksRun is one route of the comparison.
type blocksRun struct {
	Name     string             `json:"name"`
	Seconds  float64            `json:"seconds"`
	Travel   float64            `json:"travel"`
	Combat   float64            `json:"combat"`
	Grind    float64            `json:"grindSeconds"`
	Quests   int                `json:"quests"`
	XPPerHr  float64            `json:"xpPerHour"`
	Level    int                `json:"level"`
	Near     float64            `json:"near,omitempty"`
	Out      float64            `json:"out,omitempty"`
	Reach    float64            `json:"reach,omitempty"`
	Skipped  map[string]int     `json:"skipped,omitempty"`
	Blocks   []route.SweepBlock `json:"blocks,omitempty"`
	QuestIDs []int              `json:"questIds,omitempty"`
}

func grindSeconds(res *route.Result) float64 {
	t, prev := 0.0, 0.0
	for _, s := range res.Steps {
		if s.Op == "grind" {
			t += s.Time - prev
		}
		prev = s.Time
	}
	return t
}

func runOf(name string, res *route.Result) blocksRun {
	r := blocksRun{Name: name, Seconds: res.Seconds, Travel: res.Travel, Combat: res.Combat, Grind: grindSeconds(res), Quests: res.Quests, Level: res.Level}
	if res.Seconds > 0 {
		r.XPPerHr = float64(res.XP) / res.Seconds * 3600
	}
	return r
}

// bestSweep tries a few settings of the sweep and keeps the fastest route.
func bestSweep(w *route.World, set map[int]bool, target int, prefer map[int][]int) (*route.Result, []route.SweepBlock, route.SweepOptions, error) {
	var best *route.Result
	var bb []route.SweepBlock
	var bo route.SweepOptions
	var last error
	for _, near := range []float64{45, 75, 120, 180} {
		for _, out := range []float64{0, 150, 300, 600, 1200} {
			for _, reach := range []float64{300, 500, 800} {
				o := route.SweepOptions{Target: target, Near: near, Reach: reach, Out: out, Prefer: prefer}
				res, bl, err := w.Sweep(set, o)
				if err != nil {
					last = err
					continue
				}
				if res.Level >= target && (best == nil || res.Seconds < best.Seconds) {
					best, bb, bo = res, bl, o
				}
			}
		}
	}
	if best == nil {
		if last == nil {
			last = fmt.Errorf("no sweep reaches level %d", target)
		}
		return nil, nil, bo, last
	}
	return best, bb, bo, nil
}

// prune leaves out, one at a time, the quest whose absence makes the route fastest, until none does: what is left
// is worth its time in this route.
func prune(w *route.World, set map[int]bool, target int, prefer map[int][]int) (map[int]bool, *route.Result, []route.SweepBlock, route.SweepOptions, error) {
	cur := map[int]bool{}
	for id := range set {
		cur[id] = true
	}
	res, bl, o, err := bestSweep(w, cur, target, prefer)
	if err != nil {
		return nil, nil, nil, o, err
	}
	for {
		ids := make([]int, 0, len(cur))
		for id := range cur {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		drop, gain := 0, 1.0 // a quest must cost more than a second to be left out
		var dres *route.Result
		var dbl []route.SweepBlock
		used := acceptedIn(res)
		for _, id := range ids {
			if !used[id] {
				continue // the route never took it: leaving it out changes nothing
			}
			delete(cur, id)
			r, b, err := w.Sweep(cur, o)
			cur[id] = true
			if err != nil || r.Level < target {
				continue
			}
			if g := res.Seconds - r.Seconds; g > gain {
				drop, gain, dres, dbl = id, g, r, b
			}
		}
		if drop == 0 {
			return acceptedIn(res), res, bl, o, nil
		}
		delete(cur, drop)
		res, bl = dres, dbl
	}
}

func cmdBlocks(args []string) error {
	fs := newFlags("blocks")
	path := fs.String("data", "data/routes-data.json.gz", "dataset")
	forever := fs.String("forever", "data/audits/foreverdb-full.json.gz", "ForeverDB's full quest list (tools/foreverdb_full.py); empty = chains as the dataset has them")
	fallbackFile := fs.String("fallback", "data/fallbacks/rxp.json.gz", "RestedXP chapters to compare with")
	race := fs.String("race", "Orc", "race")
	class := fs.String("class", "Shaman", "class")
	faction := fs.String("faction", "", "A or H (only for Skyborne)")
	profile := fs.String("profile", automaticProfile, combatProfileHelp)
	target := fs.Int("target", 12, "target level")
	zones := fs.String("zones", "", "comma separated zones of the own selection (default: the zones the guide's quests start in)")
	out := fs.String("out", "", "write the comparison (JSON) here")
	fs.Parse(args)

	var list *blocks.List
	var changes []blocks.Change
	var prefer map[int][]int
	edit := func(d *dataset.Dataset) {}
	if *forever != "" {
		var err error
		if list, err = blocks.Load(*forever); err != nil {
			return err
		}
		edit = func(d *dataset.Dataset) { changes, prefer = list.Apply(d) }
	}
	cat, err := fallback.Load(*fallbackFile)
	if err != nil {
		return err
	}
	char := route.Character{Race: *race, Class: *class, Faction: *faction}
	guide, err := cat.For(char)
	if err != nil {
		return err
	}

	type report struct {
		Character route.Character   `json:"character"`
		Target    int               `json:"target"`
		ForeverDB string            `json:"foreverdbFetched,omitempty"`
		Chains    int               `json:"prerequisitesRewritten"`
		Changes   []blocks.Change   `json:"prerequisiteChanges,omitempty"`
		Missed    []route.GuideMiss `json:"guideMissed,omitempty"`
		Runs      []blocksRun       `json:"runs"`
		Paces     []blocksRun       `json:"paces"`
		Confirmed map[string]int    `json:"confirmedQuests"`
		Unseen    []int             `json:"questsNotSeenInForever,omitempty"`
		OnlyGuide []int             `json:"onlyGuide,omitempty"`
		OnlyOwn   []int             `json:"onlyOwn,omitempty"`
	}
	rep := report{Character: char, Target: *target, Confirmed: map[string]int{}}
	if list != nil {
		rep.ForeverDB = list.Fetched
	}

	// the guide with the dataset's own chains, then everything with the corrected ones
	w0, err := blocksWorld(*path, *race, *class, *faction, *profile, 1, nil)
	if err != nil {
		return err
	}
	g0, err := w0.FollowGuide(guide.Events, *target)
	if err != nil {
		return err
	}
	r := runOf("RestedXP order, chains as in the dataset", g0.Result)
	r.Grind, r.Skipped = g0.Grind, g0.Skipped
	rep.Runs = append(rep.Runs, r)

	w, err := blocksWorld(*path, *race, *class, *faction, *profile, 1, edit)
	if err != nil {
		return err
	}
	rep.Chains, rep.Changes = len(changes), changes
	g, err := w.FollowGuide(guide.Events, *target)
	if err != nil {
		return err
	}
	r = runOf("RestedXP order", g.Result)
	r.Grind, r.Skipped = g.Grind, g.Skipped
	guideSet := map[int]bool{}
	guideZones := map[string]bool{}
	for _, s := range g.Steps {
		if s.Op == "accept" {
			guideSet[s.Quest] = true
			guideZones[w.ZoneName(s.At)] = true
		}
	}
	r.QuestIDs = sortedIDs(guideSet)
	rep.Runs = append(rep.Runs, r)
	rep.Missed = g.Missed

	// the same quests, ordered by areas
	res, bl, o, err := bestSweep(w, guideSet, *target, prefer)
	if err != nil {
		return err
	}
	r = runOf("blocks, RestedXP's quests", res)
	r.Near, r.Out, r.Reach, r.Blocks, r.QuestIDs = o.Near, o.Out, o.Reach, bl, sortedIDs(acceptedIn(res))
	rep.Runs = append(rep.Runs, r)

	// an own selection: every quest of those zones the character can take below the target, less what is not worth it
	if *zones != "" {
		guideZones = map[string]bool{}
		for _, z := range strings.Split(*zones, ",") {
			guideZones[strings.TrimSpace(z)] = true
		}
	}
	own := map[int]bool{}
	for _, q := range w.Quests {
		if q.MinLevel < *target && q.Dungeon == nil && q.Type == "" && !q.Repeatable && len(q.StartSources) == 0 && guideZones[w.ZoneName(q.StartPos)] {
			own[q.ID] = true
		}
	}
	if all, _, _, err := bestSweep(w, own, *target, prefer); err == nil {
		r = runOf("blocks, every quest of the zones", all)
		r.QuestIDs = sortedIDs(acceptedIn(all))
		rep.Runs = append(rep.Runs, r)
	}
	kept, pres, pbl, po, err := prune(w, own, *target, prefer)
	if err == nil {
		r = runOf("blocks, own selection", pres)
		r.Near, r.Out, r.Reach, r.Blocks, r.QuestIDs = po.Near, po.Out, po.Reach, pbl, sortedIDs(kept)
		rep.Runs = append(rep.Runs, r)
		for id := range guideSet {
			if !kept[id] {
				rep.OnlyGuide = append(rep.OnlyGuide, id)
			}
		}
		for id := range kept {
			if !guideSet[id] {
				rep.OnlyOwn = append(rep.OnlyOwn, id)
			}
		}
		sort.Ints(rep.OnlyGuide)
		sort.Ints(rep.OnlyOwn)
	}

	// how well the quests used are confirmed in Forever
	if list != nil {
		used := map[int]bool{}
		for id := range guideSet {
			used[id] = true
		}
		for id := range kept {
			used[id] = true
		}
		for id := range used {
			c := list.Quests[id].Confirmed()
			rep.Confirmed[c]++
			if c != blocks.Seen {
				rep.Unseen = append(rep.Unseen, id)
			}
		}
		sort.Ints(rep.Unseen)
	}

	// does the comparison depend on how fast the player fights?
	for _, pace := range []float64{0.5, 2} {
		wp, err := blocksWorld(*path, *race, *class, *faction, *profile, pace, edit)
		if err != nil {
			return err
		}
		gp, err := wp.FollowGuide(guide.Events, *target)
		if err != nil {
			return err
		}
		r = runOf(fmt.Sprintf("RestedXP order, fights x%.1f", pace), gp.Result)
		r.Grind = gp.Grind
		rep.Paces = append(rep.Paces, r)
		if sp, _, _, err := bestSweep(wp, guideSet, *target, prefer); err == nil {
			rep.Paces = append(rep.Paces, runOf(fmt.Sprintf("blocks, RestedXP's quests, fights x%.1f", pace), sp))
		}
		if len(kept) > 0 {
			if sp, _, _, err := bestSweep(wp, kept, *target, prefer); err == nil {
				rep.Paces = append(rep.Paces, runOf(fmt.Sprintf("blocks, own selection, fights x%.1f", pace), sp))
			}
		}
	}

	fmt.Printf("%s %s to level %d, profile %s; %d prerequisites rewritten\n", *race, *class, *target, w.Opt.Profile.Name(), rep.Chains)
	row := func(r blocksRun) {
		fmt.Printf("  %-44s %6.2f h  travel %5.2f  combat %5.2f  grind %5.2f  quests %3d  %6.0f XP/h  level %d %v\n",
			r.Name, r.Seconds/3600, r.Travel/3600, r.Combat/3600, r.Grind/3600, r.Quests, r.XPPerHr, r.Level, r.Skipped)
	}
	for _, r := range rep.Runs {
		row(r)
	}
	for _, r := range rep.Paces {
		row(r)
	}
	name := func(id int) string {
		if q := w.ByID[id]; q != nil {
			return fmt.Sprintf("%d %s", id, q.Name)
		}
		return fmt.Sprint(id)
	}
	for _, r := range rep.Runs {
		if len(r.Blocks) == 0 {
			continue
		}
		fmt.Printf("\n%s (near %.0f s, out %.0f s, reach %.0f yd): %d blocks\n", r.Name, r.Near, r.Out, r.Reach, len(r.Blocks))
		for i, b := range r.Blocks {
			fmt.Printf("  %2d %-5s %-12s %5.1f-%5.1f min  lvl %2d  +%5d XP  take %d work %d hand in %d\n", i+1, b.Kind, b.Zone, b.Start/60, b.End/60, b.Level, b.XP, len(b.Accepted), len(b.Worked), len(b.TurnedIn))
		}
	}
	fmt.Printf("\nguide events the dataset cannot follow (%d):\n", len(rep.Missed))
	for _, m := range rep.Missed {
		fmt.Printf("   %s %s: %s %s\n", m.Op, name(m.Quest), m.Reason, m.Detail)
	}
	fmt.Printf("\nonly in the guide (%d):\n", len(rep.OnlyGuide))
	for _, id := range rep.OnlyGuide {
		fmt.Println("  ", name(id))
	}
	fmt.Printf("only in the own selection (%d):\n", len(rep.OnlyOwn))
	for _, id := range rep.OnlyOwn {
		fmt.Println("  ", name(id))
	}
	fmt.Printf("confirmed in Forever: %v; not seen by players: %d\n", rep.Confirmed, len(rep.Unseen))
	if *out != "" {
		b, _ := json.MarshalIndent(rep, "", " ")
		return os.WriteFile(*out, append(b, '\n'), 0o644)
	}
	return nil
}

// acceptedIn: the quests a replayed route takes.
func acceptedIn(res *route.Result) map[int]bool {
	out := map[int]bool{}
	for _, s := range res.Steps {
		if s.Op == "accept" {
			out[s.Quest] = true
		}
	}
	return out
}

func sortedIDs(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}
