package route

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Site is where a creature or object is worked: the densest group of its spawns.
type Site struct {
	Center  dataset.Point
	Spawns  int     // spawns in the group
	Spacing float64 // yards between two neighbouring spawns
	Key     int     // unique per site in a world (respawn bookkeeping)
}

// Source is one way to make progress on an objective: kill a creature or use an object, with the chance per try.
type Source struct {
	NPC    *dataset.NPC
	Object int
	Chance float64 // 0..1 progress per kill/use
	Site   Site
	Talk   bool // talk to the NPC: no fight
	// Assumed: the data gives no chance for this source; the model's AssumedDrop is used and the result says so
	Assumed bool
	// Gather: the items lie at the site (a guide's places, no creature or object known): picked up like using objects
	Gather bool
}

// Objective of a quest, resolved to the places that serve it.
type Objective struct {
	Kind    string
	ID      int
	Count   int
	Sources []Source
	Inside  bool // a dungeon run does it (group routes)
}

// Quest is a quest this character can do, with resolved positions.
type Quest struct {
	*dataset.Quest
	Index        int
	StartPos     dataset.Point
	EndPos       dataset.Point
	StartNPC     int
	EndNPC       int
	Objectives   []Objective
	Unlocks      []int    // quests that need this one
	StartSources []Source // for a quest started by an item: where the item drops
	Dungeon      *Dungeon // the dungeon a dungeon quest belongs to (group routes)
	StartInside  bool     // given inside the dungeon: taken during the run
	EndInside    bool     // handed in inside the dungeon: turned in during the run
}

// Farm is a place to grind.
type Farm struct {
	NPC  *dataset.NPC
	Site Site
}

// World is the dataset seen by one character.
type World struct {
	Data      *dataset.Dataset
	Char      Character
	Faction   string
	Opt       Options
	Quests    []*Quest
	ByID      map[int]*Quest
	NPC       map[int]*dataset.NPC
	Objects   map[int]*dataset.Object
	Start     dataset.Point
	Excluded  map[int]string // quest -> why this character cannot use it
	Farms     []Farm
	net       *network
	walkGrids walks
	deaths    *deaths
	lifts     []dataset.Lift
	danger    dangerMap
	Dungeons  []*Dungeon // dungeons a group route may run (Options.Dungeons)
	dungeonOf map[string]*Dungeon
	landMemo  sync.Map // landKey -> landResult
	landN     atomic.Int64
	inns      []dataset.Point
	siteKeys  int
	siteMemo  map[*dataset.Point][]Site
	maps      map[int]bool
}

