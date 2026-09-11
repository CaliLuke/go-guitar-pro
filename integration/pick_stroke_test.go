// SPDX-License-Identifier: MIT

package integration_test

import (
	"reflect"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestGP8PickStroke(t *testing.T) {
	for _, pick := range []gp.BeatStrokeDirection{gp.BeatStrokeDirectionUp, gp.BeatStrokeDirectionDown, gp.BeatStrokeDirectionNone} {
		for _, brush := range []gp.BeatStrokeDirection{gp.BeatStrokeDirectionNone, gp.BeatStrokeDirectionUp, gp.BeatStrokeDirectionDown} {
			song := pickStrokeSong(t, pick, brush)
			before := song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect
			options := gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}}
			preflight := gp.PreflightExport(song, gp.ExportFormatGP8, options)
			data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, options)
			if err != nil || len(report.Entries) != 0 || !reflect.DeepEqual(report, preflight) {
				t.Fatalf("pick=%d brush=%d export: %v %#v", pick, brush, err, report)
			}
			parsed, err := gp.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			effect := parsed.Tracks[0].Measures[0].Voices[0].Beats[0].Effect
			if effect.PickStroke != pick || effect.Stroke.Direction != brush {
				t.Fatalf("pick=%d brush=%d got %#v", pick, brush, effect)
			}
			// Public edits after import are authoritative, including clearing the mark.
			parsed.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.PickStroke = gp.BeatStrokeDirectionNone
			edited, _, err := gp.ExportWithReport(parsed, gp.ExportFormatGP8, gp.ExportOptions{})
			if err != nil {
				t.Fatal(err)
			}
			cleared, err := gp.Parse(edited)
			if err != nil {
				t.Fatal(err)
			}
			effect = cleared.Tracks[0].Measures[0].Voices[0].Beats[0].Effect
			if effect.PickStroke != gp.BeatStrokeDirectionNone || effect.Stroke.Direction != brush {
				t.Fatalf("cleared effect %#v", effect)
			}
			if !reflect.DeepEqual(song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect, before) {
				t.Fatal("read-only export changed source")
			}
		}
	}
}

func TestPickStrokeValidation(t *testing.T) {
	for _, invalid := range []gp.BeatStrokeDirection{-1, 3, 127} {
		song := pickStrokeSong(t, gp.BeatStrokeDirectionUp, gp.BeatStrokeDirectionDown)
		song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.PickStroke = invalid
		diagnostics := gp.ValidateSong(song)
		if len(diagnostics) != 1 || diagnostics[0].Code != "score.beat.pick-stroke" || diagnostics[0].Location != (gp.ScoreLocation{}) {
			t.Fatalf("invalid %d diagnostics %#v", invalid, diagnostics)
		}
		preflight := gp.PreflightExport(song, gp.ExportFormatGP8, gp.ExportOptions{})
		data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
		if err == nil || len(data) != 0 || !reflect.DeepEqual(report, preflight) || len(report.Entries) != 1 || report.Entries[0].Code != "gp8.reject.score.beat.pick-stroke" {
			t.Fatalf("invalid %d export %v %#v", invalid, err, report)
		}
	}
}

func pickStrokeSong(t *testing.T, pick, brush gp.BeatStrokeDirection) *gp.Song {
	t.Helper()
	quarter := gp.Duration{Value: 4, TupletEnters: 1, TupletTimes: 1}
	effect := gp.BeatEffects{PickStroke: pick}
	if brush != gp.BeatStrokeDirectionNone {
		effect.Stroke = gp.BeatStroke{Kind: gp.BeatStrokeKindBrush, Direction: brush, Duration: gp.NoteValue(gp.DurationEighth)}
	}
	beat := gp.Beat{
		Duration: quarter, Status: gp.BeatStatusNormal, Dynamics: 47, Effect: effect,
		Notes: []gp.Note{{Value: 3, String: 1, Kind: gp.NoteTypeNormal, Velocity: 47, DurationPercent: 1}},
	}
	song := &gp.Song{
		Tempo: 120,
		MeasureHeaders: []gp.MeasureHeader{{Number: 1, TimeSignature: gp.TimeSignature{
			Numerator: 4, Denominator: quarter, Beams: [4]uint8{2, 2, 2, 2},
		}}},
		Channels: []gp.MidiChannel{{Channel: 0, EffectChannel: 1, Instrument: 25, Volume: 100, Balance: 64}},
		Tracks: []gp.Track{{
			Name: "Guitar", Visible: true, Settings: gp.TrackSettings{Notation: true},
			Strings:  []gp.GuitarString{{Number: 1, Value: 64}},
			Measures: []gp.Measure{{Voices: []gp.Voice{{Beats: []gp.Beat{beat}}}}},
		}},
	}
	if err := gp.FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if ds := gp.ValidateSong(song); len(ds) != 0 {
		t.Fatalf("invalid public score %#v", ds)
	}
	return song
}
