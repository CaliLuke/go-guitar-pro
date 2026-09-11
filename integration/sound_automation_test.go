// SPDX-License-Identifier: MIT

package integration_test

import (
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"

	guitarpro "github.com/CaliLuke/go-guitar-pro"
)

func TestSoundAutomationOpeningPreroll(t *testing.T) {
	song := soundPrerollFixture(t)
	track := &song.Tracks[0]
	wantSound := guitarpro.TrackSound{Name: "Acoustic Guitar (Steel)", Label: "Acoustic Guitar (Steel)", Path: "Stringed/Acoustic Guitars/Steel Guitar", Role: "User", Program: 25}
	if !slices.Equal(track.SoundAutomations, []guitarpro.SoundAutomation{{Bar: 0, Position: -0.125, Sound: 0}}) || track.Sounds[0] != wantSound {
		t.Fatalf("source sound event = %#v, sounds = %#v", track.SoundAutomations, track.Sounds)
	}
	assertSoundPrerollExport(t, song)

	// Edit the parsed records directly, with deliberately unsorted and equal
	// positions. The slice order and distinct referenced sounds must survive.
	track.Sounds = append(track.Sounds,
		guitarpro.TrackSound{Name: "Zero", Path: "Midi/26", Role: "User", Program: 26, Bank: 77},
		guitarpro.TrackSound{Name: "Preroll", Path: "Midi/27", Role: "User", Program: 27, Bank: 256})
	track.SoundAutomations = []guitarpro.SoundAutomation{
		{Bar: 0, Position: -0.125, Sound: 0},
		{Bar: 0, Position: 0, Sound: 1},
		{Bar: 0, Position: -0.125, Sound: 2, Linear: true, Text: "authored order"},
		{Bar: 0, Position: -0.125, Sound: 1},
	}
	assertSoundPrerollExport(t, song)
}

func TestSoundAutomationPositionBounds(t *testing.T) {
	for _, test := range []struct {
		name     string
		bar      int
		position float64
		sound    int
		code     string
	}{
		{name: "opening lower bound", position: -0.125},
		{name: "opening inside lower bound", position: math.Nextafter(-0.125, 0)},
		{name: "opening just negative", position: math.Nextafter(0, -1)},
		{name: "zero"},
		{name: "positive", position: 0.25},
		{name: "upper bound", position: 1},
		{name: "later zero", bar: 1},
		{name: "later upper bound", bar: 1, position: 1},
		{name: "before preroll", position: math.Nextafter(-0.125, -1), code: "location"},
		{name: "after upper bound", position: math.Nextafter(1, 2), code: "location"},
		{name: "later preroll", bar: 1, position: -0.125, code: "location"},
		{name: "later just negative", bar: 1, position: math.Nextafter(0, -1), code: "location"},
		{name: "NaN", position: math.NaN(), code: "location"},
		{name: "positive infinity", position: math.Inf(1), code: "location"},
		{name: "negative infinity", position: math.Inf(-1), code: "location"},
		{name: "negative bar", bar: -1, code: "location"},
		{name: "high bar", bar: 999, code: "location"},
		{name: "negative sound", sound: -1, code: "reference"},
		{name: "high sound", sound: 1, code: "reference"},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := soundPrerollFixture(t)
			header := song.MeasureHeaders[0]
			header.Number = 2
			song.MeasureHeaders = append(song.MeasureHeaders, header)
			measure := song.Tracks[0].Measures[0]
			measure.Number, measure.HeaderIndex, measure.Voices = 2, 1, nil
			song.Tracks[0].Measures = append(song.Tracks[0].Measures, measure)
			if err := guitarpro.FinalizeSong(song); err != nil {
				t.Fatal(err)
			}
			song.Tracks[0].SoundAutomations = []guitarpro.SoundAutomation{{Bar: test.bar, Position: test.position, Sound: test.sound}}
			if test.code == "" {
				assertSoundPrerollExport(t, song)
				return
			}
			code := "score.sound-automation." + test.code
			diagnostics := guitarpro.ValidateSong(song)
			if !slices.ContainsFunc(diagnostics, func(d guitarpro.ScoreDiagnostic) bool {
				return d.Code == code && d.Location.Track == 0 && strings.Contains(d.Reason, "sound automation 0")
			}) {
				t.Fatalf("diagnostics = %#v, want scoped %s", diagnostics, code)
			}
			preflight := guitarpro.PreflightExport(song, guitarpro.ExportFormatGP8, guitarpro.ExportOptions{})
			data, report, err := guitarpro.ExportWithReport(song, guitarpro.ExportFormatGP8, guitarpro.ExportOptions{})
			if err == nil || len(data) != 0 || !reflect.DeepEqual(report, preflight) || !slices.ContainsFunc(report.Entries, func(e guitarpro.ExportReportEntry) bool {
				return e.Code == "gp8.reject."+code && e.Disposition == guitarpro.ExportDispositionRejected
			}) {
				t.Fatalf("export = %d bytes, %#v, %v; preflight = %#v", len(data), report, err, preflight)
			}
		})
	}
}

func soundPrerollFixture(t *testing.T) *guitarpro.Song {
	t.Helper()
	song, err := guitarpro.ParseFile("../testdata/gp7/grace.gp")
	if err != nil {
		t.Fatal(err)
	}
	return song
}

func assertSoundPrerollExport(t *testing.T, song *guitarpro.Song) {
	t.Helper()
	want := slices.Clone(song.Tracks[0].SoundAutomations)
	wantSounds := slices.Clone(song.Tracks[0].Sounds)
	for _, d := range guitarpro.ValidateSong(song) {
		if strings.HasPrefix(d.Code, "score.sound-automation.") {
			t.Fatalf("valid sound event rejected: %#v", d)
		}
	}
	preflight := guitarpro.PreflightExport(song, guitarpro.ExportFormatGP8, guitarpro.ExportOptions{})
	data, report, err := guitarpro.ExportWithReport(song, guitarpro.ExportFormatGP8, guitarpro.ExportOptions{})
	if err != nil || !reflect.DeepEqual(report, preflight) {
		t.Fatalf("export = %#v, %v; preflight = %#v", report, err, preflight)
	}
	got, err := guitarpro.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got.Tracks[0].SoundAutomations, want) || !slices.Equal(got.Tracks[0].Sounds, wantSounds) {
		t.Fatalf("round trip = %#v, %#v; want %#v, %#v", got.Tracks[0].SoundAutomations, got.Tracks[0].Sounds, want, wantSounds)
	}
	if !slices.Equal(song.Tracks[0].SoundAutomations, want) || !slices.Equal(song.Tracks[0].Sounds, wantSounds) {
		t.Fatal("validation or export mutated authored sound events")
	}
}
