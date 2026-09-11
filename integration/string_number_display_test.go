// SPDX-License-Identifier: MIT

package integration_test

import (
	"reflect"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestStringNumberDisplayPublicOwnership(t *testing.T) {
	data := mustReadFixture(t, "../testdata/gp8/string-number-display.gp")
	result, err := gp.ParseWithOptions(data, gp.ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	song := result.Song
	beats := song.Tracks[0].Measures[0].Voices[0].Beats
	for i, want := range []bool{true, false, true, false} {
		if beats[i].Notes[0].ShowStringNumber != want {
			t.Fatalf("beat %d display=%t", i, beats[i].Notes[0].ShowStringNumber)
		}
	}
	if beats[0].Notes[0].String != 3 || beats[0].Notes[0].Value != 9 {
		t.Fatal("non-default string/fret changed")
	}
	before := beats[0].Notes[0]
	tuning := append([]gp.GuitarString(nil), song.Tracks[0].Strings...)
	beats[0].Notes[0].ShowStringNumber = false
	beats[1].Notes[0].ShowStringNumber = true
	if !beats[2].Notes[0].ShowStringNumber {
		t.Fatal("reused note definition shared display state")
	}
	song.Version = gp.Version{}
	song.Tracks[0].Settings = gp.TrackSettings{Notation: true}
	output, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("export=%v %#v", err, report.Entries)
	}
	parsed, err := gp.Parse(output)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []bool{false, true, true, false} {
		if parsed.Tracks[0].Measures[0].Voices[0].Beats[i].Notes[0].ShowStringNumber != want {
			t.Fatalf("edited beat %d display changed", i)
		}
	}
	before.ShowStringNumber = false
	if !reflect.DeepEqual(beats[0].Notes[0], before) || !reflect.DeepEqual(song.Tracks[0].Strings, tuning) {
		t.Fatal("display edit changed pitch, tuning or other note fields")
	}
}
