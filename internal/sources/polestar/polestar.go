// Package polestar reads forever-db, the open WoW Forever quest database of the Polestar guide addon
// (https://github.com/fitzforge/polestar, folder forever-db), at a pinned commit. Polestar's Forge builds it from the
// Forever game client and independent open sources: quests with their Forever XP, levels, race and class masks (with
// the Skyborne bits), givers, turn-ins, objectives with counts and prerequisites; NPCs, objects, items, what drops an
// item, spawn points in world yards and the frames of the in-game maps. The tables used here are GPL-3.0
// (forever-db/LICENSE); the dataset credits Polestar with the commit.
package polestar

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// Repository is the upstream project.
const Repository = "fitzforge/polestar"

// Credit is the attribution the dataset carries.
const Credit = "Polestar forever-db (github.com/fitzforge/polestar, GPL-3.0): Forever client quest data"

// Tables read from forever-db/data.
var Tables = []string{"quest", "quest_start", "quest_end", "quest_objective", "quest_prereq", "npc", "object", "item", "item_source", "spawn", "ui_map"}

// Skyborne is one race on both sides; Forever marks it with its own bit per faction, past the 32 bits of the classic
// races.
const (
	SkyborneAlliance = 1 << 32
	SkyborneHorde    = 1 << 33
)

// Quest is a row of quest.
type Quest struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Level    int    `json:"level"`
	MinLevel int    `json:"min_level"`
	Races    int    `json:"races"`   // client mask, 0 = every race
	Classes  int    `json:"classes"` // client mask, 0 = every class
	Side     int    `json:"side"`    // 0 both, 1 Alliance, 2 Horde
	XP       int    `json:"xp"`
	Type     int    `json:"type"` // 81 dungeon, 1 group/elite, 41 PvP, 62 raid
	Zone     int    `json:"zone_area"`
	Repeat   bool   `json:"repeatable"`
}

// Ref is a giver or turn-in (quest_start, quest_end): npc, object or item.
type Ref struct {
	Quest int    `json:"quest"`
	Kind  string `json:"kind"`
	ID    int    `json:"id"`
}

// Objective is a row of quest_objective: kill, item, object, event or talk; target and count.
type Objective struct {
	Quest  int    `json:"quest"`
	N      int    `json:"n"`
	Kind   string `json:"kind"`
	Target int    `json:"target"`
	Count  int    `json:"count"`
}

// Prereq is a row of quest_prereq.
type Prereq struct {
	Quest      int    `json:"quest"`
	Pre        int    `json:"pre"`
	Mode       string `json:"mode"` // all | any
	Breadcrumb bool   `json:"breadcrumb"`
}

// NPC is a row of npc.
type NPC struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Subname  string `json:"subname"`
	LevelMin int    `json:"level_min"`
	LevelMax int    `json:"level_max"`
	Rank     *int   `json:"rank"`
	Faction  int    `json:"faction_template"`
}

// Object is a row of object.
type Object struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type int    `json:"type"`
}

// Item is a row of item.
type Item struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	QuestItem bool   `json:"quest_item"`
}

// ItemSource is a row of item_source: what drops or holds an item; the chance only where a loot table gives one.
type ItemSource struct {
	Item   int      `json:"item"`
	Kind   string   `json:"kind"` // npc | object | item (a container)
	ID     int      `json:"id"`
	Chance *float64 `json:"chance"`
}

// Spawn is a row of spawn: a point in world yards on the client map (x north, y west).
type Spawn struct {
	Kind string  `json:"kind"`
	ID   int     `json:"id"`
	Map  int     `json:"map"`
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
}

// UIMap is the frame of an in-game map in world yards.
type UIMap struct {
	UIMap  int     `json:"ui_map"`
	Map    int     `json:"map"`
	Left   float64 `json:"left"`
	Right  float64 `json:"right"`
	Top    float64 `json:"top"`
	Bottom float64 `json:"bottom"`
}

// DB is forever-db, indexed.
type DB struct {
	Revision    string
	Build       string // client build of the release
	Quests      map[int]*Quest
	Starts      map[int][]Ref
	Ends        map[int][]Ref
	Objectives  map[int][]Objective
	Prereqs     map[int][]Prereq
	NPCs        map[int]*NPC
	Objects     map[int]*Object
	Items       map[int]*Item
	ItemSources map[int][]ItemSource
	Spawns      map[string]map[int][]Spawn // kind -> id -> points
	UIMaps      []UIMap
}

