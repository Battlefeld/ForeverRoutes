package build

import (
	"encoding/json"
	"sort"
)

// ChainLink is a quest ForeverDB's players saw come before another (chain.previous of a ForeverDB quest row).
type ChainLink struct {
	Pre      int    `json:"questId"`
	Relation string `json:"relation"` // Complete first, Chain link, Offered after, Next quest, Quest line
}

// ForeverChains reads chain.previous from ForeverDB's quest rows: quest -> the quests before it.
func ForeverChains(rows []json.RawMessage) map[int][]ChainLink {
	out := map[int][]ChainLink{}
	for _, raw := range rows {
		var r struct {
			ID    int `json:"id"`
			Chain *struct {
				Previous []ChainLink `json:"previous"`
			} `json:"chain"`
		}
		if json.Unmarshal(raw, &r) != nil || r.Chain == nil || len(r.Chain.Previous) == 0 {
			continue
		}
		out[r.ID] = r.Chain.Previous
	}
	return out
}

// foreverChains gives quests without any prerequisite in the other sources the ones ForeverDB's players saw before
// them: "Complete first" and "Chain link" as required (one of them when there are several: they are often the
// race or faction variants of one step), "Offered after" as one of them. A route then takes the chain first instead
// of sending the player to a giver who does not offer the quest yet.
func (b *builder) foreverChains() {
	if len(b.in.ForeverChains) == 0 {
		return
	}
	known := map[int]bool{}
	for _, q := range b.out {
		known[q.ID] = true
	}
	before := map[int]map[int]bool{} // what each quest already needs, to keep the chains free of loops
	for _, q := range b.out {
		before[q.ID] = map[int]bool{}
		for _, p := range append(append([]int(nil), q.PreAll...), q.PreAny...) {
			before[q.ID][p] = true
		}
	}
	var needs func(a, b int, depth int) bool // a needs b (directly or through its chain)
	needs = func(a, c int, depth int) bool {
		if depth > 12 {
			return false
		}
		for p := range before[a] {
			if p == c || needs(p, c, depth+1) {
				return true
			}
		}
		return false
	}
	for i := range b.out {
		q := &b.out[i]
		if len(q.PreAll)+len(q.PreAny) > 0 {
			continue
		}
		var strict, offered []int
		for _, l := range b.in.ForeverChains[q.ID] {
			if !known[l.Pre] || l.Pre == q.ID || needs(l.Pre, q.ID, 0) {
				continue
			}
			switch l.Relation {
			case "Complete first", "Chain link":
				strict = append(strict, l.Pre)
			case "Offered after":
				offered = append(offered, l.Pre)
			}
		}
		sort.Ints(strict)
		sort.Ints(offered)
		switch {
		case len(strict) == 1:
			q.PreAll = strict
		case len(strict) > 1:
			q.PreAny = strict
		case len(offered) > 0:
			q.PreAny = offered
		default:
			continue
		}
		for _, p := range append(append([]int(nil), q.PreAll...), q.PreAny...) {
			before[q.ID][p] = true
		}
		q.Sources["prerequisites"] = "foreverdb_chain"
		b.stats["prerequisites_from_foreverdb_chains"]++
	}
}
