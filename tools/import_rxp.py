#!/usr/bin/env python3
"""Package pinned, ordinary RXP chapters unchanged; build a labelled quest projection.

Usage: python tools/import_rxp.py /path/to/pinned/RXPGuides [--license PATH]
Original instructions and all dynamic guards stay in chapter bodies. The event list
is only a quest-workload projection, never an exact RXP travel/trainer simulation.
"""
import argparse, gzip, hashlib, json, pathlib, re, shutil

ROOT = pathlib.Path(__file__).resolve().parents[1]
WARNINGS = set()

def matches(expr, character):
    if not expr.strip(): return True
    known = {'Alliance','Horde','Human','Orc','Dwarf','NightElf','Undead','Scourge',
             'Tauren','Gnome','Troll','Skyborne','Warrior','Paladin','Hunter','Rogue',
             'Priest','Shaman','Mage','Warlock','Druid','None','none','skip','sod','SoD','Sod','era'}
    side = character.get('faction') or ('H' if character['race'] in ['Orc','Troll','Undead','Tauren'] else 'A')
    truth = {character['race'], character['class'], 'Alliance' if side=='A' else 'Horde','era'}
    if character['race']=='Undead': truth.add('Scourge')
    for part in expr.split('/'):
        yes = True
        for token in part.split():
            token = token.strip('()')
            key = token.lstrip('!')
            if key not in known:
                WARNINGS.add('Unknown source condition excluded from projection: '+expr)
                yes=False
                continue
            yes &= (key not in truth) if token.startswith('!') else (key in truth)
        if yes: return True
    return False

