// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestConformanceMIDIBank(t *testing.T) {
	runConformanceMIDIBank(newConformanceRun(t))
}

func runConformanceMIDIBank(run *conformanceRun) {
	t := run.t
	result, err := ParseWithOptions(conformanceGPIFArchive(t, conformanceMIDIBankGPIF()), ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	track := &result.Song.Tracks[0]
	wantSounds := []TrackSound{
		{Name: "Initial", Label: "Initial", Path: "Midi/25", Role: "Factory", Program: 25, Bank: 0},
		{Name: "LSB", Label: "LSB", Path: "Midi/26", Role: "User", Program: 26, Bank: 77},
		{Name: "MSB", Label: "MSB", Path: "Midi/27", Role: "User", Program: 27, Bank: 256},
	}
	wantAutomations := []SoundAutomation{
		{Bar: 0, Position: 0, Sound: 0},
		{Bar: 1, Position: 0.5, Sound: 1},
		{Bar: 1, Position: 0.5, Sound: 2},
	}
	run.Preserved("Track.Sounds", track.Sounds, wantSounds)
	run.Preserved("TrackSound.Bank", []int32{track.Sounds[0].Bank, track.Sounds[1].Bank, track.Sounds[2].Bank}, []int32{0, 77, 256})
	run.Preserved("Track.SoundAutomations", track.SoundAutomations, wantAutomations)
	run.ClaimPrimary(claimSite("midi-bank", "import", "M05-MIDI-BANK", "banks 0, 77, 256, and 16383")).Preserved("MidiChannel.Bank", result.Song.Channels[track.ChannelIndex].Bank, int32(0))
	if got := result.Song.Channels[track.ChannelIndex].Instrument; got != 25 {
		t.Fatalf("explicit first sound program mirror = %d, want 25", got)
	}

	// The explicit sound table is authoritative. The channel is its import-time
	// compatibility mirror and is the fallback only when the table is absent.
	track.Sounds[0].Bank = 16383
	data, report, err := ExportWithReport(result.Song, ExportFormatGP8, ExportOptions{})
	if err != nil || !hasExportCode(report, "gp8.normalize.sound-authority") {
		t.Fatalf("explicit sound authority export = %v, %#v", err, report.Entries)
	}
	run.ClaimReport(claimSite("midi-bank", "export", "M05-MIDI-BANK", "banks 0, 77, 256, and 16383")).Report("M05-MIDI-BANK", reportCodes(report), []string{"gp8.normalize.source-version", "gp8.omit.track-display-settings", "gp8.normalize.sound-authority"})
	wire := conformanceSingleWireTrack(t, data)
	run.Wire("gpifSound.LSB", conformanceMIDIBankLSBs(wire), []int{127, 77, 0})
	run.ClaimSerialization(claimSite("midi-bank", "export", "M05-MIDI-BANK", "banks 0, 77, 256, and 16383")).Wire("gpifSound.MSB", conformanceMIDIBankMSBs(wire), []int{127, 0, 2})
	run.Wire("gpifSound.Program", conformanceMIDIBankPrograms(wire), []int{25, 26, 27})
	run.Wire("gpifTrack.Automations", len(wire.Automations.Automations), 3)
	run.Wire("gpifAutomation.Position", []float64{wire.Automations.Automations[0].Position, wire.Automations.Automations[1].Position, wire.Automations.Automations[2].Position}, []float64{0, 0.5, 0.5})
	run.Wire("gpifAutomation.Value", []string{wire.Automations.Automations[0].Value.Text, wire.Automations.Automations[1].Value.Text, wire.Automations.Automations[2].Value.Text}, []string{"Midi/25;Initial;Factory", "Midi/26;LSB;User", "Midi/27;MSB;User"})
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("TrackSound.Bank", []int32{roundTrip.Tracks[0].Sounds[0].Bank, roundTrip.Tracks[0].Sounds[1].Bank, roundTrip.Tracks[0].Sounds[2].Bank}, []int32{16383, 77, 256})
	run.Preserved("Track.SoundAutomations", roundTrip.Tracks[0].SoundAutomations, wantAutomations)
	run.ClaimPrimary(claimSite("midi-bank", "model", "M05-MIDI-BANK", "banks 0, 77, 256, and 16383")).Preserved("MidiChannel.Bank", roundTrip.Channels[roundTrip.Tracks[0].ChannelIndex].Bank, int32(16383))

	fallback := semanticValidPitchedGP8Song(t)
	fallback.Tracks[0].Settings.Notation = true
	fallback.Tracks[0].Sounds = nil
	fallback.Tracks[0].SoundAutomations = nil
	fallback.Channels[0].Bank = 77
	fallbackData, fallbackReport, err := ExportWithReport(fallback, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(fallbackReport.Entries) != 0 {
		t.Fatalf("channel fallback export = %v, %#v", err, fallbackReport.Entries)
	}
	fallbackWire := conformanceSingleWireTrack(t, fallbackData)
	run.Wire("gpifSound.LSB", fallbackWire.Sounds.Sounds[0].LSB, 77)
	run.Wire("gpifSound.MSB", fallbackWire.Sounds.Sounds[0].MSB, 0)
	fallbackRoundTrip, err := Parse(fallbackData)
	if err != nil {
		t.Fatal(err)
	}
	run.ClaimPrimary(claimSite("midi-bank", "export", "M05-MIDI-BANK", "banks 0, 77, 256, and 16383")).Preserved("MidiChannel.Bank", fallbackRoundTrip.Channels[fallbackRoundTrip.Tracks[0].ChannelIndex].Bank, int32(77))

	for _, invalid := range []int32{-1, 16384} {
		t.Run("invalid channel bank", func(t *testing.T) {
			probe := semanticValidPitchedGP8Song(t)
			probe.Channels[0].Bank = invalid
			if diagnostics := ValidateSong(probe); !slices.ContainsFunc(diagnostics, func(d ScoreDiagnostic) bool { return d.Code == "score.channel.bank" }) {
				t.Fatalf("channel bank diagnostics = %#v", diagnostics)
			}
			if output, exportErr := Export(probe, ExportFormatGP8); len(output) != 0 || exportErr == nil {
				t.Fatalf("channel bank export = %d bytes, %v", len(output), exportErr)
			}
		})
		t.Run("invalid sound bank", func(t *testing.T) {
			probe := semanticValidPitchedGP8Song(t)
			probe.Tracks[0].Sounds = []TrackSound{{Name: "bad", Program: 25, Bank: invalid}}
			if diagnostics := ValidateSong(probe); !slices.ContainsFunc(diagnostics, func(d ScoreDiagnostic) bool { return d.Code == "score.track-sound.bank" }) {
				t.Fatalf("track sound bank diagnostics = %#v", diagnostics)
			}
			if output, exportErr := Export(probe, ExportFormatGP8); len(output) != 0 || exportErr == nil {
				t.Fatalf("track sound bank export = %d bytes, %v", len(output), exportErr)
			}
		})
	}

	for _, replacement := range []string{"<MSB>-1</MSB>", "<MSB>128</MSB>", "<LSB>-1</LSB>", "<LSB>128</LSB>"} {
		t.Run("invalid source "+replacement, func(t *testing.T) {
			source := strings.Replace(conformanceMIDIBankGPIF(), "<LSB>77</LSB><MSB>0</MSB>", replacement, 1)
			permissive, parseErr := ParseWithOptions(conformanceGPIFArchive(t, source), ParseOptions{})
			if parseErr != nil || conformanceSourceAuditDiagnosticByCode(permissive.Diagnostics, "GPIF.Track.Sound.MIDI.Bank.Invalid") == nil {
				t.Fatalf("permissive invalid source = %#v, %v", permissive, parseErr)
			}
			strict, strictErr := ParseWithOptions(conformanceGPIFArchive(t, source), ParseOptions{Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticInvalidData}})
			var policyErr *StrictParseError
			if strict == nil || !errors.As(strictErr, &policyErr) {
				t.Fatalf("strict invalid source = %#v, %v", strict, strictErr)
			}
		})
	}
}

func TestAlphaTabPreservesMIDIBanks(t *testing.T) {
	requireAlphaTabConformance(t)
	root := "references/alphaTab/packages/alphatab/test-data"
	for _, test := range []struct {
		path      string
		wantBanks []int32
	}{
		{path: root + "/guitarpro5/bank.gp5", wantBanks: []int32{0, 77}},
		{path: root + "/guitarpro8/bank.gp", wantBanks: []int32{0, 77, 256}},
	} {
		data, err := os.ReadFile(test.path)
		if err != nil {
			t.Fatal(err)
		}
		song, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		got := make([]int32, len(song.Tracks))
		for index := range song.Tracks {
			got[index] = song.Channels[song.Tracks[index].ChannelIndex].Bank
		}
		if !slices.Equal(got, test.wantBanks) {
			t.Fatalf("%s banks = %v, want %v", test.path, got, test.wantBanks)
		}
	}

	data, err := os.ReadFile(root + "/guitarpro8/bank-change.gp")
	if err != nil {
		t.Fatal(err)
	}
	song, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := song.Tracks[0].SoundAutomations, []SoundAutomation{{Bar: 0, Sound: 0}, {Bar: 1, Sound: 1}}; !slices.Equal(got, want) {
		t.Fatalf("bank-change sound automations = %#v, want %#v", got, want)
	}
	exported, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var facts []conformanceAlphaTabMIDIBankTrack
	readAlphaTabOracleFacts(t, "--midi-bank", writeConformanceFixture(t, exported), &facts)
	if len(facts) != 1 || facts[0].Bank != 0 || facts[0].Program != 25 {
		t.Fatalf("AlphaTab bank-change track facts = %#v", facts)
	}
	want := []conformanceAlphaTabMIDIBankAutomation{
		{Bar: 0, Position: 0, Type: "instrument", Value: 25},
		{Bar: 1, Position: 0, Type: "bank", Value: 256},
		{Bar: 1, Position: 0, Type: "instrument", Value: 25},
	}
	if !slices.Equal(facts[0].Automations, want) {
		t.Fatalf("AlphaTab bank-change automations = %#v, want %#v", facts[0].Automations, want)
	}

	programmatic := semanticValidPitchedGP8Song(t)
	programmatic.Tracks[0].Settings.Notation = true
	programmatic.Tracks[0].Sounds = []TrackSound{
		{Name: "Initial", Path: "Midi/25", Role: "Factory", Program: 25},
		{Name: "LSB", Path: "Midi/26", Role: "User", Program: 26, Bank: 77},
		{Name: "MSB", Path: "Midi/27", Role: "User", Program: 27, Bank: 256},
	}
	programmatic.Tracks[0].SoundAutomations = []SoundAutomation{
		{Bar: 0, Position: 0, Sound: 0},
		{Bar: 1, Position: 0.5, Sound: 1},
		{Bar: 1, Position: 0.5, Sound: 2, Linear: true},
	}
	programmaticData, err := Export(programmatic, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	readAlphaTabOracleFacts(t, "--midi-bank", writeConformanceFixture(t, programmaticData), &facts)
	want = []conformanceAlphaTabMIDIBankAutomation{
		{Bar: 0, Position: 0, Type: "instrument", Value: 25},
		{Bar: 1, Position: 0.5, Type: "bank", Value: 77},
		{Bar: 1, Position: 0.5, Type: "instrument", Value: 26},
		{Bar: 1, Position: 0.5, Type: "bank", Value: 256},
		{Bar: 1, Position: 0.5, Type: "instrument", Value: 27, Linear: true},
	}
	if len(facts) != 1 || !slices.Equal(facts[0].Automations, want) {
		t.Fatalf("AlphaTab duplicate-position bank/program order = %#v, want %#v", facts, want)
	}
	conformanceIndependentClaim(t, "field:MidiChannel.Bank", claimSite("midi-bank", "import", "M05-MIDI-BANK", "banks 0, 77, 256, and 16383"), claimSite("midi-bank", "model", "M05-MIDI-BANK", "banks 0, 77, 256, and 16383"), claimSite("midi-bank", "export", "M05-MIDI-BANK", "banks 0, 77, 256, and 16383"))
}

type conformanceAlphaTabMIDIBankTrack struct {
	Track       int                                     `json:"track"`
	Bank        int32                                   `json:"bank"`
	Program     int32                                   `json:"program"`
	Automations []conformanceAlphaTabMIDIBankAutomation `json:"automations"`
}

type conformanceAlphaTabMIDIBankAutomation struct {
	Bar      int     `json:"bar"`
	Position float64 `json:"position"`
	Type     string  `json:"type"`
	Value    int32   `json:"value"`
	Linear   bool    `json:"linear"`
}

func conformanceMIDIBankLSBs(track gpifTrack) []int {
	values := make([]int, len(track.Sounds.Sounds))
	for index := range values {
		values[index] = track.Sounds.Sounds[index].LSB
	}
	return values
}

func conformanceMIDIBankMSBs(track gpifTrack) []int {
	values := make([]int, len(track.Sounds.Sounds))
	for index := range values {
		values[index] = track.Sounds.Sounds[index].MSB
	}
	return values
}

func conformanceMIDIBankPrograms(track gpifTrack) []int {
	values := make([]int, len(track.Sounds.Sounds))
	for index := range values {
		values[index] = track.Sounds.Sounds[index].Program
	}
	return values
}

func conformanceMIDIBankGPIF() string {
	return `<?xml version="1.0" encoding="utf-8"?>
<GPIF>
  <GPVersion>8.0</GPVersion>
  <Score><Title>MIDI bank</Title></Score>
  <MasterTrack><Tracks>0</Tracks></MasterTrack>
  <Tracks><Track id="0"><Name>Guitar</Name>
    <Sounds>
      <Sound><Name>Initial</Name><Label>Initial</Label><Path>Midi/25</Path><Role>Factory</Role><MIDI><LSB>0</LSB><MSB>0</MSB><Program>25</Program></MIDI></Sound>
      <Sound><Name>LSB</Name><Label>LSB</Label><Path>Midi/26</Path><Role>User</Role><MIDI><LSB>77</LSB><MSB>0</MSB><Program>26</Program></MIDI></Sound>
      <Sound><Name>MSB</Name><Label>MSB</Label><Path>Midi/27</Path><Role>User</Role><MIDI><LSB>0</LSB><MSB>2</MSB><Program>27</Program></MIDI></Sound>
    </Sounds>
    <Automations>
      <Automation><Type>Sound</Type><Linear>false</Linear><Bar>0</Bar><Position>0</Position><Visible>true</Visible><Value>Midi/25;Initial;Factory</Value></Automation>
      <Automation><Type>Sound</Type><Linear>false</Linear><Bar>1</Bar><Position>0.5</Position><Visible>true</Visible><Value>Midi/26;LSB;User</Value></Automation>
      <Automation><Type>Sound</Type><Linear>false</Linear><Bar>1</Bar><Position>0.5</Position><Visible>true</Visible><Value>Midi/27;MSB;User</Value></Automation>
    </Automations>
    <GeneralMidi><Program>99</Program><Port>0</Port><PrimaryChannel>0</PrimaryChannel><SecondaryChannel>1</SecondaryChannel></GeneralMidi>
    <Properties/><Staves><Staff><Properties><Property name="Tuning"><Pitches>40 45 50 55 59 64</Pitches></Property></Properties></Staff></Staves>
  </Track></Tracks>
  <MasterBars><MasterBar><Key><Mode>Major</Mode><AccidentalCount>0</AccidentalCount></Key><Time>4/4</Time><Bars>0</Bars></MasterBar><MasterBar><Key><Mode>Major</Mode><AccidentalCount>0</AccidentalCount></Key><Time>4/4</Time><Bars>1</Bars></MasterBar></MasterBars>
  <Bars><Bar id="0"><Clef>G2</Clef><Voices>-1</Voices></Bar><Bar id="1"><Clef>G2</Clef><Voices>-1</Voices></Bar></Bars>
  <Voices/><Beats/><Notes/><Rhythms/>
</GPIF>`
}
