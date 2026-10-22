"""Pinned classic-level XP requirements and cMaNGOS solo-kill XP fallback.

The server can modify all values. No Forever-server observation is implied.
No group, rested, heirloom or food multiplier is applied here.
"""
from __future__ import annotations

XP_TO_NEXT = dict(enumerate((
    400, 900, 1400, 2100, 2800, 3600, 4500, 5400, 6500,
    7600, 8800, 10100, 11400, 12900, 14400, 16000, 17700,
    19400, 21300, 23200, 25200, 27300, 29400, 31700,
    34000, 36400, 38900, 41400, 44300, 47400
), 1))

XP_LEVEL_SOURCE = 'https://wowpedia.fandom.com/wiki/Experience_to_level#Vanilla_/_Classic_table'
KILL_XP_SOURCE = 'https://github.com/cmangos/mangos-classic/blob/master/src/game/Tools/Formulas.h'


def zero_difference(level: int) -> int:
    if level < 8: return 5
    if level < 10: return 6
    if level < 12: return 7
    if level < 16: return 8
    if level < 20: return 9
    if level < 30: return 11
    if level < 40: return 12
    if level < 45: return 13
    if level < 50: return 14
    if level < 55: return 15
    if level < 60: return 16
    return 17


def gray_level(player_level: int) -> int:
    if player_level <= 5: return 0
    if player_level <= 39: return player_level - 5 - player_level // 10
    if player_level <= 59: return player_level - 1 - player_level // 5
    return player_level - 9


def base_kill_xp(player_level: int, mob_level: int, *,
                 elite: bool = False, no_xp: bool = False,
                 elite_mult: float = 2.5) -> int:
    """cMaNGOS Classic solo kill baseline. Excludes server XP multipliers.

    Floats use Python rounding (nearest/even) like C++ nearbyint with default
    rounding mode. Elite and pet/critter status require verified creature flags.
    """
    if player_level < 1 or mob_level < 1:
        raise ValueError('Levels must be positive')
    if elite_mult <= 0:
        raise ValueError('Invalid elite multiplier')
    if no_xp or mob_level <= gray_level(player_level):
        return 0
    base = player_level * 5 + 45
    if mob_level >= player_level:
        factor = 1 + 0.05 * min(mob_level - player_level, 4)
    else:
        factor = max(0.0, 1 - (player_level - mob_level) / zero_difference(player_level))
    return round(base * factor * (elite_mult if elite else 1.0))


def xp_to_level_30() -> int:
    return sum(XP_TO_NEXT[level] for level in range(1, 30))