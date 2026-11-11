package questie

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/internal/lua"
)

// QuestieDB's Forever layers on top of the Era seed (docs/forever-data.md), in the order its generator applies them:
// the Wowhead-Forever delta-base (generated), the players' traces (QuestieTrace), then the authored Forever
// corrections. Each is a Lua file whose Load returns `id -> field -> value`; only those literal tables are read, no
// Lua runs. A field key `name_add` / `name_remove` adds or removes elements (tablePatch.lua), `{}` deletes a field.
// First come the inherited corrections (src/corrections/Forever/legacy): Questie's fixes of the Era seed, still active
// for Forever ("the converted Classic baseline").

// fieldNames are the row layouts (src/meta/*Meta.lua), 1-based in Lua, 0-based in a row here.
var fieldNames = map[string][]string{
	"quests": {"name", "startedBy", "finishedBy", "requiredLevel", "questLevel", "requiredRaces", "requiredClasses",
		"objectivesText", "triggerEnd", "objectives", "sourceItemId", "preQuestGroup", "preQuestSingle", "childQuests",
		"inGroupWith", "exclusiveTo", "zoneOrSort", "requiredSkill", "requiredMinRep", "requiredMaxRep",
		"requiredSourceItems", "nextQuestInChain", "questFlags", "specialFlags", "parentQuest", "reputationReward",
		"breadcrumbForQuestId", "breadcrumbs", "extraObjectives", "requiredSpell", "requiredSpecialization",
		"requiredMaxLevel", "availableUntilCompleted", "availableStartingWith", "requiredRanks", "disabledByQuest"},
	"npc_positions": {"name", "minLevelHealth", "maxLevelHealth", "minLevel", "maxLevel", "rank", "spawns", "waypoints",
		"zoneID", "questStarts", "questEnds", "factionID", "friendlyToFaction", "subName", "npcFlags"},
	"objects": {"name", "questStarts", "questEnds", "spawns", "zoneID", "factionID", "waypoints"},
	"items": {"name", "npcDrops", "objectDrops", "itemDrops", "startQuest", "questRewards", "flags", "foodType",
		"itemLevel", "requiredLevel", "ammoType", "class", "subClass", "vendors", "relatedQuests", "teachesSpell"},
}

// grouped fields merge add/remove per group: quest givers per kind, objectives per kind, spawns per zone
var grouped = map[string]bool{"startedBy": true, "finishedBy": true, "objectives": true, "spawns": true, "waypoints": true}

// Enum files the correction constants come from.
var enumFiles = map[string]string{
	"enum_zones":       "src/corrections/enum/zones.lua",
	"enum_quests":      "src/corrections/enum/quests.lua",
	"enum_professions": "src/corrections/enum/professions.lua",
	"enum_factions":    "src/corrections/enum/factions.lua",
	"enum_expansions":  "src/corrections/enum/expansions.lua",
}

// The layers, by row kind, in application order.
var layers = []struct{ kind, file string }{
	{"quests", "legacy_quests"}, {"npc_positions", "legacy_npcs"}, {"objects", "legacy_objects"}, {"items", "legacy_items"},
	{"items", "legacy_item_starts"}, {"quests", "legacy_quest_reputation"},
	{"quests", "base_quests"}, {"npc_positions", "base_npcs"}, {"objects", "base_objects"}, {"items", "base_items"},
	{"quests", "trace_quests"}, {"npc_positions", "trace_npcs"}, {"objects", "trace_objects"}, {"items", "trace_items"},
	{"quests", "fix_quests"}, {"npc_positions", "fix_npcs"}, {"objects", "fix_objects"}, {"items", "fix_items"},
}

var layerFiles = map[string]string{
	"legacy_quests":           "src/corrections/Forever/legacy/classicQuestFixes.lua",
	"legacy_npcs":             "src/corrections/Forever/legacy/classicNPCFixes.lua",
	"legacy_objects":          "src/corrections/Forever/legacy/classicObjectFixes.lua",
	"legacy_items":            "src/corrections/Forever/legacy/classicItemFixes.lua",
	"legacy_item_starts":      "src/corrections/Forever/legacy/itemStartFixes.lua",
	"legacy_quest_reputation": "src/corrections/Forever/legacy/classicQuestReputationFixes.lua",
	"base_quests":             "src/corrections/Forever/generated/foreverBaseQuest.lua",
	"base_npcs":               "src/corrections/Forever/generated/foreverBaseNpc.lua",
	"base_objects":            "src/corrections/Forever/generated/foreverBaseObject.lua",
	"base_items":              "src/corrections/Forever/generated/foreverBaseItem.lua",
	"fix_quests":              "src/corrections/Forever/foreverQuestFixes.lua",
	"fix_npcs":                "src/corrections/Forever/foreverNPCFixes.lua",
	"fix_objects":             "src/corrections/Forever/foreverObjectFixes.lua",
	"fix_items":               "src/corrections/Forever/foreverItemFixes.lua",
}

