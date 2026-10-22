"""Build provisional WoW Forever 1–30 quest/NPC facts with Classic fallbacks.

Source files downloaded LOCALLY at pinned QuestieDB commit; not vendored or
licensed as ForeverRoutes data. Only field-specific verified Forever evidence
supersedes a Classic reference. Python 3.10+, standard library only.
"""
from __future__ import annotations

import argparse
import json
import re
from pathlib import Path
from urllib.request import Request, urlopen

QUESTIE_SHA = "f844007b329b72f18cf2bda3d87540012294732a"
FILES = {
    "quest": "data/Forever/foreverQuestDB.lua",
    "xp": "support/Forever/QuestXP/xpDB-classic.lua",
    "npc": "data/Classic/classicNpcDB.lua",
}
PROVENANCE = {
    "questie_forever": 20, "questie_classic_xp": 10,
    "questie_classic_npc_raw": 10, "foreverdb_classic": 10,
    "foreverdb_client": 40, "wowhead_forever": 50,
    "foreverdb_observed": 60,
}
RECORD = re.compile(r"^\[(\d+)\]\s*=\s*(\{.*\}),?\s*$")
XP_RECORD = re.compile(r"^\s*\[(\d+)\]\s*=\s*\{(\d+),\s*(\d+)\}")

def fields(raw: str) -> list[str]:
    """Parse only outer Lua table cells, without running third-party Lua."""
    raw = raw.strip()
    if not raw.startswith("{") or not raw.endswith("}"):
        raise ValueError("Expected outer Lua table")
    cells, cell, depth, quote, escaped = [], [], 0, None, False
    for char in raw[1:-1]:
        if quote is not None:
            cell.append(char)
            if escaped: escaped = False
            elif char == "\\": escaped = True
            elif char == quote: quote = None
        elif char in ("'", '"'):
            quote = char
            cell.append(char)
        elif char == "{":
            depth += 1
            cell.append(char)
        elif char == "}":
            depth -= 1
            if depth < 0: raise ValueError("Unbalanced Lua row")
            cell.append(char)
        elif char == "," and depth == 0:
            cells.append("".join(cell).strip())
            cell = []
        else: cell.append(char)
    if depth or quote is not None: raise ValueError("Unterminated Lua row")
    cells.append("".join(cell).strip())
    return cells

def rows(raw: str):
    for line in raw.splitlines():
        m = RECORD.match(line)
        if m: yield int(m[1]), fields(m[2])

def evidence(value, source: str, url: str, **extras):
    if source not in PROVENANCE or not isinstance(url, str) or not url.startswith("https://"):
        raise ValueError("Missing valid evidence provenance")
    return {"value": value, "source": source, "url": url, **extras}

def _url(path):
    return f"https://github.com/Questie/QuestieDB/blob/{QUESTIE_SHA}/{path}"

def parse_quests(quest_raw: str, xp_raw: str, maximum_level: int = 30) -> dict:
    classic_xp = {}
    for line in xp_raw.splitlines():
        m = XP_RECORD.match(line)
        if m: classic_xp[int(m[1])] = int(m[3])
    result = {}
    for quest_id, row in rows(quest_raw):
        if len(row) < 13: continue
        try:
            lvl, required = int(row[4]), int(row[3])
        except ValueError:
            continue
        if not 1 <= lvl <= maximum_level: continue
        url = _url(FILES["quest"])
        def fact(x): return evidence(x, "questie_forever", url)
        def ids(token): return [int(x) for x in re.findall(r"\d+", token)] if token != "nil" else []
        starters = fields(row[1]) if row[1].startswith("{") else []
        finishers = fields(row[2]) if row[2].startswith("{") else []
        f = {
            "name": fact(row[0][1:-1]), "quest_level": fact(lvl),
            "required_level": fact(required),
            "starter_npc_ids": fact(ids(starters[0]) if starters else []),
            "finisher_npc_ids": fact(ids(finishers[0]) if finishers else []),
            "prequest_all": fact(ids(row[11])),
            "prequest_any": fact(ids(row[12])),
            "race_mask": fact(int(row[5]) if row[5].isdigit() else None),
            "class_mask": fact(int(row[6]) if row[6].isdigit() else None),
        }
        if quest_id in classic_xp:
            f["xp_classic_reference"] = evidence(classic_xp[quest_id], "questie_classic_xp", _url(FILES["xp"]))
        result[str(quest_id)] = {"id": quest_id, "fields": f}
    return result

def parse_classic_npcs(npc_raw: str, maximum_mob_level: int = 35) -> dict:
    """Get real numeric Classic HP from raw file, NOT Questie API placeholders 0/1."""
    out = {}
    for npc_id, row in rows(npc_raw):
        if len(row) < 6: continue
        try:
            hp_min, hp_max, min_level, max_level = map(int, row[1:5])
        except ValueError:
            continue
        if not (1 <= min_level <= maximum_mob_level and max_level >= min_level):
            continue
        if hp_min <= 1 or hp_max < hp_min:
            continue  # unhelpful sentinel, not real monster HP
        u = _url(FILES["npc"])
        def fact(v): return evidence(v, "questie_classic_npc_raw", u)
        out[str(npc_id)] = {"id": npc_id, "fields": {
            "name": fact(row[0][1:-1]),
            "min_level": fact(min_level), "max_level": fact(max_level),
            "hp_min": fact(hp_min), "hp_max": fact(hp_max)}}
    return out

