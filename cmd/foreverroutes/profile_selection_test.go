package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/route"
)

func TestLivePlanningDefaultsToClassProfileAndHonorsExplicitEstimate(t *testing.T) {
	dir := t.TempDir()
	w, err := loadWorld("../../data/routes-data.json.gz", "Human", "Mage", "", "")
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(liveRequest{Race: "Human", Class: "Mage", Live: route.Live{Level: 1, Pos: &w.Start}})
	if err != nil {
		t.Fatal(err)
	}
	live := filepath.Join(dir, "live.json")
	if err = os.WriteFile(live, raw, 0600); err != nil {
		t.Fatal(err)
	}
	for _, estimate := range []bool{false, true} {
		out := filepath.Join(dir, "next.json")
		args := []string{"-data", "../../data/routes-data.json.gz", "-live", live,
			"-routes", dir, "-fallbacks", "../../data/fallbacks/rxp.json.gz", "-horizon", "30", "-out", out}
		if estimate {
			args = append(args, "-profile", "")
		}
		if err = cmdNext(args); err != nil {
			t.Fatal(err)
		}
		raw, err = os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		var reply nextReply
		if err = json.Unmarshal(raw, &reply); err != nil {
			t.Fatal(err)
		}
		if reply.Mode != "rxp-fallback" || len(reply.Steps) == 0 || isEngineProfile(reply.Profile) == estimate {
			t.Fatalf("estimate=%v: unexpected planning model/mode: %s / %s", estimate, reply.Profile, reply.Mode)
		}
	}
}

func TestAutomaticCombatProfileCoversEverySupportedClass(t *testing.T) {
	seen := map[string]bool{}
	for _, character := range route.SupportedCharacters() {
		if seen[character.Class] {
			continue
		}
		seen[character.Class] = true
		t.Run(character.Class, func(t *testing.T) {
			w, err := loadWorld("../../data/routes-data.json.gz", character.Race, character.Class, character.Faction, automaticProfile)
			if err != nil {
				t.Fatal(err)
			}
			p, ok := w.Opt.Profile.(*route.TableProfile)
			if !ok || !p.Fits(character) {
				t.Fatalf("class-specific combat profile not selected for %s: %T", character.Class, w.Opt.Profile)
			}
			fight, rest := p.KillSeconds(character, 15, &dataset.NPC{HP: 300, MinLevel: 15, MaxLevel: 15, Dmg: 10})
			if fight <= 0 || rest <= 0 {
				t.Fatalf("invalid fight/recovery costs: %g/%g", fight, rest)
			}
		})
	}
	if len(seen) != 9 {
		t.Fatalf("tested %d classes, expected 9", len(seen))
	}
}

func TestAutomaticCombatProfileUsesDatasetDirectory(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "routes-data.json.gz")
	if err := os.Mkdir(filepath.Join(dir, "profiles"), 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "profiles", "shaman.json")
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	selected, err := resolveCombatProfile(data, "Shaman", automaticProfile)
	if err != nil || selected != path {
		t.Fatalf("got %q, %v; want dataset-relative %q", selected, err, path)
	}
	if selected, err = resolveCombatProfile(data, "Mage", automaticProfile); err != nil || selected != "" {
		t.Fatalf("missing profile must fall back to labelled estimate: %q, %v", selected, err)
	}
	if selected, err = resolveCombatProfile(data, "Shaman", ""); err != nil || selected != "" {
		t.Fatalf("explicit estimate was overridden: %q, %v", selected, err)
	}
	if selected, err = resolveCombatProfile(data, "Shaman", "custom.json"); err != nil || selected != "custom.json" {
		t.Fatalf("explicit profile was overridden: %q, %v", selected, err)
	}
	if err = os.Mkdir(filepath.Join(dir, "profiles", "mage.json"), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err = resolveCombatProfile(data, "Mage", automaticProfile); err == nil {
		t.Fatal("directory must not be treated as a missing combat profile")
	}
	if _, err = resolveCombatProfile(data, "Unknown", automaticProfile); err == nil {
		t.Fatal("unknown class must not select a profile")
	}
}

func TestAutomaticCombatProfileRejectsCorruptOrForeignClass(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "routes-data.json.gz")
	raw, err := os.ReadFile("../../data/routes-data.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(data, raw, 0644); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(filepath.Join(dir, "profiles"), 0755); err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(dir, "profiles", "shaman.json")
	if err = os.WriteFile(profile, []byte("not JSON"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = loadWorld(data, "Orc", "Shaman", "", automaticProfile); err == nil {
		t.Fatal("broken profile silently fell back to class-neutral estimates")
	}
	raw, err = os.ReadFile("../../data/profiles/mage.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(profile, raw, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err = loadWorld(data, "Orc", "Shaman", "", automaticProfile); err == nil || !strings.Contains(err.Error(), "profile, not Shaman") {
		t.Fatalf("foreign class profile accepted: %v", err)
	}
}
