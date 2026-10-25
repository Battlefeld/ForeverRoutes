package build

import (
	"sort"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// curatedDungeon is one dungeon of data/sources/dungeons.json.
type curatedDungeon struct {
	Key      string  `json:"key"`
	Entrance string  `json:"entrance"` // Polestar entrance key when it is not Key (the wings of a monastery)
	Name     string  `json:"name"`
	Instance int     `json:"instance"`
	Areas    []int   `json:"areas"`
	Levels   [2]int  `json:"levels"`
	Minutes  float64 `json:"minutes"`
	Faction  string  `json:"faction"`
	Quests   []int   `json:"quests"`
}

// dungeons builds the dungeons routes with dungeons may run and ties the dungeon quests to them: by the quest's client area,
// else by the instance an objective's creature stands in. An objective with no open-world source is marked inside (a
// run completes it); a dungeon quest whose only problem is that its objective or giver is inside the dungeon is no
// longer unsupported (it stays a dungeon quest, out of solo routes).
func (b *builder) dungeons() []dataset.Dungeon {
	if len(b.curatedDungeons) == 0 || b.pdb == nil {
		return nil
	}
	doors := map[string]dataset.Point{}
	for _, e := range b.pdb.Entrances {
		if p, ok := b.uiToWorld(e.UIMap, e.X, e.Y); ok {
			doors[e.Key] = p
		}
	}
	wings := map[int]int{}
	for _, c := range b.curatedDungeons {
		wings[c.Instance]++
	}
	var out []dataset.Dungeon
	byArea, byInstance, byQuest := map[int]int{}, map[int]int{}, map[int]int{}
	for _, c := range b.curatedDungeons {
		door := c.Entrance
		if door == "" {
			door = c.Key
		}
		p, ok := doors[door]
		if !ok {
			b.stats["dungeons_without_entrance"]++
			continue
		}
		i := len(out)
		for _, a := range c.Areas {
			if _, dup := byArea[a]; !dup {
				byArea[a] = i
			}
		}
		if _, dup := byInstance[c.Instance]; !dup {
			byInstance[c.Instance] = i
		}
		for _, q := range c.Quests {
			byQuest[q] = i
		}
		out = append(out, dataset.Dungeon{Key: c.Key, Name: c.Name, Instance: c.Instance, Faction: c.Faction, Min: c.Levels[0], Max: c.Levels[1],
			Minutes: c.Minutes, Entrance: p, Mobs: b.dungeonMobs(c.Instance, wings[c.Instance]),
			Source: "data/sources/dungeons.json (levels, run time); entrance: Polestar entrances; trash: cmangos instance spawns"})
	}
	npcs := map[int]*dataset.NPC{}
	for i := range b.npcOut {
		npcs[b.npcOut[i].ID] = &b.npcOut[i]
	}
	objs := map[int]*dataset.Object{}
	for i := range b.objOut {
		objs[b.objOut[i].ID] = &b.objOut[i]
	}
	inWorld := func(ps []dataset.Point) bool {
		for _, p := range ps {
			if p.Map == 0 || p.Map == 1 {
				return true
			}
		}
		return false
	}
	npcOut := func(id int) bool { n := npcs[id]; return n != nil && inWorld(n.Spawns) }
	objOut := func(id int) bool { o := objs[id]; return o != nil && inWorld(o.Spawns) }
	// placed: the quest's giver or turn-in stands in the open world. Some sources put an NPC inside the dungeon at
	// its entrance; that is inside too.
	placed := func(refs []dataset.Ref, at *dataset.Point, door dataset.Point) bool {
		if at != nil {
			return !dataset.AtEntrance(*at, door)
		}
		for _, r := range refs {
			if r.Kind == "npc" && npcs[r.ID] != nil {
				for _, p := range npcs[r.ID].Spawns {
					if (p.Map == 0 || p.Map == 1) && !dataset.AtEntrance(p, door) {
						return true
					}
				}
			}
			if r.Kind == "object" && objOut(r.ID) {
				return true
			}
		}
		return false
	}
	instanceOf := func(q *dataset.Quest) (int, bool) {
		for _, o := range q.Objectives {
			ids := []int{o.ID}
			if o.Kind == "item" {
				ids = ids[:0]
				for _, d := range o.Drops {
					if d.NPC > 0 {
						ids = append(ids, d.NPC)
					}
				}
			} else if o.Kind != "kill" {
				continue
			}
			for _, id := range ids {
				for m, cr := range b.w.inside {
					if i, ok := byInstance[m]; ok && cr[id] > 0 {
						return i, true
					}
				}
			}
		}
		return 0, false
	}
	for qi := range b.out {
		q := &b.out[qi]
		if q.Type != "dungeon" {
			continue
		}
		i, ok := byQuest[q.ID]
		if !ok {
			if pq := b.pdb.Quests[q.ID]; pq != nil {
				i, ok = byArea[pq.Zone]
			}
		}
		if !ok {
			i, ok = instanceOf(q)
		}
		if !ok {
			b.stats["dungeon_quests_without_dungeon"]++
			continue
		}
		q.Dungeon = out[i].Key
		out[i].Quests = append(out[i].Quests, q.ID)
		b.stats["dungeon_quests_tied"]++
		inside := false
		for j := range q.Objectives {
			o := &q.Objectives[j]
			switch o.Kind {
			case "kill", "talk":
				o.Inside = !npcOut(o.ID)
			case "object":
				o.Inside = !objOut(o.ID)
			case "item":
				src := len(o.Vendors) > 0 || len(o.Places) > 0
				for _, d := range o.Drops {
					src = src || (d.NPC > 0 && npcOut(d.NPC)) || (d.Object > 0 && objOut(d.Object))
				}
				o.Inside = !src
			case "event":
				o.Inside = len(o.Places) == 0
			}
			inside = inside || o.Inside
		}
		cleared := q.Unsupported != ""
		if cleared {
			q.Sources["unsupported_solo"] = q.Unsupported
			q.Unsupported, q.NeedItems = "", nil
			b.stats["dungeon_quests_inside"]++
		}
		if !inside && (cleared || (len(q.Objectives) == 0 && (!placed(q.Start, q.StartAt, out[i].Entrance) || !placed(q.End, q.EndAt, out[i].Entrance)))) {
			// what is done inside is not known in detail: the run does it
			q.Objectives = append(q.Objectives, dataset.Objective{Kind: "run", ID: out[i].Instance, Count: 1, Source: "dungeon", Inside: true})
		}
	}
	for i := range out {
		sort.Ints(out[i].Quests)
	}
	b.stats["dungeons"] = len(out)
	return out
}

// dungeonMobs: the creatures cmangos spawns in an instance, by level and elite; a monastery's wings share the
// instance, each gets its share.
func (b *builder) dungeonMobs(instance, wings int) []dataset.DungeonMob {
	n := map[[2]int]int{}
	ct := b.w.ct
	for id, c := range b.w.inside[instance] {
		r := b.w.creature[id]
		if r == nil {
			continue
		}
		l := (ct.Int(r, "MinLevel") + ct.Int(r, "MaxLevel") + 1) / 2
		elite := 0
		if ct.Int(r, "Rank") > 0 && ct.Int(r, "Rank") != 4 {
			elite = 1
		}
		if l <= 1 {
			continue // triggers and critters
		}
		n[[2]int{l, elite}] += c
	}
	var out []dataset.DungeonMob
	for k, c := range n {
		if c = (c + wings/2) / wings; c > 0 {
			out = append(out, dataset.DungeonMob{Level: k[0], Elite: k[1] == 1, N: c})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Level != out[j].Level {
			return out[i].Level < out[j].Level
		}
		return !out[i].Elite && out[j].Elite
	})
	return out
}
