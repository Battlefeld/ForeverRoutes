// Package build makes the route dataset from its sources: the merged quest database (ForeverDB, QuestieDB Forever,
// EverythingQuests, WarcraftDB; built in earlier ForeverRoutes work) and the cmangos classic world database, which fills
// what the merged database does not know: objective counts, drop chances, spawn positions in world yards, respawn
// times, exclusive groups and the XP of quests that have none.
package build

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/internal/sqldump"
	"github.com/Battlefeld/ForeverRoutes/internal/warcraftdb"
	"github.com/Battlefeld/ForeverRoutes/xp"
)

// Inputs names the source files and their pinned revisions.
type Inputs struct {
	Merged          string // merged quest database JSON (schema foreverroutes.unified_quests.v1)
	MergedRevision  string
	CMangos         string // cmangos classic-db full dump (.sql or .sql.gz)
	CMangosRevision string
	WarcraftDB      string // optional overlay of Forever client quest facts (fetch-warcraftdb)
	MaxQuestLevel   int    // quests above this level are left out (0 = 35)
}

type obj = map[string]any

func get(m obj, path ...string) any {
	var cur any = m
	for _, p := range path {
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = mm[p]
	}
	return cur
}
func num(v any) (float64, bool) {
	f, ok := v.(float64)
	return f, ok
}
func ints(v any) []int {
	arr, _ := v.([]any)
	var out []int
	for _, x := range arr {
		if f, ok := x.(float64); ok {
			out = append(out, int(f))
		}
	}
	return out
}
func field(q obj, name string) (any, string) {
	f, _ := get(q, "fields", name).(map[string]any)
	if f == nil {
		return nil, ""
	}
	s, _ := f["source"].(string)
	return f["value"], s
}

type world struct {
	qt, ct, cr, cl, gob, gt, gl, cqr, cir, gqr, gir *sqldump.Table
	creature                                        map[int][]any
	spawns                                          map[int][]dataset.Point
	respawn                                         map[int]int
	quest                                           map[int][]any
	lootCreatures                                   map[int][]int // loot id -> creatures
	objSpawns                                       map[int][]dataset.Point
	objLoot                                         map[int][]int // loot id -> objects
	dropsByItem                                     map[int][]dataset.Drop
}

func loadCMangos(path string) (*world, error) {
	t, err := sqldump.ReadFile(path, "quest_template", "creature_template", "creature", "creature_loot_template",
		"gameobject", "gameobject_template", "gameobject_loot_template", "creature_questrelation",
		"creature_involvedrelation", "gameobject_questrelation", "gameobject_involvedrelation", "creature_spawn_entry",
		"gameobject_spawn_entry")
	if err != nil {
		return nil, err
	}
	w := &world{qt: t["quest_template"], ct: t["creature_template"], cr: t["creature"], cl: t["creature_loot_template"],
		gob: t["gameobject"], gt: t["gameobject_template"], gl: t["gameobject_loot_template"], cqr: t["creature_questrelation"],
		cir: t["creature_involvedrelation"], gqr: t["gameobject_questrelation"], gir: t["gameobject_involvedrelation"],
		creature: map[int][]any{}, spawns: map[int][]dataset.Point{}, respawn: map[int]int{}, quest: map[int][]any{},
		lootCreatures: map[int][]int{}, objSpawns: map[int][]dataset.Point{}, objLoot: map[int][]int{}, dropsByItem: map[int][]dataset.Drop{}}
	if len(w.qt.Rows) == 0 || len(w.ct.Rows) == 0 {
		return nil, fmt.Errorf("cmangos dump without quest_template/creature_template")
	}
	for _, r := range w.ct.Rows {
		e := w.ct.Int(r, "Entry")
		w.creature[e] = r
		if l := w.ct.Int(r, "LootId"); l > 0 {
			w.lootCreatures[l] = append(w.lootCreatures[l], e)
		}
	}
	// spawns with entry 0 take one of several entries at random (creature_spawn_entry / gameobject_spawn_entry); each
	// possible entry is counted at that place
	entries := func(tab *sqldump.Table) map[int][]int {
		out := map[int][]int{}
		if tab != nil {
			for _, r := range tab.Rows {
				out[tab.Int(r, "guid")] = append(out[tab.Int(r, "guid")], tab.Int(r, "entry"))
			}
		}
		return out
	}
	idsOf := func(tab *sqldump.Table, r []any, alt map[int][]int) []int {
		if id := tab.Int(r, "id"); id != 0 {
			return []int{id}
		}
		return alt[tab.Int(r, "guid")]
	}
	crAlt, gobAlt := entries(t["creature_spawn_entry"]), entries(t["gameobject_spawn_entry"])
	respawns := map[int][]int{}
	for _, r := range w.cr.Rows {
		m := w.cr.Int(r, "map")
		if m != 0 && m != 1 { // instances and other maps are not part of open-world routes
			continue
		}
		for _, id := range idsOf(w.cr, r, crAlt) {
			w.spawns[id] = append(w.spawns[id], dataset.Point{Map: m, X: round1(w.cr.Float(r, "position_x")), Y: round1(w.cr.Float(r, "position_y"))})
			respawns[id] = append(respawns[id], w.cr.Int(r, "spawntimesecsmin"))
		}
	}
	for id, list := range respawns {
		sort.Ints(list)
		w.respawn[id] = list[len(list)/2]
	}
	for _, r := range w.qt.Rows {
		w.quest[w.qt.Int(r, "entry")] = r
	}
	objType := map[int]int{}
	objLootID := map[int]int{}
	for _, r := range w.gt.Rows {
		e := w.gt.Int(r, "entry")
		objType[e] = w.gt.Int(r, "type")
		objLootID[e] = w.gt.Int(r, "data1") // chests (type 3): loot id
	}
	for _, r := range w.gt.Rows { // in table order, so the build is repeatable
		if e := w.gt.Int(r, "entry"); objType[e] == 3 && objLootID[e] > 0 {
			w.objLoot[objLootID[e]] = append(w.objLoot[objLootID[e]], e)
		}
	}
	for _, r := range w.gob.Rows {
		m := w.gob.Int(r, "map")
		if m != 0 && m != 1 {
			continue
		}
		for _, id := range idsOf(w.gob, r, gobAlt) {
			w.objSpawns[id] = append(w.objSpawns[id], dataset.Point{Map: m, X: round1(w.gob.Float(r, "position_x")), Y: round1(w.gob.Float(r, "position_y"))})
		}
	}
	for _, r := range w.cl.Rows {
		item, chance := w.cl.Int(r, "item"), math.Abs(w.cl.Float(r, "ChanceOrQuestChance"))
		if chance <= 0 || w.cl.Int(r, "mincountOrRef") < 0 {
			continue
		}
		for _, c := range w.lootCreatures[w.cl.Int(r, "entry")] {
			w.dropsByItem[item] = append(w.dropsByItem[item], dataset.Drop{NPC: c, Chance: chance})
		}
	}
	for _, r := range w.gl.Rows {
		item, chance := w.gl.Int(r, "item"), math.Abs(w.gl.Float(r, "ChanceOrQuestChance"))
		if chance <= 0 || w.gl.Int(r, "mincountOrRef") < 0 {
			continue
		}
		for _, o := range w.objLoot[w.gl.Int(r, "entry")] {
			w.dropsByItem[item] = append(w.dropsByItem[item], dataset.Drop{Object: o, Chance: chance})
		}
	}
	return w, nil
}