// raceKeys are Forever's (enum/expansions.lua byExpansion.Forever).
var raceKeys = map[string]float64{"NONE": 0, "HUMAN": 1, "ORC": 2, "DWARF": 4, "NIGHT_ELF": 8, "UNDEAD": 16, "TAUREN": 32,
	"GNOME": 64, "TROLL": 128, "GOBLIN": 256, "SKYBORNE_ALLIANCE": 4294967296, "SKYBORNE_HORDE": 8589934592,
	"ALL_ALLIANCE": 1 + 4 + 8 + 64 + 4294967296, "ALL_HORDE": 2 + 16 + 32 + 128 + 8589934592}

var enumBlock = regexp.MustCompile(`(?m)^\s*(?:constants\.)?(\w+)\s*=\s*\{`)
var enumEntry = regexp.MustCompile(`^\s*([A-Z_][A-Z0-9_]*)\s*=\s*(-?\d+)\s*,?`)

// enums reads `NAME = { KEY = number, ... }` blocks, nested ones too; the first block of a name wins (classKeys:
// Classic's, the same in Forever).
func enums(raw map[string]string) map[string]map[string]float64 {
	out := map[string]map[string]float64{"raceKeys": raceKeys}
	for _, k := range []string{"enum_zones", "enum_quests", "enum_professions", "enum_factions", "enum_expansions"} {
		var stack []map[string]float64 // nil for a block whose name was taken already, or a non-named table
		for _, l := range strings.Split(raw[k], "\n") {
			l = stripComment(l)
			opened := strings.Count(l, "{")
			if m := enumBlock.FindStringSubmatch(l); m != nil {
				cur := map[string]float64{} // a later block of the same name adds only keys the first lacks
				if prev, seen := out[m[1]]; seen {
					cur = map[string]float64{}
					defer mergeMissing(prev, cur)
				} else {
					out[m[1]] = cur
				}
				stack = append(stack, cur)
				opened--
			} else if m := enumEntry.FindStringSubmatch(l); m != nil && len(stack) > 0 && stack[len(stack)-1] != nil {
				v, _ := strconv.ParseFloat(m[2], 64)
				stack[len(stack)-1][m[1]] = v
			}
			for i := 0; i < opened; i++ {
				stack = append(stack, nil)
			}
			for i := strings.Count(l, "}"); i > 0 && len(stack) > 0; i-- {
				stack = stack[:len(stack)-1]
			}
		}
	}
	return out
}

func mergeMissing(into, from map[string]float64) {
	for k, v := range from {
		if _, ok := into[k]; !ok {
			into[k] = v
		}
	}
}

var iconRef = regexp.MustCompile(`\bQuestie\.ICON_TYPE_[A-Z_]+\b`)
var l10nCall = regexp.MustCompile(`\bl10n\(("(?:[^"\\]|\\.)*")\)`)

var loadFunc = regexp.MustCompile(`:Load(?:AutomaticQuestStarts)?\(\)`)

var aliasLine = regexp.MustCompile(`(?m)^\s*local\s+(\w+)\s*=\s*\w+\.(\w+)\s*$`)
var constRef = regexp.MustCompile(`\b([a-zA-Z]\w*)\.([A-Z_][A-Z0-9_]*)\b`)
var keyRef = regexp.MustCompile(`\[\s*\w+Keys\.(\w+)\s*\]`)
var sum = regexp.MustCompile(`(-?\d+(?:\.\d+)?)\s*([+-])\s*(\d+(?:\.\d+)?)`)
var entryStart = regexp.MustCompile(`^\s*\[(\d+)\]\s*=\s*\{`)

// Correction is one parsed layer: id -> field name -> value, with the entries that could not be read as literals.
type Correction struct {
	Rows    map[int]map[string]any
	Skipped []int
}

