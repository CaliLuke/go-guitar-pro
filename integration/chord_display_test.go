// SPDX-License-Identifier: MIT

package integration_test

import (
	"reflect"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestChordDisplayPublicOwnership(t *testing.T) {
	data := mustReadFixture(t, "../testdata/gp8/chord-display.gp")
	parsed, err := gp.ParseWithOptions(data, gp.ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	song := parsed.Song
	for st := range song.Tracks[0].Staves {
		for n := 0; n < 8; n++ {
			chord := song.Tracks[0].Staves[st].Measures[n/4].Voices[0].Beats[n%4].Effect.Chord
			flags := n
			if st == 1 {
				flags = 7 - n
			}
			if chord.ShowName == nil || chord.ShowDiagram == nil || chord.ShowFingering == nil {
				t.Fatal("missing display values")
			}
			if *chord.ShowName != (flags&1 != 0) || *chord.ShowDiagram != (flags&2 != 0) || *chord.ShowFingering != (flags&4 != 0) {
				t.Fatalf("staff %d chord %d flags changed", st, n)
			}
		}
	}
	first := song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Effect.Chord
	repeated := song.Tracks[0].Staves[0].Measures[2].Voices[0].Beats[2].Effect.Chord
	opposite := song.Tracks[0].Staves[1].Measures[0].Voices[0].Beats[0].Effect.Chord
	*first.ShowName = true
	*first.ShowDiagram = true
	*first.ShowFingering = true
	if *repeated.ShowName || *repeated.ShowDiagram || *repeated.ShowFingering || !*opposite.ShowName {
		t.Fatal("display fields alias another occurrence")
	}
	song.Version = gp.Version{}
	song.Tracks[0].Settings = gp.TrackSettings{Notation: true}
	before := append([]int8(nil), first.Strings...)
	exported, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("strict export: %v %#v", err, report)
	}
	got, err := gp.Parse(exported)
	if err != nil {
		t.Fatal(err)
	}
	actual := got.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Effect.Chord
	if !*actual.ShowName || !*actual.ShowDiagram || !*actual.ShowFingering || !reflect.DeepEqual(before, actual.Strings) {
		t.Fatal("edited display or chord contents changed")
	}
}
