// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
)

type conformanceIndependentClaimReceipt struct {
	Site       conformanceClaimSite
	Obligation string
}

type conformanceIndependentClaimRecorder struct {
	t        *testing.T
	receipts map[string]conformanceIndependentClaimReceipt
}

var conformanceIndependentRecorders sync.Map

var conformanceIndependentEvidenceExecutors = map[string]func(*testing.T){
	"alphatab-automation":    runConformanceAutomationIndependentEvidence,
	"alphatab-backing-track": TestConformanceBackingTrackExport,
	"alphatab-beat-octave":   TestConformanceBeatEffects,
	"alphatab-capo":          TestAlphaTabPreservesIndependentStaffCapos,
	"alphatab-clef-octave":   TestAlphaTabPreservesClefOctaves,
	"alphatab-directions":    TestAlphaTabPreservesDirections,
	"alphatab-dynamics":      TestAlphaTabGP8ReadsNormalAndRestDynamic,
	"alphatab-fermata":       TestAlphaTabPreservesFermatas,
	"alphatab-free-time":     TestAlphaTabPreservesFreeTime,
	"alphatab-key":           TestAlphaTabPreservesKeyModes,
	"alphatab-legato":        TestAlphaTabPreservesLegato,
	"alphatab-midi-bank":     TestAlphaTabPreservesMIDIBanks,
	"alphatab-ownership":     TestAlphaTabMultiStaffTrackOrdering,
	"alphatab-simile":        TestAlphaTabPreservesSimileMarks,
	"alphatab-sustain-pedal": TestAlphaTabPreservesSustainPedals,
	"alphatab-tempo":         TestAlphaTabTempoReferences,
	"alphatab-timing":        TestAlphaTabGPIFTiming,
	"alphatab-transposition": TestAlphaTabPreservesTransposition,
	"alphatab-tremolo":       TestAlphaTabPreservesTremoloPicking,
	"alphatab-barre":         TestAlphaTabPreservesBeatBarres,
	"alphatab-beaming":       TestAlphaTabPreservesBeaming,
	"alphatab-beat-lyrics":   TestAlphaTabPreservesBeatLyrics,
	"alphatab-beat-vibrato":  TestAlphaTabPreservesBeatVibrato,
	"alphatab-beat-fade":     TestAlphaTabPreservesBeatFade,
	"alphatab-dead-slap":     TestAlphaTabPreservesDeadSlap,
	"alphatab-brush":         TestAlphaTabPreservesBrush,
	"alphatab-chord-diagram": TestAlphaTabPreservesChordDiagrams,
	"alphatab-tuning":        TestAlphaTabPreservesTuningLabels,
}

func runConformanceAutomationIndependentEvidence(t *testing.T) {
	TestConformanceAutomationSemantics(t)
	TestConformanceSourceDispatchAndDiagnostics(t)
}

func conformanceIndependentClaim(t *testing.T, obligation string, sites ...conformanceClaimSite) {
	t.Helper()
	var recorder *conformanceIndependentClaimRecorder
	conformanceIndependentRecorders.Range(func(name, candidate any) bool {
		if strings.HasPrefix(t.Name(), name.(string)) {
			recorder = candidate.(*conformanceIndependentClaimRecorder)
			return false
		}
		return true
	})
	if recorder == nil {
		return
	}
	for _, site := range sites {
		key := site.Capability + ":" + site.Stage
		receipt := conformanceIndependentClaimReceipt{Site: site, Obligation: obligation}
		if previous, exists := recorder.receipts[key]; exists && previous != receipt {
			recorder.t.Errorf("independent claim %s emitted conflicting receipts %#v and %#v", key, previous, receipt)
			continue
		}
		recorder.receipts[key] = receipt
	}
}

func TestConformanceCapabilityIndependentEvidence(t *testing.T) {
	requireAlphaTabConformance(t)
	ledger := readSemanticContractLedger(t)
	want := make(map[string]map[string]conformanceIndependentClaimReceipt)
	for _, claim := range ledger.SemanticMatrix.CapabilityClaims {
		if claim.IndependentEvidence == nil {
			continue
		}
		id := claim.IndependentEvidence.ID
		if want[id] == nil {
			want[id] = make(map[string]conformanceIndependentClaimReceipt)
		}
		key := claim.Capability + ":" + claim.Stage
		want[id][key] = conformanceIndependentClaimReceipt{
			Site: conformanceClaimSite{
				Capability:     claim.Capability,
				Stage:          claim.Stage,
				Case:           claim.Case,
				Source:         claim.Source,
				Value:          claim.Value,
				AssertionStage: claim.AssertionStage,
			},
			Obligation: claim.Obligation,
		}
	}
	ids := make([]string, 0, len(conformanceIndependentEvidenceExecutors))
	for id := range conformanceIndependentEvidenceExecutors {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			recorder := &conformanceIndependentClaimRecorder{t: t, receipts: make(map[string]conformanceIndependentClaimReceipt)}
			conformanceIndependentRecorders.Store(t.Name(), recorder)
			defer conformanceIndependentRecorders.Delete(t.Name())
			conformanceIndependentEvidenceExecutors[id](t)
			if !reflect.DeepEqual(recorder.receipts, want[id]) {
				t.Errorf("independent evidence %s receipts = %#v, want %#v", id, recorder.receipts, want[id])
			}
		})
	}
}
