"""NPC-health reference import and provenance-preserving world enrichment.

QuestieDB's *raw* foreverNpcDB.lua currently contains old min/max NPC HP.
The public QuestieDB API deliberately returns 0/1 compatibility placeholders;
never import HP using those getters. Raw reference values are not proven
Forever-server observations. No Questie data is bundled with this MIT project.
"""
from __future__ import annotations

import argparse
import copy
import json
import math
from pathlib import Path
import re
from urllib.request import Request, urlopen

QUESTIE_REVISION = 'f844007b329b72f18cf2bda3d87540012294732a'
QUESTIE_PATH = 'data/Forever/foreverNpcDB.lua'
QUESTIE_URL = f'https://raw.githubusercontent.com/Questie/QuestieDB/{QUESTIE_REVISION}/{QUESTIE_PATH}'
QUESTIE_PAGE = f'https://github.com/Questie/QuestieDB/blob/{QUESTIE_REVISION}/{QUESTIE_PATH}'
SOURCE_PRIORITY = {'questiedb_classic_reference': 10, 'wowhead_forever': 30,
                   'foreverdb_observed': 40, 'forever_verified': 50}
ROW = re.compile(r'^\[(\d+)\]\s*=\s*(\{.*\}),?\s*$')


def split_lua_fields(value: str) -> list[str]:
    """Parse top-level fields with Lua single/double quotes and nested tables."""
    if not value.startswith('{') or not value.endswith('}'):
        raise ValueError('Expected one Lua table')
    fields, buf, depth, quote, escape = [], [], 0, None, False
    for c in value[1:-1]:
        if quote:
            buf.append(c)
            if escape:
                escape = False
            elif c == '\\':
                escape = True
            elif c == quote:
                quote = None
        elif c in "\"'":
            quote = c
            buf.append(c)
        elif c == '{':
            depth += 1
            buf.append(c)
        elif c == '}':
            depth -= 1
            if depth < 0:
                raise ValueError('Unbalanced Lua row')
            buf.append(c)
        elif c == ',' and depth == 0:
            fields.append(''.join(buf).strip())
            buf = []
        else:
            buf.append(c)
    if quote or depth != 0:
        raise ValueError('Unbalanced Lua string or table')
    fields.append(''.join(buf).strip())
    return fields


def _unquote_lua(value: str) -> str:
    if len(value) < 2 or value[0] not in "\"'" or value[0] != value[-1]:
        raise ValueError('Invalid NPC name')
    # Keep deliberately minimal: names are only labels, never executed.
    return value[1:-1].replace('\\'+value[0], value[0]).replace('\\\\', '\\')


def parse_npc_references(raw: str, *, max_level: int = 33) -> dict:
    """Extract NPCs overlapping levels 1..max_level; discard sentinel HP 0/1.

    Values are raw QuestieDB health *references*, which may be inherited from
    Classic. Min/max values correspond to minimum/maximum NPC levels.
    """
    if max_level < 1:
        raise ValueError('max_level must be positive')
    npcs = {}
    discarded_hp = 0
    eligible = 0
    for line in raw.splitlines():
        match = ROW.match(line)
        if not match:
            continue
        fields = split_lua_fields(match.group(2))
        if len(fields) < 5:
            raise ValueError(f'Incomplete NPC row {match.group(1)}')
        try:
            low_hp, high_hp = int(fields[1]), int(fields[2])
            low_level, high_level = int(fields[3]), int(fields[4])
        except ValueError as exc:
            raise ValueError(f'Malformed NPC stats {match.group(1)}') from exc
        if low_level > max_level or high_level < 1:
            continue
        eligible += 1
        if not (low_level >= 1 and high_level >= low_level and 1 < low_hp <= high_hp):
            discarded_hp += 1
            continue
        npcs[match.group(1)] = {
            'name': _unquote_lua(fields[0]),
            'min_level': low_level, 'max_level': high_level,
            'min_hp': low_hp, 'max_hp': high_hp,
            'rank': int(fields[5]) if len(fields)>5 and re.fullmatch(r'-?\d+',fields[5]) else None,
        }
    return {'schema': 1, 'source': 'questiedb_classic_reference',
            'source_url': QUESTIE_PAGE, 'source_revision': QUESTIE_REVISION,
            'max_import_level': max_level, 'eligible_npc_count': eligible,
            'discarded_invalid_hp': discarded_hp, 'npcs': npcs}


