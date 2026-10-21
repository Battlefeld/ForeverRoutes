"""Read LOCAL reference guide exports for coverage without redistributing guide text.

No route is declared playable until every conditional and positional step has
been resolved against an authoritative world. This is NOT a time benchmark.
"""
from __future__ import annotations
import csv
import re
from collections import Counter
from pathlib import Path


HEADERS = {
    'journey': {'guide': 'guide_ids', 'step': 'schritt', 'action':'aktion', 'quest':'quest_id', 'condition':'schritt_bedingung', 'option':''},
    'wisp': {'guide': 'Guide-ID', 'step': 'Schritt-ID', 'action':'Aktion', 'quest':'Quest-ID', 'condition':'Bedingungen', 'option':''},
    'polestar': {'guide':'Guide-ID', 'step':'Schritt', 'action':'Aktion', 'quest':'Quest-ID', 'condition':'Aktion-Bedingung', 'option':'Optional'},
    'rxp': {'guide':'Guide', 'step':'Schritt', 'action':'Aktion', 'quest':'Quest-ID', 'condition':'Bedingung', 'option':''},
}
ACTIONS = {'accept':'accept','turnin':'turnin','complete':'complete',
           'annehmen':'accept','abgeben':'turnin','questziel':'complete',
           'kill':'complete','collect':'complete','interact':'complete'}


def parse_rows(path: str | Path, source: str, guide: str | None = None):
    """Get normalized quest references, retaining unresolved conditions."""
    if source not in HEADERS:
        raise ValueError(f'unknown source {source}')
    spec = HEADERS[source]
    with Path(path).open(encoding='utf-8-sig', newline='') as fp:
        reader = csv.DictReader(fp, delimiter=';')
        required = [spec[k] for k in ('guide', 'step', 'action', 'quest')]
        if reader.fieldnames is None or any(k not in reader.fieldnames for k in required):
            raise ValueError(f'{source}: missing CSV columns {required}')
        for row in reader:
            g = row.get(spec['guide'],'').strip()
            if guide is not None and g != guide:continue
            if 'aoe' in g.lower():continue
            op = ACTIONS.get(row.get(spec['action'],'').strip().lower())
            if not op:continue
            raw_id = row.get(spec['quest'],'').strip()
            if not re.fullmatch(r'\d+',raw_id) or int(raw_id) <= 0:continue
            cond = (row.get(spec['condition'],'') if spec['condition'] else '') or ''
            opt = (row.get(spec['option'],'') if spec['option'] else '') or ''
            yield {'source': source,'guide':g,'step':row.get(spec['step'],''),
                   'action':op,'quest_id':int(raw_id),
                   'condition':cond.strip(),'optional':str(opt).lower() in ('1','true','yes')}


def coverage(rows):
    rows = list(rows)
    c = Counter(row['action'] for row in rows)
    cond = [r for r in rows if r['condition']]
    return {
        'source':rows[0]['source'] if rows else None,
        'guide_count':len(set(r['guide'] for r in rows)),
        'actions':dict(sorted(c.items())),
        'unique_quest_ids':len(set(r['quest_id'] for r in rows)),
        'conditionally_scoped':len(cond),
        'optional_actions':sum(r['optional'] for r in rows),
        'unresolved_conditions':bool(cond),
        'game_time_validated':False,
        'note':'Quest references only: conditional steps, execution order, objective sites, mob data and travel require verification.'
    }