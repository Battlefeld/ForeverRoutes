package build

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// questFix is one entry of data/sources/quest-fixes.json: a fact the sources get wrong or leave out, with its page.
type questFix struct {
	Quest           int   `json:"quest"`
	PreAll          []int `json:"preAll,omitempty"`
	AddStartObjects []int `json:"addStartObjects,omitempty"`
	// PlacesFromObjects: the start and end places a source gives are wrong, they follow from the objects
	PlacesFromObjects bool `json:"placesFromObjects,omitempty"`
	// Instant: taken and handed in at the same NPC with nothing to do between (a talk quest that teaches a spell)
	Instant bool   `json:"instant,omitempty"`
	Source  string `json:"source"`
	Note    string `json:"note"`
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
			if f.PreAll != nil {
				q.PreAll, q.PreAny = f.PreAll, nil
				q.Sources["pre"] = f.Source
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
