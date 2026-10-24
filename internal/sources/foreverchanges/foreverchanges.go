// Package foreverchanges reads the one-line summary of a quest from ForeverChanges (https://foreverchanges.pro), a
// WoW Forever database built on the game client and Wowhead's Forever data. The summary names the first objective
// ("Collect 6 Soft Nightsaber Pelts from Nightsabers in Teldrassil for ...") and the kind of quest ("A level 30 elite
// quest for both factions"). Only quests the other sources leave open are fetched, one request per second, cached on
// disk. robots.txt allows the quest pages; every entry keeps the page link.
package foreverchanges

import (
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Base is the site.
const Base = "https://foreverchanges.pro"

// Credit is the attribution kept with the data.
const Credit = "ForeverChanges, https://foreverchanges.pro (data from the WoW Forever client and Wowhead's Forever database)"

// Quest is what the summary says.
type Quest struct {
	ID    int    `json:"id"`
	Kind  string `json:"kind"`            // normal, elite, dungeon, or whatever the page calls it ("" = could not read)
	Level int    `json:"level,omitempty"` //
	Text  string `json:"text,omitempty"`  // the objective sentence
	URL   string `json:"url"`
}

// Overlay is the file the build reads.
type Overlay struct {
	Schema  string        `json:"schema"`
	Source  string        `json:"source"`
	Fetched string        `json:"fetched"`
	Quests  map[int]Quest `json:"quests"`
}

// Client fetches pages with a delay and a disk cache.
type Client struct {
	HTTP  *http.Client
	Cache string
	Delay time.Duration
	Calls int
	last  time.Time
}

var errNotFound = fmt.Errorf("not found")

// IsNotFound tells whether the site does not know the quest.
func IsNotFound(err error) bool { return err == errNotFound }

func (c *Client) get(id int) ([]byte, error) {
	cache := filepath.Join(c.Cache, "quest_"+strconv.Itoa(id)+".html")
	if b, err := os.ReadFile(cache); err == nil {
		return b, nil
	}
	if wait := c.Delay - time.Since(c.last); !c.last.IsZero() && wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
	c.Calls++
	req, err := http.NewRequest(http.MethodGet, Base+"/quest/"+strconv.Itoa(id), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ForeverRoutes (https://github.com/Battlefeld/ForeverRoutes)")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, errNotFound
	case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("foreverchanges: HTTP %d, stopping", resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("foreverchanges: HTTP %d for quest %d", resp.StatusCode, id)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(c.Cache, 0o755); err != nil {
		return nil, err
	}
	return b, os.WriteFile(cache, b, 0o644)
}

// Quest fetches and parses one quest page.
func (c *Client) Quest(id int) (Quest, error) {
	b, err := c.get(id)
	if err != nil {
		return Quest{}, err
	}
	return Parse(id, b)
}

var (
	desc = regexp.MustCompile(`<meta name="description" content="(.*?)"`)
	kind = regexp.MustCompile(`^(.*?)\s*A level (\d+) ([a-z ]*?)quest for `)
)

// Parse reads the page's description line.
func Parse(id int, page []byte) (Quest, error) {
	q := Quest{ID: id, URL: Base + "/quest/" + strconv.Itoa(id)}
	m := desc.FindSubmatch(page)
	if m == nil {
		return q, fmt.Errorf("foreverchanges: quest %d: no description, page layout changed?", id)
	}
	line := html.UnescapeString(string(m[1]))
	k := kind.FindStringSubmatch(line)
	if k == nil {
		q.Text = strings.TrimSpace(line)
		return q, nil
	}
	q.Text = strings.TrimSpace(k[1])
	q.Level, _ = strconv.Atoi(k[2])
	q.Kind = strings.TrimSpace(k[3])
	if q.Kind == "" {
		q.Kind = "normal"
	}
	return q, nil
}

// Source is a creature (and where) an objective sentence says an item comes from.
type Source struct {
	Name string
	Zone string
}

var (
	fromRx   = regexp.MustCompile(`\bfrom (?:the |a |an |nearby )?([A-Z][A-Za-z' -]+?(?:(?:,| and | or ) ?(?:the )?[A-Z][A-Za-z' -]+?)*)(?: (?:in|near|at|around|on) (?:the )?([A-Z][A-Za-z' -]+?))?(?: for | and (?:bring|return|take|deliver)|,? then |,|\.|$)`)
	splitter = regexp.MustCompile(`, | and | or `)
)

// Sources reads "…from <Creatures> in <Zone>…" out of an objective sentence. Names may be plural; the build matches
// them against the creature data.
func Sources(text string) []Source {
	m := fromRx.FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	var out []Source
	for _, n := range splitter.Split(m[1], -1) {
		n = strings.TrimSpace(strings.TrimPrefix(n, "the "))
		if n != "" {
			out = append(out, Source{Name: n, Zone: strings.TrimSpace(m[2])})
		}
	}
	return out
}
