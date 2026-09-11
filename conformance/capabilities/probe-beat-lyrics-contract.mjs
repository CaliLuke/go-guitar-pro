// Focused regression for the independent beat-lyrics probe gate.
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
import * as alphaTab from '@coderline/alphatab';

const here = path.dirname(fileURLToPath(import.meta.url));
const root = path.resolve(here, '../..');
const oracle = JSON.parse(fs.readFileSync(path.join(root, 'conformance/oracle.json')));
alphaTab.Logger.logLevel = alphaTab.LogLevel.None;

function load(file) {
  const settings = new alphaTab.Settings();
  Object.assign(settings.importer, oracle.importerSettings);
  return alphaTab.importer.ScoreLoader.loadScoreFromBytes(new Uint8Array(fs.readFileSync(file)), settings);
}

function beatLyrics(score, sourceHasBeatLyrics) {
  if (!sourceHasBeatLyrics) return [];
  const facts = [];
  for (const track of score.tracks) {
    for (const staff of track.staves) {
      for (const bar of staff.bars) {
        for (const voice of bar.voices) {
          for (const beat of voice.beats) {
            if (beat.lyrics === null || beat.lyrics === undefined) continue;
            facts.push({
              path: `tracks[${track.index}].staves[${staff.index}].bars[${bar.index}].voices[${voice.index}].beats[${beat.index}].lyrics`,
              value: Array.from(beat.lyrics)
            });
          }
        }
      }
    }
  }
  return facts;
}

const fixturePath = path.join(root, 'testdata/gp7/beat-lyrics.gp');
const legacyPath = path.join(root, 'testdata/gp5/Demo v5.gp5');
const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'beat-lyrics-probe-contract-'));
try {
  // Remove the separate track-scoped Lyrics input so an import/model drop has
  // no derived AlphaTab target lyrics that could obscure the missing beat data.
  const sourcePath = path.join(temporary, 'authored-beat-lyrics.gp');
  execFileSync('python3', ['-c',
    'import sys,zipfile,xml.etree.ElementTree as E\nwith zipfile.ZipFile(sys.argv[1]) as src, zipfile.ZipFile(sys.argv[2],"w") as dst:\n for info in src.infolist():\n  data=src.read(info)\n  if info.filename.endswith("score.gpif"):\n   tree=E.fromstring(data)\n   for track in tree.findall("./Tracks/Track"):\n    lyrics=track.find("Lyrics")\n    if lyrics is not None: track.remove(lyrics)\n   data=E.tostring(tree,encoding="utf-8")\n  dst.writestr(info,data)',
    fixturePath, sourcePath]);
  const bridge = process.env.BEAT_LYRICS_PROBE_BRIDGE ?? path.join(here, 'probe.go');
  const goFlags = [process.env.GOFLAGS, process.env.BEAT_LYRICS_PROBE_GOFLAGS].filter(Boolean).join(' ');
  const output = execFileSync('go', ['run', bridge], {
    cwd: root,
    input: JSON.stringify({ Paths: [sourcePath, legacyPath], Output: temporary }),
    encoding: 'utf8',
    maxBuffer: 128 * 1024 * 1024,
    env: { ...process.env, ...(goFlags ? { GOFLAGS: goFlags } : {}) }
  });
  const [sourceReceipt, legacyReceipt] = output.trim().split('\n').map(line => JSON.parse(line));
  if (!sourceReceipt.SourceBeatLyrics) {
    throw new Error('beat-lyrics probe did not detect authored source GPIF independently');
  }
  const source = beatLyrics(load(sourcePath), sourceReceipt.SourceBeatLyrics);
  const target = beatLyrics(load(sourceReceipt.Output), sourceReceipt.SourceBeatLyrics);
  if (source.length !== 6) {
    throw new Error(`beat-lyrics probe source count = ${source.length}, want 6`);
  }
  if (target.length !== 6) {
    throw new Error(`beat-lyrics probe target count = ${target.length}, want 6`);
  }
  if (JSON.stringify(source) !== JSON.stringify(target)) {
    throw new Error('beat-lyrics probe source and target facts differ');
  }
  if (legacyReceipt.SourceBeatLyrics || beatLyrics(load(legacyPath), legacyReceipt.SourceBeatLyrics).length !== 0) {
    throw new Error('beat-lyrics probe included legacy track/score lyric application owned by #98');
  }
  process.stdout.write('beat-lyrics probe contract preserves 6/6 authored facts and excludes legacy application\n');
} finally {
  fs.rmSync(temporary, { recursive: true, force: true });
}
