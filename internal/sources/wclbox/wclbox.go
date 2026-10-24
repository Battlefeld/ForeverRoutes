// Package wclbox reads where an item comes from out of WCLBox's WoW Forever database
// (https://wowforever.wclbox.com, "新手盒子·魔兽世界无限数据库"). Its item pages say "Source: <creature>, <zone>" taken from
// the game client's item records and quest data. Its quest pages (data from Wowhead's Forever database) name, per item
// the quest asks for, what it comes from (a creature, an object, a vendor) or that the giver provides it, and mark
// where the objectives are on the zone's map. Only the items a quest needs and no other source explains are
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

// QuestItem is an item a quest page lists: what it comes from ("fallen log", "Avala", "Vendor: Boyle"; empty when the
// page says nothing) and whether the giver provides it.
type QuestItem struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	From     []string `json:"from,omitempty"`
	Provided bool     `json:"provided,omitempty"`
}

// Point is a place on an in-game map in percent.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Quest is what a quest page says about the objectives.
type Quest struct {
	ID    int         `json:"id"`
	Items []QuestItem `json:"items,omitempty"`
	UIMap int         `json:"ui,omitempty"`    // the map the points are on
	Where []Point     `json:"where,omitempty"` // where the objectives are
	URL   string      `json:"url"`
}

// Overlay is the file the build reads.
type Overlay struct {
	Schema  string        `json:"schema"`
	Source  string        `json:"source"`
	Fetched string        `json:"fetched"`
	Build   string        `json:"build,omitempty"` // Forever build the site showed
	Items   map[int]Item  `json:"items"`
	Quests  map[int]Quest `json:"quests,omitempty"`
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

func (c *Client) get(id int) ([]byte, error) { return c.page("item", "wupin", id) }

func (c *Client) page(kind, path string, id int) ([]byte, error) {
	cache := filepath.Join(c.Cache, kind+"_"+strconv.Itoa(id)+".html")
	if b, err := os.ReadFile(cache); err == nil {
		return b, nil
	}
	if wait := c.Delay - time.Since(c.last); !c.last.IsZero() && wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
	c.Calls++
	req, err := http.NewRequest(http.MethodGet, Base+"/en/"+path+"/"+strconv.Itoa(id), nil)
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
		return nil, fmt.Errorf("wclbox: HTTP %d for %s %d", resp.StatusCode, kind, id)
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

// Quest fetches and parses one quest page.
func (c *Client) Quest(id int) (Quest, error) {
	b, err := c.page("quest", "renwu", id)
	if err != nil {
		return Quest{}, err
	}
	return ParseQuest(id, b)
}

var (
	questHead = regexp.MustCompile(`<h2> Objectives </h2>`)
	questObjs = regexp.MustCompile(`(?s)<h2> Objectives </h2>(.*?)</section>`)
	provided  = regexp.MustCompile(`(?s)<p class="qd-provided">(.*?)</p>`)
	provItem  = regexp.MustCompile(`(?s)href="/en/wupin/(\d+)".*?<span class="qd-item-name[^"]*">(.*?)</span>`)
	notFound  = regexp.MustCompile(`We Could Not Find This Page`)
	needList  = regexp.MustCompile(`(?s)<ul class="qd-need">(.*?)</ul>`)
	needItem  = regexp.MustCompile(`(?s)<li>(.*?)</li>`)
	itemLink  = regexp.MustCompile(`href="/en/wupin/(\d+)"`)
	itemName  = regexp.MustCompile(`(?s)<span class="qd-item-name[^"]*">(.*?)</span>`)
	itemFrom  = regexp.MustCompile(`(?s)<span class="qd-from">(.*)$`)
	fromPart  = regexp.MustCompile(`(?s)<span>(.*?)</span>`)
	mapImage  = regexp.MustCompile(`/ditu/(\d+)\.webp`)
	mapDot    = regexp.MustCompile(`class="qd-dot" style="left:([\d.]+)%;top:([\d.]+)%"`)
	moreFrom  = regexp.MustCompile(`^(.*?)\s*and \d+ more$`)
)

// ParseQuest reads a quest page.
func ParseQuest(id int, page []byte) (Quest, error) {
	s := string(page)
	if notFound.MatchString(s) {
		return Quest{}, errNotFound
	}
	if !questHead.MatchString(s) {
		return Quest{}, fmt.Errorf("wclbox: quest %d: no objectives section, page layout changed?", id)
	}
	q := Quest{ID: id, URL: Base + "/en/renwu/" + strconv.Itoa(id)}
	sec := questObjs.FindStringSubmatch(s)
	if sec != nil {
		if m := provided.FindStringSubmatch(sec[1]); m != nil {
			for _, it := range provItem.FindAllStringSubmatch(m[1], -1) {
				id, _ := strconv.Atoi(it[1])
				q.Items = append(q.Items, QuestItem{ID: id, Name: clean(it[2]), Provided: true})
			}
		}
		if ul := needList.FindStringSubmatch(sec[1]); ul != nil {
			for _, li := range needItem.FindAllStringSubmatch(ul[1], -1) {
				it := QuestItem{}
				if m := itemLink.FindStringSubmatch(li[1]); m != nil {
					it.ID, _ = strconv.Atoi(m[1])
				}
				if m := itemName.FindStringSubmatch(li[1]); m != nil {
					it.Name = clean(m[1])
				}
				if m := itemFrom.FindStringSubmatch(li[1]); m != nil {
					for _, part := range fromPart.FindAllStringSubmatch(m[1], -1) {
						text := clean(part[1])
						if mm := moreFrom.FindStringSubmatch(text); mm != nil {
							text = mm[1]
						}
						for _, n := range strings.Split(text, ",") {
							if n = strings.Trim(strings.TrimSpace(n), "."); n != "" && !moreFrom.MatchString(n) {
								it.From = append(it.From, n)
							}
						}
					}
				}
				if it.ID != 0 {
					q.Items = append(q.Items, it)
				}
			}
		}
	}
	if m := mapImage.FindStringSubmatch(s); m != nil {
		q.UIMap, _ = strconv.Atoi(m[1])
	}
	for _, d := range mapDot.FindAllStringSubmatch(s, -1) {
		x, _ := strconv.ParseFloat(d[1], 64)
		y, _ := strconv.ParseFloat(d[2], 64)
		q.Where = append(q.Where, Point{X: x, Y: y})
	}
	return q, nil
}

func clean(s string) string {
	return strings.TrimSpace(html.UnescapeString(tags.ReplaceAllString(s, "")))
}
