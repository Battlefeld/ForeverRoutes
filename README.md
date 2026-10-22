> **WORKING-DATA POLICY:** QuestieDB Forever quest structure + Classic XP/HP raw reference values are now accepted for **provisional route simulation** when actual Forever values are missing. ForeverDB and Wowhead Forever override per field. Such modeled times are not independently verified /played. See [data pipeline](docs/QUEST_DATA_PIPELINE.md). The legacy strictly-verified checker remains available for a separate evidence-only audit.

# ForeverRoutes — Quest + Grind Simulator, Level 1–30

Standalone **offline** route-simulation platform for WoW Forever. No dependency on ForeverSim, no in-game addon yet. Python 3.10+, standard library only. MIT (original code); no third-party guide text is checked in.

**Current state: working model + search, not validated WoW 1–30 optimization.** A fully synthetic 1–30 world demonstrates end-to-end evaluation. Do **not** interpret its seconds as /played or claim the route beats RestedXP, Journey, Polestar or ForeverWisp. Real Forever quest, spawn, geometry, kill, XP and drop data are **not** bundled. Licensing of those sources must be checked before distribution.

## Quick start

```sh
python -m unittest discover -s tests -v
python -m foreverroutes fixture --output /tmp/fictional_30.json
python -m foreverroutes audit --world /tmp/fictional_30.json --start Spawn --target-level 30
python -m foreverroutes search --world /tmp/fictional_30.json --start Spawn --target-level 30 --restarts 12 --seed 42 --output /tmp/route_result.json
python -m foreverroutes replay --world /tmp/fictional_30.json --route /tmp/route_result.json --start Spawn --target-level 30
python -m foreverroutes exact --world examples/synthetic_tiny.json --target-level 2 --start Spawn
```

`search` uses seeded multi-start quest/grind search; the winner is **best found, not mathematically proven globally optimal**. `exact` uses Dijkstra over full game state for **very small** worlds and honors a state-count limit; it never calls a truncated run optimal. `replay` validates each action independently. `audit` reports uncovered data and always labels model predictions as uncalibrated.

## Model format (JSON)

```
{
  "source": "QuestieDB Forever + measured data", "revision": "YOUR_PINNED_VERSION",
  "nodes": ["spawn", "hub", "wolves"],
  "edges": [["spawn", "hub", 20], ["hub", "wolves", 40]],
  "level_xp": {"1": 400, "2": 900},
  "dps": 9.2, "max_open": 40,
  "mobs": {"wolf": {"hp": 70, "xp": 40, "level": 2, "downtime": 3}},
  "quests": [{"id": 1, "giver": "hub", "turnin": "hub", "level": 1, "xp": 250,
    "prereqs": [], "objectives": [{"kind": "kill", "target": "wolf", "location": "wolves", "count": 5}]}],
  "farms": [{"at": "wolves", "mob": "wolf"}]
}
```

Values in this schema are **illustrative**, NOT real quest 1 / mobs. `level` on a quest is currently the minimum allowed level, **not** its display level. `edges` are measured travel seconds on walkable links, never invented direct distances. `mob.kill_seconds` overrides `hp/dps`. Collect objectives consume **expected** kills `1/drop_chance` per item; overlapping *kill* quests are credited at the same spot, but probabilistic multi-drop correlations and dynamic collection progress remain approximations.

## Safety / model assumptions

- A missing road is impassable; unknown NPC HP, XP, mob levels, drop rates, class DPS and journey times should prevent claiming a calibrated result.
- Out-of-level mobs cannot be used as an easy grind exploit; grey mob XP is suppressed in a simplified way.
- Quest requirements, class, race, faction, questlog cap, ordered turn-ins and shared kills are checked during replay.
- Solo model only. No escorts, timed quests, phasing, flights/boats/hearth, deathskips, PvP, food XP buff or detailed class rotations **yet**. This engine never pretends otherwise.
- Search local greedy macro-plans are not guaranteed globally optimal; front-loaded quest pickups sharing objectives and zero-XP prereq quests are represented. The planner can still miss beneficial multi-zone detours or delayed hand-ins.


## Reference NPC health for leveling 1–30

Classic HP is now an **accepted provisional input** where no measured Forever HP exists. This does NOT claim a verified server value. Prefer per-level ForeverDB observations, then Wowhead Forever, then QuestieDB's Classic-inherited NPC raw health. Explicit provenance is attached to each imported mob.

```sh
python -m foreverroutes.mobdata --fetch --out .cache/mob_hp_1_33.json --max-level 33
python -m foreverroutes.mobdata --fetch --out .cache/mob_hp_1_33.json --world world_input.json --world-out world_with_hp.json
```

For individual overrides, pass `--overrides examples/mob_hp_overrides_template.json` with level-specific observations. See [`docs/NPC_HP_REFERENCE_1_30.md`](docs/NPC_HP_REFERENCE_1_30.md). The QuestieDB public *API* NPC-HP getters return **0/1 placeholders**; this importer reads the pinned raw NPC table, not those getters. **Mob XP, combat DPS, roads, and quests remain independent inputs.** This change does not claim a fully validated 1–30 route.

## Workflow for *real* Level 1–30

1. Import QuestieDB Forever and ForeverDB **locally** to a normalized world with pinned source revisions; resolve quest prerequisites, exact quest XP, class masks and objective NPCs (no guide-text copies). Add measured kill XP, HP, drop chance, mob ranks, spawn clustering and geographic paths.
2. Run `audit` for both Horde and Alliance. Zero missing level XP, no invented unreachable links, coverage report for each start and class.
3. Replay reference guide actions (Journey, RestedXP **without AoE**, ForeverWisp and Polestar) under *identical* character, source version, XP and time settings. Do not regard different level windows or class variants as comparable.
4. Improve the search with quest packages, beam search, terrain pathing and actual encounter sampling; estimate intervals and replay every result. Use recordings to calibrate uncertainty.
5. Only after the offline benchmark is reliable, expose a typed JSON API to ForeverCore. **Do not change `Battlefeld/ForeverSim`** for Phase 1.

## Layout

- `foreverroutes/core.py`: world graph, quest-state evaluator, exact tiny-world solver.
- `foreverroutes/planner.py`: seeded multi-start quest + grind optimizer for the 1–30 experiment.
- `foreverroutes/audit.py`: missing-data and realism report.
- `foreverroutes/__main__.py`: CLI (`audit`, `search`, `exact`, `replay`, `fixture`).
- `examples/generate_synthetic_30.py`: generated fictional 58-quest, 29-zone test world.
- `tests/`: replay, invalid-route, class restrictions, grind gating and 1–30 integration tests.

## Licence / provenance

Original simulator code is MIT. Polestar is GPL-3.0; RestedXP is CC BY-NC-SA; Journey restricts redistribution/reverse engineering of content. No source implementation or protected guide text from those projects has been copied into this repository. Public facts (quest IDs, numeric source facts) should still be sourced and versioned. Use only data you are entitled to process and redistribute.

## Questie/Classic fallback import (new)

```sh
python -m foreverroutes.questdata --overlay data/overlays/foreverdb_public_spotchecks_2026-10-22.json --overlay data/overlays/wowhead_forever_manual_2026-10-22.json --output .generated/forever_quests_and_mobs_1_30.json
```

Downloads the pinned QuestieDB Forever quest file, Era quest-XP file, and raw Classic NPC HP/level file locally. See [docs/QUEST_DATA_PIPELINE.md](docs/QUEST_DATA_PIPELINE.md) for provenance, field priority and caveats.