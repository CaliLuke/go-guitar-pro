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
  // Dead-slap changes AlphaTab's playback/display rest predicate, but it does
  // not turn the source beat into an authored note-bearing beat.
  if (beat.deadSlapped && beat.notes.length === 0) {
    return 'rest';
  }
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
    position: Math.round(point.offset * 12 / 60),
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

export function loadNormalizedScore(fixture) {
  return normalizeScore(loadScore(fixture));
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

export function loadBeatLyricFacts(fixture) {
  const score = loadScore(fixture);
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

function main() {
  const args = process.argv.slice(2);
  if (args.length === 0) {
    console.error('usage: node oracle.mjs FIXTURE | --batch FIXTURE...');
    process.exit(2);
  }
  if (args[0] === '--batch') {
    const fixtures = args.slice(1);
    process.stdout.write(`${JSON.stringify(fixtures.map(fixture => ({
      fixture,
      score: loadNormalizedScore(fixture)
    })))}\n`);
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
  if (args[0] === '--barre' && args.length === 2) {
    process.stdout.write(`${JSON.stringify(loadBarreFacts(args[1]), null, 2)}\n`);
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
