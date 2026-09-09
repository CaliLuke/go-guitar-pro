import fs from 'node:fs';
import path from 'node:path';
import { pathToFileURL } from 'node:url';
import * as alphaTab from '@coderline/alphatab';

const oracle = JSON.parse(fs.readFileSync(new URL('./oracle.json', import.meta.url), 'utf8'));
alphaTab.Logger.logLevel = alphaTab.LogLevel.None;

export function enumName(values, value) {
  const name = values?.[value];
  return typeof name === 'string' ? name.toLowerCase() : String(value);
}

function finite(value) {
  return Number.isFinite(value) ? value : null;
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

function percussionInput(note, staff) {
  return note.percussionArticulation >= 0 && note.percussionArticulation < staff.track.percussionArticulations.length
    ? staff.track.percussionArticulations[note.percussionArticulation].id
    : null;
}

function normalizeGrace(note, beat, staff) {
  return {
    rawFret: finite(note.fret),
    dead: Boolean(note.isDead),
    onBeat: beat.graceType === alphaTab.model.GraceType.OnBeat,
    dynamic: enumName(alphaTab.model.DynamicValue, note.dynamics),
    transition: note.slideOutType !== alphaTab.model.SlideOutType.None
      ? 'slide'
      : note.isHammerPullOrigin ? 'hammer' : 'none',
    staffPercussion: Boolean(staff.isPercussion)
  };
}

function normalizeNote(note, staff, graces) {
  const isPercussion = Boolean(staff.isPercussion);
  let midi = finite(note.realValueWithoutHarmonic);
  if (isPercussion && note.percussionArticulation >= 0 && note.percussionArticulation < staff.track.percussionArticulations.length) {
    midi = staff.track.percussionArticulations[note.percussionArticulation].outputMidiNumber;
  }
  return {
    string: isPercussion ? note.string : staff.tuning.length - note.string + 1,
    fret: isPercussion ? null : finite(note.fret),
    percussionArticulation: note.percussionArticulation >= 0 ? note.percussionArticulation : null,
    percussionInput: isPercussion ? percussionInput(note, staff) : null,
    midi,
    kind: note.isDead ? 'dead' : note.isTieDestination ? 'tie' : 'normal',
    dynamic: enumName(alphaTab.model.DynamicValue, note.dynamics),
    tieOrigin: Boolean(note.tieDestination),
    tieDestination: Boolean(note.isTieDestination),
    effects: {
      accent: enumName(alphaTab.model.AccentuationType, note.accentuated),
      ghost: Boolean(note.isGhost),
      hammerOrigin: Boolean(note.isHammerPullOrigin),
      letRing: Boolean(note.isLetRing),
      palmMute: Boolean(note.isPalmMute),
      staccato: Boolean(note.isStaccato),
      vibrato: enumName(alphaTab.model.VibratoType, note.vibrato),
      harmonic: note.harmonicType === alphaTab.model.HarmonicType.None ? null : {
        kind: enumName(alphaTab.model.HarmonicType, note.harmonicType),
        fret: finite(note.harmonicValue)
      },
      bend: normalizeBend(note.bendPoints),
      trill: !Number.isFinite(note.trillValue) || note.trillValue < 0 ? null : {
        fret: note.trillValue,
        duration: finite(note.trillSpeed)
      },
      slides: [
        enumName(alphaTab.model.SlideInType, note.slideInType),
        enumName(alphaTab.model.SlideOutType, note.slideOutType)
      ].filter(value => value !== 'none')
    },
    graces
  };
}

function normalizeVoice(voice, staff) {
  const beats = [];
  let pendingGraceBeats = [];
  for (const beat of voice.beats) {
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
      start: finite(beat.displayStart),
      status: beat.isEmpty ? 'empty' : beat.isRest ? 'rest' : 'normal',
      duration: beat.duration,
      durationTicks: beat.displayDuration,
      dots: beat.dots,
      tuplet: beat.tupletNumerator > 0 && beat.tupletDenominator > 0
        ? [beat.tupletNumerator, beat.tupletDenominator]
        : [1, 1],
      dynamic: enumName(alphaTab.model.DynamicValue, beat.dynamics),
      text: beat.text ?? '',
      hairpin: enumName(alphaTab.model.CrescendoType, beat.crescendo),
      tremoloPicking: beat.tremoloPicking ? 1 << (beat.tremoloPicking.marks + 2) : null,
      notes
    });
  }
  for (const graceBeat of pendingGraceBeats) {
    beats.push({
      start: finite(graceBeat.displayStart),
      status: 'orphan-grace',
      duration: graceBeat.duration,
      durationTicks: graceBeat.displayDuration,
      dots: graceBeat.dots,
      tuplet: [1, 1],
      dynamic: enumName(alphaTab.model.DynamicValue, graceBeat.dynamics),
      text: graceBeat.text ?? '',
      hairpin: 'none',
      tremoloPicking: null,
      notes: graceBeat.notes.map(note => normalizeNote(note, staff, []))
    });
  }
  return { index: voice.index, beats };
}

function normalizeStaff(staff) {
  return {
    index: staff.index,
    percussion: Boolean(staff.isPercussion),
    standardNotationLineCount: staff.standardNotationLineCount,
    tuning: Array.from(staff.tuning ?? []),
    bars: staff.bars.map(bar => ({
      index: bar.index,
      clef: normalizeClef(bar.clef),
      voices: bar.voices.filter(voice => !voice.isEmpty).map(voice => normalizeVoice(voice, staff))
    }))
  };
}

function normalizeScore(score) {
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
      repeatStart: Boolean(masterBar.isRepeatStart),
      repeatCount: masterBar.repeatCount,
      alternateEndings: masterBar.alternateEndings,
      tripletFeel: masterBar.tripletFeel === alphaTab.model.TripletFeel.NoTripletFeel
        ? 'none'
        : enumName(alphaTab.model.TripletFeel, masterBar.tripletFeel).replace('tripletfeel', ''),
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

function main() {
  const fixture = process.argv[2];
  if (!fixture) {
    console.error('usage: node oracle.mjs FIXTURE');
    process.exit(2);
  }

  const settings = new alphaTab.Settings();
  Object.assign(settings.importer, oracle.importerSettings);
  const bytes = new Uint8Array(fs.readFileSync(path.resolve(fixture)));
  const score = alphaTab.importer.ScoreLoader.loadScoreFromBytes(bytes, settings);
  process.stdout.write(`${JSON.stringify(normalizeScore(score), null, 2)}\n`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  main();
}
