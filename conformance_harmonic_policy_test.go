// SPDX-License-Identifier: MIT
package goguitarpro

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestConformanceHarmonicConsumerPolicy(t *testing.T) {
	runConformanceHarmonicConsumerPolicy(newConformanceRun(t))
}
func TestAlphaTabHarmonicConsumerPolicy(t *testing.T) {
	requireAlphaTabConformance(t)
	runConformanceHarmonicConsumerPolicy(newConformanceRun(t))
}
func runConformanceHarmonicConsumerPolicy(run *conformanceRun) {
	t := run.t
	for _, c := range []struct {
		kind           HarmonicType
		wire, consumer string
	}{{HarmonicTypeNatural, "Natural", "Natural"}, {HarmonicTypeArtificial, "Artificial", "Artificial"}, {HarmonicTypePinch, "Pinch", "Pinch"}, {HarmonicTypeTapped, "Tap", "Tap"}, {HarmonicTypeSemi, "Semi", "Semi"}, {HarmonicTypeFeedback, "Feedback", "Feedback"}} {
		for mask := 0; mask < 4; mask++ {
			if mask != 0 && c.kind != HarmonicTypeNatural && c.kind != HarmonicTypeArtificial {
				continue
			}
			song := conformanceHarmonicSong(t)
			fret := int8(2)
			exact := 2.4
			h := &HarmonicEffect{Kind: c.kind, Fret: &fret, FretFloat: &exact}
			pitch := PitchClass{Note: "C#", Just: 0, Accidental: 1, Value: 1, Sharp: true}
			octave := OctaveOttava
			if mask&1 != 0 {
				h.Pitch = &pitch
			}
			if mask&2 != 0 {
				h.Octave = &octave
			}
			conformanceHarmonicFirstNote(song).Effect.Harmonic = h
			codes := []string{}
			if mask != 0 {
				codes = []string{"gp8.omit.harmonic-pitch"}
			}
			before, _ := json.Marshal(song)
			data, report := assertConsumerLossPolicy(t, song, codes)
			run.Report("M13-HARMONIC-CONSUMER-POLICY", reportCodes(report), codes)
			if mask != 0 && (len(report.Entries) != 1 || report.Entries[0].Location != (ScoreLocation{}) || report.Entries[0].Disposition != ExportDispositionOmitted) {
				t.Fatal(report)
			}
			run.Preserved("HarmonicEffect.Kind", h.Kind, c.kind)
			run.Preserved("HarmonicEffect.FretFloat", *h.FretFloat, 2.4)
			run.Omitted("HarmonicEffect.Pitch", h.Pitch, func() *PitchClass {
				if mask&1 != 0 {
					return &pitch
				}
				return nil
			}())
			run.Omitted("HarmonicEffect.Octave", h.Octave, func() *Octave {
				if mask&2 != 0 {
					return &octave
				}
				return nil
			}())
			properties := extractHarmonicWire(t, data)
			run.Wire("gpifProperty.HType", conformanceHarmonicWireValue(properties, "HarmonicType", "HType"), c.wire)
			run.Wire("gpifProperty.HFret", conformanceHarmonicWireValue(properties, "HarmonicFret", "HFret"), "2.4")
			parsed, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			got := conformanceHarmonicFirstNote(parsed).Effect.Harmonic
			if got == nil || got.Kind != c.kind || got.FretFloat == nil || *got.FretFloat != 2.4 || got.Pitch != nil || got.Octave != nil {
				t.Fatalf("harmonic %#v", got)
			}
			after, _ := json.Marshal(song)
			if string(before) != string(after) {
				t.Fatal("export mutated source")
			}
			if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
				var facts []struct {
					Kind string
					Fret float64
				}
				readAlphaTabOracleFacts(t, "--harmonics", writeConformanceFixture(t, data), &facts)
				want := []struct {
					Kind string
					Fret float64
				}{{c.consumer, 2.4}}
				if !reflect.DeepEqual(facts, want) {
					t.Fatalf("consumer %#v want %#v", facts, want)
				}
			}
		}
	}
}
