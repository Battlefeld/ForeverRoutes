// Package merge joins the quest sources field by field into the merged quest database
// (foreverroutes.unified_quests.v1) that the dataset build reads. Every value keeps its source, quality and link.
//
// Priority: values seen in WoW Forever, then Forever client values, then QuestieDB Forever and addon data, then Classic
// references. A Classic value never replaces a QuestieDB Forever value. Individually sourced facts (web databases and
// in-game observations, data/sources/manual-facts.json) override generic references. Nothing is invented: an unknown
// objective count stays unknown.
package merge

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/internal/sources/everythingquests"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/foreverdb"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/questie"
	"github.com/Battlefeld/ForeverRoutes/xp"
)

// Schema of the merged database.
const Schema = "foreverroutes.unified_quests.v1"

const (
	urlQuestie = "https://github.com/Questie/QuestieDB"
	urlForever = "https://foreverdb.net"
)

// Fact is one value with where it comes from.
type Fact struct {
	Value          any    `json:"value"`
	Source         string `json:"source"`
	Quality        string `json:"quality"` // forever_observed, forever_reference, addon_reference, classic_reference
	URL            string `json:"url"`
	Build          string `json:"build,omitempty"`
	OriginalSource string `json:"original_source,omitempty"`
	Revision       string `json:"revision,omitempty"`
	Evidence       string `json:"evidence,omitempty"`
	Players        any    `json:"players,omitempty"`
}

// Objective is a quest target; counts come from other sources later (cmangos, WarcraftDB) or stay unknown.
type Objective struct {
	Kind            string `json:"kind"`
	IDs             []int  `json:"ids"`
	ObjectiveIndex  *int   `json:"objective_index,omitempty"`
	RequiredCount   *int   `json:"required_count"`
	CountSource     string `json:"count_source"`
	QuantityQuality string `json:"quantity_quality"`
	TargetSource    string `json:"target_source,omitempty"`
}

// Crosscheck keeps what EverythingQuests says about a quest, for audits.
type Crosscheck struct {
	Starts  []everythingquests.Point  `json:"starts"`
	Turnins []everythingquests.Point  `json:"turnins"`
	Targets []everythingquests.Target `json:"target_refs"`
	PreAny  []int                     `json:"prereq_any"`
	PreAll  []int                     `json:"prereq_all"`
	Source  map[string]string         `json:"source"`
}

// Quest of the merged database.
type Quest struct {
	ID            int              `json:"id"`
	Status        string           `json:"status"` // forever_listed, questie_reference_only, addon_reference_only
	Presence      []string         `json:"source_presence"`
	Known         *bool            `json:"known_in_forever"`
	Fields        map[string]*Fact `json:"fields"`
	Objectives    []Objective      `json:"objectives"`
	Locations     map[string]*Fact `json:"locations"`
	NotActionable bool             `json:"not_actionable"`
	ForeverDBURL  *string          `json:"foreverdb_url"`
	Addon         *Crosscheck      `json:"addon_crosscheck,omitempty"`
}

// Conflict records two sources disagreeing on a field.
type Conflict struct {
	Quest    int    `json:"quest_id"`
	Field    string `json:"field"`
	Base     *Fact  `json:"questie"`
	Incoming *Fact  `json:"foreverdb"`
	Selected string `json:"selected"`
}

// Facts is the file of individually sourced facts.
type Facts struct {
	Schema string                      `json:"schema"`
	Quests map[string]map[string]*Fact `json:"quests"`
}

// Doc is the merged database.
type Doc struct {
	Schema    string                     `json:"schema"`
	Build     string                     `json:"build"`
	Policy    string                     `json:"policy"`
	Sources   map[string]any             `json:"sources"`
	Stats     map[string]int             `json:"stats"`
	Quests    map[string]*Quest          `json:"quests"`
	NPCs      map[string]*questie.NPC    `json:"npcs"`
	Objects   map[string]*questie.Object `json:"objects"`
	Items     map[string]*questie.Item   `json:"items"`
	LevelXP   map[string]int             `json:"level_xp_classic"`
	Changes   []json.RawMessage          `json:"changes_seen_unverified"`
	Conflicts []Conflict                 `json:"field_conflicts"`
	Addon     map[string]string          `json:"addon_source"`
}

// Inputs of a merge.
type Inputs struct {
	Questie   *questie.Catalog
	ForeverDB *foreverdb.Snapshot
	Addon     *everythingquests.Data // optional
	Facts     *Facts                 // optional
}

