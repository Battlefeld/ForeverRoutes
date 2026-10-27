package guides

import "testing"

func TestRXPObjectiveActionsRetainCommentsAndTargets(t *testing.T) {
	facts := ParseRXP(map[string]string{"g.lua": `RXPGuides.RegisterGuide([[
step
 .goto 1454,49.6,50.4
 .skipgossip 3216
 .complete 5727,1 --Speak with Neeru Fireblade
 .target Neeru Fireblade::3216
step
 .goto 1411,44.6,68.5
 .use 16114
 .complete 5441,1 --Wake lazy peons
 .target Lazy Peon::10556
]])`})
	if len(facts) != 2 {
		t.Fatalf("facts: %+v", facts)
	}
	if facts[0].Action != "talk" || len(facts[0].Targets) != 1 || facts[0].Targets[0] != 3216 {
		t.Fatalf("conversation: %+v", facts[0])
	}
	if facts[1].Action != "use" && facts[1].Action != "interact" {
		t.Fatalf("item use: %+v", facts[1])
	}
}

func TestPolestarUseAndCompleteKeepInlinePlaces(t *testing.T) {
	facts := ParsePolestar(map[string]string{"g.guide": `zone Mulgore
step Test
 use 278078 quest 97485.2 @53.3,87.9
 complete 92708 @40.0,50.0
`})
	if len(facts) != 2 {
		t.Fatalf("facts: %+v", facts)
	}
	if facts[0].Action != "use" || facts[0].Item != 0 || !facts[0].Complete || facts[0].Places[0].X != 53.3 {
		t.Fatalf("use: %+v", facts[0])
	}
	if !facts[1].Complete || len(facts[1].Places) != 1 || facts[1].Places[0].X != 40 {
		t.Fatalf("complete lost coordinates: %+v", facts[1])
	}
}
