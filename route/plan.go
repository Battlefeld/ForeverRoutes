package route

import (
	"errors"
	"math"
	"math/rand"
	"sort"
	"sync"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// PlanOptions steer the search.
type PlanOptions struct {
	TargetLevel int   // stop at this level
	Restarts    int   // independent runs; the first is purely greedy, the others pick among the best few options
	Seed        int64 // makes the runs repeatable
	Pool        int   // how many of the best options a randomized run picks from
	MaxSteps    int   // decisions per run
	Lookahead   int   // candidates played out further before deciding (0 = default 6, -1 = none)
	Depth       int   // decisions each candidate is played out (default 3)
}

func (p PlanOptions) withDefaults() PlanOptions {
	if p.TargetLevel == 0 {
		p.TargetLevel = 30
	}
	if p.Restarts == 0 {
		p.Restarts = 4
	}
	if p.Pool == 0 {
		p.Pool = 3
	}
	if p.MaxSteps == 0 {
		p.MaxSteps = 4000
	}
	if p.Lookahead == 0 {
		p.Lookahead = 6
	}
	if p.Depth == 0 {
		p.Depth = 3
	}
	return p
}

// macro is a candidate block of actions: one quest with its neighbours, a quest hub, or grinding.
type macro struct {
	actions []Action
	state   *State
	kind    string
	score   float64 // lower is better: minus the value beyond grinding (or seconds per value point when no grind spot is near)
}

// Plan searches for a fast route to the target level. The best route of all runs is replayed and returned; it is the
// best found, not a proven optimum.
func (w *World) Plan(po PlanOptions) (*Result, error) {
	po = po.withDefaults()
	if po.TargetLevel < 2 || po.TargetLevel > len(w.Data.LevelXP) {
		return nil, errors.New("target level out of range")
	}
	rng := rand.New(rand.NewSource(po.Seed))
	var best []Action
	bestTime := math.Inf(1)
	for run := 0; run < po.Restarts; run++ {
		s := w.NewState()
		var route []Action
		for step := 0; step < po.MaxSteps && w.Level(s) < po.TargetLevel; step++ {
			options := w.options(s, po.TargetLevel)
			if len(options) == 0 {
				break
			}
			pick := w.pick(s, options, po, run, rng)
			route = append(route, pick.actions...)
			s = pick.state
		}
		if w.Level(s) >= po.TargetLevel && s.Time < bestTime {
			best, bestTime = route, s.Time
		}
	}
	if best == nil {
		return nil, errors.New("no route reaches the target level with this data")
	}
	res, err := w.Replay(best)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// pick chooses the next block. The best few by score are each played out Depth more greedy decisions; the one whose
// stretch brings the most XP per second wins. A randomized run picks among the best Pool of them.
func (w *World) pick(s *State, options []macro, po PlanOptions, run int, rng *rand.Rand) macro {
	sort.SliceStable(options, func(a, b int) bool { return options[a].score < options[b].score })
	k := po.Lookahead
	if k < 0 {
		k = 1
	}
	if k > len(options) {
		k = len(options)
	}
	cands := options[:k]
	rate := make([]float64, k)
	if k > 1 {
		var wg sync.WaitGroup
		for i := range cands {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				end := cands[i].state
				for d := 0; d < po.Depth && w.Level(end) < po.TargetLevel; d++ {
					next := w.options(end, po.TargetLevel)
					if len(next) == 0 {
						break
					}
					best := 0
					for j := range next {
						if next[j].score < next[best].score {
							best = j
						}
					}
					end = next[best].state
				}
				rate[i] = float64(end.XP-s.XP) / (end.Time - s.Time)
			}(i)
		}
		wg.Wait()
	}
	order := make([]int, k)
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return rate[order[a]] > rate[order[b]] })
	if run > 0 {
		n := po.Pool
		if n > k {
			n = k
		}
		return cands[order[rng.Intn(n)]]
	}
	return cands[order[0]]
}

