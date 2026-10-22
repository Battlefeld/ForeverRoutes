import unittest

from foreverroutes.unified import SCHEMA, merge, audit


def qbase():
    return {"schema": "foreverroutes.catalog.1",
       "sources": {"quests": {"revision": "a" * 40}},
       "quests": {
           "42": {"id":42, "name":"Classic Name", "required_level":2, "quest_level":4,
                  "started_by":{"npc":[100], "object":[], "item":[]},
                  "finished_by":{"npc":[101], "object":[]},
                  "prequest_all":[7], "prequest_any":[],
                  "objectives":[{"kind":"npc", "ids":[200], "required_count":None}],
                  "xp":{"value":250, "source":"classic_quest_xp"}},
           "99":{"id":99,"candidate_only":True,"source_guides":["rxp"],"objectives":[]},
           "77":{"id":77,"name":"Classic Only","required_level":5, "quest_level":8,
                 "objectives":[], "started_by":{"npc":[22]}, "finished_by":{"npc":[23]}}},
       "npcs":{"200":{"hp_min":50,"hp_max":60, "spawns":{}}},
       "items":{}, "objects":{}, "level_xp_classic":{"1":400}}
def fbase():
    return {"schema":"foreverroutes.foreverdb_snapshot.v1",
      "source":{"label":"Data from ForeverDB.net","url":"https://foreverdb.net"},
      "game_metadata":{"build":"1.60.1.test"},
      "quests":[
          {"id":42,"name":"Forever Name","level":5,"requiredLevel":3,"source":"forever",
           "knownInForever":True,"starts":{"type":"npc","id":100,
             "location":{"uiMapId":1429,"x":40,"y":60,"source":"seen in WoW Forever"}},
           "ends":{"type":"npc","id":101}},
          {"id":99,"name":"Forever Quest","level":6,"requiredLevel":4,"source":"forever",
           "starts":{"type":"npc","id":301},"ends":{"type":"npc","id":302}},
          {"id":111,"name":"Classic API only","level":9,"requiredLevel":7,"source":"classic",
           "starts":{"type":"npc","id":501},"ends":{"type":"npc","id":502}}
      ],"quest_details":{"42":{"data":{"id":42}}},
      "quest_changes_seen":[]}


class UnifiedTests(unittest.TestCase):
    def test_union_and_precedence(self):
        r=merge(qbase(),fbase())
        self.assertEqual(r["schema"],SCHEMA)
        self.assertEqual(len(r["quests"]),4)
        self.assertEqual(r["quests"]["42"]["fields"]["name"]["value"],"Forever Name")
        self.assertEqual(r["quests"]["42"]["fields"]["xp"]["value"],250)
        self.assertEqual(r["quests"]["42"]["fields"]["xp"]["quality"],"classic_reference")
        self.assertEqual(r["quests"]["42"]["fields"]["prequest_all"]["value"],[7])
        self.assertEqual(r["quests"]["42"]["objectives"][0]["required_count"],None)
        self.assertEqual(r["quests"]["42"]["locations"]["starts"]["quality"],"forever_observed")
        self.assertEqual(r["quests"]["99"]["fields"]["name"]["value"],"Forever Quest")
        self.assertEqual(r["quests"]["77"]["status"],"questie_reference_only")
        self.assertEqual(r["quests"]["111"]["fields"]["name"]["quality"],"classic_reference")
        self.assertEqual(r["stats"]["identified_quests"],4)
        self.assertGreater(r["stats"]["conflicts"],0)
        self.assertFalse(r["quests"]["42"]["not_actionable"])
        self.assertEqual(audit(r)["schema"],"foreverroutes.unified_audit.v1")

    def test_unresolved_candidate_is_not_actionable(self):
        src=fbase()
        src["quests"]=[]
        r=merge(qbase(),src)
        self.assertTrue(r["quests"]["99"]["not_actionable"])
        self.assertEqual(r["stats"]["unresolved_guide_candidates"],1)

    def test_classic_foreverdb_does_not_override_questie(self):
        src=fbase()
        src["quests"][0]["source"]="classic"
        r=merge(qbase(),src)
        self.assertEqual(r["quests"]["42"]["fields"]["name"]["value"],"Classic Name")

    def test_reject_malformed_duplicate(self):
        s=fbase()
        s["quests"].append(s["quests"][0])
        with self.assertRaises(ValueError):
            merge(qbase(),s)

    def test_reject_fabricated_evidence(self):
        r=merge(qbase(),fbase(),{"observations":[{"quest_id":42,"fields":{
           "xp_by_player_level":{"value":{"5":8000},"source":"made_up","url":"https://bad.test"}}}]})
        self.assertNotIn("xp_observed_by_player_level",r["quests"]["42"]["fields"])

    def test_explicit_source_observations_win(self):
        r=merge(qbase(),fbase(),{"observations":[{"quest_id":42,"fields":{
           "xp_by_player_level":{"value":{"5":380},"source":"foreverdb_observed","url":"https://foreverdb.net/quest/42"}}}]})
        self.assertEqual(r["quests"]["42"]["fields"]["xp_observed_by_player_level"]["value"]["5"],380)



    def test_wowhead_0_xp_counts_as_resolved_and_40_xp_forever(self):
        reference={"schema":"foreverroutes.quest_xp_overrides.v1","observations":[
            {"quest_id":77,"fields":{"xp":{"value":0,"source":"wowhead_classic",
                "quality":"classic_reference",
                "url":"https://www.wowhead.com/classic/quest=77"}}},
            {"quest_id":99,"fields":{"xp":{"value":40,"source":"wowhead_forever",
                "quality":"forever_reference",
                "url":"https://www.wowhead.com/forever/quest=99"}}}
        ]}
        result=merge(qbase(),fbase(),xp_reference=reference)
        self.assertEqual(result["quests"]["77"]["fields"]["xp"]["value"],0)
        self.assertEqual(result["quests"]["99"]["fields"]["xp"]["value"],40)
        self.assertEqual(result["stats"]["zero_xp_quests"],1)
        self.assertEqual(result["stats"]["wowhead_xp_reference_quests"],2)
        missing=audit(result)["missing"]["quests_without_xp"]
        self.assertNotIn(77,missing)
        self.assertNotIn(99,missing)
        self.assertEqual(result["quests"]["77"]["fields"]["xp"]["quality"],"classic_reference")
        self.assertEqual(result["quests"]["99"]["fields"]["xp"]["quality"],"forever_reference")

    def test_wowhead_references_cannot_inject_unsourced_xp(self):
        entry={"quest_id":77,"fields":{"xp":{"value":1,"source":"wowhead_classic",
                "quality":"classic_reference","url":"https://wrong.example/quest=77"}}}
        with self.assertRaises(ValueError):
            merge(qbase(),fbase(),xp_reference={"schema":"foreverroutes.quest_xp_overrides.v1",
                                           "observations":[entry]})

    def test_wowhead_reference_overrides_classic_xp_and_reports_conflict(self):
        reference={"schema":"foreverroutes.quest_xp_overrides.v1","observations":[
            {"quest_id":42,"fields":{"xp":{"value":0,"source":"wowhead_classic",
                "quality":"classic_reference",
                "url":"https://www.wowhead.com/classic/quest=42"}}}
        ]}
        result=merge(qbase(),fbase(),xp_reference=reference)
        self.assertEqual(result["quests"]["42"]["fields"]["xp"]["value"],0)
        self.assertTrue(any(x["field"]=="xp" and x["quest_id"]==42 for x in result["field_conflicts"]))


if __name__=="__main__":
    unittest.main()
