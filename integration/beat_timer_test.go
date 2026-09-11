// SPDX-License-Identifier: MIT
package integration_test

import (
	"encoding/json"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestBeatTimerPublicStates(t *testing.T) {
	result, err := gp.ParseWithOptions(mustReadFixture(t, "../testdata/gp8/beat-timer.gp"), gp.ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	song := result.Song
	first := song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats
	second := song.Tracks[0].Staves[0].Measures[1].Voices[0].Beats
	if first[0].Timer != nil || first[1].Timer == nil || first[1].Timer.Milliseconds != nil || first[2].Timer == nil || *first[2].Timer.Milliseconds != 0 || *first[3].Timer.Milliseconds != 12345 || second[0].Timer == nil || second[0].Timer.Milliseconds != nil || *second[1].Timer.Milliseconds != 9007199254740991 {
		t.Fatal("three timer states changed")
	}
	*first[3].Timer.Milliseconds = 4000
	if *second[3].Timer.Milliseconds != 12345 {
		t.Fatal("reused beat shares timer storage")
	}
	first[0].Timer = &gp.BeatTimer{Milliseconds: timerMilliseconds(50)}
	first[1].Timer = nil
	first[2].Timer = &gp.BeatTimer{}
	song.Version = gp.Version{}
	song.Tracks[0].Settings = gp.TrackSettings{Notation: true}
	before, _ := json.Marshal(song)
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("strict export=%v %#v", err, report)
	}
	after, _ := json.Marshal(song)
	if string(before) != string(after) {
		t.Fatal("export mutates source")
	}
	parsed, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := parsed.Tracks[0].Staves[0].Measures[0].Voices[0].Beats
	if *got[0].Timer.Milliseconds != 50 || got[1].Timer != nil || got[2].Timer == nil || got[2].Timer.Milliseconds != nil || *got[3].Timer.Milliseconds != 4000 {
		t.Fatal("edited timer states changed")
	}
	for _, bad := range []int64{-1, 9007199254740992, 9223372036854775807} {
		first[3].Timer.Milliseconds = timerMilliseconds(bad)
		data, report, err = gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{AllowedCodes: []string{"gp8.reject.score.beat.timer"}}})
		if err == nil || len(data) != 0 {
			t.Fatalf("invalid timer %d exported: %#v", bad, report)
		}
		found := false
		for _, entry := range report.Entries {
			if entry.Code == "gp8.reject.score.beat.timer" {
				found = true
				if entry.Location.Track != 0 || entry.Location.Staff != 0 || entry.Location.Measure != 0 || entry.Location.Voice != 0 || entry.Location.Beat != 3 {
					t.Fatalf("timer rejection location=%#v", entry.Location)
				}
			}
		}
		if !found {
			t.Fatalf("missing timer rejection for %d: %#v", bad, report)
		}
	}
}
func timerMilliseconds(value int64) *int64 { return &value }
