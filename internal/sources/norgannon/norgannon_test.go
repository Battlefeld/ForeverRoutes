package norgannon

import "testing"

func TestParse(t *testing.T) {
	raw := `-- NorgannonDB 1.4
local R = NorgannonDB:_PackedLoader("items")
R(17,"Martin Fury")
R(247824,"Thunder Applejack",[[level=1,relatedQuests={91738},requiredLevel=1,rewardOf={91736},subClass=5]])
R(5220,"Gnarlpine Fang",[[class=12,drops={npcs={[2006]=0,[2007]=12.5}},level=1]])
R(5646,"Vial of Blessed Water",[[class=12,drops={objects={[138498]=0}},flags=64,level=1]])
`
	d, err := Parse(raw, "x")
	if err != nil {
		t.Fatal(err)
	}
	if d.Items[17] != nil || len(d.Items) != 3 {
		t.Fatalf("items: %d", len(d.Items))
	}
	if it := d.Items[247824]; len(it.RewardOf) != 1 || it.RewardOf[0] != 91736 {
		t.Fatalf("reward: %+v", it)
	}
	if it := d.Items[5220]; len(it.NPCs) != 2 || it.NPCs[0].ID != 2007 || it.NPCs[0].Chance != 12.5 {
		t.Fatalf("drops: %+v", it.NPCs)
	}
	if it := d.Items[5646]; len(it.Objects) != 1 || it.Objects[0].ID != 138498 {
		t.Fatalf("objects: %+v", it.Objects)
	}
}