// maxSwitches is how often a planned route may change continents by a level: one trip there and back up to level
// 20, then one more every five levels. Players do not cross the sea for a few quests; the time model alone does not
// see what that costs them.
func maxSwitches(level int) int {
	if level <= 20 {
		return 2
	}
	return 2 + (level-16)/5
}

// classQuestValue is what finishing one of the first class quests is worth beyond its XP.
const classQuestValue = 1500

// startArea: class quests given this close to where the character starts belong to its start (the start zone and its
// first town); other start zones have their own copies.
const startArea = 1500

// classQuestsLeft: the character is below level 6, or a class quest of its start (given by level 6, near the start)
// can be taken now or is in the log. The route stays at home until then: these quests teach the class its first
// abilities.
func (w *World) classQuestsLeft(s *State) bool {
	if w.Level(s) < 6 {
		return true
	}
	for _, q := range w.Quests {
		if q.Classes == 0 || q.MinLevel > 6 || distance(q.StartPos, w.Start) > startArea {
			continue
		}
		if st := s.Status[q.Index]; st == active || (st == notTaken && w.CanAccept(s, q) == nil) {
			return true
		}
	}
	return false
}

func (w *World) options(s *State, target int) []macro {
	var out []macro
	seen := map[int]bool{}
	cap := maxSwitches(w.Level(s))
	early := w.classQuestsLeft(s)
	add := func(m *macro) {
		if m != nil && m.state.Switches > s.Switches && (m.state.Switches > cap || early) {
			return
		}
		if m != nil && len(m.actions) > 0 && !math.IsInf(m.score, 0) && !math.IsNaN(m.score) {
			out = append(out, *m)
		}
	}
	// quests in the log, and quests that can be taken now
	for _, q := range w.Quests {
		st := s.Status[q.Index]
		if st == done || seen[q.ID] || q.Dungeon != nil {
			continue
		}
		if st == notTaken && w.CanAccept(s, q) != nil {
			continue
		}
		seen[q.ID] = true
		add(w.questMacro(s, q))
	}
	// hubs: groups of quests that can be taken at one place
	for _, h := range w.hubs(s) {
		add(w.hubMacro(s, h))
	}
	for _, d := range w.Dungeons {
		add(w.dungeonMacro(s, d))
	}
	g := w.grindMacro(s, target)
	add(g)
	// Grinding is possible almost anywhere and never runs out; quests are bound to places and fade as the character
	// outlevels them. So a block is measured by what it brings beyond grinding for the same time: its value minus
	// the XP the best grind spot here would give in that time. Grinding itself scores zero.
	if g != nil {
		rate := 1 / g.score
		// Where grinding is slow (the first levels) it is a poor yardstick: a far, slow block would win only by the
		// sheer XP it brings. The yardstick is then what the good quests here bring per second: the third best rate.
		var rates []float64
		for i := range out {
			if dt := out[i].state.Time - s.Time; dt > 0 {
				rates = append(rates, w.value(s, out[i].state)/dt)
			}
		}
		sort.Sort(sort.Reverse(sort.Float64Slice(rates)))
		if len(rates) >= 3 && rates[2] > rate {
			rate = rates[2]
		}
		for i := range out {
			m := &out[i]
			dt := m.state.Time - s.Time
			m.score = -(w.value(s, m.state) - rate*dt)
		}
	}
	return out
}

// value of a finished block: the XP it brought, plus a share of what the quests it finished unlock.
func (w *World) value(before, after *State) float64 {
	v := float64(after.XP - before.XP)
	for _, q := range w.Quests {
		if before.Status[q.Index] != done && after.Status[q.Index] == done {
			if q.Classes != 0 && q.MinLevel <= 10 && distance(q.StartPos, w.Start) <= startArea {
				v += classQuestValue // the first class quests teach abilities (totems, forms, pets): worth more than their XP
			}
			for _, u := range q.Unlocks {
				if uq := w.ByID[u]; uq != nil {
					v += 0.25 * float64(uq.XP)
				}
			}
		}
	}
	return v
}

