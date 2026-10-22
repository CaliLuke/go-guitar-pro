// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/binary"
	"fmt"
)

// gpxReadFiles reads the GPX container and extracts files.
// Supports both BCFZ (compressed) and BCFS (uncompressed) headers.
func gpxReadFiles(data []byte) (map[string][]byte, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("data too short for GPX container")
	}
	header := string(data[:4])
	switch header {
	case "BCFZ":
		decompressed, err := gpxDecompress(data[4:])
		if err != nil {
			return nil, fmt.Errorf("decompressing BCFZ: %w", err)
		}
		if len(decompressed) < 4 || string(decompressed[:4]) != "BCFS" {
			return nil, fmt.Errorf("invalid BCFZ payload: missing BCFS header")
		}
		// Decompressed data starts with "BCFS" header — skip it.
		return gpxReadUncompressedBlock(decompressed[4:])
	case "BCFS":
		return gpxReadUncompressedBlock(data[4:])
	default:
		return nil, fmt.Errorf("unsupported GPX header: %s", header)
	}
}

const maxGPXDecompressedSize = 256 << 20

const initialGPXDecompressCapacity = 1 << 20

const maxGPXExtractedFilesSize = maxGPXDecompressedSize

// gpxDecompress decompresses BCFZ data using GPX's custom bit-based compression.
func gpxDecompress(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("compressed data too short")
	}
	expectedLength := int64(binary.LittleEndian.Uint32(data[:4]))
	if expectedLength > maxGPXDecompressedSize {
		return nil, fmt.Errorf("BCFZ claims to decompress to %d bytes, over the %d byte limit",
			expectedLength, maxGPXDecompressedSize)
	}
	return gpxDecompressPayload(data, int(expectedLength)), nil
}

// gpxDecompressPayload decodes as much payload as is available. Valid GPX
// files can end with incomplete padding bits, so EOF terminates decoding.
func gpxDecompressPayload(data []byte, expectedLength int) []byte {
	br := &bitReader{data: data, pos: 4, bitPos: 0}
	result := make([]byte, 0, min(expectedLength, initialGPXDecompressCapacity))

	for len(result) < expectedLength {
		flag, err := br.readBits(1)
		if err != nil {
			break // Some valid GPX files end with incomplete padding bits.
		}
		if flag == 1 {
			// Compressed: copy from already-decompressed data
			wordSize, err := br.readBits(4)
			if err != nil {
				break
			}
			offset, err := br.readBitsReversed(int(wordSize))
			if err != nil {
				break
			}
			size, err := br.readBitsReversed(int(wordSize))
			if err != nil {
				break
			}
			sourcePos := len(result) - int(offset)
			toRead := int(offset)
			if int(size) < toRead {
				toRead = int(size)
			}
			if sourcePos >= 0 && sourcePos+toRead <= len(result) {
				toRead = min(toRead, expectedLength-len(result))
				result = append(result, result[sourcePos:sourcePos+toRead]...)
			}
		} else {
			// Raw: read literal bytes
			size, err := br.readBitsReversed(2)
			if err != nil {
				break
			}
			for i := 0; i < int(size) && len(result) < expectedLength; i++ {
				b, err := br.readByte()
				if err != nil {
					break
				}
				result = append(result, b)
			}
		}
	}
	return result
}

