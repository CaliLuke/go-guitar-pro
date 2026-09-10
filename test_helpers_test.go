// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustReadFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func isGuitarProFixture(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".gp", ".gp3", ".gp4", ".gp5", ".gpx":
		return true
	default:
		return false
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
