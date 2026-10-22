> **WICHTIG: Echte Forever-Routen sind noch nicht zeitlich bewertet.** Die bisherige synthetische Level-30-Demo ist nur ein Test der Software, kein Vergleich von RestedXP, Journey, ForeverWisp und Polestar. Die neue, strikte Quellenprüfung lässt Era-Werte und erfundene Zeiten nicht als Forever-Messwerte gelten. Siehe [Real-Data-Benchmark](docs/REAL_FOREVER_BENCHMARK.md).

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

## Reference guide auditing (local, no guide files committed)

```sh
python -m foreverroutes baseline --source journey --source-path /local/Journey_Quest_Routen_Analyse.csv
python -m foreverroutes baseline --source wisp --source-path /local/ForeverWisp_Quest_Routen_Analyse.csv
python -m foreverroutes baseline --source polestar --source-path /local/Polestar_Quest_Routen_Analyse.csv
```

The CSV adapter normalizes quest references and counts unresolved conditions; **it deliberately does not assign a time to reference guides**. Those guides can only be replayed faithfully once their class/quest flags, movements, grouped objectives and actual world facts are normalized. AoE guide names are excluded.