// SPDX-License-Identifier: MIT

package integration_test

import (
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestChordDiagramConflictReportsUseStableOrder(t *testing.T) {
	song := dynamicPolicySong(t, 47, false)
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[1]
	beat.Effect.Chord = &gp.Chord{
		Strings:    []int8{3, 3, 5, 5},
		Fingerings: []gp.Fingering{gp.FingeringIndex, gp.FingeringIndex, gp.FingeringMiddle, gp.FingeringMiddle},
	}

	want := "repeated finger 1 at fret 3 requires an exact matching barre range"
	for range 20 {
		_, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, entry := range report.Entries {
			if entry.Code != "gp8.omit.chord-diagram-conflict" {
				continue
			}
			found = true
			if entry.Reason != want {
				t.Fatalf("chord conflict reason = %q, want %q", entry.Reason, want)
			}
		}
		if !found {
			t.Fatalf("report = %#v, want chord diagram conflict", report.Entries)
		}
	}
}
