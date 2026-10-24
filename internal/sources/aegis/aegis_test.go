package aegis

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "merged.json")
	os.WriteFile(path, []byte(`{"levels":{"2":{"xpMax":{"min":900,"max":900,"count":3}}},"quests":{"92456":{"title":"A Green Sample",
		"pickups":{"npc:1@1424":{"kind":"npc","id":1,"mapID":1424,"x":0.25,"y":0.5,"count":2},"npc:2@1424":{"kind":"npc","id":2,"mapID":1424,"x":0.3,"y":0.5,"count":5}},
		"poi":{"objective":{"1424:24:7":79,"1415:45:31":40}},"rewards":{"xp":{"20":{"min":800,"max":875,"count":2},"25":{"max":200,"count":1}}}}}}`), 0o644)
	d, err := Load(path, "abc")
	if err != nil {
		t.Fatal(err)
	}
	q := d.Quests[92456]
	if q == nil || d.LevelXP[2].Max != 900 {
		t.Fatalf("%+v", d)
	}
	c := q.ObjectiveCells()
	if len(c) != 2 || c[1].UIMap != 1424 || c[1].X != 24.5 || c[1].Y != 7.5 {
		t.Fatalf("cells %+v", c)
	}
	if p, ok := Best(q.Pickups); !ok || p.ID != 2 {
		t.Fatalf("pickup %+v", p)
	}
	if q.MaxXP(22) != 875 {
		t.Fatalf("xp %d", q.MaxXP(22))
	}
}
