package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/route"
)

// tableCombos: every race and class a route is made for (the Skyborne for each side).
var tableCombos = []route.Character{
	{Race: "Human", Class: "Warrior"}, {Race: "Human", Class: "Paladin"}, {Race: "Human", Class: "Rogue"}, {Race: "Human", Class: "Priest"},
	{Race: "Human", Class: "Mage"}, {Race: "Human", Class: "Warlock"}, {Race: "Human", Class: "Hunter"},
	{Race: "Dwarf", Class: "Warrior"}, {Race: "Dwarf", Class: "Paladin"}, {Race: "Dwarf", Class: "Hunter"}, {Race: "Dwarf", Class: "Rogue"},
	{Race: "Dwarf", Class: "Priest"}, {Race: "Dwarf", Class: "Shaman"},
	{Race: "Gnome", Class: "Warrior"}, {Race: "Gnome", Class: "Rogue"}, {Race: "Gnome", Class: "Mage"}, {Race: "Gnome", Class: "Warlock"},
	{Race: "Gnome", Class: "Priest"},
	{Race: "NightElf", Class: "Warrior"}, {Race: "NightElf", Class: "Hunter"}, {Race: "NightElf", Class: "Rogue"}, {Race: "NightElf", Class: "Priest"},
	{Race: "NightElf", Class: "Druid"},
	{Race: "Orc", Class: "Warrior"}, {Race: "Orc", Class: "Hunter"}, {Race: "Orc", Class: "Rogue"}, {Race: "Orc", Class: "Shaman"},
	{Race: "Orc", Class: "Warlock"}, {Race: "Orc", Class: "Mage"},
	{Race: "Undead", Class: "Warrior"}, {Race: "Undead", Class: "Rogue"}, {Race: "Undead", Class: "Priest"}, {Race: "Undead", Class: "Mage"},
	{Race: "Undead", Class: "Warlock"}, {Race: "Undead", Class: "Paladin"},
	{Race: "Tauren", Class: "Warrior"}, {Race: "Tauren", Class: "Hunter"}, {Race: "Tauren", Class: "Shaman"}, {Race: "Tauren", Class: "Druid"},
	{Race: "Troll", Class: "Warrior"}, {Race: "Troll", Class: "Hunter"}, {Race: "Troll", Class: "Rogue"}, {Race: "Troll", Class: "Shaman"},
	{Race: "Troll", Class: "Priest"}, {Race: "Troll", Class: "Mage"},
}

// tablePlace is a place of the walk table with the height a spawn there gives (NaN: none known).
type tablePlace struct {
	dataset.Point
	Z float64
}

