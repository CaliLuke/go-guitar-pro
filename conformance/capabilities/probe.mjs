// Development audit: compare authored AlphaTab values before and after Go GP8 export.
// Differences are candidates for review, not automatically accepted semantic losses.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import crypto from 'node:crypto';
import * as alphaTab from '@coderline/alphatab';

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, '../..');
const oracle = JSON.parse(fs.readFileSync(path.join(root, 'conformance/oracle.json')));
const catalog = JSON.parse(fs.readFileSync(path.join(here, 'catalog.json')));
alphaTab.Logger.logLevel = alphaTab.LogLevel.None;

const definitions = {
  MasterBar: {
    'repeat-count': ['isRepeatStart', 'repeatCount'], 'alternate-endings': ['alternateEndings'],
    'directions': ['directions'], 'fermata': ['fermata'], 'free-time': ['isFreeTime'],
    'common-time': ['timeSignatureCommon'], 'key': ['keySignature', 'keySignatureType'],
    'sections': ['section'], 'double-bar': ['isDoubleBar'], 'beaming': ['beamingRules'],
    'layout': ['displayScale', 'displayWidth']
  },
  Track: { 'short-name': ['shortName'], 'layout': ['defaultSystemsLayout', 'systemsLayout'],
    'notation-visibility': ['isVisibleOnMultiTrack'] },
  Staff: { 'transposition': ['transpositionPitch', 'displayTranspositionPitch'],
    'capo': ['capo'], 'slash': ['showSlash'], 'numbered': ['showNumbered'],
    'notation-visibility': ['showTablature', 'showStandardNotation'] },
  Bar: { 'simile': ['simileMark'], 'clef-octave': ['clefOttava'],
    'sustain-pedal': ['sustainPedals'], 'layout': ['displayScale', 'displayWidth'],
    'barlines': ['barLineLeft', 'barLineRight', 'barNumberDisplay'] },
  Beat: { 'fade-in': ['fadeIn'], 'fade-other': ['fade'], 'beat-vibrato': ['vibrato'],
    'legato-slurs': ['isLegatoOrigin'], 'beat-lyrics': ['lyrics'], 'slash': ['slashed'],
    'dead-slap': ['deadSlapped'], 'golpe': ['golpe'], 'wah': ['wahPedal'],
    'barre': ['barreFret', 'barreShape'], 'rasgueado': ['rasgueado'],
    'pick-stroke': ['pickStroke'], 'brush': ['brushType', 'brushDuration'],
    'tap-slap-pop': ['tap', 'slap', 'pop'], 'tremolo': ['tremoloPicking'],
    'beaming': ['beamingMode', 'invertBeamDirection', 'preferredBeamDirection'],
    'timer': ['showTimer', 'timer'], 'beat-octave': ['ottava'] },
  Note: { 'note-vibrato': ['vibrato'], 'tenuto': ['accentuated'], 'bends': ['bendPoints'],
    'tapping': ['isLeftHandTapped'], 'ornaments': ['ornament'],
    'fingering': ['leftHandFinger', 'rightHandFinger'],
    'show-string': ['showStringNumber'], 'note-display': ['isVisible'],
    'accidentals': ['accidentalMode'], 'sound-duration': ['durationPercent'],
    'legato-slurs': ['isSlurDestination'] }
};

function plain(value) {
  if (value === undefined || value === null) return null;
  if (typeof value !== 'object') return value;
  if (value instanceof Uint8Array) return { byteLength: value.length, sha256: crypto.createHash('sha256').update(value).digest('hex') };
  if (value instanceof Map) return [...value].map(([k, v]) => [k, plain(v)]).sort((a, b) => String(a[0]).localeCompare(String(b[0])));
  if (value instanceof Set) return [...value].sort();
  if (Array.isArray(value)) return value.map(plain);
  // Only explicitly selected authored sub-records. Never serialize graph pointers.
  const keys = ['type', 'length', 'marker', 'text', 'marks', 'style', 'groups', 'ratioPosition', 'pedalType', 'offset', 'value'];
  return Object.fromEntries(keys.filter(k => Object.hasOwn(value, k)).map(k => [k, plain(value[k])]));
}

