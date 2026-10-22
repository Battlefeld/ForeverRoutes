import unittest
from foreverroutes.guide_audit import audit_guide

class GuideAuditTests(unittest.TestCase):
    def test_blocks_fake_speed_claim(self):
        catalog={'quests':{'788':{'xp':{'value':170},'started_by':{'npc':[3143]},'finished_by':{'npc':[3143]},'objectives':[{'kind':'npc','ids':[3098],'required_count':None}]}},'npcs':{'3143':{'spawns':{'14':[[42,68]]}}},'objects':{},'items':{}}
        rows=[{'quest_id':788,'condition':'Warlock','optional':False},{'quest_id':97279,'condition':'','optional':True}]
        result=audit_guide(catalog,rows,name='Orc starting')
        self.assertEqual(result['missing_quest_ids'],[97279])
        self.assertEqual(result['unknown_objective_quantity_quest_ids'],[788])
        self.assertEqual(result['unresolved_conditional_actions'],1)
        self.assertFalse(result['can_claim_fastest_route'])
    def test_missing_item_drop_is_reported(self):
        catalog={'quests':{'789':{'xp':{'value':250},'started_by':{},'finished_by':{},'objectives':[{'kind':'item','ids':[4862],'required_count':None}]}},'items':{'4862':{'npc_sources':[3124],'drop_chance_by_npc':{}}},'npcs':{},'objects':{}}
        result=audit_guide(catalog,[{'quest_id':789,'condition':'','optional':False}])
        self.assertEqual(result['item_without_usable_drop_rate'],[{'quest':789,'item_id':4862}])
        self.assertEqual(result['missing_starter_ids'],[789])