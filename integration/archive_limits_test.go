// SPDX-License-Identifier: MIT

package integration_test

import (
	"archive/zip"
	"bytes"
	"errors"
	"strings"
	"testing"

	guitarpro "github.com/CaliLuke/go-guitar-pro"
)

func TestParseRejectsOversizedArchiveMembers(t *testing.T) {
	for _, member := range []struct {
		name string
		size uint64
	}{
		{"Content/score.gpif", 16<<20 + 1},
		{"Content/BinaryStylesheet", 1<<20 + 1},
		{"Content/PartConfiguration", 1<<20 + 1},
		{"Content/LayoutConfiguration", 1<<20 + 1},
		{"Content/Assets/audio.bin", 256<<20 + 1},
	} {
		t.Run(member.name, func(t *testing.T) {
			var buffer bytes.Buffer
			archive := zip.NewWriter(&buffer)
			if member.name != "Content/score.gpif" {
				writer, err := archive.Create("Content/score.gpif")
				if err != nil {
					t.Fatal(err)
				}
				_, err = writer.Write([]byte(`<GPIF><BackingTrack><Enabled>true</Enabled><Source>Local</Source><AssetId>audio</AssetId><FramePadding>0</FramePadding></BackingTrack><Assets><Asset id="audio"><EmbeddedFilePath>Content/Assets/audio.bin</EmbeddedFilePath></Asset></Assets></GPIF>`))
				if err != nil {
					t.Fatal(err)
				}
			}
			// A declared size alone must be rejected, before opening or allocating
			// the member. No large payload is needed to exercise this boundary.
			_, err := archive.CreateRaw(&zip.FileHeader{
				Name: member.name, Method: zip.Store, UncompressedSize64: member.size,
			})
			if err != nil {
				t.Fatal(err)
			}
			if closeErr := archive.Close(); closeErr != nil {
				t.Fatal(closeErr)
			}
			_, err = guitarpro.Parse(buffer.Bytes())
			var parseErr *guitarpro.ParseError
			if !errors.As(err, &parseErr) || !strings.Contains(err.Error(), "limit") {
				t.Fatalf("oversized %s: error = %v, want size-limit ParseError", member.name, err)
			}
		})
	}
}