func classicOrForever(source string) (string, string) {
	if source == "forever" {
		return "foreverdb_forever", "forever_reference"
	}
	return "foreverdb_classic", "classic_reference"
}

func apiFact(qid int, value any, source, url, build string) *Fact {
	if value == nil || (reflect.ValueOf(value).Kind() == reflect.Ptr && reflect.ValueOf(value).IsNil()) {
		return nil
	}
	if rv := reflect.ValueOf(value); rv.Kind() == reflect.Ptr {
		value = rv.Elem().Interface()
	}
	src, q := classicOrForever(source)
	if url == "" {
		url = urlForever + "/quest/" + strconv.Itoa(qid)
	}
	return &Fact{Value: value, Source: src, Quality: q, URL: url, Build: build}
}

func starter(n *foreverdb.Node, qid int, source, build string, withItem bool) *Fact {
	if n == nil || n.ID <= 0 || (n.Type != "npc" && n.Type != "object" && n.Type != "item") {
		return nil
	}
	refs := map[string][]int{"npc": {}, "object": {}, "item": {}}
	refs[n.Type] = []int{n.ID}
	for _, o := range n.Others {
		if _, ok := refs[o.Type]; ok && o.ID > 0 && !contains(refs[o.Type], o.ID) {
			refs[o.Type] = append(refs[o.Type], o.ID)
		}
	}
	if !withItem {
		delete(refs, "item")
	}
	return apiFact(qid, refs, source, "", build)
}

func location(n *foreverdb.Node, qid int, build string) *Fact {
	if n == nil || n.Location == nil || n.Location.UIMap <= 0 || n.Location.X == nil || n.Location.Y == nil {
		return nil
	}
	x, y := *n.Location.X, *n.Location.Y
	if x < 0 || x > 100 || y < 0 || y > 100 {
		return nil
	}
	label := n.Location.Source
	if label == "" {
		label = n.Source
	}
	if label == "" {
		label = "unverified"
	}
	f := &Fact{Value: map[string]any{"uiMapId": n.Location.UIMap, "x": x, "y": y}, Source: "foreverdb_classic_position",
		Quality: "classic_reference", URL: urlForever + "/quest/" + strconv.Itoa(qid), Build: build, OriginalSource: label}
	if strings.HasPrefix(label, "seen in WoW Forever") {
		f.Source, f.Quality = "foreverdb_observed_position", "forever_observed"
	}
	return f
}

func contains(l []int, v int) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}
	return false
}

func sameValue(a, b any) bool {
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	return string(ja) == string(jb)
}

// choose keeps base against a Classic incoming value, else takes incoming; disagreements are recorded.
func choose(field string, base, incoming *Fact, conflicts *[]Conflict, qid int) *Fact {
	if incoming == nil {
		return base
	}
	if base != nil && !sameValue(base.Value, incoming.Value) {
		sel := "questie"
		if incoming.Quality != "classic_reference" {
			sel = "foreverdb"
		}
		*conflicts = append(*conflicts, Conflict{Quest: qid, Field: field, Base: base, Incoming: incoming, Selected: sel})
	}
	if base != nil && incoming.Quality == "classic_reference" {
		return base
	}
	return incoming
}

func anyRefs(f *Fact) bool {
	if f == nil {
		return false
	}
	switch v := f.Value.(type) {
	case map[string][]int:
		for _, l := range v {
			if len(l) > 0 {
				return true
			}
		}
	}
	return false
}

