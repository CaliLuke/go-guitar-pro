import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';

const root = path.resolve(new URL('..', import.meta.url).pathname);
const ledger = JSON.parse(fs.readFileSync(path.join(root, 'conformance/feature-ledger.json'), 'utf8'));
const declared = new Set(ledger.semanticContracts.behaviorContracts.map(contract => contract.id));

const mutations = [
  {
    id: 'disabled-semantic-assertion',
    contract: 'field-disposition-evidence',
    category: 'executable-evidence',
    file: 'semantic_matrix_m01_test.go',
    before: '\t\t\trun.Field("Song.Name", song.Name, "Title")\n',
    after: '\t\t\t_ = song.Name\n',
    test: '^TestSemanticMatrixInventory$',
    want: 'semantic assertions for M01-METADATA-CORPUS mismatch'
  },
  {
    id: 'unclassified-model-field',
    category: 'inventory',
    file: 'song.go',
    before: '\tClipboard      *Clipboard\n',
    after: '\tClipboard      *Clipboard\n\tSemanticMutationProbe bool\n',
    test: '^TestSemanticContractInventory$',
    want: 'SemanticMutationProbe'
  },
  {
    id: 'unclassified-source-dispatch',
    category: 'inventory',
    file: 'gpif.go',
    before: '\tswitch property.Name {\n\tcase "Brush", "PickStroke":',
    after: '\tswitch property.Name {\n\tcase "SemanticMutationProbe":\n\t\treturn\n\tcase "Brush", "PickStroke":',
    test: '^TestSemanticContractInventory$',
    want: 'SemanticMutationProbe'
  },
  {
    id: 'unclassified-semantic-selector',
    category: 'inventory',
    file: 'gpif.go',
    before: '\t\t\t\t\t\t\t\t\tswitch b.GraceNotes {\n',
    after: '\t\t\t\t\t\t\t\t\tswitch b.GraceNotes {\n\t\t\t\t\t\t\t\t\tcase "ReviewUnclassifiedGrace":\n\t\t\t\t\t\t\t\t\t\tisGrace = true\n',
    test: '^TestSemanticContractInventory$',
    want: 'ReviewUnclassifiedGrace'
  },
  {
    id: 'unclassified-gpif-wire-field',
    category: 'inventory',
    file: 'gpif.go',
    before: '\tGraceNotes string         `xml:"GraceNotes,omitempty"`\n',
    after: '\tGraceNotes string         `xml:"GraceNotes,omitempty"`\n\tReviewIgnoredText string       `xml:"ReviewIgnoredText,omitempty"`\n',
    test: '^TestSemanticContractInventory$',
    want: 'gpifBeat.ReviewIgnoredText'
  },
  {
    id: 'unclassified-dispatch-in-moved-helper',
    contract: 'unclassified-source-dispatch',
    category: 'inventory',
    file: 'parser.go',
    before: ')\n\n// ParseDiagnosticKind classifies',
    after: ')\n\nfunc semanticMutationMovedHelper(value struct{ Name string }) {\n\tswitch value.Name {\n\tcase "MovedHelperProbe":\n\t}\n}\n\n// ParseDiagnosticKind classifies',
    test: '^TestSemanticContractInventory$',
    want: 'semanticMutationMovedHelper:value.Name'
  },
  {
    id: 'automation-dispatch-diagnostic',
    category: 'diagnostic',
    file: 'gpif.go',
    before: '\t\tcase "SustainPedal":\n',
    after: '\t\tcase "DisabledSustainPedal":\n',
    test: '^TestGPIFAutomationDispatchDiagnostics$',
    want: 'unsupported_sustain_pedal'
  },
  {
    id: 'removed-automation-diagnostic',
    contract: 'automation-dispatch-diagnostic',
    category: 'diagnostic',
    file: 'gpif.go',
    before: '\t\tcase "SustainPedal":\n\t\t\tcontext.add(diagnosticSource("GPIF.Track.Automation.SustainPedal", "score-core", ParseDiagnosticUnsupportedFeature), ParseDiagnostic{\n\t\t\t\tSourcePath: path + "/Type/SustainPedal", ObjectID: track.ID,\n\t\t\t\tReason: "sustain-pedal automation has no Song destination",\n\t\t\t})\n',
    after: '\t\tcase "SustainPedal":\n',
    test: '^TestGPIFAutomationDispatchDiagnostics$',
    want: 'want unsupported-feature'
  },
  {
    id: 'supported-effect-serialization',
    category: 'serialization',
    file: 'gp8_writer.go',
    before: '\t\tresult.Hairpin = "Crescendo"\n',
    after: '\t\tresult.Hairpin = ""\n',
    test: '^TestExportGP8PreservesHairpins$',
    want: 'first beat hairpin'
  },
  {
    id: 'shared-authored-export-validation',
    category: 'validation',
    file: 'export_report.go',
    before: '\tfor _, diagnostic := range authoredScoreDiagnostics(song) {\n',
    after: '\tfor _, diagnostic := range []ScoreDiagnostic(nil) {\n',
    test: '^TestGP8ExportRejectsSharedAuthoredInvariants$',
    want: 'half_tuplet'
  },
  {
    id: 'tempo-compatibility-authority',
    category: 'compatibility',
    file: 'gp8_writer.go',
    before: '\t\t\treturn float64(song.Tempo), true, nil\n',
    after: '\t\t\treturn float64(exact), true, nil\n',
    test: '^TestGP8ExportReconcilesSemanticAndLegacyTempo$',
    want: 'round-trip tempo'
  },
  {
    id: 'chord-occurrence-isolation',
    category: 'ownership',
    file: 'gpif.go',
    before: '\tclone.Strings = slices.Clone(source.Strings)\n',
    after: '\tclone.Strings = source.Strings\n',
    test: '^TestRepeatedGPIFChordOccurrencesOwnMutablePayloads$',
    want: 'second chord changed'
  },
  {
    id: 'master-bar-narrowing-boundary',
    category: 'numeric-boundary',
    file: 'gpif.go',
    before: '\t\t\tif numeratorErr != nil || numerator <= 0 || numerator > math.MaxInt8 {\n',
    after: '\t\t\tif numeratorErr != nil || numerator <= 0 {\n',
    test: '^TestGPIFMasterBarValuesDoNotWrapAtLegacyBoundaries$',
    want: 'modulo_numerator'
  },
  {
    id: 'master-bar-denominator-boundary',
    category: 'numeric-boundary',
    file: 'gpif.go',
    before: '\t\t\tif denominatorErr != nil || denominator <= 0 || denominator > math.MaxUint16 {\n',
    after: '\t\t\tif denominatorErr != nil || denominator <= 0 {\n',
    test: '^TestGPIFMasterBarValuesDoNotWrapAtLegacyBoundaries$',
    want: 'denominator_modulo_overflow'
  },
  {
    id: 'inspected-note-duration-percent',
    category: 'loss-report',
    file: 'gp8_writer.go',
    before: '\tif note.DurationPercent != 1 {\n\t\tbuilder.addReport("gp8.omit.note-duration-percent", "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 has no note duration-percent field")\n\t}\n',
    after: '',
    test: '^TestGP8StrictExportCoversInspectedSemanticFields$',
    want: 'note_duration_percent'
  },
  {
    id: 'adapter-ownership-projection',
    contract: 'field-disposition-evidence',
    category: 'adapter',
    file: 'conformance_test.go',
    before: '\t\t\t"index": track["index"], "name": track["name"], "staves": track["staves"],\n',
    after: '\t\t\t"index": track["index"], "name": track["name"], "staves": []any{},\n',
    test: '^TestStaffOwnershipProjectionDetectsContentMutations$',
    want: 'staff-ownership projection did not detect a second-staff pitch mutation'
  },
  {
    id: 'adapter-enum-collapse',
    contract: 'field-disposition-evidence',
    category: 'adapter',
    file: 'conformance_test.go',
    before: '\tcase SlideIntoFromAbove:\n\t\treturn "into-from-above"\n',
    after: '\tcase SlideIntoFromAbove:\n\t\treturn "into-from-below"\n',
    test: '^TestM25AdapterMutationSensitivity$',
    want: 'slide adapter collapsed distinct enum values'
  },
  {
    id: 'field-disposition-evidence',
    category: 'classification',
    file: 'conformance/feature-ledger.json',
    replacements: [
      { before: '"preserved":["Song.Album"', after: '"preserved":["Chord.Barres","Song.Album"' },
      { before: '"Clipboard.SubBarCopy","Chord.Barres","Note.DurationPercent"', after: '"Clipboard.SubBarCopy","Note.DurationPercent"' }
    ],
    command: 'verify',
    want: 'Chord.Barres evidence proves omitted, but the field partition claims preserved'
  },
  {
    id: 'structural-export-resilience',
    category: 'serialization',
    file: 'gp8_writer.go',
    before: '\t\tbarIDs := make([]string, 0, len(builder.song.Tracks))\n\t\tfor trackIndex := range builder.song.Tracks {\n',
    after: '\t\tbarIDs := make([]string, 0, len(builder.song.Tracks))\n\t\tfor trackIndex := range min(2, len(builder.song.Tracks)) {\n',
    test: '^TestSemanticMatrixM25StructuralResilience$',
    want: 'measure count'
  }
];

