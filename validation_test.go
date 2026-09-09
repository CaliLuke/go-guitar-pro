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

func TestValidateSongReportsHeaderTimingDisagreement(t *testing.T) {
	song := syntheticGP8Song()
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	measure := &song.Tracks[0].Measures[0]
	measure.Start = 123
	measure.ExactStart = mustScoreTime(t, 123, 1)

	diagnostics := ValidateSong(song)
	for _, code := range []string{"score.measure.start", "score.measure.exact-start", "score.beat.exact-start"} {
		if !hasScoreDiagnostic(diagnostics, code) {
			t.Errorf("diagnostics = %#v, want %s", diagnostics, code)
		}
	}
}

func TestValidateSongUsesLegacyStartWhenExactStartIsUnset(t *testing.T) {
	start := DurationQuarterTime
	song := Song{
		MeasureHeaders: []MeasureHeader{defaultMeasureHeader()},
		Tracks: []Track{{ChannelIndex: -1, Measures: []Measure{{
			Start:       start,
			HeaderIndex: 0,
			Voices: []Voice{{Beats: []Beat{{
				Start:    &start,
				Duration: defaultDuration(),
			}}}},
		}}}},
	}

	if diagnostics := ValidateSong(&song); hasScoreDiagnostic(diagnostics, "score.beat.start") {
		t.Fatalf("legacy-only score diagnostics = %#v", diagnostics)
	}
}

func TestValidateSongReportsStructuralAlignmentAndOwnership(t *testing.T) {
	song := Song{
		MeasureHeaders: []MeasureHeader{defaultMeasureHeader(), defaultMeasureHeader()},
		Tracks: []Track{{ChannelIndex: -1, Measures: []Measure{{
			HeaderIndex: 1,
			TrackIndex:  99,
			StaffIndex:  99,
			Voices:      []Voice{{MeasureIndex: 99}},
		}}}},
	}

	diagnostics := ValidateSong(&song)
	for _, code := range []string{
		"score.staff.measure-count",
		"score.measure.header-alignment",
		"score.measure.track-ownership",
		"score.measure.staff-ownership",
		"score.voice.measure-ownership",
	} {
		if !hasScoreDiagnostic(diagnostics, code) {
			t.Errorf("diagnostics = %#v, want %s", diagnostics, code)
		}
	}
}

func TestFinalizeSongReconcilesFirstStaffCompatibilityView(t *testing.T) {
	song := syntheticGP8Song()
	track := &song.Tracks[0]
	track.populateSingleStaff()
	replacement := append([]Measure(nil), track.Measures...)
	for index := range replacement {
		replacement[index].HeaderIndex = index
	}
	replacement[1].Voices = []Voice{{Beats: []Beat{defaultBeat()}}}
	track.Measures = replacement

	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if &track.Measures[0] != &track.Staves[0].Measures[0] {
		t.Fatal("first-staff measures and compatibility measures are separate writable trees")
	}
	if got := track.Measures[1].Voices[0].MeasureIndex; got != 1 {
		t.Fatalf("replacement voice measure index = %d, want 1", got)
	}
	beat := track.Staves[0].Measures[1].Voices[0].Beats[0]
	if beat.ExactStart == nil || beat.ExactStart.Compare(song.MeasureHeaders[1].ExactStart) != 0 {
		t.Fatalf("replacement beat exact start = %#v, want header start", beat.ExactStart)
	}
}

func mustScoreTime(t *testing.T, numerator, denominator int64) ScoreTime {
	t.Helper()
	value, err := NewScoreTime(numerator, denominator)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func hasScoreDiagnostic(diagnostics []ScoreDiagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