func (w *World) finish(before, after *State, actions []Action, kind string) *macro {
	dt := after.Time - before.Time
	v := w.value(before, after)
	if v <= 0 || dt <= 0 {
		return nil
	}
	return &macro{actions: actions, state: after, kind: kind, score: dt / v}
}

// try applies an action to s and records it; false when it is not possible.
func (w *World) try(s *State, acts *[]Action, a Action) bool {
	if err := w.Apply(s, a); err != nil {
		return false
	}
	*acts = append(*acts, a)
	return true
}

// bestSource picks the source of an objective that finishes it fastest from the current position.
func (w *World) bestSource(s *State, q *Quest, oi int) (Action, bool) {
	o := q.Objectives[oi]
	left := float64(o.Count) - s.Prog[q.Index][oi]
	best, bt := Action{}, math.Inf(1)
	for _, src := range o.Sources {
		if w.tooDangerous(s, src) {
			continue
		}
		work := w.Opt.InteractSecs
		if !src.Talk {
			work, _, _ = w.killSeconds(s, src.NPC, src.Site, left/src.Chance) // respawns included: a lone rare is slow
		}
		t := w.TravelTime(s, s.Pos, src.Site.Center) + work
		if t < bt {
			bt = t
			best = Action{Op: "work", Quest: q.ID, Obj: oi, At: src.Site.Center}
			if src.NPC != nil {
				best.NPC = src.NPC.ID
			} else {
				best.Object = src.Object
			}
		}
	}
	return best, !math.IsInf(bt, 1)
}

// workLog works the unfinished objectives of the given quests, nearest first, until none is left or none can be done.
func (w *World) workLog(s *State, acts *[]Action, quests []*Quest) bool {
	for {
		var next Action
		bd, found := math.Inf(1), false
		for _, q := range quests {
			if s.Status[q.Index] != active {
				continue
			}
			for oi, o := range q.Objectives {
				if s.Prog[q.Index][oi] >= float64(o.Count) {
					continue
				}
				a, ok := w.bestSource(s, q, oi)
				if !ok {
					return false
				}
				if d := w.TravelTime(s, s.Pos, a.At); d < bd {
					bd, next, found = d, a, true
				}
			}
		}
		if !found {
			return true
		}
		if !w.try(s, acts, next) {
			return false
		}
	}
}

// turnInNear turns in every finished quest whose ender stands within radius of the character, nearest first.
func (w *World) turnInNear(s *State, acts *[]Action, radius float64) {
	for {
		var next *Quest
		bd := math.Inf(1)
		for qi := range s.Prog {
			q := w.Quests[qi]
			if !w.complete(s, q) {
				continue
			}
			if d := distance(s.Pos, q.EndPos); d <= radius && d < bd {
				next, bd = q, d
			}
		}
		if next == nil {
			return
		}
		w.try(s, acts, Action{Op: "turnin", Quest: next.ID, At: next.EndPos})
	}
}

func sharesSource(a, b *Quest) bool {
	for _, oa := range a.Objectives {
		for _, sa := range oa.Sources {
			for _, ob := range b.Objectives {
				for _, sb := range ob.Sources {
					if (sa.NPC != nil && sa.NPC == sb.NPC) || (sa.Object != 0 && sa.Object == sb.Object) {
						return true
					}
				}
			}
		}
	}
	return false
}

// questMacro: take the quest (and quests at the same giver that share its targets), finish it, turn it in, and turn in
// whatever else is finished nearby.
func (w *World) questMacro(s0 *State, q *Quest) *macro {
	s := s0.clone()
	var acts []Action
	group := []*Quest{q}
	if s.Status[q.Index] == notTaken {
		if !w.try(s, &acts, Action{Op: "accept", Quest: q.ID, At: w.startAt(s, q)}) {
			return nil
		}
		for _, o := range w.Quests {
			if o == q || s.Status[o.Index] != notTaken || distance(o.StartPos, q.StartPos) > 30 || !sharesSource(o, q) {
				continue
			}
			if w.try(s, &acts, Action{Op: "accept", Quest: o.ID, At: w.startAt(s, o)}) {
				group = append(group, o)
			}
		}
	}
	if !w.workLog(s, &acts, group) {
		return nil
	}
	if !w.complete(s, q) || !w.try(s, &acts, Action{Op: "turnin", Quest: q.ID, At: q.EndPos}) {
		return nil
	}
	w.turnInNear(s, &acts, 40)
	return w.finish(s0, s, acts, "quest")
}

