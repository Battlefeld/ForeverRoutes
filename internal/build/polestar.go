package build

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/polestar"
)

// Polestar's forever-db (internal/sources/polestar) is read from the Forever client: where it knows a quest, its XP,
// levels, race and class masks are the game's own and win over every reference except values seen in the game. It
// gives objectives with counts to quests the other sources leave without, prerequisites, givers, what drops an item,
// and positions for creatures and objects the other sources do not place. Quests only it knows are added.

const polestarSrc = "polestar_client"

// Forever quest types (QuestInfo) that a solo route does not do.
var polestarTypes = map[int]string{81: "dungeon", 62: "raid", 41: "pvp"}

// raceMask keeps the classic race bits and Forever's Skyborne bits of a client mask.
func raceMask(m int) int { return m & (0xFFFF | polestar.SkyborneAlliance | polestar.SkyborneHorde) }

// observed tells whether a field's value was seen in the game (it beats the client data).
func observed(source string) bool { return strings.Contains(source, "observed") }

// applyPolestar completes a quest of the merged database from forever-db.
func (b *builder) applyPolestar(out *dataset.Quest) {
	p := b.pdb.Quests[out.ID]
	if p == nil {
		return
	}
	if p.XP > 0 && !observed(out.Sources["xp"]) && out.XP != p.XP {
		out.XP = p.XP
		out.Sources["xp"] = polestarSrc
		b.stats["xp_from_polestar"]++
	}
	if p.Level > 0 && !observed(out.Sources["level"]) {
		out.Level = p.Level
		out.Sources["level"] = polestarSrc
	}
	if p.MinLevel > 0 {
		out.MinLevel = p.MinLevel
	}
	out.Races, out.Classes = raceMask(p.Races), p.Classes&0xFFFF
	out.Type = polestarTypes[p.Type]
	if p.Repeat {
		out.Repeatable = true
	}
	if len(out.Start) == 0 {
		out.Start = b.polestarRefs(b.pdb.Starts[out.ID], true)
		out.Sources["start"] = polestarSrc
	}
	if len(out.End) == 0 {
		out.End = b.polestarRefs(b.pdb.Ends[out.ID], false)
		out.Sources["end"] = polestarSrc
	}
	if len(out.PreAll) == 0 && len(out.PreAny) == 0 {
		out.PreAll, out.PreAny, out.Breadcrumb = b.polestarPrereqs(out.ID, out.Breadcrumb)
	}
	// a quest the client marks as a breadcrumb leads here but is not needed (other sources list it as a prerequisite)
	for _, p := range b.pdb.Prereqs[out.ID] {
		if p.Breadcrumb {
			out.PreAll, out.PreAny = without(out.PreAll, p.Pre), without(out.PreAny, p.Pre)
		}
	}
	// objectives: the client's list where the other sources have none, or only some without a count
	if objectivesUnknown(out) && len(b.pdb.Objectives[out.ID]) > 0 {
		out.Objectives, out.Unsupported, out.NeedItems = nil, "", nil
		b.polestarObjectives(out)
		b.stats["objectives_from_polestar"]++
	}
}

// objectivesUnknown: the quest has no objectives, or is held back because an objective's target or count is unknown.
func objectivesUnknown(out *dataset.Quest) bool {
	if len(out.Objectives) == 0 {
		return true
	}
	u := out.Unsupported
	return strings.Contains(u, "count unknown") || strings.Contains(u, "objectives unknown") || strings.Contains(u, "objective type")
}

