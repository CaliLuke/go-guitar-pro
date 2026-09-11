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

type conformanceLegatoFact struct {
	Track       int  `json:"track"`
	Staff       int  `json:"staff"`
	Bar         int  `json:"bar"`
	Voice       int  `json:"voice"`
	Beat        int  `json:"beat"`
	Origin      bool `json:"origin"`
	Destination bool `json:"destination"`
}

type conformanceLegatoWireFact struct {
	Origin      string
	Destination string
}

func TestConformanceLegato(t *testing.T) {
	runConformanceLegato(newConformanceRun(t))
}

func runConformanceLegato(run *conformanceRun) {
	t := run.t
	song := parseTestFixture(t, "testdata/gp7/numbered.gp")
	beats := song.Tracks[0].Measures[39].Voices[0].Beats
	want := []BeatLegato{
		{Origin: true},
		{Origin: true, Destination: true},
		{Origin: true, Destination: true},
		{Destination: true},
	}
	got := conformanceBeatLegatos(t, beats)
	run.ClaimPrimary(claimSite("legato-slurs", "import", "M10-LEGATO", "origin and destination endpoints")).Preserved("Beat.Legato", got, want)
	run.Preserved("BeatLegato.Origin", conformanceLegatoOrigins(got), []bool{true, true, true, false})
	run.Preserved("BeatLegato.Destination", conformanceLegatoDestinations(got), []bool{false, true, true, true})

	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	wire := conformanceLegatoWire(t, data, 39)
	wantWire := []conformanceLegatoWireFact{
		{Origin: "true", Destination: "false"},
		{Origin: "true", Destination: "true"},
		{Origin: "true", Destination: "true"},
		{Origin: "false", Destination: "true"},
	}
	run.Wire("gpifBeat.Legato", wire, wantWire)
	run.Wire("gpifLegato.Origin", conformanceLegatoWireOrigins(wire), []string{"true", "true", "true", "false"})
	run.Wire("gpifLegato.Destination", conformanceLegatoWireDestinations(wire), []string{"false", "true", "true", "true"})

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.ClaimPrimary(claimSite("legato-slurs", "model", "M10-LEGATO", "origin and destination endpoints")).Preserved("Beat.Legato", conformanceBeatLegatos(t, roundTrip.Tracks[0].Measures[39].Voices[0].Beats), want)

	// The destination-only final beat is an excerpt boundary and must remain
	// authored even though this package does not synthesize phrase links.
	excerpt := semanticValidPitchedGP8Song(t)
	excerpt.Tracks[0].Settings.Notation = true
	excerptBeat := &excerpt.Tracks[0].Measures[0].Voices[0].Beats[0]
	excerptBeat.Legato = &BeatLegato{Destination: true}
	excerptData, report, err := ExportWithReport(excerpt, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("destination-only legato export = %v, %#v", err, report.Entries)
	}
	run.Report("M10-LEGATO", reportCodes(report), []string{})
	excerptWire := conformanceLegatoWire(t, excerptData, 0)
	if !slices.Equal(excerptWire, []conformanceLegatoWireFact{{Origin: "false", Destination: "true"}}) {
		t.Fatalf("destination-only legato wire = %#v, want exact false/true attributes", excerptWire)
	}
	excerptRoundTrip, err := Parse(excerptData)
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("Beat.Legato", *excerptRoundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Legato, BeatLegato{Destination: true})

	// A reused GPIF definition produces independent mutable occurrence state.
	reusedResult, err := ParseWithOptions(conformanceGPIFArchive(t, conformanceReusedLegatoGPIF()), ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	reused := reusedResult.Song
	first := reused.Tracks[0].Measures[0].Voices[0].Beats[1].Legato
	second := reused.Tracks[0].Measures[1].Voices[0].Beats[0].Legato
	if first == nil || second == nil || first == second {
		t.Fatalf("reused legato ownership = %p and %p, want distinct non-nil records", first, second)
	}
	first.Origin = false
	if !second.Origin || !second.Destination {
		t.Fatalf("first occurrence edit changed second occurrence: %#v", second)
	}
	if absent := reused.Tracks[0].Measures[1].Voices[0].Beats[1].Legato; absent != nil {
		t.Fatalf("beat without authored Legato = %#v, want nil", absent)
	}
	gp6Source := strings.Replace(conformanceReusedLegatoGPIF(), "<GPVersion>8.0</GPVersion>", "<GPVersion>6.0</GPVersion>", 1)
	gp6Result, err := ParseWithOptions(conformanceGPIFArchive(t, gp6Source), ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	gp6 := gp6Result.Song
	if got := gp6.Tracks[0].Measures[0].Voices[0].Beats[0].Legato; got == nil || *got != (BeatLegato{Origin: true}) {
		t.Fatalf("GP6 legato = %#v, want origin-only endpoint", got)
	}

	for _, test := range []struct {
		name string
		old  string
		new  string
		code string
		path string
	}{
		{name: "missing origin", old: ` origin="true"`, code: "GPIF.Beat.Legato.InvalidOrigin", path: "/GPIF/Beats/Beat[@id=\"0\"]/Legato/@origin"},
		{name: "invalid origin", old: `origin="true"`, new: `origin="yes"`, code: "GPIF.Beat.Legato.InvalidOrigin", path: "/GPIF/Beats/Beat[@id=\"0\"]/Legato/@origin"},
		{name: "missing destination", old: ` destination="false"`, code: "GPIF.Beat.Legato.InvalidDestination", path: "/GPIF/Beats/Beat[@id=\"0\"]/Legato/@destination"},
		{name: "invalid destination", old: `destination="false"`, new: `destination="1"`, code: "GPIF.Beat.Legato.InvalidDestination", path: "/GPIF/Beats/Beat[@id=\"0\"]/Legato/@destination"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := strings.Replace(conformanceReusedLegatoGPIF(), test.old, test.new, 1)
			result, err := ParseWithOptions(conformanceGPIFArchive(t, source), ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			diagnostic := conformanceSourceAuditDiagnosticByCode(result.Diagnostics, test.code)
			if diagnostic == nil || diagnostic.Kind != ParseDiagnosticInvalidData || diagnostic.SourcePath != test.path {
				t.Fatalf("legato diagnostic = %#v, want %s at %s", diagnostic, test.code, test.path)
			}
			strict, strictErr := ParseWithOptions(conformanceGPIFArchive(t, source), ParseOptions{Strict: true})
			var parseErr *StrictParseError
			if strict == nil || !errors.As(strictErr, &parseErr) {
				t.Fatalf("strict malformed legato = %#v, %v, want non-nil result and StrictParseError", strict, strictErr)
			}
		})
	}
}

func TestAlphaTabPreservesLegato(t *testing.T) {
	requireAlphaTabConformance(t)
	song := parseTestFixture(t, "testdata/gp7/numbered.gp")
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var facts []conformanceLegatoFact
	readAlphaTabOracleFacts(t, "--legato", writeConformanceFixture(t, data), &facts)
	want := []conformanceLegatoFact{
		{Track: 0, Staff: 0, Bar: 39, Voice: 0, Beat: 0, Origin: true},
		{Track: 0, Staff: 0, Bar: 39, Voice: 0, Beat: 1, Origin: true, Destination: true},
		{Track: 0, Staff: 0, Bar: 39, Voice: 0, Beat: 2, Origin: true, Destination: true},
		{Track: 0, Staff: 0, Bar: 39, Voice: 0, Beat: 3, Destination: true},
	}
	if !slices.Equal(facts, want) {
		t.Fatalf("AlphaTab legato facts = %#v, want %#v", facts, want)
	}

	// Verify that a public edit controls AlphaTab's independent origin and
	// derived destination state in the emitted score.
	song.Tracks[0].Measures[39].Voices[0].Beats[1].Legato.Origin = false
	edited, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	readAlphaTabOracleFacts(t, "--legato", writeConformanceFixture(t, edited), &facts)
	want = []conformanceLegatoFact{
		{Track: 0, Staff: 0, Bar: 39, Voice: 0, Beat: 0, Origin: true},
		{Track: 0, Staff: 0, Bar: 39, Voice: 0, Beat: 1, Destination: true},
		{Track: 0, Staff: 0, Bar: 39, Voice: 0, Beat: 2, Origin: true},
		{Track: 0, Staff: 0, Bar: 39, Voice: 0, Beat: 3, Destination: true},
	}
	if !slices.Equal(facts, want) {
		t.Fatalf("AlphaTab edited legato facts = %#v, want %#v", facts, want)
	}
	conformanceIndependentClaim(t, "field:Beat.Legato", claimSite("legato-slurs", "import", "M10-LEGATO", "origin and destination endpoints"), claimSite("legato-slurs", "model", "M10-LEGATO", "origin and destination endpoints"))
}

func conformanceBeatLegatos(t *testing.T, beats []Beat) []BeatLegato {
	t.Helper()
	result := make([]BeatLegato, len(beats))
	for index := range beats {
		if beats[index].Legato == nil {
			t.Fatalf("beat %d legato is nil", index)
		}
		result[index] = *beats[index].Legato
	}
	return result
}

func conformanceLegatoOrigins(values []BeatLegato) []bool {
	result := make([]bool, len(values))
	for index := range values {
		result[index] = values[index].Origin
	}
	return result
}

func conformanceLegatoDestinations(values []BeatLegato) []bool {
	result := make([]bool, len(values))
	for index := range values {
		result[index] = values[index].Destination
	}
	return result
}

func conformanceLegatoWireOrigins(values []conformanceLegatoWireFact) []string {
	result := make([]string, len(values))
	for index := range values {
		result[index] = values[index].Origin
	}
	return result
}

func conformanceLegatoWireDestinations(values []conformanceLegatoWireFact) []string {
	result := make([]string, len(values))
	for index := range values {
		result[index] = values[index].Destination
	}
	return result
}

func conformanceLegatoWire(t *testing.T, data []byte, measureIndex int) []conformanceLegatoWireFact {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	if err := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document); err != nil {
		t.Fatal(err)
	}
	barID := splitIDs(document.MasterBars.MasterBars[measureIndex].Bars)[0]
	bar := document.Bars.Bars[slices.IndexFunc(document.Bars.Bars, func(bar gpifBar) bool { return bar.ID == barID })]
	voiceID := splitIDs(bar.Voices)[0]
	voice := document.Voices.Voices[slices.IndexFunc(document.Voices.Voices, func(voice gpifVoice) bool { return voice.ID == voiceID })]
	result := make([]conformanceLegatoWireFact, 0, len(splitIDs(voice.Beats)))
	for _, beatID := range splitIDs(voice.Beats) {
		beat := document.Beats.Beats[slices.IndexFunc(document.Beats.Beats, func(beat gpifBeat) bool { return beat.ID == beatID })]
		if beat.Legato != nil {
			result = append(result, conformanceLegatoWireFact{Origin: beat.Legato.Origin, Destination: beat.Legato.Destination})
		}
	}
	return result
}

func conformanceReusedLegatoGPIF() string {
	return `<?xml version="1.0" encoding="utf-8"?>
<GPIF>
  <GPVersion>8.0</GPVersion>
  <Score><Title>Legato</Title></Score>
  <MasterTrack><Tracks>0</Tracks></MasterTrack>
  <Tracks><Track id="0"><Name>Guitar</Name><Properties/><Staves><Staff><Properties><Property name="Tuning"><Pitches>40 45 50 55 59 64</Pitches></Property></Properties></Staff></Staves><MidiConnection><Port>0</Port><PrimaryChannel>0</PrimaryChannel><SecondaryChannel>1</SecondaryChannel></MidiConnection></Track></Tracks>
  <MasterBars><MasterBar><Time>4/4</Time><Bars>0</Bars></MasterBar><MasterBar><Time>4/4</Time><Bars>1</Bars></MasterBar></MasterBars>
  <Bars><Bar id="0"><Clef>G2</Clef><Voices>0</Voices></Bar><Bar id="1"><Clef>G2</Clef><Voices>1</Voices></Bar></Bars>
  <Voices><Voice id="0"><Beats>0 1 2 3</Beats></Voice><Voice id="1"><Beats>1 4</Beats></Voice></Voices>
  <Beats>
    <Beat id="0"><Rhythm ref="0"/><Legato origin="true" destination="false"/></Beat>
    <Beat id="1"><Rhythm ref="0"/><Legato origin="true" destination="true"/></Beat>
    <Beat id="2"><Rhythm ref="0"/><Legato origin="true" destination="true"/></Beat>
    <Beat id="3"><Rhythm ref="0"/><Legato origin="false" destination="true"/></Beat>
    <Beat id="4"><Rhythm ref="0"/></Beat>
  </Beats>
  <Notes/><Rhythms><Rhythm id="0"><NoteValue>Quarter</NoteValue></Rhythm></Rhythms>
</GPIF>`
}
