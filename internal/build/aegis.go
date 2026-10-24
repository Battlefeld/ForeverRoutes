package build

import (
	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/aegis"
)

// AEGIS Pathfinder's player data (internal/sources/aegis) is what players saw in game. It fills gaps only: the places
// of an item objective nothing else explains, where a quest is taken or handed in when no source says, the XP of a
// quest no source gives. Where it overlaps with the client data it is a check: the XP agreements are counted.

// continents: points on a continent or world map are too coarse to plan with.
var continents = map[int]bool{947: true, 1414: true, 1415: true}

func (b *builder) aegisPoints(cells []aegis.Cell) []dataset.Point {
	var out []dataset.Point
	for _, c := range cells {
		if continents[c.UIMap] {
			continue
		}
		if p, ok := b.uiPoint(c.UIMap, c.X, c.Y); ok {
			out = append(out, p)
		}
	}
	return out
}

// fromAegis: an item objective no other source explains gets the places where players advanced the quest's objectives.
func (b *builder) fromAegis(out *dataset.Quest, o *dataset.Objective) bool {
	if b.aeg == nil || b.aeg.Quests[out.ID] == nil {
		return false
	}
	pts := b.aegisPoints(b.aeg.Quests[out.ID].ObjectiveCells())
	if len(pts) == 0 {
		return false
	}
	o.Places = dedupePoints(pts)
	o.Source += "+aegis"
	b.stats["items_places_from_aegis"]++
	return true
}

// aegisQuests fills start and end places and XP, and counts how often the observed XP agrees with the data.
func (b *builder) aegisQuests() {
	if b.aeg == nil {
		return
	}
	for i := range b.out {
		out := &b.out[i]
		q := b.aeg.Quests[out.ID]
		if q == nil {
			continue
		}
		place := func(m map[string]aegis.Place) *dataset.Point {
			p, ok := aegis.Best(m)
			if !ok || continents[p.UIMap] {
				return nil
			}
			if pt, ok := b.uiPoint(p.UIMap, p.X*100, p.Y*100); ok {
				return &pt
			}
			return nil
		}
		if out.StartAt == nil {
			if p := place(q.Pickups); p != nil {
				out.StartAt = p
				out.Sources["startAt"] = "aegis"
				b.stats["start_places_from_aegis"]++
			}
		}
		if out.EndAt == nil {
			if p := place(q.Turnins); p != nil {
				out.EndAt = p
				out.Sources["endAt"] = "aegis"
				b.stats["end_places_from_aegis"]++
			}
		}
		xp := q.MaxXP(out.Level)
		switch {
		case xp == 0:
		case out.XP == 0:
			out.XP = xp
			out.Sources["xp"] = "aegis"
			b.stats["xp_from_aegis"]++
		case xp == out.XP:
			b.stats["aegis_xp_agrees"]++
		default:
			b.stats["aegis_xp_differs"]++
		}
	}
}
