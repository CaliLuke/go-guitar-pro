// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"path/filepath"
)

func parseGP7ZipWithContext(data []byte, context *parseContext) (*Song, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("opening ZIP: %w", err)
	}

	var song *Song
	for _, f := range r.File {
		if filepath.Base(f.Name) == "score.gpif" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("opening score.gpif: %w", err)
			}
			gpifData, err := io.ReadAll(rc)
			if err != nil {
				_ = rc.Close()
				return nil, fmt.Errorf("reading score.gpif: %w", err)
			}
			if closeErr := rc.Close(); closeErr != nil {
				return nil, fmt.Errorf("closing score.gpif: %w", closeErr)
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
	if song.BackingTrack == nil || song.BackingTrack.EmbeddedFilePath == "" {
		return song, nil
	}

	for _, f := range r.File {
		if f.Name != song.BackingTrack.EmbeddedFilePath {
			continue
		}
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
		break
	}
	return song, nil
}
