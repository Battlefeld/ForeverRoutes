package route

import (
	"bytes"
	"encoding/json"
)

// The addon writes its state from Lua, where an empty list and an empty table are the same thing: a character with
// no quest in the log, no finished quest or no known flight master sends {} where a list belongs. Those read as
// empty lists here instead of failing the whole plan.

func emptyObjectAsList(fields map[string]json.RawMessage, names ...string) {
	for _, n := range names {
		if v, ok := fields[n]; ok && bytes.Equal(bytes.TrimSpace(v), []byte("{}")) {
			fields[n] = json.RawMessage("[]")
		}
	}
}

func decodeLenient(data []byte, into any, lists ...string) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	emptyObjectAsList(fields, lists...)
	fixed, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	return json.Unmarshal(fixed, into)
}

// UnmarshalJSON reads Live, taking {} for an empty list.
func (l *Live) UnmarshalJSON(data []byte) error {
	type plain Live
	var p plain
	if err := decodeLenient(data, &p, "completed", "active", "flightMasters"); err != nil {
		return err
	}
	*l = Live(p)
	return nil
}

// UnmarshalJSON reads LiveQuest, taking {} for an empty list of objectives.
func (q *LiveQuest) UnmarshalJSON(data []byte) error {
	type plain LiveQuest
	var p plain
	if err := decodeLenient(data, &p, "objectives"); err != nil {
		return err
	}
	*q = LiveQuest(p)
	return nil
}
