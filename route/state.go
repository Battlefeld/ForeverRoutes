package route

import (
	"fmt"
	"math"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/xp"
)

// Quest status in a state.
const (
	notTaken byte = iota
	active
	done
)

// State is the character at one moment of the route.
type State struct {
	Pos                  dataset.Point
	XP                   int
	Time                 float64
	Status               []byte            // per quest index
	Prog                 map[int][]float64 // active quest index -> progress per objective
	Active               int               // quests in the log
	Known                uint64            // flight masters found (bit per network node)
	Bind                 dataset.Point     // where the hearthstone leads
	Hearth               float64           // time from which the hearthstone can be used again
	Travel, Combat, Talk float64
	Wait                 float64        // part of Combat spent waiting for respawns
	Assumed              int            // steps that rely on an assumed drop chance
	Dead                 map[int]dead   // site key -> mobs of that site still dead
	Deathskips           int            // deaths taken on purpose to travel
	Runs                 uint64         // dungeons run (bit per World.Dungeons index)
	Switches             int            // changes between Kalimdor and the Eastern Kingdoms
	via                  string         // how the last action travelled when not on foot (hearth, teleport, network, deathskip: <graveyard>)
	viaAt                *dataset.Point // where to die for that deathskip
}

// dead counts the mobs of one creature killed at its site, as of a time. They come back over the respawn time.
type dead struct {
	N  float64
	At float64
}

// NewState is a fresh character at the start position.
func (w *World) NewState() *State {
	return &State{Pos: w.Start, Bind: w.Start, Status: make([]byte, len(w.Quests)), Prog: map[int][]float64{}, Dead: map[int]dead{}}
}

func (s *State) clone() *State {
	c := *s
	c.Status = append([]byte(nil), s.Status...)
	c.Prog = make(map[int][]float64, len(s.Prog))
	for k, v := range s.Prog {
		c.Prog[k] = append([]float64(nil), v...)
	}
	c.Dead = make(map[int]dead, len(s.Dead))
	for k, v := range s.Dead {
		if s.Time-v.At < 900 { // everything is back after at most a quarter hour; forget older entries
			c.Dead[k] = v
		}
	}
	return &c
}

// Level of the character in this state.
func (w *World) Level(s *State) int {
	l, _ := xp.Level(w.Data.LevelXP, s.XP)
	return l
}

// Action is one step of a route. Op: accept, turnin, work (objective Obj of Quest at source NPC/Object until done),
// grind (Kills of NPC at its site).
type Action struct {
	Op     string `json:"op"`
	Quest  int    `json:"quest,omitempty"`
	Obj    int    `json:"obj,omitempty"`
	NPC    int    `json:"npc,omitempty"`
	Object int    `json:"object,omitempty"`
	Kills  int    `json:"kills,omitempty"`
	// Dungeon: the dungeon a "dungeon" step runs with the group (key of Dataset.Dungeons)
	Dungeon string        `json:"dungeon,omitempty"`
	At      dataset.Point `json:"at"`
}

// RouteError explains why an action is not possible in a state.
type RouteError struct{ Reason string }

func (e *RouteError) Error() string { return e.Reason }

func fail(format string, a ...any) error { return &RouteError{fmt.Sprintf(format, a...)} }

func (w *World) move(s *State, to dataset.Point) error {
	t, how := w.travel(s, s.Pos, to)
	if math.IsInf(t, 1) {
		return fail("unreachable position")
	}
	from := s.Pos
	if from.Map != to.Map && from.Map <= 1 && to.Map <= 1 {
		s.Switches++
	}
	switch how {
	case byHearth:
		s.Hearth = s.Time + hearthCooldown
		from = s.Bind
		s.via = "hearth"
	case byTeleport:
		from = moonglade
		s.via = "teleport"
		w.discoverAlong(s, from, to)
	case byDeath:
		if _, g, die := w.deathskipAt(s, from, to, math.Inf(1)); g >= 0 {
			from = w.deaths.graves[g].At
			s.via = "deathskip: " + w.deaths.graves[g].Name
			s.viaAt = &die
		}
		s.Deathskips++
		w.discoverAlong(s, from, to)
	case byNetwork:
		if s.via == "" {
			s.via = "network"
		}
	case byWalk:
		w.discoverAlong(s, from, to)
	}
	s.Time += t
	s.Travel += t
	s.Pos = to
	w.discover(s)
	return nil
}

