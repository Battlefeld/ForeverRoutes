package build

import (
	"strings"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/wclbox"
)

// WCLBox's quest pages (data from Wowhead's Forever database) say per item a quest asks for what it comes from or that
// the giver provides it, and mark on the zone's map where the objectives are. They fill items no database explains:
// the creatures named near the marks become drops of unknown chance, objects named there drops at 100 %, else the marks
// become the objective's places. Names the page gives in a translation that matches nothing only leave the marks.

// wclQuestItem is what the quest page says about an item; known tells whether the page lists the item at all.
func (b *builder) wclQuestItem(quest, item int) (wclbox.QuestItem, bool) {
	if b.wcl == nil {
		return wclbox.QuestItem{}, false
	}
	for _, it := range b.wcl.Quests[quest].Items {
		if it.ID == item {
			return it, true
		}
	}
	return wclbox.QuestItem{}, false
}

func (b *builder) fromWCLBoxQuest(out *dataset.Quest, o *dataset.Objective) bool {
	it, known := b.wclQuestItem(out.ID, o.ID)
	if !known {
		return false
	}
	q := b.wcl.Quests[out.ID]
	var pts []dataset.Point
	for _, w := range q.Where {
		if p, ok := b.uiPoint(q.UIMap, w.X, w.Y); ok {
			pts = append(pts, p)
		}
	}
	var npcs, objs []int
	for _, name := range it.From {
		if strings.HasPrefix(name, "Vendor:") {
			b.itemWhy[o.ID] = "bought from a vendor (" + strings.TrimSpace(strings.TrimPrefix(name, "Vendor:")) + ", WCLBox)"
			return false
		}
		zone := b.uiZone(q.UIMap, pts)
		if len(pts) == 0 && zone == "" {
			continue
		}
		npcs = append(npcs, b.namedCreatures([]string{name}, pts, zone)...)
		objs = append(objs, b.namedObjects(name, pts, q.UIMap)...)
	}
	for _, id := range dedupe(sortedInts(npcs)) {
		o.Drops = append(o.Drops, dataset.Drop{NPC: id, Unknown: true})
		b.npcWanted[id] = true
	}
	for _, id := range dedupe(sortedInts(objs)) {
		o.Drops = append(o.Drops, dataset.Drop{Object: id, Chance: 100})
		b.objWanted[id] = true
	}
	if len(o.Drops) > 12 {
		o.Drops = o.Drops[:12]
	}
	if len(o.Drops) == 0 {
		if len(pts) == 0 {
			return false
		}
		o.Places = dedupePoints(pts)
	}
	o.Source += "+wclbox_quest"
	b.stats["items_sources_from_wclbox_quests"]++
	return true
}

// uiZone is the zone name of an in-game map, for pages that mark no points (then the map is all that says where).
func (b *builder) uiZone(ui int, pts []dataset.Point) string {
	if len(pts) > 0 || ui == 0 {
		return ""
	}
	for name, id := range b.zoneID {
		if id == ui {
			return name
		}
	}
	return ""
}

// namedObjects are the objects of that name standing near the points, or on the map when there are none.
func (b *builder) namedObjects(name string, near []dataset.Point, ui int) []int {
	if b.pdb == nil {
		return nil
	}
	var out []int
	for id, ob := range b.pdb.Objects {
		if normName(ob.Name) != normName(name) {
			continue
		}
		pts := append([]dataset.Point(nil), b.w.objSpawns[id]...)
		for _, s := range b.pdb.Spawns["object"][id] {
			pts = append(pts, dataset.Point{Map: s.Map, X: s.X, Y: s.Y})
		}
		zone := b.uiZone(ui, near)
		for _, p := range pts {
			if len(near) > 0 && closeTo([]dataset.Point{p}, near) || zone != "" && b.inZone(zone, p) {
				out = append(out, id)
				break
			}
		}
	}
	return sortedInts(out)
}
