// SPDX-License-Identifier: MIT

package integration_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestExportReportSummaryPreservesOccurrenceEntries(t *testing.T) {
	song := doubleBarSong(t, []bool{false, false, false})
	for i := range song.MeasureHeaders {
		song.MeasureHeaders[i].Marker = &gp.Marker{Title: "Section", Color: int32(i + 1)}
		song.MeasureHeaders[i].TimeSignature.Beams = [4]uint8{3, 2, 2, 2}
	}
	if diagnostics := gp.ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("fixture invalid: %#v", diagnostics)
	}

	report := gp.PreflightExport(song, gp.ExportFormatGP8, gp.ExportOptions{})
	wantSummary := []gp.ExportReportSummary{
		{Code: "gp8.omit.marker-color", Feature: "score-core", Disposition: gp.ExportDispositionOmitted, Count: 3},
		{Code: "gp8.omit.time-signature-beams", Feature: "rhythm", Disposition: gp.ExportDispositionOmitted, Count: 3},
	}
	if got := report.Summary(); !reflect.DeepEqual(got, wantSummary) {
		t.Fatalf("summary = %#v, want %#v", got, wantSummary)
	}
	if len(report.Entries) != 6 {
		t.Fatalf("summary discarded occurrences: entries=%d", len(report.Entries))
	}
	for i, entry := range report.Entries {
		if entry.Location.Measure != i/2 {
			t.Fatalf("entry %d lost its source occurrence: %#v", i, entry)
		}
	}

	before := append([]gp.ExportReportEntry(nil), report.Entries...)
	first := report.Summary()
	first[0].Count = 99
	first[0].Code = "mutated"
	second := report.Summary()
	if !reflect.DeepEqual(second, wantSummary) {
		t.Fatalf("summary results alias: %#v", second)
	}
	if !reflect.DeepEqual(report.Entries, before) {
		t.Fatal("Summary mutated report entries")
	}
}

func TestExportLossErrorSummarizesOnlyRefusedCodes(t *testing.T) {
	song := doubleBarSong(t, []bool{false, false})
	for i := range song.MeasureHeaders {
		song.MeasureHeaders[i].Marker = &gp.Marker{Title: "Section", Color: int32(i + 1)}
		song.MeasureHeaders[i].TimeSignature.Beams = [4]uint8{3, 2, 2, 2}
	}
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{
		LossPolicy: gp.ExportLossPolicy{
			RequirePreservation: true,
			AllowedCodes:        []string{"gp8.omit.marker-color"},
		},
	})
	var loss *gp.ExportLossError
	if len(data) != 0 || !errors.As(err, &loss) {
		t.Fatalf("strict export = %d bytes, %v; want ExportLossError", len(data), err)
	}
	if len(report.Entries) != 4 || len(loss.Entries) != 2 {
		t.Fatalf("full/refused occurrences = %d/%d, want 4/2", len(report.Entries), len(loss.Entries))
	}
	for _, entry := range loss.Entries {
		if entry.Code != "gp8.omit.time-signature-beams" {
			t.Fatalf("allowed code leaked into refusal: %#v", loss.Entries)
		}
	}
	wantError := "export loss policy rejected 2 conversion(s) (codes: gp8.omit.time-signature-beams=2)"
	if err.Error() != wantError {
		t.Fatalf("error = %q, want %q", err, wantError)
	}
	if got := report.Summary(); len(got) != 2 || got[0].Count != 2 || got[1].Count != 2 {
		t.Fatalf("full report summary = %#v", got)
	}
}

func TestExportReportSummaryGroupsExactFieldsAndCountsOccurrences(t *testing.T) {
	report := gp.ExportReport{Entries: []gp.ExportReportEntry{
		{Code: "z", Feature: "f", Disposition: gp.ExportDispositionOmitted, Location: gp.ScoreLocation{Measure: 3}, Reason: "one"},
		{Code: "a", Feature: "f2", Disposition: gp.ExportDispositionRejected},
		{Code: "a", Feature: "f", Disposition: gp.ExportDispositionOmitted, Location: gp.ScoreLocation{Measure: 0}, Reason: "first"},
		{Code: "a", Feature: "f", Disposition: gp.ExportDispositionOmitted, Location: gp.ScoreLocation{Measure: 0}, Reason: "second"},
		{Code: "a", Feature: "f", Disposition: gp.ExportDispositionNormalized},
		{Code: "a", Feature: "f2", Disposition: gp.ExportDispositionOmitted},
	}}
	want := []gp.ExportReportSummary{
		{Code: "a", Feature: "f", Disposition: gp.ExportDispositionNormalized, Count: 1},
		{Code: "a", Feature: "f", Disposition: gp.ExportDispositionOmitted, Count: 2},
		{Code: "a", Feature: "f2", Disposition: gp.ExportDispositionOmitted, Count: 1},
		{Code: "a", Feature: "f2", Disposition: gp.ExportDispositionRejected, Count: 1},
		{Code: "z", Feature: "f", Disposition: gp.ExportDispositionOmitted, Count: 1},
	}
	entries := append([]gp.ExportReportEntry(nil), report.Entries...)
	for i := 0; i < 5; i++ {
		if got := report.Summary(); !reflect.DeepEqual(got, want) {
			t.Fatalf("summary run %d = %#v, want %#v", i, got, want)
		}
	}
	if !reflect.DeepEqual(report.Entries, entries) {
		t.Fatal("summary changed occurrence order or content")
	}
}

func TestExportReportSummaryAndLossErrorEmptyBehavior(t *testing.T) {
	if got := (gp.ExportReport{}).Summary(); len(got) != 0 {
		t.Fatalf("empty summary = %#v, want no entries", got)
	}
	const prefix = "export loss policy rejected 0 conversion(s)"
	if got := (&gp.ExportLossError{}).Error(); got != prefix {
		t.Fatalf("empty error = %q, want %q", got, prefix)
	}
	if !strings.HasPrefix((&gp.ExportLossError{Entries: []gp.ExportReportEntry{{Code: "x"}}}).Error(), "export loss policy rejected 1 conversion(s)") {
		t.Fatal("nonempty error lost the existing count prefix")
	}
}

func TestExportLossErrorCountsDistinctCodes(t *testing.T) {
	errorWithUnorderedCodes := &gp.ExportLossError{Entries: []gp.ExportReportEntry{
		{Code: "z.code"}, {Code: "a.code"}, {Code: "a.code"},
	}}
	const wantError = "export loss policy rejected 3 conversion(s) (codes: a.code=2, z.code=1)"
	for i := 0; i < 5; i++ {
		if got := errorWithUnorderedCodes.Error(); got != wantError {
			t.Fatalf("error run %d = %q, want %q", i, got, wantError)
		}
	}
}