// bindNear binds the hearthstone at an inn within reach of the character, as routes do in each new quest hub.
func (w *World) bindNear(s *State) {
	for _, in := range w.inns {
		if distance(s.Pos, in) <= bindRadius && distance(s.Bind, in) > 1 {
			s.Bind = in
			s.Time += w.Opt.InteractSecs
			s.Talk += w.Opt.InteractSecs
			return
		}
	}
}

// CanAccept tells whether quest q can be taken now (ignoring where the character stands).
func (w *World) CanAccept(s *State, q *Quest) error {
	switch {
	case s.Status[q.Index] != notTaken:
		return fail("quest %d already taken", q.ID)
	case w.Level(s) < q.MinLevel:
		return fail("quest %d needs level %d", q.ID, q.MinLevel)
	case s.Active >= w.Opt.QuestLogCap:
		return fail("quest log full")
	}
	for _, p := range q.PreAll {
		if pq := w.ByID[p]; pq == nil || s.Status[pq.Index] != done {
			return fail("quest %d needs quest %d first", q.ID, p)
		}
	}
	if len(q.PreAny) > 0 {
		ok := false
		for _, p := range q.PreAny {
			if pq := w.ByID[p]; pq != nil && s.Status[pq.Index] == done {
				ok = true
			}
		}
		if !ok {
			return fail("quest %d needs one of its prerequisites", q.ID)
		}
	}
	if q.Exclusive > 0 {
		for _, o := range w.Quests {
			if o != q && o.Exclusive == q.Exclusive && s.Status[o.Index] != notTaken {
				return fail("quest %d excludes quest %d", o.ID, q.ID)
			}
		}
	}
	if q.Breadcrumb > 0 {
		if tq := w.ByID[q.Breadcrumb]; tq != nil && s.Status[tq.Index] != notTaken {
			return fail("breadcrumb %d: its quest is already taken", q.ID)
		}
	}
	return nil
}

func (w *World) complete(s *State, q *Quest) bool {
	p := s.Prog[q.Index]
	for i, o := range q.Objectives {
		if p == nil || p[i] < float64(o.Count)-1e-9 {
			return false
		}
	}
	return true
}

// mobTime is the time per kill at a site: fight, rest, walk to the next mob.
func (w *World) mobTime(s *State, n *dataset.NPC, site Site) float64 {
	f, r := w.Opt.Profile.KillSeconds(w.Char, w.Level(s), n)
	return f + r + site.Spacing*w.Opt.PathFactor/w.Opt.RunSpeed
}

func mobLevel(n *dataset.NPC) int { return (n.MinLevel + n.MaxLevel + 1) / 2 }

// kill applies k kills of creature n (or k uses of object obj) at site: time, XP, and the credit for every quest in the
// log that counts this creature or gets items from it.
// killSeconds is how long k kills (or uses of an object) take at a site from s, and the part of it spent waiting
// for respawns: mobs still dead from earlier visits are not there, and more kills than living mobs means waiting.
// after is how many of the site's mobs are dead when done.
func (w *World) killSeconds(s *State, n *dataset.NPC, site Site, k float64) (t, wait, after float64) {
	per := 0.0
	if n != nil {
		per = w.mobTime(s, n, site)
	} else {
		per = 4 + site.Spacing*w.Opt.PathFactor/w.Opt.RunSpeed // using an object
	}
	t = per * k
	if n == nil || site.Spawns <= 0 {
		return t, 0, 0
	}
	respawn := float64(n.Respawn)
	if respawn <= 0 {
		respawn = 300
	}
	spawns := float64(site.Spawns)
	gone := 0.0
	if d, ok := s.Dead[site.Key]; ok {
		gone = d.N * math.Max(0, 1-(s.Time-d.At)/respawn)
	}
	alive := spawns - gone
	after = gone + k
	if extra := k - alive; extra > 0 {
		cycles := math.Ceil(extra / spawns)
		if cycle := per * spawns; respawn > cycle {
			wait = cycles * (respawn - cycle)
			t += wait
		}
		after = spawns
	}
	return t, wait, math.Min(spawns, after)
}

