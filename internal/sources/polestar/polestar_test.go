package polestar

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"quest.jsonl":           `{"id":214,"level":17,"min_level":14,"races":4294967373,"side":1,"title":"Red Silk Bandanas","type":81,"xp":2950,"zone_area":1581}` + "\n",
		"quest_start.jsonl":     `{"id":234,"kind":"npc","quest":214}` + "\n",
		"quest_end.jsonl":       `{"id":234,"kind":"npc","quest":214}` + "\n",
		"quest_objective.jsonl": `{"count":10,"kind":"item","n":1,"quest":214,"target":915}` + "\n",
		"quest_prereq.jsonl":    `{"mode":"any","pre":153,"quest":214}` + "\n",
		"npc.jsonl":             `{"faction_template":35,"id":234,"level_max":20,"level_min":20,"name":"Gryan Stoutmantle","rank":0}` + "\n",
		"object.jsonl":          "",
		"item.jsonl":            `{"id":915,"name":"Red Silk Bandana"}` + "\n",
		"item_source.jsonl":     `{"chance":30.5,"id":116,"item":915,"kind":"npc"}` + "\n" + `{"id":117,"item":915,"kind":"npc"}` + "\n",
		"spawn.jsonl":           `{"id":234,"kind":"npc","map":0,"n":1,"x":-10510,"y":1047}` + "\n",
		"ui_map.jsonl":          `{"bottom":1247.917,"left":4231.25,"map":2991,"right":-1331.25,"top":4956.25,"ui_map":2521}` + "\n",
		"index.json":            `{"clientBuild":"1.60.1.70532"}`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	db, err := Load(dir, "rev")
	if err != nil {
		t.Fatal(err)
	}
	q := db.Quests[214]
	if q == nil || q.XP != 2950 || q.Races&SkyborneAlliance == 0 || q.Type != 81 || db.Build != "1.60.1.70532" {
		t.Fatalf("quest %+v build %q", q, db.Build)
	}
	if len(db.Objectives[214]) != 1 || db.Objectives[214][0].Count != 10 || len(db.Prereqs[214]) != 1 || db.Starts[214][0].ID != 234 {
		t.Errorf("quest tables: %+v %+v %+v", db.Objectives[214], db.Prereqs[214], db.Starts[214])
	}
	src := db.ItemSources[915]
	if len(src) != 2 || src[0].Chance == nil || *src[0].Chance != 30.5 || src[1].Chance != nil {
		t.Errorf("item sources %+v", src)
	}
	if sp := db.Spawns["npc"][234]; len(sp) != 1 || sp[0].X != -10510 || len(db.UIMaps) != 1 {
		t.Errorf("spawns %+v maps %+v", sp, db.UIMaps)
	}
}
