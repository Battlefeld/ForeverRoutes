# First data-first estimated route replay — 2026-10-23

**Experimental model, NOT verified /played and NOT a defensible fastest-route ranking.** All four unmodified guide chains were evaluated separately with the same pinned Polestar engine and the ForeverRoutes cached quest overlays. Source coverage differs (RestedXP through 22, Wisp through 20, Journey through 30, Polestar through 60).

| Level | RestedXP | Journey | ForeverWisp | Polestar |
|---:|---:|---:|---:|---:|
| 6 | 0:49:58 | 0:46:11 | 0:49:09 | 0:45:26 |
| 12 | 3:11:07 | 3:38:11 | 3:14:01 | 3:07:05 |
| 20 | 11:10:49 | 10:24:36 | 11:15:18 | 10:17:25 |
| 30 | — | 24:53:21 | — | 24:51:52 |

## Provenance and limitations

- Requested precedence: ForeverRoutes → QuestieDB Forever → pfQuest → CMaNGOS → Wago/DB2 → ForeverDungeonJournal → WarcraftDB → ForeverDB → Wowhead Forever → labelled estimation. This is a priority policy, **not evidence every provider was fetched live**. QuestieDB, pfQuest and CMaNGOS were already merged in the pinned simulation input, with ForeverRoutes XP overlays; other providers are only used where values exist in stored input or reference snapshots.
- Engine world XP: 1,454 present ForeverRoutes/merged rewards; 43 zero rewards (including explicit zeros); 112 model-estimated XP values. These counts refer to Horde/Orc solo-world quests, **not all 5,414 source quest records**.
- Estimated fallback numbers are not measurements. Missing objective counts use the engine's 1-target default; other quest activities can receive deterministic proxy time/XP with an explicit assumption trace. Known 0 XP remains 0. Two route conditions in Journey can be uncertain.
- Unknown-world-quest IDs: RestedXP 10, Journey 8, ForeverWisp 13, Polestar 2. Remaining unknown target-count quests in the sampled guides: RestedXP 1, Journey 1, Wisp 2, Polestar 2. Skipped accepts remain: 13/3/1/0. These incomplete/unsupported actions can change rankings.
- All four can be compared at level 6, 12 and 20 *as exploratory model checkpoints*; level 30 only compares Journey vs Polestar and is especially sensitive to small differences.

## Reproduce

Run the pinned GitHub Actions workflow or in `benchmarks/2026-10-23-orc-warlock` with the saved inputs and a pinned Polestar checkout:

```bash
node scripts/run_route_comparison.cjs --saved-inputs --estimate-missing
```

Full tested run: https://github.com/Battlefeld/ForeverRoutes/actions/runs/39949581165
Implementation PR: https://github.com/Battlefeld/ForeverRoutes/pull/10

Do not publish speed claims or optimize routes based on this report until unsupported route actions, missing route chapters and location/condition uncertainties are assessed.
