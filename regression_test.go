// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/binary"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
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
