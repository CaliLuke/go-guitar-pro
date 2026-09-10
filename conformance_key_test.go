// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

type conformanceKeyModeCase struct {
	mode  string
	count int8
	minor bool
}

type conformanceAlphaTabKeyFact struct {
	AccidentalCount float64 `json:"accidentalCount"`
	Mode            string  `json:"mode"`
}

func TestConformanceKeyModes(t *testing.T) {
	runConformanceKeyModes(newConformanceRun(t))
}

func runConformanceKeyModes(run *conformanceRun) {
	t := run.t
	known := []conformanceKeyModeCase{
		{mode: "Major", count: -3},
		{mode: "major", count: 4},
		{mode: "Minor", count: -5, minor: true},
		{mode: "minor", count: 6, minor: true},
		{mode: "", count: 2},
	}
	result, err := ParseWithOptions(conformanceGPIFArchive(t, conformanceKeyModeGPIF(known)), ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	want := conformanceKeySignatures(known)
	gotHeaders := make([]KeySignature, len(result.Song.MeasureHeaders))
	for index := range result.Song.MeasureHeaders {
		gotHeaders[index] = result.Song.MeasureHeaders[index].KeySignature
	}
	gotMeasures := make([]KeySignature, len(result.Song.Tracks[0].Measures))
	for index := range result.Song.Tracks[0].Measures {
		gotMeasures[index] = result.Song.Tracks[0].Measures[index].KeySignature
	}
	run.Preserved("MeasureHeader.KeySignature", gotHeaders, want)
	run.Preserved("KeySignature.Key", conformanceKeyCounts(gotHeaders), conformanceKeyCounts(want))
	run.Preserved("KeySignature.IsMinor", conformanceKeyMinorModes(gotHeaders), conformanceKeyMinorModes(want))
	run.Normalized("Measure.KeySignature", gotMeasures, want)
	run.Dispatch("parseGPIFWithContext:mb.Key.Mode", gotHeaders, want)

	data, err := Export(result.Song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	wires := extractMeasureWire(t, data)
	wantModes := []string{"Major", "Major", "Minor", "Minor", "Major"}
	wantCounts := []int{-3, 4, -5, 6, 2}
	gotModes := make([]string, len(wires.masterBars))
	gotCounts := make([]int, len(wires.masterBars))
	for index := range wires.masterBars {
		gotModes[index] = wires.masterBars[index].key.mode
		gotCounts[index] = wires.masterBars[index].key.count
	}
	run.Wire("gpifKey.Mode", gotModes, wantModes)
	run.Wire("gpifKey.AccidentalCount", gotCounts, wantCounts)
	run.Wire("gpifMasterBar.Key", len(wires.masterBars), len(known))

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	roundTripKeys := make([]KeySignature, len(roundTrip.MeasureHeaders))
	for index := range roundTrip.MeasureHeaders {
		roundTripKeys[index] = roundTrip.MeasureHeaders[index].KeySignature
	}
	run.Preserved("MeasureHeader.KeySignature", roundTripKeys, want)

	// A post-parse header edit is authoritative over the stale per-staff
	// compatibility copy and keeps the adjacent headers independent.
	result.Song.MeasureHeaders[1].KeySignature = KeySignature{Key: 7, IsMinor: true}
	if result.Song.Tracks[0].Measures[1].KeySignature != (KeySignature{Key: 4}) {
		t.Fatalf("header edit mutated compatibility measure = %#v", result.Song.Tracks[0].Measures[1].KeySignature)
	}
	edited, editedReport, err := ExportWithReport(result.Song, ExportFormatGP8, ExportOptions{})
	if err != nil || !hasExportCode(editedReport, "gp8.normalize.measure-key-authority") {
		t.Fatalf("edited key export = %v, %#v", err, editedReport.Entries)
	}
	editedWire := extractMeasureWire(t, edited)
	run.Wire("gpifMasterBar.Key", editedWire.masterBars[1].key, conformanceMeasureWireKey{mode: "Minor", count: 7})
	editedRoundTrip, err := Parse(edited)
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("MeasureHeader.KeySignature", editedRoundTrip.MeasureHeaders[1].KeySignature, KeySignature{Key: 7, IsMinor: true})

	unknown := []conformanceKeyModeCase{{mode: "Dorian", count: -2}}
	permissive, err := ParseWithOptions(conformanceGPIFArchive(t, conformanceKeyModeGPIF(unknown)), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	diagnostic := conformanceSourceAuditDiagnosticByCode(permissive.Diagnostics, "GPIF.MasterBar.Key.Mode.InvalidValue")
	if diagnostic == nil || diagnostic.Kind != ParseDiagnosticUnsupportedFeature || diagnostic.SourcePath != "/GPIF/MasterBars/MasterBar[0]/Key/Mode" {
		t.Fatalf("unknown key mode diagnostic = %#v", diagnostic)
	}
	if key := permissive.Song.MeasureHeaders[0].KeySignature; key != (KeySignature{Key: -2}) {
		t.Fatalf("unknown key mode projection = %#v, want permissive major", key)
	}
	strictResult, strictErr := ParseWithOptions(conformanceGPIFArchive(t, conformanceKeyModeGPIF(unknown)), ParseOptions{
		Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticUnsupportedFeature},
	})
	var parseErr *StrictParseError
	if !errors.As(strictErr, &parseErr) || strictResult == nil || conformanceSourceAuditDiagnosticByCode(strictResult.Diagnostics, "GPIF.MasterBar.Key.Mode.InvalidValue") == nil {
		t.Fatalf("unknown key mode strict parse = %#v, %v, want StrictParseError", strictResult, strictErr)
	}
}

func TestAlphaTabPreservesKeyModes(t *testing.T) {
	requireAlphaTabConformance(t)
	supported := []conformanceKeyModeCase{
		{mode: "Major", count: -3},
		{mode: "major", count: 4},
		{mode: "Minor", count: -5, minor: true},
		{mode: "minor", count: 6, minor: true},
	}
	parsed, err := Parse(conformanceGPIFArchive(t, conformanceKeyModeGPIF(supported)))
	if err != nil {
		t.Fatal(err)
	}
	exported, err := Export(parsed, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var facts []conformanceAlphaTabKeyFact
	readAlphaTabOracleFacts(t, "--keys", writeConformanceFixture(t, exported), &facts)
	want := []conformanceAlphaTabKeyFact{
		{AccidentalCount: -3, Mode: "major"},
		{AccidentalCount: 4, Mode: "major"},
		{AccidentalCount: -5, Mode: "minor"},
		{AccidentalCount: 6, Mode: "minor"},
	}
	if !slices.Equal(facts, want) {
		t.Fatalf("AlphaTab Go-export key facts = %#v, want %#v", facts, want)
	}

	var unknownFacts []conformanceAlphaTabKeyFact
	unknown := conformanceGPIFArchive(t, conformanceKeyModeGPIF([]conformanceKeyModeCase{{mode: "Dorian", count: -2}}))
	readAlphaTabOracleFacts(t, "--keys", writeConformanceFixture(t, unknown), &unknownFacts)
	wantUnknown := []conformanceAlphaTabKeyFact{{AccidentalCount: -2, Mode: "major"}}
	if !slices.Equal(unknownFacts, wantUnknown) {
		t.Fatalf("AlphaTab unknown-source key facts = %#v, want %#v", unknownFacts, wantUnknown)
	}
}

func conformanceKeySignatures(cases []conformanceKeyModeCase) []KeySignature {
	result := make([]KeySignature, len(cases))
	for index, test := range cases {
		result[index] = KeySignature{Key: test.count, IsMinor: test.minor}
	}
	return result
}

func conformanceKeyCounts(keys []KeySignature) []int8 {
	result := make([]int8, len(keys))
	for index := range keys {
		result[index] = keys[index].Key
	}
	return result
}

func conformanceKeyMinorModes(keys []KeySignature) []bool {
	result := make([]bool, len(keys))
	for index := range keys {
		result[index] = keys[index].IsMinor
	}
	return result
}

func conformanceKeyModeGPIF(cases []conformanceKeyModeCase) string {
	var masterBars strings.Builder
	var bars strings.Builder
	for index, test := range cases {
		fmt.Fprintf(&masterBars, `<MasterBar><Key><Mode>%s</Mode><AccidentalCount>%d</AccidentalCount></Key><Time>4/4</Time><Bars>%d</Bars></MasterBar>`, test.mode, test.count, index)
		fmt.Fprintf(&bars, `<Bar id="%d"><Clef>G2</Clef><Voices>-1</Voices></Bar>`, index)
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<GPIF>
  <GPVersion>8.0</GPVersion>
  <Score><Title>Key modes</Title></Score>
  <MasterTrack><Tracks>0</Tracks></MasterTrack>
  <Tracks><Track id="0"><Name>Guitar</Name><Properties/><Staves><Staff><Properties><Property name="Tuning"><Pitches>40 45 50 55 59 64</Pitches></Property></Properties></Staff></Staves><MidiConnection><Port>0</Port><PrimaryChannel>0</PrimaryChannel><SecondaryChannel>1</SecondaryChannel></MidiConnection></Track></Tracks>
  <MasterBars>%s</MasterBars>
  <Bars>%s</Bars>
  <Voices/><Beats/><Notes/><Rhythms/>
</GPIF>`, masterBars.String(), bars.String())
}