// NewWorld prepares the dataset for one character.
func NewWorld(d *dataset.Dataset, c Character, opt Options) (*World, error) {
	faction, race, ok := c.side()
	cls := ClassBits[c.Class]
	if !ok || cls == 0 {
		return nil, fmt.Errorf("unknown race or class: %s %s %s", c.Race, c.Class, c.Faction)
	}
	w := &World{Data: d, Char: c, Faction: faction, Opt: opt.withDefaults(), ByID: map[int]*Quest{}, NPC: map[int]*dataset.NPC{},
		Objects: map[int]*dataset.Object{}, Excluded: map[int]string{}}
	w.walkGrids = newWalks(w.Opt.Walks)
	w.deaths = newDeaths(d.Travel, c, faction)
	w.lifts = d.Travel.Lifts
	w.danger = newDanger(d.NPCs, faction)
	for i := range d.NPCs {
		w.NPC[d.NPCs[i].ID] = &d.NPCs[i]
	}
	for i := range d.Objects {
		w.Objects[d.Objects[i].ID] = &d.Objects[i]
	}
	if dp, ok := w.Opt.Profile.(DefaultProfile); ok && dp.EvenHP == nil {
		dp.EvenHP = medianByLevel(d, func(n *dataset.NPC) float64 { return float64(n.HP) })
		dp.EvenDmg = medianByLevel(d, func(n *dataset.NPC) float64 { return n.Dmg })
		w.Opt.Profile = dp
	}
	if tp, ok := w.Opt.Profile.(*TableProfile); ok && tp.Fallback.EvenHP == nil {
		tp.Fallback.EvenHP = medianByLevel(d, func(n *dataset.NPC) float64 { return float64(n.HP) })
		tp.Fallback.EvenDmg = medianByLevel(d, func(n *dataset.NPC) float64 { return n.Dmg })
	}
	startNPC := w.NPC[StartNPC[c.Race]]
	if startNPC == nil || len(startNPC.Spawns) == 0 {
		return nil, fmt.Errorf("start position of %s not in the dataset", c.Race)
	}
	w.Start = startNPC.Spawns[0]
	w.buildNetwork()
	w.maps = w.reachableMaps()
	w.dungeons()
	for i := range d.Quests {
		q := &d.Quests[i]
		if why := w.usable(q, race, cls); why != "" {
			w.Excluded[q.ID] = why
			continue
		}
		var rq *Quest
		var why string
		if q.Type == "dungeon" {
			rq, why = w.dungeonQuest(q)
		} else {
			rq, why = w.resolve(q)
		}
		if why != "" {
			w.Excluded[q.ID] = why
			continue
		}
		rq.Index = len(w.Quests)
		w.Quests = append(w.Quests, rq)
		w.ByID[q.ID] = rq
	}
	// a quest whose prerequisites can never be done cannot be done either
	for changed := true; changed; {
		changed = false
		for _, q := range w.Quests {
			if _, gone := w.Excluded[q.ID]; gone {
				continue
			}
			for _, p := range q.PreAll {
				if _, ok := w.ByID[p]; !ok || w.Excluded[p] != "" {
					w.Excluded[q.ID] = fmt.Sprintf("needs quest %d, which this character cannot do", p)
					changed = true
					break
				}
			}
			if w.Excluded[q.ID] == "" && len(q.PreAny) > 0 {
				ok := false
				for _, p := range q.PreAny {
					if _, in := w.ByID[p]; in && w.Excluded[p] == "" {
						ok = true
					}
				}
				if !ok {
					w.Excluded[q.ID] = "none of its prerequisites can be done"
					changed = true
				}
			}
		}
	}
	kept := w.Quests[:0]
	for _, q := range w.Quests {
		if w.Excluded[q.ID] == "" {
			q.Index = len(kept)
			kept = append(kept, q)
		} else {
			delete(w.ByID, q.ID)
		}
	}
	w.Quests = kept
	for _, q := range w.Quests {
		if q.Dungeon != nil {
			q.Dungeon.Quests = append(q.Dungeon.Quests, q)
		}
	}
	for _, q := range w.Quests {
		for _, p := range append(append([]int{}, q.PreAll...), q.PreAny...) {
			if pq := w.ByID[p]; pq != nil {
				pq.Unlocks = append(pq.Unlocks, q.ID)
			}
		}
	}
	w.farms()
	return w, nil
}

func (w *World) usable(q *dataset.Quest, race, cls int) string {
	switch {
	case q.Races != 0 && q.Races&race == 0:
		return "other race"
	case q.Classes != 0 && q.Classes&cls == 0:
		return "other class"
	case q.Repeatable:
		return "repeatable"
	case q.Type == "dungeon" && w.Opt.Dungeons && q.Dungeon != "":
		// a group route runs it
	case q.Type != "":
		return q.Type + " quest (not solo)"
	case q.Unsupported != "":
		return "data: " + q.Unsupported
	}
	return ""
}

// friendly: an NPC that does not attack this faction can give quests; one that does cannot.
func (w *World) friendly(n *dataset.NPC) bool {
	return n.Hostile == "" || n.Hostile == "-" || n.Hostile != w.Faction
}

func (w *World) refPos(refs []dataset.Ref, given *dataset.Point) (dataset.Point, int, bool) {
	var npcID int
	for _, r := range refs {
		if r.Kind == "npc" {
			if n := w.NPC[r.ID]; n != nil && len(n.Spawns) > 0 && w.friendly(n) {
				npcID = r.ID
				if given == nil {
					return nearest(n.Spawns, w.Start), npcID, true
				}
				// an NPC that stands in several start zones (the class spirits): the one nearest to this character's
				// start, though a source gave another one's place
				if p := nearest(n.Spawns, w.Start); distance(p, w.Start) < distance(*given, w.Start)-1000 {
					return p, npcID, true
				}
				break
			}
		}
	}
	if given != nil {
		return *given, npcID, true
	}
	for _, r := range refs {
		if r.Kind == "object" {
			if o := w.Objects[r.ID]; o != nil && len(o.Spawns) > 0 {
				return nearest(o.Spawns, w.Start), 0, true
			}
		}
	}
	return dataset.Point{}, 0, false
}

func nearest(ps []dataset.Point, to dataset.Point) dataset.Point {
	best, bd := ps[0], math.Inf(1)
	for _, p := range ps {
		if d := distance(p, to); d < bd {
			best, bd = p, d
		}
	}
	return best
}

