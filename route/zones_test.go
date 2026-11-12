package route

import (
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// The Mirage Raceway on the Shimmering Flats is Thousand Needles, though Gadgetzan (Tanaris) is nearer than Freewind.
func TestShimmeringFlatsIsThousandNeedles(t *testing.T) {
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Skip("dataset not available:", err)
	}
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Shaman"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []dataset.Point{{Map: 1, X: -6233, Y: -3856}, {Map: 1, X: -5568, Y: -4068}, {Map: 1, X: -6544, Y: -3882}} {
		if z := w.ZoneName(p); z != "Thousand Needles" {
			t.Errorf("%v: %s", p, z)
		}
	}
	if z := w.ZoneName(dataset.Point{Map: 1, X: -7150, Y: -3780}); z != "Tanaris" {
		t.Errorf("Gadgetzan: %s", z)
	}
}

// The player's choice of dungeons: only those exist for the route, and a chosen one is run once it is due.
func TestChosenDungeonsOnly(t *testing.T) {
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Skip("dataset not available:", err)
	}
	w, err := NewWorld(d, Character{Race: "Tauren", Class: "Hunter"}, Options{Dungeons: true, DungeonKeys: []string{"wailing-caverns"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Dungeons) != 1 || w.Dungeons[0].Key != "wailing-caverns" {
		t.Fatalf("dungeons %d", len(w.Dungeons))
	}
	for _, q := range w.Quests {
		if q.Dungeon != nil && q.Dungeon.Key != "wailing-caverns" {
			t.Fatalf("quest %d of a dungeon not chosen", q.ID)
		}
	}
	if due := w.DungeonDueLevel("wailing-caverns"); due != 18 {
		t.Errorf("due level %d", due)
	}
	s := w.NewState()
	if run, _ := w.RunChosenDungeon(s, "wailing-caverns"); run != nil {
		t.Error("run at level 1")
	}
	at18, _, err := w.StateFrom(Live{Level: 18, Pos: &s.Pos})
	if err != nil {
		t.Fatal(err)
	}
	run, err := w.RunChosenDungeon(at18, "wailing-caverns")
	if err != nil || run == nil {
		t.Fatalf("no run at 18: %v", err)
	}
	// the quests before a dungeon quest come first: Leaders of the Fang (914) is taken after its chain (Nara Wildmane,
	// 1490) is handed in, and before the run
	pos := map[string]int{}
	for i, st := range run.Steps {
		switch {
		case st.Op == "turnin" && st.Quest == 1490:
			pos["chain"] = i + 1
		case st.Op == "accept" && st.Quest == 914:
			pos["accept"] = i + 1
		case st.Op == "dungeon":
			pos["run"] = i + 1
		case st.Op == "turnin" && st.Quest == 914:
			pos["turnin"] = i + 1
		}
	}
	if pos["chain"] == 0 || !(pos["chain"] < pos["accept"] && pos["accept"] < pos["run"] && pos["run"] < pos["turnin"]) {
		t.Errorf("Leaders of the Fang and its chain: %v", pos)
	}
	after, err := w.StateAfterFrom(at18, run.Steps)
	if err != nil || !w.DungeonRun(after, "wailing-caverns") {
		t.Fatalf("the run is not behind the state: %v", err)
	}
	if all, _ := NewWorld(d, Character{Race: "Tauren", Class: "Hunter"}, Options{Dungeons: true}); len(all.Dungeons) < 10 {
		t.Errorf("no choice: %d dungeons", len(all.Dungeons))
	}
}
