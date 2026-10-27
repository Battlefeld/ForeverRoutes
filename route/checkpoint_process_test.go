package route

import (
	"encoding/json"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCheckpointCrossProcessKeepsTheFasterRoute(t *testing.T) {
	routes := [][]GuideEvent{
		{{Op: "accept", Quest: 1}, {Op: "complete", Quest: 1}, {Op: "turnin", Quest: 1}},
		{{Op: "accept", Quest: 2}, {Op: "complete", Quest: 2}, {Op: "turnin", Quest: 2}},
	}
	w := tinyWorld(t)
	key, err := w.optimizationKey(OptimizeOptions{Target: 3}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if path := os.Getenv("FR_CP_CHILD_PATH"); path != "" {
		pick := 0
		if os.Getenv("FR_CP_CHILD_PICK") == "1" {
			pick = 1
		}
		for i := 0; i < 20; i++ {
			if err := w.saveCheckpoint(path, key, routes[pick], w.Preparations(), w.WandDetour(), 3); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	path := filepath.Join(t.TempDir(), "best.json")
	commands := []*exec.Cmd{}
	for _, pick := range []string{"0", "1"} {
		c := exec.Command(os.Args[0], "-test.run=^TestCheckpointCrossProcessKeepsTheFasterRoute$", "-test.timeout=20s")
		c.Env = append(os.Environ(), "FR_CP_CHILD_PATH="+path, "FR_CP_CHILD_PICK="+pick)
		if err := c.Start(); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, c)
	}
	for _, c := range commands {
		if err := c.Wait(); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cp OptimizationCheckpoint
	if err = json.Unmarshal(raw, &cp); err != nil {
		t.Fatal(err)
	}
	a, err := w.FollowGuide(routes[0], 3)
	if err != nil {
		t.Fatal(err)
	}
	b, err := w.FollowGuide(routes[1], 3)
	if err != nil {
		t.Fatal(err)
	}
	if cp.Seconds != math.Min(a.Seconds, b.Seconds) {
		t.Fatalf("slower writer overwrote incumbent: %.1f", cp.Seconds)
	}
	if _, err := os.Stat(path + ".lock"); !os.IsNotExist(err) {
		t.Fatal("writer left a lock behind")
	}
}
