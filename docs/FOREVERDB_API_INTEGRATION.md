# ForeverDB-API in ForeverRoutes

**Attribution (required)**: [Data from ForeverDB.net](https://foreverdb.net). API documentation and binding rules: https://foreverdb.net/developers

## Setup

1. GitHub → Battlefeld/ForeverRoutes → Settings → Secrets and variables → Actions → New repository secret.
2. Name: `FOREVERDB_API_KEY`, value: your approved personal ForeverDB API key. Never put the key in commits or chat.
3. GitHub → Actions → **Import ForeverDB API (manual)** → Run workflow.
4. Download the `foreverdb-quests-1-30` artifact for offline data checks.

## Limits and use

- Official limits: **60 requests/minute and 10,000 per day per key**. The importer normally throttles at 2.1 seconds/request (~28.6/min) and caps uncached network requests at **240 per run**, so one run will not exhaust the daily quota.
- Multiple concurrent runs **share the same ForeverDB key limits**. Avoid running more than one workflow for the same key simultaneously; quotas are not coordinated across machines or secrets consumers.
- Cached responses are reused for 7 days by default. The Workflow is **manual**; no background automatic API traffic on commits.
- `429` aborts the run and reports `Retry-After`. The exporter **does not keep hammering** the service. `401/403` reports a key/access issue without logging the key.
- Quest list `minLevel=1&maxLevel=30` queries *quest level*. Quest requirement level is a different field; this list alone does not prove coverage of every quest a level-30 character can accept.
- `/api/v1/quests/{id}` is documented to contain level/side/zone, starters and finishers with source status. **It does not promise objective counts/XP**. `/api/v1/changes-seen?kind=quest` includes recorded player-observed changes such as quest XP, but the schema must be inspected before mapping any values automatically.
- Snapshot stores source metadata, counts, raw documented API rows and visible ForeverDB link, **not verified simulator inputs**. QuestieDB remains the structural baseline; field-level overlays follow only after mapping and validation.
- ForeverDB terms: visibly link to https://foreverdb.net **wherever its data appears** (README, any eventual in-game UI, reports), do not resell the data or claim it as your own. Blizzard game data remains Blizzard game data.

## Optional local use

```sh
export FOREVERDB_API_KEY='...'
python -m foreverroutes.foreverdb_api --max-details 100 --max-requests 240
```

Never include the key in command-line flags, shell history, shared screenshots, test fixtures, artifacts or logs. Use shell/CI secret injection instead of literal exports when possible.