import unittest
from foreverroutes import warcraftdb_gaps as gap

PAGE_FOREVER = """Rare Books
Objectives
Geomancy for Curious Young Wizards
Arcane Explainer: Magical Stuff in Simple Words
Rewards
Choose one:
Northshire Hammer
Militia Sword
790 Experience Points
+100 reputation with Stormwind
Name | Rare Books
Type | Elite
Zone | Elwynn Forest
Level | 7
Minimum Level | 1
"""
PAGE_CLASSIC = """Sharptalon's Claw
Objectives
Sharptalon's Claw
Rewards
2,450 Experience Points
+250 reputation with Thunder Bluff
Name | Sharptalon's Claw
Zone | Ashenvale
Side | Horde
Level | 30
Requires Level | 20
"""

class GapTests(unittest.TestCase):
    def test_pending_unseen_quest_ids_and_zero_xp(self):
        import datetime as dt
        now=dt.datetime(2026,10,23, tzinfo=dt.timezone.utc)
        scanned=now.isoformat()
        rows={
            "16":{"fields":{"xp":0},"checked":scanned},
            "254":{"fields":{},"checked":scanned},
            "91775":{"fields":{},"checked":(now-dt.timedelta(hours=49)).isoformat()}
        }
        # 308 is new: never-scanned IDs MUST be processed immediately.
        self.assertEqual(gap.pending_ids([16,254,308,91775],rows,48,now),
                         [308,91775])

    def test_warcraftdb_xp_short_circuits_unneeded_wowhead(self):
        from unittest.mock import patch
        calls=[]
        def fake_get_page(page,url,robots,delay):
            calls.append(url)
            return "ok",("Verified Quest - Forever - Warcraft DB",
                         "Name | Verified Quest\nLevel | 12\n350 Experience Points"),{}
        with patch.object(gap,"get_page",side_effect=fake_get_page):
            out=gap.attempts_for_quest(None,90001,{},0)
        self.assertEqual(out["fields"]["xp"],350)
        self.assertEqual(len(calls),1)
        self.assertTrue(calls[0].startswith("https://forever.warcraftdb.com"))

    def test_missing_robots_txt_allows_crawling_404_but_not_errors(self):
        from unittest.mock import patch
        from urllib.error import HTTPError
        url="https://forever.warcraftdb.com/data/quest/90001"
        with patch.object(gap,"urlopen",side_effect=HTTPError(url,404,"not found",{},None)):
            self.assertTrue(gap.robots_allowed(url,{}))
        with patch.object(gap,"urlopen",side_effect=HTTPError(url,403,"forbidden",{},None)):
            with self.assertRaises(PermissionError):
                gap.robots_allowed(url,{})

    def test_831_ids_from_actual_audit_not_file_list(self):
        ids=list(range(1,832))
        a={"schema":"foreverroutes.unified_audit.v1","missing":{"quests_without_xp":ids}}
        self.assertEqual(len(gap.ids_from_audit(a)),831)
    def test_parse_forever_site_xp_and_objectives(self):
        q=gap.parse_warcraftdb("Rare Books - Forever - Warcraft DB",PAGE_FOREVER)
        self.assertEqual(q["xp"],790)
        self.assertEqual(q["min_level"],1)
        self.assertEqual(q["level"],7)
        self.assertEqual(q["reputation"][0]["amount"],100)
        self.assertEqual(len(q["objectives"]),2)
    def test_parse_classic_site(self):
        q=gap.parse_warcraftdb("Sharptalon's Claw - WoW Classic - Warcraft DB",PAGE_CLASSIC)
        self.assertEqual(q["xp"],2450)
        self.assertEqual(q["min_level"],20)
        self.assertEqual(q["side"],"Horde")
    def test_zero_xp_is_present(self):
        q=gap.parse_warcraftdb("Quest No XP", "Name | Quest No XP\nLevel | 1\n0 Experience Points")
        self.assertIn("xp",q)
        self.assertEqual(q["xp"],0)
    def test_not_assume_missing_means_zero(self):
        q=gap.parse_warcraftdb("Quest Unknown XP", "Name | Quest Unknown XP\nLevel | 5")
        self.assertNotIn("xp",q)
    def test_web_fills_missing_and_preserves_other_facts(self):
        raw={"schema":"foreverroutes.unified_quests.v1","stats":{"quests_missing_xp":1},"sources":{},
             "quests":{"91777":{"id":91777,"status":"forever_listed",
             "fields":{"name":{"value":"Original","source":"foreverdb_forever","quality":"forever_reference","url":"https://foreverdb.net/quest/91777"}},
             "objectives":[{"kind":"object","id":123}],"locations":{},"not_actionable":True}}}
        overlay={"schema":gap.SCHEMA,"generated":"2026-10-23","target_count":831,"quests":{"91777":{
            "fields":{"name":"Rare Books","level":7,"min_level":1,"xp":790,
                      "objectives":["Book A","Book B"]},
            "field_sources":{k:{"source":"warcraftdb_forever","url":"https://forever.warcraftdb.com/data/quest/91777",
                          "quality":"forever_reference"} for k in ("name","level","min_level","xp","objectives")}}}}
        data,changes=gap.apply(raw,overlay)
        q=data["quests"]["91777"]
        self.assertEqual(q["fields"]["xp"]["value"],790)
        self.assertEqual(q["fields"]["name"]["value"],"Original")
        self.assertEqual(q["objectives"],[{"kind":"object","id":123}])
        self.assertEqual(data["stats"]["quests_missing_xp"],0)
        self.assertEqual(changes["xp_inserted"],1)
    def test_classic_value_labelled_not_forever(self):
        raw={"schema":"foreverroutes.unified_quests.v1","stats":{},"sources":{},
             "quests":{"16":{"id":16,"status":"questie_reference_only","fields":{},"objectives":[],"locations":{},"not_actionable":True}}}
        overlay={"schema":gap.SCHEMA,"quests":{"16":{"fields":{"xp":0},"field_sources":{
          "xp":{"source":"warcraftdb_classic","url":"https://classic.warcraftdb.com/data/quest/16",
                "quality":"classic_reference"}}}}}
        db,_=gap.apply(raw,overlay)
        self.assertEqual(db["quests"]["16"]["fields"]["xp"]["quality"],"classic_reference")
        self.assertEqual(db["quests"]["16"]["fields"]["xp"]["value"],0)

if __name__=="__main__":
    unittest.main()