// walk-places writes the places of every route (route.World.Places) with their heights from forever-db's spawns, and
// the pairs within -radius yards to measure: the input of tools/walktable/walktable.lua.
func cmdWalkPlaces(args []string) error {
	fs := newFlags("walk-places")
	path := fs.String("data", "data/routes-data.json.gz", "dataset")
	spawns := fs.String("spawns", "", "forever-db's data/spawn.jsonl (heights of the places)")
	radius := fs.Float64("radius", 3000, "pairs up to this straight distance (yards)")
	out := fs.String("out", "", "file to write")
	fs.Parse(args)
	if *spawns == "" || *out == "" {
		return fmt.Errorf("walk-places needs -spawns and -out")
	}
	d, err := dataset.Load(*path)
	if err != nil {
		return err
	}
	// the places, one per 2 yards
	seen := map[[3]int]bool{}
	var places []tablePlace
	for _, c := range append(tableCombos, route.Character{Race: "Skyborne", Faction: "A", Class: "Warrior"}, route.Character{Race: "Skyborne", Faction: "H", Class: "Warrior"}) {
		for _, cls := range map[bool][]string{true: {"Warrior", "Mage", "Hunter", "Priest", "Rogue", "Warlock", "Druid", "Shaman", "Paladin"}, false: {c.Class}}[c.Race == "Skyborne"] {
			c.Class = cls
			w, err := route.NewWorld(d, c, route.Options{Dungeons: true})
			if err != nil {
				continue
			}
			for _, p := range w.Places() {
				if p.Map != 0 && p.Map != 1 && p.Map != 2991 {
					continue
				}
				k := [3]int{p.Map, int(math.Round(p.X / 2)), int(math.Round(p.Y / 2))}
				if !seen[k] {
					seen[k] = true
					places = append(places, tablePlace{p, math.NaN()})
				}
			}
		}
	}
	sort.Slice(places, func(i, j int) bool {
		a, b := places[i], places[j]
		if a.Map != b.Map {
			return a.Map < b.Map
		}
		if a.X != b.X {
			return a.X < b.X
		}
		return a.Y < b.Y
	})
	// heights: the nearest spawn within 4 yards that has one, else (a site's centre among its spawns: a mine's
	// kobolds) the median of the spawns within 30 yards, so a place in a cave is not taken for the ground above it
	type sp struct{ x, y, z float64 }
	const near, around = 4.0, 30.0
	cells := map[[3]int][]sp{}
	f, err := os.Open(*spawns)
	if err != nil {
		return err
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		var s struct {
			Map int      `json:"map"`
			X   float64  `json:"x"`
			Y   float64  `json:"y"`
			Z   *float64 `json:"z"`
		}
		if json.Unmarshal(sc.Bytes(), &s) != nil || s.Z == nil {
			continue
		}
		k := [3]int{s.Map, int(math.Floor(s.X / around)), int(math.Floor(s.Y / around))}
		cells[k] = append(cells[k], sp{s.X, s.Y, *s.Z})
	}
	f.Close()
	withZ := 0
	for i := range places {
		p := &places[i]
		bd := near
		var zs []float64
		ci, cj := int(math.Floor(p.X/around)), int(math.Floor(p.Y/around))
		for di := -1; di <= 1; di++ {
			for dj := -1; dj <= 1; dj++ {
				for _, s := range cells[[3]int{p.Map, ci + di, cj + dj}] {
					dd := math.Hypot(s.x-p.X, s.y-p.Y)
					if dd <= bd {
						bd, p.Z = dd, s.z
					}
					if dd <= around {
						zs = append(zs, s.z)
					}
				}
			}
		}
		if math.IsNaN(p.Z) && len(zs) > 0 {
			sort.Float64s(zs)
			p.Z = zs[len(zs)/2]
		}
		if !math.IsNaN(p.Z) {
			withZ++
		}
	}
	// pairs: within the radius, both on the same side of every lift (a walk through a lift keeps the lift model)
	lift := func(p dataset.Point) int {
		in := 0
		for k, l := range d.Travel.Lifts {
			if l.At.Map == p.Map && math.Hypot(p.X-l.At.X, p.Y-l.At.Y) <= l.Radius {
				in |= 1 << k
			}
		}
		return in
	}
	o, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer o.Close()
	bw := bufio.NewWriter(o)
	defer bw.Flush()
	for i, p := range places {
		z := "-"
		if !math.IsNaN(p.Z) {
			z = strconv.FormatFloat(p.Z, 'f', 1, 64)
		}
		fmt.Fprintf(bw, "p %d %d %.1f %.1f %s\n", i, p.Map, p.X, p.Y, z)
	}
	byCell := map[[3]int][]int{}
	cell := func(p dataset.Point) [3]int {
		return [3]int{p.Map, int(math.Floor(p.X / *radius)), int(math.Floor(p.Y / *radius))}
	}
	for i, p := range places {
		byCell[cell(p.Point)] = append(byCell[cell(p.Point)], i)
	}
	pairs := 0
	for i, p := range places {
		c := cell(p.Point)
		li := lift(p.Point)
		var js []int
		for di := -1; di <= 1; di++ {
			for dj := -1; dj <= 1; dj++ {
				for _, j := range byCell[[3]int{c[0], c[1] + di, c[2] + dj}] {
					if j > i && math.Hypot(places[j].X-p.X, places[j].Y-p.Y) <= *radius && lift(places[j].Point) == li {
						js = append(js, j)
					}
				}
			}
		}
		sort.Ints(js)
		if len(js) > 0 {
			var sb strings.Builder
			for _, j := range js {
				sb.WriteString(" ")
				sb.WriteString(strconv.Itoa(j))
			}
			fmt.Fprintf(bw, "q %d%s\n", i, sb.String())
			pairs += len(js)
		}
	}
	fmt.Fprintf(os.Stderr, "%d places (%d with a height), %d pairs\n", len(places), withZ, pairs)
	return nil
}

