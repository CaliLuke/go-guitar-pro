import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import * as alphaTab from '@coderline/alphatab';

const oracle = JSON.parse(fs.readFileSync(new URL('./oracle.json', import.meta.url), 'utf8'));
alphaTab.Logger.logLevel = alphaTab.LogLevel.None;

export function enumName(values, value) {
  const name = values?.[value];
  return typeof name === 'string' ? name.toLowerCase() : `unknown:${value}`;
}

function finite(value) {
  return Number.isFinite(value) ? value : null;
}

export function normalizeLegacyDurationPercent(value) {
  if (!Number.isFinite(value) || value === 0 || Math.abs(value) >= 2.2250738585072014e-308) {
    return finite(value);
  }
  const bytes = new ArrayBuffer(8);
  const view = new DataView(bytes);
  view.setFloat64(0, value, false);
  return view.getFloat64(0, true);
}

function greatestCommonDivisor(left, right) {
  while (right !== 0n) {
    [left, right] = [right, left % right];
  }
  return left;
}

function exactAlphaTabDuration(beat) {
  if (!Number.isInteger(beat.duration) || beat.duration === 0) {
    return null;
  }
  let numerator = 3840n;
  let denominator = BigInt(beat.duration);
  if (beat.duration < 0) {
    numerator *= BigInt(-beat.duration);
    denominator = 1n;
  }
  if (beat.dots === 2) {
    numerator *= 7n;
    denominator *= 4n;
  } else if (beat.dots === 1) {
    numerator *= 3n;
    denominator *= 2n;
  }
  if (beat.tupletNumerator > 0 && beat.tupletDenominator > 0) {
    numerator *= BigInt(beat.tupletDenominator);
    denominator *= BigInt(beat.tupletNumerator);
  }
  const divisor = greatestCommonDivisor(numerator, denominator);
  return { numerator: numerator / divisor, denominator: denominator / divisor };
}

export function exactBeatStarts(beats, advances = () => true) {
  let current = { numerator: 0n, denominator: 1n };
  return beats.map(beat => {
    const start = Number(current.numerator / current.denominator);
    const duration = exactAlphaTabDuration(beat);
    if (duration && advances(beat)) {
      const numerator = current.numerator * duration.denominator + duration.numerator * current.denominator;
      const denominator = current.denominator * duration.denominator;
      const divisor = greatestCommonDivisor(numerator, denominator);
      current = { numerator: numerator / divisor, denominator: denominator / divisor };
    }
    return start;
  });
}

function normalizeArticulation(articulation) {
  return {
    elementName: articulation.elementType,
    inputMidiNumber: articulation.id,
    staffLine: articulation.staffLine,
    noteheadDefault: enumName(alphaTab.model.MusicFontSymbol, articulation.noteHeadDefault),
    noteheadHalf: enumName(alphaTab.model.MusicFontSymbol, articulation.noteHeadHalf),
    noteheadWhole: enumName(alphaTab.model.MusicFontSymbol, articulation.noteHeadWhole),
    techniquePlacement: enumName(alphaTab.model.TechniqueSymbolPlacement, articulation.techniqueSymbolPlacement),
    techniqueSymbol: enumName(alphaTab.model.MusicFontSymbol, articulation.techniqueSymbol),
    outputMidiNumber: articulation.outputMidiNumber
  };
}

export function normalizeAccent(value) {
  switch (value) {
    case alphaTab.model.AccentuationType.None:
      return 'none';
    case alphaTab.model.AccentuationType.Normal:
      return 'normal';
    case alphaTab.model.AccentuationType.Heavy:
      return 'heavy';
    case alphaTab.model.AccentuationType.Tenuto:
      return 'tenuto';
    default:
      return `unknown:${value}`;
  }
}

export function normalizeDynamic(value) {
  return enumName(alphaTab.model.DynamicValue, value);
}

export function normalizeHairpin(value) {
  return enumName(alphaTab.model.CrescendoType, value);
}

export function normalizeHarmonicKind(value) {
  return enumName(alphaTab.model.HarmonicType, value);
}

export function normalizeVibrato(value) {
  return enumName(alphaTab.model.VibratoType, value);
}

export function normalizeSlideIn(value) {
  switch (value) {
    case alphaTab.model.SlideInType.None:
      return 'none';
    case alphaTab.model.SlideInType.IntoFromBelow:
      return 'into-from-below';
    case alphaTab.model.SlideInType.IntoFromAbove:
      return 'into-from-above';
    default:
      return `unknown:${value}`;
  }
}

export function normalizeSlideOut(value) {
  switch (value) {
    case alphaTab.model.SlideOutType.None:
      return 'none';
    case alphaTab.model.SlideOutType.Shift:
      return 'shift';
    case alphaTab.model.SlideOutType.Legato:
      return 'legato';
    case alphaTab.model.SlideOutType.OutUp:
      return 'out-up';
    case alphaTab.model.SlideOutType.OutDown:
      return 'out-down';
    case alphaTab.model.SlideOutType.PickSlideDown:
      return 'pick-slide-down';
    case alphaTab.model.SlideOutType.PickSlideUp:
      return 'pick-slide-up';
    default:
      return `unknown:${value}`;
  }
}

export function normalizeTripletFeel(value) {
  switch (value) {
    case alphaTab.model.TripletFeel.NoTripletFeel:
      return 'none';
    case alphaTab.model.TripletFeel.Triplet16th:
      return 'triplet-16th';
    case alphaTab.model.TripletFeel.Triplet8th:
      return 'triplet-8th';
    case alphaTab.model.TripletFeel.Dotted16th:
      return 'dotted-16th';
    case alphaTab.model.TripletFeel.Dotted8th:
      return 'dotted-8th';
    case alphaTab.model.TripletFeel.Scottish16th:
      return 'scottish-16th';
    case alphaTab.model.TripletFeel.Scottish8th:
      return 'scottish-8th';
    default:
      return `unknown:${value}`;
  }
}

export function normalizeOttavia(value) {
  switch (value) {
    case alphaTab.model.Ottavia._15ma:
      return '15ma';
    case alphaTab.model.Ottavia._8va:
      return '8va';
    case alphaTab.model.Ottavia.Regular:
      return 'none';
    case alphaTab.model.Ottavia._8vb:
      return '8vb';
    case alphaTab.model.Ottavia._15mb:
      return '15mb';
    default:
      return `unknown:${value}`;
  }
}

