package route

import (
	"encoding/json"
	"testing"
)

func TestLiveReadsEmptyLuaTablesAsEmptyLists(t *testing.T) {
	var l Live
	in := `{"level":2,"xp":10,"pos":{"m":1,"x":-600,"y":-4200},"completed":{},"active":{},"flightMasters":{}}`
	if err := json.Unmarshal([]byte(in), &l); err != nil {
		t.Fatal(err)
	}
	if l.Level != 2 || len(l.Active) != 0 || len(l.Completed) != 0 || l.Pos == nil {
		t.Fatalf("%+v", l)
	}
	in = `{"level":3,"xp":0,"pos":{"m":1,"x":0,"y":0},"completed":[4641],"active":[{"id":788,"objectives":{}},{"id":789,"ready":true,"objectives":[{"type":"monster","have":2,"need":10}]}]}`
	if err := json.Unmarshal([]byte(in), &l); err != nil {
		t.Fatal(err)
	}
	if len(l.Active) != 2 || len(l.Active[0].Objectives) != 0 || l.Active[1].Objectives[0].Have != 2 || !l.Active[1].Ready || l.Completed[0] != 4641 {
		t.Fatalf("%+v", l)
	}
}
