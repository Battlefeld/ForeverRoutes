"""Build actionable, provenance-tagged reference facts from QuestieDB Forever.

No Lua execution. Never silently turn a reference into a measured Forever fact.
Includes quests, starters, prerequisites, objective target IDs, NPC spawns,
HP, quest item sources, Classic item-drop reference rates and XP formulas.
Walkable-path graph and combat DPS are separate mandatory simulation inputs.
"""
from __future__ import annotations

import argparse
from collections import Counter
from datetime import datetime, timezone
import json
from pathlib import Path
import re
from urllib.request import Request, urlopen

QUESTIE_SHA = "f844007b329b72f18cf2bda3d87540012294732a"
from .xp_reference import XP_TO_NEXT, XP_LEVEL_SOURCE, KILL_XP_SOURCE

UPSTREAM = 'Questie/QuestieDB'
SOURCE_FILES = {
    'quests': 'data/Forever/foreverQuestDB.lua',
    'npc_positions': 'data/Forever/foreverNpcDB.lua',
    'npc_health': 'data/Classic/classicNpcDB.lua',
    'items': 'data/Forever/foreverItemDB.lua',
    'objects': 'data/Forever/foreverObjectDB.lua',
    'quest_xp': 'support/Forever/QuestXP/xpDB-classic.lua',
    'item_drop_rates': 'support/Forever/DropTables/classicItemDrops.lua',
}

class ParseError(ValueError):
    pass

class _Lua:
    """Strict recursive reader for numeric/string/nested Lua table literals only."""
    def __init__(self, s: str):
        self.s=s; self.i=0
    def ws(self):
        while self.i<len(self.s) and self.s[self.i].isspace(): self.i+=1
    def take(self, c):
        self.ws()
        if self.i<len(self.s) and self.s[self.i]==c:
            self.i+=1; return True
        return False
    def value(self):
        self.ws()
        if self.i>=len(self.s): raise ParseError('Unexpected end of Lua literal')
        c=self.s[self.i]
        if c=='{': return self.table()
        if c in ('"', "'"):
            self.i+=1; out=[]
            while self.i<len(self.s):
                char=self.s[self.i]; self.i+=1
                if char==c: return ''.join(out)
                if char=='\\':
                    if self.i==len(self.s): raise ParseError('Truncated escape')
                    esc=self.s[self.i]; self.i+=1
                    out.append({'n':'\n','r':'\r','t':'\t'}.get(esc,esc))
                else: out.append(char)
            raise ParseError('Unterminated string')
        m=re.match(r'-?(?:\d+(?:\.\d*)?|\.\d+)(?:[eE][+-]?\d+)?',self.s[self.i:])
        if m:
            self.i+=len(m.group());n=m.group()
            return float(n) if any(c in n for c in '.eE') else int(n)
        m=re.match(r'(?:true|false|nil)(?!\w)',self.s[self.i:])
        if m:
            self.i+=len(m.group())
            return {'true':True,'false':False,'nil':None}[m.group()]
        raise ParseError(f'Unsupported Lua token at {self.i}: {self.s[self.i:self.i+30]}')
    def table(self):
        if not self.take('{'):raise ParseError('Expected opening brace')
        sequential=[]; indexed={}
        while True:
            self.ws()
            if self.take('}'): break
            if self.take('['):
                key=self.value()
                if not self.take(']') or not self.take('='):
                    raise ParseError('Malformed indexed Lua key')
                indexed[key]=self.value()
            else:
                sequential.append(self.value())
            if self.take('}'):break
            if not self.take(',') and not self.take(';'):
                raise ParseError('Expected comma in Lua table')
        if indexed:
            for i,v in enumerate(sequential,1):
                indexed.setdefault(i,v)
            return indexed
        return sequential


def parse_lua(raw: str):
    parser=_Lua(raw);result=parser.value();parser.ws()
    if parser.i!=len(raw):raise ParseError('Trailing Lua tokens')
    return result


def arr(value):
    return value if isinstance(value,list) else []


def table_refs(value):
    if isinstance(value,int): return [value] if value>0 else []
    if isinstance(value,(list,dict)):
        seq=value.values() if isinstance(value,dict) else value
        out=[]
        for v in seq: out.extend(table_refs(v))
        return out
    return []


def spawns(value):
    """[zone] = {{x,y}, ...}; reject sentinels and invalid map coordinates."""
    result={}
    if not isinstance(value,dict): return result
    for zone,points in value.items():
        if not isinstance(zone,int) or not isinstance(points,list): continue
        clean=[]
        for point in points:
            if isinstance(point,list) and len(point)>=2:
                x,y=point[:2]
                if all(isinstance(n,(int,float)) for n in (x,y)) and 0<=x<=100 and 0<=y<=100:
                    clean.append([x,y])
        if clean: result[str(zone)]=clean
    return result