export function normalizeDirection(value) {
  switch (value) {
    case alphaTab.model.Direction.TargetCoda:
      return 'Coda';
    case alphaTab.model.Direction.TargetDoubleCoda:
      return 'DoubleCoda';
    case alphaTab.model.Direction.TargetSegno:
      return 'Segno';
    case alphaTab.model.Direction.TargetSegnoSegno:
      return 'SegnoSegno';
    case alphaTab.model.Direction.TargetFine:
      return 'Fine';
    case alphaTab.model.Direction.JumpDaCapo:
      return 'DaCapo';
    case alphaTab.model.Direction.JumpDaCapoAlCoda:
      return 'DaCapoAlCoda';
    case alphaTab.model.Direction.JumpDaCapoAlDoubleCoda:
      return 'DaCapoAlDoubleCoda';
    case alphaTab.model.Direction.JumpDaCapoAlFine:
      return 'DaCapoAlFine';
    case alphaTab.model.Direction.JumpDalSegno:
      return 'DaSegno';
    case alphaTab.model.Direction.JumpDalSegnoAlCoda:
      return 'DaSegnoAlCoda';
    case alphaTab.model.Direction.JumpDalSegnoAlDoubleCoda:
      return 'DaSegnoAlDoubleCoda';
    case alphaTab.model.Direction.JumpDalSegnoAlFine:
      return 'DaSegnoAlFine';
    case alphaTab.model.Direction.JumpDalSegnoSegno:
      return 'DaSegnoSegno';
    case alphaTab.model.Direction.JumpDalSegnoSegnoAlCoda:
      return 'DaSegnoSegnoAlCoda';
    case alphaTab.model.Direction.JumpDalSegnoSegnoAlDoubleCoda:
      return 'DaSegnoSegnoAlDoubleCoda';
    case alphaTab.model.Direction.JumpDalSegnoSegnoAlFine:
      return 'DaSegnoSegnoAlFine';
    case alphaTab.model.Direction.JumpDaCoda:
      return 'DaCoda';
    case alphaTab.model.Direction.JumpDaDoubleCoda:
      return 'DaDoubleCoda';
    default:
      return `unknown:${value}`;
  }
}

export function normalizeBeatStatus(beat) {
  if (beat.isEmpty && beat.isRest) {
    return 'unknown:empty+rest';
  }
  return beat.isEmpty ? 'empty' : beat.isRest ? 'rest' : 'normal';
}

export function normalizeNoteKind(note) {
  if (note.isDead && note.isTieDestination) {
    return 'unknown:dead+tie';
  }
  return note.isDead ? 'dead' : note.isTieDestination ? 'tie' : 'normal';
}

export function normalizeTuning(tuning) {
  return Array.from(tuning ?? []);
}

export function normalizeAutomation(automation, bar) {
  return {
    bar,
    position: finite(automation.ratioPosition) ?? 0,
    type: enumName(alphaTab.model.AutomationType, automation.type),
    value: finite(automation.value),
    linear: Boolean(automation.isLinear)
  };
}

export function normalizeBend(points) {
  if (!points || points.length === 0) {
    return null;
  }
  return points.map(point => ({
    position: Math.round(finite(point.offset) * 1e9) / 1e9,
    value: point.value
  }));
}

export function normalizeWhammy(points) {
  if (!points || points.length === 0) {
    return null;
  }
  return points.map(point => ({
    position: Math.round(point.offset * 12 / 60 * 1e9) / 1e9,
    value: point.value
  }));
}

export function normalizeClef(value) {
  switch (value) {
    case alphaTab.model.Clef.F4:
      return 'bass';
    case alphaTab.model.Clef.C3:
      return 'alto';
    case alphaTab.model.Clef.C4:
      return 'tenor';
    case alphaTab.model.Clef.G2:
      return 'treble';
    case alphaTab.model.Clef.Neutral:
      return 'neutral';
    default:
      return `unknown:${value}`;
  }
}

export function normalizeSimileMark(value) {
	return enumName(alphaTab.model.SimileMark, value).replaceAll('ofdouble', '-of-double');
}

function percussionInput(note, staff) {
  return note.percussionArticulation >= 0 && note.percussionArticulation < staff.track.percussionArticulations.length
    ? staff.track.percussionArticulations[note.percussionArticulation].id
    : null;
}

export function normalizeNotePitch(note, staff) {
  const isPercussion = Boolean(staff.isPercussion);
  const isStringed = !isPercussion && staff.tuning.length > 0 && note.string > 0;
  let midi = finite(note.realValueWithoutHarmonic);
  if (isPercussion && note.percussionArticulation >= 0 && note.percussionArticulation < staff.track.percussionArticulations.length) {
    midi = staff.track.percussionArticulations[note.percussionArticulation].outputMidiNumber;
  }
  return {
    string: isPercussion ? note.string : isStringed ? staff.tuning.length - note.string + 1 : 0,
    fret: isStringed ? finite(note.fret) : null,
    percussionArticulation: note.percussionArticulation >= 0 ? note.percussionArticulation : null,
    percussionInput: isPercussion ? percussionInput(note, staff) : null,
    midi
  };
}

export function normalizeGrace(note, beat, staff) {
  let onBeat;
  switch (beat.graceType) {
    case alphaTab.model.GraceType.OnBeat:
      onBeat = true;
      break;
    case alphaTab.model.GraceType.BeforeBeat:
      onBeat = false;
      break;
    case alphaTab.model.GraceType.BendGrace:
      onBeat = 'bend-grace';
      break;
    case alphaTab.model.GraceType.None:
      onBeat = 'none';
      break;
    default:
      onBeat = `unknown:${beat.graceType}`;
  }
  const slide = normalizeSlideOut(note.slideOutType);
  const transition = slide.startsWith('unknown:')
    ? slide
    : slide !== 'none' ? 'slide' : note.isHammerPullOrigin ? 'hammer' : 'none';
  return {
    rawFret: finite(note.fret),
    dead: Boolean(note.isDead),
    onBeat,
    dynamic: normalizeDynamic(note.dynamics),
    transition,
    staffPercussion: Boolean(staff.isPercussion)
  };
}

