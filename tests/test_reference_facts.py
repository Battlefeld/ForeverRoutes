import unittest
from foreverroutes.reference_facts import (
    parse_classic_npcs, parse_quests, merge, quest_xp, npc_hp, fields
)
QUEST='[788] = {"Cutting Teeth",{{3143}},{{3143}},1,2,8589934770,nil,{"Kill boars"},nil,{{{3098}}},nil,nil,nil,nil},'
XP='[788] = {2, 170}, -- Classic'
NPC="[3098] = {'Mottled Boar',42,55,1,2,0,{[14]={{45.16,70.47}}},nil,14,nil,nil,189,nil,nil,0},"
URL='https://foreverdb.net/quest/788'

class RealReferenceTests(unittest.TestCase):
    def test_split_lua_single_double_nested(self):
        self.assertEqual(fields('{"a,b",{{1,2}},\'c,d\'}'), ['"a,b"','{{1,2}}',"'c,d'"])
    def test_real_classic_hp(self):
        mobs=parse_classic_npcs(NPC)
        self.assertEqual(npc_hp(mobs['3098'],1)['value'],42)
        self.assertEqual(npc_hp(mobs['3098'],2)['value'],55)
        self.assertIsNone(npc_hp(mobs['3098'],3))
        self.assertEqual(npc_hp(mobs['3098'],1)['quality'],'classic_reference')
    def test_compatibility_placeholder_never_used(self):
        self.assertFalse(parse_classic_npcs("[44] = {'Dummy',0,1,1,2,0,nil,nil,14,nil,nil,nil,nil,nil,0},"))
    def test_classic_quest_xp_used(self):
        quests=parse_quests(QUEST,XP)
        self.assertEqual(quest_xp(quests['788'],2)['value'],170)
        self.assertEqual(quest_xp(quests['788'],2)['quality'],'classic_reference')
    def test_new_forever_quest_and_observed_priority(self):
        quests=parse_quests(QUEST,XP)
        merge(quests,[{"quest_id":97279,"fields":{"xp_by_player_level":{"value":{"2":170},"source":"foreverdb_observed","url":URL}}}])
        self.assertEqual(quest_xp(quests['97279'],2)['value'],170)
        self.assertEqual(quest_xp(quests['97279'],2)['quality'],'forever_observed')
        self.assertIsNone(quest_xp(quests['97279'],3))
    def test_observed_hp_beats_classic_reference(self):
        mobs=parse_classic_npcs(NPC)
        merge(mobs,[{"npc_id":3098,"fields":{"hp_at_level_2":{"value":60,"source":"foreverdb_observed","url":URL}}}],identity="npc_id")
        self.assertEqual(npc_hp(mobs['3098'],2)['value'],60)
        self.assertEqual(npc_hp(mobs['3098'],1)['value'],42)
    def test_conflict_preserved_and_source_ranking(self):
        quests=parse_quests(QUEST,XP)
        differences=merge(quests,[{"quest_id":788,"fields":{"quest_level":{"value":4,"source":"wowhead_forever","url":URL}}}])
        self.assertEqual(len(differences),1)
        self.assertEqual(quests['788']['fields']['quest_level']['value'],4)
        merge(quests,[{"quest_id":788,"fields":{"quest_level":{"value":2,"source":"foreverdb_classic","url":URL}}}])
        self.assertEqual(quests['788']['fields']['quest_level']['value'],4)

if __name__=="__main__": unittest.main()
