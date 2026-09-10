// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"slices"
	"testing"
)

func TestPercussionUsesTrackChannel(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp5/nightwish.gp5")
	if len(song.Tracks) != 11 {
		t.Fatalf("tracks = %d, want 11", len(song.Tracks))
	}
	if song.Tracks[9].PercussionTrack {
		t.Error("track 9 uses a melodic channel")
	}
	drums := song.Tracks[10]
	if !drums.PercussionTrack {
		t.Error("track 10 uses the percussion channel")
	}
	if drums.ChannelIndex < 0 || song.Channels[drums.ChannelIndex].Channel != DefaultPercussionChannel {
		t.Errorf("track 10 channel index = %d, want the percussion channel", drums.ChannelIndex)
	}
}

func TestGP5PercussionGraceUsesDrumArticulation(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp5/motherload-percussion-grace.gp5")
	var track *Track
	for index := range song.Tracks {
		if song.Tracks[index].Name == "Percussion" {
			track = &song.Tracks[index]
			break
		}
	}
	if track == nil {
		t.Fatal("Percussion track not found")
	}
	if !track.PercussionTrack {
		t.Fatal("Percussion track is not marked as percussion")
	}

	var graceBars []int
	for measureIndex, measure := range track.Measures {
		for _, voice := range measure.Voices {
			for _, beat := range voice.Beats {
				for _, note := range beat.Notes {
					if len(note.Effect.Graces) == 0 {
						continue
					}
					if note.Value != 38 {
						t.Errorf("bar %d grace parent articulation = %d, want 38", measureIndex+1, note.Value)
					}
					for _, grace := range note.Effect.Graces {
						graceBars = append(graceBars, measureIndex+1)
						if grace.Fret != 38 {
							t.Errorf("bar %d grace articulation = %d, want 38", measureIndex+1, grace.Fret)
						}
						if grace.RawFret == nil || *grace.RawFret != 0 {
							t.Errorf("bar %d raw grace fret = %v, want 0", measureIndex+1, grace.RawFret)
						}
					}
				}
			}
		}
	}
	wantBars := []int{2, 2, 3, 26, 27, 51, 74, 75}
	if !slices.Equal(graceBars, wantBars) {
		t.Errorf("percussion grace bars = %v, want %v", graceBars, wantBars)
	}
}
func TestGPIFPercussionPreservesArticulations(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/percussion.gp")
	track := &song.Tracks[0]
	if len(track.PercussionArticulations) != 95 {
		t.Fatalf("percussion articulations = %d, want 95", len(track.PercussionArticulations))
	}
	if got := track.Staves[0].StandardNotationLineCount; got != 1 {
		t.Errorf("standard notation line count = %d, want 1", got)
	}

	hit := track.PercussionArticulations[68]
	returned := track.PercussionArticulations[69]
	if hit.ElementName != "Cabasa" || hit.ElementType != "cabasa" || hit.Name != "Cabasa (hit)" {
		t.Errorf("hit identity = %#v", hit)
	}
	if hit.StaffLine != 0 || hit.NoteheadDefault != "noteheadBlack" || hit.NoteheadHalf != "noteheadHalf" || hit.NoteheadWhole != "noteheadWhole" {
		t.Errorf("hit notation = %#v", hit)
	}
	if hit.TechniquePlacement != "outside" || hit.TechniqueSymbol != "" || !slices.Equal(hit.InputMIDINumbers, []int{69}) || hit.OutputRSESound != "hand.hit.hit" || hit.OutputMIDINumber != 69 {
		t.Errorf("hit playback = %#v", hit)
	}
	if returned.ElementName != "Cabasa" || returned.ElementType != "cabasa" || returned.Name != "Cabasa (return)" {
		t.Errorf("return identity = %#v", returned)
	}
	if returned.StaffLine != 0 || returned.TechniquePlacement != "outside" || returned.TechniqueSymbol != "stringsUpBow" || !slices.Equal(returned.InputMIDINumbers, []int{117}) || returned.OutputRSESound != "hand.hit.return" || returned.OutputMIDINumber != 69 {
		t.Errorf("return metadata = %#v", returned)
	}

	var notes []Note
	for _, measure := range track.Staves[0].Measures {
		for _, voice := range measure.Voices {
			for _, beat := range voice.Beats {
				notes = append(notes, beat.Notes...)
			}
		}
	}
	if len(notes) < 2 {
		t.Fatalf("notes = %d, want at least 2", len(notes))
	}
	if !notes[0].HasPercussionArticulation || notes[0].Value != 69 || notes[0].PercussionArticulation != 68 {
		t.Errorf("first note value/articulation/presence = %d/%d/%t, want 69/68/true", notes[0].Value, notes[0].PercussionArticulation, notes[0].HasPercussionArticulation)
	}
	if !notes[1].HasPercussionArticulation || notes[1].Value != 117 || notes[1].PercussionArticulation != 69 {
		t.Errorf("second note value/articulation/presence = %d/%d/%t, want 117/69/true", notes[1].Value, notes[1].PercussionArticulation, notes[1].HasPercussionArticulation)
	}
}

