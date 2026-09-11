// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const conformanceBrushCase = "M10-BRUSH"

func TestConformanceBrush(t *testing.T) {
	runConformanceBrush(newConformanceRun(t))
}

func runConformanceBrush(run *conformanceRun) {
	t := run.t
	colors := parseTestFixture(t, "testdata/gp7/colors.gp")
	stroke := &colors.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Stroke
	run.Preserved("BeatStroke.Kind", stroke.Kind, BeatStrokeKindArpeggio)
	run.Preserved("BeatStroke.Direction", stroke.Direction, BeatStrokeDirectionDown)
	run.Preserved("BeatStroke.Duration", stroke.Duration, NoteValue(DurationEighth))
	if stroke.ExactDuration == nil {
		t.Fatal("colors brush exact duration is nil")
	}
	run.ClaimPrimary(claimImportModel("brush", conformanceBrushCase, "arpeggio down with 117 exact ticks")...).Preserved("BeatStroke.ExactDuration", *stroke.ExactDuration, mustScoreTime(t, 117, 1))
	run.Dispatch("gpifApplyBeatEffects:b.Arpeggio", stroke.Kind, BeatStrokeKindArpeggio)
	brushDispatch := Beat{}
	direction := "Up"
	gpifApplyBeatEffects(&gpifBeat{Properties: gpifProperties{Properties: []gpifProperty{{Name: "Brush", Direction: &direction}}}}, &brushDispatch)
	run.Dispatch("gpifAuditBrush:property.Name", brushDispatch.Effect.Stroke.Kind, BeatStrokeKindBrush)
	run.Dispatch("gpifApplyBeatEffects:p.Direction", brushDispatch.Effect.Stroke.Direction, BeatStrokeDirectionUp)
	for code, want := range []int64{0, 30, 30, 60, 120, 240, 480} {
		if code == 0 {
			continue
		}
		got := strokeScoreTime(int8(code))
		if got == nil || got.Numerator() != want || got.Denominator() != 1 {
			t.Fatalf("binary stroke code %d = %#v, want %d exact ticks", code, got, want)
		}
	}

	for _, fixture := range []string{"testdata/gp3/strokes.gp3", "testdata/gp4/Strokes.gp4", "testdata/gp5/Strokes.gp5", "testdata/gp6/strokes.gpx"} {
		song := parseTestFixture(t, fixture)
		strokes := conformanceBrushStrokes(song)
		if len(strokes) == 0 {
			t.Fatalf("%s has no imported strokes", fixture)
		}
		for _, imported := range strokes {
			if imported.Kind != BeatStrokeKindBrush || imported.ExactDuration == nil {
				t.Fatalf("%s stroke = %#v, want exact brush", fixture, imported)
			}
		}
	}
	absenceData, err := os.ReadFile("testdata/gp7/brush.gp")
	if err != nil {
		t.Fatal(err)
	}
	absenceArchive, err := zip.NewReader(bytes.NewReader(absenceData), int64(len(absenceData)))
	if err != nil {
		t.Fatal(err)
	}
	absenceSource := string(readZipMember(t, absenceArchive, "Content/score.gpif"))
	absenceSource = strings.ReplaceAll(absenceSource, "<XProperties>\n<XProperty id=\"687935489\">\n<Int>60</Int>\n</XProperty>\n</XProperties>\n", "")
	absentSong, err := Parse(conformanceGPIFArchive(t, absenceSource))
	if err != nil {
		t.Fatal(err)
	}
	if absentSong.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Stroke.ExactDuration != nil {
		t.Fatal("absent brush timing became present")
	}
	absentOutput, err := Export(absentSong, ExportFormatGP8)
	if err != nil || strings.Contains(conformanceBrushGPIF(t, absentOutput), gpifBeatBrushDurationID) {
		t.Fatalf("absent brush timing output = %v", err)
	}

	brushSong := parseTestFixture(t, "testdata/gp7/brush.gp")
	brushOutput, err := Export(brushSong, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	brushWire := conformanceBrushGPIF(t, brushOutput)
	if !strings.Contains(brushWire, `name="Brush"`) || strings.Contains(brushWire, "<Arpeggio>") {
		t.Fatal("GPIF Brush property was not kept distinct from Arpeggio")
	}

	sharedDefinition := gpifBeat{Arpeggio: "Down", XProperties: &gpifXProperties{Properties: []gpifXProperty{gp8IntXProperty(gpifBeatBrushDurationID, 117)}}}
	firstOccurrence, secondOccurrence := Beat{}, Beat{}
	gpifApplyBeatEffects(&sharedDefinition, &firstOccurrence)
	gpifApplyBeatEffects(&sharedDefinition, &secondOccurrence)
	if firstOccurrence.Effect.Stroke.ExactDuration == secondOccurrence.Effect.Stroke.ExactDuration {
		t.Fatal("reused beat definition shares exact stroke timing")
	}
	firstEdited := mustScoreTime(t, 240, 1)
	firstOccurrence.Effect.Stroke.ExactDuration = &firstEdited
	if secondOccurrence.Effect.Stroke.ExactDuration.Numerator() != 117 {
		t.Fatal("editing one stroke occurrence changed another")
	}

	for index, kind := range []BeatStrokeKind{BeatStrokeKindNone, BeatStrokeKindBrush, BeatStrokeKindArpeggio} {
		run.Enum([]string{"BeatStrokeKind.BeatStrokeKindNone", "BeatStrokeKind.BeatStrokeKindBrush", "BeatStrokeKind.BeatStrokeKindArpeggio"}[index], kind, BeatStrokeKind(index))
	}

	before := conformanceContractSnapshot(colors, false)
	report := PreflightExport(colors, ExportFormatGP8, ExportOptions{})
	run.ClaimReport(claimSite("brush", "export", conformanceBrushCase, "arpeggio down with 117 exact ticks")).Report(conformanceBrushCase, reportCodes(report), []string{"gp8.omit.pan-automation-consumer", "gp8.omit.volume-automation-consumer", "gp8.normalize.source-version", "gp8.omit.track-use-rse", "gp8.omit.track-display-settings"})
	data, _, err := ExportWithReport(colors, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: reportCodes(report)}})
	if err != nil {
		t.Fatal(err)
	}
	if got := conformanceContractSnapshot(colors, false); got != before {
		t.Fatal("brush preflight/export mutated the score")
	}
	wire := conformanceBrushGPIF(t, data)
	run.Wire("gpifBeat.Arpeggio", strings.Contains(wire, `<Arpeggio>Down</Arpeggio>`), true)
	run.ClaimSerialization(claimSite("brush", "export", conformanceBrushCase, "arpeggio down with 117 exact ticks")).Wire("gpifBeat.XProperties", strings.Contains(wire, `id="687935489"`) && strings.Contains(wire, `<Int>117</Int>`), true)
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	roundStroke := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Stroke
	run.Field("BeatStroke.Kind", roundStroke.Kind, BeatStrokeKindArpeggio)
	run.Field("BeatStroke.Direction", roundStroke.Direction, BeatStrokeDirectionDown)
	if roundStroke.ExactDuration == nil || roundStroke.ExactDuration.Numerator() != 117 {
		t.Fatalf("round-trip exact duration = %#v, want 117", roundStroke.ExactDuration)
	}
	run.ClaimPrimary(claimSite("brush", "export", conformanceBrushCase, "arpeggio down with 117 exact ticks")).Field("BeatStroke.ExactDuration", *roundStroke.ExactDuration, mustScoreTime(t, 117, 1))

	edited := parseTestFixture(t, "testdata/gp7/colors.gp")
	editedStroke := &edited.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Stroke
	editedStroke.Duration = NoteValue(DurationSixteenth)
	editedReport := PreflightExport(edited, ExportFormatGP8, ExportOptions{})
	if !hasExportCode(editedReport, "gp8.normalize.stroke-duration-authority") {
		t.Fatalf("edited duration report = %#v", editedReport.Entries)
	}
	editedData, _, err := ExportWithReport(edited, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: reportCodes(editedReport)}})
	if err != nil || !strings.Contains(conformanceBrushGPIF(t, editedData), `<Int>240</Int>`) {
		t.Fatalf("edited legacy duration export = %v", err)
	}
	editedStroke.ExactDuration = nil
	clearedReport := PreflightExport(edited, ExportFormatGP8, ExportOptions{})
	clearedData, _, err := ExportWithReport(edited, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: reportCodes(clearedReport)}})
	if err != nil || !strings.Contains(conformanceBrushGPIF(t, clearedData), `<Int>240</Int>`) {
		t.Fatalf("cleared exact duration export = %v", err)
	}

	legacy := semanticExportProbeSong(t)
	legacyStroke := &legacy.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Stroke
	legacyStroke.Kind = BeatStrokeKindNone
	legacyStroke.Direction = BeatStrokeDirectionDown
	legacyStroke.Duration = NoteValue(DurationThirtySecond)
	legacyData, err := Export(legacy, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	legacyWire := conformanceBrushGPIF(t, legacyData)
	if !strings.Contains(legacyWire, "<Arpeggio>Down</Arpeggio>") || !strings.Contains(legacyWire, `<Int>120</Int>`) {
		t.Fatalf("legacy stroke wire does not preserve arpeggio compatibility: %s", legacyWire)
	}

	fractional := semanticExportProbeSong(t)
	fractionalStroke := &fractional.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Stroke
	fractionalStroke.Kind = BeatStrokeKindBrush
	fractionalStroke.Direction = BeatStrokeDirectionUp
	fractionalStroke.Duration = NoteValue(DurationEighth)
	exact := mustScoreTime(t, 235, 2)
	fractionalStroke.ExactDuration = &exact
	if fractionalReport := PreflightExport(fractional, ExportFormatGP8, ExportOptions{}); !hasExportCode(fractionalReport, "gp8.omit.stroke-exact-duration") {
		t.Fatalf("fractional report = %#v", fractionalReport.Entries)
	}
	tooLarge := mustScoreTime(t, math.MaxInt32+1, 1)
	fractionalStroke.ExactDuration = &tooLarge
	if tooLargeReport := PreflightExport(fractional, ExportFormatGP8, ExportOptions{}); !hasExportCode(tooLargeReport, "gp8.omit.stroke-exact-duration") {
		t.Fatalf("too-large report = %#v", tooLargeReport.Entries)
	}

	for _, edit := range []func(*BeatStroke){
		func(value *BeatStroke) { value.Kind = BeatStrokeKind(9) },
		func(value *BeatStroke) { value.Direction = BeatStrokeDirection(9) },
		func(value *BeatStroke) {
			value.Kind, value.Direction, value.Duration = BeatStrokeKindBrush, BeatStrokeDirectionNone, NoteValue(DurationEighth)
		},
		func(value *BeatStroke) {
			value.Kind, value.Direction, value.Duration = BeatStrokeKindBrush, BeatStrokeDirectionUp, 3
		},
	} {
		invalid := semanticExportProbeSong(t)
		edit(&invalid.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Stroke)
		if len(ValidateSong(invalid)) == 0 {
			t.Fatal("invalid stroke passed validation")
		}
	}

	for _, boundary := range []int64{0, math.MaxInt32} {
		value := mustScoreTime(t, boundary, 1)
		if ticks, ok := gp8StrokeTicks(value); !ok || ticks != boundary {
			t.Fatalf("boundary %d = %d, %t", boundary, ticks, ok)
		}
		boundarySong := semanticExportProbeSong(t)
		boundaryStroke := &boundarySong.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Stroke
		boundaryStroke.Kind, boundaryStroke.Direction, boundaryStroke.Duration, boundaryStroke.ExactDuration = BeatStrokeKindBrush, BeatStrokeDirectionUp, NoteValue(DurationEighth), &value
		boundaryData, boundaryErr := Export(boundarySong, ExportFormatGP8)
		if boundaryErr != nil || !strings.Contains(conformanceBrushGPIF(t, boundaryData), `<Int>`+fmt.Sprint(boundary)+`</Int>`) {
			t.Fatalf("boundary %d export = %v", boundary, boundaryErr)
		}
	}
}

