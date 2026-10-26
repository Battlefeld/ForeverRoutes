package route

import (
	"math"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/xp"
)

// Dungeon is a dungeon a route with dungeons may run (Options.Dungeons): its quests this character can do.
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
	// runClear: the share of the instance's trash a usual quest run kills (what is in the way).
	runClear = 0.4
	// groupShares: the group members who each need their own drops of a collection quest; the run's kills bring one
	// share, the others cost extra kills
	groupShares = 5
	// groupKill: one more trash kill for the group (seconds)
	groupKill = 15.0
	// maxExtraShare: the extra kills of collection quests take at most this share of the run's minutes
	maxExtraShare = 0.5
	// groupSearch: finding a group for the run, before it starts (seconds); groupSearchRole: for a class that can tank
	// or heal (warrior, paladin, shaman, priest, druid), whom groups look for: almost at once
	groupSearch     = 600.0
	groupSearchRole = 60.0
	// elitesPerMinute: elites per minute of a usual run where the instance's spawns are not known (Ragefire 6.6,
	// Shadowfang 3.3, Wailing Caverns 3.4, Deadmines 2.7, Blackfathom 4.8 in cmangos)
	elitesPerMinute = 3.5
	// pickupReach: a dungeon quest further than this from the last stop is left for another time (a quest across the
	// sea or at the other end of the continent costs more than it brings to this run).
	pickupReach = 900.0
	// pickupNear: a dungeon quest whose giver is this close (travel seconds) is taken on the way, long before the run.
	pickupNear = 600.0
	// pickupShare: what taking a dungeon quest early is worth to the planner, as a share of its XP (the run pays it).
	pickupShare = 0.5
)

// pickupMacros: taking a dungeon quest early, for a dungeon still to run (from three levels below it), when its giver
// is within pickupNear: on the way through a town, so the run later needs no detour for it. One block per quest; the planner weighs
// the detour.
func (w *World) pickupMacros(s0 *State) []*macro {
	var out []*macro
	l := w.Level(s0)
	for _, d := range w.Dungeons {
		if s0.Runs&(1<<d.Index) != 0 || l > d.Max || l < d.Min-3 {
			continue
		}
		for _, q := range d.Quests {
			if q.StartInside || !q.runnable() || s0.Status[q.Index] != notTaken || w.CanAccept(s0, q) != nil ||
				w.TravelTime(s0, s0.Pos, q.StartPos) > pickupNear {
				continue
			}
			s := s0.clone()
			var acts []Action
			if w.try(s, &acts, Action{Op: "accept", Quest: q.ID, At: q.StartPos}) {
				out = append(out, w.finish(s0, s, acts, "pickup"))
			}
		}
	}
	return out
}

// dungeons prepares the dungeons this character can reach (routes with dungeons only).
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

// dungeonQuest resolves a dungeon quest for a route with dungeons: a giver or turn-in without an open-world place (or placed
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
		ro := Objective{Kind: o.Kind, ID: o.ID, Count: o.Count, Inside: o.Inside, Chance: w.Opt.AssumedDrop}
		for _, dr := range o.Drops {
			if !dr.Unknown && dr.Chance/100 > 0 && (ro.Chance == w.Opt.AssumedDrop || dr.Chance/100 > ro.Chance) {
				ro.Chance = math.Min(1, dr.Chance/100)
			}
		}
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
		// trash unknown (a dungeon cmangos does not have): elites of the dungeon's middle level, as many per minute of
		// the run as the Classic dungeons of these levels hold (3 to 7, about 3.5)
		mid := (d.Min + d.Max + 1) / 2
		return int(math.Round(float64(xp.Kill(l, mid, true)) * groupShare * runClear * elitesPerMinute * d.Minutes))
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
	search := w.Opt.GroupSearch
	if !w.canTank(s) {
		search = groupSearch // a druid without bear form, a warrior without defensive stance: queued as damage
	}
	secs := d.Minutes*60 + search
	s.Time += secs
	s.Combat += secs
	trash := w.killXP(d, l)
	s.XP += trash + restedGain(s, trash) // rested experience doubles the run's kills too
	// collection quests: every member needs their own drops; the run's kills bring one share, the other four cost more
	// kills (and bring their XP). A single item (a boss's) everyone loots.
	extra := 0.0
	for _, q := range d.Quests {
		if s.Status[q.Index] != active && !(q.StartInside && q.runnable() && s.Status[q.Index] == notTaken && w.CanAccept(s, q) == nil) {
			continue
		}
		for i, o := range q.Objectives {
			if o.Inside && o.Kind == "item" && o.Count > 1 && o.Chance > 0 && (s.Prog[q.Index] == nil || s.Prog[q.Index][i] < float64(o.Count)) {
				extra += (groupShares - 1) * float64(o.Count) / o.Chance
			}
		}
	}
	if extra > 0 {
		// the drops of dungeon quests are made for groups: the extra kills take at most half a run more
		extra = math.Min(extra, d.Minutes*60*maxExtraShare/groupKill)
		s.Time += extra * groupKill
		s.Combat += extra * groupKill
		if n := d.trashKills(); n > 0 {
			s.XP += int(math.Round(float64(trash) / n * extra))
		}
	}
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
	w.pickUpDungeonQuests(s, &acts, d)
	if !w.try(s, &acts, Action{Op: "dungeon", Dungeon: d.Key, At: d.Entrance}) {
		return nil
	}
	w.handInDungeonQuests(s, &acts, d)
	return w.finish(s0, s, acts, "dungeon")
}