def parse_rows(raw: str) -> dict[int,list]:
    out={}
    for line in raw.splitlines():
        m = re.match(r"^\[(\d+)\]\s*=\s*(\{.*\}),?\s*$", line)
        if not m: continue
        id_ = int(m.group(1))
        row = parse_lua(m.group(2))
        if not isinstance(row, list): raise ParseError(f"Expected indexed row for {id_}")
        out[id_] = row
    return out


def at(row, ix, default=None):
    return row[ix] if len(row)>ix else default


def xp_records(raw):
    return {int(m.group(1)):{'level':int(m.group(2)),'xp':int(m.group(3))}
            for m in re.finditer(r'^\s*\[(\d+)\]\s*=\s*\{(\d+)\s*,\s*(\d+)\}',raw,re.M)}


def drop_chances(raw):
    """Parse the explicitly Classic/Wowhead-sampled item-to-NPC rate table.

    Does NOT interpret the percent as a confirmed Forever loot probability.
    """
    result={};active=None
    for line in raw.splitlines():
        start=re.match(r'^\s{2,}\[(\d+)\]\s*=\s*\{\s*--',line)
        if start:
            active=int(start.group(1));result[active]={};continue
        if active is None:continue
        if re.match(r'^\s*\},?',line):active=None;continue
        m=re.match(r'^\s*\[(\d+)\]\s*=\s*(\d+(?:\.\d+)?),?\s*$',line)
        if m:
            value=float(m.group(2))
            if 0<value<=100:result[active][int(m.group(1))]=round(value/100,7)
    return result


def _source(path):
    return {'kind':'classic_reference' if ('classic' in path.lower()) else 'questiedb_forever_client_seed',
            'repository':UPSTREAM,'revision':QUESTIE_SHA,'path':path}


