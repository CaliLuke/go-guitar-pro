// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

func TestSemanticMatrixM05PlaybackRouting(t *testing.T) {
	runSemanticMatrixM05PlaybackRouting(newSemanticMatrixRun(t))
}

func runSemanticMatrixM05PlaybackRouting(run *semanticMatrixRun) {
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
	track.Sounds = []TrackSound{
		{Name: "Clean", Label: "A", Path: "factory/clean", Role: "main", Program: 42},
		{Name: "Lead", Label: "B", Path: "factory/lead", Role: "solo", Program: 81},
	}
	track.SoundAutomations = []SoundAutomation{{Bar: 0, Position: 0.5, Sound: 1}}
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
	run.Field("Song.Channels", song.Channels, []MidiChannel{channel})
	run.Field("MidiChannel.Channel", channel.Channel, uint8(18))
	run.Field("MidiChannel.EffectChannel", channel.EffectChannel, uint8(19))
	run.Field("MidiChannel.Instrument", channel.Instrument, int32(42))
	run.Field("MidiChannel.Bank", channel.Bank, uint8(2))
	run.Field("MidiChannel.Volume", channel.Volume, int8(101))
	run.Field("MidiChannel.Balance", channel.Balance, int8(33))
	run.Field("MidiChannel.Chorus", channel.Chorus, int8(12))
	run.Field("MidiChannel.Reverb", channel.Reverb, int8(23))
	run.Field("MidiChannel.Phaser", channel.Phaser, int8(34))
	run.Field("MidiChannel.Tremolo", channel.Tremolo, int8(45))
	run.Field("Track.ChannelIndex", track.ChannelIndex, 0)
	run.Field("Track.Mute", track.Mute, true)
	run.Field("Track.Solo", track.Solo, false)
	run.Field("Track.UseRse", track.UseRse, true)
	run.Field("Track.Sounds", track.Sounds, track.Sounds)
	run.Field("Track.SoundAutomations", track.SoundAutomations, track.SoundAutomations)
	for index := range track.Sounds {
		sound := track.Sounds[index]
		run.Field("TrackSound.Name", sound.Name, []string{"Clean", "Lead"}[index])
		run.Field("TrackSound.Label", sound.Label, []string{"A", "B"}[index])
		run.Field("TrackSound.Path", sound.Path, []string{"factory/clean", "factory/lead"}[index])
		run.Field("TrackSound.Role", sound.Role, []string{"main", "solo"}[index])
		run.Field("TrackSound.Program", sound.Program, []int32{42, 81}[index])
	}
	automation := track.SoundAutomations[0]
	run.Field("SoundAutomation.Bar", automation.Bar, 0)
	run.Field("SoundAutomation.Position", automation.Position, 0.5)
	run.Field("SoundAutomation.Sound", automation.Sound, 1)
	assertM05RSEFields(run, song.MasterEffect, track.Rse)

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

func assertM05RSEFields(run *semanticMatrixRun, master RseMasterEffect, track TrackRse) {
	run.Field("Song.MasterEffect", master, master)
	run.Field("RseMasterEffect.Equalizer", master.Equalizer, master.Equalizer)
	run.Field("RseMasterEffect.Volume", master.Volume, float32(0.75))
	run.Field("RseMasterEffect.Reverb", master.Reverb, float32(0.4))
	for _, equalizer := range []RseEqualizer{master.Equalizer, track.Equalizer} {
		run.Field("RseEqualizer.Knobs", equalizer.Knobs, equalizer.Knobs)
		run.Field("RseEqualizer.Gain", equalizer.Gain, equalizer.Gain)
	}
	run.Field("Track.Rse", track, track)
	run.Field("TrackRse.Instrument", track.Instrument, track.Instrument)
	run.Field("TrackRse.Equalizer", track.Equalizer, track.Equalizer)
	run.Field("TrackRse.Humanize", track.Humanize, uint8(3))
	run.Field("TrackRse.AutoAccentuation", track.AutoAccentuation, AccentuationStrong)
	run.Field("RseInstrument.EffectCategory", track.Instrument.EffectCategory, "Delay")
	run.Field("RseInstrument.Effect", track.Instrument.Effect, "Echo")
	run.Field("RseInstrument.Instrument", track.Instrument.Instrument, int16(7))
	run.Field("RseInstrument.Unknown", track.Instrument.Unknown, int16(8))
	run.Field("RseInstrument.SoundBank", track.Instrument.SoundBank, int16(9))
	run.Field("RseInstrument.EffectNumber", track.Instrument.EffectNumber, int16(10))
}
