package blocks

import (
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

func list() *List {
	return &List{Quests: map[int]*Quest{
		1: {ID: 1, Source: "forever", Giver: true, Players: 3, Before: []Link{{Quest: 10, Kind: Optional}, {Quest: 11, Kind: Preferred, Seen: true}}},
		2: {ID: 2, Source: "forever", Before: []Link{{Quest: 10, Kind: Required}, {Quest: 11, Kind: Optional}}},
		3: {ID: 3, Source: "classic", Before: []Link{{Quest: 10, Kind: Required}}},
		4: {ID: 4, Source: "classic", Before: []Link{{Quest: 10, Kind: Optional}}},
	}}
}

func TestApplyKeepsOnlyRequired(t *testing.T) {
	chain := map[string]string{"prerequisites": "foreverdb_chain"}
	d := &dataset.Dataset{Quests: []dataset.Quest{
		{ID: 1, PreAll: []int{10}, Sources: chain},               // a chain link alone: free
		{ID: 2, PreAny: []int{10, 11}, Sources: chain},           // required and a chain link: the required one
		{ID: 3, PreAll: []int{10}, Sources: chain},               // required: as it was
		{ID: 4, PreAll: []int{10}, Sources: map[string]string{}}, // another source's prerequisite: untouched
		{ID: 5, PreAll: []int{10}, Sources: chain},               // not in ForeverDB's list: untouched
	}}
	changes, preferred := list().Apply(d)
	if len(changes) != 2 || changes[0].Quest != 1 || changes[1].Quest != 2 {
		t.Fatalf("changes %+v", changes)
	}
	q := d.Quests
	if len(q[0].PreAll)+len(q[0].PreAny) != 0 {
		t.Errorf("quest 1 keeps %v %v", q[0].PreAll, q[0].PreAny)
	}
	if len(q[1].PreAll) != 1 || q[1].PreAll[0] != 10 || len(q[1].PreAny) != 0 {
		t.Errorf("quest 2 has %v %v, want 10 required", q[1].PreAll, q[1].PreAny)
	}
	for _, i := range []int{2, 3, 4} {
		if len(q[i].PreAll) != 1 || q[i].PreAll[0] != 10 {
			t.Errorf("quest %d changed: %v", q[i].ID, q[i].PreAll)
		}
	}
	if len(preferred) != 0 {
		t.Errorf("preferred %v: quest 11 was no prerequisite of quest 1 in the dataset", preferred)
	}
}

func TestApplyReturnsPreferred(t *testing.T) {
	d := &dataset.Dataset{Quests: []dataset.Quest{{ID: 1, PreAny: []int{11}, Sources: map[string]string{"prerequisites": "foreverdb_chain"}}}}
	_, preferred := list().Apply(d)
	if len(d.Quests[0].PreAny) != 0 || len(preferred[1]) != 1 || preferred[1][0] != 11 {
		t.Errorf("quest 1: prerequisites %v, preferred %v", d.Quests[0].PreAny, preferred)
	}
}

func TestConfirmed(t *testing.T) {
	l := list()
	for id, want := range map[int]string{1: Seen, 2: Forever, 3: Classic, 99: Unknown} {
		if got := l.Quests[id].Confirmed(); got != want {
			t.Errorf("quest %d: %s, want %s", id, got, want)
		}
	}
}

func TestLoadShippedList(t *testing.T) {
	l, err := Load("../data/audits/foreverdb-full.json.gz")
	if err != nil {
		t.Skip("no ForeverDB list")
	}
	if len(l.Quests) < 3000 {
		t.Fatalf("%d quests", len(l.Quests))
	}
	// Plainstrider Menace: Sergra Darkthorn only leads to it
	q := l.Quests[844]
	if q == nil {
		t.Fatal("quest 844 missing")
	}
	for _, b := range q.Before {
		if b.Quest == 860 && b.Kind != Optional {
			t.Errorf("860 before 844 is %s, want optional", b.Kind)
		}
	}
}

func TestShareableShipped(t *testing.T) {
	s, err := Shareable("../data/audits/quest-flags-classic.json")
	if err != nil {
		t.Skip("no quest flags")
	}
	// Ragefire Chasm: Testing an Enemy's Strength can be shared, Returning the Lost Satchel (given inside) cannot
	if !s[5723] || s[5724] {
		t.Errorf("5723 shareable %v (want true), 5724 %v (want false)", s[5723], s[5724])
	}
}
