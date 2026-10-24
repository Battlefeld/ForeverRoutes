package foreverchanges

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	p := `<meta name="description" content="Collect the Highly Toxic Strain from Witherfang in Ruins of Lordaeron for Theodore Griffs in Undercity. A level 22 dungeon quest for the Horde, from level 16." />`
	q, err := Parse(95216, []byte(p))
	if err != nil || q.Kind != "dungeon" || q.Level != 22 || q.Text != "Collect the Highly Toxic Strain from Witherfang in Ruins of Lordaeron for Theodore Griffs in Undercity." {
		t.Fatalf("%+v %v", q, err)
	}
	n, _ := Parse(1, []byte(`<meta name="description" content="Bring 10 Bronze Bars to Morhan. A level 30 quest for both factions." />`))
	if n.Kind != "normal" {
		t.Errorf("kind %q", n.Kind)
	}
	if _, err := Parse(1, []byte("<html></html>")); err == nil {
		t.Error("a page without a description must be an error")
	}
}

func TestSources(t *testing.T) {
	cases := []struct {
		text string
		want []Source
	}{
		{"Collect 6 Savannah Lion Hides from the Savannah Matriarchs and Savannah Patriarchs in the Barrens.", []Source{{"Savannah Matriarchs", "Barrens"}, {"Savannah Patriarchs", "Barrens"}}},
		{"Collect 5 Spiked Collars from Gnoll Thrashers for Deputy Feldon near Lakeshire.", []Source{{"Gnoll Thrashers", ""}}},
		{"Collect 20 feathers from the Shadowgale Shrieklings in Shadowgale Forest.", []Source{{"Shadowgale Shrieklings", "Shadowgale Forest"}}},
		{"Bring 10 Bronze Bars to Morhan Coppertongue.", nil},
	}
	for _, c := range cases {
		if got := Sources(c.text); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q: got %v, want %v", c.text, got, c.want)
		}
	}
}
