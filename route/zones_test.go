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
