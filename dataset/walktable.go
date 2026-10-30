package dataset

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// WalkTableSchema identifies the walk table file.
const WalkTableSchema = "foreverroutes.walktable.v1"

// WalkTableMap is one map's table: the places (X[i], Y[i]) and the measured walks between them. Pairs has three
// numbers per walk, in order of the first place, then the second: the first place as a step from the previous walk's
// first place, the second place as a step from the previous walk's second place when the first is the same, else
// from the first place, and the running yards.
type WalkTableMap struct {
	Map   int       `json:"map"`
	X     []float32 `json:"x"`
	Y     []float32 `json:"y"`
	Pairs []int32   `json:"pairs"`
}

// WalkTable is the walk table file: walks between the places routes go to, measured with Shortest Path Forever's
// path finder on its navigation maps, as the yards a player runs (the path drawn straight, slopes included,
// swimming at its own speed).
type WalkTable struct {
	Schema string         `json:"schema"`
	SPF    string         `json:"spf"` // the Shortest Path Forever commit of the navigation maps
	Maps   []WalkTableMap `json:"maps"`
}

// LoadWalkTable reads a walk table file (plain or gzip JSON).
func LoadWalkTable(path string) (*WalkTable, error) {
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
	var t WalkTable
	if err := json.NewDecoder(r).Decode(&t); err != nil {
		return nil, fmt.Errorf("walk table: %w", err)
	}
	if t.Schema != WalkTableSchema {
		return nil, fmt.Errorf("walk table: schema %q, want %q", t.Schema, WalkTableSchema)
	}
	for _, m := range t.Maps {
		if len(m.X) != len(m.Y) || len(m.Pairs)%3 != 0 {
			return nil, fmt.Errorf("walk table: map %d: %d x, %d y, %d pair numbers", m.Map, len(m.X), len(m.Y), len(m.Pairs))
		}
	}
	return &t, nil
}
