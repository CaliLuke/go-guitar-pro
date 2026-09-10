// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"os"
	"strconv"
	"testing"
)

func TestReadCountRejectsInvalidCounts(t *testing.T) {
	for _, data := range [][]byte{
		{0xff, 0xff, 0xff, 0xff, 0, 0},
		{0xff, 0xff, 0xff, 0x7f, 0, 0},
		{1, 0},
	} {
		if _, err := newCursor(data).readCount(2, "beat count"); err == nil {
			t.Error("readCount() error = nil")
		}
	}
}

func TestCorruptFilesReturnParseError(t *testing.T) {
	original, err := os.ReadFile("testdata/gp5/serenade.gp5")
	if err != nil {
		t.Fatal(err)
	}
	for _, fraction := range []int{2, 3, 4, 8, 16, 32, 64} {
		t.Run("truncated-1/"+strconv.Itoa(fraction), func(t *testing.T) {
			_, err := Parse(original[:len(original)/fraction])
			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("error = %v, want *ParseError", err)
			}
		})
	}
}

func TestGPXRejectsAbsurdDecompressedLength(t *testing.T) {
	data := []byte{0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0}
	if _, err := gpxDecompress(data); err == nil {
		t.Error("gpxDecompress() error = nil")
	}
}
