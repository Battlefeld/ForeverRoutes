package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/internal/build"
)

func cmdBuildData(args []string) error {
	fs := newFlags("build-data")
	merged := fs.String("merged", "data/sources/merged-quests-1-30.json.gz", "merged quest database (plain or gzip JSON)")
	mergedRev := fs.String("merged-revision", "7d66f93074d538a9f9e056c4ee712d176219d1f1", "revision of the merged database")
	cm := fs.String("cmangos", "", "cmangos classic-db dump (Full_DB/*.sql.gz)")
	cmRev := fs.String("cmangos-revision", "", "cmangos classic-db commit")
	wdb := fs.String("warcraftdb", "data/sources/warcraftdb-forever.json", "WarcraftDB Forever overlay (fetch-warcraftdb); empty = none")
	out := fs.String("out", "data/routes-data.json.gz", "output file")
	fs.Parse(args)
	if *cm == "" {
		return fmt.Errorf("-cmangos is required (Full_DB/ClassicDB_*.sql.gz of cmangos/classic-db)")
	}
	d, err := build.Build(build.Inputs{Merged: *merged, MergedRevision: *mergedRev, CMangos: *cm, CMangosRevision: *cmRev, WarcraftDB: *wdb})
	if err != nil {
		return err
	}
	if err := dataset.Write(*out, d); err != nil {
		return err
	}
	return printJSON(d.Stats)
}

func cmdAudit(args []string) error {
	fs := newFlags("audit")
	path := fs.String("data", "data/routes-data.json.gz", "dataset")
	fs.Parse(args)
	d, err := dataset.Load(*path)
	if err != nil {
		return err
	}
	reasons := map[string]int{}
	for _, q := range d.Quests {
		if q.Unsupported != "" {
			key := q.Unsupported
			if i := indexColon(key); i > 0 {
				key = key[i+2:]
			}
			reasons[key]++
		}
	}
	keys := make([]string, 0, len(reasons))
	for k := range reasons {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return printJSON(map[string]any{"build": d.Build, "stats": d.Stats, "unsupported": reasons, "sources": d.Sources})
}

func indexColon(s string) int {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == ':' && s[i+1] == ' ' {
			return i
		}
	}
	return -1
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
