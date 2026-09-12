// SPDX-License-Identifier: MIT

package integration_test

import (
	"errors"
	"reflect"
	"testing"

	guitarpro "github.com/CaliLuke/go-guitar-pro"
)

func TestOpeningTempoVisibility(t *testing.T) {
	for _, hidden := range []bool{false, true} {
		for _, explicit := range []bool{false, true} {
			song, err := guitarpro.ParseFile("../testdata/gp7/tempo.gp")
			if err != nil {
				t.Fatal(err)
			}
			song.Tempo = 133
			song.InitialTempo = guitarpro.KnownSourceValue(guitarpro.BPM(132.5))
			song.TempoName = "Opening label"
			song.HideTempo = hidden
			song.TempoAutomations = []guitarpro.TempoAutomation{
				{Bar: 0, Position: 0.75, Tempo: 91.25, Text: "later", Linear: true},
				{Bar: 0, Position: 0.5, Tempo: 98.5, Text: "earlier", Hidden: true},
				{Bar: 0, Position: 0.5, Tempo: 98.5, Text: "equal position", Linear: true},
			}
			wantOpening := guitarpro.TempoAutomation{Tempo: 132.5, Text: "Opening label", Hidden: hidden}
			if explicit {
				wantOpening.Hidden = !hidden
				wantOpening.Linear = true
				song.TempoAutomations = append(song.TempoAutomations, wantOpening)
			}
			before := append([]guitarpro.TempoAutomation(nil), song.TempoAutomations...)
			data, report, err := guitarpro.ExportWithReport(song, guitarpro.ExportFormatGP8, guitarpro.ExportOptions{})
			if err != nil {
				t.Fatal(err)
			}
			allowed := []string{}
			conflicts := 0
			for _, entry := range report.Entries {
				if entry.Code == "gp8.omit.tempo-visibility" {
					t.Fatal("blanket tempo visibility omission remains")
				}
				if entry.Code == "gp8.normalize.tempo-visibility-authority" {
					conflicts++
					continue
				}
				allowed = append(allowed, entry.Code)
			}
			if (conflicts == 1) != explicit {
				t.Fatalf("hidden=%v explicit=%v conflicts=%d", hidden, explicit, conflicts)
			}
			_, _, strictErr := guitarpro.ExportWithReport(song, guitarpro.ExportFormatGP8, guitarpro.ExportOptions{LossPolicy: guitarpro.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
			var loss *guitarpro.ExportLossError
			if explicit && !errors.As(strictErr, &loss) || !explicit && strictErr != nil {
				t.Fatalf("strict explicit=%v: %v", explicit, strictErr)
			}
			allowed = append(allowed, "gp8.normalize.tempo-visibility-authority")
			_, _, allowedErr := guitarpro.ExportWithReport(song, guitarpro.ExportFormatGP8, guitarpro.ExportOptions{LossPolicy: guitarpro.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
			if allowedErr != nil {
				t.Fatal(allowedErr)
			}
			roundTrip, err := guitarpro.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			want := before
			if !explicit {
				want = append([]guitarpro.TempoAutomation{wantOpening}, before...)
			}
			if !reflect.DeepEqual(roundTrip.TempoAutomations, want) {
				t.Fatalf("hidden=%v explicit=%v got %#v want %#v", hidden, explicit, roundTrip.TempoAutomations, want)
			}
			if !reflect.DeepEqual(song.TempoAutomations, before) || song.HideTempo != hidden {
				t.Fatal("export mutated authored values")
			}
		}
	}
}

func TestGP5HiddenOpeningTempo(t *testing.T) {
	song, err := guitarpro.ParseFile("../testdata/gp5/nightwish.gp5")
	if err != nil {
		t.Fatal(err)
	}
	if !song.HideTempo || song.Version.Number != [3]byte{5, 1, 0} {
		t.Fatal("fixture must retain its GP5.1 hidden opening tempo")
	}
	wantChanges := []guitarpro.TempoAutomation{
		{Bar: 88, Position: 0.625, Tempo: 85, Linear: false, Hidden: true},
		{Bar: 89, Tempo: 90, Linear: false, Hidden: true},
		{Bar: 92, Tempo: 80, Linear: false, Hidden: true},
		{Bar: 93, Tempo: 80, Linear: false, Hidden: true},
		{Bar: 93, Position: 0.875, Tempo: 60, Linear: false, Hidden: true},
		{Bar: 94, Tempo: 80, Linear: false, Hidden: true},
		{Bar: 85, Tempo: 90, Linear: false},
		{Bar: 93, Position: 0.75, Tempo: 70, Linear: false},
	}
	if !reflect.DeepEqual(song.TempoAutomations, wantChanges) {
		t.Fatalf("legacy mix-table tempos = %#v", song.TempoAutomations)
	}
	// Replace the corpus fixture's non-UTF-8 label with valid authored text.
	song.TempoName = "Legacy hidden"
	data, _, err := guitarpro.ExportWithReport(song, guitarpro.ExportFormatGP8, guitarpro.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := guitarpro.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	want := append([]guitarpro.TempoAutomation{{Tempo: float64(song.InitialTempo.Value), Text: song.TempoName, Hidden: true}}, wantChanges...)
	if !reflect.DeepEqual(got.TempoAutomations, want) || !got.HideTempo {
		t.Fatalf("legacy visibility export = %#v", got.TempoAutomations)
	}
}
