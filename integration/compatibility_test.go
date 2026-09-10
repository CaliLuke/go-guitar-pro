// SPDX-License-Identifier: MIT

package integration_test

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	guitarpro "github.com/CaliLuke/go-guitar-pro"
)

func TestParseFixtures(t *testing.T) {
	err := filepath.WalkDir("../testdata", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !isGuitarProFixture(path) {
			return nil
		}

		t.Run(path, func(t *testing.T) {
			song, err := guitarpro.ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile() error = %v", err)
			}
			if len(song.Tracks) == 0 {
				t.Error("ParseFile() returned a song without tracks")
			}
			if len(song.MeasureHeaders) == 0 {
				t.Error("ParseFile() returned a song without measure headers")
			}
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walking fixtures: %v", err)
	}
}

func TestParseRejectsShortData(t *testing.T) {
	if _, err := guitarpro.Parse([]byte("GP")); err == nil {
		t.Fatal("Parse() error = nil, want an invalid-data error")
	}
}

func isGuitarProFixture(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".gp", ".gp3", ".gp4", ".gp5", ".gpx":
		return true
	default:
		return false
	}
}
