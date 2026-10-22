# ForeverRoutes: Level 1–30 data input contract (2026-10-22)

## Goal

Supply an auditable **provisional** input catalog for real guide replay and
quest-versus-grind comparisons. “Using a Classic reference” is not the same as
“validated Forever playtime.” Every value has a source class and is overrideable.

## Sources in order

1. QuestieDB Forever raw quest/npc/item/object tables pinned to
   `f844007b329b72f18cf2bda3d87540012294732a` (Questie team). Not
   automatically Forever-observed; most baseline content is Classic Era.
2. Raw **Classic** NPC HP fields from QuestieDB Classic NPC data, never the
   new API's 0/1 compatibility placeholders.
3. QuestieDB's `support/Forever/QuestXP/xpDB-classic.lua`: *Classic* quest-XP
   reference numbers, not confirmed Forever quest rewards.
4. Questie's Classic Wowhead-derived itemDrop table for **sampled chance per
   NPC**; chance absent => `null`, never guess 30% or 100%. ItemDB links item
   sources, not probabilities.
5. Classic XP-to-next level table 1..30 and cMaNGOS solo XP formulas. No XP from
   critters, pets or no-XP mobs: the NPC's no-XP flag is **not reliably
   present**, so individual mobs need filtering before farming.
6. ForeverDB public recorded observations and Wowhead Forever overrides for
   known field/value/player-level pairs. The API will be added when the key is
   approved; it is not silently queried or represented as complete.

## Generated catalog

`python -m foreverroutes.catalog --cache .cache/questie --output .generated/level_1_30_catalog.json`

Inputs are **downloaded on a machine with internet** from a pinned GitHub
revision. The output contains `quests`, `npcs`, `items`, `objects`,
`level_xp_classic`, `stats`, `sources`, `metadata`.

- Quest selection: all rows with minimum required level 1–30, including quests
  that display a higher quest level. NPC references include a broader level
  window plus quest-related higher-level NPCs.
- Quests: start/finish NPC, object, item IDs; player-level requirements; race
  and class masks; prerequisites (all/any); target IDs by kind; Classic XP
  where present. **Required objective counts are not encoded in these QuestieDB
  reference rows**; the catalog sets them to `null`. Quantities must be
  imported from in-game quest observation or verified guide steps.
- Mobs: spawn coordinates by **zone** (not world meters), min/max level, raw
  Classic min/max HP, rank, faction, NPC flags, model-XP function and a
  *potential* grind flag, not a recommendation.
- Items: quest-item NPC/object sources and Classic Wowhead-derived per-NPC
  item-drop chance where present. If absent, keep `null`; gather a verified
  observation or bound interval instead of arbitrary 30%.
- Objects: zone-relative map coordinates for quest-related objects.

## Blockers before credible fastest-guide rankings

| Input | Catalog status | Additional work |
|---|---|---|
| Quest IDs/prerequisites | Classic baseline | Forever-only quests via ForeverDB/Wowhead |
| Quest XP | Classic baseline | Server overrides & level-scaling |
| Mob HP | Classic baseline | Build-specific corrections |
| Mob kill XP | cMaNGOS formula | Exclude critters/no-XP, server rate |
| Item drop chance | Classic sampled for subset | Server-side drop and correlations |
| Objective quantities | **Unknown by default** | In-game log, verified quest objectives |
| Quest objective location | Zone-relative coordinates | Proper map geometry and walkable paths |
| Travel time | **Not in QuestieDB** | Actual path graph, flights, hearths, deathskips |
| Kill duration | **Not inferred from HP alone** | Class-specific sim DPS / GCD / downtime |
| Respawn/competition | **Not in client tables** | Sampled uncertainty / measured wait times |

**Never feed this catalog directly into `World` with fabricated seconds or
counts to make a run look complete.** The data QA stage must block claiming a
measured or calibrated fastest route when required parts are missing.

## License

QuestieDB root `LICENSE.md` notes unresolved historical contribution rights,
consent limitations and intended GPLv3 terms. The ForeverRoutes MIT license
covers the original importer implementation only; do not copy the QuestieDB
source tables into this repository under an MIT label. Download sourced data
locally and retain attribution and source revision. Data artifacts can be
shared only as permitted by the source's licensing terms.