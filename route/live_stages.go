package route

import (
	"math"
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
	stopAt   float64
	cap      int
}

// PlanStages plans from s for about StageHorizon seconds: the best of the zone sequences compared within budget, nil
// when nothing can be planned.
func (w *World) PlanStages(s *State, po PlanOptions, budget time.Duration) (*Result, []Stage) {
	po = po.withDefaults()
	ss := &stageSearch{w: w, s0: s, po: po, deadline: time.Now().Add(budget), stopAt: s.Time + StageHorizon, cap: min(po.TargetLevel, w.Level(s)+stageLevels)}
	best := ss.run(s.clone(), nil, w.zoneOf(s.Pos), stageDepth, true)
	if best == nil || len(best.acts) == 0 {
		return nil, nil
	}
	res, err := w.replayFrom(s, best.acts)
	if err != nil {
		return nil, nil
	}
	return res, w.Stages(s, res)
}

func (ss *stageSearch) rate(r *stageRun) float64 {
	if r == nil || r.end.Time <= ss.s0.Time {
		return math.Inf(-1)
	}
	v := float64(r.end.XP - ss.s0.XP)
	for _, a := range r.acts {
		if a.Op == "turnin" && ss.w.liveGoals[a.Quest] {
			v += goalBonus // a goal switched on counts in the whole stretch too, not only in the next decision
		}
	}
	return v / (r.end.Time - ss.s0.Time)
}

func (ss *stageSearch) done(s *State) bool {
	return s.Time >= ss.stopAt || ss.w.Level(s) >= ss.cap
}

// run plays on from s in zone: the zone's best block while it holds one worth more than grinding; at a point of
// leaving (and at the start) the best few zones to go to are each played out and the best whole run is kept.
func (ss *stageSearch) run(s *State, acts []Action, zone string, depth int, root bool) *stageRun {
	w := ss.w
	for steps := 0; steps < 400 && !ss.done(s); steps++ {
		ss.sleep(s, &acts)
		options := w.options(s, ss.po.TargetLevel)
		if len(options) == 0 {
			break
		}
		ss.favourGoals(options)
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
		stay := here != nil && (here.score < 0 || here.kind == "grind" && len(zones) == 0)
		if !root && stay || depth <= 0 || time.Now().After(ss.deadline) || len(zones) == 0 {
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
		results := make([]*stageRun, len(cands))
		var wg sync.WaitGroup
		for i, c := range cands {
			wg.Add(1)
			go func(i int, c cand) {
				defer wg.Done()
				a := append(append([]Action(nil), acts...), c.m.actions...)
				results[i] = ss.run(c.m.state, a, c.zone, depth-1, false)
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
const goalBonus = 10000.0

// SetLiveGoals names the quests of the goals switched on in the addon (their whole chains).
func (w *World) SetLiveGoals(quests []int) {
	w.liveGoals = map[int]bool{}
	for _, id := range quests {
		w.liveGoals[id] = true
	}
}

// favourGoals lowers the score of blocks that hand in quests of the goals switched on.
func (ss *stageSearch) favourGoals(options []macro) {
	if len(ss.w.liveGoals) == 0 {
		return
	}
	for i := range options {
		for _, a := range options[i].actions {
			if a.Op == "turnin" && ss.w.liveGoals[a.Quest] {
				options[i].score -= goalBonus
			}
		}
	}
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