def build_catalog(raw: dict[str,str], *, max_player_level=30, max_mob_level=35,
                  quest_overlays=None, npc_overlays=None):
    quest_src=parse_rows(raw['quests']);npc_src=parse_rows(raw['npc_positions']);hp_src=parse_rows(raw['npc_health'])
    object_src=parse_rows(raw['objects']);item_src=parse_rows(raw['items'])
    xp=xp_records(raw['quest_xp']);rates=drop_chances(raw['item_drop_rates'])
    # Include every quest that could be picked up before cap, without assuming
    # a displayed quest level is the minimum character level.
    selected={id_:row for id_,row in quest_src.items() if isinstance(at(row,3),int)
              and 1<=at(row,3)<=max_player_level and isinstance(at(row,4),int)}
    relevant_npcs=set(); relevant_objects=set(); relevant_items=set()
    quests={}
    for id_,row in sorted(selected.items()):
        starts=arr(at(row,1));ends=arr(at(row,2));objs=arr(at(row,9));
        s_npcs=table_refs(at(starts,0)); e_npcs=table_refs(at(ends,0));
        s_objects=table_refs(at(starts,1));e_objects=table_refs(at(ends,1));
        s_items=table_refs(at(starts,2))
        relevant_npcs.update(s_npcs+e_npcs);relevant_objects.update(s_objects+e_objects);relevant_items.update(s_items)
        objectives=[]
        for i,targets in enumerate(objs):
            typ=['npc','object','item','reputation','kill_credit','spell'][i] if i<6 else f'other_{i}'
            for entry in arr(targets):
                # The entry shape is {targetID, optionalText, optionalIconID}.
                # Only the first element identifies the creature/object/item.
                if typ in ('npc', 'object', 'item') and isinstance(entry,list):
                    ids = table_refs(entry[0]) if entry else []
                else:
                    ids = table_refs(entry)
                if not ids:continue
                # Quantity is NOT encoded in QuestieDB target IDs. Do not assume one.
                objectives.append({'kind':typ,'ids':ids,'required_count':None,'count_source':'unresolved'})
                if typ in ('npc','kill_credit'): relevant_npcs.update(ids)
                elif typ=='object':relevant_objects.update(ids)
                elif typ=='item':relevant_items.update(ids)
        quest={
            'id':id_, 'name':at(row,0), 'required_level':at(row,3), 'quest_level':at(row,4),
            'required_races_mask':at(row,5), 'required_classes_mask':at(row,6),
            'started_by':{'npc':s_npcs,'object':s_objects,'item':s_items},
            'finished_by':{'npc':e_npcs,'object':e_objects},
            'prequest_all':table_refs(at(row,11)), 'prequest_any':table_refs(at(row,12)),
            'objectives':objectives,'zone_or_sort':at(row,15),
            'xp': {'value':xp[id_]['xp'],'source':'classic_quest_xp'} if id_ in xp else None,
            'data_source':'questiedb_forever_classic_seed'
        }
        quests[str(id_)]=quest
    mobs={}
    for id_,row in sorted(npc_src.items()):
        lo=at(row,3);hi=at(row,4)
        if not isinstance(lo,int) or not isinstance(hi,int):continue
        if lo>max_mob_level and id_ not in relevant_npcs:continue
        ref=hp_src.get(id_,[]); hp_lo=at(ref,1);hp_hi=at(ref,2)
        okhp=(isinstance(hp_lo,int) and isinstance(hp_hi,int) and hp_lo>1 and hp_hi>=hp_lo)
        position=spawns(at(row,6));
        mobs[str(id_)]={
            'id':id_,'name':at(row,0),'min_level':lo,'max_level':hi,'rank':at(row,5),
            'hp_min':hp_lo if okhp else None,'hp_max':hp_hi if okhp else None,
            'hp_source':'questiedb_classic_raw' if okhp else 'unknown',
            'spawns':position,'spawn_source':'questiedb_forever_raw',
            'friendly_to_faction':at(row,12),'npc_flags':at(row,14),
            'faction_id':at(row,11),
            'potential_grind_candidate':(lo<=max_mob_level and at(row,5)==0
                and at(row,12) is None and bool(position) and okhp),
            'kill_xp_model':'cmangos_classic_solo_level_formula',
            'is_critter_or_no_xp':'unverified'
        }
    objects={}
    for id_ in sorted(relevant_objects):
        row=object_src.get(id_)
        if row is None:continue
        objects[str(id_)]={'id':id_,'name':at(row,0),'spawns':spawns(at(row,3)),'spawn_source':'questiedb_forever_raw'}
    items={}
    for id_ in sorted(relevant_items):
        row=item_src.get(id_)
        if row is None:continue
        drops={str(npc):{'chance':rate,'source':'classic_wowhead_sample'} for npc,rate in sorted(rates.get(id_,{}).items())}
        npc_drops=table_refs(at(row,1));obj_drops=table_refs(at(row,2))
        relevant_npcs.update(npc_drops);relevant_objects.update(obj_drops)
        items[str(id_)]={'id':id_,'name':at(row,0),'npc_sources':npc_drops,'object_sources':obj_drops,
                         'drop_chance_by_npc':drops,'drop_source':'questiedb_classic_wowhead'}
    # All required prerequisite quest IDs outside level window remain visible.
    for obs in quest_overlays or []:
        id_=str(int(obs['quest_id']));
        entry=quests.setdefault(id_,{'id':int(id_),'overlay_only':True})
        for field,fact in obs.get('fields',{}).items():
            if fact.get('source') in ('foreverdb_observed','foreverdb_client','wowhead_forever') and fact.get('url','').startswith('https://'):
                if field=='xp_by_player_level':
                    entry['xp_observed_by_player_level']=fact['value']
                elif field=='quest_level': entry['quest_level']=fact['value']
                elif field=='required_level': entry['required_level']=fact['value']
                entry.setdefault('overlays',{})[field]=fact
    for obs in npc_overlays or []:
        npc=mobs.get(str(int(obs['npc_id'])))
        if not npc:continue
        for field,fact in obs.get('fields',{}).items():
            if field.startswith('hp_at_level_') and fact.get('source') in ('foreverdb_observed','wowhead_forever') and fact.get('url','').startswith('https://'):
                npc.setdefault('hp_overrides',{})[field[12:]]=fact
    counters=Counter()
    counters['quest_count']=len(quests)
    counters['classic_quest_xp']=sum(bool(q.get('xp')) for q in quests.values())
    counters['quests_with_objectives']=sum(bool(q.get('objectives')) for q in quests.values())
    counters['objectives_with_unknown_count']=sum(len(q.get('objectives',[])) for q in quests.values())
    counters['missing_prequest_ids']=sum(sum(ref not in quest_src for ref in q.get('prequest_all',[])+q.get('prequest_any',[])) for q in quests.values())
    counters['npc_count']=len(mobs)
    counters['npc_hp_reference']=sum(x['hp_min'] is not None for x in mobs.values())
    counters['npc_with_spawn']=sum(bool(x['spawns']) for x in mobs.values())
    counters['potential_grind_candidates']=sum(x['potential_grind_candidate'] for x in mobs.values())
    counters['item_quest_objective_count']=len(items)
    counters['item_with_classic_drop_rate']=sum(bool(x['drop_chance_by_npc']) for x in items.values())
    counters['object_spawn_count']=sum(bool(x['spawns']) for x in objects.values())
    return {'schema':'foreverroutes.catalog.1','sources':{
                k:{'revision':QUESTIE_SHA,'path':v,'repository':UPSTREAM} for k,v in SOURCE_FILES.items()},
            'xp_sources':{'level':XP_LEVEL_SOURCE,'mob':KILL_XP_SOURCE},
            'level_xp_classic':{str(k):v for k,v in XP_TO_NEXT.items()},
            'metadata':{'player_cap':max_player_level,'npc_reference_cap':max_mob_level,
              'temporary_fallback':True,'level_30_model_requires_travel_graph':True,
              'quest_counts_unknown':True,'mob_xp_is_formula':True},
            'stats':dict(counters),'quests':quests,'npcs':mobs,'objects':objects,'items':items}


