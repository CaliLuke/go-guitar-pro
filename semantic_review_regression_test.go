// SPDX-License-Identifier: MIT

package goguitarpro

import "testing"

func TestReviewedM01M06LossesAreReported(t *testing.T) {
	tests := []struct {
		name string
		code string
		edit func(*Song)
	}{
		{name: "empty notice", code: "gp8.omit.empty-notice", edit: func(song *Song) { song.Notice = []string{""} }},
		{name: "reverse double bar", code: "gp8.normalize.measure-double-bar-authority", edit: func(song *Song) { song.Tracks[0].Measures[2].HasDoubleBar = false }},
		{name: "custom string number", code: "gp8.normalize.string-number", edit: func(song *Song) { song.Tracks[0].Strings[0].Number = 6 }},
		{name: "mute and solo", code: "gp8.normalize.playback-state", edit: func(song *Song) { song.Tracks[0].Mute, song.Tracks[0].Solo = true, true }},
		{name: "unbound channel", code: "gp8.normalize.channel-binding", edit: func(song *Song) { song.Tracks[0].ChannelIndex = -1 }},
		{name: "different effect port", code: "gp8.normalize.effect-channel-port", edit: func(song *Song) { song.Channels[0].Channel, song.Channels[0].EffectChannel = 1, 18 }},
		{name: "opening tempo conflict", code: "gp8.normalize.tempo-automation-authority", edit: func(song *Song) {
			song.Tempo = 120
			song.InitialTempo = KnownSourceValue(BPM(120))
			song.TempoAutomations = []TempoAutomation{{Bar: 0, Tempo: 90}}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			song := syntheticGP8Song()
			test.edit(song)
			report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
			if !hasExportReportEntry(report, test.code) {
				t.Fatalf("report = %#v, want %s", report.Entries, test.code)
			}
		})
	}
}

func TestReviewedM03ExportPreservesInteriorVoiceSlot(t *testing.T) {
	song := semanticValidPitchedGP8Song(t)
	track := &song.Tracks[0]
	track.Settings.Notation = true
	measure := &track.Measures[0]
	populated := measure.Voices[0]
	measure.Voices = []Voice{populated, {}, populated, populated}
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	voices := roundTrip.Tracks[0].Staves[0].Measures[0].Voices
	if len(voices) != 4 || len(voices[0].Beats) == 0 || len(voices[1].Beats) != 0 || len(voices[2].Beats) == 0 || len(voices[3].Beats) == 0 {
		t.Fatalf("round-trip voices = %#v, want populated, empty, populated, populated", voices)
	}
}