function normalizeNote(note, staff, graces) {
  const pitch = normalizeNotePitch(note, staff);
  return {
    ...pitch,
    kind: normalizeNoteKind(note),
    dynamic: normalizeDynamic(note.dynamics),
    durationPercent: normalizeLegacyDurationPercent(note.durationPercent),
    tieOrigin: Boolean(note.tieDestination),
    tieDestination: Boolean(note.isTieDestination),
    effects: {
      accent: normalizeAccent(note.accentuated),
      ghost: Boolean(note.isGhost),
      hammerOrigin: Boolean(note.isHammerPullOrigin),
      letRing: Boolean(note.isLetRing),
      palmMute: Boolean(note.isPalmMute),
      staccato: Boolean(note.isStaccato),
      vibrato: normalizeVibrato(note.vibrato),
      harmonic: note.harmonicType === alphaTab.model.HarmonicType.None ? null : {
        kind: normalizeHarmonicKind(note.harmonicType),
        fret: finite(note.harmonicValue)
      },
      bend: normalizeBend(note.bendPoints),
      trill: !Number.isFinite(note.trillValue) || note.trillValue < 0 ? null : {
        fret: note.trillValue,
        duration: finite(note.trillSpeed)
      },
      slides: [
        normalizeSlideIn(note.slideInType),
        normalizeSlideOut(note.slideOutType)
      ].filter(value => value !== 'none')
    },
    graces
  };
}

function normalizeVoice(voice, staff) {
  const beats = [];
  let pendingGraceBeats = [];
  const exactStarts = exactBeatStarts(
    voice.beats,
    beat => beat.graceType === alphaTab.model.GraceType.None
  );
  for (let beatIndex = 0; beatIndex < voice.beats.length; beatIndex++) {
    const beat = voice.beats[beatIndex];
    if (beat.graceType !== alphaTab.model.GraceType.None) {
      pendingGraceBeats.push(beat);
      continue;
    }
    const notes = beat.notes.map((note, noteIndex) => {
      const matching = [];
      for (const graceBeat of pendingGraceBeats) {
        for (let graceIndex = 0; graceIndex < graceBeat.notes.length; graceIndex++) {
          const graceNote = graceBeat.notes[graceIndex];
          if (graceNote.string === note.string || (staff.isPercussion && graceIndex === noteIndex)) {
            matching.push(normalizeGrace(graceNote, graceBeat, staff));
          }
        }
      }
      return normalizeNote(note, staff, matching);
    });
    pendingGraceBeats = [];
    beats.push({
      start: exactStarts[beatIndex],
      status: normalizeBeatStatus(beat),
      graceRole: 'none',
      duration: beat.duration,
      durationTicks: beat.displayDuration,
      dots: beat.dots,
      tuplet: beat.tupletNumerator > 0 && beat.tupletDenominator > 0
        ? [beat.tupletNumerator, beat.tupletDenominator]
        : [1, 1],
      dynamic: normalizeDynamic(beat.dynamics),
      text: beat.text ?? '',
      octave: normalizeOttavia(beat.ottava),
      hairpin: normalizeHairpin(beat.crescendo),
      tremoloPicking: beat.tremoloPicking ? 1 << (beat.tremoloPicking.marks + 2) : null,
      whammy: normalizeWhammy(beat.whammyBarPoints),
      notes
    });
  }
  for (const graceBeat of pendingGraceBeats) {
    const beatIndex = voice.beats.indexOf(graceBeat);
    beats.push({
      start: exactStarts[beatIndex],
      status: normalizeBeatStatus(graceBeat),
      graceRole: 'orphan',
      duration: graceBeat.duration,
      durationTicks: graceBeat.displayDuration,
      dots: graceBeat.dots,
      tuplet: [1, 1],
      dynamic: normalizeDynamic(graceBeat.dynamics),
      text: graceBeat.text ?? '',
      octave: normalizeOttavia(graceBeat.ottava),
      hairpin: 'none',
      tremoloPicking: null,
      whammy: normalizeWhammy(graceBeat.whammyBarPoints),
      notes: graceBeat.notes.map(note => normalizeNote(note, staff, []))
    });
  }
  return { index: voice.index, beats };
}

function normalizeStaff(staff) {
  return {
    index: staff.index,
    capo: finite(staff.capo),
    percussion: Boolean(staff.isPercussion),
    standardNotationLineCount: staff.standardNotationLineCount,
    tuning: normalizeTuning(staff.tuning),
    bars: staff.bars.map(bar => ({
      index: bar.index,
      clef: normalizeClef(bar.clef),
	  clefOctave: normalizeOttavia(bar.clefOttava),
	  simileMark: normalizeSimileMark(bar.simileMark),
      voices: bar.voices.filter(voice => !voice.isEmpty).map(voice => normalizeVoice(voice, staff))
    }))
  };
}

export function normalizeScore(score) {
  const tempoAutomations = [];
  for (const masterBar of score.masterBars) {
    for (const automation of masterBar.tempoAutomations) {
      tempoAutomations.push(normalizeAutomation(automation, masterBar.index));
    }
  }
  return {
    schemaVersion: 1,
    metadata: {
      title: score.title ?? '',
      subtitle: score.subTitle ?? '',
      artist: score.artist ?? '',
      album: score.album ?? '',
      words: score.words ?? '',
      music: score.music ?? '',
      copyright: score.copyright ?? '',
      instructions: score.instructions ?? ''
    },
    masterBars: score.masterBars.map(masterBar => ({
      index: masterBar.index,
      start: masterBar.start,
      timeSignature: [masterBar.timeSignatureNumerator, masterBar.timeSignatureDenominator],
      freeTime: Boolean(masterBar.isFreeTime),
      repeatStart: Boolean(masterBar.isRepeatStart),
      repeatCount: masterBar.repeatCount,
      alternateEndings: masterBar.alternateEndings,
      tripletFeel: normalizeTripletFeel(masterBar.tripletFeel),
      directions: Array.from(masterBar.directions ?? []).map(normalizeDirection).sort(),
      pickup: Boolean(masterBar.isAnacrusis)
    })),
    tempoAutomations,
    tracks: score.tracks.map(track => ({
      index: track.index,
      name: track.name ?? '',
      program: track.playbackInfo.program,
      primaryChannel: track.playbackInfo.primaryChannel,
      percussionArticulations: track.percussionArticulations.map(normalizeArticulation),
      staves: track.staves.map(normalizeStaff)
    }))
  };
}

function loadScore(fixture) {
  const settings = new alphaTab.Settings();
  Object.assign(settings.importer, oracle.importerSettings);
  const bytes = new Uint8Array(fs.readFileSync(path.resolve(fixture)));
  return alphaTab.importer.ScoreLoader.loadScoreFromBytes(bytes, settings);
}

// Retained consumer controls, before the normalized comparison projection.
// Retained native consumer offsets (0..60), without comparison rounding.
export function loadAccidentalFacts(fixture) {
  const facts = [];
  for (const track of loadScore(fixture).tracks) for (const staff of track.staves)
    for (const bar of staff.bars) for (const voice of bar.voices) for (const beat of voice.beats)
      for (const note of beat.notes) facts.push({track:track.index,staff:staff.index,bar:bar.index,
        voice:voice.index,beat:beat.index,note:note.index,mode:note.accidentalMode,
        midi:note.realValueWithoutHarmonic,display:note.displayValueWithoutBend,
        fret:note.fret,string:note.string,articulation:note.percussionArticulation,
        percussionMidi:track.percussionArticulations[note.percussionArticulation]?.outputMidiNumber ?? null});
  return facts;
}

