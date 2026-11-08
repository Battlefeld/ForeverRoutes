package build

import "fmt"

// Event quests: holiday and world event quests (Lunar Festival, Scourge Invasion, ...) are given only while their
// event runs. cmangos names them in game_event_quest, or gives them through creatures that appear only with an event
// (every spawn of the giver in game_event_creature with a positive event). They are no leveling quests: a route that
// sends the player to them fails outside the event.
func (b *builder) eventQuests() {
	if b.w == nil || b.w.eventCreatures == nil || b.w.cr == nil || b.w.cqr == nil {
		return
	}
	names := map[int]string{}
	if t := b.w.events; t != nil {
		for _, r := range t.Rows {
			names[t.Int(r, "entry")] = t.String(r, "description")
		}
	}
	event := map[int]int{} // quest -> event
	if t := b.w.eventQuests; t != nil {
		for _, r := range t.Rows {
			event[t.Int(r, "quest")] = t.Int(r, "event")
		}
	}
	spawnEvent := map[int]int{} // creature guid -> its (positive) event
	for _, r := range b.w.eventCreatures.Rows {
		if e := b.w.eventCreatures.Int(r, "event"); e > 0 {
			spawnEvent[b.w.eventCreatures.Int(r, "guid")] = e
		}
	}
	// a creature appears only with an event when every spawn of it does
	onlyWith := map[int]int{} // creature -> event, 0 when it is there without one
	for _, r := range b.w.cr.Rows {
		id, e := b.w.cr.Int(r, "id"), spawnEvent[b.w.cr.Int(r, "guid")]
		if prev, seen := onlyWith[id]; !seen || prev != 0 {
			if e == 0 {
				onlyWith[id] = 0
			} else if !seen {
				onlyWith[id] = e
			}
		}
	}
	givers := map[int][]int{}
	for _, r := range b.w.cqr.Rows {
		givers[b.w.cqr.Int(r, "quest")] = append(givers[b.w.cqr.Int(r, "quest")], b.w.cqr.Int(r, "id"))
	}
	for q, cs := range givers {
		if event[q] != 0 || len(cs) == 0 {
			continue
		}
		e := 0
		for _, c := range cs {
			if ev, ok := onlyWith[c]; !ok || ev == 0 {
				e = 0
				break
			} else {
				e = ev
			}
		}
		if e != 0 {
			event[q] = e
		}
	}
	for i := range b.out {
		q := &b.out[i]
		if e := event[q.ID]; e != 0 {
			name := names[e]
			if name == "" {
				name = fmt.Sprintf("event %d", e)
			}
			q.Unsupported = fmt.Sprintf("event quest, not a leveling quest (%s)", name)
			b.stats["event_quests"]++
		}
	}
}
