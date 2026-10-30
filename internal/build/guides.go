package build

import (
	"math"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/guides"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/polestar"
)

// Guide facts (internal/sources/guides) fill what no database knows, mostly for Forever's own quests: which creature
// drops an item (the creatures a guide sends the player to kill for it; the chance stays unknown), where items lie on
// the ground (the guide's places), where a creature or object stands that no source places, and where the item that
// starts a quest drops. A guide fact never replaces a database value; it only fills a gap.

const guideSrc = "guide"

// guideNear: a named creature counts for a guide step when it stands this close to one of the step's places.
const guideNear = 400

type guideIndex struct {
	byQuest  map[int][]guides.Fact
	credit   map[string]bool // sources used
	friendly map[int]bool    // quest givers, turn-ins and NPCs with a title (vendors, trainers): never a drop
}

func (b *builder) loadGuides(facts []guides.Fact) {
	b.gdx = &guideIndex{byQuest: map[int][]guides.Fact{}, credit: map[string]bool{}, friendly: map[int]bool{}}
	for _, f := range facts {
		b.gdx.byQuest[f.Quest] = append(b.gdx.byQuest[f.Quest], f)
	}
	if b.pdb != nil {
		for _, refs := range []map[int][]polestar.Ref{b.pdb.Starts, b.pdb.Ends} {
			for _, l := range refs {
				for _, r := range l {
					if r.Kind == "npc" {
						b.gdx.friendly[r.ID] = true
					}
				}
			}
		}
		for id, n := range b.pdb.NPCs {
			if n.Subname != "" {
				b.gdx.friendly[id] = true
			}
		}
	}
}

// guidePoints turns a step's places into world yards.
func (b *builder) guidePoints(ps []guides.Place) []dataset.Point {
	var out []dataset.Point
	for _, p := range ps {
		switch {
		case p.World:
			out = append(out, dataset.Point{Map: p.Map, X: round1(p.X), Y: round1(p.Y)})
		case p.UIMap > 0:
			if pt, ok := b.uiPoint(p.UIMap, p.X, p.Y); ok {
				out = append(out, pt)
			}
		case p.Zone != "":
			if ui, ok := b.zoneID[normName(p.Zone)]; ok {
				if pt, ok := b.uiPoint(ui, p.X, p.Y); ok {
					out = append(out, pt)
				}
			}
		}
	}
	return out
}

// uiPoint converts map percent with forever-db's map frames, else the fitted transforms.
func (b *builder) uiPoint(ui int, x, y float64) (dataset.Point, bool) {
	if b.pdb != nil {
		if p, ok := b.uiToWorld(ui, x, y); ok {
			return p, true
		}
	}
	if t, ok := b.uiT[strconv.Itoa(ui)]; ok {
		return t.apply(x, y), true
	}
	return dataset.Point{}, false
}

// creaturePoints is where a creature stands in any source.
func (b *builder) creaturePoints(id int) []dataset.Point {
	pts := b.worldPoints(id)
	if b.pdb != nil {
		for _, s := range b.pdb.Spawns["npc"][id] {
			pts = append(pts, dataset.Point{Map: s.Map, X: s.X, Y: s.Y})
		}
	}
	return pts
}

// inZone tells whether a point lies inside a zone's in-game map.
func (b *builder) inZone(zone string, p dataset.Point) bool {
	ui, ok := b.zoneID[normName(zone)]
	if !ok || b.pdb == nil {
		return false
	}
	for _, m := range b.pdb.UIMaps {
		if m.UIMap == ui && m.Map == p.Map &&
			p.X >= math.Min(m.Top, m.Bottom) && p.X <= math.Max(m.Top, m.Bottom) &&
			p.Y >= math.Min(m.Left, m.Right) && p.Y <= math.Max(m.Left, m.Right) {
			return true
		}
	}
	return false
}

// namedCreatures are the creatures a guide names, near the step's places when it gives any, else in the step's zone
// when the guide names one.
func (b *builder) namedCreatures(names []string, near []dataset.Point, zone string) []int {
	var out []int
	for _, name := range names {
		ids := append([]int(nil), b.byName[normName(name)]...)
		if b.pdb != nil {
			for id, n := range b.pdb.NPCs {
				if normName(n.Name) == normName(name) {
					ids = append(ids, id)
				}
			}
		}
		ids = dedupe(sortedInts(ids))
		var close []int
		for _, id := range ids {
			if b.gdx.friendly[id] {
				continue
			}
			switch {
			case len(near) > 0:
				if closeTo(b.creaturePoints(id), near) {
					close = append(close, id)
				}
			case zone != "":
				for _, p := range b.creaturePoints(id) {
					if b.inZone(zone, p) {
						close = append(close, id)
						break
					}
				}
			default:
				close = append(close, id)
			}
		}
		if len(close) == 0 && len(near) == 0 && zone == "" {
			close = ids
		}
		out = append(out, close...)
	}
	return dedupe(sortedInts(out))
}

