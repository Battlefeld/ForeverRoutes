// Package wclbox reads where an item comes from out of WCLBox's WoW Forever database
// (https://wowforever.wclbox.com, "新手盒子·魔兽世界无限数据库"). Its item pages say "Source: <creature>, <zone>" taken from
// the game client's item records and quest data. Only the items a quest needs and no other source explains are
// fetched, one request per second, cached on disk. robots.txt allows the pages and the site invites automated readers
// (llms.txt); it asks that the data is credited to WCLBox with the page link, which every entry keeps.
//
// The page names the creature and its zone but not the drop chance; the chance stays unknown.
package wclbox

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
const Base = "https://wowforever.wclbox.com"

// Credit is the attribution the site asks for.
const Credit = "WCLBox · WoW Forever Database (新手盒子·魔兽世界无限数据库), https://wowforever.wclbox.com"

// Source is one place an item comes from: a creature (or an activity such as Mining) and the zone.
type Source struct {
	Name string `json:"name"`
	Zone string `json:"zone,omitempty"`
}

// Item is what the page says about one item.
type Item struct {
	ID      int      `json:"id"`
	Name    string   `json:"name"`
	Sources []Source `json:"sources,omitempty"` // empty: the page names no source
	URL     string   `json:"url"`
}

// Overlay is the file the build reads.
type Overlay struct {
	Schema  string       `json:"schema"`
	Source  string       `json:"source"`
	Fetched string       `json:"fetched"`
	Build   string       `json:"build,omitempty"` // Forever build the site showed
	Items   map[int]Item `json:"items"`
}

// Client fetches pages with a delay and a disk cache.
type Client struct {
	HTTP  *http.Client
	Cache string
	Delay time.Duration
	Calls int
	last  time.Time
	Build string // Forever build seen on the last page
}

var errNotFound = fmt.Errorf("not found")

// IsNotFound tells whether the site does not know the item.
func IsNotFound(err error) bool { return err == errNotFound }

func (c *Client) get(id int) ([]byte, error) {
	cache := filepath.Join(c.Cache, "item_"+strconv.Itoa(id)+".html")
	if b, err := os.ReadFile(cache); err == nil {
		return b, nil
	}
	if wait := c.Delay - time.Since(c.last); !c.last.IsZero() && wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
	c.Calls++
	req, err := http.NewRequest(http.MethodGet, Base+"/en/wupin/"+strconv.Itoa(id), nil)
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
		return nil, fmt.Errorf("wclbox: HTTP %d, stopping", resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("wclbox: HTTP %d for item %d", resp.StatusCode, id)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(c.Cache, 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(cache, b, 0o644); err != nil {
		return nil, err
	}
	return b, nil
}

// Item fetches and parses one item page.
func (c *Client) Item(id int) (Item, error) {
	b, err := c.get(id)
	if err != nil {
		return Item{}, err
	}
	it, build, err := Parse(id, b)
	if build != "" {
		c.Build = build
	}
	return it, err
}

var (
	title   = regexp.MustCompile(`(?s)<title>(.*?) \(Item \d+\)`)
	drops   = regexp.MustCompile(`(?s)<p class="wp-drops">(.*?)</p>`)
	bold    = regexp.MustCompile(`(?s)<b>(.*?)</b>([^<]*)`)
	tags    = regexp.MustCompile(`<[^>]+>`)
	buildRx = regexp.MustCompile(`Beta build ([0-9.]+)`)
)

// Parse reads an item page. A page without a source line gives an item without sources.
func Parse(id int, page []byte) (Item, string, error) {
	s := string(page)
	t := title.FindStringSubmatch(s)
	if t == nil {
		return Item{}, "", fmt.Errorf("wclbox: item %d: no title, page layout changed?", id)
	}
	it := Item{ID: id, Name: clean(t[1]), URL: Base + "/en/wupin/" + strconv.Itoa(id)}
	build := ""
	if m := buildRx.FindStringSubmatch(s); m != nil {
		build = m[1]
	}
	if m := drops.FindStringSubmatch(s); m != nil {
		for _, b := range bold.FindAllStringSubmatch(m[1], -1) {
			src := Source{Name: clean(b[1])}
			src.Zone = strings.Trim(clean(strings.TrimLeft(strings.TrimSpace(b[2]), ",")), " ,.")
			if src.Name != "" {
				it.Sources = append(it.Sources, src)
			}
		}
	}
	return it, build, nil
}

func clean(s string) string {
	return strings.TrimSpace(html.UnescapeString(tags.ReplaceAllString(s, "")))
}
