// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unicode"
)

type conformanceRun struct {
	t                 *testing.T
	assertions        map[string]int
	fieldDispositions map[string]string
	claimReceipts     map[string]*conformanceClaimReceipt
	pendingClaims     []conformanceClaimSite
	pendingComponent  string
}

type conformanceClaimSite struct {
	Capability     string
	Stage          string
	Case           string
	Source         semanticClaimSource
	Value          string
	AssertionStage string
}

type conformanceClaimReceipt struct {
	Site            conformanceClaimSite
	Obligation      string
	Serialization   string
	ReportAssertion string
}

func newConformanceRun(t *testing.T) *conformanceRun {
	t.Helper()
	return &conformanceRun{
		t:                 t,
		assertions:        make(map[string]int),
		fieldDispositions: make(map[string]string),
		claimReceipts:     make(map[string]*conformanceClaimReceipt),
	}
}

func claimSite(capability, stage, caseID, value string) conformanceClaimSite {
	assertionStage := stage
	if stage == "model" {
		assertionStage = "programmatic"
	}
	return claimSiteAtStage(capability, stage, caseID, value, assertionStage)
}

func claimSiteAtStage(capability, stage, caseID, value, assertionStage string) conformanceClaimSite {
	return conformanceClaimSite{
		Capability: capability,
		Stage:      stage,
		Case:       caseID,
		Source: semanticClaimSource{
			Kind: "matrix-scenario",
			ID:   caseID + "/" + capability + "/" + conformanceClaimSlug(value),
		},
		Value:          value,
		AssertionStage: assertionStage,
	}
}

func claimImportModel(capability, caseID, value string) []conformanceClaimSite {
	return []conformanceClaimSite{
		claimSite(capability, "import", caseID, value),
		claimSite(capability, "model", caseID, value),
	}
}

func claimAllStages(capability, caseID, value string) []conformanceClaimSite {
	return append(claimImportModel(capability, caseID, value), claimSite(capability, "export", caseID, value))
}

func conformanceClaimSlug(value string) string {
	var result strings.Builder
	separator := false
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if separator && result.Len() != 0 {
				result.WriteByte('-')
			}
			result.WriteRune(r)
			separator = false
		} else {
			separator = true
		}
	}
	return result.String()
}

func (run *conformanceRun) ClaimPrimary(sites ...conformanceClaimSite) *conformanceRun {
	return run.claimComponent("primary", sites)
}

func (run *conformanceRun) ClaimSerialization(sites ...conformanceClaimSite) *conformanceRun {
	return run.claimComponent("serialization", sites)
}

func (run *conformanceRun) ClaimReport(sites ...conformanceClaimSite) *conformanceRun {
	return run.claimComponent("report", sites)
}

