# ForeverRoutes — generated data snapshots

This private `artifacts` branch is separate from the simulator source on `main`.

- `foreverdb/quests-1-30.json`: ForeverDB quest API snapshot, source labeled as
  **Data from ForeverDB.net**. Access subject to ForeverDB's usage rules.
- `catalog/level-1-30.json`: QuestieDB/ForeverDB reference catalog.
- `legacy-actions/runs/<run-id>/<artifact-id>-<name>.zip`: historical GitHub
  Actions artifact ZIPs, if there are any to preserve.

New snapshots are saved directly to this branch instead of consuming GitHub
Actions artifact storage. The optional migration workflow backs up any remaining
old Actions ZIPs, verifies the remote Git copies by SHA-256, then deletes the
originals. Empty inventories require no changes.

Large ZIPs are split into 40-MiB Git blobs plus a SHA-256 `.parts.json` manifest.
API credentials and user secrets must never be committed.