func closeTo(pts, near []dataset.Point) bool {
	for _, p := range pts {
		for _, n := range near {
			if p.Map == n.Map && math.Hypot(p.X-n.X, p.Y-n.Y) <= guideNear {
				return true
			}
		}
	}
	return false
}

func sortedInts(l []int) []int { sort.Ints(l); return l }

// guideFacts are the facts about an item of a quest: by the item's id, or by the objective's number when forever-db
// names the item behind it.
func (b *builder) guideFacts(quest, item int, start bool) []guides.Fact {
	if b.gdx == nil {
		return nil
	}
	var out []guides.Fact
	for _, f := range b.gdx.byQuest[quest] {
		if f.Start != start {
			continue
		}
		match := f.Item == item
		if !match && !start && f.Obj > 0 && b.pdb != nil {
			for _, o := range b.pdb.Objectives[quest] {
				if o.N == f.Obj && o.Target == item {
					match = true
				}
			}
		}
		if match {
			out = append(out, f)
		}
	}
	return out
}

// guideItem fills an item objective (or a quest's starting item) that no source explains: the creatures the guides
// send the player to kill become drops of unknown chance; without creatures, the places where the items lie become
// the objective's places. It tells whether anything was found.
func (b *builder) guideItem(out *dataset.Quest, o *dataset.Objective, start bool) bool {
	facts := b.guideFacts(out.ID, o.ID, start)
	if len(facts) == 0 {
		return false
	}
	var npcs []int
	var places []dataset.Point
	for _, f := range facts {
		pts := b.guidePoints(f.Places)
		for _, id := range f.Near {
			pts = append(pts, b.creaturePoints(id)...)
		}
		ids := append(b.namedCreatures(f.Mobs, pts, f.Zone), f.NPCs...)
		npcs = append(npcs, ids...)
		if len(ids) == 0 && len(f.Objects) == 0 {
			places = append(places, pts...)
		}
		for _, ob := range f.Objects {
			o.Drops = append(o.Drops, dataset.Drop{Object: ob, Chance: 100})
			b.objWanted[ob] = true
		}
		b.gdx.credit[f.Source] = true
	}
	for _, id := range dedupe(sortedInts(npcs)) {
		o.Drops = append(o.Drops, dataset.Drop{NPC: id, Unknown: true})
		b.npcWanted[id] = true
	}
	if len(o.Drops) > 12 {
		o.Drops = o.Drops[:12]
	}
	if len(o.Drops) == 0 && len(places) > 0 {
		o.Places = dedupePoints(places)
	}
	if len(o.Drops) == 0 && len(o.Places) == 0 {
		return false
	}
	o.Source += "+" + guideSrc
	b.stats["items_sources_from_guides"]++
	return true
}

// guideSpawns: creatures and objects a quest needs that no source places get the guides' places for that objective.
func (b *builder) guideSpawns() {
	npcs := map[int]*dataset.NPC{}
	for i := range b.npcOut {
		npcs[b.npcOut[i].ID] = &b.npcOut[i]
	}
	// RestedXP names the NPC a step talks to with its id: where it stands, when no source places it there (an NPC
	// Forever put in one more start zone: the Dwarf shamans' earth spirit in Coldridge Valley)
	for _, t := range b.rxpTargets {
		n := npcs[t.NPC]
		pts := b.guidePoints([]guides.Place{t.Place})
		if n == nil || len(pts) == 0 || (pts[0].Map != 0 && pts[0].Map != 1) {
			continue
		}
		near := false
		for _, s := range n.Spawns {
			near = near || (s.Map == pts[0].Map && math.Hypot(s.X-pts[0].X, s.Y-pts[0].Y) <= 300)
		}
		if !near {
			n.Spawns = append(n.Spawns, pts[0])
			if !strings.Contains(n.Source, "rxp_target") {
				n.Source += "+rxp_target"
			}
			b.stats["npc_spawns_from_rxp_targets"]++
		}
	}
	if b.gdx == nil {
		return
	}
	objs := map[int]*dataset.Object{}
	for i := range b.objOut {
		objs[b.objOut[i].ID] = &b.objOut[i]
	}
	for _, q := range b.out {
		for _, f := range b.gdx.byQuest[q.ID] {
			if f.Obj == 0 || len(f.Places) == 0 || b.pdb == nil {
				continue
			}
			for _, o := range b.pdb.Objectives[q.ID] {
				if o.N != f.Obj {
					continue
				}
				pts := b.guidePoints(f.Places)
				switch o.Kind {
				case "kill", "talk":
					if n := npcs[o.Target]; n != nil && len(n.Spawns) == 0 && len(pts) > 0 {
						n.Spawns, n.Source = pts, "guide_places"
						b.stats["npcs_placed_by_guides"]++
					}
				case "object":
					if ob := objs[o.Target]; ob != nil && len(ob.Spawns) == 0 && len(pts) > 0 {
						ob.Spawns = pts
						b.stats["objects_placed_by_guides"]++
					}
				}
			}
		}
	}
}