func (run *conformanceRun) claimComponent(component string, sites []conformanceClaimSite) *conformanceRun {
	run.t.Helper()
	if run.pendingComponent != "" {
		run.t.Fatalf("claim component %s was not consumed before %s", run.pendingComponent, component)
	}
	run.pendingComponent = component
	run.pendingClaims = sites
	return run
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

func (run *conformanceRun) Report(path string, got, want any) {
	run.equal("report", path, got, want)
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
	component, sites := run.pendingComponent, run.pendingClaims
	run.pendingComponent = ""
	run.pendingClaims = nil
	if !reflect.DeepEqual(got, want) {
		run.t.Errorf("%s = %#v, want %#v", key, got, want)
		return
	}
	for _, site := range sites {
		run.recordClaimComponent(site, component, key)
	}
}

func (run *conformanceRun) recordClaimComponent(site conformanceClaimSite, component, assertion string) {
	run.t.Helper()
	key := site.Capability + ":" + site.Stage
	receipt := run.claimReceipts[key]
	if receipt == nil {
		receipt = &conformanceClaimReceipt{Site: site}
		run.claimReceipts[key] = receipt
	} else if receipt.Site != site {
		run.t.Errorf("claim site %s emitted conflicting identities %#v and %#v", key, receipt.Site, site)
		return
	}
	switch component {
	case "primary":
		receipt.Obligation = assertion
	case "serialization":
		receipt.Serialization = assertion
	case "report":
		receipt.ReportAssertion = assertion
	default:
		run.t.Fatalf("unknown claim component %q", component)
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
	"TestConformanceTuningLabels":                       runConformanceTuningLabels,
	"TestConformanceStaffCapo":                          runConformanceStaffCapo,
	"TestConformancePlaybackRouting":                    runConformancePlaybackRouting,
	"TestConformanceMIDIProgramReferences":              runConformanceMIDIProgramReferences,
	"TestConformanceMIDIBank":                           runConformanceMIDIBank,
	"TestConformanceTempoAuthority":                     runConformanceTempoAuthority,
	"TestConformanceMasterBars":                         runConformanceMasterBars,
	"TestConformanceKeyModes":                           runConformanceKeyModes,
	"TestConformanceDirections":                         runConformanceDirections,
	"TestConformanceClefOctave":                         runConformanceClefOctave,
	"TestConformanceFermatas":                           runConformanceFermatas,
	"TestConformanceFreeTime":                           runConformanceFreeTime,
	"TestConformanceAuthorityAndBoundaries":             runConformanceAuthorityAndBoundaries,
	"TestConformanceDurations":                          runConformanceDurations,
	"TestConformanceExactTiming":                        runConformanceExactTiming,
	"TestConformanceNoteRepresentation":                 runConformanceNoteRepresentation,
	"TestConformanceNoteValidation":                     runConformanceNoteValidation,
	"TestConformanceBeatSemantics":                      runConformanceBeatSemantics,
	"TestConformanceBeatEffects":                        runConformanceBeatEffects,
	"TestConformanceLegato":                             runConformanceLegato,
	"TestConformanceBarre":                              runConformanceBarre,
	"TestConformanceBeaming":                            runConformanceBeaming,
	"TestConformanceDynamicQuantization":                runConformanceDynamicQuantization,
	"TestConformanceTechniqueDispositions":              runConformanceTechniqueDispositions,
	"TestConformanceTremoloPicking":                     runConformanceTremoloPicking,
	"TestConformanceSourceDistinctions":                 runConformanceSourceDistinctions,
	"TestConformanceValidation":                         runConformanceValidation,
	"TestConformanceCurvePreservation":                  runConformanceCurvePreservation,
	"TestConformanceWhammyContexts":                     runConformanceWhammyContexts,
	"TestGP8WhammyMiddleHoldReportsInterpretedLoss":     runConformanceWhammyTargetInterpretation,
	"TestConformanceCurveLossPolicy":                    runConformanceCurveLossPolicy,
	"TestConformanceCurveValidation":                    runConformanceCurveValidation,
	"TestConformanceExactBendOffsets":                   runConformanceExactBendOffsets,
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
	"TestConformanceNativePercussionFallbacks":          runConformanceNativePercussionFallbacks,
	"TestConformanceNoteheadOptions":                    runConformanceNoteheadOptions,
	"TestConformanceAutomationSemantics":                runConformanceAutomationSemantics,
	"TestConformanceSourceDispatchAndDiagnostics":       runConformanceSourceDispatchAndDiagnostics,
	"TestConformanceAutomationValidation":               runConformanceAutomationValidation,
	"TestConformanceSustainPedals":                      runConformanceSustainPedals,
	"TestConformanceTransposition":                      runConformanceTransposition,
	"TestConformanceBinaryMixTable":                     runConformanceBinaryMixTable,
	"TestConformanceLyricScopes":                        runConformanceLyricScopes,
	"TestConformanceBinaryScoreLyrics":                  runConformanceBinaryScoreLyrics,
	"TestConformanceSourceLyricsDispatch":               runConformanceSourceLyricsDispatch,
	"TestConformanceBackingAssetsAndSyncPoints":         runConformanceBackingAssetsAndSyncPoints,
	"TestConformanceBackingTrackExport":                 runConformanceBackingTrackExport,
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
