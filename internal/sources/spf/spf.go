// Package spf reads the travel data of Shortest Path Forever (https://github.com/cjber/shortest-path-forever,
// GPL-3.0-or-later) by Cillian Berragan: the flight paths of WoW Forever with their durations (measured in game by
// InFlight for most, MIT), the boats and zeppelins with their timetables (from the client's transport animations and
// cmangos), and the portals. Read from its Data/*.lua files as Lua tables; no code is run.
package spf

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/internal/lua"
)

// Repository is the upstream project.
const Repository = "cjber/shortest-path-forever"

// Point is a position in world yards (x north, y west) on a map.
type Point struct {
	Map  int
	X, Y float64
}

// TaxiNode is a flight point.
type TaxiNode struct {
	ID      int
	At      Point
	Name    string
	Faction string // Alliance | Horde | "" (both)
}

// TaxiPath is a direct flight between two flight points, one way.
type TaxiPath struct {
	From, To int
	Seconds  float64
}

// Stop is where a transport stops: its dock, and when it arrives and leaves, in seconds into the loop.
type Stop struct {
	Dock           int
	Arrive, Depart float64
}

// Route is a boat or zeppelin line.
type Route struct {
	ID      int
	Kind    string
	Faction string
	Period  float64 // seconds per loop
	Stops   []Stop
}

// Portal is a one-way portal or passage.
type Portal struct {
	Name     string
	Kind     string
	Faction  string
	From, To Point
	Seconds  float64
}

// Lift is a lift's site: its stops (where it runs, top and bottom alike in x and y), the ride and the loop of each
// car, and how many cars run there.
type Lift struct {
	Site   string
	At     Point
	Ride   float64 // seconds from leaving one end to reaching the other
	Period float64 // seconds per loop of one car
	Cars   int
}

// Zone is a zone of Zones.lua: its level range, or the side whose capital it is, and the box its map covers (boxes
// overlap; a box bounds where the zone can be).
type Zone struct {
	UIMap      int
	Low, High  int
	City       string // Alliance | Horde for a capital
	Map        int
	MinX, MinY float64
	MaxX, MaxY float64
}

// Data is the travel data.
type Data struct {
	Revision string
	Nodes    map[int]TaxiNode
	Paths    []TaxiPath
	Docks    map[int]Point // dock -> where passengers board (the dry boarding surface when known)
	Routes   []Route
	Portals  []Portal
	Lifts    []Lift
	Zones    []Zone
}

