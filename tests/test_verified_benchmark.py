import tempfile
import unittest
from pathlib import Path
from foreverroutes.verified_benchmark import FactBook, Fact, EvidenceError, inspect_candidate, compare_candidates

class VerifiedBenchmarkTests(unittest.TestCase):
    def test_classic_seed_not_forever(self):
        book = FactBook([Fact("quest_xp", "788", 170, "classic_era_seeded", "QuestieDB", "commit")])
        self.assertFalse(book.get("quest_xp", 788).is_forever_verified())

    def test_client_does_not_prove_server_xp(self):
        self.assertFalse(Fact("quest_xp", "788", 170, "forever_client_direct", "DB2", "build").is_forever_verified())

    def test_server_observation_requires_date_and_sample(self):
        self.assertFalse(Fact("quest_xp", "788", 170, "forever_server_observed", "recorder", "build").is_forever_verified())
        self.assertTrue(Fact("quest_xp", "788", 170, "forever_server_observed", "recorder", "build", "2026-10-22", 3).is_forever_verified())

    def test_conflicts_do_not_get_averaged(self):
        f = Fact("quest_xp", "788", 170, "classic_era_seeded", "x", "y")
        with self.assertRaises(EvidenceError):
            FactBook([f, f])

    def test_missing_data_cannot_yield_fake_winner(self):
        with tempfile.TemporaryDirectory() as d:
            file = Path(d)/"route.csv"
            file.write_text("guide_ids;schritt;aktion;quest_id;schritt_bedingung\nH01D_FVR;1;accept;788;WARLOCK\nH01D_FVR;2;complete;788;\nH01D_FVR;3;turnin;788;\n", encoding="utf8")
            book = FactBook([Fact("quest_xp", "788", 170, "classic_era_seeded", "QuestieDB", "commit")])
            r = inspect_candidate(source="journey", csv_path=file, guide="H01D_FVR", facts=book)
            self.assertIsNone(r["elapsed_seconds"])
            self.assertEqual(r["non_forever_verified_xp"], 1)
            self.assertIn("conditions_not_yet_evaluated_for_character", r["blockers"])
            self.assertIsNone(compare_candidates([r])["winner"])

    def test_identical_world_revision_required(self):
        r = {"status":"VERIFIED_REPLAY","source":"a","elapsed_seconds":100,
             "race":"Orc","class":"Warlock","faction":"Horde","target_level":30,"world_revision":"rev1"}
        with self.assertRaises(EvidenceError):
            compare_candidates([r, {**r,"source":"b","elapsed_seconds":90,"world_revision":"rev2"}])

if __name__ == "__main__":
    unittest.main()
