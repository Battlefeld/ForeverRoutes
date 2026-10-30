package build

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Client capture is deliberately narrow: contextual XP and objective text cannot
// safely replace base XP, target IDs, prerequisites or world coordinates.
func (b *builder) clientQuestNames(path string) error {
	if path == "" {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc struct {
		Schema   int `json:"schema"`
		Captures map[string]struct {
			Build  string `json:"build"`
			Locale string `json:"locale"`
			Region int    `json:"region"`
			Quests map[string]struct {
				ID     int    `json:"id"`
				Title  string `json:"title"`
				SeenAt int64  `json:"seenAt"`
			} `json:"quests"`
		} `json:"captures"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	if doc.Schema != 1 {
		return fmt.Errorf("unsupported client quest capture schema")
	}
	type fact struct {
		title, source         string
		build, region, locale int
		seen                  int64
	}
	best := map[int]fact{}
	hash := fmt.Sprintf("%x", sha256.Sum256(raw))
	for key, c := range doc.Captures {
		if !regexp.MustCompile(`^1\.60\.\d+\.\d+$`).MatchString(c.Build) || key != fmt.Sprintf("%s/%d/%s", c.Build, c.Region, c.Locale) {
			return fmt.Errorf("invalid client capture identity")
		}
		if c.Locale != "enUS" && c.Locale != "enGB" {
			continue
		} // Dataset names are English.
		parts := strings.Split(c.Build, ".")
		build, _ := strconv.Atoi(parts[3])
		for id, q := range c.Quests {
			if id != strconv.Itoa(q.ID) || q.ID <= 0 || q.Title == "" || q.SeenAt <= 0 {
				return fmt.Errorf("invalid captured quest %s", id)
			}
			region, locale := 0, 0
			if c.Region == 3 {
				region = 1
			}
			if c.Locale == "enUS" {
				locale = 1
			}
			f := fact{q.Title, fmt.Sprintf("observed Forever client %s region=%d %s sha256=%s", c.Build, c.Region, c.Locale, hash), build, region, locale, q.SeenAt}
			old, ok := best[q.ID]
			if ok && f.build == old.build && f.region == old.region && f.locale == old.locale && f.seen == old.seen && f.title != old.title {
				return fmt.Errorf("conflicting client quest title %d", q.ID)
			}
			if !ok || f.build > old.build || (f.build == old.build && (f.region > old.region || (f.region == old.region && (f.locale > old.locale || (f.locale == old.locale && f.seen > old.seen))))) {
				best[q.ID] = f
			}
		}
	}
	for i := range b.out {
		q := &b.out[i]
		if f, ok := best[q.ID]; ok {
			if q.Sources == nil {
				q.Sources = map[string]string{}
			}
			q.Name = f.title
			q.Sources["name"] = f.source
			b.stats["client_quest_names"]++
		}
	}
	return nil
}
