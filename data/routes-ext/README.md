# Routes continued past 20

Each file here continues the route of the same name in `data/routes` (levels 1-20) to a higher level, planned by
zones (`foreverroutes extend -stages`): the live planner's stages, four hours at a time, each zone worked off before
the next. The bundle and the addon take a file here in place of its base route.

Only the routes here are continued; when the data or a base route changes, the update workflow continues them again
from the new base (a minute each), the other routes stay at 20. To add one:

    go run ./cmd/foreverroutes extend -route data/routes/<key>.json -target 40 -stages 20s -budget 0 -out data/routes-ext/<key>.json
