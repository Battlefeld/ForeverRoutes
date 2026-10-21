import json
import unittest
from copy import deepcopy
from pathlib import Path
from foreverroutes import World, Action, evaluate, RouteError
from foreverroutes.audit import audit
from foreverroutes.planner import Character, search, _quest_actions
from examples.generate_synthetic_30 import build


class LevelThirtyTests(unittest.TestCase):
    def test_reaches_30_realistic_size_synthetic_only(self):
        doc=build()
        w=World(doc)
        a=search(w,Character(),restarts=3,seed=7,max_decisions=150)
        self.assertTrue(a['reached_target'])
        self.assertEqual(a['level'],30)
        self.assertFalse(a['optimal_in_model'])
        self.assertTrue(a['steps'])
        self.assertAlmostEqual(a['seconds'], a['travel']+a['combat']+a['interaction'],places=6)
        self.assertEqual(a['seconds'], search(w,Character(),restarts=3,seed=7,max_decisions=150)['seconds'])
        self.assertTrue(all(t['reached_target'] for t in a['trials']))

    def test_source_audit_never_calls_estimates_game_measured(self):
        a=audit(World(build()),target_level=30,start='Spawn')
        self.assertTrue(a['synthetic'])
        self.assertEqual(a['quests'],58)
        self.assertEqual(a['missing_xp_levels'],[])
        self.assertFalse(a['model_is_game_calibrated'])

    def test_no_level_30_without_xp_table(self):
        d=build();d['level_xp'].pop('17')
        with self.assertRaisesRegex(ValueError,'XP thresholds missing'):
            search(World(d),Character(),restarts=1)

    def test_high_level_grind_forbidden(self):
        w=World(build()); s=w.quests[0]
        with self.assertRaises(RouteError):
            evaluate(w,[Action('grind',at='L29_Field',target='mob_L29')],'Spawn','Orc','Warlock','Horde',2)

    def test_related_quests_share_kills(self):
        w=World(build());state=__import__('foreverroutes.core',fromlist=['initial']).initial(w,'Spawn')
        c=_quest_actions(w,state,101,Character())
        self.assertIsNotNone(c)
        # Two parallel quests to kill the same creatures are collected in one trip.
        self.assertIn(102,[a.quest for a in c.actions if a.op=='accept'])
        self.assertGreater(c.xp,0)

    def test_zero_xp_breadcrumb_can_be_selected(self):
        d=build()
        q=d['quests'][0];q['xp']=0
        d['quests'][1]['prereqs']=[101]
        d['mobs']['mob_L1']['xp']=0
        w=World(d)
        c=_quest_actions(w,__import__('foreverroutes.core',fromlist=['initial']).initial(w,'Spawn'),101,Character())
        self.assertEqual(c.xp,0)
        self.assertGreater(c.value,0)

if __name__=='__main__':unittest.main()