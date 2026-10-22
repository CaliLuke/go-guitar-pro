// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/binary"
	"strings"
	"testing"
)

func TestGPXFileDataStopsAtDeclaredSize(t *testing.T) {
	const sectorSize = 0x1000
	data := make([]byte, 4*sectorSize)
	entryOffset := sectorSize
	binary.LittleEndian.PutUint32(data[entryOffset:], 2)
	copy(data[entryOffset+4:], "payload.bin")
	binary.LittleEndian.PutUint32(data[entryOffset+0x8c:], 1)
	binary.LittleEndian.PutUint32(data[entryOffset+0x94:], 2)
	data[2*sectorSize] = 'A'

	files, err := gpxReadUncompressedBlock(data)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(files["payload.bin"]); got != "A" {
		t.Fatalf("payload = %q, want declared one-byte payload", got)
	}
}

func TestGPXRejectsMoreSectorsThanDeclaredSizeRequires(t *testing.T) {
	const sectorSize = 0x1000
	data := make([]byte, 4*sectorSize)
	entryOffset := sectorSize
	binary.LittleEndian.PutUint32(data[entryOffset:], 2)
	copy(data[entryOffset+4:], "payload.bin")
	binary.LittleEndian.PutUint32(data[entryOffset+0x8c:], 1)
	for index := range 3 {
		binary.LittleEndian.PutUint32(data[entryOffset+0x94+4*index:], 2)
	}

	_, err := gpxReadUncompressedBlock(data)
	if err == nil || !strings.Contains(err.Error(), "more sectors than its declared size") {
		t.Fatalf("error = %v, want excess GPX sector error", err)
	}
}

func TestGPXRejectsOversizedExtractedData(t *testing.T) {
	const sectorSize = 0x1000
	data := make([]byte, 2*sectorSize)
	entryOffset := sectorSize
	binary.LittleEndian.PutUint32(data[entryOffset:], 2)
	copy(data[entryOffset+4:], "large.bin")
	binary.LittleEndian.PutUint32(data[entryOffset+0x8c:], maxGPXExtractedFilesSize+1)

	_, err := gpxReadUncompressedBlock(data)
	if err == nil || !strings.Contains(err.Error(), "extracted file data exceeds") {
		t.Fatalf("error = %v, want GPX extracted-data limit error", err)
	}
}

func TestGPXDecompressDoesNotExceedDeclaredLength(t *testing.T) {
	for _, test := range []struct {
		name           string
		bits           string
		expectedLength int
		want           string
	}{
		{name: "literal", bits: gpxRawLiteralBits("ABC"), expectedLength: 1, want: "A"},
		{name: "backreference", bits: gpxRawLiteralBits("ABC") + "10010" + "11" + "11", expectedLength: 4, want: "ABCA"},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := make([]byte, 4)
			data = append(data, packGPXBits(test.bits)...)
			got := gpxDecompressPayload(data, test.expectedLength)
			if string(got) != test.want {
				t.Fatalf("decompressed data = %q, want %q", got, test.want)
			}
		})
	}
}

func TestGPXRejectsInvalidBCFZPayloadHeader(t *testing.T) {
	compressed := []byte("BCFZ")
	compressed = binary.LittleEndian.AppendUint32(compressed, 4)
	compressed = append(compressed, packGPXBits(gpxRawLiteralBits("NOPE"))...)

	_, err := gpxReadFiles(compressed)
	if err == nil || !strings.Contains(err.Error(), "missing BCFS header") {
		t.Fatalf("error = %v, want invalid BCFZ payload header error", err)
	}
}

func gpxRawLiteralBits(value string) string {
	var bits strings.Builder
	bits.Grow(len(value)*9 + len(value)/3*3)
	for offset := 0; offset < len(value); {
		count := min(3, len(value)-offset)
		bits.WriteByte('0')
		for bit := 0; bit < 2; bit++ {
			bits.WriteByte('0' + byte((count>>bit)&1))
		}
		for _, b := range []byte(value[offset : offset+count]) {
			for shift := 7; shift >= 0; shift-- {
				bits.WriteByte('0' + ((b >> shift) & 1))
			}
		}
		offset += count
	}
	return bits.String()
}

func packGPXBits(bits string) []byte {
	packed := make([]byte, (len(bits)+7)/8)
	for index, bit := range bits {
		if bit == '1' {
			packed[index/8] |= 1 << (7 - index%8)
		}
	}
	return packed
}
