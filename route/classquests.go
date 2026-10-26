package route

import (
	"fmt"
	"math/bits"
	"sort"
)

// Class quests a route does while levelling, by what their reward is worth for it. The quest named is the one that
// gives the reward; its prerequisites come with it. Chosen from what each reward does (cmangos quest rewards) and
// what Forever changed (foreverchanges.pro).
var (
	// classRequired: done in every route, also when they take longer.
	classRequired = map[int]string{
		1470: "Wichtel", 1485: "Wichtel", 1598: "Wichtel", 1599: "Wichtel",
		1471: "Leerwandler", 1504: "Leerwandler", 1689: "Leerwandler",
		6081: "Begleiter zähmen", 6086: "Begleiter zähmen", 6089: "Begleiter zähmen", 6103: "Begleiter zähmen",
		94050: "Begleiter zähmen", 94793: "Begleiter zähmen",
		6001: "Bärengestalt", 6002: "Bärengestalt", 94006: "Bärengestalt",
		1518: "Erdtotem", 1521: "Erdtotem", 94375: "Erdtotem", 92468: "Erdtotem",
		1527: "Feuertotem", 94468: "Feuertotem", 97257: "Feuertotem",
		1498: "Verteidigungshaltung", 1665: "Verteidigungshaltung", 1678: "Verteidigungshaltung",
		1683: "Verteidigungshaltung", 1819: "Verteidigungshaltung",
		2381: "Schlossknacken", 2282: "Schlossknacken",
	}
	// classRecommended: offered as options worth their time (a green item, a strong spell, good XP on the way).
	classRecommended = map[int]string{
		1667: "Waffe", 1682: "Waffe", 1693: "Waffe", 1822: "Waffe", 1503: "Waffe",
		2242: "Dolch", 2206: "Dolch", 2239: "Dolch", 1978: "Dolch", 1858: "Dolch",
		1861: "Stab", 1880: "Stab", 1884: "Stab", 1882: "Stab", 93797: "Gegenstand", 1941: "Robe", 1962: "Robe",
		5650: "Brustteil", 5648: "Brustteil", 5624: "Brustteil", 5625: "Brustteil", 5621: "Brustteil",
		5627: "Volkszauber", 5628: "Volkszauber", 5629: "Volkszauber", 5630: "Volkszauber", 5631: "Volkszauber",
		5632: "Volkszauber", 5633: "Volkszauber", 5634: "Volkszauber", 5635: "Volkszauber", 5636: "Volkszauber",
		5637: "Volkszauber", 5638: "Volkszauber", 5639: "Volkszauber", 5640: "Volkszauber", 5658: "Volkszauber",
		5660: "Volkszauber", 5661: "Volkszauber", 5662: "Volkszauber", 5663: "Volkszauber",
		94773: "Volkszauber", 94774: "Volkszauber", 94775: "Volkszauber", 94776: "Volkszauber", 94777: "Volkszauber",
		94778: "Volkszauber", 94779: "Volkszauber", 94817: "Volkszauber", 94821: "Volkszauber", 94822: "Volkszauber",
		94823: "Volkszauber", 94824: "Volkszauber", 94825: "Volkszauber", 94826: "Volkszauber",
		91285: "XP und Gegenstand", 91316: "XP und Gegenstand", 91317: "XP", 95803: "XP",
		94911: "unbekannt (neu)", 94912: "unbekannt (neu)", 94638: "unbekannt (neu)",
	}
)

// extraOptions: quest chains for every class a route offers as options (the quest named gives the reward).
var extraOptions = map[int]string{
	79976: "Gemütlicher Schlafsack", // Cozy Sleeping Bag, from 14: a 12-slot bag, 10,950 XP over the chain, +3 % XP resting
}

// ClassChain is a class quest with the quests before it, as a route does them.
type ClassChain struct {
	Goal   *Quest
	Quests []*Quest // in order, the goal last
	Reward string
	Tier   string // required, recommended, optional

	foreign bool // begins in another race's home: taken only when no other version is there
}

// classLetter: the class quests of the first two levels (the letter to the trainer, Forever's first class quest):
// done at the start, they cost next to nothing.
func classLetter(q *Quest) bool { return bits.OnesCount32(uint32(q.Classes)) == 1 && q.MinLevel <= 2 }

