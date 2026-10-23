"""Normalize locally supplied guide facts; retain guards and source line numbers."""
from __future__ import annotations
import csv,json,re
from pathlib import Path

ROOT=Path(__file__).resolve().parents[1]
INPUT=ROOT.parent/'route_inputs'
OUT=ROOT/'.comparison/events'

def csvrows(name):
    with (INPUT/name).open(encoding='utf-8-sig',newline='') as f:
        return list(csv.DictReader(f,delimiter=';'))

def save(name,doc):
    OUT.mkdir(parents=True,exist_ok=True)
    (OUT/(name+'.json')).write_text(json.dumps(doc,ensure_ascii=False,indent=1)+'\n')

def journey():
    chain=['H01D_FVR','H06D_FVR','H12B_FVR','H15S_FVR','H16B_FVR','H20SM_FVR','H21A_FVR','H22SB_FVR','H23S_FVR','H25SB_FVR','H25TN_FVR','H26A_FVR','H27SM_FVR','H28TN_FVR','H29HF_FVR']
    rows=csvrows('Journey_Quest_Routen_Analyse.csv');groups=[]
    for guide in chain:
        selected=[r for r in rows if r['guide_ids']==guide]
        if not selected:raise ValueError('Missing Journey guide '+guide)
        groups.append({'guide':guide,'title':selected[0]['guide_name'],'events':[{'t':'guide','title':selected[0]['guide_name']}]})
        last=None
        for r in selected:
            key=(r['schritt'],r['schritt_bedingung'])
            if key!=last:
                groups.append({'guide':guide,'step':key[0],'condition':key[1],'condition_format':'journey','events':[]});last=key
            if r['aktion'] in ['accept','complete','turnin'] and r['quest_id'].isdigit():
                groups[-1]['events'].append({'t':r['aktion'],'q':int(r['quest_id'])})
    save('journey',{'id':'journey','label':'Journey 4.00.02','coverage':[1,30],'groups':groups,'limits':['CSV omits non-quest travel, training, grind and inventory instructions','H12B default chosen; H12SF is an alternative','Classic Era flags provisionally enabled alongside FOREVER; requires runtime validation']})

def wisp():
    rs=[r for r in csvrows('ForeverWisp_Quest_Routen_Analyse.csv') if r['Guide-ID']=='forever-orc-troll-1-20']
    groups=[];ops={'Annehmen':'accept','Abgeben':'turnin','Questziel':'complete'}
    for r in rs:
        if r['Aktion'] not in ops:continue
        cond=json.loads(r['Bedingungen']) if r['Bedingungen'] else {}
        groups.append({'guide':r['Guide-ID'],'step':r['Schritt-ID'],'condition_format':'wisp','condition':cond,'events':[{'t':ops[r['Aktion']],'q':int(r['Quest-ID'])}]})
    save('wisp',{'id':'wisp','label':'ForeverWisp 0.32.20-beta.1','coverage':[1,20],'groups':groups,'limits':['Optional grinding and suggested-level messages excluded','Only class-compatible quest records are scored','Complete actions represent the quest workload, not a measured combat recording']})

def rxp():
    rdir=ROOT.parent/'RXPGuides/Guides/forever';guides={}
    allowed=['Horde-01-12_Durotar.lua','Horde-01-14_Undead.lua','Horde-12-22_Barrens.lua']
    for name in allowed:
        raw=(rdir/name).read_text(encoding='utf-8-sig')
        for m in re.finditer(r'RXPGuides\.RegisterGuide\(\[\[([\s\S]*?)\]\]\)',raw):
            code=m.group(1);title=re.search(r'^#name\s+(.+)$',code,re.M).group(1).strip();guides[title]=(name,code)
    # RXP's explicit class-specific #next chain for Orc Warlock.
    chain=['1-6 Durotar','6-10 Durotar','10-12 Durotar','10-12 Tirisfal (Orc/Troll)','12-14 Silverpine Forest','12-17 The Barrens','17-22 Stonetalon/Barrens/Ashenvale']
    groups=[]
    for title in chain:
        name,code=guides[title]
        groups.append({'guide':title,'events':[{'t':'guide','title':title}]})
        group=None;step=0
        for n,line in enumerate(code.splitlines(),1):
            s=line.strip()
            if s.startswith('--'):continue
            if re.match(r'^step(?:\s|$)',s):
                step+=1;cond=s.split('<<',1)[1].strip() if '<<' in s else ''
                group={'guide':title,'file':name,'step':step,'condition_format':'rxp','condition':cond,'guards':[],'flags':[],'events':[]};groups.append(group);continue
            if group is None:continue
            if s.startswith('#hardcore'):group['flags'].append('hardcore');continue
            if s.startswith('#softcore'):group['flags'].append('softcore');continue
            if s.startswith('#completewith'):group['flags'].append('parallel');continue
            gm=re.match(r'^\.(isOnQuest|isNotOnQuest|isQuestComplete|isNotQuestComplete|isQuestTurnedIn|isNotQuestTurnedIn)\s+(\d+)',s)
            if gm:group['guards'].append([gm[1],int(gm[2])]);continue
            a=re.match(r'^\.(accept|turnin|complete)\s+(\d+)(.*)$',s)
            if a:
                tail=a[3];cond=tail.split('<<',1)[1].split('--',1)[0].strip() if '<<' in tail else ''
                group['events'].append({'t':a[1],'q':int(a[2]),'condition_format':'rxp','condition':cond,'line':n})
            xp=re.match(r'^\.xp\s+(\d+)(?:\+(\d+))?(?:\s|$)',s)
            if xp:group['events'].append({'t':'level','level':int(xp[1]),'xp':int(xp[2] or 0)})
    save('rxp',{'id':'rxp','label':'RestedXP Forever (ohne AoE)','coverage':[1,22],'groups':groups,'limits':['Orc Warlock class-specific #next chain through Tirisfal and Silverpine','Quest workloads normalized like the other sources; parallel objectives postponed to hand-in','Deathskips, vendor transactions, money gates and detailed trainer trips are not replayed']})

if __name__=='__main__':
    journey();wisp();rxp()
    for p in OUT.glob('*.json'):
        d=json.loads(p.read_text());print(p.name,len(d['groups']),sum(len(g['events']) for g in d['groups']))
