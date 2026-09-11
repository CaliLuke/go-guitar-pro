// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"hash/crc32"
	"os"
	"slices"
	"strings"
)

const (
	gp8DocumentVersion     = "8.1.3"
	gp8RevisionRequired    = "12024"
	gp8RevisionRecommended = "13000"
	gp8Revision            = "13007"
	gp8ContainerVersion    = "7.0"
)

// ExportFormat identifies a Guitar Pro serialization target.
type ExportFormat uint8

const (
	// ExportFormatGP8 writes the native Guitar Pro 8 .gp container format.
	ExportFormatGP8 ExportFormat = 8
)

// ExportOptions configures format-specific serialization behavior.
type ExportOptions struct {
	GP8        GP8ExportOptions
	LossPolicy ExportLossPolicy
}

// GP8ExportOptions configures native Guitar Pro 8 serialization.
type GP8ExportOptions struct {
	// PercussionNoteheads overrides the rendered notehead for a MIDI percussion
	// value without changing its MIDI or RSE playback articulation.
	PercussionNoteheads map[int16]GP8PercussionNotehead
}

// GP8PercussionNotehead identifies a rendered percussion notehead family.
type GP8PercussionNotehead uint8

const (
	// GP8PercussionNoteheadDefault uses Guitar Pro's native kit notation.
	GP8PercussionNoteheadDefault GP8PercussionNotehead = iota
	// GP8PercussionNoteheadFilled uses duration-sensitive filled, half, and whole heads.
	GP8PercussionNoteheadFilled
	// GP8PercussionNoteheadX uses an X head for every duration.
	GP8PercussionNoteheadX
	// GP8PercussionNoteheadCircleX uses a circled X head for every duration.
	GP8PercussionNoteheadCircleX
	// GP8PercussionNoteheadHeavyX uses a heavy X head for every duration.
	GP8PercussionNoteheadHeavyX
)

// Export serializes song in target format and returns the complete file bytes.
// Export currently supports [ExportFormatGP8]. It does not mutate song.
func Export(song *Song, target ExportFormat) ([]byte, error) {
	return ExportWithOptions(song, target, ExportOptions{})
}

// ExportWithOptions serializes song in target format using options and returns
// the complete file bytes. It does not mutate song or options.
func ExportWithOptions(song *Song, target ExportFormat, options ExportOptions) ([]byte, error) {
	data, _, err := ExportWithReport(song, target, options)
	return data, err
}

// ExportWithReport preflights and serializes through one conversion decision path.
// A strict loss-policy failure returns no output bytes and the complete report.
func ExportWithReport(song *Song, target ExportFormat, options ExportOptions) ([]byte, ExportReport, error) {
	report, plan := planExport(song, target, options)
	for _, entry := range report.Entries {
		if entry.Disposition == ExportDispositionRejected {
			return nil, report, fmt.Errorf("exporting Guitar Pro file: %s", entry.Reason)
		}
	}
	if refused := refusedExportEntries(report, options.LossPolicy); len(refused) != 0 {
		return nil, report, &ExportLossError{Entries: refused}
	}
	if plan == nil {
		return nil, report, fmt.Errorf("exporting Guitar Pro file: conversion plan is unavailable")
	}
	gpif, err := xml.MarshalIndent(plan.document, "", "  ")
	if err != nil {
		return nil, report, fmt.Errorf("marshaling GPIF XML: %w", err)
	}
	gpif = append(append([]byte(xml.Header), gpif...), '\n')

	entries := []gp8ArchiveEntry{
		{name: "VERSION", method: zip.Store, data: []byte(gp8ContainerVersion)},
		{name: "meta.json", method: zip.Store, data: []byte("{\n}\n")},
		{name: "Content/", method: zip.Store},
		{name: "Content/BinaryStylesheet", method: zip.Store, data: buildGP8BinaryStylesheet()},
		{name: "Content/PartConfiguration", method: zip.Store, data: buildGP8PartConfiguration(song)},
		{name: "Content/LayoutConfiguration", method: zip.Store, data: buildGP8LayoutConfiguration(song)},
		{name: "Content/score.gpif", method: zip.Store, data: gpif},
	}
	if plan.backingTrackAsset != nil {
		entries = append(entries, gp8ArchiveEntry{
			name:   plan.backingTrackAsset.name,
			method: zip.Store,
			data:   plan.backingTrackAsset.data,
		})
	}
	data, err := writeGP8Archive(entries)
	return data, report, err
}

