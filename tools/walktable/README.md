# Walk table

`data/walktable.json.gz` holds the walks between the places routes go to, measured with the path finder of
[Shortest Path Forever](https://github.com/cjber/shortest-path-forever) by Cillian Berragan (GPL-3.0-or-later) on
its navigation maps: an 8-yard grid with floors (a tunnel under a mountain, Undercity under the ruins), baked from
the WoW: Forever client with TrinityCore and Recast/Detour. The planner takes a walk between two places of the
table from it (`route/walktable.go`) and any other walk from the coarse walk grid (`tools/walkgrid`).

Redo it when the routes' places change much (new quests, new spawns) or Shortest Path Forever rebakes its maps
(`spf` in `data/sources/lock.json` names the commit; use the same one):

```
git clone https://github.com/cjber/shortest-path-forever spf && git -C spf checkout <commit>
python3 -I tools/walkgrid/unpack.py spf nav                    # inflate the packed map pieces once
go run ./cmd/foreverroutes walk-places -spawns <forever-db>/data/spawn.jsonl -out places.txt
for k in 0 1 2 3; do                                           # LuaJIT, one shard per core: about 2 hours on 4
  luajit tools/walktable/walktable.lua spf nav places.txt $k 4 > out/$k.txt &
done; wait
go run ./cmd/foreverroutes build-walktable -places places.txt -in out -spf <commit>
```

What is measured:

- **The places** (`walk-places`): every place a route of any race and class can walk from or to
  (`route.World.Places`): quest givers and enders, the sites of objectives and starting items, grinding sites,
  flight masters, docks and portals, graveyards and the guides' places to die, dungeon entrances, the trainers for
  wands and professions. One per 2 yards. Each gets a height from forever-db's spawns (the nearest within 4 yards,
  else the median of those within 30, so a site in a mine is not taken for the hill above it).
- **The pairs**: every two places of a map up to 3000 yards apart, except a walk into or out of a place only lifts
  reach (Thunder Bluff's mesa, Undercity): the planner keeps its lift time for those (`travel.lifts`).
- **The yards** (`walktable.lua`): up to 600 yards apart the path the addon draws (string-pulled), measured along
  the ground with its slopes, the water swum at 4.72 yards a second against 7 running; further apart the finder's
  cost times 0.970, the drawn path's share of it on the routes' walks (0.968, p10 0.946, p90 0.983; slopes 0.3 %).
  Where the finder finds no way from a place's height, the walk is tried on the ground.

Only Shortest Path Forever's path finder and maps are run; walktable.lua loads `Routing/PathGrid.lua`,
`Routing/Path.lua` and the `Nav` files, nothing else.
