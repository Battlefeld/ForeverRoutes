"""Data quality gate: never substitute imaginary Questie/Forever values."""
from __future__ import annotations
from collections import Counter
from .core import World


def audit(world: World, *, target_level: int = 30, start: str | None = None) -> dict:
    missing = [l for l in range(1, target_level) if l not in world.level_xp]
    mobs_missing_lv = sorted(k for k, m in world.mobs.items() if m.level is None)
    unknown_ttk = sorted(k for k,m in world.mobs.items() if m.kill_seconds is None)
    incomplete = sorted(q.id for q in world.quests if q.xp == 0)
    location_errors = []
    if start and start in world.nodes:
        for loc in world.nodes:
            if world.travel(start,loc) == float('inf'):
                location_errors.append(loc)
    source = getattr(world, 'metadata', {})
    return {
        'synthetic': bool(source.get('synthetic',False)),
        'data_source': source.get('source','unspecified'),
        'data_revision': source.get('revision','unspecified'),
        'level_target': target_level,
        'nodes':len(world.nodes), 'quests':len(world.quests),
        'mobs':len(world.mobs), 'farms':len(world.farms),
        'max_open_quests':world.max_open,
        'missing_xp_levels':missing,
        'zero_xp_quests':incomplete,
        'mobs_without_level':mobs_missing_lv,
        'mobs_using_dps_model':unknown_ttk,
        'unreachable_locations':sorted(location_errors),
        'model_is_game_calibrated':False,  # not even complete source metadata proves gameplay accuracy
        'note':'Provide verified Forever data and calibrate against recordings before using hours as estimates.'
    }