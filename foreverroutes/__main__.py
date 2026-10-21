"""Command line for reproducible quest/grind 1-30 scenario runs."""
import argparse
import json
from pathlib import Path
from .core import World, optimize, evaluate
from .planner import Character, search
from .audit import audit


def main():
    p=argparse.ArgumentParser(prog='foreverroutes', description='Offline 1-30 Quest + Grind planner; models need real data for real estimates')
    p.add_argument('command',choices=['audit','search','exact','replay','fixture','baseline'])
    p.add_argument('--world',default='examples/synthetic_tiny.json')
    p.add_argument('--source',choices=['journey','wisp','polestar','rxp'])
    p.add_argument('--source-path')
    p.add_argument('--guide')
    p.add_argument('--route')
    p.add_argument('--output')
    p.add_argument('--start',default='Spawn')
    p.add_argument('--race',default='Orc')
    p.add_argument('--class-name',default='Warlock')
    p.add_argument('--faction',default='Horde')
    p.add_argument('--target-level',type=int,default=30)
    p.add_argument('--restarts',type=int,default=12)
    p.add_argument('--seed',type=int,default=1)
    p.add_argument('--max-decisions',type=int,default=1800)
    p.add_argument('--max-states',type=int,default=50000)
    a=p.parse_args()
    if a.command=='baseline':
        from .baselines import parse_rows,coverage
        if not a.source or not a.source_path:p.error('--source and --source-path required with baseline')
        result=coverage(parse_rows(a.source_path,a.source,a.guide))
    elif a.command=='fixture':
        from importlib.util import spec_from_file_location, module_from_spec
        f=Path(__file__).resolve().parents[1]/'examples/generate_synthetic_30.py'
        spec=spec_from_file_location('fixture',f);mod=module_from_spec(spec);spec.loader.exec_module(mod)
        result=mod.build()
    else:
        w=World.load(a.world)
        if a.command=='audit':result=audit(w,target_level=a.target_level,start=a.start)
        elif a.command=='search':
            c=Character(a.race,a.class_name,a.faction,a.start,a.target_level)
            result=search(w,c,restarts=a.restarts,seed=a.seed,max_decisions=a.max_decisions)
            result['source_audit']=audit(w,target_level=a.target_level,start=a.start)
        elif a.command=='exact':result=optimize(w,a.start,a.race,a.class_name,a.faction,a.target_level,a.max_states)
        else:
            if not a.route:p.error('--route is required with replay')
            actions=json.loads(Path(a.route).read_text(encoding='utf8'))
            if isinstance(actions,dict): actions=actions['steps']
            actions=[{k:v for k,v in act.items() if k in ('op','quest','obj','at','target')} for act in actions]
            result=evaluate(w,actions,a.start,a.race,a.class_name,a.faction,a.target_level)
    encoded=json.dumps(result,indent=2,ensure_ascii=False)+'\n'
    if a.output:Path(a.output).write_text(encoded,encoding='utf8')
    else:print(encoded)

if __name__=='__main__':main()