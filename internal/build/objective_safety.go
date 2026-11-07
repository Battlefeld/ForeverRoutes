package build

import "fmt"

// A waypoint proves where progress was observed, not whether the item was looted
// from a boss, crafted, bought, or picked up. Keep unresolved sources out of plans.
func (b *builder) objectiveSourceSafety() {
	for i := range b.out {
		q := &b.out[i]
		for _, o := range q.Objectives {
			if o.Kind == "item" && !o.Inside && len(o.Drops) == 0 && len(o.Vendors) == 0 && !((o.Ground || o.UseItem > 0) && len(o.Places) > 0) {
				b.unsupported(q, fmt.Sprintf("item %d: acquisition unresolved; waypoints alone are not a gathering source", o.ID))
				b.stats["item_objectives_acquisition_unresolved"]++
			}
		}
	}
}
