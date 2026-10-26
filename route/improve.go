package route

import (
	"math"
	"slices"
	"sort"
	"time"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Improve makes a guide faster under the model, the way Polestar improves its shipped orders: it starts from the
// guide's own steps and keeps only local edits that reach the target level sooner when the whole guide is walked
// again (FollowGuide): leaving out a quest or a level step, or putting in a quest the guide passes by (taken, done
// and turned in where the guide comes closest to its giver). So the result is never slower than the guide it starts
// from. The budget bounds the time spent searching.
func (w *World) Improve(events []GuideEvent, target int, budget time.Duration) ([]GuideEvent, *GuideResult, error) {
	events = w.withClassQuests(events, target)
	best, err := w.FollowGuide(events, target)
	if err != nil {
		return nil, nil, err
	}
	cur := append([]GuideEvent(nil), events...)
	deadline := time.Now().Add(budget)
	// the required class chains the route finishes stay finished: an edit that loses one is no improvement
	var goals []int
	for _, c := range w.ClassChains(target) {
		if c.Tier == "required" && handsIn(best, c.Goal.ID) {
			goals = append(goals, c.Goal.ID)
		}
	}
	tankFirst := w.tankFirst(best)
	try := func(cand []GuideEvent) bool {
		r, err := w.FollowGuide(cand, target)
		if err != nil || r.Level < target || r.Seconds >= best.Seconds-1 {
			return false
		}
		for _, g := range goals {
			if !handsIn(r, g) {
				return false
			}
		}
		if tankFirst && !w.tankFirst(r) {
			return false
		}
		cur, best = cand, r
		return true
	}
	for pass := 0; time.Now().Before(deadline); pass++ {
		improved := false
		// leave out: each quest the guide does, each level step
		for _, q := range questsOf(cur) {
			if time.Now().After(deadline) {
				break
			}
			if w.keepClassQuest(q, target) {
				continue // the first class quests teach abilities: never left out for time
			}
			if try(without(cur, func(e GuideEvent) bool { return e.Quest == q })) {
				improved = true
			}
		}
		for i := 0; i < len(cur) && time.Now().Before(deadline); i++ {
			if cur[i].Op == "level" || (cur[i].Op == "dungeon" && Runs(cur) > 1) {
				// a route with dungeons keeps at least one run: whoever asks for dungeons runs the one that costs least
				if try(append(append([]GuideEvent(nil), cur[:i]...), cur[i+1:]...)) {
					improved = true
					i--
				}
			}
		}
		// put in: usable quests the guide does not do, at the step whose place is closest to the giver
		in := map[int]bool{}
		for _, q := range questsOf(cur) {
			in[q] = true
		}
		for _, q := range w.Quests {
			if time.Now().After(deadline) {
				break
			}
			if in[q.ID] || q.MinLevel > target || q.Level > target+3 || q.Repeatable {
				continue
			}
			i, d := w.closestStep(cur, q.StartPos)
			if d > 250 {
				continue
			}
			block := []GuideEvent{{Op: "accept", Quest: q.ID}, {Op: "complete", Quest: q.ID}, {Op: "turnin", Quest: q.ID}}
			cand := append(append(append([]GuideEvent(nil), cur[:i+1]...), block...), cur[i+1:]...)
			if try(cand) {
				improved = true
				in[q.ID] = true
			}
		}
		if !improved {
			break
		}
	}
	return cur, best, nil
}

// WithDungeons puts dungeon runs into a route (a guide has none): each dungeon at the one of twenty places spread over
// the route that brings it fastest to the target, when one does. When none does, the run that costs least goes in:
// a route with dungeons runs at least one. A run takes the dungeon's quests on the way and hands them in after, as
// the planner's dungeon block.
// runPlaces: the cheapest places of a run whose quests are then put where the route passes their givers.
const runPlaces = 3

func (w *World) WithDungeons(events []GuideEvent, target int) []GuideEvent {
	best, err := w.FollowGuide(events, target)
	if err != nil {
		return events
	}
	cur := events
	var least []GuideEvent
	leastT := math.Inf(1)
	// dungeon after dungeon, again while one more pays: each tried at every step of the route that has its level
	// (up to dungeonWindow levels above its lowest), where it costs least
	for changed := true; changed; {
		changed = false
		for _, d := range w.Dungeons {
			have := false
			for _, e := range cur {
				have = have || (e.Op == "dungeon" && e.Dungeon == d.Key)
			}
			if have || d.Min > target {
				continue
			}
			// the dungeon's quest chains, so its quests can be taken on the way: a run is worth its quests far more than
			// its trash (the Deadmines' Defias Brotherhood after the Westfall chain before it)
			with := cur
			levels := best.levels
			if pre := w.dungeonPrereqs(d, cur); len(pre) > 0 {
				with = w.withQuestsIn(cur, pre, target, dungeonWindow)
				r, err := w.FollowGuide(with, target)
				if err != nil {
					continue
				}
				levels = r.levels
			}
			var pick []GuideEvent
			type place struct {
				events []GuideEvent
				r      *GuideResult
			}
			var places []place // the run's cheapest places, its quests then taken where the route passes their givers
			for i := 0; i <= len(with); i++ {
				l := target
				if i < len(levels) {
					l = levels[i]
				}
				if l < d.Min || l > d.Min+dungeonWindow {
					continue
				}
				cand := append(append(append([]GuideEvent(nil), with[:i]...), GuideEvent{Op: "dungeon", Dungeon: d.Key}), with[i:]...)
				r, err := w.FollowGuide(cand, target)
				if err != nil || r.Level < target || !w.tankFirst(r) {
					continue // a tank class runs its first dungeon with its tank form or stance
				}
				if r.Seconds < best.Seconds-1 {
					best, pick = r, cand
				}
				if r.Seconds < leastT {
					least, leastT = cand, r.Seconds
				}
				places = append(places, place{cand, r})
			}
			sort.Slice(places, func(a, b int) bool { return places[a].r.Seconds < places[b].r.Seconds })
			for k := 0; k < len(places) && k < runPlaces; k++ {
				if e, r := w.withDungeonPickups(places[k].events, d, target, places[k].r); r.Seconds < best.Seconds-1 && w.tankFirst(r) {
					best, pick = r, e
				}
			}
			if pick != nil {
				cur, changed = pick, true
			}
		}
	}
	if Runs(cur) == 0 && least != nil {
		return least // no run made it faster: until then cur is the route given, least one run put into it
	}
	return w.MergeTrips(cur, target)
}

// dungeonPrereqs: the quests before a dungeon's quests (taken outside, on its continent) that the route does not do,
// prerequisites first: the chains that open them.
func (w *World) dungeonPrereqs(d *Dungeon, events []GuideEvent) []*Quest {
	in := map[int]bool{}
	for _, q := range questsOf(events) {
		in[q] = true
	}
	var out []*Quest
	seen := map[int]bool{}
	var add func(id int)
	add = func(id int) {
		q := w.ByID[id]
		if q == nil || seen[id] {
			return
		}
		seen[id] = true
		for _, p := range q.PreAll {
			add(p)
		}
		if len(q.PreAny) > 0 {
			pick := 0
			for _, p := range q.PreAny {
				if in[p] {
					pick = -1 // the route does one already
					break
				}
				if pick == 0 && w.ByID[p] != nil {
					pick = p
				}
			}
			if pick > 0 {
				add(pick)
			}
		}
		if !in[id] && q.Dungeon == nil {
			out = append(out, q)
		}
	}
	for _, q := range d.Quests {
		if q.StartInside || q.StartPos.Map != d.Entrance.Map {
			continue
		}
		for _, p := range q.PreAll {
			add(p)
		}
		for _, p := range q.PreAny {
			if in[p] {
				break
			}
			if w.ByID[p] != nil {
				add(p)
				break
			}
		}
	}
	return out
}

// dungeonWindow: how many levels above its lowest a dungeon is tried in a route.
const dungeonWindow = 6

// Runs counts the dungeon runs of a route.
// RunKeys: the dungeons a route runs, in order.
func RunKeys(events []GuideEvent) []string {
	var out []string
	for _, e := range events {
		if e.Op == "dungeon" {
			out = append(out, e.Dungeon)
		}
	}
	return out
}

func Runs(events []GuideEvent) int {
	n := 0
	for _, e := range events {
		if e.Op == "dungeon" {
			n++
		}
	}
	return n
}

// keepClassQuest: a quest of a required class chain (classQuests). Never left out for time.
func (w *World) keepClassQuest(id, target int) bool {
	if w.required == nil || w.requiredTarget != target {
		w.required, w.requiredTarget = map[int]bool{}, target
		for _, q := range w.classQuests(target) {
			w.required[q.ID] = true
		}
	}
	return w.required[id]
}

// homeCapital: each race's capital. A race's home is its start area and its capital; a class quest chain that begins
// in another race's home is that race's version (the data lets every race of the side take them).
var homeCapital = map[string]dataset.Point{
	"Human": {Map: 0, X: -8833, Y: 622}, "Dwarf": {Map: 0, X: -4918, Y: -956}, "Gnome": {Map: 0, X: -4918, Y: -956},
	"NightElf": {Map: 1, X: 9947, Y: 2482}, "Orc": {Map: 1, X: 1630, Y: -4400}, "Troll": {Map: 1, X: 1630, Y: -4400},
	"Tauren": {Map: 1, X: -1250, Y: 100}, "Undead": {Map: 0, X: 1630, Y: 240}, "Scourge": {Map: 0, X: 1630, Y: 240},
}

const (
	homeStart = 2500.0 // yards around a race's first quest giver that are its home
	homeCity  = 900.0  // yards around its capital
)

// home: the races whose home p lies in.
func (w *World) home(p dataset.Point) map[string]bool {
	out := map[string]bool{}
	for race, id := range StartNPC {
		if race == "Scourge" {
			continue
		}
		if n := w.NPC[id]; n != nil {
			for _, sp := range n.Spawns {
				if distance(sp, p) <= homeStart {
					out[race] = true
				}
			}
		}
		if c, ok := homeCapital[race]; ok && distance(c, p) <= homeCity {
			out[race] = true
		}
	}
	return out
}

// classQuests are the class quests every route up to the target does: the required chains (ClassChains), in order.
func (w *World) classQuests(target int) []*Quest {
	var out []*Quest
	seen := map[int]bool{}
	for _, c := range w.ClassChains(target) {
		if c.Tier != "required" {
			continue
		}
		for _, q := range c.Quests {
			if !seen[q.ID] {
				seen[q.ID] = true
				out = append(out, q)
			}
		}
	}
	return out
}

// classWindow: a class quest is put in within this many levels of when it can be taken (its spell or pet helps from
// then on; a detour is worth it).
const classWindow = 2

// withClassQuests puts in the class quests up to the target the route lacks (classQuests; a guide leaves many out),
// chain after chain in order: each after its prerequisites, within classWindow levels of when it can be taken, where
// the route comes closest to its giver. Where the route never has the level, at its end.
func (w *World) withClassQuests(events []GuideEvent, target int) []GuideEvent {
	out := events
	for _, c := range w.ClassChains(target) {
		if c.Tier != "required" {
			continue
		}
		tank := w.tankGoals()[c.Goal.ID]
		if r, err := w.FollowGuide(out, target); err == nil && handsIn(r, c.Goal.ID) && (!tank || w.tankFirst(r)) {
			continue // the route does it already (a tank form or stance before the first dungeon)
		}
		// the chain's class quests taken out and put in again whole, in order (a guide may do part of it, or out of
		// order); where that does not finish it before the target, each part at the first place it can be taken
		class := map[int]bool{}
		_, extra := extraOptions[c.Goal.ID]
		for _, q := range c.Quests {
			class[q.ID] = q.Classes != 0 || extra // an extra option's steps are all its own
		}
		rest := without(out, func(e GuideEvent) bool { return class[e.Quest] })
		if tank {
			// the tank form or stance before the first dungeon: the chain is put in without the runs, then each run
			// that came before its turn-in goes right after it
			bare := without(rest, func(e GuideEvent) bool { return e.Op == "dungeon" })
			cand := w.withQuestsIn(bare, c.Quests, target, classWindow)
			if r, err := w.FollowGuide(cand, target); err != nil || !handsIn(r, c.Goal.ID) {
				cand = w.chainInOnePiece(bare, c, target, cand)
			}
			out = w.runsAfter(rest, cand, c.Goal.ID)
			continue
		}
		cand := w.withQuestsIn(rest, c.Quests, target, classWindow)
		if r, err := w.FollowGuide(cand, target); err != nil || !handsIn(r, c.Goal.ID) {
			if early := w.withQuestsIn(rest, c.Quests, target, 0); early != nil {
				if r, err := w.FollowGuide(early, target); err == nil && handsIn(r, c.Goal.ID) {
					cand = early
				} else {
					// the whole chain in one piece, from the step that reaches its level backwards
					cand = w.chainInOnePiece(rest, c, target, cand)
				}
			}
		}
		out = cand
	}
	return out
}

// chainInOnePiece puts a class chain into the route in one piece at the latest place, from where the route reaches the
// chain's level backwards, that hands its last quest in; fallback when there is none.
func (w *World) chainInOnePiece(events []GuideEvent, c ClassChain, target int, fallback []GuideEvent) []GuideEvent {
	var block []GuideEvent
	_, extra := extraOptions[c.Goal.ID]
	for _, q := range c.Quests {
		if q.Classes != 0 || extra {
			block = append(block, GuideEvent{Op: "accept", Quest: q.ID}, GuideEvent{Op: "complete", Quest: q.ID}, GuideEvent{Op: "turnin", Quest: q.ID})
		}
	}
	levels := w.eventLevels(events, target)
	start := len(events)
	for j, l := range levels {
		if l >= target {
			start = j
			break
		}
	}
	for p, tries := start, 0; p >= 0 && tries < 40; p, tries = p-3, tries+1 {
		cand := append(append(append([]GuideEvent(nil), events[:p]...), block...), events[p:]...)
		if r, err := w.FollowGuide(cand, target); err == nil && handsIn(r, c.Goal.ID) {
			return cand
		}
	}
	return fallback
}

// runsAfter puts the dungeon runs of the route before (orig) into the route with a chain put in (chain, no runs): each
// after the quest it followed, a run before the goal's turn-in right after that turn-in.
func (w *World) runsAfter(orig, chain []GuideEvent, goal int) []GuideEvent {
	// the event each run followed in the original
	after := map[int][]GuideEvent{} // index in chain -> runs
	pos := map[GuideEvent]int{}
	for i, e := range chain {
		if _, seen := pos[e]; !seen {
			pos[e] = i
		}
	}
	gi := -1
	for i, e := range chain {
		if e.Quest == goal && e.Op == "turnin" {
			gi = i
		}
	}
	prev := -1
	for _, e := range orig {
		if e.Op != "dungeon" {
			if i, ok := pos[e]; ok {
				prev = i
			}
			continue
		}
		at := max(prev, gi)
		after[at] = append(after[at], e)
	}
	var out []GuideEvent
	out = append(out, after[-1]...)
	for i, e := range chain {
		out = append(out, e)
		out = append(out, after[i]...)
	}
	return out
}

// handsIn: the walked route hands in the quest.
func handsIn(r *GuideResult, id int) bool {
	for _, st := range r.Steps {
		if st.Op == "turnin" && st.Quest == id {
			return true
		}
	}
	return false
}

// withQuests puts in the given quests the route lacks, as withClassQuests.
func (w *World) withQuests(events []GuideEvent, quests []*Quest, target int) []GuideEvent {
	return w.withQuestsIn(events, quests, target, classWindow)
}

// withQuestsIn: withQuests, each quest within window levels of when it can be taken.
func (w *World) withQuestsIn(events []GuideEvent, quests []*Quest, target, window int) []GuideEvent {
	in := map[int]bool{}
	for _, q := range questsOf(events) {
		in[q] = true
	}
	var missing []*Quest
	for _, q := range quests {
		if !in[q.ID] {
			missing = append(missing, q)
		}
	}
	// in the order given: a chain's quests come in their order, a follow-up after its prerequisite
	out := append([]GuideEvent(nil), events...)
	for len(missing) > 0 {
		// the next quest whose class prerequisites are in the route already
		next := -1
		for k, q := range missing {
			ready := true
			for _, p := range append(append([]int(nil), q.PreAll...), q.PreAny...) {
				if pq := w.ByID[p]; pq != nil && pq.Classes != 0 && !in[p] && slices.Contains(missing, pq) {
					ready = false
				}
			}
			if ready {
				next = k
				break
			}
		}
		if next < 0 {
			next = 0 // a loop in the data: take them in order
		}
		q := missing[next]
		missing = append(missing[:next:next], missing[next+1:]...)
		levels := w.eventLevels(out, target)
		after := -1 // a follow-up goes after its prerequisite's turn-in
		for _, p := range append(append([]int(nil), q.PreAll...), q.PreAny...) {
			for j, e := range out {
				if e.Quest == p && e.Op == "turnin" && j > after {
					after = j
				}
			}
		}
		// within classWindow levels of when it can be taken, where the route comes closest to its giver; where the
		// route reaches its level only on the way to the target (a last grinding step), before that step: the follower
		// levels for it there
		cut := len(out)
		for j := after + 1; j < len(out); j++ {
			if levels[j] >= target {
				cut = j
				break
			}
		}
		at := max(cut-2, after) // before the step that reaches the target
		lo := -1
		for j := after + 1; j < cut; j++ {
			if levels[j] >= q.MinLevel {
				lo = j
				break
			}
		}
		if lo >= 0 {
			best := math.Inf(1)
			for j := lo; j < cut && levels[j] <= levels[lo]+window; j++ {
				eq := w.ByID[out[j].Quest]
				if eq == nil {
					continue
				}
				p := eq.StartPos
				if out[j].Op == "turnin" {
					p = eq.EndPos
				}
				if d := distance(p, q.StartPos); d < best {
					at, best = j, d
				}
			}
			if math.IsInf(best, 1) {
				at = lo
			}
		}
		block := []GuideEvent{{Op: "accept", Quest: q.ID}, {Op: "complete", Quest: q.ID}, {Op: "turnin", Quest: q.ID}}
		out = append(append(append([]GuideEvent(nil), out[:at+1]...), block...), out[at+1:]...)
		in[q.ID] = true
	}
	return out
}

// eventLevels: the character's level at each event when the route is walked (the target for what comes after it is
// reached).
func (w *World) eventLevels(events []GuideEvent, target int) []int {
	r, err := w.FollowGuide(events, target)
	if err != nil || r.levels == nil {
		l := make([]int, len(events))
		for i := range l {
			l[i] = 1
		}
		return l
	}
	return r.levels
}

// questsOf lists the quests a guide names, in first-seen order.
func questsOf(events []GuideEvent) []int {
	seen := map[int]bool{}
	var out []int
	for _, e := range events {
		if e.Quest != 0 && !seen[e.Quest] {
			seen[e.Quest] = true
			out = append(out, e.Quest)
		}
	}
	return out
}

func without(events []GuideEvent, drop func(GuideEvent) bool) []GuideEvent {
	out := make([]GuideEvent, 0, len(events))
	for _, e := range events {
		if !drop(e) {
			out = append(out, e)
		}
	}
	return out
}

// closestStep is the index of the guide step whose place (the giver or ender of its quest) is closest to p.
func (w *World) closestStep(events []GuideEvent, p dataset.Point) (int, float64) {
	bi, bd := 0, math.Inf(1)
	for i, e := range events {
		q := w.ByID[e.Quest]
		if q == nil {
			continue
		}
		at := q.StartPos
		if e.Op == "turnin" {
			at = q.EndPos
		}
		if d := distance(at, p); d < bd {
			bi, bd = i, d
		}
	}
	return bi, bd
}

// EventsOf turns a planned route into guide steps, so planned routes and guides are improved alike. The planner's
// grinding becomes a level step (grind to the level it reached there): left out, the follower would do the next
// quests a level lower and grind at the end.
func EventsOf(steps []Step) []GuideEvent {
	var out []GuideEvent
	done := map[int]bool{}
	for _, s := range steps {
		switch s.Op {
		case "accept", "turnin":
			out = append(out, GuideEvent{Op: s.Op, Quest: s.Quest})
		case "work":
			if !done[s.Quest] {
				done[s.Quest] = true
				out = append(out, GuideEvent{Op: "complete", Quest: s.Quest})
			}
		case "dungeon":
			out = append(out, GuideEvent{Op: "dungeon", Dungeon: s.Dungeon})
		case "grind":
			if n := len(out); n > 0 && out[n-1].Op == "level" {
				out[n-1].Level = s.Level
			} else {
				out = append(out, GuideEvent{Op: "level", Level: s.Level})
			}
		}
	}
	return out
}

// ClassCost is what a class quest chain costs a route: the time without it (a chain the route does) or with it put in
// (an option).
type ClassCost struct {
	Reward   string  `json:"reward"`
	Tier     string  `json:"tier"` // required, recommended, optional
	Quest    int     `json:"quest"`
	Name     string  `json:"name"`
	MinLevel int     `json:"minLevel"`
	Quests   []int   `json:"quests"`
	InRoute  bool    `json:"inRoute"`         // the route hands in the chain's last quest
	Minutes  float64 `json:"minutes"`         // time it adds to the route (negative: it saves time)
	Possible bool    `json:"possible"`        // put in, it is handed in
	Level    int     `json:"level,omitempty"` // the level the route hands it in at
	// Events: for an option, the route with it (each step where the route comes closest to it), improved
	Events []GuideEvent `json:"events,omitempty"`
}

// optionWindows: how many levels after an option's step can be taken it may go in, tried in turn (weaveIn).
var optionWindows = []int{2, 4, 6, 20}

// weaveIn puts an option's chain into a route place by place (weaveLegs): of the windows that finish the chain before
// the target, the fastest route.
func (w *World) weaveIn(events []GuideEvent, c ClassChain, target int) []GuideEvent {
	var best []GuideEvent
	bt := math.Inf(1)
	for _, win := range optionWindows {
		cand := w.weaveLegs(events, c, target, win)
		if r, err := w.FollowGuide(cand, target); err == nil && handsIn(r, c.Goal.ID) && r.Seconds < bt {
			best, bt = cand, r.Seconds
		}
	}
	if best == nil {
		return w.chainInOnePiece(events, c, target, w.withQuestsIn(events, c.Quests, target, 2))
	}
	return best
}

// legRadius: chain steps this close together are done in one visit.
const legRadius = 150.0

// legTries: the places of a route nearest a leg that weaveLegs measures it at (besides right after the leg before).
const legTries = 8

// leg is one visit of a chain: its steps at one place, from a level on.
type leg struct {
	events   []GuideEvent
	at       dataset.Point
	minLevel int
}

// chainLegs: the steps of a chain the route lacks, in order, grouped by place: a turn-in and the next quest taken where
// it was handed in are one visit (accept and complete together: the follower works the objectives right after).
func (w *World) chainLegs(events []GuideEvent, c ClassChain) []leg {
	in := map[int]bool{}
	for _, q := range questsOf(events) {
		in[q] = true
	}
	var legs []leg
	add := func(e GuideEvent, p dataset.Point, min int) {
		if n := len(legs); n > 0 && legs[n-1].at.Map == p.Map && distance(legs[n-1].at, p) <= legRadius {
			legs[n-1].events = append(legs[n-1].events, e)
			legs[n-1].minLevel = max(legs[n-1].minLevel, min)
			return
		}
		legs = append(legs, leg{events: []GuideEvent{e}, at: p, minLevel: min})
	}
	for _, q := range c.Quests {
		if in[q.ID] {
			continue
		}
		add(GuideEvent{Op: "accept", Quest: q.ID}, q.StartPos, q.MinLevel)
		add(GuideEvent{Op: "complete", Quest: q.ID}, q.StartPos, q.MinLevel)
		add(GuideEvent{Op: "turnin", Quest: q.ID}, q.EndPos, 0)
	}
	return legs
}

// eventPos: where a route's event takes place (accept at the giver, else at the ender); false for a level or a run.
func (w *World) eventPos(e GuideEvent) (dataset.Point, bool) {
	q := w.ByID[e.Quest]
	if q == nil {
		return dataset.Point{}, false
	}
	if e.Op == "turnin" {
		return q.EndPos, true
	}
	return q.StartPos, true
}

// weaveLegs puts a chain's legs into a route in order, each at the place within window levels after it can be taken
// that costs the route least (measured: the legTries places the route comes closest to it, and right after the leg
// before), so a leg is done when the route is there anyway (the Barrens steps in the Barrens) or goes straight on.
func (w *World) weaveLegs(events []GuideEvent, c ClassChain, target, window int) []GuideEvent {
	out := append([]GuideEvent(nil), events...)
	after := -1 // the last event of the leg before
	legs := w.chainLegs(events, c)
	for i, l := range legs {
		levels := w.eventLevels(out, target)
		cut := len(out)
		for j := after + 1; j < len(out); j++ {
			if levels[j] >= target {
				cut = j
				break
			}
		}
		lo := -1
		for j := after + 1; j < cut; j++ {
			if levels[j] >= l.minLevel {
				lo = j
				break
			}
		}
		if lo < 0 {
			lo = max(cut-2, after+1) // the route reaches its level only on the way to the target: the follower levels for it
		}
		type cand struct {
			j int
			d float64
		}
		var cs []cand
		for j := lo; j < cut && levels[j] <= levels[lo]+window; j++ {
			if p, ok := w.eventPos(out[j]); ok && p.Map == l.at.Map {
				cs = append(cs, cand{j, distance(p, l.at)})
			}
		}
		sort.Slice(cs, func(a, b int) bool { return cs[a].d < cs[b].d })
		tries := []int{max(lo-1, after)}
		for k := 0; k < len(cs) && k < legTries; k++ {
			tries = append(tries, cs[k].j)
		}
		insert := func(j int, rest bool) []GuideEvent {
			block := append([]GuideEvent(nil), l.events...)
			if rest {
				for _, m := range legs[i+1:] {
					block = append(block, m.events...)
				}
			}
			return append(append(append([]GuideEvent(nil), out[:j+1]...), block...), out[j+1:]...)
		}
		// measured with the legs after it straight on: what comes after the target costs nothing, so a leg measured
		// alone would go as late as it can and the chain not be finished
		best, bt := tries[0], math.Inf(1)
		for _, j := range tries {
			if r, err := w.FollowGuide(insert(j, true), target); err == nil && handsIn(r, c.Goal.ID) && r.Seconds < bt {
				best, bt = j, r.Seconds
			}
		}
		out = insert(best, false)
		after = best + len(l.events)
	}
	return out
}

// WithoutExtraOptions leaves out the steps of the extra options' chains (extraOptions) a route does not finish: a
// guide picks up their first quest on the way; the addon switches the whole chain on as an option.
func (w *World) WithoutExtraOptions(events []GuideEvent, target int) []GuideEvent {
	var drop map[int]bool
	r, err := w.FollowGuide(events, target)
	for _, c := range w.ClassChains(target) {
		if _, extra := extraOptions[c.Goal.ID]; !extra || (err == nil && handsIn(r, c.Goal.ID)) {
			continue
		}
		if drop == nil {
			drop = map[int]bool{}
		}
		for _, q := range c.Quests {
			drop[q.ID] = true
		}
	}
	if drop == nil {
		return events
	}
	return without(events, func(e GuideEvent) bool { return drop[e.Quest] })
}

// reportBudget: how long ClassReport improves each side of a measure.
const reportBudget = 8 * time.Second

// setClassOverride turns a chain off or on for a measure ("" back to its tier).
func (w *World) setClassOverride(goal int, v string) {
	if w.classOverride == nil {
		w.classOverride = map[int]string{}
	}
	if v == "" {
		delete(w.classOverride, goal)
	} else {
		w.classOverride[goal] = v
	}
	w.required = nil // keepClassQuest's cache
}

// ClassReport measures every class quest chain of this character (ClassChains) against a route.
func (w *World) ClassReport(events []GuideEvent, target int) []ClassCost {
	base, err := w.FollowGuide(events, target)
	if err != nil {
		return nil
	}
	turnedIn := func(r *GuideResult, id int) int {
		for _, st := range r.Steps {
			if st.Op == "turnin" && st.Quest == id {
				return max(st.Level, 1)
			}
		}
		return 0
	}
	// the route itself improved as briefly, so both sides of every measure had the same search
	ref := base
	if _, r, err := w.Improve(events, target, reportBudget); err == nil {
		ref = r
	}
	var out []ClassCost
	for _, c := range w.ClassChains(target) {
		cc := ClassCost{Reward: c.Reward, Tier: c.Tier, Quest: c.Goal.ID, Name: c.Goal.Name, MinLevel: c.Goal.MinLevel}
		ids := map[int]bool{}
		_, extra := extraOptions[c.Goal.ID]
		for _, q := range c.Quests {
			cc.Quests = append(cc.Quests, q.ID)
			ids[q.ID] = q.Classes != 0 || extra // a class chain's class quests: an ordinary quest before them stays
		}
		// each side improved briefly before they are compared: taken out or put in without re-planning, a chain moves
		// the levels the route reaches its runs and quests at, and that alone can look like hours
		measure := func(evs []GuideEvent, override string) ([]GuideEvent, *GuideResult) {
			w.setClassOverride(c.Goal.ID, override)
			defer w.setClassOverride(c.Goal.ID, "")
			if e, r, err := w.Improve(evs, target, reportBudget); err == nil {
				return e, r
			}
			return nil, nil
		}
		if l := turnedIn(base, c.Goal.ID); l > 0 {
			cc.InRoute, cc.Possible, cc.Level = true, true, l
			if _, r := measure(without(events, func(e GuideEvent) bool { return ids[e.Quest] }), "off"); r != nil {
				cc.Minutes = math.Round((ref.Seconds-r.Seconds)/6) / 10
			}
		} else {
			// an option: woven into the route as it is, visit by visit (weaveIn), not re-planned around it: the addon and
			// the live plan put its steps into this route, so what it costs is measured on this route
			rest := without(events, func(e GuideEvent) bool { return ids[e.Quest] }) // a guide's part of it, out of place
			woven := w.weaveIn(rest, c, target)
			if r, err := w.FollowGuide(woven, target); err == nil {
				cc.Minutes = math.Round((r.Seconds-base.Seconds)/6) / 10
				cc.Level = turnedIn(r, c.Goal.ID)
				cc.Possible = cc.Level > 0
				if cc.Possible {
					cc.Events = woven
				}
			}
		}
		out = append(out, cc)
	}
	return out
}

// MergeTrips joins a route's stays on one continent that a short trip to the other splits (Silverpine, two levels in
// the Barrens, back to Lordaeron for the Ruins, the Barrens again): the stay after the trip is moved before it, so
// the sea is crossed once. A run counts as its entrance's continent. Each join only when the route gets faster; again
// while one does.
func (w *World) MergeTrips(events []GuideEvent, target int) []GuideEvent {
	best, err := w.FollowGuide(events, target)
	if err != nil {
		return events
	}
	mapOf := func(e GuideEvent) int {
		if e.Op == "dungeon" {
			if d := w.dungeonOf[e.Dungeon]; d != nil {
				return d.Entrance.Map
			}
			return -1
		}
		if p, ok := w.eventPos(e); ok {
			return p.Map
		}
		return -1
	}
	for changed := true; changed; {
		changed = false
		// blocks: [start, end) of events on one continent; events without a place stay with the block before
		var starts []int
		var maps []int
		for i, e := range events {
			m := mapOf(e)
			if m < 0 || len(maps) > 0 && maps[len(maps)-1] == m {
				continue
			}
			starts, maps = append(starts, i), append(maps, m)
		}
		if len(starts) > 0 {
			starts[0] = 0
		}
		starts = append(starts, len(events))
		for k := 1; k+1 < len(maps); k++ {
			if maps[k-1] != maps[k+1] {
				continue
			}
			y, a := events[starts[k]:starts[k+1]], events[starts[k+1]:starts[k+2]]
			cand := slices.Concat(events[:starts[k]], a, y, events[starts[k+2]:])
			if r, err := w.FollowGuide(cand, target); err == nil && r.Level >= target && r.Seconds < best.Seconds-1 {
				best, events, changed = r, cand, true
				break
			}
		}
	}
	return events
}
