package route

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

type QuestAudit struct {
	Quest            int               `json:"quest"`
	Name             string            `json:"name"`
	MinLevel         int               `json:"minLevel"`
	QuestLevel       int               `json:"questLevel"`
	Status           string            `json:"status"`
	ResolvedContexts int               `json:"resolvedContexts"`
	FeasibleContexts int               `json:"feasibleByTargetContexts"`
	Reasons          []string          `json:"reasons,omitempty"`
	Warnings         []string          `json:"warnings,omitempty"`
	Sources          map[string]string `json:"sources"`
}

type QuestAuditReport struct {
	Schema            string         `json:"schema"`
	Target            int            `json:"target"`
	Examined          int            `json:"examined"`
	CharacterContexts int            `json:"characterContexts"`
	Checks            []string       `json:"checks"`
	Limitations       []string       `json:"limitations"`
	Statuses          map[string]int `json:"statuses"`
	Quests            []QuestAudit   `json:"quests"`
}

// AuditQuests visits every quest accepted by the target level, including intentional exclusions. A quest that is
// missing data or blocked by its chain is never presented as a measured, usable quest. This checks the model and its
// data; it is not an in-game verification of beta values or calibrated event durations.
func AuditQuests(d *dataset.Dataset, target int) (*QuestAuditReport, error) {
	if target < 1 || target > len(d.LevelXP)+1 {
		return nil, fmt.Errorf("audit target out of range")
	}
	report := &QuestAuditReport{Schema: "foreverroutes.quest-audit.v1", Target: target, Statuses: map[string]int{}, Checks: []string{"levels", "objective kinds/counts", "source references", "race/class masks", "prerequisite graph", "solver resolution", "objective feasibility by target"}}
	report.Limitations = []string{"Source/model audit, not verification in the running game", "Combat, travel, drop, dungeon and event durations remain model estimates", "Unknown XP stays conservative zero; unresolved quests never become free quest rewards", "Profession and PvP/raid quests are classified but excluded from the leveling profile"}
	type context struct {
		w        *World
		feasible map[int]bool
	}
	var contexts []context
	races := []string{"Human", "Orc", "Dwarf", "NightElf", "Undead", "Tauren", "Gnome", "Troll", "Skyborne"}
	// Roster shipped in data/routes, including Forever's added race/class combinations.
	roster := map[string][]string{
		"Human":     {"Hunter", "Mage", "Paladin", "Priest", "Rogue", "Warlock", "Warrior"},
		"Orc":       {"Hunter", "Mage", "Rogue", "Shaman", "Warlock", "Warrior"},
		"Dwarf":     {"Hunter", "Paladin", "Priest", "Rogue", "Shaman", "Warrior"},
		"NightElf":  {"Druid", "Hunter", "Priest", "Rogue", "Warrior"},
		"Undead":    {"Mage", "Paladin", "Priest", "Rogue", "Warlock", "Warrior"},
		"Tauren":    {"Druid", "Hunter", "Shaman", "Warrior"},
		"Gnome":     {"Mage", "Priest", "Rogue", "Warlock", "Warrior"},
		"Troll":     {"Hunter", "Mage", "Priest", "Rogue", "Shaman", "Warlock", "Warrior"},
		"SkyborneA": {"Druid", "Hunter", "Mage", "Rogue", "Warrior"},
		"SkyborneH": {"Druid", "Hunter", "Rogue", "Shaman", "Warrior"},
	}
	for _, race := range races {
		factions := []string{""}
		if race == "Skyborne" {
			factions = []string{"A", "H"}
		}
		for _, side := range factions {
			for _, class := range roster[race+side] {
				w, err := NewWorld(d, Character{Race: race, Class: class, Faction: side}, Options{Dungeons: true})
				if err != nil {
					return nil, err
				}
				memo, possible := map[int]bool{}, map[int]bool{}
				for _, q := range w.Quests {
					ok := q.MinLevel <= target
					if q.Dungeon != nil && q.Dungeon.Min > target {
						ok = false
					}
					for _, o := range q.Objectives {
						if o.Inside {
							continue
						}
						can, capacity := false, 0
						for _, src := range o.Sources {
							if !w.dangerousAt(target, src) {
								can = true
								capacity += max(1, src.Site.Spawns)
							}
						}
						if !can || ((o.Kind == "talk" || o.Kind == "interact") && capacity < o.Count) {
							ok = false
						}
					}
					if len(q.StartSources) > 0 {
						can := false
						for _, src := range q.StartSources {
							can = can || !w.dangerousAt(target, src)
						}
						ok = ok && can
					}
					possible[q.ID] = ok
				}
				// Monotone closure handles alternative prerequisites correctly even when another alternative cycles.
				for changed := true; changed; {
					changed = false
					for _, q := range w.Quests {
						if memo[q.ID] || !possible[q.ID] {
							continue
						}
						ok := true
						for _, pre := range q.PreAll {
							ok = ok && memo[pre]
						}
						if len(q.PreAny) > 0 {
							any := false
							for _, pre := range q.PreAny {
								any = any || memo[pre]
							}
							ok = ok && any
						}
						if ok {
							memo[q.ID] = true
							changed = true
						}
					}
				}
				contexts = append(contexts, context{w, memo})
			}
		}
	}
	report.CharacterContexts = len(contexts)
	ids := map[int]bool{}
	for _, q := range d.Quests {
		if q.MinLevel > target {
			continue
		}
		row := QuestAudit{Quest: q.ID, Name: q.Name, MinLevel: q.MinLevel, QuestLevel: q.Level, Sources: q.Sources}
		reasons := map[string]bool{}
		invalid := false
		if q.ID <= 0 || ids[q.ID] {
			invalid = true
			reasons["invalid or duplicate quest id"] = true
		}
		ids[q.ID] = true
		if q.MinLevel < 1 || q.XP < 0 {
			invalid = true
			reasons["invalid minimum level or XP"] = true
		}
		for _, o := range q.Objectives {
			if o.Kind == "event" {
				row.Warnings = append(row.Warnings, "event duration is an interaction estimate; scripted combat/escort needs observation")
			}
			if o.Count <= 0 {
				invalid = true
				reasons["objective count is not positive"] = true
			}
			switch o.Kind {
			case "kill", "duel", "talk", "interact", "item", "object", "event", "skill", "run":
			default:
				invalid = true
				reasons["unknown objective kind: "+o.Kind] = true
			}
			if o.Source == "" {
				row.Warnings = append(row.Warnings, "objective without provenance")
			}
			for _, drop := range o.Drops {
				if drop.Unknown {
					row.Warnings = append(row.Warnings, "drop chance estimated")
					break
				}
			}
		}
		if q.Sources["xp"] == "unknown" {
			row.Warnings = append(row.Warnings, "XP unknown: conservative zero, not verified zero")
		}
		if strings.Contains(q.Sources["objectives"], "assumed") {
			row.Warnings = append(row.Warnings, "objective supply assumed")
		}
		for _, ctx := range contexts {
			_, race, _ := ctx.w.Char.side()
			cls := ClassBits[ctx.w.Char.Class]
			if q.Races != 0 && q.Races&race == 0 || q.Classes != 0 && q.Classes&cls == 0 {
				continue
			}
			if ctx.w.ByID[q.ID] != nil {
				row.ResolvedContexts++
			}
			if ctx.feasible[q.ID] {
				row.FeasibleContexts++
			}
			if why := ctx.w.Excluded[q.ID]; why != "" {
				reasons[why] = true
			}
		}
		switch {
		case invalid:
			row.Status = "data_gap"
		case q.Unsupported != "" && strings.Contains(q.Unsupported, "not a leveling quest"):
			row.Status = "outside_leveling_scope"
		case q.Unsupported != "" && strings.Contains(q.Unsupported, "a profession's quest"):
			row.Status = "profession_not_selected"
		case q.Repeatable && q.Classes == 0 && row.FeasibleContexts > 0:
			row.Status = "repeatable_modelled_once"
		case q.Type == "pvp" || q.Type == "raid":
			row.Status = "group_or_pvp_not_selected"
		case q.Unsupported != "":
			row.Status = "data_gap"
			reasons[q.Unsupported] = true
		case row.FeasibleContexts > 0:
			row.Status = "modelled"
		case row.ResolvedContexts > 0:
			row.Status = "not_feasible_by_target"
			reasons["objective or prerequisite is not feasible by the target level"] = true
		default:
			row.Status = "unresolved"
		}
		for why := range reasons {
			row.Reasons = append(row.Reasons, why)
		}
		sort.Strings(row.Reasons)
		sort.Strings(row.Warnings)
		row.Warnings = slices.Compact(row.Warnings)
		report.Quests = append(report.Quests, row)
		report.Statuses[row.Status]++
	}
	sort.Slice(report.Quests, func(i, j int) bool { return report.Quests[i].Quest < report.Quests[j].Quest })
	report.Examined = len(report.Quests)
	return report, nil
}
