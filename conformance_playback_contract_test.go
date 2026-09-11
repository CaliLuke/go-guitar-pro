// SPDX-License-Identifier: MIT

package goguitarpro

import "testing"

func TestConformanceMIDIRoutingTargetLimits(t *testing.T) {
	runMIDIRoutingTargetLimits(newConformanceRun(t), false)
}

func runConformanceMIDIRoutingTargetLimits(run *conformanceRun) {
	runMIDIRoutingTargetLimits(run, false)
}

func TestAlphaTabMIDIRoutingTargetLimits(t *testing.T) {
	requireAlphaTabConformance(t)
	runMIDIRoutingTargetLimits(newConformanceRun(t), true)
}

func runMIDIRoutingTargetLimits(run *conformanceRun, oracle bool) {
	t := run.t
	for _, test := range []struct {
		name       string
		effect     uint8
		mute, solo bool
		state      string
		codes      []string
	}{
		{name: "port", effect: 18, state: "Default", codes: []string{"gp8.normalize.effect-channel-port"}},
		{name: "state", effect: 2, mute: true, solo: true, state: "Mute", codes: []string{"gp8.normalize.playback-state"}},
		{name: "both", effect: 18, mute: true, solo: true, state: "Mute", codes: []string{"gp8.normalize.playback-state", "gp8.normalize.effect-channel-port"}},
		{name: "default control", effect: 2, state: "Default", codes: []string{}},
		{name: "mute control", effect: 2, mute: true, state: "Mute", codes: []string{}},
		{name: "solo control", effect: 2, solo: true, state: "Solo", codes: []string{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := consumerLimitSong(t)
			song.Channels[0].Channel, song.Channels[0].EffectChannel = 1, test.effect
			song.Tracks[0].Mute, song.Tracks[0].Solo = test.mute, test.solo
			if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
				t.Fatalf("valid routing score: %#v", diagnostics)
			}
			data, report := assertConsumerLossPolicy(t, song, test.codes)
			run.Report("M05-ROUTING-LIMITS", reportCodes(report), test.codes)
			for _, entry := range report.Entries {
				if entry.Location != (ScoreLocation{Track: 0}) || entry.Disposition != ExportDispositionNormalized {
					t.Fatalf("unscoped routing report: %#v", entry)
				}
			}
			wire := conformanceSingleWireTrack(t, data)
			run.Wire("gpifMidiConnection.Port", wire.MidiConnection.Port, 0)
			run.Wire("gpifMidiConnection.PrimaryChannel", wire.MidiConnection.PrimaryChannel, 1)
			run.Wire("gpifMidiConnection.SecondaryChannel", wire.MidiConnection.SecondaryChannel, 2)
			run.Wire("gpifTrack.PlaybackState", wire.PlaybackState, test.state)
			parsed, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			run.Preserved("MidiChannel.Channel", parsed.Channels[0].Channel, uint8(1))
			run.Normalized("MidiChannel.EffectChannel", parsed.Channels[0].EffectChannel, uint8(2))
			run.Normalized("Track.Mute", parsed.Tracks[0].Mute, test.mute)
			run.Normalized("Track.Solo", parsed.Tracks[0].Solo, test.solo && !test.mute)
			if oracle {
				var facts []conformancePlaybackRouting
				readAlphaTabOracleFacts(t, "--playback-routing", writeConformanceFixture(t, data), &facts)
				want := []conformancePlaybackRouting{{Track: 0, Port: 0, PrimaryChannel: 1, SecondaryChannel: 2, IsMute: test.mute, IsSolo: test.solo && !test.mute}}
				if len(facts) != 1 || facts[0] != want[0] {
					t.Fatalf("raw AlphaTab routing = %#v, want %#v", facts, want)
				}
			}
		})
	}
}

type conformancePlaybackRouting struct {
	Track            int  `json:"track"`
	Port             int  `json:"port"`
	PrimaryChannel   int  `json:"primaryChannel"`
	SecondaryChannel int  `json:"secondaryChannel"`
	IsMute           bool `json:"isMute"`
	IsSolo           bool `json:"isSolo"`
}

func TestAlphaTabMIDIRoutingPortControl(t *testing.T) {
	requireAlphaTabConformance(t)
	song := consumerLimitSong(t)
	song.Channels[0].Channel, song.Channels[0].EffectChannel = 18, 19
	data, _ := assertConsumerLossPolicy(t, song, []string{})
	var facts []conformancePlaybackRouting
	readAlphaTabOracleFacts(t, "--playback-routing", writeConformanceFixture(t, data), &facts)
	want := conformancePlaybackRouting{Track: 0, Port: 1, PrimaryChannel: 2, SecondaryChannel: 3}
	if len(facts) != 1 || facts[0] != want {
		t.Fatalf("raw same-port control = %#v, want %#v", facts, want)
	}
}
