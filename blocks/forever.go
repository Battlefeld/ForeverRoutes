// Package blocks prepares quest data for routes built from area blocks: what ForeverDB's players saw of a quest and
// which of the quests before it are really required.
//
// Data from ForeverDB.net (https://foreverdb.net).
package blocks

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

// Kinds of a link to an earlier quest.
const (
	Required  = "required"  // "Complete first": the quest is not offered before
	Preferred = "preferred" // "Offered after": players took it after that quest; done first when the route has both
	Optional  = "optional"  // "Chain link": a quest that leads to this one, not needed for it
)

// Confirmation of a quest in Forever, best first.
const (
	Seen    = "seen"    // players saw its giver in WoW Forever
	Forever = "forever" // Forever data, its giver still from Classic data
	Classic = "classic" // Classic data only
	Unknown = "unknown" // not in ForeverDB's list
)

// Link is one earlier quest of a quest.
type Link struct {
	Quest   int    `json:"quest"`
	Kind    string `json:"kind"`
	Seen    bool   `json:"seen"` // seen in WoW Forever (else Classic data)
	Players int    `json:"players,omitempty"`
}

// Quest is what ForeverDB knows of one quest.
type Quest struct {
	ID      int
	Source  string // forever or classic
	Players int    // players who saw its giver
	Giver   bool   // the giver was seen in WoW Forever
	Before  []Link
}

// Confirmed tells how well the quest is confirmed in Forever.
func (q *Quest) Confirmed() string {
	switch {
	case q == nil:
		return Unknown
	case q.Giver:
		return Seen
	case q.Source == "forever":
		return Forever
	}
	return Classic
}

// List is ForeverDB's quest list by quest id.
type List struct {
	Fetched string
	Build   string
	Quests  map[int]*Quest
}

const seenInForever = "seen in WoW Forever"

// Load reads the full list written by tools/foreverdb_full.py.
func Load(path string) (*List, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Schema  string `json:"schema"`
		Fetched string `json:"fetched"`
		Game    struct {
			Build string `json:"build"`
		} `json:"game_metadata"`
		Quests []struct {
			ID     int    `json:"id"`
			Source string `json:"source"`
			Starts *struct {
				Source  string `json:"source"`
				Players int    `json:"players"`
			} `json:"starts"`
			Chain struct {
				Previous []struct {
					Quest    int    `json:"questId"`
					Relation string `json:"relation"`
					Source   string `json:"source"`
					Players  int    `json:"players"`
				} `json:"previous"`
			} `json:"chain"`
		} `json:"quests"`
	}
	if err := json.NewDecoder(z).Decode(&doc); err != nil {
		return nil, err
	}
	if doc.Schema != "foreverroutes.foreverdb_full.v1" {
		return nil, fmt.Errorf("%s: unknown schema %q", path, doc.Schema)
	}
	l := &List{Fetched: doc.Fetched, Build: doc.Game.Build, Quests: map[int]*Quest{}}
	for _, r := range doc.Quests {
		q := &Quest{ID: r.ID, Source: r.Source}
		if r.Starts != nil {
			q.Giver, q.Players = r.Starts.Source == seenInForever, r.Starts.Players
		}
		for _, p := range r.Chain.Previous {
			kind := ""
			switch p.Relation {
			case "Complete first", "Must be active":
				kind = Required
			case "Offered after":
				kind = Preferred
			case "Chain link":
				kind = Optional
			default:
				continue // "Next quest", "Quest line": no order
			}
			q.Before = append(q.Before, Link{Quest: p.Quest, Kind: kind, Seen: p.Source == seenInForever, Players: p.Players})
		}
		l.Quests[r.ID] = q
	}
	return l, nil
}

// Change is one quest whose prerequisites Apply rewrote.
type Change struct {
	Quest     int   `json:"quest"`
	Was       []int `json:"was"`
	Required  []int `json:"required,omitempty"`
	Preferred []int `json:"preferred,omitempty"`
	Optional  []int `json:"optional,omitempty"`
}

// Apply rewrites, in d, the prerequisites the dataset took from ForeverDB's chains (and only those): a quest keeps
// the earlier quests marked "Complete first" and loses the ones that only lead to it ("Chain link") or were merely
// taken before it ("Offered after"). The preferred quests are returned so a route can still do them first.
func (l *List) Apply(d *dataset.Dataset) (changes []Change, preferred map[int][]int) {
	preferred = map[int][]int{}
	for i := range d.Quests {
		q := &d.Quests[i]
		if q.Sources["prerequisites"] != "foreverdb_chain" {
			continue
		}
		f := l.Quests[q.ID]
		if f == nil {
			continue
		}
		had := map[int]bool{}
		was := append(append([]int(nil), q.PreAll...), q.PreAny...)
		for _, p := range was {
			had[p] = true
		}
		c := Change{Quest: q.ID, Was: was}
		for _, b := range f.Before {
			if !had[b.Quest] {
				continue // the build left it out (unknown quest, a loop)
			}
			switch b.Kind {
			case Required:
				c.Required = append(c.Required, b.Quest)
			case Preferred:
				c.Preferred = append(c.Preferred, b.Quest)
			case Optional:
				c.Optional = append(c.Optional, b.Quest)
			}
		}
		sort.Ints(c.Required)
		if len(c.Required) == len(was) {
			continue // nothing but required quests: as it was
		}
		q.PreAll, q.PreAny = nil, nil
		switch {
		case len(c.Required) == 1:
			q.PreAll = c.Required
		case len(c.Required) > 1:
			q.PreAny = c.Required // several: the race or faction variants of one step, as the build reads them
		}
		if len(c.Preferred) > 0 {
			preferred[q.ID] = c.Preferred
		}
		changes = append(changes, c)
	}
	return changes, preferred
}

// Shareable reads the quests a player can share with the group (tools output data/audits/quest-flags-classic.json:
// cmangos' QuestFlags, Classic data, not confirmed in Forever).
func Shareable(path string) (map[int]bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Schema    string `json:"schema"`
		Shareable []int  `json:"shareable"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, err
	}
	if doc.Schema != "foreverroutes.quest_flags.v1" {
		return nil, fmt.Errorf("%s: unknown schema %q", path, doc.Schema)
	}
	out := make(map[int]bool, len(doc.Shareable))
	for _, id := range doc.Shareable {
		out[id] = true
	}
	return out, nil
}
