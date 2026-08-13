// SPDX-License-Identifier: MIT

// Package goguitarpro parses Guitar Pro files from versions 3 through 8.
package goguitarpro

import (
	"fmt"
)

// ParseError reports an unsupported or damaged Guitar Pro file.
type ParseError struct {
	Err error
}

// Error returns the parser error message.
func (e *ParseError) Error() string {
	return "unparseable Guitar Pro file: " + e.Err.Error()
}

// Unwrap returns the underlying parser error.
func (e *ParseError) Unwrap() error {
	return e.Err
}

// Parse detects the file format. Then Parse parses a supported Guitar Pro file.
// Parse supports GP3, GP4, GP5 (binary), GP6/GPX, GP7, and GP8.
// Parse returns a [ParseError] for unsupported or damaged input.
func Parse(data []byte) (song *Song, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			song, err = nil, &ParseError{Err: fmt.Errorf("reader failed on malformed input: %v", recovered)}
		}
	}()

	song, err = parse(data)
	if err != nil {
		return nil, &ParseError{Err: err}
	}
	return song, nil
}

func parse(data []byte) (*Song, error) {
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
