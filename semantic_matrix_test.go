// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

type semanticMatrixRun struct {
	t                 *testing.T
	assertions        map[string]int
	fieldDispositions map[string]string
}

func newSemanticMatrixRun(t *testing.T) *semanticMatrixRun {
	t.Helper()
	return &semanticMatrixRun{
		t:                 t,
		assertions:        make(map[string]int),
		fieldDispositions: make(map[string]string),
	}
}

func (run *semanticMatrixRun) Field(path string, got, want any) {
	run.equal("field", path, got, want)
}

func (run *semanticMatrixRun) Preserved(path string, got, want any) {
	run.fieldWithDisposition(path, "preserved", got, want)
}

func (run *semanticMatrixRun) Normalized(path string, got, want any) {
	run.fieldWithDisposition(path, "normalized", got, want)
}

func (run *semanticMatrixRun) Omitted(path string, got, want any) {
	run.fieldWithDisposition(path, "omitted", got, want)
}

func (run *semanticMatrixRun) Rejected(path string, got, want any) {
	run.fieldWithDisposition(path, "rejected", got, want)
}

func (run *semanticMatrixRun) Derived(path string, got, want any) {
	run.fieldWithDisposition(path, "derived", got, want)
}

func (run *semanticMatrixRun) OutOfScope(path string, got, want any) {
	run.fieldWithDisposition(path, "out-of-scope", got, want)
}

func (run *semanticMatrixRun) fieldWithDisposition(path, disposition string, got, want any) {
	run.t.Helper()
	if previous, ok := run.fieldDispositions[path]; ok && previous != disposition {
		run.t.Errorf("field:%s has conflicting executable dispositions %s and %s", path, previous, disposition)
	}
	run.fieldDispositions[path] = disposition
	run.equal("field", path, got, want)
}

func (run *semanticMatrixRun) Wire(path string, got, want any) {
	run.equal("wire", path, got, want)
}

func (run *semanticMatrixRun) Dispatch(path string, got, want any) {
	run.equal("dispatch", path, got, want)
}

func (run *semanticMatrixRun) Enum(path string, got, want any) {
	run.equal("enum", path, got, want)
}

func (run *semanticMatrixRun) equal(kind, path string, got, want any) {
	run.t.Helper()
	key := kind + ":" + path
	run.assertions[key]++
	if !reflect.DeepEqual(got, want) {
		run.t.Errorf("%s = %#v, want %#v", key, got, want)
	}
}

func (run *semanticMatrixRun) constructs() []string {
	result := make([]string, 0, len(run.assertions))
	for construct, count := range run.assertions {
		if count == 0 {
			panic(fmt.Sprintf("semantic assertion %s did not execute", construct))
		}
		result = append(result, construct)
	}
	sort.Strings(result)
	return result
}

