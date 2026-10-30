package clientquests

import (
	"github.com/Battlefeld/ForeverRoutes/dataset"
	"os"
	"path/filepath"
	"testing"
)

func TestLiveClientNamesDoNotChangeQuestRewardsOrRequirements(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client-quests.json")
	raw := `{"schema":1,"captures":{"1.60.1.70685/3/enUS":{"build":"1.60.1.70685","region":3,"locale":"enUS","quests":{"971":{"id":971,"title":"Live title","seenAt":123,"xp":999999,"minLevel":99}}}}}`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	d := &dataset.Dataset{Quests: []dataset.Quest{{ID: 971, Name: "Old", XP: 6550, MinLevel: 10}}}
	n, err := Apply(path, d)
	if err != nil || n != 1 {
		t.Fatalf("%d %v", n, err)
	}
	q := d.Quests[0]
	if q.Name != "Live title" || q.XP != 6550 || q.MinLevel != 10 || q.Sources["name"] == "" {
		t.Fatalf("unsafe live overlay: %+v", q)
	}
}
func TestMalformedObservationDoesNotPartiallyApply(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(path, []byte(`{"schema":1,"captures":{"bad":{"build":"12.0.0.123","region":3,"locale":"enUS","quests":{}}}}`), 0600)
	d := &dataset.Dataset{Quests: []dataset.Quest{{ID: 971, Name: "Old"}}}
	if _, err := Apply(path, d); err == nil || d.Quests[0].Name != "Old" {
		t.Fatal("invalid overlay changed dataset")
	}
}
