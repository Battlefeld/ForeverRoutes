// Package norgannon reads NorgannonDB's item table (github.com/zmarotrix/NorgannonDB, GPL-3.0): an open,
// crowd-sourced WoW Forever database built from QuestieDB, pfQuest, Forever's own game tables and players' collected
// data. Of it ForeverRoutes takes where an item comes from: the creatures and objects it drops from (with the chance in
// percent, 0 for unknown) and the quests that reward it. The file is read as data only; no Lua is run.
package norgannon

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/internal/lua"
)

// Repository is the upstream project.
const Repository = "zmarotrix/NorgannonDB"

// ItemsFile is the item table in the repository.
const ItemsFile = "data/items.lua"

// Drop is a creature or object an item drops from; Chance in percent, 0 when unknown.
type Drop struct {
	ID     int
	Chance float64
}

// Item is where one item comes from.
type Item struct {
	ID       int
	Name     string
	NPCs     []Drop
	Objects  []Drop
	RewardOf []int // quests that reward it
}

// Data is the item table of one NorgannonDB commit.
type Data struct {
	Revision string
	Items    map[int]*Item
}

// record: R(id,"name") or R(id,"name",[[fields]]), one per line.
var record = regexp.MustCompile(`^R\((\d+),("(?:[^"\\]|\\.)*")(?:,\[\[(.*)\]\])?\)\s*$`)

// Load reads the item table from a file.
func Load(path, revision string) (*Data, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(b), revision)
}

// Parse reads the item table.
func Parse(raw, revision string) (*Data, error) {
	d := &Data{Revision: revision, Items: map[int]*Item{}}
	for _, line := range strings.Split(raw, "\n") {
		m := record.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil {
			continue
		}
		id, _ := strconv.Atoi(m[1])
		name, _ := lua.Parse(m[2])
		it := &Item{ID: id}
		it.Name, _ = name.(string)
		if m[3] != "" && (strings.Contains(m[3], "drops=") || strings.Contains(m[3], "rewardOf=")) {
			v, err := lua.Parse("{" + m[3] + "}")
			if err != nil {
				return nil, fmt.Errorf("norgannon item %d: %w", id, err)
			}
			t, _ := v.(lua.Table)
			if drops, ok := t["drops"].(lua.Table); ok {
				it.NPCs = drops2(drops["npcs"])
				it.Objects = drops2(drops["objects"])
			}
			it.RewardOf = lua.IDs(t["rewardOf"])
		}
		if len(it.NPCs)+len(it.Objects)+len(it.RewardOf) > 0 {
			d.Items[id] = it
		}
	}
	return d, nil
}

// drops2 reads {[id]=chance}, best chance first, ids ascending among equals.
func drops2(v any) []Drop {
	t, ok := v.(lua.Table)
	if !ok {
		return nil
	}
	var out []Drop
	for _, k := range lua.SortedKeys(t) {
		id, ok := k.(int)
		if !ok || id <= 0 {
			continue
		}
		c := 0.0
		switch n := t[k].(type) {
		case int:
			c = float64(n)
		case float64:
			c = n
		}
		out = append(out, Drop{ID: id, Chance: c})
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Chance > out[j-1].Chance; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
