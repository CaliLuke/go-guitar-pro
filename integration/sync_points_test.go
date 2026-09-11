// SPDX-License-Identifier: MIT

package integration_test

import (
	"reflect"
	"slices"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestGP8SyncPointExport(t *testing.T) {
	song := doubleBarSong(t, []bool{false, false})
	song.BackingTrack = &gp.BackingTrack{Enabled: true, Source: "Local", AssetID: "audio", EmbeddedFilePath: "Content/BackingTrack/audio.bin", AudioData: []byte{1, 2, 3}, FramePadding: -22050}
	p, _ := gp.NewBarPosition(1, 4)
	q, _ := gp.NewBarPosition(3, 4)
	song.SyncPoints = []gp.SyncPoint{
		{Bar: 0, Position: 0.25, BarPosition: p, BarOccurrence: 1, FrameOffset: 44100, AudioFrame: 44100, MediaTimeMS: 1500, ModifiedTempo: 123.5, OriginalTempo: 100.25, Visible: true},
		{Bar: 1, Position: 0.75, BarPosition: q, BarOccurrence: 3, FrameOffset: 88200, AudioFrame: 88200, MediaTimeMS: 2500, ModifiedTempo: 91.25, OriginalTempo: 99.5, Linear: true},
	}
	before := append([]gp.SyncPoint(nil), song.SyncPoints...)
	code := "gp8.omit.sync-point-consumer-tempo"
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{code}}})
	if err != nil || len(report.Entries) != 2 {
		t.Fatalf("export %v %#v", err, report)
	}
	for i, e := range report.Entries {
		if e.Code != code || e.Location != (gp.ScoreLocation{Measure: i}) {
			t.Fatal(e)
		}
	}
	parsed, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(parsed.SyncPoints, before) {
		t.Fatalf("points %#v want %#v", parsed.SyncPoints, before)
	}
	if !reflect.DeepEqual(song.SyncPoints, before) {
		t.Fatal("export mutated input")
	}
	for _, allowed := range [][]string{nil, {"gp8.omit.sync-points"}} {
		data, _, err = gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
		if err == nil || len(data) != 0 {
			t.Fatal("unallowed consumer metadata loss succeeded")
		}
	}
}

func TestSyncPointCheckedAuthority(t *testing.T) {
	song := doubleBarSong(t, []bool{false})
	position, _ := gp.NewBarPosition(1, 4)
	song.SyncPoints = []gp.SyncPoint{{BarPosition: position, Position: 0.25, AudioFrame: 44100, FrameOffset: 44100, MediaTimeMS: 1000, Visible: true}}
	data, _, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(data) == 0 {
		t.Fatalf("exact strict control %v", err)
	}
	song.SyncPoints[0].FrameOffset = 88200
	song.SyncPoints[0].Position = 0.75
	codes := []string{"gp8.normalize.sync-point-frame-authority", "gp8.normalize.sync-point-position-authority"}
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: codes}})
	if err != nil || len(report.Entries) != 2 {
		t.Fatalf("authority %v %#v", err, report)
	}
	parsed, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.SyncPoints[0].FrameOffset != 44100 || parsed.SyncPoints[0].Position != 0.25 {
		t.Fatal(parsed.SyncPoints)
	}
	song.SyncPoints[0].BarPosition, _ = gp.NewBarPosition(1, 3)
	data, report, err = gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
	if err == nil || len(data) != 0 || !slices.ContainsFunc(report.Entries, func(e gp.ExportReportEntry) bool { return e.Code == "gp8.reject.sync-point-position-precision" }) {
		t.Fatalf("precision %v %#v", err, report)
	}
}