// readMaybeGzip reads a file that may be gzip compressed.
func readMaybeGzip(path string) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) < 2 || raw[0] != 0x1f || raw[1] != 0x8b {
		return raw, err
	}
	zr, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

// Build makes the dataset.
func Build(in Inputs) (*dataset.Dataset, error) {
	if in.MaxQuestLevel == 0 {
		in.MaxQuestLevel = 35
	}
	raw, err := readMaybeGzip(in.Merged)
	if err != nil {
		return nil, err
	}
	var merged obj
	if err := json.Unmarshal(raw, &merged); err != nil {
		return nil, fmt.Errorf("merged database: %w", err)
	}
	if s, _ := merged["schema"].(string); s != "foreverroutes.unified_quests.v1" {
		return nil, fmt.Errorf("merged database: unknown schema %q", s)
	}
	w, err := loadCMangos(in.CMangos)
	if err != nil {
		return nil, err
	}
	b := &builder{in: in, merged: merged, w: w, stats: map[string]int{}, npcWanted: map[int]bool{}, objWanted: map[int]bool{}}
	if in.WarcraftDB != "" {
		if b.wdb, err = warcraftdb.Load(in.WarcraftDB); err != nil {
			return nil, fmt.Errorf("warcraftdb overlay: %w", err)
		}
	}
	b.fitTransforms()
	b.quests()
	b.npcs()
	b.objects()
	travel := b.travel()
	build, _ := merged["build"].(string)
	d := &dataset.Dataset{Schema: dataset.Schema, Build: build, LevelXP: b.levelXP(), Quests: b.out, NPCs: b.npcOut, Objects: b.objOut, Travel: travel, Stats: b.stats,
		Sources: b.mergedSources()}
	d.Sources = append(d.Sources, dataset.Source{Name: "cmangos classic-db", URL: "https://github.com/cmangos/classic-db", Revision: in.CMangosRevision, License: "GPL-3.0",
		Note: "Classic world data: objective counts, drop chances, spawn positions, respawn times, quest groups. Classic, not Forever: Forever data wins where both exist."})
	if b.wdb != nil {
		d.Sources = append(d.Sources, dataset.Source{Name: "WarcraftDB Forever", URL: "https://forever.warcraftdb.com", Revision: b.wdb.Fetched,
			Note: "Forever client quest data (objectives with amounts, XP, levels, positions) for quests the other sources lack"})
	}
	return d, nil
}

