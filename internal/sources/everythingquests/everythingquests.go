// Package everythingquests reads the generated Forever data of the EverythingQuests addon (quest starts, turn-ins,
// targets, prerequisites, level gates) at a pinned commit. Only the literal data tables are read; no Lua is run and
// no guide text is taken. Missing quantities stay unknown.
package everythingquests

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/internal/lua"
)

// Repository is the upstream project.
const Repository = "wheelbarrel00/EverythingQuests"

var files = map[string]string{
	"available": "Data/QuestAvailable_Forever.lua",
	"targets":   "Data/QuestNPCs_Forever.lua",
	"turnins":   "Data/QuestTurnIn_Forever.lua",
}

var sha = regexp.MustCompile(`^[0-9a-f]{40}$`)

// Fetch downloads the three data files at a commit.
func Fetch(ctx context.Context, client *http.Client, revision string) (map[string]string, error) {
	if !sha.MatchString(revision) {
		return nil, fmt.Errorf("everythingquests: a full commit SHA is required, got %q", revision)
	}
	raw := map[string]string{}
	for kind, path := range files {
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
		b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20+1))
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK || len(b) <= 100 || len(b) > 8<<20 {
			return nil, fmt.Errorf("everythingquests: %s: HTTP %d, %d bytes", url, resp.StatusCode, len(b))
		}
		raw[kind] = string(b)
	}
	return raw, nil
}

