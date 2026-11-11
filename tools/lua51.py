"""Minimal lua5.1 stand-in: lua51.py script.lua [args...]  /  lua51.py -p file.lua (syntax check)."""
import sys
from lupa import lua51

rt = lua51.LuaRuntime(unpack_returned_tuples=True, register_eval=False, register_builtins=False)
if sys.argv[1] == "-p":
    bad = 0
    for f in sys.argv[2:]:
        res = rt.eval("function(p) local f, e = loadfile(p); return e end")(f)
        if res:
            print(res)
            bad = 1
    sys.exit(bad)
script, args = sys.argv[1], sys.argv[2:]
rt.execute("arg = {}")
g = rt.globals()
g.arg[0] = script
for i, a in enumerate(args, 1):
    g.arg[i] = a
run = rt.eval(
    "function(p, ...) local f, e = loadfile(p); if not f then return e end; "
    "local ok, err = xpcall(function(...) return f(...) end, debug.traceback, ...); "
    "if ok then return nil end; return tostring(err) end"
)
try:
    err = run(script, *args)
except SystemExit:
    raise
except Exception as e:  # os.exit inside Lua ends the process itself
    err = str(e)
if err:
    print(err, file=sys.stderr)
    sys.exit(1)
