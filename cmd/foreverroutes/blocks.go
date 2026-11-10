package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
func blocksWorld(path, race, class, faction, profile string, pace float64, dungeons bool, edit func(*dataset.Dataset)) (*route.World, error) {
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
	opt.Dungeons = dungeons
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

// dungeonQuest is one quest of a dungeon block.
type dungeonQuest struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Shareable *bool   `json:"shareable,omitempty"` // Classic data; nil = unknown (a Forever quest)
	Before    []int   `json:"before,omitempty"`    // quests that lead to it and must be done first
	Inside    bool    `json:"givenInside,omitempty"`
	Taken     float64 `json:"takenMinute,omitempty"`
	HandedIn  float64 `json:"handedInMinute,omitempty"`
	State     string  `json:"state"` // done, taken (not handed in by the end), not taken
}

// dungeonBlock is one run of a route with what belongs to it.
type dungeonBlock struct {
	Key    string         `json:"dungeon"`
	Name   string         `json:"name"`
	Level  int            `json:"level"`
	Start  float64        `json:"startMinute"`
	End    float64        `json:"endMinute"`
	Quests []dungeonQuest `json:"quests"`
}

// dungeonBlocks reads the runs of a replayed route with their quests: taken before, what leads to them, handed in.
func dungeonBlocks(w *route.World, res *route.Result, shareable map[int]bool) []dungeonBlock {
	taken, handed := map[int]float64{}, map[int]float64{}
	for _, s := range res.Steps {
		switch s.Op {
		case "accept":
			taken[s.Quest] = s.Time / 60
		case "turnin":
			handed[s.Quest] = s.Time / 60
		}
	}
	var out []dungeonBlock
	prev := 0.0
	for _, s := range res.Steps {
		if s.Op == "dungeon" {
			for _, d := range w.Dungeons {
				if d.Key != s.Dungeon {
					continue
				}
				b := dungeonBlock{Key: d.Key, Name: d.Name, Level: s.Level, Start: prev / 60, End: s.Time / 60}
				for _, q := range d.Quests {
					dq := dungeonQuest{ID: q.ID, Name: q.Name, Inside: q.StartInside, Taken: taken[q.ID], HandedIn: handed[q.ID], State: "not taken"}
					if shareable != nil && q.ID < 90000 { // Forever's own quests are not in the Classic data
						v := shareable[q.ID]
						dq.Shareable = &v
					}
					for _, p := range w.ChainBefore(q) {
						dq.Before = append(dq.Before, p.ID)
					}
					switch {
					case q.EndInside && (taken[q.ID] > 0 || q.StartInside):
						dq.State = "done"
					case handed[q.ID] > 0:
						dq.State = "done"
					case taken[q.ID] > 0:
						dq.State = "taken"
					}
					b.Quests = append(b.Quests, dq)
				}
				out = append(out, b)
			}
		}
		prev = s.Time
	}
	return out
}

