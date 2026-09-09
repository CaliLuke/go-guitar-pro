// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestSemanticMatrixM10DynamicQuantization(t *testing.T) {
	runSemanticMatrixM10DynamicQuantization(newSemanticMatrixRun(t))
}

func runSemanticMatrixM10DynamicQuantization(run *semanticMatrixRun) {
	t := run.t
	canonical := []struct {
		velocity int16
		marking  string
	}{
		{15, "PPP"}, {31, "PP"}, {47, "P"}, {63, "MP"},
		{79, "MF"}, {95, "F"}, {111, "FF"}, {127, "FFF"},
	}
	for _, test := range canonical {
		for _, status := range []BeatStatus{BeatStatusNormal, BeatStatusRest} {
			name := fmt.Sprintf("canonical/%s/%d", ticket33StatusName(status), test.velocity)
			t.Run(name, func(t *testing.T) {
				ticket33AssertDynamicConversion(run, status, test.velocity, test.velocity, test.marking)
			})
		}
	}

	boundaries := []struct {
		velocity int16
		target   int16
		marking  string
	}{
		{22, 15, "PPP"}, {23, 31, "PP"}, {24, 31, "PP"},
		{38, 31, "PP"}, {39, 47, "P"}, {40, 47, "P"},
		{54, 47, "P"}, {55, 63, "MP"}, {56, 63, "MP"},
		{70, 63, "MP"}, {71, 79, "MF"}, {72, 79, "MF"},
		{86, 79, "MF"}, {87, 95, "F"}, {88, 95, "F"},
		{102, 95, "F"}, {103, 111, "FF"}, {104, 111, "FF"},
		{118, 111, "FF"}, {119, 127, "FFF"}, {120, 127, "FFF"},
	}
	for _, test := range boundaries {
		name := fmt.Sprintf("boundary/%d", test.velocity)
		t.Run(name, func(t *testing.T) {
			ticket33AssertDynamicConversion(run, BeatStatusNormal, test.velocity, test.target, test.marking)
		})
	}

	for _, test := range []struct {
		velocity int16
		target   int16
		marking  string
	}{{1, 15, "PPP"}, {14, 15, "PPP"}, {15, 15, "PPP"}, {126, 127, "FFF"}, {127, 127, "FFF"}} {
		name := fmt.Sprintf("endpoint/%d", test.velocity)
		t.Run(name, func(t *testing.T) {
			ticket33AssertDynamicConversion(run, BeatStatusNormal, test.velocity, test.target, test.marking)
		})
	}
}

func TestTicket33AuthoredDynamicCanonicalNoteRegression(t *testing.T) {
	song := ticket33Song(t, BeatStatusNormal, 80, []int16{79})
	ticket33AssertDynamicReports(t, song, []string{"gp8.normalize.beat-dynamic"})
}

func TestTicket33AuthoredDynamicRestRegression(t *testing.T) {
	song := ticket33Song(t, BeatStatusRest, 80, nil)
	ticket33AssertDynamicReports(t, song, []string{"gp8.normalize.beat-dynamic"})
}

func TestTicket33DynamicAuthorityCombinations(t *testing.T) {
	tests := []struct {
		name       string
		dynamics   int16
		velocities []int16
		wantCodes  []string
	}{
		{name: "same noncanonical note", dynamics: 80, velocities: []int16{80}, wantCodes: []string{"gp8.normalize.beat-dynamic", "gp8.normalize.note-velocity"}},
		{name: "canonical target note", dynamics: 80, velocities: []int16{79}, wantCodes: []string{"gp8.normalize.beat-dynamic"}},
		{name: "different canonical note", dynamics: 80, velocities: []int16{95}, wantCodes: []string{"gp8.normalize.beat-dynamic", "gp8.normalize.note-velocity"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			song := ticket33Song(t, BeatStatusNormal, test.dynamics, test.velocities)
			ticket33AssertDynamicReports(t, song, test.wantCodes)
		})
	}

	t.Run("noncanonical rest", func(t *testing.T) {
		song := ticket33Song(t, BeatStatusRest, 80, nil)
		ticket33AssertDynamicReports(t, song, []string{"gp8.normalize.beat-dynamic"})
	})

	t.Run("explicit empty is independent", func(t *testing.T) {
		song := ticket33Song(t, BeatStatusEmpty, 80, nil)
		preflight := PreflightExport(song, ExportFormatGP8, ExportOptions{})
		if got := ticket33ReportCodes(preflight); !slices.Equal(got, []string{"gp8.normalize.empty-beat", "gp8.normalize.beat-dynamic"}) {
			t.Fatalf("report codes = %v", got)
		}
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{
			RequirePreservation: true, AllowedCodes: []string{"gp8.normalize.beat-dynamic"},
		}})
		var lossErr *ExportLossError
		if len(data) != 0 || !errors.As(err, &lossErr) || !reflect.DeepEqual(report, preflight) {
			t.Fatalf("selective strict export = %d bytes, %#v, %v", len(data), report.Entries, err)
		}
		if len(lossErr.Entries) != 1 || lossErr.Entries[0].Code != "gp8.normalize.empty-beat" {
			t.Fatalf("refused entries = %#v", lossErr.Entries)
		}
	})

	t.Run("chord notes", func(t *testing.T) {
		song := ticket33Song(t, BeatStatusNormal, 80, []int16{79, 95, 111})
		beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
		beat.Effect.Chord = &Chord{Name: "G", Length: 6, Strings: []int8{3, 0, 0, 0, 2, 3}}
		ticket33AssertDynamicReports(t, song, []string{"gp8.normalize.beat-dynamic", "gp8.normalize.note-velocity"})
	})
}

