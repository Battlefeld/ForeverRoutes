package route

import (
	"fmt"
	"math"
)

// GuideEvent is one step of a quest guide: accept, complete or turn in a quest, or reach a level.
type GuideEvent struct {
	Op      string `json:"op"` // accept, complete, turnin, level, dungeon (a group run)
	Quest   int    `json:"quest,omitempty"`
	Level   int    `json:"level,omitempty"`
	Dungeon string `json:"dungeon,omitempty"`
}

// GuideResult is a guide walked with this model, for comparing guides with planned routes on equal terms.
type GuideResult struct {
	*Result
	Quests  int            `json:"guideQuests"`  // quests turned in
	Grind   float64        `json:"grindSeconds"` // grinding the guide asks for (level steps) and to the target level
	Skipped map[string]int `json:"skipped"`      // events that could not be followed, by reason
	Missed  []GuideMiss    `json:"missed"`       // each skipped quest event with its reason
	levels  []int          // the character's level as each event came up (the target after the walk ended)
}

// GuideMiss is a guide event that could not be followed.
type GuideMiss struct {
	Op     string `json:"op"`
	Quest  int    `json:"quest"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"` // why the dataset leaves the quest out for this character
}

// FollowGuide walks the events of a guide in order: objectives are worked when the guide completes or turns in a
// quest, level steps are reached by grinding at the best spot nearby, and after the last event the character grinds to
// the target level. Quests this character or the dataset cannot do are skipped and counted.
// guideStartItem: the longest a guide follower farms the item that starts a quest.
const guideStartItem = 1200.0

// guideLevelUp: a follower levels up to this many levels for an objective the guide comes too early for.
const guideLevelUp = 2

func (w *World) FollowGuide(events []GuideEvent, target int) (*GuideResult, error) {
	return w.followFrom(w.NewState(), events, target, math.Inf(1))
}

// FollowGuideFrom walks a guide from a live state for at most horizon seconds of play: what the character has done
// already is skipped (its accepts fail silently, its objectives are complete), the rest follows the guide's order.
// The result's times count from s.
func (w *World) FollowGuideFrom(s *State, events []GuideEvent, target int, horizon float64) (*GuideResult, error) {
	return w.followFrom(s.clone(), events, target, horizon)
}

func (w *World) followFrom(s *State, events []GuideEvent, target int, horizon float64) (*GuideResult, error) {
	return w.follow(s, events, target, horizon, false)
}

// walkFast walks a guide as FollowGuide does without replaying it for the time line: the result has the time, the
// level, the levels per event and the steps' actions only (enough to weigh a route, as Optimize does).
// It stops once the walk takes longer than limit (then the level is below the target).
func (w *World) walkFast(events []GuideEvent, target int, limit float64) (*GuideResult, error) {
	return w.follow(w.NewState(), events, target, limit, true)
}

