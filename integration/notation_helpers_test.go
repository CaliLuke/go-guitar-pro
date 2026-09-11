// SPDX-License-Identifier: MIT

package integration_test

import (
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func notationPublicSong(t *testing.T) *gp.Song {
	t.Helper()
	song, err := gp.ParseFile("../testdata/gp8/section-track-names.gp")
	if err != nil {
		t.Fatal(err)
	}
	song.Version = gp.Version{}
	for i := range song.Tracks {
		song.Tracks[i].Settings = gp.TrackSettings{Notation: true, Tablature: true}
	}
	return song
}
