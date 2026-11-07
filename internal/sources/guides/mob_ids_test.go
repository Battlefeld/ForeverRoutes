package guides

import "testing"

func TestRXPExplicitMobIdentifiersArePreserved(t *testing.T) {
	files := map[string]string{"test.lua": "RegisterGuide([[\nstep\n.goto 2521,64.38,63.586\n.complete 97244,1\n.mob Skypriest Faladiel::268602\nstep\n.mob +Kuramaa::268605\n.complete 97245,1\n]])"}
	facts := ParseRXP(files)
	if len(facts) != 2 {
		t.Fatalf("facts: %+v", facts)
	}
	for i, want := range []int{268602, 268605} {
		if len(facts[i].NPCs) != 1 || facts[i].NPCs[0] != want {
			t.Fatalf("explicit mob ID lost: %+v", facts[i])
		}
		if len(facts[i].Mobs) != 1 || facts[i].Ground {
			t.Fatalf("combat became pickup: %+v", facts[i])
		}
	}
}
