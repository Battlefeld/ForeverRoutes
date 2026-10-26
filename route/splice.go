package route

// Splice is a route that follows own up to the given level and then other (another race's or class's route of the
// same side) from where that one reaches the level: the way to try another race's areas, e.g. an Undead that goes to
// the Barrens at 10 like the Orc, where the dungeon quests of Ragefire Chasm are on the way. Quests own accepted and
// did not hand in before the cut are dropped unless other hands them in; events of quests already handed in are left
// out of other's part. The cut is at the character's level as followed, not at the guides' level steps.
func (w *World) Splice(own, other []GuideEvent, level, target int) ([]GuideEvent, error) {
	cut := func(events []GuideEvent) (int, error) {
		r, err := w.FollowGuide(events, target)
		if err != nil {
			return 0, err
		}
		for i, l := range r.levels {
			if l >= level {
				return i, nil
			}
		}
		return len(events), nil
	}
	a, err := cut(own)
	if err != nil {
		return nil, err
	}
	b, err := cut(other)
	if err != nil {
		return nil, err
	}
	done, open := map[int]bool{}, map[int]bool{}
	for _, e := range own[:a] {
		switch e.Op {
		case "accept":
			open[e.Quest] = true
		case "turnin":
			done[e.Quest], open[e.Quest] = true, false
		}
	}
	later := map[int]bool{}
	for _, e := range other[b:] {
		if e.Op == "turnin" {
			later[e.Quest] = true
		}
	}
	var out []GuideEvent
	for _, e := range own[:a] {
		if e.Quest != 0 && open[e.Quest] && !later[e.Quest] {
			continue
		}
		out = append(out, e)
	}
	for _, e := range other[b:] {
		if e.Quest != 0 && done[e.Quest] {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}