export function loadWhammyControls(fixture) {
  const facts = [];
  for (const track of loadScore(fixture).tracks) for (const staff of track.staves)
    for (const bar of staff.bars) for (const voice of bar.voices) for (const beat of voice.beats)
      if (beat.whammyBarPoints?.length) facts.push({track: track.index, staff: staff.index,
        bar: bar.index, voice: voice.index, beat: beat.index,
        points: beat.whammyBarPoints.map(p => ({offset: p.offset, value: p.value}))});
  return facts;
}

export function loadSystemLayout(fixture) {
  const score = loadScore(fixture);
  return { default: score.defaultSystemsLayout, systems: Array.from(score.systemsLayout),
    masters: score.masterBars.map(bar => bar.displayScale),
    tracks: score.tracks.map(track => ({ default: track.defaultSystemsLayout, systems: Array.from(track.systemsLayout),
      scales: track.staves.map(staff => staff.bars.map(bar => bar.displayScale)) })) };
}

export function loadCurveGraceFacts(fixture) {
  const facts = [];
  for (const track of loadScore(fixture).tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          for (const beat of voice.beats) {
            for (const note of beat.notes) {
              if (!note.bendPoints && beat.graceType === 0 && (!beat.previousBeat || beat.previousBeat.graceType === 0)) continue;
              facts.push({track: track.index, staff: staff.index, bar: bar.index,
                voice: voice.index, beat: beat.index, note: note.index,
                graceType: beat.graceType, string: note.string, fret: note.fret,
                duration: beat.duration, dynamic: note.dynamics,
                bend: (note.bendPoints ?? []).map(p => ({offset: p.offset, value: p.value})),
                slide: note.slideOutType, hammer: note.isHammerPullOrigin});
            }
          }
        }
      }
    }
  }
  return facts;
}

// Raw GPIF consumer facts: durationPercent bypasses the legacy endian correction.
export function loadNoteDurationTrillFacts(fixture) {
  const facts = [];
  for (const track of loadScore(fixture).tracks) for (const staff of track.staves)
    for (const bar of staff.bars) for (const voice of bar.voices) for (const beat of voice.beats)
      for (const note of beat.notes) facts.push({track: track.index, staff: staff.index,
        bar: bar.index, voice: voice.index, beat: beat.index, note: note.index,
        duration: beat.duration, dots: beat.dots, numerator: beat.tupletNumerator,
        denominator: beat.tupletDenominator, start: beat.playbackStart, length: beat.playbackDuration,
        fret: note.fret, string: note.string, percent: note.durationPercent,
        tieOrigin: note.isTieOrigin, tieDestination: note.isTieDestination,
        letRing: note.isLetRing, palmMute: note.isPalmMute, staccato: note.isStaccato,
        trillFret: note.trillValue, trillDuration: note.trillSpeed});
  return facts;
}

export function loadNormalizedScore(fixture) {
  return normalizeScore(loadScore(fixture));
}

export function loadVoiceRestFacts(fixture) {
  const score = loadScore(fixture);
  return score.tracks.map(track => track.staves.map(staff => staff.bars.map(bar =>
    bar.voices.map(voice => ({index: voice.index, empty: voice.isEmpty, beats: voice.beats.map(beat => ({
      empty: beat.isEmpty, rest: beat.isRest, duration: beat.duration, dots: beat.dots,
      numerator: beat.tupletNumerator, denominator: beat.tupletDenominator,
      start: beat.playbackStart, length: beat.playbackDuration, text: beat.text, notes: beat.notes.length
    }))})))));
}

export function loadDoubleBarFacts(fixture) {
  const score = loadScore(fixture);
  return {
    flags: score.masterBars.map(bar => bar.isDoubleBar),
    tracks: score.tracks.map(track => track.staves.map(staff => staff.bars.map(bar => ({
      requested: alphaTab.model.BarLineStyle[bar.barLineRight],
      actual: alphaTab.model.BarLineStyle[bar.getActualBarLineRight()]
    }))))
  };
}

export function loadVolumeAutomationFacts(fixture) {
  const facts = [];
  const score = loadScore(fixture);
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          for (const beat of voice.beats) {
            for (const automation of beat.automations) {
              if (automation.type === alphaTab.model.AutomationType.Volume) {
                facts.push({ track: track.index, staff: staff.index, voice: voice.index,
                  beat: beat.index, ...normalizeAutomation(automation, bar.index) });
              }
            }
          }
        }
      }
    }
  }
  return facts;
}

export function loadAutomationFacts(fixture) {
  const score = loadScore(fixture);
  const detail = (automation, bar) => ({
    ...normalizeAutomation(automation, bar),
    text: automation.text ?? '',
    visible: Boolean(automation.isVisible)
  });
  const tempo = [];
  for (const masterBar of score.masterBars) {
    for (const automation of masterBar.tempoAutomations) {
      tempo.push(detail(automation, masterBar.index));
    }
  }
  const sound = [];
  for (const track of score.tracks) {
    const staff = track.staves[0];
    if (!staff) continue;
    for (const bar of staff.bars) {
      for (const voice of bar.voices) {
        for (const beat of voice.beats) {
          for (const automation of beat.automations) {
            if (automation.type === alphaTab.model.AutomationType.Instrument) {
              sound.push({ track: track.index, ...detail(automation, bar.index) });
            }
          }
        }
      }
    }
  }
  return { tempo, sound };
}

export function loadBarreFacts(fixture) {
  const score = loadScore(fixture);
  const facts = [];
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          for (const beat of voice.beats) {
            if (beat.barreFret < 0 && beat.barreShape === alphaTab.model.BarreShape.None) continue;
            facts.push({
              track: track.index,
              staff: staff.index,
              bar: bar.index,
              voice: voice.index,
              beat: beat.index,
              fret: finite(beat.barreFret),
              shape: enumName(alphaTab.model.BarreShape, beat.barreShape)
            });
          }
        }
      }
    }
  }
  return facts;
}

