package merge

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Battlefeld/ForeverRoutes/internal/sources/everythingquests"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/foreverdb"
	"github.com/Battlefeld/ForeverRoutes/internal/sources/questie"
)

func inputs(t *testing.T) Inputs {
	t.Helper()
	xp := 170
	cat := &questie.Catalog{Revision: strings.Repeat("a", 40), Quests: map[int]*questie.Quest{
		1: {ID: 1, Name: "Questie name", Required: 1, Level: 2, StartedBy: questie.Refs{NPC: []int{10}, Object: []int{}, Item: []int{}},
			FinishedBy: questie.Refs{NPC: []int{10}, Object: []int{}}, PreAll: []int{}, PreAny: []int{}, XP: &xp,
			Objectives: []questie.Objective{{Kind: "npc", IDs: []int{20}, CountSource: "unresolved"}}},
		2: {ID: 2, Name: "Classic in API", Required: 3, Level: 4, StartedBy: questie.Refs{NPC: []int{11}, Object: []int{}, Item: []int{}},
			FinishedBy: questie.Refs{NPC: []int{}, Object: []int{}}, PreAll: []int{}, PreAny: []int{}, Objectives: []questie.Objective{}},
	}}
	rows := []string{
		`{"id":1,"name":"Forever name","level":2,"requiredLevel":1,"side":"h","source":"forever","url":"https://foreverdb.net/quest/1-x",
		  "starts":{"type":"npc","id":10,"location":{"uiMapId":1411,"x":42.1,"y":68.4,"source":"seen in WoW Forever"}},
		  "ends":{"type":"npc","id":10,"location":null}}`,
		`{"id":2,"name":"Classic API name","level":4,"requiredLevel":3,"source":"classic","url":"https://foreverdb.net/quest/2-y"}`,
		`{"id":3,"name":"Forever only","level":5,"requiredLevel":4,"source":"forever","url":"https://foreverdb.net/quest/3-z",
		  "starts":{"type":"npc","id":12},"ends":{"type":"object","id":13}}`,
	}
	snap := &foreverdb.Snapshot{Game: json.RawMessage(`{"build":"1.60.1.70450"}`), Source: foreverdb.Credit}
	for _, r := range rows {
		snap.Quests = append(snap.Quests, json.RawMessage(r))
	}
	ent := 14
	name, req, lvl := "Addon only", 5, 6
	addon := &everythingquests.Data{Revision: strings.Repeat("b", 40), Quests: map[int]*everythingquests.Quest{
		2: {ID: 2, Turnins: []everythingquests.Point{{UIMap: 1411, X: 50, Y: 50, Entity: &ent, Kind: "npc"}}, PreAll: []int{1}},
		4: {ID: 4, Name: &name, Required: &req, Level: &lvl},
	}}
	facts := &Facts{Schema: "foreverroutes.facts.v1", Quests: map[string]map[string]*Fact{
		"1": {"xp": {Value: 175, Source: "warcraftdb_forever", Quality: "forever_reference", URL: "https://forever.warcraftdb.com/data/quest/1"}},
	}}
	return Inputs{Questie: cat, ForeverDB: snap, Addon: addon, Facts: facts}
}

func TestMergePriorities(t *testing.T) {
	d, err := Merge(inputs(t))
	if err != nil {
		t.Fatal(err)
	}
	q1 := d.Quests["1"]
	if q1.Fields["name"].Value != "Forever name" || q1.Fields["name"].Source != "foreverdb_forever" {
		t.Errorf("a Forever value wins over QuestieDB: %+v", q1.Fields["name"])
	}
	if loc := q1.Locations["starts"]; loc == nil || loc.Quality != "forever_observed" {
		t.Errorf("seen position: %+v", loc)
	}
	if q1.Fields["xp"].Value != 175 || q1.Fields["xp"].Source != "warcraftdb_forever" {
		t.Errorf("a sourced fact wins over Classic XP: %+v", q1.Fields["xp"])
	}
	if q1.Objectives[0].RequiredCount != nil || q1.Objectives[0].QuantityQuality != "unresolved" {
		t.Error("an unknown count must stay unknown")
	}
	q2 := d.Quests["2"]
	if q2.Fields["name"].Value != "Classic in API" {
		t.Errorf("a Classic API value must not replace QuestieDB: %v", q2.Fields["name"].Value)
	}
	if q2.Fields["finished_by"].Source != "everythingquests_forever" || q2.NotActionable {
		t.Errorf("the addon fills a missing turn-in: %+v, not actionable %v", q2.Fields["finished_by"], q2.NotActionable)
	}
	if q2.Fields["prequest_all"].Source != "everythingquests_forever" {
		t.Error("the addon fills empty prerequisites")
	}
	if q3 := d.Quests["3"]; q3.Status != "forever_listed" || q3.NotActionable {
		t.Errorf("ForeverDB-only quest: %+v", q3)
	}
	if q4 := d.Quests["4"]; q4 == nil || q4.Status != "addon_reference_only" || !q4.NotActionable {
		t.Errorf("addon-only quest without giver must not be actionable: %+v", q4)
	}
	if len(d.Conflicts) < 2 {
		t.Errorf("conflicts should be recorded: %d", len(d.Conflicts))
	}
	if d.Build != "1.60.1.70450" || d.LevelXP["30"] != 47400 {
		t.Errorf("build %q level xp %v", d.Build, d.LevelXP["30"])
	}
}

func TestFactsNeedASource(t *testing.T) {
	in := inputs(t)
	in.Facts.Quests["1"]["xp"].URL = "http://example.com"
	if _, err := Merge(in); err == nil {
		t.Error("a fact without an https link must be refused")
	}
	in = inputs(t)
	in.Facts.Quests["1"]["xp"].Source = "my_guess"
	if _, err := Merge(in); err == nil {
		t.Error("a fact from an unknown source must be refused")
	}
}

func TestObjectivesFact(t *testing.T) {
	in := inputs(t)
	in.Facts.Quests["3"] = map[string]*Fact{"objectives": {Value: []any{
		map[string]any{"kind": "talk", "ids": []any{252476}, "required_count": 1},
		map[string]any{"kind": "npc", "ids": []any{10721}, "required_count": 3},
	}, Source: "wowhead_forever", Quality: "forever_reference", URL: "https://www.wowhead.com/forever/quest=3"}}
	d, err := Merge(in)
	if err != nil {
		t.Fatal(err)
	}
	o := d.Quests["3"].Objectives
	if len(o) != 2 || o[0].Kind != "talk" || *o[1].RequiredCount != 3 || d.Quests["3"].Fields["objectives"] != nil {
		t.Errorf("objectives %+v", o)
	}
	in.Facts.Quests["3"]["objectives"].Value = []any{map[string]any{"kind": "npc", "ids": []any{}, "required_count": 1}}
	if _, err := Merge(in); err == nil {
		t.Error("an objective without a target must be refused")
	}
}
