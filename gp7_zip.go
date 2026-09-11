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

const maxGPIFXMLSize = 16 << 20

func parseGP7ZipWithContext(data []byte, context *parseContext) (*Song, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("opening ZIP: %w", err)
	}

	var song *Song
	for _, f := range r.File {
		if filepath.Base(f.Name) == "score.gpif" {
			if f.UncompressedSize64 > maxGPIFXMLSize {
				return nil, fmt.Errorf("score.gpif size %d exceeds %d-byte limit", f.UncompressedSize64, maxGPIFXMLSize)
			}
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("opening score.gpif: %w", err)
			}
			gpifData, err := io.ReadAll(io.LimitReader(rc, maxGPIFXMLSize+1))
			if err != nil {
				_ = rc.Close()
				return nil, fmt.Errorf("reading score.gpif: %w", err)
			}
			if closeErr := rc.Close(); closeErr != nil {
				return nil, fmt.Errorf("closing score.gpif: %w", closeErr)
			}
			if len(gpifData) > maxGPIFXMLSize {
				return nil, fmt.Errorf("score.gpif exceeds %d-byte limit", maxGPIFXMLSize)
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
		if filepath.Base(f.Name) != "LayoutConfiguration" || f.UncompressedSize64 > 1<<20 {
			continue
		}
		rc, openErr := f.Open()
		if openErr != nil {
			return nil, fmt.Errorf("opening LayoutConfiguration: %w", openErr)
		}
		layout, readErr := io.ReadAll(io.LimitReader(rc, 1<<20))
		closeErr := rc.Close()
		if readErr != nil {
			return nil, fmt.Errorf("reading LayoutConfiguration: %w", readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("closing LayoutConfiguration: %w", closeErr)
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
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("opening backing track %q: %w", f.Name, err)
		}
		song.BackingTrack.AudioData, err = io.ReadAll(rc)
		if err != nil {
			_ = rc.Close()
			return nil, fmt.Errorf("reading backing track %q: %w", f.Name, err)
		}
		if err := rc.Close(); err != nil {
			return nil, fmt.Errorf("closing backing track %q: %w", f.Name, err)
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
		if file.UncompressedSize64 > uint64(limit) {
			return fmt.Errorf("%s size %d exceeds %d-byte limit", name, file.UncompressedSize64, limit)
		}
		stream, err := file.Open()
		if err != nil {
			return fmt.Errorf("opening %s: %w", name, err)
		}
		data, readErr := io.ReadAll(io.LimitReader(stream, int64(limit)+1))
		closeErr := stream.Close()
		if readErr != nil {
			return fmt.Errorf("reading %s: %w", name, readErr)
		}
		if closeErr != nil {
			return fmt.Errorf("closing %s: %w", name, closeErr)
		}
		return apply(song, data)
	}
	return nil
}
