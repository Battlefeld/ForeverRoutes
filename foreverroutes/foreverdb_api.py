"""Conservative, on-demand ForeverDB API exporter for ForeverRoutes.

Only public endpoints documented at https://foreverdb.net/developers.
Credentials are taken exclusively from FOREVERDB_API_KEY, never CLI arguments.
Exports are *source snapshots*, not invented Forever quest facts.
"""
from __future__ import annotations

import argparse
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import re
import time
from urllib.parse import urlencode, urlsplit
from urllib.request import Request, urlopen
from urllib.error import HTTPError

BASE = "https://foreverdb.net/api/v1"
CREDIT = {"label": "Data from ForeverDB.net", "url": "https://foreverdb.net", "api_docs": "https://foreverdb.net/developers"}


class APIError(RuntimeError):
    pass


class QuotaError(APIError):
    pass


class ForeverDBClient:
    def __init__(self, *, key: str, cache: Path, max_requests: int = 240,
                 ttl_seconds: int = 7 * 86400, min_interval: float = 2.1,
                 opener=None, clock=None, sleep=None):
        if not key or not key.strip():
            raise ValueError("FOREVERDB_API_KEY is required; do not put secrets in code")
        if not 1 <= max_requests <= 1000:
            raise ValueError("max_requests must be 1..1000 (below API's 10,000/day)")
        if min_interval < 1.05:
            raise ValueError("At least 1.05 seconds between requests (60/min limit)")
        if ttl_seconds < 0:
            raise ValueError("ttl_seconds must be nonnegative")
        self.key = key.strip()
        self.cache = Path(cache)
        self.max_requests = max_requests
        self.ttl_seconds = ttl_seconds
        self.min_interval = min_interval
        self.opener = opener or urlopen
        self.clock = clock or time.monotonic
        self.sleep = sleep or time.sleep
        self.last_request = None
        self.requests = 0
        self.cache_hits = 0
        self.cache.mkdir(parents=True, exist_ok=True)

    def get(self, endpoint: str, params: dict | None = None):
        # Never follow API-provided pagination URLs; only approved first-party endpoint paths.
        if not re.fullmatch(r"(?:meta|quests(?:/\d+)?|changes-seen)", endpoint):
            raise ValueError("Unsupported API endpoint")
        param = urlencode(sorted((params or {}).items()))
        url = BASE + "/" + endpoint + ("?" + param if param else "")
        key = hashlib.sha256(url.encode()).hexdigest()
        path = self.cache / (key + ".json")
        if path.exists() and time.time() - path.stat().st_mtime <= self.ttl_seconds:
            self.cache_hits += 1
            return json.loads(path.read_text(encoding="utf-8"))
        if self.requests >= self.max_requests:
            raise QuotaError("Local request budget exhausted; no more API calls made")
        if self.last_request is not None:
            self.sleep(max(0, self.min_interval - (self.clock() - self.last_request)))
        self.last_request = self.clock()
        self.requests += 1  # Count even failed requests.
        request = Request(url, headers={"X-API-Key": self.key, "Accept": "application/json",
                                        "User-Agent": "ForeverRoutes/0.6 (https://github.com/Battlefeld/ForeverRoutes)"})
        try:
            with self.opener(request, timeout=35) as response:
                data = response.read(8_000_000)
        except HTTPError as ex:
            if ex.code == 429:
                retry = ex.headers.get("Retry-After", "unknown") if ex.headers else "unknown"
                raise QuotaError(f"ForeverDB rate limit (429), Retry-After={retry}; stop this run") from None
            if ex.code in (401, 403):
                raise APIError(f"ForeverDB access rejected ({ex.code}); check key approval/secret") from None
            raise APIError(f"ForeverDB HTTP {ex.code} for {endpoint}") from None
        try:
            payload = json.loads(data)
        except (UnicodeDecodeError, ValueError) as exc:
            raise APIError(f"Non-JSON response from {endpoint}") from exc
        if not isinstance(payload, dict) or "data" not in payload:
            raise APIError(f"Unexpected API response for {endpoint}: missing data")
        # Never write a secret if a server unexpectedly echoes it.
        scrubbed = json.dumps(payload, ensure_ascii=False).replace(self.key, "[REDACTED]")
        tmp = path.with_suffix(".tmp")
        tmp.write_text(scrubbed, encoding="utf-8")
        tmp.replace(path)
        return json.loads(scrubbed)

    @staticmethod
    def _list(response: dict, label: str) -> list:
        data = response.get("data")
        if isinstance(data, list):
            return data
        if isinstance(data, dict):
            for candidate in (label, "items", "results", "rows"):
                if isinstance(data.get(candidate), list):
                    return data[candidate]
        raise APIError("Unknown list shape; refusing to silently discard API data")

    def pages(self, endpoint: str, *, label: str, params=None,
              limit: int = 100, max_pages: int = 120):
        if not 1 <= limit <= 100 or max_pages < 1:
            raise ValueError("Invalid pagination settings")
        offset = 0
        all_rows = []
        for _ in range(max_pages):
            p = dict(params or {}, limit=limit, offset=offset)
            answer = self.get(endpoint, p)
            batch = self._list(answer, label)
            all_rows.extend(batch)
            meta = answer.get("meta") or {}
            if not isinstance(meta, dict):
                raise APIError("Invalid pagination metadata")
            total = meta.get("total")
            if isinstance(total, int) and total >= 0:
                if offset + len(batch) >= total:
                    return all_rows
            elif len(batch) < limit:
                return all_rows
            if not batch:
                return all_rows
            # Always derive offset locally; never fetch a URL returned in metadata.
            offset += len(batch)
            if offset <= p["offset"]:
                raise APIError("Pagination not progressing")
        raise APIError("Pagination exceeded max_pages; incomplete result not exported")