// build-walktable turns the places (walk-places) and the yards measured by tools/walktable/walktable.lua into
// data/walktable.json.gz.
func cmdBuildWalktable(args []string) error {
	fs := newFlags("build-walktable")
	placesFile := fs.String("places", "", "walk-places output")
	in := fs.String("in", "", "folder with walktable.lua's output files (*.txt)")
	rev := fs.String("spf", "", "Shortest Path Forever commit the navigation maps came from")
	out := fs.String("out", "data/walktable.json.gz", "file to write")
	fs.Parse(args)
	if *placesFile == "" || *in == "" || *rev == "" {
		return fmt.Errorf("build-walktable needs -places, -in and -spf")
	}
	f, err := os.Open(*placesFile)
	if err != nil {
		return err
	}
	type pl struct {
		m    int
		x, y float64
	}
	var places []pl
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<24), 1<<24)
	for sc.Scan() {
		t := strings.Fields(sc.Text())
		if len(t) >= 5 && t[0] == "p" {
			m, _ := strconv.Atoi(t[2])
			x, _ := strconv.ParseFloat(t[3], 64)
			y, _ := strconv.ParseFloat(t[4], 64)
			places = append(places, pl{m, x, y})
		}
	}
	f.Close()
	files, err := filepath.Glob(filepath.Join(*in, "*.txt"))
	if err != nil {
		return err
	}
	type pair struct{ i, j int }
	yards := map[pair]float64{}
	for _, fn := range files {
		f, err := os.Open(fn)
		if err != nil {
			return err
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			t := strings.Fields(sc.Text())
			if len(t) != 3 || t[2] == "-" {
				continue
			}
			i, _ := strconv.Atoi(t[0])
			j, _ := strconv.Atoi(t[1])
			v, err := strconv.ParseFloat(t[2], 64)
			if err == nil && i >= 0 && j >= 0 && i < len(places) && j < len(places) {
				yards[pair{min(i, j), max(i, j)}] = v
			}
		}
		f.Close()
	}
	// by map: the places, then each pair as the delta of its first place, the delta of its second (from the first
	// place, or from the previous second of the same first) and the yards
	byMap := map[int]*dataset.WalkTableMap{}
	local := make([]int, len(places))
	for i, p := range places {
		m := byMap[p.m]
		if m == nil {
			m = &dataset.WalkTableMap{Map: p.m}
			byMap[p.m] = m
		}
		local[i] = len(m.X)
		m.X = append(m.X, float32(p.x))
		m.Y = append(m.Y, float32(p.y))
	}
	keys := make([]pair, 0, len(yards))
	for k := range yards {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(a, b int) bool {
		if keys[a].i != keys[b].i {
			return keys[a].i < keys[b].i
		}
		return keys[a].j < keys[b].j
	})
	last := map[int][2]int{} // map -> previous pair (local)
	for _, k := range keys {
		pi, pj := places[k.i], places[k.j]
		if pi.m != pj.m {
			continue
		}
		m := byMap[pi.m]
		li, lj := local[k.i], local[k.j]
		prev, ok := last[pi.m]
		if !ok {
			prev = [2]int{0, 0}
		}
		di := li - prev[0]
		dj := lj - li
		if di == 0 && ok {
			dj = lj - prev[1]
		}
		m.Pairs = append(m.Pairs, int32(di), int32(dj), int32(math.Round(yards[k])))
		last[pi.m] = [2]int{li, lj}
	}
	t := dataset.WalkTable{Schema: dataset.WalkTableSchema, SPF: *rev}
	for _, m := range byMap {
		t.Maps = append(t.Maps, *m)
	}
	sort.Slice(t.Maps, func(a, b int) bool { return t.Maps[a].Map < t.Maps[b].Map })
	o, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer o.Close()
	z, _ := gzip.NewWriterLevel(o, gzip.BestCompression)
	if err := json.NewEncoder(z).Encode(t); err != nil {
		return err
	}
	if err := z.Close(); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "%d places, %d pairs\n", len(places), len(keys))
	return nil
}
