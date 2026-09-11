// SPDX-License-Identifier: MIT
package integration_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestHarmonicSpellingSelectivePolicy(t *testing.T) {
	for _, kind := range []gp.HarmonicType{gp.HarmonicTypeNatural, gp.HarmonicTypeArtificial} {
		for mask := 0; mask < 4; mask++ {
			for _, legacy := range []int8{2, 3} {
				song := dynamicPolicySong(t, 47, false)
				song.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[1].Velocity = 47
				exact := 2.4
				h := &gp.HarmonicEffect{Kind: kind, Fret: &legacy, FretFloat: &exact}
				pitch := gp.PitchClass{Note: "C#", Just: 0, Accidental: 1, Value: 1, Sharp: true}
				oct := gp.OctaveOttava
				if mask&1 != 0 {
					h.Pitch = &pitch
				}
				if mask&2 != 0 {
					h.Octave = &oct
				}
				song.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[1].Effect.Harmonic = h
				if ds := gp.ValidateSong(song); len(ds) != 0 {
					t.Fatal(ds)
				}
				before, _ := json.Marshal(song)
				want := map[string]gp.ExportDisposition{}
				if mask != 0 {
					want["gp8.omit.harmonic-pitch"] = gp.ExportDispositionOmitted
				}
				if legacy == 3 {
					want["gp8.normalize.harmonic-fret-authority"] = gp.ExportDispositionNormalized
				}
				for _, allowed := range [][]string{nil, {"gp8.omit.harmonic-pitch"}, {"gp8.normalize.harmonic-fret-authority"}, {"gp8.omit.harmonic-pitch", "gp8.normalize.harmonic-fret-authority"}, {"gp8.omit.note-duration-percent"}} {
					opts := gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}}
					b, r, e := gp.ExportWithReport(song, gp.ExportFormatGP8, opts)
					if !reflect.DeepEqual(r, gp.PreflightExport(song, gp.ExportFormatGP8, opts)) || len(r.Entries) != len(want) {
						t.Fatalf("report %#v want %#v", r, want)
					}
					blocked := false
					for _, entry := range r.Entries {
						disposition, ok := want[entry.Code]
						if !ok || entry.Disposition != disposition || entry.Location != (gp.ScoreLocation{Beat: 1, Note: 1}) {
							t.Fatal(entry)
						}
						found := false
						for _, code := range allowed {
							found = found || code == entry.Code
						}
						blocked = blocked || !found
					}
					if blocked {
						var loss *gp.ExportLossError
						if len(b) != 0 || !errors.As(e, &loss) {
							t.Fatalf("strict bypass %d %v", len(b), e)
						}
						continue
					}
					if e != nil {
						t.Fatal(e)
					}
					parsed, e := gp.Parse(b)
					if e != nil {
						t.Fatal(e)
					}
					note := parsed.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[1]
					got := note.Effect.Harmonic
					if note.String != 2 || note.Value != 5 || got.Kind != kind || got.FretFloat == nil || *got.FretFloat != 2.4 || got.Pitch != nil || got.Octave != nil {
						t.Fatalf("allowed output %#v", note)
					}
				}
				after, _ := json.Marshal(song)
				if string(before) != string(after) {
					t.Fatal("source mutated")
				}
			}
		}
	}
}
