// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

type conformanceRun struct {
	t                 *testing.T
	assertions        map[string]int
	fieldDispositions map[string]string
}

func newConformanceRun(t *testing.T) *conformanceRun {
	t.Helper()
	return &conformanceRun{
		t:                 t,
		assertions:        make(map[string]int),
		fieldDispositions: make(map[string]string),
	}
}

func (run *conformanceRun) Field(path string, got, want any) {
	run.equal("field", path, got, want)
}

func (run *conformanceRun) Preserved(path string, got, want any) {
	run.fieldWithDisposition(path, "preserved", got, want)
}

func (run *conformanceRun) Normalized(path string, got, want any) {
	run.fieldWithDisposition(path, "normalized", got, want)
}

func (run *conformanceRun) Omitted(path string, got, want any) {
	run.fieldWithDisposition(path, "omitted", got, want)
}

func (run *conformanceRun) Rejected(path string, got, want any) {
	run.fieldWithDisposition(path, "rejected", got, want)
}

func (run *conformanceRun) Derived(path string, got, want any) {
	run.fieldWithDisposition(path, "derived", got, want)
}

func (run *conformanceRun) OutOfScope(path string, got, want any) {
	run.fieldWithDisposition(path, "out-of-scope", got, want)
}

func (run *conformanceRun) fieldWithDisposition(path, disposition string, got, want any) {
	run.t.Helper()
	if previous, ok := run.fieldDispositions[path]; ok && previous != disposition {
		run.t.Errorf("field:%s has conflicting executable dispositions %s and %s", path, previous, disposition)
	}
	run.fieldDispositions[path] = disposition
	run.equal("field", path, got, want)
}

func (run *conformanceRun) Wire(path string, got, want any) {
	run.equal("wire", path, got, want)
}

func (run *conformanceRun) Dispatch(path string, got, want any) {
	run.equal("dispatch", path, got, want)
}

func (run *conformanceRun) Enum(path string, got, want any) {
	run.equal("enum", path, got, want)
}

func (run *conformanceRun) equal(kind, path string, got, want any) {
	run.t.Helper()
	key := kind + ":" + path
	run.assertions[key]++
	if !reflect.DeepEqual(got, want) {
		run.t.Errorf("%s = %#v, want %#v", key, got, want)
	}
}

