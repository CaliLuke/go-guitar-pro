// SPDX-License-Identifier: MIT
package integration_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestGraceTimerPublicSource(t *testing.T) {
	song := publicGraceTimerSong(t)
	want := []*gp.BeatTimer{nil, {}, {Milliseconds: timerMilliseconds(0)}, {Milliseconds: timerMilliseconds(99)}, {Milliseconds: timerMilliseconds(9007199254740991)}}
	for _, measure := range song.Tracks[0].Measures {
		if len(measure.Voices[0].Beats) != 1 {
			t.Fatal("grace beats did not attach")
		}
		beat := measure.Voices[0].Beats[0]
		if beat.Timer == nil || *beat.Timer.Milliseconds != 25 {
			t.Fatal("owner timer changed")
		}
		for _, note := range beat.Notes {
			if len(note.Effect.Graces) != len(want) {
				t.Fatal("grace count", note.Effect.Graces)
			}
			for i, g := range note.Effect.Graces {
				if !reflect.DeepEqual(g.Timer, want[i]) || g.Sequence != uint8(i) {
					t.Fatalf("grace%d=%+v", i, g)
				}
			}
		}
	}
	first := song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes
	second := song.Tracks[0].Measures[1].Voices[0].Beats[0].Notes
	*first[0].Effect.Graces[3].Timer.Milliseconds = 4000
	if *first[1].Effect.Graces[3].Timer.Milliseconds != 99 || *second[0].Effect.Graces[3].Timer.Milliseconds != 99 {
		t.Fatal("grace timers share mutable storage")
	}
	for i := range first {
		first[i].Effect.Graces[0].Timer = &gp.BeatTimer{Milliseconds: timerMilliseconds(0)}
		first[i].Effect.Graces[1].Timer = nil
		first[i].Effect.Graces[2].Timer = &gp.BeatTimer{}
		*first[i].Effect.Graces[3].Timer.Milliseconds = 4000
		first[i].Effect.Graces[4].Timer = nil
	}
	before, _ := json.Marshal(song)
	if err := gp.FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatal(err, report)
	}
	after, _ := json.Marshal(song)
	if string(before) != string(after) {
		t.Fatal("finalization/export mutated authored score")
	}
	parsed, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	want = []*gp.BeatTimer{{Milliseconds: timerMilliseconds(0)}, nil, {}, {Milliseconds: timerMilliseconds(4000)}, nil}
	for _, note := range parsed.Tracks[0].Measures[0].Voices[0].Beats[0].Notes {
		for i, g := range note.Effect.Graces {
			if !reflect.DeepEqual(g.Timer, want[i]) {
				t.Fatalf("edited grace%d timer%+v", i, g.Timer)
			}
		}
	}
}

func TestGraceTimerPublicInvalidAndConflictingEdits(t *testing.T) {
	for _, bad := range []int64{-1, 9007199254740992, 9223372036854775807} {
		s := publicGraceTimerSong(t)
		notes := s.Tracks[0].Measures[0].Voices[0].Beats[0].Notes
		for i := range notes {
			notes[i].Effect.Graces[3].Timer.Milliseconds = timerMilliseconds(bad)
		}
		assertGraceTimerRejection(t, s, "score.grace.timer", 0)
	}
	states := []*gp.BeatTimer{nil, {}, {Milliseconds: timerMilliseconds(0)}}
	for i, first := range states {
		for j, second := range states {
			if i == j {
				continue
			}
			s := publicGraceTimerSong(t)
			notes := s.Tracks[0].Measures[0].Voices[0].Beats[0].Notes
			notes[0].Effect.Graces[3].Timer = first
			notes[1].Effect.Graces[3].Timer = second
			assertGraceTimerRejection(t, s, "score.grace.timer-conflict", 1)
		}
	}
}
func assertGraceTimerRejection(t *testing.T, s *gp.Song, code string, note int) {
	t.Helper()
	before, _ := json.Marshal(s)
	found := false
	for _, d := range gp.ValidateSong(s) {
		if d.Code == code && d.Location == (gp.ScoreLocation{Note: note}) && strings.Contains(d.Reason, "grace 3") {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing %s at note%d: %+v", code, note, gp.ValidateSong(s))
	}
	data, r, e := gp.ExportWithReport(s, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{AllowedCodes: []string{"gp8.reject." + code}}})
	if e == nil || len(data) != 0 {
		t.Fatal("invalid timer exported", r)
	}
	after, _ := json.Marshal(s)
	if string(before) != string(after) {
		t.Fatal("invalid export mutated authored score")
	}
}
func publicGraceTimerSong(t *testing.T) *gp.Song {
	t.Helper()
	r, e := gp.ParseWithOptions(mustReadFixture(t, "../testdata/gp8/grace-timer.gp"), gp.ParseOptions{Strict: true})
	if e != nil {
		t.Fatal(e)
	}
	if len(r.Diagnostics) != 0 {
		t.Fatal(r.Diagnostics)
	}
	r.Song.Version = gp.Version{}
	r.Song.Tracks[0].Settings = gp.TrackSettings{Notation: true}
	return r.Song
}

func TestGraceTimerLongSequence(t *testing.T) {
	for _, equalTail := range []bool{false, true} {
		result, err := gp.ParseWithOptions(mustReadFixture(t, "../testdata/gp8/grace-timer-sequence.gp"), gp.ParseOptions{Strict: true})
		if err != nil {
			t.Fatal(err)
		}
		song := result.Song
		for generation := 0; generation < 3; generation++ {
			for _, measure := range song.Tracks[0].Measures {
				beats := measure.Voices[0].Beats
				if len(beats) != 258 {
					t.Fatalf("generation %d retained %d beats, want 258", generation, len(beats))
				}
				for index, beat := range beats {
					want := int64(index)
					if index == 257 {
						want = 25
					}
					if equalTail && index == 256 {
						want = 255
						if generation == 0 {
							*beat.Timer.Milliseconds = want
						}
					}
					if len(beat.Notes) != 2 || beat.Timer == nil || beat.Timer.Milliseconds == nil || *beat.Timer.Milliseconds != want {
						t.Fatalf("generation %d beat %d lost timer or chord membership: %+v", generation, index, beat)
					}
				}
			}
			if generation == 2 {
				break
			}
			song.Version = gp.Version{}
			song.Tracks[0].Settings = gp.TrackSettings{Notation: true}
			before, _ := json.Marshal(song)
			data, report, exportErr := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}})
			if exportErr != nil || len(report.Entries) != 0 {
				t.Fatal(exportErr, report)
			}
			after, _ := json.Marshal(song)
			if string(before) != string(after) {
				t.Fatal("export mutated long grace sequence")
			}
			result, err = gp.ParseWithOptions(data, gp.ParseOptions{Strict: true})
			if err != nil {
				t.Fatal(err)
			}
			song = result.Song
		}
	}
}
