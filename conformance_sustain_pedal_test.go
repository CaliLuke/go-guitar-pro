// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
)

type conformanceSustainPedalFact struct {
	Track   int                                 `json:"track"`
	Staff   int                                 `json:"staff"`
	Bar     int                                 `json:"bar"`
	Markers []conformanceSustainPedalMarkerFact `json:"markers"`
}

type conformanceSustainPedalMarkerFact struct {
	Position float64 `json:"position"`
	Type     string  `json:"type"`
}

func TestConformanceSustainPedals(t *testing.T) {
	runConformanceSustainPedals(newConformanceRun(t))
}

func runConformanceSustainPedals(run *conformanceRun) {
	t := run.t
	song := parseTestFixture(t, "testdata/gp7/sustain.gp")
	want := conformanceSustainFixtureMarkers()
	got := conformanceSustainMeasureMarkers(song.Tracks[0].Measures)
	run.ClaimPrimary(claimSite("sustain-pedal", "import", "M17-SUSTAIN-PEDALS", "down, hold, and release"), claimSite("sustain-pedal", "model", "M17-SUSTAIN-PEDALS", "down, hold, and release"), claimSite("sustain-pedal", "export", "M17-SUSTAIN-PEDALS", "down, hold, and release")).Preserved("Measure.SustainPedals", got, want)
	run.Preserved("SustainPedalMarker.Type", conformanceSustainTypes(got), conformanceSustainTypes(want))
	run.Preserved("SustainPedalMarker.Position", conformanceSustainPositions(got), conformanceSustainPositions(want))
	run.Enum("SustainPedalType.SustainPedalTypeDown", got[0][0].Type, SustainPedalTypeDown)
	run.Enum("SustainPedalType.SustainPedalTypeHold", got[3][0].Type, SustainPedalTypeHold)
	run.Enum("SustainPedalType.SustainPedalTypeRelease", got[0][1].Type, SustainPedalTypeRelease)
	if len(song.Tracks[0].Staves) != 1 || !slices.Equal(song.Tracks[0].Staves[0].Measures[4].SustainPedals, []SustainPedalMarker{{Type: SustainPedalTypeHold}}) {
		t.Fatalf("staff-0 hold markers = %#v", song.Tracks[0].Staves)
	}

	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	wire := conformanceSustainWire(t, data)
	wantWire := conformanceSustainFixtureWire()
	run.Wire("gpifTrack.Automations", len(wire[0]), len(wantWire))
	run.ClaimSerialization(claimSite("sustain-pedal", "export", "M17-SUSTAIN-PEDALS", "down, hold, and release")).Wire("gpifAutomation.Type", conformanceSustainWireStrings(wire[0], func(value gpifAutomation) string { return value.Type }), conformanceSustainWireStrings(wantWire, func(value gpifAutomation) string { return value.Type }))
	run.Wire("gpifAutomation.Value", conformanceSustainWireStrings(wire[0], func(value gpifAutomation) string { return value.Value.Text }), conformanceSustainWireStrings(wantWire, func(value gpifAutomation) string { return value.Value.Text }))
	run.Wire("gpifAutomation.Bar", conformanceSustainWireInts(wire[0], func(value gpifAutomation) int { return value.Bar }), conformanceSustainWireInts(wantWire, func(value gpifAutomation) int { return value.Bar }))
	run.Wire("gpifAutomation.Position", conformanceSustainWireFloats(wire[0]), conformanceSustainWireFloats(wantWire))
	run.Wire("gpifAutomation.Linear", conformanceSustainWireBools(wire[0]), make([]bool, len(wantWire)))
	run.Wire("gpifAutomation.Visible", conformanceSustainWireStrings(wire[0], func(value gpifAutomation) string { return value.Visible }), slices.Repeat([]string{"true"}, len(wantWire)))

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if roundTripGot := conformanceSustainMeasureMarkers(roundTrip.Tracks[0].Measures); !reflect.DeepEqual(roundTripGot, want) {
		t.Fatalf("sustain-pedal round trip = %#v, want %#v", roundTripGot, want)
	}

	programmatic := semanticValidPitchedGP8Song(t)
	first := &programmatic.Tracks[0]
	first.Settings.Notation = true
	first.Measures[0].SustainPedals = []SustainPedalMarker{{Type: SustainPedalTypeDown, Position: 0}}
	first.Measures[1].SustainPedals = []SustainPedalMarker{{Type: SustainPedalTypeHold, Position: 0}}
	first.Measures[2].SustainPedals = []SustainPedalMarker{{Type: SustainPedalTypeRelease, Position: 1}}
	second := *first
	second.Name = "Second pedal"
	second.Number = 2
	second.Staves = nil
	second.Measures = slices.Clone(first.Measures)
	for index := range second.Measures {
		second.Measures[index].SustainPedals = nil
	}
	second.Measures[0].SustainPedals = []SustainPedalMarker{
		{Type: SustainPedalTypeDown, Position: 0.25},
		{Type: SustainPedalTypeRelease, Position: 0.75},
	}
	programmatic.Tracks = append(programmatic.Tracks, second)
	err = FinalizeSong(programmatic)
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics := ValidateSong(programmatic); len(diagnostics) != 0 {
		t.Fatalf("valid programmatic sustain pedals = %#v", diagnostics)
	}
	programmaticData, report, err := ExportWithReport(programmatic, ExportFormatGP8, ExportOptions{})
	if err != nil {
		t.Fatalf("programmatic sustain export = %v, %#v", err, report.Entries)
	}
	run.ClaimReport(claimSite("sustain-pedal", "export", "M17-SUSTAIN-PEDALS", "down, hold, and release")).Report("M17-SUSTAIN-PEDALS", reportCodes(report), []string{})
	programmaticWire := conformanceSustainWire(t, programmaticData)
	if got, want := conformanceSustainWirePositionsByTrack(programmaticWire), [][]float64{{0, 1}, {0.25, 0.75}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("track sustain positions = %#v, want %#v", got, want)
	}
	programmaticRoundTrip, err := Parse(programmaticData)
	if err != nil {
		t.Fatal(err)
	}
	if got := programmaticRoundTrip.Tracks[0].Measures[1].SustainPedals; !slices.Equal(got, []SustainPedalMarker{{Type: SustainPedalTypeHold}}) {
		t.Fatalf("derived programmatic hold = %#v", got)
	}
	programmaticRoundTrip.Tracks[0].Measures[0].SustainPedals[0].Position = 0.125
	if got := programmaticRoundTrip.Tracks[1].Measures[0].SustainPedals[0].Position; got != 0.25 {
		t.Fatalf("track occurrence edit changed second track position to %v", got)
	}
	run.Dispatch("gpifReadSustainPedals:automation.Type", conformanceSustainMeasureMarkers(programmaticRoundTrip.Tracks[1].Measures)[0], []SustainPedalMarker{{Type: SustainPedalTypeDown, Position: 0.25}, {Type: SustainPedalTypeRelease, Position: 0.75}})

	for _, test := range []struct {
		name       string
		mutate     func(string) string
		code       string
		pathSuffix string
	}{
		{name: "invalid reference", mutate: func(gpif string) string { return strings.Replace(gpif, "<Value>0 1</Value>", "<Value>0 2</Value>", 1) }, code: "GPIF.Track.Automation.SustainPedal.Value.Invalid", pathSuffix: "/Value"},
		{name: "invalid position", mutate: func(gpif string) string {
			return strings.Replace(gpif, "<Position>0.25</Position>", "<Position>NaN</Position>", 1)
		}, code: "GPIF.Track.Automation.SustainPedal.Position.Invalid", pathSuffix: "/Position"},
		{name: "invalid bar", mutate: func(gpif string) string {
			return conformanceReplaceFirstSustainLeaf(t, gpif, "<Bar>0</Bar>", "<Bar>99</Bar>")
		}, code: "GPIF.Track.Automation.SustainPedal.Bar.Invalid", pathSuffix: "/Bar"},
		{name: "non-increasing order", mutate: func(gpif string) string {
			return strings.Replace(gpif, "<Position>0.75</Position>", "<Position>0.25</Position>", 1)
		}, code: "GPIF.Track.Automation.SustainPedal.Order.Invalid", pathSuffix: "/Position"},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalid := rewriteConformanceGPIF(t, programmaticData, test.mutate)
			result, err := ParseWithOptions(invalid, ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			diagnostic := conformanceSourceAuditDiagnosticByCode(result.Diagnostics, test.code)
			if diagnostic == nil || diagnostic.Kind != ParseDiagnosticInvalidData || !strings.HasSuffix(diagnostic.SourcePath, test.pathSuffix) {
				t.Fatalf("source diagnostic = %#v, want %s at *%s", diagnostic, test.code, test.pathSuffix)
			}
			strict, strictErr := ParseWithOptions(invalid, ParseOptions{Strict: true})
			var parseErr *StrictParseError
			if strict == nil || !errors.As(strictErr, &parseErr) {
				t.Fatalf("strict sustain source = %#v, %v, want non-nil result and StrictParseError", strict, strictErr)
			}
		})
	}

	for _, test := range []struct {
		name string
		set  func(*Song)
		code string
	}{
		{name: "undefined type", set: func(song *Song) {
			song.Tracks[0].Measures[0].SustainPedals = []SustainPedalMarker{{Type: SustainPedalType(99)}}
		}, code: "score.measure.sustain-pedal.type"},
		{name: "NaN position", set: func(song *Song) {
			song.Tracks[0].Measures[0].SustainPedals = []SustainPedalMarker{{Type: SustainPedalTypeDown, Position: math.NaN()}}
		}, code: "score.measure.sustain-pedal.position"},
		{name: "high position", set: func(song *Song) {
			song.Tracks[0].Measures[0].SustainPedals = []SustainPedalMarker{{Type: SustainPedalTypeDown, Position: 1.01}}
		}, code: "score.measure.sustain-pedal.position"},
		{name: "duplicate position", set: func(song *Song) {
			song.Tracks[0].Measures[0].SustainPedals = []SustainPedalMarker{{Type: SustainPedalTypeDown, Position: 0.5}, {Type: SustainPedalTypeRelease, Position: 0.5}}
		}, code: "score.measure.sustain-pedal.order"},
		{name: "out of order", set: func(song *Song) {
			song.Tracks[0].Measures[0].SustainPedals = []SustainPedalMarker{{Type: SustainPedalTypeDown, Position: 0.75}, {Type: SustainPedalTypeRelease, Position: 0.25}}
		}, code: "score.measure.sustain-pedal.order"},
		{name: "orphan hold", set: func(song *Song) {
			song.Tracks[0].Measures[0].SustainPedals = []SustainPedalMarker{{Type: SustainPedalTypeHold}}
		}, code: "score.measure.sustain-pedal.hold"},
		{name: "noncanonical hold", set: func(song *Song) {
			song.Tracks[0].Measures[0].SustainPedals = []SustainPedalMarker{{Type: SustainPedalTypeDown}}
			song.Tracks[0].Measures[1].SustainPedals = []SustainPedalMarker{{Type: SustainPedalTypeHold, Position: 0.25}}
		}, code: "score.measure.sustain-pedal.hold"},
		{name: "down after release in entered-down bar", set: func(song *Song) {
			song.Tracks[0].Measures[0].SustainPedals = []SustainPedalMarker{{Type: SustainPedalTypeDown}}
			song.Tracks[0].Measures[1].SustainPedals = []SustainPedalMarker{
				{Type: SustainPedalTypeRelease, Position: 0.25},
				{Type: SustainPedalTypeDown, Position: 0.5},
			}
		}, code: "score.measure.sustain-pedal.down"},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalid := semanticValidPitchedGP8Song(t)
			test.set(invalid)
			diagnostics := ValidateSong(invalid)
			if !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == test.code }) {
				t.Fatalf("validation diagnostics = %#v, want %s", diagnostics, test.code)
			}
			output, report, err := ExportWithReport(invalid, ExportFormatGP8, ExportOptions{})
			if len(output) != 0 || err == nil || !hasExportCode(report, "gp8.reject."+test.code) {
				t.Fatalf("invalid sustain export = %d bytes, %#v, %v", len(output), report.Entries, err)
			}
		})
	}

	multiStaff := parseTestFixture(t, "testdata/gp7/grand-staff.gp")
	multiStaff.Tracks[0].Staves[1].Measures[0].SustainPedals = []SustainPedalMarker{{Type: SustainPedalTypeDown}}
	multiStaff.Tracks[0].Measures = multiStaff.Tracks[0].Staves[0].Measures
	if diagnostics := ValidateSong(multiStaff); !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool {
		return diagnostic.Code == "score.measure.sustain-pedal.staff" && diagnostic.Location.Staff == 1
	}) {
		t.Fatalf("later-staff sustain diagnostics = %#v", diagnostics)
	}
}

