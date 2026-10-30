// Package clientquests applies observed names without changing quest rules or rewards.
package clientquests

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Battlefeld/ForeverRoutes/dataset"
)

type Quest struct {
	ID     int    `json:"id"`
	Title  string `json:"title"`
	SeenAt int64  `json:"seenAt"`
}
type Capture struct {
	Build  string           `json:"build"`
	Locale string           `json:"locale"`
	Region int              `json:"region"`
	Quests map[string]Quest `json:"quests"`
}
type Document struct {
	Schema   int                `json:"schema"`
	Captures map[string]Capture `json:"captures"`
}

func Apply(path string, d *dataset.Dataset) (int, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	if len(raw) > 64<<20 {
		return 0, fmt.Errorf("client quest observations too large")
	}
	var doc Document
	if err = json.Unmarshal(raw, &doc); err != nil {
		return 0, err
	}
	if doc.Schema != 1 || doc.Captures == nil {
		return 0, fmt.Errorf("unsupported client quest schema")
	}
	type fact struct {
		q                     Quest
		build, region, locale int
		source                string
	}
	best := map[int]fact{}
	hash := fmt.Sprintf("%x", sha256.Sum256(raw))
	pattern := regexp.MustCompile(`^1\.60\.\d+\.\d+$`)
	for key, c := range doc.Captures {
		if !pattern.MatchString(c.Build) || c.Region < 0 || c.Region > 5 || key != fmt.Sprintf("%s/%d/%s", c.Build, c.Region, c.Locale) {
			return 0, fmt.Errorf("invalid client quest capture identity")
		}
		for id, q := range c.Quests {
			if q.ID <= 0 || id != strconv.Itoa(q.ID) || q.Title == "" || q.SeenAt <= 0 {
				return 0, fmt.Errorf("invalid quest observation %s", id)
			}
		}
		if c.Locale != "enUS" && c.Locale != "enGB" {
			continue
		}
		parts := strings.Split(c.Build, ".")
		build, _ := strconv.Atoi(parts[3])
		region, locale := 0, 0
		if c.Region == 3 {
			region = 1
		}
		if c.Locale == "enUS" {
			locale = 1
		}
		for _, q := range c.Quests {
			f := fact{q, build, region, locale, fmt.Sprintf("observed Forever client %s region=%d %s sha256=%s", c.Build, c.Region, c.Locale, hash)}
			old, ok := best[q.ID]
			if ok && f.build == old.build && f.region == old.region && f.locale == old.locale && f.q.SeenAt == old.q.SeenAt && f.q.Title != old.q.Title {
				return 0, fmt.Errorf("conflicting client title %d", q.ID)
			}
			if !ok || f.build > old.build || (f.build == old.build && (f.region > old.region || (f.region == old.region && (f.locale > old.locale || (f.locale == old.locale && f.q.SeenAt > old.q.SeenAt))))) {
				best[q.ID] = f
			}
		}
	}
	n := 0
	for i := range d.Quests {
		q := &d.Quests[i]
		if f, ok := best[q.ID]; ok {
			if q.Sources == nil {
				q.Sources = map[string]string{}
			}
			q.Name = f.q.Title
			q.Sources["name"] = f.source
			n++
		}
	}
	return n, nil
}
