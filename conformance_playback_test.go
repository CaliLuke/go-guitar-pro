// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestConformancePlaybackRouting(t *testing.T) {
	runConformancePlaybackRouting(newConformanceRun(t))
}

func runConformancePlaybackRouting(run *conformanceRun) {
	t := run.t
	song := semanticValidPitchedGP8Song(t)
	track := &song.Tracks[0]
	track.Settings.Notation = true
	track.ChannelIndex = 0
	track.Port = 1
	track.Mute = true
	track.Solo = false
	track.UseRse = true
	song.Channels[0] = MidiChannel{
		Channel: 18, EffectChannel: 19, Instrument: 42, Bank: 2,
		Volume: 101, Balance: 33, Chorus: 12, Reverb: 23, Phaser: 34, Tremolo: 45,
	}
	wantSounds := []TrackSound{
		{Name: "Clean", Label: "A", Path: "factory/clean", Role: "main", Program: 42, Bank: 2},
		{Name: "Lead", Label: "B", Path: "factory/lead", Role: "solo", Program: 81, Bank: 256},
	}
	track.Sounds = slices.Clone(wantSounds)
	wantSoundAutomations := []SoundAutomation{{Bar: 0, Position: 0.5, Sound: 1}}
	track.SoundAutomations = slices.Clone(wantSoundAutomations)
	track.Rse = TrackRse{
		Humanize: 3, AutoAccentuation: AccentuationStrong,
		Equalizer:  RseEqualizer{Knobs: []float32{0.25, -0.5}, Gain: 1.5},
		Instrument: RseInstrument{EffectCategory: "Delay", Effect: "Echo", Instrument: 7, Unknown: 8, SoundBank: 9, EffectNumber: 10},
	}
	song.MasterEffect = RseMasterEffect{
		Equalizer: RseEqualizer{Knobs: []float32{0.5, -0.25}, Gain: 2.5},
		Volume:    0.75, Reverb: 0.4,
	}

	channel := song.Channels[0]
	run.Preserved("Song.Channels", song.Channels, []MidiChannel{channel})
	run.ClaimPrimary(claimSite("midi-routing", "import", "M05-PLAYBACK-ROUTING", "distinct primary and effect channels"), claimSite("midi-routing", "model", "M05-PLAYBACK-ROUTING", "distinct primary and effect channels")).Preserved("MidiChannel.Channel", channel.Channel, uint8(18))
	run.Normalized("MidiChannel.EffectChannel", channel.EffectChannel, uint8(19))
	run.Preserved("MidiChannel.Instrument", channel.Instrument, int32(42))
	run.Preserved("MidiChannel.Bank", channel.Bank, int32(2))
	run.Preserved("MidiChannel.Volume", channel.Volume, int8(101))
	run.Preserved("MidiChannel.Balance", channel.Balance, int8(33))
	run.Omitted("MidiChannel.Chorus", channel.Chorus, int8(12))
	run.Omitted("MidiChannel.Reverb", channel.Reverb, int8(23))
	run.Omitted("MidiChannel.Phaser", channel.Phaser, int8(34))
	run.Omitted("MidiChannel.Tremolo", channel.Tremolo, int8(45))
	run.Normalized("Track.ChannelIndex", track.ChannelIndex, 0)
	run.Normalized("Track.Mute", track.Mute, true)
	run.Normalized("Track.Solo", track.Solo, false)
	run.Omitted("Track.UseRse", track.UseRse, true)
	run.Preserved("Track.Sounds", track.Sounds, wantSounds)
	run.Preserved("Track.SoundAutomations", track.SoundAutomations, wantSoundAutomations)
	for index := range track.Sounds {
		sound := track.Sounds[index]
		run.Preserved("TrackSound.Name", sound.Name, []string{"Clean", "Lead"}[index])
		run.Preserved("TrackSound.Label", sound.Label, []string{"A", "B"}[index])
		run.Preserved("TrackSound.Path", sound.Path, []string{"factory/clean", "factory/lead"}[index])
		run.Preserved("TrackSound.Role", sound.Role, []string{"main", "solo"}[index])
		run.Preserved("TrackSound.Program", sound.Program, []int32{42, 81}[index])
	}
	automation := track.SoundAutomations[0]
	run.Preserved("SoundAutomation.Bar", automation.Bar, 0)
	run.Preserved("SoundAutomation.Position", automation.Position, 0.5)
	run.Preserved("SoundAutomation.Sound", automation.Sound, 1)
	assertPlaybackRSEFields(run, song.MasterEffect, track.Rse)
	for _, controller := range []struct {
		name string
		set  func(*MidiChannel)
	}{
		{"chorus", func(channel *MidiChannel) { channel.Chorus = 12 }},
		{"reverb", func(channel *MidiChannel) { channel.Reverb = 23 }},
		{"phaser", func(channel *MidiChannel) { channel.Phaser = 34 }},
		{"tremolo", func(channel *MidiChannel) { channel.Tremolo = 45 }},
	} {
		t.Run("isolated "+controller.name, func(t *testing.T) {
			probe := semanticValidPitchedGP8Song(t)
			controller.set(&probe.Channels[0])
			if report := PreflightExport(probe, ExportFormatGP8, ExportOptions{}); !hasExportCode(report, "gp8.omit.midi-effects") {
				t.Fatalf("isolated %s report = %#v, want gp8.omit.midi-effects", controller.name, report.Entries)
			}
			data, _, err := ExportWithReport(probe, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
			if len(data) != 0 || err == nil {
				t.Fatalf("isolated %s strict export = %d bytes, %v", controller.name, len(data), err)
			}
		})
	}

	wantCodes := []string{"gp8.omit.master-rse", "gp8.omit.track-rse", "gp8.omit.track-use-rse", "gp8.omit.midi-effects"}
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	for _, code := range wantCodes {
		if !slices.ContainsFunc(report.Entries, func(entry ExportReportEntry) bool { return entry.Code == code }) {
			t.Errorf("report = %#v, want %s", report.Entries, code)
		}
	}
	strictData, _, strictErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	var lossErr *ExportLossError
	if len(strictData) != 0 || !errors.As(strictErr, &lossErr) {
		t.Fatalf("strict export = %d bytes, %v", len(strictData), strictErr)
	}
	data, exported, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: wantCodes}})
	if err != nil || len(data) == 0 || !reflect.DeepEqual(exported, report) {
		t.Fatalf("allowlisted export = %d bytes, %#v, %v", len(data), exported, err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	gotTrack := roundTrip.Tracks[0]
	gotChannel := roundTrip.Channels[gotTrack.ChannelIndex]
	for path, pair := range map[string][2]any{
		"MidiChannel.Channel":       {gotChannel.Channel, uint8(18)},
		"MidiChannel.EffectChannel": {gotChannel.EffectChannel, uint8(19)},
		"MidiChannel.Instrument":    {gotChannel.Instrument, int32(42)},
		"MidiChannel.Bank":          {gotChannel.Bank, int32(2)},
		"MidiChannel.Volume":        {gotChannel.Volume, int8(101)},
		"MidiChannel.Balance":       {gotChannel.Balance, int8(33)},
		"Track.Mute":                {gotTrack.Mute, true},
		"Track.Solo":                {gotTrack.Solo, false},
		"Track.Sounds":              {gotTrack.Sounds, track.Sounds},
		"Track.SoundAutomations":    {gotTrack.SoundAutomations, track.SoundAutomations},
	} {
		run.Field(path, pair[0], pair[1])
	}
	values := extractGPIFLeafText(t, data)
	run.Wire("gpifMidiConnection.Port", values["GPIF/Tracks/Track/MidiConnection/Port"], "1")
	run.Wire("gpifMidiConnection.PrimaryChannel", values["GPIF/Tracks/Track/MidiConnection/PrimaryChannel"], "2")
	run.Wire("gpifMidiConnection.SecondaryChannel", values["GPIF/Tracks/Track/MidiConnection/SecondaryChannel"], "3")
	run.Wire("gpifTrack.PlaybackState", values["GPIF/Tracks/Track/PlaybackState"], "Mute")
	run.Wire("gpifTrack.AudioEngineState", values["GPIF/Tracks/Track/AudioEngineState"], "MIDI")
	run.Wire("gpifTrack.Sounds", len(gotTrack.Sounds), 2)
	run.Wire("gpifTrack.Automations", len(gotTrack.SoundAutomations), 1)
	run.Wire("gpifSound.Name", gotTrack.Sounds[1].Name, "Lead")
	run.Wire("gpifSound.Label", gotTrack.Sounds[1].Label, "B")
	run.Wire("gpifSound.Path", gotTrack.Sounds[1].Path, "factory/lead")
	run.Wire("gpifSound.Role", gotTrack.Sounds[1].Role, "solo")
	run.Wire("gpifSound.Program", gotTrack.Sounds[1].Program, int32(81))
	run.Wire("gpifSound.Channel", gotChannel.Channel%16, uint8(2))

	generalMIDIData := rewriteConformanceGPIF(t, data, func(source string) string {
		start := strings.Index(source, "<MidiConnection>")
		if start < 0 {
			t.Fatal("GPIF has no MIDI connection")
		}
		end := strings.Index(source[start:], "</MidiConnection>")
		if end < 0 {
			t.Fatal("GPIF MIDI connection is not closed")
		}
		end += start + len("</MidiConnection>")
		return source[:start] + "<GeneralMidi><Program>81</Program><Port>2</Port><PrimaryChannel>4</PrimaryChannel><SecondaryChannel>5</SecondaryChannel></GeneralMidi>" + source[end:]
	})
	generalMIDIResult, err := ParseWithOptions(generalMIDIData, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	generalTrack := generalMIDIResult.Song.Tracks[0]
	generalChannel := generalMIDIResult.Song.Channels[generalTrack.ChannelIndex]
	generalValues := extractGPIFLeafText(t, generalMIDIData)
	run.Wire("gpifGeneralMidi.Program", generalChannel.Instrument, int32(42))
	run.Wire("gpifGeneralMidi.Port", generalValues["GPIF/Tracks/Track/GeneralMidi/Port"], "2")
	run.Wire("gpifGeneralMidi.PrimaryChannel", generalChannel.Channel, uint8(36))
	run.Wire("gpifGeneralMidi.SecondaryChannel", generalChannel.EffectChannel, uint8(37))

	unbound := *song
	unbound.Tracks = slices.Clone(song.Tracks)
	unbound.Tracks[0].ChannelIndex = -1
	if diagnostics := ValidateSong(&unbound); slices.ContainsFunc(diagnostics, func(d ScoreDiagnostic) bool { return d.Code == "score.track.channel-reference" }) {
		t.Fatalf("unbound channel diagnostics = %#v", diagnostics)
	}
	for _, invalid := range []int{-2, len(song.Channels)} {
		unbound.Tracks[0].ChannelIndex = invalid
		if diagnostics := ValidateSong(&unbound); !slices.ContainsFunc(diagnostics, func(d ScoreDiagnostic) bool { return d.Code == "score.track.channel-reference" }) {
			t.Errorf("channel %d diagnostics = %#v", invalid, diagnostics)
		}
	}
	invalidMixer := *song
	invalidMixer.Channels = slices.Clone(song.Channels)
	invalidMixer.Channels[0].Volume = -1
	if diagnostics := ValidateSong(&invalidMixer); !slices.ContainsFunc(diagnostics, func(d ScoreDiagnostic) bool { return d.Code == "score.channel.volume" }) {
		t.Fatalf("invalid mixer diagnostics = %#v", diagnostics)
	}
	if data, _, err := ExportWithReport(&invalidMixer, ExportFormatGP8, ExportOptions{}); err == nil || len(data) != 0 {
		t.Fatalf("invalid mixer export = %d bytes, %v", len(data), err)
	}
}

func assertPlaybackRSEFields(run *conformanceRun, master RseMasterEffect, track TrackRse) {
	wantMaster := RseMasterEffect{Equalizer: RseEqualizer{Knobs: []float32{0.5, -0.25}, Gain: 2.5}, Volume: 0.75, Reverb: 0.4}
	wantTrack := TrackRse{Humanize: 3, AutoAccentuation: AccentuationStrong, Equalizer: RseEqualizer{Knobs: []float32{0.25, -0.5}, Gain: 1.5}, Instrument: RseInstrument{EffectCategory: "Delay", Effect: "Echo", Instrument: 7, Unknown: 8, SoundBank: 9, EffectNumber: 10}}
	run.Omitted("Song.MasterEffect", master, wantMaster)
	run.Omitted("RseMasterEffect.Equalizer", master.Equalizer, wantMaster.Equalizer)
	run.Omitted("RseMasterEffect.Volume", master.Volume, float32(0.75))
	run.Omitted("RseMasterEffect.Reverb", master.Reverb, float32(0.4))
	for index, equalizer := range []RseEqualizer{master.Equalizer, track.Equalizer} {
		want := []RseEqualizer{wantMaster.Equalizer, wantTrack.Equalizer}[index]
		run.Omitted("RseEqualizer.Knobs", equalizer.Knobs, want.Knobs)
		run.Omitted("RseEqualizer.Gain", equalizer.Gain, want.Gain)
	}
	run.Omitted("Track.Rse", track, wantTrack)
	run.Omitted("TrackRse.Instrument", track.Instrument, wantTrack.Instrument)
	run.Omitted("TrackRse.Equalizer", track.Equalizer, wantTrack.Equalizer)
	run.Omitted("TrackRse.Humanize", track.Humanize, uint8(3))
	run.Omitted("TrackRse.AutoAccentuation", track.AutoAccentuation, AccentuationStrong)
	run.Omitted("RseInstrument.EffectCategory", track.Instrument.EffectCategory, "Delay")
	run.Omitted("RseInstrument.Effect", track.Instrument.Effect, "Echo")
	run.Omitted("RseInstrument.Instrument", track.Instrument.Instrument, int16(7))
	run.Omitted("RseInstrument.Unknown", track.Instrument.Unknown, int16(8))
	run.Omitted("RseInstrument.SoundBank", track.Instrument.SoundBank, int16(9))
	run.Omitted("RseInstrument.EffectNumber", track.Instrument.EffectNumber, int16(10))
	for index, value := range []Accentuation{AccentuationNone, AccentuationVerySoft, AccentuationSoft, AccentuationMedium, AccentuationStrong, AccentuationVeryStrong} {
		probe := semanticValidPitchedGP8Song(run.t)
		probe.Tracks[0].Rse.AutoAccentuation = value
		probeReport := PreflightExport(probe, ExportFormatGP8, ExportOptions{})
		run.Enum([]string{"Accentuation.AccentuationNone", "Accentuation.AccentuationVerySoft", "Accentuation.AccentuationSoft", "Accentuation.AccentuationMedium", "Accentuation.AccentuationStrong", "Accentuation.AccentuationVeryStrong"}[index], hasExportCode(probeReport, "gp8.omit.track-rse"), value != AccentuationNone)
	}
}
