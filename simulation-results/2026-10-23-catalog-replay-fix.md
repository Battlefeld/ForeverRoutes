# Full-catalog replay repairs — 23 October 2026

All four saved Orc Warlock chains reach level 30 with `--extend-to-30`. **Wisp now passes the complete provisional model gate.** Its 20–30 continuation remains generated, not an official Wisp chapter. The four-original-guide comparison is not complete; the combined strict command still returns **2** and no speed ranking is published.

## Root causes repaired

- The old CSV replay flattened nested Journey conditions, lost Wisp optional/class/trip metadata, and applied a conditional RXP quest guard to an entire step. Replay now loads the existing full catalog while retaining each source's own chapter sequence and SHA.
- RXP step 42 of 12–17 Barrens: the `isOnQuest 842` guard applies only to Tauren Shamans. It no longer prevents an Orc Warlock from accepting Plainstrider Menace (845), avoiding the downstream chain of rejected accepts.
- RXP XP skipstep guards execute before the step and retain per-directive class scope. Unknown inventory guards block the step explicitly instead of attempting an unjustified turn-in. Unknown inventory is never assumed to be zero or silently granted.
- An optional Shaman variant of a Journey acceptance no longer makes the mandatory Warlock variant optional. Nested expansion branches, `NONE`, `HC` and `INGROUP` are preserved/evaluated.
- Wisp profession/dungeon options stay optional according to the source. `--with-optional` retains their original trip and profession metadata for diagnosis, without claiming unsupported optional content is modeled.
- A requested acceptance rejected only for a prerequisite can be retried on the next actual visit to the same giver. Engine prerequisites, log capacity, level, and talk/work costs remain enforced. Wisp's 3090 is recovered after turning in 788 at the same giver; input events are unchanged and the recovery is recorded.
- A legal NPC acceptance is not rejected merely because the displayed quest level is red. The real minimum level remains enforced. Item-starting combat work keeps its original rules.
- Indexed completions now execute the specified objective rather than completing the entire quest. Mixed-site kill objectives use NPC IDs; item objectives use pinned Questie drop references. Single-object spawn grouping reproduces the pinned world's 70-yard first-site rule and source-objective order. Missing objective definitions stay explicit errors (97891/98252).
- RXP `completewith` instructions remain parallel hints, with objective work fulfilled by later required instructions/turn-in rather than forcing an early extra trip.
- Wisp's explicit movement coordinates are converted using pinned published map frames and added to the run geometry before evaluator construction. The shared travel model prices actual moves. No synthetic quest, extra XP or upstream engine edits are used.
- The cached world incorrectly excluded real quests 1149 and 1150 because their titles contain “Test”. Their source-defined giver, objective, coordinates, prerequisite, XP and drop probability are restored from pinned data. The scripted jump duration of 1149 remains explicitly unvalidated.
- Missing XP for 79096 (160) and 98070 (2450) is filled from indexed Wowhead Forever primary-page excerpts. The direct pages require JavaScript verification; evidence quality and URLs are recorded. Existing known rewards/zero rewards retain precedence.

## Validation

42 tests passed, 0 failed. Integration runs replay all four chains twice and compare milestone times, check target 20, reject a mismatched Warrior chain, verify unchanged original exports, and confirm failure exit code 2 when the combined gate is incomplete.

| Chain | End level | Rejected accepts | Failed quest actions / missing mappings | Missing quest models | Complete provisional gate |
| --- | ---: | ---: | ---: | ---: | --- |
| RestedXP + generated continuation | 30 | 0 | 15 | 10 | No |
| Journey | 30 | 0 | 12 | 4 | No |
| Wisp + generated continuation | 30 | 0 | 0 | 0 | Yes |
| Polestar | 30 | 0 | 0 | 0 | No |

Polestar still has unresolved quantities/mechanics for 1090 and 4966, plus the unvalidated scripted duration of 1149. RXP/Journey still contain unimplemented source action types and inventory conditions, and quests outside the cached solo world (including elite, dungeon and profession work). Such actions are now included in acceptance rather than disappearing through the CSV export. New timing outputs are diagnostic and are not a speed ranking across these incomplete workloads.

Exact source hashes, remaining failures, omitted action counts, generated quest IDs, recovery traces and milestone outputs are in `2026-10-23-catalog-replay-verification.json`. The previous repair report is a historical snapshot.

Runtime uses only the committed catalog/snapshots and pinned Polestar engine; the Questie checkout/Lua interpreter are not required to replay. Data provenance is embedded in `data/world-gap-sources.json`, `data/objective-drop-refs.json`, `data/replay-map-frames.json` and `data/replay-xp-references.json`.
