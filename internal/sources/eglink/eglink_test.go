package eglink

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	for build, loot := range map[string]string{
		"1.60.1.1": `{"sources":{"npc:10|quest":{"windows":6,"items":{"500":{"drops":3,"chance":0.2}},"quests":{"77":{"windows":4,"items":{"500":{"drops":3,"chance":0.75}}}}},
			"npc:10|spell:8613":{"windows":9,"items":{"600":{"drops":9,"chance":1}}},"object:20|spell:22810":{"windows":5,"items":{"700":{"drops":5,"chance":1}}}}}`,
		"1.60.1.2": `{"sources":{"npc:10|quest|group":{"windows":4,"items":{},"quests":{"77":{"windows":4,"items":{"500":{"drops":1,"chance":0.25}}}}}}}`,
	} {
		os.MkdirAll(filepath.Join(dir, build), 0o755)
		os.WriteFile(filepath.Join(dir, build, "loot.json"), []byte(loot), 0o644)
	}
	os.WriteFile(filepath.Join(dir, "1.60.1.2", "objectives.json"), []byte(`{"quests":{"77":{"1":{"positions":[{"instance":1,"worldX":-498,"worldY":-4282.6}]}}}}`), 0o644)
	d, err := Load(dir, "rev")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Builds) != 2 {
		t.Fatalf("builds %v", d.Builds)
	}
	l := d.Drops[500]
	if len(l) != 1 || l[0].Kind != "npc" || l[0].ID != 10 || l[0].Windows != 8 || l[0].Chance != 0.5 {
		t.Fatalf("quest item: %+v", l)
	}
	if len(d.Drops[600]) != 0 {
		t.Fatalf("skinning kept: %+v", d.Drops[600])
	}
	if o := d.Drops[700]; len(o) != 1 || o[0].Kind != "object" || o[0].Chance != 1 {
		t.Fatalf("object: %+v", o)
	}
	if p := d.Objectives[77][1]; len(p) != 1 || p[0].Map != 1 || p[0].X != -498 {
		t.Fatalf("objectives: %+v", d.Objectives)
	}
}
