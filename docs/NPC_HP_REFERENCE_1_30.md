# NPC HP for ForeverRoutes level 1–30 (reference mode)

**Decision 2026-10-22:** Classic HP is an ACCEPTED provisional fallback, not a blocker. Do not pretend it was directly measured on the WoW Forever server. NPC XP, drop rates and travel still need their own sources.

## Verified upstream structure

Pinned QuestieDB revision: `f844007b329b72f18cf2bda3d87540012294732a`.

`data/Forever/foreverNpcDB.lua` contains raw values named `minLevelHealth`, `maxLevelHealth`, `minLevel`, `maxLevel`. This source has **10,119 NPC records**, including **4,545** overlapping NPC level 1–30; **4,508** of those have non-sentinel HP (`>1`) in both columns. For combat slightly above level 30, **4,786** records overlap 1–33, though some have sentinel values. Counts refer to raw NPC IDs, NOT spawn populations, distinct enemies, or separately verified Forever measurements.

**Critical warning:** `QuestieDB`'s **public Lua API explicitly returns compatibility placeholders 0 and 1** from the deprecated NPC HP getters. It is **incorrect** to import combat HP from the runtime QuestieDB API; use the raw NPC table for this fallback. Source: [QuestieDB API documentation](https://github.com/Questie/QuestieDB/blob/f844007b329b72f18cf2bda3d87540012294732a/docs/api.md).

## HP selection

1. Explicit level-specific observed Forever value: `forever_verified` / `foreverdb_observed`.
2. Level-specific NPC HP reported on Wowhead Forever: `wowhead_forever`.
3. Raw NPC HP from QuestieDB Forever, carried as `questiedb_classic_reference` because the initial Forever DB imported the Classic baseline.
4. If neither exists, unresolved – no invented HP formula.

For NPCs with variable levels, use the raw NPC's min/max endpoints. Middle levels receive **linear interpolation labelled as interpolation** (not observed HP). A specific level beyond the NPC's observed/recorded level range is not silently extrapolated.

For each `mobs` entry, the importer stores source, source URL, source revision, NPC ID, selected NPC level, and selection/interpolation method in `mob_provenance`. Existing health values are preserved; no XP or DPS values are invented.

## Usage

```bash
# Download a source-revision-pinned local QuestieDB raw file and build an NPC reference snapshot.
python -m foreverroutes.mobdata --fetch --out .cache/mob_hp_1_33.json --max-level 33

# Or import an already-downloaded local file without accessing the network.
python -m foreverroutes.mobdata --npc-source ./data/Forever/foreverNpcDB.lua --out .cache/mob_hp_1_33.json

# Enrich only missing HP in a world model. Existing XP and DPS remain unchanged.
python -m foreverroutes.mobdata --fetch --out .cache/mob_hp_1_33.json --world world_input.json --world-out world_with_reference_hp.json --overrides examples/mob_hp_overrides_template.json
```

### Input model

```json
{"mobs":{"npc:3126":{"hp":null,"xp":48,"level":8}}}
```

### HP overlay format

The optional `--overrides` file is organized as NPC ID → exact NPC level → one observation or an array of observations. For example, see `examples/mob_hp_overrides_template.json`. Values and source links in that file are **format examples, not a verified release dataset**.

`foreverdb_observed` always beats `wowhead_forever`; both beat `questiedb_classic_reference`. If two input sources disagree, neither observation is silently deleted from the overlay input file.

## Known limitations

- Classic HP may differ on Forever, particularly for new or rebalanced NPCs. Exception handling must work per NPC and level, not as a global multiplier.
- Mob `level` in the world model must be the **NPC level**, not the player's current level.
- Mob rank (elite/rare), group fights, spawn density, armor, mana regeneration, downtime and mob XP remain separate concerns.
- Full 1–30 guide replay needs valid paths, NPC/quest matches, XP, and dynamic game constraints. Merely importing HP does not establish which guide is faster.
- QuestieDB source text/data licensing is not fully uniform; importer downloads data locally. The MIT repo does not redistribute QuestieDB's complete raw files or generated dataset.