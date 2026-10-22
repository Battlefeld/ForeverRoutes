import unittest
from foreverroutes.mobdata import (
    parse_npc_references, hp_for_npc, enrich_world, split_lua_fields)
from foreverroutes.core import World

NPCS = '''QuestieDB.npcData = [[return {
[3281] = {'Sarkoth',86,86,4,4,0,{[14]={{40.5,66.83}}},nil,14,nil,nil,14,nil,nil,0},
[3126] = {'Armored Scorpid',137,156,7,8,0,{[14]={{43.37,32.43}}},nil,14,nil,nil,14,nil,nil,0},
[330] = {'Princess',228,228,9,9,0,{[12]={{69.66,79.25}}},nil,12,nil,nil,7,"AH",nil,0},
[345] = {'Bellygrub',1139,1139,24,24,0,nil,nil,44,nil,nil,14,nil,nil,0},
[3098] = {'Mottled Boar',42,55,1,2,0,nil,nil,14,nil,nil,14,nil,nil,0},
[66666] = {'Placeholder',0,1,2,2,0,nil,nil,14,nil,nil,14,nil,nil,0},
[77777] = {'Too High',1000,1200,44,45,0,nil,nil,14,nil,nil,14,nil,nil,0},
}]]'''

class MobHealthTest(unittest.TestCase):
    def setUp(self):
        self.db = parse_npc_references(NPCS)
    def test_quotes_and_nested_fields(self):
        self.assertEqual(split_lua_fields("{'A\\'b,c',{[14]={{40,60}}},42}"),["'A\\'b,c'", "{[14]={{40,60}}}", "42"])
    def test_scope_counts_and_sentinel(self):
        self.assertEqual(self.db['eligible_npc_count'], 6)
        self.assertEqual(self.db['discarded_invalid_hp'], 1)
        self.assertEqual(len(self.db['npcs']), 5)
    def test_exact_health(self):
        self.assertEqual(hp_for_npc(self.db,3281,4)['hp'],86)
        self.assertEqual(hp_for_npc(self.db,330,9)['hp'],228)
        self.assertEqual(hp_for_npc(self.db,345,24)['hp'],1139)
    def test_level_range_endpoints(self):
        self.assertEqual(hp_for_npc(self.db,3126,7)['hp'],137)
        self.assertEqual(hp_for_npc(self.db,3126,8)['hp'],156)
        self.assertEqual(hp_for_npc(self.db,3098,1)['hp'],42)
        self.assertEqual(hp_for_npc(self.db,3098,2)['hp'],55)
        self.assertIsNone(hp_for_npc(self.db,3098,3))
    def test_reference_is_explicit(self):
        self.assertEqual(hp_for_npc(self.db,3281,4)['source'],'questiedb_classic_reference')
        self.assertIsNone(hp_for_npc(self.db,66666,2))
        self.assertIsNone(hp_for_npc(self.db,77777,44))
    def test_overrides_wowhead(self):
        overrides={'330':{'9':{'hp':229,'source':'wowhead_forever','url':'https://www.wowhead.com/forever/npc=330'}}}
        self.assertEqual(hp_for_npc(self.db,330,9,overrides)['hp'],229)
        self.assertEqual(hp_for_npc(self.db,330,9,overrides)['source'],'wowhead_forever')
        self.assertEqual(hp_for_npc(self.db,3126,8,overrides)['hp'],156)
    def test_multiple_sources_prioritize_forever_observation(self):
        overrides={'330':{'9':[
            {'hp':229,'source':'wowhead_forever','url':'https://www.wowhead.com/forever/npc=330'},
            {'hp':231,'source':'foreverdb_observed','url':'https://foreverdb.net/npc/330'}]}}
        choice=hp_for_npc(self.db,330,9,overrides)
        self.assertEqual(choice['hp'],231)
        self.assertEqual(choice['source'],'foreverdb_observed')
    def test_invalid_override_rejected(self):
        x={'330':{'9':{'hp':0,'source':'wowhead_forever','url':'https://wowhead.com'}}}
        with self.assertRaises(ValueError):hp_for_npc(self.db,330,9,x)
    def test_existing_hp_never_overwritten(self):
        doc={'mobs':{'npc:330':{'hp':225,'xp':35,'level':9},'npc:3126':{'hp':None,'xp':28,'level':8}}}
        result,report=enrich_world(doc,self.db)
        self.assertEqual(result['mobs']['npc:330']['hp'],225)
        self.assertEqual(result['mobs']['npc:3126']['hp'],156)
        self.assertEqual(result['mobs']['npc:3126']['xp'],28)
        self.assertEqual(result['mob_provenance']['npc:3126']['source'],'questiedb_classic_reference')
        self.assertEqual(report['imported'],1)
        self.assertEqual(report['preserved_existing'],1)
        self.assertIsNone(doc['mobs']['npc:3126']['hp'])
    def test_unresolved_no_guess(self):
        doc={'mobs':{'npc:99999':{'hp':None,'xp':0,'level':20}}}
        result,report=enrich_world(doc,self.db)
        self.assertIsNone(result['mobs']['npc:99999']['hp'])
        self.assertEqual(len(report['unresolved']),1)
    def test_world_integration(self):
        world={'nodes':['a'],'edges':[], 'level_xp':{'1':100},'dps':5,'quests':[],
               'mobs':{'npc:3281':{'hp':None,'xp':20,'level':4}},'farms':[{'at':'a','mob':'npc:3281'}]}
        doc,report=enrich_world(world,self.db)
        assert World(doc).mobs['npc:3281'].hp==86
        self.assertEqual(report['imported'],1)
    def test_fixed_rank_field(self):
        self.assertEqual(self.db['npcs']['3281']['rank'],0)

if __name__=='__main__':unittest.main()