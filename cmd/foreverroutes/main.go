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
	case "build-walks":
		err = cmdBuildWalks(os.Args[2:])
	case "walk-places":
		err = cmdWalkPlaces(os.Args[2:])
	case "build-walktable":
		err = cmdBuildWalktable(os.Args[2:])
	case "audit":
		err = cmdAudit(os.Args[2:])
	case "update":
		err = cmdUpdate(os.Args[2:])
	case "plan":
		err = cmdPlan(os.Args[2:])
	case "replay":
		err = cmdReplay(os.Args[2:])
	case "improve":
		err = cmdImprove(os.Args[2:])
	case "next":
		err = cmdNext(os.Args[2:])
	case "guide":
		err = cmdGuide(os.Args[2:])
	case "bundle":
		err = cmdBundle(os.Args[2:])
	case "export-addon":
		err = cmdExportAddon(os.Args[2:])
	case "extend":
		err = cmdExtend(os.Args[2:])
	case "check-routes":
		err = cmdCheckRoutes(os.Args[2:])
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

  update       check every source for a new state; rebuild and check the dataset when one changed
  build-data   rebuild data/routes-data.json.gz exactly from the pinned sources (data/sources/lock.json)
  build-walks  turn tools/walkgrid's measured walking costs into data/walks.json.gz
  walk-places  the places of every route with their heights and the pairs to measure (tools/walktable)
  build-walktable  turn tools/walktable's measured walks into data/walktable.json.gz
  audit        what the dataset knows and what is missing
  plan         plan a route (-race Orc -class Warlock -target 30)
  replay       check a planned route step by step
  improve      a guide and our planned route, each improved by local edits; the faster wins
  next         the next stretch of route from the character's live state (-live state.json -horizon 3600)
  bundle       pack dataset, walk grid, routes, profiles (and the Windows planner) for ForeverCore's update
  extend       continue a route to a higher level (-route data/routes/orc-warrior.json -target 40 [-dungeons])
  export-addon write the routes as Lua data for the in-game addon
  check-routes replay every route file and report long steps, quests no guide does, assumed drops, deathskips
  guide        walk a quest guide (list of accept/complete/turnin/level events) with the same model`)
}

func newFlags(name string) *flag.FlagSet { return flag.NewFlagSet(name, flag.ExitOnError) }
