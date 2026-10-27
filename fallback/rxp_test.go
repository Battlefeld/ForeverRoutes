package fallback

import (
	"github.com/Battlefeld/ForeverRoutes/route"
	"testing"
)

func TestShippedFallbackCoversRoster(t *testing.T) {
	c, err := Load("../data/fallbacks/rxp.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Routes) != 56 {
		t.Fatalf("%d routes", len(c.Routes))
	}
	for _, ch := range route.SupportedCharacters() {
		r, err := c.For(ch)
		if err != nil {
			t.Fatal(err)
		}
		if r.Chapters[0] == "" {
			t.Fatal(ch)
		}
		if ch.Class == "Mage" {
			for _, name := range r.Chapters {
				if len(name) >= 3 && name[len(name)-3:] == "AoE" {
					t.Fatal("AoE substituted for normal guide")
				}
			}
		}
	}
	a, _ := c.For(route.Character{Race: "Skyborne", Class: "Hunter", Faction: "Alliance"})
	h, _ := c.For(route.Character{Race: "Skyborne", Class: "Hunter", Faction: "Horde"})
	if a.Key == h.Key || a.Chapters[len(a.Chapters)-1] == h.Chapters[len(h.Chapters)-1] {
		t.Fatal("Skyborne faction chains mixed")
	}
}