func dedupePoints(ps []dataset.Point) []dataset.Point {
	seen := map[dataset.Point]bool{}
	var out []dataset.Point
	for _, p := range ps {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	return thin(out, 24)
}

// breadcrumbPrereqs: a breadcrumb leads to its quest and is not needed for it, though the client's data lists some as
// prerequisites (The New Horde for Cutting Teeth, the three capitals' rogue quests for Redridge Rendezvous, The
// Slaughtered Lamb for Surena Caledon). cmangos names them (BreadcrumbForQuestId).
func (b *builder) breadcrumbPrereqs() {
	leads := map[int][]int{}
	for _, q := range b.out {
		if q.Breadcrumb > 0 {
			leads[q.Breadcrumb] = append(leads[q.Breadcrumb], q.ID)
		}
	}
	for i := range b.out {
		q := &b.out[i]
		for _, bc := range leads[q.ID] {
			if slices.Contains(q.PreAll, bc) || slices.Contains(q.PreAny, bc) {
				q.PreAll, q.PreAny = without(q.PreAll, bc), without(q.PreAny, bc)
				b.stats["prereqs_breadcrumbs_dropped"]++
			}
		}
	}
}

// rxpPrereqs takes out the prerequisites RestedXP's Forever guides show are not needed (trusted for levels 1-20): a
// guide takes the quest while the prerequisite is still in its log. Absence from a guide is not evidence. Class
// quests keep theirs, also as prerequisites: the guides leave many class quests out.
func (b *builder) rxpPrereqs() {
	if len(b.rxpOrder) == 0 {
		return
	}
	accepted := map[int]bool{}
	for _, steps := range b.rxpOrder {
		for _, st := range steps {
			if st.Op == "accept" {
				accepted[st.Quest] = true
			}
		}
	}
	class := map[int]bool{}
	for _, q := range b.out {
		class[q.ID] = q.Classes != 0
	}
	// open: a guide takes y (its first time in that guide) while x is taken and not handed in
	open := func(y, x int) bool {
		for _, name := range sortedFiles(b.rxpOrder) {
			taken := false
			for _, st := range b.rxpOrder[name] {
				if st.Quest == x {
					taken = st.Op == "accept"
				} else if st.Quest == y && st.Op == "accept" {
					if taken {
						return true
					}
					break
				}
			}
		}
		return false
	}
	for i := range b.out {
		q := &b.out[i]
		if q.Classes != 0 || !accepted[q.ID] || len(q.PreAll)+len(q.PreAny) == 0 {
			continue
		}
		needless := func(x int) bool { return !class[x] && open(q.ID, x) }
		var all, anyOf []int
		for _, x := range q.PreAll {
			if !needless(x) {
				all = append(all, x)
			}
		}
		// An alternative that was not required in one guide may still be a valid way to unlock the quest.
		// Drop the OR requirement only when the guide demonstrates that none of its alternatives was handed in.
		anyOf = q.PreAny
		if len(q.PreAny) > 0 {
			allOpen := true
			for _, x := range q.PreAny {
				allOpen = allOpen && needless(x)
			}
			if allOpen {
				anyOf = nil
			}
		}
		if len(all) != len(q.PreAll) || len(anyOf) != len(q.PreAny) {
			q.PreAll, q.PreAny = all, anyOf
			q.Sources["pre"] = "RestedXP Forever guides (taken without it)"
			b.stats["prereqs_dropped_by_rxp_order"]++
		}
	}
}

func sortedFiles(m map[string][]guides.OrderStep) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
