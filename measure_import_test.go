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
