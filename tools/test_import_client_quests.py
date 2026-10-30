import copy
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location('importer', Path(__file__).with_name('import_client_quests.py'))
m = importlib.util.module_from_spec(spec); spec.loader.exec_module(m)


def capture(title='Quest Ä', seen=100):
    return {'schema':1, 'captures':{'1.60.1.70685/3/enUS':{'build':'1.60.1.70685','region':3,'locale':'enUS','quests':{'971':{'id':971,'title':title,'seenAt':seen}},'turnins':{'1':{'id':971,'seenAt':100,'observedXP':150}}}}}


class ImportTests(unittest.TestCase):
    def test_savedvariables_is_read_as_data_never_executed(self):
        with tempfile.TemporaryDirectory() as folder:
            p = Path(folder)/'ForeverRoutes.lua'
            p.write_text('error("must never execute")\nForeverRoutesQuestExport = ' + json.dumps(json.dumps(capture(), ensure_ascii=False), ensure_ascii=False))
            self.assertEqual(m.load(p), capture())

    def test_idempotent_merge_and_newer_same_build_observation(self):
        first = capture()
        self.assertEqual(m.merge(first,first),first)
        updated=m.merge(first,capture('New',101))
        self.assertEqual(updated['captures']['1.60.1.70685/3/enUS']['quests']['971']['title'],'New')
        with self.assertRaises(ValueError): m.merge(first,capture('Conflict',100))

    def test_other_game_and_bad_identity_rejected(self):
        for key,value in [('build','12.0.0.123'),('region',9),('locale','xxXX')]:
            doc=capture();doc['captures']['1.60.1.70685/3/enUS'][key]=value
            with tempfile.TemporaryDirectory() as folder:
                p=Path(folder)/'x.json';p.write_text(json.dumps(doc))
                with self.assertRaises(ValueError):m.load(p)

    def test_lua_decimal_escape_and_truncated_export(self):
        self.assertEqual(m.lua_string('ForeverRoutesQuestExport = "a\\034b\\092c"'),'a"b\\c')
        self.assertEqual(m.lua_string('ForeverRoutesQuestExport = "\\195\\164"'),'ä')
        with self.assertRaises(ValueError):m.lua_string('ForeverRoutesQuestExport = "broken')

if __name__ == '__main__': unittest.main()
