"""Quest-ID and missing-fact coverage for published leveling-guide candidate lists.

This is a pre-flight check, NOT a simulated route duration or a reproduced
paid leveling guide. A guide's action order still requires separately supplied
authorized step data.
"""
from __future__ import annotations

import argparse
import json
from pathlib import Path


def report(db, candidates):
    if db.get("schema") != "foreverroutes.unified_quests.v1":
        raise ValueError("Expected merged ForeverRoutes quest catalog")
    quests = db.get("quests", {})
    sources = {}
    for source, guides in candidates.get("sources", {}).items():
        sources[source] = {}
        for guide, raw in guides.items():
            ids = sorted({int(x) for x in raw if int(x) > 0})
            missing = []
            no_starter_or_turnin = []
            no_objectives = []
            unknown_objective_counts = []
            missing_xp = []
            for qid in ids:
                entry = quests.get(str(qid))
                if not entry or entry.get("status") == "unresolved_guide_candidate":
                    missing.append(qid)
                    continue
                if entry.get("not_actionable"):
                    no_starter_or_turnin.append(qid)
                if not entry.get("objectives"):
                    no_objectives.append(qid)
                elif any(o.get("required_count") is None for o in entry["objectives"]):
                    unknown_objective_counts.append(qid)
                if not (entry.get("fields", {}).get("xp") or
                        entry.get("fields", {}).get("xp_observed_by_player_level")):
                    missing_xp.append(qid)
            sources[source][guide] = {
                "guide_quest_ids": len(ids),
                "known_quest_ids": len(ids) - len(missing),
                "missing_quest_ids": missing,
                "no_confirmed_starter_turnin": no_starter_or_turnin,
                "no_objective_references": no_objectives,
                "unknown_objective_quantities": unknown_objective_counts,
                "missing_quest_xp": missing_xp,
                "can_calculate_valid_travel_times": False,
                "notes": "Quest-coverage audit only; no path graph or ordered guide actions are included",
            }
    return {"schema":"foreverroutes.guide_coverage.v1",
            "database_build": db.get("build"),
            "sources": sources,
            "policy": "Classic reference values may be used provisionally; missing objectives are never assumed complete."}


def main(argv=None):
    p = argparse.ArgumentParser()
    p.add_argument("--database", type=Path, required=True)
    p.add_argument("--guide-ids", type=Path, required=True)
    p.add_argument("--output", type=Path, required=True)
    a = p.parse_args(argv)
    db = json.loads(a.database.read_text(encoding="utf-8"))
    candidates = json.loads(a.guide_ids.read_text(encoding="utf-8"))
    result = report(db, candidates)
    a.output.parent.mkdir(parents=True,exist_ok=True)
    a.output.write_text(json.dumps(result,ensure_ascii=False,separators=(",",":"))+"\n",encoding="utf-8")
    print(json.dumps({source:{guide:{"known":v["known_quest_ids"],"total":v["guide_quest_ids"]}
                     for guide,v in groups.items()} for source,groups in result["sources"].items()}))


if __name__ == "__main__":
    main()