// blocksFile is a route as named blocks in order: edit the order (or a block's quests) and play it with -from-blocks.
type blocksFile struct {
	Schema    string             `json:"schema"`
	Key       string             `json:"key"`
	Character route.Character    `json:"character"`
	Target    int                `json:"target"`
	Hours     float64            `json:"hours"`
	Near      float64            `json:"near"`
	Out       float64            `json:"out"`
	Reach     float64            `json:"reach"`
	Blocks    []route.SweepBlock `json:"blocks"`
	Names     map[string]string  `json:"questNames"`
	Shareable map[string]bool    `json:"shareable"` // Classic data; a quest without an entry is unknown
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
	Look     float64            `json:"look,omitempty"`
	Zones    bool               `json:"zones,omitempty"`
	MayGrind bool               `json:"grindCandidate,omitempty"`
	Reach    float64            `json:"reach,omitempty"`
	Skipped  map[string]int     `json:"skipped,omitempty"`
	Blocks   []route.SweepBlock `json:"blocks,omitempty"`
	Dungeons []dungeonBlock     `json:"dungeonBlocks,omitempty"`
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

// bestSweep tries a few settings of the sweep and keeps the fastest route: first the rules alone, then the best of
// them with moves between places played ahead.
// quickSweep: only a few settings and no look ahead (long routes: -quick).
var quickSweep bool

// startLive: the character the block routes begin as (-start-level, -start-at); nil = a new character.
var startLive *route.Live

func bestSweep(w *route.World, set map[int]bool, target int, prefer map[int][]int) (*route.Result, []route.SweepBlock, route.SweepOptions, error) {
	var best *route.Result
	var bb []route.SweepBlock
	var bo route.SweepOptions
	var last error
	var from *route.State
	if startLive != nil {
		var err error
		if from, _, err = w.StateFrom(*startLive); err != nil {
			return nil, nil, bo, err
		}
	}
	try := func(o route.SweepOptions) {
		o.From = from
		res, bl, err := w.Sweep(set, o)
		if err != nil {
			last = err
			return
		}
		if res.Level >= target && (best == nil || res.Seconds < best.Seconds) {
			best, bb, bo = res, bl, o
		}
	}
	nears, outs, reaches := []float64{45, 75, 120, 180}, []float64{0, 150, 300, 600, 1200}, []float64{300, 500, 800}
	if quickSweep {
		nears, outs, reaches = []float64{75, 180}, []float64{0, 300}, []float64{500}
	}
	if startLive != nil {
		// a stretch to play: the log is worked off before the way back, whatever the model's clock says
		nears, outs, reaches = []float64{120}, []float64{900}, []float64{500}
	}
	for _, near := range nears {
		for _, out := range outs {
			for _, reach := range reaches {
				for _, zones := range []bool{false, true} {
					try(route.SweepOptions{Target: target, Near: near, Reach: reach, Out: out, Prefer: prefer, Dungeons: w.Opt.Dungeons, Zones: zones})
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
	base := bo
	if startLive != nil {
		return best, bb, bo, nil
	}
	if quickSweep {
		o := base
		o.Look, o.Grind = 900, true
		try(o)
		o.Look = 1800
		try(o)
		return best, bb, bo, nil
	}
	for _, look := range []float64{900, 1800, 3600} {
		for _, near := range []float64{base.Near, 75} {
			o := base
			o.Look, o.Near = look, near
			try(o)
			o.Out = 0
			try(o)
		}
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
	if o.Look > 0 {
		fast := o
		fast.Look = 0
		if r, b, err := w.Sweep(cur, fast); err == nil && r.Level >= target {
			res, bl = r, b
		}
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
			fast := o
			fast.Look = 0 // the selection is made with the rules alone: a look ahead for every quest left out takes too long
			r, b, err := w.Sweep(cur, fast)
			cur[id] = true
			if err != nil || r.Level < target {
				continue
			}
			if g := res.Seconds - r.Seconds; g > gain {
				drop, gain, dres, dbl = id, g, r, b
			}
		}
		if drop == 0 {
			// the selection stands: the best route through it
			if r, b, bo, err := bestSweep(w, cur, target, prefer); err == nil && r.Seconds < res.Seconds {
				res, bl, o = r, b, bo
			}
			return acceptedIn(res), res, bl, o, nil
		}
		delete(cur, drop)
		res, bl = dres, dbl
	}
}

// pruneBatch is prune for long routes: one pass measures what leaving out each quest alone gains, then the quests
// are left out in that order, each kept out only when the route really gets faster; a few passes.
func pruneBatch(w *route.World, set map[int]bool, target int, o route.SweepOptions, passes int) (map[int]bool, *route.Result, []route.SweepBlock) {
	cur := map[int]bool{}
	for id := range set {
		cur[id] = true
	}
	res, bl, err := w.Sweep(cur, o)
	if err != nil {
		return nil, nil, nil
	}
	for pass := 0; pass < passes; pass++ {
		type gain struct {
			id int
			g  float64
		}
		var gains []gain
		for _, id := range sortedIDs(acceptedIn(res)) {
			if q := w.ByID[id]; q == nil || q.Dungeon != nil || q.Classes != 0 {
				continue // dungeon and class quests stay
			}
			delete(cur, id)
			if r, _, err := w.Sweep(cur, o); err == nil && r.Level >= target && res.Seconds-r.Seconds > 30 {
				gains = append(gains, gain{id, res.Seconds - r.Seconds})
			}
			cur[id] = true
		}
		sort.Slice(gains, func(i, j int) bool { return gains[i].g > gains[j].g })
		changed := false
		for _, c := range gains {
			delete(cur, c.id)
			if r, b, err := w.Sweep(cur, o); err == nil && r.Level >= target && r.Seconds < res.Seconds-1 {
				res, bl, changed = r, b, true
				continue
			}
			cur[c.id] = true
		}
		fmt.Fprintf(os.Stderr, "leaving out, pass %d: %d candidates, %.2f h\n", pass+1, len(gains), res.Seconds/3600)
		if !changed {
			break
		}
	}
	return acceptedIn(res), res, bl
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
	startLevel := fs.Int("start-level", 0, "begin the block routes as a character of this level (with -start-at) instead of a new one")
	startAt := fs.Int("start-at", 0, "with -start-level: the quest whose giver the character stands at")
	startActive := fs.String("start-active", "", "with -start-level: quests in the log, comma separated")
	skip := fs.String("skip", "", "quests the block routes leave out, comma separated")
	noProf := fs.Bool("no-professions", false, "the block routes learn no professions on the way")
	hearthSave := fs.Float64("hearth-save", 0, "the hearthstone only when it saves this many seconds over walking (0: any saving)")
	startDone := fs.String("start-done", "", "with -start-level: quests handed in, comma separated")
	quick := fs.Bool("quick", false, "long routes: few sweep settings, no look ahead, no leaving out of single quests")
	against := fs.String("against", "", "a route file to replay beside the others (a shipped or extended route)")
	paces := fs.Bool("paces", true, "also compare with half and double fight times")
	routeOut := fs.String("route-out", "", "write the fastest block route as a route file (for export-addon -forever) here")
	blocksOut := fs.String("blocks-out", "", "write the blocks of that route (named, in order) here")
	fromBlocks := fs.String("from-blocks", "", "a blocks file: play its blocks in their order instead of searching")
	polish := fs.Duration("polish", 0, "improve each route afterwards with the existing optimizer's local edits for this long (0 = off)")
	dungeons := fs.Bool("dungeons", false, "run the dungeons as blocks (the guide stays the ordinary solo guide)")
	flags := fs.String("flags", "data/audits/quest-flags-classic.json", "quests a player can share (Classic data); empty = unknown")
	zones := fs.String("zones", "", "comma separated zones of the own selection (default: the zones the guide's quests start in)")
	out := fs.String("out", "", "write the comparison (JSON) here")
	fs.Parse(args)

	quickSweep = *quick
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
	var shareable map[int]bool
	if *flags != "" && fileExists(*flags) {
		var err error
		if shareable, err = blocks.Shareable(*flags); err != nil {
			return err
		}
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
	w0, err := blocksWorld(*path, *race, *class, *faction, *profile, 1, false, nil)
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

	wg, err := blocksWorld(*path, *race, *class, *faction, *profile, 1, false, edit)
	if err != nil {
		return err
	}
	w := wg
	skipped := map[int]bool{}
	for _, f := range strings.Split(*skip, ",") {
		if id, err := strconv.Atoi(strings.TrimSpace(f)); err == nil {
			skipped[id] = true
		}
	}
	if *dungeons {
		if w, err = blocksWorld(*path, *race, *class, *faction, *profile, 1, true, edit); err != nil {
			return err
		}
	}
	rep.Chains, rep.Changes = len(changes), changes
	var model RouteModel
	if *noProf {
		p := w.Preparations()
		p.Camping, p.ForceProfessions = false, false
		if err := w.SetPreparations(p); err != nil {
			return err
		}
		model.Preparations = &p
	}
	if *hearthSave > 0 {
		w.Opt.HearthMinSave, model.HearthMinSave = *hearthSave, *hearthSave
	}
	g, err := wg.FollowGuide(guide.Events, *target)
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
	polished := func(name string, world *route.World, events []route.GuideEvent) {
		if *polish <= 0 {
			return
		}
		if _, pg, err := world.Improve(events, *target, *polish); err == nil {
			pr := runOf(name+", improved", pg.Result)
			pr.Grind, pr.QuestIDs = pg.Grind, sortedIDs(acceptedIn(pg.Result))
			rep.Runs = append(rep.Runs, pr)
		}
	}
	polished("RestedXP order", wg, guide.Events)
	if *against != "" {
		raw, err := os.ReadFile(*against)
		if err != nil {
			return err
		}
		var rf struct {
			RouteModel
			Events []route.GuideEvent `json:"events"`
			Wand   float64            `json:"wandDetour"`
		}
		if err := json.Unmarshal(raw, &rf); err != nil || len(rf.Events) == 0 {
			return fmt.Errorf("%s: malformed or empty route", *against)
		}
		wa, err := blocksWorld(*path, *race, *class, *faction, *profile, 1, hasDungeons(rf.Events), edit)
		if err != nil {
			return err
		}
		if err := rf.RouteModel.apply(wa); err != nil {
			return err
		}
		wa.SetWandDetour(rf.Wand)
		ga, err := wa.FollowGuide(rf.Events, *target)
		if err != nil {
			return err
		}
		ar := runOf("route "+filepath.Base(*against), ga.Result)
		ar.Grind, ar.Skipped = ga.Grind, ga.Skipped
		rep.Runs = append(rep.Runs, ar)
	}

	if *startLevel > 1 {
		q := w.ByID[*startAt]
		if q == nil {
			return fmt.Errorf("-start-at: quest %d is not a quest of this character", *startAt)
		}
		pos := q.StartPos
		startLive = &route.Live{Level: *startLevel, Pos: &pos, Bind: &pos, Completed: []int{}, Active: []route.LiveQuest{}}
		for _, f := range strings.Split(*startDone, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(f)); err == nil {
				startLive.Completed = append(startLive.Completed, id)
			}
		}
		for _, f := range strings.Split(*startActive, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(f)); err == nil {
				startLive.Active = append(startLive.Active, route.LiveQuest{ID: id})
			}
		}
		defer func() { startLive = nil }()
	}
	// the same quests, ordered by areas
	res, bl, o, err := bestSweep(w, guideSet, *target, prefer)
	if err != nil {
		return err
	}
	r = runOf("blocks, RestedXP's quests", res)
	r.Near, r.Out, r.Look, r.Reach, r.Blocks, r.QuestIDs = o.Near, o.Out, o.Look, o.Reach, bl, sortedIDs(acceptedIn(res))
	r.Dungeons = dungeonBlocks(w, res, shareable)
	rep.Runs = append(rep.Runs, r)
	polished("blocks, RestedXP's quests", w, route.EventsOf(res.Steps))
	bestRes, bestBlocks, bestName := res, bl, r.Name

	// an own selection: every quest of those zones the character can take below the target, less what is not worth it
	if *zones != "" {
		guideZones = map[string]bool{}
		for _, z := range strings.Split(*zones, ",") {
			guideZones[strings.TrimSpace(z)] = true
		}
	}
	own := map[int]bool{}
	for _, q := range w.Quests {
		if q.MinLevel < *target && q.Dungeon == nil && q.Type == "" && !q.Repeatable && len(q.StartSources) == 0 && !skipped[q.ID] && (guideZones["all"] || guideZones[w.ZoneName(q.StartPos)]) {
			own[q.ID] = true
		}
	}
	all, abl, ao, aerr := bestSweep(w, own, *target, prefer)
	if aerr == nil {
		r = runOf("blocks, every quest of the zones", all)
		r.QuestIDs = sortedIDs(acceptedIn(all))
		rep.Runs = append(rep.Runs, r)
	}
	var kept map[int]bool
	var pres *route.Result
	var pbl []route.SweepBlock
	var po route.SweepOptions
	if *quick {
		if aerr != nil {
			return aerr
		}
		kept, pres, pbl, po, err = acceptedIn(all), all, abl, ao, nil
		if k, r, b := pruneBatch(w, own, *target, ao, 3); r != nil && r.Seconds < pres.Seconds {
			kept, pres, pbl = k, r, b
		}
	} else {
		kept, pres, pbl, po, err = prune(w, own, *target, prefer)
	}
	if err == nil && aerr == nil && all.Seconds < pres.Seconds {
		kept, pres, pbl, po = acceptedIn(all), all, abl, ao // leaving quests out did not help
	}
	if err == nil {
		r = runOf("blocks, own selection", pres)
		r.Near, r.Out, r.Look, r.Reach, r.Blocks, r.QuestIDs = po.Near, po.Out, po.Look, po.Reach, pbl, sortedIDs(kept)
		r.Zones, r.MayGrind = po.Zones, po.Grind
		r.Dungeons = dungeonBlocks(w, pres, shareable)
		rep.Runs = append(rep.Runs, r)
		polished("blocks, own selection", w, route.EventsOf(pres.Steps))
		if pres.Seconds < bestRes.Seconds {
			bestRes, bestBlocks, bestName = pres, pbl, r.Name
		}
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

	if *fromBlocks != "" {
		raw, err := os.ReadFile(*fromBlocks)
		if err != nil {
			return err
		}
		var bf blocksFile
		if err := json.Unmarshal(raw, &bf); err != nil || len(bf.Blocks) == 0 {
			return fmt.Errorf("%s: malformed or empty blocks file", *fromBlocks)
		}
		set := map[int]bool{}
		for _, b := range bf.Blocks {
			for _, l := range [][]int{b.Accepted, b.Worked, b.TurnedIn} {
				for _, id := range l {
					set[id] = true
				}
			}
		}
		o := route.SweepOptions{Target: *target, Near: bf.Near, Reach: bf.Reach, Out: bf.Out, Prefer: prefer, Dungeons: w.Opt.Dungeons, Order: bf.Blocks}
		ores, obl, err := w.Sweep(set, o)
		if err != nil {
			return err
		}
		r = runOf("blocks in the order of "+filepath.Base(*fromBlocks), ores)
		r.Blocks, r.QuestIDs, r.Dungeons = obl, sortedIDs(acceptedIn(ores)), dungeonBlocks(w, ores, shareable)
		rep.Runs = append(rep.Runs, r)
		bestRes, bestBlocks, bestName = ores, obl, r.Name
	}
	key := char.RouteKey()
	if *dungeons {
		key += "-dungeons"
	}
	if startLive != nil && aerr == nil {
		// a route from the middle is for playing a stretch of quests: every quest of the zones, not the fastest by the
		// model (which would leave most out and grind)
		bestRes, bestBlocks, bestName = all, abl, "blocks, every quest of the zones"
	}
	if *routeOut != "" {
		doc := map[string]any{
			"schema": "foreverroutes.route.v1", "key": key, "character": char, "target": *target, "winner": "blocks",
			"basis":   "ForeverRoutes block route (" + bestName + "); quest selection partly from ordinary RestedXP Forever guides (RestedXP/RXPGuides, CC BY-NC-SA 4.0)",
			"license": "CC-BY-NC-SA-4.0", "hours": map[string]float64{"blocks": bestRes.Seconds / 3600}, "seconds": bestRes.Seconds,
			"level": bestRes.Level, "quests": bestRes.Quests, "killProfile": bestRes.Profile, "events": route.EventsOf(bestRes.Steps),
			"replayVerified": true,
		}
		if startLive != nil {
			doc["start"] = startLive
		}
		if model.Preparations != nil {
			doc["preparations"] = model.Preparations
		}
		if model.HearthMinSave > 0 {
			doc["hearthMinSave"] = model.HearthMinSave
		}
		b, _ := json.MarshalIndent(doc, "", " ")
		if err := os.WriteFile(*routeOut, append(b, '\n'), 0o644); err != nil {
			return err
		}
	}
	if *blocksOut != "" {
		var bo route.SweepOptions
		for _, x := range rep.Runs {
			if x.Name == bestName {
				bo = route.SweepOptions{Near: x.Near, Out: x.Out, Reach: x.Reach}
			}
		}
		bf := blocksFile{Schema: "foreverroutes.blocks.v1", Key: key, Character: char, Target: *target, Hours: bestRes.Seconds / 3600,
			Near: bo.Near, Out: bo.Out, Reach: bo.Reach, Blocks: bestBlocks, Names: map[string]string{}, Shareable: map[string]bool{}}
		for _, b := range bestBlocks {
			for _, l := range [][]int{b.Accepted, b.Worked, b.TurnedIn} {
				for _, id := range l {
					if q := w.ByID[id]; q != nil {
						bf.Names[fmt.Sprint(id)] = q.Name
						if shareable != nil && id < 90000 {
							bf.Shareable[fmt.Sprint(id)] = shareable[id]
						}
					}
				}
			}
		}
		b, _ := json.MarshalIndent(bf, "", " ")
		if err := os.WriteFile(*blocksOut, append(b, '\n'), 0o644); err != nil {
			return err
		}
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
		if !*paces {
			break
		}
		wgp, err := blocksWorld(*path, *race, *class, *faction, *profile, pace, false, edit)
		if err != nil {
			return err
		}
		wp := wgp
		if *dungeons {
			if wp, err = blocksWorld(*path, *race, *class, *faction, *profile, pace, true, edit); err != nil {
				return err
			}
		}
		gp, err := wgp.FollowGuide(guide.Events, *target)
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
		fmt.Printf("\n%s (near %.0f s, out %.0f s, look %.0f s, reach %.0f yd, zones %v, grind %v): %d blocks\n", r.Name, r.Near, r.Out, r.Look, r.Reach, r.Zones, r.MayGrind, len(r.Blocks))
		for i, b := range r.Blocks {
			fmt.Printf("  %2d %-7s %-12s %5.1f-%5.1f min  lvl %2d  +%5d XP  take %d work %d hand in %d\n", i+1, b.Kind, b.Zone, b.Start/60, b.End/60, b.Level, b.XP, len(b.Accepted), len(b.Worked), len(b.TurnedIn))
		}
	}
	for _, r := range rep.Runs {
		for _, b := range r.Dungeons {
			fmt.Printf("\n%s: %s at level %d, minute %.0f-%.0f\n", r.Name, b.Name, b.Level, b.Start, b.End)
			for _, q := range b.Quests {
				share := "shareable unknown"
				if q.Shareable != nil {
					share = map[bool]string{true: "shareable", false: "NOT shareable"}[*q.Shareable]
				}
				if q.Inside {
					share = "given inside"
				}
				fmt.Printf("   %-9s %-34s %-17s before %v\n", q.State, name(q.ID), share, q.Before)
			}
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
