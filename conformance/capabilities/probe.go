//go:build ignore

// SPDX-License-Identifier: MIT

// Development-only public API bridge for probe.mjs. Not part of the Go package.
package main

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"

	guitarpro "github.com/CaliLuke/go-guitar-pro"
)

type request struct {
	Paths  []string
	Output string
}

type receipt struct {
	Path             string
	Output           string
	ParseError       string
	ExportError      string
	SourceBeatLyrics bool
	Diagnostics      []guitarpro.ParseDiagnostic
	Report           guitarpro.ExportReport
}

func sourceHasBeatLyrics(data []byte) bool {
	gpif, ok := sourceGPIF(data)
	if !ok {
		return false
	}
	decoder := xml.NewDecoder(bytes.NewReader(gpif))
	stack := make([]xml.Name, 0, 16)
	for {
		token, err := decoder.Token()
		if err != nil {
			return false
		}
		switch value := token.(type) {
		case xml.StartElement:
			if value.Name.Local == "Lyrics" && len(stack) > 0 && stack[len(stack)-1].Local == "Beat" {
				return true
			}
			stack = append(stack, value.Name)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}
}

func sourceGPIF(data []byte) ([]byte, bool) {
	if len(data) < 4 {
		return nil, false
	}
	switch string(data[:4]) {
	case "PK\x03\x04":
		archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, false
		}
		for _, file := range archive.File {
			if filepath.Base(file.Name) != "score.gpif" {
				continue
			}
			reader, openErr := file.Open()
			if openErr != nil {
				return nil, false
			}
			gpif, readErr := io.ReadAll(io.LimitReader(reader, 256<<20))
			closeErr := reader.Close()
			return gpif, readErr == nil && closeErr == nil
		}
	case "BCFZ", "BCFS":
		return sourceGPXGPIF(data)
	}
	return nil, false
}

func sourceGPXGPIF(data []byte) ([]byte, bool) {
	block := data[4:]
	if string(data[:4]) == "BCFZ" {
		decompressed, ok := sourceGPXDecompress(block)
		if !ok || len(decompressed) < 4 || string(decompressed[:4]) != "BCFS" {
			return nil, false
		}
		block = decompressed[4:]
	}
	const sectorSize = 0x1000
	for offset := sectorSize; offset+0x94 <= len(block); offset += sectorSize {
		if binary.LittleEndian.Uint32(block[offset:offset+4]) != 2 {
			continue
		}
		nameBytes := block[offset+4 : min(offset+4+127, len(block))]
		if end := bytes.IndexByte(nameBytes, 0); end >= 0 {
			nameBytes = nameBytes[:end]
		}
		fileSize := int(binary.LittleEndian.Uint32(block[offset+0x8c : offset+0x90]))
		var fileData []byte
		for pointer := offset + 0x94; pointer+4 <= len(block); pointer += 4 {
			sector := int(binary.LittleEndian.Uint32(block[pointer : pointer+4]))
			if sector == 0 {
				break
			}
			start := sector * sectorSize
			if start < 0 || start >= len(block) {
				return nil, false
			}
			fileData = append(fileData, block[start:min(start+sectorSize, len(block))]...)
			offset = start
		}
		if string(nameBytes) == "score.gpif" && fileSize >= 0 && fileSize <= len(fileData) {
			return fileData[:fileSize], true
		}
	}
	return nil, false
}

func sourceGPXDecompress(data []byte) ([]byte, bool) {
	if len(data) < 4 {
		return nil, false
	}
	expected := int(binary.LittleEndian.Uint32(data[:4]))
	if expected < 0 || expected > 256<<20 {
		return nil, false
	}
	reader := sourceBitReader{data: data, byteOffset: 4}
	result := make([]byte, 0, min(expected, 1<<20))
	for len(result) < expected {
		flag, ok := reader.bits(1, false)
		if !ok {
			break
		}
		if flag == 0 {
			size, sizeOK := reader.bits(2, true)
			if !sizeOK {
				break
			}
			for range int(size) {
				value, valueOK := reader.bits(8, false)
				if !valueOK {
					return result, len(result) > 0
				}
				result = append(result, byte(value))
			}
			continue
		}
		wordSize, wordOK := reader.bits(4, false)
		if !wordOK {
			break
		}
		offset, offsetOK := reader.bits(int(wordSize), true)
		size, sizeOK := reader.bits(int(wordSize), true)
		if !offsetOK || !sizeOK || offset == 0 || int(offset) > len(result) {
			break
		}
		count := min(int(offset), int(size))
		start := len(result) - int(offset)
		result = append(result, result[start:start+count]...)
	}
	return result, len(result) > 0
}

type sourceBitReader struct {
	data       []byte
	byteOffset int
	bitOffset  uint
}

func (reader *sourceBitReader) bits(count int, reversed bool) (uint32, bool) {
	var result uint32
	for index := range count {
		if reader.byteOffset >= len(reader.data) {
			return result, false
		}
		bit := uint32((reader.data[reader.byteOffset] >> (7 - reader.bitOffset)) & 1)
		if reversed {
			result |= bit << index
		} else {
			result = result<<1 | bit
		}
		reader.bitOffset++
		if reader.bitOffset == 8 {
			reader.bitOffset = 0
			reader.byteOffset++
		}
	}
	return result, true
}

func run() error {
	var input request
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		return err
	}
	encoder := json.NewEncoder(os.Stdout)
	for index, path := range input.Paths {
		result := receipt{Path: path}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		result.SourceBeatLyrics = sourceHasBeatLyrics(data)
		parsed, err := guitarpro.ParseWithOptions(data, guitarpro.ParseOptions{})
		if parsed != nil {
			result.Diagnostics = parsed.Diagnostics
		}
		if err != nil {
			result.ParseError = err.Error()
		} else {
			exported, report, exportErr := guitarpro.ExportWithReport(parsed.Song, guitarpro.ExportFormatGP8, guitarpro.ExportOptions{})
			result.Report = report
			if exportErr != nil {
				result.ExportError = exportErr.Error()
			} else {
				result.Output = filepath.Join(input.Output, fmt.Sprintf("%d.gp", index))
				if writeErr := os.WriteFile(result.Output, exported, 0o600); writeErr != nil {
					return writeErr
				}
			}
		}
		if err := encoder.Encode(result); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
