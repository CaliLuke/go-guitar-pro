// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestConformanceClefOctave(t *testing.T) {
	runConformanceClefOctave(newConformanceRun(t))
}

func runConformanceClefOctave(run *conformanceRun) {
	t := run.t
	fixture := parseTestFixture(t, "testdata/gp7/colors.gp")
	fixtureValues := make([]Octave, 0, 6)
	for measureIndex := 3; measureIndex <= 8; measureIndex++ {
		fixtureValues = append(fixtureValues, fixture.Tracks[0].Staves[0].Measures[measureIndex].ClefOctave)
	}
	run.ClaimPrimary(claimSite("clef-octave", "import", "M07-CLEF-OCTAVE", "8va")).Preserved("Measure.ClefOctave", fixtureValues, []Octave{
		OctaveQuindicesima, OctaveQuindicesima, OctaveQuindicesima,
		OctaveQuindicesima, OctaveQuindicesima, OctaveQuindicesima,
	})

	song := conformanceClefOctaveSong(t)
	wantLocations := []Octave{
		OctaveOttava, OctaveQuindicesima, OctaveNone,
		OctaveOttavaBassa, OctaveQuindicesimaBassa, OctaveNone,
	}
	run.ClaimPrimary(claimSite("clef-octave", "model", "M07-CLEF-OCTAVE", "8va")).Preserved("Measure.ClefOctave", conformanceClefOctaves(&song.Tracks[0]), wantLocations)
	run.Preserved("Beat.Octave", song.Tracks[0].Measures[0].Voices[0].Beats[0].Octave, OctaveQuindicesimaBassa)

	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true},
	})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("clef octave export = %v, %#v", err, report.Entries)
	}
	run.ClaimReport(claimSite("clef-octave", "export", "M07-CLEF-OCTAVE", "8va")).Report("M07-CLEF-OCTAVE", reportCodes(report), []string{})
	wantWire := []string{"8va", "8vb", "15ma", "15mb", "", ""}
	run.ClaimSerialization(claimSite("clef-octave", "export", "M07-CLEF-OCTAVE", "8va")).Wire("gpifBar.Ottavia", conformanceClefOctaveWire(t, data), wantWire)
	run.Wire("gpifBeat.Ottavia", extractGPIFLeafText(t, data)["GPIF/Beats/Beat/Ottavia"], "15mb15mb")

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Dispatch("parseGPIFWithContext:bar.Ottavia", conformanceClefOctaves(&roundTrip.Tracks[0]), wantLocations)
	run.Preserved("Measure.ClefOctave", conformanceClefOctaves(&roundTrip.Tracks[0]), wantLocations)
	if got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Octave; got != OctaveQuindicesimaBassa {
		t.Fatalf("round-trip beat octave = %d, want %d", got, OctaveQuindicesimaBassa)
	}

	// The first-staff compatibility measures and later staff measures remain
	// independent after parsing. Swap two bar values without changing the beat.
	roundTrip.Tracks[0].Measures[1].ClefOctave = OctaveOttavaBassa
	roundTrip.Tracks[0].Staves[1].Measures[0].ClefOctave = OctaveQuindicesima
	edited, editedReport, err := ExportWithReport(roundTrip, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{
			"gp8.normalize.source-version", "gp8.omit.track-display-settings",
		}},
	})
	if err != nil {
		t.Fatalf("edited clef octave export = %v, %#v", err, editedReport.Entries)
	}
	wantEditedLocations := []Octave{
		OctaveOttava, OctaveOttavaBassa, OctaveNone,
		OctaveQuindicesima, OctaveQuindicesimaBassa, OctaveNone,
	}
	editedRoundTrip, err := Parse(edited)
	if err != nil {
		t.Fatal(err)
	}
	run.ClaimPrimary(claimSite("clef-octave", "export", "M07-CLEF-OCTAVE", "8va")).Preserved("Measure.ClefOctave", conformanceClefOctaves(&editedRoundTrip.Tracks[0]), wantEditedLocations)
	if got := editedRoundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Octave; got != OctaveQuindicesimaBassa {
		t.Fatalf("edited round-trip beat octave = %d, want %d", got, OctaveQuindicesimaBassa)
	}
	invalid := rewriteConformanceGPIF(t, edited, func(gpif string) string {
		return strings.Replace(gpif, "<Ottavia>8va</Ottavia>", "<Ottavia>22ma</Ottavia>", 1)
	})
	result, strictErr := ParseWithOptions(invalid, ParseOptions{
		Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticUnsupportedFeature},
	})
	var parseErr *StrictParseError
	if !errors.As(strictErr, &parseErr) || result == nil {
		t.Fatalf("invalid clef octave strict parse = %#v, %v, want StrictParseError", result, strictErr)
	}
	diagnostic := conformanceSourceAuditDiagnosticByCode(result.Diagnostics, "GPIF.Bar.Ottavia.InvalidValue")
	if diagnostic == nil || diagnostic.ObjectID == "" || !strings.HasSuffix(diagnostic.SourcePath, "/Ottavia") {
		t.Fatalf("invalid clef octave diagnostic = %#v", diagnostic)
	}

	undefined := conformanceClefOctaveSong(t)
	undefined.Tracks[0].Measures[0].ClefOctave = Octave(99)
	diagnostics := ValidateSong(undefined)
	if !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool {
		return diagnostic.Code == "score.measure.clef-octave" && diagnostic.Location.Track == 0 && diagnostic.Location.Staff == 0 && diagnostic.Location.Measure == 0
	}) {
		t.Fatalf("undefined clef octave diagnostics = %#v", diagnostics)
	}
	undefinedData, undefinedReport, undefinedErr := ExportWithReport(undefined, ExportFormatGP8, ExportOptions{})
	if len(undefinedData) != 0 || undefinedErr == nil || !hasExportCode(undefinedReport, "gp8.reject.score.measure.clef-octave") {
		t.Fatalf("undefined clef octave export = %d bytes, %#v, %v", len(undefinedData), undefinedReport.Entries, undefinedErr)
	}
}

