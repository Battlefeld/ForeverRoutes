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
	PlacesFromObjects bool   `json:"placesFromObjects,omitempty"`
	Source            string `json:"source"`
	Note              string `json:"note"`
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
			b.stats["quest_fixes_applied"]++
		}
	}
}