def merge(table: dict, records: list, *, identity: str = "quest_id") -> list:
    conflicts = []
    for record in records:
        id_ = int(record[identity])
        if id_ <= 0: raise ValueError("Invalid ID")
        entry = table.setdefault(str(id_), {"id": id_, "fields": {}})
        for key, incoming in record.get("fields", {}).items():
            if incoming["source"] not in PROVENANCE: raise ValueError("Unknown source")
            if not isinstance(incoming.get("url"), str) or not incoming["url"].startswith("https://"):
                raise ValueError("Missing evidence URL")
            expansion = []
            if key == "xp_by_player_level":
                if not isinstance(incoming["value"], dict): raise ValueError("Invalid XP map")
                for lv, xp in incoming["value"].items():
                    if not str(lv).isdigit() or type(xp) is not int or xp < 0:
                        raise ValueError("Invalid observed XP")
                    expansion.append((f"xp_at_player_level_{lv}", xp))
            else:
                expansion = [(key, incoming["value"])]
            for field, value in expansion:
                candidate = evidence(value, incoming["source"], incoming["url"],
                                     **{n: incoming[n] for n in ("players", "build") if n in incoming})
                previous = entry["fields"].get(field)
                if previous and previous["value"] != value:
                    conflicts.append({"id": id_, "field": field, "previous": previous, "candidate": candidate})
                if previous is None or PROVENANCE[candidate["source"]] > PROVENANCE[previous["source"]]:
                    entry["fields"][field] = candidate
    return conflicts

def quest_xp(quest: dict, player_level: int):
    f = quest.get("fields", {})
    measured = f.get(f"xp_at_player_level_{player_level}")
    if measured: return {**measured, "quality": "forever_observed"}
    old = f.get("xp_classic_reference")
    if old: return {**old, "quality": "classic_reference", "method": "unadjusted_classic_xp"}
    return None

def npc_hp(npc: dict, mob_level: int):
    f = npc.get("fields", {})
    measured = f.get(f"hp_at_level_{mob_level}")
    if measured and type(measured["value"]) is int and measured["value"] > 1:
        return {**measured, "quality": "forever_observed"}
    if any(key not in f for key in ("hp_min", "hp_max", "min_level", "max_level")):
        return None
    lo, hi = f["min_level"]["value"], f["max_level"]["value"]
    low, high = f["hp_min"]["value"], f["hp_max"]["value"]
    if not (lo <= mob_level <= hi and low > 1 and high >= low):
        return None
    hp = low if lo == hi else round(low + (high - low) * (mob_level - lo) / (hi - lo))
    return {**f["hp_min"], "value": hp, "quality": "classic_reference",
            "method": "linear_hp_interpolation" if lo < mob_level < hi else "classic_endpoint"}

def questie_download(cache_dir: Path) -> dict:
    cache_dir.mkdir(parents=True, exist_ok=True)
    data = {}
    for kind, filename in FILES.items():
        dest = cache_dir / f"{QUESTIE_SHA}_{kind}.lua"
        if not dest.exists():
            url = f"https://raw.githubusercontent.com/Questie/QuestieDB/{QUESTIE_SHA}/{filename}"
            with urlopen(Request(url, headers={"User-Agent":"ForeverRoutes/0.4"}), timeout=40) as response:
                body = response.read(5_000_000)
            if len(body) < 10_000: raise ValueError("Partial download")
            dest.write_bytes(body)
        data[kind] = dest.read_text(encoding="utf-8")
    return data

def main():
    cli = argparse.ArgumentParser()
    cli.add_argument("--cache", type=Path, default=Path(".cache/questie"))
    cli.add_argument("--quest-overlay", type=Path, action="append", default=[])
    cli.add_argument("--npc-overlay", type=Path, action="append", default=[])
    cli.add_argument("--output", type=Path, default=Path(".generated/forever_quest_npc_1_30.json"))
    a = cli.parse_args()
    source = questie_download(a.cache)
    quests, npcs = parse_quests(source["quest"], source["xp"]), parse_classic_npcs(source["npc"])
    conflicts = []
    for path in a.quest_overlay:
        conflicts += merge(quests, json.loads(path.read_text(encoding="utf8"))["observations"])
    for path in a.npc_overlay:
        conflicts += merge(npcs, json.loads(path.read_text(encoding="utf8"))["observations"], identity="npc_id")
    a.output.parent.mkdir(parents=True, exist_ok=True)
    output = {"schema":"foreverroutes.quest_npc_reference.v1", "questie_sha":QUESTIE_SHA,
              "policy":"observed Forever first, Classic fallback; modeled not server-proven",
              "quests":quests, "npcs":npcs, "conflicts":conflicts}
    a.output.write_text(json.dumps(output, ensure_ascii=False, indent=1), encoding="utf8")
    print(json.dumps({"quests":len(quests), "mobs_with_classic_hp":len(npcs),
                      "conflicts":len(conflicts), "output":str(a.output)}))

if __name__ == "__main__": main()
