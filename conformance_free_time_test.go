// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"slices"
	"strings"
	"testing"
)

func TestConformanceFreeTime(t *testing.T) {
	runConformanceFreeTime(newConformanceRun(t))
}

func runConformanceFreeTime(run *conformanceRun) {
	t := run.t
	fixture := parseTestFixture(t, "testdata/gp7/free-time.gp")
	wantFixture := []bool{false, true, true, false, false, false, true}
	run.ClaimPrimary(claimSite("free-time", "import", "M07-FREE-TIME", "present")).Preserved("MeasureHeader.FreeTime", conformanceFreeTimeFlags(fixture), wantFixture)

	// Free time is authored notation, not a request to derive measure length
	// from voice contents. A non-pickup measure advances by its numeric meter
	// even when one voice is shorter and another is longer than that meter.
	timing := conformanceUnequalFreeTimeSong()
	if err := FinalizeSong(timing); err != nil {
		t.Fatal(err)
	}
	wantStarts := []int64{DurationQuarterTime, 5 * DurationQuarterTime, 9 * DurationQuarterTime}
	gotStarts := make([]int64, len(timing.MeasureHeaders))
	for index := range timing.MeasureHeaders {
		gotStarts[index] = timing.MeasureHeaders[index].Start
	}
	run.Derived("MeasureHeader.Start", gotStarts, wantStarts)
	run.Derived("MeasureHeader.ExactStart", []int64{
		timing.MeasureHeaders[0].ExactStart.FloorTicks(),
		timing.MeasureHeaders[1].ExactStart.FloorTicks(),
		timing.MeasureHeaders[2].ExactStart.FloorTicks(),
	}, wantStarts)

	song := conformanceFreeTimeSong(t)
	want := []bool{true, false, true}
	run.ClaimPrimary(claimSite("free-time", "model", "M07-FREE-TIME", "present")).Preserved("MeasureHeader.FreeTime", conformanceFreeTimeFlags(song), want)
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true},
	})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("free-time strict export = %v, %#v", err, report.Entries)
	}
	run.ClaimReport(claimSite("free-time", "export", "M07-FREE-TIME", "present")).Report("M07-FREE-TIME", reportCodes(report), []string{})
	run.ClaimSerialization(claimSite("free-time", "export", "M07-FREE-TIME", "present")).Wire("gpifMasterBar.FreeTime", conformanceFreeTimeWire(t, data), want)
	if raw := conformanceGPIFText(t, data); strings.Count(raw, "<FreeTime></FreeTime>") != 2 {
		t.Fatalf("GPIF FreeTime elements = %q, want two empty presence markers", raw)
	}

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("MeasureHeader.FreeTime", conformanceFreeTimeFlags(roundTrip), want)
	before := conformanceFreeTimeContext(roundTrip)
	roundTrip.MeasureHeaders[0].FreeTime = false
	roundTrip.MeasureHeaders[1].FreeTime = true
	wantEdited := []bool{false, true, true}
	edited, editedReport, err := ExportWithReport(roundTrip, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{
			"gp8.normalize.source-version", "gp8.omit.track-display-settings",
		}},
	})
	if err != nil {
		t.Fatalf("edited free-time export = %v, %#v", err, editedReport.Entries)
	}
	run.Wire("gpifMasterBar.FreeTime", conformanceFreeTimeWire(t, edited), wantEdited)
	editedRoundTrip, err := Parse(edited)
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("MeasureHeader.FreeTime", conformanceFreeTimeFlags(editedRoundTrip), wantEdited)
	if after := conformanceFreeTimeContext(editedRoundTrip); after != before {
		t.Fatalf("free-time edit changed meter, timing, or content:\nbefore %s\nafter  %s", before, after)
	}

	// GPIF defines FreeTime by element presence. Text payload is not a boolean
	// and must not invent a diagnostic or reinterpret an authored marker.
	falsePayload := rewriteConformanceGPIF(t, edited, func(gpif string) string {
		return strings.Replace(gpif, "<FreeTime></FreeTime>", "<FreeTime>false</FreeTime>", 1)
	})
	result, strictErr := ParseWithOptions(falsePayload, ParseOptions{Strict: true})
	if strictErr != nil || result == nil {
		t.Fatalf("FreeTime payload parse = %#v, %v", result, strictErr)
	}
	run.ClaimPrimary(claimSite("free-time", "export", "M07-FREE-TIME", "present")).Preserved("MeasureHeader.FreeTime", conformanceFreeTimeFlags(result.Song), wantEdited)
}

