import unittest
from foreverroutes.route_coverage import report


class GuideCoverageTests(unittest.TestCase):
    def test_coverage_is_quest_id_based_and_no_fake_times(self):
        database = {"schema":"foreverroutes.unified_quests.v1","build":"1.60.1.test",
          "quests":{
             "10":{"status":"forever_listed","not_actionable":False,
                    "fields":{"xp":{"value":30}},
                    "objectives":[{"kind":"npc","required_count":None}]},
             "11":{"status":"unresolved_guide_candidate","not_actionable":True},
             "12":{"status":"addon_reference_only","not_actionable":True,"fields":{},"objectives":[]},
          }}
        data=report(database,{"sources":{"rxp":{"Elwynn":[10,10,11,12,999]}}})
        q=data["sources"]["rxp"]["Elwynn"]
        self.assertEqual(q["guide_quest_ids"],4)
        self.assertEqual(q["known_quest_ids"],2)
        self.assertEqual(q["missing_quest_ids"],[11,999])
        self.assertEqual(q["unknown_objective_quantities"],[10])
        self.assertEqual(q["missing_quest_xp"],[12])
        self.assertFalse(q["can_calculate_valid_travel_times"])


if __name__=="__main__":
    unittest.main()