// parseCorrection reads the table Load returns, entry by entry; an entry with code (a function call) is skipped.
func parseCorrection(raw string, en map[string]map[string]float64) (Correction, error) {
	out := Correction{Rows: map[int]map[string]any{}}
	if strings.TrimSpace(raw) == "" || strings.HasPrefix(raw, "-- not in this revision") {
		return out, nil
	}
	load := loadFunc.FindStringIndex(raw) // Load, or LoadAutomaticQuestStarts; LoadFactionFixes and LoadDynamic are not read
	if load == nil {
		return out, fmt.Errorf("no Load function")
	}
	body := raw[load[0]:]
	alias := map[string]map[string]float64{}
	head := body
	if r := strings.Index(body, "return {"); r >= 0 {
		head = body[:r]
	}
	for _, m := range aliasLine.FindAllStringSubmatch(head, -1) {
		name := m[2]
		switch name {
		case "profKeys":
			name = "professionKeys"
		}
		if t := en[name]; t != nil {
			alias[m[1]] = t
		}
	}
	r := strings.Index(body, "return {")
	if r < 0 {
		return out, fmt.Errorf("no return table")
	}
	tbl, err := matchBraces(body[r+len("return "):])
	if err != nil {
		return out, err
	}
	for _, e := range topEntries(tbl[1 : len(tbl)-1]) {
		m := entryStart.FindStringSubmatch(e)
		if m == nil {
			continue
		}
		id, _ := strconv.Atoi(m[1])
		text := keyRef.ReplaceAllString(e[strings.Index(e, "=")+1:], `["$1"]`)
		text = iconRef.ReplaceAllString(text, "0")   // map icons: no meaning for routes
		text = l10nCall.ReplaceAllString(text, "$1") // a translated text: its English source
		bad := false
		text = constRef.ReplaceAllStringFunc(text, func(s string) string {
			p := constRef.FindStringSubmatch(s)
			if t := alias[p[1]]; t != nil {
				if v, ok := t[p[2]]; ok {
					return strconv.FormatFloat(v, 'f', -1, 64)
				}
			}
			bad = true
			return s
		})
		for i := 0; i < 20 && sum.MatchString(text); i++ {
			text = sum.ReplaceAllStringFunc(text, func(s string) string {
				p := sum.FindStringSubmatch(s)
				a, _ := strconv.ParseFloat(p[1], 64)
				b, _ := strconv.ParseFloat(p[3], 64)
				if p[2] == "-" {
					b = -b
				}
				return strconv.FormatFloat(a+b, 'f', -1, 64)
			})
		}
		text = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text), ","))
		v, err := lua.Parse(text)
		if bad || err != nil {
			out.Skipped = append(out.Skipped, id)
			continue
		}
		fields := map[string]any{}
		if t, ok := v.(lua.Table); ok {
			for k, x := range t {
				if s, ok := k.(string); ok {
					fields[s] = x
				}
			}
		}
		out.Rows[id] = fields
	}
	sort.Ints(out.Skipped)
	return out, nil
}

// matchBraces returns s's leading {...}, outside strings and comments.
func matchBraces(s string) (string, error) {
	depth, quote := 0, byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0 && c == '\\':
			i++
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '-' && i+1 < len(s) && s[i+1] == '-':
			for i < len(s) && s[i] != '\n' {
				i++
			}
		case c == '{':
			depth++
		case c == '}':
			depth--
			if depth == 0 {
				return s[:i+1], nil
			}
		}
	}
	return "", fmt.Errorf("unbalanced table")
}

// topEntries splits a table body at its top-level commas, comments removed.
func topEntries(s string) []string {
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		lines = append(lines, stripComment(l))
	}
	s = strings.Join(lines, "\n")
	var out []string
	depth, quote, start := 0, byte(0), 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0 && c == '\\':
			i++
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '{':
			depth++
		case c == '}':
			depth--
		case c == ',' && depth == 0:
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if strings.TrimSpace(s[start:]) != "" {
		out = append(out, s[start:])
	}
	return out
}

// LayerStats counts what each layer changed.
type LayerStats struct {
	Layer            string
	Entries, Created int
	Skipped          []int
}

