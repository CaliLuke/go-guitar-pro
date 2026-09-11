// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const metadataTextCase = "M01-METADATA-TEXT"

type metadataTextScenario struct {
	value, consumer string
	trim            bool
}

func metadataTextScenarios() []metadataTextScenario {
	return []metadataTextScenario{
		{"line\nnext\t<&> 日本語", "line\nnext\t<&> 日本語", false},
		{" \n\tboundary\t\n ", " \n\tboundary\t\n ", false},
		{"A<&]]>B", "A<&]]>B", false},
		{"\n\tA<&]]>B\t\n", "A<&]]>B", true},
		{"carriage\rreturn", "carriage\rreturn", false},
		{"\rboundary\r\n", "boundary", true},
	}
}

func metadataTextFields(song *Song) []struct{ public, wire, oracle, value string } {
	return []struct{ public, wire, oracle, value string }{
		{"Name", "Title", "title", song.Name}, {"Subtitle", "SubTitle", "subtitle", song.Subtitle},
		{"Artist", "Artist", "artist", song.Artist}, {"Album", "Album", "album", song.Album},
		{"Words", "Words", "words", song.Words}, {"Author", "Music", "music", song.Author},
		{"Copyright", "Copyright", "copyright", song.Copyright}, {"Transcriber", "Tabber", "tabber", song.Transcriber},
		{"Instructions", "Instructions", "instructions", song.Instructions},
	}
}

func metadataTextSong(t *testing.T, value string) *Song {
	song := consumerLimitSong(t)
	for _, field := range metadataTextFields(song) {
		reflect.ValueOf(song).Elem().FieldByName(field.public).SetString(value)
	}
	song.Notice = []string{value}
	return song
}

func TestConformanceMetadataText(t *testing.T) { runConformanceMetadataText(newConformanceRun(t)) }
func runConformanceMetadataText(run *conformanceRun) {
	for _, scenario := range metadataTextScenarios() {
		run.t.Run(scenario.value, func(t *testing.T) {
			song := metadataTextSong(t, scenario.value)
			codes := []string(nil)
			if strings.Contains(scenario.value, "\n") {
				codes = append(codes, "gp8.normalize.notice-lines")
			}
			if scenario.trim {
				codes = append(codes, "gp8.normalize.metadata-text-consumer")
			}
			data, report := assertMetadataTextPolicy(t, song, codes)
			run.Report(metadataTextCase, reportCodes(report), append([]string{}, codes...))
			wire := extractGPIFLeafText(t, data)
			for _, field := range metadataTextFields(song) {
				run.Preserved("Song."+field.public, field.value, scenario.value)
				run.Wire("gpifScore."+field.wire, wire["GPIF/Score/"+field.wire], scenario.value)
			}
			run.Normalized("Song.Notice", song.Notice, []string{scenario.value})
			run.Wire("gpifScore.Notices", wire["GPIF/Score/Notices"], scenario.value)
			parsed, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			for _, field := range metadataTextFields(parsed) {
				if field.value != scenario.value {
					t.Fatalf("%s=%q", field.public, field.value)
				}
			}
		})
	}
	for _, test := range []struct {
		notices []string
		codes   []string
	}{
		{[]string{"First notice", "Second notice"}, nil},
		{[]string{"", "embedded\nline", ""}, []string{"gp8.normalize.notice-lines"}},
		{[]string{""}, []string{"gp8.omit.empty-notice"}},
		{[]string{"", "middle", ""}, nil},
		{[]string{"", ""}, nil},
	} {
		song := consumerLimitSong(run.t)
		song.Notice = test.notices
		data, report := assertMetadataTextPolicy(run.t, song, test.codes)
		run.Report(metadataTextCase, reportCodes(report), append([]string{}, test.codes...))
		var wire struct {
			Notices string `xml:"Score>Notices"`
		}
		if err := xml.Unmarshal([]byte(conformanceBrushGPIF(run.t, data)), &wire); err != nil {
			run.t.Fatal(err)
		}
		run.Wire("gpifScore.Notices", wire.Notices, strings.Join(test.notices, "\n"))
	}
	runConformanceLegacyMetadataLosses(run)
}

