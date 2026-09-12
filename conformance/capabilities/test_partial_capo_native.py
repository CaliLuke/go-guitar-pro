"""Independent native facts for string ordering, capo units and open-note behavior."""
import hashlib
import json
import unittest
from pathlib import Path

from native_repair_facts import midi_facts
from partial_capo_facts import partial_capo_facts

ROOT = Path(__file__).resolve().parents[2]
DIRECTORY = ROOT / 'conformance/capabilities/evidence/partial-capo-native'


class PartialCapoNativeEvidence(unittest.TestCase):
    def test_artifact_integrity(self):
        receipt = json.loads(DIRECTORY.with_suffix('.json').read_text())
        self.assertEqual(receipt['application']['version'], '8.1.5')
        self.assertEqual(receipt['application']['build'], 31)
        for artifact in receipt['artifacts']:
            with self.subTest(path=artifact['path']):
                path = ROOT / artifact['path']
                self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), artifact['sha256'])
                if artifact.get('facts_kind') == 'gpif':
                    self.assertEqual(json.loads(json.dumps(partial_capo_facts(path))), artifact['facts'])
                elif artifact.get('facts_kind') == 'midi':
                    self.assertEqual(midi_facts(path), artifact['facts'])

    def test_native_string_order_and_pitches(self):
        cases = [
            ('control', 0, 0, '000000', [64, 59, 55, 50, 45, 40]),
            ('asymmetric', 0, 3, '001001', [67, 59, 55, 53, 45, 40]),
            ('single-first', 0, 3, '000001', [67, 59, 55, 50, 45, 40]),
            ('single-fourth', 0, 3, '001000', [64, 59, 55, 53, 45, 40]),
            ('combined', 2, 3, '001001', [69, 61, 57, 55, 47, 42]),
            ('at-whole', 5, 0, '001001', [69, 64, 60, 55, 50, 45]),
            ('fretted', 0, 3, '111111', [67, 60, 57, 53, 49, 45]),
        ]
        for name, whole, offset, flags, pitches in cases:
            with self.subTest(name=name):
                source = partial_capo_facts(ROOT / f'testdata/gp8/partial-capo-{name}.gp')
                self.assertEqual(source['staves'], [{'track': 0, 'staff': 0, 'whole': whole,
                    'partial_offset': offset, 'wire_flags': flags,
                    'wire_tuning': [40, 45, 50, 55, 59, 64]}])
                notes = source['note_occurrences']
                self.assertEqual([n['path'] for n in notes], [[0, 0, 0, i, 0] for i in range(6)])
                self.assertEqual([int(n['properties']['String']) for n in notes], [5, 4, 3, 2, 1, 0])
                self.assertEqual([int(n['properties']['Fret']) for n in notes], list(range(6)) if name == 'fretted' else [0] * 6)
                expected = []
                for i, pitch in enumerate(pitches):
                    expected.extend([[480 * i, 144, pitch, 89], [480 * (i + 1), 128, pitch, 64]])
                self.assertEqual(midi_facts(DIRECTORY / f'{name}.mid'), {'division': 480, 'note_events': [[], expected]})

    def test_named_native_ui_and_boundary(self):
        for name, label in [('asymmetric-authored', 'On strings 1, 4'),
                            ('single-first', 'On string 1'), ('single-fourth', 'On string 4')]:
            text = (DIRECTORY / f'{name}-ui.txt').read_text()
            self.assertIn(label, text)
            self.assertIn('34 stepper (settable) Value: 3', text)
        combined = (DIRECTORY / 'combined-absolute-authored-ui.txt').read_text()
        self.assertIn('34 stepper (settable) Value: 5', combined)
        self.assertIn('36 stepper (settable) Value: 2', combined)
        rejected = (DIRECTORY / 'below-rejected-ui.txt').read_text()
        self.assertIn('34 stepper (settable) Value: 5', rejected)
        self.assertIn('36 stepper (settable) Value: 5', rejected)

    def test_library_exports_in_native_guitar_pro(self):
        for name in ['asymmetric', 'combined', 'fretted']:
            with self.subTest(name=name):
                exported = partial_capo_facts(DIRECTORY / f'exports/{name}.gp')
                native = partial_capo_facts(DIRECTORY / f'exports/{name}-native.gp')
                source = partial_capo_facts(ROOT / f'testdata/gp8/partial-capo-{name}.gp')
                self.assertEqual(exported['staves'], source['staves'])
                self.assertEqual(native['staves'], source['staves'])
                self.assertEqual(exported['note_occurrences'], native['note_occurrences'])
                self.assertEqual(len(native['note_occurrences']), 6)
                self.assertEqual(midi_facts(DIRECTORY / f'exports/{name}-native.mid'),
                                 midi_facts(DIRECTORY / f'{name}.mid'))
        for name, pitches in [('grace', [67, 65, 59, 55, 53, 45, 40]),
                              ('edited', [64, 60, 55, 50, 45, 41]),
                              ('cleared', [64, 59, 55, 50, 45, 40])]:
            with self.subTest(name=name):
                exported = partial_capo_facts(DIRECTORY / f'exports/{name}.gp')
                native = partial_capo_facts(DIRECTORY / f'exports/{name}-native.gp')
                self.assertEqual(exported['note_occurrences'], native['note_occurrences'])
                self.assertEqual(len(native['note_occurrences']), len(pitches))
                midi = midi_facts(DIRECTORY / f'exports/{name}-native.mid')
                self.assertEqual([e[2] for tr in midi['note_events'] for e in tr if e[1] >> 4 == 9], pitches)
                if name != 'grace':
                    expected = []
                    for i, pitch in enumerate(pitches):
                        expected.extend([[480 * i, 144, pitch, 89], [480 * (i + 1), 128, pitch, 64]])
                    self.assertEqual(midi, {'division': 480, 'note_events': [[], expected]})

    def test_baseline_and_pinned_limits(self):
        baseline = json.loads((DIRECTORY / 'go-baseline.json').read_text())
        self.assertEqual(baseline['source_commit'], '9d282eb5dd2d403a647cbda98634210d42bf7196')
        self.assertEqual(len(baseline['results']), 7)
        for case in baseline['results']:
            self.assertEqual(case['Error'], '')
            self.assertIsNone(case['Staves'][0]['PartialCapo'])
            paths = [d['SourcePath'] for d in case['Diagnostics'] if d['Code'] == 'GPIF.Staff.Property.Unknown']
            for index, name in [(2, 'PartialCapoFret'), (3, 'PartialCapoStringFlags')]:
                self.assertIn(f'/GPIF/Tracks/Track[@id="0"]/Staves/Staff[0]/Properties/Property[{index}][@name="{name}"]', paths)
        pinned = json.loads((DIRECTORY / 'alphatab.json').read_text())
        self.assertEqual(len(pinned['results']), 7)
        for case in pinned['results']:
            self.assertNotIn('error', case)
            whole = case['staves'][0]['capo']
            frets = range(6) if case['file'].endswith('-fretted.gp') else [0] * 6
            self.assertEqual([n['midi'] for n in case['notes']],
                             [tuning + whole + fret for tuning, fret in zip([64, 59, 55, 50, 45, 40], frets)])
