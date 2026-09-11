"""Check that inventory failures and uncertainty remain visible."""

import sqlite3
import copy
import os
import re
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

    def ownership_inputs(self):
        sources, constructs = manage.discover()
        ownership = manage.read_upstream_ownership()
        catalog = manage.read_json(manage.HERE / "catalog.json")
        capability_ids = {capability["id"] for capability in catalog["capabilities"]}
        return sources, constructs, ownership, capability_ids

    def test_discovery_includes_gp_importer_assignments_and_dispatches(self):
        _, constructs = manage.discover()
        identities = {(construct["kind"], construct["name"]) for construct in constructs}
        self.assertIn(("assignment", "GpifParser._parseScoreNode:this.score.title"), identities)
        self.assertIn(("dispatch", "GpifParser._parseBeatProperties:Brush"), identities)

    def test_in_scope_gp_construct_requires_exact_review(self):
        sources, constructs, ownership, capability_ids = self.ownership_inputs()
        ownership = copy.deepcopy(ownership)
        target = "packages/alphatab/src/importer/GpifParser.ts::dispatch::GpifParser._parseBeatProperties:Brush::1"
        ownership["construct_reviews"] = [review for review in ownership["construct_reviews"]
                                          if review["construct_id"] != target]
        with self.assertRaisesRegex(ValueError, "dispatch or assignment has no review"):
            manage.validate_upstream_ownership(ownership, sources, constructs, capability_ids)

    def test_authored_construct_requires_non_wildcard_primary_owner(self):
        sources, constructs, ownership, capability_ids = self.ownership_inputs()
        ownership = copy.deepcopy(ownership)
        review = next(review for review in ownership["construct_reviews"]
                      if review["construct_id"].endswith("GpifParser._parseBeatProperties:Brush::1"))
        review["primary_capability"] = "*"
        with self.assertRaisesRegex(ValueError, "Wildcard-only authored ownership"):
            manage.validate_upstream_ownership(ownership, sources, constructs, capability_ids)

    def test_importer_populated_model_field_or_enum_requires_owner(self):
        sources, constructs, ownership, capability_ids = self.ownership_inputs()
        ownership = copy.deepcopy(ownership)
        review = next(review for review in ownership["model_reviews"]
                      if review["declaration"] == "Score.title")
        review["primary_capability"] = None
        with self.assertRaisesRegex(ValueError, "field or enum has no owner"):
            manage.validate_upstream_ownership(ownership, sources, constructs, capability_ids)

    def test_model_review_cannot_disappear_with_assignment_links(self):
        sources, constructs, ownership, capability_ids = self.ownership_inputs()
        ownership = copy.deepcopy(ownership)
        for review in ownership["construct_reviews"]:
            if review.get("model_declaration") == "Score.title":
                review["model_declaration"] = None
        ownership["model_reviews"] = [review for review in ownership["model_reviews"]
                                      if review["declaration"] != "Score.title"]
        ownership["migration"]["reviewed_model_declarations"] -= 1
        with self.assertRaisesRegex(ValueError, "inferred authored assignment has no model declaration link Score.title"):
            manage.validate_upstream_ownership(ownership, sources, constructs, capability_ids)

    def test_typed_alias_model_review_cannot_disappear_with_assignment_links(self):
        sources, constructs, original, capability_ids = self.ownership_inputs()
        for declaration in ("Automation.value", "BendPoint.value", "SyncPointData.barOccurence"):
            with self.subTest(declaration=declaration):
                ownership = copy.deepcopy(original)
                for review in ownership["construct_reviews"]:
                    if review.get("model_declaration") == declaration:
                        review["model_declaration"] = None
                ownership["model_reviews"] = [review for review in ownership["model_reviews"]
                                              if review["declaration"] != declaration]
                ownership["migration"]["reviewed_model_declarations"] -= 1
                with self.assertRaisesRegex(
                        ValueError, f"inferred authored assignment has no model declaration link {re.escape(declaration)}"):
                    manage.validate_upstream_ownership(ownership, sources, constructs, capability_ids)

    def test_stale_construct_review_is_rejected(self):
        sources, constructs, ownership, capability_ids = self.ownership_inputs()
        ownership = copy.deepcopy(ownership)
        ownership["construct_reviews"][0]["construct_id"] += ":stale"
        with self.assertRaisesRegex(ValueError, "Stale unmatched upstream construct review"):
            manage.validate_upstream_ownership(ownership, sources, constructs, capability_ids)

    def test_other_product_construct_stays_visible_without_gp_obligation(self):
        _, constructs = manage.discover_files({
            "packages/alphatab/src/importer/MusicXmlImporter.ts": b"""
export class MusicXmlImporter {
    public read() {
        target.reviewProbe = true;
        switch (target.kind) {
        case 'ReviewProbe':
        }
    }
}
"""})
        self.assertEqual({construct["kind"] for construct in constructs},
                         {"class", "method", "assignment", "dispatch"})
        ownership = manage.read_upstream_ownership()
        policy = manage.match_source_policy(ownership["path_policies"], constructs[0]["path"])
        self.assertEqual(policy["source_scope"], "excluded-other-product")
        self.assertEqual(policy["review_kinds"], [])
        renderer = manage.match_source_policy(
            ownership["path_policies"], "packages/alphatab/src/rendering/ScoreRenderer.ts")
        self.assertEqual(renderer["source_scope"], "excluded-renderer")

    def test_canonical_model_owner_migration_is_bounded(self):
        inventory = manage.read_json(manage.ROOT / "conformance/upstream-inventory.json")
        migration = inventory["modelSymbolMigration"]
        self.assertEqual((migration["previous"], migration["current"], migration["renamed"]),
                         (384, 384, 28))
        self.assertEqual(migration["unexplainedRemovals"], [])

    def test_reviewed_decoder_and_stylesheet_owners_are_semantic(self):
        ownership = manage.read_upstream_ownership()
        construct_reviews = {review["construct_id"]: review for review in ownership["construct_reviews"]}
        self.assertEqual(construct_reviews[
            "packages/alphatab/src/importer/Gp3To5Importer.ts::dispatch::Gp3To5Importer._toStrokeValue:1::1"
        ]["primary_capability"], "brush")
        self.assertEqual(construct_reviews[
            "packages/alphatab/src/importer/Gp3To5Importer.ts::dispatch::Gp3To5Importer.readArtificialHarmonic:15::1"
        ]["primary_capability"], "harmonics")
        self.assertEqual(construct_reviews[
            "packages/alphatab/src/importer/BinaryStylesheet.ts::assignment::BinaryStylesheet.apply:score.stylesheet.firstSystemTrackNameMode::1"
        ]["primary_capability"], "stylesheet")
        model_reviews = {review["declaration"]: review for review in ownership["model_reviews"]}
        self.assertEqual(model_reviews["RenderStylesheet.firstSystemTrackNameMode"]["primary_capability"],
                         "stylesheet")
        self.assertEqual(model_reviews["RenderStylesheet.barNumberDisplay"]["primary_capability"],
                         "barlines")
        self.assertEqual(model_reviews["Tuning.name"]["primary_capability"], "tuning")

    def test_percussion_notehead_dispatches_have_semantic_owner(self):
        ownership = manage.read_upstream_ownership()
        prefix = (
            "packages/alphatab/src/importer/GpifParser.ts::dispatch::"
            "GpifParser.parseNoteHead:"
        )
        reviews = [review for review in ownership["construct_reviews"]
                   if review["construct_id"].startswith(prefix)]
        self.assertEqual(len(reviews), 22)
        for review in reviews:
            self.assertEqual(review["primary_capability"], "percussion")
            self.assertEqual(review["secondary_capabilities"], [])

        model_reviews = {review["declaration"]: review for review in ownership["model_reviews"]}
        for declaration in (
                "InstrumentArticulation.noteHeadDefault",
                "InstrumentArticulation.noteHeadHalf",
                "InstrumentArticulation.noteHeadWhole"):
            self.assertEqual(model_reviews[declaration]["primary_capability"], "percussion")

    def test_other_format_display_fields_stay_outside_guitar_pro_scope(self):
        catalog = manage.read_json(manage.HERE / "catalog.json")
        capabilities = {item["id"]: item for item in catalog["capabilities"]}
        inventory = manage.read_json(manage.ROOT / "conformance/upstream-inventory.json")
        symbols = {item["name"]: item for item in inventory["modelSymbols"]}
        pin = manage.read_json(manage.ROOT / "conformance/oracle.json")["sourceRevision"]
        cases = {
            "common-time": (["MasterBar.timeSignatureCommon"], [
                "importer/MusicXmlImporter.ts#L1960", "importer/CapellaParser.ts#L486",
                "importer/alphaTex/AlphaTex1LanguageHandler.ts#L641",
                "importer/GpifParser.ts#L1359", "exporter/GpifWriter.ts#L1614"]),
            "display-duration-override": (["Beat.overrideDisplayDuration"], [
                "importer/MusicXmlImporter.ts#L2511", "importer/MusicXmlImporter.ts#L3155",
                "importer/MusicXmlImporter.ts#L3271"]),
            "note-display": (["Note.isVisible", "Note.style", "NoteStyle.noteHead",
                              "NoteStyle.noteHeadCenterOnStem"], [
                "importer/MusicXmlImporter.ts#L2783", "importer/MusicXmlImporter.ts#L2838",
                "importer/MusicXmlImporter.ts#L3931",
                "importer/alphaTex/AlphaTex1LanguageHandler.ts#L2322",
                "model/Note.ts#L885", "importer/GpifParser.ts#L795"]),
        }
        con = self.database()
        for item in catalog["capabilities"]:
            self.capability(con, item["id"], item["scope"])
            con.executemany("INSERT INTO assessment VALUES (?,?,?)", [
                (item["id"], stage, status) for stage, status in item["stages"].items()])
        gaps = {row[0] for row in con.execute("SELECT id FROM gaps")}
        for capability, (names, references) in cases.items():
            with self.subTest(capability=capability):
                item = capabilities[capability]
                self.assertEqual(item["scope"], "excluded")
                self.assertEqual(item["formats"], ["gp3", "gp4", "gp5", "gp6", "gp7", "gp8"])
                self.assertEqual(item["stages"], dict.fromkeys(("import", "model", "export"), "out-of-scope"))
                self.assertNotIn(capability, gaps)
                urls = {e["reference"] for e in item["evidence"] if e["kind"] == "upstream-source"}
                for reference in references:
                    self.assertIn(f"https://github.com/CoderLine/alphaTab/blob/{pin}/packages/alphatab/src/{reference}", urls)
                for name in names:
                    self.assertEqual(symbols[name]["disposition"], "out-of-scope")
                    self.assertIn("GP3-GP8", symbols[name]["reason"])
                    self.assertIn(pin, symbols[name]["reason"])
        self.assertEqual(len(inventory["modelSymbols"]), 384)
        for name in ("meter", "duration", "tuplets", "percussion"):
            self.assertEqual(capabilities[name]["scope"], "guitar-pro")
        self.assertEqual(catalog["website_mapping"]["Time Signatures"], ["meter"])
        self.assertEqual(symbols["Beat.overrideDisplayDuration"]["feature"], "rhythm")
        self.assertEqual(symbols["MasterBar.timeSignatureCommon"]["feature"], "staff-ownership")

    def test_shared_automation_assignments_use_specific_semantic_owners(self):
        ownership = manage.read_upstream_ownership()
        reviews = {review["construct_id"]: review for review in ownership["construct_reviews"]}
        prefix = "packages/alphatab/src/importer/Gp3To5Importer.ts::assignment::Gp3To5Importer.readMixTableChange:"
        expected = {
            "balanceAutomation.type::1": "pan-automation",
            "instrumentAutomation.type::1": "sound-automation",
            "tempoAutomation.type::1": "tempo",
            "volumeAutomation.type::1": "volume-automation",
            "tableChange.instrument::1": "sound-automation",
        }
        for suffix, capability in expected.items():
            self.assertEqual(reviews[prefix + suffix]["primary_capability"], capability)

    def test_authored_importer_carriers_are_not_classified_as_derived_state(self):
        ownership = manage.read_upstream_ownership()
        reviews = {review["construct_id"]: review for review in ownership["construct_reviews"]}
        expected = {
            "packages/alphatab/src/importer/Gp3To5Importer.ts::assignment::Gp3To5Importer.readScore:this._initialTempo.text::1": "tempo",
            "packages/alphatab/src/importer/Gp3To5Importer.ts::assignment::Gp3To5Importer.readScore:this._initialTempo.value::1": "tempo",
            "packages/alphatab/src/importer/Gp3To5Importer.ts::assignment::Gp3To5Importer.readScore:this._globalTripletFeel::1": "triplet-feel",
            "packages/alphatab/src/importer/Gp3To5Importer.ts::assignment::Gp3To5Importer.readLyrics:this._lyricsTrack::1": "lyrics",
            "packages/alphatab/src/importer/GpifParser.ts::assignment::GpifParser._parseMasterTrackNode:this._hasAnacrusis::1": "pickup",
            "packages/alphatab/src/importer/GpifParser.ts::assignment::GpifParser._parseMasterTrackNode:this._tracksMapping::1": "ownership",
            "packages/alphatab/src/importer/GpifParser.ts::assignment::GpifParser._parseBackingTrackNode:this._backingTrackPadding::1": "backing-track",
            "packages/alphatab/src/importer/GpifParser.ts::assignment::GpifParser._parseBackingTrackNode:this._backingTrackAssetId::1": "backing-track",
        }
        for construct_id, capability in expected.items():
            self.assertEqual(reviews[construct_id]["disposition"], "authored")
            self.assertEqual(reviews[construct_id]["primary_capability"], capability)

    def test_unique_exact_catalog_owner_is_enforced(self):
        sources, constructs, ownership, capability_ids = self.ownership_inputs()
        ownership = copy.deepcopy(ownership)
        review = next(review for review in ownership["model_reviews"]
                      if review["declaration"] == "Beat.brushDuration")
        review["primary_capability"] = "duration"
        with self.assertRaisesRegex(ValueError, "Unique exact catalog owner mismatch.*Beat.brushDuration"):
            manage.validate_upstream_ownership(ownership, sources, constructs, capability_ids)

    def test_inferred_assignment_cannot_hide_wrong_owner_by_removing_model_link(self):
        sources, constructs, ownership, capability_ids = self.ownership_inputs()
        ownership = copy.deepcopy(ownership)
        target = next(review for review in ownership["construct_reviews"]
                      if review["construct_id"].endswith(
                          "Gp3To5Importer.readBeatEffects:beat.brushDuration::1"))
        target["model_declaration"] = None
        target["primary_capability"] = "duration"
        with self.assertRaisesRegex(
                ValueError, "inferred authored assignment has no model declaration link Beat.brushDuration"):
            manage.validate_upstream_ownership(ownership, sources, constructs, capability_ids)

    def test_semantic_model_owner_distinctions_match_exact_catalog_routes(self):
        ownership = manage.read_upstream_ownership()
        models = {review["declaration"]: review for review in ownership["model_reviews"]}
        expected = {
            "Beat.brushDuration": "brush",
            "Beat.lyrics": "beat-lyrics",
            "Beat.ottava": "beat-octave",
            "Beat.tupletNumerator": "tuplets",
            "Beat.tupletDenominator": "tuplets",
            "Note.durationPercent": "sound-duration",
        }
        for declaration, capability in expected.items():
            self.assertEqual(models[declaration]["primary_capability"], capability)
        for review in ownership["construct_reviews"]:
            declaration = review.get("model_declaration")
            if review["disposition"] == "authored" and declaration in expected:
                self.assertEqual(review["primary_capability"], expected[declaration])

    def test_catalog_owner_ambiguities_are_explicit_and_bounded(self):
        ownership = manage.read_upstream_ownership()
        ambiguities = {item["declaration"]: item for item in ownership["catalog_owner_ambiguities"]}
        self.assertEqual(set(ambiguities), {
            "Automation.isLinear", "Automation.isVisible", "Automation.ratioPosition",
            "Automation.text", "Automation.type", "Automation.value", "Beat.fade",
            "SyncPointData.millisecondOffset",
        })
        self.assertIn("tempo", ambiguities["Automation.value"]["allowed_primary_capabilities"])
        self.assertEqual(ambiguities["Beat.fade"]["catalog_capability"], "fade-other")

    def test_upstream_sensitivity_fixture_requires_every_authored_syntax_review(self):
        fixture = Path(os.environ.get(
            "UPSTREAM_SENSITIVITY_OVERLAY", manage.HERE / "upstream_sensitivity.ts"))
        source_path = "packages/alphatab/src/importer/GpifParser.ts"
        sources, constructs = manage.discover_files({source_path: fixture.read_bytes()})
        ownership = {
            "source_revision": manage.read_json(manage.ROOT / "conformance/oracle.json")["sourceRevision"],
            "path_policies": [{
                "pattern": source_path, "source_scope": "guitar-pro-importer", "formats": ["gp6", "gp7", "gp8"],
                "review_kinds": ["assignment", "dispatch"], "reason": "Sensitivity fixture.",
            }],
            "construct_reviews": [
                {
                    "construct_id": f"{source_path}::assignment::GpifParser.parse:beat.brush::1",
                    "source_scope": "guitar-pro-importer", "formats": ["gp6", "gp7", "gp8"],
                    "disposition": "authored", "reason": "Reviewed assignment.",
                    "primary_capability": "brush", "secondary_capabilities": [], "model_declaration": None,
                },
                {
                    "construct_id": f"{source_path}::dispatch::GpifParser.parse:Brush::1",
                    "source_scope": "guitar-pro-importer", "formats": ["gp6", "gp7", "gp8"],
                    "disposition": "authored", "reason": "Reviewed dispatch.",
                    "primary_capability": "brush", "secondary_capabilities": [], "model_declaration": None,
                },
            ],
            "model_reviews": [],
            "migration": {
                "previous_constructs": 2, "retained_constructs": 2, "added_assignments": 2,
                "current_constructs": 4, "removed_constructs": 0, "unexplained_removals": [],
                "source_files": 1, "reviewed_gp_constructs": 2, "reviewed_model_declarations": 0,
            },
        }
        manage.validate_upstream_ownership(ownership, sources, constructs, {"brush"}, baseline_constructs=2)


if __name__ == "__main__":
    unittest.main()