func (w *World) kill(s *State, n *dataset.NPC, obj int, site Site, k float64) {
	if k <= 0 {
		return
	}
	t, wait, after := w.killSeconds(s, n, site, k)
	s.Wait += wait
	if n != nil && site.Spawns > 0 {
		s.Dead[site.Key] = dead{N: after, At: s.Time + t}
	}
	s.Time += t
	s.Combat += t
	if n != nil {
		whole := int(math.Ceil(k - 1e-9))
		for i := 0; i < whole; i++ {
			gain := xp.Kill(w.Level(s), mobLevel(n), n.Rank == 1)
			if float64(i) >= k-1 { // the last, partial kill counts in part (expected value)
				gain = int(math.Round(float64(gain) * (k - float64(i))))
			}
			s.XP += gain
		}
	}
	// credit for the log
	for qi, prog := range s.Prog {
		q := w.Quests[qi]
		for oi, o := range q.Objectives {
			if prog[oi] >= float64(o.Count) {
				continue
			}
			for _, src := range o.Sources {
				if (n != nil && src.NPC == n) || (n == nil && src.Object == obj && obj != 0) {
					if o.Kind == "kill" && n != nil || o.Kind == "object" || o.Kind == "item" {
						prog[oi] = math.Min(float64(o.Count), prog[oi]+k*src.Chance)
					}
					break
				}
			}
		}
	}
}

