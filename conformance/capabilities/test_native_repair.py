"""Hash-bound native application evidence, independent of Go self-round-trips."""
import hashlib
import json
import unittest
from pathlib import Path
from native_repair_facts import gpif_facts, midi_facts

ROOT = Path(__file__).resolve().parents[2]
EVIDENCE = ROOT / 'conformance/capabilities/evidence/guitar-pro-repair'


class NativeRepairEvidence(unittest.TestCase):
    def test_artifacts(self):
        receipt = json.loads(EVIDENCE.with_suffix('.json').read_text())
        self.assertEqual(receipt['application'], {'name': 'Guitar Pro 8', 'version': '8.1.5', 'build': 31})
        for artifact in receipt['artifacts']:
            with self.subTest(path=artifact['path']):
                path = ROOT / artifact['path']
                self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), artifact['sha256'])
                if path.suffix == '.gp':
                    self.assertEqual(json.loads(json.dumps(gpif_facts(path))), artifact['facts'])
                elif path.suffix == '.mid':
                    self.assertEqual(midi_facts(path), artifact['facts'])

    def test_mix_native_acceptance(self):
        original = EVIDENCE.parent / 'guitar-pro-native/original-native.mid'
        self.assertEqual(midi_facts(original), midi_facts(EVIDENCE / 'mix-native.mid'))
        before = gpif_facts(EVIDENCE / 'mix-export.gp')
        after = gpif_facts(EVIDENCE / 'mix-native.gp')
        self.assertEqual(len(before['note_occurrences']), 12)
        self.assertEqual(before['note_occurrences'], after['note_occurrences'])
        events = after['automations']
        for kind, position, value in [('Tempo', '0.25', '173 2'), ('DSPParam_12', '0.25', '0.56'), ('DSPParam_11', '0.25', '0.3125')]:
            self.assertIn({'Type': kind, 'Bar': '0', 'Position': position, 'Value': value, 'Linear': 'false'}, events)
        sounds = [a for a in events if a['Type'] == 'Sound']
        self.assertEqual([(a['Bar'], a['Position'], a['Linear']) for a in sounds], [('0', '1', 'false'), ('4', '0', 'false')])
        self.assertIn('stepper (settable) 481', (EVIDENCE / 'sound-dialog.txt').read_text())

    def test_full_native_volume_scale(self):
        expected = [0, .05, .10, .14, .20, .25, .31, .37, .43, .48, .50, .53, .56, .58, .61, .64, .66]
        original = (ROOT / 'testdata/gp3/mix-table-events.gp3').read_bytes()
        for value, gain in enumerate(expected):
            with self.subTest(value=value):
                source = (EVIDENCE / f'volume/v{value:02}.gp3').read_bytes()
                self.assertEqual(source[:980], original[:980])
                self.assertEqual(source[981:], original[981:])
                self.assertEqual(source[980], value)
                events = gpif_facts(EVIDENCE / f'volume/v{value:02}.gp')['automations']
                event = [a for a in events if a['Type'] == 'DSPParam_12' and a['Position'] == '0.25']
                self.assertEqual(len(event), 1)
                self.assertEqual(float(event[0]['Value']), gain)
                self.assertEqual(event[0]['Linear'], 'false')

    def test_context_playback(self):
        pitches = {'default': [61], 'flat': [61], 'capo': [64], 'display': [61], 'clef': [61], 'octave': [61], 'harmonic': [84], 'absolute': [61], 'sounding': [61], 'independent-capo': [61, 64]}
        for name, expected in pitches.items():
            with self.subTest(context=name):
                facts = midi_facts(EVIDENCE / f'contexts/{name}-native.mid')
                self.assertEqual(facts['division'], 480)
                tracks = [t for t in facts['note_events'] if t]
                self.assertEqual(len(tracks), len(expected))
                for track, pitch in zip(tracks, expected):
                    self.assertEqual(track, [[0, 144, pitch, 89], [1920, 128, pitch, 64]])
        grace = midi_facts(EVIDENCE / 'contexts/grace-native.mid')
        self.assertEqual([e[2] for t in grace['note_events'] for e in t if e[1] >> 4 == 9], [63, 61])
    def test_notation_native_acceptance(self):
        import struct
        import zipfile
        for name in ['export.gp', 'notation-native.gp']:
            with zipfile.ZipFile(EVIDENCE / 'notation' / name) as archive:
                data = archive.read('Content/PartConfiguration')
            # Three native views: full score, track1 grand staff, track2 single staff.
            self.assertEqual(struct.unpack_from('>i', data)[0], 3)
            position = 4
            flags = []
            for _ in range(3):
                count = struct.unpack_from('>i', data, position + 1)[0]
                position += 5
                flags.append(list(data[position:position + count]))
                position += count
            self.assertEqual(flags, [[1, 14], [1], [14]])
        before = gpif_facts(EVIDENCE / 'notation/export.gp')
        after = gpif_facts(EVIDENCE / 'notation/notation-native.gp')
        self.assertEqual(before['note_occurrences'], after['note_occurrences'])
        self.assertEqual(len(after['note_occurrences']), 3)
        midi = midi_facts(EVIDENCE / 'notation/notation-native.mid')
        self.assertEqual(midi['note_events'], [[], [[0, 144, 58, 89], [1920, 128, 58, 64]], [[0, 144, 58, 89], [1920, 128, 58, 64]], [[0, 146, 58, 89], [1920, 130, 58, 64]]])
