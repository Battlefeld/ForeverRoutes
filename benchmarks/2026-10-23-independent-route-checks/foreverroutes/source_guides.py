"""Import source-native guide facts, retaining scopes, optional work and nonquest steps."""
from __future__ import annotations
import argparse,base64,hashlib,json,re
from pathlib import Path
from .guide_literals import read_literal,children
from .guide_facts import standard_files

SCHEMA='foreverroutes.guide_events.v2'

def plain(x):
    if isinstance(x,dict):
        if x and all(isinstance(k,int) for k in x) and sorted(x)==list(range(1,len(x)+1)):return [plain(x[k]) for k in sorted(x)]
        return {str(k):plain(v) for k,v in x.items()}
    return x

def origin(path,step=None,line=None):
    return {'path':str(path),'step':step,**({'line':line} if line else {})}

def journey_body(root,*,guide,path):
    groups=[]
    for step in children(root):
        if not isinstance(step,dict):continue
        group={'guide':guide,'step':step.get('id'),'events':[],'origin':origin(path,step.get('id'))}
        def visit(node,conditions=(),optional=False):
            if not isinstance(node,dict):return
            scopes=conditions+((node['condition'],) if node.get('condition') else ())
            opt=optional or node.get('optional') is True
            base={'conditions':list(scopes),'condition_format':'journey','optional':opt,'origin':group['origin']}
            for k in ['accept','complete','turnin']:
                if isinstance(node.get(k),int):group['events'].append({**base,'t':k,'q':node[k],**({'obj':node['objective']} if isinstance(node.get('objective'),int) else {})})
            if isinstance(node.get('level'),int):group['events'].append({**base,'t':'level','level':node['level'],'xp':node.get('xp',0)})
            for k,t in [('sethearthstone','bind'),('hearthstone','hearth'),('learntaxi','learn_flight'),('taketaxi','transport'),('die','deathskip'),('spiritres','resurrect'),('guide','transition'),('getitem','inventory_goal'),('destroy','destroy'),('setvar','set_variable'),('deposit','bank_deposit'),('withdraw','bank_withdraw'),('startwork','track_objective'),('goto','choice')]:
                if k in node:group['events'].append({**base,'t':t,'value':plain(node[k]),**({'count':node['quantity']} if 'quantity' in node else {})})
            if 'merchant' in node:group['events'].append({**base,'t':'service','service':plain(node['merchant'])})
            if 'locations' in node:
                for loc in children(node['locations']):
                    if not isinstance(loc,dict):continue
                    for value in children(loc):
                        if isinstance(value,str) and '|' in value:
                            cs=list(scopes)+([loc['condition']] if loc.get('condition') else [])
                            group['events'].append({**base,'t':'move','coordinates':value,'conditions':cs,'npc':loc.get('npc'),'object':loc.get('gameobject')})
            for callback in ['oncomplete','onleave','onactivate','onenter']:
                if callback in node:group['events'].append({**base,'t':'callback','callback':callback,'value':plain(node[callback])})
            for child in children(node):visit(child,scopes,opt)
        visit(step)
        if group['events']:groups.append(group)
    return groups

def load_journey(folder):
    folder=Path(folder);raws={}
    for p in folder.glob('*.html'):
        m=re.search(r'<p>(.*?)</p>',p.read_text(),re.S)
        if m:raws[p.stem]=base64.b64decode(re.sub(r'\s+','',m[1])).decode()
    index,_=read_literal(raws['guides']);out=[]
    for meta in children(index):
        if not isinstance(meta,dict):continue
        slot=f"{meta['slot']:04d}";root,_=read_literal(raws[slot]);lo,hi=map(int,re.findall(r'\d+',meta['levels'])[:2])
        out.append({'schema':SCHEMA,'id':'journey:'+meta['guideID'],'source':'journey','guideId':meta['guideID'],'label':meta['description'],'faction':meta.get('faction'),'coverage':[lo,hi],'condition':meta.get('condition'),'recommended':meta.get('recommended'),'defaultZone':meta.get('zone'),'groups':journey_body(root,guide=meta['guideID'],path=slot+'.html'),'sourceSha256':hashlib.sha256(raws[slot].encode()).hexdigest()})
    return out

