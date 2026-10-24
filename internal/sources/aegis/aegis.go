// Package aegis reads the quest data players collect in game with the AEGIS: Pathfinder (Forever) addon
// (https://github.com/Torchlite-bit/Aegis_Pathfinder_Forever, data/merged.json, MIT): per quest where it was taken and
// handed in, the client's own map points for its objectives, where the player stood when an objective's count went
// up, and the XP the turn-in gave; per level the XP it takes. Only observations, never a character's name.
package aegis

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Repository is the upstream project.
const Repository = "Torchlite-bit/Aegis_Pathfinder_Forever"

// Place is where a quest was taken or handed in: the in-game map and the mean position on it (0-1).
type Place struct {
	Kind  string  `json:"kind"`
	ID    int     `json:"id"`
	UIMap int     `json:"mapID"`
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Count int     `json:"count"`
}

// Range is a number seen more than once.
type Range struct {
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Count int     `json:"count"`
}

// Cell is a point on an in-game map in percent (the collector's 1 % grid), with how often it was seen.
type Cell struct {
	UIMap int
	X, Y  float64
	Count int
}

// Quest is what players saw of one quest.
type Quest struct {
	Title    string                        `json:"title"`
	Pickups  map[string]Place              `json:"pickups"`
	Turnins  map[string]Place              `json:"turnins"`
	Progress map[string]map[string]int     `json:"progress"` // objective index -> cells
	POI      map[string]map[string]int     `json:"poi"`      // objective | turnin | available -> cells
	Rewards  struct{ XP map[string]Range } `json:"rewards"`
}

// Data is merged.json, indexed.
type Data struct {
	Revision string
	Quests   map[int]*Quest
	LevelXP  map[int]Range // XP to the next level, by level
}

// Load reads merged.json.
func Load(path, revision string) (*Data, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var raw struct {
		Quests map[string]*Quest `json:"quests"`
		Levels map[string]struct {
			XPMax Range `json:"xpMax"`
		} `json:"levels"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("aegis: %w", err)
	}
	d := &Data{Revision: revision, Quests: map[int]*Quest{}, LevelXP: map[int]Range{}}
	for k, q := range raw.Quests {
		if id, err := strconv.Atoi(k); err == nil && id > 0 && q != nil {
			d.Quests[id] = q
		}
	}
	for k, l := range raw.Levels {
		if lv, err := strconv.Atoi(k); err == nil && l.XPMax.Count > 0 {
			d.LevelXP[lv] = l.XPMax
		}
	}
	return d, nil
}

// Cells parses a cell list ("<uiMapID>:<x%>:<y%>" -> count) into points at the centre of each cell, in a fixed order.
func Cells(m map[string]int) []Cell {
	var out []Cell
	for k, n := range m {
		p := strings.Split(k, ":")
		if len(p) != 3 {
			continue
		}
		ui, e1 := strconv.Atoi(p[0])
		x, e2 := strconv.ParseFloat(p[1], 64)
		y, e3 := strconv.ParseFloat(p[2], 64)
		if e1 != nil || e2 != nil || e3 != nil {
			continue
		}
		out = append(out, Cell{UIMap: ui, X: x + 0.5, Y: y + 0.5, Count: n})
	}
	sort.Slice(out, func(a, b int) bool {
		if out[a].UIMap != out[b].UIMap {
			return out[a].UIMap < out[b].UIMap
		}
		if out[a].X != out[b].X {
			return out[a].X < out[b].X
		}
		return out[a].Y < out[b].Y
	})
	return out
}

// ObjectiveCells are where a quest's objectives are: where counts went up, else the client's objective points.
func (q *Quest) ObjectiveCells() []Cell {
	var out []Cell
	keys := make([]string, 0, len(q.Progress))
	for k := range q.Progress {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, Cells(q.Progress[k])...)
	}
	if len(out) == 0 {
		out = Cells(q.POI["objective"])
	}
	return out
}

// Best is the place seen most often (ties: the lower key), ok false when there is none with a position.
func Best(m map[string]Place) (Place, bool) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var best Place
	found := false
	for _, k := range keys {
		p := m[k]
		if p.UIMap == 0 || p.X <= 0 && p.Y <= 0 {
			continue
		}
		if !found || p.Count > best.Count {
			best, found = p, true
		}
	}
	return best, found
}

// MaxXP is the most XP the turn-in gave a player at or below the quest's level (before XP falls off), 0 if unseen.
func (q *Quest) MaxXP(questLevel int) int {
	best := 0.0
	for k, r := range q.Rewards.XP {
		if lv, err := strconv.Atoi(k); err == nil && lv <= questLevel && r.Max > best {
			best = r.Max
		}
	}
	return int(best)
}
