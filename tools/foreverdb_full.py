#!/usr/bin/env python3
"""Reads the whole ForeverDB quest list (every level) and a few quest details for comparison.

Data from ForeverDB.net (https://foreverdb.net). The key comes only from FOREVERDB_API_KEY and is never printed or
written. At most 60 requests a minute (2.1 s apart); a full list is about 70 requests.
"""
import argparse
import gzip
import json
import os
import sys
import time
import urllib.error
import urllib.parse
import urllib.request

BASE = "https://foreverdb.net/api/v1"
INTERVAL = 2.1
last = 0.0
calls = 0


def get(key, endpoint, params=None):
    global last, calls
    url = BASE + "/" + endpoint + ("?" + urllib.parse.urlencode(params) if params else "")
    for attempt in range(5):
        wait = INTERVAL - (time.monotonic() - last)
        if wait > 0:
            time.sleep(wait)
        last = time.monotonic()
        calls += 1
        req = urllib.request.Request(url, headers={
            "X-API-Key": key, "Accept": "application/json",
            "User-Agent": "ForeverRoutes (https://github.com/Battlefeld/ForeverRoutes)"})
        try:
            with urllib.request.urlopen(req, timeout=35) as r:
                return json.load(r)
        except urllib.error.HTTPError as e:
            if e.code == 429 or 500 <= e.code <= 599:
                time.sleep(min(120, int(e.headers.get("Retry-After") or 10 * (attempt + 1))))
                continue
            raise SystemExit(f"foreverdb {endpoint}: HTTP {e.code}")
        except (urllib.error.URLError, TimeoutError) as e:
            time.sleep(10 * (attempt + 1))
    raise SystemExit(f"foreverdb {endpoint}: no answer after retries")


def rows(answer, label):
    data = answer.get("data")
    if isinstance(data, list):
        return data
    for k in (label, "items", "results", "rows"):
        if isinstance(data, dict) and isinstance(data.get(k), list):
            return data[k]
    raise SystemExit("foreverdb: unknown list shape")


def pages(key, endpoint, label, params=None):
    out, offset = [], 0
    for _ in range(200):
        p = dict(params or {}, limit=100, offset=offset)
        a = get(key, endpoint, p)
        batch = rows(a, label)
        out += batch
        total = (a.get("meta") or {}).get("total")
        if not batch or (total is not None and offset + len(batch) >= total) or (total is None and len(batch) < 100):
            return out
        offset += len(batch)
    raise SystemExit(f"foreverdb {endpoint}: more than 200 pages")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", required=True)
    ap.add_argument("--details", default="844,870,848,5041,788,6062", help="quest ids read through the detail endpoint")
    args = ap.parse_args()
    key = (os.environ.get("FOREVERDB_API_KEY") or "").strip()
    if not key:
        raise SystemExit("FOREVERDB_API_KEY is not set")
    meta = get(key, "meta")
    quests = pages(key, "quests", "quests")
    details = {i: get(key, "quests/" + i).get("data") for i in args.details.split(",") if i}
    changes = pages(key, "changes-seen", "changes", {"kind": "quest"})
    doc = {"schema": "foreverroutes.foreverdb_full.v1",
           "source": {"label": "Data from ForeverDB.net", "url": "https://foreverdb.net",
                      "api_docs": "https://foreverdb.net/developers"},
           "fetched": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
           "game_metadata": meta.get("data"), "game_build_metadata": meta.get("meta"),
           "quests": quests, "quest_details_sample": details, "quest_changes_seen": changes}
    with gzip.GzipFile(args.out, "wb", mtime=0) as f:
        f.write(json.dumps(doc, ensure_ascii=False, sort_keys=True, separators=(",", ":")).encode())
    print(f"{len(quests)} quests, {len(details)} details, {len(changes)} changes, {calls} requests", file=sys.stderr)


if __name__ == "__main__":
    main()