def load_wisp(file):
    s=Path(file).read_text(encoding='utf-8-sig');marker='ForeverWisp_RegisterGuides(';pos=s.index(marker+'{')+len(marker);root,_=read_literal(s,pos,False);out=[]
    for g in children(root):
        route=g.get('route',{});groups=[];trip_steps={}
        for trip in children(route.get('trips',{})):
            for sid in children(trip.get('stepIds',{})):trip_steps[sid]={'id':trip.get('id'),'groupSize':trip.get('groupSize'),'dungeonId':trip.get('dungeonId')}
        for step in children(g.get('steps',{})):
            c=step.get('completion') or {};kind=c.get('type');base={'origin':origin(Path(file).name,step.get('id')),'conditions':[plain(step.get('requirements',{}))],'condition_format':'wisp','optional':step.get('optional') is True,'classes':[step['class']] if step.get('class') else None,'races':plain(step.get('races')),'excludedClasses':plain(step.get('excludedClasses')),'trip':trip_steps.get(step.get('id')),'background':step.get('background') is True,'recoveryOnly':step.get('recoveryOnly') is True,'advisory':step.get('advisory') is True}
            events=[];loc=step.get('location')
            if loc:events.append({**base,'t':'move','map':loc.get('mapId'),'x':loc.get('x'),'y':loc.get('y'),'npc':loc.get('npcId'),'approximate':loc.get('approximate',False)})
            if kind=='quest':
                t={'accepted':'accept','objectives':'complete','turned-in':'turnin'}.get(c.get('state'))
                if t:events.append({**base,'t':t,'q':c['questId'],'objectives':plain(step.get('objectives')),'providedObjectiveItems':plain(step.get('providedObjectiveItems')),'activationQuest':step.get('activationQuest'),'activationQuestReward':step.get('activationQuestReward'),'campingLesson':plain(step.get('campingLesson'))})
                else:events.append({**base,'t':'unsupported','directive':'quest-state','value':plain(c)})
            elif kind=='level':events.append({**base,'t':'level','level':c.get('level'),'xp':c.get('xp',0)})
            elif kind in ['flight','hearth','arrival','manual','item','skill']:events.append({**base,'t':{'flight':'learn_flight','hearth':'hearth','arrival':'arrival','manual':'manual','item':'inventory_goal','skill':'profession'}[kind],'value':plain(c)})
            else:events.append({**base,'t':'unsupported','directive':kind,'value':plain(c)})
            if events:groups.append({'guide':g['id'],'step':step['id'],'events':events})
        out.append({'schema':SCHEMA,'id':'wisp:'+g['id'],'source':'wisp','guideId':g['id'],'label':g['title'],'faction':route.get('faction'),'races':plain(route.get('races')) or [route.get('race')],'coverage':plain(route.get('targetLevels')),'nextGuide':route.get('chapter',{}).get('next',{}).get('guideId'),'groups':groups,'sourceSha256':route.get('sourceHash')})
    return out

RXP_KIND={'goto':'move','accept':'accept','turnin':'turnin','complete':'complete','xp':'level','collect':'inventory_goal','destroy':'destroy','hs':'hearth','home':'bind','fp':'learn_flight','fly':'transport','zone':'arrival','train':'train','trainer':'train','vendor':'vendor','buy':'vendor','deathskip':'deathskip','death':'deathskip','skill':'profession','cast':'cast','use':'use_item','link':'choice','skipgossip':'gossip','gossip':'gossip','subzone':'arrival','zoneskip':'arrival','isOnQuest':'quest_guard','isNotOnQuest':'quest_guard','isQuestComplete':'quest_guard','isNotQuestComplete':'quest_guard','isQuestTurnedIn':'quest_guard','isNotQuestTurnedIn':'quest_guard','isQuestAvailable':'quest_guard','isQuestNotAvailable':'quest_guard','isOnQuest':'quest_guard','isQuestActive':'quest_guard','money':'money_guard','itemcount':'inventory_guard'}
RXP_METADATA={'target','mob','npc','icon','waypoint','highlight','waypointname','itemcount','vendor','petfamily'}

