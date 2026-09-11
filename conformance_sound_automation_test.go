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

func TestConformanceSoundSelectionIdentity(t *testing.T) {
	runSoundSelectionIdentity(newConformanceRun(t), false)
}

func runConformanceSoundSelectionIdentity(run *conformanceRun) {
	runSoundSelectionIdentity(run, false)
}

func TestAlphaTabSoundSelectionIdentity(t *testing.T) {
	requireAlphaTabConformance(t)
	runSoundSelectionIdentity(newConformanceRun(t), true)
}

func runSoundSelectionIdentity(run *conformanceRun, oracle bool) {
	t := run.t
	song := consumerLimitSong(t)
	song.Channels[0].Instrument = 27
	song.Tracks[0].Sounds = []TrackSound{
		{Name: "Clean", Label: "Clean label", Path: "factory/clean", Role: "main", Program: 27},
		{Name: "Lead", Label: "Lead label", Path: "user/lead", Role: "solo", Program: 27},
	}
	song.Tracks[0].SoundAutomations = []SoundAutomation{
		{Sound: 1, Text: "opening"}, {Position: 0.75, Sound: 0, Text: "later"},
		{Position: 0.25, Sound: 1, Text: "first at quarter"}, {Position: 0.25, Sound: 0, Text: "second at quarter", Linear: true},
	}
	var expectedReferences []string
	for _, event := range song.Tracks[0].SoundAutomations {
		sound := song.Tracks[0].Sounds[event.Sound]
		expectedReferences = append(expectedReferences, sound.Path+";"+sound.Name+";"+sound.Role)
	}
	for _, reordered := range []bool{false, true} {
		if reordered {
			slices.Reverse(song.Tracks[0].Sounds)
			for i := range song.Tracks[0].SoundAutomations {
				song.Tracks[0].SoundAutomations[i].Sound = 1 - song.Tracks[0].SoundAutomations[i].Sound
			}
		}
		data, report := assertConsumerLossPolicy(t, song, []string{})
		run.Report("M17-SOUND-IDENTITY", reportCodes(report), []string{})
		wire := conformanceSingleWireTrack(t, data)
		run.Wire("gpifTrack.Sounds", len(wire.Sounds.Sounds), 2)
		for i, sound := range song.Tracks[0].Sounds {
			run.Wire("gpifSound.Name", wire.Sounds.Sounds[i].Name, sound.Name)
			run.Wire("gpifSound.Path", wire.Sounds.Sounds[i].Path, sound.Path)
			run.Wire("gpifSound.Role", wire.Sounds.Sounds[i].Role, sound.Role)
			run.Wire("gpifSound.Program", wire.Sounds.Sounds[i].Program, 27)
		}
		if len(wire.Automations.Automations) != len(expectedReferences) {
			t.Fatalf("wire events = %#v", wire.Automations)
		}
		for i, event := range wire.Automations.Automations {
			run.Wire("gpifAutomation.Value", event.Value.Text, expectedReferences[i])
			run.Wire("gpifAutomation.Position", event.Position, song.Tracks[0].SoundAutomations[i].Position)
		}
		parsed, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		run.Preserved("Track.Sounds", parsed.Tracks[0].Sounds, song.Tracks[0].Sounds)
		run.Preserved("Track.SoundAutomations", parsed.Tracks[0].SoundAutomations, song.Tracks[0].SoundAutomations)
		for i, event := range parsed.Tracks[0].SoundAutomations {
			run.Preserved("SoundAutomation.Sound", event.Sound, song.Tracks[0].SoundAutomations[i].Sound)
		}
		if oracle {
			assertSoundIdentityConsumer(t, data)
		}
	}
}

func assertSoundIdentityConsumer(t *testing.T, data []byte) {
	t.Helper()
	var facts struct {
		Sound []conformanceSoundEvent `json:"sound"`
	}
	readAlphaTabOracleFacts(t, "--automations", writeConformanceFixture(t, data), &facts)
	want := []conformanceSoundEvent{
		{Track: 0, Bar: 0, Position: 0, Type: "instrument", Value: 27, Text: "opening", Visible: true},
		{Track: 0, Bar: 0, Position: 0.75, Type: "instrument", Value: 27, Text: "later", Visible: true},
		{Track: 0, Bar: 0, Position: 0.25, Type: "instrument", Value: 27, Text: "first at quarter", Visible: true},
		{Track: 0, Bar: 0, Position: 0.25, Type: "instrument", Value: 27, Text: "second at quarter", Visible: true, Linear: true},
	}
	if !slices.Equal(facts.Sound, want) {
		t.Fatalf("retained AlphaTab same-program events = %#v, want %#v", facts.Sound, want)
	}
}

