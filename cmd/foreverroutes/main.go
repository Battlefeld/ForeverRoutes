// Command foreverroutes builds the route dataset and plans, replays and checks quest routes offline.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "build-data":
		err = cmdBuildData(os.Args[2:])
	case "audit":
		err = cmdAudit(os.Args[2:])
	case "fetch-warcraftdb":
		err = cmdFetchWarcraftDB(os.Args[2:])
	case "plan":
		err = cmdPlan(os.Args[2:])
	case "replay":
		err = cmdReplay(os.Args[2:])
	case "guide":
		err = cmdGuide(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "foreverroutes:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `foreverroutes <command> [flags]

  build-data   build data/routes-data.json.gz from the merged quest database and the cmangos dump
  audit        what the dataset knows and what is missing
  fetch-warcraftdb  fetch missing quest facts (objectives, XP, positions) from WarcraftDB Forever
  plan         plan a route (-race Orc -class Warlock -target 30)
  replay       check a planned route step by step
  guide        walk a quest guide (list of accept/complete/turnin/level events) with the same model`)
}

func newFlags(name string) *flag.FlagSet { return flag.NewFlagSet(name, flag.ExitOnError) }