def load_rxp(folder):
    out=[]
    for p in standard_files(folder):
        text=p.read_text(encoding='utf-8-sig')
        for block in re.finditer(r'RXPGuides\.RegisterGuide\(\[\[([\s\S]*?)\]\]\)',text):
            code=block[1];tag=lambda key:re.findall(r'^#'+re.escape(key)+r'\s+(.+)$',code,re.M);name=tag('name')[0].strip()
            if 'aoe' in name.lower():continue
            groups=[];group=None;step=0
            for n,line in enumerate(code.splitlines(),text.count('\n',0,block.start())+1):
                s=line.strip()
                if s.startswith('--') or not s:continue
                if re.match(r'^step(?:\s|$)',s):
                    step+=1;group={'guide':name,'step':step,'flags':[],'condition_format':'rxp','condition':s.split('<<',1)[1].strip() if '<<' in s else '', 'events':[]};groups.append(group);continue
                if group is None:continue
                if s.startswith('#'):
                    flag=s.split()[0][1:]
                    group['flags'].append(s[1:])
                    continue
                m=re.match(r'^\.([A-Za-z]+)(?:\s+(.*))?$',s)
                if not m:continue
                cmd,args=m[1],m[2] or '';no_comment=args.split('--',1)[0];condition=no_comment.split('<<',1)[1].strip() if '<<' in no_comment else ''
                value=no_comment.split('<<',1)[0].split('>>',1)[0].strip();kind=RXP_KIND.get(cmd)
                if cmd in ['target','mob','npc','icon','waypoint','highlight','waypointname','petfamily']:continue
                e={'t':kind or 'unsupported','directive':cmd,'args':value,'conditions':[condition] if condition else [],'condition_format':'rxp','origin':origin(p.name,step,n)}
                nums=re.match(r'^(-?\d+)(?:\s*,\s*(\d+))?',value)
                if kind in ['accept','turnin','complete'] and nums:
                    q=int(nums[1]);e['q']=abs(q) if kind=='turnin' else q
                    if kind=='complete' and nums[2]:e['obj']=int(nums[2])
                    if kind=='turnin' and q<0:
                        # RXP functions.turnin: a negative id means a conditional
                        # hand-in of an already held quest, not an invalid id.
                        e['skipIfMissing']=True
                        e['skipIfIncomplete']=bool(nums[2] and int(nums[2])>0)
                if kind=='level':
                    xp=re.fullmatch(r'(\d+)(?:\+(\d+))?',value)
                    if xp:e.update(level=int(xp[1]),xp=int(xp[2] or 0))
                    else:e.update(t='level_guard')
                group['events'].append(e)
            ranges=re.match(r'(\d+)-(\d+)',name)
            out.append({'schema':SCHEMA,'id':'rxp:'+name,'source':'rxp','guideId':name,'label':name,'faction':'Horde' if p.name.startswith('Horde') else 'Alliance','coverage':[int(ranges[1]),int(ranges[2])] if ranges else None,'defaultFor':tag('defaultfor'),'nextChoices':tag('next'),'groups':groups,'sourceSha256':hashlib.sha256(code.encode()).hexdigest()})
    return out

def main():
    p=argparse.ArgumentParser();p.add_argument('--journey',type=Path,required=True);p.add_argument('--wisp',type=Path,required=True);p.add_argument('--rxp',type=Path,required=True);p.add_argument('--output',type=Path,required=True);a=p.parse_args()
    docs=load_journey(a.journey)+load_wisp(a.wisp)+load_rxp(a.rxp)
    if len({d['id'] for d in docs})!=len(docs):raise ValueError('Duplicate guide identifiers; refusing ambiguous route catalog')
    a.output.mkdir(parents=True,exist_ok=True)
    (a.output/'catalog.json').write_text(json.dumps({'schema':SCHEMA,'guides':docs},ensure_ascii=False,separators=(',',':'))+'\n')
    from collections import Counter
    print(json.dumps({'guides':dict(Counter(d['source'] for d in docs)),'events':dict(Counter(e['t'] for d in docs for g in d['groups'] for e in g['events']))}))

if __name__=='__main__':main()
