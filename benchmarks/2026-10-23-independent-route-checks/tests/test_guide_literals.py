import unittest
from foreverroutes.guide_literals import read_literal,GuideSyntaxError,children

class GuideLiteralTests(unittest.TestCase):
    def test_nested_conditions_and_objective_indices_survive(self):
        d,_=read_literal('''{-- comment\n id="s1", condition="WARLOCK", {condition="TBC", {complete=9431, objective=2}}, {accept=788}}''')
        self.assertEqual(children(d)[0]['condition'],'TBC')
        self.assertEqual(children(children(d)[0])[0]['objective'],2)
    def test_mixed_metadata_and_long_text_are_literals(self):
        d,_=read_literal('{guideVersion="1",{id="g",text=[=[a -- literal]=], optional=false}}')
        self.assertEqual(children(d)[0]['text'],'a -- literal')
        self.assertIs(children(d)[0]['optional'],False)
    def test_code_is_never_executed(self):
        for s in ['os.execute("bad")','{x=loadstring("bad")}','{}; os.execute("bad")']:
            with self.assertRaises(GuideSyntaxError):read_literal(s)

if __name__=='__main__':unittest.main()
