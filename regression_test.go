// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/binary"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestDurationTime(t *testing.T) {
	tests := []struct {
		name string
		in   Duration
		want uint32
	}{
		{"quarter", Duration{Value: 4, TupletEnters: 1, TupletTimes: 1}, 960},
		{"dotted quarter", Duration{Value: 4, Dotted: true, TupletEnters: 1, TupletTimes: 1}, 1440},
		{"double dotted quarter", Duration{Value: 4, DoubleDotted: true, TupletEnters: 1, TupletTimes: 1}, 1680},
		{"quarter triplet", Duration{Value: 4, TupletEnters: 3, TupletTimes: 2}, 640},
		{"unset value", Duration{TupletEnters: 1, TupletTimes: 1}, 0},
		{"unset tuplet", Duration{Value: 4}, 960},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.in.time(); got != test.want {
				t.Fatalf("time() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestTupletBeatsAdvanceByPlayedLength(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp5/tuplets.gp5")
	beats := song.Tracks[0].Measures[0].Voices[0].Beats
	if len(beats) < 3 {
		t.Fatalf("beats = %d, want at least 3", len(beats))
	}
	base := *beats[0].Start
	for index, beat := range beats[:3] {
		want := base + int64(index)*640
		if beat.Start == nil || *beat.Start != want {
			t.Errorf("beat %d starts at %v, want %d", index, beat.Start, want)
		}
	}
}

func TestMarkerTitlesExcludeLengthPrefix(t *testing.T) {
	markerCount := 0
	err := filepath.WalkDir("testdata", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !isGuitarProFixture(path) {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(path))
		if extension != ".gp3" && extension != ".gp4" && extension != ".gp5" {
			return nil
		}
		song := parseTestFixture(t, path)
		for _, header := range song.MeasureHeaders {
			if header.Marker == nil || header.Marker.Title == "" {
				continue
			}
			markerCount++
			title := header.Marker.Title
			if strings.ContainsRune(title, 0) {
				t.Errorf("%s: invalid marker title %q", path, title)
			}
			if title != "" && int(title[0]) == len(title)-1 {
				t.Errorf("%s: marker title contains its length prefix: %q", path, title)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if markerCount == 0 {
		t.Fatal("no marker titles in the compatibility corpus")
	}
}

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

	graceCount := 0
	for measureIndex, measure := range track.Measures {
		for _, voice := range measure.Voices {
			for _, beat := range voice.Beats {
				for _, note := range beat.Notes {
					if note.Effect.Grace == nil {
						continue
					}
					graceCount++
					if note.Value != 38 {
						t.Errorf("bar %d grace parent articulation = %d, want 38", measureIndex+1, note.Value)
					}
					if got := note.Effect.Grace.Fret; got != int8(note.Value) {
						t.Errorf("bar %d grace articulation = %d, want parent articulation %d", measureIndex+1, got, note.Value)
					}
				}
			}
		}
	}
	if graceCount != 8 {
		t.Errorf("percussion grace notes = %d, want 8", graceCount)
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
		{name: "instrument set", track: gpifTrack{InstrumentSet: gpifInstrumentSet{Type: "drums"}}, want: true},
		{name: "GP6 instrument", track: gpifTrack{Instrument: gpifInstrument{Ref: "drmkt"}}, want: true},
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
		{Type: "Tempo", Bar: 4, Value: "180 2"},
		{Type: "Tempo", Bar: 0, Position: 0.25, Value: "90 2", Text: "Slow"},
		{Type: "Tempo", Bar: 0, Value: "140 2", Text: "Allegro"},
		{Type: "Tempo", Bar: 2, Value: "invalid"},
	}, song)
	if song.Tempo != 140 || song.TempoName != "Allegro" {
		t.Errorf("opening tempo = %d %q, want 140 %q", song.Tempo, song.TempoName, "Allegro")
	}
	if len(song.TempoAutomations) != 3 {
		t.Errorf("tempo automations = %d, want 3", len(song.TempoAutomations))
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

func TestReadCountRejectsInvalidCounts(t *testing.T) {
	for _, data := range [][]byte{
		{0xff, 0xff, 0xff, 0xff, 0, 0},
		{0xff, 0xff, 0xff, 0x7f, 0, 0},
		{1, 0},
	} {
		if _, err := newCursor(data).readCount(2, "beat count"); err == nil {
			t.Error("readCount() error = nil")
		}
	}
}

func TestCorruptFilesReturnParseError(t *testing.T) {
	original, err := os.ReadFile("testdata/gp5/serenade.gp5")
	if err != nil {
		t.Fatal(err)
	}
	for _, fraction := range []int{2, 3, 4, 8, 16, 32, 64} {
		t.Run("truncated-1/"+strconv.Itoa(fraction), func(t *testing.T) {
			_, err := Parse(original[:len(original)/fraction])
			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("error = %v, want *ParseError", err)
			}
		})
	}
}

func TestGPXRejectsAbsurdDecompressedLength(t *testing.T) {
	data := []byte{0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0}
	if _, err := gpxDecompress(data); err == nil {
		t.Error("gpxDecompress() error = nil")
	}
}

func parseTestFixture(t *testing.T, path string) *Song {
	t.Helper()
	song, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile(%q): %v", path, err)
	}
	return song
}