export function loadChordDiagramFacts(fixture) {
  const score = loadScore(fixture);
  const facts = [];
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          for (const beat of voice.beats) {
            if (!beat.chord) continue;
            facts.push({
              track: track.index,
              staff: staff.index,
              bar: bar.index,
              voice: voice.index,
              beat: beat.index,
              name: beat.chord.name ?? '',
              firstFret: finite(beat.chord.firstFret),
              strings: Array.from(beat.chord.strings),
              barreFrets: Array.from(beat.chord.barreFrets)
            });
          }
        }
      }
    }
  }
  return facts;
}

export function loadBeatTechniqueFacts(fixture) {
  const score = loadScore(fixture);
  const facts = [];
  for (const track of score.tracks) for (const staff of track.staves)
    for (const bar of staff.bars) for (const voice of bar.voices)
      for (const beat of voice.beats) facts.push({
        track: track.index, staff: staff.index, bar: bar.index,
        voice: voice.index, beat: beat.index, rasgueado: beat.rasgueado,
        tap: beat.tap, slap: beat.slap, pop: beat.pop, wah: beat.wahPedal,
        leftHandTapped: beat.notes.map(note => note.isLeftHandTapped)
      });
  return facts;
}

export function loadMetadataTextFacts(fixture) {
  const score = loadScore(fixture);
  return {
    title: score.title, subtitle: score.subTitle, artist: score.artist,
    album: score.album, words: score.words, music: score.music,
    copyright: score.copyright, tabber: score.tab,
    instructions: score.instructions, notices: score.notices
  };
}

// Observe the raw lines delivered to the pinned consumer's normal dispatcher.
// Call the original method so the final beat facts remain independent evidence.
export function loadAssignedLyricFacts(fixture) {
  const lines = [];
  const apply = alphaTab.model.Track.prototype.applyLyrics;
  let score;
  alphaTab.model.Track.prototype.applyLyrics = function (lyrics) {
    lines.push({track: this.index, lines: lyrics.map(line => ({text: line.text, offset: line.startBar}))});
    return apply.call(this, lyrics);
  };
  try { score = loadScore(fixture); }
  finally { alphaTab.model.Track.prototype.applyLyrics = apply; }
  return {lines, beats: beatLyricFacts(score)};
}

export function loadBeatLyricFacts(fixture) {
  return beatLyricFacts(loadScore(fixture));
}

function beatLyricFacts(score) {
  const facts = [];
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          for (const beat of voice.beats) {
            if (beat.lyrics === null || beat.lyrics === undefined) continue;
            facts.push({
              track: track.index,
              staff: staff.index,
              bar: bar.index,
              voice: voice.index,
              beat: beat.index,
              lyrics: Array.from(beat.lyrics),
              text: beat.text ?? ''
            });
          }
        }
      }
    }
  }
  return facts;
}

export function loadDeadSlapFacts(fixture) {
  const score = loadScore(fixture);
  const facts = [];
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          for (const beat of voice.beats) {
            if (!beat.deadSlapped) continue;
            facts.push({
              track: track.index,
              staff: staff.index,
              bar: bar.index,
              voice: voice.index,
              beat: beat.index,
              deadSlapped: Boolean(beat.deadSlapped),
              isEmpty: Boolean(beat.isEmpty),
              isRest: Boolean(beat.isRest),
              noteCount: beat.notes.length
            });
          }
        }
      }
    }
  }
  return facts;
}

export function loadBeamingFacts(fixture) {
  const score = loadScore(fixture);
  const masterBars = [];
  const beats = [];
  for (const masterBar of score.masterBars) {
    if (masterBar.beamingRules) {
      masterBars.push({
        bar: masterBar.index,
        rules: Array.from(masterBar.beamingRules.groups, ([duration, groups]) => ({
          duration: finite(duration),
          groups: Array.from(groups)
        }))
      });
    }
  }
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          for (const beat of voice.beats) {
            if (beat.beamingMode === alphaTab.model.BeatBeamingMode.Auto &&
                !beat.invertBeamDirection && beat.preferredBeamDirection === null) continue;
            beats.push({
              track: track.index,
              staff: staff.index,
              bar: bar.index,
              voice: voice.index,
              beat: beat.index,
              mode: enumName(alphaTab.model.BeatBeamingMode, beat.beamingMode),
              invert: Boolean(beat.invertBeamDirection),
              direction: beat.preferredBeamDirection === null
                ? 'none'
                : enumName(alphaTab.rendering.BeamDirection, beat.preferredBeamDirection)
            });
          }
        }
      }
    }
  }
  return { masterBars, beats };
}

export function loadBackingTrackFacts(fixture) {
  const backingTrack = loadScore(fixture).backingTrack;
  return {
    enabled: Boolean(backingTrack),
    audioBytes: backingTrack?.rawAudioFile ? Array.from(backingTrack.rawAudioFile) : []
  };
}

export function loadFermataFacts(fixture) {
  const score = loadScore(fixture);
  const normalizeFermata = fermata => ({
    type: enumName(alphaTab.model.FermataType, fermata.type),
    length: finite(fermata.length)
  });
  return score.masterBars.map(masterBar => {
    const authored = Array.from(masterBar.fermata ?? [], ([offset, fermata]) => ({
      offset,
      ...normalizeFermata(fermata)
    }));
    const derivedBeats = [];
    for (const track of score.tracks) {
      for (const staff of track.staves) {
        const bar = staff.bars[masterBar.index];
        if (!bar) continue;
        for (const voice of bar.voices) {
          for (const beat of voice.beats) {
            if (beat.fermata) {
              derivedBeats.push({
                track: track.index,
                staff: staff.index,
                voice: voice.index,
                beat: beat.index,
                offset: beat.playbackStart,
                ...normalizeFermata(beat.fermata)
              });
            }
          }
        }
      }
    }
    return { authored, derivedBeats };
  });
}

export function loadKeyFacts(fixture) {
  const score = loadScore(fixture);
  const bars = score.tracks[0]?.staves[0]?.bars ?? [];
  return bars.map(bar => ({
    accidentalCount: finite(bar.keySignature),
    mode: enumName(alphaTab.model.KeySignatureType, bar.keySignatureType)
  }));
}

export function loadLegatoFacts(fixture) {
  const facts = [];
  const score = loadScore(fixture);
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          for (let beatIndex = 0; beatIndex < voice.beats.length; beatIndex++) {
            const beat = voice.beats[beatIndex];
            const origin = Boolean(beat.isLegatoOrigin);
            const destination = Boolean(beat.isLegatoDestination);
            if (origin || destination) {
              facts.push({
                track: track.index,
                staff: staff.index,
                bar: bar.index,
                voice: voice.index,
                beat: beatIndex,
                origin,
                destination
              });
            }
          }
        }
      }
    }
  }
  return facts;
}

