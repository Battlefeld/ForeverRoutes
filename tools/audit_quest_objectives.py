"""Audit every shipped quest's objective references; structural checks are not in-game verification."""
import argparse
import collections
import gzip
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--dataset', type=Path, default=ROOT/'data/routes-data.json.gz')
p.add_argument('--out', type=Path, default=ROOT/'data/audits/quest-objectives-v25')
a = p.parse_args()
d = json.loads(gzip.decompress(a.dataset.read_bytes()))
npcs = {n['id']: n for n in d['npcs']}
objects = {o['id']: o for o in d['objects']}
rows = []
for q in d['quests']:
    issues, objectives = [], []
    for o in q.get('obj', []):
        problems = []
        kind = o['k']
        if o['n'] <= 0:
            problems.append('non-positive count')
        refs = []
        if kind in ('kill', 'talk', 'interact', 'duel'):
            refs += [('npc', i) for i in [o['id']] + o.get('targets', [])]
        if kind == 'object':
            refs.append(('object', o['id']))
        refs += [('npc', i) for i in o.get('vendors', [])]
        refs += [('npc' if r.get('npc') else 'object', r.get('npc') or r.get('obj')) for r in o.get('drops', [])]
        sources = []
        for refkind, target in refs:
            entity = (npcs if refkind == 'npc' else objects).get(target)
            if entity is None:
                problems.append(f'missing {refkind} {target}')
            elif not entity.get('spawns') and not o.get('places') and not o.get('inside'):
                problems.append(f'unplaced {refkind} {target}')
            if entity and refkind == 'npc' and kind not in ('talk', 'interact') and target not in o.get('vendors', []) and entity.get('max', 0) < 1:
                problems.append(f'unknown combat level for npc {target}')
            sources.append({'kind': refkind, 'id': target, 'entity': entity})
        if kind == 'item' and not o.get('drops') and not o.get('vendors') and not o.get('inside') and not ((o.get('ground') or o.get('useItem')) and o.get('places')):
            problems.append('unresolved item acquisition; waypoint is not a source')
        if kind in ('event', 'skill') and not o.get('places') and not o.get('inside'):
            problems.append('event has no recorded place')
        objectives.append({'kind': kind, 'id': o['id'], 'count': o['n'], 'source': o.get('src'),
                           'groundEvidence': o.get('ground', False), 'providedItemUse': o.get('useItem'), 'targets': sources, 'issues': problems})
        issues.extend(problems)
    status = 'structural_references_present'
    if q.get('type'):
        status = 'group_or_special_quest'
    if q.get('unsupported') or issues:
        status = 'incomplete_or_unsupported'
    rows.append({'id': q['id'], 'name': q['name'], 'level': q['lvl'], 'type': q.get('type', ''),
                 'status': status, 'unsupported': q.get('unsupported', ''), 'issues': issues,
                 'sources': q.get('src', {}), 'objectives': objectives})
counts = dict(collections.Counter(r['status'] for r in rows))
report = {'schema': 'foreverroutes.objective-audit.v1', 'questCount': len(rows),
          'maxQuestLevel': max(q['lvl'] for q in d['quests']), 'statusCounts': counts,
          'typeCounts': dict(collections.Counter(q.get('type', 'open_world') for q in d['quests'])),
          'sources': d.get('sources', []), 'quests': rows,
          'limitation': 'Reference, count and source checks only. Not a manual or in-game verification of every quest. Runtime also checks faction, class, prerequisites, reachability and combat safety.'}
a.out.parent.mkdir(parents=True, exist_ok=True)
Path(str(a.out)+'.json.gz').write_bytes(gzip.compress(json.dumps(report, ensure_ascii=False, sort_keys=True).encode(), mtime=0))
lines = ['# Quest objective audit v2.5', '',
         f"All {len(rows)} shipped quests through level {report['maxQuestLevel']} were checked for objective counts and source references.", '',
         'This is a structural audit of the pinned source rebuild, not an assertion that every quest has been verified in game.', '',
         '| Status | Quests |', '|---|---:|']
lines += [f'| {k} | {v} |' for k, v in sorted(counts.items())]
lines += ['', 'The full compressed report contains every quest, target, source attribution and unresolved reference.', '',
          'Solo planning excludes elite/group quest types. Ordinary elite combat is permitted only by the explicit pet policy, never above player level; class-quest encounters retain their separate level checks.',
          'An item waypoint alone no longer establishes a safe pickup. A ground pickup needs explicit evidence; a container interaction needs an explicit use of an item provided by this quest. Otherwise a real drop/vendor source is required. Unresolved acquisition stays unsupported.', '',
          '## Unresolved item acquisition', '', '| Quest | Name | Reason |', '|---|---|---|']
for r in rows:
    if 'acquisition unresolved' in r['unsupported']:
        lines.append(f"| {r['id']} | {r['name'].replace('|', '/')} | {r['unsupported']} |")
Path(str(a.out)+'.md').write_text('\n'.join(lines)+'\n', encoding='utf-8')
print(json.dumps({k:v for k,v in report.items() if k not in ('sources','quests')}, ensure_ascii=False))
