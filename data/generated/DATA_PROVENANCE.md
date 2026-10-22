# QuestieDB working dataset (private reference only)

The numeric quest/NPC/spawn/XP/loot facts in `questie_level1_30_reference.json` are **derived from**:
- QuestieDB (Questie team): https://github.com/Questie/QuestieDB, SHA `f844007b329b72f18cf2bda3d87540012294732a`
- Classic XP formula: https://github.com/cmangos/mangos-classic/blob/master/src/game/Tools/Formulas.h
- User-authorized ForeverDB public observations: https://foreverdb.net

**Licensing:** this private reference snapshot is **not** distributed under the ForeverRoutes MIT code license. QuestieDB's root LICENSE.md notes complex historic rights / proposed GPLv3 terms. Keep source attribution; do not publicly redistribute the dataset until applicable database and code licensing is reviewed.

**Accuracy:** Classic quest XP, raw Classic NPC HP, Classic item-drop samples and Classic XP equations are reference values, **not** live Forever measurements. The catalog marks objective counts unknown; a real travel graph, class-specific combat profiles, respawn observations and complete Forever-only quests must still be supplied before fastest 1–30 guide claims.

The source code can regenerate the reference file using the pinned inputs; generated data is intentionally stored in a separate path rather than mixed with original simulator source.
