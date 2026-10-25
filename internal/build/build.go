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
	"github.com/Battlefeld/ForeverRoutes/internal/sources/aegis"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/eglink"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/foreverchanges"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/guides"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/polestar"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/spf"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/wclbox"
	"github.com/Battlefeld/ForeverRoutes/internal/sqldump"
	"github.com/Battlefeld/ForeverRoutes/internal/warcraftdb"
	"github.com/Battlefeld/ForeverRoutes/xp"
)

// Inputs names the source files and their pinned revisions.
type Inputs struct {
	Merged           string // merged quest database JSON (schema foreverroutes.unified_quests.v1)
	MergedRevision   string
	CMangos          string // cmangos classic-db full dump (.sql or .sql.gz)
	CMangosRevision  string
	WarcraftDB       string // optional overlay of Forever client quest facts
	WCLBox           string // optional overlay of item sources from WCLBox (internal/sources/wclbox)
	ForeverChanges   string // optional overlay of quest summaries from ForeverChanges (internal/sources/foreverchanges)
	MaxQuestLevel    int    // quests above this level are left out (0 = 45: routes to 40)
	Polestar         string // optional folder with Polestar's forever-db tables (internal/sources/polestar)
	PolestarRevision string
	// optional guide folders (internal/sources/guides): RestedXP's Forever guides and Polestar's
	GuidesRXP         string
	GuidesRXPRevision string
	GuidesPolestar    string
	Deathskips        string // optional: curated deathskips (data/sources/deathskips.json)
	Dungeons          string // optional: curated dungeons (data/sources/dungeons.json)
	SPF               string // optional: Shortest Path Forever's Data folder (internal/sources/spf)
	SPFRevision       string
	EGLink            string // optional: EG Link output's data/wow_classic_beta (internal/sources/eglink)
	EGLinkRevision    string
	Aegis             string // optional: AEGIS Pathfinder's data/merged.json (internal/sources/aegis)
	AegisRevision     string
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
	safeLocs, graveZones                            *sqldump.Table
	creature                                        map[int][]any
	spawns                                          map[int][]dataset.Point
	respawn                                         map[int]int
	quest                                           map[int][]any
	lootCreatures                                   map[int][]int // loot id -> creatures
	objSpawns                                       map[int][]dataset.Point
	objLoot                                         map[int][]int // loot id -> objects
	dropsByItem                                     map[int][]dataset.Drop
	inside                                          map[int]map[int]float64 // instance map -> creature -> spawns (a spawn of one of several creatures counts for each in part)
}

