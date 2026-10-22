"""Compare a reference guide's IDs with the pinned source catalog.

This is *coverage only*: no fake travel seconds, target counts, DPS or claims
that a guide is fast. Conditions/optional actions remain unresolved.
"""
from __future__ import annotations
import argparse
import json
from pathlib import Path
from .baselines import parse_rows


def audit_guide(catalog: dict, rows: list[dict], *, name: str = '') -> dict:
    qtab=catalog.get('quests',{})
    nt=catalog.get('npcs',{})
    ot=catalog.get('objects',{})
    it=catalog.get('items',{})
    ids={int(r['quest_id']) for r in rows}
    missing_quests=sorted(q for q in ids if str(q) not in qtab)
    unknown_counts=sorted(q for q in ids if any(o.get('required_count') is None for o in qtab.get(str(q),{}).get('objectives',[])))
    missing_xp=sorted(q for q in ids if not (qtab.get(str(q),{}).get('xp') or qtab.get(str(q),{}).get('xp_observed_by_player_level')))
    missing_starters=[];missing_finishers=[];spawn_missing=[];drop_missing=[]
    for qid in ids:
        q=qtab.get(str(qid))
        if not q:continue
        st=q.get('started_by',{});fin=q.get('finished_by',{})
        if not any(st.get(k) for k in ('npc','object','item')):missing_starters.append(qid)
        if not any(fin.get(k) for k in ('npc','object')):missing_finishers.append(qid)
        for k in ('npc','object'):
            for id_ in st.get(k,[])+fin.get(k,[]):
                base=(nt if k=='npc' else ot).get(str(id_))
                if base is None or not base.get('spawns'):spawn_missing.append({'quest':qid,'kind':k,'id':id_})
        for obj in q.get('objectives',[]):
            if obj['kind']=='item':
                for id_ in obj.get('ids',[]):
                    item=it.get(str(id_))
                    if not item or (item.get('npc_sources') and not item.get('drop_chance_by_npc')):
                        drop_missing.append({'quest':qid,'item_id':id_})
    conditional=sum(bool(r.get('condition')) for r in rows)
    optional=sum(bool(r.get('optional')) for r in rows)
    return {
      'guide':name,'source_rows':len(rows),'quest_ids':len(ids),
      'missing_quest_ids':missing_quests,'missing_xp_quest_ids':missing_xp,
      'unknown_objective_quantity_quest_ids':unknown_counts,
      'missing_starter_ids':sorted(missing_starters),
      'missing_finisher_ids':sorted(missing_finishers),
      'missing_spawns':spawn_missing,'item_without_usable_drop_rate':drop_missing,
      'unresolved_conditional_actions':conditional,'optional_actions':optional,
      'has_walkable_travel_graph':False,
      'has_class_calibrated_combat_times':False,
      'can_claim_fastest_route':False,
      'status':'reference_coverage_only',
      'note':'No timing until pathing, count, class, travel & action conditions are resolved.'
    }


def run(argv=None):
    p=argparse.ArgumentParser(description='Audit guide fact coverage against generated Questie catalog')
    p.add_argument('--catalog',type=Path,required=True)
    p.add_argument('--source',choices=('journey','wisp','polestar','rxp'),required=True)
    p.add_argument('--csv',type=Path,required=True)
    p.add_argument('--guide',default=None)
    p.add_argument('--output',type=Path)
    args=p.parse_args(argv)
    cat=json.loads(args.catalog.read_text(encoding='utf8'))
    rows=list(parse_rows(args.csv,args.source,args.guide))
    result=audit_guide(cat,rows,name=args.guide or args.source)
    encoded=json.dumps(result,ensure_ascii=False,indent=2)
    if args.output:
        args.output.parent.mkdir(parents=True,exist_ok=True)
        args.output.write_text(encoded+'\n',encoding='utf8')
    print(encoded)
    return result

if __name__=='__main__':run()