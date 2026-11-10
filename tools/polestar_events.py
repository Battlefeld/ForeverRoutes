"""Polestar guide chapters (GPL-3.0, github.com/fitzforge/polestar) -> ForeverRoutes guide events for one class.
polestar_events.py CLASS RACE chapter.guide... > events.json   (only ids and order are taken, no guide text)"""
import json, re, sys

cls, race, files = sys.argv[1], sys.argv[2], sys.argv[3:]
events, chapters = [], []
for f in files:
    only = None
    title = levels = ''
    for raw in open(f, encoding='utf-8'):
        line = raw.split('--')[0].rstrip()
        if not line.strip() or line.lstrip().startswith('#'):
            continue
        if not raw.startswith(' '):
            word = line.split()[0]
            if word == 'step':
                only = None
            elif word == 'title':
                title = line[6:].strip()
            elif word == 'levels':
                levels = line[7:].strip()
            continue
        parts = line.split()
        op = parts[0]
        if op == 'only':
            spec = parts[1]
            neg = spec.startswith('!')
            names = set(spec.lstrip('!').split(','))
            hit = cls in names or race in names
            only = (not hit) if neg else hit
            continue
        if only is False:
            continue
        if op in ('accept', 'turnin', 'complete') and len(parts) > 1 and parts[1].isdigit():
            if 'optional' in parts[2:]:
                continue
            e = {'op': op, 'quest': int(parts[1])}
            events.append(e)
        elif op == 'level' and len(parts) > 1 and parts[1].isdigit():
            events.append({'op': 'level', 'level': int(parts[1])})
    chapters.append({'file': f.split('/')[-1], 'title': title, 'levels': levels})
json.dump({'events': events, 'chapters': chapters}, sys.stdout)
