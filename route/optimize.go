package route

import (
	"fmt"
	"math"
	"math/rand"
	"slices"
	"sort"
	"time"
)

type OptimizeOptions struct {
	Target          int
	Budget          time.Duration
	Seed            int64
	Keep            []int
	History         int
	Stats           *OptimizeStats
	Seeds           []OptimizationSeed
	RequireDungeons []string
	ExcludeDungeons []string
	Checkpoint      string
	// MaxIterations bounds stochastic evaluations deterministically when set; wall-clock Budget still bounds all phases.
	MaxIterations int
}
type DungeonCombinationCheck struct {
	Keys           []string `json:"keys"`
	CandidateFound bool     `json:"candidateFound"`
	Seconds        float64  `json:"seconds,omitempty"`
}
type OptimizeStats struct {
	DungeonCombinationsAllowed   int
	DungeonChecks                []DungeonCombinationCheck
	Walks, Kept, Better          [11]int
	SeedsEvaluated               int
	DungeonCombinationsEvaluated int
	DungeonCombinationsTotal     int
	DungeonEnumerationComplete   bool
	PreparationCandidates        int
	Restarts                     int
	CheckpointLoaded             bool
	BestSeconds                  float64
}

const optimizeReach = 400.0

// Optimize keeps a replay-verified incumbent across seeds, dungeon subsets, preparation policies and large moves.
// The simulation remains authoritative: every candidate is evaluated forward with level, quest and buff state.
func (w *World) Optimize(events []GuideEvent, o OptimizeOptions) ([]GuideEvent, *GuideResult, error) {
	if o.Target < 2 || o.Target > len(w.Data.LevelXP)+1 || o.Budget < 0 {
		return nil, nil, fmt.Errorf("invalid target or search budget")
	}
	originalPlan, originalWand := w.Preparations(), w.WandDetour()
	success := false
	defer func() {
		if !success {
			w.preparation = originalPlan
			w.SetWandDetour(originalWand)
		}
	}()
	stats := o.Stats
	if stats == nil {
		stats = &OptimizeStats{}
	}
	*stats = OptimizeStats{}
	deadline := time.Now().Add(o.Budget)
	phaseDeadline := time.Now().Add(o.Budget / 3)
	preparationDeadline := time.Now().Add(o.Budget * 2 / 3)
	var goals []int
	for _, c := range w.ClassChains(o.Target) {
		if c.Tier == "required" {
			goals = append(goals, c.Goal.ID)
		}
	}
	goals = append(goals, o.Keep...)
	if w.Opt.Preparations.SleepingBag == "on" {
		goals = append(goals, 79976)
	}
	sort.Ints(goals)
	goals = slices.Compact(goals)
	for _, id := range goals {
		if w.ByID[id] == nil {
			return nil, nil, fmt.Errorf("required quest %d is not available", id)
		}
	}
	for _, key := range append(slices.Clone(o.RequireDungeons), o.ExcludeDungeons...) {
		if w.dungeonOf[key] == nil {
			return nil, nil, fmt.Errorf("unknown or unavailable dungeon %q", key)
		}
	}
	for _, key := range o.RequireDungeons {
		if slices.Contains(o.ExcludeDungeons, key) {
			return nil, nil, fmt.Errorf("dungeon %s is both required and excluded", key)
		}
	}
	valid := func(r *GuideResult) bool {
		if r == nil || r.Level < o.Target || !w.tankFirst(r) || !w.planAllowed(w.Preparations()) {
			return false
		}
		for _, id := range goals {
			if !handsIn(r, id) {
				return false
			}
		}
		if w.Opt.Preparations.SleepingBag == "off" && handsIn(r, 79976) {
			return false
		}
		keys := actualDungeonKeys(r)
		for _, key := range o.RequireDungeons {
			if !slices.Contains(keys, key) {
				return false
			}
		}
		for _, key := range o.ExcludeDungeons {
			if slices.Contains(keys, key) {
				return false
			}
		}
		if w.Opt.Preparations.Wands == "on" && o.Target >= lesserWandLevel && !r.CraftedWand {
			return false
		}
		if w.Opt.Preparations.Camping == "on" && o.Target >= campLevel && !r.ProfessionsLearned {
			return false
		}

		return true
	}
	cost := func(e []GuideEvent, limit float64) *GuideResult {
		r, err := w.walkFast(e, o.Target, limit+1)
		if err != nil || !valid(r) || w.runAbove(e, o.Target) {
			return nil
		}
		return r
	}
	type candidate struct {
		events []GuideEvent
		plan   PreparationPlan
		wand   float64
		res    *GuideResult
	}
	var best *candidate
	var checkpointErr error
	key := ""
	if o.Checkpoint != "" {
		var err error
		key, err = w.optimizationKey(o, goals)
		if err != nil {
			return nil, nil, err
		}
	}
	consider := func(e []GuideEvent) *GuideResult {
		limit := math.Inf(1)
		r := cost(e, limit)
		if r != nil && (best == nil || r.Seconds < best.res.Seconds-.05) {
			best = &candidate{slices.Clone(e), w.Preparations(), w.WandDetour(), r}
			stats.BestSeconds = r.Seconds
			if o.Checkpoint != "" {
				checkpointErr = w.saveCheckpoint(o.Checkpoint, key, best.events, best.plan, best.wand, o.Target)
			}
		}
		return r
	}
	// A checkpoint is re-evaluated, never trusted for its recorded seconds. Its context must match exactly.
	if o.Checkpoint != "" {
		cp, err := loadCheckpoint(o.Checkpoint, key)
		if err != nil {
			return nil, nil, err
		}
		if cp != nil {
			if err := w.SetPreparations(cp.Preparations); err != nil {
				return nil, nil, err
			}
			w.SetWandDetour(cp.WandDetour)
			if consider(cp.Events) == nil {
				return nil, nil, fmt.Errorf("checkpoint is no longer feasible")
			}
			stats.CheckpointLoaded = true
		}
	}
	seeds := append([]OptimizationSeed{{Events: events, Preparations: &originalPlan, WandDetour: originalWand}}, o.Seeds...)
	allowed := map[string]bool{}
	for _, d := range w.Dungeons {
		allowed[d.Key] = !slices.Contains(o.ExcludeDungeons, d.Key)
	}
	// Named input guides remain independent seeds; no result is attributed back to an unchanged guide.
	for _, seed := range seeds {
		p := originalPlan
		if seed.Preparations != nil {
			p = *seed.Preparations
		}
		if w.Opt.Preparations.Camping == "on" {
			p.Camping = true
		}
		if w.Opt.Preparations.Camping == "off" {
			p.Camping = false
		}
		if w.Opt.Preparations.Wands == "on" {
			p.Wands = true
		}
		if w.Opt.Preparations.Wands == "off" {
			p.Wands = false
		}
		if w.Opt.Preparations.SleepingBag == "off" {
			p.BagStacks = 0
		}
		if err := w.SetPreparations(p); err != nil {
			return nil, nil, err
		}
		w.SetWandDetour(seed.WandDetour)
		e := w.withClassQuests(w.filterDungeons(seed.Events, allowed), o.Target)
		if w.Opt.Preparations.SleepingBag == "off" {
			for _, c := range w.ClassChains(o.Target) {
				if c.Goal.ID == 79976 {
					ids := map[int]bool{}
					for _, q := range c.Quests {
						ids[q.ID] = true
					}
					e = without(e, func(e GuideEvent) bool { return ids[e.Quest] || e.Op == "sleep" })
				}
			}
		}
		for _, c := range w.ClassChains(o.Target) {
			if slices.Contains(goals, c.Goal.ID) {
				if r, err := w.walkFast(e, o.Target, math.Inf(1)); err == nil && handsIn(r, c.Goal.ID) {
					continue
				}
				e = w.WeaveIn(e, c, o.Target)
			}
		}
		for _, id := range o.Keep {
			q := w.ByID[id]
			if q != nil && !slices.ContainsFunc(e, func(x GuideEvent) bool { return x.Quest == id }) {
				at, _ := w.closestStep(e, q.StartPos)
				if len(e) == 0 {
					at = -1
				}
				if added := w.insertQuestChain(e, q, at+1); added != nil {
					e = added
				}
			}
		}
		if len(o.RequireDungeons) > 0 {
			selected := map[string]bool{}
			for _, k := range RunKeys(e) {
				selected[k] = true
			}
			for _, k := range o.RequireDungeons {
				selected[k] = true
			}
			if placed := w.dungeonSubset(e, selected, o.Target, time.Now().Add(30*time.Second)); placed != nil {
				e = placed
			}
		}
		if consider(e) == nil {
			fallback := p
			if w.Opt.Preparations.Wands == "on" {
				fallback.ForceWand = true
			}
			if w.Opt.Preparations.Camping == "on" {
				fallback.ForceProfessions = true
			}
			if fallback != p {
				w.preparation = fallback
				consider(e)
			}
		}
		stats.SeedsEvaluated++
	}
	if best == nil {
		return nil, nil, fmt.Errorf("no starting route satisfies the level, class and selected-option constraints")
	}
	w.preparation = best.plan
	w.SetWandDetour(best.wand)
	var pool []*Quest
	for _, q := range w.Quests {
		if q.MinLevel <= o.Target {
			pool = append(pool, q)
		}
	}
	// First compare complete dungeon subsets, including zero. More than eight eligible dungeons uses a bounded portfolio.
	var ds []*Dungeon
	for _, d := range w.Dungeons {
		if d.Min < o.Target && allowed[d.Key] {
			ds = append(ds, d)
		}
	}
	n := len(ds)
	stats.DungeonCombinationsTotal = -1
	if n <= 30 {
		stats.DungeonCombinationsTotal = 1 << n
	}
	masks := []uint64{}
	if n <= 8 {
		for mask := uint64(0); mask < uint64(1)<<n; mask++ {
			masks = append(masks, mask)
		}
	} else {
		masks = append(masks, 0)
		for i := 0; i < min(n, 63); i++ {
			masks = append(masks, uint64(1)<<i)
		}
	}
	for _, mask := range masks {
		matches := true
		for _, key := range o.RequireDungeons {
			included := false
			for i, d := range ds {
				if d.Key == key && i < 64 && mask&(uint64(1)<<i) != 0 {
					included = true
				}
			}
			matches = matches && included
		}
		if matches {
			stats.DungeonCombinationsAllowed++
		}
	}
	subsetSeed := slices.Clone(best.events)
	for _, mask := range masks {
		if time.Now().After(phaseDeadline) {
			break
		}
		chosen := map[string]bool{}
		for i, d := range ds {
			if i < 64 && mask&(uint64(1)<<i) != 0 {
				chosen[d.Key] = true
			}
		}
		skip := false
		for _, k := range o.RequireDungeons {
			skip = skip || !chosen[k]
		}
		if skip {
			continue
		}
		w.preparation = best.plan
		w.SetWandDetour(best.wand)
		check := DungeonCombinationCheck{Keys: []string{}}
		for _, d := range ds {
			if chosen[d.Key] {
				check.Keys = append(check.Keys, d.Key)
			}
		}
		if e := w.dungeonSubset(subsetSeed, chosen, o.Target, phaseDeadline); e != nil {
			if r := cost(e, math.Inf(1)); r != nil {
				actual := actualDungeonKeys(r)
				exact := len(actual) == len(check.Keys)
				for _, key := range check.Keys {
					exact = exact && slices.Contains(actual, key)
				}
				if exact {
					check.CandidateFound = true
					check.Seconds = r.Seconds
					consider(e)
				}
			}
		}
		stats.DungeonChecks = append(stats.DungeonChecks, check)
		stats.DungeonCombinationsEvaluated++
	}
	stats.DungeonEnumerationComplete = n <= 8 && stats.DungeonCombinationsEvaluated == stats.DungeonCombinationsAllowed
	// Evaluate acquisition AND subsequent use together: an expensive setup alone may hide a later benefit.
	prepSeeds := [][]GuideEvent{slices.Clone(best.events)}
	basePlan := best.plan
	if w.Opt.Preparations.SleepingBag != "off" && w.ByID[79976] != nil && !handsIn(best.res, 79976) && time.Now().Before(preparationDeadline) {
		for _, c := range w.ClassChains(o.Target) {
			if c.Goal.ID == 79976 {
				prepSeeds = append(prepSeeds, w.WeaveIn(best.events, c, o.Target))
			}
		}
	}
	camping := []bool{basePlan.Camping, !basePlan.Camping}
	wands := []bool{basePlan.Wands}
	if wandClasses[w.Char.Class] {
		wands = append(wands, !basePlan.Wands)
	}
	for _, prepSeed := range prepSeeds {
		for _, camp := range camping {
			for _, wand := range wands {
				for _, level := range []int{1, 8, 10} {
					for _, stacks := range []int{0, 1, 2, 3} {
						if time.Now().After(preparationDeadline) {
							break
						}
						if !camp && level != 1 {
							continue
						}
						p := basePlan
						p.Camping = camp
						p.Wands = wand
						p.ProfessionLevel = level
						p.BagStacks = stacks
						if !w.planAllowed(p) {
							continue
						}
						w.preparation = p
						w.SetWandDetour(best.wand)
						for _, forceWand := range []bool{false, true} {
							if !wand && forceWand {
								continue
							}
							p.ForceWand = forceWand
							for _, forceProf := range []bool{false, true} {
								if !camp && forceProf {
									continue
								}
								if time.Now().After(preparationDeadline) {
									break
								}
								p.ForceProfessions = forceProf
								w.preparation = p
								consider(w.bagRests(prepSeed, stacks, o.Target))
								stats.PreparationCandidates++
							}
						}
					}
				}
			}
		}
	}

	w.preparation = best.plan
	w.SetWandDetour(best.wand)
	rng := rand.New(rand.NewSource(o.Seed))
	hist := o.History
	if hist <= 0 {
		hist = 400
	}
	cur := slices.Clone(best.events)
	curC := best.res.Seconds
	late := make([]float64, hist)
	for i := range late {
		late[i] = curC
	}
	weights := [3]int{6, 3, 2}
	stale := 0
	for k := 0; time.Now().Before(deadline) && (o.MaxIterations <= 0 || k < o.MaxIterations); k++ {
		wandBefore := w.WandDetour()
		var e []GuideEvent
		kind := 0
		draw := rng.Intn(weights[0] + weights[1] + weights[2])
		op := 0
		switch {
		case draw < weights[0]:
			e, kind = w.edit(cur, rng, pool, o.Target, wandClasses[w.Char.Class] && w.preparation.Wands)
		case draw < weights[0]+weights[1]:
			op = 1
			kind = 9
			e = w.destroyRepair(cur, rng, o.Target)
		default:
			op = 2
			kind = 10
			e = w.moveVisit(cur, rng)
		}
		if e == nil {
			stale++
			continue
		}
		stats.Walks[kind]++
		r := cost(e, math.Max(curC, late[k%hist]))
		improved := false
		if r != nil && (r.Seconds <= curC || r.Seconds <= late[k%hist]) {
			cur, curC = e, r.Seconds
			stats.Kept[kind]++
			late[k%hist] = curC
			if curC < best.res.Seconds-.05 {
				consider(cur)
				stats.Better[kind]++
				weights[op] = min(20, weights[op]+1)
				improved = true
			}
		} else {
			w.SetWandDetour(wandBefore)
		}
		if improved {
			stale = 0
		} else {
			stale++
		}
		if stale >= 500 {
			// Restart from a different valid seed or the incumbent; the incumbent itself is never discarded.
			w.preparation = best.plan
			w.SetWandDetour(best.wand)
			cur = slices.Clone(best.events)
			curC = best.res.Seconds
			if len(seeds) > 1 {
				s := seeds[rng.Intn(len(seeds))]
				if r := cost(s.Events, math.Inf(1)); r != nil {
					cur, curC = slices.Clone(s.Events), r.Seconds
				}
			}
			for i := range late {
				late[i] = curC
			}
			stats.Restarts++
			stale = 0
		}
	}
	w.preparation = best.plan
	w.SetWandDetour(best.wand)
	// Remove inert source events together. Each produced no action/state change,
	// so replaying every long obsolete suffix once per event is unnecessary.
	var pruned []GuideEvent
	for i, e := range best.events {
		if i < len(best.res.used) && best.res.used[i] {
			pruned = append(pruned, e)
		}
	}
	if len(pruned) < len(best.events) {
		if r := cost(pruned, best.res.Seconds); r != nil && r.Seconds <= best.res.Seconds {
			best.events, best.res = pruned, r
		}
	}
	for i := len(best.events) - 1; i >= 0; i-- {
		e := slices.Delete(slices.Clone(best.events), i, i+1)
		if r := cost(e, best.res.Seconds); r != nil && r.Seconds <= best.res.Seconds {
			best.events, best.res = e, r
		}
	}
	if o.Checkpoint != "" {
		cp, err := loadCheckpoint(o.Checkpoint, key)
		if err != nil {
			return nil, nil, err
		}
		if cp != nil {
			w.preparation = cp.Preparations
			w.SetWandDetour(cp.WandDetour)
			consider(cp.Events)
		}
		w.preparation = best.plan
		w.SetWandDetour(best.wand)
	}
	full, err := w.FollowGuide(best.events, o.Target)
	if err != nil || !valid(full) {
		return nil, nil, fmt.Errorf("optimized route failed final replay: %v", err)
	}
	stats.BestSeconds = full.Seconds
	if o.Checkpoint != "" {
		if err := w.saveCheckpoint(o.Checkpoint, key, best.events, best.plan, best.wand, o.Target); err != nil {
			return nil, nil, err
		}
	}
	if checkpointErr != nil {
		return nil, nil, checkpointErr
	}
	success = true
	return best.events, full, nil
}