// applyLayers folds the Forever layers into the rows, in order.
func applyLayers(raw map[string]string, rows map[string]map[int][]any) ([]LayerStats, error) {
	en := enums(raw)
	var stats []LayerStats
	for _, l := range layers {
		c, err := parseCorrection(raw[l.file], en)
		if err != nil {
			return nil, fmt.Errorf("questie %s: %w", l.file, err)
		}
		st := LayerStats{Layer: l.file, Skipped: c.Skipped}
		names := fieldNames[l.kind]
		index := map[string]int{}
		for i, n := range names {
			index[n] = i
		}
		ids := make([]int, 0, len(c.Rows))
		for id := range c.Rows {
			ids = append(ids, id)
		}
		sort.Ints(ids)
		for _, id := range ids {
			row, ok := rows[l.kind][id]
			if !ok {
				row = []any{}
				st.Created++
			}
			fields := c.Rows[id]
			keys := make([]string, 0, len(fields))
			for k := range fields {
				keys = append(keys, k)
			}
			sort.Strings(keys) // replacements before _add/_remove of other fields; one field never has both
			for _, k := range keys {
				name, op := k, ""
				if b, ok := strings.CutSuffix(k, "_add"); ok {
					name, op = b, "add"
				} else if b, ok := strings.CutSuffix(k, "_remove"); ok {
					name, op = b, "remove"
				}
				i, ok := index[name]
				if !ok {
					continue
				}
				for len(row) <= i {
					row = append(row, nil)
				}
				switch op {
				case "":
					row[i] = emptyNil(fields[k])
				default:
					row[i] = patch(row[i], fields[k], op, grouped[name])
				}
			}
			rows[l.kind][id] = row
			st.Entries++
		}
		stats = append(stats, st)
	}
	return stats, nil
}

// emptyNil: `{}` deletes a field.
func emptyNil(v any) any {
	switch t := v.(type) {
	case []any:
		if len(t) == 0 {
			return nil
		}
	case lua.Table:
		if len(t) == 0 {
			return nil
		}
	}
	return v
}

// patch adds or removes elements (set semantics, deep equality); grouped values per group (position or zone key).
func patch(cur, operand any, op string, isGrouped bool) any {
	if !isGrouped {
		return emptyNil(listPatch(asList(cur), asList(operand), op))
	}
	groups := map[any]any{}
	order := []any{}
	put := func(k, v any) {
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = v
	}
	positional := true
	switch t := cur.(type) {
	case []any:
		for i, v := range t {
			put(i+1, v)
		}
	case lua.Table:
		positional = false
		for _, k := range lua.SortedKeys(t) {
			put(k, t[k])
		}
	}
	switch t := operand.(type) {
	case []any:
		for i, v := range t {
			if v == nil {
				continue
			}
			put(i+1, listPatch(asList(groups[i+1]), asList(v), op))
		}
	case lua.Table:
		if _, isZone := firstKey(t).(int); isZone && len(t) > 0 {
			for _, k := range lua.SortedKeys(t) {
				put(k, listPatch(asList(groups[k]), asList(t[k]), op))
			}
			if _, ok := cur.([]any); !ok {
				positional = false
			}
			if cur == nil {
				positional = false
			}
		}
	}
	if positional {
		n := 0
		for _, k := range order {
			if i, ok := k.(int); ok && i > n {
				n = i
			}
		}
		out := make([]any, n)
		for i := 1; i <= n; i++ {
			if v := groups[i]; v != nil && len(asList(v)) > 0 {
				out[i-1] = v
			}
		}
		return emptyNil(out)
	}
	out := lua.Table{}
	for _, k := range order {
		if v := groups[k]; len(asList(v)) > 0 {
			out[k] = v
		}
	}
	return emptyNil(out)
}

func firstKey(t lua.Table) any {
	for _, k := range lua.SortedKeys(t) {
		return k
	}
	return nil
}

func asList(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case nil:
		return nil
	case lua.Table:
		var out []any
		for _, k := range lua.SortedKeys(t) {
			out = append(out, t[k])
		}
		return out
	}
	return []any{v}
}

func listPatch(cur, operand []any, op string) []any {
	has := func(l []any, v any) bool {
		for _, x := range l {
			if same(x, v) {
				return true
			}
		}
		return false
	}
	var out []any
	if op == "remove" {
		for _, x := range cur {
			if !has(operand, x) {
				out = append(out, x)
			}
		}
		return out
	}
	out = append(out, cur...)
	for _, v := range operand {
		if v != nil && !has(out, v) {
			out = append(out, v)
		}
	}
	return out
}

// same compares parsed values, ints and floats alike.
func same(a, b any) bool {
	return reflect.DeepEqual(norm(a), norm(b))
}

func norm(v any) any {
	switch t := v.(type) {
	case int:
		return float64(t)
	case []any:
		out := make([]any, len(t))
		for i, x := range t {
			out[i] = norm(x)
		}
		return out
	case lua.Table:
		out := map[string]any{}
		for k, x := range t {
			out[fmt.Sprint(k)] = norm(x)
		}
		return out
	}
	return v
}
