// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/binary"
	"slices"
	"testing"
)

func TestGPIFTempoAutomations(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp8/beat-tempo-change.gp")
	if song.Tempo != 120 {
		t.Errorf("tempo = %d, want 120", song.Tempo)
	}
	if len(song.TempoAutomations) != 9 {
		t.Fatalf("tempo automations = %d, want 9", len(song.TempoAutomations))
	}
	first := song.TempoAutomations[0]
	if first.Bar != 0 || first.Position != 0 || first.Tempo != 120 {
		t.Errorf("first tempo automation = %#v", first)
	}
}

func TestGPIFTrackMixerState(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/track-balance.gp")
	if len(song.Tracks) < 2 || len(song.Channels) < 2 {
		t.Fatalf("tracks/channels = %d/%d, want at least 2", len(song.Tracks), len(song.Channels))
	}
	first := song.Channels[song.Tracks[0].ChannelIndex]
	second := song.Channels[song.Tracks[1].ChannelIndex]
	if first.Channel != 0 || first.EffectChannel != 1 || second.Channel != 2 || second.EffectChannel != 3 {
		t.Errorf("GPIF MIDI channels = [%d/%d %d/%d], want [0/1 2/3]", first.Channel, first.EffectChannel, second.Channel, second.EffectChannel)
	}
	if first.Volume != 101 || first.Balance != 0 || second.Balance != 32 {
		t.Errorf("GPIF channel strip = %#v %#v", first, second)
	}
	if len(song.VolumeAutomations) < 2 {
		t.Fatalf("volume automations = %d, want at least 2", len(song.VolumeAutomations))
	}
	if got := song.VolumeAutomations[0]; got.Track != 0 || got.Bar != 0 || got.Position != 0 || got.Value != 0.72 || got.Linear {
		t.Errorf("first volume automation = %#v", got)
	}
}

func TestGPIFTrackTunings(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/multi-track.gp")
	if len(song.Tracks) < 5 {
		t.Fatalf("tracks = %d, want at least 5", len(song.Tracks))
	}
	want := []GuitarString{{1, 62}, {2, 57}, {3, 53}, {4, 48}, {5, 43}, {6, 36}}
	if got := song.Tracks[3].Strings; !slices.Equal(got, want) {
		t.Errorf("Drop C tuning = %#v, want %#v", got, want)
	}
}

func TestGPIFNormalizesStringNumbersAndOpenFrets(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/brush.gp")
	notes := song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes
	if len(notes) < 3 {
		t.Fatalf("opening chord notes = %d, want at least 3", len(notes))
	}
	if got := notes[0]; got.String != 1 || got.Value != 0 {
		t.Errorf("highest open string = string %d fret %d, want string 1 fret 0", got.String, got.Value)
	}
	if got := notes[1]; got.String != 2 || got.Value != 1 {
		t.Errorf("second string note = string %d fret %d, want string 2 fret 1", got.String, got.Value)
	}
	if got := notes[2]; got.String != 3 || got.Value != 0 {
		t.Errorf("third open string = string %d fret %d, want string 3 fret 0", got.String, got.Value)
	}
}

func TestGPIFTrackMuteState(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/dead-slap.gp")
	muted := 0
	for _, track := range song.Tracks {
		if track.Mute {
			muted++
		}
	}
	if muted != 1 {
		t.Errorf("muted tracks = %d, want 1", muted)
	}
}

func TestGPIFUsesOpeningTempoAutomation(t *testing.T) {
	song := &Song{Tempo: 120}
	gpifReadTempoAutomations([]gpifAutomation{
		{Type: "Tempo", Bar: 4, Value: gpifAutomationValue{Text: "180 2"}},
		{Type: "Tempo", Bar: 0, Position: 0.25, Value: gpifAutomationValue{Text: "90 2"}, Text: "Slow"},
		{Type: "Tempo", Bar: 0, Value: gpifAutomationValue{Text: "140 2"}, Text: "Allegro"},
		{Type: "Tempo", Bar: 2, Value: gpifAutomationValue{Text: "invalid"}},
	}, song, nil)
	if song.Tempo != 140 || song.TempoName != "Allegro" {
		t.Errorf("opening tempo = %d %q, want 140 %q", song.Tempo, song.TempoName, "Allegro")
	}
	if len(song.TempoAutomations) != 3 {
		t.Errorf("tempo automations = %d, want 3", len(song.TempoAutomations))
	}
}

func TestGPIFAppliesTempoReferenceUnits(t *testing.T) {
	values := []string{"100 1", "100 2", "100 3", "100 4", "100 5", "100", "100 invalid", "100 0", "100 6", "100 3x", "100 3.5", "81.5 3"}
	want := []float64{50, 100, 150, 200, 300, 50, 100, 100, 100, 150, 150, 122.25}
	automations := make([]gpifAutomation, 0, len(values))
	for index, value := range values {
		automations = append(automations, gpifAutomation{
			Type: "Tempo", Bar: index, Value: gpifAutomationValue{Text: value},
		})
	}
	song := &Song{}
	gpifReadTempoAutomations(automations, song, nil)
	if len(song.TempoAutomations) != len(want) {
		t.Fatalf("tempo automations = %d, want %d", len(song.TempoAutomations), len(want))
	}
	for index := range want {
		if got := song.TempoAutomations[index].Tempo; got != want[index] {
			t.Errorf("tempo %q = %v, want %v", values[index], got, want[index])
		}
	}
	if song.Tempo != 50 {
		t.Errorf("initial tempo = %d, want 50", song.Tempo)
	}
}

func TestMixTableTempoAboveByte(t *testing.T) {
	data := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	data = binary.LittleEndian.AppendUint32(data, 300)
	data = append(data, 0, 0)
	song := &Song{Version: Version{Number: [3]byte{4, 0, 0}}}
	change, err := song.readMixTableChange(newCursor(data))
	if err != nil {
		t.Fatal(err)
	}
	if change.Tempo == nil || change.Tempo.Value != 300 {
		t.Errorf("tempo = %#v, want 300", change.Tempo)
	}
}
