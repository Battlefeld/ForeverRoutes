"""Common quest-workload references extracted from explicitly numbered guide goals.

These are guide references, never promoted to observed Forever server facts.
All routes share the same resulting overlay.
"""
from __future__ import annotations
from collections import defaultdict
from pathlib import Path
import argparse,json,re

RXP_REV='9536b5a9ee7e55745618a57dad43df2c4312ea16'
STANDARD={'Alliance-1-10_NightElf.lua','Alliance-1-13_Human.lua','Alliance-1-14_DwarfGnome.lua','Alliance-11-20.lua','Horde-01-12_Durotar.lua','Horde-01-14_Undead.lua','Horde-1-12_Mulgore.lua','Horde-12-22_Barrens.lua'}
GOAL=re.compile(r'^\s*\.complete\s+(\d+)\s*,\s*(\d+)(.*)$')
NUMBERED=re.compile(r'(?<!\d)(\d+)\s*/\s*(\d+)(?!\d)')

def standard_files(root):
    root=Path(root)
    folder=root/'Guides/forever' if (root/'Guides/forever').is_dir() else root/'forever' if (root/'forever').is_dir() else root
    return sorted(p for p in folder.glob('*.lua') if p.name in STANDARD)

def numbered_goal_counts(rxp_root):
    candidates=defaultdict(list)
    for file in standard_files(rxp_root):
        for line_no,line in enumerate(file.read_text(encoding='utf-8-sig').splitlines(),1):
            m=GOAL.match(line)
            if not m or '--' not in m[3]:continue
            comment=m[3].split('--',1)[1];n=NUMBERED.search(comment)
            if not n or not 0<int(n[2])<=10000 or int(n[1])>int(n[2]):continue
            q,oi=int(m[1]),int(m[2])
            candidates[q,oi].append({'count':int(n[2]),'source':'restedxp_explicit_objective_fraction','quality':'guide_reference','revision':RXP_REV,'path':'Guides/forever/'+file.name,'line':line_no})
    values={};conflicts=[]
    for (q,oi),records in candidates.items():
        counts={r['count'] for r in records}
        if len(counts)!=1:conflicts.append({'quest':q,'objective':oi,'candidates':records});continue
        values.setdefault(str(q),{})[str(oi)]={'count':next(iter(counts)),'evidence':records}
    return values,conflicts

def enrich(quests,counts):
    filled=[];conflicts=[]
    for sid,refs in counts.items():
        q=quests.get(sid)
        if not q:continue
        for oi,ref in refs.items():
            idx=int(oi)-1;objs=q.get('objectives') or []
            if not 0<=idx<len(objs):continue
            obj=objs[idx];current=obj.get('count')
            if isinstance(current,(int,float)) and current>0:
                if current!=ref['count']:conflicts.append({'quest':int(sid),'objective':int(oi),'retained':current,'guide':ref['count']})
                continue
            obj['count']=ref['count'];obj['countProvenance']=ref
            filled.append({'quest':int(sid),'objective':int(oi),'count':ref['count'],'type':obj.get('type')})
    return filled,conflicts

def main():
    p=argparse.ArgumentParser();p.add_argument('--rxp',type=Path,required=True);p.add_argument('--quests',type=Path,required=True);p.add_argument('--output',type=Path,required=True);a=p.parse_args()
    qs=json.loads(a.quests.read_text());refs,conflicts=numbered_goal_counts(a.rxp);filled,existing=enrich(qs,refs)
    a.output.mkdir(parents=True,exist_ok=True)
    (a.output/'quests.json').write_text(json.dumps(qs,ensure_ascii=False,separators=(',',':'))+'\n')
    report={'schema':'foreverroutes.guide_objective_facts.v1','reference_count':sum(map(len,refs.values())),'filled_count':len(filled),'filled':filled,'conflicts':conflicts+existing,'policy':'One source-tagged overlay shared by all compared routes; unknown counts remain unknown.'}
    (a.output/'objective-facts.json').write_text(json.dumps(report,indent=2)+'\n')
    print(json.dumps({'reference_count':report['reference_count'],'filled_count':len(filled),'conflicts':len(report['conflicts'])}))

if __name__=='__main__':main()