func TestTicket33UnsetDynamicFallback(t *testing.T) {
	tests := []struct {
		name       string
		status     BeatStatus
		velocities []int16
		wantCodes  []string
	}{
		{name: "canonical note", status: BeatStatusNormal, velocities: []int16{79}},
		{name: "noncanonical note", status: BeatStatusNormal, velocities: []int16{80}, wantCodes: []string{"gp8.normalize.note-velocity"}},
		{name: "rest", status: BeatStatusRest},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			song := ticket33Song(t, test.status, 0, test.velocities)
			ticket33AssertDynamicReports(t, song, test.wantCodes)
		})
	}
}

func TestTicket33RejectsInvalidBeatDynamics(t *testing.T) {
	for _, dynamics := range []int16{-1, 128} {
		t.Run(fmt.Sprint(dynamics), func(t *testing.T) {
			song := ticket33SongUnchecked(t, BeatStatusNormal, dynamics, []int16{79})
			diagnostics := ValidateSong(song)
			if !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool {
				return diagnostic.Code == "score.beat.dynamics" && diagnostic.Location == (ScoreLocation{})
			}) {
				t.Fatalf("diagnostics = %#v", diagnostics)
			}
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
			if len(data) != 0 || err == nil || !hasExportCode(report, "gp8.reject.score.beat.dynamics") {
				t.Fatalf("export = %d bytes, %#v, %v", len(data), report.Entries, err)
			}
		})
	}
}

func TestTicket33DynamicPoliciesAndReadOnlyPlanning(t *testing.T) {
	song := ticket33Song(t, BeatStatusNormal, 80, []int16{95})
	options := ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}}
	songBefore := fmt.Sprintf("%#v", song)
	optionsBefore := fmt.Sprintf("%#v", options)
	preflight := PreflightExport(song, ExportFormatGP8, options)
	data, report, err := ExportWithReport(song, ExportFormatGP8, options)
	var lossErr *ExportLossError
	if len(data) != 0 || !errors.As(err, &lossErr) || !reflect.DeepEqual(report, preflight) {
		t.Fatalf("strict export = %d bytes, %#v, %v", len(data), report.Entries, err)
	}
	if fmt.Sprintf("%#v", song) != songBefore || fmt.Sprintf("%#v", options) != optionsBefore {
		t.Fatal("preflight or export mutated its input")
	}

	for _, test := range []struct {
		name    string
		allowed []string
		want    string
	}{
		{name: "dynamic only", allowed: []string{"gp8.normalize.beat-dynamic"}, want: "gp8.normalize.note-velocity"},
		{name: "unrelated", allowed: []string{"gp8.omit.writer"}, want: "gp8.normalize.beat-dynamic"},
	} {
		t.Run(test.name, func(t *testing.T) {
			exportData, _, exportErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{
				RequirePreservation: true, AllowedCodes: test.allowed,
			}})
			var policyErr *ExportLossError
			if len(exportData) != 0 || !errors.As(exportErr, &policyErr) || !slices.ContainsFunc(policyErr.Entries, func(entry ExportReportEntry) bool { return entry.Code == test.want }) {
				t.Fatalf("selective export = %d bytes, %#v, %v", len(exportData), policyErr, exportErr)
			}
		})
	}

	rest := ticket33Song(t, BeatStatusRest, 80, nil)
	data, report, err = ExportWithReport(rest, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{
		RequirePreservation: true, AllowedCodes: []string{"gp8.normalize.beat-dynamic"},
	}})
	if err != nil || len(data) == 0 || !slices.Equal(ticket33ReportCodes(report), []string{"gp8.normalize.beat-dynamic"}) {
		t.Fatalf("allowed rest export = %d bytes, %#v, %v", len(data), report.Entries, err)
	}
}

func TestTicket33ReportsExactBeatLocation(t *testing.T) {
	song := ticket33Song(t, BeatStatusNormal, Forte, []int16{Forte})
	voice := &song.Tracks[0].Measures[0].Voices[0]
	voice.Beats = append(voice.Beats, Beat{Duration: defaultDuration(), Status: BeatStatusRest, Dynamics: 80})
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	entry := exportReportEntry(report, "gp8.normalize.beat-dynamic")
	want := ScoreLocation{Track: 0, Staff: 0, Measure: 0, Voice: 0, Beat: 1}
	if entry == nil || entry.Location != want {
		t.Fatalf("dynamic report = %#v, want location %#v", entry, want)
	}
}

