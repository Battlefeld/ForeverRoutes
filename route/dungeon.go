package route

import (
	"math"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/xp"
)

// Dungeon is a dungeon a group route may run (Options.Dungeons): its quests this character can do.
type Dungeon struct {
	*dataset.Dungeon
	Index  int
	Quests []*Quest
}

const (
	// insideRadius: objective sources this close to a dungeon's entrance are in its halls before or around the portal;
	// a run does them on the way.
	insideRadius = 600
	// groupShare: a full group of five shares a kill's XP; the group bonus is 1.4 (Classic party XP), each gets 0.28.
	groupShare = 1.4 / 5
	// runClear: the share of the instance's trash a usual quest run kills.
	runClear = 0.6
)

// dungeons prepares the dungeons this character can reach (group routes only).
func (w *World) dungeons() {
	if !w.Opt.Dungeons {
		return
	}
	w.dungeonOf = map[string]*Dungeon{}
	for i := range w.Data.Dungeons {
		d := &w.Data.Dungeons[i]
		if (d.Faction != "" && d.Faction != w.Faction) || !w.maps[d.Entrance.Map] || len(w.Dungeons) >= 64 {
			continue
		}
		rd := &Dungeon{Dungeon: d, Index: len(w.Dungeons)}
		w.Dungeons = append(w.Dungeons, rd)
		w.dungeonOf[d.Key] = rd
	}
}

// dungeonQuest resolves a dungeon quest for a group route: a giver or turn-in without an open-world place (or placed
// at the entrance) is inside; an objective without a solo source, or with sources only around the entrance, is done by
// the run.
func (w *World) dungeonQuest(q *dataset.Quest) (*Quest, string) {
	d := w.dungeonOf[q.Dungeon]
	if d == nil {
		return nil, "dungeon not reachable for this character"
	}
	rq := &Quest{Quest: q, Dungeon: d}
	var ok bool
	if rq.StartPos, rq.StartNPC, ok = w.refPos(q.Start, q.StartAt); !ok || dataset.AtEntrance(rq.StartPos, d.Entrance) {
		rq.StartPos, rq.StartNPC, rq.StartInside = d.Entrance, 0, true
	}
	if rq.EndPos, rq.EndNPC, ok = w.refPos(q.End, q.EndAt); !ok || dataset.AtEntrance(rq.EndPos, d.Entrance) {
		rq.EndPos, rq.EndNPC, rq.EndInside = d.Entrance, 0, true
	}
	if !w.maps[rq.StartPos.Map] || !w.maps[rq.EndPos.Map] {
		return nil, "other continent"
	}
	for _, o := range q.Objectives {
		ro := Objective{Kind: o.Kind, ID: o.ID, Count: o.Count, Inside: o.Inside}
		if !o.Inside {
			solo, _ := w.resolve(&dataset.Quest{ID: q.ID, Start: q.Start, End: q.End, StartAt: &rq.StartPos, EndAt: &rq.EndPos, Objectives: []dataset.Objective{o}})
			if solo != nil {
				ro.Sources = solo.Objectives[0].Sources
			}
			near := len(ro.Sources) > 0
			for _, src := range ro.Sources {
				near = near && distance(src.Site.Center, d.Entrance) <= insideRadius
			}
			ro.Inside = len(ro.Sources) == 0 || near
		}
		rq.Objectives = append(rq.Objectives, ro)
	}
	return rq, ""
}

// runnable: every objective is done by the run, so taking the quest before the run finishes it there.
func (q *Quest) runnable() bool {
	for _, o := range q.Objectives {
		if !o.Inside {
			return false
		}
	}
	return true
}

// killXP a run's trash gives one member of a full group at level l.
func (w *World) killXP(d *Dungeon, l int) int {
	if len(d.Mobs) == 0 {
		// trash unknown (a dungeon cmangos does not have): elite mobs of the dungeon's middle level, as many as a
		// usual Classic dungeon of its size
		mid := (d.Min + d.Max + 1) / 2
		return int(math.Round(float64(xp.Kill(l, mid, true)) * groupShare * runClear * 200))
	}
	v := 0.0
	for _, m := range d.Mobs {
		v += float64(xp.Kill(l, m.Level, m.Elite)*m.N) * groupShare * runClear
	}
	return int(math.Round(v))
}

// run plays a group run of a dungeon: at the entrance, quests given inside are taken (and their follow-ups), every
// objective of the log the run does is finished, quests handed in inside are turned in, and the trash gives its XP.
func (w *World) run(s *State, d *Dungeon) error {
	if s.Runs&(1<<d.Index) != 0 {
		return fail("%s already run", d.Name)
	}
	if l := w.Level(s); l < d.Min {
		return fail("%s needs level %d", d.Name, d.Min)
	}
	if err := w.move(s, d.Entrance); err != nil {
		return err
	}
	l := w.Level(s)
	secs := d.Minutes * 60
	s.Time += secs
	s.Combat += secs
	s.XP += w.killXP(d, l)
	for changed := true; changed; {
		changed = false
		for _, q := range d.Quests {
			switch st := s.Status[q.Index]; {
			case st == notTaken && q.StartInside && q.runnable() && w.CanAccept(s, q) == nil:
				s.Status[q.Index] = active
				s.Active++
				s.Prog[q.Index] = make([]float64, len(q.Objectives))
				s.Time += w.Opt.InteractSecs
				s.Talk += w.Opt.InteractSecs
				changed = true
			case st == active:
				for i, o := range q.Objectives {
					if o.Inside && s.Prog[q.Index][i] < float64(o.Count) {
						s.Prog[q.Index][i] = float64(o.Count)
						changed = true
					}
				}
				if q.EndInside && w.complete(s, q) {
					s.XP += xp.Quest(q.XP, w.Level(s), q.Level)
					s.Status[q.Index] = done
					s.Active--
					delete(s.Prog, q.Index)
					s.Time += w.Opt.InteractSecs
					s.Talk += w.Opt.InteractSecs
					changed = true
				}
			}
		}
	}
	s.Runs |= 1 << d.Index
	return nil
}

// dungeonMacro: take the dungeon's quests that are given outside and that the run finishes, nearest first, run the
// dungeon, then hand in what is handed in outside, nearest first.
func (w *World) dungeonMacro(s0 *State, d *Dungeon) *macro {
	if l := w.Level(s0); s0.Runs&(1<<d.Index) != 0 || l < d.Min || l > d.Max {
		return nil
	}
	s := s0.clone()
	var acts []Action
	for {
		var next *Quest
		bt := math.Inf(1)
		for _, q := range d.Quests {
			if q.StartInside || !q.runnable() || s.Status[q.Index] != notTaken || w.CanAccept(s, q) != nil {
				continue
			}
			if t := w.TravelTime(s, s.Pos, q.StartPos); t < bt {
				next, bt = q, t
			}
		}
		if next == nil || !w.try(s, &acts, Action{Op: "accept", Quest: next.ID, At: next.StartPos}) {
			break
		}
	}
	if !w.try(s, &acts, Action{Op: "dungeon", Dungeon: d.Key, At: d.Entrance}) {
		return nil
	}
	for {
		var next *Quest
		bt := math.Inf(1)
		for _, q := range d.Quests {
			if q.EndInside || s.Status[q.Index] != active || !w.complete(s, q) {
				continue
			}
			if t := w.TravelTime(s, s.Pos, q.EndPos); t < bt {
				next, bt = q, t
			}
		}
		if next == nil || !w.try(s, &acts, Action{Op: "turnin", Quest: next.ID, At: next.EndPos}) {
			break
		}
	}
	return w.finish(s0, s, acts, "dungeon")
}