// Apply performs one action; the state is changed only when the action is possible.
func (w *World) Apply(s *State, a Action) error {
	switch a.Op {
	case "accept":
		q := w.ByID[a.Quest]
		if q == nil {
			return fail("quest %d not available to this character", a.Quest)
		}
		if err := w.CanAccept(s, q); err != nil {
			return err
		}
		if q.StartInside {
			return fail("quest %d is given inside %s", q.ID, q.Dungeon.Name)
		}
		if len(q.StartSources) > 0 {
			// started by an item: kill or loot at the chosen spawn group until it drops
			src := nearestSource(q.StartSources, a.At)
			if w.tooDangerous(s, *src) {
				return fail("quest %d: its item is found among creatures too high level", q.ID)
			}
			if err := w.move(s, src.Site.Center); err != nil {
				return err
			}
			if src.Assumed {
				s.Assumed++
			}
			w.kill(s, src.NPC, src.Object, src.Site, 1/src.Chance)
		} else if err := w.move(s, q.StartPos); err != nil {
			return err
		}
		s.Status[q.Index] = active
		s.Active++
		s.Prog[q.Index] = make([]float64, len(q.Objectives))
		s.Time += w.Opt.InteractSecs
		s.Talk += w.Opt.InteractSecs
		w.bindNear(s)
	case "turnin":
		q := w.ByID[a.Quest]
		if q == nil || s.Status[q.Index] != active {
			return fail("quest %d is not in the log", a.Quest)
		}
		if !w.complete(s, q) {
			return fail("quest %d is not complete", a.Quest)
		}
		if q.EndInside {
			return fail("quest %d is handed in inside %s", q.ID, q.Dungeon.Name)
		}
		if err := w.move(s, q.EndPos); err != nil {
			return err
		}
		s.XP += xp.Quest(q.XP, w.Level(s), q.Level)
		s.Status[q.Index] = done
		s.Active--
		delete(s.Prog, q.Index)
		s.Time += w.Opt.InteractSecs
		s.Talk += w.Opt.InteractSecs
		w.bindNear(s)
	case "work":
		q := w.ByID[a.Quest]
		if q == nil || s.Status[q.Index] != active {
			return fail("quest %d is not in the log", a.Quest)
		}
		if a.Obj < 0 || a.Obj >= len(q.Objectives) {
			return fail("quest %d has no objective %d", a.Quest, a.Obj)
		}
		o := q.Objectives[a.Obj]
		if w.Opt.QuestAtLevel && w.Level(s) < q.Level {
			return fail("quest %d is level %d: worked from that level on", q.ID, q.Level)
		}
		// the source is the creature or object; of its spawn groups the one at the action's position
		var src *Source
		bd := math.Inf(1)
		for i := range o.Sources {
			if (a.NPC != 0 && o.Sources[i].NPC != nil && o.Sources[i].NPC.ID == a.NPC) || (a.Object != 0 && o.Sources[i].Object == a.Object) ||
				(a.NPC == 0 && a.Object == 0 && o.Sources[i].Gather) {
				if d := distance(o.Sources[i].Site.Center, a.At); d < bd {
					src, bd = &o.Sources[i], d
				}
			}
		}
		if src == nil {
			return fail("quest %d objective %d: unknown source", a.Quest, a.Obj)
		}
		if w.tooDangerous(s, *src) {
			return fail("quest %d objective %d: too high level there", a.Quest, a.Obj)
		}
		left := float64(o.Count) - s.Prog[q.Index][a.Obj]
		if src.Assumed {
			s.Assumed++
		}
		if src.Talk {
			if err := w.move(s, src.Site.Center); err != nil {
				return err
			}
			s.Time += w.Opt.InteractSecs
			s.Talk += w.Opt.InteractSecs
			s.Prog[q.Index][a.Obj] = float64(o.Count)
			return nil
		}
		if left <= 0 {
			return fail("quest %d objective %d already done", a.Quest, a.Obj)
		}
		if err := w.move(s, src.Site.Center); err != nil {
			return err
		}
		w.kill(s, src.NPC, src.Object, src.Site, left/src.Chance)
		s.Prog[q.Index][a.Obj] = float64(o.Count)
	case "grind":
		n := w.NPC[a.NPC]
		var farm *Farm
		bd := math.Inf(1)
		for i := range w.Farms {
			if w.Farms[i].NPC == n {
				if d := distance(w.Farms[i].Site.Center, a.At); d < bd {
					farm, bd = &w.Farms[i], d
				}
			}
		}
		if farm == nil || a.Kills <= 0 {
			return fail("no farm for creature %d", a.NPC)
		}
		if mobLevel(n) > w.Level(s)+w.Opt.MaxMobAbove {
			return fail("%s is too high level", n.Name)
		}
		if err := w.move(s, farm.Site.Center); err != nil {
			return err
		}
		w.kill(s, n, 0, farm.Site, float64(a.Kills))
	case "dungeon":
		d := w.dungeonOf[a.Dungeon]
		if d == nil {
			return fail("dungeon %q not part of this route", a.Dungeon)
		}
		return w.run(s, d)
	default:
		return fail("unknown action %q", a.Op)
	}
	return nil
}

// nearestSource is the source whose spawn group is closest to p.
func nearestSource(src []Source, p dataset.Point) *Source {
	best, bd := &src[0], math.Inf(1)
	for i := range src {
		if d := distance(src[i].Site.Center, p); d < bd {
			best, bd = &src[i], d
		}
	}
	return best
}

