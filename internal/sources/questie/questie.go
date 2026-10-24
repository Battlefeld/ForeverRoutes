// Package questie reads QuestieDB's Forever data files (quests, NPCs, objects, items, Classic quest XP and drop
// rates) at a pinned commit and turns them into the catalog the merge step uses. The files are read as data only;
// no Lua is run. Values that are Classic references stay labelled as such, and objective counts, which QuestieDB does
// not hold, stay unknown.
package questie

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/internal/lua"
)

// Repository is the upstream project.
const Repository = "Questie/QuestieDB"

// Files are the data files read, by kind.
var Files = map[string]string{
	"quests":          "data/Forever/foreverQuestDB.lua",
	"npc_positions":   "data/Forever/foreverNpcDB.lua",
	"npc_health":      "data/Classic/classicNpcDB.lua",
	"items":           "data/Forever/foreverItemDB.lua",
	"objects":         "data/Forever/foreverObjectDB.lua",
	"quest_xp":        "support/Forever/QuestXP/xpDB-classic.lua",
	"item_drop_rates": "support/Forever/DropTables/classicItemDrops.lua",
}

var sha = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Fetch downloads the data files at a commit, keeping a copy per commit in cache.
func Fetch(ctx context.Context, client *http.Client, revision, cache string) (map[string]string, error) {
	if !sha.MatchString(revision) {
		return nil, fmt.Errorf("questie: a full commit SHA is required, got %q", revision)
	}
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return nil, err
	}
	raw := map[string]string{}
	for kind, path := range Files {
		file := filepath.Join(cache, revision+"_"+kind+".lua")
		if b, err := os.ReadFile(file); err == nil {
			raw[kind] = string(b)
			continue
		}
		url := "https://raw.githubusercontent.com/" + Repository + "/" + revision + "/" + path
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "ForeverRoutes (https://github.com/Battlefeld/ForeverRoutes)")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(resp.Body, 12<<20))
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK || len(b) < 1000 {
			return nil, fmt.Errorf("questie: %s: HTTP %d, %d bytes", url, resp.StatusCode, len(b))
		}
		if err := os.WriteFile(file, b, 0o644); err != nil {
			return nil, err
		}
		raw[kind] = string(b)
	}
	return raw, nil
}

// Refs lists who starts or ends a quest.
type Refs struct {
	NPC    []int `json:"npc"`
	Object []int `json:"object"`
	Item   []int `json:"item,omitempty"`
}

// Objective is a target QuestieDB names for a quest; the count is never known from QuestieDB.
type Objective struct {
	Kind          string `json:"kind"` // npc, object, item, reputation, kill_credit, spell
	IDs           []int  `json:"ids"`
	RequiredCount *int   `json:"required_count"`
	CountSource   string `json:"count_source"`
}

// Quest of the catalog. Nil pointers are values QuestieDB does not give.
type Quest struct {
	ID          int         `json:"id"`
	Name        any         `json:"name"`
	Required    any         `json:"required_level"`
	Level       any         `json:"quest_level"`
	Races       any         `json:"required_races_mask"`
	Classes     any         `json:"required_classes_mask"`
	StartedBy   Refs        `json:"started_by"`
	FinishedBy  Refs        `json:"finished_by"`
	PreAll      []int       `json:"prequest_all"`
	PreAny      []int       `json:"prequest_any"`
	Objectives  []Objective `json:"objectives"`
	ZoneOrSort  any         `json:"zone_or_sort"`
	XP          *int        `json:"xp_value,omitempty"`
	XPLevelHint int         `json:"-"`
}

// NPC of the catalog (the merged database and the build read these keys).
type NPC struct {
	ID        int                     `json:"id"`
	Name      any                     `json:"name"`
	MinLevel  int                     `json:"min_level"`
	MaxLevel  int                     `json:"max_level"`
	Rank      any                     `json:"rank"`
	HPMin     *int                    `json:"hp_min"`
	HPMax     *int                    `json:"hp_max"`
	HPSource  string                  `json:"hp_source"`
	Spawns    map[string][][2]float64 `json:"spawns"`
	SpawnSrc  string                  `json:"spawn_source"`
	Friendly  any                     `json:"friendly_to_faction"`
	NPCFlags  any                     `json:"npc_flags"`
	FactionID any                     `json:"faction_id"`
	Grind     bool                    `json:"potential_grind_candidate"`
	KillXP    string                  `json:"kill_xp_model"`
	Critter   string                  `json:"is_critter_or_no_xp"`
}

// Object of the catalog.
type Object struct {
	ID       int                     `json:"id"`
	Name     any                     `json:"name"`
	Spawns   map[string][][2]float64 `json:"spawns"`
	SpawnSrc string                  `json:"spawn_source"`
}

// Drop is a Classic drop rate sample (Wowhead via QuestieDB), not a confirmed Forever rate.
type Drop struct {
	Chance float64 `json:"chance"`
	Source string  `json:"source"`
}