func assertMetadataTextPolicy(t *testing.T, song *Song, codes []string) ([]byte, ExportReport) {
	t.Helper()
	before := conformanceContractSnapshot(song, false)
	for _, allowed := range [][]string{nil, {"unrelated"}, codes} {
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
		if !slices.Equal(reportCodes(report), codes) {
			t.Fatalf("codes=%v want%v", reportCodes(report), codes)
		}
		if len(codes) > 0 && !slices.Equal(allowed, codes) {
			var loss *ExportLossError
			if len(data) != 0 || !errors.As(err, &loss) {
				t.Fatalf("strict bytes%d err%v", len(data), err)
			}
		} else if err != nil || len(data) == 0 {
			t.Fatalf("allowed bytes%d err%v", len(data), err)
		}
		if after := conformanceContractSnapshot(song, false); !reflect.DeepEqual(after, before) {
			t.Fatal("export mutated input")
		}
	}
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return data, report
}

func runConformanceLegacyMetadataLosses(run *conformanceRun) {
	tests := []struct {
		field, code string
		value       any
		edit        func(*Song)
	}{
		{"Song.Writer", "gp8.omit.writer", "legacy writer", func(s *Song) { s.Writer = "legacy writer" }},
		{"Song.Comments", "gp8.omit.comments", "legacy comments", func(s *Song) { s.Comments = "legacy comments" }},
		{"Song.Date", "gp8.omit.date", "2026-09-11", func(s *Song) { s.Date = "2026-09-11" }},
		{"Song.Clipboard", "gp8.omit.clipboard-range", &Clipboard{StartMeasure: 1, StopMeasure: 3}, func(s *Song) { s.Clipboard = &Clipboard{StartMeasure: 1, StopMeasure: 3} }},
	}
	for _, test := range tests {
		song := consumerLimitSong(run.t)
		test.edit(song)
		_, report := assertMetadataTextPolicy(run.t, song, []string{test.code})
		if report.Entries[0].Location != (ScoreLocation{}) {
			run.t.Fatal(report)
		}
		run.Omitted(test.field, reflect.ValueOf(song).Elem().FieldByName(strings.TrimPrefix(test.field, "Song.")).Interface(), test.value)
		run.Report(metadataTextCase, reportCodes(report), []string{test.code})
	}
	song := consumerLimitSong(run.t)
	song.Writer = "writer"
	song.Comments = "comments"
	song.Date = "date"
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.omit.writer", "gp8.omit.comments"}}})
	if err == nil || len(data) != 0 || !slices.Equal(reportCodes(report), []string{"gp8.omit.writer", "gp8.omit.comments", "gp8.omit.date"}) {
		run.t.Fatal("partial legacy allowances escaped", report, err)
	}
}

func TestAlphaTabMetadataText(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, scenario := range metadataTextScenarios() {
		t.Run(scenario.value, func(t *testing.T) {
			song := metadataTextSong(t, scenario.value)
			output, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]string
			readAlphaTabOracleFacts(t, "--metadata-text", writeConformanceFixture(t, output), &got)
			for _, field := range metadataTextFields(song) {
				if got[field.oracle] != scenario.consumer {
					t.Fatalf("%s got%q want%q", field.oracle, got[field.oracle], scenario.consumer)
				}
			}
			if got["notices"] != scenario.consumer {
				t.Fatalf("notices=%q want%q", got["notices"], scenario.consumer)
			}
		})
	}
	for _, notices := range [][]string{{"First notice", "Second notice"}, {"", "embedded\nline", ""}, {""}, {"", "middle", ""}, {"", ""}} {
		song := consumerLimitSong(t)
		song.Notice = notices
		song.Author = "independent music"
		song.Transcriber = "independent tabber"
		song.Writer = "separate omitted writer"
		data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
		if err != nil {
			t.Fatal(err)
		}
		var got map[string]string
		readAlphaTabOracleFacts(t, "--metadata-text", writeConformanceFixture(t, data), &got)
		if got["notices"] != strings.Join(notices, "\n") || got["music"] != song.Author || got["tabber"] != song.Transcriber {
			t.Fatalf("consumer=%#v", got)
		}
	}
}
