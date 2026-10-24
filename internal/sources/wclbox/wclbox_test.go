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

func TestParseQuest(t *testing.T) {
	page := []byte(`<h1>finding warmth</h1><div class="qd-paper"><section><h2> Objectives </h2><p>Collect 14 firewood.</p>` +
		`<p class="qd-provided"><span> Provided: </span><a class="qd-item" href="/en/wupin/254545"><span class="qd-icon"></span><span class="qd-item-name q1">Well Water Sample Kit</span></a></p>` +
		`<ul class="qd-need"><li><a class="qd-item" href="/en/wupin/286339"><span class="qd-icon"><img src="x.jpg"/></span><span class="qd-item-name q1">Mostly Dry Firewood</span></a><span class="qd-from"><span>fallen log</span></span></li>` +
		`<li><a class="qd-item" href="/en/wupin/258771"><span class="qd-item-name q1">Ear</span></a><span class="qd-from"><span>Orakis Guardian, Aurakis Summoner</span><span>and 9 more</span></span></li></ul></section></div>` +
		`<img src="/jiemian/renwu/ditu/1426.webp"/><span class="qd-dot" style="left:49.8%;top:51.6%"></span><span class="qd-dot" style="left:50.8%;top:49.9%"></span>`)
	q, err := ParseQuest(99159, page)
	if err != nil {
		t.Fatal(err)
	}
	if len(q.Items) != 3 || !q.Items[0].Provided || q.Items[0].ID != 254545 {
		t.Fatalf("items: %+v", q.Items)
	}
	if f := q.Items[1]; f.ID != 286339 || len(f.From) != 1 || f.From[0] != "fallen log" {
		t.Fatalf("firewood: %+v", f)
	}
	if e := q.Items[2]; len(e.From) != 2 || e.From[1] != "Aurakis Summoner" {
		t.Fatalf("ear: %+v", e)
	}
	if q.UIMap != 1426 || len(q.Where) != 2 || q.Where[1].X != 50.8 || q.Where[1].Y != 49.9 {
		t.Fatalf("map: %+v", q)
	}
	if _, err := ParseQuest(490, []byte("<h1>We Could Not Find This Page</h1>")); !IsNotFound(err) {
		t.Fatalf("404 page: %v", err)
	}
}
