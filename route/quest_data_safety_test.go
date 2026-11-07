package route

import (
	"testing"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

func TestOrcShamanCannotPlanCholarukAsPickup(t *testing.T) {
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWorld(d, Character{Race: "Orc", Class: "Shaman"}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{97003, 97005} {
		var found bool
		for _, q := range d.Quests {
			if q.ID != id {
				continue
			}
			found = true
			if q.Type != "elite" {
				t.Fatalf("quest %d lost elite classification", id)
			}
			if len(q.Objectives) != 1 || len(q.Objectives[0].Drops) != 1 || q.Objectives[0].Drops[0].NPC != 267307 {
				t.Fatalf("quest %d lost its actual head source", id)
			}
		}
		if !found {
			t.Fatalf("quest %d disappeared from dataset", id)
		}
		if w.ByID[id] != nil {
			t.Fatalf("quest %d remains in solo Shaman planning", id)
		}
	}
	if n := w.NPC[267307]; n == nil || n.Rank != 1 || n.MaxLevel != 20 {
		t.Fatal("Chol'aruk's elite level metadata is missing")
	}
}

func TestAllItemObjectivesRequireAnAcquisitionSource(t *testing.T) {
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range d.Quests {
		if q.Unsupported != "" {
			continue
		}
		for _, o := range q.Objectives {
			if o.Kind == "item" && !o.Inside && len(o.Drops) == 0 && len(o.Vendors) == 0 && !((o.Ground || o.UseItem > 0) && len(o.Places) > 0) {
				t.Errorf("quest %d item %d is plannable without acquisition evidence", q.ID, o.ID)
			}
		}
	}
}

func TestProvidedContainerInteractionIsKept(t *testing.T) {
	d, err := dataset.Load("../data/routes-data.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range d.Quests {
		if q.ID == 921 {
			if q.Unsupported != "" || len(q.Objectives) != 1 || q.Objectives[0].UseItem != 5185 {
				t.Fatalf("Crown of the Earth lost its provided phial interaction: %+v", q)
			}
			return
		}
	}
	t.Fatal("container quest missing")
}