def _validate_overlay(overrides: dict, npc_id: int, npc_level: int) -> dict | None:
    entries = overrides.get(str(npc_id), {}).get(str(npc_level))
    if entries is None:
        return None
    if isinstance(entries, dict):
        entries = [entries]
    if not isinstance(entries, list) or not entries:
        raise ValueError('Override entries must be a nonempty object or array')
    checked = []
    for item in entries:
        hp, source, url = item.get('hp'), item.get('source'), item.get('url')
        if source not in SOURCE_PRIORITY or source == 'questiedb_classic_reference':
            raise ValueError('Override must be a named measured/third-party source')
        if isinstance(hp, bool) or not isinstance(hp, int) or hp <= 1 or not isinstance(url,str) or not url.startswith('https://'):
            raise ValueError('Override HP must be >1 and have an HTTPS citation')
        checked.append(item)
    # Stable source precedence even if inputs are supplied in reverse order.
    chosen = max(enumerate(checked),key=lambda pair:(SOURCE_PRIORITY[pair[1]['source']],-pair[0]))[1]
    return {'hp': chosen['hp'], 'source': chosen['source'], 'source_url': chosen['url'],
            'npc_id': npc_id, 'npc_level': npc_level, 'method': 'specific_level_override'}


def hp_for_npc(snapshot: dict, npc_id: int, mob_level: int,
               overrides: dict | None = None) -> dict | None:
    """Prefer observed Forever, then Wowhead Forever, then Classic HP.

    An override applies ONLY to its specified NPC level. Values for other
    levels cannot be extrapolated. Classic min/max interpolation is labelled.
    """
    if not isinstance(mob_level, int) or mob_level < 1:
        raise ValueError('A real NPC level is required to choose HP')
    entry = snapshot['npcs'].get(str(npc_id))
    options = (overrides or {}).get(str(npc_id), {})
    override = _validate_overlay(overrides or {}, npc_id, mob_level) if str(mob_level) in options else None
    if override is not None:
        return override
    if not entry or not entry['min_level'] <= mob_level <= entry['max_level']:
        return None
    lo, hi = entry['min_level'], entry['max_level']
    if lo == hi:
        hp, method = entry['min_hp'], 'raw_fixed_level'
    else:
        ratio = (mob_level-lo)/(hi-lo)
        hp = math.floor(entry['min_hp']+ratio*(entry['max_hp']-entry['min_hp'])+0.5)
        method = 'linear_interpolation_between_classic_level_endpoints'
    return {'hp': hp, 'source': 'questiedb_classic_reference',
            'source_url': snapshot['source_url'], 'source_revision': snapshot['source_revision'],
            'npc_id': npc_id, 'npc_level': mob_level, 'method': method}


