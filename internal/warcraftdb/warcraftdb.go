// Package warcraftdb reads quest facts of WoW Forever from WarcraftDB (https://forever.warcraftdb.com), which shows the
// game client's own quest tables: objectives with their amounts, XP, levels and where a quest starts and ends. It fills
// what the merged database lacks for quests Forever added. Requests are slow on purpose (one every 1.5 s), cached on
// disk, and only made for quests that need them.
package warcraftdb

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"
)

// Base is the Forever site.
const Base = "https://forever.warcraftdb.com"

// Objective as the client stores it. Type: 0 creature, 1 item, 2 object, 3 talk to, 6/7 reputation, 10 area trigger.
type Objective struct {
	Type        int    `json:"type"`
	ID          int    `json:"id"`
	Amount      int    `json:"amount"`
	Description string `json:"desc,omitempty"`
}

// Place is a position on a game map as fractions of its width and height.
type Place struct {
	UIMap int     `json:"uiMap"`
	U     float64 `json:"u"`
	V     float64 `json:"v"`
	Label string  `json:"label,omitempty"`
}

// Quest is what the site knows about one quest.
type Quest struct {
	ID         int         `json:"id"`
	Name       string      `json:"name"`
	Level      int         `json:"level"`
	MinLevel   int         `json:"min"`
	XP         int         `json:"xp"`
	StartNPC   []int       `json:"startNpc,omitempty"`
	EndNPC     []int       `json:"endNpc,omitempty"`
	Starts     []Place     `json:"starts,omitempty"`
	Ends       []Place     `json:"ends,omitempty"`
	Objectives []Objective `json:"objectives"`
	URL        string      `json:"url"`
}

// Overlay is the file the builder reads.
type Overlay struct {
	Schema  string        `json:"schema"`
	Source  string        `json:"source"`
	Fetched string        `json:"fetched"`
	Quests  map[int]Quest `json:"quests"`
	Missing []int         `json:"missing,omitempty"` // quests the site does not have
}

// Client fetches with a delay between requests and a disk cache.
type Client struct {
	HTTP  *http.Client
	Cache string
	Delay time.Duration
	last  time.Time
	Calls int
}

func (c *Client) get(path string) ([]byte, error) {
	cache := filepath.Join(c.Cache, regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(path, "_")+".json")
	if b, err := os.ReadFile(cache); err == nil {
		return b, nil
	}
	if wait := c.Delay - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
	c.Calls++
	req, _ := http.NewRequest("GET", Base+path, nil)
	req.Header.Set("User-Agent", "ForeverRoutes/1.0 (route planning research; https://github.com/Battlefeld/ForeverRoutes)")
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil, errNotFound
	}
	if resp.StatusCode == 429 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("warcraftdb: HTTP %d, stopping", resp.StatusCode)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("warcraftdb: HTTP %d for %s", resp.StatusCode, path)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	os.MkdirAll(c.Cache, 0o755)
	os.WriteFile(cache, b, 0o644)
	return b, nil
}

var errNotFound = fmt.Errorf("not found")

// IsNotFound tells whether the site does not know the quest.
func IsNotFound(err error) bool { return err == errNotFound }

var npcLink = regexp.MustCompile(`data-tooltip="npc:(\d+)"`)

// Quest fetches and parses one quest.
func (c *Client) Quest(id int) (Quest, error) {
	b, err := c.get("/api/data/node/quest/" + strconv.Itoa(id))
	if err != nil {
		return Quest{}, err
	}
	return Parse(id, b)
}

// Parse reads the site's quest JSON.
func Parse(id int, b []byte) (Quest, error) {
	var raw struct {
		Data struct {
			Name     string `json:"name"`
			Level    any    `json:"level"`
			MinLevel any    `json:"min_level"`
			Starts   string `json:"starts"`
			Ends     string `json:"ends"`
			Rewards  struct {
				XP any `json:"xp"`
			} `json:"rewards"`
			Objectives []struct {
				Description string `json:"description"`
				Amount      any    `json:"amount"`
				Type        any    `json:"type"`
				ObjectID    any    `json:"object_id"`
			} `json:"objectives"`
		} `json:"data"`
		Extra struct {
			QuestMap struct {
				Starts  []place `json:"starts"`
				TurnIns []place `json:"turn_ins"`
			} `json:"quest_map"`
		} `json:"extra"`
	}
	if err := json.Unmarshal(b, &raw); err != nil {
		return Quest{}, fmt.Errorf("quest %d: %w", id, err)
	}
	q := Quest{ID: id, Name: raw.Data.Name, Level: toInt(raw.Data.Level), MinLevel: toInt(raw.Data.MinLevel), XP: toInt(raw.Data.Rewards.XP),
		URL: fmt.Sprintf("%s/data/quest/%d", Base, id), Objectives: []Objective{}}
	for _, m := range npcLink.FindAllStringSubmatch(raw.Data.Starts, -1) {
		n, _ := strconv.Atoi(m[1])
		q.StartNPC = append(q.StartNPC, n)
	}
	for _, m := range npcLink.FindAllStringSubmatch(raw.Data.Ends, -1) {
		n, _ := strconv.Atoi(m[1])
		q.EndNPC = append(q.EndNPC, n)
	}
	for _, o := range raw.Data.Objectives {
		q.Objectives = append(q.Objectives, Objective{Type: toInt(o.Type), ID: toInt(o.ObjectID), Amount: toInt(o.Amount), Description: o.Description})
	}
	for _, p := range raw.Extra.QuestMap.Starts {
		q.Starts = append(q.Starts, p.place())
	}
	for _, p := range raw.Extra.QuestMap.TurnIns {
		q.Ends = append(q.Ends, p.place())
	}
	return q, nil
}

type place struct {
	UIMap any     `json:"ui_map_id"`
	U     float64 `json:"u"`
	V     float64 `json:"v"`
	Label string  `json:"label"`
}

func (p place) place() Place { return Place{UIMap: toInt(p.UIMap), U: p.U, V: p.V, Label: p.Label} }

func toInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(x)
		return n
	}
	return 0
}

// Load reads an overlay file.
func Load(path string) (*Overlay, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var o Overlay
	if err := json.Unmarshal(b, &o); err != nil {
		return nil, err
	}
	return &o, nil
}