// newPolestarQuest makes a quest only forever-db knows; ok is false for one outside the level range.
func (b *builder) newPolestarQuest(p *polestar.Quest) (dataset.Quest, bool) {
	out := dataset.Quest{ID: p.ID, Name: p.Title, Level: p.Level, MinLevel: p.MinLevel, XP: p.XP, Races: raceMask(p.Races), Classes: p.Classes & 0xFFFF,
		Repeatable: p.Repeat, Type: polestarTypes[p.Type], Sources: map[string]string{"name": polestarSrc, "level": polestarSrc, "xp": polestarSrc, "start": polestarSrc, "end": polestarSrc}}
	if out.Level > b.in.MaxQuestLevel || out.MinLevel > b.in.MaxQuestLevel || out.Name == "" {
		return out, false
	}
	if out.MinLevel < 1 {
		out.MinLevel = 1
	}
	if out.XP == 0 {
		out.Sources["xp"] = "unknown"
	}
	out.Start = b.polestarRefs(b.pdb.Starts[p.ID], true)
	out.End = b.polestarRefs(b.pdb.Ends[p.ID], false)
	out.PreAll, out.PreAny, out.Breadcrumb = b.polestarPrereqs(p.ID, 0)
	b.polestarObjectives(&out)
	if len(out.Objectives) == 0 && out.Unsupported == "" && len(b.pdb.Objectives[p.ID]) == 0 {
		// the client lists no objective: a delivery or talk quest, done by turning it in
		b.stats["polestar_quests_without_objectives"]++
	}
	b.startItem(&out)
	if out.Unsupported != "" {
		b.stats["quests_unsupported"]++
	}
	b.stats["quests_from_polestar"]++
	return out, true
}

func (b *builder) polestarRefs(rs []polestar.Ref, withItem bool) []dataset.Ref {
	var out []dataset.Ref
	for _, r := range rs {
		switch r.Kind {
		case "npc", "object":
			out = append(out, dataset.Ref{Kind: r.Kind, ID: r.ID})
		case "item":
			if withItem {
				out = append(out, dataset.Ref{Kind: r.Kind, ID: r.ID})
			}
		}
	}
	sort.Slice(out, func(a, c int) bool {
		if out[a].Kind != out[c].Kind {
			return out[a].Kind < out[c].Kind
		}
		return out[a].ID < out[c].ID
	})
	return out
}

func (b *builder) polestarPrereqs(id, breadcrumb int) (all, anyOf []int, bc int) {
	bc = breadcrumb
	for _, p := range b.pdb.Prereqs[id] {
		if p.Breadcrumb {
			// a breadcrumb leads to the quest; it is not needed for it
			continue
		}
		if p.Mode == "all" {
			all = append(all, p.Pre)
		} else {
			anyOf = append(anyOf, p.Pre)
		}
	}
	sort.Ints(all)
	sort.Ints(anyOf)
	return all, anyOf, bc
}

// polestarObjectives sets a quest's objectives from forever-db: kills, talks and objects as they are, items with what
// drops them (forever-db, then cmangos and the other item sources). An event objective (explore, escort, use) is not
// modelled: the quest stays in the data, unsupported.
func (b *builder) polestarObjectives(out *dataset.Quest) {
	for _, o := range b.pdb.Objectives[out.ID] {
		if o.Count <= 0 {
			o.Count = 1
		}
		switch o.Kind {
		case "kill", "talk":
			if o.Kind == "kill" && o.Target <= 0 {
				b.unsupported(out, "kill objective without a target")
				continue
			}
			if o.Kind == "talk" && (o.Target <= 0 || len(b.pdb.Spawns["npc"][o.Target]) == 0 && len(b.w.spawns[o.Target]) == 0) {
				// a talk target no source places is an event in disguise (an area to reach): planned without it
				out.Sources["objectives"] = "event objective not modelled (" + polestarSrc + ")"
				b.stats["polestar_event_objectives_skipped"]++
				continue
			}
			out.Objectives = append(out.Objectives, dataset.Objective{Kind: o.Kind, ID: o.Target, Count: o.Count, Source: polestarSrc})
			b.npcWanted[o.Target] = true
		case "object":
			out.Objectives = append(out.Objectives, dataset.Objective{Kind: "object", ID: o.Target, Count: o.Count, Source: polestarSrc})
			b.objWanted[o.Target] = true
		case "item":
			if o.Target == b.providedItem(out.ID) || b.fromStartItem(out, o.Target) {
				continue // handed over with the quest (cmangos SrcItemId, or what the starting item becomes): nothing to collect
			}
			obj := dataset.Objective{Kind: "item", ID: o.Target, Count: o.Count, Source: polestarSrc}
			b.polestarDrops(&obj)
			if len(obj.Drops) == 0 {
				obj.Drops = append(obj.Drops, b.w.dropsByItem[o.Target]...)
			}
			keep, ok := b.moreItemSources(out, &obj)
			if !keep {
				continue
			}
			if !ok {
				b.unsupported(out, b.noSourceReason(o.Target, fmt.Sprintf("item %d: source unknown (crafted, bought or not in the data)", o.Target)))
			}
			for _, d := range obj.Drops {
				if d.NPC > 0 {
					b.npcWanted[d.NPC] = true
				} else if d.Object > 0 {
					b.objWanted[d.Object] = true
				}
			}
			out.Objectives = append(out.Objectives, obj)
		default:
			// an event (explore an area, escort someone, use something): its place is not in the data, so the quest
			// is planned without it, as the other sources have it; the note keeps it visible
			out.Sources["objectives"] = "event objective not modelled (" + polestarSrc + ")"
			b.stats["polestar_event_objectives_skipped"]++
		}
	}
}