def _quest_id(row) -> int | None:
    if not isinstance(row, dict):
        return None
    value = row.get("id", row.get("questId", row.get("quest_id")))
    try:
        n = int(value)
    except (ValueError, TypeError):
        return None
    return n if n > 0 else None


def candidate_ids(path: Path) -> set[int]:
    doc = json.loads(path.read_text(encoding="utf-8"))
    result = set()
    for guides in doc.get("sources", {}).values():
        for ids in guides.values():
            result.update(n for v in ids if (n := int(v)) > 0)
    return result


def export(client: ForeverDBClient, *, candidates: set[int] | None,
           max_details: int = 100, with_changes: bool = True):
    if max_details < 0 or max_details > 300:
        raise ValueError("max_details must be 0..300")
    overview = client.get("meta")
    # According to ForeverDB docs min/maxLevel filters quest *level*, not minimum take level.
    listing = client.pages("quests", label="quests", params={"minLevel": 1, "maxLevel": 30})
    seen_ids = {_quest_id(x) for x in listing}
    requested = sorted((candidates or set()) & seen_ids)
    # Candidate quests are preferentially detailed; cap protects the daily API budget.
    selected = requested[:max_details]
    details = {}
    for qid in selected:
        details[str(qid)] = client.get(f"quests/{qid}")
    changes = client.pages("changes-seen", label="changes", params={"kind": "quest"}) if with_changes else []
    snapshot = {
        "schema": "foreverroutes.foreverdb_snapshot.v1",
        "source": CREDIT,
        "game_metadata": overview.get("data"),
        "game_build_metadata": overview.get("meta"),
        "queried_quest_level_range": [1, 30],
        "quests": listing,
        "quest_details": details,
        "quest_changes_seen": changes,
        "audit": {
            "quest_list_rows": len(listing),
            "quest_details_fetched": len(details),
            "relevant_candidates_found": len(requested),
            "relevant_candidates_not_detailed": len(requested) - len(details),
            "change_events": len(changes),
            "calls_this_run": client.requests,
            "cache_hits": client.cache_hits,
            "note": "Source snapshot. The API does not promise all objective counts, XP and spawn details in quest detail JSON. Validate field schemas before mapping to simulator input."
        }
    }
    return snapshot


def main(argv=None):
    ap = argparse.ArgumentParser(description="Private ForeverDB source exporter with bounded API usage")
    ap.add_argument("--cache", type=Path, default=Path(".cache/foreverdb-api"))
    ap.add_argument("--output", type=Path, default=Path(".generated/foreverdb_quests_1_30.json"))
    ap.add_argument("--guide-ids", type=Path, default=Path("data/validation/guide_quest_id_candidates.json"))
    ap.add_argument("--max-details", type=int, default=100)
    ap.add_argument("--max-requests", type=int, default=240)
    ap.add_argument("--min-interval", type=float, default=2.1)
    ap.add_argument("--cache-days", type=int, default=7)
    ap.add_argument("--skip-changes", action="store_true")
    args = ap.parse_args(argv)
    secret = os.environ.get("FOREVERDB_API_KEY")
    if not secret:
        ap.error("FOREVERDB_API_KEY environment variable not set; configure a GitHub Actions secret")
    c = ForeverDBClient(key=secret, cache=args.cache, max_requests=args.max_requests,
                       ttl_seconds=args.cache_days * 86400, min_interval=args.min_interval)
    ids = candidate_ids(args.guide_ids) if args.guide_ids.is_file() else set()
    result = export(c, candidates=ids, max_details=args.max_details, with_changes=not args.skip_changes)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    tmp = args.output.with_suffix(args.output.suffix + ".tmp")
    tmp.write_text(json.dumps(result, ensure_ascii=False, indent=1), encoding="utf-8")
    tmp.replace(args.output)
    print(json.dumps({"output": str(args.output), "audit": result["audit"]}, ensure_ascii=False))


if __name__ == "__main__":
    main()