// Load reads dir (the repository's Data folder).
func Load(dir, revision string) (*Data, error) {
	d := &Data{Revision: revision, Nodes: map[int]TaxiNode{}, Docks: map[int]Point{}}
	taxi, err := blocks(filepath.Join(dir, "Taxi.lua"), "TaxiNodes", "TaxiPaths")
	if err != nil {
		return nil, err
	}
	if nodes, ok := taxi["TaxiNodes"].(lua.Table); ok {
		for _, k := range lua.SortedKeys(nodes) {
			id, _ := lua.Int(k)
			t, _ := nodes[k].(lua.Table)
			if id > 0 && t != nil {
				d.Nodes[id] = TaxiNode{ID: id, At: point(t), Name: str(t["name"]), Faction: str(t["faction"])}
			}
		}
	}
	for _, v := range list(taxi["TaxiPaths"]) {
		t, _ := v.(lua.Table)
		from, _ := lua.Int(t["from"])
		to, _ := lua.Int(t["to"])
		if s := num(t["seconds"]); from > 0 && to > 0 && s > 0 {
			d.Paths = append(d.Paths, TaxiPath{From: from, To: to, Seconds: s})
		}
	}
	routes, err := blocks(filepath.Join(dir, "Routes.lua"), "Docks", "Routes")
	if err != nil {
		return nil, err
	}
	for i, v := range list(routes["Docks"]) {
		t, _ := v.(lua.Table)
		if t == nil {
			continue
		}
		p := point(t)
		if w, ok := t["walk"].(lua.Table); ok {
			p = point(w)
		}
		d.Docks[i+1] = p
	}
	if rs, ok := routes["Routes"].(lua.Table); ok {
		for _, k := range lua.SortedKeys(rs) {
			id, _ := lua.Int(k)
			t, _ := rs[k].(lua.Table)
			if t == nil {
				continue
			}
			r := Route{ID: id, Kind: str(t["kind"]), Faction: str(t["faction"]), Period: num(t["period"]) / 1000}
			for _, sv := range list(t["stops"]) {
				st, _ := sv.(lua.Table)
				dock, _ := lua.Int(st["dock"])
				r.Stops = append(r.Stops, Stop{Dock: dock, Arrive: num(st["arrive"]) / 1000, Depart: num(st["depart"]) / 1000})
			}
			if r.Period > 0 && len(r.Stops) >= 2 {
				d.Routes = append(d.Routes, r)
			}
		}
	}
	portals, err := blocks(filepath.Join(dir, "Portals.lua"), "Portals")
	if err != nil {
		return nil, err
	}
	for _, v := range list(portals["Portals"]) {
		t, _ := v.(lua.Table)
		from, _ := t["from"].(lua.Table)
		to, _ := t["to"].(lua.Table)
		if from == nil || to == nil {
			continue
		}
		d.Portals = append(d.Portals, Portal{Name: str(t["name"]), Kind: str(t["kind"]), Faction: str(t["faction"]),
			From: point(from), To: point(to), Seconds: num(t["seconds"])})
	}
	zones, err := blocks(filepath.Join(dir, "Zones.lua"), "Zones")
	if err != nil {
		return nil, err
	}
	if zt, ok := zones["Zones"].(lua.Table); ok {
		for _, k := range lua.SortedKeys(zt) {
			ui, _ := lua.Int(k)
			t, _ := zt[k].(lua.Table)
			if t == nil {
				continue
			}
			lo, _ := lua.Int(t["low"])
			hi, _ := lua.Int(t["high"])
			m, _ := lua.Int(t["map"])
			d.Zones = append(d.Zones, Zone{UIMap: ui, Low: lo, High: hi, City: str(t["city"]), Map: m,
				MinX: num(t["minX"]), MinY: num(t["minY"]), MaxX: num(t["maxX"]), MaxY: num(t["maxY"])})
		}
	}
	if err := d.readLifts(filepath.Join(dir, "Transports.lua")); err != nil {
		return nil, err
	}
	if len(d.Nodes) == 0 || len(d.Paths) == 0 {
		return nil, fmt.Errorf("spf: no flight data in %s", dir)
	}
	return d, nil
}

// Ride is the time from boarding at stop a to arriving at stop b of a route, with the average wait for the departure
// (half the loop: a traveller arrives at a random moment).
func (r Route) Ride(a, b int) float64 {
	dep, arr := r.Stops[a].Depart, r.Stops[b].Arrive
	t := arr - dep
	for t <= 0 {
		t += r.Period
	}
	return t + r.Period/2
}

// blocks reads the tables assigned to ns.<name> in a Lua file.
func blocks(path string, names ...string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	src := stripComments(string(b))
	out := map[string]any{}
	for _, name := range names {
		i := strings.Index(src, "ns."+name+" = {")
		if i < 0 {
			return nil, fmt.Errorf("spf: %s: no ns.%s", filepath.Base(path), name)
		}
		start := i + len("ns."+name+" = ")
		end, err := matching(src, start)
		if err != nil {
			return nil, fmt.Errorf("spf: %s: ns.%s: %w", filepath.Base(path), name, err)
		}
		v, err := lua.Parse(src[start:end])
		if err != nil {
			return nil, fmt.Errorf("spf: %s: ns.%s: %w", filepath.Base(path), name, err)
		}
		out[name] = v
	}
	// Transports.lua style single assignments are not read; Routes.lua holds the docks
	return out, nil
}