func (run *conformanceRun) constructs() []string {
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

var conformanceExecutors = map[string]func(*conformanceRun){
	"TestConformanceMetadataImport":                     runConformanceMetadataImport,
	"TestConformanceMetadataCorpus":                     runConformanceMetadataCorpus,
	"TestConformanceMetadataExportPolicy":               runConformanceMetadataExportPolicy,
	"TestConformanceBinaryClipboard":                    runConformanceBinaryClipboard,
	"TestConformanceOracleKeepsAuthorAndWriterDistinct": runConformanceOracleKeepsAuthorAndWriterDistinct,
	"TestConformanceDisplayExportPolicy":                runConformanceDisplayExportPolicy,
	"TestConformancePageSetupImport":                    runConformancePageSetupImport,
	"TestConformanceOwnershipImport":                    runConformanceOwnershipImport,
	"TestConformanceCompatibilityAuthority":             runConformanceCompatibilityAuthority,
	"TestConformanceBarCardinalityDiagnostics":          runConformanceBarCardinalityDiagnostics,
	"TestConformanceInstrumentContext":                  runConformanceInstrumentContext,
	"TestConformancePlaybackRouting":                    runConformancePlaybackRouting,
	"TestConformanceTempoAuthority":                     runConformanceTempoAuthority,
	"TestConformanceMasterBars":                         runConformanceMasterBars,
	"TestConformanceAuthorityAndBoundaries":             runConformanceAuthorityAndBoundaries,
	"TestConformanceDurations":                          runConformanceDurations,
	"TestConformanceExactTiming":                        runConformanceExactTiming,
	"TestConformanceNoteRepresentation":                 runConformanceNoteRepresentation,
	"TestConformanceNoteValidation":                     runConformanceNoteValidation,
	"TestConformanceBeatSemantics":                      runConformanceBeatSemantics,
	"TestConformanceBeatEffects":                        runConformanceBeatEffects,
	"TestConformanceDynamicQuantization":                runConformanceDynamicQuantization,
	"TestConformanceTechniqueDispositions":              runConformanceTechniqueDispositions,
	"TestConformanceSourceDistinctions":                 runConformanceSourceDistinctions,
	"TestConformanceValidation":                         runConformanceValidation,
	"TestConformanceCurvePreservation":                  runConformanceCurvePreservation,
	"TestConformanceWhammyContexts":                     runConformanceWhammyContexts,
	"TestGP8WhammyMiddleHoldReportsInterpretedLoss":     runConformanceWhammyTargetInterpretation,
	"TestConformanceCurveLossPolicy":                    runConformanceCurveLossPolicy,
	"TestConformanceCurveValidation":                    runConformanceCurveValidation,
	"TestConformanceHarmonicVariants":                   runConformanceHarmonicVariants,
	"TestConformanceGPIFCompatibilityAndDiagnostics":    runConformanceGPIFCompatibilityAndDiagnostics,
	"TestConformanceHarmonicAuthorityAndValidation":     runConformanceHarmonicAuthorityAndValidation,
	"TestConformanceChordDefinitions":                   runConformanceChordDefinitions,
	"TestConformanceChordDefaultsAndLength":             runConformanceChordDefaultsAndLength,
	"TestConformanceChordScopeAndIsolation":             runConformanceChordScopeAndIsolation,
	"TestConformanceOrderedGraceExport":                 runConformanceOrderedGraceExport,
	"TestConformanceGraceAuthorityAndLoss":              runConformanceGraceAuthorityAndLoss,
	"TestConformanceBinaryGraceSource":                  runConformanceBinaryGraceSource,
	"TestConformanceCombinedGraceEffects":               runConformanceCombinedGraceEffects,
	"TestConformancePercussionIdentity":                 runConformancePercussionIdentity,
	"TestConformanceSourceAndValidation":                runConformanceSourceAndValidation,
	"TestConformanceNoteheadOptions":                    runConformanceNoteheadOptions,
	"TestConformanceAutomationSemantics":                runConformanceAutomationSemantics,
	"TestConformanceSourceDispatchAndDiagnostics":       runConformanceSourceDispatchAndDiagnostics,
	"TestConformanceAutomationValidation":               runConformanceAutomationValidation,
	"TestConformanceBinaryMixTable":                     runConformanceBinaryMixTable,
	"TestConformanceLyricScopes":                        runConformanceLyricScopes,
	"TestConformanceBinaryScoreLyrics":                  runConformanceBinaryScoreLyrics,
	"TestConformanceSourceLyricsDispatch":               runConformanceSourceLyricsDispatch,
	"TestConformanceBackingAssetsAndSyncPoints":         runConformanceBackingAssetsAndSyncPoints,
	"TestConformanceBackingSourceDiagnostics":           runConformanceBackingSourceDiagnostics,
	"TestConformanceValidationAndExportPolicy":          runConformanceValidationAndExportPolicy,
	"TestConformanceSourceAudit":                        runConformanceSourceAudit,
	"TestConformanceBehaviorReconciliation":             runConformanceBehaviorReconciliation,
	"TestConformanceSharedValidation":                   runConformanceSharedValidation,
	"TestConformanceAuthoredAndDerivedDistinction":      runConformanceAuthoredAndDerivedDistinction,
	"TestConformanceFinalizeContracts":                  runConformanceFinalizeContracts,
	"TestConformanceReadOnlyAndFileContracts":           runConformanceReadOnlyAndFileContracts,
	"TestConformanceMultiStaffContext":                  runConformanceMultiStaffContext,
	"TestConformancePickupTupletTempo":                  runConformancePickupTupletTempo,
	"TestConformanceChordOccurrence":                    runConformanceChordOccurrence,
	"TestConformanceGraceCombinations":                  runConformanceGraceCombinations,
	"TestConformanceEffectsAndDynamics":                 runConformanceEffectsAndDynamics,
	"TestConformanceLyricsPlaybackAutomation":           runConformanceLyricsPlaybackAutomation,
	"TestConformanceBackingSyncTempo":                   runConformanceBackingSyncTempo,
	"TestConformanceSelectiveAllowlist":                 runConformanceSelectiveAllowlist,
	"TestConformanceLegacyEditAuthority":                runConformanceLegacyEditAuthority,
	"TestConformanceWholeCorpusAccounting":              runConformanceWholeCorpusAccounting,
	"TestConformanceOracleCorrectness":                  runConformanceOracleCorrectness,
	"TestConformancePublicAPIEnums":                     runConformancePublicAPIEnums,
	"TestConformanceStructuralResilience":               runConformanceStructuralResilience,
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
