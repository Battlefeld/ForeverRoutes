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
	try := func(cand []GuideEvent) bool {
		r, err := w.FollowGuide(cand, target)
		if err != nil || r.Level < target || r.Seconds >= best.Seconds-1 {
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
func (w *World) WithDungeons(events []GuideEvent, target int) []GuideEvent {
	best, err := w.FollowGuide(events, target)
	if err != nil {
		return events
	}
	cur := events
	var least []GuideEvent
	leastT := math.Inf(1)
	for _, d := range w.Dungeons {
		have := false
		for _, e := range cur {
			have = have || (e.Op == "dungeon" && e.Dungeon == d.Key)
		}
		if have || d.Min > target {
			continue
		}
		var pick []GuideEvent
		for k := 1; k <= 20; k++ {
			i := len(cur) * k / 21
			cand := append(append(append([]GuideEvent(nil), cur[:i]...), GuideEvent{Op: "dungeon", Dungeon: d.Key}), cur[i:]...)
			r, err := w.FollowGuide(cand, target)
			if err != nil || r.Level < target {
				continue
			}
			if r.Seconds < best.Seconds-1 {
				best, pick = r, cand
			}
			if r.Seconds < leastT {
				least, leastT = cand, r.Seconds
			}
		}
		if pick != nil {
			cur = pick
		}
	}
	if Runs(cur) == 0 && least != nil {
		return least // no run made it faster: until then cur is the route given, least one run put into it
	}
	return cur
}

// Runs counts the dungeon runs of a route.
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
	return w.withQuests(events, w.classQuests(target), target)
}

// withQuests puts in the given quests the route lacks, as withClassQuests.
func (w *World) withQuests(events []GuideEvent, quests []*Quest, target int) []GuideEvent {
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
	sort.Slice(missing, func(i, j int) bool { return missing[i].Index < missing[j].Index })
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
		// within classWindow levels of when it can be taken, where the route comes closest to its giver
		at := len(out) - 1
		lo := -1
		for j := after + 1; j < len(out); j++ {
			if levels[j] >= q.MinLevel {
				lo = j
				break
			}
		}
		if lo >= 0 {
			best := math.Inf(1)
			for j := lo; j < len(out) && levels[j] <= levels[lo]+classWindow; j++ {
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
	var out []ClassCost
	for _, c := range w.ClassChains(target) {
		cc := ClassCost{Reward: c.Reward, Tier: c.Tier, Quest: c.Goal.ID, Name: c.Goal.Name, MinLevel: c.Goal.MinLevel}
		ids := map[int]bool{}
		for _, q := range c.Quests {
			cc.Quests = append(cc.Quests, q.ID)
			ids[q.ID] = q.Classes != 0 // the class quests only: an ordinary quest before them stays in the route
		}
		if l := turnedIn(base, c.Goal.ID); l > 0 {
			cc.InRoute, cc.Possible, cc.Level = true, true, l
			if r, err := w.FollowGuide(without(events, func(e GuideEvent) bool { return ids[e.Quest] }), target); err == nil {
				cc.Minutes = math.Round((base.Seconds-r.Seconds)/6) / 10
			}
		} else if r, err := w.FollowGuide(w.withQuests(events, c.Quests, target), target); err == nil {
			cc.Minutes = math.Round((r.Seconds-base.Seconds)/6) / 10
			cc.Level = turnedIn(r, c.Goal.ID)
			cc.Possible = cc.Level > 0
		}
		out = append(out, cc)
	}
	return out
}