const mutationContracts = new Set(mutations.map(mutation => mutation.contract ?? mutation.id));
for (const mutation of mutations) {
  const contract = mutation.contract ?? mutation.id;
  if (!declared.has(contract)) throw new Error(`${mutation.id} has no behavior contract ${contract}`);
  if (!mutation.category) throw new Error(`${mutation.id} has no mutation category`);
}
for (const id of declared) {
  const contract = ledger.semanticContracts.behaviorContracts.find(item => item.id === id);
  if (contract.mutationSensitive && !mutationContracts.has(id)) {
    throw new Error(`${id} claims mutation sensitivity without a mutation`);
  }
}

const temporary = fs.mkdtempSync(path.join(os.tmpdir(), 'go-guitar-pro-mutants-'));
try {
  for (const mutation of mutations) {
    const original = path.join(root, mutation.file);
    const source = fs.readFileSync(original, 'utf8');
    const replacements = mutation.replacements ?? [{ before: mutation.before, after: mutation.after }];
    let mutatedSource = source;
    for (const replacement of replacements) {
      const occurrences = mutatedSource.split(replacement.before).length - 1;
      if (occurrences !== 1) {
        throw new Error(`${mutation.id} expected one source match in ${mutation.file}, found ${occurrences}`);
      }
      mutatedSource = mutatedSource.replace(replacement.before, replacement.after);
    }
    const mutated = path.join(temporary, `${mutation.id}-${path.basename(mutation.file)}`);
    fs.writeFileSync(mutated, mutatedSource);
    let result;
    if (mutation.command === 'verify') {
      result = spawnSync('node', ['conformance/verify.mjs'], {
        cwd: root, encoding: 'utf8', env: { ...process.env, SEMANTIC_LEDGER_OVERLAY: mutated }
      });
    } else {
      const overlay = path.join(temporary, `${mutation.id}-overlay.json`);
      fs.writeFileSync(overlay, JSON.stringify({ Replace: { [original]: mutated } }));
      result = spawnSync('go', ['test', '-count=1', '-overlay', overlay, '-run', mutation.test, '.'], {
        cwd: root,
        encoding: 'utf8',
        env: { ...process.env, SEMANTIC_INVENTORY_OVERLAY: mutated, SEMANTIC_INVENTORY_FILE: mutation.file }
      });
    }
    const output = `${result.stdout ?? ''}${result.stderr ?? ''}`;
    if (result.status === 0) throw new Error(`${mutation.id} survived ${mutation.test ?? mutation.command}`);
    const expectedFailure = mutation.command === 'verify' || output.includes('--- FAIL:');
    if (!expectedFailure || !output.includes(mutation.want)) {
      throw new Error(`${mutation.id} failed for an unexpected reason:\n${output}`);
    }
    process.stdout.write(`killed ${mutation.id} [${mutation.category}]\n`);
  }
} finally {
  fs.rmSync(temporary, { recursive: true, force: true });
}
