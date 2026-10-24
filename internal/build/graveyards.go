package build

import (
	"math"
	"sort"
	"strconv"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Deathskips need two things: the graveyards (where a released ghost appears; only those with a Spirit Healer, who
// resurrects there) and which graveyard the server gives for a death at a place. CMaNGOS knows both for the classic
// world (world_safe_locs, game_graveyard_zone: the graveyards linked to the zone of the death, for the faction; of
// those the closest). The zone of a place comes from the creatures standing around it (QuestieDB's spawns are filed
// by zone). Graveyards Forever added (a Spirit Healer far from every classic one) serve the places nearest to them.

const (
	deathCell   = 200.0 // yards per cell of the death grid
	healerReach = 100.0 // a graveyard's Spirit Healer stands this close
)

func (b *builder) spiritHealers() []dataset.Point {
	var out []dataset.Point
	for _, r := range b.w.ct.Rows {
		if b.w.ct.String(r, "Name") == "Spirit Healer" {
			out = append(out, b.w.spawns[b.w.ct.Int(r, "Entry")]...)
		}
	}
	if b.pdb != nil {
		ids := make([]int, 0)
		for id, n := range b.pdb.NPCs {
			if n.Name == "Spirit Healer" {
				ids = append(ids, id)
			}
		}
		sort.Ints(ids)
		for _, id := range ids {
			for _, s := range b.pdb.Spawns["npc"][id] {
				out = append(out, dataset.Point{Map: s.Map, X: round1(s.X), Y: round1(s.Y)})
			}
		}
	}
	return out
}

func near(pts []dataset.Point, p dataset.Point, r float64) (dataset.Point, bool) {
	best, bd := dataset.Point{}, math.Inf(1)
	for _, q := range pts {
		if q.Map == p.Map {
			if d := math.Hypot(q.X-p.X, q.Y-p.Y); d < bd {
				best, bd = q, d
			}
		}
	}
	return best, bd <= r
}

func (b *builder) graveyards(t *dataset.Travel) {
	if b.w.safeLocs == nil || b.w.graveZones == nil {
		return
	}
	healers := b.spiritHealers()
	type grave struct {
		node    dataset.Node
		zones   map[int]string // zone -> "A", "H" or "" (both)
		classic bool
	}
	var graves []*grave
	byLoc := map[int]*grave{}
	sl := b.w.safeLocs
	for _, r := range sl.Rows {
		m := sl.Int(r, "map")
		if !overland[m] {
			continue
		}
		at := dataset.Point{Map: m, X: round1(sl.Float(r, "x")), Y: round1(sl.Float(r, "y"))}
		h, ok := near(healers, at, healerReach)
		if !ok {
			continue
		}
		g := &grave{node: dataset.Node{NPC: sl.Int(r, "id"), Name: sl.String(r, "name"), At: h}, zones: map[int]string{}, classic: true}
		byLoc[g.node.NPC] = g
		graves = append(graves, g)
	}
	gz := b.w.graveZones
	for _, r := range gz.Rows {
		g := byLoc[gz.Int(r, "id")]
		if g == nil || gz.Int(r, "link_kind") != 0 {
			continue
		}
		f := ""
		switch gz.Int(r, "faction") {
		case 469:
			f = "A"
		case 67:
			f = "H"
		}
		zone := gz.Int(r, "ghost_loc")
		if old, ok := g.zones[zone]; ok && old != f {
			f = ""
		}
		g.zones[zone] = f
	}
	// Forever's own graveyards: a Spirit Healer with no classic graveyard near it
	extra := 0
	for _, h := range healers {
		if !overland[h.Map] {
			continue
		}
		taken := false
		for _, g := range graves {
			if g.node.At.Map == h.Map && math.Hypot(g.node.At.X-h.X, g.node.At.Y-h.Y) <= 2*healerReach {
				taken = true
				break
			}
		}
		if !taken {
			extra++
			graves = append(graves, &grave{node: dataset.Node{NPC: -(5000 + extra), Name: "Spirit Healer", At: h}, zones: map[int]string{}})
		}
	}
	sort.SliceStable(graves, func(a, c int) bool { return graves[a].node.NPC < graves[c].node.NPC })
	index := map[*grave]int{}
	for i, g := range graves {
		t.Graveyards = append(t.Graveyards, g.node)
		index[g] = i + 1
	}
	b.stats["graveyards"] = len(graves)
	b.stats["graveyards_forever"] = extra

	// the zone of each cell: the zone most of the creature spawns in it are filed under
	type key struct{ m, i, j int }
	votes := map[key]map[int]int{}
	ids := make([]string, 0)
	if npcs, ok := b.merged["npcs"].(map[string]any); ok {
		for id := range npcs {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	for _, id := range ids {
		n, _ := strconv.Atoi(id)
		rec := b.npcRecord(n)
		sp, _ := rec["spawns"].(map[string]any)
		for zs, list := range sp {
			zone, err := strconv.Atoi(zs)
			t, ok := b.zoneT[zs]
			if err != nil || !ok {
				continue
			}
			l, _ := list.([]any)
			for _, raw := range l {
				xy, _ := raw.([]any)
				if len(xy) != 2 {
					continue
				}
				px, _ := num(xy[0])
				py, _ := num(xy[1])
				p := t.apply(px, py)
				if !overland[p.Map] {
					continue
				}
				k := key{p.Map, int(math.Floor(p.X / deathCell)), int(math.Floor(p.Y / deathCell))}
				if votes[k] == nil {
					votes[k] = map[int]int{}
				}
				votes[k][zone]++
			}
		}
	}
	// cells: those with creatures (elsewhere the planner takes the nearest graveyard)
	cells := map[key]int{} // -> zone (0 unknown)
	for k, v := range votes {
		best, bn := 0, 0
		for z, n := range v {
			if n > bn || n == bn && z < best {
				best, bn = z, n
			}
		}
		cells[k] = best
	}
	pick := func(m int, x, y float64, zone int, faction string) int {
		best, bd := 0, math.Inf(1)
		linked := false
		for _, g := range graves {
			if f, ok := g.zones[zone]; ok && g.node.At.Map == m && (f == "" || f == faction) {
				linked = true
				break
			}
		}
		for _, g := range graves {
			if g.node.At.Map != m {
				continue
			}
			if linked {
				f, ok := g.zones[zone]
				if !ok || f != "" && f != faction {
					continue
				}
			}
			if d := math.Hypot(g.node.At.X-x, g.node.At.Y-y); d < bd {
				best, bd = index[g], d
			}
		}
		return best
	}
	keys := make([]key, 0, len(cells))
	for k := range cells {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, c int) bool {
		if keys[a].m != keys[c].m {
			return keys[a].m < keys[c].m
		}
		if keys[a].i != keys[c].i {
			return keys[a].i < keys[c].i
		}
		return keys[a].j < keys[c].j
	})
	dg := &dataset.DeathGrid{Cell: deathCell}
	for _, k := range keys {
		x, y := (float64(k.i)+0.5)*deathCell, (float64(k.j)+0.5)*deathCell
		a, h := pick(k.m, x, y, cells[k], "A"), pick(k.m, x, y, cells[k], "H")
		if a != 0 || h != 0 {
			dg.Cells = append(dg.Cells, k.m, k.i, k.j, a, h)
		}
	}
	if len(dg.Cells) > 0 {
		t.Deaths = dg
	}
	b.stats["death_cells"] = len(dg.Cells) / 5
}