// Point is a position (zone map percent) with the entity standing there.
type Point struct {
	UIMap  int     `json:"uiMapId"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Entity *int    `json:"entity_id"`
	Kind   string  `json:"entity_kind"` // npc, object, item_source, dungeon_npc_entrance, dungeon_object_entrance
}

// Target is an objective target.
type Target struct {
	Objective int    `json:"objective_index"`
	Kind      string `json:"kind"` // npc, object, loot_source_npc
	ID        int    `json:"id"`
}

// Quest as the addon describes it.
type Quest struct {
	ID       int      `json:"id"`
	Name     *string  `json:"name"`
	Required *int     `json:"required_level"`
	Level    *int     `json:"quest_level"`
	Starts   []Point  `json:"starts"`
	Turnins  []Point  `json:"turnins"`
	Targets  []Target `json:"target_refs"`
	PreAny   []int    `json:"prereq_any"`
	PreAll   []int    `json:"prereq_all"`
}

// Data is the addon's Forever data at one commit.
type Data struct {
	Revision string
	Quests   map[int]*Quest
}

var rowRx = regexp.MustCompile(`^\s*\[(\d+)\]\s*=\s*(\{.*\}|".*"|'.*'|-?\d+)\s*,?\s*(?:--.*)?$`)

// section reads one flat table of the generated file (one quest per line).
func section(raw, heading string) (map[int]any, error) {
	var start *regexp.Regexp
	switch heading {
	case "targets":
		start = regexp.MustCompile(`(?m)^ns\.CLASSIC_QUEST_NPCS\s*=\s*\{\s*$`)
	case "turnins":
		start = regexp.MustCompile(`(?m)^ns\.CLASSIC_QUEST_TURNIN\s*=\s*\{\s*$`)
	default:
		start = regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(heading) + `\s*=\s*\{\s*$`)
	}
	loc := start.FindStringIndex(raw)
	if loc == nil {
		return nil, fmt.Errorf("everythingquests: section %s missing", heading)
	}
	out := map[int]any{}
	for _, line := range strings.Split(raw[loc[1]:], "\n") {
		line = strings.TrimRight(line, "\r")
		if t := strings.TrimSpace(line); t == "}," || t == "}" {
			break
		}
		m := rowRx.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		id, _ := strconv.Atoi(m[1])
		v, err := lua.Parse(m[2])
		if err != nil {
			return nil, fmt.Errorf("everythingquests %s %d: %w", heading, id, err)
		}
		if _, dup := out[id]; dup {
			return nil, fmt.Errorf("everythingquests: duplicate %s quest %d", heading, id)
		}
		out[id] = v
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("everythingquests: section %s is empty", heading)
	}
	return out, nil
}

var pointKinds = map[int]string{1: "npc", 2: "object", 3: "item_source", 5: "dungeon_npc_entrance", 6: "dungeon_object_entrance"}

// points decodes {[uiMap] = {packed, ...}}: packed = entity*1e9 + kind*1e8 + x*1e6 + y*1e2 (x, y in 1/100 percent).
func points(v any) []Point {
	t, ok := v.(lua.Table)
	if !ok {
		return []Point{}
	}
	out := []Point{}
	for _, k := range lua.SortedKeys(t) {
		ui, ok := k.(int)
		vals, ok2 := t[k].([]any)
		if !ok || !ok2 {
			continue
		}
		for _, raw := range vals {
			p, ok := raw.(int)
			if !ok || p < 0 {
				continue
			}
			entity := p / 1_000_000_000
			kind := (p / 100_000_000) % 10
			base := p % 100_000_000
			x, y := float64(base/10_000)/100, float64(base%10_000)/100
			name, known := pointKinds[kind]
			if !known || x > 100 || y > 100 {
				continue
			}
			pt := Point{UIMap: ui, X: x, Y: y, Kind: name}
			if entity > 0 {
				e := entity
				pt.Entity = &e
			}
			out = append(out, pt)
		}
	}
	return out
}

var targetKinds = map[int]string{1: "npc", 2: "object", 3: "loot_source_npc"}

// targets decodes {packed, ...}: packed = objective*1e8 + kind*1e7 + id.
func targets(v any) []Target {
	l, _ := v.([]any)
	out := []Target{}
	for _, raw := range l {
		p, ok := raw.(int)
		if !ok || p < 0 {
			continue
		}
		t := Target{Objective: p / 100_000_000, Kind: targetKinds[(p/10_000_000)%10], ID: p % 10_000_000}
		if t.ID == 0 || t.Kind == "" {
			continue
		}
		dup := false
		for _, o := range out {
			dup = dup || o == t
		}
		if !dup {
			out = append(out, t)
		}
	}
	return out
}

func intList(v any) []int {
	l, _ := v.([]any)
	out := []int{}
	for _, e := range l {
		if n, ok := e.(int); ok {
			out = append(out, n)
		}
	}
	return out
}

// Build decodes the raw files.
func Build(raw map[string]string, revision string) (*Data, error) {
	get := func(file, heading string) (map[int]any, error) { return section(raw[file], heading) }
	starts, err := get("available", "start")
	if err != nil {
		return nil, err
	}
	names, err := get("available", "names")
	if err != nil {
		return nil, err
	}
	gates, err := get("available", "gates")
	if err != nil {
		return nil, err
	}
	pre, err := get("available", "pre")
	if err != nil {
		return nil, err
	}
	preAll, err := get("available", "preAll")
	if err != nil {
		return nil, err
	}
	objs, err := get("targets", "targets")
	if err != nil {
		return nil, err
	}
	turnins, err := get("turnins", "turnins")
	if err != nil {
		return nil, err
	}
	ids := map[int]bool{}
	for _, m := range []map[int]any{starts, names, gates, pre, preAll, objs, turnins} {
		for id := range m {
			ids[id] = true
		}
	}
	d := &Data{Revision: revision, Quests: map[int]*Quest{}}
	keys := make([]int, 0, len(ids))
	for id := range ids {
		keys = append(keys, id)
	}
	sort.Ints(keys)
	for _, id := range keys {
		q := &Quest{ID: id, Starts: points(starts[id]), Turnins: points(turnins[id]), Targets: targets(objs[id]),
			PreAny: intList(pre[id]), PreAll: intList(preAll[id])}
		if s, ok := names[id].(string); ok {
			q.Name = &s
		}
		if g, ok := gates[id].(int); ok {
			req, lvl := g/100_000_000_000, (g/1_000_000_000)%100
			q.Required, q.Level = &req, &lvl
		}
		d.Quests[id] = q
	}
	if len(d.Quests) < 1000 {
		return nil, fmt.Errorf("everythingquests: only %d quests, refusing a partial source", len(d.Quests))
	}
	return d, nil
}
