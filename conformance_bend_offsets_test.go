// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"bytes"
	"encoding/binary"
	"math"
	"slices"
	"testing"
)

func TestConformanceExactBendOffsets(t *testing.T) {
	runConformanceExactBendOffsets(newConformanceRun(t))
}

func runConformanceExactBendOffsets(run *conformanceRun) {
	t := run.t
	base, err := Export(conformanceCurveSong(t), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	data := rewriteConformanceGPIF(t, base, func(gpif string) string {
		return insertFirstNoteProperty(t, gpif,
			`<Property name="Bended"><Enable/></Property>`+
				`<Property name="BendOriginOffset"><Float>0</Float></Property>`+
				`<Property name="BendOriginValue"><Float>0</Float></Property>`+
				`<Property name="BendMiddleOffset1"><Float>35</Float></Property>`+
				`<Property name="BendMiddleOffset2"><Float>35.5</Float></Property>`+
				`<Property name="BendMiddleValue"><Float>50</Float></Property>`+
				`<Property name="BendDestinationOffset"><Float>100</Float></Property>`+
				`<Property name="BendDestinationValue"><Float>0</Float></Property>`)
	})
	result, err := ParseWithOptions(data, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if slices.ContainsFunc(result.Diagnostics, func(diagnostic ParseDiagnostic) bool {
		return diagnostic.Code == "GPIF.Note.Property.BendNumber.Quantized"
	}) {
		t.Fatalf("exact offsets received a quantization diagnostic: %#v", result.Diagnostics)
	}
	bend := firstConformanceBend(t, result.Song)
	if len(bend.Points) != 4 {
		t.Fatalf("bend points = %#v, want four controls", bend.Points)
	}
	run.Preserved("BendPoint.ExactOffset", dereferenceBendOffset(bend.Points[1].ExactOffset), 35.0)
	run.Field("BendPoint.Position", []uint8{bend.Points[0].Position, bend.Points[1].Position, bend.Points[2].Position, bend.Points[3].Position}, []uint8{0, 4, 4, 12})
	if got := dereferenceBendOffset(bend.Points[2].ExactOffset); got != 35.5 {
		t.Fatalf("second middle exact offset = %v, want 35.5", got)
	}
	edited35 := 35.0
	edited355 := 35.5
	bend.Points[1].ExactOffset = &edited35
	bend.Points[2].ExactOffset = &edited355

	exported, report, err := ExportWithReport(result.Song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{
		RequirePreservation: true,
		AllowedCodes:        []string{"gp8.normalize.source-version", "gp8.omit.track-display-settings", "gp8.omit.bend-summary"},
	}})
	if err != nil {
		t.Fatalf("export exact offsets: %#v, %v", report.Entries, err)
	}
	if hasExportCode(report, "gp8.normalize.bend-curve") {
		t.Fatalf("exact controls were normalized: %#v", report.Entries)
	}
	wire := readCurveWire(t, exported)
	run.Wire("gpifProperty.Float", []string{
		wire.bend["BendOriginOffset"], wire.bend["BendMiddleOffset1"], wire.bend["BendMiddleOffset2"], wire.bend["BendDestinationOffset"],
	}, []string{"0", "35", "35.5", "100"})
	roundTrip, err := Parse(exported)
	if err != nil {
		t.Fatal(err)
	}
	roundTripBend := firstConformanceBend(t, roundTrip)
	if got := []float64{resolvedBendOffset(roundTripBend.Points[1]), resolvedBendOffset(roundTripBend.Points[2])}; !slices.Equal(got, []float64{35, 35.5}) {
		t.Fatalf("round-trip exact offsets = %#v, want [35 35.5]", got)
	}

	parsedPoint := bend.Points[1]
	editedExact := 36.25
	parsedPoint.ExactOffset = &editedExact
	if got := resolvedBendOffset(parsedPoint); got != 36.25 {
		t.Fatalf("edited exact offset resolves to %v, want 36.25", got)
	}
	parsedPoint.Position = 6
	if got := resolvedBendOffset(parsedPoint); got != 50 {
		t.Fatalf("edited legacy position resolves to %v, want 50", got)
	}
	parsedPoint.ExactOffset = nil
	if got := resolvedBendOffset(parsedPoint); got != 50 {
		t.Fatalf("cleared exact offset resolves to %v, want 50", got)
	}
	programmaticExact := 35.5
	programmatic := BendPoint{Position: 4, ExactOffset: &programmaticExact}
	if got := resolvedBendOffset(programmatic); got != 35.5 {
		t.Fatalf("programmatic exact offset resolves to %v, want 35.5", got)
	}
	duplicateOffset := 35.5
	if got := simplifyNoteBendPoints([]BendPoint{programmatic, {Position: 4, ExactOffset: &duplicateOffset}}); len(got) != 1 {
		t.Fatalf("semantic duplicate exact offsets = %#v, want one point", got)
	}
	collinearMiddle := 35.5
	collinearEnd := 71.0
	if got := simplifyNoteBendPoints([]BendPoint{{}, {Position: 4, ExactOffset: &collinearMiddle, Value: 1}, {Position: 9, ExactOffset: &collinearEnd, Value: 2}}); len(got) != 2 {
		t.Fatalf("exact-offset collinear points = %#v, want endpoints", got)
	}
	isolatedOffset := 35.5
	expanded := canonicalizeStandardBendPoints([]BendPoint{{}, {Position: 4, ExactOffset: &isolatedOffset, Value: 2}, {Position: 12}})
	if len(expanded) != 4 || expanded[1].ExactOffset == expanded[2].ExactOffset {
		t.Fatalf("expanded bend offsets are not independent: %#v", expanded)
	}
	*expanded[1].ExactOffset = 36
	if got := *expanded[2].ExactOffset; got != 35.5 {
		t.Fatalf("expanded bend offset shared mutation: got %v, want 35.5", got)
	}
	if isolatedOffset != 35.5 {
		t.Fatalf("expanded bend offset mutated its source: got %v, want 35.5", isolatedOffset)
	}

	preciseOffset := 35.123456789
	preciseSong := conformanceCurveSong(t)
	firstConformanceBend(t, preciseSong).Points = []BendPoint{
		{},
		{Position: 4, ExactOffset: &preciseOffset, Value: 2},
		{Position: 9, Value: 2},
		{Position: 12},
	}
	preciseData, preciseReport, preciseErr := ExportWithReport(preciseSong, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if preciseErr != nil {
		t.Fatalf("strict precise export: %#v, %v", preciseReport.Entries, preciseErr)
	}
	preciseWire := readCurveWire(t, preciseData)
	if got := preciseWire.bend["BendMiddleOffset1"]; got != "35.123456789" {
		t.Fatalf("precise wire offset = %q, want 35.123456789", got)
	}
	preciseRoundTrip, preciseParseErr := Parse(preciseData)
	if preciseParseErr != nil {
		t.Fatal(preciseParseErr)
	}
	if got := resolvedBendOffset(firstConformanceBend(t, preciseRoundTrip).Points[1]); got != preciseOffset {
		t.Fatalf("precise round-trip offset = %.12g, want %.12g", got, preciseOffset)
	}

	conflicting := bend.Points[1]
	conflicting.Position = 6
	conflictSong := conformanceCurveSong(t)
	firstConformanceBend(t, conflictSong).Points = []BendPoint{{}, conflicting, {Position: 12}}
	if conflictReport := PreflightExport(conflictSong, ExportFormatGP8, ExportOptions{}); !hasExportCode(conflictReport, "gp8.normalize.bend-offset-authority") {
		t.Fatalf("legacy edit report = %#v, want bend offset authority normalization", conflictReport.Entries)
	}

	for _, invalid := range []float64{math.NaN(), math.Inf(1), -0.5, 100.5} {
		invalid := invalid
		song := conformanceCurveSong(t)
		firstConformanceBend(t, song).Points = []BendPoint{{ExactOffset: &invalid}, {Position: 12}}
		report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
		if !hasExportCode(report, "gp8.reject.score.note.bend.exact-offset") {
			t.Errorf("exact offset %v report = %#v, want rejection", invalid, report.Entries)
		}
	}

	binaryBend, err := (&Song{}).readNoteBendEffect(newCursor(binaryBendRecord(t, 0, 21)))
	if err != nil {
		t.Fatal(err)
	}
	if got := resolvedBendOffset(binaryBend.Points[1]); got != 35 {
		t.Fatalf("binary raw offset 21 resolves to %v%%, want 35%%", got)
	}
	normalized := normalizeGoBend(binaryBend).([]any)
	if got := normalized[1].(map[string]any)["position"]; got != float64(21) {
		t.Fatalf("adapter binary raw offset = %v, want 21", got)
	}
	if _, err := (&Song{}).readNoteBendEffect(newCursor(binaryBendRecord(t, 0, 61))); err == nil {
		t.Fatal("binary raw offset 61 was accepted")
	}

	exactText := "35"
	enable := ""
	definition := &gpifNote{Properties: gpifProperties{Properties: []gpifProperty{
		{Name: "Bended", Enable: &enable},
		{Name: "BendDestinationOffset", Float: &exactText},
		{Name: "BendDestinationValue", Float: bendString("50")},
	}}}
	first, firstErr := gpifNoteToNote(definition, 6, false)
	second, secondErr := gpifNoteToNote(definition, 6, false)
	if firstErr != nil || secondErr != nil {
		t.Fatalf("reused note definition: %v, %v", firstErr, secondErr)
	}
	*first.Effect.Bend.Points[1].ExactOffset = 36
	if got := *second.Effect.Bend.Points[1].ExactOffset; got != 35 {
		t.Fatalf("reused bend exact offset shared mutation: got %v, want 35", got)
	}
}

func TestAlphaTabPreservesExactBendOffsets(t *testing.T) {
	requireAlphaTabConformance(t)
	exact35 := 35.0
	exact355 := 35.5
	song := conformanceCurveSong(t)
	firstConformanceBend(t, song).Points = []BendPoint{
		{Position: 0},
		{Position: 4, ExactOffset: &exact35, Value: 2},
		{Position: 4, ExactOffset: &exact355, Value: 2},
		{Position: 12},
	}
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil {
		t.Fatalf("export exact offsets: %#v, %v", report.Entries, err)
	}
	got := conformanceCurveAlphaTabFirstBendOffsets(t, data)
	want := []float64{0, 21, 21.3, 60}
	if !slices.Equal(got, want) {
		t.Fatalf("AlphaTab bend offsets = %#v, want %#v", got, want)
	}
}

func firstConformanceBend(t *testing.T, song *Song) *BendEffect {
	t.Helper()
	for trackIndex := range song.Tracks {
		for measureIndex := range song.Tracks[trackIndex].Measures {
			for voiceIndex := range song.Tracks[trackIndex].Measures[measureIndex].Voices {
				for beatIndex := range song.Tracks[trackIndex].Measures[measureIndex].Voices[voiceIndex].Beats {
					for noteIndex := range song.Tracks[trackIndex].Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes {
						bend := song.Tracks[trackIndex].Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes[noteIndex].Effect.Bend
						if bend != nil {
							return bend
						}
					}
				}
			}
		}
	}
	bend := &BendEffect{}
	song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Bend = bend
	return bend
}

func dereferenceBendOffset(offset *float64) float64 {
	if offset == nil {
		return math.NaN()
	}
	return *offset
}

func bendString(value string) *string {
	return &value
}

func binaryBendRecord(t *testing.T, offsets ...int32) []byte {
	t.Helper()
	var raw bytes.Buffer
	raw.WriteByte(byte(BendTypeNone))
	if err := binary.Write(&raw, binary.LittleEndian, int32(0)); err != nil {
		t.Fatal(err)
	}
	if err := binary.Write(&raw, binary.LittleEndian, int32(len(offsets))); err != nil {
		t.Fatal(err)
	}
	for _, offset := range offsets {
		if err := binary.Write(&raw, binary.LittleEndian, offset); err != nil {
			t.Fatal(err)
		}
		if err := binary.Write(&raw, binary.LittleEndian, int32(0)); err != nil {
			t.Fatal(err)
		}
		raw.WriteByte(0)
	}
	return raw.Bytes()
}

func conformanceCurveAlphaTabFirstBendOffsets(t *testing.T, data []byte) []float64 {
	t.Helper()
	root := readAlphaTabScore(t, writeConformanceFixture(t, data)).(map[string]any)
	tracks := root["tracks"].([]any)
	staves := tracks[0].(map[string]any)["staves"].([]any)
	bars := staves[0].(map[string]any)["bars"].([]any)
	voices := bars[0].(map[string]any)["voices"].([]any)
	beats := voices[0].(map[string]any)["beats"].([]any)
	notes := beats[0].(map[string]any)["notes"].([]any)
	points := notes[0].(map[string]any)["effects"].(map[string]any)["bend"].([]any)
	result := make([]float64, 0, len(points))
	for _, raw := range points {
		result = append(result, raw.(map[string]any)["position"].(float64))
	}
	return result
}
