package build

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

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

// classicSources are quest facts from Classic data that was not updated for Forever (cmangos, ForeverDB's rows that
// only repeat cmangos, Classic XP tables). QuestieDB's Forever data counts as Forever: it is the Era seed checked
// against Wowhead Forever, with the players' traces and the Forever corrections on top.
var classicSources = map[string]bool{"cmangos": true, "foreverdb_classic": true, "cmangos_money_formula": true,
	"octowow_classic": true, "classicdb_classic": true}

func classicSource(s string) bool { return classicSources[strings.SplitN(s, "+", 2)[0]] }

// foreverOnly takes Classic-only facts out (blocks.md, "Nur Forever-Daten"): a giver or turn-in only Classic data
// knows makes the quest unsupported; a Classic XP becomes the typical Forever XP of the quest's level, marked.
func (b *builder) foreverOnly() {
	byLevel := map[int][]int{}
	for _, q := range b.out {
		if q.XP > 0 && q.Sources["xp"] != "" && q.Sources["xp"] != "unknown" && !classicSource(q.Sources["xp"]) {
			byLevel[q.Level] = append(byLevel[q.Level], q.XP)
		}
	}
	for _, l := range byLevel {
		sort.Ints(l)
	}
	for i := range b.out {
		q := &b.out[i]
		for _, side := range []string{"start", "end"} {
			if classicSource(q.Sources[side]) {
				b.unsupported(q, side+" known only from Classic data (not seen in Forever)")
				b.stats["quests_classic_only_"+side]++
			}
		}
		if q.Sources["xp"] != questFixSource && (classicSource(q.Sources["xp"]) || q.XP == 0) {
			if l := byLevel[q.Level]; len(l) >= 3 {
				q.XP = l[len(l)/2]
				q.Sources["xp"] = "estimate: typical Forever XP of a level-" + strconv.Itoa(q.Level) + " quest"
				b.stats["xp_estimated_from_forever"]++
			}
		}
	}
}
