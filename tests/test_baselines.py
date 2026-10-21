import tempfile,unittest
from pathlib import Path
from foreverroutes.baselines import parse_rows, coverage

class BaselineTests(unittest.TestCase):
    def rows(self,text,source):
        with tempfile.TemporaryDirectory() as t:
            f=Path(t)/'ex.csv';f.write_text(text,encoding='utf8')
            return list(parse_rows(f,source))

    def test_wisp(self):
        rows=self.rows('Guide-ID;Schritt-ID;Aktion;Quest-ID;Bedingungen\nhello;1;Annehmen;788;\nhello;2;Questziel;788;class=Warlock\nhello;3;Abgeben;788;\n', 'wisp')
        self.assertEqual([r['action'] for r in rows],['accept','complete','turnin'])
        self.assertTrue(coverage(rows)['unresolved_conditions'])

    def test_polestar_aoe_excluded(self):
        rows=self.rows('Guide-ID;Schritt;Aktion;Quest-ID;Aktion-Bedingung;Optional\nroute;1;accept;788;;0\nroute-AoE;2;accept;999;;0\nroute;3;turnin;788;;1\n','polestar')
        self.assertEqual(len(rows),2)
        self.assertEqual(coverage(rows)['optional_actions'],1)

    def test_journey_conditions_preserved(self):
        rows=self.rows('guide_ids;schritt;aktion;quest_id;schritt_bedingung\nA1;1;accept;1;FOREVER and Warlock\nA1;2;complete;1;\n','journey')
        self.assertEqual(rows[0]['condition'],'FOREVER and Warlock')
        self.assertEqual(coverage(rows)['unique_quest_ids'],1)

    def test_unknown_source_rejected(self):
        with self.assertRaises(ValueError):list(parse_rows('missing.csv','unknown'))

if __name__=='__main__': unittest.main()