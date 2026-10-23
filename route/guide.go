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
		q := w.ByID[e.Quest]
		if q == nil {
			res.Skipped["quest not available"]++
			continue
		}
		switch e.Op {
		case "accept":
			if s.Status[q.Index] != notTaken || w.CanAccept(s, q) != nil || !w.try(s, &acts, Action{Op: "accept", Quest: q.ID, At: q.StartPos}) {
				res.Skipped["cannot accept"]++
			}
		case "complete", "turnin":
			if s.Status[q.Index] != active {
				res.Skipped[e.Op+": not in the log"]++
				continue
			}
			if !w.workLog(s, &acts, []*Quest{q}) {
				res.Skipped[e.Op+": objective not possible"]++
				continue
			}
			if e.Op == "turnin" {
				if w.try(s, &acts, Action{Op: "turnin", Quest: q.ID, At: q.EndPos}) {
					res.Quests++
				} else {
					res.Skipped["turnin failed"]++
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