func TestAlphaTabPreservesSustainPedals(t *testing.T) {
	requireAlphaTabConformance(t)
	var sourceFacts []conformanceSustainPedalFact
	readAlphaTabOracleFacts(t, "--sustain-pedals", "testdata/gp7/sustain.gp", &sourceFacts)
	want := conformanceSustainAlphaTabFacts()
	if !slices.EqualFunc(sourceFacts, want, conformanceSustainFactEqual) {
		t.Fatalf("AlphaTab source sustain facts = %#v, want %#v", sourceFacts, want)
	}

	song := parseTestFixture(t, "testdata/gp7/sustain.gp")
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var outputFacts []conformanceSustainPedalFact
	readAlphaTabOracleFacts(t, "--sustain-pedals", writeConformanceFixture(t, data), &outputFacts)
	if !slices.EqualFunc(outputFacts, sourceFacts, conformanceSustainFactEqual) {
		t.Fatalf("AlphaTab Go-output sustain facts = %#v, want source %#v", outputFacts, sourceFacts)
	}

	// A public edit controls the independently consumed wire position.
	song.Tracks[0].Measures[5].SustainPedals[0].Position = 0.5
	edited, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	readAlphaTabOracleFacts(t, "--sustain-pedals", writeConformanceFixture(t, edited), &outputFacts)
	if got := outputFacts[5].Markers[0]; got != (conformanceSustainPedalMarkerFact{Position: 0.5, Type: "release"}) {
		t.Fatalf("AlphaTab edited sustain marker = %#v", got)
	}

	// AlphaTab decides whether a Down is a Hold from the bar-entry state. It
	// therefore changes even a Down preceded by a same-bar Release into Hold.
	consumerCase := semanticValidPitchedGP8Song(t)
	consumerCase.Tracks[0].Settings.Notation = true
	consumerCase.Tracks[0].Measures[0].SustainPedals = []SustainPedalMarker{{Type: SustainPedalTypeDown}}
	consumerCase.Tracks[0].Measures[1].SustainPedals = []SustainPedalMarker{{Type: SustainPedalTypeRelease, Position: 0.25}}
	consumerData, err := Export(consumerCase, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	consumerData = conformanceAppendSustainWire(t, consumerData, 0, gpifAutomation{
		Type: "SustainPedal", Value: gpifAutomationValue{Text: "0 1"}, Visible: "true", Bar: 1, Position: 0.5,
	})
	readAlphaTabOracleFacts(t, "--sustain-pedals", writeConformanceFixture(t, consumerData), &outputFacts)
	if got, want := outputFacts[1].Markers, []conformanceSustainPedalMarkerFact{{Position: 0.25, Type: "release"}, {Position: 0.5, Type: "hold"}}; !slices.Equal(got, want) {
		t.Fatalf("AlphaTab entered-down bar marker types = %#v, want %#v", got, want)
	}
	conformanceIndependentClaim(t, "field:Measure.SustainPedals", claimSite("sustain-pedal", "import", "M17-SUSTAIN-PEDALS", "down, hold, and release"), claimSite("sustain-pedal", "model", "M17-SUSTAIN-PEDALS", "down, hold, and release"), claimSite("sustain-pedal", "export", "M17-SUSTAIN-PEDALS", "down, hold, and release"))
}

func conformanceSustainFixtureMarkers() [][]SustainPedalMarker {
	return [][]SustainPedalMarker{
		{{Type: SustainPedalTypeDown}, {Type: SustainPedalTypeRelease, Position: 0.25}, {Type: SustainPedalTypeDown, Position: 0.5}, {Type: SustainPedalTypeRelease, Position: 0.625}, {Type: SustainPedalTypeDown, Position: 0.875}},
		{{Type: SustainPedalTypeRelease}},
		{{Type: SustainPedalTypeDown}},
		{{Type: SustainPedalTypeHold}},
		{{Type: SustainPedalTypeHold}},
		{{Type: SustainPedalTypeRelease, Position: 0.25}},
		{{Type: SustainPedalTypeDown}, {Type: SustainPedalTypeRelease, Position: 0.25}, {Type: SustainPedalTypeDown, Position: 0.5}, {Type: SustainPedalTypeRelease, Position: 0.875}},
		{{Type: SustainPedalTypeDown, Position: 0.5}, {Type: SustainPedalTypeRelease, Position: 1}},
	}
}

func conformanceSustainFixtureWire() []gpifAutomation {
	var result []gpifAutomation
	for bar, markers := range conformanceSustainFixtureMarkers() {
		for _, marker := range markers {
			if marker.Type == SustainPedalTypeHold {
				continue
			}
			reference := "0 1"
			if marker.Type == SustainPedalTypeRelease {
				reference = "0 3"
			}
			result = append(result, gpifAutomation{Type: "SustainPedal", Value: gpifAutomationValue{Text: reference}, Visible: "true", Bar: bar, Position: marker.Position})
		}
	}
	return result
}

func conformanceSustainMeasureMarkers(measures []Measure) [][]SustainPedalMarker {
	result := make([][]SustainPedalMarker, len(measures))
	for index := range measures {
		result[index] = slices.Clone(measures[index].SustainPedals)
	}
	return result
}

func conformanceSustainTypes(values [][]SustainPedalMarker) []SustainPedalType {
	var result []SustainPedalType
	for _, markers := range values {
		for _, marker := range markers {
			result = append(result, marker.Type)
		}
	}
	return result
}

func conformanceSustainPositions(values [][]SustainPedalMarker) []float64 {
	var result []float64
	for _, markers := range values {
		for _, marker := range markers {
			result = append(result, marker.Position)
		}
	}
	return result
}

func conformanceSustainWire(t *testing.T, data []byte) [][]gpifAutomation {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	if err := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document); err != nil {
		t.Fatal(err)
	}
	result := make([][]gpifAutomation, len(document.Tracks.Tracks))
	for trackIndex, track := range document.Tracks.Tracks {
		for _, automation := range track.Automations.Automations {
			if automation.Type == "SustainPedal" {
				result[trackIndex] = append(result[trackIndex], automation)
			}
		}
	}
	return result
}