// mergedSources lists the sources of the merged quest database with their revisions.
func (b *builder) mergedSources() []dataset.Source {
	src, _ := b.merged["sources"].(map[string]any)
	rev := func(k string) string {
		m, _ := src[k].(map[string]any)
		s, _ := m["revision"].(string)
		return s
	}
	out := []dataset.Source{}
	if b.in.MergedRevision != "" {
		out = append(out, dataset.Source{Name: "ForeverRoutes merged quest database", URL: "https://github.com/Battlefeld/ForeverRoutes (data/sources/merged-quests-1-30.json.gz)",
			Revision: b.in.MergedRevision, Note: "frozen input"})
	}
	if r := rev("questiedb"); r != "" && b.in.MergedRevision == "" {
		out = append(out, dataset.Source{Name: "QuestieDB", URL: "https://github.com/Questie/QuestieDB", Revision: r,
			Note: "Forever quests, NPCs, objects and items; Classic quest XP and drop rates labelled as Classic references"})
	}
	if b.in.MergedRevision == "" {
		build, _ := b.merged["build"].(string)
		out = append(out, dataset.Source{Name: "ForeverDB", URL: "https://foreverdb.net", Revision: build,
			Note: "Data from ForeverDB.net (https://foreverdb.net): quest list, givers, turn-ins and positions seen in WoW Forever. ForeverDB is not affiliated with Blizzard Entertainment."})
	}
	if r := rev("everythingquests"); r != "" && b.in.MergedRevision == "" {
		out = append(out, dataset.Source{Name: "EverythingQuests", URL: "https://github.com/wheelbarrel00/EverythingQuests", Revision: r, License: "MIT",
			Note: "Forever quest starts, turn-ins, targets and prerequisites (partly from AllTheThings)"})
	}
	if b.in.MergedRevision == "" {
		out = append(out, dataset.Source{Name: "Individually sourced facts", URL: "https://github.com/Battlefeld/ForeverRoutes (data/sources/manual-facts.json)",
			Note: "Quest XP, levels and repeatable flags from WarcraftDB, OctoWow, ClassicDB and Wowhead pages, each with its link"})
	}
	return out
}

type builder struct {
	in        Inputs
	merged    obj
	w         *world
	stats     map[string]int
	zoneT     map[string]transform // questie zone id -> transform
	uiT       map[string]transform // uiMapId -> transform
	out       []dataset.Quest
	npcOut    []dataset.NPC
	objOut    []dataset.Object
	npcWanted map[int]bool
	objWanted map[int]bool
	wdb       *warcraftdb.Overlay
}

func (b *builder) levelXP() []int {
	t, _ := b.merged["level_xp_classic"].(map[string]any)
	if len(t) == 0 {
		return xp.ClassicLevelXP
	}
	out := make([]int, 0, len(t))
	for i := 1; ; i++ {
		v, ok := num(t[strconv.Itoa(i)])
		if !ok {
			break
		}
		out = append(out, int(v))
	}
	if len(out) < 30 {
		return xp.ClassicLevelXP
	}
	return out
}

func (b *builder) npcRecord(id int) obj {
	n, _ := get(b.merged, "npcs", strconv.Itoa(id)).(map[string]any)
	return n
}

func (b *builder) fitTransforms() {
	zonePairs := map[string][]pair{}
	for _, raw := range b.merged["npcs"].(map[string]any) {
		n := raw.(map[string]any)
		id, _ := num(n["id"])
		cm := b.w.spawns[int(id)]
		if len(cm) != 1 {
			continue
		}
		sp, _ := n["spawns"].(map[string]any)
		if len(sp) != 1 {
			continue
		}
		for zone, list := range sp {
			pts, _ := list.([]any)
			if len(pts) != 1 {
				continue
			}
			xy, _ := pts[0].([]any)
			if len(xy) != 2 {
				continue
			}
			px, _ := num(xy[0])
			py, _ := num(xy[1])
			zonePairs[zone] = append(zonePairs[zone], pair{px: px, py: py, m: cm[0].Map, wx: cm[0].X, wy: cm[0].Y})
		}
	}
	uiPairs := map[string][]pair{}
	for _, raw := range b.merged["quests"].(map[string]any) {
		q := raw.(map[string]any)
		st, _ := get(q, "locations", "starts", "value").(map[string]any)
		sb, _ := field(q, "started_by")
		npcs := ints(get(obj{"v": sb}, "v", "npc"))
		if st == nil || len(npcs) != 1 || len(b.w.spawns[npcs[0]]) != 1 {
			continue
		}
		ui, _ := num(st["uiMapId"])
		px, _ := num(st["x"])
		py, _ := num(st["y"])
		cm := b.w.spawns[npcs[0]][0]
		key := strconv.Itoa(int(ui))
		uiPairs[key] = append(uiPairs[key], pair{px: px, py: py, m: cm.Map, wx: cm.X, wy: cm.Y})
	}
	b.zoneT, b.uiT = map[string]transform{}, map[string]transform{}
	for k, ps := range zonePairs {
		if t, ok := fitTransform(ps); ok {
			b.zoneT[k] = t
		}
	}
	for k, ps := range uiPairs {
		if t, ok := fitTransform(ps); ok {
			b.uiT[k] = t
		}
	}
	b.stats["map_transforms_zone"] = len(b.zoneT)
	b.stats["map_transforms_uimap"] = len(b.uiT)
}

