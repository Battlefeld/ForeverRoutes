# Working on ForeverRoutes (for every coding agent: Codex, Claude, ...)

Several agents change this repository. These rules keep main green and the daily data update running.

## Before every push
- `gofmt -l .` empty, `go vet ./...`, and `go test -timeout 25m ./...` (all packages, also the slow ones:
  `cmd/foreverroutes` holds the live-planner tests `TestStages*` that use the shipped routes).
- `lua5.1 _dev/harness/routes_hud_test.lua` when anything under `ForeverRoutes/` changed.
- A test you did not write fails after your change: find out why. Loosen it only when it asks more than the code
  it guards does (say so in the commit), never skip or delete it.

## Planner model changes
- A change that alters what the offline planner computes (danger rules, kill times, XP, scores) makes the shipped
  routes stale. Touch `route/model-stamp.txt` (one dated line saying what changed) in the same commit: test.yml then
  starts the route recompute and release.yml waits for it instead of failing.
- The live planner (`route/live_stages.go`, `cmd/foreverroutes/next.go`) does not change shipped routes. Its
  behaviour is guarded by `TestStages*` against the shipped routes: after regenerating all routes, run those tests.
  `next` takes the stages from `stagesPreferred` (0.9) of the route's XP per hour; tests measure against the same.

## Data, workflows, releases
- `update.yml` recomputes and publishes routes (daily and on data fixes); `release.yml` publishes the addon bundle
  after pushes to main; ForeverCore (repository ForeverSim, workflow `forevercore-package.yml`) packages the addon.
- A failed workflow means something is wrong: a real bug, a test asking too much, or GitHub itself (5xx: re-run
  once). Do not leave main red; do not push to main while an update run is pushing routes (see its concurrency).

## Conventions
- Commit messages of the owner's sessions are German; code comments English. No model names in commits.
- The live HUD and planner texts the player sees are German.
