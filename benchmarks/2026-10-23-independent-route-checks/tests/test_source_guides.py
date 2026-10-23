import unittest,tempfile
from pathlib import Path
from foreverroutes.guide_literals import read_literal
from foreverroutes.source_guides import journey_body,load_rxp

class SourceGuideTests(unittest.TestCase):
    def test_journey_preserves_nested_version_guard_and_objective(self):
        root,_=read_literal('{ {id="s1",condition="TBC or WOTLK",{condition="HUNTER",{complete=9431,objective=2}}}}')
        groups=journey_body(root,guide='g',path='x.html')
        event=groups[0]['events'][0]
        self.assertEqual(event['conditions'],['TBC or WOTLK','HUNTER'])
        self.assertEqual(event['obj'],2)
    def test_journey_preserves_optional_and_level_grinding(self):
        root,_=read_literal('{{id="s",optional=true,{accept=123},{level=6,xp=200}}}')
        es=journey_body(root,guide='g',path='x.html')[0]['events']
        self.assertTrue(all(e['optional'] for e in es));self.assertEqual(es[1]['xp'],200)
    def test_journey_does_not_sort_nested_movement_before_quest_actions(self):
        root,_=read_literal('{{id="original",{accept=1},{locations={{"1|2|3"}}},{turnin=1}}}')
        es=journey_body(root,guide='independent',path='source.html')[0]['events']
        self.assertEqual([e['t'] for e in es],['accept','move','turnin'])
    def test_rxp_does_not_duplicate_translations_or_drop_objective_scopes(self):
        with tempfile.TemporaryDirectory() as d:
            p=Path(d)/'Guides/forever';p.mkdir(parents=True)
            s='RXPGuides.RegisterGuide([[\n#name 1-6 Durotar\n#next 6-10 Durotar << Warlock\nstep << Warlock\n.complete 788,2 << Orc --|6/6 Boars\n.hs\n.turnin -788,1\n]])'
            (p/'Horde-01-12_Durotar.lua').write_text(s)
            t=Path(d)/'lang/Guides-deDE/forever';t.mkdir(parents=True);(t/'Horde-01-12_Durotar.lua').write_text(s)
            guides=load_rxp(d)
        self.assertEqual(len(guides),1)
        e=guides[0]['groups'][0]['events'][0]
        self.assertEqual(e['obj'],2);self.assertEqual(e['conditions'],['Orc'])
        self.assertEqual(guides[0]['groups'][0]['events'][1]['t'],'hearth')
        turnin=guides[0]['groups'][0]['events'][2]
        self.assertEqual(turnin['q'],788);self.assertNotIn('obj',turnin)
        self.assertTrue(turnin['skipIfMissing']);self.assertTrue(turnin['skipIfIncomplete'])

if __name__=='__main__':unittest.main()