func conformanceClefOctaveSong(t *testing.T) *Song {
	t.Helper()
	song := semanticValidPitchedGP8Song(t)
	track := &song.Tracks[0]
	track.Settings.Notation = true
	track.Measures[0].ClefOctave = OctaveOttava
	track.Measures[1].ClefOctave = OctaveQuindicesima
	track.Measures[0].Voices[0].Beats[0].Octave = OctaveQuindicesimaBassa
	track.Staves[0].Measures = track.Measures
	secondMeasures := slices.Clone(track.Measures)
	secondMeasures[0].ClefOctave = OctaveOttavaBassa
	secondMeasures[1].ClefOctave = OctaveQuindicesimaBassa
	track.Staves = append(track.Staves, Staff{
		Measures:                  secondMeasures,
		Strings:                   slices.Clone(track.Strings),
		StandardNotationLineCount: 5,
	})
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("clef octave song is invalid: %#v", diagnostics)
	}
	return song
}

func conformanceClefOctaves(track *Track) []Octave {
	values := make([]Octave, 0)
	for staffIndex := range track.Staves {
		for measureIndex := range track.Staves[staffIndex].Measures {
			values = append(values, track.Staves[staffIndex].Measures[measureIndex].ClefOctave)
		}
	}
	return values
}

func conformanceClefOctaveWire(t *testing.T, data []byte) []string {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	if err := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document); err != nil {
		t.Fatal(err)
	}
	values := make([]string, len(document.Bars.Bars))
	for index := range document.Bars.Bars {
		values[index] = document.Bars.Bars[index].Ottavia
	}
	return values
}

func conformanceAlphaTabClefOctaves(t *testing.T, data []byte) []string {
	t.Helper()
	root := readAlphaTabScore(t, writeConformanceFixture(t, data)).(map[string]any)
	tracks := root["tracks"].([]any)
	staves := tracks[0].(map[string]any)["staves"].([]any)
	values := make([]string, 0)
	for _, rawStaff := range staves {
		bars := rawStaff.(map[string]any)["bars"].([]any)
		for _, rawBar := range bars {
			values = append(values, rawBar.(map[string]any)["clefOctave"].(string))
		}
	}
	return values
}
