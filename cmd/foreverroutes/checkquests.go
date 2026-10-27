package main

import (
	"encoding/json"
	"os"

	"github.com/Battlefeld/ForeverRoutes/dataset"
	"github.com/Battlefeld/ForeverRoutes/route"
)

func cmdCheckQuests(args []string) error {
	fs := newFlags("check-quests")
	path := fs.String("data", "data/routes-data.json.gz", "dataset")
	target := fs.Int("target", 20, "inspect every quest with minimum level at most this")
	out := fs.String("out", "", "write the complete JSON audit")
	fs.Parse(args)
	d, err := dataset.Load(*path)
	if err != nil {
		return err
	}
	report, err := route.AuditQuests(d, *target)
	if err != nil {
		return err
	}
	if *out == "" {
		return printJSON(report)
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(*out, append(b, '\n'), 0644)
}