// fromStartItem: a quest started by an item asks for an item of the same name that nothing drops: it is the starting
// item itself (the client gives the quest's own copy), not something to collect.
func (b *builder) fromStartItem(out *dataset.Quest, item int) bool {
	it := b.pdb.Items[item]
	if it == nil || len(b.pdb.ItemSources[item]) > 0 || len(b.w.dropsByItem[item]) > 0 {
		return false
	}
	for _, r := range b.pdb.Starts[out.ID] {
		if s := b.pdb.Items[r.ID]; r.Kind == "item" && s != nil && s.Name == it.Name {
			return true
		}
	}
	return false
}

func without(l []int, v int) []int {
	out := l[:0:0]
	for _, x := range l {
		if x != v {
			out = append(out, x)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// providedItem is the item the giver hands over with the quest (cmangos SrcItemId), 0 if none or unknown.
func (b *builder) providedItem(quest int) int {
	if cm := b.w.quest[quest]; cm != nil {
		return b.w.qt.Int(cm, "SrcItemId")
	}
	return 0
}

func (b *builder) unsupported(out *dataset.Quest, why string) {
	if out.Unsupported == "" {
		out.Unsupported = why
	}
}

// polestarDrops adds what forever-db says drops an item: creatures and objects, with the chance where the loot table
// gives one (else marked unknown).
func (b *builder) polestarDrops(o *dataset.Objective) bool {
	if b.pdb == nil || len(o.Drops) > 0 {
		return false
	}
	for _, s := range b.pdb.ItemSources[o.ID] {
		d := dataset.Drop{}
		switch s.Kind {
		case "npc":
			d.NPC = s.ID
		case "object":
			d.Object = s.ID
		default:
			continue
		}
		if s.Chance != nil && *s.Chance > 0 {
			d.Chance = round1(*s.Chance)
		} else {
			d.Unknown = true
		}
		o.Drops = append(o.Drops, d)
	}
	sort.Slice(o.Drops, func(a, c int) bool {
		if o.Drops[a].Chance != o.Drops[c].Chance {
			return o.Drops[a].Chance > o.Drops[c].Chance
		}
		return o.Drops[a].NPC+o.Drops[a].Object < o.Drops[c].NPC+o.Drops[c].Object
	})
	if len(o.Drops) > 12 {
		o.Drops = o.Drops[:12]
	}
	if len(o.Drops) > 0 {
		o.Source += "+polestar_drops"
		b.stats["items_drops_from_polestar"]++
		return true
	}
	return false
}

// polestarNPC fills an NPC the other sources do not know or do not place.
func (b *builder) polestarNPC(n *dataset.NPC, known bool) bool {
	p := b.pdb.NPCs[n.ID]
	if !known {
		if p == nil {
			return false
		}
		n.Name, n.MinLevel, n.MaxLevel = p.Name, p.LevelMin, p.LevelMax
		if p.Rank != nil {
			n.Rank = *p.Rank
		}
	}
	if len(n.Spawns) == 0 {
		for _, s := range b.pdb.Spawns["npc"][n.ID] {
			n.Spawns = append(n.Spawns, dataset.Point{Map: s.Map, X: s.X, Y: s.Y})
		}
		if len(n.Spawns) > 0 {
			n.Source = "polestar_positions"
			b.stats["npcs_placed_by_polestar"]++
		}
	}
	return true
}

func (b *builder) polestarObject(o *dataset.Object) {
	if o.Name == "" {
		if p := b.pdb.Objects[o.ID]; p != nil {
			o.Name = p.Name
		}
	}
	if len(o.Spawns) == 0 {
		for _, s := range b.pdb.Spawns["object"][o.ID] {
			o.Spawns = append(o.Spawns, dataset.Point{Map: s.Map, X: s.X, Y: s.Y})
		}
		if len(o.Spawns) > 0 {
			b.stats["objects_placed_by_polestar"]++
		}
	}
}

// skyborneExits: the Skyborne leave Zephras Isle once, the Horde by airship to northern Mulgore, the Alliance by
// skycutter to the Kirin Tor camp in the Alterac Mountains and on through the Skyborne Portal to Stormwind (Polestar's
// Skyborne guides). The board points on the isle are the guides' map positions (uiMap 2521); a landing is the NPC who
// waits there (forever-db spawns). One way: the guides never go back. The seconds are an estimate of wait plus ride;
// no timetable of these ships is read here.
var skyborneExits = []struct {
	name, faction string
	from          int     // node id: an NPC, or a negative id for a board point without one
	ui            int     // board point on this in-game map, when from has no spawn
	x, y          float64 // map percent
	object        int     // or the board point is this object's spawn
	to            int     // NPC at the landing
	seconds       float64
}{
	{name: "Airship Zephras Isle - Mulgore", faction: "H", from: -2991001, ui: 2521, x: 57.92, y: 80.78, to: 259119, seconds: 240},
	{name: "Skycutter Zephras Isle - Alterac Mountains", faction: "A", from: -2991002, ui: 2521, x: 65.81, y: 83.44, to: 259084, seconds: 240},
	{name: "Skyborne Portal to Stormwind", faction: "A", from: -631299, object: 631299, to: 1748, seconds: 20},
}

// uiToWorld turns a point in map percent into world yards with forever-db's map frames.
func (b *builder) uiToWorld(ui int, x, y float64) (dataset.Point, bool) {
	for _, m := range b.pdb.UIMaps {
		if m.UIMap == ui {
			return dataset.Point{Map: m.Map, X: round1(m.Top + (m.Bottom-m.Top)*y/100), Y: round1(m.Left + (m.Right-m.Left)*x/100)}, true
		}
	}
	return dataset.Point{}, false
}

func (b *builder) npcPoint(id int) (dataset.Point, bool) {
	if sp := b.w.spawns[id]; len(sp) > 0 {
		return sp[0], true
	}
	if sp := b.pdb.Spawns["npc"][id]; len(sp) > 0 {
		return dataset.Point{Map: sp[0].Map, X: sp[0].X, Y: sp[0].Y}, true
	}
	return dataset.Point{}, false
}

// polestarTravel adds the Skyborne exits to the travel network.
func (b *builder) polestarTravel(t *dataset.Travel) {
	have := map[int]bool{}
	for _, d := range t.Docks {
		have[d.NPC] = true
	}
	add := func(id int, name, faction string, at dataset.Point) {
		if !have[id] {
			t.Docks = append(t.Docks, dataset.Node{NPC: id, Name: name, Faction: faction, At: at})
			have[id] = true
		}
	}
	for _, e := range skyborneExits {
		var from dataset.Point
		var ok bool
		switch {
		case e.object > 0:
			if sp := b.pdb.Spawns["object"][e.object]; len(sp) > 0 {
				from, ok = dataset.Point{Map: sp[0].Map, X: sp[0].X, Y: sp[0].Y}, true
			}
		default:
			from, ok = b.uiToWorld(e.ui, e.x, e.y)
		}
		to, okTo := b.npcPoint(e.to)
		if !ok || !okTo {
			b.stats["skyborne_exits_unplaced"]++
			continue
		}
		add(e.from, e.name+" (board)", e.faction, from)
		add(e.to, e.name+" (landing)", e.faction, to)
		t.Links = append(t.Links, dataset.Link{From: e.from, To: e.to, Seconds: e.seconds, Name: e.name})
		b.stats["skyborne_exits"]++
	}
}
