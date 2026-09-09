// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"slices"
	"strings"
	"testing"
)

func TestGPIFChordCollectionAndBaseFretRoundTrip(t *testing.T) {
	firstFret := uint8(5)
	song := syntheticGP8Song()
	song.Tracks[0].Measures[0].Voices[0].Beats[1].Effect.Chord = &Chord{
		Name: "Base-fret chord", FirstFret: &firstFret, Strings: []int8{5}, Length: 1,
	}
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	data = rewriteConformanceGPIF(t, data, func(gpif string) string {
		return moveFirstChordCollectionToTrack(t, gpif)
	})

	result, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	assertFirstChordBaseFret(t, result, firstFret, 5)

	reexported, err := Export(result, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	roundTripped, err := Parse(reexported)
	if err != nil {
		t.Fatal(err)
	}
	assertFirstChordBaseFret(t, roundTripped, firstFret, 5)
}

func moveFirstChordCollectionToTrack(t *testing.T, gpif string) string {
	t.Helper()
	start := strings.Index(gpif, `<Property name="DiagramCollection">`)
	if start < 0 {
		t.Fatal("GPIF has no diagram collection")
	}
	end := strings.Index(gpif[start:], "</Property>")
	if end < 0 {
		t.Fatal("GPIF diagram collection is not closed")
	}
	end += start + len("</Property>")
	collection := strings.Replace(gpif[start:end], `name="DiagramCollection"`, `name="ChordCollection"`, 1)
	gpif = gpif[:start] + gpif[end:]
	staves := strings.Index(gpif, "<Staves>")
	if staves < 0 {
		t.Fatal("GPIF track has no staves")
	}
	return gpif[:staves] + "<Properties>" + collection + "</Properties>" + gpif[staves:]
}

func assertFirstChordBaseFret(t *testing.T, song *Song, firstFret uint8, fret int8) {
	t.Helper()
	chord := song.Tracks[0].Measures[0].Voices[0].Beats[1].Effect.Chord
	if chord == nil || chord.FirstFret == nil || *chord.FirstFret != firstFret || !slices.Equal(chord.Strings, []int8{fret}) {
		t.Fatalf("chord = %#v, want first fret %d and strings [%d]", chord, firstFret, fret)
	}
}
