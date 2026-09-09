// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"testing"
)

func TestExportPreflightReportsActualGP8Losses(t *testing.T) {
	clean := PreflightExport(syntheticGP8Song(), ExportFormatGP8, ExportOptions{})
	if len(clean.Entries) != 0 {
		t.Fatalf("clean preflight = %#v", clean.Entries)
	}

	song := syntheticGP8Song()
	song.BackingTrack = &BackingTrack{Name: "omitted audio", AudioData: []byte("audio")}
	song.SyncPoints = []SyncPoint{{Bar: 0}}
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	beat.Notes = append(beat.Notes, beat.Notes[0])
	beat.Notes[1].Velocity = beat.Notes[0].Velocity + 1

	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	for _, code := range []string{"gp8.omit.backing-track", "gp8.omit.sync-points", "gp8.normalize.note-velocity"} {
		if !hasExportReportEntry(report, code) {
			t.Errorf("report = %#v, want %s", report.Entries, code)
		}
	}
	velocity := exportReportEntry(report, "gp8.normalize.note-velocity")
	if velocity == nil || velocity.Location.Track != 0 || velocity.Location.Measure != 0 || velocity.Location.Beat != 0 {
		t.Fatalf("velocity entry location = %#v", velocity)
	}
}

func TestExportStrictLossPolicyUsesStableAllowlist(t *testing.T) {
	song := syntheticGP8Song()
	song.BackingTrack = &BackingTrack{Name: "omitted audio", AudioData: []byte("audio")}

	options := ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}}
	data, report, err := ExportWithReport(song, ExportFormatGP8, options)
	var lossErr *ExportLossError
	if !errors.As(err, &lossErr) || len(data) != 0 || !hasExportReportEntry(report, "gp8.omit.backing-track") {
		t.Fatalf("strict export = %d bytes, %#v, %v", len(data), report, err)
	}

	options.LossPolicy.AllowedCodes = []string{"gp8.omit.backing-track"}
	data, report, err = ExportWithReport(song, ExportFormatGP8, options)
	if err != nil || len(data) == 0 || !hasExportReportEntry(report, "gp8.omit.backing-track") {
		t.Fatalf("allowed export = %d bytes, %#v, %v", len(data), report, err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if roundTrip.BackingTrack != nil {
		t.Fatalf("independent output retained omitted backing track: %#v", roundTrip.BackingTrack)
	}
}

func hasExportReportEntry(report ExportReport, code string) bool {
	return exportReportEntry(report, code) != nil
}

func exportReportEntry(report ExportReport, code string) *ExportReportEntry {
	for index := range report.Entries {
		if report.Entries[index].Code == code {
			return &report.Entries[index]
		}
	}
	return nil
}
