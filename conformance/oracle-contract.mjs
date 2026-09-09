import assert from 'node:assert/strict';
import * as alphaTab from '@coderline/alphatab';
import { normalizeAutomation, normalizeBend, normalizeClef } from './oracle.mjs';

const clefCases = [
  [alphaTab.model.Clef.G2, 'treble'],
  [alphaTab.model.Clef.F4, 'bass'],
  [alphaTab.model.Clef.C4, 'tenor'],
  [alphaTab.model.Clef.C3, 'alto'],
  [alphaTab.model.Clef.Neutral, 'neutral']
];
assert.equal(new Set(clefCases.map(([value]) => value)).size, clefCases.length, 'clef enum values must remain distinct');
for (const [value, expected] of clefCases) {
  assert.equal(normalizeClef(value), expected);
}
assert.equal(normalizeClef(999), 'unknown:999');

assert.deepEqual(normalizeBend([{ offset: 0, value: 2 }, { offset: 60, value: 4 }]), [
  { position: 0, value: 2 },
  { position: 12, value: 4 }
]);
assert.deepEqual(normalizeAutomation({ ratioPosition: 0.25, value: 120.5, isLinear: true, type: 0 }, 3), {
  bar: 3,
  position: 0.25,
  type: 'tempo',
  value: 120.5,
  linear: true
});
