// Package foreverdb reads quest data from the ForeverDB API (https://foreverdb.net/developers).
//
// Data from ForeverDB.net (https://foreverdb.net): every place that shows this data must credit ForeverDB.net with a
// visible link. The API key comes only from the environment (FOREVERDB_API_KEY) and is never written anywhere.
package foreverdb

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Base is the API root.
const Base = "https://foreverdb.net/api/v1"

// Credit is the attribution ForeverDB asks for.
var Credit = map[string]string{"label": "Data from ForeverDB.net", "url": "https://foreverdb.net", "api_docs": "https://foreverdb.net/developers"}

// ErrQuota means the local call budget or the API's rate limit stopped the run.
var ErrQuota = errors.New("foreverdb: quota reached")

// Client calls the API politely: at most MaxCalls per run, at least Interval apart.
type Client struct {
	Key      string
	HTTP     *http.Client
	Base     string
	Interval time.Duration // 2.1 s (the API allows 60 per minute)
	MaxCalls int           // 380 per run (the API allows 10,000 per day)
	Calls    int
	last     time.Time
}

// NewClient makes a client with the usual limits.
func NewClient(key string) (*Client, error) {
	if strings.TrimSpace(key) == "" {
		return nil, errors.New("foreverdb: FOREVERDB_API_KEY is not set")
	}
	return &Client{Key: strings.TrimSpace(key), HTTP: &http.Client{Timeout: 35 * time.Second}, Base: Base,
		Interval: 2100 * time.Millisecond, MaxCalls: 380}, nil
}

// answer is the envelope of every response.
type answer struct {
	Data json.RawMessage `json:"data"`
	Meta json.RawMessage `json:"meta"`
}

var errTransient = errors.New("foreverdb: temporary server failure")

func (c *Client) get(ctx context.Context, endpoint string, params url.Values) (*answer, error) {
	for attempt := 1; ; attempt++ {
		a, err := c.getOnce(ctx, endpoint, params)
		if err == nil {
			return a, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		var network net.Error
		transient := errors.As(err, &network) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) || errors.Is(err, errTransient)
		if !transient || attempt == 3 {
			return nil, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt) * time.Second):
		}
	}
}

// Every retry goes through the same interval and call budget as a normal request.
func (c *Client) getOnce(ctx context.Context, endpoint string, params url.Values) (*answer, error) {
	switch endpoint {
	case "meta", "quests", "changes-seen":
	default:
		return nil, fmt.Errorf("foreverdb: endpoint %q not allowed", endpoint)
	}
	if c.Calls >= c.MaxCalls {
		return nil, ErrQuota
	}
	if wait := c.Interval - time.Since(c.last); !c.last.IsZero() && wait > 0 {
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	c.last = time.Now()
	c.Calls++ // failed calls count too
	u := c.Base + "/" + endpoint
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Key", c.Key)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ForeverRoutes (https://github.com/Battlefeld/ForeverRoutes)")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("foreverdb %s: %w", endpoint, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, (8<<20)+1))
	if err != nil {
		return nil, fmt.Errorf("foreverdb %s: reading response: %w", endpoint, err)
	}
	if len(body) > 8<<20 {
		return nil, fmt.Errorf("foreverdb %s: response exceeds 8 MiB", endpoint)
	}
	switch {
	case resp.StatusCode >= 500 && resp.StatusCode <= 599 || resp.StatusCode == http.StatusRequestTimeout:
		return nil, fmt.Errorf("%w: %s HTTP %d", errTransient, endpoint, resp.StatusCode)
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, fmt.Errorf("%w: HTTP 429, Retry-After %s", ErrQuota, resp.Header.Get("Retry-After"))
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return nil, fmt.Errorf("foreverdb: access refused (HTTP %d): check the key", resp.StatusCode)
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("foreverdb %s: HTTP %d", endpoint, resp.StatusCode)
	}
	// never keep the key should a server echo it
	body = []byte(strings.ReplaceAll(string(body), c.Key, "[REDACTED]"))
	var a answer
	if err := json.Unmarshal(body, &a); err != nil || len(a.Data) == 0 {
		return nil, fmt.Errorf("foreverdb %s: unexpected response", endpoint)
	}
	return &a, nil
}

