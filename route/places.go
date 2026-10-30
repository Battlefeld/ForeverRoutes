package route

import "github.com/Battlefeld/ForeverRoutes/dataset"

// Places lists every place a route of this character can walk from or to: the givers and enders of its quests, the
// sites of their objectives and starting items, the grinding sites, the flight masters, docks and portals, the
// graveyards and the guides' places to die, the dungeon entrances, the trainers for wands and professions and the
// start. The walk table (walktable.go) is measured between them.
func (w *World) Places() []dataset.Point {
	var out []dataset.Point
	add := func(p dataset.Point) {
		if p.Map >= 0 && (p.X != 0 || p.Y != 0) {
			out = append(out, p)
		}
	}
	add(w.Start)
	for _, q := range w.Quests {
		add(q.StartPos)
		add(q.EndPos)
		for _, o := range q.Objectives {
			for _, src := range o.Sources {
				add(src.Site.Center)
			}
		}
		for _, src := range q.StartSources {
			add(src.Site.Center)
		}
	}
	for i := range w.Farms {
		add(w.Farms[i].Site.Center)
	}
	for _, p := range w.net.nodes {
		add(p)
	}
	for _, g := range w.Data.Travel.Graveyards {
		add(g.At)
	}
	for _, k := range w.Data.Travel.Deathskips {
		add(k.At)
	}
	for _, d := range w.Dungeons {
		add(d.Entrance)
	}
	for _, p := range wandSites[w.Faction] {
		add(p)
	}
	for _, p := range profSites[w.Faction] {
		add(p)
	}
	return out
}