func TestGPIFBrushDiagnostics(t *testing.T) {
	sourceData, err := os.ReadFile("testdata/gp7/brush.gp")
	if err != nil {
		t.Fatal(err)
	}
	archive, err := zip.NewReader(bytes.NewReader(sourceData), int64(len(sourceData)))
	if err != nil {
		t.Fatal(err)
	}
	source := string(readZipMember(t, archive, "Content/score.gpif"))
	tests := []struct{ name, old, replacement, code string }{
		{"invalid duration", "<Int>60</Int>", "<Int>2147483648</Int>", "GPIF.Beat.Brush.Duration.Invalid"},
		{"duplicate duration", "<XProperty id=\"687935489\">\n<Int>60</Int>\n</XProperty>", "<XProperty id=\"687935489\"><Int>60</Int></XProperty><XProperty id=\"687935489\"><Int>60</Int></XProperty>", "GPIF.Beat.Brush.Duration.Duplicate"},
		{"conflicting kind", `<Beat id="0">`, `<Beat id="0"><Arpeggio>Up</Arpeggio>`, "GPIF.Beat.Brush.Kind.Conflict"},
		{"orphan duration", "<Property name=\"Brush\">\n<Direction>Down</Direction>\n</Property>", ``, "GPIF.Beat.Brush.Duration.Orphan"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mutated := strings.Replace(source, test.old, test.replacement, 1)
			result, parseErr := ParseWithOptions(conformanceGPIFArchive(t, mutated), ParseOptions{})
			if parseErr != nil || !slices.ContainsFunc(result.Diagnostics, func(d ParseDiagnostic) bool { return d.Code == test.code }) {
				t.Fatalf("diagnostics = %#v, %v; want %s", result.Diagnostics, parseErr, test.code)
			}
			conformanceBrushRequireStrictInvalid(t, mutated)
		})
	}
}

