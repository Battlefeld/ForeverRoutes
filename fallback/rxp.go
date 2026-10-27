// Package fallback loads ordinary, pinned RestedXP guides independently of optimized routes.
package fallback

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/Battlefeld/ForeverRoutes/route"
	"os"
)

const Schema = "foreverroutes.rxp-fallback.v1"

type Chapter struct {
	Name string `json:"name"`
	File string `json:"file"`
	Line int    `json:"line"`
	Body string `json:"body"`
}
type Route struct {
	Key            string             `json:"key"`
	Character      route.Character    `json:"character"`
	Target         int                `json:"target"`
	Chapters       []string           `json:"chapters"`
	Events         []route.GuideEvent `json:"events"`
	ProjectionOnly bool               `json:"projectionOnly"`
}
type Catalog struct {
	Schema      string             `json:"schema"`
	Revision    string             `json:"revision"`
	Source      string             `json:"source"`
	License     string             `json:"license"`
	Chapters    map[string]Chapter `json:"chapters"`
	Routes      []Route            `json:"routes"`
	Limitations []string           `json:"limitations"`
	Warnings    []string           `json:"warnings"`
}

func Load(path string) (*Catalog, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer z.Close()
	var c Catalog
	if err = json.NewDecoder(z).Decode(&c); err != nil {
		return nil, err
	}
	if c.Schema != Schema || len(c.Revision) != 40 || c.License != "CC-BY-NC-SA-4.0" {
		return nil, fmt.Errorf("invalid RXP fallback metadata")
	}
	seen := map[string]bool{}
	for _, r := range c.Routes {
		if r.Key != r.Character.RouteKey() || seen[r.Key] || r.Target != 20 || len(r.Chapters) == 0 || len(r.Events) == 0 || !r.ProjectionOnly {
			return nil, fmt.Errorf("invalid or duplicate RXP fallback %s", r.Key)
		}
		seen[r.Key] = true
		for _, key := range r.Chapters {
			ch, ok := c.Chapters[key]
			if !ok || ch.Name != key || ch.Body == "" || ch.File == "" || ch.Line < 1 {
				return nil, fmt.Errorf("RXP fallback %s: invalid chapter %q", r.Key, key)
			}
		}
	}
	for _, ch := range route.SupportedCharacters() {
		if !seen[ch.RouteKey()] {
			return nil, fmt.Errorf("missing RXP fallback %s", ch.RouteKey())
		}
	}
	return &c, nil
}

func (c *Catalog) For(character route.Character) (*Route, error) {
	for i := range c.Routes {
		r := &c.Routes[i]
		if route.SameCharacter(r.Character, character) {
			return r, nil
		}
	}
	return nil, fmt.Errorf("no RXP fallback for %s", character.RouteKey())
}
