package route

import (
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Area sweep: a route built from blocks instead of a search over single actions.
//
// The character always does what is near first (hand in, accept, work an objective), so a hub is emptied and an area
// is finished before it is left. Only when nothing is near it moves on, to the place where the most is to do for the
// way there. A block is what happens between two such moves: one stay at a hub or one round through an area.
// The quests come from outside (a guide's selection, a zone); the sweep decides the order only. Its actions are
// replayed by the same rules as every other route.

// SweepOptions tune the sweep.
type SweepOptions struct {
	Target int // level to reach
	// Near: seconds of travel within which the next thing is simply done (0 = 75). Everything at a hub and every
	// objective of an area lies within it; a larger value keeps the character in an area longer.
	Near float64
	// Reach: yards around a place whose open work counts for going there (0 = 500)
	Reach float64
	// Out: while an objective of the log can be worked within this many seconds of travel, the character stays out
	// and works it instead of walking to a quest giver that is not near (0 = no such rule): a hub is left with a
	// full log and seen again when the log is done
	Out float64
	// Look: seconds a move to another place is played ahead for each candidate place; the one with the most XP per
	// hour over that time is taken (0 = the place with the most to do for the way). Wide: places compared (0 = 4).
	Look float64
	Wide int
	// Dungeons: the dungeons of the world (Options.Dungeons) are run as blocks: their quests and what leads to them
	// are taken on the way, the run is one candidate among the places to go to
	Dungeons bool
	// Order: saved blocks to follow in their order: what a block takes, works and hands in is done before the
	// next block begins; within a block the sweep's rules decide. Empty: the sweep finds the blocks itself.
	Order []SweepBlock
	// Prefer: quest -> quests players took before it ("Offered after"), beside the dataset's own (Quest.PreSoft):
	// while one of them is in the set and still open, the quest waits, unless nothing else is left to do
	Prefer map[int][]int
}

// SweepBlock is one stay: what was taken, worked and handed in before the character moved on.
type SweepBlock struct {
	Name     string        `json:"name"` // zone and number of the stay in it: "Durotar 3"
	Zone     string        `json:"zone"`
	At       dataset.Point `json:"at"`
	Kind     string        `json:"kind"` // hub (quest givers), round (objectives), mixed, grind, dungeon
	Dungeon  string        `json:"dungeon,omitempty"`
	Accepted []int         `json:"accepted,omitempty"`
	Worked   []int         `json:"worked,omitempty"` // quests with an objective worked here
	TurnedIn []int         `json:"turnedIn,omitempty"`
	Start    float64       `json:"start"` // seconds into the route
	End      float64       `json:"end"`
	Level    int           `json:"level"` // at the end
	XP       int           `json:"xp"`    // gained in the block
}

type sweepTodo struct {
	a    Action
	cost float64 // seconds to get there
	m    *macro  // a dungeon block (its quests outside, the run, the hand-ins) instead of one action
}

type sweepKey [3]int

type sweeper struct {
	w      *World
	o      SweepOptions
	set    map[int]bool
	quests []*Quest
	rank   map[sweepKey]int // with an order: the block an action belongs to (objectives by quest: Obj is 0)
	cur    int              // the block being played
}

// ranked: the block of an action in the order; false for what no block names (free once the order is through).
func (sw *sweeper) ranked(a Action) (int, bool) {
	k := sweepKeyOf(a)
	k[2] = 0
	r, ok := sw.rank[k]
	return r, ok
}

func sweepKeyOf(a Action) sweepKey {
	op := map[string]int{"accept": 1, "turnin": 2, "work": 3}[a.Op]
	return sweepKey{op, a.Quest, a.Obj}
}

func (sw *sweeper) waits(s *State, q *Quest) bool {
	for _, p := range append(append([]int(nil), q.PreSoft...), sw.o.Prefer[q.ID]...) {
		if pq := sw.w.ByID[p]; pq != nil && sw.set[p] && s.Status[pq.Index] != done {
			return true
		}
	}
	return false
}

// todo lists what the character can do now: quests to take, objectives to work, quests to hand in. waiting tells
// that a quest was held back for a preferred one.
func (sw *sweeper) todo(s *State, failed map[sweepKey]int, relaxed bool) (todo []sweepTodo, waiting bool) {
	w, lvl := sw.w, sw.w.Level(s)
	add := func(a Action) {
		if at, ok := failed[sweepKeyOf(a)]; ok && at == lvl {
			return
		}
		if sw.rank != nil && sw.cur < len(sw.o.Order) {
			if r, ok := sw.ranked(a); !ok || r > sw.cur {
				return // a later block's, or no block's while the order is being played
			}
		}
		if t := w.TravelTime(s, s.Pos, a.At); !math.IsInf(t, 1) {
			todo = append(todo, sweepTodo{a: a, cost: t})
		}
	}
	for _, q := range sw.quests {
		switch s.Status[q.Index] {
		case notTaken:
			if q.StartInside || w.CanAccept(s, q) != nil {
				continue
			}
			if !relaxed && sw.waits(s, q) {
				waiting = true
				continue
			}
			add(Action{Op: "accept", Quest: q.ID, At: w.startAt(s, q)})
		case active:
			if w.complete(s, q) {
				if !q.EndInside {
					add(Action{Op: "turnin", Quest: q.ID, At: q.EndPos})
				}
				continue
			}
			for oi, ob := range q.Objectives {
				if ob.Inside || s.Prog[q.Index][oi] >= float64(ob.Count) {
					continue
				}
				if a, ok := w.bestSource(s, q, oi); ok {
					add(a)
				}
			}
		}
	}
	sort.SliceStable(todo, func(i, j int) bool { return todo[i].cost < todo[j].cost })
	return todo, waiting
}

// levelUp: nothing can be done at this level: grind to where the next quest or objective opens.
func (sw *sweeper) levelUp(s *State) *macro {
	w, lvl, next := sw.w, sw.w.Level(s), 0
	for _, q := range sw.quests {
		n := 0
		switch s.Status[q.Index] {
		case notTaken:
			if q.MinLevel > lvl {
				n = q.MinLevel
			}
		case active:
			if !w.complete(s, q) {
				n = w.levelFor(s, q, sw.o.Target)
			}
		}
		if n > lvl && (next == 0 || n < next) {
			next = n
		}
	}
	if next == 0 || next > sw.o.Target {
		next = sw.o.Target
	}
	return w.grindMacro(s, next)
}

// places picks, of what is to do, the places worth going to when nothing is near: each open thing within reach of a
// place counts for it, against the way there; the best few places that are not each other's neighbours.
func (sw *sweeper) places(todo []sweepTodo) []sweepTodo {
	type scored struct {
		t sweepTodo
		v float64
	}
	all := make([]scored, len(todo))
	for i, c := range todo {
		n := 0
		for _, d := range todo {
			if distance(c.a.At, d.a.At) <= sw.o.Reach {
				n++
			}
		}
		all[i] = scored{c, float64(n) / (c.cost + 30*float64(n))}
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].v > all[j].v })
	wide := sw.o.Wide
	if wide <= 0 {
		wide = 4
	}
	var out []sweepTodo
	for _, c := range all {
		near := false
		for _, o := range out {
			near = near || distance(c.t.a.At, o.a.At) <= sw.o.Reach
		}
		if !near {
			if out = append(out, c.t); len(out) >= wide {
				break
			}
		}
	}
	return out
}

