package route

import "fmt"

// GuideEvent is one step of a quest guide: accept, complete or turn in a quest, or reach a level.
type GuideEvent struct {
	Op    string `json:"op"` // accept, complete, turnin, level
	Quest int    `json:"quest,omitempty"`
	Level int    `json:"level,omitempty"`
}

// GuideResult is a guide walked with this model, for comparing guides with planned routes on equal terms.
type GuideResult struct {
	*Result
	Quests  int            `json:"guideQuests"`  // quests turned in
	Grind   float64        `json:"grindSeconds"` // grinding the guide asks for (level steps) and to the target level
	Skipped map[string]int `json:"skipped"`      // events that could not be followed, by reason
	Missed  []GuideMiss    `json:"missed"`       // each skipped quest event with its reason
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
func (w *World) FollowGuide(events []GuideEvent, target int) (*GuideResult, error) {
	s := w.NewState()
	var acts []Action
	res := &GuideResult{Skipped: map[string]int{}}
	grindTo := func(level int) {
		for w.Level(s) < level {
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
	for _, e := range events {
		if w.Level(s) >= target {
			break
		}
		if e.Op == "level" {
			grindTo(e.Level)
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
				grindTo(q.MinLevel) // a follower grinds to the level the next quest needs
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
			if !w.workLog(s, &acts, []*Quest{q}) {
				miss(e.Op+": objective not possible", "")
				continue
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
	grindTo(target)
	r, err := w.Replay(acts)
	if err != nil {
		return nil, err
	}
	res.Result = r
	return res, nil
}
