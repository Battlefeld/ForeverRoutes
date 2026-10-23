package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/internal/warcraftdb"
)

// needsFacts lists the quests of a dataset whose objectives, XP or positions are not known.
func needsFacts(d *dataset.Dataset) []int {
	spawned := map[int]bool{}
	for _, n := range d.NPCs {
		if len(n.Spawns) > 0 {
			spawned[n.ID] = true
		}
	}
	located := func(refs []dataset.Ref, at *dataset.Point) bool {
		if at != nil {
			return true
		}
		for _, r := range refs {
			if r.Kind == "npc" && spawned[r.ID] {
				return true
			}
		}
		return false
	}
	var ids []int
	for _, q := range d.Quests {
		u := q.Unsupported
		if strings.Contains(u, "unknown") || q.Sources["xp"] == "unknown" || !located(q.Start, q.StartAt) || !located(q.End, q.EndAt) {
			ids = append(ids, q.ID)
		}
	}
	sort.Ints(ids)
	return ids
}

func cmdFetchWarcraftDB(args []string) error {
	fs := newFlags("fetch-warcraftdb")
	path := fs.String("data", "data/routes-data.json.gz", "dataset whose gaps are fetched")
	out := fs.String("out", "data/sources/warcraftdb-forever.json", "overlay file")
	cache := fs.String("cache", ".cache/warcraftdb", "cache directory")
	limit := fs.Int("limit", 0, "fetch at most this many quests (0 = all)")
	fs.Parse(args)
	d, err := dataset.Load(*path)
	if err != nil {
		return err
	}
	ov := &warcraftdb.Overlay{Schema: "foreverroutes.warcraftdb.v1", Quests: map[int]warcraftdb.Quest{},
		Source: "WarcraftDB Forever (https://forever.warcraftdb.com), game client quest data"}
	if old, err := warcraftdb.Load(*out); err == nil {
		ov.Quests, ov.Missing = old.Quests, old.Missing
	}
	missing := map[int]bool{}
	for _, id := range ov.Missing {
		missing[id] = true
	}
	c := &warcraftdb.Client{HTTP: &http.Client{Timeout: 30 * time.Second}, Cache: *cache, Delay: 1500 * time.Millisecond}
	ids := needsFacts(d)
	done := 0
	for _, id := range ids {
		if _, have := ov.Quests[id]; have || missing[id] {
			continue
		}
		if *limit > 0 && done >= *limit {
			break
		}
		q, err := c.Quest(id)
		done++
		if warcraftdb.IsNotFound(err) {
			missing[id] = true
			continue
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "stopping:", err)
			break
		}
		ov.Quests[id] = q
	}
	ov.Missing = ov.Missing[:0]
	for id := range missing {
		ov.Missing = append(ov.Missing, id)
	}
	sort.Ints(ov.Missing)
	ov.Fetched = time.Now().UTC().Format("2006-01-02")
	os.MkdirAll("data/sources", 0o755)
	b, _ := json.MarshalIndent(ov, "", " ")
	if err := os.WriteFile(*out, b, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "%d quests needed facts, %d known now, %d not on the site, %d requests\n", len(ids), len(ov.Quests), len(ov.Missing), c.Calls)
	return nil
}
