import copy,json, unittest
from pathlib import Path
from foreverroutes import World,Action,RouteError, WorldError,evaluate,optimize
from foreverroutes.core import initial,action_candidates,apply,level

ROOT=Path(__file__).resolve().parents[1]
WORLD=json.loads((ROOT/'examples/synthetic_tiny.json').read_text())

class RouteTests(unittest.TestCase):
    def setUp(self):self.world=World(copy.deepcopy(WORLD))

    def test_solver_prefers_nearby_shared_kills(self):
        out=optimize(self.world,'Spawn','Orc','Warlock','Horde',2)
        self.assertTrue(out['optimal_in_model']);self.assertTrue(out['reached_target'])
        self.assertEqual(set(out['completed_quests']),{101,102})
        self.assertEqual(sum(s['op']=='work' for s in out['steps']),2)
        self.assertLess(out['seconds'],300)
        self.assertNotIn(103,out['completed_quests'])

    def test_quest_dependency_requires_turnin(self):
        s=initial(self.world,'Spawn')
        with self.assertRaises(RouteError): apply(self.world,s,Action('accept',104,at='Hub'),'Orc','Warlock','Horde')

    def test_xp_thresholds_are_cumulative(self):
        self.assertEqual(level(self.world,99),1)
        self.assertEqual(level(self.world,100),2)
        self.assertEqual(level(self.world,199),2)
        self.assertEqual(level(self.world,200),3)

    def test_race_filter(self):
        doc=copy.deepcopy(WORLD)
        doc['quests'][0]['races']=['Troll']
        world=World(doc)
        self.assertNotIn(Action('accept',101,at='Hub'), set(action_candidates(world,initial(world,'Spawn'),'Orc','Warlock','Horde')))

    def test_faction_class_filter(self):
        doc=copy.deepcopy(WORLD)
        doc['quests'][0]['classes']=['Hunter'];doc['quests'][1]['faction']='Alliance'
        world=World(doc);state=initial(world,'Spawn')
        actions=set(action_candidates(world,state,'Orc','Warlock','Horde'))
        self.assertNotIn(Action('accept',101,at='Hub'),actions)
        self.assertNotIn(Action('accept',102,at='Hub'),actions)

    def test_no_shortcut_across_disconnected_graph(self):
        doc=copy.deepcopy(WORLD)
        doc['quests'][0]['giver']='Blocked';doc['quests'][1]['giver']='Blocked'
        world=World(doc)
        self.assertEqual(world.travel('Hub','Blocked'),float('inf'))
        with self.assertRaises(RouteError):apply(world,initial(world,'Spawn'),Action('accept',101,at='Blocked'),'Orc','Warlock','Horde')

    def test_questlog_limit(self):
        doc=copy.deepcopy(WORLD);doc['max_open']=1
        world=World(doc);s=initial(world,'Spawn')
        s,_=apply(world,s,Action('accept',101,at='Hub'),'Orc','Warlock','Horde')
        with self.assertRaises(RouteError):apply(world,s,Action('accept',102,at='Hub'),'Orc','Warlock','Horde')

    def test_invalid_turnin_rejected(self):
        with self.assertRaises(RouteError):apply(self.world,initial(self.world,'Spawn'),Action('turnin',101,at='Hub'),'Orc','Warlock','Horde')

    def test_duplicate_quests_rejected(self):
        doc=copy.deepcopy(WORLD);doc['quests'].append(copy.deepcopy(doc['quests'][0]))
        with self.assertRaises(WorldError): World(doc)

    def test_replay_same_cost(self):
        out=optimize(self.world,'Spawn','Orc','Warlock','Horde',2)
        replay=evaluate(self.world,[{k:v for k,v in s.items() if k in ('op','quest','obj','at','target')}
                              for s in out['steps']],'Spawn','Orc','Warlock','Horde',2)
        self.assertAlmostEqual(out['seconds'],replay['seconds'])

    def test_solver_limit_not_claimed_optimal(self):
        out=optimize(self.world,'Spawn','Orc','Warlock','Horde',2,max_states=1)
        self.assertFalse(out['optimal_in_model']);self.assertEqual(out['reason'],'state_limit')

    def test_missing_mob_rejected(self):
        doc=copy.deepcopy(WORLD);doc['quests'][0]['objectives'][0]['target']='invented'
        with self.assertRaises(WorldError):World(doc)

if __name__=='__main__':unittest.main()