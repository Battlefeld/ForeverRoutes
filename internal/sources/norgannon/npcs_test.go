package norgannon

import "testing"

func TestNPCCombatMetadata(t *testing.T) {
	d, err := ParseNPCs("R(1,\"Normal\",[[minLevel=17,maxLevel=20,rank=0]])\nR(2,\"Elite\",[[rank=1]])\nR(3,\"Unknown\")\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(d) != 2 || d[1].Rank == nil || *d[1].Rank != 0 || d[1].MaxLevel != 20 || d[2].Rank == nil || *d[2].Rank != 1 {
		t.Fatalf("combat metadata lost: %+v", d)
	}
	if _, err = ParseNPCs("R(4,\"Bad\",[[rank=os.execute(\"bad\")]])"); err == nil {
		t.Fatal("executable Lua accepted as NPC data")
	}
}
