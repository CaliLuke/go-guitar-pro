"""Check that inventory failures and uncertainty remain visible."""

import sqlite3
from contextlib import closing
import tempfile
from pathlib import Path
import unittest
from unittest.mock import patch

import manage


class CapabilityDatabaseTests(unittest.TestCase):
    def database(self):
        con = sqlite3.connect(":memory:")
        con.executescript((manage.HERE / "schema.sql").read_text())
        self.addCleanup(con.close)
        return con

    def capability(self, con, name, scope="guitar-pro"):
        con.execute("INSERT INTO capability VALUES (?,?,?,?,?,?,?,?)", (
            name, "test", name, scope, 1, '["gp8"]', "Evidence", "Acceptance"))

    def test_progress_excludes_other_products_and_does_not_credit_partial(self):
        con = self.database()
        for name, status, scope in [("complete", "supported", "guitar-pro"),
                                    ("incomplete", "partial", "guitar-pro"),
                                    ("unknown", "unverified", "guitar-pro"),
                                    ("player", "out-of-scope", "excluded")]:
            self.capability(con, name, scope)
            for stage in ("import", "model", "export"):
                con.execute("INSERT INTO assessment VALUES (?,?,?)", (name, stage, status))
        self.assertEqual(con.execute("SELECT capabilities,percent_of_scope FROM progress WHERE stage='import' AND status='supported'").fetchone(), (1, 33.3))
        self.assertEqual({r[0] for r in con.execute("SELECT id FROM gaps")}, {"incomplete", "unknown"})

    def test_dangling_mapping_and_invalid_status_fail(self):
        con = self.database()
        with self.assertRaises(sqlite3.IntegrityError):
            con.execute("INSERT INTO assessment VALUES ('absent','import','supported')")
        self.capability(con, "present")
        with self.assertRaises(sqlite3.IntegrityError):
            con.execute("INSERT INTO assessment VALUES ('present','import','probably')")

    def test_new_upstream_construct_remains_visible_without_a_mapping(self):
        con = self.database()
        con.execute("INSERT INTO upstream_construct(id,path,line,kind,name,declaration) VALUES ('new','Note.ts',1,'field','Note.newEffect','public newEffect = true;')")
        self.assertEqual(con.execute("SELECT name,review_status FROM unreviewed_constructs").fetchall(), [("Note.newEffect", "unverified")])

    def test_failed_target_consumer_is_distinct_from_a_difference(self):
        con = self.database()
        self.capability(con, "fermata")
        con.execute("INSERT INTO probe_result VALUES ('bad.gp',NULL,'rejected',NULL,'consumer rejected','[]','[]')")
        con.execute("INSERT INTO probe_comparison VALUES ('bad.gp','fermata',1,NULL,NULL,'target-blocked','[1]',NULL)")
        self.assertEqual(con.execute("SELECT nondefault_fixtures,differing_fixtures,source_blocked_fixtures,target_blocked_fixtures,blocked_fixtures FROM probe_coverage").fetchone(), (1, 0, 0, 1, 1))
        self.assertEqual(con.execute("SELECT COUNT(*) FROM observed_differences").fetchone()[0], 0)

    def test_source_and_target_consumer_blockers_are_independent(self):
        con = self.database()
        self.capability(con, "fermata")
        con.executemany("INSERT INTO probe_result VALUES (?,?,?,?,?,?,?)", [
            ("source.gp", None, None, "source failed", None, "[]", "[]"),
            ("target.gp", None, None, None, "target failed", "[]", "[]")])
        con.executemany("INSERT INTO probe_comparison VALUES (?,?,?,?,?,?,?,?)", [
            ("source.gp", "fermata", None, 1, None, "source-blocked", "null", "[1]"),
            ("target.gp", "fermata", 1, None, None, "target-blocked", "[1]", None)])
        self.assertEqual(con.execute("SELECT source_blocked_fixtures,target_blocked_fixtures,blocked_fixtures FROM probe_coverage").fetchone(), (1, 1, 2))

    def test_aggregate_blocker_count_does_not_double_count_both_sides(self):
        con = self.database()
        self.capability(con, "fermata")
        con.execute("INSERT INTO probe_result VALUES ('both.gp',NULL,NULL,'source failed','target failed','[]','[]')")
        con.execute("INSERT INTO probe_comparison VALUES ('both.gp','fermata',NULL,NULL,NULL,'source-blocked','null',NULL)")
        self.assertEqual(con.execute("SELECT source_blocked_fixtures,target_blocked_fixtures,blocked_fixtures FROM probe_coverage").fetchone(), (1, 1, 1))

    def test_default_only_probe_rows_remain_visible(self):
        con = self.database()
        self.capability(con, "fermata")
        con.execute("INSERT INTO probe_result VALUES ('default.gp',NULL,NULL,NULL,NULL,'[]','[]')")
        con.execute("INSERT INTO probe_comparison VALUES ('default.gp','fermata',0,0,1,'default-only','[]','[]')")
        self.assertEqual(con.execute("SELECT fixtures,default_only_fixtures,nondefault_fixtures FROM probe_coverage").fetchone(), (1, 1, 0))

    def test_supported_stage_without_claim_stays_visible(self):
        con = self.database()
        self.capability(con, "fermata")
        con.execute("INSERT INTO assessment VALUES ('fermata','import','supported')")
        self.assertEqual(con.execute("SELECT capability_id,stage FROM supported_without_claims").fetchall(), [("fermata", "import")])

    def test_supported_stage_without_claim_is_rejected(self):
        catalog = {"capabilities": [{"id": "fermata", "stages": {"import": "supported", "model": "partial", "export": "partial"}}]}
        ledger = {"semanticMatrix": {"capabilityClaims": []}}
        with self.assertRaisesRegex(ValueError, "without executable claim"):
            manage.validate_support_claims(catalog, ledger)

    def test_claim_for_non_supported_stage_is_rejected(self):
        catalog = {"capabilities": [{"id": "fermata", "stages": {"import": "partial", "model": "partial", "export": "partial"}}]}
        ledger = {"semanticMatrix": {"capabilityClaims": [{"capability": "fermata", "stage": "import"}]}}
        with self.assertRaisesRegex(ValueError, "non-supported stage"):
            manage.validate_support_claims(catalog, ledger)

    def test_check_rejects_stale_database_without_replacing_it(self):
        with tempfile.TemporaryDirectory() as directory:
            db = Path(directory) / "capabilities.sqlite"
            with closing(sqlite3.connect(db)) as con:
                con.execute("CREATE TABLE sentinel(value)")
                con.execute("INSERT INTO sentinel VALUES ('original')")
                con.commit()
            before = db.read_bytes()

            def build(path):
                with closing(sqlite3.connect(path)) as con:
                    con.execute("CREATE TABLE sentinel(value)")
                    con.execute("INSERT INTO sentinel VALUES ('changed')")
                    con.commit()

            with patch.object(manage, "HERE", Path(directory)), patch.object(manage, "DATABASE", db), patch.object(manage, "build", build):
                with self.assertRaisesRegex(ValueError, "stale"):
                    manage.refresh(check=True)
            self.assertEqual(db.read_bytes(), before)

    def test_discovery_keeps_actual_owner_and_enum_members(self):
        _, symbols = manage.discover()
        names = {(s["name"], s["kind"]) for s in symbols}
        self.assertIn(("BeamingRules.groups", "field"), names)
        self.assertNotIn(("MasterBar.groups", "field"), names)
        self.assertIn(("SyncPointData.barOccurence", "field"), names)
        self.assertNotIn(("Automation.barOccurence", "field"), names)
        self.assertIn(("SimileMark.FirstOfDouble", "enum-member"), names)


if __name__ == "__main__":
    unittest.main()