func TestGPIFPitchedTrackHasNoPercussionDefinitions(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/tuning.gp")
	for trackIndex, track := range song.Tracks {
		if !track.PercussionTrack && len(track.PercussionArticulations) != 0 {
			t.Fatalf("pitched track %d has %d percussion definitions", trackIndex, len(track.PercussionArticulations))
		}
	}
}

func TestGP5TrackMixerUsesNormalizedMidiRange(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp5/Demo v5.gp5")
	if len(song.Tracks) < 2 {
		t.Fatalf("tracks = %d, want at least 2", len(song.Tracks))
	}

	rhythm := song.Channels[song.Tracks[0].ChannelIndex]
	solo := song.Channels[song.Tracks[1].ChannelIndex]
	if rhythm.Volume != 87 || rhythm.Balance != 64 {
		t.Errorf("rhythm channel = %#v, want volume 87 and balance 64", rhythm)
	}
	if solo.Volume != 119 || solo.Balance != 64 {
		t.Errorf("solo channel = %#v, want volume 119 and balance 64", solo)
	}
}

func TestGP5PlaybackChannelsUseAuthoredPairs(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp5/Demo v5.gp5")
	want := [][2]uint8{{0, 1}, {4, 3}, {12, 7}, {8, 5}, {9, 9}}
	if len(song.Tracks) != len(want) {
		t.Fatalf("tracks = %d, want %d", len(song.Tracks), len(want))
	}
	for index, track := range song.Tracks {
		channel := song.Channels[track.ChannelIndex]
		if channel.Channel != want[index][0] || channel.EffectChannel != want[index][1] {
			t.Errorf("track %d channels = %d/%d, want %d/%d", index, channel.Channel, channel.EffectChannel, want[index][0], want[index][1])
		}
	}
}

func TestGP5LastExplicitNoteDynamicAppliesToBeat(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp5/Demo v5.gp5")
	beat := song.Tracks[0].Measures[0].Voices[0].Beats[0]
	if len(beat.Notes) != 3 {
		t.Fatalf("opening beat notes = %d, want 3", len(beat.Notes))
	}
	if beat.Notes[0].Velocity != Forte || beat.Dynamics != MinVelocity+VelocityIncrement*7 {
		t.Errorf("opening note/beat dynamics = %d/%d, want %d/%d", beat.Notes[0].Velocity, beat.Dynamics, Forte, MinVelocity+VelocityIncrement*7)
	}
}

func TestGP6PercussionTrack(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp6/full-song.gpx")
	if len(song.Tracks) != 11 {
		t.Fatalf("tracks = %d, want 11", len(song.Tracks))
	}
	drums := song.Tracks[10]
	if drums.Name != "Jukka" {
		t.Fatalf("track 10 name = %q, want %q", drums.Name, "Jukka")
	}
	if !drums.PercussionTrack {
		t.Error("GP6 drumkit track is not marked as percussion")
	}
}

func TestGPIFPercussionSignals(t *testing.T) {
	tests := []struct {
		name  string
		track gpifTrack
		want  bool
	}{
		{name: "instrument set", track: gpifTrack{InstrumentSet: &gpifInstrumentSet{Type: "drums"}}, want: true},
		{name: "GP6 instrument", track: gpifTrack{Instrument: &gpifInstrument{Ref: "drmkt"}}, want: true},
		{name: "percussion channel", track: gpifTrack{GeneralMidi: &gpifGeneralMidi{PrimaryChannel: 9}}, want: true},
		{name: "melodic channel", track: gpifTrack{GeneralMidi: &gpifGeneralMidi{PrimaryChannel: 8}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.track.isPercussionTrack(); got != test.want {
				t.Errorf("isPercussionTrack() = %t, want %t", got, test.want)
			}
		})
	}
}
