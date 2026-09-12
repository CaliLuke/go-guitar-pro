"""Native key signatures and playback for the corrected semitone transposition."""
import hashlib
import json
import unittest
from pathlib import Path

from native_repair_facts import gpif_facts, midi_facts

ROOT = Path(__file__).resolve().parents[2]
EVIDENCE = ROOT / 'conformance/capabilities/evidence/guitar-pro-key-authority'


class NativeKeyAuthorityEvidence(unittest.TestCase):
    def test_hashes_and_raw_facts(self):
        receipt = json.loads(EVIDENCE.with_suffix('.json').read_text())
        for artifact in receipt['artifacts']:
            path = ROOT / artifact['path']
            with self.subTest(path=path):
                self.assertEqual(hashlib.sha256(path.read_bytes()).hexdigest(), artifact['sha256'])
                if path.suffix == '.gp':
                    self.assertEqual(json.loads(json.dumps(gpif_facts(path))), artifact['gpif'])
                elif path.suffix == '.mid':
                    self.assertEqual(midi_facts(path), artifact['midi'])

    def test_pitch_coordinates_and_playback(self):
        for name, octave in [('minus-one', '5'), ('plus-eleven', '4')]:
            for stage in ['export', 'native']:
                facts = gpif_facts(EVIDENCE / f'{name}-{stage}.gp')['note_occurrences']
                self.assertEqual(len(facts), 1)
                self.assertEqual(facts[0]['properties']['ConcertPitch'], {'Step': 'C', 'Accidental': '', 'Octave': '5'})
                self.assertEqual(facts[0]['properties']['TransposedPitch'], {'Step': 'D', 'Accidental': 'b', 'Octave': octave})
            midi = midi_facts(EVIDENCE / f'{name}-native.mid')
            self.assertEqual(midi['division'], 480)
            self.assertEqual([track for track in midi['note_events'] if track], [[[0, 144, 60, 89], [1920, 128, 60, 64]]])
