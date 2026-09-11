// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"slices"
	"testing"
)

func TestConformanceSoundPreroll(t *testing.T) {
	runConformanceSoundPreroll(newConformanceRun(t))
}

func runConformanceSoundPreroll(run *conformanceRun) {
	t := run.t
	source, err := ParseFile("testdata/gp7/grace.gp")
	if err != nil {
		t.Fatal(err)
	}
	wantSource := []SoundAutomation{{Bar: 0, Position: -0.125, Sound: 0}}
	wantSounds := []TrackSound{{Name: "Acoustic Guitar (Steel)", Label: "Acoustic Guitar (Steel)", Path: "Stringed/Acoustic Guitars/Steel Guitar", Role: "User", Program: 25}}
	run.Preserved("Track.SoundAutomations", source.Tracks[0].SoundAutomations, wantSource)
	run.Preserved("Track.Sounds", source.Tracks[0].Sounds, wantSounds)
	exported, err := Export(source, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(exported)
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("Track.SoundAutomations", roundTrip.Tracks[0].SoundAutomations, wantSource)
	run.Preserved("Track.Sounds", roundTrip.Tracks[0].Sounds, wantSounds)

	song := soundPrerollConformanceSong(t)
	want := slices.Clone(song.Tracks[0].SoundAutomations)
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("valid preroll diagnostics = %#v", diagnostics)
	}
	options := ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}}
	preflight := PreflightExport(song, ExportFormatGP8, options)
	run.Report("M17-SOUND-PREROLL", reportCodes(preflight), []string{})
	data, report, err := ExportWithReport(song, ExportFormatGP8, options)
	if err != nil {
		t.Fatal(err)
	}
	run.Report("M17-SOUND-PREROLL", report, preflight)
	wire := conformanceSingleWireTrack(t, data).Automations.Automations
	if len(wire) != len(want) {
		t.Fatalf("wire events = %#v, want %#v", wire, want)
	}
	for index, event := range wire {
		run.Wire("gpifAutomation.Type", event.Type, "Sound")
		run.Wire("gpifAutomation.Bar", event.Bar, want[index].Bar)
		run.Wire("gpifAutomation.Position", event.Position, want[index].Position)
		sound := song.Tracks[0].Sounds[want[index].Sound]
		run.Wire("gpifAutomation.Value", event.Value.Text, sound.Path+";"+sound.Name+";"+sound.Role)
	}
	roundTrip, err = Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("Track.SoundAutomations", roundTrip.Tracks[0].SoundAutomations, want)
	for index, event := range roundTrip.Tracks[0].SoundAutomations {
		run.Preserved("SoundAutomation.Bar", event.Bar, want[index].Bar)
		run.Preserved("SoundAutomation.Position", event.Position, want[index].Position)
		run.Preserved("SoundAutomation.Sound", event.Sound, want[index].Sound)
	}
	run.Preserved("Track.SoundAutomations", song.Tracks[0].SoundAutomations, want)
}

func TestAlphaTabSoundPreroll(t *testing.T) {
	requireAlphaTabConformance(t)
	var facts []conformanceAlphaTabMIDIBankTrack
	readAlphaTabOracleFacts(t, "--midi-bank", "testdata/gp7/grace.gp", &facts)
	wantSource := []conformanceAlphaTabMIDIBankAutomation{{Bar: 0, Position: -0.125, Type: "instrument", Value: 25}}
	assertSoundPrerollConsumer(t, facts, wantSource)
	source, err := ParseFile("testdata/gp7/grace.gp")
	if err != nil {
		t.Fatal(err)
	}
	data, err := Export(source, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	readAlphaTabOracleFacts(t, "--midi-bank", writeConformanceFixture(t, data), &facts)
	assertSoundPrerollConsumer(t, facts, wantSource)

	data, err = Export(soundPrerollConformanceSong(t), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	readAlphaTabOracleFacts(t, "--midi-bank", writeConformanceFixture(t, data), &facts)
	wantOrdered := []conformanceAlphaTabMIDIBankAutomation{
		{Bar: 0, Position: -0.125, Type: "instrument", Value: 25},
		{Bar: 0, Position: 0, Type: "bank", Value: 77},
		{Bar: 0, Position: 0, Type: "instrument", Value: 26},
		{Bar: 0, Position: -0.125, Type: "bank", Value: 256},
		{Bar: 0, Position: -0.125, Type: "instrument", Value: 27, Linear: true},
		{Bar: 0, Position: -0.125, Type: "bank", Value: 77},
		{Bar: 0, Position: -0.125, Type: "instrument", Value: 26},
	}
	assertSoundPrerollConsumer(t, facts, wantOrdered)
}

func assertSoundPrerollConsumer(t *testing.T, facts []conformanceAlphaTabMIDIBankTrack, want []conformanceAlphaTabMIDIBankAutomation) {
	t.Helper()
	if len(facts) != 1 || facts[0].Track != 0 || facts[0].Program != 25 || facts[0].Bank != 0 || !slices.Equal(facts[0].Automations, want) {
		t.Fatalf("AlphaTab preroll = %#v, want %#v", facts, want)
	}
}

func soundPrerollConformanceSong(t *testing.T) *Song {
	t.Helper()
	song := semanticValidPitchedGP8Song(t)
	song.Tracks[0].Settings.Notation = true
	song.Tracks[0].Sounds = []TrackSound{
		{Name: "Initial", Path: "Midi/25", Role: "User", Program: 25},
		{Name: "Zero", Path: "Midi/26", Role: "User", Program: 26, Bank: 77},
		{Name: "Preroll", Path: "Midi/27", Role: "User", Program: 27, Bank: 256},
	}
	song.Tracks[0].SoundAutomations = []SoundAutomation{
		{Bar: 0, Position: -0.125, Sound: 0},
		{Bar: 0, Position: 0, Sound: 1},
		{Bar: 0, Position: -0.125, Sound: 2, Linear: true, Text: "authored order"},
		{Bar: 0, Position: -0.125, Sound: 1},
	}
	return song
}
