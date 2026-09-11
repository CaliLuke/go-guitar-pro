// SPDX-License-Identifier: MIT
package integration_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestGP8TimedEmptyAndAbsentVoicePolicy(t *testing.T) {
	for _, empty := range []bool{false, true} {
		song := restVoicePolicySong(t, empty)
		before, _ := json.Marshal(song)
		preflight := gp.PreflightExport(song, gp.ExportFormatGP8, gp.ExportOptions{})
		wantCount := 0
		if empty {
			wantCount = 1
		}
		if len(preflight.Entries) != wantCount {
			t.Fatalf("empty=%v report=%#v", empty, preflight)
		}
		if empty {
			e := preflight.Entries[0]
			if e.Code != "gp8.normalize.empty-beat" || e.Location != (gp.ScoreLocation{Beat: 1}) || e.Disposition != gp.ExportDispositionNormalized {
				t.Fatal(e)
			}
		}
		for _, allowed := range [][]string{nil, {"gp8.omit.note-duration-percent"}, {"gp8.normalize.empty-beat"}} {
			data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
			if !reflect.DeepEqual(preflight, report) {
				t.Fatal("preflight/export differ")
			}
			if empty && (len(allowed) == 0 || allowed[0] != "gp8.normalize.empty-beat") {
				var loss *gp.ExportLossError
				if len(data) != 0 || !errors.As(err, &loss) || len(loss.Entries) != 1 {
					t.Fatalf("strict=%d bytes %v", len(data), err)
				}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := gp.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			voices := parsed.Tracks[0].Measures[0].Voices
			if len(voices) != 3 || len(voices[0].Beats) != 3 || len(voices[1].Beats) != 0 || len(voices[2].Beats) != 1 {
				t.Fatalf("voice slots changed: %#v", voices)
			}
			for i, beat := range voices[0].Beats {
				source := song.Tracks[0].Measures[0].Voices[0].Beats[i]
				status := source.Status
				if status == gp.BeatStatusEmpty {
					status = gp.BeatStatusRest
				}
				if beat.Text != source.Text || beat.Duration != source.Duration || !reflect.DeepEqual(beat.ExactStart, source.ExactStart) || beat.Status != status || len(beat.Notes) != len(source.Notes) {
					t.Fatalf("beat %d changed: %#v source %#v", i, beat, source)
				}
			}
		}
		after, _ := json.Marshal(song)
		if string(before) != string(after) {
			t.Fatal("export mutated source")
		}
	}
}
func restVoicePolicySong(t *testing.T, empty bool) *gp.Song {
	t.Helper()
	q := gp.Duration{Value: 4, TupletEnters: 1, TupletTimes: 1}
	e := gp.Duration{Value: 8, Dotted: true, TupletEnters: 3, TupletTimes: 2}
	note := gp.Beat{Duration: q, Status: gp.BeatStatusNormal, Dynamics: 95, Text: "normal", Notes: []gp.Note{{Value: 3, String: 1, Kind: gp.NoteTypeNormal, Velocity: 95, DurationPercent: 1}}}
	status := gp.BeatStatusRest
	if empty {
		status = gp.BeatStatusEmpty
	}
	song := &gp.Song{Tempo: 120, Channels: []gp.MidiChannel{{Channel: 0, EffectChannel: 1, Instrument: 25, Volume: 100, Balance: 64}}, MeasureHeaders: []gp.MeasureHeader{{TimeSignature: gp.TimeSignature{Numerator: 4, Denominator: q, Beams: [4]uint8{2, 2, 2, 2}}}}, Tracks: []gp.Track{{Name: "Guitar", Settings: gp.TrackSettings{Notation: true}, Strings: []gp.GuitarString{{Number: 1, Value: 64}}, Measures: []gp.Measure{{Voices: []gp.Voice{{Beats: []gp.Beat{note, {Duration: e, Status: status, Dynamics: 95, Text: "timed empty"}, {Duration: q, Status: gp.BeatStatusRest, Dynamics: 95, Text: "ordinary rest"}}}, {}, {Beats: []gp.Beat{note}}}}}}}}
	if err := gp.FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if d := gp.ValidateSong(song); len(d) != 0 {
		t.Fatal(d)
	}
	return song
}
