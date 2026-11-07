package norgannon

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/internal/lua"
)

// CombatMetadata is sourced NPC difficulty, independent of item-drop records.
// Rank is a pointer so an explicit normal rank is distinct from missing data.
type CombatMetadata struct {
	MinLevel, MaxLevel int
	Rank               *int
}

func ParseNPCs(raw string) (map[int]CombatMetadata, error) {
	out := map[int]CombatMetadata{}
	for _, line := range strings.Split(raw, "\n") {
		m := record.FindStringSubmatch(strings.TrimRight(line, "\r"))
		if m == nil || (!strings.Contains(m[3], "rank=") && !strings.Contains(m[3], "minLevel=") && !strings.Contains(m[3], "maxLevel=")) {
			continue
		}
		id, _ := strconv.Atoi(m[1])
		v, err := lua.Parse("{" + m[3] + "}")
		if err != nil {
			return nil, fmt.Errorf("norgannon npc %d: %w", id, err)
		}
		t, _ := v.(lua.Table)
		n := CombatMetadata{}
		if v, ok := t["minLevel"].(int); ok {
			n.MinLevel = v
		}
		if v, ok := t["maxLevel"].(int); ok {
			n.MaxLevel = v
		}
		if v, ok := t["rank"].(int); ok {
			n.Rank = &v
		}
		out[id] = n
	}
	return out, nil
}