// startAt is where to take quest q from state s: its giver, or for a quest started by an item the drop site that is
// quickest to reach and farm.
func (w *World) startAt(s *State, q *Quest) dataset.Point {
	if len(q.StartSources) == 0 {
		return q.StartPos
	}
	best, bt := q.StartPos, math.Inf(1)
	for _, src := range q.StartSources {
		if w.tooDangerous(s, src) {
			continue
		}
		per := 4 + src.Site.Spacing*w.Opt.PathFactor/w.Opt.RunSpeed
		if src.NPC != nil {
			per = w.mobTime(s, src.NPC, src.Site)
		}
		if t := w.TravelTime(s, s.Pos, src.Site.Center) + per/src.Chance; t < bt {
			best, bt = src.Site.Center, t
		}
	}
	return best
}

// Step is one replayed action with the character after it.
type Step struct {
	Action
	Via   string         `json:"via,omitempty"`   // how the step travelled when not on foot: hearth, teleport (a druid's to Moonglade), network, deathskip: <graveyard>
	ViaAt *dataset.Point `json:"viaAt,omitempty"` // where to die for a deathskip
	Name  string         `json:"name"`
	Time  float64        `json:"time"`
	Level int            `json:"level"`
	XP    int            `json:"xp"`
}

// Result of a replay.
type Result struct {
	Character Character `json:"character"`
	Seconds   float64   `json:"seconds"`
	Travel    float64   `json:"travel"`
	Combat    float64   `json:"combat"`
	Talk      float64   `json:"talk"`
	Level     int       `json:"level"`
	XP        int       `json:"xp"`
	Quests    int       `json:"questsDone"`
	Profile   string    `json:"killProfile"`
	Assumed   int       `json:"assumedDropSteps"` // steps that rely on a drop chance the data does not give
	Deaths    int       `json:"deathskips"`
	Steps     []Step    `json:"steps"`
}

// Replay checks a route action by action and reports the time line.
func (w *World) Replay(actions []Action) (*Result, error) {
	return w.replayFrom(w.NewState(), actions)
}

// replayFrom replays the actions from a copy of s; times in the result count from s.
func (w *World) replayFrom(s0 *State, actions []Action) (*Result, error) {
	s := s0.clone()
	t0, tr0, c0, tk0 := s.Time, s.Travel, s.Combat, s.Talk
	res := &Result{Character: w.Char, Profile: w.Opt.Profile.Name()}
	d0 := s.Deathskips
	for i, a := range actions {
		s.via, s.viaAt = "", nil
		if err := w.Apply(s, a); err != nil {
			return nil, fmt.Errorf("step %d (%s %d): %w", i+1, a.Op, a.Quest, err)
		}
		res.Steps = append(res.Steps, Step{Action: a, Via: s.via, ViaAt: s.viaAt, Name: w.describe(a), Time: round1(s.Time - t0), Level: w.Level(s), XP: s.XP})
	}
	res.Deaths = s.Deathskips - d0
	res.Seconds, res.Travel, res.Combat, res.Talk = round1(s.Time-t0), round1(s.Travel-tr0), round1(s.Combat-c0), round1(s.Talk-tk0)
	res.Level, res.XP, res.Assumed = w.Level(s), s.XP, s.Assumed
	for _, st := range s.Status {
		if st == done {
			res.Quests++
		}
	}
	return res, nil
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }

func (w *World) describe(a Action) string {
	switch a.Op {
	case "accept", "turnin", "work":
		if q := w.ByID[a.Quest]; q != nil {
			if a.Op == "work" {
				o := q.Objectives[a.Obj]
				target := "where the guides find it"
				if a.NPC != 0 {
					target = w.NPC[a.NPC].Name
				} else if ob := w.Objects[a.Object]; ob != nil {
					target = ob.Name
				}
				return fmt.Sprintf("%s: %s %d × %s", q.Name, o.Kind, o.Count, target)
			}
			return q.Name
		}
	case "grind":
		if n := w.NPC[a.NPC]; n != nil {
			return fmt.Sprintf("%d × %s", a.Kills, n.Name)
		}
	case "dungeon":
		if d := w.dungeonOf[a.Dungeon]; d != nil {
			return d.Name + " (dungeon run)"
		}
	}
	return ""
}
