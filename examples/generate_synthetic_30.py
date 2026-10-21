"""Creates an unmistakably FICTIONAL 1-30 stress world. NOT WoW data."""
import json
from pathlib import Path


def build():
    nodes = ['Spawn']
    edges = []
    mobs = {}
    quests = []
    farms = []
    thresholds = {str(level): int(350 + level*110) for level in range(1,30)}
    for lvl in range(1,30):
        hub = f'L{lvl}_Hub'; field = f'L{lvl}_Field'
        nodes += [hub,field]
        edges.append([nodes[-3],hub,35] if lvl == 1 else [f'L{lvl-1}_Field',hub,100])
        edges.append([hub,field,48])
        mob = f'mob_L{lvl}'
        mobs[mob] = {'hp':50+lvl*25, 'xp':25+lvl*12, 'downtime':3.5, 'level':lvl}
        farms.append({'at':field,'mob':mob})
        for k in (0,1):
            qid=lvl*100+k+1
            quest={'id':qid,'giver':hub,'turnin':hub,'level':lvl,
                   'xp':int(thresholds[str(lvl)]*.62),
                   'prereqs':([((lvl-1)*100+1)] if lvl>1 and k==0 else []),
                   'objectives':[{'kind':'kill','target':mob,'location':field,'count':5+k*2}]}
            quests.append(quest)
    # Synthetic combat stats ensure each zone supplies multiple plausible paths.
    return {'synthetic': True,'description':'Fictional performance fixture; do not call game facts',
            'schema':1,'nodes':nodes,'edges':edges,'level_xp':thresholds,
            'dps':55.0,'max_open':40,'mobs':mobs,'quests':quests,'farms':farms}

if __name__=='__main__':
    out=Path(__file__).with_name('synthetic_level1_30.json')
    out.write_text(json.dumps(build(),ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
    print(out)