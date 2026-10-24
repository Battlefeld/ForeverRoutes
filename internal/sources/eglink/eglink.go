// Package eglink reads EG Link output by Efficient Games (https://github.com/JIVESCORP/eg-link-output-wowf, CC BY 4.0):
// what players' EG Link addon recorded in WoW Forever, one folder per client build. Two datasets are read, over every
// build: loot (per creature or object, what dropped and how often, with the chance per kill while on the quest that
// needs the item) and objectives (where a quest objective's count went up, in world yards).
package eglink

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Repository is the upstream project.
const Repository = "JIVESCORP/eg-link-output-wowf"

// Credit is the attribution the licence asks for.
const Credit = "Data from Efficient Games (https://efficient.games), licensed under CC BY 4.0"

// Product is the client folder read.
const Product = "wow_classic_beta"

// MinWindows: a chance from fewer loot windows than this is too rough to use.
const MinWindows = 5

// Drop is an item a source gave: a creature ("npc") or an object ("object"), the chance per kill or per opening (0-1),
// and the loot windows it rests on.
type Drop struct {
	Kind    string
	ID      int
	Chance  float64
	Windows int
}

// Point is a position in world yards on a map (0 Eastern Kingdoms, 1 Kalimdor, else the map id).
type Point struct {
	Map  int
	X, Y float64
}

// Data is EG Link's loot and objective places, combined over the builds.
type Data struct {
	Revision   string
	Builds     []string
	Drops      map[int][]Drop          // item -> sources, most windows first
	Objectives map[int]map[int][]Point // quest -> objective (1-based) -> where its count went up
}

type lootItem struct {
	Drops  int     `json:"drops"`
	Chance float64 `json:"chance"`
}

type lootSource struct {
	Windows int                 `json:"windows"`
	Items   map[string]lootItem `json:"items"`
	Quests  map[string]struct {
		Windows int                 `json:"windows"`
		Items   map[string]lootItem `json:"items"`
	} `json:"quests"`
}

type position struct {
	Instance int     `json:"instance"`
	WorldX   float64 `json:"worldX"`
	WorldY   float64 `json:"worldY"`
}

// Load reads every build folder below dir (data/wow_classic_beta).
func Load(dir, revision string) (*Data, error) {
	builds, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	d := &Data{Revision: revision, Drops: map[int][]Drop{}, Objectives: map[int]map[int][]Point{}}
	type acc struct{ sum, windows float64 }
	drops := map[int]map[string]*acc{} // item -> "kind:id" -> chance x windows, windows
	for _, b := range builds {
		if !b.IsDir() {
			continue
		}
		d.Builds = append(d.Builds, b.Name())
		var loot struct {
			Sources map[string]lootSource `json:"sources"`
		}
		if err := readJSON(filepath.Join(dir, b.Name(), "loot.json"), &loot); err != nil {
			return nil, err
		}
		for key, s := range loot.Sources {
			src, ok := sourceOf(key)
			if !ok {
				continue
			}
			add := func(item string, it lootItem, windows int) {
				id, err := strconv.Atoi(item)
				if err != nil || windows <= 0 {
					return
				}
				if drops[id] == nil {
					drops[id] = map[string]*acc{}
				}
				a := drops[id][src]
				if a == nil {
					a = &acc{}
					drops[id][src] = a
				}
				a.sum += it.Chance * float64(windows)
				a.windows += float64(windows)
			}
			// a quest item's chance is the one counted under the quest that needs it (it drops only then); the key's
			// own chance mixes in every other quest's kills
			perQuest := map[string]bool{}
			for _, q := range s.Quests {
				for item, it := range q.Items {
					add(item, it, q.Windows)
					perQuest[item] = true
				}
			}
			for item, it := range s.Items {
				if !perQuest[item] {
					add(item, it, s.Windows)
				}
			}
		}
		var obj struct {
			Quests map[string]map[string]struct {
				Positions []position `json:"positions"`
			} `json:"quests"`
		}
		if err := readJSON(filepath.Join(dir, b.Name(), "objectives.json"), &obj); err != nil {
			return nil, err
		}
		for qs, objs := range obj.Quests {
			q, err := strconv.Atoi(qs)
			if err != nil {
				continue
			}
			for ns, o := range objs {
				n, err := strconv.Atoi(ns)
				if err != nil {
					continue
				}
				for _, p := range o.Positions {
					if d.Objectives[q] == nil {
						d.Objectives[q] = map[int][]Point{}
					}
					d.Objectives[q][n] = append(d.Objectives[q][n], Point{Map: p.Instance, X: p.WorldX, Y: p.WorldY})
				}
			}
		}
	}
	sort.Strings(d.Builds)
	for item, srcs := range drops {
		for src, a := range srcs {
			if a.windows < MinWindows {
				continue
			}
			kind, id, _ := strings.Cut(src, ":")
			n, _ := strconv.Atoi(id)
			d.Drops[item] = append(d.Drops[item], Drop{Kind: kind, ID: n, Chance: a.sum / a.windows, Windows: int(a.windows)})
		}
		l := d.Drops[item]
		sort.Slice(l, func(a, b int) bool {
			if l[a].Windows != l[b].Windows {
				return l[a].Windows > l[b].Windows
			}
			if l[a].Kind != l[b].Kind {
				return l[a].Kind < l[b].Kind
			}
			return l[a].ID < l[b].ID
		})
	}
	for _, objs := range d.Objectives {
		for n, pts := range objs {
			sort.Slice(pts, func(a, b int) bool {
				if pts[a].Map != pts[b].Map {
					return pts[a].Map < pts[b].Map
				}
				if pts[a].X != pts[b].X {
					return pts[a].X < pts[b].X
				}
				return pts[a].Y < pts[b].Y
			})
			objs[n] = pts
		}
	}
	return d, nil
}

// sourceOf turns a loot key into "npc:<id>" or "object:<id>": a corpse or an object opened. Skinning, pick pocketing,
// fishing and containers are left out (no quest item a route plans comes from them).
func sourceOf(key string) (string, bool) {
	parts := strings.Split(key, "|")
	kind, id, ok := strings.Cut(parts[0], ":")
	if !ok || kind != "npc" && kind != "object" {
		return "", false
	}
	if _, err := strconv.Atoi(id); err != nil {
		return "", false
	}
	for _, p := range parts[1:] {
		if strings.HasPrefix(p, "spell:") && kind == "npc" {
			return "", false // skinning or pick pocket
		}
	}
	return kind + ":" + id, true
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil // a build without this dataset
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("eglink %s: %w", path, err)
	}
	return nil
}