// pickUpDungeonQuests takes the dungeon's quests that are given outside on its continent and that the run finishes,
// nearest first, each within pickupReach of the last stop.
func (w *World) pickUpDungeonQuests(s *State, acts *[]Action, d *Dungeon) {
	for {
		var next *Quest
		bt := math.Inf(1)
		for _, q := range d.Quests {
			if q.StartInside || !q.runnable() || s.Status[q.Index] != notTaken || w.CanAccept(s, q) != nil || q.StartPos.Map != d.Entrance.Map {
				continue
			}
			if t := w.TravelTime(s, s.Pos, q.StartPos); t < bt && t <= pickupReach {
				next, bt = q, t
			}
		}
		if next == nil || !w.try(s, acts, Action{Op: "accept", Quest: next.ID, At: next.StartPos}) {
			return
		}
	}
}

// handInDungeonQuests turns in the finished dungeon quests on this continent, nearest first: this run's wherever they
// are handed in on the continent, other runs' within pickupReach (Ragefire's quest from Undercity when a later run
// brings the character to Lordaeron). A quest handed in across the sea waits until the route is there anyway.
func (w *World) handInDungeonQuests(s *State, acts *[]Action, d *Dungeon) {
	for {
		var next *Quest
		bt := math.Inf(1)
		for _, rd := range w.Dungeons {
			for _, q := range rd.Quests {
				if q.EndInside || s.Status[q.Index] != active || !w.complete(s, q) || q.EndPos.Map != s.Pos.Map {
					continue
				}
				t := w.TravelTime(s, s.Pos, q.EndPos)
				if rd != d && t > pickupReach {
					continue
				}
				if t < bt {
					next, bt = q, t
				}
			}
		}
		if next == nil || !w.try(s, acts, Action{Op: "turnin", Quest: next.ID, At: next.EndPos}) {
			return
		}
	}
}

// handInMacros: a finished dungeon quest left in the log (handed in across the sea) is turned in when the route comes
// by; one block per quest, the planner weighs the detour against its XP.
func (w *World) handInMacros(s0 *State) []*macro {
	var out []*macro
	for _, d := range w.Dungeons {
		for _, q := range d.Quests {
			if q.EndInside || s0.Status[q.Index] != active || !w.complete(s0, q) || q.EndPos.Map != s0.Pos.Map ||
				w.TravelTime(s0, s0.Pos, q.EndPos) > pickupNear {
				continue
			}
			s := s0.clone()
			var acts []Action
			if w.try(s, &acts, Action{Op: "turnin", Quest: q.ID, At: q.EndPos}) {
				out = append(out, w.finish(s0, s, acts, "handin"))
			}
		}
	}
	return out
}

// StartedInside lists the quests of a dungeon this character takes inside it (from an NPC or an item in there).
func (w *World) StartedInside(key string) []*Quest {
	d := w.dungeonOf[key]
	if d == nil {
		return nil
	}
	var out []*Quest
	for _, q := range d.Quests {
		if q.StartInside && q.runnable() {
			out = append(out, q)
		}
	}
	return out
}

// trashKills is how many mobs a usual run kills (the share runClear of the instance's spawns, else the estimate).
func (d *Dungeon) trashKills() float64 {
	n := 0
	for _, m := range d.Mobs {
		n += m.N
	}
	if n == 0 {
		return elitesPerMinute * d.Minutes * runClear
	}
	return float64(n) * runClear
}
