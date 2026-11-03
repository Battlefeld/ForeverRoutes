#!/usr/bin/env bash
# Recompute data/profiles/<class>.json from the engine ForeverSim currently ships, when it differs from the one the
# profiles were made with. Needs Go, Python 3, protoc (with protoc-gen-go) and lua5.1. Prints "changed" or "unchanged".
# A failure leaves the profiles as they are (the caller decides whether that stops anything).
set -euo pipefail
root="$(cd "$(dirname "$0")/.." && pwd)"
work="${KILL_PROFILE_WORK:-$(mktemp -d)}"
want="$(curl -fsSL https://raw.githubusercontent.com/Battlefeld/ForeverSim/main/ForeverCore/engine.lock.json \
  | python3 -c 'import json,sys; print(json.load(sys.stdin)["buildIdentity"])')"
have="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["source"].rsplit("Engine ",1)[-1])' "$root/data/profiles/warrior.json")"
if [ "$want" = "$have" ]; then echo unchanged; exit 0; fi
echo "kill profiles: engine $have -> $want" >&2
git clone -q --depth 1 https://github.com/Battlefeld/ForeverSim "$work/ForeverSim"
got="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["buildIdentity"])' "$work/ForeverSim/ForeverCore/engine.lock.json")"
[ "$got" = "$want" ] || { echo "ForeverSim moved while cloning ($got)" >&2; exit 1; }
(cd "$work/ForeverSim/ForeverCore" \
  && python3 checkout_engine.py --destination "$work/engine" >&2 \
  && python3 build_engine.py --engine-source "$work/engine" --output "$work/ForeverEngine" >&2 \
  && go build -o "$work/forevercore" . )
python3 - "$work" "$want" <<'PY'
import json, sys
work, revision = sys.argv[1], sys.argv[2]
json.dump({'engine': f'{work}/ForeverEngine', 'engineRevision': revision, 'runtime': f'{work}/ForeverSim',
           'data': f'{work}/data', 'lua': '/usr/bin/lua5.1'}, open(f'{work}/config.json', 'w'))
# the races the frozen profiles use: the class's first race in the starting roster
races = {'druid': 'NightElf', 'shaman': 'Orc'}
for c in ['druid', 'hunter', 'mage', 'paladin', 'priest', 'rogue', 'shaman', 'warlock', 'warrior']:
    json.dump({'schema': 3, 'session': 'kp', 'sequence': 1, 'revision': '1', 'kind': 'kill-profile',
               'payload': {'class': c, 'race': races.get(c, 'Human'), 'fromLevel': 10, 'toLevel': 40, 'iterations': 300}},
              open(f'{work}/req-{c}.json', 'w'))
PY
pids=()
for c in druid hunter mage paladin priest rogue shaman warlock warrior; do
  "$work/forevercore" -config "$work/config.json" -request "$work/req-$c.json" -output "$work/out-$c.json" >"$work/log-$c.txt" 2>&1 &
  pids+=($!)
done
for p in "${pids[@]}"; do wait "$p"; done
python3 - "$work" "$root" "$want" <<'PY'
import json, sys
work, root, revision = sys.argv[1:]
profiles = {}
for c in ['druid', 'hunter', 'mage', 'paladin', 'priest', 'rogue', 'shaman', 'warlock', 'warrior']:
    out = json.load(open(f'{work}/out-{c}.json'))
    if out.get('status') != 'ok' or not out.get('result', {}).get('levels'):
        sys.exit(f'kill profile {c} failed: {out.get("error")}')
    if not out['result'].get('source', '').endswith(revision):
        sys.exit(f'kill profile {c} came from another engine: {out["result"].get("source")}')
    profiles[c] = out['result']
# all nine or none: a half-updated set would mix two engines
for c, result in profiles.items():
    open(f'{root}/data/profiles/{c}.json', 'w').write(json.dumps(result, indent=2, ensure_ascii=False) + '\n')
PY
echo changed
