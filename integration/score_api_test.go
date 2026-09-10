// SPDX-License-Identifier: MIT

package integration_test

import (
	"os"
	"testing"

	guitarpro "github.com/CaliLuke/go-guitar-pro"
)

func TestParseScoreUsesOneSemanticModel(t *testing.T) {
	score, err := guitarpro.ParseScore(mustReadFixture(t, "../testdata/gp7/percussion.gp"))
	if err != nil {
		t.Fatal(err)
	}
	if len(score.Tracks[0].Staves) == 0 || len(score.Tracks[0].PercussionArticulations) == 0 {
		t.Fatalf("semantic score omitted staff or percussion identity: %#v", score.Tracks[0])
	}
	beat := score.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0]
	if beat.ExactStart == nil {
		t.Fatal("semantic score omitted exact finalized timing")
	}
	legacy := requireSong(score)
	if legacy != score {
		t.Fatal("Score and Song created independent writable trees")
	}
}

func requireSong(song *guitarpro.Song) *guitarpro.Song {
	return song
}

func mustReadFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