func loadCMangos(path string) (*world, error) {
	t, err := sqldump.ReadFile(path, "quest_template", "creature_template", "creature", "creature_loot_template",
		"gameobject", "gameobject_template", "gameobject_loot_template", "creature_questrelation",
		"creature_involvedrelation", "gameobject_questrelation", "gameobject_involvedrelation", "creature_spawn_entry",
		"gameobject_spawn_entry", "world_safe_locs", "game_graveyard_zone")
	if err != nil {
		return nil, err
	}
	w := &world{qt: t["quest_template"], ct: t["creature_template"], cr: t["creature"], cl: t["creature_loot_template"],
		gob: t["gameobject"], gt: t["gameobject_template"], gl: t["gameobject_loot_template"], cqr: t["creature_questrelation"],
		cir: t["creature_involvedrelation"], gqr: t["gameobject_questrelation"], gir: t["gameobject_involvedrelation"],
		safeLocs: t["world_safe_locs"], graveZones: t["game_graveyard_zone"],
		creature: map[int][]any{}, spawns: map[int][]dataset.Point{}, respawn: map[int]int{}, quest: map[int][]any{},
		lootCreatures: map[int][]int{}, inside: map[int]map[int]float64{}, objSpawns: map[int][]dataset.Point{}, objLoot: map[int][]int{}, dropsByItem: map[int][]dataset.Drop{}}
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
		if m != 0 && m != 1 { // instances and other maps are not part of open-world routes; dungeons count their trash
			if w.inside[m] == nil {
				w.inside[m] = map[int]float64{}
			}
			ids := idsOf(w.cr, r, crAlt)
			for _, id := range ids {
				w.inside[m][id] += 1 / float64(len(ids))
			}
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
		in.MaxQuestLevel = 45
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
	b := &builder{in: in, merged: merged, w: w, stats: map[string]int{}, npcWanted: map[int]bool{}, objWanted: map[int]bool{}, itemWhy: map[int]string{}}
	if in.WarcraftDB != "" {
		if b.wdb, err = warcraftdb.Load(in.WarcraftDB); err != nil {
			return nil, fmt.Errorf("warcraftdb overlay: %w", err)
		}
	}
	if in.WCLBox != "" {
		if b.wcl, err = loadWCLBox(in.WCLBox); err != nil {
			return nil, fmt.Errorf("wclbox overlay: %w", err)
		}
	}
	if in.ForeverChanges != "" {
		raw, err := os.ReadFile(in.ForeverChanges)
		if err != nil {
			return nil, fmt.Errorf("foreverchanges overlay: %w", err)
		}
		b.fch = &foreverchanges.Overlay{}
		if err := json.Unmarshal(raw, b.fch); err != nil {
			return nil, fmt.Errorf("foreverchanges overlay: %w", err)
		}
	}
	if in.Polestar != "" {
		if b.pdb, err = polestar.Load(in.Polestar, in.PolestarRevision); err != nil {
			return nil, err
		}
	}
	b.fitTransforms()
	b.indexNames()
	var facts []guides.Fact
	if in.GuidesRXP != "" {
		files, err := guides.ReadDir(in.GuidesRXP, ".lua")
		if err != nil {
			return nil, err
		}
		facts = append(facts, guides.ParseRXP(files)...)
		b.rxpSkips = guides.RXPDeathskips(files)
		b.rxpTargets = guides.RXPTargets(files)
		b.rxpQuests = guides.RXPQuests(files)
	}
	if in.GuidesPolestar != "" {
		files, err := guides.ReadDir(in.GuidesPolestar, ".guide")
		if err != nil {
			return nil, err
		}
		facts = append(facts, guides.ParsePolestar(files)...)
	}
	if in.Deathskips != "" {
		raw, err := os.ReadFile(in.Deathskips)
		if err != nil {
			return nil, err
		}
		var f struct {
			Skips []curatedSkip `json:"skips"`
		}
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, fmt.Errorf("deathskips: %w", err)
		}
		b.curatedSkips = f.Skips
	}
	if in.Dungeons != "" {
		raw, err := os.ReadFile(in.Dungeons)
		if err != nil {
			return nil, err
		}
		var f struct {
			Dungeons []curatedDungeon `json:"dungeons"`
		}
		if err := json.Unmarshal(raw, &f); err != nil {
			return nil, fmt.Errorf("dungeons: %w", err)
		}
		b.curatedDungeons = f.Dungeons
	}
	if in.SPF != "" {
		if b.spf, err = spf.Load(in.SPF, in.SPFRevision); err != nil {
			return nil, err
		}
	}
	if in.EGLink != "" {
		if b.egl, err = eglink.Load(in.EGLink, in.EGLinkRevision); err != nil {
			return nil, err
		}
	}
	if in.Aegis != "" {
		if b.aeg, err = aegis.Load(in.Aegis, in.AegisRevision); err != nil {
			return nil, err
		}
	}
	if len(facts) > 0 {
		b.loadGuides(facts)
		b.stats["guide_facts"] = len(facts)
	}
	b.quests()
	b.npcs()
	b.objects()
	b.guideSpawns()
	b.capitalNPCs()
	b.instantQuests()
	b.aegisQuests()
	b.egChances()
	// dungeon quests stay dungeon quests, also where their objectives lie before the portal: the mobs there are
	// elites in Forever (the outer caves and mines); the route with dungeons does them on its run
	dungeons := b.dungeons()
	travel := b.travel()
	build, _ := merged["build"].(string)
	d := &dataset.Dataset{Schema: dataset.Schema, Build: build, LevelXP: b.levelXP(), Quests: b.out, NPCs: b.npcOut, Objects: b.objOut, Travel: travel, Stats: b.stats,
		Dungeons: dungeons, Sources: b.mergedSources()}
	d.Sources = append(d.Sources, dataset.Source{Name: "cmangos classic-db", URL: "https://github.com/cmangos/classic-db", Revision: in.CMangosRevision, License: "GPL-3.0",
		Note: "Classic world data: objective counts, drop chances, spawn positions, respawn times, quest groups. Classic, not Forever: Forever data wins where both exist."})
	if b.spf != nil {
		d.Sources = append(d.Sources, dataset.Source{Name: "Shortest Path Forever", URL: "https://github.com/" + spf.Repository, Revision: in.SPFRevision, License: "GPL-3.0-or-later",
			Note: "by Cillian Berragan: flight times (most measured in game by InFlight, MIT), boat and zeppelin timetables, portals"})
	}
	if b.egl != nil {
		d.Sources = append(d.Sources, dataset.Source{Name: "EG Link output (Efficient Games)", URL: "https://github.com/" + eglink.Repository, Revision: in.EGLinkRevision, License: "CC BY 4.0",
			Note: eglink.Credit + ". Players' recordings over the Forever beta builds: drops with the chance per kill while on the quest, where objective counts went up. Combined over the builds by ForeverRoutes."})
	}
	if b.aeg != nil {
		d.Sources = append(d.Sources, dataset.Source{Name: "AEGIS: Pathfinder (Forever)", URL: "https://github.com/" + aegis.Repository, Revision: in.AegisRevision, License: "MIT",
			Note: "quest data players collected in game: where quests are taken and handed in, where objectives are, the XP a turn-in gave"})
	}
	if b.wcl != nil {
		d.Sources = append(d.Sources, dataset.Source{Name: "WCLBox", URL: "https://wowforever.wclbox.com", Revision: b.wcl.Fetched,
			Note: "WoW Forever database (新手盒子·魔兽世界无限数据库): which creature an item drops from, with the zone; the page gives no drop chance"})
	}
	if b.fch != nil {
		d.Sources = append(d.Sources, dataset.Source{Name: "ForeverChanges", URL: "https://foreverchanges.pro", Revision: b.fch.Fetched,
			Note: "WoW Forever database (client data and Wowhead's Forever data): kind of quest (normal, elite, dungeon) and the creature named in the objective sentence"})
	}
	if b.pdb != nil {
		d.Sources = append(d.Sources, dataset.Source{Name: "Polestar forever-db", URL: "https://github.com/" + polestar.Repository + "/tree/" + in.PolestarRevision + "/forever-db",
			Revision: in.PolestarRevision, License: "GPL-3.0",
			Note: "Forever client quest data (client build " + b.pdb.Build + "): Forever XP (dungeon quests raised), levels, race masks with Skyborne, objectives with counts, prerequisites, item sources, positions"})
	}
	if b.gdx != nil && b.gdx.credit["rxp"] {
		d.Sources = append(d.Sources, dataset.Source{Name: "RestedXP Forever guides", URL: "https://github.com/RestedXP/RXPGuides", Revision: in.GuidesRXPRevision, License: "CC BY-NC-SA 4.0",
			Note: "Where Forever quest items drop and where they lie (the creatures and places the guides send the player to); facts only, no guide text"})
	}
	if b.gdx != nil && b.gdx.credit["polestar"] {
		d.Sources = append(d.Sources, dataset.Source{Name: "Polestar Forever guides", URL: "https://github.com/" + polestar.Repository + "/tree/" + in.PolestarRevision + "/guides/forever", Revision: in.PolestarRevision, License: "GPL-3.0",
			Note: "Creatures, objects and places of quest objectives (kill, collect, interact steps); facts only"})
	}
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
	in              Inputs
	merged          obj
	w               *world
	stats           map[string]int
	zoneT           map[string]transform // questie zone id -> transform
	uiT             map[string]transform // uiMapId -> transform
	out             []dataset.Quest
	npcOut          []dataset.NPC
	objOut          []dataset.Object
	npcWanted       map[int]bool
	objWanted       map[int]bool
	wdb             *warcraftdb.Overlay
	wcl             *wclbox.Overlay
	fch             *foreverchanges.Overlay
	pdb             *polestar.DB
	aeg             *aegis.Data
	egl             *eglink.Data
	spf             *spf.Data
	rxpSkips        []guides.Deathskip
	rxpTargets      []guides.Sighting
	rxpQuests       map[int]*guides.QuestSteps
	curatedDungeons []curatedDungeon
	curatedSkips    []curatedSkip
	gdx             *guideIndex
	byName          map[string][]int // lower-case creature name -> cmangos entries
	zoneID          map[string]int   // lower-case zone name -> uiMapId
	itemWhy         map[int]string   // item -> why it has no usable source
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
	// the table ends where the Forever beta's level cap did; above it the Classic values (the same up to there)
	if len(out) < len(xp.ClassicLevelXP) {
		out = append(out, xp.ClassicLevelXP[len(out):]...)
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
	seen := map[int]bool{}
	for _, id := range ids {
		q := qs[strconv.Itoa(id)].(map[string]any)
		if na, _ := q["not_actionable"].(bool); na {
			b.stats["quests_not_actionable"]++
			continue
		}
		seen[id] = true
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
			out.Races = raceMask(int(f))
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
		if b.pdb != nil {
			b.applyPolestar(&out)
		}
		b.classify(&out)
		b.wantRefs(out)
		b.out = append(b.out, out)
	}
	if b.pdb == nil {
		return
	}
	extra := make([]int, 0)
	for id := range b.pdb.Quests {
		if !seen[id] {
			extra = append(extra, id)
		}
	}
	sort.Ints(extra)
	for _, id := range extra {
		out, ok := b.newPolestarQuest(b.pdb.Quests[id])
		if !ok {
			continue
		}
		b.classify(&out)
		b.wantRefs(out)
		b.out = append(b.out, out)
	}
	sort.Slice(b.out, func(a, c int) bool { return b.out[a].ID < b.out[c].ID })
}

// wantRefs marks the givers and turn-ins of a quest for the NPC and object tables.
func (b *builder) wantRefs(out dataset.Quest) {
	for _, r := range append(append([]dataset.Ref{}, out.Start...), out.End...) {
		switch r.Kind {
		case "npc":
			b.npcWanted[r.ID] = true
		case "object":
			b.objWanted[r.ID] = true
		}
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
			keep, ok := b.moreItemSources(out, &o)
			if !keep {
				continue
			}
			if !ok && out.Unsupported == "" {
				out.Unsupported = b.noSourceReason(id, fmt.Sprintf("item %d: source unknown (crafted, bought or not in the data)", id))
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
		k := map[string]string{"npc": "kill", "item": "item", "object": "object", "talk": "talk"}[kind]
		for _, id := range ints(o["ids"]) {
			if known[k+":"+strconv.Itoa(id)] {
				continue
			}
			if n, ok := num(o["required_count"]); ok && n > 0 && k != "" {
				obj := dataset.Objective{Kind: k, ID: id, Count: int(n), Source: "merged"}
				switch k {
				case "kill", "talk":
					b.npcWanted[id] = true
				case "object":
					b.objWanted[id] = true
				case "item":
					obj.Drops = b.w.dropsByItem[id]
					b.moreItemSources(out, &obj)
				}
				out.Objectives = append(out.Objectives, obj)
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
	b.startItem(out)
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
		n := dataset.NPC{ID: id}
		if rec == nil && cm == nil {
			if b.pdb == nil || !b.polestarNPC(&n, false) {
				b.stats["npcs_unknown"]++
				continue
			}
			b.stats["npcs_from_polestar"]++
			if len(n.Spawns) == 0 {
				n.Source = "no_positions"
				b.stats["npcs_without_positions"]++
			}
			n.Spawns = thin(n.Spawns, 80)
			if n.MaxLevel < n.MinLevel {
				n.MaxLevel = n.MinLevel
			}
			b.npcOut = append(b.npcOut, n)
			continue
		}
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
		if len(n.Spawns) == 0 && b.pdb != nil {
			b.polestarNPC(&n, true)
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
		if b.pdb != nil {
			b.polestarObject(&o)
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
			keep, ok := b.moreItemSources(out, &d)
			if !keep {
				continue
			}
			if !ok && b.delivered(out.ID, polestar.Objective{Target: o.ID, Count: o.Amount}) {
				b.stats["items_delivered_by_giver"]++
				out.Sources["objectives"] = "item assumed handed over by the giver (delivery quest)"
				continue
			}
			if !ok && why == "" {
				why = b.noSourceReason(o.ID, fmt.Sprintf("item %d: source unknown (Forever item, no drop data)", o.ID))
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

// startItem resolves a quest that only an item starts: the item's drops (cmangos, else QuestieDB's Classic rates)
// become where the quest begins. Without drops the quest is unsupported.
func (b *builder) startItem(out *dataset.Quest) {
	item := 0
	for _, r := range out.Start {
		switch r.Kind {
		case "npc", "object":
			return // a giver is enough
		case "item":
			if item == 0 {
				item = r.ID
			}
		}
	}
	if item == 0 {
		return
	}
	o := dataset.Objective{Kind: "item", ID: item, Count: 1, Drops: append([]dataset.Drop(nil), b.w.dropsByItem[item]...)}
	sort.Slice(o.Drops, func(a, c int) bool {
		if o.Drops[a].Chance != o.Drops[c].Chance {
			return o.Drops[a].Chance > o.Drops[c].Chance
		}
		return o.Drops[a].NPC+o.Drops[a].Object < o.Drops[c].NPC+o.Drops[c].Object
	})
	if len(o.Drops) == 0 && b.guideItem(out, &o, true) && len(o.Drops) == 0 && len(o.Places) > 0 {
		// the starting item lies on the ground: the quest starts where the guides find it
		at := o.Places[0]
		out.StartAt, out.Sources["start"], out.Sources["startAt"] = &at, "guide place", guideSrc
		b.stats["quests_started_by_guide_places"]++
		return
	}
	if len(o.Drops) == 0 {
		b.moreItemSources(out, &o)
	}
	if len(o.Drops) > 12 {
		o.Drops = o.Drops[:12]
	}
	if st := b.rxpQuests[out.ID]; len(o.Drops) == 0 && st != nil && len(st.UseAt) > 0 {
		// RestedXP has the player use the item at a place, no creature to kill: it is at hand there (a book a trainer
		// hands over, an item from the quest before)
		if pts := b.guidePoints(st.UseAt[:1]); len(pts) > 0 {
			out.StartAt, out.Sources["start"], out.Sources["startAt"] = &pts[0], "item used where RestedXP has it", "rxp"
			b.stats["quests_started_by_rxp_use"]++
			return
		}
	}
	if len(o.Drops) == 0 {
		if out.Unsupported == "" {
			out.Unsupported = "started by an item"
		}
		return
	}
	for _, d := range o.Drops {
		if d.NPC > 0 {
			b.npcWanted[d.NPC] = true
		} else {
			b.objWanted[d.Object] = true
		}
	}
	out.StartDrops = o.Drops
	out.Sources["start"] = "item drops"
	b.stats["quests_started_by_item_drops"]++
}

// dungeonZones are the dungeon and raid names; an item that only drops there is not a solo item.
var dungeonZones = []string{"Hall of Thanes", "Ragefire Chasm", "Ruins of Lordaeron", "Wailing Caverns", "The Deadmines", "Shadowfang Keep",
	"Excavation Site", "Blackfathom Deeps", "The Stockade", "City of Dalaran", "Gnomeregan", "Razorfen Kraul", "Scarlet Monastery",
	"The Drowned City", "Razorfen Downs", "Krol'dok Stronghold", "Uldaman", "Zul'Farrak", "Maraudon", "Alcaz Prison", "Sunken Temple",
	"Blackrock Depths", "Dire Maul", "Blackmaw Hold", "Blackrock Spire", "Scholomance", "Shaper's Terrace", "Stratholme"}

func isDungeon(zone string) bool {
	for _, d := range dungeonZones {
		if strings.HasPrefix(zone, d) || strings.Contains(zone, d) {
			return true
		}
	}
	return false
}

func loadWCLBox(path string) (*wclbox.Overlay, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var o wclbox.Overlay
	if err := json.Unmarshal(raw, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

// indexNames prepares the creature names and zone names that WCLBox's "Source: <creature>, <zone>" is matched against.
func (b *builder) indexNames() {
	b.byName = map[string][]int{}
	add := func(name string, id int) {
		if n := normName(name); n != "" {
			b.byName[n] = append(b.byName[n], id)
		}
	}
	for e, row := range b.w.creature {
		add(b.w.ct.String(row, "Name"), e)
	}
	for _, raw := range b.merged["npcs"].(map[string]any) { // creatures only Forever has
		n := raw.(map[string]any)
		id, _ := num(n["id"])
		name, _ := n["name"].(string)
		if _, inCMaNGOS := b.w.creature[int(id)]; !inCMaNGOS {
			add(name, int(id))
		}
	}
	for k, ids := range b.byName {
		sort.Ints(ids)
		b.byName[k] = dedupe(ids)
	}
	count := map[string]map[int]int{}
	for _, raw := range b.merged["quests"].(map[string]any) {
		z, _ := get(raw.(map[string]any), "fields", "zone", "value").(map[string]any)
		name, _ := z["name"].(string)
		ui, ok := num(z["uiMapId"])
		if name == "" || !ok || ui <= 0 {
			continue
		}
		k := normName(name)
		if count[k] == nil {
			count[k] = map[int]int{}
		}
		count[k][int(ui)]++
	}
	b.zoneID = map[string]int{}
	for k, m := range count {
		best, bn := 0, 0
		for ui, n := range m {
			if n > bn || n == bn && ui < best {
				best, bn = ui, n
			}
		}
		b.zoneID[k] = best
	}
}

func normName(s string) string { return strings.ToLower(strings.Trim(strings.TrimSpace(s), " .")) }

func dedupe(ids []int) []int {
	out := ids[:0]
	for i, id := range ids {
		if i == 0 || id != ids[i-1] {
			out = append(out, id)
		}
	}
	return out
}

// worldPoints are where a creature stands: Forever's positions converted to yards, plus cmangos' spawns.
func (b *builder) worldPoints(id int) []dataset.Point {
	pts := append([]dataset.Point(nil), b.w.spawns[id]...)
	if rec := b.npcRecord(id); rec != nil {
		sp, _ := rec["spawns"].(map[string]any)
		for zone, list := range sp {
			t, ok := b.zoneT[zone]
			if !ok {
				continue
			}
			l, _ := list.([]any)
			for _, raw := range l {
				xy, _ := raw.([]any)
				if len(xy) == 2 {
					px, _ := num(xy[0])
					py, _ := num(xy[1])
					pts = append(pts, t.apply(px, py))
				}
			}
		}
	}
	return pts
}

// creaturesNamed are the creatures with that name that stand in the zone (when the zone's map is known) and could be
// fought by a solo character: placed in the open world, not elite.
func (b *builder) creaturesNamed(name, zone string) []int {
	var out []int
	var t transform
	zoned := false
	if ui, ok := b.zoneID[normName(zone)]; ok {
		t, zoned = b.uiT[strconv.Itoa(ui)]
	}
	for _, id := range b.byName[normName(name)] {
		rank := -1
		if rec := b.npcRecord(id); rec != nil {
			if f, ok := num(rec["rank"]); ok {
				rank = int(f)
			}
		}
		if cm := b.w.creature[id]; cm != nil && rank < 0 {
			rank = b.w.ct.Int(cm, "Rank")
		}
		pts := b.worldPoints(id)
		if rank != 0 || len(pts) == 0 {
			continue
		}
		if zoned {
			in := false
			for _, p := range pts {
				if t.contains(p) {
					in = true
					break
				}
			}
			if !in {
				continue
			}
		}
		out = append(out, id)
	}
	return out
}

// classify marks an unresolved quest that ForeverChanges calls elite or a dungeon quest: it is not a data gap, a solo
// route does not do it.
func (b *builder) classify(out *dataset.Quest) {
	if b.fch == nil || out.Unsupported == "" {
		return
	}
	q, ok := b.fch.Quests[out.ID]
	if !ok || q.Kind == "" || q.Kind == "normal" {
		return
	}
	out.Unsupported = q.Kind + " quest, not solo (" + out.Unsupported + ")"
	out.NeedItems = nil
	b.stats["quests_not_solo"]++
}

// fromQuestText takes the creature a quest's objective sentence names ("... from Gnoll Thrashers in Redridge ...") as
// the source of the item, when the sentence names the item too and the creature is unambiguous in the zone.
func (b *builder) fromQuestText(questID int, o *dataset.Objective) bool {
	if b.fch == nil {
		return false
	}
	q, ok := b.fch.Quests[questID]
	if !ok || q.Text == "" {
		return false
	}
	name := b.itemName(o.ID)
	if name == "" || !strings.Contains(strings.ToLower(q.Text), strings.TrimSuffix(strings.ToLower(name), "s")) {
		return false
	}
	var found []int
	for _, src := range foreverchanges.Sources(q.Text) {
		if isDungeon(src.Zone) || isDungeon(src.Name) {
			b.itemWhy[o.ID] = "only in a dungeon or raid (" + src.Name + ", " + src.Zone + ")"
			b.stats["items_only_in_dungeons"]++
			return false
		}
		ids := b.creaturesNamed(src.Name, src.Zone)
		for _, alt := range singulars(src.Name) {
			if len(ids) == 0 {
				ids = b.creaturesNamed(alt, src.Zone)
			}
		}
		if _, known := b.zoneID[normName(src.Zone)]; !known && len(ids) > 3 {
			continue // zone not known and too many creatures with that name
		}
		found = append(found, ids...)
	}
	found = dedupeSorted(found)
	if len(found) == 0 {
		return false
	}
	for _, id := range found {
		if len(o.Drops) >= 12 {
			break
		}
		o.Drops = append(o.Drops, dataset.Drop{NPC: id, Unknown: true})
		b.npcWanted[id] = true
	}
	o.Source += "+foreverchanges_text"
	b.stats["items_sources_from_quest_text"]++
	return true
}

func dedupeSorted(ids []int) []int {
	sort.Ints(ids)
	return dedupe(ids)
}

// singulars are the likely singular forms of a plural creature name.
func singulars(n string) []string {
	var out []string
	switch {
	case strings.HasSuffix(n, "ies"):
		out = append(out, strings.TrimSuffix(n, "ies")+"y")
	case strings.HasSuffix(n, "ves"):
		out = append(out, strings.TrimSuffix(n, "ves")+"f", strings.TrimSuffix(n, "s"))
	case strings.HasSuffix(n, "es"):
		out = append(out, strings.TrimSuffix(n, "es"), strings.TrimSuffix(n, "s"))
	case strings.HasSuffix(n, "s"):
		out = append(out, strings.TrimSuffix(n, "s"))
	}
	return out
}

// itemName is the name of an item as the merged database or WCLBox gives it.
func (b *builder) itemName(id int) string {
	if rec, _ := get(b.merged, "items", strconv.Itoa(id)).(map[string]any); rec != nil {
		if n, _ := rec["name"].(string); n != "" {
			return n
		}
	}
	if b.wcl != nil {
		return b.wcl.Items[id].Name
	}
	return ""
}

// noSourceReason is the note for a quest whose item has no usable source: the specific reason found, else def.
func (b *builder) noSourceReason(item int, def string) string {
	if why := b.itemWhy[item]; why != "" {
		return fmt.Sprintf("item %d: %s", item, why)
	}
	return def
}

// moreItemSources completes an item objective cmangos has no drops for: QuestieDB's item table (Classic drop rates,
// vendors, quests that give it), then WCLBox's source line (creature and zone, no chance). An item that one earlier
// quest gives becomes that quest as a prerequisite and the objective is dropped (keep false). ok tells whether the
// item can be obtained; an item no source explains yet is listed in the quest's needItems.
func (b *builder) moreItemSources(out *dataset.Quest, o *dataset.Objective) (keep, ok bool) {
	if len(o.Drops) > 0 {
		return true, true
	}
	if it, known := b.wclQuestItem(out.ID, o.ID); known && it.Provided {
		b.stats["items_provided_by_giver_wclbox"]++
		return false, true
	}
	if b.polestarDrops(o) {
		return true, true
	}
	if rec, _ := get(b.merged, "items", strconv.Itoa(o.ID)).(map[string]any); rec != nil {
		if chances, _ := rec["drop_chance_by_npc"].(map[string]any); len(chances) > 0 {
			for k, v := range chances {
				npc, _ := strconv.Atoi(k)
				c, _ := num(get(obj{"v": v}, "v", "chance"))
				if npc > 0 && c > 0 {
					o.Drops = append(o.Drops, dataset.Drop{NPC: npc, Chance: round1(c * 100)})
					b.npcWanted[npc] = true
				}
			}
			sort.Slice(o.Drops, func(a, c int) bool {
				if o.Drops[a].Chance != o.Drops[c].Chance {
					return o.Drops[a].Chance > o.Drops[c].Chance
				}
				return o.Drops[a].NPC < o.Drops[c].NPC
			})
			if len(o.Drops) > 12 {
				o.Drops = o.Drops[:12]
			}
			if len(o.Drops) > 0 {
				o.Source += "+questiedb_drops"
				b.stats["items_drops_from_questiedb"]++
			}
		}
		for _, v := range ints(rec["vendors"]) {
			o.Vendors = append(o.Vendors, v)
			b.npcWanted[v] = true
		}
		if len(o.Vendors) > 0 {
			b.stats["items_sold_by_vendors"]++
		}
		if len(o.Drops) > 0 || len(o.Vendors) > 0 {
			return true, true
		}
		if rewards := ints(rec["quest_rewards"]); len(rewards) > 0 && o.Count <= 1 {
			switch {
			case len(rewards) == 1:
				out.PreAll = append(out.PreAll, rewards[0])
				b.stats["items_from_quest_rewards"]++
				return false, true
			case len(out.PreAny) == 0:
				out.PreAny = rewards
				b.stats["items_from_quest_rewards"]++
				return false, true
			}
		}
	}
	if b.fromEGLink(o) || b.fromWCLBoxQuest(out, o) || b.guideItem(out, o, false) || b.fromWCLBox(o) || b.fromAegis(out, o) || b.fromEGLinkPlaces(out, o) || b.fromQuestText(out.ID, o) {
		return true, true
	}
	if b.wcl == nil || b.wcl.Items[o.ID].ID == 0 {
		out.NeedItems = appendUnique(out.NeedItems, o.ID)
	}
	return true, false
}

func appendUnique(l []int, v int) []int {
	for _, x := range l {
		if x == v {
			return l
		}
	}
	return append(l, v)
}

// fromWCLBox turns WCLBox's source line into drops with an unknown chance. It records why an item stays without one.
func (b *builder) fromWCLBox(o *dataset.Objective) bool {
	if b.wcl == nil {
		return false
	}
	it := b.wcl.Items[o.ID]
	if it.ID == 0 {
		return false
	}
	if len(it.Sources) == 0 {
		b.itemWhy[o.ID] = "no source named by WCLBox"
		return false
	}
	var dungeon, other string
	for _, s := range it.Sources {
		switch {
		case isDungeon(s.Zone) || isDungeon(s.Name):
			dungeon = s.Name + ", " + s.Zone
			continue
		case s.Zone == "Crafting" || s.Zone == "Vendor":
			other = s.Name + " (" + s.Zone + ")"
			continue
		}
		for _, id := range b.creaturesNamed(s.Name, s.Zone) {
			if len(o.Drops) >= 12 {
				break
			}
			o.Drops = append(o.Drops, dataset.Drop{NPC: id, Unknown: true})
			b.npcWanted[id] = true
		}
		if len(o.Drops) == 0 {
			other = s.Name + ", " + s.Zone + " (no such creature in the data)"
		}
	}
	if len(o.Drops) > 0 {
		o.Source += "+wclbox"
		b.stats["items_sources_from_wclbox"]++
		return true
	}
	switch {
	case dungeon != "":
		b.itemWhy[o.ID] = "only in a dungeon or raid (" + dungeon + ")"
		b.stats["items_only_in_dungeons"]++
	case other != "":
		b.itemWhy[o.ID] = "source not usable: " + other
	}
	return false
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
	if b.pdb != nil {
		b.polestarTravel(&t)
	}
	if b.spf != nil {
		b.spfTravel(&t)
	}
	b.graveyards(&t)
	b.stats["flight_masters"] = len(t.FlightMasters)
	b.stats["inns"] = len(t.Inns)
	b.stats["transport_links"] = len(t.Links)
	return t
}
