// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unicode/utf8"
)

const maxBinaryStylesheetSize = 1 << 20

type binaryStyleRecord struct {
	key   string `wire:"key"`
	kind  byte   `wire:"type"`
	value []byte `wire:"value"`
}
type binaryStyleReader struct {
	data     []byte
	position int
}

func (reader *binaryStyleReader) take(count int) ([]byte, error) {
	if count < 0 || count > len(reader.data)-reader.position {
		return nil, fmt.Errorf("record truncated at offset %d: need %d bytes, have %d", reader.position, count, len(reader.data)-reader.position)
	}
	value := reader.data[reader.position : reader.position+count]
	reader.position += count
	return value, nil
}
func readBinaryStylesheet(data []byte) ([]binaryStyleRecord, error) {
	if len(data) > maxBinaryStylesheetSize {
		return nil, fmt.Errorf("size %d exceeds %d-byte limit", len(data), maxBinaryStylesheetSize)
	}
	reader := binaryStyleReader{data: data}
	header, err := reader.take(4)
	if err != nil {
		return nil, err
	}
	count := int64(int32(binary.BigEndian.Uint32(header)))
	if count < 0 || count > int64(len(data)-4)/3 {
		return nil, fmt.Errorf("record count %d exceeds remaining stylesheet length %d", count, len(data)-4)
	}
	records := make([]binaryStyleRecord, 0, int(count))
	for index := int64(0); index < count; index++ {
		record, readErr := reader.record()
		if readErr != nil {
			return nil, fmt.Errorf("record %d: %w", index, readErr)
		}
		records = append(records, record)
	}
	if reader.position != len(data) {
		return nil, fmt.Errorf("unexpected %d trailing bytes at offset %d", len(data)-reader.position, reader.position)
	}
	return records, nil
}
func (reader *binaryStyleReader) record() (binaryStyleRecord, error) {
	length, err := reader.take(1)
	if err != nil {
		return binaryStyleRecord{}, err
	}
	key, err := reader.take(int(length[0]))
	if err != nil {
		return binaryStyleRecord{}, err
	}
	if !utf8.Valid(key) {
		return binaryStyleRecord{}, fmt.Errorf("key is not valid UTF-8")
	}
	kind, err := reader.take(1)
	if err != nil {
		return binaryStyleRecord{}, err
	}
	count := 0
	start := reader.position
	switch kind[0] {
	case 0:
		count = 1
	case 1, 2, 7:
		count = 4
	case 4, 5:
		count = 8
	case 6:
		count = 16
	case 3:
		rawLength, readErr := reader.take(2)
		if readErr != nil {
			return binaryStyleRecord{}, readErr
		}
		count = int(int16(binary.BigEndian.Uint16(rawLength)))
		if count < 0 {
			return binaryStyleRecord{}, fmt.Errorf("key %q has negative string length %d", key, count)
		}
	default:
		return binaryStyleRecord{}, fmt.Errorf("key %q has unknown type %d", key, kind[0])
	}
	value, err := reader.take(count)
	if err != nil {
		return binaryStyleRecord{}, fmt.Errorf("key %q: %w", key, err)
	}
	if kind[0] == 3 && !utf8.Valid(value) {
		return binaryStyleRecord{}, fmt.Errorf("key %q string is not valid UTF-8", key)
	}
	if kind[0] == 0 && value[0] > 1 {
		return binaryStyleRecord{}, fmt.Errorf("key %q has invalid boolean byte %d", key, value[0])
	}
	return binaryStyleRecord{key: string(key), kind: kind[0], value: bytes.Clone(reader.data[start:reader.position])}, nil
}
func applyBinaryStylesheet(song *Song, data []byte) error {
	records, err := readBinaryStylesheet(data)
	if err != nil {
		return fmt.Errorf("reading BinaryStylesheet: %w", err)
	}
	style := &ScoreStyle{records: records}
	for _, record := range records {
		if err := applyHeaderFooterRecord(style, record); err != nil {
			return err
		}
		switch record.key {
		case "System/ExtendedBarLines":
			if record.kind != 0 {
				return fmt.Errorf("reading BinaryStylesheet: key %q requires boolean type 0, got %d", record.key, record.kind)
			}
			value := record.value[0] == 1
			style.ExtendedBarLines = &value
		case "System/barIndexDrawType":
			if record.kind != 1 {
				return fmt.Errorf("reading BinaryStylesheet: key %q requires integer type 1, got %d", record.key, record.kind)
			}
			value := int32(binary.BigEndian.Uint32(record.value))
			if value < 0 || value > int32(BarNumberHide) {
				return fmt.Errorf("reading BinaryStylesheet: key %q policy %d is outside 0..2", record.key, value)
			}
			policy := BarNumberPolicy(value)
			style.BarNumbers = &policy
		}
	}
	song.Style = style
	song.projectHeaderFooterCompatibility()
	return nil
}
func scoreStyleRecords(style *ScoreStyle) []binaryStyleRecord {
	if style == nil {
		return nil
	}
	records := make([]binaryStyleRecord, 0, len(style.records)+2)
	for _, record := range style.records {
		if record.key != "System/ExtendedBarLines" && record.key != "System/barIndexDrawType" {
			records = append(records, record)
		}
	}
	if style.ExtendedBarLines != nil {
		value := byte(0)
		if *style.ExtendedBarLines {
			value = 1
		}
		records = append(records, binaryStyleRecord{key: "System/ExtendedBarLines", kind: 0, value: []byte{value}})
	}
	if style.BarNumbers != nil {
		value := make([]byte, 4)
		binary.BigEndian.PutUint32(value, uint32(*style.BarNumbers))
		records = append(records, binaryStyleRecord{key: "System/barIndexDrawType", kind: 1, value: value})
	}
	return records
}
func buildGP8BinaryStylesheet(song *Song) []byte {
	records := mergeHeaderFooterRecords(song, scoreStyleRecords(song.Style))
	output := make([]byte, 4)
	binary.BigEndian.PutUint32(output, uint32(len(records)))
	for _, record := range records {
		output = append(output, byte(len(record.key)))
		output = append(output, record.key...)
		output = append(output, record.kind)
		output = append(output, record.value...)
	}
	return output
}
func validateScoreStyle(style *ScoreStyle, diagnostics *[]ScoreDiagnostic) {
	if style == nil {
		return
	}
	if style.BarNumbers != nil && *style.BarNumbers > BarNumberHide {
		*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.style.bar-number-policy", Kind: ScoreDiagnosticValue, Reason: fmt.Sprintf("bar-number policy %d is undefined", *style.BarNumbers)})
	}
}
