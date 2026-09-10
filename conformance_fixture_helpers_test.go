// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func conformanceExportSong() *Song {
	song := syntheticGP8Song()
	track := &song.Tracks[0]
	track.Name = "Guitar"
	track.CapoFret = 2
	track.PercussionTrack = false
	track.Strings = []GuitarString{{Number: 1, Value: 64}, {Number: 2, Value: 59}, {Number: 3, Value: 55}, {Number: 4, Value: 50}, {Number: 5, Value: 45}, {Number: 6, Value: 40}}
	song.Channels[0].Channel = 0
	song.Channels[0].EffectChannel = 1
	song.Channels[0].Instrument = 25
	for measureIndex := range track.Measures {
		for voiceIndex := range track.Measures[measureIndex].Voices {
			for beatIndex := range track.Measures[measureIndex].Voices[voiceIndex].Beats {
				beat := &track.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex]
				for noteIndex := range beat.Notes {
					note := &beat.Notes[noteIndex]
					note.String = int8(noteIndex + 1)
					note.Value = int16(3 + noteIndex*2)
					for graceIndex := range note.Effect.Graces {
						note.Effect.Graces[graceIndex].Fret = 2
					}
				}
			}
		}
	}
	firstBeat := &track.Measures[0].Voices[0].Beats[0]
	firstBeat.Effect.Hairpin = HairpinDiminuendo
	harmonicFret := 2.4
	firstBeat.Notes[0].Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeArtificial, FretFloat: &harmonicFret}
	return song
}

func conformanceExportCase(t *testing.T, id string) *Song {
	t.Helper()
	switch id {
	case "gp8-semantic-export":
		return conformanceExportSong()
	case "gp8-percussion-export":
		return parseTestFixture(t, "testdata/gp7/percussion.gp")
	case "gp8-multi-staff-export":
		return parseTestFixture(t, "testdata/gp7/grand-staff.gp")
	default:
		t.Fatalf("export conformance case %q has no source builder", id)
		return nil
	}
}

func writeConformanceFixture(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "score.gp")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func conformanceGPIFArchive(t *testing.T, gpif string) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	entry, err := writer.Create("Content/score.gpif")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(gpif)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func conformanceSingleWireTrack(t *testing.T, data []byte) gpifTrack {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	if err := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Tracks.Tracks) != 1 {
		t.Fatalf("wire tracks = %d, want 1", len(document.Tracks.Tracks))
	}
	return document.Tracks.Tracks[0]
}

func rewriteConformanceGPIF(t interface {
	Helper()
	Fatal(args ...any)
}, data []byte, mutate func(string) string) []byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, file := range reader.File {
		input, openErr := file.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		contents, readErr := io.ReadAll(input)
		closeErr := input.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		if file.Name == "Content/score.gpif" {
			contents = []byte(mutate(string(contents)))
		}
		header := file.FileHeader
		header.Method = zip.Deflate
		entry, createErr := writer.CreateHeader(&header)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := entry.Write(contents); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func conformanceFixtureDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}
