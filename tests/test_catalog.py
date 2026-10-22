import unittest
from foreverroutes.catalog import parse_lua,ParseError,build_catalog,drop_chances

BASE={
'quests':'''[788] = {"Cutting Teeth",{{3143}},{{3143}},1,2,8589934770,nil,{"Kill 10."},nil,{{{3098}}},nil,nil,nil,nil,nil,363},
[789] = {"Sting of the Scorpid",{{3143}},{{3143}},1,3,8589934770,nil,{"Get 10 tails"},nil,{nil,nil,{{4862}}},nil,nil,{788},nil,nil,363},''',
'quest_xp':'[788] = {2, 170}, -- Classic\n[789] = {3, 250}, -- Classic',
'npc_positions':'''[3098] = {'Mottled Boar',42,55,1,2,0,{[14]={{45.16,70.47}}},nil,14,nil,nil,189,nil,nil,0},
[3143] = {'Gornek',71,71,1,1,0,{[14]={{42,68}}},nil,14,nil,nil,189,"H",nil,0},
[3124] = {'Scorpid Worker',71,71,3,3,0,{[14]={{40.91,62.03}}},nil,14,nil,nil,189,nil,nil,0},''',
'npc_health':'''[3098] = {'Mottled Boar',42,55,1,2,0,{[14]={{45.16,70.47}}},nil,14,nil,nil,189,nil,nil,0},
[3143] = {'Gornek',71,71,1,1,0,nil,nil,14,nil,nil,189,"H",nil,0},
[3124] = {'Scorpid Worker',71,71,3,3,0,nil,nil,14,nil,nil,189,nil,nil,0},''',
'items':"[4862] = {'Scorpid Worker Tail',{3124},nil,nil,nil,nil,nil,nil,1,0,0,12,0},",
'objects':"[171938] = {'Cactus',nil,nil,{[14]={{42,60}}},14},",
'item_drop_rates':'''QuestieClassicItemDrops.wowheadData = [[return {
    [4862] = { -- Scorpid Worker Tail
        [3124] = 59.7968,
    },
}]]'''
}

class CatalogTests(unittest.TestCase):
    def test_lua_safe(self):
        self.assertEqual(parse_lua("{nil,{[14]={{45.16,70.47}}},'A\\'b'}"),[None,{14:[[45.16,70.47]]},"A'b"])
        with self.assertRaises(ParseError):parse_lua('{os.execute("hello")}')
        with self.assertRaises(ParseError):parse_lua('{1,2')
    def test_drop_rates(self):
        self.assertEqual(drop_chances(BASE['item_drop_rates'])[4862][3124],.597968)
    def test_catalog(self):
        c=build_catalog(BASE)
        self.assertEqual(c['quests']['788']['required_level'],1)
        self.assertEqual(c['quests']['788']['xp']['value'],170)
        self.assertEqual(c['quests']['789']['prequest_any'],[788])
        self.assertEqual(c['quests']['788']['objectives'][0]['ids'],[3098])
        self.assertIsNone(c['quests']['788']['objectives'][0]['required_count'])
        self.assertEqual(c['npcs']['3098']['hp_max'],55)
        self.assertEqual(c['npcs']['3098']['spawns']['14'],[[45.16,70.47]])
        self.assertEqual(c['items']['4862']['npc_sources'],[3124])
        self.assertEqual(c['items']['4862']['drop_chance_by_npc']['3124']['chance'],0.597968)
        self.assertFalse(c['metadata']['quest_counts_unknown'] is False)
    def test_observations(self):
        q=[{'quest_id':97279,'fields':{'quest_level':{'value':2,'url':'https://foreverdb.net/quest/97279','source':'foreverdb_client'},'xp_by_player_level':{'value':{'2':170},'url':'https://foreverdb.net/quest/97279','source':'foreverdb_observed'}}}]
        n=[{'npc_id':3098,'fields':{'hp_at_level_2':{'value':58,'url':'https://foreverdb.net/npc/3098','source':'foreverdb_observed'}}}]
        c=build_catalog(BASE,quest_overlays=q,npc_overlays=n)
        self.assertEqual(c['quests']['97279']['xp_observed_by_player_level']['2'],170)
        self.assertEqual(c['npcs']['3098']['hp_overrides']['2']['value'],58)

if __name__=='__main__': unittest.main()