function project(score, options = {}) {
  const out = {};
  const add = (capability, location, value, defaultValue = null) => {
    const normalized = plain(value);
    if (JSON.stringify(normalized) !== JSON.stringify(plain(defaultValue))) {
      (out[capability] ??= []).push({ path: location, value: normalized });
    }
  };
  const collect = (type, object, location) => {
    const defaults = new alphaTab.model[type]();
    for (const [capability, fields] of Object.entries(definitions[type])) {
      if (capability === 'beat-lyrics' && !options.authoredBeatLyrics) continue;
      for (const field of fields) {
        // These rows isolate the non-default variant named by the capability.
        if (capability === 'fade-other' && ![alphaTab.model.FadeType.FadeOut, alphaTab.model.FadeType.VolumeSwell].includes(object[field])) continue;
        if (capability === 'tenuto' && object[field] !== alphaTab.model.AccentuationType.Tenuto) continue;
        // Detached MasterBar key getters require a score. Use documented enum defaults.
        const defaultValue = type === 'MasterBar' && ['keySignature', 'keySignatureType'].includes(field)
          ? 0 : defaults[field];
        add(capability, `${location}.${field}`, object[field], defaultValue);
      }
    }
  };
  for (const bar of score.masterBars) collect('MasterBar', bar, `masterBars[${bar.index}]`);
  for (const track of score.tracks) {
    const tp = `tracks[${track.index}]`;
    collect('Track', track, tp);
    add('midi-bank', `${tp}.bank`, track.playbackInfo.bank, new alphaTab.model.PlaybackInformation().bank);
    for (const staff of track.staves) {
      const sp = `${tp}.staves[${staff.index}]`;
      collect('Staff', staff, sp);
      add('tuning', `${sp}.tuningName`, staff.stringTuning.name, new alphaTab.model.Tuning().name);
      for (const bar of staff.bars) {
        const bp = `${sp}.bars[${bar.index}]`;
        collect('Bar', bar, bp);
        for (const voice of bar.voices) {
          let regular = 0, grace = 0;
          for (const beat of voice.beats) {
            const isGrace = beat.graceType !== alphaTab.model.GraceType.None;
            const address = isGrace ? `before[${regular}].graces[${grace++}]` : `beats[${regular++}]`;
            if (!isGrace) grace = 0;
            const p = `${bp}.voices[${voice.index}].${address}`;
            collect('Beat', beat, p);
            if (beat.chord) {
              for (const field of ['showName', 'showDiagram', 'showFingering']) {
                add('chord-display', `${p}.chord.${field}`, beat.chord[field], new alphaTab.model.Chord()[field]);
              }
              add('chord-diagram', `${p}.chord.firstFret`, beat.chord.firstFret, new alphaTab.model.Chord().firstFret);
              add('chord-diagram', `${p}.chord.strings`, beat.chord.strings, []);
              add('chord-diagram', `${p}.chord.barreFrets`, beat.chord.barreFrets, []);
            }
            for (const note of beat.notes) collect('Note', note, `${p}.notes[${note.index}]`);
            for (const [index, automation] of beat.automations.entries()) {
              const cap = automation.type === alphaTab.model.AutomationType.Volume ? 'volume-automation'
                : automation.type === alphaTab.model.AutomationType.Balance ? 'pan-automation'
                  : automation.type === alphaTab.model.AutomationType.Bank ? 'midi-bank' : null;
              if (cap) add(cap, `${p}.automations[${index}]`, { ratioPosition: automation.ratioPosition, type: automation.type, text: String(automation.value) });
            }
          }
        }
      }
    }
  }
  const stylesheet = new alphaTab.model.RenderStylesheet();
  for (const field of Object.keys(stylesheet).sort()) {
    const cap = /MultiBarRest/.test(field) ? 'multi-rest' : 'stylesheet';
    add(cap, `stylesheet.${field}`, score.stylesheet[field], stylesheet[field]);
  }
  return out;
}

function load(file) {
  const settings = new alphaTab.Settings();
  Object.assign(settings.importer, oracle.importerSettings);
  return alphaTab.importer.ScoreLoader.loadScoreFromBytes(new Uint8Array(fs.readFileSync(file)), settings);
}