func (w *World) follow(s *State, events []GuideEvent, target int, horizon float64, fast bool) (*GuideResult, error) {
	s0 := s.clone()
	var acts []Action
	res := &GuideResult{Skipped: map[string]int{}}
	grindTo := func(level int) {
		for w.Level(s) < level && s.Time-s0.Time < horizon {
			m := w.grindMacro(s, target)
			if m == nil {
				res.Skipped["no grind spot"]++
				return
			}
			res.Grind += m.state.Time - s.Time
			acts = append(acts, m.actions...)
			s = m.state
		}
	}
	res.levels = make([]int, len(events))
	for i := range res.levels {
		res.levels[i] = target
	}
	for i, e := range events {
		if w.Level(s) >= target || s.Time-s0.Time >= horizon {
			break
		}
		if p, ok := w.eventPos(e); ok {
			if a := w.wandAction(s, p); a != nil {
				w.try(s, &acts, *a) // a caster fetches its wands where a capital is on its way (wand.go)
			}
			if !wandClasses[w.Char.Class] {
				if a := w.profAction(s, p); a != nil {
					w.try(s, &acts, *a) // the trainers for Skinning, Cooking, Leatherworking on the way (camp.go)
				}
			}
		}
		res.levels[i] = w.Level(s)
		if q := w.ByID[e.Quest]; q != nil && s0.Status[q.Index] == done {
			continue // done before this walk began
		}
		if e.Op == "level" {
			grindTo(e.Level)
			continue
		}
		if e.Op == "dungeon" {
			d := w.dungeonOf[e.Dungeon]
			if d == nil {
				res.Skipped["dungeon not in this route"]++
				continue
			}
			if w.Level(s) < d.Min {
				grindTo(d.Min)
			}
			// as the planner's dungeon block: its quests on the way, the run, the hand-ins on this continent
			w.pickUpDungeonQuests(s, &acts, d)
			if !w.try(s, &acts, Action{Op: "dungeon", Dungeon: d.Key, At: d.Entrance}) {
				res.Skipped["dungeon run not possible"]++
				continue
			}
			w.handInDungeonQuests(s, &acts, d)
			continue
		}
		miss := func(reason, detail string) {
			res.Skipped[reason]++
			res.Missed = append(res.Missed, GuideMiss{Op: e.Op, Quest: e.Quest, Reason: reason, Detail: detail})
		}
		q := w.ByID[e.Quest]
		if q == nil {
			detail, known := w.Excluded[e.Quest]
			if !known {
				detail = "not in the dataset"
			}
			miss("quest not available", detail)
			continue
		}
		switch e.Op {
		case "accept":
			if s.Status[q.Index] == notTaken && w.Level(s) < q.MinLevel && q.MinLevel <= target {
				if q.Dungeon != nil {
					// a dungeon quest is not worth grinding for: the dungeon block takes it on the way when its level is there
					miss("cannot accept", fmt.Sprintf("dungeon quest from level %d", q.MinLevel))
					continue
				}
				grindTo(q.MinLevel) // a follower grinds to the level the next quest needs
			}
			if len(q.StartSources) > 0 && s.Status[q.Index] == notTaken && w.CanAccept(s, q) == nil {
				// started by an item: a guide means "when it drops" (a rare's drop is luck), so a follower does not farm
				// it longer than guideStartItem
				c := s.clone()
				if err := w.Apply(c, Action{Op: "accept", Quest: q.ID, At: w.startAt(s, q)}); err == nil && c.Time-s.Time > guideStartItem {
					miss("cannot accept", fmt.Sprintf("its starting item takes %.0f min to farm", (c.Time-s.Time)/60))
					continue
				}
			}
			if s.Status[q.Index] != notTaken || w.CanAccept(s, q) != nil || !w.try(s, &acts, Action{Op: "accept", Quest: q.ID, At: w.startAt(s, q)}) {
				detail := ""
				if err := w.CanAccept(s, q); err != nil {
					detail = err.Error()
				} else if s.Status[q.Index] != notTaken {
					detail = "already taken or done"
				}
				miss("cannot accept", detail)
			}
		case "complete", "turnin":
			if s.Status[q.Index] != active {
				miss(e.Op+": not in the log", "")
				continue
			}
			if w.Opt.QuestAtLevel && w.Level(s) < q.Level && q.Level <= target {
				grindTo(q.Level) // a melee follower works a quest from its level on
			}
			if !w.workLog(s, &acts, []*Quest{q}) {
				// the guide comes too early for this character (a melee character fights at most one level up): it
				// levels first, as a route that reaches the area later
				n := w.levelFor(s, q, target)
				if n == 0 || (n > w.Level(s)+guideLevelUp && !w.keepClassQuest(q.ID, target)) { // a required class quest is always done
					miss(e.Op+": objective not possible", "")
					continue
				}
				grindTo(n)
				if !w.workLog(s, &acts, []*Quest{q}) {
					miss(e.Op+": objective not possible", "")
					continue
				}
			}
			if e.Op == "turnin" {
				if w.try(s, &acts, Action{Op: "turnin", Quest: q.ID, At: q.EndPos}) {
					res.Quests++
				} else {
					miss("turnin failed", "")
				}
			}
		default:
			return nil, fmt.Errorf("unknown guide event %q", e.Op)
		}
	}
	if s.Time-s0.Time < horizon {
		grindTo(target)
	}
	if len(acts) == 0 {
		return nil, fmt.Errorf("nothing left to do in this guide")
	}
	if fast {
		r := &Result{Character: w.Char, Seconds: round1(s.Time - s0.Time), Level: w.Level(s), XP: s.XP, Steps: make([]Step, len(acts))}
		for i, a := range acts {
			r.Steps[i].Action = a
		}
		res.Result = r
		return res, nil
	}
	r, err := w.replayFrom(s0, acts)
	if err != nil {
		return nil, err
	}
	res.Result = r
	return res, nil
}

// levelFor is the lowest level up to the target at which every open objective of an active quest has a source the
// character may work, 0 when there is none.
func (w *World) levelFor(s *State, q *Quest, target int) int {
	for n := w.Level(s) + 1; n <= target; n++ {
		ok := true
		for oi, o := range q.Objectives {
			if s.Prog[q.Index][oi] >= float64(o.Count) {
				continue
			}
			found := false
			for _, src := range o.Sources {
				found = found || (!w.dangerousAt(n, src) && (!src.Skill || n >= campLevel))
			}
			if !found {
				ok = false
				break
			}
		}
		if ok {
			return n
		}
	}
	return 0
}

// WithOptions puts options into a route: of each option's route (its events, the route with it) the steps of its own
// quests, each after the step of the route it follows there (before the first when none), in the option's order.
func WithOptions(base []GuideEvent, options [][]GuideEvent, quests []map[int]bool) []GuideEvent {
	out := append([]GuideEvent(nil), base...)
	for k, opt := range options {
		pos := map[GuideEvent]int{}
		for i, e := range out {
			if _, seen := pos[e]; !seen {
				pos[e] = i
			}
		}
		at := -1
		var ins []struct {
			after int
			e     GuideEvent
		}
		for _, e := range opt {
			if quests[k][e.Quest] {
				ins = append(ins, struct {
					after int
					e     GuideEvent
				}{at, e})
				continue
			}
			if i, ok := pos[e]; ok {
				at = i
			}
		}
		// from the back, so the indexes before stay valid; equal anchors keep their order
		for i := len(ins) - 1; i >= 0; i-- {
			j := ins[i].after + 1
			out = append(out[:j], append([]GuideEvent{ins[i].e}, out[j:]...)...)
		}
	}
	return out
}