type gp8ArchiveEntry = struct {
	name   string
	method uint16
	data   []byte
}

func writeGP8Archive(entries []gp8ArchiveEntry) ([]byte, error) {
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	for _, entry := range entries {
		encoded := entry.data
		if entry.method == zip.Deflate {
			var compressed bytes.Buffer
			compressor, createErr := flate.NewWriter(&compressed, flate.DefaultCompression)
			if createErr != nil {
				_ = archive.Close()
				return nil, fmt.Errorf("creating compressor for archive member %q: %w", entry.name, createErr)
			}
			if _, writeErr := compressor.Write(entry.data); writeErr != nil {
				_ = compressor.Close()
				_ = archive.Close()
				return nil, fmt.Errorf("compressing archive member %q: %w", entry.name, writeErr)
			}
			if closeErr := compressor.Close(); closeErr != nil {
				_ = archive.Close()
				return nil, fmt.Errorf("closing compressor for archive member %q: %w", entry.name, closeErr)
			}
			encoded = compressed.Bytes()
		}
		header := &zip.FileHeader{
			Name:               entry.name,
			Method:             entry.method,
			Flags:              0x0800,
			CRC32:              crc32.ChecksumIEEE(entry.data),
			CompressedSize64:   uint64(len(encoded)),
			UncompressedSize64: uint64(len(entry.data)),
		}
		if strings.HasSuffix(entry.name, "/") {
			header.SetMode(os.ModeDir | 0o755)
		} else {
			header.SetMode(0o644)
		}
		writer, createErr := archive.CreateRaw(header)
		if createErr != nil {
			_ = archive.Close()
			return nil, fmt.Errorf("creating archive member %q: %w", entry.name, createErr)
		}
		if _, writeErr := writer.Write(encoded); writeErr != nil {
			_ = archive.Close()
			return nil, fmt.Errorf("writing archive member %q: %w", entry.name, writeErr)
		}
	}
	if err := archive.Close(); err != nil {
		return nil, fmt.Errorf("closing Guitar Pro archive: %w", err)
	}
	return output.Bytes(), nil
}

func buildGP8BinaryStylesheet() []byte {
	// A zero-entry stylesheet is valid and lets Guitar Pro apply its defaults.
	return make([]byte, 4)
}

func buildGP8PartConfiguration(song *Song) []byte {
	var output bytes.Buffer
	writeBigEndianInt32(&output, len(song.Tracks)+1)

	// The first view contains every track. Each following view contains one track.
	output.WriteByte(0)
	writeBigEndianInt32(&output, len(song.Tracks))
	for index := range song.Tracks {
		output.WriteByte(gp8TrackViewFlags(&song.Tracks[index]))
	}
	for index := range song.Tracks {
		output.WriteByte(0)
		writeBigEndianInt32(&output, 1)
		output.WriteByte(gp8TrackViewFlags(&song.Tracks[index]))
	}

	writeBigEndianInt32(&output, 1)
	return output.Bytes()
}

func gp8TrackViewFlags(track *Track) byte {
	if track.PercussionTrack {
		return 0x01
	}
	var flags byte
	if track.Settings.Notation {
		flags |= 0x01
	}
	if track.Settings.Tablature {
		flags |= 0x02
	}
	if flags == 0 {
		flags = 0x01
	}
	return flags
}