func TestTicket33AlphaTabReadsNormalAndRestDynamic(t *testing.T) {
	requireAlphaTabConformance(t)
	song := ticket33Song(t, BeatStatusNormal, 80, []int16{79})
	voice := &song.Tracks[0].Measures[0].Voices[0]
	voice.Beats = append(voice.Beats, Beat{Duration: defaultDuration(), Status: BeatStatusRest, Dynamics: 80})
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{
		RequirePreservation: true, AllowedCodes: []string{"gp8.normalize.beat-dynamic"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	alphaScore := readAlphaTabScore(t, writeConformanceFixture(t, data))
	facts := collectConformanceFacts(alphaScore, map[string]bool{"dynamic": true}, nil)
	values := make([]string, 0, len(facts))
	for _, fact := range facts {
		item := fact.(map[string]any)
		if !strings.Contains(item["path"].(string), "/notes/") {
			values = append(values, item["value"].(string))
		}
	}
	if !slices.Equal(values, []string{"mf", "mf"}) {
		t.Fatalf("AlphaTab dynamics = %v", values)
	}
}

func ticket33AssertDynamicConversion(run *semanticMatrixRun, status BeatStatus, source, target int16, marking string) {
	t := run.t
	t.Helper()
	var velocities []int16
	if status == BeatStatusNormal {
		velocities = []int16{target}
	}
	song := ticket33Song(t, status, source, velocities)
	wantCodes := []string(nil)
	if source != target {
		wantCodes = []string{"gp8.normalize.beat-dynamic"}
	}
	preflight := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	if got := ticket33ReportCodes(preflight); !slices.Equal(got, wantCodes) {
		t.Fatalf("preflight codes = %v, want %v", got, wantCodes)
	}
	options := ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: wantCodes}}
	data, report, err := ExportWithReport(song, ExportFormatGP8, options)
	if err != nil || !reflect.DeepEqual(report, preflight) {
		t.Fatalf("export report = %#v, preflight = %#v, err = %v", report.Entries, preflight.Entries, err)
	}
	values := extractGPIFLeafText(t, data)
	run.Wire("gpifBeat.Dynamic", values["GPIF/Beats/Beat/Dynamic"], marking)
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Normalized("Beat.Dynamics", roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Dynamics, target)
	if source != target {
		strict, _, strictErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
		var lossErr *ExportLossError
		if len(strict) != 0 || !errors.As(strictErr, &lossErr) {
			t.Fatalf("strict export = %d bytes, %v", len(strict), strictErr)
		}
	}
}

func ticket33AssertDynamicReports(t *testing.T, song *Song, wantCodes []string) {
	t.Helper()
	preflight := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	if got := ticket33ReportCodes(preflight); !slices.Equal(got, wantCodes) {
		t.Fatalf("preflight codes = %v, want %v", got, wantCodes)
	}
	for _, entry := range preflight.Entries {
		if slices.Contains([]string{"gp8.normalize.beat-dynamic", "gp8.normalize.note-velocity"}, entry.Code) && entry.Location != (ScoreLocation{}) {
			t.Fatalf("dynamic location = %#v", entry.Location)
		}
	}
}

func ticket33ReportCodes(report ExportReport) []string {
	var result []string
	for _, entry := range report.Entries {
		if stringsHasDynamicCode(entry.Code) || entry.Code == "gp8.normalize.empty-beat" {
			result = append(result, entry.Code)
		}
	}
	return result
}

func stringsHasDynamicCode(code string) bool {
	return code == "gp8.normalize.beat-dynamic" || code == "gp8.normalize.note-velocity"
}

func ticket33Song(t *testing.T, status BeatStatus, dynamics int16, velocities []int16) *Song {
	t.Helper()
	song := ticket33SongUnchecked(t, status, dynamics, velocities)
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("ticket 33 score diagnostics = %#v", diagnostics)
	}
	return song
}

func ticket33SongUnchecked(t *testing.T, status BeatStatus, dynamics int16, velocities []int16) *Song {
	t.Helper()
	song := m11Song(t)
	song.Tracks[0].Settings.Notation = true
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	beat.Status = status
	beat.Dynamics = dynamics
	beat.Notes = nil
	for index, velocity := range velocities {
		beat.Notes = append(beat.Notes, Note{
			Value: int16(index + 1), String: int8(index + 1), Kind: NoteTypeNormal,
			Velocity: velocity, DurationPercent: 1, Effect: defaultNoteEffect(),
		})
	}
	return song
}

func ticket33StatusName(status BeatStatus) string {
	if status == BeatStatusRest {
		return "rest"
	}
	return "normal"
}
