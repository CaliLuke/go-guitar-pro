import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';

const root = path.resolve(new URL('..', import.meta.url).pathname);
const ledger = JSON.parse(fs.readFileSync(path.join(root, 'conformance/feature-ledger.json'), 'utf8'));
const declared = new Set(ledger.semanticContracts.behaviorContracts.map(contract => contract.id));

const mutations = [
  {
    id: 'supported-capability-claim-removed',
    contract: 'semantic-obligation-shape',
    category: 'capability-closure',
    file: 'conformance/feature-ledger.json',
    before: '{"capability":"repeat-count","stage":"import","case":"M07-MASTER-BARS","source":{"kind":"matrix-scenario","id":"M07-MASTER-BARS/repeat-count/repeat-counts-1-6-and-128"},"value":"repeat counts 1, 6, and 128","assertionStage":"import","obligation":"field:MeasureHeader.RepeatCount","reportNotApplicable":"not-applicable-non-export-stage","independentLimit":{"id":"limit-repeat-count-import","kind":"no-claim-specific-consumer","oracle":"@coderline/alphatab@1.8.4","source":"TestConformanceMasterBars","obligation":"field:MeasureHeader.RepeatCount","reason":"The pinned consumer registry has no claim-specific import callback for repeat-count; TestConformanceMasterBars supplies the exact field:MeasureHeader.RepeatCount matrix assertion."}},',
    after: '',
    command: 'ledger-test', test: '^TestSemanticMatrixInventory$',
    want: 'supported capability stage repeat-count:import has 0 executable closure claims'
  },
  {
    id: 'supported-capability-source-mismatch', contract: 'semantic-obligation-shape', category: 'capability-closure',
    file: 'conformance/feature-ledger.json',
    before: '"capability":"repeat-count","stage":"import","case":"M07-MASTER-BARS","source":{"kind":"matrix-scenario","id":"M07-MASTER-BARS/repeat-count/repeat-counts-1-6-and-128"},',
    after: '"capability":"repeat-count","stage":"import","case":"M07-MASTER-BARS","source":{"kind":"matrix-scenario","id":"fabricated"},',
    command: 'ledger-test', test: '^TestSemanticMatrixInventory$', want: 'does not match assertion-site receipt'
  },
  {
    id: 'supported-capability-value-mismatch', contract: 'semantic-obligation-shape', category: 'capability-closure',
    file: 'conformance/feature-ledger.json',
    before: '"value":"repeat counts 1, 6, and 128","assertionStage":"import",',
    after: '"value":"fabricated","assertionStage":"import",',
    command: 'ledger-test', test: '^TestSemanticMatrixInventory$', want: 'does not match assertion-site receipt'
  },
  {
    id: 'supported-capability-stage-mismatch', contract: 'semantic-obligation-shape', category: 'capability-closure',
    file: 'conformance/feature-ledger.json',
    before: '"capability":"repeat-count","stage":"import","case":"M07-MASTER-BARS",',
    after: '"capability":"repeat-count","stage":"fabricated","case":"M07-MASTER-BARS",',
    command: 'ledger-test', test: '^TestSemanticMatrixInventory$', want: 'has no assertion-site receipt'
  },
  {
    id: 'supported-capability-assertion-stage-mismatch', contract: 'semantic-obligation-shape', category: 'capability-closure',
    file: 'conformance/feature-ledger.json',
    before: '"value":"repeat counts 1, 6, and 128","assertionStage":"import",',
    after: '"value":"repeat counts 1, 6, and 128","assertionStage":"programmatic",',
    command: 'ledger-test', test: '^TestSemanticMatrixInventory$', want: 'does not match assertion-site receipt'
  },
  {
    id: 'supported-capability-primary-obligation-mismatch', contract: 'semantic-obligation-shape', category: 'capability-closure',
    file: 'conformance/feature-ledger.json',
    before: '"assertionStage":"import","obligation":"field:MeasureHeader.RepeatCount",',
    after: '"assertionStage":"import","obligation":"field:MeasureHeader.TimeSignature",',
    command: 'ledger-test', test: '^TestSemanticMatrixInventory$', want: 'does not match assertion-site receipt'
  },
  {
    id: 'supported-capability-synchronized-scenario-redirect', contract: 'semantic-obligation-shape', category: 'capability-closure',
    file: 'conformance/feature-ledger.json',
    before: '{"capability":"repeat-count","stage":"import","case":"M07-MASTER-BARS","source":{"kind":"matrix-scenario","id":"M07-MASTER-BARS/repeat-count/repeat-counts-1-6-and-128"},"value":"repeat counts 1, 6, and 128","assertionStage":"import","obligation":"field:MeasureHeader.RepeatCount","reportNotApplicable":"not-applicable-non-export-stage","independentLimit":{"id":"limit-repeat-count-import","kind":"no-claim-specific-consumer","oracle":"@coderline/alphatab@1.8.4","source":"TestConformanceMasterBars","obligation":"field:MeasureHeader.RepeatCount","reason":"The pinned consumer registry has no claim-specific import callback for repeat-count; TestConformanceMasterBars supplies the exact field:MeasureHeader.RepeatCount matrix assertion."}}',
    after: '{"capability":"repeat-count","stage":"import","case":"M07-MASTER-BARS","source":{"kind":"matrix-scenario","id":"M07-MASTER-BARS/repeat-count/meter-endpoints"},"value":"meter endpoints","assertionStage":"import","obligation":"field:MeasureHeader.TimeSignature","reportNotApplicable":"not-applicable-non-export-stage","independentLimit":{"id":"limit-repeat-count-import","kind":"no-claim-specific-consumer","oracle":"@coderline/alphatab@1.8.4","source":"TestConformanceMasterBars","obligation":"field:MeasureHeader.TimeSignature","reason":"The pinned consumer registry has no claim-specific import callback for repeat-count; TestConformanceMasterBars supplies the exact field:MeasureHeader.TimeSignature matrix assertion."}}',
    command: 'ledger-test', test: '^TestSemanticMatrixInventory$', want: 'does not match assertion-site receipt'
  },
  {
    id: 'supported-capability-unrelated-serialization', contract: 'semantic-obligation-shape', category: 'capability-closure',
    file: 'conformance/feature-ledger.json',
    before: '"serialization":"wire:gpifRepeat.Count","reportAssertion":"report:M07-MASTER-BARS",',
    after: '"serialization":"wire:gpifMasterBar.Time","reportAssertion":"report:M07-MASTER-BARS",',
    command: 'ledger-test', test: '^TestSemanticMatrixInventory$', want: 'does not match assertion-site receipt'
  },
  {
    id: 'supported-capability-report-mismatch', contract: 'semantic-obligation-shape', category: 'capability-closure',
    file: 'conformance/feature-ledger.json',
    before: '"serialization":"wire:gpifRepeat.Count","reportAssertion":"report:M07-MASTER-BARS",',
    after: '"serialization":"wire:gpifRepeat.Count","reportAssertion":"report:fabricated",',
    command: 'ledger-test', test: '^TestSemanticMatrixInventory$', want: 'does not match assertion-site receipt'
  },
  {
    id: 'supported-capability-independent-evidence-fabricated', contract: 'semantic-obligation-shape', category: 'capability-closure',
    file: 'conformance/feature-ledger.json',
    before: '"capability":"simile","stage":"import","case":"M07-MASTER-BARS","source":{"kind":"matrix-scenario","id":"M07-MASTER-BARS/simile/all-simile-marks"},"value":"all simile marks","assertionStage":"import","obligation":"field:Measure.SimileMark","reportNotApplicable":"not-applicable-non-export-stage","independentEvidence":{"id":"alphatab-simile"}',
    after: '"capability":"simile","stage":"import","case":"M07-MASTER-BARS","source":{"kind":"matrix-scenario","id":"M07-MASTER-BARS/simile/all-simile-marks"},"value":"all simile marks","assertionStage":"import","obligation":"field:Measure.SimileMark","reportNotApplicable":"not-applicable-non-export-stage","independentEvidence":{"id":"fabricated"}',
    command: 'ledger-test', test: '^TestSemanticMatrixInventory$', want: 'cites unknown independent evidence'
  },
  {
    id: 'supported-capability-independent-evidence-redirect', contract: 'semantic-obligation-shape', category: 'capability-closure',
    file: 'conformance/feature-ledger.json',
    before: '"capability":"simile","stage":"import","case":"M07-MASTER-BARS","source":{"kind":"matrix-scenario","id":"M07-MASTER-BARS/simile/all-simile-marks"},"value":"all simile marks","assertionStage":"import","obligation":"field:Measure.SimileMark","reportNotApplicable":"not-applicable-non-export-stage","independentEvidence":{"id":"alphatab-simile"}',
    after: '"capability":"simile","stage":"import","case":"M07-MASTER-BARS","source":{"kind":"matrix-scenario","id":"M07-MASTER-BARS/simile/all-simile-marks"},"value":"all simile marks","assertionStage":"import","obligation":"field:Measure.SimileMark","reportNotApplicable":"not-applicable-non-export-stage","independentEvidence":{"id":"alphatab-free-time"}',
    command: 'alphatab-ledger-test', test: '^TestConformanceCapabilityIndependentEvidence$', want: 'independent evidence alphatab-free-time receipts'
  },
  {
    id: 'supported-capability-independent-limit-invalid', contract: 'semantic-obligation-shape', category: 'capability-closure',
    file: 'conformance/feature-ledger.json',
    before: '"independentLimit":{"id":"limit-repeat-count-import",',
    after: '"independentLimit":{"id":".",',
    command: 'ledger-test', test: '^TestSemanticMatrixInventory$', want: 'has invalid independent limit'
  },
  {
    id: 'disabled-semantic-assertion',
    contract: 'field-disposition-evidence',
    category: 'executable-evidence',
    file: 'conformance_metadata_test.go',
    before: '\t\t\trun.Preserved("Song.Name", song.Name, "Title")\n',
    after: '\t\t\t_ = song.Name\n',
    test: '^TestSemanticMatrixInventory$',
    want: 'semantic assertions for M01-METADATA-CORPUS mismatch'
  },
  {
    id: 'false-preservation-uninspected-field',
    contract: 'field-disposition-evidence',
    category: 'classification',
    file: 'conformance_chords_test.go',
    before: '\trun.Omitted("Chord.Fingerings", chord.Fingerings, []Fingering{FingeringLittle, FingeringOpen, FingeringOpen, FingeringAnnular, FingeringIndex, FingeringMiddle})\n',
    after: '\trun.Preserved("Chord.Fingerings", chord.Fingerings, []Fingering{FingeringLittle, FingeringOpen, FingeringOpen, FingeringAnnular, FingeringIndex, FingeringMiddle})\n',
    test: '^TestSemanticMatrixInventory$',
    want: 'Chord.Fingerings executable evidence proves preserved, but the field partition claims omitted'
  },
  {
    id: 'swapped-uninspected-field-dispositions',
    contract: 'field-disposition-evidence',
    category: 'classification',
    file: 'conformance/feature-ledger.json',
    replacements: [
      { before: '"preserved":["Song.Album"', after: '"preserved":["Song.HideTempo"' },
      { before: '"TimeSignature.Beams","Song.HideTempo","SoundAutomation.Hidden","MidiChannel.Tremolo"', after: '"TimeSignature.Beams","Song.Album","SoundAutomation.Hidden","MidiChannel.Tremolo"' }
    ],
    command: 'ledger-test',
    test: '^TestSemanticMatrixInventory$',
    want: 'Song.HideTempo executable evidence proves omitted, but the field partition claims preserved'
  },
  {
    id: 'removed-represented-field-serialization',
    contract: 'field-disposition-evidence',
    category: 'serialization',
    file: 'gp8_builder.go',
    before: '\t\tTitle:        song.Name,\n',
    after: '\t\tTitle:        "",\n',
    test: '^TestSemanticMatrixInventory$',
    want: 'wire:gpifScore.Title = "", want "Title & 名"'
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
    file: 'gpif_audit.go',
    before: '\tswitch property.Name {\n\tcase "Brush", "PickStroke":',
    after: '\tswitch property.Name {\n\tcase "SemanticMutationProbe":\n\t\treturn\n\tcase "Brush", "PickStroke":',
    test: '^TestSemanticContractInventory$',
    want: 'SemanticMutationProbe'
  },
  {
    id: 'unclassified-numeric-source-discriminant',
    contract: 'unclassified-source-dispatch',
    category: 'inventory',
    file: 'note.go',
    before: '\tcase 22:\n\t\the.Kind = HarmonicTypeArtificial\n',
    after: '\tcase 99:\n\t\the.Kind = HarmonicTypeFeedback\n\tcase 22:\n\t\the.Kind = HarmonicTypeArtificial\n',
    test: '^TestSemanticContractInventory$',
    want: '99'
  },
  {
    id: 'unclassified-semantic-selector',
    category: 'inventory',
    file: 'gpif_parser.go',
    before: '\t\t\t\t\t\t\t\t\tswitch b.GraceNotes {\n',
    after: '\t\t\t\t\t\t\t\t\tswitch b.GraceNotes {\n\t\t\t\t\t\t\t\t\tcase "ReviewUnclassifiedGrace":\n\t\t\t\t\t\t\t\t\t\tisGrace = true\n',
    test: '^TestSemanticContractInventory$',
    want: 'ReviewUnclassifiedGrace'
  },
  {
    id: 'unclassified-gpif-wire-field',
    category: 'inventory',
    file: 'gpif_types.go',
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
    id: 'upstream-new-authored-assignment-and-dispatch',
    contract: 'unclassified-source-dispatch',
    category: 'inventory',
    file: 'conformance/capabilities/upstream_sensitivity.ts',
    replacements: [
      { before: '        beat.brush = true;\n', after: '        beat.brush = true;\n        beat.reviewProbe = true;\n' },
      { before: "        case 'Brush':\n", after: "        case 'Brush':\n        case 'ReviewProbe':\n" }
    ],
    command: 'capability-test',
    want: 'dispatch or assignment has no review'
  },
  {
    id: 'upstream-authored-owner-removed',
    contract: 'unclassified-source-dispatch',
    category: 'ownership',
    file: 'conformance/capabilities/upstream-ownership.json',
    before: '      "construct_id": "packages/alphatab/src/importer/GpifParser.ts::dispatch::GpifParser._parseBeatProperties:Brush::1",\n      "source_scope": "guitar-pro-importer",\n      "formats": [\n        "gp6",\n        "gp7",\n        "gp8"\n      ],\n      "disposition": "authored",\n      "reason": "The containing Guitar Pro importer or configuration reader can reach this authored syntax or model assignment.",\n      "primary_capability": "brush",',
    after: '      "construct_id": "packages/alphatab/src/importer/GpifParser.ts::dispatch::GpifParser._parseBeatProperties:Brush::1",\n      "source_scope": "guitar-pro-importer",\n      "formats": [\n        "gp6",\n        "gp7",\n        "gp8"\n      ],\n      "disposition": "authored",\n      "reason": "The containing Guitar Pro importer or configuration reader can reach this authored syntax or model assignment.",\n      "primary_capability": null,',
    command: 'capability-check',
    want: 'Authored upstream construct has no primary capability owner'
  },
  {
    id: 'upstream-unique-catalog-owner-regression',
    contract: 'unclassified-source-dispatch',
    category: 'ownership',
    file: 'conformance/capabilities/upstream-ownership.json',
    before: '      "declaration": "Beat.brushDuration",\n      "formats": [\n        "gp3",\n        "gp4",\n        "gp5",\n        "gp6",\n        "gp7",\n        "gp8"\n      ],\n      "disposition": "authored",\n      "reason": "A reviewed Guitar Pro importer assignment or enum reference reaches this model declaration.",\n      "primary_capability": "brush",',
    after: '      "declaration": "Beat.brushDuration",\n      "formats": [\n        "gp3",\n        "gp4",\n        "gp5",\n        "gp6",\n        "gp7",\n        "gp8"\n      ],\n      "disposition": "authored",\n      "reason": "A reviewed Guitar Pro importer assignment or enum reference reaches this model declaration.",\n      "primary_capability": "duration",',
    command: 'capability-check',
    want: 'Unique exact catalog owner mismatch for Beat.brushDuration'
  },
  {
    id: 'upstream-inferred-model-link-bypass',
    contract: 'unclassified-source-dispatch',
    category: 'ownership',
    file: 'conformance/capabilities/upstream-ownership.json',
    before: '      "construct_id": "packages/alphatab/src/importer/Gp3To5Importer.ts::assignment::Gp3To5Importer.readBeatEffects:beat.brushDuration::1",\n      "source_scope": "guitar-pro-importer",\n      "formats": [\n        "gp3",\n        "gp4",\n        "gp5"\n      ],\n      "disposition": "authored",\n      "reason": "The containing Guitar Pro importer or configuration reader can reach this authored syntax or model assignment.",\n      "primary_capability": "brush",\n      "secondary_capabilities": [],\n      "model_declaration": "Beat.brushDuration"',
    after: '      "construct_id": "packages/alphatab/src/importer/Gp3To5Importer.ts::assignment::Gp3To5Importer.readBeatEffects:beat.brushDuration::1",\n      "source_scope": "guitar-pro-importer",\n      "formats": [\n        "gp3",\n        "gp4",\n        "gp5"\n      ],\n      "disposition": "authored",\n      "reason": "The containing Guitar Pro importer or configuration reader can reach this authored syntax or model assignment.",\n      "primary_capability": "duration",\n      "secondary_capabilities": [],\n      "model_declaration": null',
    command: 'capability-check',
    want: 'inferred authored assignment has no model declaration link Beat.brushDuration'
  },
  {
    id: 'upstream-model-review-and-links-removed',
    contract: 'unclassified-source-dispatch',
    category: 'ownership',
    file: 'conformance/capabilities/upstream-ownership.json',
    replacements: [
      {
        before: '"construct_id": "packages/alphatab/src/importer/Gp3To5Importer.ts::assignment::Gp3To5Importer.readScoreInformation:this._score.title::1",\n      "source_scope": "guitar-pro-importer",\n      "formats": [\n        "gp3",\n        "gp4",\n        "gp5"\n      ],\n      "disposition": "authored",\n      "reason": "The containing Guitar Pro importer or configuration reader can reach this authored syntax or model assignment.",\n      "primary_capability": "metadata",\n      "secondary_capabilities": [],\n      "model_declaration": "Score.title"',
        after: '"construct_id": "packages/alphatab/src/importer/Gp3To5Importer.ts::assignment::Gp3To5Importer.readScoreInformation:this._score.title::1",\n      "source_scope": "guitar-pro-importer",\n      "formats": [\n        "gp3",\n        "gp4",\n        "gp5"\n      ],\n      "disposition": "authored",\n      "reason": "The containing Guitar Pro importer or configuration reader can reach this authored syntax or model assignment.",\n      "primary_capability": "metadata",\n      "secondary_capabilities": [],\n      "model_declaration": null'
      },
      {
        before: '"construct_id": "packages/alphatab/src/importer/GpifParser.ts::assignment::GpifParser._parseScoreNode:this.score.title::1",\n      "source_scope": "guitar-pro-importer",\n      "formats": [\n        "gp6",\n        "gp7",\n        "gp8"\n      ],\n      "disposition": "authored",\n      "reason": "The containing Guitar Pro importer or configuration reader can reach this authored syntax or model assignment.",\n      "primary_capability": "metadata",\n      "secondary_capabilities": [],\n      "model_declaration": "Score.title"',
        after: '"construct_id": "packages/alphatab/src/importer/GpifParser.ts::assignment::GpifParser._parseScoreNode:this.score.title::1",\n      "source_scope": "guitar-pro-importer",\n      "formats": [\n        "gp6",\n        "gp7",\n        "gp8"\n      ],\n      "disposition": "authored",\n      "reason": "The containing Guitar Pro importer or configuration reader can reach this authored syntax or model assignment.",\n      "primary_capability": "metadata",\n      "secondary_capabilities": [],\n      "model_declaration": null'
      },
      {
        before: '    {\n      "declaration": "Score.title",\n      "formats": [\n        "gp3",\n        "gp4",\n        "gp5",\n        "gp6",\n        "gp7",\n        "gp8"\n      ],\n      "disposition": "authored",\n      "reason": "A reviewed Guitar Pro importer assignment or enum reference reaches this model declaration.",\n      "primary_capability": "metadata",\n      "secondary_capabilities": []\n    },\n',
        after: ''
      },
      { before: '"reviewed_model_declarations": 312', after: '"reviewed_model_declarations": 311' }
    ],
    command: 'capability-check',
    want: 'inferred authored assignment has no model declaration link Score.title'
  },
  {
    id: 'upstream-typed-alias-model-review-and-link-removed',
    contract: 'unclassified-source-dispatch',
    category: 'ownership',
    file: 'conformance/capabilities/upstream-ownership.json',
    replacements: [
      {
        before: '"construct_id": "packages/alphatab/src/importer/GpifParser.ts::assignment::GpifParser._parseAutomation:syncPointValue.barOccurence::1",\n      "source_scope": "guitar-pro-importer",\n      "formats": [\n        "gp6",\n        "gp7",\n        "gp8"\n      ],\n      "disposition": "authored",\n      "reason": "The containing Guitar Pro importer or configuration reader can reach this authored syntax or model assignment.",\n      "primary_capability": "automation-detail",\n      "secondary_capabilities": [\n        "sync-points"\n      ],\n      "model_declaration": "SyncPointData.barOccurence"',
        after: '"construct_id": "packages/alphatab/src/importer/GpifParser.ts::assignment::GpifParser._parseAutomation:syncPointValue.barOccurence::1",\n      "source_scope": "guitar-pro-importer",\n      "formats": [\n        "gp6",\n        "gp7",\n        "gp8"\n      ],\n      "disposition": "authored",\n      "reason": "The containing Guitar Pro importer or configuration reader can reach this authored syntax or model assignment.",\n      "primary_capability": "automation-detail",\n      "secondary_capabilities": [\n        "sync-points"\n      ],\n      "model_declaration": null'
      },
      {
        before: '    {\n      "declaration": "SyncPointData.barOccurence",\n      "formats": [\n        "gp6",\n        "gp7",\n        "gp8"\n      ],\n      "disposition": "authored",\n      "reason": "A reviewed Guitar Pro importer assignment or enum reference reaches this model declaration.",\n      "primary_capability": "automation-detail",\n      "secondary_capabilities": [\n        "sync-points"\n      ]\n    },\n',
        after: ''
      },
      { before: '"reviewed_model_declarations": 312', after: '"reviewed_model_declarations": 311' }
    ],
    command: 'capability-check',
    want: 'inferred authored assignment has no model declaration link SyncPointData.barOccurence'
  },
  {
    id: 'upstream-declaring-class-regression',
    contract: 'unclassified-source-dispatch',
    category: 'ownership',
    file: 'conformance/upstream-inventory.json',
    before: '      "name": "SyncPointData.barOccurence",\n      "kind": "field",',
    after: '      "name": "Automation.barOccurence",\n      "kind": "field",',
    command: 'upstream-check',
    want: 'AlphaTab upstream inventory is stale'
  },
  {
    id: 'automation-dispatch-diagnostic',
    category: 'diagnostic',
    file: 'gpif_automations.go',
    before: '\t\tcase "SustainPedal":\n',
    after: '\t\tcase "DisabledSustainPedal":\n',
    test: '^TestGPIFAutomationDispatchDiagnostics$',
    want: 'want invalid-data at path containing "Value"'
  },
  {
    id: 'removed-automation-diagnostic',
    contract: 'automation-dispatch-diagnostic',
    category: 'diagnostic',
    file: 'gpif_automations.go',
    before: '\t\t\tif !valueOK {\n\t\t\t\tcontext.add(diagnosticSource("GPIF.Track.Automation.SustainPedal.Value.Invalid", "sustain-pedal", ParseDiagnosticInvalidData), ParseDiagnostic{\n\t\t\t\t\tSourcePath: path + "/Value", ObjectID: track.ID,\n\t\t\t\t\tReason: fmt.Sprintf("sustain-pedal value %q must contain a finite value and reference 1 (down) or 3 (release)", automation.Value.Text),\n\t\t\t\t})\n\t\t\t\tcontinue\n\t\t\t}\n',
    after: '\t\t\tif !valueOK {\n\t\t\t\tcontinue\n\t\t\t}\n',
    test: '^TestGPIFAutomationDispatchDiagnostics$',
    want: 'want invalid-data at path containing "Value"'
  },
  {
    id: 'supported-effect-serialization',
    category: 'serialization',
    file: 'gp8_notation.go',
    before: '\t\tresult.Hairpin = "Crescendo"\n',
    after: '\t\tresult.Hairpin = ""\n',
    test: '^TestExportGP8PreservesHairpins$',
    want: 'first beat hairpin'
  },
  {
    id: 'beat-dynamic-normal-note-report',
    contract: 'beat-dynamic-quantization',
    category: 'loss-report',
    file: 'gp8_builder.go',
    before: '\tif dynamic.authoredNormalized {\n\t\tbuilder.addReport("gp8.normalize.beat-dynamic", "note-and-beat-semantics", ExportDispositionNormalized, location, "GPIF stores the authored beat dynamic as one of eight canonical markings")\n\t}\n',
    after: '\t_ = dynamic.authoredNormalized\n',
    test: '^TestGP8AuthoredDynamicCanonicalNoteRegression$',
    want: 'preflight codes = [], want [gp8.normalize.beat-dynamic]'
  },
  {
    id: 'beat-dynamic-rest-report',
    contract: 'beat-dynamic-quantization',
    category: 'loss-report',
    file: 'gp8_builder.go',
    before: '\tif dynamic.authoredNormalized {\n\t\tbuilder.addReport("gp8.normalize.beat-dynamic", "note-and-beat-semantics", ExportDispositionNormalized, location, "GPIF stores the authored beat dynamic as one of eight canonical markings")\n\t}\n',
    after: '\t_ = dynamic.authoredNormalized\n',
    test: '^TestGP8AuthoredDynamicRestRegression$',
    want: 'preflight codes = [], want [gp8.normalize.beat-dynamic]'
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
    file: 'gp8_builder.go',
    before: '\t\t\treturn float64(song.Tempo), true, nil\n',
    after: '\t\t\treturn float64(exact), true, nil\n',
    test: '^TestGP8ExportReconcilesSemanticAndLegacyTempo$',
    want: 'round-trip tempo'
  },
  {
    id: 'chord-occurrence-isolation',
    category: 'ownership',
    file: 'gpif_model.go',
    before: '\tclone.Strings = slices.Clone(source.Strings)\n',
    after: '\tclone.Strings = source.Strings\n',
    test: '^TestRepeatedGPIFChordOccurrencesOwnMutablePayloads$',
    want: 'second chord changed'
  },
  {
    id: 'master-bar-narrowing-boundary',
    category: 'numeric-boundary',
    file: 'gpif_parser.go',
    before: '\t\t\tif numeratorErr != nil || numerator <= 0 || numerator > math.MaxInt8 {\n',
    after: '\t\t\tif numeratorErr != nil || numerator <= 0 {\n',
    test: '^TestGPIFMasterBarValuesDoNotWrapAtLegacyBoundaries$',
    want: 'modulo_numerator'
  },
  {
    id: 'master-bar-denominator-boundary',
    category: 'numeric-boundary',
    file: 'gpif_parser.go',
    before: '\t\t\tif denominatorErr != nil || denominator <= 0 || denominator > math.MaxUint16 {\n',
    after: '\t\t\tif denominatorErr != nil || denominator <= 0 {\n',
    test: '^TestGPIFMasterBarValuesDoNotWrapAtLegacyBoundaries$',
    want: 'denominator_modulo_overflow'
  },
  {
    id: 'inspected-note-duration-percent',
    category: 'loss-report',
    file: 'gp8_builder.go',
    before: '\tif note.DurationPercent != 1 {\n\t\tbuilder.addReport("gp8.omit.note-duration-percent", "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 has no note duration-percent field")\n\t}\n',
    after: '',
    test: '^TestGP8StrictExportCoversInspectedSemanticFields$',
    want: 'note_duration_percent'
  },
  {
    id: 'isolated-midi-tremolo-loss-report',
    contract: 'isolated-midi-controller-report',
    category: 'loss-report',
    file: 'gp8_builder.go',
    before: ' || channel.Phaser != 0 || channel.Tremolo != 0 {\n',
    after: ' || channel.Phaser != 0 {\n',
    test: '^TestConformancePlaybackRouting$',
    want: 'isolated tremolo report'
  },
  {
    id: 'isolated-harmonic-octave-loss-report',
    contract: 'isolated-harmonic-member-report',
    category: 'loss-report',
    file: 'gp8_builder.go',
    before: '\t\tif harmonic.Pitch != nil || harmonic.Octave != nil {\n',
    after: '\t\tif harmonic.Pitch != nil {\n',
    test: '^TestConformanceHarmonicVariants$',
    want: 'isolated octave report'
  },
  {
    id: 'isolated-chord-show-loss-report',
    contract: 'isolated-chord-legacy-report',
    category: 'loss-report',
    file: 'gp8_builder.go',
    before: ' || chord.NewFormat != nil || chord.Show != nil {\n',
    after: ' || chord.NewFormat != nil {\n',
    test: '^TestConformanceChordDefinitions$',
    want: 'isolated show report'
  },
  {
    id: 'adapter-ownership-projection',
    contract: 'field-disposition-evidence',
    category: 'adapter',
    file: 'conformance_projection_test.go',
    before: '\t\t\t"index": track["index"], "name": track["name"], "staves": track["staves"],\n',
    after: '\t\t\t"index": track["index"], "name": track["name"], "staves": []any{},\n',
    test: '^TestStaffOwnershipProjectionDetectsContentMutations$',
    want: 'staff-ownership projection did not detect a second-staff pitch mutation'
  },
  {
    id: 'adapter-enum-collapse',
    contract: 'field-disposition-evidence',
    category: 'adapter',
    file: 'conformance_go_adapter_test.go',
    before: '\tcase SlideIntoFromAbove:\n\t\treturn "into-from-above"\n',
    after: '\tcase SlideIntoFromAbove:\n\t\treturn "into-from-below"\n',
    test: '^TestResilienceAdapterMutationSensitivity$',
    want: 'slide adapter collapsed distinct enum values'
  },
  {
    id: 'structural-export-resilience',
    category: 'serialization',
    file: 'gp8_builder.go',
    before: '\t\tbarIDs := make([]string, 0, len(builder.song.Tracks))\n\t\tfor trackIndex := range builder.song.Tracks {\n',
    after: '\t\tbarIDs := make([]string, 0, len(builder.song.Tracks))\n\t\tfor trackIndex := range min(2, len(builder.song.Tracks)) {\n',
    test: '^TestConformanceStructuralResilience$',
    want: 'measure count'
  },
  {
    id: 'unclassified-public-enum-member',
    category: 'inventory',
    file: 'enums.go',
    before: '\tNoteTypeDead   NoteType = 3\n',
    after: '\tNoteTypeDead   NoteType = 3\n\tNoteTypeReviewUnclassified NoteType = 99\n',
    test: '^TestSemanticContractInventory$',
    want: 'NoteType.NoteTypeReviewUnclassified'
  },
  {
    id: 'unclassified-public-enum-conversion',
    contract: 'unclassified-public-enum-member',
    category: 'inventory',
    file: 'enums.go',
    before: '\tNoteTypeDead   NoteType = 3\n',
    after: '\tNoteTypeDead   NoteType = 3\n\tNoteTypeReviewConversion = NoteType(99)\n',
    test: '^TestSemanticContractInventory$',
    want: 'NoteType.NoteTypeReviewConversion'
  },
  {
    id: 'unclassified-public-enum-arithmetic',
    contract: 'unclassified-public-enum-member',
    category: 'inventory',
    file: 'enums.go',
    before: '\tNoteTypeDead   NoteType = 3\n',
    after: '\tNoteTypeDead   NoteType = 3\n\tNoteTypeReviewArithmetic = NoteTypeDead + 1\n',
    test: '^TestSemanticContractInventory$',
    want: 'NoteType.NoteTypeReviewArithmetic'
  },
  {
    id: 'unclassified-public-enum-alias',
    contract: 'unclassified-public-enum-member',
    category: 'inventory',
    file: 'enums.go',
    before: '\tNoteTypeDead   NoteType = 3\n',
    after: '\tNoteTypeDead   NoteType = 3\n\tNoteTypeReviewAlias = NoteTypeDead\n',
    test: '^TestSemanticContractInventory$',
    want: 'NoteType.NoteTypeReviewAlias'
  },
  {
    id: 'binary-whammy-note-canonicalizer',
    contract: 'whammy-owner-context',
    category: 'ownership',
    file: 'beat.go',
    before: '\t\tbend, err := s.readBendEffect(c)\n',
    after: '\t\tbend, err := s.readNoteBendEffect(c)\n',
    test: '^TestParseBinaryWhammyPreservesDipsAndHolds$',
    want: 'bar 0 whammy points'
  },
  {
    id: 'negative-whammy-point-discard',
    contract: 'whammy-owner-context',
    category: 'curve-preservation',
    file: 'effects.go',
    before: '\t\tbe.Points = append(be.Points, bp)\n',
    after: '\t\tif bp.Value >= 0 {\n\t\t\tbe.Points = append(be.Points, bp)\n\t\t}\n',
    test: '^(TestParseBinaryWhammyPreservesDipsAndHolds|TestParseGP3TremoloUsesItsSeparateEncoding)$',
    want: 'whammy'
  },
  {
    id: 'gpif-whammy-note-canonicalizer',
    contract: 'whammy-owner-context',
    category: 'ownership',
    file: 'gpif_effects.go',
    before: '\treturn &BendEffect{Points: canonicalizeStandardWhammyPoints(points)}\n',
    after: '\treturn &BendEffect{Points: canonicalizeStandardBendPoints(points)}\n',
    test: '^TestConformanceWhammyContexts$',
    want: 'dispatch:gpifBeatWhammyProperties:property.Name'
  },
  {
    id: 'go-whammy-adapter-drop',
    contract: 'whammy-corpus-projection',
    category: 'adapter',
    file: 'conformance_go_adapter_test.go',
    before: '\t\t"whammy":         normalizeGoWhammy(beat.Effect.TremoloBar),\n',
    after: '\t\t"whammy":         nil,\n',
    test: '^TestWhammyProjectionAdaptersExposeBeatCurves$',
    want: 'Go corpus whammy projection'
  },
  {
    id: 'alpha-whammy-adapter-drop',
    contract: 'whammy-corpus-projection',
    category: 'adapter',
    file: 'conformance/oracle.mjs',
    before: '      tremoloPicking: beat.tremoloPicking ? 1 << (beat.tremoloPicking.marks + 2) : null,\n      whammy: normalizeWhammy(beat.whammyBarPoints),\n      notes\n',
    after: '      tremoloPicking: beat.tremoloPicking ? 1 << (beat.tremoloPicking.marks + 2) : null,\n      whammy: null,\n      notes\n',
    command: 'oracle-test',
    test: '^TestAlphaTabBinaryWhammyProjection$',
    want: 'bar 0 AlphaTab whammy'
  },
  {
    id: 'whammy-raw-point-only-preservation',
    contract: 'whammy-target-interpretation',
    category: 'loss-report',
    file: 'gp8_effects.go',
    before: '\t\tnormalized: !slices.Equal(simplifyBendPoints(canonicalizeStandardWhammyPoints(simplifyBendPoints(encoded))), points),\n',
    after: '\t\tnormalized: !slices.Equal(simplifyBendPoints(encoded), points),\n',
    test: '^TestGP8WhammyMiddleHoldReportsInterpretedLoss$',
    want: 'want target GP8 and codes [gp8.normalize.whammy-curve]'
  },
  {
    id: 'octave-variant-conformance',
    category: 'serialization',
    file: 'gp8_notation.go',
    before: '\tcase OctaveQuindicesimaBassa:\n\t\tresult.Ottavia = "15mb"\n',
    after: '\tcase OctaveQuindicesimaBassa:\n\t\tresult.Ottavia = ""\n',
    test: '^TestConformanceBeatEffects$',
    want: 'Octave.OctaveQuindicesimaBassa'
  },
  {
    id: 'semantic-wire-requires-behavior',
    category: 'classification',
    file: 'conformance/feature-ledger.json',
    replacements: [
      { before: '"gpifBeat.Ottavia":["M07-CLEF-OCTAVE","M10-BEAT-EFFECTS","M20-SOURCE-AUDIT"]', after: '"gpifBeat.Ottavia":["M20-SOURCE-AUDIT"]' },
      { before: '"gpifBars.Bars":"Collection wrapper;', after: '"gpifBeat.Ottavia":"Incorrect scalar structural classification.",\n      "gpifBars.Bars":"Collection wrapper;' }
    ],
    command: 'ledger-test',
    test: '^TestSemanticMatrixInventory$',
    want: 'gpifBeat.Ottavia is a scalar semantic leaf'
  },
  {
    id: 'semantic-wire-executable-assertion',
    category: 'executable-evidence',
    file: 'conformance_beat_test.go',
    before: '\t\trun.ClaimSerialization(claimSite("beat-octave", "export", "M10-BEAT-EFFECTS", "all octave shifts")).Wire("gpifBeat.Ottavia", wire, source.value)\n',
    after: '\t\t_ = wire\n',
    test: '^TestSemanticMatrixInventory$',
    want: 'semantic assertions for M10-BEAT-EFFECTS mismatch'
  },
  {
    id: 'semantic-obligation-evidence-role',
    contract: 'semantic-obligation-shape',
    category: 'classification',
    file: 'conformance/feature-ledger.json',
    before: '"id":"M10-BEAT-SEMANTICS","family":"M10","evidenceRole":"behavior"',
    after: '"id":"M10-BEAT-SEMANTICS","family":"M10","evidenceRole":"structural"',
    command: 'ledger-test',
    test: '^TestSemanticMatrixInventory$',
    want: 'structural semantic matrix case M10-BEAT-SEMANTICS has no schema-round-trip source'
  },
  {
    id: 'semantic-obligation-evidence-source',
    contract: 'semantic-obligation-shape',
    category: 'classification',
    file: 'conformance/feature-ledger.json',
    before: '"id":"M10-BEAT-SEMANTICS","family":"M10","evidenceRole":"behavior","evidenceSources":["public-api","independent-wire","diagnostic-policy"]',
    after: '"id":"M10-BEAT-SEMANTICS","family":"M10","evidenceRole":"behavior","evidenceSources":["schema-round-trip"]',
    command: 'ledger-test',
    test: '^TestSemanticMatrixInventory$',
    want: 'behavior semantic matrix case M10-BEAT-SEMANTICS relies only on structural schema evidence'
  },
  {
    id: 'semantic-obligation-format',
    contract: 'semantic-obligation-shape',
    category: 'classification',
    file: 'conformance/feature-ledger.json',
    before: '"test":"TestConformanceBeatSemantics","formats":["GP8","programmatic"]',
    after: '"test":"TestConformanceBeatSemantics","formats":[]',
    command: 'ledger-test',
    test: '^TestSemanticMatrixInventory$',
    want: 'semantic matrix case M10-BEAT-SEMANTICS has incomplete executable evidence'
  },
  {
    id: 'semantic-obligation-stage',
    contract: 'semantic-obligation-shape',
    category: 'classification',
    file: 'conformance/feature-ledger.json',
    before: '"id":"M10-BEAT-SEMANTICS","family":"M10","evidenceRole":"behavior","evidenceSources":["public-api","independent-wire","diagnostic-policy"],"test":"TestConformanceBeatSemantics","formats":["GP8","programmatic"],"stages":["import","programmatic","finalization","preflight","export","policy"]',
    after: '"id":"M10-BEAT-SEMANTICS","family":"M10","evidenceRole":"behavior","evidenceSources":["public-api","independent-wire","diagnostic-policy"],"test":"TestConformanceBeatSemantics","formats":["GP8","programmatic"],"stages":[]',
    command: 'ledger-test',
    test: '^TestSemanticMatrixInventory$',
    want: 'semantic matrix case M10-BEAT-SEMANTICS has incomplete executable evidence'
  },
  {
    id: 'semantic-obligation-value-shape',
    contract: 'semantic-obligation-shape',
    category: 'classification',
    file: 'conformance/feature-ledger.json',
    before: '"values":["normal, rest, and empty status","text on notes and rests","rest dynamics","beat dynamic distinct from note velocity","rest status"]',
    after: '"values":[]',
    command: 'ledger-test',
    test: '^TestSemanticMatrixInventory$',
    want: 'semantic matrix case M10-BEAT-SEMANTICS has incomplete executable evidence'
  },
  {
    id: 'narrowed-semantic-obligation',
    contract: 'semantic-obligation-shape',
    category: 'classification',
    file: 'conformance/feature-ledger.json',
    before: '{"id":"M10-DYNAMIC-QUANTIZATION","family":"M10","evidenceRole":"behavior","evidenceSources":["public-api","independent-consumer","independent-wire","diagnostic-policy"],"test":"TestConformanceDynamicQuantization","formats":["GP8","programmatic"],"stages":["import","programmatic","validation","preflight","export","policy","oracle"],"values":["all PPP through FFF canonical normal and rest values","every midpoint and adjacent value","endpoint clamping","authored and note combinations","rest, empty, absent, and chord beats","invalid -1 and 128"],"oracle":"Hand-authored target markings, GPIF leaf extraction, Go reimport, and pinned AlphaTab normal and rest output.","limitations":["GPIF stores one quantized dynamic for the complete beat."]}',
    after: '{"id":"M10-DYNAMIC-QUANTIZATION","family":"M10","evidenceRole":"behavior","evidenceSources":["public-api"],"test":"TestConformanceDynamicQuantization","formats":["programmatic"],"stages":["programmatic"],"values":["canonical"],"oracle":"Self-assigned public value","limitations":[]}',
    command: 'ledger-test',
    test: '^TestSemanticMatrixInventory$',
    want: 'semantic matrix obligation digest changed'
  },
  {
    id: 'percussion-resolved-staff-resources',
    category: 'serialization',
    file: 'gp8_percussion.go',
    before: '\tvalues := make([]int16, 0)\n\tseen := make(map[int16]struct{})\n\taddFallback := func(value int16) {\n\t\tif _, ok := seen[value]; ok {\n\t\t\treturn\n\t\t}\n\t\tseen[value] = struct{}{}\n\t\tvalues = append(values, value)\n\t}\n\tfor _, staff := range gp8ExportStaves(track) {\n',
    after: '\tvalues := make([]int16, 0)\n\tseen := make(map[int16]struct{})\n\taddFallback := func(value int16) {\n\t\tif _, ok := seen[value]; ok {\n\t\t\treturn\n\t\t}\n\t\tseen[value] = struct{}{}\n\t\tvalues = append(values, value)\n\t}\n\tfor _, staff := range gp8ExportStaves(track)[:1] {\n',
    test: '^TestGP8PercussionUsesEveryResolvedStaff$',
    want: 'has no exported articulation resource'
  },
  {
    id: 'percussion-resource-order',
    category: 'serialization',
    file: 'gpif_types.go',
    before: '\tGeneralMidi      *gpifGeneralMidi    `xml:"GeneralMidi,omitempty"`\n\tStaves           gpifStaves          `xml:"Staves"`\n\tInstrumentSet    *gpifInstrumentSet  `xml:"InstrumentSet,omitempty"`\n\tNotationPatch    *gpifInstrumentSet  `xml:"NotationPatch,omitempty"`\n',
    after: '\tInstrumentSet    *gpifInstrumentSet  `xml:"InstrumentSet,omitempty"`\n\tNotationPatch    *gpifInstrumentSet  `xml:"NotationPatch,omitempty"`\n\tGeneralMidi      *gpifGeneralMidi    `xml:"GeneralMidi,omitempty"`\n\tStaves           gpifStaves          `xml:"Staves"`\n',
    test: '^TestGP8PercussionUsesEveryResolvedStaff$',
    want: 'track resource order'
  },
  {
    id: 'percussion-articulation-lookup',
    category: 'serialization',
    file: 'gp8_notation.go',
    before: '\t\t\tvar ok bool\n\t\t\tarticulation, ok = builder.articulationIDs[trackIndex][note.Value]\n\t\t\tif !ok {\n\t\t\t\treturn "", fmt.Errorf("percussion MIDI value %d has no exported articulation resource", note.Value)\n\t\t\t}\n',
    after: '\t\t\tarticulation = builder.articulationIDs[trackIndex][note.Value]\n',
    test: '^TestGP8PercussionRejectsMissingArticulationResource$',
    want: 'missing articulation resource error = <nil>'
  },
  {
    id: 'percussion-line-count-policy',
    category: 'loss-report',
    file: 'gp8_builder.go',
    before: '\t\t\tif lineCount != percussionLineCount {\n',
    after: '\t\t\tif staffIndex < 0 && lineCount != percussionLineCount {\n',
    test: '^TestGP8PercussionLineCountPolicy$',
    want: 'want one staff-1 line-count normalization'
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
const oracleMutationPath = path.join(root, 'conformance', `.sensitivity-oracle-${process.pid}.mjs`);
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
    const mutated = mutation.command === 'oracle-test'
      ? oracleMutationPath
      : path.join(temporary, `${mutation.id}-${path.basename(mutation.file)}`);
    fs.writeFileSync(mutated, mutatedSource);
    let result;
    if (mutation.command === 'verify') {
      result = spawnSync('node', ['conformance/verify.mjs'], {
        cwd: root, encoding: 'utf8', env: { ...process.env, SEMANTIC_LEDGER_OVERLAY: mutated }
      });
    } else if (mutation.command === 'capability-test') {
      result = spawnSync('python3', ['-B', '-m', 'unittest', 'discover', '-s', 'conformance/capabilities', '-p', 'test_manage.py'], {
        cwd: root, encoding: 'utf8', env: { ...process.env, UPSTREAM_SENSITIVITY_OVERLAY: mutated }
      });
    } else if (mutation.command === 'capability-check') {
      result = spawnSync('python3', ['-B', 'conformance/capabilities/manage.py', 'check'], {
        cwd: root, encoding: 'utf8', env: { ...process.env, UPSTREAM_OWNERSHIP_OVERLAY: mutated }
      });
    } else if (mutation.command === 'upstream-check') {
      result = spawnSync('node', ['conformance/sync-upstream-inventory.mjs', '--check'], {
        cwd: root, encoding: 'utf8', env: { ...process.env, UPSTREAM_INVENTORY_OVERLAY: mutated }
      });
    } else if (mutation.command === 'oracle-test') {
      result = spawnSync('go', ['test', '-count=1', '-run', mutation.test, '.'], {
        cwd: root,
        encoding: 'utf8',
        env: { ...process.env, ALPHATAB_CONFORMANCE: '1', ALPHATAB_ORACLE_OVERLAY: mutated }
      });
    } else if (mutation.command === 'ledger-test' || mutation.command === 'alphatab-ledger-test') {
      result = spawnSync('go', ['test', '-count=1', '-run', mutation.test, '.'], {
        cwd: root,
        encoding: 'utf8',
        env: {
          ...process.env,
          SEMANTIC_LEDGER_OVERLAY: mutated,
          ...(mutation.command === 'alphatab-ledger-test' ? { ALPHATAB_CONFORMANCE: '1' } : {})
        }
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
    const expectedFailure = ['verify', 'capability-test', 'capability-check', 'upstream-check'].includes(mutation.command) || output.includes('--- FAIL:');
    if (!expectedFailure || !output.includes(mutation.want)) {
      throw new Error(`${mutation.id} failed for an unexpected reason:\n${output}`);
    }
    process.stdout.write(`killed ${mutation.id} [${mutation.category}]\n`);
  }
} finally {
  fs.rmSync(temporary, { recursive: true, force: true });
  fs.rmSync(oracleMutationPath, { force: true });
}
