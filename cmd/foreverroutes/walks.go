package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// build-walks turns the output of tools/walkgrid/walkgrid.lua (one or more text files per map) into data/walks.json.gz.
func cmdBuildWalks(args []string) error {
	fs := newFlags("build-walks")
	in := fs.String("in", "", "folder with walkgrid.lua's output files (*.txt)")
	rev := fs.String("spf", "", "Shortest Path Forever commit the navigation maps came from")
	out := fs.String("out", "data/walks.json.gz", "file to write")
	fs.Parse(args)
	if *in == "" || *rev == "" {
		return fmt.Errorf("build-walks needs -in and -spf")
	}
	files, err := filepath.Glob(filepath.Join(*in, "*.txt"))
	if err != nil {
		return err
	}
	type cell struct {
		i, j  int
		x, y  float64
		costs [16]int
	}
	type grid struct {
		head  dataset.WalkMap
		cells map[[2]int]*cell
	}
	move := map[[2]int]int{}
	for k, m := range dataset.WalkMoves {
		move[m] = k
	}
	grids := map[int]*grid{}
	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			return err
		}
		var lines [][]string
		var g *grid
		sc := bufio.NewScanner(fh)
		for sc.Scan() {
			p := strings.Fields(sc.Text())
			if len(p) == 0 {
				continue
			}
			if p[0] == "g" && len(p) == 7 {
				m, _ := strconv.Atoi(p[1])
				if grids[m] == nil {
					x0, _ := strconv.ParseFloat(p[2], 64)
					y0, _ := strconv.ParseFloat(p[3], 64)
					size, _ := strconv.ParseFloat(p[4], 64)
					ni, _ := strconv.Atoi(p[5])
					nj, _ := strconv.Atoi(p[6])
					grids[m] = &grid{head: dataset.WalkMap{Map: m, X0: x0, Y0: y0, Cell: size, NI: ni, NJ: nj}, cells: map[[2]int]*cell{}}
				}
				g = grids[m]
				continue
			}
			lines = append(lines, p)
		}
		fh.Close()
		if err := sc.Err(); err != nil {
			return err
		}
		if g == nil {
			return fmt.Errorf("%s: no grid line (unfinished run?)", f)
		}
		for _, p := range lines {
			i, _ := strconv.Atoi(p[1])
			j, _ := strconv.Atoi(p[2])
			switch {
			case p[0] == "c" && len(p) == 5:
				x, _ := strconv.ParseFloat(p[3], 64)
				y, _ := strconv.ParseFloat(p[4], 64)
				if g.cells[[2]int{i, j}] == nil {
					g.cells[[2]int{i, j}] = &cell{i: i, j: j, x: x, y: y}
				}
			case p[0] == "e" && len(p) == 6:
				di, _ := strconv.Atoi(p[3])
				dj, _ := strconv.Atoi(p[4])
				cost, _ := strconv.ParseFloat(p[5], 64)
				k, ok := move[[2]int{di, dj}]
				c := g.cells[[2]int{i, j}]
				if ok && c != nil && cost > 0 {
					c.costs[k] = int(cost + 0.5)
				}
			}
		}
	}
	w := dataset.Walks{Schema: dataset.WalksSchema, Source: dataset.Source{Name: "Shortest Path Forever navigation maps",
		URL: "https://github.com/cjber/shortest-path-forever", Revision: *rev, License: "GPL-3.0-or-later",
		Note: "by Cillian Berragan: walking costs between neighbouring 200-yard cells, measured with its own path finder on its navigation maps (tools/walkgrid)"}}
	maps := make([]int, 0, len(grids))
	for m := range grids {
		maps = append(maps, m)
	}
	sort.Ints(maps)
	for _, m := range maps {
		g := grids[m]
		keys := make([][2]int, 0, len(g.cells))
		for k := range g.cells {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(a, b int) bool {
			if keys[a][0] != keys[b][0] {
				return keys[a][0] < keys[b][0]
			}
			return keys[a][1] < keys[b][1]
		})
		wm := g.head
		for _, k := range keys {
			c := g.cells[k]
			wm.Cells = append(wm.Cells, float64(c.i), float64(c.j), c.x, c.y)
			wm.Costs = append(wm.Costs, c.costs[:]...)
		}
		w.Maps = append(w.Maps, wm)
		fmt.Fprintf(os.Stderr, "map %d: %d cells\n", m, len(keys))
	}
	b, err := json.Marshal(w)
	if err != nil {
		return err
	}
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	gz, _ := gzip.NewWriterLevel(f, gzip.BestCompression)
	if _, err := gz.Write(b); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	return f.Close()
}
