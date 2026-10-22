"""Regenerate the exact outstanding quest XP ID report from the current unified audit.

This replaces the stale 834-ID text artifact and never conflates 0 XP with missing.
"""
import argparse
import csv
import io
import json
from pathlib import Path
from foreverroutes.warcraftdb_gaps import ids_from_audit

def build_report(database,audit,web=None):
    ids=ids_from_audit(audit) if audit.get("missing",{}).get("quests_without_xp") else []
    quests=database["quests"]
    rows=[]
    for ident in ids:
        quest=quests.get(str(ident),{})
        fields=quest.get("fields",{})
        presence=quest.get("source_presence",[])
        if "xp" in fields or "xp_observed_by_player_level" in fields:
            raise ValueError(f"Quest {ident} marked missing yet has XP")
        page=(web or {}).get("quests",{}).get(str(ident),{})
        status="never_checked"
        if page:
            checks=page.get("attempts",[])
            status=";".join(f"{a.get('source')}:{a.get('status')}" for a in checks) or "checked"
        rows.append({"quest_id":ident,
            "quest_name":fields.get("name",{}).get("value",""),
            "source_group":("QuestieDB" if "questiedb" in presence else
                            "ForeverDB" if "foreverdb" in presence else
                            "EverythingQuests" if "everythingquests" in presence else "Other"),
            "required_level":fields.get("required_level",{}).get("value",""),
            "status":quest.get("status","unknown"),"web_lookup_status":status})
    return rows

def render_text(rows):
    lines=[f"OFFENE QUEST-XP-IDS — ForeverRoutes 1–30",
           f"Aktueller Abgleich: {len(rows)} Quest-IDs ohne belegten Basis-XP-Wert.",
           "0 XP zählt als bekannter XP-Wert, sofern durch eine Quelle belegt.",
           ""]
    for i in range(0,len(rows),14):
        lines.append(", ".join(str(r["quest_id"]) for r in rows[i:i+14]))
    return "\n".join(lines)+"\n"

def main():
    p=argparse.ArgumentParser()
    p.add_argument("--database",type=Path,required=True)
    p.add_argument("--audit",type=Path,required=True)
    p.add_argument("--web",type=Path)
    p.add_argument("--csv",type=Path,required=True)
    p.add_argument("--text",type=Path,required=True)
    a=p.parse_args()
    db=json.loads(a.database.read_text(encoding="utf-8"))
    audit=json.loads(a.audit.read_text(encoding="utf-8"))
    web=json.loads(a.web.read_text(encoding="utf-8")) if a.web and a.web.is_file() else None
    rows=build_report(db,audit,web)
    a.csv.parent.mkdir(parents=True,exist_ok=True)
    with a.csv.open("w",encoding="utf-8",newline="") as f:
        w=csv.DictWriter(f,fieldnames=("quest_id","quest_name","source_group","required_level","status","web_lookup_status"))
        w.writeheader()
        w.writerows(rows)
    a.text.parent.mkdir(parents=True,exist_ok=True)
    a.text.write_text(render_text(rows),encoding="utf-8")
    print(f"Exact missing-XP report: {len(rows)} IDs")

if __name__=="__main__":
    main()