// edit: one random change of the route and its kind, nil when the one drawn does not apply.
func (w *World) edit(cur []GuideEvent, rng *rand.Rand, pool []*Quest, target int, wand bool) (out []GuideEvent, kind int) {
	n := len(cur)
	if n < 2 {
		return nil, kind
	}
	span := func() int { return 1 + int(math.Abs(rng.NormFloat64())*30) } // mostly near, now and then far
	switch p := rng.Intn(100); {
	case p < 20: // a step to another visit of its place: one trip to a town instead of two (a turn-in waits for the
		// quest there that can be taken later)
		kind = 8
		i := rng.Intn(n)
		pi, ok := w.eventPos(cur[i])
		if !ok {
			return nil, kind
		}
		var js []int
		for j, e := range cur {
			if pj, ok := w.eventPos(e); ok && (j < i-3 || j > i+3) && distance(pi, pj) <= visitRadius {
				js = append(js, j)
			}
		}
		if len(js) == 0 {
			return nil, kind
		}
		j := js[rng.Intn(len(js))]
		out := slices.Delete(slices.Clone(cur), i, i+1)
		if j > i {
			j-- // after the deletion
		}
		if i > j {
			j++ // after the visit's step
		}
		return slices.Insert(out, j, cur[i]), kind
	case p < 35: // one step elsewhere
		kind = 0
		i := rng.Intn(n)
		j := i + span()*(1-2*rng.Intn(2))
		if j < 0 || j >= n || j == i {
			return nil, kind
		}
		out := slices.Delete(slices.Clone(cur), i, i+1)
		return slices.Insert(out, j, cur[i]), kind
	case p < 50: // a quest's steps together, earlier or later
		kind = 1
		q := cur[rng.Intn(n)].Quest
		if q == 0 {
			return nil, kind
		}
		d := span() * (1 - 2*rng.Intn(2))
		var rest, steps []GuideEvent
		var at []int
		for i, e := range cur {
			if e.Quest == q && e.Op != "level" {
				steps, at = append(steps, e), append(at, i)
			} else {
				rest = append(rest, e)
			}
		}
		base := len(rest)
		for k, e := range steps {
			// its place among the others, moved; the quest's steps stay in order (at only grows)
			j := min(max(at[k]-k+d, 0), base) + k
			rest = slices.Insert(rest, min(j, len(rest)), e)
		}
		return rest, kind
	case p < 65: // a stretch of the route elsewhere (a stay in a zone before or after another)
		kind = 2
		l := 2 + rng.Intn(min(60, n-1))
		i := rng.Intn(n - l + 1)
		seg := slices.Clone(cur[i : i+l])
		rest := slices.Delete(slices.Clone(cur), i, i+l)
		j := rng.Intn(len(rest) + 1)
		if j == i {
			return nil, kind
		}
		return slices.Insert(rest, j, seg...), kind
	case p < 75: // a quest left out
		kind = 3
		q := cur[rng.Intn(n)].Quest
		if q == 0 || w.keepClassQuest(q, target) {
			return nil, kind
		}
		return without(cur, func(e GuideEvent) bool { return e.Quest == q }), kind
	case p < 88: // a quest put in where the route passes its giver
		kind = 4
		if len(pool) == 0 {
			return nil, kind
		}
		q := pool[rng.Intn(len(pool))]
		if slices.ContainsFunc(cur, func(e GuideEvent) bool { return e.Quest == q.ID }) {
			return nil, kind
		}
		i, d := w.closestStep(cur, q.StartPos)
		if d > optimizeReach {
			return nil, kind
		}
		return w.insertQuestChain(cur, q, i+1), kind
	case p < 94: // a dungeon run put in or left out
		kind = 5
		if !w.Opt.Dungeons || len(w.Dungeons) == 0 {
			return nil, kind
		}
		d := w.Dungeons[rng.Intn(len(w.Dungeons))]
		if at := slices.IndexFunc(cur, func(e GuideEvent) bool { return e.Op == "dungeon" && e.Dungeon == d.Key }); at >= 0 {
			return slices.Delete(slices.Clone(cur), at, at+1), kind
		}
		return slices.Insert(slices.Clone(cur), rng.Intn(n+1), GuideEvent{Op: "dungeon", Dungeon: d.Key}), kind
	case p < 97: // a level step left out
		kind = 6
		i := rng.Intn(n)
		if cur[i].Op != "level" {
			return nil, kind
		}
		return slices.Delete(slices.Clone(cur), i, i+1), kind
	default: // the detour for the wands
		kind = 7
		if !wand {
			return nil, kind
		}
		w.wandDetour = WandDetours[rng.Intn(len(WandDetours))]
		return slices.Clone(cur), kind
	}
}

// runAbove: a route runs a dungeon that is for characters above the target level.
func (w *World) runAbove(events []GuideEvent, target int) bool {
	for _, e := range events {
		if d := w.dungeonOf[e.Dungeon]; e.Op == "dungeon" && d != nil && d.Min > target {
			return true
		}
	}
	return false
}