def fetch_sources(cache:Path, *,revision=QUESTIE_SHA):
    if not re.fullmatch(r'[0-9a-f]{40}',revision):raise ValueError('Must pin SHA')
    cache.mkdir(parents=True,exist_ok=True)
    raw={}
    for kind,path in SOURCE_FILES.items():
        p=cache/(revision+'_'+kind+'.lua')
        if not p.exists():
            url=f'https://raw.githubusercontent.com/{UPSTREAM}/{revision}/{path}'
            with urlopen(Request(url,headers={'User-Agent':'ForeverRoutes/1.0'}), timeout=60) as response:
                data=response.read(6_000_000)
            if len(data)<1000: raise ValueError(f'Suspect source {url}')
            p.write_bytes(data)
        raw[kind]=p.read_text(encoding='utf-8')
    return raw


def add_guide_id_candidates(catalog: dict, candidate_doc: dict) -> dict:
    """Preserve unknown guide quest IDs for later validation, NEVER as usable quests."""
    ids = {}
    for source, guides in candidate_doc.get('sources', {}).items():
        if not isinstance(guides, dict): raise ValueError('Guide candidate source must be a mapping')
        for guide, raw_ids in guides.items():
            for raw in raw_ids:
                qid = int(raw)
                if qid <= 0: continue
                ids.setdefault(qid, set()).add(source)
    unresolved = 0
    for qid, sources in ids.items():
        key = str(qid)
        if key not in catalog['quests']:
            catalog['quests'][key] = {
                'id': qid,
                'source':'guide_id_candidate_only',
                'candidate_only': True,
                'source_guides': sorted(sources),
                'required_level': None,
                'xp': None,
                'objectives': [],
                'not_actionable': True,
            }
            unresolved += 1
    catalog['stats']['guide_only_unresolved_quest_ids'] = unresolved
    catalog['stats']['total_quest_ids_including_unresolved_candidates'] = len(catalog['quests'])
    return catalog


def run(argv=None):
    parser=argparse.ArgumentParser(description='Build a Questie Classic-reference input catalog for level 1–30')
    parser.add_argument('--cache',type=Path,default=Path('.cache/questie'))
    parser.add_argument('--output',type=Path,default=Path('.generated/level_1_30_catalog.json'))
    parser.add_argument('--overlay',type=Path,action='append',default=[])
    parser.add_argument('--npc-overlay',type=Path,action='append',default=[])
    parser.add_argument('--guide-ids',type=Path,help='Optional guide numeric ID candidates; never interpreted as fully defined quests')
    args=parser.parse_args(argv)
    q=[];n=[]
    for p in args.overlay:q+=json.loads(p.read_text(encoding='utf-8')).get('observations',[])
    for p in args.npc_overlay:n+=json.loads(p.read_text(encoding='utf-8')).get('observations',[])
    catalog=build_catalog(fetch_sources(args.cache),quest_overlays=q,npc_overlays=n)
    if args.guide_ids:
        candidates=json.loads(args.guide_ids.read_text(encoding='utf-8'))
        add_guide_id_candidates(catalog,candidates)
    args.output.parent.mkdir(parents=True,exist_ok=True)
    args.output.write_text(json.dumps(catalog,ensure_ascii=False,separators=(',',':')),encoding='utf-8')
    print(json.dumps({'output':str(args.output),'stats':catalog['stats']},ensure_ascii=False))
    return catalog

if __name__=='__main__':run()