type hub struct {
	center dataset.Point
	radius float64
	quests []*Quest
}

// hub radii: one quest giver group (60 yards), a village (250), the quest area around a village (600)
var hubRadii = []float64{60, 250, 600}

// hubs groups the quests that can be taken now by where they are given, at each hub radius.
func (w *World) hubs(s *State) []hub {
	var avail []*Quest
	for _, q := range w.Quests {
		if s.Status[q.Index] == notTaken && q.Dungeon == nil && w.CanAccept(s, q) == nil {
			avail = append(avail, q)
		}
	}
	var hs []hub
	for _, r := range hubRadii {
		used := map[int]bool{}
		for _, q := range avail {
			if used[q.ID] {
				continue
			}
			h := hub{center: q.StartPos, radius: r}
			for _, o := range avail {
				if !used[o.ID] && distance(o.StartPos, q.StartPos) <= r {
					h.quests = append(h.quests, o)
					used[o.ID] = true
				}
			}
			if len(h.quests) >= 2 {
				hs = append(hs, h)
			}
		}
	}
	return hs
}

// hubMacro: take every quest of a hub (as far as the log allows), work all objectives of the log nearest first, then
// turn everything in.
func (w *World) hubMacro(s0 *State, h hub) *macro {
	s := s0.clone()
	var acts []Action
	for _, q := range h.quests {
		w.try(s, &acts, Action{Op: "accept", Quest: q.ID, At: w.startAt(s, q)})
	}
	if len(acts) == 0 {
		return nil
	}
	var log []*Quest
	for qi := range s.Prog {
		log = append(log, w.Quests[qi])
	}
	sort.Slice(log, func(a, b int) bool { return log[a].ID < log[b].ID })
	// objectives that cannot be done yet (mobs too high) are left for later
	var doable []*Quest
	for _, q := range log {
		ok := distance(q.EndPos, h.center) <= h.radius+700
		for oi := range q.Objectives {
			if s.Prog[q.Index][oi] < float64(q.Objectives[oi].Count) {
				if a, can := w.bestSource(s, q, oi); !can || distance(a.At, h.center) > h.radius+700 {
					ok = false
				}
			}
		}
		if ok {
			doable = append(doable, q)
		}
	}
	if !w.workLog(s, &acts, doable) {
		return nil
	}
	// turn in, nearest first
	for {
		var next *Quest
		bd := math.Inf(1)
		for qi := range s.Prog {
			q := w.Quests[qi]
			if w.complete(s, q) {
				if d := w.TravelTime(s, s.Pos, q.EndPos); d < bd {
					next, bd = q, d
				}
			}
		}
		if next == nil {
			break
		}
		if !w.try(s, &acts, Action{Op: "turnin", Quest: next.ID, At: next.EndPos}) {
			break
		}
	}
	return w.finish(s0, s, acts, "hub")
}

// grindMacro: kill the best mobs nearby until the next level (at most 30 kills).
func (w *World) grindMacro(s0 *State, target int) *macro {
	var best *macro
	lvl := w.Level(s0)
	for i := range w.Farms {
		f := &w.Farms[i]
		ml := mobLevel(f.NPC)
		if ml > lvl+w.Opt.MaxMobAbove || ml < lvl-4 {
			continue
		}
		if distance(s0.Pos, f.Site.Center) > 2500 {
			continue
		}
		s := s0.clone()
		var acts []Action
		if !w.try(s, &acts, Action{Op: "grind", NPC: f.NPC.ID, Kills: 30, At: f.Site.Center}) {
			continue
		}
		m := w.finish(s0, s, acts, "grind")
		if m != nil && (best == nil || m.score < best.score) {
			best = m
		}
	}
	return best
}