func conformanceFreeTimeSong(t *testing.T) *Song {
	t.Helper()
	song := semanticValidPitchedGP8Song(t)
	song.Anacrusis = false
	song.Tracks[0].Settings.Notation = true
	song.MeasureHeaders[0].FreeTime = true
	song.MeasureHeaders[2].FreeTime = true
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("free-time song is invalid: %#v", diagnostics)
	}
	return song
}

func conformanceUnequalFreeTimeSong() *Song {
	headers := []MeasureHeader{defaultMeasureHeader(), defaultMeasureHeader(), defaultMeasureHeader()}
	for index := range headers {
		headers[index].Number = uint16(index + 1)
	}
	headers[0].FreeTime = true
	quarter := defaultBeat()
	longVoice := make([]Beat, 5)
	for index := range longVoice {
		longVoice[index] = defaultBeat()
	}
	return &Song{
		Tempo:          120,
		MeasureHeaders: headers,
		Tracks: []Track{{Measures: []Measure{
			{Number: 1, HeaderIndex: 0, Voices: []Voice{{Beats: []Beat{quarter}}, {Beats: longVoice}}},
			{Number: 2, HeaderIndex: 1, Voices: []Voice{{}}},
			{Number: 3, HeaderIndex: 2, Voices: []Voice{{Beats: []Beat{quarter}}}},
		}}},
	}
}

func conformanceFreeTimeFlags(song *Song) []bool {
	result := make([]bool, len(song.MeasureHeaders))
	for index := range song.MeasureHeaders {
		result[index] = song.MeasureHeaders[index].FreeTime
	}
	return result
}

func conformanceFreeTimeWire(t *testing.T, data []byte) []bool {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	if err := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document); err != nil {
		t.Fatal(err)
	}
	result := make([]bool, len(document.MasterBars.MasterBars))
	for index := range document.MasterBars.MasterBars {
		result[index] = document.MasterBars.MasterBars[index].FreeTime != nil
	}
	return result
}

func conformanceGPIFText(t *testing.T, data []byte) string {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	return string(readZipMember(t, archive, "Content/score.gpif"))
}

func conformanceFreeTimeContext(song *Song) string {
	normalized := normalizeGoScore(song).(map[string]any)
	for _, raw := range normalized["masterBars"].([]any) {
		delete(raw.(map[string]any), "freeTime")
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func TestAlphaTabPreservesFreeTime(t *testing.T) {
	requireAlphaTabConformance(t)
	source := conformanceFreeTimeSong(t)
	data, report, err := ExportWithReport(source, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true},
	})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("free-time AlphaTab export = %v, %#v", err, report.Entries)
	}
	if got := conformanceAlphaTabFreeTime(t, data); !slices.Equal(got, []bool{true, false, true}) {
		t.Fatalf("AlphaTab free-time flags = %v", got)
	}

	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	parsed.MeasureHeaders[0].FreeTime = false
	parsed.MeasureHeaders[1].FreeTime = true
	edited, err := Export(parsed, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	if got := conformanceAlphaTabFreeTime(t, edited); !slices.Equal(got, []bool{false, true, true}) {
		t.Fatalf("AlphaTab edited free-time flags = %v", got)
	}
	conformanceIndependentClaim(t, "field:MeasureHeader.FreeTime", claimSite("free-time", "import", "M07-FREE-TIME", "present"), claimSite("free-time", "model", "M07-FREE-TIME", "present"), claimSite("free-time", "export", "M07-FREE-TIME", "present"))
}

func conformanceAlphaTabFreeTime(t *testing.T, data []byte) []bool {
	t.Helper()
	root := readAlphaTabScore(t, writeConformanceFixture(t, data)).(map[string]any)
	masterBars := root["masterBars"].([]any)
	result := make([]bool, len(masterBars))
	for index, raw := range masterBars {
		result[index] = raw.(map[string]any)["freeTime"].(bool)
	}
	return result
}
