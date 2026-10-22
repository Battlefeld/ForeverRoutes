# Correct order: score the existing guides with real Forever data first

**No synthetic world, guide estimate or Classic Era seed may be represented as a WoW Forever speed result.**

## Protocol

1. Select identical race, class, faction, equipment, level-1 spawn and target level 30.
2. Normalize Journey, RestedXP (excluding AoE), ForeverWisp and Polestar **into action streams**. Execute each guide's class/race/faction/version rules, alternatives and optional steps. Verify quest chain ordering and objectives. Store source revision for every route.
3. Join the quests to independently sourced **Forever** data: actual quest reward XP, XP-to-level, quest prerequisites, NPC levels and health, objective counts, item drop chances and spawn coordinates. Preserve direct server observation vs client-only tables vs Classic/Era reference vs guide estimate.
4. Time the same quests on the same world and player model: character-specific encounter time and downtime, travel over walkable terrain and transport, interactions and waiting. Prefer real recordings, never default to a hidden unverified value. Collect sample size, build and date.
5. Replay **every** complete guide to the same target level, reject missing/unavailable actions, then rank elapsed seconds. Decompose differences into running / fights / downtime / interaction / XP / grind. Do not select a winner when a reference route or required fact is missing.
6. Only then allow the search algorithm to mix, improve and discover routes, and separately optimize profession leveling.

## Source caveats verified 2026-10-22

- QuestieDB's `support/Forever/QuestXP/xpDB-classic.lua` carries Classic-seeded data (e.g. quest 788 has XP 170). This is **not** proof of the live Forever-server reward. New quest 97279 is not in that XP table.
- QuestieDB's original Forever quest and NPC dataset adopted converted Era metadata (see `docs/forever-data.md`). Metadata and NPC HP cannot automatically be called contemporary Forever observations.
- Journey's exported CSV still contains unexecuted `ERA`, `FOREVER` and class branches.
- ForeverWisp has many approximate coordinates and alternative class quests.
- RestedXP's Forever standard guides cover 1–22 in the checked public tree. AoE alternatives must stay excluded; the complete per-action extraction is not yet in our local workspace.
- Polestar publishes guide sources through 60 but builds its time estimates using modeled combat and travel. The local partial CSV parser test is not a complete 1–30 baseline.

## Evidence records

`examples/era_seeded_facts.json` deliberately contains **unverified reference numbers**; it is a test for source classification, not valid Forever XP.

```json
{
  "schema": 1,
  "facts": [
    {
      "field": "quest_xp",
      "id": 788,
      "value": 170,
      "origin": "classic_era_seeded",
      "source": "QuestieDB/support/Forever/QuestXP/xpDB-classic.lua",
      "revision": "f844007b329b72f18cf2bda3d87540012294732a"
    }
  ]
}
```

Valid Forever quest XP requires `origin="forever_server_observed"`, `observed_at`, `sample_count`, a traceable `source` and `revision`. Other fields can be direct-client evidence only where the client actually states the fact.

## Run a guide-source audit

```bash
python -m unittest discover -s tests -v
python -m foreverroutes source-check \
  --source journey \
  --source-path /local/guide_exports/Journey_Quest_Routen_Analyse.csv \
  --guide H01D_FVR \
  --facts examples/era_seeded_facts.json \
  --target-level 30
```

Output reports coverage and blockers with `elapsed_seconds: null` and `status: NICHT_VERGLEICHBAR` until the route has been independently verified. Source guide texts and waypoints from restricted sources must remain outside the MIT repository.
