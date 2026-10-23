package warcraftdb

import "testing"

const sample = `{"data":{"name":"The Cirrusfly Queen","level":3,"min_level":"2","starts":"<a href=\"/data/npc/251368\" data-tooltip=\"npc:251368\">Elatrell</a>",
"ends":"<a data-tooltip=\"npc:251368\">Elatrell</a>","rewards":{"xp":315},"objectives":[{"description":"Cirrusfly Queen","amount":1,"type":0,"object_id":251404}]},
"extra":{"quest_map":{"starts":[{"ui_map_id":2521,"u":0.434056,"v":0.248468,"label":"Elatrell"}],"turn_ins":[{"ui_map_id":"2521","u":0.43,"v":0.24}]}}}`

func TestParse(t *testing.T) {
	q, err := Parse(92463, []byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if q.Name != "The Cirrusfly Queen" || q.Level != 3 || q.MinLevel != 2 || q.XP != 315 {
		t.Errorf("fields %+v", q)
	}
	if len(q.Objectives) != 1 || q.Objectives[0] != (Objective{Type: 0, ID: 251404, Amount: 1, Description: "Cirrusfly Queen"}) {
		t.Errorf("objectives %+v", q.Objectives)
	}
	if len(q.StartNPC) != 1 || q.StartNPC[0] != 251368 || len(q.EndNPC) != 1 {
		t.Errorf("npcs %v %v", q.StartNPC, q.EndNPC)
	}
	if len(q.Starts) != 1 || q.Starts[0].UIMap != 2521 || q.Ends[0].UIMap != 2521 {
		t.Errorf("places %+v %+v", q.Starts, q.Ends)
	}
}
