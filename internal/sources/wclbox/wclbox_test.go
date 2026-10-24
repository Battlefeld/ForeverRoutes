package wclbox

import "testing"

func TestParse(t *testing.T) {
	page := `<html><head><title>Bruuz&#39;s Dorsal Fin (Item 253710) - WCLBox · WoW Forever Database</title></head><body>
	<span>Beta build 1.60.1.70532</span>
	<p class="wp-drops"> Source:  <b>Brouse</b>, The Barrens</p><div>Classic</div></body></html>`
	it, build, err := Parse(253710, []byte(page))
	if err != nil {
		t.Fatal(err)
	}
	if it.Name != "Bruuz's Dorsal Fin" || len(it.Sources) != 1 || it.Sources[0] != (Source{"Brouse", "The Barrens"}) || build != "1.60.1.70532" {
		t.Errorf("%+v %q", it, build)
	}
}

func TestParseSeveralAndNone(t *testing.T) {
	two := `<title>Thing (Item 5) - x</title><p class="wp-drops"> Source:  <b>Mining</b>, Crafting <b>Brouse</b>, The Barrens</p>`
	it, _, err := Parse(5, []byte(two))
	if err != nil || len(it.Sources) != 2 || it.Sources[1].Name != "Brouse" {
		t.Errorf("%+v %v", it, err)
	}
	none := `<title>Highly Toxic Strain (Item 275443) - x</title><p>no source</p>`
	it, _, err = Parse(275443, []byte(none))
	if err != nil || len(it.Sources) != 0 || it.Name != "Highly Toxic Strain" {
		t.Errorf("%+v %v", it, err)
	}
	if _, _, err := Parse(1, []byte("<html>blocked</html>")); err == nil {
		t.Error("a page without a title must be an error")
	}
}