func (b *builder) location(q obj, which string) *dataset.Point {
	v, _ := get(q, "locations", which, "value").(map[string]any)
	if v == nil {
		return nil
	}
	ui, _ := num(v["uiMapId"])
	t, ok := b.uiT[strconv.Itoa(int(ui))]
	if !ok {
		return nil
	}
	px, _ := num(v["x"])
	py, _ := num(v["y"])
	p := t.apply(px, py)
	return &p
}

func refs(v any, withItem bool) []dataset.Ref {
	m, _ := v.(map[string]any)
	var out []dataset.Ref
	for _, id := range ints(m["npc"]) {
		out = append(out, dataset.Ref{Kind: "npc", ID: id})
	}
	for _, id := range ints(m["object"]) {
		out = append(out, dataset.Ref{Kind: "object", ID: id})
	}
	if withItem {
		for _, id := range ints(m["item"]) {
			out = append(out, dataset.Ref{Kind: "item", ID: id})
		}
	}
	return out
}

func (b *builder) cmangosRefs(t *sqldump.Table, ot *sqldump.Table, quest int) []dataset.Ref {
	var out []dataset.Ref
	for _, r := range t.Rows {
		if t.Int(r, "quest") == quest {
			out = append(out, dataset.Ref{Kind: "npc", ID: t.Int(r, "id")})
		}
	}
	for _, r := range ot.Rows {
		if ot.Int(r, "quest") == quest {
			out = append(out, dataset.Ref{Kind: "object", ID: ot.Int(r, "id")})
		}
	}
	return out
}

func (b *builder) quests() {
	qs := b.merged["quests"].(map[string]any)
	ids := make([]int, 0, len(qs))
	for k := range qs {
		if id, err := strconv.Atoi(k); err == nil {
			ids = append(ids, id)
		}
	}
	sort.Ints(ids)
	for _, id := range ids {
		q := qs[strconv.Itoa(id)].(map[string]any)
		if na, _ := q["not_actionable"].(bool); na {
			b.stats["quests_not_actionable"]++
			continue
		}
		out := dataset.Quest{ID: id, Sources: map[string]string{}}
		if v, s := field(q, "name"); v != nil {
			out.Name, _ = v.(string)
			out.Sources["name"] = s
		}
		lvl, s := field(q, "quest_level")
		if f, ok := num(lvl); ok {
			out.Level = int(f)
			out.Sources["level"] = s
		}
		if f, ok := num(get(obj{"v": first(field(q, "required_level"))}, "v")); ok {
			out.MinLevel = int(f)
		}
		if out.Level > b.in.MaxQuestLevel || out.MinLevel > b.in.MaxQuestLevel {
			b.stats["quests_above_level_range"]++
			continue
		}
		if f, ok := num(first(field(q, "required_races_mask"))); ok {
			out.Races = int(uint64(f) & 0xFFFF)
		}
		if f, ok := num(first(field(q, "required_classes_mask"))); ok {
			out.Classes = int(uint64(f) & 0xFFFF)
		}
		if r, _ := first(field(q, "repeatable")).(bool); r {
			out.Repeatable = true
		}
		cm := b.w.quest[id]
		// XP: the merged database, else the cmangos money (Classic: XP = money at max level / 0.6)
		if v, s := field(q, "xp"); v != nil {
			if f, ok := num(v); ok {
				out.XP = int(f)
				out.Sources["xp"] = s
			}
		} else if cm != nil && b.w.qt.Int(cm, "RewMoneyMaxLevel") > 0 {
			out.XP = xp.QuestFromMoney(b.w.qt.Int(cm, "RewMoneyMaxLevel"))
			out.Sources["xp"] = "cmangos_money_formula"
			b.stats["xp_filled_from_cmangos"]++
		} else {
			out.Sources["xp"] = "unknown"
			b.stats["xp_unknown"]++
		}
		if out.Level == 0 && cm != nil {
			out.Level = b.w.qt.Int(cm, "QuestLevel")
			out.Sources["level"] = "cmangos"
		}
		if out.MinLevel == 0 && cm != nil {
			out.MinLevel = b.w.qt.Int(cm, "MinLevel")
		}
		if out.MinLevel < 1 {
			out.MinLevel = 1
		}
		sb, sbs := field(q, "started_by")
		out.Start = refs(sb, true)
		eb, ebs := field(q, "finished_by")
		out.End = refs(eb, false)
		out.Sources["start"], out.Sources["end"] = sbs, ebs
		if len(out.Start) == 0 && cm != nil {
			out.Start = b.cmangosRefs(b.w.cqr, b.w.gqr, id)
			out.Sources["start"] = "cmangos"
		}
		if len(out.End) == 0 && cm != nil {
			out.End = b.cmangosRefs(b.w.cir, b.w.gir, id)
			out.Sources["end"] = "cmangos"
		}
		out.PreAll = ints(first(field(q, "prequest_all")))
		out.PreAny = ints(first(field(q, "prequest_any")))
		if cm != nil {
			out.Exclusive = b.w.qt.Int(cm, "ExclusiveGroup")
			out.Breadcrumb = b.w.qt.Int(cm, "BreadcrumbForQuestId")
		}
		out.StartAt = b.location(q, "starts")
		if out.StartAt != nil {
			out.Sources["startAt"] = "forever_location"
		}
		out.EndAt = b.location(q, "ends")
		if out.EndAt != nil {
			out.Sources["endAt"] = "forever_location"
		}
		b.objectives(&out, q, cm)
		b.applyWarcraftDB(&out)
		for _, r := range out.Start {
			if r.Kind == "npc" {
				b.npcWanted[r.ID] = true
			} else if r.Kind == "object" {
				b.objWanted[r.ID] = true
			}
		}
		for _, r := range out.End {
			if r.Kind == "npc" {
				b.npcWanted[r.ID] = true
			} else {
				b.objWanted[r.ID] = true
			}
		}
		b.out = append(b.out, out)
	}
}