// Item of the catalog: where it drops.
type Item struct {
	ID         int             `json:"id"`
	Name       any             `json:"name"`
	NPCs       []int           `json:"npc_sources"`
	Objects    []int           `json:"object_sources"`
	ChanceByID map[string]Drop `json:"drop_chance_by_npc"`
	DropSource string          `json:"drop_source"`
}

// Catalog is everything read from one QuestieDB commit.
type Catalog struct {
	Revision string
	Quests   map[int]*Quest
	NPCs     map[int]*NPC
	Objects  map[int]*Object
	Items    map[int]*Item
}

// Options limit the catalog to the levels routes need.
type Options struct {
	MaxPlayerLevel int // quests that can be taken up to this level (30)
	MaxMobLevel    int // creatures up to this level, plus every creature a quest needs (35)
}

func spawns(v any) map[string][][2]float64 {
	out := map[string][][2]float64{}
	t, ok := v.(lua.Table)
	if !ok {
		return out
	}
	for zone, pts := range t {
		z, ok := zone.(int)
		list, ok2 := pts.([]any)
		if !ok || !ok2 {
			continue
		}
		var clean [][2]float64
		for _, p := range list {
			xy, ok := p.([]any)
			if !ok || len(xy) < 2 {
				continue
			}
			x, okx := number(xy[0])
			y, oky := number(xy[1])
			if okx && oky && x >= 0 && x <= 100 && y >= 0 && y <= 100 {
				clean = append(clean, [2]float64{x, y})
			}
		}
		if len(clean) > 0 {
			out[strconv.Itoa(z)] = clean
		}
	}
	return out
}

func number(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case float64:
		return n, true
	}
	return 0, false
}

func list(v any) []any {
	l, _ := v.([]any)
	return l
}

var xpRow = regexp.MustCompile(`(?m)^\s*\[(\d+)\]\s*=\s*\{(\d+)\s*,\s*(\d+)\}`)

func questXP(raw string) map[int]int {
	out := map[int]int{}
	for _, m := range xpRow.FindAllStringSubmatch(raw, -1) {
		id, _ := strconv.Atoi(m[1])
		v, _ := strconv.Atoi(m[3])
		out[id] = v
	}
	return out
}

var (
	dropStart = regexp.MustCompile(`^\s{2,}\[(\d+)\]\s*=\s*\{\s*--`)
	dropEnd   = regexp.MustCompile(`^\s*\},?`)
	dropRow   = regexp.MustCompile(`^\s*\[(\d+)\]\s*=\s*(\d+(?:\.\d+)?),?\s*$`)
)

// dropRates reads the Classic item -> creature drop rate table (percent), as fractions.
func dropRates(raw string) map[int]map[int]float64 {
	out := map[int]map[int]float64{}
	active := -1
	for _, line := range strings.Split(raw, "\n") {
		if m := dropStart.FindStringSubmatch(line); m != nil {
			active, _ = strconv.Atoi(m[1])
			out[active] = map[int]float64{}
			continue
		}
		if active < 0 {
			continue
		}
		if dropEnd.MatchString(line) {
			active = -1
			continue
		}
		if m := dropRow.FindStringSubmatch(line); m != nil {
			v, _ := strconv.ParseFloat(m[2], 64)
			if v > 0 && v <= 100 {
				npc, _ := strconv.Atoi(m[1])
				out[active][npc] = roundTo(v/100, 7)
			}
		}
	}
	return out
}

func roundTo(v float64, digits int) float64 {
	p := 1.0
	for i := 0; i < digits; i++ {
		p *= 10
	}
	return float64(int64(v*p+0.5)) / p
}

var objectiveKinds = []string{"npc", "object", "item", "reputation", "kill_credit", "spell"}

