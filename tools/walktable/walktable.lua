-- Walks between the places routes go to, measured with Shortest Path Forever's own path finder on its navigation
-- maps (github.com/cjber/shortest-path-forever, GPL-3.0-or-later), as the yards a player runs: up to SHORT yards
-- apart the path is drawn (string-pulled, as the addon draws it) and measured along the ground with its slopes, the
-- water swum at its own speed; further apart the finder's cost (its 8-yard grid steps) times GRID, the drawn path's
-- share of it measured on the routes' walks. A place stands at its height where one is known (Undercity under the
-- ruins, a mine under its hill); when the finder finds no way from there, the walk is tried on the ground.
--
-- luajit walktable.lua <spf repo> <unpacked nav dir> <places file> [shard] [shards] > walks.txt
-- Input: walk-places' output ("p i map x y z|-", "q i j1 j2 ..."). Output lines: "i j yards" ("-": no way).
local repo, data, file, shard, shards = arg[1], arg[2], arg[3], tonumber(arg[4] or 0), tonumber(arg[5] or 1)
local SHORT = 600
local GRID = 0.970 -- drawn path / grid cost beyond 600 yards: 0.968 (p10 0.946, p90 0.983), slopes 0.3 % more
local SWIM = 7 / 4.722 - 1 -- swimming at 4.72 yards a second instead of running at 7: extra yards per yard swum

local ns = {}
for _, f in ipairs({ "Routing/PathGrid.lua", "Routing/Path.lua" }) do
	assert(loadfile(repo .. "/" .. f))("ShortestPathForever", ns)
end
local Path = ns.Path
local loaded = {}
local function load(map)
	if not loaded[map] then
		assert(loadfile(repo .. "/Nav/Nav" .. map .. ".lua"))()
		assert(loadfile(data .. "/nav" .. map .. "_data.lua"))()
		loaded[map] = true
	end
end

local P = {}
local jobs = {}
for line in io.lines(file) do
	local t = line:sub(1, 1)
	if t == "p" then
		local i, m, x, y, z = line:match("^p (%d+) (%d+) (%S+) (%S+) (%S+)")
		P[tonumber(i)] = { map = tonumber(m), x = tonumber(x), y = tonumber(y), z = tonumber(z) }
	elseif t == "q" then
		local i = tonumber(line:match("^q (%d+)"))
		if i % shards == shard then
			local js = {}
			for j in line:gmatch(" (%d+)") do
				js[#js + 1] = tonumber(j)
			end
			table.remove(js, 1) -- the first number is i
			jobs[#jobs + 1] = { i = i, js = js }
		end
	end
end

local function flat(p)
	return { map = p.map, x = p.x, y = p.y }
end

-- the drawn path's yards, or nil
local function drawn(m, a, b)
	local pts = Path.FindSync(m, a, b)
	if type(pts) ~= "table" then
		return nil
	end
	local L = 0
	for k = 2, #pts do
		local dx, dy = pts[k].x - pts[k - 1].x, pts[k].y - pts[k - 1].y
		local dz = (pts[k].z and pts[k - 1].z) and (pts[k].z - pts[k - 1].z) or 0
		L = L + math.sqrt(dx * dx + dy * dy + dz * dz)
	end
	return L + (pts.wet or 0) * SWIM
end

local out = io.stdout
for n, job in ipairs(jobs) do
	local a = P[job.i]
	load(a.map)
	local far = {}
	for _, j in ipairs(job.js) do
		local b = P[j]
		local d = math.sqrt((a.x - b.x) ^ 2 + (a.y - b.y) ^ 2)
		if d <= SHORT then
			local y = drawn(a.map, a, b)
			if not y and (a.z or b.z) then
				y = drawn(a.map, flat(a), flat(b))
			end
			out:write(("%d %d %s\n"):format(job.i, j, y and ("%.0f"):format(y) or "-"))
		else
			far[#far + 1] = j
		end
	end
	if #far > 0 then
		local targets = {}
		for k, j in ipairs(far) do
			targets[k] = P[j]
		end
		local costs = Path.FindManySync(a.map, a, targets) or {}
		for k, j in ipairs(far) do
			local c = costs[k]
			if not c and (a.z or P[j].z) then
				local again = Path.FindManySync(a.map, flat(a), { flat(P[j]) })
				c = again and again[1]
			end
			out:write(("%d %d %s\n"):format(job.i, j, c and ("%.0f"):format(c * GRID) or "-"))
		end
	end
	if n % 20 == 0 then
		io.stderr:write(("shard %d: %d/%d\n"):format(shard, n, #jobs))
		out:flush()
	end
end