func conformanceSustainWireStrings(values []gpifAutomation, selectValue func(gpifAutomation) string) []string {
	result := make([]string, len(values))
	for index := range values {
		result[index] = selectValue(values[index])
	}
	return result
}

func conformanceSustainWireInts(values []gpifAutomation, selectValue func(gpifAutomation) int) []int {
	result := make([]int, len(values))
	for index := range values {
		result[index] = selectValue(values[index])
	}
	return result
}

func conformanceSustainWireFloats(values []gpifAutomation) []float64 {
	result := make([]float64, len(values))
	for index := range values {
		result[index] = values[index].Position
	}
	return result
}

func conformanceSustainWireBools(values []gpifAutomation) []bool {
	result := make([]bool, len(values))
	for index := range values {
		result[index] = values[index].Linear
	}
	return result
}

func conformanceSustainWirePositionsByTrack(values [][]gpifAutomation) [][]float64 {
	result := make([][]float64, len(values))
	for index := range values {
		result[index] = conformanceSustainWireFloats(values[index])
	}
	return result
}

func conformanceSustainAlphaTabFacts() []conformanceSustainPedalFact {
	markers := conformanceSustainFixtureMarkers()
	result := make([]conformanceSustainPedalFact, 0, len(markers))
	for bar, values := range markers {
		if len(values) == 0 {
			continue
		}
		fact := conformanceSustainPedalFact{Track: 0, Staff: 0, Bar: bar, Markers: make([]conformanceSustainPedalMarkerFact, len(values))}
		for index, marker := range values {
			typeName := "down"
			switch marker.Type {
			case SustainPedalTypeHold:
				typeName = "hold"
			case SustainPedalTypeRelease:
				typeName = "release"
			}
			fact.Markers[index] = conformanceSustainPedalMarkerFact{Position: marker.Position, Type: typeName}
		}
		result = append(result, fact)
	}
	return result
}

