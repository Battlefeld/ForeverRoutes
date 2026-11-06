#!/usr/bin/env python3
"""Seed a roster rebuild with the scenarios of the last complete run, so it only searches what changed.

Usage: python3 tools/seed_roster_resume.py /absolute/roster/output
Writes every optimized scenario of the newest data/audits/roster-v*.json.gz as <key>-<group>-<mode>.json, the
file TestRebuildRoster reads with FR_ROSTER_RESUME=1. The rebuild keeps a scenario only when its events still
replay under the current data and model to the same time; every other scenario is searched again.
"""
import gzip,json,pathlib,sys

ROOT=pathlib.Path(__file__).resolve().parents[1]
out=pathlib.Path(sys.argv[1]);out.mkdir(parents=True,exist_ok=True)
audits=sorted((ROOT/'data/audits').glob('roster-v*.json.gz'))
audits=[a for a in audits if not a.name.endswith('-summary.json.gz')]
if not audits:raise SystemExit('no roster audit to resume from')
doc=json.loads(gzip.decompress(audits[-1].read_bytes()))
n=0
for row in doc['results']:
    if row.get('projectionOnly') or row.get('error'):continue
    group,mode=row['scenario'].split('_',1)
    (out/f"{row['key']}-{group}-{mode}.json").write_text(json.dumps(row,ensure_ascii=False)+'\n')
    n+=1
print(f'{n} scenarios of {audits[-1].name} ready to resume')
