// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"os"
)

// ParseFile reads a Guitar Pro file from the path parameter. Then ParseFile
// parses the file.
//
// ParseFile supports GP3, GP4, GP5, GP6/GPX, GP7, and GP8 files. When the file
// contents are already in memory, call Parse.
func ParseFile(path string) (*Song, error) {
	result, err := ParseFileWithOptions(path, ParseOptions{})
	if err != nil {
		return nil, err
	}
	return result.Song, nil
}

// ParseFileWithOptions reads and parses a Guitar Pro file with diagnostic options.
func ParseFileWithOptions(path string, options ParseOptions) (*ParseResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading Guitar Pro file: %w", err)
	}

	result, err := ParseWithOptions(data, options)
	if err != nil {
		return result, fmt.Errorf("parsing Guitar Pro file: %w", err)
	}
	return result, nil
}
