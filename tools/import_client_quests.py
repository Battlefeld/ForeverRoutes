#!/usr/bin/env python3
"""Import ForeverRoutes quest observations from JSON or WoW SavedVariables without executing Lua."""
import argparse
import hashlib
import json
from pathlib import Path
import re
import time


def lua_string(text):
    match = re.search(r'^ForeverRoutesQuestExport\s*=\s*"', text, re.M)
    if not match:
        raise ValueError('No ForeverRoutesQuestExport found; log out with collector enabled first')
    i, out = match.end(), []
    escapes = {'n': '\n', 'r': '\r', 't': '\t', '\\': '\\', '"': '"', "'": "'", 'a': '\a', 'b': '\b', 'f': '\f', 'v': '\v'}
    while i < len(text):
        c = text[i]; i += 1
        if c == '"': return b''.join(out).decode('utf-8')
        if c != '\\': out.append(c.encode('utf-8')); continue
        if i >= len(text): break
        c = text[i]; i += 1
        if c.isdigit():
            digits = c
            while len(digits) < 3 and i < len(text) and text[i].isdigit():
                digits += text[i]; i += 1
            value = int(digits)
            if value > 255: raise ValueError('Invalid Lua byte escape')
            out.append(bytes([value]))
        elif c in escapes: out.append(escapes[c].encode('utf-8'))
        elif c == '\n': out.append(b'\n')
        else: raise ValueError('Unsupported Lua escape')
    raise ValueError('Unterminated export string')


def load(path):
    raw = Path(path).read_bytes()
    if len(raw) > 64 * 1024 * 1024: raise ValueError('Capture exceeds 64 MiB')
    text = raw.decode('utf-8-sig')
    doc = json.loads(text if text.lstrip().startswith('{') else lua_string(text))
    if doc.get('schema') != 1 or not isinstance(doc.get('captures'), dict):
        raise ValueError('Unsupported capture schema')
    for key, capture in doc['captures'].items():
        if not re.fullmatch(r'1\.60\.\d+\.\d+', capture.get('build', '')):
            raise ValueError('Not a Forever capture')
        if capture.get('locale') not in ('enUS', 'enGB', 'deDE', 'frFR', 'esES', 'esMX', 'itIT', 'ptBR', 'ruRU', 'koKR', 'zhCN', 'zhTW'):
            raise ValueError('Unknown capture locale')
        if capture.get('region') not in range(0, 6): raise ValueError('Invalid region')
        if key != capture['build'] + '/' + str(capture['region']) + '/' + capture['locale']: raise ValueError('Capture identity mismatch')
        for ident, quest in capture['quests'].items():
            if not ident.isdigit() or int(ident) != quest.get('id') or int(ident) <= 0: raise ValueError('Invalid quest identity')
            if not isinstance(quest.get('title'), str) or not quest['title']: raise ValueError('Missing quest title')
            if not isinstance(quest.get('seenAt'), int) or quest['seenAt'] <= 0: raise ValueError('Missing observation time')
    return doc


def merge(before, incoming):
    result = json.loads(json.dumps(before))
    for key, capture in incoming['captures'].items():
        old = result['captures'].setdefault(key, dict(capture, quests={}, turnins={}))
        for ident, quest in capture['quests'].items():
            prev = old['quests'].get(ident)
            if prev and prev['seenAt'] == quest['seenAt'] and prev != quest:
                raise ValueError('Conflicting quest observation at identical timestamp')
            if not prev or quest['seenAt'] >= prev['seenAt']: old['quests'][ident] = quest
        turns = list(old.get('turnins', {}).values()) + list(capture.get('turnins', {}).values())
        unique = {hashlib.sha256(json.dumps(t, sort_keys=True).encode()).hexdigest(): t for t in turns}
        old['turnins'] = {str(i+1): t for i, t in enumerate(sorted(unique.values(), key=lambda t: t['seenAt'])[-5000:])}
    return result


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('capture', type=Path, help='SavedVariables/ForeverRoutes.lua or exported JSON')
    p.add_argument('--output', type=Path, default=Path('data/sources/client-quests.json'))
    p.add_argument('--watch', action='store_true', help='Reimport every 60 seconds when SavedVariables changes')
    a = p.parse_args(); previous_hash = None
    while True:
        try:
            incoming = load(a.capture)
            digest = hashlib.sha256(json.dumps(incoming, sort_keys=True).encode()).hexdigest()
            if digest != previous_hash:
                old = load(a.output) if a.output.exists() else {'schema': 1, 'captures': {}}
                result = merge(old, incoming)
                a.output.parent.mkdir(parents=True, exist_ok=True)
                temp = a.output.with_suffix('.tmp'); temp.write_text(json.dumps(result, ensure_ascii=False, indent=2)+'\n')
                temp.replace(a.output)
                previous_hash = digest
                print('Imported quest observations:', sum(len(c['quests']) for c in result['captures'].values()), flush=True)
        except (ValueError, OSError, KeyError, TypeError) as exc:
            if not a.watch: raise
            print('Capture not imported; previous data retained: ' + str(exc), flush=True)
        if not a.watch: break
        time.sleep(60)

if __name__ == '__main__': main()
