package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/route"
)

// A personal route starts from the character's own quest state: a quest it has in its log is worked off first, the
// quests it has finished are not planned again (Battle of Hillsbrad part 2 after part 1, not part 3).
func TestExtendFromLiveState(t *testing.T) {
	if testing.Short() {
		t.Skip("plans 25-40")
	}
	src := "../../data/routes-ext/orc-shaman-dungeons.json"
	r, err := readRoute(src)
	if err != nil {
		t.Skip("no continued route")
	}
	var done []int
	for _, e := range r.Events {
		if e.Op == "turnin" {
			done = append(done, e.Quest)
			if e.Quest == 527 {
				break
			}
		}
	}
	dir := t.TempDir()
	live := map[string]any{"live": route.Live{Level: 25, XP: 1000, Pos: &dataset.Point{Map: 0, X: -30, Y: -900}, Completed: done,
		Active: []route.LiveQuest{{ID: 528, Objectives: []route.LiveObjective{{Type: "monster", Have: 3, Need: 15}}}}}}
	b, _ := json.Marshal(live)
	in, out := filepath.Join(dir, "live.json"), filepath.Join(dir, "route.json")
	if err := os.WriteFile(in, b, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := cmdExtend([]string{"-data", "../../data/routes-data.json.gz", "-profiles", "../../data/profiles", "-route", src, "-live", in,
		"-target", "40", "-stages", "3s", "-budget", "0", "-out", out}); err != nil {
		t.Fatal(err)
	}
	got, err := readRoute(out)
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		Start *route.Live `json:"start"`
	}
	b, _ = os.ReadFile(out)
	_ = json.Unmarshal(b, &raw)
	if raw.Start == nil || raw.Start.Level != 25 {
		t.Fatal("the route does not name the state it starts from")
	}
	first := -1
	for i, e := range got.Events {
		if e.Quest == 527 {
			t.Fatalf("finished quest 527 planned again at event %d", i)
		}
		if e.Quest == 528 && first < 0 {
			first = i
		}
	}
	// early, not necessarily first: the quests of the town it stands in are taken on the way (509, 513, 501, ...)
	if first < 0 || first > 10 {
		t.Fatalf("the quest in the log (528) is not worked off first: at %d", first)
	}
}