func first(v any, _ string) any { return v }

// objectives come from cmangos (counts, drops); objectives the merged database lists that cmangos does not know are
// Forever changes with an unknown count: such a quest stays in the data but is marked unsupported.
func (b *builder) objectives(out *dataset.Quest, q obj, cm []any) {
	t := b.w.qt
	if cm != nil {
		src := t.Int(cm, "SrcItemId")
		for i := 1; i <= 4; i++ {
			id, n := t.Int(cm, fmt.Sprintf("ReqCreatureOrGOId%d", i)), t.Int(cm, fmt.Sprintf("ReqCreatureOrGOCount%d", i))
			if id == 0 || n <= 0 {
				continue
			}
			if id > 0 {
				out.Objectives = append(out.Objectives, dataset.Objective{Kind: "kill", ID: id, Count: n, Source: "cmangos"})
				b.npcWanted[id] = true
			} else {
				out.Objectives = append(out.Objectives, dataset.Objective{Kind: "object", ID: -id, Count: n, Source: "cmangos"})
				b.objWanted[-id] = true
			}
		}
		for i := 1; i <= 4; i++ {
			id, n := t.Int(cm, fmt.Sprintf("ReqItemId%d", i)), t.Int(cm, fmt.Sprintf("ReqItemCount%d", i))
			if id == 0 || n <= 0 || id == src {
				continue
			}
			o := dataset.Objective{Kind: "item", ID: id, Count: n, Source: "cmangos", Drops: b.w.dropsByItem[id]}
			sort.Slice(o.Drops, func(a, c int) bool { return o.Drops[a].Chance > o.Drops[c].Chance })
			if len(o.Drops) > 12 {
				o.Drops = o.Drops[:12]
			}
			for _, d := range o.Drops {
				if d.NPC > 0 {
					b.npcWanted[d.NPC] = true
				} else {
					b.objWanted[d.Object] = true
				}
			}
			if len(o.Drops) == 0 && out.Unsupported == "" {
				out.Unsupported = fmt.Sprintf("item %d: source unknown (crafted, bought or not in the data)", id)
			}
			out.Objectives = append(out.Objectives, o)
		}
		if len(out.Objectives) > 0 {
			b.stats["objective_counts_from_cmangos"]++
		}
	}
	// objectives listed by the merged database that cmangos does not have
	known := map[string]bool{}
	for _, o := range out.Objectives {
		known[o.Kind+":"+strconv.Itoa(o.ID)] = true
	}
	list, _ := q["objectives"].([]any)
	for _, raw := range list {
		o, _ := raw.(map[string]any)
		kind, _ := o["kind"].(string)
		k := map[string]string{"npc": "kill", "item": "item", "object": "object"}[kind]
		for _, id := range ints(o["ids"]) {
			if known[k+":"+strconv.Itoa(id)] {
				continue
			}
			if n, ok := num(o["required_count"]); ok && n > 0 {
				out.Objectives = append(out.Objectives, dataset.Objective{Kind: k, ID: id, Count: int(n), Source: "merged"})
				continue
			}
			if out.Unsupported == "" {
				out.Unsupported = fmt.Sprintf("%s %d: count unknown (Forever objective not in cmangos)", kind, id)
			}
			b.stats["objectives_count_unknown"]++
		}
	}
	if cm == nil && len(out.Objectives) == 0 && out.Unsupported == "" {
		// a Forever quest cmangos does not know and whose objectives no source lists: it may be a delivery or a kill
		// quest; without knowing, the planner must not treat it as done on the spot
		out.Unsupported = "objectives unknown (Forever quest, not in cmangos)"
		b.stats["objectives_unknown_forever_quests"]++
	}
	for _, r := range out.Start {
		if r.Kind == "item" && out.Unsupported == "" {
			out.Unsupported = "started by an item"
		}
	}
	if out.Unsupported != "" {
		b.stats["quests_unsupported"]++
	}
}

