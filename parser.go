// SPDX-License-Identifier: MIT

// Package goguitarpro parses Guitar Pro files from versions 3 through 8.
package goguitarpro

import (
	"fmt"
)

// Parse detects the file format. Then Parse parses any supported Guitar Pro file.
// Parse supports GP3, GP4, GP5 (binary), GP6/GPX, GP7, and GP8.
func Parse(data []byte) (song *Song, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			song = nil
			err = fmt.Errorf("invalid Guitar Pro data: %v", recovered)
		}
	}()

	if len(data) < 4 {
		return nil, fmt.Errorf("data too short to detect format")
	}

	// Check magic bytes
	header := string(data[:4])
	switch {
	case header == "BCFZ" || header == "BCFS":
		// GP6 (GPX container)
		return parseGPX(data)
	case data[0] == 'P' && data[1] == 'K' && data[2] == 0x03 && data[3] == 0x04:
		// GP7/8 (ZIP archive)
		return parseGP7Zip(data)
	default:
		// Try binary GP3/4/5 (31-byte version header)
		return parseBinaryGP(data)
	}
}

// parseBinaryGP parses GP3, GP4, and GP5 binary format files.
func parseBinaryGP(data []byte) (*Song, error) {
	c := newCursor(data)
	song := &Song{
		Tempo:     120,
		TempoName: "Moderate",
	}
	if err := song.readBinary(c); err != nil {
		return nil, fmt.Errorf("parsing binary GP: %w", err)
	}
	return song, nil
}
