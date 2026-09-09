// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"reflect"
	"testing"
)

func TestValidateSongReportsStructuralAndTimingDiagnostics(t *testing.T) {
	song := syntheticGP8Song()
	song.Tracks[0].ChannelIndex = len(song.Channels) + 1
	song.Tracks[0].Measures[0].HeaderIndex = 9
	song.Tracks[0].Measures[0].Voices[0].Beats[0].Duration.TupletEnters = 0
	wrong := int64(123)
	song.Tracks[0].Measures[0].Voices[0].Beats[0].Start = &wrong

	diagnostics := ValidateSong(song)
	for _, code := range []string{
		"score.track.channel-reference",
		"score.measure.header-reference",
		"score.beat.duration",
		"score.beat.start",
	} {
		if !hasScoreDiagnostic(diagnostics, code) {
			t.Errorf("diagnostics = %#v, want %s", diagnostics, code)
		}
	}
}

func TestFinalizeSongIsIdempotentAndAcceptsSpecialStructures(t *testing.T) {
	grace := defaultBeat()
	grace.isGrace = true
	song := Song{
		Anacrusis:      true,
		MeasureHeaders: []MeasureHeader{defaultMeasureHeader(), defaultMeasureHeader()},
		Tracks: []Track{{ChannelIndex: -1, Measures: []Measure{
			{HeaderIndex: 0, Voices: []Voice{{Beats: []Beat{grace}}}},
			{HeaderIndex: 1, Voices: []Voice{{Beats: []Beat{defaultBeat()}}}},
		}}},
	}
	if err := FinalizeSong(&song); err != nil {
		t.Fatal(err)
	}
	first := song
	if err := FinalizeSong(&song); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(song, first) {
		t.Fatal("repeated finalization changed the score")
	}
	if diagnostics := ValidateSong(&song); len(diagnostics) != 0 {
		t.Fatalf("valid pickup/grace score diagnostics = %#v", diagnostics)
	}
}

func hasScoreDiagnostic(diagnostics []ScoreDiagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