// Merge joins the sources.
func Merge(in Inputs) (*Doc, error) {
	if in.Questie == nil || in.ForeverDB == nil {
		return nil, fmt.Errorf("merge: QuestieDB and ForeverDB are required")
	}
	api, err := in.ForeverDB.List()
	if err != nil {
		return nil, err
	}
	byAPI := map[int]*foreverdb.Quest{}
	for i := range api {
		byAPI[api[i].ID] = &api[i]
	}
	build := in.ForeverDB.Build()
	qURL := urlQuestie + "/tree/" + in.Questie.Revision
	var conflicts []Conflict
	quests := map[string]*Quest{}
	ids := map[int]bool{}
	for id := range in.Questie.Quests {
		ids[id] = true
	}
	for id := range byAPI {
		ids[id] = true
	}
	order := make([]int, 0, len(ids))
	for id := range ids {
		order = append(order, id)
	}
	sort.Ints(order)
	for _, id := range order {
		base := in.Questie.Quests[id]
		a := byAPI[id]
		fields := map[string]*Fact{}
		if base != nil {
			put := func(k string, v any) {
				if v != nil {
					fields[k] = &Fact{Value: v, Source: "questiedb_forever", Quality: "classic_reference", URL: qURL}
				}
			}
			put("name", base.Name)
			put("required_level", base.Required)
			put("quest_level", base.Level)
			put("required_races_mask", base.Races)
			put("required_classes_mask", base.Classes)
			put("started_by", map[string][]int{"npc": base.StartedBy.NPC, "object": base.StartedBy.Object, "item": base.StartedBy.Item})
			put("finished_by", map[string][]int{"npc": base.FinishedBy.NPC, "object": base.FinishedBy.Object})
			put("prequest_all", base.PreAll)
			put("prequest_any", base.PreAny)
			put("zone_or_sort", base.ZoneOrSort)
			if base.XP != nil {
				fields["xp"] = &Fact{Value: *base.XP, Source: "questiedb_classic_xp", Quality: "classic_reference", URL: qURL}
			}
		}
		locations := map[string]*Fact{}
		if a != nil {
			src := a.Source
			if src != "forever" {
				src = "classic"
			}
			fields["name"] = choose("name", fields["name"], apiFact(id, a.Name, src, a.URL, build), &conflicts, id)
			fields["required_level"] = choose("required_level", fields["required_level"], apiFact(id, a.RequiredLevel, src, a.URL, build), &conflicts, id)
			fields["quest_level"] = choose("quest_level", fields["quest_level"], apiFact(id, a.Level, src, a.URL, build), &conflicts, id)
			for _, side := range []struct {
				field, node string
				n           *foreverdb.Node
				item        bool
			}{{"started_by", "starts", a.Starts, true}, {"finished_by", "ends", a.Ends, false}} {
				if l := location(side.n, id, build); l != nil {
					locations[side.node] = l
				}
				fields[side.field] = choose(side.field, fields[side.field], starter(side.n, id, src, build, side.item), &conflicts, id)
			}
			if a.Zone != nil {
				fields["zone"] = apiFact(id, map[string]any{"name": valueOrNilS(a.Zone.Name), "uiMapId": valueOrNil(a.Zone.UIMap)}, src, "", build)
			}
			switch a.Side {
			case "a", "h", "both", "n":
				fields["side"] = apiFact(id, a.Side, src, "", build)
			}
		}
		for k, v := range fields {
			if v == nil {
				delete(fields, k)
			}
		}
		q := &Quest{ID: id, Status: "questie_reference_only", Presence: []string{}, Fields: fields, Objectives: []Objective{}, Locations: locations}
		if base != nil {
			q.Presence = append(q.Presence, "questiedb")
			for _, o := range base.Objectives {
				q.Objectives = append(q.Objectives, Objective{Kind: o.Kind, IDs: o.IDs, RequiredCount: o.RequiredCount,
					CountSource: o.CountSource, QuantityQuality: quantity(o.RequiredCount)})
			}
		}
		if a != nil {
			q.Status = "forever_listed"
			q.Presence = append(q.Presence, "foreverdb")
			q.Known = a.Known
			u := a.URL
			q.ForeverDBURL = &u
		}
		q.NotActionable = !(anyRefs(fields["started_by"]) && anyRefs(fields["finished_by"]))
		quests[strconv.Itoa(id)] = q
	}
	audit := map[string]int{}
	if in.Addon != nil {
		mergeAddon(quests, in.Addon, audit)
	}
	if in.Facts != nil {
		if err := applyFacts(quests, in.Facts, &conflicts, audit); err != nil {
			return nil, err
		}
	}
	doc := &Doc{Schema: Schema, Build: build,
		Policy:  "Classic references accepted as provisional simulation inputs; unverified fields remain explicit.",
		Sources: map[string]any{"questiedb": map[string]string{"url": urlQuestie, "revision": in.Questie.Revision}, "foreverdb": in.ForeverDB.Source, "foreverdb_game_metadata": in.ForeverDB.Game},
		Quests:  quests, NPCs: map[string]*questie.NPC{}, Objects: map[string]*questie.Object{}, Items: map[string]*questie.Item{},
		LevelXP: map[string]int{}, Changes: in.ForeverDB.Changes, Conflicts: conflicts}
	if doc.Conflicts == nil {
		doc.Conflicts = []Conflict{}
	}
	if in.Addon != nil {
		doc.Addon = addonSource(in.Addon)
		doc.Sources["everythingquests"] = doc.Addon
	}
	for id, n := range in.Questie.NPCs {
		doc.NPCs[strconv.Itoa(id)] = n
	}
	for id, o := range in.Questie.Objects {
		doc.Objects[strconv.Itoa(id)] = o
	}
	for id, it := range in.Questie.Items {
		doc.Items[strconv.Itoa(id)] = it
	}
	for i, v := range xp.ClassicLevelXP[:30] {
		doc.LevelXP[strconv.Itoa(i+1)] = v
	}
	doc.Stats = stats(doc, audit)
	return doc, nil
}