// gpxReadUncompressedBlock reads the BCFS sector-based filesystem.
// Data starts after the 4-byte header (already stripped).
func gpxReadUncompressedBlock(data []byte) (map[string][]byte, error) {
	const sectorSize = 0x1000
	files := make(map[string][]byte)
	var totalFileDataSize uint64
	offset := sectorSize // First sector is padding

	for offset+3 < len(data) {
		if offset+4 > len(data) {
			break
		}
		entryType := int(binary.LittleEndian.Uint32(data[offset : offset+4]))
		if entryType == 2 {
			// File entry
			// fileName at offset+0x04, 127 bytes, zero-terminated
			nameEnd := offset + 0x04
			nameLen := 0
			for nameLen < 127 && nameEnd+nameLen < len(data) && data[nameEnd+nameLen] != 0 {
				nameLen++
			}
			fileName := string(data[nameEnd : nameEnd+nameLen])

			// fileSize at offset+0x8c
			if offset+0x90 > len(data) {
				break
			}
			fileSize := uint64(binary.LittleEndian.Uint32(data[offset+0x8c : offset+0x90]))
			if fileSize > maxGPXExtractedFilesSize-totalFileDataSize {
				return nil, fmt.Errorf("GPX extracted file data exceeds the %d-byte limit", maxGPXExtractedFilesSize)
			}
			totalFileDataSize += fileSize

			// Read sector indices starting at offset+0x94
			dataPointerOffset := offset + 0x94
			var fileData []byte
			sectorCount := 0
			maxSectorCount := int((fileSize + sectorSize - 1) / sectorSize)
			nextEntryOffset := offset
			for {
				ptrOff := dataPointerOffset + 4*sectorCount
				if ptrOff+4 > len(data) {
					break
				}
				sector := binary.LittleEndian.Uint32(data[ptrOff : ptrOff+4])
				if sector == 0 {
					break
				}
				if sectorCount >= maxSectorCount {
					return nil, fmt.Errorf("file %q has more sectors than its declared size %d requires", fileName, fileSize)
				}
				sectorStart64 := uint64(sector) * sectorSize
				if sectorStart64 >= uint64(len(data)) {
					return nil, fmt.Errorf("file %q references sector %d outside the GPX block", fileName, sector)
				}
				sectorStart := int(sectorStart64)
				sectorEnd := len(data)
				if len(data)-sectorStart > sectorSize {
					sectorEnd = sectorStart + sectorSize
				}
				if uint64(len(fileData)) < fileSize {
					remainingFileSize := fileSize - uint64(len(fileData))
					sectorDataSize := min(uint64(sectorEnd-sectorStart), remainingFileSize)
					fileData = append(fileData, data[sectorStart:sectorStart+int(sectorDataSize)]...)
				}
				if sectorStart > nextEntryOffset {
					nextEntryOffset = sectorStart
				}
				sectorCount++
			}
			if uint64(len(fileData)) != fileSize {
				return nil, fmt.Errorf("file %q contains %d of %d declared bytes", fileName, len(fileData), fileSize)
			}

			files[fileName] = fileData
			offset = nextEntryOffset
		}
		offset += sectorSize
	}
	return files, nil
}

// bitReader reads individual bits from a byte slice.
type bitReader struct {
	data   []byte
	pos    int
	bitPos int // 0-7, bits remaining in current byte (MSB first)
}

func (br *bitReader) readBits(count int) (uint32, error) {
	var result uint32
	for i := 0; i < count; i++ {
		if br.pos >= len(br.data) {
			return result, fmt.Errorf("EOF")
		}
		bit := (br.data[br.pos] >> (7 - br.bitPos)) & 1
		result = (result << 1) | uint32(bit)
		br.bitPos++
		if br.bitPos >= 8 {
			br.bitPos = 0
			br.pos++
		}
	}
	return result, nil
}

func (br *bitReader) readBitsReversed(count int) (uint32, error) {
	var result uint32
	for i := 0; i < count; i++ {
		if br.pos >= len(br.data) {
			return result, fmt.Errorf("EOF")
		}
		bit := (br.data[br.pos] >> (7 - br.bitPos)) & 1
		result |= uint32(bit) << i
		br.bitPos++
		if br.bitPos >= 8 {
			br.bitPos = 0
			br.pos++
		}
	}
	return result, nil
}

func (br *bitReader) readByte() (byte, error) {
	val, err := br.readBits(8)
	return byte(val), err
}

func parseGPXWithContext(data []byte, context *parseContext) (*Song, error) {
	files, err := gpxReadFiles(data)
	if err != nil {
		return nil, fmt.Errorf("reading GPX filesystem: %w", err)
	}
	gpifData, ok := files["score.gpif"]
	if !ok {
		return nil, fmt.Errorf("no score.gpif found in GPX container")
	}
	song, err := parseGPIFWithContext(gpifData, context)
	if err != nil {
		return nil, err
	}
	if stylesheet, exists := files["BinaryStylesheet"]; exists {
		if applyErr := applyBinaryStylesheet(song, stylesheet); applyErr != nil {
			return nil, applyErr
		}
	}
	if configuration, exists := files["PartConfiguration"]; exists {
		if applyErr := applyPartConfiguration(song, configuration); applyErr != nil {
			return nil, applyErr
		}
	}
	return song, nil
}
