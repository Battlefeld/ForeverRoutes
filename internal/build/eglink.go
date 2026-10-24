package build

import (
	"math"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// EG Link (internal/sources/eglink) is what players' games saw: drops with the chance per kill counted while on the
// quest, and where objective counts went up. A measured chance replaces the planner's assumed one for a drop no loot
// table gives; an item no database explains gets its measured sources, else the places where players advanced the quest.

func round4(v float64) float64 { return math.Round(v*1e4) / 1e4 }

// fromEGLink gives an item objective without drops the creatures and objects players saw drop it.
func (b *builder) fromEGLink(o *dataset.Objective) bool {
	if b.egl == nil || len(o.Drops) > 0 {
		return false
	}
	for _, d := range b.egl.Drops[o.ID] {
		if len(o.Drops) >= 12 {
			break
		}
		dr := dataset.Drop{Chance: round1(d.Chance * 100)}
		if dr.Chance <= 0 {
			continue
		}
		if d.Kind == "npc" {
			dr.NPC = d.ID
			b.npcWanted[d.ID] = true
		} else {
			dr.Object = d.ID
			b.objWanted[d.ID] = true
		}
		o.Drops = append(o.Drops, dr)
	}
	if len(o.Drops) == 0 {
		return false
	}
	o.Source += "+eglink"
	b.stats["items_sources_from_eglink"]++
	return true
}

// fromEGLinkPlaces gives an item objective nothing else explains the places where players advanced the quest.
func (b *builder) fromEGLinkPlaces(out *dataset.Quest, o *dataset.Objective) bool {
	if b.egl == nil {
		return false
	}
	var pts []dataset.Point
	for n := 1; n <= 8; n++ {
		for _, p := range b.egl.Objectives[out.ID][n] {
			pts = append(pts, dataset.Point{Map: p.Map, X: round1(p.X), Y: round1(p.Y)})
		}
	}
	if len(pts) == 0 {
		return false
	}
	o.Places = dedupePoints(pts)
	o.Source += "+eglink"
	b.stats["items_places_from_eglink"]++
	return true
}

// egChances puts measured chances on drops whose chance no loot table gives.
func (b *builder) egChances() {
	if b.egl == nil {
		return
	}
	for i := range b.out {
		for j := range b.out[i].Objectives {
			o := &b.out[i].Objectives[j]
			for k := range o.Drops {
				dr := &o.Drops[k]
				if !dr.Unknown {
					continue
				}
				for _, d := range b.egl.Drops[o.ID] {
					if d.Kind == "npc" && d.ID == dr.NPC || d.Kind == "object" && d.ID == dr.Object {
						if c := round1(d.Chance * 100); c > 0 {
							dr.Chance, dr.Unknown = c, false
							b.stats["drop_chances_from_eglink"]++
						}
					}
				}
			}
		}
	}
}
