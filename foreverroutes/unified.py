"""Merge QuestieDB + ForeverDB into a provenance-aware, usable 1–30 quest database.

Classic references are *accepted as provisional inputs*, but never described as
Forever observations. Unknown target counts remain null, not invented.
"""
from __future__ import annotations

import argparse
import json
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path

SCHEMA = "foreverroutes.unified_quests.v1"
URL_QUESTIE = "https://github.com/Questie/QuestieDB"
URL_FOREVER = "https://foreverdb.net"


def fact(value, source, quality, url, **extra):
    if value is None:
        return None
    return {"value": value, "source": source, "quality": quality, "url": url, **extra}


def source_fact(qid, field, value, source, *, url=None, quality=None, build=None):
    if value is None:
        return None
    src = "foreverdb_forever" if source == "forever" else "foreverdb_classic"
    qual = quality or ("forever_reference" if source == "forever" else "classic_reference")
    return fact(value, src, qual, url or f"{URL_FOREVER}/quest/{qid}",
                **({"build": build} if build else {}))


def valid_id(value):
    try:
        n = int(value)
        return n if n > 0 else None
    except (TypeError, ValueError):
        return None


def fstarter(row, qid, source, build):
    if not isinstance(row, dict):
        return None
    typ = row.get("type")
    entity_id = valid_id(row.get("id"))
    if typ not in ("npc", "object", "item") or not entity_id:
        return None
    result = {"npc": [], "object": [], "item": []}
    result[typ] = [entity_id]
    for item in row.get("others") or []:
        if isinstance(item, dict) and item.get("type") in result:
            extra = valid_id(item.get("id"))
            if extra and extra not in result[item["type"]]:
                result[item["type"]].append(extra)
    return source_fact(qid, "starter", result, source, build=build)


def loc(row, qid, build):
    if not isinstance(row, dict):
        return None
    value = row.get("location")
    if not isinstance(value, dict):
        return None
    zone = valid_id(value.get("uiMapId"))
    x, y = value.get("x"), value.get("y")
    if zone is None or type(x) not in (int, float) or type(y) not in (int, float) or not (0 <= x <= 100 and 0 <= y <= 100):
        return None
    label = value.get("source") or row.get("source") or "unverified"
    seen = str(label).startswith("seen in WoW Forever")
    return fact({"uiMapId": zone, "x": x, "y": y}, "foreverdb_observed_position" if seen else "foreverdb_classic_position",
                "forever_observed" if seen else "classic_reference",
                f"{URL_FOREVER}/quest/{qid}", build=build, original_source=label)


def valid_observation(ob):
    if not isinstance(ob, dict):
        return None
    fields = ob.get("fields")
    qid = valid_id(ob.get("quest_id"))
    if not qid or not isinstance(fields, dict):
        return None
    accepted = {}
    for k, rec in fields.items():
        if not isinstance(rec, dict) or rec.get("source") not in ("foreverdb_observed", "foreverdb_client", "wowhead_forever"):
            continue
        if not isinstance(rec.get("url"), str) or not rec["url"].startswith("https://"):
            continue
        if k in ("quest_level", "required_level", "xp_by_player_level"):
            accepted[k] = fact(rec.get("value"), rec["source"],
                               "forever_observed" if rec["source"] == "foreverdb_observed" else "forever_reference",
                               rec["url"], **{t: rec[t] for t in ("build", "players") if t in rec})
    return qid, accepted


def choose(field, base, incoming, discrepancies, qid):
    if incoming is None:
        return base
    if base is not None and base.get("value") != incoming.get("value"):
        discrepancies.append({
            "quest_id": qid, "field": field, "questie": base, "foreverdb": incoming,
            "selected": "foreverdb" if incoming["quality"] != "classic_reference" else "questie",
        })
    # A Classic-era API fallback must not override the dedicated Questie Forever catalog.
    if base is not None and incoming["quality"] == "classic_reference":
        return base
    return incoming