type conformanceSoundEvent struct {
	Track    int     `json:"track"`
	Bar      int     `json:"bar"`
	Position float64 `json:"position"`
	Type     string  `json:"type"`
	Value    int     `json:"value"`
	Text     string  `json:"text"`
	Visible  bool    `json:"visible"`
	Linear   bool    `json:"linear"`
}

func TestConformanceSoundOpeningAuthority(t *testing.T) {
	runSoundOpeningAuthority(newConformanceRun(t), false)
}

func runConformanceSoundOpeningAuthority(run *conformanceRun) {
	runSoundOpeningAuthority(run, false)
}

func TestAlphaTabSoundOpeningAuthority(t *testing.T) {
	requireAlphaTabConformance(t)
	runSoundOpeningAuthority(newConformanceRun(t), true)
}

func runSoundOpeningAuthority(run *conformanceRun, oracle bool) {
	t := run.t
	for _, test := range []struct {
		name                 string
		channel              int32
		definitions, opening bool
		initial, selected    int32
		codes                []string
	}{
		{name: "channel fallback", channel: 73, initial: 73, selected: 73, codes: []string{}},
		{name: "initial sound", channel: 27, definitions: true, initial: 27, selected: 27, codes: []string{}},
		{name: "initial overrides channel", channel: 99, definitions: true, initial: 27, selected: 27, codes: []string{"gp8.normalize.sound-authority"}},
		{name: "explicit opening selection", channel: 27, definitions: true, opening: true, initial: 27, selected: 81, codes: []string{}},
		{name: "three distinct programs", channel: 99, definitions: true, opening: true, initial: 27, selected: 81, codes: []string{"gp8.normalize.sound-authority"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := consumerLimitSong(t)
			song.Channels[0].Instrument = test.channel
			if test.definitions {
				song.Tracks[0].Sounds = []TrackSound{{Name: "Initial", Path: "factory/initial", Role: "main", Program: 27}, {Name: "Opening", Path: "user/opening", Role: "solo", Program: 81}}
			}
			if test.opening {
				song.Tracks[0].SoundAutomations = []SoundAutomation{{Bar: 0, Position: 0, Sound: 1}}
			}
			data, report := assertConsumerLossPolicy(t, song, test.codes)
			run.Report("M17-SOUND-AUTHORITY", reportCodes(report), test.codes)
			wire := conformanceSingleWireTrack(t, data)
			run.Wire("gpifSound.Program", wire.Sounds.Sounds[0].Program, int(test.initial))
			run.Wire("gpifTrack.Automations", len(wire.Automations.Automations), len(song.Tracks[0].SoundAutomations))
			if test.opening {
				run.Wire("gpifAutomation.Value", wire.Automations.Automations[0].Value.Text, "user/opening;Opening;solo")
			}
			parsed, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			run.Preserved("MidiChannel.Instrument", parsed.Channels[0].Instrument, test.initial)
			run.Preserved("TrackSound.Program", parsed.Tracks[0].Sounds[0].Program, test.initial)
			run.Preserved("Track.SoundAutomations", parsed.Tracks[0].SoundAutomations, song.Tracks[0].SoundAutomations)
			if oracle {
				var facts []conformanceAlphaTabMIDIBankTrack
				readAlphaTabOracleFacts(t, "--midi-bank", writeConformanceFixture(t, data), &facts)
				want := []conformanceAlphaTabMIDIBankAutomation{{Bar: 0, Position: 0, Type: "instrument", Value: test.selected}}
				if len(facts) != 1 || facts[0].Program != test.initial || !slices.Equal(facts[0].Automations, want) {
					t.Fatalf("AlphaTab base program/opening selection = %#v, want base %d and %#v", facts, test.initial, want)
				}
			}
		})
	}
}