func buildGP8LayoutConfiguration(song *Song) []byte {
	var output bytes.Buffer
	writeBigEndianInt32(&output, 4)
	output.WriteByte(0x00)
	output.WriteByte(0x00)
	for index := range song.Tracks {
		if song.Tracks[index].Visible {
			output.WriteByte(0xff)
		} else {
			output.WriteByte(0x00)
		}
	}
	for range song.Tracks {
		output.WriteByte(0xff)
	}
	return output.Bytes()
}

func writeBigEndianInt32(output *bytes.Buffer, value int) {
	var encoded [4]byte
	binary.BigEndian.PutUint32(encoded[:], uint32(value))
	output.Write(encoded[:])
}

// ExportFile serializes song in target format and writes it to path.
func ExportFile(path string, song *Song, target ExportFormat) error {
	return ExportFileWithOptions(path, song, target, ExportOptions{})
}

// ExportFileWithOptions serializes song in target format using options and
// writes it to path.
func ExportFileWithOptions(path string, song *Song, target ExportFormat, options ExportOptions) error {
	data, err := ExportWithOptions(song, target, options)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing Guitar Pro file: %w", err)
	}
	return nil
}

func validateGP8Song(song *Song) error {
	if len(song.Tracks) == 0 {
		return fmt.Errorf("song has no tracks")
	}
	if len(song.MeasureHeaders) == 0 {
		return fmt.Errorf("song has no measure headers")
	}
	openingTempo, _, tempoErr := gp8ResolvedFieldTempo(song)
	if tempoErr != nil {
		return tempoErr
	}
	hasPositiveTempo := openingTempo > 0
	for _, tempo := range song.TempoAutomations {
		hasPositiveTempo = hasPositiveTempo || tempo.Tempo > 0
	}
	if !hasPositiveTempo {
		return fmt.Errorf("song has no positive tempo")
	}
	if err := validateGP8BackingTrack(song.BackingTrack); err != nil {
		return err
	}
	for measureIndex, header := range song.MeasureHeaders {
		if header.TimeSignature.Numerator <= 0 || header.TimeSignature.Denominator.Value == 0 {
			return fmt.Errorf("measure %d has invalid time signature %d/%d", measureIndex, header.TimeSignature.Numerator, header.TimeSignature.Denominator.Value)
		}
	}
	for trackIndex := range song.Tracks {
		track := &song.Tracks[trackIndex]
		if track.ChannelIndex >= 0 && track.ChannelIndex < len(song.Channels) {
			bank := song.Channels[track.ChannelIndex].Bank
			if bank < 0 || bank > 16383 {
				return fmt.Errorf("track %d has MIDI bank %d outside 0..16383", trackIndex, bank)
			}
			program := song.Channels[track.ChannelIndex].Instrument
			if program < 0 || program > 127 {
				return fmt.Errorf("track %d has MIDI program %d outside 0..127", trackIndex, program)
			}
		}
		staves := gp8ExportStaves(track)
		for staffIndex := range staves {
			if len(staves[staffIndex].Measures) != len(song.MeasureHeaders) {
				return fmt.Errorf("track %d staff %d has %d measures, want %d", trackIndex, staffIndex, len(staves[staffIndex].Measures), len(song.MeasureHeaders))
			}
		}
		for soundIndex, sound := range track.Sounds {
			if sound.Bank < 0 || sound.Bank > 16383 {
				return fmt.Errorf("track %d sound %d has MIDI bank %d outside 0..16383", trackIndex, soundIndex, sound.Bank)
			}
			if sound.Program < 0 || sound.Program > 127 {
				return fmt.Errorf("track %d sound %d has MIDI program %d outside 0..127", trackIndex, soundIndex, sound.Program)
			}
		}
		for automationIndex, automation := range track.SoundAutomations {
			if automation.Sound < 0 || automation.Sound >= len(track.Sounds) {
				return fmt.Errorf("track %d sound automation %d uses sound index %d with %d sounds", trackIndex, automationIndex, automation.Sound, len(track.Sounds))
			}
			if automation.Bar < 0 || automation.Bar >= len(song.MeasureHeaders) || automation.Position < 0 || automation.Position > 1 {
				return fmt.Errorf("track %d sound automation %d has invalid position bar=%d position=%v", trackIndex, automationIndex, automation.Bar, automation.Position)
			}
		}
		if track.PercussionTrack {
			for articulationIndex, articulation := range track.PercussionArticulations {
				if articulation.OutputMIDINumber < 0 || articulation.OutputMIDINumber > 127 {
					return fmt.Errorf("track %d percussion articulation %d has output MIDI value %d outside 0..127", trackIndex, articulationIndex, articulation.OutputMIDINumber)
				}
				for _, input := range articulation.InputMIDINumbers {
					if input < 0 || input > 127 {
						return fmt.Errorf("track %d percussion articulation %d has input MIDI value %d outside 0..127", trackIndex, articulationIndex, input)
					}
				}
			}
		}
		for staffIndex := range staves {
			if err := validateGP8Staff(track, trackIndex, staffIndex, &staves[staffIndex]); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateGP8BackingTrack(backingTrack *BackingTrack) error {
	if !gp8EmbedsBackingTrack(backingTrack) {
		return nil
	}
	if backingTrack.AssetID == "" {
		return fmt.Errorf("enabled local backing track has no asset ID")
	}
	if backingTrack.EmbeddedFilePath == "" {
		return fmt.Errorf("enabled local backing track asset %q has no embedded file path", backingTrack.AssetID)
	}
	if gp8ReservedArchiveMember(backingTrack.EmbeddedFilePath) {
		return fmt.Errorf("enabled local backing track asset %q uses reserved archive path %q", backingTrack.AssetID, backingTrack.EmbeddedFilePath)
	}
	if len(backingTrack.AudioData) == 0 {
		return fmt.Errorf("enabled local backing track asset %q has no audio data", backingTrack.AssetID)
	}
	const (
		minGP8FramePadding = int64(-1 << 31)
		maxGP8FramePadding = int64(1<<31 - 1)
	)
	if backingTrack.FramePadding < minGP8FramePadding || backingTrack.FramePadding > maxGP8FramePadding {
		return fmt.Errorf("backing-track frame padding %d is outside the signed 32-bit GP8 range", backingTrack.FramePadding)
	}
	return nil
}

func gp8ReservedArchiveMember(name string) bool {
	switch name {
	case "VERSION", "meta.json", "Content/", "Content/BinaryStylesheet", "Content/PartConfiguration", "Content/LayoutConfiguration", "Content/score.gpif":
		return true
	default:
		return false
	}
}

func validateGP8Staff(track *Track, trackIndex, staffIndex int, staff *Staff) error {
	for measureIndex := range staff.Measures {
		measure := &staff.Measures[measureIndex]
		if len(measure.Voices) > 4 {
			return fmt.Errorf("track %d staff %d measure %d has %d voices, Guitar Pro 8 supports at most 4", trackIndex, staffIndex, measureIndex, len(measure.Voices))
		}
		for voiceIndex := range measure.Voices {
			for beatIndex := range measure.Voices[voiceIndex].Beats {
				beat := &measure.Voices[voiceIndex].Beats[beatIndex]
				if _, ok := gp8NoteValue(beat.Duration.Value); !ok {
					return fmt.Errorf("track %d staff %d measure %d voice %d beat %d has unsupported duration value %d", trackIndex, staffIndex, measureIndex, voiceIndex, beatIndex, beat.Duration.Value)
				}
				for noteIndex, note := range beat.Notes {
					if track.PercussionTrack && note.HasPercussionArticulation && (note.PercussionArticulation < 0 || note.PercussionArticulation >= len(track.PercussionArticulations)) {
						return fmt.Errorf("track %d staff %d measure %d voice %d beat %d note %d uses percussion articulation %d with %d definitions", trackIndex, staffIndex, measureIndex, voiceIndex, beatIndex, noteIndex, note.PercussionArticulation, len(track.PercussionArticulations))
					}
					if track.PercussionTrack {
						for graceIndex, grace := range note.Effect.Graces {
							if grace.HasPercussionArticulation && (grace.PercussionArticulation < 0 || grace.PercussionArticulation >= len(track.PercussionArticulations)) {
								return fmt.Errorf("track %d staff %d measure %d voice %d beat %d note %d grace %d uses percussion articulation %d with %d definitions", trackIndex, staffIndex, measureIndex, voiceIndex, beatIndex, noteIndex, graceIndex, grace.PercussionArticulation, len(track.PercussionArticulations))
							}
							graceNote := gp8PercussionGraceNote(&note, &grace)
							if graceNote.Value < 0 || graceNote.Value > 127 {
								return fmt.Errorf("track %d staff %d measure %d voice %d beat %d note %d grace %d has MIDI value %d outside 0..127", trackIndex, staffIndex, measureIndex, voiceIndex, beatIndex, noteIndex, graceIndex, graceNote.Value)
							}
							if _, ok := gp8PercussionArticulationIndex(track, &graceNote); !ok {
								element := gp8DrumElement(graceNote.Value, GP8ExportOptions{})
								if element.Type == "percussion" {
									return fmt.Errorf("track %d staff %d measure %d voice %d beat %d note %d grace %d uses percussion MIDI value %d without a native Guitar Pro drum-kit articulation", trackIndex, staffIndex, measureIndex, voiceIndex, beatIndex, noteIndex, graceIndex, graceNote.Value)
								}
							}
						}
					}
					midi := gp8NoteMIDI(track, staff.Strings, &note)
					if midi < 0 || midi > 127 {
						return fmt.Errorf("track %d staff %d measure %d voice %d beat %d note %d has MIDI value %d outside 0..127", trackIndex, staffIndex, measureIndex, voiceIndex, beatIndex, noteIndex, midi)
					}
					if track.PercussionTrack {
						if _, ok := gp8PercussionArticulationIndex(track, &note); !ok {
							element := gp8DrumElement(note.Value, GP8ExportOptions{})
							if element.Type == "percussion" {
								return fmt.Errorf("track %d staff %d measure %d voice %d beat %d note %d uses percussion MIDI value %d without a native Guitar Pro drum-kit articulation", trackIndex, staffIndex, measureIndex, voiceIndex, beatIndex, noteIndex, note.Value)
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func gp8ExportStaves(track *Track) []Staff {
	if len(track.Staves) > 0 {
		staves := slices.Clone(track.Staves)
		capos := track.resolvedStaffCapos()
		for index := range staves {
			staves[index].CapoFret = capos[index]
		}
		first := staves[0]
		if track.Measures != nil {
			first.Measures = track.Measures
		}
		if track.Strings != nil {
			first.Strings = track.Strings
		}
		first.PercussionTrack = track.PercussionTrack
		staves[0] = first
		return staves
	}
	return []Staff{{
		Measures: track.Measures, Strings: track.Strings,
		PercussionTrack: track.PercussionTrack, StandardNotationLineCount: 5, CapoFret: track.CapoFret,
	}}
}

func validateGP8ExportOptions(options GP8ExportOptions) error {
	for midi, notehead := range options.PercussionNoteheads {
		if midi < 0 || midi > 127 {
			return fmt.Errorf("percussion notehead MIDI value %d is outside 0..127", midi)
		}
		if notehead > GP8PercussionNoteheadHeavyX {
			return fmt.Errorf("percussion notehead for MIDI %d has unsupported value %d", midi, notehead)
		}
	}
	return nil
}
