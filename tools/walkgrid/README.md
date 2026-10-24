# Walk grid

`data/walks.json.gz` holds walking costs measured on the navigation maps of
[Shortest Path Forever](https://github.com/cjber/shortest-path-forever) by Cillian Berragan (GPL-3.0-or-later),
with its own path finder. The planner chains them (`route/walkgrid.go`) instead of pricing a walk as a straight
line times 1.2: a walk around a mountain or a lake costs what it costs.

The maps are about 20 MB of compressed Lua and the finder is Lua, so the costs are measured once here, offline,
and the result is committed. Redo it when Shortest Path Forever rebakes its maps (`spf` in
`data/sources/lock.json` names the commit the travel data comes from; use the same one):

```
git clone https://github.com/cjber/shortest-path-forever spf && git -C spf checkout <commit>
python3 -I tools/walkgrid/unpack.py spf nav           # inflate the packed map pieces once
for m in 2991 0 1; do                                 # LuaJIT; about 15-40 minutes per continent
  luajit tools/walkgrid/walkgrid.lua spf nav $m > out/$m.txt
done                                                  # (rows can be split: walkgrid.lua spf nav 1 0 50, 1 50 50, ...)
go run ./cmd/foreverroutes build-walks -in out -spf <commit>
```

What is measured: a grid of 200-yard cells over each map (Eastern Kingdoms 0, Kalimdor 1, Zephras Isle 2991).
Each cell gets a point the finder can stand on (its centre, else one of four points around it); cells with none
are left out. From each point the finder's cost in running yards to the points of the 16 cells around it (the 8
neighbours and the knight's moves) is kept; water is swum at the finder's cost (three times walking). The planner
runs Dijkstra over these costs from the cell a walk starts in and scales the result to the walk's own length.

Only Shortest Path Forever's path finder and maps are run; walkgrid.lua loads `Routing/PathGrid.lua`,
`Routing/Path.lua` and the `Nav` files, nothing else.
