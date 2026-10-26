package route

import "testing"

func TestWithOptions(t *testing.T) {
	a := func(q int) GuideEvent { return GuideEvent{Op: "accept", Quest: q} }
	tn := func(q int) GuideEvent { return GuideEvent{Op: "turnin", Quest: q} }
	base := []GuideEvent{a(1), tn(1), a(2), tn(2)}
	opt := []GuideEvent{a(1), a(9), tn(1), a(2), tn(9), tn(2)}
	got := WithOptions(base, [][]GuideEvent{opt}, []map[int]bool{{9: true}})
	want := []GuideEvent{a(1), a(9), tn(1), a(2), tn(9), tn(2)}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v", got)
		}
	}
}