func (w *World) resolve(q *dataset.Quest) (*Quest, string) {
	rq := &Quest{Quest: q}
	var ok bool
	if rq.StartPos, rq.StartNPC, ok = w.refPos(q.Start, q.StartAt); !ok {
		// started by an item: the quest begins where the item drops
		if len(q.StartDrops) == 0 {
			return nil, "start position unknown"
		}
		if rq.StartSources = usable(w.dropSources(q.StartDrops)); len(rq.StartSources) == 0 {
			return nil, "start item: no reachable non-elite source"
		}
		rq.StartPos = rq.StartSources[0].Site.Center
	}
	if rq.EndPos, rq.EndNPC, ok = w.refPos(q.End, q.EndAt); !ok {
		return nil, "turn-in position unknown"
	}
	if !w.maps[rq.StartPos.Map] || !w.maps[rq.EndPos.Map] {
		return nil, "other continent"
	}
	if q.Classes != 0 && q.MinLevel < 10 && rq.StartPos.Map != rq.EndPos.Map && rq.StartPos.Map <= 1 && rq.EndPos.Map <= 1 {
		// a first class quest that would cross the sea: its NPC's spawn in this start zone is not in the data (the
		// Forever Dwarf shamans' earth spirit)
		return nil, "class quest: its NPC stands only across the sea in the data"
	}
	for _, o := range q.Objectives {
		ro := Objective{Kind: o.Kind, ID: o.ID, Count: o.Count}
		switch o.Kind {
		case "talk":
			if n := w.NPC[o.ID]; n != nil && len(n.Spawns) > 0 && w.friendly(n) {
				ro.Sources = append(ro.Sources, Source{NPC: n, Chance: 1, Site: Site{Center: nearest(n.Spawns, w.Start), Spawns: 1}, Talk: true})
			}
		case "kill":
			if n := w.NPC[o.ID]; n != nil && len(n.Spawns) > 0 {
				for _, st := range w.sites(n.Spawns) {
					ro.Sources = append(ro.Sources, Source{NPC: n, Chance: 1, Site: st})
				}
			}
		case "object":
			if ob := w.Objects[o.ID]; ob != nil && len(ob.Spawns) > 0 {
				for _, st := range w.sites(ob.Spawns) {
					ro.Sources = append(ro.Sources, Source{Object: o.ID, Chance: 1, Site: st})
				}
			}
		case "event":
			// explore, escort, use something: done at the place, like gathering one thing there
			for _, st := range w.sites(o.Places) {
				ro.Sources = append(ro.Sources, Source{Chance: 1, Site: st, Gather: true})
			}
		case "item":
			// bought from a vendor: walk there and buy, like talking to an NPC
			for _, v := range o.Vendors {
				if n := w.NPC[v]; n != nil && len(n.Spawns) > 0 && w.friendly(n) {
					ro.Sources = append(ro.Sources, Source{NPC: n, Chance: 1, Site: Site{Center: nearest(n.Spawns, w.Start), Spawns: 1}, Talk: true})
				}
			}
			ro.Sources = append(ro.Sources, w.dropSources(o.Drops)...)
			if len(usable(ro.Sources)) == 0 && len(o.Places) > 0 {
				for _, st := range w.sites(o.Places) {
					ro.Sources = append(ro.Sources, Source{Chance: 1, Site: st, Gather: true})
				}
			}
		}
		ro.Sources = usable(ro.Sources)
		if len(ro.Sources) == 0 {
			return nil, fmt.Sprintf("objective %s %d: no reachable non-elite source", o.Kind, o.ID)
		}
		rq.Objectives = append(rq.Objectives, ro)
	}
	return rq, ""
}

// dropSources are the spawn groups of the creatures and objects an item drops from.
func (w *World) dropSources(drops []dataset.Drop) []Source {
	var out []Source
	for _, d := range drops {
		chance := math.Min(1, d.Chance/100)
		if d.Unknown {
			chance = w.Opt.AssumedDrop
		}
		if d.NPC > 0 {
			if n := w.NPC[d.NPC]; n != nil && len(n.Spawns) > 0 {
				for _, st := range w.sites(n.Spawns) {
					out = append(out, Source{NPC: n, Chance: chance, Site: st, Assumed: d.Unknown})
				}
			}
		} else if ob := w.Objects[d.Object]; ob != nil && len(ob.Spawns) > 0 {
			for _, st := range w.sites(ob.Spawns) {
				out = append(out, Source{Object: d.Object, Chance: chance, Site: st, Assumed: d.Unknown})
			}
		}
	}
	return out
}