def enrich_world(world_doc: dict, snapshot: dict, overrides: dict | None = None) -> tuple[dict,dict]:
    """Fill only missing HP in a World JSON; never overwrite known measurements.

    Mob mapping keys may be numeric NPC IDs or 'npc:NNN', alternatively a
    temporary npc_id field is allowed and removed before World construction.
    Existing XP, levels and kill time are NOT modified or invented.
    """
    world = copy.deepcopy(world_doc)
    provenance = world.setdefault('mob_provenance', {})
    counts = {'imported': 0, 'preserved_existing': 0,
              'unresolved': [], 'sources': {}}
    for key, mob in world['mobs'].items():
        key_match = re.fullmatch(r'(?:npc:)?(\d+)', str(key))
        npc_id = mob.pop('npc_id', None)
        if npc_id is None and key_match:
            npc_id = int(key_match.group(1))
        if mob.get('hp') is not None:
            counts['preserved_existing'] += 1
            continue
        if npc_id is None or not isinstance(mob.get('level'), int):
            counts['unresolved'].append({'mob': key, 'reason': 'missing npc_id or specific mob level'})
            continue
        choice = hp_for_npc(snapshot, int(npc_id), mob['level'], overrides)
        if choice is None:
            counts['unresolved'].append({'mob': key, 'npc_id': int(npc_id), 'reason': 'no health reference for this NPC level'})
            continue
        mob['hp'] = choice['hp']
        provenance[key] = {k:v for k,v in choice.items() if k!='hp'}
        counts['sources'][choice['source']] = counts['sources'].get(choice['source'],0)+1
        counts['imported'] += 1
    return world, counts


def download_npc_db(cache: Path) -> str:
    """Download a pinned source locally; do not commit QuestieDB data to MIT repo."""
    cache.parent.mkdir(parents=True,exist_ok=True)
    if not cache.exists():
        with urlopen(Request(QUESTIE_URL,headers={'User-Agent':'ForeverRoutes/0.4'}),timeout=60) as response:
            data = response.read(5_000_001)
        if len(data) < 1_000_000 or len(data) > 5_000_000:
            raise ValueError('Downloaded NPC source unexpected size')
        cache.write_bytes(data)
    return cache.read_text(encoding='utf-8')


def cli(argv=None):
    p=argparse.ArgumentParser(description='Import Classic fallback NPC health for Forever 1-30 (no invented values)')
    p.add_argument('--npc-source',type=Path,help='Local copy of QuestieDB data/Forever/foreverNpcDB.lua')
    p.add_argument('--fetch',action='store_true',help='Fetch pinned QuestieDB raw file over HTTPS')
    p.add_argument('--out',type=Path,required=True,help='Write a versioned local health snapshot')
    p.add_argument('--max-level',type=int,default=33)
    p.add_argument('--world',type=Path,help='Optional world JSON with missing mob HP to enrich')
    p.add_argument('--world-out',type=Path,help='Output path for enriched world JSON')
    p.add_argument('--overrides',type=Path,help='Wowhead/ForeverDB measured HP overlay by NPC ID and level')
    args=p.parse_args(argv)
    if args.fetch == bool(args.npc_source):
        p.error('Exactly one of --fetch and --npc-source required')
    if args.world and not args.world_out or args.world_out and not args.world:
        p.error('--world and --world-out must be supplied together')
    raw=download_npc_db(Path('.cache/questie')/f'{QUESTIE_REVISION}_npc.lua') if args.fetch else args.npc_source.read_text(encoding='utf-8')
    snapshot=parse_npc_references(raw,max_level=args.max_level)
    args.out.parent.mkdir(parents=True,exist_ok=True)
    args.out.write_text(json.dumps(snapshot,ensure_ascii=False,indent=2)+'\n',encoding='utf8')
    print(json.dumps({'imported':len(snapshot['npcs']),'eligible':snapshot['eligible_npc_count'],
                      'invalid_hp':snapshot['discarded_invalid_hp'], 'source':snapshot['source']}))
    if args.world:
        world_doc=json.loads(args.world.read_text(encoding='utf8'))
        overrides=json.loads(args.overrides.read_text(encoding='utf8')) if args.overrides else {}
        enriched, stats=enrich_world(world_doc,snapshot,overrides)
        args.world_out.parent.mkdir(parents=True,exist_ok=True)
        args.world_out.write_text(json.dumps(enriched,ensure_ascii=False,indent=2)+'\n',encoding='utf8')
        print(json.dumps(stats))


if __name__=='__main__':
    cli()