// do applies a choice; a failed action is remembered for this level.
func (sw *sweeper) do(s *State, acts *[]Action, failed map[sweepKey]int, c sweepTodo) *State {
	if c.m != nil {
		*acts = append(*acts, c.m.actions...)
		return c.m.state
	}
	if !sw.w.try(s, acts, c.a) {
		failed[sweepKeyOf(c.a)] = sw.w.Level(s)
	}
	return s
}

// step makes one decision and applies it; false when nothing is left to do. look: a move to another place is
// chosen by playing each candidate ahead (off inside such a look ahead).
func (sw *sweeper) step(s *State, acts *[]Action, failed map[sweepKey]int, relaxed *bool, look bool) (*State, bool) {
	w, o := sw.w, sw.o
	todo, waiting := sw.todo(s, failed, *relaxed)
	if sw.rank != nil && sw.cur < len(o.Order) {
		open := false // something of the block being played is still to do
		for _, c := range todo {
			if r, _ := sw.ranked(c.a); r == sw.cur {
				open = true
			}
		}
		if d := o.Order[sw.cur].Dungeon; d != "" && !open {
			if dd := w.dungeonOf[d]; dd != nil && s.Runs&(1<<dd.Index) == 0 {
				if m := w.dungeonMacro(s, dd); m != nil {
					*acts = append(*acts, m.actions...)
					sw.cur++
					return m.state, true
				}
			}
		}
		if !open {
			sw.cur++ // the block is done (or cannot be done now): on to the next
			return s, true
		}
	}
	if len(todo) == 0 && waiting {
		*relaxed = true
		return s, true
	}
	*relaxed = false
	var runs []sweepTodo
	if o.Dungeons && (sw.rank == nil || sw.cur >= len(o.Order)) {
		for _, d := range w.Dungeons {
			if m := w.dungeonMacro(s, d); m != nil {
				runs = append(runs, sweepTodo{a: Action{Op: "dungeon", Dungeon: d.Key, At: d.Entrance}, cost: w.TravelTime(s, s.Pos, d.Entrance), m: m})
			}
		}
	}
	if len(todo) == 0 && len(runs) == 0 {
		m := sw.levelUp(s)
		if m == nil {
			return s, false
		}
		*acts = append(*acts, m.actions...)
		return m.state, true
	}
	if len(todo) > 0 && todo[0].cost <= o.Near {
		return sw.do(s, acts, failed, todo[0]), true // what is near is simply done
	}
	if o.Out > 0 {
		for _, c := range todo {
			if c.a.Op == "work" && c.cost <= o.Out {
				return sw.do(s, acts, failed, c), true // stay out while the log can be worked
			}
		}
	}
	cands := append(sw.places(todo), runs...)
	pick := cands[0]
	if look && o.Look > 0 && len(cands) > 1 {
		best := math.Inf(-1)
		for _, c := range cands {
			f := make(map[sweepKey]int, len(failed))
			for k, v := range failed {
				f[k] = v
			}
			var tmp []Action
			keep := sw.cur
			r := sw.do(s.clone(), &tmp, f, c)
			rel := false
			for i := 0; i < 600 && r.Time-s.Time < o.Look && w.Level(r) < o.Target; i++ {
				var ok bool
				if r, ok = sw.step(r, &tmp, f, &rel, false); !ok {
					break
				}
			}
			sw.cur = keep
			if dt := r.Time - s.Time; dt > 0 {
				if v := float64(r.XP-s.XP) / dt; v > best {
					best, pick = v, c
				}
			}
		}
	} else if len(runs) > 0 && !look && o.Look > 0 {
		// inside a look ahead a run that is possible is taken when its entrance is the nearest place
		for _, c := range runs {
			if c.cost < pick.cost {
				pick = c
			}
		}
	}
	return sw.do(s, acts, failed, pick), true
}

