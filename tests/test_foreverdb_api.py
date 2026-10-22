import json
import tempfile
import unittest
from pathlib import Path
from urllib.error import HTTPError

from foreverroutes.foreverdb_api import ForeverDBClient, QuotaError, APIError, candidate_ids, export


class Response:
    def __init__(self, payload):
        self.payload = json.dumps(payload).encode()
    def read(self, size):
        return self.payload
    def __enter__(self):
        return self
    def __exit__(self, *args):
        pass


class FakeAPI:
    def __init__(self):
        self.urls = []
        self.requests = []
    def __call__(self, request, timeout):
        url = request.full_url
        self.urls.append(url)
        self.requests.append(request)
        assert request.get_header("X-api-key") == "secret-test"
        if url.endswith("/meta"):
            return Response({"data": {"build": "Forever 1.60"}, "meta": {"build": "test"}})
        if "/quests/" in url:
            return Response({"data": {"id": 42, "level": 10}, "meta": {}})
        if "/changes-seen" in url:
            return Response({"data": [{"kind": "quest"}], "meta": {"total": 1}})
        if "offset=0" in url:
            return Response({"data": [{"id": 42}, {"id": 43}], "meta": {"total": 3}})
        return Response({"data": [{"id": 44}], "meta": {"total": 3}})


class ForeverDBTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.addCleanup(self.tmp.cleanup)
        self.fake = FakeAPI()
        self.elapsed = [100.0]
        self.client = ForeverDBClient(key="secret-test", cache=Path(self.tmp.name),
                                     max_requests=20, min_interval=1.05,
                                     opener=self.fake, clock=lambda: self.elapsed[0],
                                     sleep=lambda seconds: self.elapsed.__setitem__(0, self.elapsed[0] + seconds))

    def test_pagination_and_cache(self):
        data = self.client.pages("quests", label="quests", params={"minLevel": 1, "maxLevel": 30}, limit=2)
        self.assertEqual([o["id"] for o in data], [42, 43, 44])
        self.assertEqual(self.client.requests, 2)
        again = self.client.pages("quests", label="quests", params={"minLevel": 1, "maxLevel": 30}, limit=2)
        self.assertEqual(again, data)
        self.assertEqual(self.client.requests, 2)
        self.assertEqual(self.client.cache_hits, 2)
        self.assertNotIn("secret-test", "".join(p.read_text() for p in Path(self.tmp.name).glob("*.json")))

    def test_full_export(self):
        result = export(self.client, candidates={42, 99}, max_details=2)
        self.assertEqual(result["audit"]["quest_list_rows"], 3)
        self.assertEqual(result["audit"]["quest_details_fetched"], 2)
        self.assertEqual(result["audit"]["quest_details_new_this_run"], 2)
        self.assertEqual(result["audit"]["relevant_candidates_found"], 1)
        self.assertEqual(result["audit"]["change_events"], 1)
        self.assertEqual(result["source"]["url"], "https://foreverdb.net")

    def test_incremental_resume_without_refetching_known_details(self):
        first=export(self.client, candidates={42,44}, max_details=1, with_changes=False)
        self.assertEqual(set(first["quest_details"]), {"42"})
        before=self.client.requests
        second=export(self.client, candidates={42,44}, max_details=1,
                      with_changes=False, previous=first)
        self.assertEqual(set(second["quest_details"]), {"42","44"})
        self.assertEqual(second["audit"]["quest_details_new_this_run"], 1)
        self.assertEqual(self.client.requests, before+1)
        third=export(self.client, candidates={42,44}, max_details=1,
                     with_changes=False, previous=second)
        self.assertEqual(set(third["quest_details"]), {"42","43","44"})
        self.assertTrue(third["audit"]["detail_completion"])

    def test_budget_guard(self):
        self.client.max_requests = 1
        self.client.get("meta")
        with self.assertRaises(QuotaError):
            self.client.get("quests", {"limit": 100, "offset": 0})
        self.assertEqual(len(self.fake.urls), 1)

    def test_invalid_endpoint_and_interval(self):
        with self.assertRaises(ValueError):
            self.client.get("https://evil.example/quests")
        with self.assertRaises(ValueError):
            ForeverDBClient(key="x", cache=Path(self.tmp.name), min_interval=0.1)

    def test_http_429_stops(self):
        def limited(req, timeout):
            raise HTTPError(req.full_url, 429, "too many", {"Retry-After": "8"}, None)
        self.client.opener = limited
        with self.assertRaisesRegex(QuotaError, "Retry-After=8"):
            self.client.get("meta")

    def test_empty_list_rejected_when_wrong_shape(self):
        self.client.opener = lambda req, timeout: Response({"data": {}, "meta": {"total": 4}})
        with self.assertRaises(APIError):
            self.client.pages("quests", label="quests")

    def test_candidates(self):
        path = Path(self.tmp.name) / "ids.json"
        path.write_text(json.dumps({"sources": {"a": {"zone": [42, 44]}}}))
        self.assertEqual(candidate_ids(path), {42, 44})


if __name__ == "__main__":
    unittest.main()