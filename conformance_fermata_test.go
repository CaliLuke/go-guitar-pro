// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"math"
	"slices"
	"strings"
	"testing"
)

func TestConformanceFermatas(t *testing.T) {
	runConformanceFermatas(newConformanceRun(t))
}

func runConformanceFermatas(run *conformanceRun) {
	t := run.t
	fixture := parseTestFixture(t, "testdata/gp7/fermata.gp")
	if len(fixture.MeasureHeaders) < 4 {
		t.Fatalf("fermata fixture headers = %d", len(fixture.MeasureHeaders))
	}
	run.ClaimPrimary(claimSite("fermata", "import", "M07-FERMATAS", "exact sub-tick offset")).Preserved("MeasureHeader.Fermatas", conformanceFermataValues(fixture.MeasureHeaders[0].Fermatas), []any{
		[]any{int64(0), int64(1), FermataTypeShort, float64(0)},
		[]any{int64(480), int64(1), FermataTypeShort, float64(0)},
		[]any{int64(960), int64(1), FermataTypeShort, 0.26},
		[]any{int64(1920), int64(1), FermataTypeShort, 0.5},
		[]any{int64(2880), int64(1), FermataTypeShort, float64(1)},
	})
	run.Preserved("Fermata.Offset", fixture.MeasureHeaders[3].Fermatas[1].Offset, mustScoreTime(t, 120, 1))
	run.Preserved("Fermata.Type", []FermataType{
		fixture.MeasureHeaders[0].Fermatas[0].Type,
		fixture.MeasureHeaders[1].Fermatas[0].Type,
		fixture.MeasureHeaders[2].Fermatas[0].Type,
	}, []FermataType{FermataTypeShort, FermataTypeMedium, FermataTypeLong})
	run.Preserved("Fermata.Length", []float64{
		fixture.MeasureHeaders[0].Fermatas[2].Length,
		fixture.MeasureHeaders[1].Fermatas[1].Length,
		fixture.MeasureHeaders[2].Fermatas[3].Length,
	}, []float64{0.26, 0.5, 0.21})
	run.Enum("FermataType.FermataTypeShort", fixture.MeasureHeaders[0].Fermatas[0].Type, FermataTypeShort)
	run.Enum("FermataType.FermataTypeMedium", fixture.MeasureHeaders[1].Fermatas[0].Type, FermataTypeMedium)
	run.Enum("FermataType.FermataTypeLong", fixture.MeasureHeaders[2].Fermatas[0].Type, FermataTypeLong)

	slash := parseTestFixture(t, "testdata/gp7/slash.gp")
	if got := slash.MeasureHeaders[1].Fermatas; len(got) != 1 || got[0].Offset.Compare(mustScoreTime(t, 2880, 1)) != 0 || got[0].Type != FermataTypeMedium || got[0].Length != 0.5 {
		t.Fatalf("slash fermata = %#v", got)
	}

	// Reuse the same bar, voice, beat, and chord definitions from two master
	// bars. The importer must still create independent measure occurrences and
	// independent authoritative fermata records for each master-bar occurrence.
	reusedGPIF := strings.Replace(staffScopedChordGPIF,
		`<MasterBars><MasterBar><Time>4/4</Time><Bars>0 1</Bars></MasterBar></MasterBars>`,
		`<MasterBars>
  <MasterBar><Time>4/4</Time><Fermatas><Fermata><Type>Short</Type><Offset>0/1</Offset><Length>0.25</Length></Fermata></Fermatas><Bars>0 1</Bars></MasterBar>
  <MasterBar><Time>4/4</Time><Fermatas><Fermata><Type>Long</Type><Offset>1/1</Offset><Length>1.5</Length></Fermata></Fermatas><Bars>0 1</Bars></MasterBar>
</MasterBars>`, 1)
	reusedResult, reusedErr := ParseWithOptions(conformanceGPIFArchive(t, reusedGPIF), ParseOptions{Strict: true})
	if reusedErr != nil {
		t.Fatal(reusedErr)
	}
	reusedHeaders := reusedResult.Song.MeasureHeaders
	reusedMeasures := reusedResult.Song.Tracks[0].Staves[0].Measures
	if len(reusedHeaders) != 2 || len(reusedMeasures) != 2 {
		t.Fatalf("reused GPIF occurrences = %d headers, %d measures", len(reusedHeaders), len(reusedMeasures))
	}
	if len(reusedHeaders[0].Fermatas) != 1 || len(reusedHeaders[1].Fermatas) != 1 {
		t.Fatalf("reused GPIF fermatas = %d/%d", len(reusedHeaders[0].Fermatas), len(reusedHeaders[1].Fermatas))
	}
	reusedHeaders[0].Fermatas[0].Length = 9
	reusedMeasures[0].Voices[0].Beats[0].Status = BeatStatusNormal
	if reusedHeaders[1].Fermatas[0].Length != 1.5 || reusedHeaders[1].Fermatas[0].Type != FermataTypeLong || reusedMeasures[1].Voices[0].Beats[0].Status != BeatStatusRest {
		t.Fatalf("second reused occurrence changed through first: fermata=%#v beat=%#v", reusedHeaders[1].Fermatas[0], reusedMeasures[1].Voices[0].Beats[0])
	}

	song := semanticValidPitchedGP8Song(t)
	song.Tracks[0].Settings.Notation = true
	song.MeasureHeaders[0].Fermatas = []Fermata{
		{Offset: mustScoreTime(t, 0, 1), Type: FermataTypeShort, Length: 0.25},
		{Offset: mustScoreTime(t, 480, 1), Type: FermataTypeMedium, Length: 0.5},
		{Offset: mustScoreTime(t, 1920, 1), Type: FermataTypeLong, Length: 1.25},
	}
	// A sub-tick offset stays exact instead of being truncated during GPIF
	// conversion. It is not used for AlphaTab's integer beat association.
	song.MeasureHeaders[1].Fermatas = []Fermata{{Offset: mustScoreTime(t, 1, 2), Type: FermataTypeMedium, Length: 0}}
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("fermata export = %v, %#v", err, report.Entries)
	}
	wire := conformanceFermataWire(t, data)
	run.Wire("gpifMasterBar.Fermatas", len(wire[0]), 3)
	run.Wire("gpifFermatas.Fermatas", len(wire[1]), 1)
	run.Wire("gpifFermata.Type", []string{wire[0][0].Type, wire[0][1].Type, wire[0][2].Type}, []string{"Short", "Medium", "Long"})
	run.Wire("gpifFermata.Offset", []string{wire[0][0].Offset, wire[0][1].Offset, wire[0][2].Offset, wire[1][0].Offset}, []string{"0/1", "1/2", "2/1", "1/1920"})
	run.Wire("gpifFermata.Length", []string{wire[0][0].Length, wire[0][1].Length, wire[0][2].Length, wire[1][0].Length}, []string{"0.25", "0.5", "1.25", "0"})

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.ClaimPrimary(claimSite("fermata", "model", "M07-FERMATAS", "exact sub-tick offset")).Preserved("MeasureHeader.Fermatas", conformanceFermataValues(roundTrip.MeasureHeaders[0].Fermatas), conformanceFermataValues(song.MeasureHeaders[0].Fermatas))
	if got := roundTrip.MeasureHeaders[1].Fermatas[0].Offset; got.Compare(mustScoreTime(t, 1, 2)) != 0 {
		t.Fatalf("sub-tick round-trip offset = %d/%d", got.Numerator(), got.Denominator())
	}

	// Post-parse edits to the authoritative master-bar records survive exactly.
	roundTrip.MeasureHeaders[0].Fermatas[1] = Fermata{Offset: mustScoreTime(t, 1440, 1), Type: FermataTypeLong, Length: 0.875}
	edited, err := Export(roundTrip, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	editedRoundTrip, err := Parse(edited)
	if err != nil {
		t.Fatal(err)
	}
	if got := editedRoundTrip.MeasureHeaders[0].Fermatas[1]; got.Offset.Compare(mustScoreTime(t, 1440, 1)) != 0 || got.Type != FermataTypeLong || got.Length != 0.875 {
		t.Fatalf("edited fermata = %#v", got)
	}

	invalidSource := rewriteConformanceGPIF(t, data, func(gpif string) string {
		gpif = strings.Replace(gpif, "<Type>Short</Type>", "<Type>Breath</Type>", 1)
		gpif = strings.Replace(gpif, "<Length>0.5</Length>", "<Length>NaN</Length>", 1)
		gpif = strings.Replace(gpif, "<Offset>1/2</Offset>", "<Offset>9223372036854775807/1</Offset>", 1)
		gpif = strings.Replace(gpif, "<Offset>2/1</Offset>", "<Offset>1/0</Offset>", 1)
		return gpif
	})
	result, strictErr := ParseWithOptions(invalidSource, ParseOptions{Strict: true})
	var parseErr *StrictParseError
	if !errors.As(strictErr, &parseErr) || result == nil {
		t.Fatalf("invalid fermata strict parse = %#v, %v", result, strictErr)
	}
	for _, code := range []string{
		"GPIF.MasterBar.Fermata.Type.InvalidValue",
		"GPIF.MasterBar.Fermata.Length.InvalidValue",
		"GPIF.MasterBar.Fermata.Offset.InvalidValue",
	} {
		diagnostic := conformanceSourceAuditDiagnosticByCode(result.Diagnostics, code)
		if diagnostic == nil || !strings.Contains(diagnostic.SourcePath, "/Fermatas/Fermata[") {
			t.Errorf("%s diagnostic = %#v", code, diagnostic)
		}
	}
	invalidOffsets := 0
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "GPIF.MasterBar.Fermata.Offset.InvalidValue" {
			invalidOffsets++
		}
	}
	if invalidOffsets < 2 {
		t.Errorf("invalid rational and overflow offset diagnostics = %d, want at least 2", invalidOffsets)
	}

	duplicateSource := rewriteConformanceGPIF(t, data, func(gpif string) string {
		return strings.Replace(gpif, "<Offset>1/2</Offset>", "<Offset>0/1</Offset>", 1)
	})
	duplicateResult, _ := ParseWithOptions(duplicateSource, ParseOptions{})
	if diagnostic := conformanceSourceAuditDiagnosticByCode(duplicateResult.Diagnostics, "GPIF.MasterBar.Fermata.Offset.Duplicate"); diagnostic == nil {
		t.Fatalf("duplicate source diagnostics = %#v", duplicateResult.Diagnostics)
	}

	invalid := semanticValidPitchedGP8Song(t)
	invalid.MeasureHeaders[0].Fermatas = []Fermata{
		{Offset: mustScoreTime(t, 3840, 1), Type: FermataType(99), Length: math.Inf(1)},
		{Offset: mustScoreTime(t, 3840, 1), Type: FermataTypeShort, Length: -1},
	}
	diagnostics := ValidateSong(invalid)
	for _, code := range []string{"score.measure.fermata-type", "score.measure.fermata-length", "score.measure.fermata-offset"} {
		if !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool {
			return diagnostic.Code == code && diagnostic.Location.Measure == 0
		}) {
			t.Errorf("missing %s in %#v", code, diagnostics)
		}
	}
	invalidData, invalidReport, invalidErr := ExportWithReport(invalid, ExportFormatGP8, ExportOptions{})
	if len(invalidData) != 0 || invalidErr == nil || !hasExportCode(invalidReport, "gp8.reject.score.measure.fermata-type") || !hasExportCode(invalidReport, "gp8.reject.score.measure.fermata-length") || !hasExportCode(invalidReport, "gp8.reject.score.measure.fermata-offset") {
		t.Fatalf("invalid fermata export = %d bytes, %#v, %v", len(invalidData), invalidReport.Entries, invalidErr)
	}
}

