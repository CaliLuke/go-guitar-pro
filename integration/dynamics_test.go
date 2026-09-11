// SPDX-License-Identifier: MIT

package integration_test

import (
	"errors"
	"reflect"
	"slices"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestGP8ChordAndRestDynamicPolicy(t *testing.T) {
	for _, test := range []struct {
		name    string
		dynamic int16
		rest    bool
		codes   []string
	}{
		{"chord fallback", 0, false, []string{"gp8.normalize.note-velocity"}},
		{"chord canonical", 47, false, []string{"gp8.normalize.note-velocity"}},
		{"chord quantized", 48, false, []string{"gp8.normalize.beat-dynamic", "gp8.normalize.note-velocity"}},
		{"rest canonical", 47, true, nil},
		{"rest quantized", 48, true, []string{"gp8.normalize.beat-dynamic"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := dynamicPolicySong(t, test.dynamic, test.rest)
			beat := &song.Tracks[0].Measures[0].Voices[0].Beats[1]
			before := *beat
			before.Notes = slices.Clone(beat.Notes)
			if !test.rest && (beat.Notes[0].Velocity != 47 || beat.Notes[1].Velocity != 95) {
				t.Fatal("authored velocities are not independent")
			}
			data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
			if err != nil {
				t.Fatal(err)
			}
			var codes []string
			for _, entry := range report.Entries {
				codes = append(codes, entry.Code)
				if entry.Location != (gp.ScoreLocation{Beat: 1}) || entry.Disposition != gp.ExportDispositionNormalized {
					t.Fatalf("report location or disposition = %#v", entry)
				}
			}
			if !slices.Equal(codes, test.codes) {
				t.Fatalf("codes = %v, want %v", codes, test.codes)
			}
			if preflight := gp.PreflightExport(song, gp.ExportFormatGP8, gp.ExportOptions{}); !reflect.DeepEqual(preflight, report) {
				t.Fatalf("preflight differs from export: %#v", preflight)
			}
			checkDynamicPolicyOutput(t, data, test.rest)
			for _, allowed := range [][]string{nil, {"gp8.normalize.beat-dynamic"}, {"gp8.normalize.note-velocity"}, test.codes} {
				output, strictReport, strictErr := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
				if !reflect.DeepEqual(strictReport, report) {
					t.Fatalf("strict report changed: %#v", strictReport)
				}
				var missing []string
				for _, code := range test.codes {
					if !slices.Contains(allowed, code) {
						missing = append(missing, code)
					}
				}
				if len(missing) == 0 {
					if strictErr != nil {
						t.Fatal(strictErr)
					}
					checkDynamicPolicyOutput(t, output, test.rest)
				} else {
					var loss *gp.ExportLossError
					if len(output) != 0 || !errors.As(strictErr, &loss) {
						t.Fatalf("unallowed export = %d bytes, %v", len(output), strictErr)
					}
					var refused []string
					for _, entry := range loss.Entries {
						refused = append(refused, entry.Code)
						if entry.Location != (gp.ScoreLocation{Beat: 1}) {
							t.Fatalf("refused location = %#v", entry)
						}
					}
					if !slices.Equal(refused, missing) {
						t.Fatalf("refused = %v, want %v", refused, missing)
					}
				}
			}
			if !reflect.DeepEqual(*beat, before) {
				t.Fatal("export mutated authored beat or note velocities")
			}
		})
	}
}

func dynamicPolicySong(t *testing.T, dynamic int16, rest bool) *gp.Song {
	t.Helper()
	quarter := gp.Duration{Value: 4, TupletEnters: 1, TupletTimes: 1}
	beat := gp.Beat{Duration: quarter, Status: gp.BeatStatusNormal, Dynamics: dynamic, Notes: []gp.Note{
		{Value: 3, String: 1, DurationPercent: 1, Kind: gp.NoteTypeNormal, Velocity: 47},
		{Value: 5, String: 2, DurationPercent: 1, Kind: gp.NoteTypeNormal, Velocity: 95},
	}}
	if rest {
		beat.Status = gp.BeatStatusRest
		beat.Notes = nil
	}
	song := &gp.Song{Tempo: 120, MeasureHeaders: []gp.MeasureHeader{{Number: 1, TimeSignature: gp.TimeSignature{Numerator: 4, Denominator: quarter, Beams: [4]uint8{2, 2, 2, 2}}}},
		Channels: []gp.MidiChannel{{Channel: 0, EffectChannel: 1, Instrument: 25, Volume: 100, Balance: 64}},
		Tracks: []gp.Track{{Name: "Guitar", Visible: true, Settings: gp.TrackSettings{Notation: true}, Strings: []gp.GuitarString{{Number: 1, Value: 64}, {Number: 2, Value: 59}},
			Measures: []gp.Measure{{Voices: []gp.Voice{{Beats: []gp.Beat{{Duration: quarter, Status: gp.BeatStatusRest, Dynamics: 95}, beat}}}}}}}}
	if err := gp.FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if diagnostics := gp.ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("invalid song: %#v", diagnostics)
	}
	return song
}

func checkDynamicPolicyOutput(t *testing.T, data []byte, rest bool) {
	t.Helper()
	song, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	beat := song.Tracks[0].Measures[0].Voices[0].Beats[1]
	if beat.Dynamics != 47 {
		t.Fatalf("reparsed dynamics = %d", beat.Dynamics)
	}
	if rest {
		if beat.Status != gp.BeatStatusRest || len(beat.Notes) != 0 {
			t.Fatalf("rest changed: %#v", beat)
		}
		return
	}
	if beat.Status != gp.BeatStatusNormal || len(beat.Notes) != 2 {
		t.Fatalf("chord changed: %#v", beat)
	}
	for _, note := range beat.Notes {
		if note.Velocity != 47 {
			t.Fatalf("reparsed velocity = %d", note.Velocity)
		}
	}
}