func readTable[T any](dir, table string, each func(T)) error {
	f, err := os.Open(filepath.Join(dir, table+".jsonl"))
	if err != nil {
		return err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for line := 1; sc.Scan(); line++ {
		var v T
		if err := json.Unmarshal(sc.Bytes(), &v); err != nil {
			return fmt.Errorf("polestar %s line %d: %w", table, line, err)
		}
		each(v)
	}
	return sc.Err()
}

// Load reads the tables from a folder holding <table>.jsonl (forever-db/data) and index.json.
func Load(dir, revision string) (*DB, error) {
	db := &DB{Revision: revision, Quests: map[int]*Quest{}, Starts: map[int][]Ref{}, Ends: map[int][]Ref{}, Objectives: map[int][]Objective{},
		Prereqs: map[int][]Prereq{}, NPCs: map[int]*NPC{}, Objects: map[int]*Object{}, Items: map[int]*Item{}, ItemSources: map[int][]ItemSource{},
		Spawns: map[string]map[int][]Spawn{"npc": {}, "object": {}}}
	var index struct {
		ClientBuild string `json:"clientBuild"`
	}
	if b, err := os.ReadFile(filepath.Join(dir, "index.json")); err == nil {
		_ = json.Unmarshal(b, &index)
	}
	db.Build = index.ClientBuild
	steps := []error{
		readTable(dir, "quest", func(q Quest) { db.Quests[q.ID] = &q }),
		readTable(dir, "quest_start", func(r Ref) { db.Starts[r.Quest] = append(db.Starts[r.Quest], r) }),
		readTable(dir, "quest_end", func(r Ref) { db.Ends[r.Quest] = append(db.Ends[r.Quest], r) }),
		readTable(dir, "quest_objective", func(o Objective) { db.Objectives[o.Quest] = append(db.Objectives[o.Quest], o) }),
		readTable(dir, "quest_prereq", func(p Prereq) { db.Prereqs[p.Quest] = append(db.Prereqs[p.Quest], p) }),
		readTable(dir, "npc", func(n NPC) { db.NPCs[n.ID] = &n }),
		readTable(dir, "object", func(o Object) { db.Objects[o.ID] = &o }),
		readTable(dir, "item", func(i Item) { db.Items[i.ID] = &i }),
		readTable(dir, "item_source", func(s ItemSource) { db.ItemSources[s.Item] = append(db.ItemSources[s.Item], s) }),
		readTable(dir, "spawn", func(s Spawn) {
			if m := db.Spawns[s.Kind]; m != nil {
				m[s.ID] = append(m[s.ID], s)
			}
		}),
		readTable(dir, "ui_map", func(u UIMap) { db.UIMaps = append(db.UIMaps, u) }),
	}
	for _, err := range steps {
		if err != nil {
			return nil, err
		}
	}
	for _, l := range db.Objectives {
		sort.Slice(l, func(a, b int) bool { return l[a].N < l[b].N })
	}
	if len(db.Quests) == 0 {
		return nil, fmt.Errorf("polestar: no quests in %s", dir)
	}
	return db, nil
}

var sha = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Fetch downloads the tables at a commit into cache/polestar-<commit> (once) and returns that folder.
func Fetch(ctx context.Context, client *http.Client, cache, revision string) (string, error) {
	if !sha.MatchString(revision) {
		return "", fmt.Errorf("polestar: a full commit SHA is required, got %q", revision)
	}
	dir := filepath.Join(cache, "polestar-"+revision)
	if _, err := os.Stat(filepath.Join(dir, "complete")); err == nil {
		return dir, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	for _, name := range append(append([]string{}, Tables...), "index") {
		file := name + ".jsonl"
		path := "forever-db/data/" + file
		if name == "index" {
			file, path = "index.json", "forever-db/index.json"
		}
		url := "https://raw.githubusercontent.com/" + Repository + "/" + revision + "/" + path
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		b, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
		resp.Body.Close()
		if err != nil {
			return "", err
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("polestar: HTTP %d for %s", resp.StatusCode, path)
		}
		if err := os.WriteFile(filepath.Join(dir, file), b, 0o644); err != nil {
			return "", err
		}
	}
	return dir, os.WriteFile(filepath.Join(dir, "complete"), nil, 0o644)
}
