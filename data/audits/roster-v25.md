# Roster review: timeline model v2.5

The 112 shipped routes cover 56 supported race/class/faction contexts through level 20, with solo and optional dungeon variants. The quest dataset was rebuilt through level 60 from its pinned sources; this does not extend the shipped optimized route coverage beyond 20.

All 404 optimized preparation scenarios passed final action-by-action replay and the time ledger under the compiled v2.5 model. The final repair pass verified all 404 and needed no replacements. Every required class chain is checked when resuming an older scenario as well as during final verification. This matters when improved sources restore a previously unavailable class quest.

The complete evidence is in [roster-v25.json.gz](roster-v25.json.gz), the compact variant table in [roster-v25-summary.json](roster-v25-summary.json), and the frozen input matrix in [roster-inputs.json.gz](roster-inputs.json.gz). The 112 ordinary RXP workload projections are retained separately from the 404 optimized scenarios; original RXP instructions are not rewritten.

## Quest and combat corrections

- Preserve the client's elite/group quest type, including when an objective already has a known item source. Keep sourced special-quest restrictions when another source omits the type.
- Resolve explicit RXP mob IDs such as `Name::12345` into real combat sources. Prefer current client level/rank metadata over stale Classic reference values; import missing combat metadata from NorgannonDB's NPC table as well as its item sources.
- Keep real monster drops separate from ground pickups. A waypoint alone is not proof that an item lies on the ground. Explicit use of a quest-provided container remains an item interaction.
- Exclude elite/group quest types from solo selection. Ordinary elite combat requires the pet policy and cannot exceed player level; class-quest encounters have their separate level checks. Use maximum encounter levels and reject combat targets with unknown levels.

The [quest objective audit](quest-objectives-v25.md) covers all 2,922 shipped quest records and documents unresolved references and unsupported acquisition. It is a structural source audit, not proof that every quest has been manually completed in game.

## Reproduction

Build `data/routes-data.json.gz` with `foreverroutes build-data` against [the source lock](../sources/lock.json). Seed a roster directory with `python tools/seed_roster_resume.py <directory>`, then run `TestRebuildRoster` with `FR_ROSTER_REBUILD=<directory>`, `FR_ROSTER_RESUME=1` and the desired `FR_ROSTER_BUDGET`. Run `TestRepairRosterScenarios` with `FR_ROSTER_REPAIR=<directory>` and `TestFinalizeRosterEvidence` with `FR_ROSTER_FINALIZE=<directory>`. Finally apply `python tools/apply_roster_review.py <directory>` and regenerate `python tools/audit_quest_objectives.py`.

Hours are model estimates using the bundled class profiles and preparation assumptions. Search is time-limited and does not prove a global optimum; individual equipment and talents are not separately calibrated.