// Sweep plans a route through the quests of set (quest ids) by areas and replays it. Quests of the set this
// character cannot do are left out; after the last quest it grinds to the target, as a followed guide does.
func (w *World) Sweep(set map[int]bool, o SweepOptions) (*Result, []SweepBlock, error) {
	if o.Target < 2 || o.Target > len(w.Data.LevelXP)+1 {
		return nil, nil, errors.New("target level out of range")
	}
	if o.Near <= 0 {
		o.Near = 75
	}
	if o.Reach <= 0 {
		o.Reach = 500
	}
	sw := &sweeper{w: w, o: o, set: map[int]bool{}}
	for id := range set {
		sw.set[id] = true
	}
	if o.Dungeons {
		// a dungeon's quests and what leads to them belong to its block
		for _, d := range w.Dungeons {
			for _, q := range d.Quests {
				sw.set[q.ID] = true
				for _, p := range w.ChainBefore(q) {
					sw.set[p.ID] = true
				}
			}
		}
	}
	for _, q := range w.Quests {
		if sw.set[q.ID] {
			sw.quests = append(sw.quests, q)
		}
	}
	sort.Slice(sw.quests, func(i, j int) bool { return sw.quests[i].ID < sw.quests[j].ID })
	if len(o.Order) > 0 {
		sw.rank = map[sweepKey]int{}
		put := func(op int, ids []int, i int) {
			for _, id := range ids {
				if _, ok := sw.rank[sweepKey{op, id, 0}]; !ok {
					sw.rank[sweepKey{op, id, 0}] = i
				}
			}
		}
		for i, b := range o.Order {
			put(1, b.Accepted, i)
			put(3, b.Worked, i)
			put(2, b.TurnedIn, i)
		}
	}
	s := w.NewState()
	var acts []Action
	failed := map[sweepKey]int{} // an action that did not work at a level is not tried again at that level
	relaxed := false             // nothing else is left: the quests that wait for a preferred one are taken as well
	for step := 0; step < 20000 && w.Level(s) < o.Target; step++ {
		var ok bool
		if s, ok = sw.step(s, &acts, failed, &relaxed, true); !ok {
			break
		}
	}
	for w.Level(s) < o.Target {
		m := w.grindMacro(s, o.Target)
		if m == nil {
			break
		}
		acts = append(acts, m.actions...)
		s = m.state
	}
	if len(acts) == 0 {
		return nil, nil, errors.New("nothing to do in this set of quests")
	}
	res, err := w.Replay(acts)
	if err != nil {
		return nil, nil, err
	}
	return res, w.sweepBlocks(res, o), nil
}