// Preserve raw playback fields; the broad semantic projection omits these limits.
export function loadLegacyInstrumentFacts(fixture) {
  return loadScore(fixture).tracks.map(track => ({
    program: track.playbackInfo.program,
    volume: track.playbackInfo.volume,
    balance: track.playbackInfo.balance,
    primaryChannel: track.playbackInfo.primaryChannel,
    secondaryChannel: track.playbackInfo.secondaryChannel,
    staves: track.staves.map(staff => ({tuning: Array.from(staff.tuning), capo: staff.capo}))
  }));
}

export function loadPlaybackRoutingFacts(fixture) {
  return loadScore(fixture).tracks.map(track => ({
    track: track.index,
    port: track.playbackInfo.port,
    primaryChannel: track.playbackInfo.primaryChannel,
    secondaryChannel: track.playbackInfo.secondaryChannel,
    isMute: track.playbackInfo.isMute,
    isSolo: track.playbackInfo.isSolo
  }));
}

export function loadMidiBankFacts(fixture) {
  const score = loadScore(fixture);
  return score.tracks.map(track => {
    const automations = [];
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          for (const beat of voice.beats) {
            for (const automation of beat.automations) {
              if (automation.type === alphaTab.model.AutomationType.Bank ||
                  automation.type === alphaTab.model.AutomationType.Instrument) {
                automations.push(normalizeAutomation(automation, bar.index));
              }
            }
          }
        }
      }
    }
    return {
      track: track.index,
      bank: finite(track.playbackInfo.bank),
      program: finite(track.playbackInfo.program),
      automations
    };
  });
}

export function loadSustainPedalFacts(fixture) {
  const score = loadScore(fixture);
  const facts = [];
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        if (bar.sustainPedals.length === 0) continue;
        facts.push({
          track: track.index,
          staff: staff.index,
          bar: bar.index,
          markers: bar.sustainPedals.map(marker => ({
            position: finite(marker.ratioPosition),
            type: marker.pedalType === alphaTab.model.SustainPedalMarkerType.Up
              ? 'release'
              : enumName(alphaTab.model.SustainPedalMarkerType, marker.pedalType)
          }))
        });
      }
    }
  }
  return facts;
}

export function loadTremoloPickingFacts(fixture) {
  const score = loadScore(fixture);
  const facts = [];
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          for (const beat of voice.beats) {
            if (!beat.tremoloPicking) continue;
            facts.push({
              track: track.index,
              staff: staff.index,
              bar: bar.index,
              voice: voice.index,
              beat: beat.index,
              marks: finite(beat.tremoloPicking.marks),
              style: enumName(alphaTab.model.TremoloPickingStyle, beat.tremoloPicking.style)
            });
          }
        }
      }
    }
  }
  return facts;
}

export function loadTremoloPickingModelFacts() {
  const effect = new alphaTab.model.TremoloPickingEffect();
  return {
    minMarks: alphaTab.model.TremoloPickingEffect.minMarks,
    maxMarks: alphaTab.model.TremoloPickingEffect.maxMarks,
    defaultMarks: effect.marks,
    defaultStyle: enumName(alphaTab.model.TremoloPickingStyle, effect.style),
    buzzRollStyle: enumName(alphaTab.model.TremoloPickingStyle, alphaTab.model.TremoloPickingStyle.BuzzRoll)
  };
}

export function loadTranspositionFacts(fixture) {
  const score = loadScore(fixture);
  const facts = [];
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      const notes = staff.bars.flatMap(bar => bar.voices.flatMap(voice => voice.beats.flatMap(beat => beat.notes)));
      const firstNote = notes[0] ?? null;
      facts.push({
        track: track.index,
        staff: staff.index,
        transpositionPitch: finite(staff.transpositionPitch),
        displayTranspositionPitch: finite(staff.displayTranspositionPitch),
        keys: staff.bars.map(bar => finite(bar.keySignature)),
        firstNoteString: firstNote ? finite(firstNote.string) : 0,
        firstNoteFret: firstNote ? finite(firstNote.fret) : 0,
        firstNoteSoundingMidi: firstNote ? finite(firstNote.realValue) : 0
      });
    }
  }
  return facts;
}

export function loadTuningFacts(fixture) {
  const score = loadScore(fixture);
  const facts = [];
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      facts.push({
        track: track.index,
        staff: staff.index,
        name: staff.tuningName ?? '',
        tuning: normalizeTuning(staff.tuning),
        capo: finite(staff.capo)
      });
    }
  }
  return facts;
}

export function loadBeatVibratoFacts(fixture) {
  const score = loadScore(fixture);
  const facts = [];
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          let regularBeat = 0;
          for (const beat of voice.beats) {
            if (beat.graceType !== alphaTab.model.GraceType.None) continue;
            if (beat.vibrato !== alphaTab.model.VibratoType.None) {
              facts.push({
                track: track.index,
                staff: staff.index,
                bar: bar.index,
                voice: voice.index,
                beat: regularBeat,
                vibrato: normalizeVibrato(beat.vibrato)
              });
            }
            regularBeat++;
          }
        }
      }
    }
  }
  return facts;
}

export function loadBeatFadeFacts(fixture) {
  const score = loadScore(fixture);
  const facts = [];
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          let regularBeat = 0;
          for (const beat of voice.beats) {
            if (beat.graceType !== alphaTab.model.GraceType.None) continue;
            if (beat.fade !== alphaTab.model.FadeType.None) {
              facts.push({
                track: track.index,
                staff: staff.index,
                bar: bar.index,
                voice: voice.index,
                beat: regularBeat,
                fade: enumName(alphaTab.model.FadeType, beat.fade)
              });
            }
            regularBeat++;
          }
        }
      }
    }
  }
  return facts;
}

export function loadGolpeFacts(fixture) {
  const score = loadScore(fixture);
  const facts = [];
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          let regularBeat = 0;
          for (const beat of voice.beats) {
            if (beat.graceType !== alphaTab.model.GraceType.None) continue;
            if (beat.golpe !== alphaTab.model.GolpeType.None) {
              facts.push({
                track: track.index,
                staff: staff.index,
                bar: bar.index,
                voice: voice.index,
                beat: regularBeat,
                golpe: enumName(alphaTab.model.GolpeType, beat.golpe),
                noteCount: beat.notes.length
              });
            }
            regularBeat++;
          }
        }
      }
    }
  }
  return facts;
}

