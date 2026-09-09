import assert from 'node:assert/strict';
import * as alphaTab from '@coderline/alphatab';
import {
  enumName,
  normalizeAccent,
  normalizeAutomation,
  normalizeBeatStatus,
  normalizeBend,
  normalizeClef,
  normalizeDynamic,
  normalizeGrace,
  normalizeHairpin,
  normalizeHarmonicKind,
  normalizeNoteKind,
  normalizeNotePitch,
  normalizeOttavia,
  normalizeSlideIn,
  normalizeSlideOut,
  normalizeTuning,
  normalizeTripletFeel,
  normalizeVibrato
} from './oracle.mjs';

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

const enumCases = [
  [normalizeHarmonicKind, alphaTab.model.HarmonicType, [
    ['None', 'none'], ['Natural', 'natural'], ['Artificial', 'artificial'], ['Pinch', 'pinch'],
    ['Tap', 'tap'], ['Semi', 'semi'], ['Feedback', 'feedback']
  ]],
  [normalizeSlideIn, alphaTab.model.SlideInType, [
    ['None', 'none'], ['IntoFromBelow', 'into-from-below'], ['IntoFromAbove', 'into-from-above']
  ]],
  [normalizeSlideOut, alphaTab.model.SlideOutType, [
    ['None', 'none'], ['Shift', 'shift'], ['Legato', 'legato'], ['OutUp', 'out-up'],
    ['OutDown', 'out-down'], ['PickSlideDown', 'pick-slide-down'], ['PickSlideUp', 'pick-slide-up']
  ]],
  [normalizeAccent, alphaTab.model.AccentuationType, [
    ['None', 'none'], ['Normal', 'normal'], ['Heavy', 'heavy'], ['Tenuto', 'tenuto']
  ]],
  [normalizeHairpin, alphaTab.model.CrescendoType, [
    ['None', 'none'], ['Crescendo', 'crescendo'], ['Decrescendo', 'decrescendo']
  ]],
  [normalizeVibrato, alphaTab.model.VibratoType, [
    ['None', 'none'], ['Slight', 'slight'], ['Wide', 'wide']
  ]],
  [normalizeTripletFeel, alphaTab.model.TripletFeel, [
    ['NoTripletFeel', 'none'], ['Triplet16th', 'triplet-16th'], ['Triplet8th', 'triplet-8th'],
    ['Dotted16th', 'dotted-16th'], ['Dotted8th', 'dotted-8th'],
    ['Scottish16th', 'scottish-16th'], ['Scottish8th', 'scottish-8th']
  ]],
  [normalizeOttavia, alphaTab.model.Ottavia, [
    ['_15ma', '15ma'], ['_8va', '8va'], ['Regular', 'none'], ['_8vb', '8vb'], ['_15mb', '15mb']
  ]]
];
for (const [normalize, values, cases] of enumCases) {
  assert.equal(new Set(cases.map(([name]) => values[name])).size, cases.length, 'enum values must remain distinct');
  for (const [name, expected] of cases) {
    assert.equal(normalize(values[name]), expected);
  }
  assert.equal(normalize(999), 'unknown:999');
}

for (const name of [
  'PPP', 'PP', 'P', 'MP', 'MF', 'F', 'FF', 'FFF', 'PPPP', 'PPPPP', 'PPPPPP',
  'FFFF', 'FFFFF', 'FFFFFF', 'SF', 'SFP', 'SFPP', 'FP', 'RF', 'RFZ', 'SFZ',
  'SFFZ', 'FZ', 'N', 'PF', 'SFZP'
]) {
  assert.equal(normalizeDynamic(alphaTab.model.DynamicValue[name]), name.toLowerCase());
}
assert.equal(normalizeDynamic(999), 'unknown:999');
assert.equal(enumName(alphaTab.model.DynamicValue, 999), 'unknown:999');

assert.equal(normalizeBeatStatus({ isEmpty: true, isRest: false }), 'empty');
assert.equal(normalizeBeatStatus({ isEmpty: false, isRest: true }), 'rest');
assert.equal(normalizeBeatStatus({ isEmpty: false, isRest: false }), 'normal');
assert.equal(normalizeBeatStatus({ isEmpty: true, isRest: true }), 'unknown:empty+rest');
assert.equal(normalizeNoteKind({ isDead: true, isTieDestination: false }), 'dead');
assert.equal(normalizeNoteKind({ isDead: false, isTieDestination: true }), 'tie');
assert.equal(normalizeNoteKind({ isDead: false, isTieDestination: false }), 'normal');
assert.equal(normalizeNoteKind({ isDead: true, isTieDestination: true }), 'unknown:dead+tie');

assert.deepEqual(normalizeTuning([64, 59, 55, 50, 45, 40]), [64, 59, 55, 50, 45, 40]);
assert.deepEqual(normalizeNotePitch(
  { string: 2, fret: 3, realValueWithoutHarmonic: 69, percussionArticulation: -1 },
  { isPercussion: false, tuning: [64, 59], track: { percussionArticulations: [] } }
), {
  string: 1,
  fret: 3,
  percussionArticulation: null,
  percussionInput: null,
  midi: 69
});
assert.deepEqual(normalizeNotePitch(
  { string: 0, fret: 0, realValueWithoutHarmonic: 60, percussionArticulation: -1 },
  { isPercussion: false, tuning: [], track: { percussionArticulations: [] } }
), {
  string: 0,
  fret: null,
  percussionArticulation: null,
  percussionInput: null,
  midi: 60
});
assert.deepEqual(normalizeGrace(
  {
    fret: 3,
    isDead: false,
    dynamics: alphaTab.model.DynamicValue.F,
    realValue: 99,
    slideOutType: alphaTab.model.SlideOutType.None,
    isHammerPullOrigin: false
  },
  { graceType: alphaTab.model.GraceType.OnBeat, displayStart: 480, displayDuration: 120 },
  { isPercussion: false }
), {
  rawFret: 3,
  dead: false,
  onBeat: true,
  dynamic: 'f',
  transition: 'none',
  staffPercussion: false
});
assert.equal(normalizeGrace(
  {
    fret: 3,
    isDead: false,
    dynamics: alphaTab.model.DynamicValue.F,
    slideOutType: alphaTab.model.SlideOutType.None,
    isHammerPullOrigin: false
  },
  { graceType: 999 },
  { isPercussion: false }
).onBeat, 'unknown:999');
assert.equal(normalizeGrace(
  {
    fret: 3,
    isDead: false,
    dynamics: alphaTab.model.DynamicValue.F,
    slideOutType: 999,
    isHammerPullOrigin: false
  },
  { graceType: alphaTab.model.GraceType.BeforeBeat },
  { isPercussion: false }
).transition, 'unknown:999');
assert.equal(normalizeGrace(
  {
    fret: 3,
    isDead: false,
    dynamics: alphaTab.model.DynamicValue.F,
    slideOutType: alphaTab.model.SlideOutType.None,
    isHammerPullOrigin: false
  },
  { graceType: alphaTab.model.GraceType.BendGrace },
  { isPercussion: false }
).onBeat, 'bend-grace');
assert.equal(normalizeGrace(
  {
    fret: 3,
    isDead: false,
    dynamics: alphaTab.model.DynamicValue.F,
    slideOutType: alphaTab.model.SlideOutType.None,
    isHammerPullOrigin: false
  },
  { graceType: alphaTab.model.GraceType.None },
  { isPercussion: false }
).onBeat, 'none');

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
