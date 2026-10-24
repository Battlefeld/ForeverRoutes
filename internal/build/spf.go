package build

import (
	"math"
	"sort"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/spf"
)

// Shortest Path Forever's travel data (internal/sources/spf) replaces the estimates of the travel network: flights
// take their measured time between neighbouring flight points (the planner chains them), boats and zeppelins take the
// ride from their timetable plus half a loop of waiting, and the portals and the Deeprun Tram join the network.

const (
	// spfNear: a dock or flight point of Shortest Path Forever this close to one of ours is the same place.
	spfNear = 120
	// tramSeconds: the Deeprun Tram between the Stormwind and Ironforge entrances: two trains on a 143 s loop
	// (Transports.lua) give about 36 s of waiting, the ride about 60 s, and a loading screen at each end.
	tramSeconds = 5 + 36 + 60 + 5
)

func spfFaction(f string) string {
	switch f {
	case "Alliance":
		return "A"
	case "Horde":
		return "H"
	}
	return ""
}

func spfPoint(p spf.Point) dataset.Point {
	return dataset.Point{Map: p.Map, X: round1(p.X), Y: round1(p.Y)}
}

// overland maps the planner reaches: the two continents and Zephras Isle.
var overland = map[int]bool{0: true, 1: true, 2991: true}

func (b *builder) spfTravel(t *dataset.Travel) {
	d := b.spf
	near := func(nodes []dataset.Node, p dataset.Point) int {
		best, bestD := 0, math.Inf(1)
		for _, n := range nodes {
			if n.At.Map == p.Map {
				if v := math.Hypot(n.At.X-p.X, n.At.Y-p.Y); v < bestD && v <= spfNear {
					best, bestD = n.NPC, v
				}
			}
		}
		return best
	}
	// flights: SPF's flight points onto our flight masters; a point we lack gets the flight master forever-db places
	// there, else a node of its own
	fm := map[int]int{} // SPF node -> flight master NPC
	ids := make([]int, 0, len(d.Nodes))
	for id := range d.Nodes {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		n := d.Nodes[id]
		at := spfPoint(n.At)
		if !overland[at.Map] {
			continue
		}
		npc := near(t.FlightMasters, at)
		if npc == 0 {
			npc = b.flightMasterAt(at)
			name := n.Name
			if npc == 0 {
				npc = -(3000 + id)
			}
			for _, f := range []string{"A", "H"} {
				if fac := spfFaction(n.Faction); fac == "" || fac == f {
					t.FlightMasters = append(t.FlightMasters, dataset.Node{NPC: npc, Name: name, Faction: f, At: at})
				}
			}
			b.stats["flight_masters_from_spf"]++
		}
		fm[id] = npc
	}
	sort.Slice(t.FlightMasters, func(a, c int) bool {
		if t.FlightMasters[a].NPC != t.FlightMasters[c].NPC {
			return t.FlightMasters[a].NPC < t.FlightMasters[c].NPC
		}
		return t.FlightMasters[a].Faction < t.FlightMasters[c].Faction
	})
	for _, p := range d.Paths {
		a, okA := fm[p.From]
		c, okC := fm[p.To]
		if okA && okC && a != c {
			t.Flights = append(t.Flights, dataset.Link{From: a, To: c, Seconds: math.Round(p.Seconds),
				Name: "Flight " + d.Nodes[p.From].Name + " - " + d.Nodes[p.To].Name})
		}
	}
	sort.Slice(t.Flights, func(a, c int) bool {
		if t.Flights[a].From != t.Flights[c].From {
			return t.Flights[a].From < t.Flights[c].From
		}
		return t.Flights[a].To < t.Flights[c].To
	})
	b.stats["flights_from_spf"] = len(t.Flights)

	// boats and zeppelins: our docks where they are the same place, else SPF's
	dockNPC := map[int]int{}
	node := func(dock int, kind, faction string) int {
		if npc, ok := dockNPC[dock]; ok {
			return npc
		}
		at := spfPoint(d.Docks[dock])
		npc := near(t.Docks, at)
		if npc == 0 {
			npc = -(1000 + dock)
			t.Docks = append(t.Docks, dataset.Node{NPC: npc, Name: kind + " dock", Faction: faction, At: at})
		}
		dockNPC[dock] = npc
		return npc
	}
	link := func(l dataset.Link) {
		for i := range t.Links {
			if t.Links[i].From == l.From && t.Links[i].To == l.To {
				t.Links[i] = l
				b.stats["transport_times_from_spf"]++
				return
			}
		}
		t.Links = append(t.Links, l)
		b.stats["transport_links_from_spf"]++
	}
	for _, r := range d.Routes {
		fac := spfFaction(r.Faction)
		kind := strings.ToUpper(r.Kind[:1]) + r.Kind[1:]
		for i := range r.Stops {
			for j := range r.Stops {
				if i == j || !overland[d.Docks[r.Stops[i].Dock].Map] || !overland[d.Docks[r.Stops[j].Dock].Map] {
					continue
				}
				a, c := node(r.Stops[i].Dock, kind, fac), node(r.Stops[j].Dock, kind, fac)
				link(dataset.Link{From: a, To: c, Seconds: math.Round(r.Ride(i, j)), Name: kind + " (Shortest Path Forever timetable)"})
			}
		}
	}
	// portals between overland places, and the tram between the two passages to it
	var tram []dataset.Node
	for i, p := range d.Portals {
		from, to := spfPoint(p.From), spfPoint(p.To)
		fac := spfFaction(p.Faction)
		if overland[from.Map] && to.Map == 369 {
			tram = append(tram, dataset.Node{NPC: -(2000 + 2*i), Name: p.Name, Faction: fac, At: from})
		}
		if !overland[from.Map] || !overland[to.Map] {
			continue
		}
		a, c := -(2000 + 2*i), -(2001 + 2*i)
		t.Docks = append(t.Docks, dataset.Node{NPC: a, Name: p.Name, Faction: fac, At: from}, dataset.Node{NPC: c, Name: p.Name + " (arrival)", Faction: fac, At: to})
		link(dataset.Link{From: a, To: c, Seconds: p.Seconds, Name: p.Name})
	}
	if len(tram) == 2 {
		t.Docks = append(t.Docks, tram...)
		link(dataset.Link{From: tram[0].NPC, To: tram[1].NPC, Seconds: tramSeconds, Name: "Deeprun Tram"})
		link(dataset.Link{From: tram[1].NPC, To: tram[0].NPC, Seconds: tramSeconds, Name: "Deeprun Tram"})
	}
}

var flightTitles = map[string]bool{"flight master": true, "gryphon master": true, "wind rider master": true, "hippogryph master": true, "bat handler": true}

// flightMasterAt is the flight master forever-db places at a point, 0 if none.
func (b *builder) flightMasterAt(at dataset.Point) int {
	if b.pdb == nil {
		return 0
	}
	best, bestD := 0, math.Inf(1)
	for id, n := range b.pdb.NPCs {
		if !flightTitles[strings.ToLower(n.Subname)] {
			continue
		}
		for _, s := range b.pdb.Spawns["npc"][id] {
			if s.Map == at.Map {
				if v := math.Hypot(s.X-at.X, s.Y-at.Y); v < bestD && v <= spfNear || v == bestD && id < best {
					best, bestD = id, v
				}
			}
		}
	}
	return best
}