// usable keeps the sources a solo character can use: placed, with a chance, not elite.
func usable(src []Source) []Source {
	var kept []Source
	for _, s := range src {
		if s.Site.Spawns == 0 || s.Chance <= 0 {
			continue
		}
		if !s.Talk && s.NPC != nil && (s.NPC.Rank == 1 || s.NPC.Rank == 2 || s.NPC.Rank == 3) {
			continue
		}
		kept = append(kept, s)
	}
	return kept
}

// maxSites is how many spawn groups of one creature or object a route may choose from.
const maxSites = 4

// sites are the densest groups of spawns on the continents this character can reach, largest first: the biggest group
// and the next ones that hold at least a quarter of its spawns (or 3), so a route can use the group closest to where
// it is. The result is the same for the same spawn list.
func (w *World) sites(all []dataset.Point) []Site {
	if len(all) == 0 {
		return nil
	}
	if out, ok := w.siteMemo[&all[0]]; ok {
		return out
	}
	var out []Site
	for m := range w.maps {
		out = append(out, groupsOf(all, m)...)
	}
	sort.SliceStable(out, func(a, b int) bool {
		if out[a].Spawns != out[b].Spawns {
			return out[a].Spawns > out[b].Spawns
		}
		if out[a].Center.Map != out[b].Center.Map {
			return out[a].Center.Map == w.Start.Map
		}
		return out[a].Center.X < out[b].Center.X
	})
	kept := out[:0]
	for i, st := range out {
		if i >= maxSites || (i > 0 && st.Spawns < max(3, out[0].Spawns/4)) {
			break
		}
		w.siteKeys++
		st.Key = w.siteKeys
		kept = append(kept, st)
	}
	if w.siteMemo == nil {
		w.siteMemo = map[*dataset.Point][]Site{}
	}
	w.siteMemo[&all[0]] = kept
	return kept
}

// groupsOf splits the spawns on map m into groups: repeatedly the point with the most neighbours within 120 yards
// and every remaining spawn within 150 yards of it.
func groupsOf(all []dataset.Point, m int) []Site {
	var ps []dataset.Point
	for _, p := range all {
		if p.Map == m {
			ps = append(ps, p)
		}
	}
	var out []Site
	for len(ps) > 0 {
		bestI, bestN := 0, -1
		for i, p := range ps {
			n := 0
			for _, q := range ps {
				if distance(p, q) <= 120 {
					n++
				}
			}
			if n > bestN {
				bestI, bestN = i, n
			}
		}
		var group, rest []dataset.Point
		for _, q := range ps {
			if distance(ps[bestI], q) <= 150 {
				group = append(group, q)
			} else {
				rest = append(rest, q)
			}
		}
		out = append(out, siteOf(group))
		ps = rest
	}
	return out
}

// siteOf describes one group of spawns.
func siteOf(group []dataset.Point) Site {
	var cx, cy float64
	for _, q := range group {
		cx += q.X
		cy += q.Y
	}
	s := Site{Center: dataset.Point{Map: group[0].Map, X: cx / float64(len(group)), Y: cy / float64(len(group))}, Spawns: len(group)}
	if len(group) == 1 {
		return s
	}
	sum := 0.0
	for i, p := range group {
		bd := math.Inf(1)
		for j, q := range group {
			if i != j {
				bd = math.Min(bd, distance(p, q))
			}
		}
		sum += bd
	}
	s.Spacing = math.Max(25, sum/float64(len(group))) // pulling one mob at a time keeps some distance
	return s
}

func (w *World) farms() {
	for i := range w.Data.NPCs {
		n := &w.Data.NPCs[i]
		if n.Rank != 0 || n.HP <= 0 || n.MaxLevel < 1 || n.MaxLevel > 40 || len(n.Spawns) < 6 {
			continue
		}
		if n.Hostile == "-" || (n.Hostile != "" && n.Hostile != w.Faction) {
			continue // friendly to this faction: cannot be attacked
		}
		for _, st := range w.sites(n.Spawns) {
			if st.Spawns >= 6 {
				w.Farms = append(w.Farms, Farm{NPC: n, Site: st})
			}
		}
	}
	sort.SliceStable(w.Farms, func(a, b int) bool { return w.Farms[a].NPC.ID < w.Farms[b].NPC.ID })
}