// matching is the index just past the brace that closes the one at start, skipping strings.
func matching(s string, start int) (int, error) {
	depth := 0
	for i := start; i < len(s); i++ {
		switch c := s[i]; c {
		case '"', '\'':
			for i++; i < len(s) && s[i] != c; i++ {
				if s[i] == '\\' {
					i++
				}
			}
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1, nil
			}
		}
	}
	return 0, fmt.Errorf("unbalanced braces")
}

// stripComments removes Lua line comments ("--" to the end of the line) outside strings.
func stripComments(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"' || c == '\'':
			j := i + 1
			for ; j < len(s) && s[j] != c && s[j] != '\n'; j++ {
				if s[j] == '\\' {
					j++
				}
			}
			if j >= len(s) {
				j = len(s) - 1
			}
			b.WriteString(s[i : j+1])
			i = j
		case c == '-' && i+1 < len(s) && s[i+1] == '-':
			for i < len(s) && s[i] != '\n' {
				i++
			}
			if i < len(s) {
				b.WriteByte('\n')
			}
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

func list(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case lua.Table:
		keys := lua.SortedKeys(t)
		sort.SliceStable(keys, func(a, b int) bool {
			x, _ := lua.Int(keys[a])
			y, _ := lua.Int(keys[b])
			return x < y
		})
		var out []any
		for _, k := range keys {
			if _, ok := lua.Int(k); ok {
				out = append(out, t[k])
			}
		}
		return out
	}
	return nil
}

func point(t lua.Table) Point {
	m, _ := lua.Int(t["map"])
	return Point{Map: m, X: num(t["x"]), Y: num(t["y"])}
}

func num(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case float64:
		return n
	}
	return 0
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

var (
	liftDock  = regexp.MustCompile(`(?s)ns\.Docks\[(\d+)\] = (\{.*?\n?\})\n`)
	liftRoute = regexp.MustCompile(`(?s)ns\.Routes\[(\d+)\] = (\{.*?\n\})\n`)
)

// readLifts reads the lifts of Transports.lua (single ns.Docks[...] and ns.Routes[...] assignments).
func (d *Data) readLifts(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	src := stripComments(string(b))
	docks := map[int]Point{}
	for _, m := range liftDock.FindAllStringSubmatch(src, -1) {
		id, _ := strconv.Atoi(m[1])
		v, err := lua.Parse(m[2])
		if err != nil {
			return fmt.Errorf("spf: Transports.lua dock %d: %w", id, err)
		}
		if t, ok := v.(lua.Table); ok {
			docks[id] = point(t)
		}
	}
	bySite := map[string]*Lift{}
	var order []string
	for _, m := range liftRoute.FindAllStringSubmatch(src, -1) {
		end, err := matching(m[2], 0)
		if err != nil {
			continue
		}
		v, err := lua.Parse(m[2][:end])
		if err != nil {
			return fmt.Errorf("spf: Transports.lua route %s: %w", m[1], err)
		}
		t, _ := v.(lua.Table)
		if t == nil || str(t["kind"]) != "lift" {
			continue
		}
		stops := list(t["stops"])
		if len(stops) != 2 {
			continue
		}
		a, _ := stops[0].(lua.Table)
		c, _ := stops[1].(lua.Table)
		dock, _ := lua.Int(a["dock"])
		site := str(t["site"])
		l := bySite[site]
		if l == nil {
			l = &Lift{Site: site, At: docks[dock], Ride: (num(c["arrive"]) - num(a["depart"])) / 1000, Period: num(t["period"]) / 1000}
			bySite[site] = l
			order = append(order, site)
		}
		l.Cars++
	}
	sort.Strings(order)
	for _, s := range order {
		d.Lifts = append(d.Lifts, *bySite[s])
	}
	return nil
}
