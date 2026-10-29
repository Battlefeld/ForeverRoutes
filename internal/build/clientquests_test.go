package build

import (
	"github.com/Battlefeld/ForeverRoutes/dataset"
	"os"
	"path/filepath"
	"testing"
)

func TestClientCaptureOnlyImportsEnglishNamePreservingQuestRules(t *testing.T) {
	path := filepath.Join(t.TempDir(), "capture.json")
	raw := `{"schema":1,"captures":{"1.60.1.70685/3/enUS":{"build":"1.60.1.70685","region":3,"locale":"enUS","quests":{"971":{"id":971,"title":"Client title","seenAt":123,"xp":999999,"minLevel":99}}}}}`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	b := builder{out: []dataset.Quest{{ID: 971, Name: "Old", XP: 6550, MinLevel: 10}}, stats: map[string]int{}}
	if err := b.clientQuestNames(path); err != nil {
		t.Fatal(err)
	}
	q := b.out[0]
	if q.Name != "Client title" || q.XP != 6550 || q.MinLevel != 10 || q.Sources["name"] == "" {
		t.Fatalf("unsafe import: %+v", q)
	}
}
