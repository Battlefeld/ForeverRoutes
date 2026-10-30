package main

import (
	"encoding/json"
	"github.com/Battlefeld/ForeverRoutes/fallback"
	"github.com/Battlefeld/ForeverRoutes/route"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNextUsesRXPWhenOptimizedRouteIsUnavailable(t *testing.T) {
	dir := t.TempDir()
	w, err := loadWorld("../../data/routes-data.json.gz", "Human", "Mage", "", "")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(liveRequest{Race: "Human", Class: "Mage", Live: route.Live{Level: 1, Pos: &w.Start}})
	live := filepath.Join(dir, "live.json")
	if err = os.WriteFile(live, raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []bool{false, true} {
		if invalid {
			if err = os.WriteFile(filepath.Join(dir, "human-mage.json"), []byte(`{"character":{"race":"Orc","class":"Warlock"},"events":[{"op":"level","level":2}]}`), 0600); err != nil {
				t.Fatal(err)
			}
		}
		out := filepath.Join(dir, "next.json")
		err = cmdNext([]string{"-data", "../../data/routes-data.json.gz", "-live", live, "-routes", dir, "-fallbacks", "../../data/fallbacks/rxp.json.gz", "-horizon", "30", "-out", out})
		if err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		var reply nextReply
		if err = json.Unmarshal(b, &reply); err != nil {
			t.Fatal(err)
		}
		if reply.Mode != "rxp-fallback" || len(reply.Steps) == 0 || reply.Seconds <= 0 {
			t.Fatalf("fallback was not used: %+v", reply)
		}
	}
}

func TestTimelineRejectsAnotherCharactersSeed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wrong.json")
	if err := os.WriteFile(path, []byte(`{"character":{"race":"Human","class":"Mage"},"events":[{"op":"level","level":2}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	err := cmdTimeline([]string{"-data", "../../data/routes-data.json.gz", "-seeds", path, "-budget", "0s"})
	if err == nil || !strings.Contains(err.Error(), "another character") {
		t.Fatalf("wrong seed was not rejected: %v", err)
	}
}

func TestFallbackExportRetainsOriginalInstructions(t *testing.T) {
	c, err := fallback.Load("../../data/fallbacks/rxp.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	writeFallbackCatalog(&b, c)
	s := b.String()
	if strings.Count(s, `kind="rxp-source"`) != 56 || !strings.Contains(s, "function ForeverRoutesData.SelectRoute") || !strings.Contains(s, "missing faction") || !strings.Contains(s, ".accept 92460") {
		t.Fatal("incomplete fallback export")
	}
}

func TestCheckRoutesRejectsEmptyAndMalformedInputs(t *testing.T) {
	dir := t.TempDir()
	if err := cmdCheckRoutes([]string{"-routes", dir}); err == nil {
		t.Fatal("empty release accepted")
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{broken`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := cmdCheckRoutes([]string{"-routes", dir}); err == nil {
		t.Fatal("malformed route silently omitted")
	}
}
