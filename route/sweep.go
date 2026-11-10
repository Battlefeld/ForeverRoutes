package route

import (
	"errors"
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
	// Prefer: quest -> quests players took before it ("Offered after"): while one of them is in the set and still
	// open, the quest waits, unless nothing else is left to do
	Prefer map[int][]int
}

// SweepBlock is one stay: what was taken, worked and handed in before the character moved on.
type SweepBlock struct {
	Zone     string        `json:"zone"`
	At       dataset.Point `json:"at"`
	Kind     string        `json:"kind"` // hub (quest givers), round (objectives), mixed, grind
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
	var quests []*Quest
	for _, q := range w.Quests {
		if set[q.ID] {
			quests = append(quests, q)
		}
	}
	sort.Slice(quests, func(i, j int) bool { return quests[i].ID < quests[j].ID })
	s := w.NewState()
	var acts []Action
	failed := map[[3]int]int{} // an action that did not work at a level is not tried again at that level
	key := func(a Action) [3]int {
		op := map[string]int{"accept": 1, "turnin": 2, "work": 3}[a.Op]
		return [3]int{op, a.Quest, a.Obj}
	}
	waits := func(s *State, q *Quest) bool {
		for _, p := range o.Prefer[q.ID] {
			if pq := w.ByID[p]; pq != nil && set[p] && s.Status[pq.Index] != done {
				return true
			}
		}
		return false
	}
	relaxed := false // nothing else is left: the quests that wait for a preferred one are taken as well
	for step := 0; step < 20000 && w.Level(s) < o.Target; step++ {
		lvl := w.Level(s)
		var todo []sweepTodo
		waiting := false
		add := func(a Action) {
			if at, ok := failed[key(a)]; ok && at == lvl {
				return
			}
			if t := w.TravelTime(s, s.Pos, a.At); !math.IsInf(t, 1) {
				todo = append(todo, sweepTodo{a, t})
			}
		}
		for _, q := range quests {
			switch s.Status[q.Index] {
			case notTaken:
				if w.CanAccept(s, q) == nil {
					if !relaxed && waits(s, q) {
						waiting = true
						continue
					}
					add(Action{Op: "accept", Quest: q.ID, At: w.startAt(s, q)})
				}
			case active:
				if w.complete(s, q) {
					add(Action{Op: "turnin", Quest: q.ID, At: q.EndPos})
					continue
				}
				for oi, ob := range q.Objectives {
					if s.Prog[q.Index][oi] >= float64(ob.Count) {
						continue
					}
					if a, ok := w.bestSource(s, q, oi); ok {
						add(a)
					}
				}
			}
		}
		if len(todo) == 0 && waiting {
			relaxed = true
			continue
		}
		relaxed = false
		if len(todo) == 0 {
			// nothing can be done at this level: level to where the next quest or objective opens
			next := 0
			for _, q := range quests {
				n := 0
				switch s.Status[q.Index] {
				case notTaken:
					if q.MinLevel > lvl {
						n = q.MinLevel
					}
				case active:
					if !w.complete(s, q) {
						n = w.levelFor(s, q, o.Target)
					}
				}
				if n > lvl && (next == 0 || n < next) {
					next = n
				}
			}
			if next == 0 || next > o.Target {
				next = o.Target
			}
			m := w.grindMacro(s, next)
			if m == nil {
				break
			}
			acts = append(acts, m.actions...)
			s = m.state
			continue
		}
		sort.SliceStable(todo, func(i, j int) bool { return todo[i].cost < todo[j].cost })
		pick := todo[0]
		if pick.cost > o.Near && o.Out > 0 {
			for _, c := range todo {
				if c.a.Op == "work" && c.cost <= o.Out {
					pick = c // todo is sorted: the nearest objective
					break
				}
			}
		}
		if pick.cost > o.Near && pick.a.Op != "work" || pick.cost > o.Near && o.Out <= 0 {
			// nothing near: go where the most is to do for the way (each open thing within reach of a place counts,
			// and so does the time it saves to be done with a place for good)
			best := math.Inf(-1)
			for _, c := range todo {
				n := 0
				for _, d := range todo {
					if distance(c.a.At, d.a.At) <= o.Reach {
						n++
					}
				}
				if v := float64(n) / (c.cost + 30*float64(n)); v > best {
					best, pick = v, c
				}
			}
		}
		if !w.try(s, &acts, pick.a) {
			failed[key(pick.a)] = lvl
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
		case len(cur.Worked) == 0 && len(cur.Accepted)+len(cur.TurnedIn) > 0:
			cur.Kind = "hub"
		case len(cur.Worked) > 0 && len(cur.Accepted)+len(cur.TurnedIn) == 0:
			cur.Kind = "round"
		case len(cur.Worked) > 0:
			cur.Kind = "mixed"
		default:
			cur.Kind = "grind"
		}
		out = append(out, *cur)
		cur = nil
	}
	for _, st := range res.Steps {
		if cur != nil && distance(cur.At, st.At) > o.Reach {
			flush()
		}
		if cur == nil {
			cur = &SweepBlock{Zone: w.zoneOf(st.At), At: st.At, Start: prevTime}
		}
		switch st.Op {
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
