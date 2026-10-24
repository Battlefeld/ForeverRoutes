-- Walking costs on a coarse grid, measured with Shortest Path Forever's own path finder on its navigation maps
-- (github.com/cjber/shortest-path-forever, GPL-3.0-or-later). For one map: every cell of CELL yards gets a walkable
-- point (its centre, else the first of four points around it that the finder can stand on), then the finder's cost
-- in running yards from that point to the points of the cells around it (the 8 neighbours and the 8 knight's moves).
-- ForeverRoutes chains these to price a walk anywhere (route/walkgrid.go).
--
-- luajit walkgrid.lua <spf repo> <unpacked nav dir> <map> [first row] [rows] > cells.txt
-- Output lines: "c i j x y" (a cell and its point), "e i j di dj cost" (a cost to the cell at i+di, j+dj).
local repo, data, map, first, rows = arg[1], arg[2], tonumber(arg[3]), tonumber(arg[4] or 0), tonumber(arg[5] or 1e9)
local CELL = 200
local ns = {}
for _, f in ipairs({ "Routing/PathGrid.lua", "Routing/Path.lua" }) do
	assert(loadfile(repo .. "/" .. f))("ShortestPathForever", ns)
end
assert(loadfile(repo .. "/Nav/Nav" .. map .. ".lua"))()
assert(loadfile(data .. "/nav" .. map .. "_data.lua"))()
local D = ShortestPathForeverPathData[map]
local Path = ns.Path
local T = 1600 / 3
local x0, y0 = D.cx0 * T, D.cy0 * T
local ni, nj = math.ceil(D.nx * T / CELL), math.ceil(D.ny * T / CELL)

local function standable(p)
	local c = Path.FindManySync(map, p, { { map = map, x = p.x + 8, y = p.y } })
	return c and c[1] and true or false
end

local rep = {}
local function point(i, j)
	local key = i * 100000 + j
	if rep[key] ~= nil then
		return rep[key] or nil
	end
	local cx, cy = x0 + (i + 0.5) * CELL, y0 + (j + 0.5) * CELL
	local found = false
	for _, o in ipairs({ { 0, 0 }, { 50, 50 }, { -50, 50 }, { 50, -50 }, { -50, -50 } }) do
		local p = { map = map, x = cx + o[1], y = cy + o[2] }
		if standable(p) then
			found = p
			break
		end
	end
	rep[key] = found
	return found or nil
end

local moves = {}
for di = -2, 2 do
	for dj = -2, 2 do
		local a, b = math.abs(di), math.abs(dj)
		if (a <= 1 and b <= 1 and (a + b) > 0) or (a + b == 3) then
			moves[#moves + 1] = { di, dj }
		end
	end
end

for i = first, math.min(ni - 1, first + rows - 1) do
	for j = 0, nj - 1 do
		local p = point(i, j)
		if p then
			print(("c %d %d %.1f %.1f"):format(i, j, p.x, p.y))
			local targets, which = {}, {}
			for _, m in ipairs(moves) do
				local q = (i + m[1] >= 0 and j + m[2] >= 0 and i + m[1] < ni and j + m[2] < nj) and point(i + m[1], j + m[2])
				if q then
					targets[#targets + 1] = q
					which[#which + 1] = m
				end
			end
			if #targets > 0 then
				local c = Path.FindManySync(map, p, targets)
				for k, m in ipairs(which) do
					if c and c[k] then
						print(("e %d %d %d %d %.0f"):format(i, j, m[1], m[2], c[k]))
					end
				end
			end
		end
	end
	io.stderr:write(("row %d/%d\n"):format(i, ni))
end
print(("g %d %.3f %.3f %d %d %d"):format(map, x0, y0, CELL, ni, nj))