func conformanceBrushRequireStrictInvalid(t *testing.T, source string) {
	t.Helper()
	_, err := ParseWithOptions(conformanceGPIFArchive(t, source), ParseOptions{Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticInvalidData}})
	var rejected *StrictParseError
	if !errors.As(err, &rejected) {
		t.Fatalf("strict brush parse = %v, want StrictParseError", err)
	}
}

type conformanceBrushFact struct {
	Track, Staff, Bar, Voice, Beat int
	Kind                           string
	Direction                      string
	Duration                       int64
}

func TestAlphaTabPreservesBrush(t *testing.T) {
	requireAlphaTabConformance(t)
	fixture := "testdata/gp7/colors.gp"
	var source []conformanceBrushFact
	readAlphaTabOracleFacts(t, "--brush", fixture, &source)
	if len(source) != 1 || source[0].Kind != "arpeggio" || source[0].Direction != "down" || source[0].Duration != 117 {
		t.Fatalf("AlphaTab source brush facts = %#v", source)
	}
	data, err := Export(parseTestFixture(t, fixture), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var output []conformanceBrushFact
	readAlphaTabOracleFacts(t, "--brush", writeConformanceFixture(t, data), &output)
	if !reflect.DeepEqual(output, source) {
		t.Fatalf("AlphaTab output brush facts = %#v, want %#v", output, source)
	}
	conformanceIndependentClaim(t, "field:BeatStroke.ExactDuration", claimAllStages("brush", conformanceBrushCase, "arpeggio down with 117 exact ticks")...)
}

func conformanceBrushStrokes(song *Song) []BeatStroke {
	var result []BeatStroke
	for trackIndex := range song.Tracks {
		for _, staff := range song.Tracks[trackIndex].Staves {
			for _, measure := range staff.Measures {
				for _, voice := range measure.Voices {
					for _, beat := range voice.Beats {
						if beat.Effect.Stroke.Direction != BeatStrokeDirectionNone {
							result = append(result, beat.Effect.Stroke)
						}
					}
				}
			}
		}
	}
	return result
}

func conformanceBrushGPIF(t *testing.T, data []byte) string {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	return string(readZipMember(t, archive, "Content/score.gpif"))
}