def merge(catalog, snapshot, overlays=None):
    if catalog.get("schema") != "foreverroutes.catalog.1":
        raise ValueError("Unknown Questie catalog schema")
    if snapshot.get("schema") != "foreverroutes.foreverdb_snapshot.v1":
        raise ValueError("Unknown ForeverDB snapshot schema")
    qcat = catalog["quests"]
    qapi = {str(int(r["id"])): r for r in snapshot["quests"] if isinstance(r, dict) and valid_id(r.get("id"))}
    if len(qapi) != len(snapshot["quests"]):
        raise ValueError("ForeverDB list contains duplicate or malformed quest IDs")
    conflicts = []
    quests = {}
    build = snapshot.get("game_metadata", {}).get("build")
    questie_sha = next((s.get("revision") for s in catalog.get("sources", {}).values() if isinstance(s, dict)), None)
    questie_url = f"{URL_QUESTIE}/tree/{questie_sha}" if questie_sha else URL_QUESTIE
    for id_ in sorted(set(qcat) | set(qapi), key=int):
        base = qcat.get(id_, {})
        api = qapi.get(id_)
        if not api and base.get("candidate_only"):
            quests[id_] = {
                "id": int(id_), "status": "unresolved_guide_candidate",
                "not_actionable": True, "source_presence": ["guide_id_candidate"],
                "guide_sources": base.get("source_guides", []),
                "fields": {}, "objectives": [], "locations": {},
            }
            continue
        qid = int(id_)
        fields = {}
        for key in ("name", "required_level", "quest_level", "required_races_mask", "required_classes_mask",
                    "started_by", "finished_by", "prequest_all", "prequest_any", "zone_or_sort"):
            value = base.get(key)
            if value is not None:
                fields[key] = fact(value, "questiedb_forever" if key not in ("zone_or_sort",) else "questiedb_forever",
                                   "classic_reference", questie_url)
        if base.get("xp") and isinstance(base["xp"], dict) and base["xp"].get("value") is not None:
            fields["xp"] = fact(base["xp"]["value"], "questiedb_classic_xp", "classic_reference", questie_url)
        locations = {}
        if api:
            api_source = api.get("source", "classic")
            if api_source not in ("forever", "classic"):
                api_source = "classic"
            for key, field in (("name", "name"), ("required_level", "requiredLevel"), ("quest_level", "level")):
                incoming = source_fact(qid, key, api.get(field), api_source, url=api.get("url"), build=build)
                fields[key] = choose(key, fields.get(key), incoming, conflicts, qid)
            for field, node in (("started_by", "starts"), ("finished_by", "ends")):
                incoming = fstarter(api.get(node), qid, api_source, build)
                if incoming and field == "finished_by":
                    incoming["value"].pop("item", None)
                # Location is an independent fact: an observed point is reliable even
                # if the quest's name/level data is tagged as Classic.
                loc_fact = loc(api.get(node), qid, build)
                if loc_fact:
                    locations[node] = loc_fact
                fields[field] = choose(field, fields.get(field), incoming, conflicts, qid)
            if api.get("zone"):
                z = api["zone"]
                if isinstance(z, dict):
                    fields["zone"] = source_fact(qid, "zone",
                       {"name": z.get("n"), "uiMapId": z.get("m")},
                       api_source, build=build)
            if api.get("side") in ("a", "h", "both", "n"):
                fields["side"] = source_fact(qid, "side", api["side"], api_source, build=build)
        objectives = base.get("objectives", [])
        if not isinstance(objectives, list):
            objectives = []
        clean_objectives = []
        for o in objectives:
            if not isinstance(o, dict):
                continue
            clean_objectives.append({
                **o,
                "count_source": o.get("count_source", "unresolved"),
                "quantity_quality": "unresolved" if o.get("required_count") is None else "source_reference",
            })
        quests[id_] = {
            "id": qid,
            "status": "forever_listed" if api else "questie_reference_only",
            "source_presence": (["questiedb"] if base and not base.get("candidate_only") else []) + (["foreverdb"] if api else []),
            "known_in_forever": api.get("knownInForever") if api else None,
            "source_guides": base.get("source_guides", []),
            "fields": {k: v for k, v in fields.items() if v is not None},
            "objectives": clean_objectives,
            "locations": locations,
            "not_actionable": not (fields.get("started_by") and fields.get("finished_by")),
            "foreverdb_url": api.get("url") if api else None,
        }
    # Explicit, sourced observations can override baseline fields only.
    observations = (overlays or {}).get("observations", [])
    for ob in observations:
        parsed = valid_observation(ob)
        if not parsed:
            continue
        qid, values = parsed
        entry = quests.setdefault(str(qid), {"id": qid, "status": "observation_only",
                         "source_presence": ["observation"], "fields": {},
                         "objectives": [], "locations": {}, "not_actionable": True})
        for key, value in values.items():
            if key == "xp_by_player_level" and isinstance(value["value"], dict):
                entry["fields"]["xp_observed_by_player_level"] = value
            elif key in ("quest_level", "required_level"):
                entry["fields"][key] = choose(key, entry["fields"].get(key), value, conflicts, qid)
    npcs = catalog.get("npcs", {})
    items = catalog.get("items", {})
    objects = catalog.get("objects", {})
    count = Counter()
    for q in quests.values():
        count["total_quest_ids"] += 1
        if q["status"] == "unresolved_guide_candidate":
            count["unresolved_guide_candidates"] += 1
            continue
        count["identified_quests"] += 1
        if q["status"] == "forever_listed":
            count["foreverdb_quests"] += 1
        if "questiedb" in q.get("source_presence", []):
            count["questiedb_quests"] += 1
        if q.get("source_presence") == ["foreverdb"]:
            count["foreverdb_only_quests"] += 1
        if len(q.get("source_presence", [])) == 2:
            count["overlap"] += 1
        if not q.get("fields", {}).get("xp") and not q.get("fields", {}).get("xp_observed_by_player_level"):
            count["quests_missing_xp"] += 1
        if not q.get("objectives"):
            count["quests_without_objective_references"] += 1
        count["objectives_unknown_quantity"] += sum(o.get("required_count") is None for o in q.get("objectives", []))
        if q["not_actionable"]:
            count["quests_without_complete_giver_turnin"] += 1
        count["forever_observed_locations"] += sum(v["quality"] == "forever_observed" for v in q["locations"].values())
    count["npc_reference_records"] = len(npcs)
    count["npc_hp_classic_references"] = sum(n.get("hp_min") is not None for n in npcs.values())
    count["item_reference_records"] = len(items)
    count["object_reference_records"] = len(objects)
    count["conflicts"] = len(conflicts)
    count["foreverdb_details_fetched"] = len(snapshot.get("quest_details", {}))
    return {
        "schema": SCHEMA,
        "build": build,
        "policy": "Classic references accepted as provisional simulation inputs; unverified fields remain explicit.",
        "sources": {
            "questiedb": {"url": URL_QUESTIE, "revision": questie_sha},
            "foreverdb": snapshot["source"],
            "foreverdb_game_metadata": snapshot.get("game_metadata"),
        },
        "stats": dict(count),
        "quests": quests,
        "npcs": npcs,
        "objects": objects,
        "items": items,
        "level_xp_classic": catalog.get("level_xp_classic", {}),
        "changes_seen_unverified": snapshot.get("quest_changes_seen", []),
        "field_conflicts": conflicts,
        "external_source_registry": [
            {"name": "All The Things Forever", "url": "https://github.com/ATTWoWAddon/AllTheThings", "state": "comparison_pending", "notes": "Quest chains, rewards and requirements; review redistribution terms before bulk import"},
            {"name": "Everything Quests", "url": "https://github.com/wheelbarrel00/EverythingQuests", "state": "comparison_pending", "notes": "Derived Forever quest starts and quest conditions"},
            {"name": "Forever Quest Pins", "url": "https://github.com/TylerAkins/forever-quest-markers", "state": "comparison_pending", "notes": "AllTheThings-derived locations and requirements"},
            {"name": "LibQuestDB", "url": "https://www.curseforge.com/wow/addons/questdb", "state": "comparison_pending", "notes": "Classic objective text/rewards and Forever client harvest"},
            {"name": "ForeverDiff", "url": "https://foreverdiff.gg/", "state": "comparison_pending", "notes": "Client-table inventory and collected observations"},
            {"name": "Wowhead Forever", "url": "https://www.wowhead.com/forever", "state": "manual_or_licensed_adapter", "notes": "Field-level verified observations only, no automated scraping assumed"},
        ],
    }


