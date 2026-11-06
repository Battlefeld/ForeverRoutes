package build

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// questFix is one entry of data/sources/quest-fixes.json: a fact the sources get wrong or leave out, with its page.
type questFix struct {
	Quest           int   `json:"quest"`
	MinLevel        *int  `json:"minLevel,omitempty"`
	PreAll          []int `json:"preAll,omitempty"`
	PreActive       []int `json:"preActive,omitempty"`
	EndAfter        []int `json:"endAfter,omitempty"`
	AddStartObjects []int `json:"addStartObjects,omitempty"`
	// PlacesFromObjects: the start and end places a source gives are wrong, they follow from the objects
	PlacesFromObjects bool `json:"placesFromObjects,omitempty"`
	// Instant: taken and handed in at the same NPC with nothing to do between (a talk quest that teaches a spell)
	Instant bool `json:"instant,omitempty"`
	// ObjectiveKinds maps NPC credit IDs to a sourced action (talk or interact), preserving counts and positions.
	ObjectiveKinds   map[int]string `json:"objectiveKinds,omitempty"`
	ObjectiveTargets map[int][]int  `json:"objectiveTargets,omitempty"`
	// ObjectiveDrops: the only creatures an item objective drops from (item ID to NPC IDs); a source that names
	// another creature is wrong and would send the route there
	ObjectiveDrops map[int][]int `json:"objectiveDrops,omitempty"`
	// NPCRanks: the rank of a creature of the quest the sources leave out (1 elite), NPC ID to rank
	NPCRanks   map[int]int         `json:"npcRanks,omitempty"`
	AddFight   int                 `json:"addFight,omitempty"`
	Objectives []dataset.Objective `json:"objectives,omitempty"`
	// Profession: the quest is offered only to characters with this profession (the routes learn Skinning, Cooking
	// and Leatherworking; any other one leaves the quest out, OtherProfession)
	Profession string `json:"profession,omitempty"`
	Source     string `json:"source"`
	Note       string `json:"note"`
}

func readQuestFixes(path string) ([]questFix, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f struct {
		Fixes []questFix `json:"fixes"`
	}
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("quest fixes: %w", err)
	}
	for _, fix := range f.Fixes {
		if fix.Quest <= 0 || fix.Source == "" {
			return nil, fmt.Errorf("quest fix needs an id and source")
		}
		for _, kind := range fix.ObjectiveKinds {
			if kind != "talk" && kind != "interact" && kind != "duel" {
				return nil, fmt.Errorf("quest %d: invalid objective kind %q", fix.Quest, kind)
			}
		}
		for _, o := range fix.Objectives {
			if o.ID <= 0 || o.Count <= 0 || o.Kind != "duel" {
				return nil, fmt.Errorf("quest %d: invalid replacement duel", fix.Quest)
			}
		}
	}
	return f.Fixes, nil
}

// applyQuestFixes sets the curated facts over what the sources say.
func (b *builder) applyQuestFixes() {
	for _, f := range b.questFixes {
		for i := range b.out {
			q := &b.out[i]
			if q.ID != f.Quest {
				continue
			}
			if f.MinLevel != nil {
				q.MinLevel = *f.MinLevel
				q.Sources["minLevel"] = f.Source
			}
			if f.Profession != "" && !routeProfessions[strings.ToLower(f.Profession)] {
				if q.Unsupported == "" {
					b.stats["quests_unsupported"]++
				}
				q.Unsupported = OtherProfession
				q.Sources["unsupported"] = f.Source
			}
			if f.AddFight > 0 {
				b.npcWanted[f.AddFight] = true
			}
			if f.Objectives != nil {
				q.Objectives = slices.Clone(f.Objectives)
				q.Unsupported, q.NeedItems = "", nil
				q.Sources["objectives"] = f.Source
				for j := range q.Objectives {
					q.Objectives[j].Source = f.Source
					b.npcWanted[q.Objectives[j].ID] = true
				}
			}
			if f.PreAll != nil {
				q.PreAll, q.PreAny = f.PreAll, nil
				q.Sources["pre"] = f.Source
			}
			if f.PreActive != nil {
				q.PreActive = f.PreActive
				q.Sources["preActive"] = f.Source
			}
			if f.EndAfter != nil {
				q.EndAfter = f.EndAfter
				q.Sources["endAfter"] = f.Source
			}
			for _, o := range f.AddStartObjects {
				if !slices.ContainsFunc(q.Start, func(r dataset.Ref) bool { return r.Kind == "object" && r.ID == o }) {
					q.Start = append(q.Start, dataset.Ref{Kind: "object", ID: o})
					q.StartAt = nil // the place follows from the objects
				}
				b.objWanted[o] = true
				q.Sources["start"] = f.Source
			}
			if f.PlacesFromObjects {
				q.StartAt, q.EndAt = nil, nil
				for _, r := range append(append([]dataset.Ref(nil), q.Start...), q.End...) {
					if r.Kind == "object" {
						b.objWanted[r.ID] = true
					}
				}
				q.Sources["startAt"], q.Sources["endAt"] = f.Source, f.Source
			}
			if f.Instant {
				b.instantOK[q.ID] = true
				q.Sources["objectives"] = f.Source
			}
			for j := range q.Objectives {
				o := &q.Objectives[j]
				if kind, ok := f.ObjectiveKinds[o.ID]; ok && o.Kind == "kill" {
					o.Kind, o.Source = kind, f.Source
					q.Sources["objectiveAction"] = f.Source
				}
				if npcs, ok := f.ObjectiveDrops[o.ID]; ok && o.Kind == "item" {
					var drops []dataset.Drop
					for _, id := range npcs {
						drop := dataset.Drop{NPC: id, Unknown: true}
						for _, d := range o.Drops {
							if d.NPC == id {
								drop = d
							}
						}
						drops = append(drops, drop)
						b.npcWanted[id] = true
					}
					o.Drops, o.Source = drops, f.Source
					q.Sources["objectiveDrops"] = f.Source
				}
				if targets, ok := f.ObjectiveTargets[o.ID]; ok {
					o.Targets = targets
					for _, id := range targets {
						b.npcWanted[id] = true
					}
				}
			}
			b.stats["quest_fixes_applied"]++
		}
	}
}