var semanticMatrixExecutors = map[string]func(*semanticMatrixRun){
	"TestSemanticMatrixM01MetadataImport":                     runSemanticMatrixM01MetadataImport,
	"TestSemanticMatrixM01MetadataCorpus":                     runSemanticMatrixM01MetadataCorpus,
	"TestSemanticMatrixM01MetadataExportPolicy":               runSemanticMatrixM01MetadataExportPolicy,
	"TestSemanticMatrixM01BinaryClipboard":                    runSemanticMatrixM01BinaryClipboard,
	"TestSemanticMatrixM01OracleKeepsAuthorAndWriterDistinct": runSemanticMatrixM01OracleKeepsAuthorAndWriterDistinct,
	"TestSemanticMatrixM02DisplayExportPolicy":                runSemanticMatrixM02DisplayExportPolicy,
	"TestSemanticMatrixM02PageSetupImport":                    runSemanticMatrixM02PageSetupImport,
	"TestSemanticMatrixM03OwnershipImport":                    runSemanticMatrixM03OwnershipImport,
	"TestSemanticMatrixM03CompatibilityAuthority":             runSemanticMatrixM03CompatibilityAuthority,
	"TestSemanticMatrixM03BarCardinalityDiagnostics":          runSemanticMatrixM03BarCardinalityDiagnostics,
	"TestSemanticMatrixM04InstrumentContext":                  runSemanticMatrixM04InstrumentContext,
	"TestSemanticMatrixM05PlaybackRouting":                    runSemanticMatrixM05PlaybackRouting,
	"TestSemanticMatrixM06TempoAuthority":                     runSemanticMatrixM06TempoAuthority,
	"TestSemanticMatrixM07MasterBars":                         runSemanticMatrixM07MasterBars,
	"TestSemanticMatrixM07AuthorityAndBoundaries":             runSemanticMatrixM07AuthorityAndBoundaries,
	"TestSemanticMatrixM08Durations":                          runSemanticMatrixM08Durations,
	"TestSemanticMatrixM08ExactTiming":                        runSemanticMatrixM08ExactTiming,
	"TestSemanticMatrixM09NoteRepresentation":                 runSemanticMatrixM09NoteRepresentation,
	"TestSemanticMatrixM09NoteValidation":                     runSemanticMatrixM09NoteValidation,
	"TestSemanticMatrixM10BeatSemantics":                      runSemanticMatrixM10BeatSemantics,
	"TestSemanticMatrixM10BeatEffects":                        runSemanticMatrixM10BeatEffects,
	"TestSemanticMatrixM10DynamicQuantization":                runSemanticMatrixM10DynamicQuantization,
	"TestSemanticMatrixM11TechniqueDispositions":              runSemanticMatrixM11TechniqueDispositions,
	"TestSemanticMatrixM11SourceDistinctions":                 runSemanticMatrixM11SourceDistinctions,
	"TestSemanticMatrixM11Validation":                         runSemanticMatrixM11Validation,
	"TestSemanticMatrixM12CurvePreservation":                  runSemanticMatrixM12CurvePreservation,
	"TestSemanticMatrixM12WhammyContexts":                     runSemanticMatrixM12WhammyContexts,
	"TestSemanticMatrixM12CurveLossPolicy":                    runSemanticMatrixM12CurveLossPolicy,
	"TestSemanticMatrixM12CurveValidation":                    runSemanticMatrixM12CurveValidation,
	"TestSemanticMatrixM13HarmonicVariants":                   runSemanticMatrixM13HarmonicVariants,
	"TestSemanticMatrixM13GPIFCompatibilityAndDiagnostics":    runSemanticMatrixM13GPIFCompatibilityAndDiagnostics,
	"TestSemanticMatrixM13HarmonicAuthorityAndValidation":     runSemanticMatrixM13HarmonicAuthorityAndValidation,
	"TestSemanticMatrixM14ChordDefinitions":                   runSemanticMatrixM14ChordDefinitions,
	"TestSemanticMatrixM14ChordDefaultsAndLength":             runSemanticMatrixM14ChordDefaultsAndLength,
	"TestSemanticMatrixM14ChordScopeAndIsolation":             runSemanticMatrixM14ChordScopeAndIsolation,
	"TestSemanticMatrixM15OrderedGraceExport":                 runSemanticMatrixM15OrderedGraceExport,
	"TestSemanticMatrixM15GraceAuthorityAndLoss":              runSemanticMatrixM15GraceAuthorityAndLoss,
	"TestSemanticMatrixM15BinaryGraceSource":                  runSemanticMatrixM15BinaryGraceSource,
	"TestSemanticMatrixM15CombinedGraceEffects":               runSemanticMatrixM15CombinedGraceEffects,
	"TestSemanticMatrixM16PercussionIdentity":                 runSemanticMatrixM16PercussionIdentity,
	"TestSemanticMatrixM16SourceAndValidation":                runSemanticMatrixM16SourceAndValidation,
	"TestSemanticMatrixM16NoteheadOptions":                    runSemanticMatrixM16NoteheadOptions,
	"TestSemanticMatrixM17AutomationSemantics":                runSemanticMatrixM17AutomationSemantics,
	"TestSemanticMatrixM17SourceDispatchAndDiagnostics":       runSemanticMatrixM17SourceDispatchAndDiagnostics,
	"TestSemanticMatrixM17AutomationValidation":               runSemanticMatrixM17AutomationValidation,
	"TestSemanticMatrixM17BinaryMixTable":                     runSemanticMatrixM17BinaryMixTable,
	"TestSemanticMatrixM18LyricScopes":                        runSemanticMatrixM18LyricScopes,
	"TestSemanticMatrixM18BinaryScoreLyrics":                  runSemanticMatrixM18BinaryScoreLyrics,
	"TestSemanticMatrixM18SourceLyricsDispatch":               runSemanticMatrixM18SourceLyricsDispatch,
	"TestSemanticMatrixM19BackingAssetsAndSyncPoints":         runSemanticMatrixM19BackingAssetsAndSyncPoints,
	"TestSemanticMatrixM19BackingSourceDiagnostics":           runSemanticMatrixM19BackingSourceDiagnostics,
	"TestSemanticMatrixM19ValidationAndExportPolicy":          runSemanticMatrixM19ValidationAndExportPolicy,
	"TestSemanticMatrixM20SourceAudit":                        runSemanticMatrixM20SourceAudit,
	"TestSemanticMatrixM20BehaviorReconciliation":             runSemanticMatrixM20BehaviorReconciliation,
	"TestSemanticMatrixM21SharedValidation":                   runSemanticMatrixM21SharedValidation,
	"TestSemanticMatrixM21AuthoredAndDerivedDistinction":      runSemanticMatrixM21AuthoredAndDerivedDistinction,
	"TestSemanticMatrixM21FinalizeContracts":                  runSemanticMatrixM21FinalizeContracts,
	"TestSemanticMatrixM21ReadOnlyAndFileContracts":           runSemanticMatrixM21ReadOnlyAndFileContracts,
	"TestSemanticMatrixM22MultiStaffContext":                  runSemanticMatrixM22MultiStaffContext,
	"TestSemanticMatrixM22PickupTupletTempo":                  runSemanticMatrixM22PickupTupletTempo,
	"TestSemanticMatrixM22ChordOccurrence":                    runSemanticMatrixM22ChordOccurrence,
	"TestSemanticMatrixM22GraceCombinations":                  runSemanticMatrixM22GraceCombinations,
	"TestSemanticMatrixM22EffectsAndDynamics":                 runSemanticMatrixM22EffectsAndDynamics,
	"TestSemanticMatrixM22LyricsPlaybackAutomation":           runSemanticMatrixM22LyricsPlaybackAutomation,
	"TestSemanticMatrixM22BackingSyncTempo":                   runSemanticMatrixM22BackingSyncTempo,
	"TestSemanticMatrixM22SelectiveAllowlist":                 runSemanticMatrixM22SelectiveAllowlist,
	"TestSemanticMatrixM22LegacyEditAuthority":                runSemanticMatrixM22LegacyEditAuthority,
	"TestSemanticMatrixM23WholeCorpusAccounting":              runSemanticMatrixM23WholeCorpusAccounting,
	"TestSemanticMatrixM24OracleCorrectness":                  runSemanticMatrixM24OracleCorrectness,
	"TestSemanticMatrixM25PublicAPIEnums":                     runSemanticMatrixM25PublicAPIEnums,
	"TestSemanticMatrixM25StructuralResilience":               runSemanticMatrixM25StructuralResilience,
}

