// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"os"
	"reflect"
	"slices"
	"testing"
)

func TestConformanceChordAndRestDynamics(t *testing.T) {
	runConformanceChordAndRestDynamics(newConformanceRun(t))
}

func runConformanceChordAndRestDynamics(run *conformanceRun) {
	for _, test := range []struct {
		name    string
		dynamic int16
		rest    bool
		codes   []string
	}{
		{"chord fallback", 0, false, []string{"gp8.normalize.note-velocity"}},
		{"chord canonical", 47, false, []string{"gp8.normalize.note-velocity"}},
		{"chord quantized", 48, false, []string{"gp8.normalize.beat-dynamic", "gp8.normalize.note-velocity"}},
		{"rest canonical", 47, true, []string{}},
		{"rest quantized", 48, true, []string{"gp8.normalize.beat-dynamic"}},
	} {
		run.t.Run(test.name, func(t *testing.T) {
			song := semanticExportProbeSong(t)
			beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
			beat.Dynamics = test.dynamic
			beat.Notes[0].Velocity = 47
			beat.Notes = append(beat.Notes, Note{Value: 5, String: 2, Kind: NoteTypeNormal, Velocity: 95, DurationPercent: 1})
			if test.rest {
				beat.Status = BeatStatusRest
				beat.Notes = nil
			}
			run.Field("Beat.Dynamics", beat.Dynamics, test.dynamic)
			if !test.rest {
				run.Field("Note.Velocity", []int16{beat.Notes[0].Velocity, beat.Notes[1].Velocity}, []int16{47, 95})
			}
			var originalReport ExportReport
			for _, strict := range []bool{false, true} {
				data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: strict, AllowedCodes: test.codes}})
				if err != nil {
					t.Fatal(err)
				}
				if strict && !reflect.DeepEqual(report, originalReport) {
					t.Fatal("strict report changed")
				}
				originalReport = report
				run.Report("M10-CHORD-REST-DYNAMICS", reportCodes(report), test.codes)
				for _, entry := range report.Entries {
					if entry.Location != (ScoreLocation{}) {
						t.Fatalf("report location = %#v", entry)
					}
				}
				run.Wire("gpifBeat.Dynamic", extractGPIFLeafText(t, data)["GPIF/Beats/Beat/Dynamic"], "P")
				roundTrip, err := Parse(data)
				if err != nil {
					t.Fatal(err)
				}
				got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0]
				run.Field("Beat.Dynamics", got.Dynamics, int16(47))
				run.Field("Beat.Status", got.Status, beat.Status)
				if test.rest {
					run.Field("Beat.Notes", len(got.Notes), 0)
				} else {
					if len(got.Notes) != 2 {
						t.Fatalf("reparsed notes = %#v", got.Notes)
					}
					run.Field("Note.Velocity", []int16{got.Notes[0].Velocity, got.Notes[1].Velocity}, []int16{47, 47})
				}
				if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
					assertAlphaTabChordRestDynamics(t, data, test.rest)
				}
			}
		})
	}
}

func TestAlphaTabChordAndRestDynamics(t *testing.T) {
	requireAlphaTabConformance(t)
	runConformanceChordAndRestDynamics(newConformanceRun(t))
}

func assertAlphaTabChordRestDynamics(t *testing.T, data []byte, rest bool) {
	t.Helper()
	root := readAlphaTabScore(t, writeConformanceFixture(t, data)).(map[string]any)
	track := root["tracks"].([]any)[0].(map[string]any)
	staff := track["staves"].([]any)[0].(map[string]any)
	bar := staff["bars"].([]any)[0].(map[string]any)
	voice := bar["voices"].([]any)[0].(map[string]any)
	beats := voice["beats"].([]any)
	if len(beats) != 1 {
		t.Fatalf("consumer beats = %#v", beats)
	}
	beat := beats[0].(map[string]any)
	if beat["dynamic"] != "p" {
		t.Fatalf("consumer beat dynamic = %v", beat["dynamic"])
	}
	notes := beat["notes"].([]any)
	if rest {
		if beat["status"] != "rest" || len(notes) != 0 {
			t.Fatalf("consumer rest = %#v", beat)
		}
		return
	}
	var dynamics []string
	for _, note := range notes {
		dynamics = append(dynamics, note.(map[string]any)["dynamic"].(string))
	}
	if beat["status"] != "normal" || !slices.Equal(dynamics, []string{"p", "p"}) {
		t.Fatalf("consumer chord = %#v", beat)
	}
}