func TestAlphaTabPreservesFermatas(t *testing.T) {
	requireAlphaTabConformance(t)
	source := parseTestFixture(t, "testdata/gp7/fermata.gp")
	data, err := Export(source, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var facts []conformanceAlphaTabFermataBar
	readAlphaTabOracleFacts(t, "--fermatas", writeConformanceFixture(t, data), &facts)
	if len(facts) < 4 {
		t.Fatalf("AlphaTab fermata bars = %d", len(facts))
	}
	for index, wantType := range []string{"short", "medium", "long"} {
		if len(facts[index].Authored) != 5 {
			t.Fatalf("AlphaTab bar %d authored fermatas = %#v", index, facts[index].Authored)
		}
		for _, fermata := range facts[index].Authored {
			if fermata.Type != wantType {
				t.Fatalf("AlphaTab bar %d fermata = %#v", index, fermata)
			}
		}
		if len(facts[index].DerivedBeats) == 0 {
			t.Fatalf("AlphaTab bar %d has no derived beat fermatas", index)
		}
	}
	wantOffsets := []float64{0, 480, 960, 1920, 2880}
	gotOffsets := make([]float64, len(facts[0].Authored))
	for index := range facts[0].Authored {
		gotOffsets[index] = facts[0].Authored[index].Offset
	}
	if !slices.Equal(gotOffsets, wantOffsets) {
		t.Fatalf("AlphaTab authored offsets = %v, want %v", gotOffsets, wantOffsets)
	}
	for _, want := range []conformanceAlphaTabDerivedFermata{
		{Track: 0, Staff: 0, Voice: 0, Beat: 1, Offset: 960, Type: "short", Length: 0.26},
		{Track: 0, Staff: 0, Voice: 0, Beat: 2, Offset: 1920, Type: "short", Length: 0.5},
	} {
		if !slices.Contains(facts[0].DerivedBeats, want) {
			t.Errorf("AlphaTab derived fermatas = %#v, want exact %#v", facts[0].DerivedBeats, want)
		}
	}

	// Edit the public master-bar record, then verify the exact consumer
	// location and value without creating a mutable Beat fermata in Go.
	source.MeasureHeaders[0].Fermatas[2] = Fermata{Offset: mustScoreTime(t, 960, 1), Type: FermataTypeLong, Length: 0.75}
	edited, err := Export(source, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	readAlphaTabOracleFacts(t, "--fermatas", writeConformanceFixture(t, edited), &facts)
	if got := facts[0].Authored[2]; got.Offset != 960 || got.Type != "long" || got.Length != 0.75 {
		t.Fatalf("AlphaTab edited fermata = %#v", got)
	}
	wantEditedDerived := conformanceAlphaTabDerivedFermata{Track: 0, Staff: 0, Voice: 0, Beat: 1, Offset: 960, Type: "long", Length: 0.75}
	if !slices.Contains(facts[0].DerivedBeats, wantEditedDerived) {
		t.Fatalf("AlphaTab edited derived fermatas = %#v, want exact %#v", facts[0].DerivedBeats, wantEditedDerived)
	}
	conformanceIndependentClaim(t, "field:MeasureHeader.Fermatas", claimSite("fermata", "import", "M07-FERMATAS", "exact sub-tick offset"), claimSite("fermata", "model", "M07-FERMATAS", "exact sub-tick offset"))
}

func conformanceFermataValues(values []Fermata) []any {
	result := make([]any, len(values))
	for index, value := range values {
		result[index] = []any{value.Offset.Numerator(), value.Offset.Denominator(), value.Type, value.Length}
	}
	return result
}

func conformanceFermataWire(t *testing.T, data []byte) [][]gpifFermata {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	if err := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document); err != nil {
		t.Fatal(err)
	}
	result := make([][]gpifFermata, len(document.MasterBars.MasterBars))
	for index, masterBar := range document.MasterBars.MasterBars {
		if masterBar.Fermatas != nil {
			result[index] = masterBar.Fermatas.Fermatas
		}
	}
	return result
}

type conformanceAlphaTabFermata struct {
	Offset float64 `json:"offset"`
	Type   string  `json:"type"`
	Length float64 `json:"length"`
}

type conformanceAlphaTabFermataBar struct {
	Authored     []conformanceAlphaTabFermata        `json:"authored"`
	DerivedBeats []conformanceAlphaTabDerivedFermata `json:"derivedBeats"`
}

type conformanceAlphaTabDerivedFermata struct {
	Track  int     `json:"track"`
	Staff  int     `json:"staff"`
	Voice  int     `json:"voice"`
	Beat   int     `json:"beat"`
	Offset float64 `json:"offset"`
	Type   string  `json:"type"`
	Length float64 `json:"length"`
}