export function loadStaffNotationFacts(fixture) {
  return loadScore(fixture).tracks.map(track => ({
    visible: track.isVisibleOnMultiTrack,
    staves: track.staves.map(staff => ({
      standard: staff.showStandardNotation, tablature: staff.showTablature,
      slash: staff.showSlash, numbered: staff.showNumbered,
      tuning: Array.from(staff.tuning),
      beats: staff.bars.flatMap(bar => bar.voices.flatMap(voice => voice.beats.map(beat => ({
        slashed: beat.slashed,
        notes: beat.notes.map(note => ({string: note.string, fret: note.fret, MIDI: note.realValue}))
      }))))
    }))
  }));
}

export function loadStringNumberFacts(fixture) {
  const facts = [];
  for (const track of loadScore(fixture).tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          for (let beatIndex = 0; beatIndex < voice.beats.length; beatIndex++) {
            const beat = voice.beats[beatIndex];
            for (let noteIndex = 0; noteIndex < beat.notes.length; noteIndex++) {
              const note = beat.notes[noteIndex];
              facts.push({track: track.index, staff: staff.index, bar: bar.index, voice: voice.index, beat: beatIndex, note: noteIndex, string: note.string, fret: note.fret, MIDI: note.realValue, show: note.showStringNumber});
            }
          }
        }
      }
    }
  }
  return facts;
}

export function loadFingeringFacts(fixture) {
  const score = loadScore(fixture);
  const facts = [];
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          let regularBeat = 0;
          for (const beat of voice.beats) {
            if (beat.graceType !== alphaTab.model.GraceType.None) continue;
            for (const note of beat.notes) {
              if (note.leftHandFinger === alphaTab.model.Fingers.Unknown && note.rightHandFinger === alphaTab.model.Fingers.Unknown) continue;
              facts.push({
                track: track.index,
                staff: staff.index,
                bar: bar.index,
                voice: voice.index,
                beat: regularBeat,
                note: note.index,
                left: enumName(alphaTab.model.Fingers, note.leftHandFinger),
                right: enumName(alphaTab.model.Fingers, note.rightHandFinger)
              });
            }
            regularBeat++;
          }
        }
      }
    }
  }
  return facts;
}

export function loadPickStrokeFacts(fixture) {
  const score = loadScore(fixture);
  const facts = [];
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          let regularBeat = 0;
          for (const beat of voice.beats) {
            if (beat.graceType !== alphaTab.model.GraceType.None) continue;
            facts.push({
              track: track.index, staff: staff.index, bar: bar.index,
              voice: voice.index, beat: regularBeat,
              pickStroke: enumName(alphaTab.model.PickStroke, beat.pickStroke),
              brushType: enumName(alphaTab.model.BrushType, beat.brushType)
            });
            regularBeat++;
          }
        }
      }
    }
  }
  return facts;
}

export function loadBrushFacts(fixture) {
  const score = loadScore(fixture);
  const facts = [];
  const brushType = alphaTab.model.BrushType;
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          let regularBeat = 0;
          for (const beat of voice.beats) {
            if (beat.graceType !== alphaTab.model.GraceType.None) continue;
            if (beat.brushType !== brushType.None) {
              facts.push({
                track: track.index,
                staff: staff.index,
                bar: bar.index,
                voice: voice.index,
                beat: regularBeat,
                kind: beat.brushType === brushType.BrushUp || beat.brushType === brushType.BrushDown ? 'brush' : 'arpeggio',
                direction: beat.brushType === brushType.BrushUp || beat.brushType === brushType.ArpeggioUp ? 'up' : 'down',
                duration: finite(beat.brushDuration)
              });
            }
            regularBeat++;
          }
        }
      }
    }
  }
  return facts;
}

export function loadSectionTrackNameFacts(fixture) {
  const rawNames = [];
  const finish = alphaTab.model.Track.prototype.finish;
  alphaTab.model.Track.prototype.finish = function(...args) {
    rawNames[this.index] = this.shortName;
    return finish.apply(this, args);
  };
  let score;
  try { score = loadScore(fixture); }
  finally { alphaTab.model.Track.prototype.finish = finish; }
  return {
    tracks: score.tracks.map(track => ({ name: track.name, rawShortName: rawNames[track.index], shortName: track.shortName })),
    sections: score.masterBars.map(bar => bar.section ? { letter: bar.section.marker, text: bar.section.text } : null)
  };
}

export function loadHarmonicFacts(fixture) {
  return loadScore(fixture).tracks.flatMap(track => track.staves.flatMap(staff => staff.bars.flatMap(bar => bar.voices.flatMap(voice => voice.beats.flatMap(beat => beat.notes.filter(note => note.harmonicType !== alphaTab.model.HarmonicType.None).map(note => ({kind: alphaTab.model.HarmonicType[note.harmonicType], fret: note.harmonicValue})))))));
}

export function loadNoteOrnamentFacts(fixture) {
 const facts=[];
 for(const track of loadScore(fixture).tracks)for(const staff of track.staves)for(const bar of staff.bars)for(const voice of bar.voices)for(const beat of voice.beats)for(const note of beat.notes)facts.push({track:track.index,staff:staff.index,bar:bar.index,voice:voice.index,beat:beat.index,note:note.index,ornament:alphaTab.model.NoteOrnament[note.ornament],string:note.string,fret:note.fret});
 return facts;
}