func quantity(n *int) string {
	if n == nil {
		return "unresolved"
	}
	return "source_reference"
}

func addonSource(d *everythingquests.Data) map[string]string {
	return map[string]string{"name": "EverythingQuests Forever", "repository": "https://github.com/" + everythingquests.Repository,
		"revision": d.Revision, "license": "MIT; credits upstream AllTheThings where used"}
}

// mergeAddon fills what the other sources lack from EverythingQuests; it never replaces a value.
func mergeAddon(quests map[string]*Quest, d *everythingquests.Data, audit map[string]int) {
	repo := "https://github.com/" + everythingquests.Repository
	fact := func(v any) *Fact {
		return &Fact{Value: v, Source: "everythingquests_forever", Quality: "addon_reference", URL: repo, Revision: d.Revision}
	}
	ids := make([]int, 0, len(d.Quests))
	for id := range d.Quests {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	for _, id := range ids {
		in := d.Quests[id]
		key := strconv.Itoa(id)
		q := quests[key]
		if q == nil {
			if in.Required == nil || *in.Required < 1 || *in.Required > 30 || in.Name == nil || strings.TrimSpace(*in.Name) == "" {
				audit["addon_unmatched_addon_quest_ids"]++
				continue
			}
			q = &Quest{ID: id, Status: "addon_reference_only", NotActionable: true, Presence: []string{"everythingquests"},
				Fields:     map[string]*Fact{"name": fact(*in.Name), "required_level": fact(*in.Required), "quest_level": fact(valueOrNil(in.Level))},
				Objectives: []Objective{}, Locations: map[string]*Fact{}}
			quests[key] = q
			audit["addon_added_quest_ids"]++
		} else {
			audit["addon_matched_quest_ids"]++
		}
		q.Addon = &Crosscheck{Starts: in.Starts, Turnins: in.Turnins, Targets: in.Targets, PreAny: in.PreAny, PreAll: in.PreAll,
			Source: map[string]string{"source": "everythingquests_forever", "url": repo, "revision": d.Revision, "quality": "addon_reference"}}
		if in.Name != nil && q.Fields["name"] == nil {
			q.Fields["name"] = fact(*in.Name)
		}
		for _, side := range []struct {
			label string
			pts   []everythingquests.Point
		}{{"starts", in.Starts}, {"ends", in.Turnins}} {
			if q.Locations[side.label] == nil && len(side.pts) > 0 {
				p := side.pts[0]
				q.Locations[side.label] = fact(map[string]any{"uiMapId": p.UIMap, "x": p.X, "y": p.Y})
				audit["addon_filled_"+map[string]string{"starts": "start", "ends": "turnin"}[side.label]+"_locations"]++
			}
		}
		for _, side := range []struct {
			field string
			pts   []everythingquests.Point
			item  bool
		}{{"started_by", in.Starts, true}, {"finished_by", in.Turnins, false}} {
			if anyRefs(q.Fields[side.field]) {
				continue
			}
			refs := map[string][]int{"npc": {}, "object": {}}
			if side.item {
				refs["item"] = []int{}
			}
			for _, p := range side.pts {
				if l, ok := refs[p.Kind]; ok && p.Entity != nil && !contains(l, *p.Entity) {
					refs[p.Kind] = append(l, *p.Entity)
				}
			}
			f := fact(refs)
			if anyRefs(f) {
				q.Fields[side.field] = f
			}
		}
		if empty(q.Fields["prequest_all"]) && len(in.PreAll) > 0 {
			q.Fields["prequest_all"] = fact(in.PreAll)
			audit["addon_filled_prerequisites"]++
		}
		if empty(q.Fields["prequest_any"]) && len(in.PreAny) > 0 {
			q.Fields["prequest_any"] = fact(in.PreAny)
			audit["addon_filled_prerequisites"]++
		}
		if len(q.Objectives) == 0 {
			type group struct {
				kind  string
				index int
			}
			by := map[group][]int{}
			var keys []group
			for _, t := range in.Targets {
				if t.Kind != "npc" && t.Kind != "object" {
					continue
				}
				g := group{t.Kind, t.Objective}
				if _, ok := by[g]; !ok {
					keys = append(keys, g)
				}
				if !contains(by[g], t.ID) {
					by[g] = append(by[g], t.ID)
				}
			}
			sort.Slice(keys, func(a, b int) bool {
				if keys[a].kind != keys[b].kind {
					return keys[a].kind < keys[b].kind
				}
				return keys[a].index < keys[b].index
			})
			for _, g := range keys {
				idx := g.index
				q.Objectives = append(q.Objectives, Objective{Kind: g.kind, IDs: by[g], ObjectiveIndex: &idx, CountSource: "unresolved",
					QuantityQuality: "unresolved", TargetSource: "everythingquests_forever"})
				audit["addon_filled_objectives"]++
			}
		}
		q.NotActionable = !(anyRefs(q.Fields["started_by"]) && anyRefs(q.Fields["finished_by"]))
	}
}

func valueOrNil(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

func valueOrNilS(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

// empty is a missing fact or one with an empty list (Python's falsy test of the original).
func empty(f *Fact) bool {
	if f == nil {
		return true
	}
	if l, ok := f.Value.([]int); ok {
		return len(l) == 0
	}
	return false
}

var factSources = map[string]string{
	"warcraftdb_forever": "forever_reference", "warcraftdb_classic": "classic_reference", "octowow_classic": "classic_reference",
	"classicdb_classic": "classic_reference", "wowhead_classic": "classic_reference", "wowhead_forever": "forever_reference",
	"foreverdb_client": "forever_reference", "foreverdb_observed": "forever_observed",
}

// applyFacts sets the individually sourced facts; each needs a known source, a matching quality and an https link.
func applyFacts(quests map[string]*Quest, f *Facts, conflicts *[]Conflict, audit map[string]int) error {
	if f.Schema != "foreverroutes.facts.v1" {
		return fmt.Errorf("merge: unknown facts schema %q", f.Schema)
	}
	for key, fields := range f.Quests {
		q := quests[key]
		if q == nil {
			audit["facts_for_unknown_quests"]++
			continue
		}
		id, _ := strconv.Atoi(key)
		for name, fact := range fields {
			want, ok := factSources[fact.Source]
			if !ok || fact.Quality != want || !strings.HasPrefix(fact.URL, "https://") {
				return fmt.Errorf("merge: fact %s.%s has an unknown source, wrong quality or no https link", key, name)
			}
			if old := q.Fields[name]; old != nil && !sameValue(old.Value, fact.Value) {
				*conflicts = append(*conflicts, Conflict{Quest: id, Field: name, Base: old, Incoming: fact, Selected: "fact"})
			}
			q.Fields[name] = fact
			audit["facts_applied"]++
		}
	}
	return nil
}

func stats(d *Doc, audit map[string]int) map[string]int {
	s := map[string]int{}
	for k, v := range audit {
		s[k] = v
	}
	for _, q := range d.Quests {
		s["identified_quests"]++
		switch q.Status {
		case "addon_reference_only":
			s["addon_only_quests"]++
		case "forever_listed":
			s["foreverdb_quests"]++
		}
		for _, p := range q.Presence {
			if p == "questiedb" {
				s["questiedb_quests"]++
			}
		}
		if q.Fields["xp"] == nil && q.Fields["xp_observed_by_player_level"] == nil {
			s["quests_missing_xp"]++
		}
		if len(q.Objectives) == 0 {
			s["quests_without_objective_references"]++
		}
		for _, o := range q.Objectives {
			if o.RequiredCount == nil {
				s["objectives_unknown_quantity"]++
			}
		}
		if q.NotActionable {
			s["quests_without_complete_giver_turnin"]++
		}
		for _, l := range q.Locations {
			if l.Quality == "forever_observed" {
				s["forever_observed_locations"]++
			}
		}
	}
	s["npc_reference_records"] = len(d.NPCs)
	for _, n := range d.NPCs {
		if n.HPMin != nil {
			s["npc_hp_classic_references"]++
		}
	}
	s["item_reference_records"] = len(d.Items)
	s["object_reference_records"] = len(d.Objects)
	s["conflicts"] = len(d.Conflicts)
	return s
}
