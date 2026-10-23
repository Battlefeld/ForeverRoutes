# Level 1–30 replay repair — 23 October 2026

The four saved chains can now produce **diagnostic model times to level 30** with `--extend-to-30`. This is **not** a validated comparison of four complete original guides. The strict acceptance gate still fails, intentionally.

## Changes

- Restore the Orc **Warlock** profile matching the exported RestedXP #next chain. Reject `--warrior` until its own chain has been imported. Apply the same class flags, class eligibility and combat parameters to all four sources.
- Replace the identical Journey-derived extension with two dynamic continuations from the common playable world. RestedXP continuation prioritizes estimated work/distance per XP; Wisp continuation prioritizes level proximity and hub continuity. Check live prerequisites, exclusions, quest level, known counts, known XP and quest-log space before emitting an action. Never mutate the four original event exports.
- Mark every continuation group as generated and retain original coverage separately. No claim that these are official RestedXP/Wisp chapters.
- Detect rejected completions and turn-ins as well as accepts. Report actual any-of/all-of prerequisites, active-parent gates and exclusive quests from the compiled engine arrays.
- Require the requested target, coverage, finite milestone time, known XP, known counts, resolved conditions, every event and reachable travel in the acceptance gate. `--require-complete` returns exit code 2 when any requirement fails.
- Add `--target-level` (2–30) and `--output-dir`. Keep original, generated-continuation and estimated reports separate. Do not overwrite strict status with an estimated run.
- Run the checked-out code in CI. Remove the previous replacement with the older artifacts-branch runner and the mislabeled Warrior run. A green diagnostic process alone no longer makes the acceptance job pass.

## Verification

28 tests passed, 0 failed. The integration tests run all four chains to 30 twice and compare milestone times for determinism, assert zero rejected generated actions, verify unchanged source export hashes, check target 20, reject the mismatched Warrior profile, and confirm the strict failure exit code.

| Saved chain | Original replay end | With generated continuation | Rejected original accepts | Missing playable quest models |
| --- | ---: | ---: | ---: | ---: |
| RestedXP | 22 | 30 | 13 | 10 |
| Journey | 30 | 30 | 3 | 8 |
| ForeverWisp | 21 | 30 | 1 | 11 |
| Polestar | 30 | 30 | 0 | 2 |

No generated continuation action was rejected. There are still 48/25/36/6 failed original actions respectively (including missing world quests and downstream completion/turn-in failures). Unknown XP, objective quantities and inventory conditions are retained as blockers. These are **not** silently converted into success.

The source CSV export explicitly omits parts of travel, vendors, trainers, inventory and grinding. The solo world excludes dungeon/elite and profession-gated quests. Examples include Camping 101 profession quests in Wisp and dungeon/elite objectives. RestedXP/Wisp original coverage ends at 22/20. Fixing the replay cannot supply missing official chapters or make those omitted actions measured. Their full source actions and supported quest/world data must be imported before claiming a zero-skip original-guide comparison.

## Reproduce

Use the pinned Polestar engine `eb7b47840f032bf711a3c8d7b1df9f3f6551b990`, set `POLESTAR_ROOT` to its checkout and work in `benchmarks/2026-10-23-orc-warlock`:

```sh
node scripts/run_route_comparison.cjs --saved-inputs
node --test tests/*.cjs
node scripts/run_route_comparison.cjs --saved-inputs --extend-to-30 --output-dir .comparison/results/level30
node scripts/run_route_comparison.cjs --saved-inputs --extend-to-30 --require-complete --output-dir .comparison/results/acceptance
```

The last command currently exits **2**. See `2026-10-23-replay-verification.json` for exact per-route failures, milestones, generated quest IDs and source hashes. Times are diagnostic model outputs, not measured player `/played` and not a defensible speed ranking.