func semanticValidGP8Song(t *testing.T) *Song {
	t.Helper()
	song := syntheticGP8Song()
	for index := range song.Tracks[0].Measures {
		song.Tracks[0].Measures[index].HeaderIndex = index
	}
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("semantic matrix baseline is invalid: %#v", diagnostics)
	}
	return song
}

func semanticValidPitchedGP8Song(t *testing.T) *Song {
	t.Helper()
	song := syntheticGP8Song()
	track := &song.Tracks[0]
	track.PercussionTrack = false
	track.FretCount = 24
	track.Strings = []GuitarString{{Number: 1, Value: 64}, {Number: 2, Value: 59}, {Number: 3, Value: 55}, {Number: 4, Value: 50}, {Number: 5, Value: 45}, {Number: 6, Value: 40}}
	track.PercussionArticulations = nil
	song.Channels[0] = MidiChannel{Channel: 0, EffectChannel: 1, Instrument: 25, Volume: 100, Balance: 64}
	for measureIndex := range track.Measures {
		track.Measures[measureIndex].HeaderIndex = measureIndex
		for voiceIndex := range track.Measures[measureIndex].Voices {
			for beatIndex := range track.Measures[measureIndex].Voices[voiceIndex].Beats {
				beat := &track.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex]
				for noteIndex := range beat.Notes {
					beat.Notes[noteIndex].Value = int16(noteIndex + 2)
					beat.Notes[noteIndex].String = int8(noteIndex%6 + 1)
					beat.Notes[noteIndex].HasPercussionArticulation = false
					beat.Notes[noteIndex].PercussionArticulation = 0
					for graceIndex := range beat.Notes[noteIndex].Effect.Graces {
						beat.Notes[noteIndex].Effect.Graces[graceIndex].Fret = 1
						fret := Fret(1)
						beat.Notes[noteIndex].Effect.Graces[graceIndex].ExactFret = &fret
						beat.Notes[noteIndex].Effect.Graces[graceIndex].HasPercussionArticulation = false
					}
				}
			}
		}
	}
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("semantic matrix pitched baseline is invalid: %#v", diagnostics)
	}
	return song
}
