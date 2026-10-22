import unittest

from foreverroutes.addon_sources import build, decode_points, decode_targets, section


AVAILABLE = """local _, ns = ...
ns.CLASSIC_QUEST_AVAILABLE = {
start = {
    [42]={[1429]={1231145003000}},
    [77]={[1431]={2231199004000}},
},
gates = {
    [42]=123000000000,
},
names = {
    [42]="A New Forever Quest",
    [77]="Other Quest",
},
pre = {
    [42]={7,8},
},
preAll = {
    [77]={20,21},
},
}
"""
TARGETS = """local _, ns = ...
ns.CLASSIC_QUEST_NPCS = {
    [42]={10000123,20000456,30000789},
    [77]={10000999},
}
"""
TURNINS = """local _, ns = ...
ns.CLASSIC_QUEST_TURNIN = {
    [42]={[1429]={1251260007000}},
    [77]={[1431]={2251280009000}},
}
"""


class AddonSourceTests(unittest.TestCase):
    def test_decode_points_without_fabricating_ids(self):
        r=decode_points({1429:[1231145003000]})
        self.assertEqual(r[0]["entity_id"], 1231)
        self.assertEqual(r[0]["entity_kind"], "npc")
        self.assertAlmostEqual(r[0]["x"], 45.00)
        self.assertAlmostEqual(r[0]["y"], 30.0)

    def test_decode_targets_keeps_type_and_index(self):
        d=decode_targets([10000123,20000456,30000789])
        self.assertEqual([(x["kind"], x["id"]) for x in d],
                         [("npc",123),("object",456),("loot_source_npc",789)])

    def test_extract_lua_sections_without_execution(self):
        self.assertEqual(section(AVAILABLE,"names")[42],"A New Forever Quest")
        self.assertEqual(section(AVAILABLE,"pre")[42],[7,8])

    def test_addon_data_build(self):
        d=build(AVAILABLE,TARGETS,TURNINS,"a"*40)
        self.assertEqual(d["schema"],"foreverroutes.addon_crosscheck.v1")
        self.assertEqual(len(d["quests"]),2)
        self.assertEqual(d["quests"]["42"]["prereq_any"],[7,8])
        self.assertEqual(d["quests"]["42"]["name"],"A New Forever Quest")


if __name__=="__main__":
    unittest.main()