def audit(data):
    q = data["quests"]
    missing = {
        "unresolved_guide_candidates": [v["id"] for v in q.values() if v["status"] == "unresolved_guide_candidate"],
        "quests_without_full_starter_turnin": [v["id"] for v in q.values() if v["status"] != "unresolved_guide_candidate" and v["not_actionable"]],
        "quests_with_unknown_objective_counts": [v["id"] for v in q.values() if any(o.get("required_count") is None for o in v.get("objectives", []))],
        "quests_without_objective_reference": [v["id"] for v in q.values() if v["status"] != "unresolved_guide_candidate" and not v.get("objectives")],
        "quests_without_xp": [v["id"] for v in q.values() if v["status"] != "unresolved_guide_candidate" and not ("xp" in v["fields"] or "xp_observed_by_player_level" in v["fields"])],
    }
    return {
        "schema": "foreverroutes.unified_audit.v1",
        "build": data["build"],
        "stats": data["stats"],
        "missing": missing,
        "conflicts": data["field_conflicts"],
        "external_sources": data["external_source_registry"],
        "limits": [
            "Objective quantities missing: cannot calculate precise kills and item collections",
            "Spawn points are zone coordinates, not walkable route links",
            "Forever-only quests without objectives require in-game or compatible public reference data",
            "Classic fallback XP/HP is provisional; Forever observed overrides win",
            "Not a validated fastest-route model yet",
        ],
    }


def main(argv=None):
    p = argparse.ArgumentParser(description="Build a source-tagged QuestieDB/ForeverDB quest catalog")
    p.add_argument("--catalog", type=Path, required=True)
    p.add_argument("--foreverdb", type=Path, required=True)
    p.add_argument("--overlay", type=Path, action="append", default=[])
    p.add_argument("--output", type=Path, required=True)
    p.add_argument("--audit", type=Path, required=True)
    args = p.parse_args(argv)
    overlays = {"observations": []}
    for path in args.overlay:
        overlays["observations"].extend(json.loads(path.read_text(encoding="utf-8")).get("observations", []))
    catalog = json.loads(args.catalog.read_text(encoding="utf-8"))
    snapshot = json.loads(args.foreverdb.read_text(encoding="utf-8"))
    result = merge(catalog, snapshot, overlays)
    report = audit(result)
    if result["stats"]["identified_quests"] < 500:
        raise ValueError("Quest merger detected implausibly small input; refusing destructive update")
    for dest, data in ((args.output, result), (args.audit, report)):
        dest.parent.mkdir(parents=True, exist_ok=True)
        dest.write_text(json.dumps(data, ensure_ascii=False, separators=(",", ":")) + "\n", encoding="utf-8")
    print(json.dumps({"output": str(args.output), "audit": str(args.audit), "stats": result["stats"]}, ensure_ascii=False))


if __name__ == "__main__":
    main()