// ClassChains are the class quest chains this character can do below the target level, one per reward: of the
// versions of a quest the one that begins in its own home (start area or capital) or outside every race's home,
// nearest its start.
func (w *World) ClassChains(target int) []ClassChain {
	race := w.Char.Race
	if race == "Scourge" {
		race = "Undead"
	}
	ok := func(q *Quest) bool {
		onMap := q.StartPos.Map == 0 || q.StartPos.Map == 1 || q.StartPos.Map == w.Start.Map
		_, extra := extraOptions[q.ID]
		return (q.Classes != 0 || extra) && q.MinLevel < target && q.Dungeon == nil && onMap
	}
	foreign := func(q *Quest) bool { h := w.home(q.StartPos); return len(h) > 0 && !h[race] }
	// chain: the goal and what it needs, a prerequisite of several the one not in another race's home, nearest
	var chain func(q *Quest, seen map[int]bool) ([]*Quest, bool)
	chain = func(q *Quest, seen map[int]bool) ([]*Quest, bool) {
		if seen[q.ID] {
			return nil, true
		}
		seen[q.ID] = true
		var out []*Quest
		for _, p := range q.PreAll {
			pq := w.ByID[p]
			if pq == nil {
				return nil, false
			}
			c, good := chain(pq, seen)
			if !good {
				return nil, false
			}
			out = append(out, c...)
		}
		if len(q.PreAny) > 0 {
			var best []*Quest
			bd := 1e18
			for _, p := range q.PreAny {
				pq := w.ByID[p]
				if pq == nil {
					continue
				}
				c, good := chain(pq, map[int]bool{})
				if !good {
					continue
				}
				d := 0.0
				if len(c) > 0 {
					d = distance(c[0].StartPos, w.Start)
					if foreign(c[0]) {
						d += 1e9 // another race's start only when there is no other
					}
				}
				if best == nil || d < bd {
					best, bd = c, d
				}
			}
			if best == nil {
				return nil, false
			}
			for _, b := range best {
				if !seen[b.ID] {
					seen[b.ID] = true
					out = append(out, b)
				}
			}
		}
		return append(out, q), true
	}
	var all []ClassChain
	for _, q := range w.Quests {
		if !ok(q) {
			continue
		}
		tier, reward := "optional", ""
		if q.Classes == 0 {
			if r, in := extraOptions[q.ID]; in {
				tier, reward = "recommended", r
			} else {
				continue // an ordinary quest is only a step of an extra option
			}
		} else if r, in := classRequired[q.ID]; in {
			tier, reward = "required", r
		} else if r, in := classRecommended[q.ID]; in {
			tier, reward = "recommended", r
		} else if classLetter(q) {
			tier, reward = "required", "Klassenbrief"
		}
		switch w.classOverride[q.ID] {
		case "off":
			tier = "optional"
		case "on":
			tier = "required"
		}
		qs, good := chain(q, map[int]bool{})
		if !good {
			continue
		}
		all = append(all, ClassChain{Goal: q, Quests: qs, Reward: reward, Tier: tier, foreign: foreign(qs[0])})
	}
	// one chain per reward (a required or recommended reward, else the quest's name): of the versions the one not in
	// another race's home, nearest the start; a quest that is part of a chain kept is no option of its own
	rank := map[string]int{"required": 0, "recommended": 1, "optional": 2}
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if rank[a.Tier] != rank[b.Tier] {
			return rank[a.Tier] < rank[b.Tier]
		}
		if a.foreign != b.foreign {
			return !a.foreign
		}
		return distance(a.Quests[0].StartPos, w.Start) < distance(b.Quests[0].StartPos, w.Start)
	})
	seen, inChain, own := map[string]bool{}, map[int]bool{}, map[string]bool{}
	var out []ClassChain
	for _, c := range all {
		k := c.Tier + "/" + c.Reward
		if c.Tier == "optional" || c.Reward == "Klassenbrief" || c.Reward == "Volkszauber" || c.Reward == "Waffe" || c.Reward == "Dolch" || c.Reward == "Stab" || c.Reward == "Robe" || c.Reward == "Brustteil" || c.Reward == "XP" || c.Reward == "XP und Gegenstand" || c.Reward == "unbekannt (neu)" || c.Reward == "Gegenstand" {
			k = fmt.Sprintf("%s/%d", c.Goal.Name, c.Goal.MinLevel)
		}
		if c.Tier == "recommended" && c.foreign {
			k = c.Tier + "/" + c.Reward // another race's version: only when there is none of its own, the nearest
		}
		if seen[k] || inChain[c.Goal.ID] || c.Tier == "recommended" && c.foreign && own[c.Reward] {
			continue
		}
		if !c.foreign {
			own[c.Reward] = true
		}
		if c.Tier == "optional" && (len(c.Goal.Unlocks) > 0 || c.foreign) {
			continue // a step of a longer chain (offered with its end), or another race's version
		}
		seen[k] = true
		for _, q := range c.Quests {
			inChain[q.ID] = true
		}
		out = append(out, c)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Goal.Index < out[j].Goal.Index })
	return out
}

// tankReward: the class quest reward a class needs to tank a dungeon (the quickest way into a group).
var tankReward = map[string]string{"Druid": "Bärengestalt", "Warrior": "Verteidigungshaltung"}

// tankGoals are the quests that give this character its tank form or stance, nil when it needs none.
func (w *World) tankGoals() map[int]bool {
	r, ok := tankReward[w.Char.Class]
	if !ok {
		return nil
	}
	if w.tank == nil {
		w.tank = map[int]bool{}
		for id, rw := range classRequired {
			if rw == r && w.ByID[id] != nil {
				w.tank[id] = true
			}
		}
	}
	return w.tank
}

// canTank: the character has its tank form or stance (or needs none).
func (w *World) canTank(s *State) bool {
	g := w.tankGoals()
	if len(g) == 0 {
		return true // no quest for it in the data: taken as known
	}
	for id := range g {
		if s.Status[w.ByID[id].Index] == done {
			return true
		}
	}
	return false
}

// tankFirst: a walked route with dungeons gets the tank form or stance before its first run.
func (w *World) tankFirst(r *GuideResult) bool {
	g := w.tankGoals()
	if len(g) == 0 {
		return true
	}
	for _, st := range r.Steps {
		if st.Op == "turnin" && g[st.Quest] {
			return true
		}
		if st.Op == "dungeon" {
			return false
		}
	}
	return true
}
