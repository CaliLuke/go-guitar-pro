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

func TestConformanceDirections(t *testing.T) {
	runConformanceDirections(newConformanceRun(t))
}

func runConformanceDirections(run *conformanceRun) {
	t := run.t
	all := append(slices.Clone(directionSignOrder), directionJumpOrder...)

	gp5 := parseTestFixture(t, "testdata/gp5/Directions.gp5")
	gp5Want := []DirectionSign{
		DirectionSignCoda, DirectionSignDoubleCoda, DirectionSignSegno, DirectionSignSegnoSegno, DirectionSignFine,
		DirectionSignDaCapo, DirectionSignDaCapoAlCoda, DirectionSignDaCapoAlDoubleCoda, DirectionSignDaCapoAlFine,
		DirectionSignDaSegno, DirectionSignDaSegnoSegno, DirectionSignDaSegnoAlCoda, DirectionSignDaSegnoAlDoubleCoda,
		DirectionSignDaSegnoSegnoAlCoda, DirectionSignDaSegnoSegnoAlDoubleCoda, DirectionSignDaSegnoAlFine,
		DirectionSignDaSegnoSegnoAlFine, DirectionSignDaCoda, DirectionSignDaDoubleCoda,
	}
	gotGP5 := make([]DirectionSign, len(gp5Want))
	for index := range gotGP5 {
		if len(gp5.MeasureHeaders[index].Directions) != 1 {
			t.Fatalf("GP5 measure %d directions = %v", index, gp5.MeasureHeaders[index].Directions)
		}
		gotGP5[index] = gp5.MeasureHeaders[index].Directions[0]
	}
	run.Preserved("MeasureHeader.Directions", gotGP5, gp5Want)

	gpif := parseTestFixture(t, "testdata/gp7/timer.gp")
	run.Preserved("MeasureHeader.Directions", gpif.MeasureHeaders[3].Directions, []DirectionSign{DirectionSignFine})
	run.Preserved("MeasureHeader.Directions", gpif.MeasureHeaders[7].Directions, []DirectionSign{DirectionSignDaCapoAlFine})

	song := semanticValidPitchedGP8Song(t)
	song.Tracks[0].Settings.Notation = true
	header := &song.MeasureHeaders[0]
	legacy := DirectionSignDaDoubleCoda
	header.Direction = &legacy
	header.Directions = append([]DirectionSign{DirectionSignDaDoubleCoda, DirectionSignCoda}, all...)
	before := conformanceContractSnapshot(song, false)
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("directions export = %v, %#v", err, report.Entries)
	}
	if got := conformanceContractSnapshot(song, false); got != before {
		t.Fatal("directions export mutated the authored score")
	}
	targets, jumps, hasDirections := conformanceDirectionWire(t, data)
	wantTargets := []string{"Coda", "DoubleCoda", "Segno", "SegnoSegno", "Fine"}
	wantJumps := []string{"DaCapo", "DaCapoAlCoda", "DaCapoAlDoubleCoda", "DaCapoAlFine", "DaSegno", "DaSegnoAlCoda", "DaSegnoAlDoubleCoda", "DaSegnoAlFine", "DaSegnoSegno", "DaSegnoSegnoAlCoda", "DaSegnoSegnoAlDoubleCoda", "DaSegnoSegnoAlFine", "DaCoda", "DaDoubleCoda"}
	run.Wire("gpifMasterBar.Directions", hasDirections, true)
	run.Wire("gpifDirections.Targets", targets, wantTargets)
	run.Wire("gpifDirections.Jumps", jumps, wantJumps)
	run.Preserved("MeasureHeader.Directions", header.resolvedDirections(), all)

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("MeasureHeader.Directions", roundTrip.MeasureHeaders[0].Directions, all)
	run.Preserved("MeasureHeader.Direction", *roundTrip.MeasureHeaders[0].Direction, DirectionSignDaDoubleCoda)

	// An unchanged parsed compatibility pointer leaves the complete slice in
	// control. A changed pointer, including nil, replaces that slice.
	roundTrip.MeasureHeaders[0].Directions = []DirectionSign{DirectionSignSegno, DirectionSignCoda, DirectionSignSegno}
	assertDirectionExport(t, roundTrip, []DirectionSign{DirectionSignCoda, DirectionSignSegno})
	changed := DirectionSignDaCapoAlFine
	roundTrip.MeasureHeaders[0].Direction = &changed
	roundTrip.MeasureHeaders[0].Directions = []DirectionSign{DirectionSignFine}
	assertDirectionExport(t, roundTrip, []DirectionSign{DirectionSignDaCapoAlFine})
	roundTrip.MeasureHeaders[0].Direction = nil
	assertDirectionExport(t, roundTrip, nil)

	// A non-nil programmatic slice is authoritative, even when it is empty.
	programmatic := semanticValidPitchedGP8Song(t)
	programmatic.Tracks[0].Settings.Notation = true
	ignored := DirectionSignFine
	programmatic.MeasureHeaders[0].Direction = &ignored
	programmatic.MeasureHeaders[0].Directions = []DirectionSign{}
	assertDirectionExport(t, programmatic, nil)
	programmatic.MeasureHeaders[0].Directions = nil
	assertDirectionExport(t, programmatic, []DirectionSign{DirectionSignFine})

	invalidSource := rewriteConformanceGPIF(t, data, func(gpif string) string {
		return strings.Replace(gpif, "<Target>Coda</Target>", "<Target>ToVerse</Target>", 1)
	})
	result, strictErr := ParseWithOptions(invalidSource, ParseOptions{Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticUnsupportedFeature}})
	var parseErr *StrictParseError
	if !errors.As(strictErr, &parseErr) || result == nil {
		t.Fatalf("invalid target strict parse = %#v, %v", result, strictErr)
	}
	diagnostic := conformanceSourceAuditDiagnosticByCode(result.Diagnostics, "GPIF.MasterBar.Directions.Target.InvalidValue")
	if diagnostic == nil || !strings.Contains(diagnostic.SourcePath, "/Directions/Target[0]") {
		t.Fatalf("invalid target diagnostic = %#v", diagnostic)
	}

	invalid := semanticValidPitchedGP8Song(t)
	invalid.Tracks[0].Settings.Notation = true
	invalid.MeasureHeaders[0].Directions = []DirectionSign{99}
	if diagnostics := ValidateSong(invalid); !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool {
		return diagnostic.Code == "score.measure.direction" && diagnostic.Location.Measure == 0
	}) {
		t.Fatalf("invalid direction diagnostics = %#v", diagnostics)
	}
	invalidData, invalidReport, invalidErr := ExportWithReport(invalid, ExportFormatGP8, ExportOptions{})
	if len(invalidData) != 0 || invalidErr == nil || !hasExportCode(invalidReport, "gp8.reject.score.measure.direction") {
		t.Fatalf("invalid direction export = %d bytes, %#v, %v", len(invalidData), invalidReport.Entries, invalidErr)
	}

}

func assertDirectionExport(t *testing.T, song *Song, want []DirectionSign) {
	t.Helper()
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := roundTrip.MeasureHeaders[0].Directions; !slices.Equal(got, want) {
		t.Fatalf("round-trip directions = %v, want %v", got, want)
	}
}

func conformanceDirectionWire(t *testing.T, data []byte) ([]string, []string, bool) {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	if err := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document); err != nil {
		t.Fatal(err)
	}
	directions := document.MasterBars.MasterBars[0].Directions
	if directions == nil {
		return nil, nil, false
	}
	return directions.Targets, directions.Jumps, true
}

func conformanceAlphaTabDirections(t *testing.T, data []byte) [][]string {
	t.Helper()
	root := readAlphaTabScore(t, writeConformanceFixture(t, data)).(map[string]any)
	masterBars := root["masterBars"].([]any)
	result := make([][]string, len(masterBars))
	for index, raw := range masterBars {
		values := raw.(map[string]any)["directions"].([]any)
		result[index] = make([]string, len(values))
		for valueIndex := range values {
			result[index][valueIndex] = values[valueIndex].(string)
		}
	}
	return result
}
