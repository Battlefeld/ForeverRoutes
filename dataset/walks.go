package dataset

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// WalksSchema identifies the walk grid file.
const WalksSchema = "foreverroutes.walks.v1"

// WalkMoves are the cells a cell's costs lead to, in the order of WalkMap.Costs: the 8 neighbours and the 8 knight's
// moves.
var WalkMoves = [16][2]int{{-2, -1}, {-2, 1}, {-1, -2}, {-1, -1}, {-1, 0}, {-1, 1}, {-1, 2}, {0, -1}, {0, 1}, {1, -2}, {1, -1}, {1, 0}, {1, 1}, {1, 2}, {2, -1}, {2, 1}}

// WalkMap is one map's grid: cells of Cell yards from (X0, Y0), cell (i, j) covering x in [X0+i*Cell, +Cell) and y in
// [Y0+j*Cell, +Cell). Cells lists the walkable cells as i, j, x, y (the point the costs start from), four numbers per
// cell; Costs has 16 numbers per cell, the running yards to the cell at each of WalkMoves (0: no way).
type WalkMap struct {
	Map   int       `json:"map"`
	X0    float64   `json:"x0"`
	Y0    float64   `json:"y0"`
	Cell  float64   `json:"cell"`
	NI    int       `json:"ni"`
	NJ    int       `json:"nj"`
	Cells []float64 `json:"cells"`
	Costs []int     `json:"costs"`
}

// Walks is the walk grid file: walking costs measured on navigation maps, chained by the planner.
type Walks struct {
	Schema string    `json:"schema"`
	Source Source    `json:"source"`
	Maps   []WalkMap `json:"maps"`
}

// LoadWalks reads a walk grid file (plain or gzip JSON).
func LoadWalks(path string) (*Walks, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var r io.Reader = f
	head := make([]byte, 2)
	if _, err := io.ReadFull(f, head); err != nil {
		return nil, err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	if head[0] == 0x1f && head[1] == 0x8b {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		r = gz
	}
	var w Walks
	if err := json.NewDecoder(r).Decode(&w); err != nil {
		return nil, fmt.Errorf("walks: %w", err)
	}
	if w.Schema != WalksSchema {
		return nil, fmt.Errorf("walks: schema %q, want %q", w.Schema, WalksSchema)
	}
	for _, m := range w.Maps {
		if len(m.Cells)%4 != 0 || len(m.Costs) != len(m.Cells)/4*16 {
			return nil, fmt.Errorf("walks: map %d: %d cell numbers, %d costs", m.Map, len(m.Cells), len(m.Costs))
		}
	}
	return &w, nil
}