// Build makes the catalog from the raw files.
func Build(raw map[string]string, revision string, opt Options) (*Catalog, error) {
	if opt.MaxPlayerLevel == 0 {
		opt.MaxPlayerLevel = 30
	}
	if opt.MaxMobLevel == 0 {
		opt.MaxMobLevel = 35
	}
	rows := map[string]map[int][]any{}
	for _, kind := range []string{"quests", "npc_positions", "npc_health", "objects", "items"} {
		r, err := lua.Rows(raw[kind])
		if err != nil {
			return nil, fmt.Errorf("questie %s: %w", kind, err)
		}
		rows[kind] = r
	}
	xps := questXP(raw["quest_xp"])
	rates := dropRates(raw["item_drop_rates"])
	c := &Catalog{Revision: revision, Quests: map[int]*Quest{}, NPCs: map[int]*NPC{}, Objects: map[int]*Object{}, Items: map[int]*Item{}}
	wantNPC, wantObj, wantItem := map[int]bool{}, map[int]bool{}, map[int]bool{}
	add := func(set map[int]bool, ids []int) {
		for _, id := range ids {
			set[id] = true
		}
	}
	for id, row := range rows["quests"] {
		req, ok1 := lua.At(row, 3).(int)
		_, ok2 := lua.At(row, 4).(int)
		if !ok1 || !ok2 || req < 1 || req > opt.MaxPlayerLevel {
			continue
		}
		starts, ends := list(lua.At(row, 1)), list(lua.At(row, 2))
		q := &Quest{ID: id, Name: lua.At(row, 0), Required: lua.At(row, 3), Level: lua.At(row, 4), Races: lua.At(row, 5),
			Classes: lua.At(row, 6), ZoneOrSort: lua.At(row, 15),
			StartedBy:  Refs{NPC: ids(lua.At(starts, 0)), Object: ids(lua.At(starts, 1)), Item: ids(lua.At(starts, 2))},
			FinishedBy: Refs{NPC: ids(lua.At(ends, 0)), Object: ids(lua.At(ends, 1))},
			PreAll:     ids(lua.At(row, 11)), PreAny: ids(lua.At(row, 12)), Objectives: []Objective{}}
		if q.StartedBy.Item == nil {
			q.StartedBy.Item = []int{}
		}
		add(wantNPC, q.StartedBy.NPC)
		add(wantNPC, q.FinishedBy.NPC)
		add(wantObj, q.StartedBy.Object)
		add(wantObj, q.FinishedBy.Object)
		add(wantItem, q.StartedBy.Item)
		for i, targets := range list(lua.At(row, 9)) {
			kind := fmt.Sprintf("other_%d", i)
			if i < len(objectiveKinds) {
				kind = objectiveKinds[i]
			}
			for _, entry := range list(targets) {
				var got []int
				if e, ok := entry.([]any); ok && (kind == "npc" || kind == "object" || kind == "item") {
					// {targetID, optional text, optional icon}: only the first element is the target
					if len(e) > 0 {
						got = ids(e[0])
					}
				} else {
					got = ids(entry)
				}
				if len(got) == 0 {
					continue
				}
				q.Objectives = append(q.Objectives, Objective{Kind: kind, IDs: got, CountSource: "unresolved"})
				switch kind {
				case "npc", "kill_credit":
					add(wantNPC, got)
				case "object":
					add(wantObj, got)
				case "item":
					add(wantItem, got)
				}
			}
		}
		if v, ok := xps[id]; ok {
			q.XP = &v
		}
		c.Quests[id] = q
	}
	for id, row := range rows["npc_positions"] {
		lo, ok1 := lua.At(row, 3).(int)
		hi, ok2 := lua.At(row, 4).(int)
		if !ok1 || !ok2 || (lo > opt.MaxMobLevel && !wantNPC[id]) {
			continue
		}
		n := &NPC{ID: id, Name: lua.At(row, 0), MinLevel: lo, MaxLevel: hi, Rank: lua.At(row, 5), HPSource: "unknown",
			Spawns: spawns(lua.At(row, 6)), SpawnSrc: "questiedb_forever_raw", Friendly: lua.At(row, 12),
			NPCFlags: lua.At(row, 14), FactionID: lua.At(row, 11), KillXP: "cmangos_classic_solo_level_formula", Critter: "unverified"}
		ref := rows["npc_health"][id]
		hlo, okl := lua.At(ref, 1).(int)
		hhi, okh := lua.At(ref, 2).(int)
		if okl && okh && hlo > 1 && hhi >= hlo {
			n.HPMin, n.HPMax, n.HPSource = &hlo, &hhi, "questiedb_classic_raw"
		}
		n.Grind = lo <= opt.MaxMobLevel && n.Rank == 0 && n.Friendly == nil && len(n.Spawns) > 0 && n.HPMin != nil
		c.NPCs[id] = n
	}
	for id := range wantItem {
		row, ok := rows["items"][id]
		if !ok {
			continue
		}
		it := &Item{ID: id, Name: lua.At(row, 0), NPCs: ids(lua.At(row, 1)), Objects: ids(lua.At(row, 2)),
			ChanceByID: map[string]Drop{}, DropSource: "questiedb_classic_wowhead"}
		npcs := make([]int, 0, len(rates[id]))
		for npc := range rates[id] {
			npcs = append(npcs, npc)
		}
		sort.Ints(npcs)
		for _, npc := range npcs {
			it.ChanceByID[strconv.Itoa(npc)] = Drop{Chance: rates[id][npc], Source: "classic_wowhead_sample"}
		}
		add(wantNPC, it.NPCs)
		add(wantObj, it.Objects)
		c.Items[id] = it
	}
	for id := range wantObj {
		row, ok := rows["objects"][id]
		if !ok {
			continue
		}
		c.Objects[id] = &Object{ID: id, Name: lua.At(row, 0), Spawns: spawns(lua.At(row, 3)), SpawnSrc: "questiedb_forever_raw"}
	}
	return c, nil
}

func ids(v any) []int {
	out := lua.IDs(v)
	if out == nil {
		return []int{}
	}
	return out
}
