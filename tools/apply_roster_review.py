#!/usr/bin/env python3
"""Apply a successful, complete TestRebuildRoster run to the shipped route set.

Usage: python tools/apply_roster_review.py /absolute/review/output
Every fixed-policy result is retained in a compressed audit; 112 free-choice
winners become normal route files. Original inputs and RXP bodies stay separate.
"""
import argparse,gzip,json,pathlib,re

ROOT=pathlib.Path(__file__).resolve().parents[1]
parser=argparse.ArgumentParser(description=__doc__)
parser.add_argument('results',type=pathlib.Path)
args=parser.parse_args()
doc=json.loads((args.results/'summary.json').read_text())
revision=re.search(r'const TimelineModelRevision = "([^"]+)"',(ROOT/'route/checkpoint.go').read_text())[1]
if doc['modelRevision']!=revision or doc['contexts']!=56:raise SystemExit('Model/roster differs')
if doc.get('finalModelReplayCount')!=404:raise SystemExit('Final compiled-model replay has not completed')
rows=doc['results'];optimized=[r for r in rows if not r.get('projectionOnly')]
if len(optimized)!=404 or len(rows)!=516:raise SystemExit('Incomplete scenario matrix')
if any(r.get('error') or r['level']<20 or r.get('skipped') or not r.get('replayVerified') or not r.get('finalModelReplayVerified') for r in optimized):raise SystemExit('Unsuccessful optimized scenario in report')
roster={r['character']['race']+'|'+r['character']['class']+'|'+r['character'].get('faction','') for r in rows}
if len(roster)!=56:raise SystemExit('Missing characters')
winners=[]
for row in optimized:
    if not row['scenario'].endswith('_auto'):continue
    name=row['key']+('-dungeons' if row['scenario'].startswith('dungeons_') else '')+'.json'
    src=args.results/name
    value=json.loads(src.read_text())
    value['basis']='ForeverRoutes optimization of independent routes and ordinary RestedXP Forever quest projections (RestedXP/RXPGuides, CC BY-NC-SA 4.0)'
    value['license']='CC-BY-NC-SA-4.0'
    winners.append((name,(json.dumps(value,ensure_ascii=False,indent=1)+'\n').encode()))
if len(winners)!=112 or len({n for n,_ in winners})!=112:raise SystemExit('Missing winner files')
for name,b in winners:(ROOT/'data/routes'/name).write_bytes(b)
# Full evidence includes events for every option, not only summary hours.
out=ROOT/'data/audits/roster-v22.json.gz'
out.write_bytes(gzip.compress((json.dumps(doc,ensure_ascii=False,separators=(',',':'))+'\n').encode(),mtime=0))
summary=dict(doc)
summary['results']=[{k:v for k,v in row.items() if k not in ['events','missed','classQuests','basis','hours','character','search','sleepingBagModel','wandDetour','preparations','constraints','modelRevision']} for row in rows]
(ROOT/'data/audits/roster-v22-summary.json').write_text(json.dumps(summary,ensure_ascii=False,indent=1)+'\n')
print('Applied',len(winners),'verified winners; retained',len(rows),'scenario records')

# One current table replaces historical hours in the shipped route directory.
def clock(seconds):
    seconds=int(seconds+.5)
    return f'{seconds//3600:02d}:{seconds//60%60:02d}:{seconds%60:02d}'
free={r['key']+'|'+r['scenario'].split('_',1)[0]:r for r in optimized if r['scenario'].endswith('_auto')}
short={'ragefire-chasm':'RF','wailing-caverns':'HDW','ruins-of-lordaeron':'RoL','deadmines':'DM','the-hall-of-thanes':'HdT','shadowfang-keep':'BSF'}
text=['# Geprüfte Routen bis Level 20','',f'Modell: `{revision}`. Alle 56 unterstützten Kombinationen haben zwei geprüfte Ausgaben: Solo und optionale Dungeons. Die zweite Variante darf auch ohne Dungeon enden und behält eine bekannte schnellere Solo-Route.','',
      '**Modellzeiten, keine gemessenen Ingame-Zeiten.** Vorbereitungen, Kämpfe, Reisen und Dungeonläufe enthalten weiterhin Schätzungen. Gleiche Klassen verwenden dasselbe eingefrorene Kampfprofil.','',
      '[Gesamtprüfung und Reproduktion](../audits/roster-v22.md) · [Variantenübersicht](../audits/roster-v22-summary.json) · [Normale RXP-Fallbacks](../../docs/rxp-fallback.md)','',
      '404 Optimierungsszenarien und 112 getrennte RXP-Questprojektionen. Alle optimierten Varianten wurden mit dem finalen Modell erneut nachgespielt. Originale RXP-Kapitel bleiben unverändert.','',
      '| Volk | Fraktion | Klasse | Solo | Optionale Dungeons | Gewählte Läufe |','|---|---|---|---|---|---|']
for key in sorted({r['key'] for r in optimized}):
    a=free[key+'|solo'];b=free[key+'|dungeons'];c=a['character']
    side=c.get('faction') or ('H' if c['race'] in ['Orc','Troll','Undead','Tauren'] else 'A')
    runs=' → '.join(short.get(k,k) for k in b.get('runs') or []) or 'keine'
    text.append(f"| {c['race']} | {side} | {c['class']} | {clock(a['seconds'])} | {clock(b['seconds'])} | {runs} |")
text+=['','RF: Ragefire Chasm; HDW: Wailing Caverns; RoL: Ruins of Lordaeron; DM: Deadmines; HdT: Hall of Thanes; BSF: Shadowfang Keep.','',
       'Zauberstab-/Schlafsack-Zwangsvarianten stehen getrennt in der Variantenübersicht. Die Dateien hier enthalten die beste bekannte freie Auswahl einschließlich der gewählten Vorbereitungsparameter. `classQuests` führt Pflichtreihen und tatsächlich mögliche zusätzliche Klassenoptionen. Kosten dieser Optionen sind direkte Modellvergleiche, keine zusätzlichen langlaufenden Suchen.','',
       'Normale RXP-Fallbacks stehen unabhängig davon in `data/fallbacks/rxp.json.gz` und werden im Addon-Export mit den vollständigen Originalanweisungen und ihrer Lizenz ausgeliefert.']
(ROOT/'data/routes/README.md').write_text('\n'.join(text)+'\n')