const paths = JSON.parse(fs.readFileSync(path.join(root, 'conformance/fixture-inventory.json'))).fixtures.map(f => f.path);
const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'guitar-capabilities-'));
try {
  const logicalPaths = new Map(paths.map(p => [p, p]));
  const upstream = JSON.parse(execFileSync('python3', ['-c',
    'import sys,json;sys.path.insert(0,sys.argv[1]);import manage;print(json.dumps(manage.upstream_fixtures()))', here], { encoding: 'utf8' }));
  for (const [index, fixture] of upstream.filter(f => f.local_matches.length === 0).entries()) {
    const disk = path.join(temporary, `upstream-${index}${path.extname(fixture.path)}`);
    const data = execFileSync('git', ['-C', path.join(root, 'references/alphaTab'), 'show', `${oracle.sourceRevision}:${fixture.path}`], { maxBuffer: 64 * 1024 * 1024 });
    fs.writeFileSync(disk, data);
    logicalPaths.set(disk, `upstream/${fixture.path.replace('packages/alphatab/test-data/', '')}`);
    paths.push(disk);
  }
  // A valid spelling variant missing from the corpus: AlphaTab accepts lowercase key modes.
  const lowercaseKey = path.join(temporary, 'key-mode-lowercase.gp');
  execFileSync('python3', ['-c',
    'import sys,zipfile,xml.etree.ElementTree as E\nwith zipfile.ZipFile(sys.argv[1]) as src, zipfile.ZipFile(sys.argv[2],"w") as dst:\n for info in src.infolist():\n  data=src.read(info)\n  if info.filename.endswith("score.gpif"):\n   tree=E.fromstring(data)\n   for key in tree.findall("./MasterBars/MasterBar/Key/Mode"):key.text="minor"\n   data=E.tostring(tree,encoding="utf-8")\n  dst.writestr(info,data)',
    path.join(root, 'testdata/gp7/notes.gp'), lowercaseKey]);
  logicalPaths.set(lowercaseKey, 'synthetic/key-mode-lowercase.gp');
  paths.push(lowercaseKey);
  let output;
  try {
    output = execFileSync('go', ['run', path.join(here, 'probe.go')], {
      cwd: root, input: JSON.stringify({ Paths: paths, Output: temporary }),
      encoding: 'utf8', maxBuffer: 256 * 1024 * 1024
    });
  } catch (error) {
    throw new Error(`Public API bridge failed: ${error.stderr ?? error.message}`);
  }
  const fixtures = [];
  const catalogIDs = new Set(catalog.capabilities.map(c => c.id));
  for (const [index, result] of output.trim().split('\n').map(line => JSON.parse(line)).entries()) {
    let source = null, target = null, sourceConsumerError = null, targetConsumerError = null;
    // Gate AlphaTab's beat.lyrics projection from the original GPIF container,
    // independently of the Go model under test. This excludes legacy lyrics
    // applied by AlphaTab while keeping an import regression visible.
    const authoredBeatLyrics = result.SourceBeatLyrics;
    try {
      source = project(load(path.resolve(root, result.Path)), { authoredBeatLyrics });
    } catch (error) {
      sourceConsumerError = String(error);
    }
    if (result.Output) {
      try {
        target = project(load(result.Output), { authoredBeatLyrics });
      } catch (error) {
        targetConsumerError = String(error);
      }
    }
    const comparisons = [];
    const tested = Object.values(definitions).flatMap(group => Object.keys(group));
    tested.push('midi-bank', 'tuning', 'chord-display', 'chord-diagram', 'volume-automation', 'pan-automation', 'multi-rest', 'stylesheet');
    for (const id of [...new Set([...tested, ...Object.keys(source ?? {}), ...Object.keys(target ?? {})])].sort()) {
      if (!catalogIDs.has(id)) throw new Error(`Unknown probe capability ${id}`);
      const from = source === null ? null : source[id] ?? [];
      const to = target === null ? null : target[id] ?? [];
      const equal = from === null || to === null ? null : JSON.stringify(from) === JSON.stringify(to);
      const comparisonStatus = from === null ? 'source-blocked'
        : to === null ? 'target-blocked'
          : from.length === 0 && to.length === 0 ? 'default-only'
            : equal ? 'equal-nondefault' : 'different';
      comparisons.push({ capability: id, source: from, target: to, equal, comparisonStatus });
    }
    // Aggregate repetitive reports, but preserve exact source/output values and paths.
    const summarize = (items, key) => {
      const groups = new Map();
      for (const item of items ?? []) {
        const k = `${item.Code}:${item[key]}`;
        const old = groups.get(k);
        if (old) old.count++;
        else groups.set(k, { code: item.Code, kind: item[key], count: 1, example: item });
      }
      return [...groups.values()];
    };
    fixtures.push({ path: logicalPaths.get(result.Path), sha256: crypto.createHash('sha256').update(fs.readFileSync(path.resolve(root, result.Path))).digest('hex'),
      parseError: result.ParseError || null, exportError: result.ExportError || null,
      sourceConsumerError, targetConsumerError,
      diagnostics: summarize(result.Diagnostics, 'Kind'), report: summarize(result.Report.Entries, 'Disposition'), comparisons });
    if ((index + 1) % 50 === 0) process.stderr.write(`Audited ${index + 1}/${paths.length} fixtures\n`);
  }
  const hashes = JSON.parse(execFileSync('python3', ['-c',
    'import sys,json;sys.path.insert(0,sys.argv[1]);import manage;print(json.dumps(manage.input_hashes()))', here], { encoding: 'utf8' }));
  const result = { schema_version: 2, run_at: new Date().toISOString(), oracle,
    source_commit: execFileSync('git', ['rev-parse', 'HEAD'], { cwd: root, encoding: 'utf8' }).trim(),
    input_hashes: hashes, probe_sha256: crypto.createHash('sha256').update(fs.readFileSync(fileURLToPath(import.meta.url))).digest('hex'),
    bridge_sha256: crypto.createHash('sha256').update(fs.readFileSync(path.join(here, 'probe.go'))).digest('hex'),
    scope: 'Selected authored values loaded by pinned AlphaTab before and after public Go import and GP8 export.',
    limitations: ['Differences require review: export limitations, importer differences and AlphaTab derivation can all contribute.',
      'Default values are omitted; absence of a row is not coverage. No difference is not a full-support certificate.',
      'Grace addresses use regular-beat ordinal plus grace ordinal; changed grace grouping can produce alignment differences.',
      'Source and target consumer failures are loaded independently and remain visible as separate comparison statuses.'], fixtures };
  fs.writeFileSync(path.join(here, 'probe-results.json'), `${JSON.stringify(result, null, 2)}\n`);
  process.stdout.write(`Saved ${fixtures.length} fixture receipts\n`);
} finally {
  fs.rmSync(temporary, { recursive: true, force: true });
}
