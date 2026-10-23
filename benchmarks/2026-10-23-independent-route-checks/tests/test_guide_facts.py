import unittest,tempfile
from pathlib import Path
from foreverroutes.guide_facts import numbered_goal_counts,enrich

class GuideFactTests(unittest.TestCase):
    def test_numbered_goals_fill_counts_without_treating_objective_index_as_quantity(self):
        with tempfile.TemporaryDirectory() as d:
            p=Path(d)/'Guides/forever';p.mkdir(parents=True)
            (p/'Horde-01-12_Durotar.lua').write_text('.complete 100,2 --|4/4 Pieces\n.complete 101,3 --Collect eggs\n')
            refs,conflicts=numbered_goal_counts(d)
        self.assertEqual(refs['100']['2']['count'],4);self.assertNotIn('101',refs)
        qs={'100':{'objectives':[{'count':7},{'count':None}]}}
        filled,_=enrich(qs,refs)
        self.assertEqual(qs['100']['objectives'][1]['count'],4)
        self.assertEqual(qs['100']['objectives'][0]['count'],7)
        self.assertEqual(qs['100']['objectives'][1]['countProvenance']['evidence'][0]['quality'],'guide_reference')
    def test_conflicting_guides_do_not_choose_an_arbitrary_count(self):
        with tempfile.TemporaryDirectory() as d:
            p=Path(d)/'forever';p.mkdir()
            (p/'Horde-01-12_Durotar.lua').write_text('.complete 100,1 --|2/2 Pieces\n.complete 100,1 --|3/3 Pieces\n')
            refs,conflicts=numbered_goal_counts(d)
        self.assertEqual(refs,{});self.assertEqual(len(conflicts),1)

if __name__=='__main__':unittest.main()