// skillQuests: Forever's "Camping 101" quests for Skinning and Leatherworking ask for the profession at 20 and are
// handed in at its trainer (warcraft.wiki.gg/wiki/Camping_101:_Skinning). Their objective is that skill, done at the
// trainer once the route has it (route/camp.go).
func (b *builder) skillQuests() {
	skills := map[string]int{"Camping 101: Skinning": dataset.SkillSkinning, "Camping 101: Leatherworking": dataset.SkillLeatherworking}
	for i := range b.out {
		q := &b.out[i]
		skill, ok := skills[q.Name]
		if !ok || (q.Unsupported != "" && q.Unsupported != "event objective without a place") {
			continue
		}
		var at []dataset.Point
		for _, r := range q.End {
			if r.Kind == "npc" {
				if p, ok := b.npcPoint(r.ID); ok {
					at = append(at, p)
				}
			}
		}
		if len(at) == 0 {
			continue
		}
		q.Objectives = []dataset.Objective{{Kind: "skill", ID: skill, Count: 20, Source: "warcraft.wiki.gg", Places: at[:1]}}
		q.Unsupported = ""
		b.stats["skill_quests"]++
	}
}

// NotLeveling is the reason of a quest no leveling route does: a holiday's, a battleground's or a world event's, or a
// placeholder ("REUSE"). Such quests are left out and not counted as gaps in the data.
const NotLeveling = "not a leveling quest (holiday, battleground or event)"

// OtherProfession is the reason of a profession's tutorial quest the routes do not take: the routes learn Skinning,
// Cooking and Leatherworking (route/camp.go).
const OtherProfession = "a profession's quest (the routes learn Skinning, Cooking and Leatherworking)"

// routeProfessions: the professions the routes learn (route/camp.go).
var routeProfessions = map[string]bool{"skinning": true, "cooking": true, "leatherworking": true}

var campingOther = regexp.MustCompile(`^Camping 101: (Alchemy|Blacksmithing|Enchanting|Engineering|First Aid|Fishing|Herbalism|Mining|Tailoring)$`)

// notLevelingName: names of quests that belong to holidays, battlegrounds, world events and placeholders.
var notLevelingName = regexp.MustCompile(`(?i)^(REUSE|Tip the Scales|Past (Victories|Efforts) in|Five Signets for War Supplies|` +
	`Mark of Honor|Talismans of Merit|Warsong (Runner|Scout) Update|(Battle|Fight) for Warsong Gulch|Battle of Warsong Gulch|` +
	`(Gnomes|Goblins) Win!|Wabbit Pelts|Zombie Cleansing|What Is Love\?|Love Hurts|A (Carefully Wrapped|Gently Shaken|Gaily Wrapped) Present|` +
	`A (Gently Shaken|Festive) Gift|.*Gulch|.*Arathi Basin|.*Alterac Valley)`)

// notLevelingZones: ForeverDB's zones of quests outside leveling.
var notLevelingZones = map[string]bool{"Seasonal": true, "Battlegrounds": true, "Darkmoon Faire": true, "Lunar Festival": true,
	"Midsummer": true, "Brewfest": true, "Noblegarden": true, "Love is in the Air": true, "Hallow's End": true,
	"Winter Veil": true, "Children's Week": true, "Warsong Gulch": true, "Arathi Basin": true, "Alterac Valley": true}

// notLeveling marks the quests no leveling route does (NotLeveling).
func (b *builder) notLeveling() {
	for i := range b.out {
		q := &b.out[i]
		zone, _ := get(b.merged, "quests", fmt.Sprint(q.ID), "fields", "zone", "value", "name").(string)
		if campingOther.MatchString(q.Name) {
			if q.Unsupported == "" {
				b.stats["quests_unsupported"]++
			}
			q.Unsupported = OtherProfession
			b.stats["quests_other_profession"]++
			continue
		}
		if notLevelingName.MatchString(strings.TrimSpace(q.Name)) || notLevelingZones[zone] {
			if q.Unsupported == "" {
				b.stats["quests_unsupported"]++
			}
			q.Unsupported = NotLeveling
			b.stats["quests_not_leveling"]++
		}
	}
}

// applyNPCRanks sets the creature ranks the quest fixes give (an elite the sources list as a plain creature).
func (b *builder) applyNPCRanks() {
	for _, f := range b.questFixes {
		for id, rank := range f.NPCRanks {
			for i := range b.npcOut {
				if b.npcOut[i].ID == id {
					b.npcOut[i].Rank = rank
					b.stats["npc_ranks_fixed"]++
				}
			}
		}
	}
}