// sweepBlocks cuts a replayed route into its stays: a new block begins when the character travels farther than the
// sweep's reach from where the block began.
func (w *World) sweepBlocks(res *Result, o SweepOptions) []SweepBlock {
	var out []SweepBlock
	var cur *SweepBlock
	prevXP, prevTime := 0, 0.0
	stays := map[string]int{}
	has := func(l []int, id int) bool {
		for _, x := range l {
			if x == id {
				return true
			}
		}
		return false
	}
	flush := func() {
		if cur == nil {
			return
		}
		switch {
		case cur.Dungeon != "":
			cur.Kind = "dungeon"
		case len(cur.Worked) == 0 && len(cur.Accepted)+len(cur.TurnedIn) > 0:
			cur.Kind = "hub"
		case len(cur.Worked) > 0 && len(cur.Accepted)+len(cur.TurnedIn) == 0:
			cur.Kind = "round"
		case len(cur.Worked) > 0:
			cur.Kind = "mixed"
		default:
			cur.Kind = "grind"
		}
		stays[cur.Zone]++
		cur.Name = fmt.Sprintf("%s %d", cur.Zone, stays[cur.Zone])
		out = append(out, *cur)
		cur = nil
	}
	for _, st := range res.Steps {
		if cur != nil && (distance(cur.At, st.At) > o.Reach || st.Op == "dungeon" || cur.Dungeon != "") {
			flush()
		}
		if cur == nil {
			cur = &SweepBlock{Zone: w.zoneOf(st.At), At: st.At, Start: prevTime}
		}
		switch st.Op {
		case "dungeon":
			cur.Dungeon = st.Dungeon
		case "accept":
			cur.Accepted = append(cur.Accepted, st.Quest)
		case "turnin":
			cur.TurnedIn = append(cur.TurnedIn, st.Quest)
		case "work":
			if !has(cur.Worked, st.Quest) {
				cur.Worked = append(cur.Worked, st.Quest)
			}
		}
		cur.End, cur.Level = st.Time, st.Level
		cur.XP += st.XP - prevXP
		prevXP, prevTime = st.XP, st.Time
	}
	flush()
	return out
}
