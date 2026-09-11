"""Protect work dispatch, evidence requirements, and publication identity."""

import copy
import base64
import json
import sqlite3
import unittest
from contextlib import closing
from unittest.mock import patch

import backlog
import manage


class BacklogTests(unittest.TestCase):
    def setUp(self):
        self.items = backlog.load()
        # Exercise workflow transitions independently of live ticket progress.
        for item in self.items:
            item.update(status='todo', owner=None, resolution=None, verification=[])
            if item.get('issue'):
                item['issue']['state'] = 'OPEN'
        self.catalog = backlog.read(backlog.HERE / 'catalog.json')
        self.meta = {'published': False}

    def test_uncovered_gap_is_rejected(self):
        gap = next(c['id'] for c in self.catalog['capabilities']
                   if c['scope'] == 'guitar-pro' and any(stage != 'supported' for stage in c['stages'].values()))
        items = [w for w in self.items if w['capability'] != gap]
        with self.assertRaisesRegex(ValueError, 'Uncovered'):
            backlog.validate(items, self.catalog, self.meta)

    def test_missing_dependency_and_cycles_are_rejected(self):
        items = copy.deepcopy(self.items)
        items[0]['depends_on'] = ['does-not-exist']
        with self.assertRaisesRegex(ValueError, 'Missing dependency'):
            backlog.validate(items, self.catalog, self.meta)
        items[0]['depends_on'] = [items[1]['id']]
        items[1]['depends_on'] = [items[0]['id']]
        with self.assertRaisesRegex(ValueError, 'cycle'):
            backlog.validate(items, self.catalog, self.meta)

    def test_done_without_receipt_and_empty_acceptance_are_rejected(self):
        items = copy.deepcopy(self.items)
        items[0]['status'] = 'done'
        with self.assertRaisesRegex(ValueError, 'resolution and verification'):
            backlog.validate(items, self.catalog, self.meta)
        items[0]['status'] = 'todo'
        items[0]['acceptance'] = ['Fix it']
        with self.assertRaisesRegex(ValueError, 'observable acceptance'):
            backlog.validate(items, self.catalog, self.meta)

    def test_published_backlog_requires_issue_for_every_item(self):
        items = copy.deepcopy(self.items)
        items[0]['issue'] = None
        with self.assertRaisesRegex(ValueError, 'unticketed'):
            backlog.validate(items, self.catalog, {'published': True})

    def test_completed_investigation_does_not_promote_support(self):
        items = copy.deepcopy(self.items)
        w = next(w for w in items if not w['depends_on'])
        w['reproduction'].pop('external_inputs', None)
        w.update(kind='investigation', status='done', resolution='Confirmed target-consumer limit; exact decision in evidence.', verification=[{
            'commit': 'a' * 40, 'commands': ['focused comparison: PASS'], 'evidence': ['decision with non-default source and wire values']
        }])
        before = copy.deepcopy(self.catalog)
        backlog.validate(items, self.catalog, self.meta)
        self.assertEqual(self.catalog, before)

    def test_external_inputs_block_dispatch_and_completion_until_evidenced(self):
        items = copy.deepcopy(self.items)
        w = next(w for w in items if w['id'] == 'backing-track')
        required = {'id': 'authorized-fixture', 'state': 'missing',
                    'requirement': 'A valid non-default source fixture with provenance.',
                    'acquisition': 'Create and verify the fixture using an authorized source editor.'}
        w['reproduction']['external_inputs'] = [required]
        with closing(sqlite3.connect(':memory:')) as con:
            con.executescript((manage.HERE / 'schema.sql').read_text())
            for c in self.catalog['capabilities']:
                con.execute('INSERT INTO capability VALUES (?,?,?,?,?,?,?,?)',
                            (c['id'], c['domain'], c['title'], c['scope'], c['priority'], '[]', c['finding'], c['acceptance']))
            with patch.object(backlog, 'load', return_value=items):
                backlog.populate(con, self.catalog)
            self.assertIsNone(con.execute("SELECT id FROM ready_work WHERE id='backing-track'").fetchone())
            self.assertEqual(con.execute("SELECT input_id,requirement FROM external_input_blockers WHERE id='backing-track'").fetchone(),
                             ('authorized-fixture', required['requirement']))
            self.assertIn('needs-input', backlog.report(items, self.meta))
            w['status'] = 'done'
            with self.assertRaisesRegex(ValueError, 'missing external inputs'):
                backlog.validate(items, self.catalog, self.meta)
            w['status'] = 'todo'
            required['state'] = 'available'
            with self.assertRaisesRegex(ValueError, 'evidence reference'):
                backlog.validate(items, self.catalog, self.meta)
            required['reference'] = 'A reviewed fixture receipt with authoring settings and checksum.'
            backlog.validate(items, self.catalog, self.meta)
            con.execute("UPDATE work_item SET reproduction_json=? WHERE id='backing-track'", (json.dumps(w['reproduction']),))
            self.assertEqual(con.execute("SELECT id FROM ready_work WHERE id='backing-track'").fetchone(), ('backing-track',))
            self.assertIsNone(con.execute("SELECT id FROM external_input_blockers WHERE id='backing-track'").fetchone())

    def test_dispatch_respects_dependencies_ownership_and_issue_state(self):
        items = copy.deepcopy(self.items)
        backing_track = next(w for w in items if w['id'] == 'backing-track')
        backing_track.update(status='todo', owner=None)
        with closing(sqlite3.connect(':memory:')) as con:
            con.executescript((manage.HERE / 'schema.sql').read_text())
            for c in self.catalog['capabilities']:
                con.execute('INSERT INTO capability VALUES (?,?,?,?,?,?,?,?)', (c['id'], c['domain'], c['title'], c['scope'], c['priority'], '[]', c['finding'], c['acceptance']))
            with patch.object(backlog, 'load', return_value=items):
                backlog.populate(con, self.catalog)
            ready = {r[0] for r in con.execute('SELECT id FROM ready_work')}
            self.assertIn('backing-track', ready)
            self.assertNotIn('sync-points', ready)
            con.execute("UPDATE work_item SET status='done' WHERE id='backing-track'")
            self.assertIn('sync-points', {r[0] for r in con.execute('SELECT id FROM ready_work')})
            con.execute("UPDATE work_item SET status='in_progress',owner='agent' WHERE id='fermata'")
            con.execute("UPDATE work_item SET issue_state='CLOSED' WHERE id='key'")
            ready = {r[0] for r in con.execute('SELECT id FROM ready_work')}
            self.assertNotIn('fermata', ready)
            self.assertNotIn('key', ready)
            self.assertEqual(con.execute("SELECT id FROM backlog_state_drift WHERE id='key'").fetchone(), ('key',))

    def test_publication_identity_includes_closed_issues_and_rejects_duplicates(self):
        issues = [{'number': 10, 'state': 'CLOSED', 'body': backlog.marker('simile')}]
        self.assertEqual(backlog.find_existing(issues, 'simile')['number'], 10)
        self.assertIsNone(backlog.find_existing(issues, 'directions'))
        with self.assertRaisesRegex(ValueError, 'Duplicate GitHub markers'):
            backlog.find_existing(issues * 2, 'simile')

    def test_github_control_byte_display_is_lossless_and_does_not_mutate_evidence(self):
        value = {'text': chr(0) + 'Intro' + chr(0), 'other': 'Verse\nChorus'}
        shown = backlog.display_evidence(value)
        self.assertEqual(shown['text']['encoding'], 'base64-utf8')
        self.assertEqual(base64.b64decode(shown['text']['data']).decode(), value['text'])
        self.assertEqual(shown['other'], value['other'])
        self.assertTrue(value['text'].startswith(chr(0)))
        self.assertNotIn('\\u0000', json.dumps(shown))

    def test_unprobed_capability_has_no_phantom_blocked_comparison(self):
        with closing(sqlite3.connect(':memory:')) as con:
            con.executescript((manage.HERE / 'schema.sql').read_text())
            con.execute("INSERT INTO capability VALUES ('unprobed','test','Unprobed','guitar-pro',1,'[]','Unknown','Review')")
            self.assertEqual(con.execute('SELECT fixtures,nondefault_fixtures,differing_fixtures,blocked_fixtures FROM probe_coverage').fetchone(), (0, 0, 0, 0))


if __name__ == '__main__':
    unittest.main()
