// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

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

func TestLegacyRepeatCountsNormalizeAtReaderBoundary(t *testing.T) {
	tests := []struct {
		name    string
		version byte
		wire    byte
		want    uint8
	}{
		{name: "GP3 ordinary", version: 3, wire: 1, want: 2},
		{name: "GP4 maximum", version: 4, wire: 127, want: 128},
		{name: "GP5 ordinary", version: 5, wire: 2, want: 2},
		{name: "GP5 maximum", version: 5, wire: 128, want: 128},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			song := Song{Version: Version{Number: [3]byte{test.version}}}
			data := []byte{0x08, test.wire}
			var (
				header MeasureHeader
				err    error
			)
			if test.version == 5 {
				data = append(data, 0, 0)
				header, err = song.readMeasureHeaderV5(newCursor(data), 1, nil)
			} else {
				header, err = song.readMeasureHeader(newCursor(data), 1, nil)
			}
			if err != nil {
				t.Fatal(err)
			}
			if header.RepeatCount != test.want {
				t.Fatalf("repeat count = %d, want %d total passes", header.RepeatCount, test.want)
			}
		})
	}

	for _, test := range []struct {
		version byte
		wire    byte
	}{
		{version: 4, wire: 128},
		{version: 5, wire: 129},
	} {
		song := Song{Version: Version{Number: [3]byte{test.version}}}
		if _, err := song.readMeasureHeader(newCursor([]byte{0x08, test.wire}), 1, nil); err == nil || !strings.Contains(err.Error(), "exceeds supported maximum 128") {
			t.Fatalf("GP%d repeat overflow error = %v", test.version, err)
		}
	}
}

func TestLegacyTrackAndMeasureNumbersAreOneBased(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp5/layout-configuration-multi-track-1.gp5")
	for trackIndex := range song.Tracks {
		track := &song.Tracks[trackIndex]
		if track.Number != int32(trackIndex+1) {
			t.Errorf("track %d number = %d, want %d", trackIndex, track.Number, trackIndex+1)
		}
		for measureIndex, measure := range track.Measures {
			if measure.Number != measureIndex+1 {
				t.Errorf("track %d measure %d number = %d, want %d", trackIndex, measureIndex, measure.Number, measureIndex+1)
			}
		}
	}
}
