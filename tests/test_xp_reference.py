import unittest
from foreverroutes.xp_reference import XP_TO_NEXT,base_kill_xp,gray_level,zero_difference,xp_to_level_30

class XPFallbackTests(unittest.TestCase):
    def test_table(self):
        self.assertEqual(XP_TO_NEXT[1],400)
        self.assertEqual(XP_TO_NEXT[29],44300)
        self.assertEqual(XP_TO_NEXT[30],47400)
        self.assertEqual(xp_to_level_30(),499000)
    def test_same_level_and_higher(self):
        self.assertEqual(base_kill_xp(10,10),95)
        self.assertEqual(base_kill_xp(10,14),114)
        self.assertEqual(base_kill_xp(10,100),114)
    def test_lower_and_gray(self):
        self.assertEqual(gray_level(30),22)
        self.assertEqual(base_kill_xp(30,22),0)
        self.assertEqual(base_kill_xp(10,8),round(95*(1-2/7)))
    def test_elite_and_no_xp(self):
        self.assertEqual(base_kill_xp(10,10,elite=True),238)
        self.assertEqual(base_kill_xp(10,10,no_xp=True),0)
    def test_bad(self):
        with self.assertRaises(ValueError): base_kill_xp(0,1)