// pages reads a paged list, deriving offsets itself (never following URLs from the response).
func (c *Client) pages(ctx context.Context, endpoint, label string, params url.Values) ([]json.RawMessage, error) {
	var all []json.RawMessage
	offset := 0
	for page := 0; page < 120; page++ {
		p := url.Values{}
		for k, v := range params {
			p[k] = v
		}
		p.Set("limit", "100")
		p.Set("offset", strconv.Itoa(offset))
		a, err := c.get(ctx, endpoint, p)
		if err != nil {
			return nil, err
		}
		var batch []json.RawMessage
		if err := json.Unmarshal(a.Data, &batch); err != nil {
			var wrapped map[string]json.RawMessage
			if json.Unmarshal(a.Data, &wrapped) != nil {
				return nil, fmt.Errorf("foreverdb %s: unknown list shape", endpoint)
			}
			found := false
			for _, k := range []string{label, "items", "results", "rows"} {
				if raw, ok := wrapped[k]; ok && json.Unmarshal(raw, &batch) == nil {
					found = true
					break
				}
			}
			if !found {
				return nil, fmt.Errorf("foreverdb %s: unknown list shape", endpoint)
			}
		}
		all = append(all, batch...)
		var meta struct {
			Total *int `json:"total"`
		}
		_ = json.Unmarshal(a.Meta, &meta)
		if len(batch) == 0 || (meta.Total != nil && offset+len(batch) >= *meta.Total) || (meta.Total == nil && len(batch) < 100) {
			return all, nil
		}
		offset += len(batch)
	}
	return nil, fmt.Errorf("foreverdb %s: more than 120 pages, refusing an incomplete list", endpoint)
}

// Snapshot is what one run reads: game metadata, the quest list of levels 1-30 and the changes seen in game.
type Snapshot struct {
	Schema   string            `json:"schema"`
	Source   map[string]string `json:"source"`
	Game     json.RawMessage   `json:"game_metadata"`
	GameMeta json.RawMessage   `json:"game_build_metadata"`
	Quests   []json.RawMessage `json:"quests"`
	Changes  []json.RawMessage `json:"quest_changes_seen"`
}

// Fetch reads a snapshot. The quest details endpoint is not used: it returns the same fields as the list.
func (c *Client) Fetch(ctx context.Context) (*Snapshot, error) {
	meta, err := c.get(ctx, "meta", nil)
	if err != nil {
		return nil, err
	}
	quests, err := c.pages(ctx, "quests", "quests", url.Values{"minLevel": {"1"}, "maxLevel": {"30"}})
	if err != nil {
		return nil, err
	}
	changes, err := c.pages(ctx, "changes-seen", "changes", url.Values{"kind": {"quest"}})
	if err != nil {
		return nil, err
	}
	return &Snapshot{Schema: "foreverroutes.foreverdb_snapshot.v1", Source: Credit, Game: meta.Data, GameMeta: meta.Meta,
		Quests: quests, Changes: changes}, nil
}

// Quest is the part of a list row the merge uses.
type Quest struct {
	ID            int     `json:"id"`
	Name          *string `json:"name"`
	Level         *int    `json:"level"`
	RequiredLevel *int    `json:"requiredLevel"`
	Side          string  `json:"side"`
	Zone          *struct {
		Name  *string `json:"n"`
		UIMap *int    `json:"m"`
	} `json:"zone"`
	Known  *bool  `json:"knownInForever"`
	Source string `json:"source"` // forever or classic
	Starts *Node  `json:"starts"`
	Ends   *Node  `json:"ends"`
	URL    string `json:"url"`
}

// Node is who starts or ends a quest, with where ForeverDB saw it.
type Node struct {
	Type     string `json:"type"` // npc, object, item
	ID       int    `json:"id"`
	Location *struct {
		UIMap  int      `json:"uiMapId"`
		X      *float64 `json:"x"`
		Y      *float64 `json:"y"`
		Source string   `json:"source"`
	} `json:"location"`
	Source string `json:"source"`
	Others []struct {
		Type string `json:"type"`
		ID   int    `json:"id"`
	} `json:"others"`
}

// Build is the game build the snapshot describes.
func (s *Snapshot) Build() string {
	var g struct {
		Build string `json:"build"`
	}
	_ = json.Unmarshal(s.Game, &g)
	return g.Build
}

// List decodes the quest rows; rows without a positive id or duplicates are an error.
func (s *Snapshot) List() ([]Quest, error) {
	out := make([]Quest, 0, len(s.Quests))
	seen := map[int]bool{}
	for _, raw := range s.Quests {
		var q Quest
		if err := json.Unmarshal(raw, &q); err != nil || q.ID <= 0 || seen[q.ID] {
			return nil, fmt.Errorf("foreverdb: malformed or duplicate quest row %s", string(raw[:min(len(raw), 80)]))
		}
		seen[q.ID] = true
		out = append(out, q)
	}
	return out, nil
}