func conformanceSustainFactEqual(left, right conformanceSustainPedalFact) bool {
	return left.Track == right.Track && left.Staff == right.Staff && left.Bar == right.Bar && slices.Equal(left.Markers, right.Markers)
}

func conformanceAppendSustainWire(t *testing.T, data []byte, track int, automation gpifAutomation) []byte {
	t.Helper()
	return rewriteConformanceGPIF(t, data, func(gpif string) string {
		var document gpifDocument
		if err := xml.Unmarshal([]byte(gpif), &document); err != nil {
			t.Fatal(err)
		}
		document.Tracks.Tracks[track].Automations.Automations = append(document.Tracks.Tracks[track].Automations.Automations, automation)
		encoded, err := xml.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		return xml.Header + string(encoded)
	})
}

func conformanceReplaceFirstSustainLeaf(t *testing.T, gpif, old, replacement string) string {
	t.Helper()
	start := strings.Index(gpif, "<Type>SustainPedal</Type>")
	if start < 0 {
		t.Fatal("GPIF has no SustainPedal automation")
	}
	offset := strings.Index(gpif[start:], old)
	if offset < 0 {
		t.Fatalf("SustainPedal automation has no %q", old)
	}
	offset += start
	return gpif[:offset] + replacement + gpif[offset+len(old):]
}