function main() {
  const args = process.argv.slice(2);
  if(args[0] === "--note-ornaments" && args.length===2){process.stdout.write(JSON.stringify(loadNoteOrnamentFacts(args[1])));return;}
  if (args[0] === "--harmonics" && args.length === 2) { process.stdout.write(JSON.stringify(loadHarmonicFacts(args[1]))); return; }
  if (args.length === 0) {
    console.error('usage: node oracle.mjs FIXTURE | --batch FIXTURE...');
    process.exit(2);
  }
  if (args[0] === '--system-layout' && args.length === 2) {
    process.stdout.write(JSON.stringify(loadSystemLayout(args[1])));
    return;
  }
  if (args[0] === '--batch-receipts') {
    process.stdout.write(`${JSON.stringify(args.slice(1).map(fixture => {
      try {
        return { fixture, score: loadNormalizedScore(fixture) };
      } catch (error) {
        return { fixture, score: null, error: { type: error.constructor.name, message: error.message } };
      }
    }))}\n`);
    return;
  }
  if (args[0] === '--section-track-names' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadSectionTrackNameFacts(args[1]))}\n`);
    return;
  }
  if (args[0] === '--accidental-facts' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadAccidentalFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--whammy-controls' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadWhammyControls(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--curve-grace' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadCurveGraceFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--note-duration-trill' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadNoteDurationTrillFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--batch') {
    const fixtures = args.slice(1);
    process.stdout.write(`${JSON.stringify(fixtures.map(fixture => ({
      fixture,
      score: loadNormalizedScore(fixture)
    })))}\n`);
    return;
  }
  if (args[0] === '--sync-points' && args.length === 2) { process.stdout.write(JSON.stringify(loadSyncPointFacts(args[1]))); return; }
  if (args[0] === '--pan-initial-balances' && args.length === 2) { process.stdout.write(JSON.stringify(loadScore(args[1]).tracks.map(t => t.playbackInfo.balance))); return; }
  if (args[0] === '--pan-automations' && args.length === 2) { process.stdout.write(JSON.stringify(loadPanFacts(args[1]))); return; }
  if (args[0] === '--volume-automations' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadVolumeAutomationFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--automations' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadAutomationFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--backing-track' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadBackingTrackFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--voice-rests' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadVoiceRestFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--double-bars' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadDoubleBarFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--barre' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadBarreFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--beat-timer' && args.length === 2) { process.stdout.write(`${JSON.stringify(loadBeatTimerFacts(args[1]), null, 2)}\n`); return; }
  if (args[0] === '--chord-display' && args.length === 2) { process.stdout.write(`${JSON.stringify(loadChordDisplayFacts(args[1]), null, 2)}\n`); return; }
  if (args[0] === '--chord-diagrams' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadChordDiagramFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--metadata-text' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadMetadataTextFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--assigned-lyrics' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadAssignedLyricFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--beat-techniques' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadBeatTechniqueFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--beat-lyrics' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadBeatLyricFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--beat-vibrato' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadBeatVibratoFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--beat-fade' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadBeatFadeFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--golpe' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadGolpeFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--staff-notation' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadStaffNotationFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--multi-rest' && args.length === 2) { process.stdout.write(`${JSON.stringify(loadMultiRestFacts(args[1]), null, 2)}\n`); return; }
  if (args[0] === '--score-style' && args.length === 2) { process.stdout.write(`${JSON.stringify(loadScoreStyleFacts(args[1]), null, 2)}\n`); return; }
  if (args[0] === '--string-number-display' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadStringNumberFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--fingering' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadFingeringFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--dead-slap' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadDeadSlapFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--pick-stroke' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadPickStrokeFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--brush' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadBrushFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--beaming' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadBeamingFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--fermatas' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadFermataFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--keys' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadKeyFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--legato' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadLegatoFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--legacy-instrument' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadLegacyInstrumentFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--playback-routing' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadPlaybackRoutingFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--midi-bank' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadMidiBankFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--sustain-pedals' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadSustainPedalFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--tremolo-picking' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadTremoloPickingFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--tremolo-picking-model' && args.length === 1) {
    process.stdout.write(`${JSON.stringify(loadTremoloPickingModelFacts(), null, 2)}\n`);
    return;
  }
  if (args[0] === '--transposition' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadTranspositionFacts(args[1]), null, 2)}\n`);
    return;
  }
  if (args[0] === '--tuning-labels' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadTuningFacts(args[1]), null, 2)}\n`);
    return;
  }
  process.stdout.write(`${JSON.stringify(loadNormalizedScore(args[0]), null, 2)}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main();
}

export function loadSyncPointFacts(fixture) {
  const score = loadScore(fixture);
  return score.masterBars.flatMap(bar => (bar.syncPoints ?? []).map(a => ({
    bar: bar.index, position: a.ratioPosition, occurrence: a.syncPointValue.barOccurence,
    milliseconds: a.syncPointValue.millisecondOffset, linear: a.isLinear, visible: a.isVisible,
    modifiedTempoPresent: Object.hasOwn(a.syncPointValue, 'modifiedTempo'),
    originalTempoPresent: Object.hasOwn(a.syncPointValue, 'originalTempo')
  })));
}

export function loadPanFacts(fixture) {
  const score = loadScore(fixture);
  return score.tracks.flatMap(track => track.staves.flatMap(staff => staff.bars.flatMap(bar => bar.voices.flatMap(voice => voice.beats.flatMap(beat => beat.automations.filter(a => a.type === alphaTab.model.AutomationType.Balance).map(a => ({track:track.index,bar:bar.index,beat:beat.index,value:a.value,linear:a.isLinear})))))));
}

export function loadChordDisplayFacts(fixture) {
 const score=loadScore(fixture);
 return score.tracks.flatMap(track=>track.staves.flatMap(staff=>staff.bars.flatMap(bar=>bar.voices.flatMap(voice=>voice.beats.filter(beat=>beat.chord).map(beat=>({track:track.index,staff:staff.index,bar:bar.index,voice:voice.index,beat:beat.index,name:beat.chord.name,showName:beat.chord.showName,showDiagram:beat.chord.showDiagram,showFingering:beat.chord.showFingering,strings:Array.from(beat.chord.strings)}))))));
}

export function loadBeatTimerFacts(fixture) {
 const score=loadScore(fixture);
 return score.tracks.flatMap(track=>track.staves.flatMap(staff=>staff.bars.flatMap(bar=>bar.voices.flatMap(voice=>voice.beats.map(beat=>({track:track.index,staff:staff.index,bar:bar.index,voice:voice.index,beat:beat.index,show:beat.showTimer,milliseconds:beat.timer,notes:beat.notes.map(note=>({string:note.string,fret:note.fret,midi:note.realValue}))}))))));
}

export function loadScoreStyleFacts(fixture) {
 const score=loadScore(fixture),s=score.stylesheet;
 const other={};for(const key of ['hideDynamics','bracketExtendMode','useSystemSignSeparator','globalDisplayTuning','globalDisplayChordDiagramsOnTop','globalDisplayChordDiagramsInScore','singleTrackTrackNamePolicy','multiTrackTrackNamePolicy','firstSystemTrackNameMode','otherSystemsTrackNameMode','firstSystemTrackNameOrientation','otherSystemsTrackNameOrientation'])other[key]=s[key];
 const headers=Array.from(score.style?.headerAndFooter??[]).map(([key,value])=>({element:alphaTab.model.ScoreSubElement[key],template:value.template,visible:value.isVisible,align:value.textAlign}));
 return {extended:s.extendBarLines,numbers:alphaTab.model.BarNumberDisplay[s.barNumberDisplay],other,headers};
}

export function loadMultiRestFacts(fixture) {
 const score=loadScore(fixture);
 return {global:score.stylesheet.multiTrackMultiBarRest,tracks:score.stylesheet.perTrackMultiBarRest===null?null:Array.from(score.stylesheet.perTrackMultiBarRest),measureCounts:score.tracks.map(t=>t.staves.map(s=>s.bars.length)),notation:loadStaffNotationFacts(fixture)};
}
