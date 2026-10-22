// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	maxGPIFXMLSize             = 16 << 20
	maxBackingTrackSize        = 256 << 20
	maxLayoutConfigurationSize = 1 << 20
)

func parseGP7ZipWithContext(data []byte, context *parseContext) (*Song, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("opening ZIP: %w", err)
	}

	var song *Song
	for _, f := range r.File {
		if filepath.Base(f.Name) == "score.gpif" {
			gpifData, readErr := readGP7Member(f, maxGPIFXMLSize)
			if readErr != nil {
				return nil, readErr
			}
			song, err = parseGPIFWithContext(gpifData, context)
			if err != nil {
				return nil, err
			}
			break
		}
	}
	if song == nil {
		return nil, fmt.Errorf("no score.gpif found in ZIP archive")
	}
	for _, configuration := range []struct {
		name  string
		limit int
		apply func(*Song, []byte) error
	}{
		{"BinaryStylesheet", maxBinaryStylesheetSize, applyBinaryStylesheet},
		{"PartConfiguration", maxPartConfigurationSize, applyPartConfiguration},
	} {
		if applyErr := applyGP7Configuration(r, song, configuration.name, configuration.limit, configuration.apply); applyErr != nil {
			return nil, applyErr
		}
	}

	for _, f := range r.File {
		if filepath.Base(f.Name) != "LayoutConfiguration" {
			continue
		}
		layout, readErr := readGP7Member(f, maxLayoutConfigurationSize)
		if readErr != nil {
			return nil, readErr
		}
		if len(layout) >= 6+len(song.Tracks) && binary.BigEndian.Uint32(layout[:4]) == 4 {
			for index := range song.Tracks {
				song.Tracks[index].Visible = layout[6+index] != 0
			}
		}
		break
	}
	if song.BackingTrack == nil {
		return song, nil
	}
	if !song.BackingTrack.Enabled || !strings.EqualFold(song.BackingTrack.Source, "Local") {
		return song, nil
	}
	embeddedReferenceSource := diagnosticSource("GPIF.BackingTrack.EmbeddedFile.Reference", "score-core", ParseDiagnosticInvalidData)
	if song.BackingTrack.EmbeddedFilePath == "" {
		if song.BackingTrack.Enabled && strings.EqualFold(song.BackingTrack.Source, "Local") {
			context.add(embeddedReferenceSource, ParseDiagnostic{
				SourcePath: "/GPIF/Assets/Asset[@id=" + strconv.Quote(song.BackingTrack.AssetID) + "]/EmbeddedFilePath",
				ObjectID:   song.BackingTrack.AssetID,
				Reason:     "enabled local backing track has no embedded file path",
			})
		}
		return song, nil
	}

	foundBackingTrack := false
	for _, f := range r.File {
		if f.Name != song.BackingTrack.EmbeddedFilePath {
			continue
		}
		foundBackingTrack = true
		song.BackingTrack.AudioData, err = readGP7Member(f, maxBackingTrackSize)
		if err != nil {
			return nil, err
		}
		if len(song.BackingTrack.AudioData) == 0 {
			context.add(diagnosticSource("GPIF.BackingTrack.EmbeddedFile.Empty", "score-core", ParseDiagnosticInvalidData), ParseDiagnostic{
				SourcePath: "/GPIF/Assets/Asset[@id=" + strconv.Quote(song.BackingTrack.AssetID) + "]/EmbeddedFilePath",
				ObjectID:   song.BackingTrack.AssetID,
				Reason:     fmt.Sprintf("embedded backing track %q contains no audio bytes", f.Name),
			})
		}
		break
	}
	if !foundBackingTrack {
		context.add(embeddedReferenceSource, ParseDiagnostic{
			SourcePath: "/GPIF/Assets/Asset[@id=" + strconv.Quote(song.BackingTrack.AssetID) + "]/EmbeddedFilePath",
			ObjectID:   song.BackingTrack.AssetID,
			Reason:     fmt.Sprintf("embedded backing track %q is absent from the archive", song.BackingTrack.EmbeddedFilePath),
		})
	}
	return song, nil
}

// applyGP7Configuration reads one bounded binary metadata member.
func applyGP7Configuration(archive *zip.Reader, song *Song, name string, limit int, apply func(*Song, []byte) error) error {
	for _, file := range archive.File {
		if filepath.Base(file.Name) != name {
			continue
		}
		data, err := readGP7Member(file, limit)
		if err != nil {
			return err
		}
		return apply(song, data)
	}
	return nil
}

// readGP7Member enforces the same allocation and integrity boundary for every
// ZIP payload the parser consumes, including optional metadata and audio.
func readGP7Member(file *zip.File, limit int) ([]byte, error) {
	if file.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("%s size %d exceeds %d-byte limit", file.Name, file.UncompressedSize64, limit)
	}
	stream, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", file.Name, err)
	}
	data, readErr := io.ReadAll(io.LimitReader(stream, int64(limit)+1))
	closeErr := stream.Close()
	if readErr != nil {
		return nil, fmt.Errorf("reading %s: %w", file.Name, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("closing %s: %w", file.Name, closeErr)
	}
	if len(data) > limit {
		return nil, fmt.Errorf("%s exceeds %d-byte limit", file.Name, limit)
	}
	return data, nil
}
