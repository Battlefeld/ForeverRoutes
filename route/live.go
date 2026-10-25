package route

import (
	"errors"
	"fmt"
	"math"
	"math/rand"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/xp"
)

// Live is the character as the addon reads it in the game, the start of a route planned while playing.
type Live struct {
	Level         int            `json:"level"`
	XP            int            `json:"xp"`  // into the current level (UnitXP)
	Pos           *dataset.Point `json:"pos"` // world position in yards and continent (C_Map.GetWorldPosFromMapPos)
	Completed     []int          `json:"completed"`
	Active        []LiveQuest    `json:"active"`
	Bind          *dataset.Point `json:"bind,omitempty"`          // where the hearthstone leads; unknown: here
	HearthIn      float64        `json:"hearthIn,omitempty"`      // seconds until the hearthstone is ready
	FlightMasters []int          `json:"flightMasters,omitempty"` // NPC ids of the flight masters the character knows
}

// LiveQuest is a quest in the log.
type LiveQuest struct {
	ID         int             `json:"id"`
	Ready      bool            `json:"ready,omitempty"` // all objectives done, waiting to be turned in
	Objectives []LiveObjective `json:"objectives,omitempty"`
}

// LiveObjective is one line of C_QuestLog.GetQuestObjectives.
type LiveObjective struct {
	Type string `json:"type"` // monster | item | object | event | ...
	Have int    `json:"have"`
	Need int    `json:"need"`
	Done bool   `json:"done,omitempty"`
}

// StateFrom turns the live character into a planning state. Notes say what could not be carried over: quests in the
// log the planner does not know (they still take a log slot) and objectives whose progress could not be matched (they
// count as not started, so the plan never assumes progress it cannot see).
func (w *World) StateFrom(l Live) (*State, []string, error) {
	if l.Level < 1 || l.Level > len(w.Data.LevelXP) {
		return nil, nil, fmt.Errorf("level %d out of range", l.Level)
	}
	if l.Pos == nil {
		return nil, nil, errors.New("position missing")
	}
	s := w.NewState()
	s.XP = xp.Total(w.Data.LevelXP, l.Level) + max(0, l.XP)
	s.Pos, s.Bind = *l.Pos, *l.Pos
	if l.Bind != nil {
		s.Bind = *l.Bind
	}
	s.Hearth = max(0, l.HearthIn)
	var notes []string
	for _, id := range l.Completed {
		if q := w.ByID[id]; q != nil {
			s.Status[q.Index] = done
		}
	}
	for _, lq := range l.Active {
		s.Active++
		q := w.ByID[lq.ID]
		if q == nil {
			notes = append(notes, fmt.Sprintf("quest %d in the log is not plannable for this character (%s)", lq.ID, w.Excluded[lq.ID]))
			continue
		}
		s.Status[q.Index] = active
		prog := make([]float64, len(q.Objectives))
		if lq.Ready {
			for i, o := range q.Objectives {
				prog[i] = float64(o.Count)
			}
		} else if miss := matchObjectives(q, lq.Objectives, prog); miss > 0 {
			notes = append(notes, fmt.Sprintf("%s: progress of %d objective(s) not matched, counted from zero", q.Name, miss))
		}
		s.Prog[q.Index] = prog
	}
	for _, npc := range l.FlightMasters {
		if i, ok := w.net.byNPC[npc]; ok && w.net.fm[i] {
			s.Known |= 1 << i
		}
	}
	w.discover(s)
	// a dungeon counts as run when a quest done inside it is done (the game does not tell runs apart)
	for _, d := range w.Dungeons {
		for _, q := range d.Quests {
			inside := q.StartInside || q.EndInside
			for _, o := range q.Objectives {
				inside = inside || o.Inside
			}
			if inside && s.Status[q.Index] == done {
				s.Runs |= 1 << d.Index
				break
			}
		}
	}
	return s, notes, nil
}

// matchObjectives carries the game's progress over to the quest's objectives. The game lists them in its own order,
// so each one goes to an objective of the same required count, of the same kind if there is one. It returns how many
// game objectives found no partner.
func matchObjectives(q *Quest, game []LiveObjective, prog []float64) int {
	kinds := map[string]string{"monster": "kill", "item": "item", "object": "object"}
	used := make([]bool, len(q.Objectives))
	miss := 0
	for _, g := range game {
		best := -1
		for i, o := range q.Objectives {
			if used[i] || o.Count != g.Need {
				continue
			}
			if best < 0 || (o.Kind == kinds[g.Type] && q.Objectives[best].Kind != kinds[g.Type]) {
				best = i
			}
		}
		if best < 0 {
			if g.Have > 0 || g.Done {
				miss++
			}
			continue
		}
		used[best] = true
		have := float64(g.Have)
		if g.Done {
			have = float64(q.Objectives[best].Count)
		}
		prog[best] = math.Min(have, float64(q.Objectives[best].Count))
	}
	return miss
}

// PlanAhead plans the next blocks from s until they fill the horizon (seconds of play) or reach the target level. It is
// made to be called again whenever the game moves on: each call decides from what the character has now. The result's
// times count from s.
func (w *World) PlanAhead(s *State, po PlanOptions, horizon float64) (*Result, error) {
	po = po.withDefaults()
	start := s.clone()
	cur := s.clone()
	var acts []Action
	rng := rand.New(rand.NewSource(po.Seed))
	for step := 0; step < po.MaxSteps && w.Level(cur) < po.TargetLevel && cur.Time-start.Time < horizon; step++ {
		options := w.options(cur, po.TargetLevel)
		if len(options) == 0 {
			break
		}
		pick := w.pick(cur, options, po, 0, rng)
		acts = append(acts, pick.actions...)
		cur = pick.state
	}
	if len(acts) == 0 {
		return nil, errors.New("nothing to do from here with this data")
	}
	return w.replayFrom(start, acts)
}

// StateAfter is the state a route's steps lead to from a fresh character.
func (w *World) StateAfter(steps []Step) (*State, error) {
	s := w.NewState()
	for i, st := range steps {
		s.via, s.viaAt = "", nil
		if err := w.Apply(s, st.Action); err != nil {
			return nil, fmt.Errorf("step %d (%s %d): %w", i+1, st.Op, st.Quest, err)
		}
	}
	return s, nil
}
