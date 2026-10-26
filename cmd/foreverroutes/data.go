package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/internal/update"
)

// cmdBuildData rebuilds the dataset exactly from the pinned sources (data/sources/lock.json).
func cmdBuildData(args []string) error {
	fs := newFlags("build-data")
	out := fs.String("out", update.DatasetFile, "output file")
	cache := fs.String("cache", ".cache", "download cache")
	fs.Parse(args)
	l, err := update.ReadLock(".")
	if err != nil {
		return err
	}
	snap, err := update.ReadSnapshot(update.SnapshotFile)
	if err != nil {
		return err
	}
	d, _, err := update.Rebuild(context.Background(), update.Options{Cache: *cache}, l, update.Inputs{Snapshot: snap, Overlay: update.OverlayFile, WCLBox: update.WCLBoxFile, FChanges: update.FChangesFile, Deathskips: update.DeathskipsFile, Dungeons: update.DungeonsFile, QuestFixes: update.QuestFixesFile})
	if err != nil {
		return err
	}
	if err := dataset.Write(*out, d); err != nil {
		return err
	}
	return printJSON(d.Stats)
}

// cmdUpdate checks every source for a new state and rebuilds the dataset when one changed.
func cmdUpdate(args []string) error {
	fs := newFlags("update")
	force := fs.Bool("force", false, "rebuild even when no source changed")
	require := fs.Bool("require-foreverdb", false, "fail when FOREVERDB_API_KEY is not set")
	budget := fs.Int("warcraftdb-budget", 150, "WarcraftDB requests per run")
	report := fs.String("report", "", "write the report (Markdown) here")
	cache := fs.String("cache", ".cache", "download cache")
	fs.Parse(args)
	rep, err := update.Run(context.Background(), update.Options{Cache: *cache, ForeverDBKey: os.Getenv("FOREVERDB_API_KEY"),
		RequireForeverDB: *require, WarcraftDBBudget: *budget, Force: *force,
		Log: func(f string, a ...any) { fmt.Fprintf(os.Stderr, f+"\n", a...) }})
	if err != nil {
		return err
	}
	md := rep.Markdown()
	if *report != "" {
		if err := os.WriteFile(*report, []byte(md), 0o644); err != nil {
			return err
		}
	}
	switch {
	case len(rep.Problems) > 0:
		fmt.Fprint(os.Stderr, md)
		return fmt.Errorf("checks failed, nothing written")
	case rep.Changed:
		fmt.Fprint(os.Stderr, "dataset updated\n"+md)
	case len(rep.Sources) > 0:
		fmt.Fprint(os.Stderr, "sources changed, the dataset stayed the same\n"+md)
	default:
		fmt.Fprintln(os.Stderr, "nothing new")
	}
	return nil
}

func cmdAudit(args []string) error {
	fs := newFlags("audit")
	path := fs.String("data", update.DatasetFile, "dataset")
	fs.Parse(args)
	d, err := dataset.Load(*path)
	if err != nil {
		return err
	}
	unsupported := map[string]int{}
	for _, q := range d.Quests {
		if q.Unsupported != "" {
			unsupported[reason(q.Unsupported)]++
		}
	}
	return printJSON(map[string]any{"build": d.Build, "sources": d.Sources, "stats": d.Stats, "unsupported": unsupported})
}

// reason drops the quest-specific part ("item 3713: ") of an unsupported note.
func reason(s string) string {
	for i := 0; i+1 < len(s); i++ {
		if s[i] == ':' && s[i+1] == ' ' {
			return s[i+2:]
		}
	}
	return s
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