func (b *builder) npcs() {
	// grind candidates up to the level range, plus every creature a quest refers to
	for _, raw := range b.merged["npcs"].(map[string]any) {
		n := raw.(map[string]any)
		if g, _ := n["potential_grind_candidate"].(bool); g {
			if mx, _ := num(n["max_level"]); int(mx) <= b.in.MaxQuestLevel+2 {
				id, _ := num(n["id"])
				b.npcWanted[int(id)] = true
			}
		}
	}
	ids := make([]int, 0, len(b.npcWanted))
	for id := range b.npcWanted {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		rec := b.npcRecord(id)
		cm := b.w.creature[id]
		if rec == nil && cm == nil {
			b.stats["npcs_unknown"]++
			continue
		}
		n := dataset.NPC{ID: id}
		if rec != nil {
			n.Name, _ = rec["name"].(string)
			if f, ok := num(rec["min_level"]); ok {
				n.MinLevel = int(f)
			}
			if f, ok := num(rec["max_level"]); ok {
				n.MaxLevel = int(f)
			}
			if f, ok := num(rec["hp_max"]); ok {
				n.HP = int(f)
			}
			if f, ok := num(rec["rank"]); ok {
				n.Rank = int(f)
			}
			switch rec["friendly_to_faction"] {
			case "A":
				n.Hostile = "H"
			case "H":
				n.Hostile = "A"
			case "AH":
				n.Hostile = "-"
			}
		}
		if cm != nil {
			t := b.w.ct
			if n.Name == "" {
				n.Name = t.String(cm, "Name")
			}
			if n.MinLevel == 0 {
				n.MinLevel, n.MaxLevel = t.Int(cm, "MinLevel"), t.Int(cm, "MaxLevel")
			}
			if n.HP == 0 {
				n.HP = t.Int(cm, "MaxLevelHealth")
			}
			n.Rank = t.Int(cm, "Rank")
			n.Dmg = round1((t.Float(cm, "MinMeleeDmg") + t.Float(cm, "MaxMeleeDmg")) / 2)
		}
		n.Respawn = b.w.respawn[id]
		// Forever positions first (converted to yards); cmangos positions for what Forever data lacks
		if rec != nil {
			if sp, _ := rec["spawns"].(map[string]any); len(sp) > 0 {
				keys := make([]string, 0, len(sp))
				for k := range sp {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, zone := range keys {
					t, ok := b.zoneT[zone]
					if !ok {
						continue
					}
					pts, _ := sp[zone].([]any)
					for _, raw := range pts {
						xy, _ := raw.([]any)
						if len(xy) != 2 {
							continue
						}
						px, _ := num(xy[0])
						py, _ := num(xy[1])
						n.Spawns = append(n.Spawns, t.apply(px, py))
					}
				}
				if len(n.Spawns) > 0 {
					n.Source = "forever_positions"
				}
			}
		}
		if len(n.Spawns) == 0 && len(b.w.spawns[id]) > 0 {
			n.Spawns = append(n.Spawns, b.w.spawns[id]...)
			n.Source = "cmangos_positions"
		}
		if len(n.Spawns) == 0 {
			n.Source = "no_positions"
			b.stats["npcs_without_positions"]++
		}
		n.Spawns = thin(n.Spawns, 80)
		if n.MaxLevel < n.MinLevel {
			n.MaxLevel = n.MinLevel
		}
		b.npcOut = append(b.npcOut, n)
	}
	b.stats["npcs"] = len(b.npcOut)
}

func (b *builder) objects() {
	ids := make([]int, 0, len(b.objWanted))
	for id := range b.objWanted {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	names := map[int]string{}
	for _, r := range b.w.gt.Rows {
		names[b.w.gt.Int(r, "entry")] = b.w.gt.String(r, "name")
	}
	for _, id := range ids {
		o := dataset.Object{ID: id, Name: names[id]}
		if rec, _ := get(b.merged, "objects", strconv.Itoa(id)).(map[string]any); rec != nil {
			if n, _ := rec["name"].(string); n != "" {
				o.Name = n
			}
			if sp, _ := rec["spawns"].(map[string]any); len(sp) > 0 {
				keys := make([]string, 0, len(sp))
				for k := range sp {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, zone := range keys {
					t, ok := b.zoneT[zone]
					if !ok {
						continue
					}
					pts, _ := sp[zone].([]any)
					for _, raw := range pts {
						xy, _ := raw.([]any)
						if len(xy) == 2 {
							px, _ := num(xy[0])
							py, _ := num(xy[1])
							o.Spawns = append(o.Spawns, t.apply(px, py))
						}
					}
				}
			}
		}
		if len(o.Spawns) == 0 {
			o.Spawns = b.w.objSpawns[id]
		}
		o.Spawns = thin(o.Spawns, 60)
		b.objOut = append(b.objOut, o)
	}
	b.stats["objects"] = len(b.objOut)
	b.stats["quests"] = len(b.out)
}

// thin keeps at most max positions, evenly picked, so huge spawn lists stay small without losing their spread.
func thin(ps []dataset.Point, max int) []dataset.Point {
	if len(ps) <= max {
		return ps
	}
	out := make([]dataset.Point, 0, max)
	step := float64(len(ps)) / float64(max)
	for i := 0; i < max; i++ {
		out = append(out, ps[int(float64(i)*step)])
	}
	return out
}

// applyWarcraftDB fills a quest from the Forever client facts WarcraftDB shows: XP, levels, quest givers, positions and,
// for quests whose objectives were unknown, the objectives with their amounts.
func (b *builder) applyWarcraftDB(out *dataset.Quest) {
	if b.wdb == nil {
		return
	}
	f, ok := b.wdb.Quests[out.ID]
	if !ok {
		return
	}
	const src = "warcraftdb_forever"
	if (out.Sources["xp"] == "unknown" || out.XP == 0) && f.XP > 0 {
		out.XP = f.XP
		out.Sources["xp"] = src
		b.stats["wdb_xp_filled"]++
	}
	if out.Level == 0 && f.Level > 0 {
		out.Level = f.Level
	}
	if out.MinLevel <= 1 && f.MinLevel > 1 {
		out.MinLevel = f.MinLevel
	}
	if len(out.Start) == 0 {
		for _, n := range f.StartNPC {
			out.Start = append(out.Start, dataset.Ref{Kind: "npc", ID: n})
			b.npcWanted[n] = true
		}
		out.Sources["start"] = src
	}
	if len(out.End) == 0 {
		for _, n := range f.EndNPC {
			out.End = append(out.End, dataset.Ref{Kind: "npc", ID: n})
			b.npcWanted[n] = true
		}
		out.Sources["end"] = src
	}
	place := func(ps []warcraftdb.Place) *dataset.Point {
		for _, p := range ps {
			if t, ok := b.uiT[strconv.Itoa(p.UIMap)]; ok {
				pt := t.apply(p.U*100, p.V*100)
				return &pt
			}
		}
		return nil
	}
	if out.StartAt == nil {
		if p := place(f.Starts); p != nil {
			out.StartAt = p
			out.Sources["startAt"] = src
		}
	}
	if out.EndAt == nil {
		if p := place(f.Ends); p != nil {
			out.EndAt = p
			out.Sources["endAt"] = src
		}
	}
	unknown := strings.Contains(out.Unsupported, "objectives unknown") || strings.Contains(out.Unsupported, "count unknown")
	if !unknown {
		return
	}
	var objs []dataset.Objective
	why := ""
	for _, o := range f.Objectives {
		switch o.Type {
		case 0:
			objs = append(objs, dataset.Objective{Kind: "kill", ID: o.ID, Count: o.Amount, Source: src})
			b.npcWanted[o.ID] = true
		case 1:
			d := dataset.Objective{Kind: "item", ID: o.ID, Count: o.Amount, Source: src, Drops: b.w.dropsByItem[o.ID]}
			sort.Slice(d.Drops, func(a, c int) bool { return d.Drops[a].Chance > d.Drops[c].Chance })
			if len(d.Drops) > 12 {
				d.Drops = d.Drops[:12]
			}
			for _, dr := range d.Drops {
				if dr.NPC > 0 {
					b.npcWanted[dr.NPC] = true
				} else {
					b.objWanted[dr.Object] = true
				}
			}
			if len(d.Drops) == 0 && why == "" {
				why = fmt.Sprintf("item %d: source unknown (Forever item, no drop data)", o.ID)
			}
			objs = append(objs, d)
		case 2:
			objs = append(objs, dataset.Objective{Kind: "object", ID: o.ID, Count: o.Amount, Source: src})
			b.objWanted[o.ID] = true
		case 3:
			objs = append(objs, dataset.Objective{Kind: "talk", ID: o.ID, Count: 1, Source: src})
			b.npcWanted[o.ID] = true
		default:
			if why == "" {
				why = fmt.Sprintf("objective type %d (reputation, area or spell) not modelled", o.Type)
			}
		}
	}
	out.Objectives = objs
	out.Unsupported = why
	b.stats["wdb_objectives_filled"]++
	if why == "" {
		b.stats["quests_unsupported"]--
	}
}

// dock is where a zeppelin or boat is boarded, anchored to an NPC standing there (cmangos positions). At overrides
// the NPC's first spawn when the NPC has several. Faction "" means both factions may use it.
type dock struct {
	NPC     int
	Faction string
	At      *dataset.Point
}

var docks = []dock{
	{NPC: 9564, Faction: "H"},  // Orgrimmar tower, to Undercity
	{NPC: 12136, Faction: "H"}, // Orgrimmar tower, to Grom'gol
	{NPC: 9566, Faction: "H"},  // Undercity tower, to Orgrimmar
	{NPC: 3150, Faction: "H"},  // Undercity tower, to Grom'gol
	{NPC: 3149, Faction: "H"},  // Grom'gol tower, to Orgrimmar
	{NPC: 12137, Faction: "H"}, // Grom'gol tower, to Undercity
	{NPC: 3179, Faction: "A"},  // Menethil Harbor pier
	{NPC: 12716, Faction: "A"}, // Theramore pier
	{NPC: 6086, Faction: "A", At: &dataset.Point{Map: 1, X: 6423, Y: 826}}, // Auberdine pier
	{NPC: 3607, Faction: "A"}, // Rut'theran Village pier
	{NPC: 9559, Faction: ""},  // Booty Bay, shipmaster
	{NPC: 9558, Faction: ""},  // Ratchet, shipmaster
}

// transports: one ride including the average wait takes about one full cycle of the transport (cmangos transports
// table, period in milliseconds rounded to seconds).
var transports = []dataset.Link{
	{From: 9564, To: 9566, Seconds: 356, Name: "Zeppelin Orgrimmar - Undercity"},
	{From: 12136, To: 3149, Seconds: 303, Name: "Zeppelin Orgrimmar - Grom'gol"},
	{From: 3150, To: 12137, Seconds: 333, Name: "Zeppelin Undercity - Grom'gol"},
	{From: 3179, To: 12716, Seconds: 329, Name: "Boat Menethil Harbor - Theramore"},
	{From: 3179, To: 6086, Seconds: 296, Name: "Boat Menethil Harbor - Auberdine"},
	{From: 6086, To: 3607, Seconds: 316, Name: "Boat Auberdine - Rut'theran Village"},
	{From: 9559, To: 9558, Seconds: 351, Name: "Boat Booty Bay - Ratchet"},
}

func (b *builder) travel() dataset.Travel {
	var t dataset.Travel
	faction := func(id int, sub string) string {
		if rec := b.npcRecord(id); rec != nil {
			switch rec["friendly_to_faction"] {
			case "A":
				return "A"
			case "H":
				return "H"
			}
		}
		switch sub {
		case "Gryphon Master", "Hippogryph Master":
			return "A"
		case "Wind Rider Master", "Bat Handler":
			return "H"
		}
		return ""
	}
	ct := b.w.ct
	for _, r := range ct.Rows {
		if ct.Int(r, "NpcFlags")&8 == 0 {
			continue
		}
		id := ct.Int(r, "Entry")
		sp := b.w.spawns[id]
		if len(sp) == 0 {
			continue
		}
		f := faction(id, ct.String(r, "SubName"))
		if f == "" {
			continue
		}
		t.FlightMasters = append(t.FlightMasters, dataset.Node{NPC: id, Name: ct.String(r, "Name"), Faction: f, At: sp[0]})
	}
	sort.Slice(t.FlightMasters, func(a, c int) bool { return t.FlightMasters[a].NPC < t.FlightMasters[c].NPC })
	for _, r := range ct.Rows {
		if ct.String(r, "SubName") != "Innkeeper" {
			continue
		}
		id := ct.Int(r, "Entry")
		if sp := b.w.spawns[id]; len(sp) > 0 {
			t.Inns = append(t.Inns, dataset.Node{NPC: id, Name: ct.String(r, "Name"), Faction: faction(id, ""), At: sp[0]})
		}
	}
	sort.Slice(t.Inns, func(a, c int) bool { return t.Inns[a].NPC < t.Inns[c].NPC })
	placed := map[int]bool{}
	for _, d := range docks {
		at := d.At
		if at == nil {
			if sp := b.w.spawns[d.NPC]; len(sp) > 0 {
				at = &sp[0]
			}
		}
		if at == nil {
			continue
		}
		t.Docks = append(t.Docks, dataset.Node{NPC: d.NPC, Name: ct.String(b.w.creature[d.NPC], "Name"), Faction: d.Faction, At: *at})
		placed[d.NPC] = true
	}
	for _, l := range transports {
		if !placed[l.From] || !placed[l.To] {
			continue
		}
		back := l
		back.From, back.To = l.To, l.From
		if i := strings.Index(l.Name, " - "); i > 0 {
			kind, rest, _ := strings.Cut(l.Name[:i], " ")
			back.Name = kind + " " + l.Name[i+3:] + " - " + rest
		}
		t.Links = append(t.Links, l, back)
	}
	b.stats["flight_masters"] = len(t.FlightMasters)
	b.stats["inns"] = len(t.Inns)
	b.stats["transport_links"] = len(t.Links)
	return t
}
