package route

import (
	"github.com/Battlefeld/ForeverRoutes/dataset"
	"math"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

// Live stages: the long way from where the character stands, planned the way a player plans it.
//
// A player first decides where to go for the next hours (the Barrens until 16, Thunder Bluff to hand in, Undercity and
// the Ruins, back to the Barrens) and then works each place off before moving on. The free block planner (live_block)
// decides block by block: whatever single block is best right now wins, so a large hub far away pulls the character
// across the map and the next decision pulls it back. Here a run stays in its zone while the zone still holds quests
// worth more than grinding; only then (or right at the start) does it leave, and the next zones are chosen by playing
// each candidate out over the horizon and comparing the XP per hour of the whole stretch.

// Stage is one leg of the long plan: a zone (or a city, or a dungeon run), with the levels and time spent there.
type Stage struct {
	Zone      string  `json:"zone"`
	Dungeon   string  `json:"dungeon,omitempty"`
	FromLevel int     `json:"fromLevel"`
	ToLevel   int     `json:"toLevel"`
	Minutes   float64 `json:"minutes"`
	FirstStep int     `json:"firstStep"` // index of the stage's first step in the plan
	Quests    int     `json:"quests"`    // quests handed in during the stage
}

const (
	StageHorizon  = 4 * 3600.0 // seconds of play the stages plan
	stageLevels   = 6          // ... or this many levels, whichever comes first
	stageBranches = 3          // zones compared at each point where the run may leave its zone
	stageDepth    = 3          // points of leaving at which zones are compared; later ones follow the best option
)

// macroZone: the zone a block keeps the character in ("" when it leaves it) and the zone it ends in.
func (w *World) macroZone(m *macro, zone string) (stays bool, end string) {
	stays = true
	for _, a := range m.actions {
		if w.zoneOf(a.At) != zone {
			stays = false
		}
	}
	return stays, w.zoneOf(m.actions[len(m.actions)-1].At)
}

type stageRun struct {
	acts []Action
	end  *State
}

// stageSearch holds what all runs of one PlanStages share.
type stageSearch struct {
	w        *World
	s0       *State
	po       PlanOptions
	deadline time.Time
	home     string // the zone the plan starts in
	stopAt   float64
	cap      int
}

// PlanStages plans from s for about StageHorizon seconds: the best of the zone sequences compared within budget, nil
// when nothing can be planned.
func (w *World) PlanStages(s *State, po PlanOptions, budget time.Duration) (*Result, []Stage) {
	po = po.withDefaults()
	w.liveLocal = true // set once before the search; the world is only read concurrently after
	ss := &stageSearch{w: w, s0: s, po: po, deadline: time.Now().Add(budget), stopAt: s.Time + StageHorizon, cap: min(po.TargetLevel, w.Level(s)+stageLevels)}
	ss.home = w.zoneOf(s.Pos)
	best := ss.run(s.clone(), nil, ss.home, stageDepth, true, false)
	if best == nil || len(best.acts) == 0 {
		return nil, nil
	}
	res, err := w.replayFrom(s, w.takeAlong(s, best.acts))
	if err != nil {
		return nil, nil
	}
	return res, w.Stages(s, res)
}

func (ss *stageSearch) rate(r *stageRun) float64 {
	if r == nil || r.end.Time <= ss.s0.Time {
		return math.Inf(-1)
	}
	// a goal switched on is an obligation: what of it is left at the end of the run still has to be walked, so it
	// counts as time. Done now or later then costs the same unless the run does it on its way (cheaper), and the
	// goal is neither rushed (inside the horizon) nor pushed past it.
	return float64(r.end.XP-ss.s0.XP) / (r.end.Time - ss.s0.Time + goalLater*ss.w.goalLeft(r.end))
}

func (ss *stageSearch) done(s *State) bool {
	return s.Time >= ss.stopAt || ss.w.Level(s) >= ss.cap
}

// run plays on from s in zone: the zone's best block while it holds one worth more than grinding; at a point of
// leaving (and at the start) the best few zones to go to are each played out and the best whole run is kept.
func (ss *stageSearch) run(s *State, acts []Action, zone string, depth int, root, homeDone bool) *stageRun {
	w := ss.w
	for steps := 0; steps < 400 && !ss.done(s); steps++ {
		ss.sleep(s, &acts)
		options := w.options(s, ss.po.TargetLevel)
		if len(options) == 0 {
			break
		}
		options = ss.readyDungeons(s, withoutPickups(options))
		if !homeDone {
			options = ss.withoutErrandsAway(options, zone) // a message to deliver elsewhere waits for the trip there
		}
		if !homeDone && !ss.w.goalBegun(s) {
			options = withoutGoals(ss.w, options) // goals wait until the start zone is worked off (below)
		}
		ss.favourGoals(s, options)
		sort.SliceStable(options, func(a, b int) bool { return options[a].score < options[b].score })
		// the best block that stays in the zone, and the best block into each other zone
		var here *macro
		away := map[string]*macro{}
		var zones []string
		for i := range options {
			m := &options[i]
			stays, end := w.macroZone(m, zone)
			if stays {
				if here == nil {
					here = m
				}
				continue
			}
			if away[end] == nil {
				away[end] = m
				zones = append(zones, end)
			}
		}
		// worth staying: a block here brings more than grinding (grinding itself scores zero)
		// worth staying: the zone's best block brings more per second than grinding here (the planner's scores measure
		// against the best quests anywhere, which would call a zone done as soon as something far pays more)
		// worth staying: a block of the zone brings more per second than grinding here and not much less than the
		// best block elsewhere (the planner's scores measure against the best quests anywhere, which calls a zone done
		// as soon as anything far pays more; plain rates keep a zone while it is good, not only while it is best)
		awayRate := 0.0
		for _, m := range away {
			awayRate = math.Max(awayRate, ss.rateOf(s, m, rateSpan))
		}
		stay := here != nil && here.kind == "grind" && len(zones) == 0
		if here != nil && !stay {
			// in the zone the block that brings the most per second, the way there included: the near quests before
			// the far ones, so the run works the zone off instead of crossing it for the biggest block each time
			best := 0.0
			for i := range options {
				m := &options[i]
				if m.kind == "grind" {
					continue
				}
				if stays, _ := w.macroZone(m, zone); stays && ss.beatsGrinding(s, m) {
					r := ss.rateOf(s, m, nearSpan)
					pick := r
					if m.kind == "hub" {
						pick *= hubPreferred // take what is offered here at once, then work it off, then hand it in
					}
					if r >= stayShare*awayRate && pick > best {
						here, stay, best = m, true, pick
					}
				}
			}
		}
		homeDone = homeDone || zone == ss.home && !stay
		if stay || depth <= 0 || time.Now().After(ss.deadline) || len(zones) == 0 {
			next := here
			if !stay || next == nil {
				next = &options[0]
			}
			acts = append(acts, next.actions...)
			s = next.state
			if _, end := w.macroZone(next, zone); end != zone {
				zone = end
			}
			root = false
			continue
		}
		// a point of leaving: compare staying (when worth it) with the best zones to go to
		type cand struct {
			m    *macro
			zone string
		}
		var cands []cand
		if stay {
			cands = append(cands, cand{here, zone})
		}
		for _, z := range zones {
			if len(cands) >= stageBranches+1 || !stay && len(cands) >= stageBranches {
				break
			}
			cands = append(cands, cand{away[z], z})
		}
		// a goal that needs its own trip waits until the zone the plan starts in is worked off: the player finishes
		// what is around first, then sets out
		if g := w.goalMacro(options); g != nil && homeDone {
			_, end := w.macroZone(g, zone)
			dup := false
			for _, c := range cands {
				dup = dup || c.m == g
			}
			if !dup {
				cands = append(cands, cand{g, end})
			}
		}
		results := make([]*stageRun, len(cands))
		var wg sync.WaitGroup
		for i, c := range cands {
			wg.Add(1)
			go func(i int, c cand) {
				defer wg.Done()
				a := append(append([]Action(nil), acts...), c.m.actions...)
				results[i] = ss.run(c.m.state, a, c.zone, depth-1, false, homeDone)
			}(i, c)
		}
		wg.Wait()
		var best *stageRun
		for _, r := range results {
			if best == nil || ss.rate(r) > ss.rate(best) {
				best = r
			}
		}
		return best
	}
	return &stageRun{acts: acts, end: s}
}

// sleep lays the sleeping bag down when its buff has run out and it is ready again: at the end, not before (a fresh
// bag before the old buff ends throws the rest of it away), and only while the stretch left pays for the minutes.
func (ss *stageSearch) sleep(s *State, acts *[]Action) {
	w := ss.w
	stacks := w.preparation.BagStacks
	if stacks == 0 && w.liveGoals[79976] && w.Opt.Preparations.SleepingBag != "off" {
		stacks = 3 // the bag chain switched on in the addon: the bag is for resting on
	}
	if stacks < 1 || s.BagUntil > s.Time || s.Time < s.BagReady || !w.bagOwned(s) || !w.canSleep(s) {
		return
	}
	m := w.Opt.SleepingBag
	cost := m.PlacementSeconds + float64(stacks)*m.StackSeconds
	if ss.stopAt-s.Time < cost/(float64(stacks)*m.PerStack) {
		return
	}
	w.try(s, acts, Action{Op: "sleep", Stacks: stacks, At: s.Pos})
}

// Stages splits a plan into its legs: consecutive steps in one zone (a dungeon run is a leg of its own). Short
// passes through a zone (on the way, under two minutes and no quest) belong to the leg after them.
func (w *World) Stages(s *State, r *Result) []Stage {
	var out []Stage
	var keys []string
	prevTime, prevLevel := 0.0, w.Level(s) // step times count from s
	for i, st := range r.Steps {
		zone, dungeon := w.zoneOf(st.At), ""
		key := zone
		if around, ok := capitals[zone]; ok {
			key = around // a capital and the zone around it are one leg (Undercity and Tirisfal)
		}
		if st.Op == "dungeon" {
			dungeon = st.Dungeon
			if d := w.dungeonByKey(st.Dungeon); d != nil {
				zone = d.Name
			}
			key = "dungeon:" + st.Dungeon
		}
		if n := len(out); n == 0 || keys[n-1] != key {
			if n > 0 && out[n-1].Quests == 0 && out[n-1].Minutes < 2 && out[n-1].Dungeon == "" {
				out, keys = out[:n-1], keys[:n-1] // a pass on the way
			}
			if n := len(out); n > 0 && keys[n-1] == key {
				// the leg before the pass goes on
			} else {
				out, keys = append(out, Stage{Zone: zone, Dungeon: dungeon, FromLevel: prevLevel, FirstStep: i}), append(keys, key)
			}
		}
		cur := &out[len(out)-1]
		if dungeon == "" && !strings.Contains(" / "+cur.Zone+" / ", " / "+zone+" / ") {
			cur.Zone += " / " + zone
		}
		cur.Minutes += (st.Time - prevTime) / 60
		cur.ToLevel = st.Level
		if st.Op == "turnin" {
			cur.Quests++
		}
		prevTime, prevLevel = st.Time, st.Level
	}
	for i := range out {
		out[i].Minutes = math.Round(out[i].Minutes)
	}
	return out
}

func (w *World) dungeonByKey(key string) *Dungeon {
	for _, d := range w.Dungeons {
		if d.Key == key {
			return d
		}
	}
	return nil
}

// Head is the start of a plan to show: its steps up to seconds of play, and at least through the given number of
// hand-ins (the addon plays a block of quests before it asks again).
func (r *Result) Head(seconds float64, turnins int) *Result {
	out := *r
	handed := 0
	for i, st := range r.Steps {
		if st.Time > seconds && handed >= turnins {
			out.Steps = r.Steps[:i]
			break
		}
		if st.Op == "turnin" {
			handed++
		}
	}
	if n := len(out.Steps); n > 0 && n < len(r.Steps) {
		last := out.Steps[n-1]
		out.Seconds, out.Level, out.XP = last.Time, last.Level, last.XP
	}
	return &out
}

// goalBonus: what a quest of a goal switched on in the addon (the sleeping bag chain, a class chain) is worth beyond
// its XP to the stages: the player wants the goal, the run takes its quests along where they lie.
const goalBonus = 3000.0

// goalLater: what is left of a goal at the end of a run is charged a bit more than its bare walk (the bonus it would
// have brought meanwhile, and a later trip may not combine with anything), so a goal is done once it is on the way,
// not put off at every new plan.
const goalLater = 1.0

// SetLiveGoals names the goals switched on in the addon: each goal's chain in the order it is played.
func (w *World) SetLiveGoals(chains [][]int) {
	w.liveGoals, w.liveGoalChains = map[int]bool{}, nil
	for _, c := range chains {
		for _, id := range c {
			w.liveGoals[id] = true
		}
		w.liveGoalChains = append(w.liveGoalChains, c)
	}
}

// favourGoals: a chain once begun is followed on (its next step before other blocks of the same worth); a chain not
// begun is chosen at a point of leaving (run), never by the bonus alone.
func (ss *stageSearch) favourGoals(s *State, options []macro) {
	for _, chain := range ss.w.liveGoalChains {
		begun, left := false, false
		for _, id := range chain {
			if q := ss.w.ByID[id]; q != nil {
				begun = begun || s.Status[q.Index] != notTaken
				left = left || s.Status[q.Index] != done
			}
		}
		if !begun || !left {
			continue
		}
		for i := range options {
			for _, a := range options[i].actions {
				if (a.Op == "turnin" || a.Op == "accept") && slices.Contains(chain, a.Quest) {
					options[i].score -= goalBonus
					break
				}
			}
		}
	}
}

// goalLeft: seconds of travel the goals switched on still need from s: from where s stands through every quest left,
// in chain order (each giver, then its ender).
func (w *World) goalLeft(s *State) float64 {
	t, at := 0.0, s.Pos
	for _, chain := range w.liveGoalChains {
		for _, id := range chain {
			q := w.ByID[id]
			if q == nil || s.Status[q.Index] == done {
				continue
			}
			if s.Status[q.Index] == notTaken {
				t += w.TravelTime(s, at, q.StartPos)
				at = q.StartPos
			}
			t += w.TravelTime(s, at, q.EndPos)
			at = q.EndPos
		}
	}
	return t
}

// goalMacro: the best block that takes or hands in a quest of a goal (a candidate at every point of leaving).
func (w *World) goalMacro(options []macro) *macro {
	for i := range options {
		if w.forGoal(&options[i]) {
			return &options[i]
		}
	}
	return nil
}

// forGoal: the block takes or hands in a quest of a goal switched on.
func (w *World) forGoal(m *macro) bool {
	for _, a := range m.actions {
		if (a.Op == "accept" || a.Op == "turnin") && w.liveGoals[a.Quest] {
			return true
		}
	}
	return false
}

// GoalLeft is goalLeft for reports.
func (w *World) GoalLeft(s *State) float64 { return w.goalLeft(s) }

// dungeonFrom: the level from which the leveling guides send a group into a dungeon (the dataset's minimum is the
// first level the game lets in). Live, a run is planned from here on: a fresh group of the lowest level wipes.
var dungeonFrom = map[string]int{"ragefire-chasm": 14, "ruins-of-lordaeron": 17, "wailing-caverns": 18,
	"deadmines": 18, "shadowfang-keep": 22, "blackfathom-deeps": 25, "razorfen-kraul": 29}

// readyDungeons drops the dungeon runs the character is still below the guides' level for.
func (ss *stageSearch) readyDungeons(s *State, options []macro) []macro {
	l := ss.w.Level(s)
	out := options[:0]
	for _, m := range options {
		ok := true
		for _, a := range m.actions {
			if a.Op == "dungeon" && l < dungeonFrom[a.Dungeon] {
				ok = false
			}
		}
		if ok {
			out = append(out, m)
		}
	}
	return out
}

// goalBegun: a goal's chain has a quest taken or done and quests left.
func (w *World) goalBegun(s *State) bool {
	for _, chain := range w.liveGoalChains {
		begun, left := false, false
		for _, id := range chain {
			if q := w.ByID[id]; q != nil {
				begun = begun || s.Status[q.Index] != notTaken
				left = left || s.Status[q.Index] != done
			}
		}
		if begun && left {
			return true
		}
	}
	return false
}

func withoutGoals(w *World, options []macro) []macro {
	out := options[:0]
	for _, m := range options {
		if !w.forGoal(&m) {
			out = append(out, m)
		}
	}
	return out
}

// beatsGrinding: the block brings more XP per second (its value: XP and what it unlocks) than the best grind spot near.
func (ss *stageSearch) beatsGrinding(s *State, m *macro) bool {
	dt := m.state.Time - s.Time
	if dt <= 0 {
		return true
	}
	g := ss.w.grindMacro(s, ss.po.TargetLevel)
	if g == nil {
		return true
	}
	gdt := g.state.Time - s.Time
	return gdt <= 0 || ss.w.value(s, m.state)/dt > ss.w.value(s, g.state)/gdt
}

// stayShare: a zone is kept while a block of it brings at least this share of the best block elsewhere per second.
const stayShare = 0.7

// rateOf: the value a block brings per second, over at least span seconds. Elsewhere a block is measured over
// rateSpan: handing in finished quests takes a minute and pays a lot, which says nothing about how good a zone is.
// The XP of quests finished already does not count: it is earned, whenever the character hands them in.
func (ss *stageSearch) rateOf(s *State, m *macro, span float64) float64 {
	v := ss.w.value(s, m.state)
	for _, a := range m.actions {
		if q := ss.w.ByID[a.Quest]; a.Op == "turnin" && q != nil && ss.w.complete(s, q) {
			v -= float64(q.XP)
		}
	}
	return v / math.Max(math.Max(span, 1), m.state.Time-s.Time)
}

const rateSpan = 900.0

// takeAlong adds the errands a plan passes by: a quest without objectives (deliver a message, report to someone) that
// is offered where a step of the plan stands and handed in where a later step stands is taken there and handed in
// there. It costs a few seconds of talking; the plan goes there anyway.
func (w *World) takeAlong(s0 *State, acts []Action) []Action {
	const near, aside = 60.0, 90.0 // yards to the giver; seconds aside to the ender
	states := make([]*State, len(acts)+1)
	s := s0.clone()
	states[0] = s.clone()
	for i, a := range acts {
		if w.Apply(s, a) != nil {
			return acts
		}
		states[i+1] = s.clone()
	}
	planned := map[int]bool{}
	for _, a := range acts {
		if a.Op == "accept" {
			planned[a.Quest] = true
		}
	}
	out := acts
	for _, q := range w.Quests {
		if len(q.Objectives) > 0 || q.Dungeon != nil || planned[q.ID] || s0.Status[q.Index] != notTaken {
			continue
		}
		// where it is offered (the first step there that may take it) and where it is handed in after that
		from, to := -1, -1
		for i := range out {
			if from < 0 {
				if distance(out[i].At, q.StartPos) <= near && i+1 < len(states) && w.CanAccept(states[i+1], q) == nil {
					from = i
				}
			} else if i+1 < len(states) && w.TravelTime(states[i+1], out[i].At, q.EndPos) <= aside {
				to = i
				break
			}
		}
		if from < 0 || to < 0 {
			continue
		}
		try := slices.Clone(out[:from+1])
		try = append(try, Action{Op: "accept", Quest: q.ID, At: q.StartPos})
		try = append(try, out[from+1:to+1]...)
		try = append(try, Action{Op: "turnin", Quest: q.ID, At: q.EndPos})
		try = append(try, out[to+1:]...)
		if _, err := w.replayFrom(s0, try); err != nil {
			continue
		}
		out = try
		// the states after the insertion point moved by one or two steps: rebuild them for the next quest
		s = s0.clone()
		states = append(states[:0], s.clone())
		for _, a := range out {
			w.Apply(s, a)
			states = append(states, s.clone())
		}
	}
	return out
}

// nearSpan: a block in the zone is measured over at least this many seconds (a hand-in next door is not worth a
// detour for itself).
const nearSpan = 300.0

// withoutPickups drops the blocks that only take dungeon quests: the run of the dungeon takes them on its way there,
// and taken alone they pull the character across the zone for a quest done hours later.
func withoutPickups(options []macro) []macro {
	out := options[:0]
	for _, m := range options {
		if m.kind != "pickup" {
			out = append(out, m)
		}
	}
	return out
}

// withoutErrandsAway drops the blocks that only talk (deliver a message, hand in a finished quest) in another zone:
// such a trip is made when the zone here is worked off, and the errands go along then (takeAlong, the hand-ins there).
func (ss *stageSearch) withoutErrandsAway(options []macro, zone string) []macro {
	out := options[:0]
	for _, m := range options {
		if stays, _ := ss.w.macroZone(&m, zone); stays || !errand(ss.w, &m) {
			out = append(out, m)
		}
	}
	return out
}

// errand: a block of hand-ins and quests without objectives only, no fighting or gathering.
func errand(w *World, m *macro) bool {
	for _, a := range m.actions {
		switch a.Op {
		case "turnin":
		case "accept":
			if q := w.ByID[a.Quest]; q == nil || len(q.Objectives) > 0 {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// nearestFirst orders quests by their givers, each time the nearest to the last one (from p).
func (w *World) nearestFirst(p dataset.Point, quests []*Quest) []*Quest {
	left := slices.Clone(quests)
	out := make([]*Quest, 0, len(left))
	for len(left) > 0 {
		bi := 0
		for i, q := range left {
			if distance(p, q.StartPos) < distance(p, left[bi].StartPos) {
				bi = i
			}
		}
		out = append(out, left[bi])
		p = left[bi].StartPos
		left = slices.Delete(left, bi, bi+1)
	}
	return out
}

// hubPreferred: a hub (every quest offered at a place taken at once, worked off together, handed in together) is
// preferred to single quests of a little more XP per second: one quest after the other from the same town means
// walking out and back for each.
const hubPreferred = 1.25