def strip_condition(line, character):
    if '<<' not in line: return line
    line, expr = line.split('<<',1)
    expr = expr.split('--',1)[0].strip()
    return line.rstrip() if matches(expr,character) else None

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('source',type=pathlib.Path)
    parser.add_argument('--license',type=pathlib.Path)
    parser.add_argument('--check',action='store_true',help='verify catalog content without writing it')
    args=parser.parse_args()
    revision=json.loads((ROOT/'data/sources/lock.json').read_text())['rxp']
    # This cache directory is named after a verified pinned checkout. A regular
    # checkout is checked by git without fetching or changing it.
    import subprocess
    if args.source.name != 'rxp-'+revision:
        actual=subprocess.check_output(['git','-C',str(args.source),'rev-parse','HEAD'],text=True).strip()
        if actual!=revision: raise ValueError('Source checkout does not match lock.json')
    files=sorted((args.source/'Guides/forever').glob('*.lua'))
    files=[p for p in files if 'Mage-' not in p.name and 'ADV-' not in p.name]
    files += [args.source/'Guides/Forever/RestedXP-Skyborne.lua']
    chapters={}; sources=[]
    for path in files:
        raw=path.read_bytes(); text=raw.decode('utf-8-sig')
        rel=path.relative_to(args.source).as_posix()
        sources.append({'path':rel,'sha256':hashlib.sha256(raw).hexdigest()})
        for m in re.finditer(r'RegisterGuide\(\[\[([\s\S]*?)\]\]',text):
            body=m[1]; header=body.split('\nstep',1)[0]
            name=re.search(r'^#name\s+(.+)$',header,re.M).group(1).strip()
            if name in chapters: raise ValueError('Duplicate chapter '+name)
            chapters[name]={'name':name,'file':rel,'line':text[:m.start(1)].count('\n')+1,
                'body':body,'header':header}
    roster=[]
    for path in sorted((ROOT/'data/routes').glob('*.json')):
        if path.name.endswith('-dungeons.json'): continue
        doc=json.loads(path.read_text());roster.append((path.stem,doc['character']))
    if len(roster)!=56: raise ValueError('Expected 56 independent roster entries')
    starts={'Human':'1-6 Northshire','Dwarf':'1-5 Coldridge Valley','Gnome':'1-5 Coldridge Valley',
            'NightElf':'1-6 Shadowglen','Orc':'1-6 Durotar','Troll':'1-6 Durotar',
            'Undead':'1-6 Tirisfal Glades','Tauren':'1-6 Mulgore','Skyborne':'1-9 Zephras Isle'}
    routes=[]
    for key,c in roster:
        chain=[];events=[];name=starts[c['race']]
        while True:
            if name in chain: raise ValueError('Cyclic #next for '+key)
            ch=chapters[name];chain.append(name)
            active=True;parallel=False;step_lines=[]
            def flush():
                if not step_lines: return
                # Keep dynamic instructions in the authoritative raw chapter;
                # projected parallel objectives wait until a later explicit step/turn-in.
                for line in step_lines:
                    a=re.match(r'^\.(accept|turnin|complete)\s+(\d+)(?:,(\d+))?',line)
                    if a and (a[1]!='complete' or not parallel):
                        event={'op':a[1],'quest':int(a[2])}
                        if a[1]=='complete' and a[3]:event['objective']=int(a[3])
                        events.append(event)
                    xp=re.match(r'^\.xp\s+(\d+)(?:\+(\d+))?(?:\s|$)',line)
                    if xp and not parallel:events.append({'op':'level','level':int(xp[1]),'xp':int(xp[2] or 0)})
            for raw in ch['body'].splitlines():
                line=raw.strip()
                if re.match(r'^step(?:\s|$)',line):
                    flush();step_lines=[];parallel=False
                    active=matches(line.split('<<',1)[1].split('--',1)[0] if '<<' in line else '',c)
                    continue
                if not active or line.startswith('--'): continue
                line=strip_condition(line,c)
                if line is None:continue
                if line.startswith('#hardcore'):active=False;step_lines=[];continue
                if line.startswith('#completewith'):parallel=True
                step_lines.append(line)
            flush()
            match=re.match(r'^(\d+)-(\d+) ',name)
            if match and int(match[2])>=20: break
            next_name=None
            for line in ch['header'].splitlines():
                if not line.startswith('#next '):continue
                line=strip_condition(line,c)
                if not line:continue
                for choice in line[6:].split(';'):
                    choice=choice.strip().split('\\')[-1]
                    if choice not in chapters:continue
                    head=chapters[choice]['header']
                    # A chapter-level << condition decides alternatives in #next lists.
                    restrictions=re.findall(r'^<<\s+(.+)$',head,re.M)
                    if all(matches(x,c) for x in restrictions):next_name=choice;break
                if next_name:break
            if not next_name:raise ValueError('No compatible #next for '+key+' after '+name)
            name=next_name
        routes.append({'key':key,'character':c,'target':20,'chapters':chain,'events':events,
            'projectionOnly':True})
    for ch in chapters.values():ch.pop('header')
    doc={'schema':'foreverroutes.rxp-fallback.v1','revision':revision,
         'source':'https://github.com/RestedXP/RXPGuides','license':'CC-BY-NC-SA-4.0',
         'sources':sources,'chapters':chapters,'routes':routes,'warnings':sorted(WARNINGS),
         'limitations':['Quest-workload projections retain source order but do not execute every dynamic RXP guard, money gate, trainer, travel or inventory instruction.',
            'Original chapter bodies preserve every instruction and are the authoritative in-game fallback; projection times are not exact RXP play times.',
            'Ordinary solo guides only; AoE and dungeon guide variants are not substituted.']}
    out=ROOT/'data/fallbacks';out.mkdir(parents=True,exist_ok=True)
    payload=(json.dumps(doc,ensure_ascii=False,separators=(',',':'))+'\n').encode()
    if args.check:
        if json.loads(gzip.decompress((out/'rxp.json.gz').read_bytes()))!=doc:raise ValueError('Shipped RXP catalog differs from pinned sources/roster')
    else:
        (out/'rxp.json.gz').write_bytes(gzip.compress(payload,mtime=0))
        if args.license:shutil.copyfile(args.license,out/'RXP-LICENSE.txt')
    print('Packaged',len(chapters),'unchanged RXP chapters and',len(routes),'character projections; bytes',len(payload))

if __name__=='__main__':main()
