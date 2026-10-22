"""Read independently published EverythingQuests Forever data as an optional audit overlay.

Only input already published as generated Lua literal data is used. No arbitrary Lua
is executed, no guides are copied, and missing quantities stay unknown.
The source repository and exact revision are retained in every fact.
"""
from __future__ import annotations

import argparse
import json
import re
from collections import Counter
from pathlib import Path
from urllib.request import Request, urlopen

from .catalog import parse_lua

REPO = "wheelbarrel00/EverythingQuests"
FILES = {
    "available": "Data/QuestAvailable_Forever.lua",
    "targets": "Data/QuestNPCs_Forever.lua",
    "turnins": "Data/QuestTurnIn_Forever.lua",
}


def download(revision, name):
    if not re.fullmatch(r"[a-f0-9]{40}", revision):
        raise ValueError("A pinned EverythingQuests git SHA is required")
    url = f"https://raw.githubusercontent.com/{REPO}/{revision}/{FILES[name]}"
    req = Request(url, headers={"User-Agent": "ForeverRoutes/0.8 (source provenance audit)"})
    with urlopen(req, timeout=60) as resp:
        body = resp.read(8_000_001)
    if not (100 < len(body) < 8_000_000):
        raise ValueError(f"Incomplete/suspicious EverythingQuests {name} input")
    return body.decode("utf-8")


def section(raw, heading):
    """Extract a flat top-level Lua table section, without interpreting code."""
    if heading == "targets":
        match = re.search(r"(?m)^ns\.CLASSIC_QUEST_NPCS\s*=\s*\{\s*$", raw)
    elif heading == "turnins":
        match = re.search(r"(?m)^ns\.CLASSIC_QUEST_TURNIN\s*=\s*\{\s*$", raw)
    else:
        match = re.search(rf"(?m)^\s*{re.escape(heading)}\s*=\s*\{{\s*$", raw)
    if not match:
        raise ValueError(f"Missing source section: {heading}")
    # Source is generated with exactly one quest-index row per line.
    lines = raw[match.end():].splitlines()
    table = {}
    rx = re.compile(r"^\s*\[(\d+)\]\s*=\s*(\{.*\}|\".*\"|'.*'|-?\d+)\s*,?\s*(?:--.*)?$")
    for line in lines:
        if line.strip() == "}," or line.strip() == "}":
            break
        m = rx.match(line)
        if m:
            key = int(m.group(1))
            value = parse_lua(m.group(2))
            if key in table:
                raise ValueError(f"Duplicate {heading} quest {key}")
            table[key] = value
    if not table:
        raise ValueError(f"Source section {heading} has no usable entries")
    return table


def decode_points(value):
    if not isinstance(value, dict):
        return []
    positions = []
    for map_id, values in value.items():
        if not isinstance(map_id, int) or not isinstance(values, list):
            continue
        for p in values:
            if not isinstance(p, int) or p < 0:
                continue
            source_id = p // 1_000_000_000
            kind = (p // 100_000_000) % 10
            base = p % 100_000_000
            x = (base // 10_000) / 100
            y = (base % 10_000) / 100
            if 0 <= x <= 100 and 0 <= y <= 100 and kind in (1, 2, 3, 5, 6):
                positions.append({"uiMapId": map_id, "x": x, "y": y,
                                  "entity_id": source_id or None,
                                  "entity_kind": {1:"npc", 2:"object", 3:"item_source", 5:"dungeon_npc_entrance",
                                                  6:"dungeon_object_entrance"}[kind]})
    return positions


def decode_targets(value):
    if not isinstance(value, list):
        return []
    result = []
    for packed in value:
        if not isinstance(packed, int) or packed < 0:
            continue
        idx = packed // 100_000_000
        kind = packed // 10_000_000 % 10
        target = packed % 10_000_000
        if target and kind in (1,2,3):
            record = {"objective_index": idx, "kind": {1:"npc",2:"object",3:"loot_source_npc"}[kind],
                      "id": target}
            if record not in result:
                result.append(record)
    return result


def build(available, targets, turnins, revision):
    starts = section(available, "start")
    names = section(available, "names")
    gates = section(available, "gates")
    # Requirements are source comparisons, not considered confirmed unless the
    # source explicitly documents semantics. Keep individual IDs, never inject
    # a guessed unlock condition into route search.
    pre = section(available, "pre")
    preall = section(available, "preAll")
    objective_map = section(targets, "targets")
    turnin_map = section(turnins, "turnins")
    ids = set(starts) | set(names) | set(pre) | set(preall) | set(objective_map) | set(turnin_map) | set(gates)
    quests = {}
    counters = Counter()
    for id_ in sorted(ids):
        record = {
            "id": id_,
            "name": names.get(id_) if isinstance(names.get(id_), str) else None,
            "required_level": gates[id_] // 100_000_000_000 if isinstance(gates.get(id_), int) else None,
            "quest_level": (gates[id_] // 1_000_000_000) % 100 if isinstance(gates.get(id_), int) else None,
            "starts": decode_points(starts.get(id_)),
            "turnins": decode_points(turnin_map.get(id_)),
            "target_refs": decode_targets(objective_map.get(id_)),
            "prereq_any": [x for x in pre.get(id_, []) if isinstance(x, int)] if isinstance(pre.get(id_), list) else [],
            "prereq_all": [x for x in preall.get(id_, []) if isinstance(x, int)] if isinstance(preall.get(id_), list) else [],
        }
        for key in ("starts", "turnins", "target_refs", "prereq_any", "prereq_all"):
            if record[key]:
                counters[f"quests_with_{key}"] += 1
        quests[str(id_)] = record
    counters["source_quest_ids"] = len(quests)
    return {"schema": "foreverroutes.addon_crosscheck.v1",
            "source": {"name": "EverythingQuests Forever", "repository": f"https://github.com/{REPO}",
                       "revision": revision, "license": "MIT; credits upstream AllTheThings where used"},
            "stats": dict(counters), "quests": quests}


def main(argv=None):
    p = argparse.ArgumentParser()
    p.add_argument("--revision", required=True, help="Exactly pinned EverythingQuests commit SHA")
    p.add_argument("--output", required=True, type=Path)
    a = p.parse_args(argv)
    raw = {k: download(a.revision, k) for k in FILES}
    result = build(raw["available"], raw["targets"], raw["turnins"], a.revision)
    if result["stats"]["source_quest_ids"] < 1000:
        raise ValueError("Addon export coverage suspicious; preserving old artifact")
    a.output.parent.mkdir(parents=True, exist_ok=True)
    a.output.write_text(json.dumps(result, ensure_ascii=False, separators=(",", ":"))+"\n", encoding="utf8")
    print(json.dumps(result["stats"]))


if __name__ == "__main__":
    main()
