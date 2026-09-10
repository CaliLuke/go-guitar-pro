// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"cmp"
	"compress/flate"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"hash/crc32"
	"os"
	"slices"
	"strconv"
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

	entries := []struct {
		name   string
		method uint16
		data   []byte
	}{
		{name: "VERSION", method: zip.Store, data: []byte(gp8ContainerVersion)},
		{name: "meta.json", method: zip.Deflate, data: []byte("{\n}\n")},
		{name: "Content/", method: zip.Store},
		{name: "Content/BinaryStylesheet", method: zip.Deflate, data: buildGP8BinaryStylesheet()},
		{name: "Content/PartConfiguration", method: zip.Deflate, data: buildGP8PartConfiguration(song)},
		{name: "Content/LayoutConfiguration", method: zip.Deflate, data: buildGP8LayoutConfiguration(song)},
		{name: "Content/score.gpif", method: zip.Deflate, data: gpif},
	}
	data, err := writeGP8Archive(entries)
	return data, report, err
}

func writeGP8Archive(entries []struct {
	name   string
	method uint16
	data   []byte
}) ([]byte, error) {
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
	for measureIndex, header := range song.MeasureHeaders {
		if header.TimeSignature.Numerator <= 0 || header.TimeSignature.Denominator.Value == 0 {
			return fmt.Errorf("measure %d has invalid time signature %d/%d", measureIndex, header.TimeSignature.Numerator, header.TimeSignature.Denominator.Value)
		}
	}
	for trackIndex := range song.Tracks {
		track := &song.Tracks[trackIndex]
		if track.ChannelIndex >= 0 && track.ChannelIndex < len(song.Channels) {
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
		PercussionTrack: track.PercussionTrack, StandardNotationLineCount: 5,
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

type gp8Builder struct {
	song               *Song
	options            GP8ExportOptions
	doc                gpifDocument
	rhythmIDs          map[Duration]string
	chordIDs           []map[*Chord]string
	articulationIDs    []map[int16]int
	percussionElements [][]gpifElement
	report             *ExportReport
}

func buildGP8DocumentWithReport(song *Song, options GP8ExportOptions, report *ExportReport) (gpifDocument, error) {
	builder := gp8Builder{
		song:               song,
		options:            options,
		rhythmIDs:          make(map[Duration]string),
		chordIDs:           make([]map[*Chord]string, len(song.Tracks)),
		articulationIDs:    make([]map[int16]int, len(song.Tracks)),
		percussionElements: make([][]gpifElement, len(song.Tracks)),
		report:             report,
	}
	score := builder.buildScore()
	builder.doc = gpifDocument{
		GPVersion: gp8DocumentVersion,
		GPRevision: gpifRevision{
			Required:    gp8RevisionRequired,
			Recommended: gp8RevisionRecommended,
			Value:       gp8Revision,
		},
		Encoding: gpifEncoding{Description: "GP8"},
		Score:    score,
	}
	builder.doc.MasterTrack.Tracks = sequentialIDs(len(song.Tracks))
	if song.Anacrusis {
		builder.doc.MasterTrack.Anacrusis = &struct{}{}
	}
	builder.doc.MasterTrack.Automations = buildGP8TempoAutomations(song)

	for trackIndex := range song.Tracks {
		builder.prepareTrack(trackIndex)
		builder.doc.Tracks.Tracks = append(builder.doc.Tracks.Tracks, builder.buildTrack(trackIndex))
	}
	if err := builder.buildScoreGraph(); err != nil {
		return gpifDocument{}, err
	}
	return builder.doc, nil
}

func (builder *gp8Builder) buildScore() gpifScore {
	song := builder.song
	if song.MasterEffect.Volume != 0 || song.MasterEffect.Reverb != 0 || song.MasterEffect.Equalizer.Gain != 0 || len(song.MasterEffect.Equalizer.Knobs) != 0 {
		builder.addReport("gp8.omit.master-rse", "score-core", ExportDispositionOmitted, ScoreLocation{}, "GP8 writer does not emit the legacy master RSE effect")
	}
	if song.PageSetup != (PageSetup{}) {
		builder.addReport("gp8.omit.page-setup", "score-core", ExportDispositionOmitted, ScoreLocation{}, "GP8 writer does not emit page dimensions, margins, header selections, or text templates")
	}
	if song.Writer != "" {
		builder.addReport("gp8.omit.writer", "score-core", ExportDispositionOmitted, ScoreLocation{}, "GP8 has no separate destination for the legacy writer field")
	}
	if song.Comments != "" {
		builder.addReport("gp8.omit.comments", "score-core", ExportDispositionOmitted, ScoreLocation{}, "GP8 writer does not emit score comments")
	}
	if song.Date != "" {
		builder.addReport("gp8.omit.date", "score-core", ExportDispositionOmitted, ScoreLocation{}, "GP8 writer does not emit the score date")
	}
	if song.Version.Data != "" || song.Version.Number != [3]byte{} || song.Version.Clipboard {
		builder.addReport("gp8.normalize.source-version", "score-core", ExportDispositionNormalized, ScoreLocation{}, "GP8 output uses the writer version instead of source-format provenance")
	}
	if song.Clipboard != nil {
		builder.addReport("gp8.omit.clipboard-range", "score-core", ExportDispositionOmitted, ScoreLocation{}, "GP8 output is a score and does not emit a clipboard range")
	}
	for _, notice := range song.Notice {
		if strings.Contains(notice, "\n") {
			builder.addReport("gp8.normalize.notice-lines", "score-core", ExportDispositionNormalized, ScoreLocation{}, "GP8 stores notices as newline-separated text and cannot retain an embedded line as one slice item")
			break
		}
	}
	if len(song.Notice) == 1 && song.Notice[0] == "" {
		builder.addReport("gp8.omit.empty-notice", "score-core", ExportDispositionOmitted, ScoreLocation{}, "GP8 cannot distinguish one empty notice from no notices")
	}
	if song.Key != (KeySignature{}) {
		builder.addReport("gp8.normalize.song-key-authority", "score-core", ExportDispositionNormalized, ScoreLocation{}, "GP8 stores keys on master bars and does not emit the legacy song-level key")
	}
	if song.TripletFeel != TripletFeelNone {
		builder.addReport("gp8.normalize.song-triplet-feel-authority", "rhythm", ExportDispositionNormalized, ScoreLocation{}, "GP8 stores triplet feel on master bars and does not emit the legacy song-level value")
	}
	if song.HideTempo {
		builder.addReport("gp8.omit.tempo-visibility", "tempo-automations", ExportDispositionOmitted, ScoreLocation{}, "GP8 writer emits the opening tempo as visible")
	}
	return gpifScore{
		Title:        song.Name,
		SubTitle:     song.Subtitle,
		Artist:       song.Artist,
		Album:        song.Album,
		Words:        song.Words,
		Music:        song.Author,
		Copyright:    song.Copyright,
		Tabber:       song.Transcriber,
		Instructions: song.Instructions,
		Notices:      strings.Join(song.Notice, "\n"),
	}
}

func buildGP8TempoAutomations(song *Song) gpifAutomations {
	tempos := slices.Clone(song.TempoAutomations)
	openingTempo, conflict, _ := gp8ResolvedFieldTempo(song)
	hasInitial := false
	for index, tempo := range tempos {
		if tempo.Bar == 0 && tempo.Position == 0 {
			hasInitial = true
			if conflict && openingTempo > 0 {
				tempos[index].Tempo = openingTempo
			}
			break
		}
	}
	if !hasInitial {
		if openingTempo > 0 {
			tempos = append([]TempoAutomation{{Tempo: openingTempo}}, tempos...)
		}
	}

	automations := gpifAutomations{Automations: make([]gpifAutomation, 0, len(tempos))}
	for index, tempo := range tempos {
		if tempo.Tempo <= 0 {
			continue
		}
		text := ""
		if index == 0 {
			text = song.TempoName
		}
		automations.Automations = append(automations.Automations, gpifAutomation{
			Type:     "Tempo",
			Linear:   false,
			Value:    gpifAutomationValue{Text: strconv.FormatFloat(tempo.Tempo, 'f', -1, 64) + " 2"},
			Visible:  "true",
			Text:     text,
			Bar:      tempo.Bar,
			Position: tempo.Position,
		})
	}
	return automations
}

func gp8ResolvedFieldTempo(song *Song) (float64, bool, error) {
	if song.InitialTempo.State == SourceValueKnown {
		exact, exactErr := NewBPM(float64(song.InitialTempo.Value))
		if song.Tempo > 0 {
			if exactErr != nil {
				return gp8LegacyTempo(song.Tempo, true)
			}
			legacy, legacyErr := exact.LegacyTempo()
			if legacyErr == nil && legacy == song.Tempo {
				return float64(exact), false, nil
			}
			for _, automation := range song.TempoAutomations {
				if automation.Bar != 0 || automation.Position != 0 {
					continue
				}
				automationBPM, automationErr := NewBPM(automation.Tempo)
				if automationErr == nil {
					automationLegacy, projectionErr := automationBPM.LegacyTempo()
					if projectionErr == nil && automationLegacy == song.Tempo && automation.Tempo != float64(exact) {
						return float64(exact), true, nil
					}
				}
				break
			}
			return float64(song.Tempo), true, nil
		}
		if exactErr != nil {
			return 0, false, fmt.Errorf("song has invalid initial tempo: %w", exactErr)
		}
		return float64(exact), false, nil
	}
	if song.Tempo > 0 {
		return float64(song.Tempo), false, nil
	}
	return 0, false, nil
}

func gp8LegacyTempo(tempo int16, conflict bool) (float64, bool, error) {
	return float64(tempo), conflict, nil
}

func (builder *gp8Builder) prepareTrack(trackIndex int) {
	track := &builder.song.Tracks[trackIndex]
	chords := make(map[*Chord]string)
	for _, staff := range gp8ExportStaves(track) {
		for measureIndex := range staff.Measures {
			for voiceIndex := range staff.Measures[measureIndex].Voices {
				for beatIndex := range staff.Measures[measureIndex].Voices[voiceIndex].Beats {
					chord := staff.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Effect.Chord
					if chord != nil {
						if _, exists := chords[chord]; !exists {
							chords[chord] = strconv.Itoa(len(chords))
						}
					}
				}
			}
		}
	}
	builder.chordIDs[trackIndex] = chords

	articulationIDs := make(map[int16]int)
	if track.PercussionTrack {
		elements := gp8PercussionElements(track, builder.options)
		builder.percussionElements[trackIndex] = elements
		articulationIDs = gp8DrumArticulationIDs(elements)
	}
	builder.articulationIDs[trackIndex] = articulationIDs
}

func (builder *gp8Builder) buildTrack(trackIndex int) gpifTrack {
	track := &builder.song.Tracks[trackIndex]
	location := ScoreLocation{Track: trackIndex}
	if track.FretCount != 0 && track.FretCount != 24 {
		builder.addReport("gp8.omit.track-fret-count", "staff-ownership", ExportDispositionOmitted, location, "GP8 writer does not emit the authored fret count")
	}
	if track.Port != 0 && track.Port != 1 {
		builder.addReport("gp8.omit.track-port", "staff-ownership", ExportDispositionOmitted, location, "GP8 writer derives the connection port from the selected MIDI channel")
	}
	if track.TwelveStringedGuitarTrack {
		builder.addReport("gp8.omit.track-twelve-stringed", "staff-ownership", ExportDispositionOmitted, location, "GP8 writer does not emit the legacy twelve-string track flag")
	}
	if track.BanjoTrack {
		builder.addReport("gp8.omit.track-banjo", "staff-ownership", ExportDispositionOmitted, location, "GP8 writer does not emit the legacy banjo track flag")
	}
	if track.UseRse {
		builder.addReport("gp8.omit.track-use-rse", "score-core", ExportDispositionOmitted, location, "GP8 writer does not emit the legacy UseRse flag")
	}
	if track.Mute && track.Solo {
		builder.addReport("gp8.normalize.playback-state", "score-core", ExportDispositionNormalized, location, "GP8 stores mute and solo as one exclusive playback state, so mute takes precedence")
	}
	if track.Rse.Humanize != 0 || track.Rse.AutoAccentuation != AccentuationNone || track.Rse.Equalizer.Gain != 0 || len(track.Rse.Equalizer.Knobs) != 0 || track.Rse.Instrument != (RseInstrument{}) {
		builder.addReport("gp8.omit.track-rse", "score-core", ExportDispositionOmitted, location, "GP8 writer does not emit the legacy track RSE record")
	}
	if track.IndicateTuning {
		builder.addReport("gp8.omit.track-indicate-tuning", "score-core", ExportDispositionOmitted, location, "GP8 writer does not emit the tuning-display preference")
	}
	unsupportedSettings := track.Settings
	unsupportedSettings.Tablature = false
	unsupportedSettings.Notation = false
	if unsupportedSettings != (TrackSettings{}) {
		builder.addReport("gp8.omit.track-display-settings", "score-core", ExportDispositionOmitted, location, "GP8 writer emits notation and tablature selection but not the remaining track display settings")
	}
	if !track.PercussionTrack && !track.Settings.Tablature && !track.Settings.Notation {
		builder.addReport("gp8.normalize.track-view", "score-core", ExportDispositionNormalized, location, "GP8 requires a pitched track view and defaults to standard notation")
	}
	channel := defaultMidiChannel()
	if track.ChannelIndex >= 0 && track.ChannelIndex < len(builder.song.Channels) {
		channel = builder.song.Channels[track.ChannelIndex]
	} else if track.PercussionTrack {
		channel.Channel = DefaultPercussionChannel
		channel.EffectChannel = DefaultPercussionChannel
	}
	if channel.Bank != 0 || channel.Chorus != 0 || channel.Reverb != 0 || channel.Phaser != 0 || channel.Tremolo != 0 {
		builder.addReport("gp8.omit.midi-effects", "score-core", ExportDispositionOmitted, location, "GP8 writer emits channel, program, volume, and balance but not the legacy bank and effect controllers")
	}
	if track.ChannelIndex == -1 {
		builder.addReport("gp8.normalize.channel-binding", "score-core", ExportDispositionNormalized, location, "GP8 output binds an unbound track to a default channel")
	}
	if channel.Channel/16 != channel.EffectChannel/16 {
		builder.addReport("gp8.normalize.effect-channel-port", "score-core", ExportDispositionNormalized, location, "GP8 stores one port for both primary and effect channels")
	}

	red := (uint32(track.Color) >> 16) & 0xff
	green := (uint32(track.Color) >> 8) & 0xff
	blue := uint32(track.Color) & 0xff
	primaryChannel := int(channel.Channel % 16)
	result := gpifTrack{
		ID:    strconv.Itoa(trackIndex),
		Name:  track.Name,
		Color: fmt.Sprintf("%d %d %d", red, green, blue),
		Sounds: gpifSounds{Sounds: []gpifSound{{
			Name:    track.Name,
			Program: int(channel.Instrument),
			Channel: &primaryChannel,
		}}},
		RSE: &gpifTrackRSE{ChannelStrip: gpifChannelStrip{
			Parameters: gp8ChannelStripParameters(channel),
		}},
		MidiConnection: gpifMidiConnection{
			Port:             int(channel.Channel) / 16,
			PrimaryChannel:   int(channel.Channel) % 16,
			SecondaryChannel: int(channel.EffectChannel) % 16,
		},
		PlaybackState:    "Default",
		AudioEngineState: "MIDI",
	}
	if track.Offset != 0 {
		capo := int(track.Offset)
		result.Properties = append(result.Properties, gpifStaffProperty{Name: "CapoFret", Fret: &capo})
	}
	if len(track.Sounds) > 0 {
		result.Sounds.Sounds = make([]gpifSound, 0, len(track.Sounds))
		for _, sound := range track.Sounds {
			result.Sounds.Sounds = append(result.Sounds.Sounds, gpifSound{
				Name: sound.Name, Label: sound.Label, Path: sound.Path, Role: sound.Role,
				Program: int(sound.Program), Channel: &primaryChannel,
			})
		}
	}
	for _, automation := range track.SoundAutomations {
		sound := track.Sounds[automation.Sound]
		result.Automations.Automations = append(result.Automations.Automations, gpifAutomation{
			Type: "Sound", Value: gpifAutomationValue{Text: sound.Path + ";" + sound.Name + ";" + sound.Role},
			Visible: "true", Bar: automation.Bar, Position: automation.Position,
		})
	}
	if len(track.Lyrics) > 0 {
		result.Lyrics = &gpifLyrics{Dispatched: true, Lines: make([]gpifLyricLine, 0, len(track.Lyrics))}
		for _, line := range track.Lyrics {
			result.Lyrics.Lines = append(result.Lyrics.Lines, gpifLyricLine(line))
		}
	}
	switch {
	case track.Mute:
		result.PlaybackState = "Mute"
	case track.Solo:
		result.PlaybackState = "Solo"
	}

	staves := gp8ExportStaves(track)
	percussionLineCount := 5
	if track.PercussionTrack && len(staves) > 0 && staves[0].StandardNotationLineCount > 0 {
		percussionLineCount = staves[0].StandardNotationLineCount
	}
	for staffIndex := range staves {
		if !track.PercussionTrack && staves[staffIndex].StandardNotationLineCount != 0 && staves[staffIndex].StandardNotationLineCount != 5 {
			builder.addReport("gp8.omit.staff-line-count", "staff-ownership", ExportDispositionOmitted, ScoreLocation{Track: trackIndex, Staff: staffIndex}, "GP8 writer emits custom staff line counts only for percussion tracks")
		}
		if track.PercussionTrack {
			lineCount := staves[staffIndex].StandardNotationLineCount
			if lineCount <= 0 {
				lineCount = 5
			}
			if lineCount != percussionLineCount {
				builder.addReport("gp8.normalize.percussion-line-count", "staff-ownership", ExportDispositionNormalized, ScoreLocation{Track: trackIndex, Staff: staffIndex}, "GP8 stores one notation line count for every percussion staff in a track")
			}
		}
		for stringIndex, guitarString := range staves[staffIndex].Strings {
			if guitarString.Number != int8(stringIndex+1) {
				builder.addReport("gp8.normalize.string-number", "staff-ownership", ExportDispositionNormalized, ScoreLocation{Track: trackIndex, Staff: staffIndex}, "GP8 derives string numbers from tuning order")
				break
			}
		}
	}
	result.Staves.Staff = make([]gpifStaff, len(staves))
	for staffIndex := range staves {
		staff := &staves[staffIndex]
		if len(staff.Strings) == 0 {
			continue
		}
		pitches := make([]string, 0, len(staff.Strings))
		for index := len(staff.Strings) - 1; index >= 0; index-- {
			pitches = append(pitches, strconv.Itoa(int(staff.Strings[index].Value)))
		}
		result.Staves.Staff[staffIndex].Properties = []gpifStaffProperty{{
			Name: "Tuning", Pitches: strings.Join(pitches, " "),
		}}
	}
	if len(builder.chordIDs[trackIndex]) > 0 {
		items := make([]gpifItem, 0, len(builder.chordIDs[trackIndex]))
		for chord, id := range builder.chordIDs[trackIndex] {
			items = append(items, gp8ChordItem(id, chord, len(staves[0].Strings)))
		}
		slices.SortFunc(items, func(a, b gpifItem) int { return cmp.Compare(a.ID, b.ID) })
		result.Properties = append(result.Properties, gpifStaffProperty{Name: "DiagramCollection", Items: &gpifItems{Items: items}})
	}

	if track.PercussionTrack {
		result.InstrumentSet = &gpifInstrumentSet{
			Name:      "Drums",
			Type:      "drumKit",
			LineCount: percussionLineCount,
			Elements:  gpifElements{Elements: builder.percussionElements[trackIndex]},
		}
	} else {
		name, instrumentType := gp8PitchedInstrumentSet(channel.Instrument)
		result.InstrumentSet = &gpifInstrumentSet{
			Name:      name,
			Type:      instrumentType,
			LineCount: 5,
			Elements: gpifElements{Elements: []gpifElement{{
				Name: "Pitched",
				Type: "pitched",
				Articulations: gpifArticulations{Articulations: []gpifArticulation{{
					StaffLine:          0,
					Noteheads:          "noteheadBlack noteheadHalf noteheadWhole",
					TechniquePlacement: "outside",
				}}},
			}}},
		}
	}
	return result
}

func gp8PitchedInstrumentSet(program int32) (string, string) {
	name := "Acoustic Piano"
	switch program {
	case 2, 4, 5:
		name = "Electric Piano"
	case 6, 7:
		name = "Harpsichord"
	case 8, 113, 119:
		name = "Celesta"
	case 9, 10, 11, 14, 114:
		name = "Vibraphone"
	case 12, 13, 108, 112, 115, 116, 117, 118:
		name = "Xylophone"
	case 15, 104, 105, 107:
		name = "Banjo"
	case 16, 17, 18, 19, 20, 21, 23:
		name = "Electric Organ"
	case 22, 74, 76, 78, 121, 122, 123, 124, 125, 126:
		name = "Recorder"
	case 24:
		name = "Nylon Guitar"
	case 25, 120:
		name = "Steel Guitar"
	case 26, 27, 28, 29, 30, 31:
		name = "Electric Guitar"
	case 32, 35:
		name = "Acoustic Bass"
	case 33, 34, 36, 37:
		name = "Electric Bass"
	case 38, 39:
		name = "Synth Bass"
	case 40, 44, 45, 48, 49, 50, 51, 110:
		name = "Violin"
	case 41:
		name = "Viola"
	case 42:
		name = "Cello"
	case 43:
		name = "Contrabass"
	case 46:
		name = "Harp"
	case 47, 127:
		name = "Timpani"
	case 52, 53, 54:
		name = "Voice"
	case 55, 88, 89, 90, 91, 92, 93, 94, 95, 96, 97, 98, 99:
		name = "Pad Synthesizer"
	case 56, 59, 61, 62, 63, 103:
		name = "Trumpet"
	case 57:
		name = "Trombone"
	case 58:
		name = "Tuba"
	case 60:
		name = "French Horn"
	case 64, 65, 66, 67:
		name = "Saxophone"
	case 68:
		name = "Oboe"
	case 69:
		name = "English Horn"
	case 70, 109:
		name = "Bassoon"
	case 71:
		name = "Clarinet"
	case 72:
		name = "Piccolo"
	case 73, 75, 77, 79, 111:
		name = "Flute"
	case 80, 81, 82, 83, 84, 85, 86, 87, 100, 101, 102:
		name = "Lead Synthesizer"
	case 106:
		name = "Ukulele"
	}
	parts := strings.Fields(name)
	parts[0] = strings.ToLower(parts[0][:1]) + parts[0][1:]
	return name, strings.Join(parts, "")
}

func gp8ChannelStripParameters(channel MidiChannel) string {
	values := []string{"0.5", "0.5", "0.5", "0.5", "0.5", "0.5", "0.5", "0.5", "0.5", "0", "0.5", "0.5", "0.5", "0.5", "0.5", "0.5"}
	values[11] = strconv.FormatFloat(float64(channel.Balance)/127, 'f', 6, 64)
	values[12] = strconv.FormatFloat(float64(channel.Volume)/127, 'f', 6, 64)
	return strings.Join(values, " ")
}

func gp8ChordItem(id string, chord *Chord, fallbackStringCount int) gpifItem {
	stringCount := len(chord.Strings)
	if stringCount == 0 {
		stringCount = fallbackStringCount
	}
	baseFret := 0
	if chord.FirstFret != nil && *chord.FirstFret > 0 {
		baseFret = int(*chord.FirstFret) - 1
	}
	diagram := &gpifDiagram{StringCount: stringCount, FretCount: 5, BaseFret: baseFret}
	for index, fret := range chord.Strings {
		if fret >= 0 {
			diagram.Frets = append(diagram.Frets, gpifDiagramFret{
				String: stringCount - index - 1,
				Fret:   int(fret) - baseFret,
			})
		}
	}
	return gpifItem{ID: id, Name: chord.Name, Diagram: diagram, Chord: &struct{}{}}
}

func (builder *gp8Builder) buildScoreGraph() error {
	for measureIndex := range builder.song.MeasureHeaders {
		header := &builder.song.MeasureHeaders[measureIndex]
		headerLocation := ScoreLocation{Measure: measureIndex}
		if header.Marker != nil && header.Marker.Color != 0 {
			builder.addReport("gp8.omit.marker-color", "score-core", ExportDispositionOmitted, headerLocation, "GP8 writer emits section text but not marker color")
		}
		if header.Direction != nil {
			builder.addReport("gp8.omit.measure-direction", "score-core", ExportDispositionOmitted, headerLocation, "GP8 writer does not emit legacy navigation directions")
		}
		if header.Tempo != 0 {
			builder.addReport("gp8.omit.measure-tempo", "tempo-automations", ExportDispositionOmitted, headerLocation, "GP8 writer does not emit the legacy measure-header tempo field")
		}
		if header.TimeSignature.Beams != defaultTimeSignature().Beams {
			builder.addReport("gp8.omit.time-signature-beams", "rhythm", ExportDispositionOmitted, headerLocation, "GP8 writer emits the meter but not its authored beam grouping")
		}
		barIDs := make([]string, 0, len(builder.song.Tracks))
		for trackIndex := range builder.song.Tracks {
			track := &builder.song.Tracks[trackIndex]
			staves := gp8ExportStaves(track)
			for staffIndex := range staves {
				staff := &staves[staffIndex]
				location := ScoreLocation{Track: trackIndex, Staff: staffIndex, Measure: measureIndex}
				barID := strconv.Itoa(len(builder.doc.Bars.Bars))
				barIDs = append(barIDs, barID)
				measure := &staff.Measures[measureIndex]
				if measure.LineBreak != LineBreakNone {
					builder.addReport("gp8.omit.measure-line-break", "score-core", ExportDispositionOmitted, location, "GP8 writer does not emit measure line-break preferences")
				}
				if measure.HasDoubleBar != header.DoubleBar {
					builder.addReport("gp8.normalize.measure-double-bar-authority", "score-core", ExportDispositionNormalized, location, "GP8 writer uses the master-bar double-bar value instead of the compatibility measure value")
				}
				if measure.KeySignature != (KeySignature{}) && measure.KeySignature != header.KeySignature {
					builder.addReport("gp8.normalize.measure-key-authority", "score-core", ExportDispositionNormalized, location, "GP8 writer uses the master-bar key instead of the compatibility measure value")
				}
				if measure.TimeSignature != (TimeSignature{}) && measure.TimeSignature != header.TimeSignature {
					builder.addReport("gp8.normalize.measure-time-authority", "rhythm", ExportDispositionNormalized, location, "GP8 writer uses the master-bar time signature instead of the compatibility measure value")
				}
				voiceIDs := make([]string, 0, 4)
				for voiceIndex := range measure.Voices {
					voice := &measure.Voices[voiceIndex]
					if voice.Direction != VoiceDirectionNone {
						voiceLocation := location
						voiceLocation.Voice = voiceIndex
						builder.addReport("gp8.omit.voice-direction", "score-core", ExportDispositionOmitted, voiceLocation, "GP8 writer does not emit the voice beam direction")
					}
					if len(voice.Beats) == 0 {
						voiceIDs = append(voiceIDs, "-1")
						continue
					}
					voiceID := strconv.Itoa(len(builder.doc.Voices.Voices))
					voiceIDs = append(voiceIDs, voiceID)
					beatIDs := make([]string, 0, len(voice.Beats))
					for beatIndex := range voice.Beats {
						location := ScoreLocation{Track: trackIndex, Staff: staffIndex, Measure: measureIndex, Voice: voiceIndex, Beat: beatIndex}
						builder.reportBeatConversion(&voice.Beats[beatIndex], location)
						graceIDs, err := builder.addGraceBeats(trackIndex, staff.Strings, &voice.Beats[beatIndex])
						if err != nil {
							return fmt.Errorf("track %d staff %d measure %d voice %d beat %d grace notes: %w", trackIndex, staffIndex, measureIndex, voiceIndex, beatIndex, err)
						}
						beatIDs = append(beatIDs, graceIDs...)
						beatID, err := builder.addBeat(trackIndex, staff.Strings, &voice.Beats[beatIndex])
						if err != nil {
							return fmt.Errorf("track %d staff %d measure %d voice %d beat %d: %w", trackIndex, staffIndex, measureIndex, voiceIndex, beatIndex, err)
						}
						beatIDs = append(beatIDs, beatID)
					}
					builder.doc.Voices.Voices = append(builder.doc.Voices.Voices, gpifVoice{ID: voiceID, Beats: strings.Join(beatIDs, " ")})
				}
				for len(voiceIDs) < 4 {
					voiceIDs = append(voiceIDs, "-1")
				}
				builder.doc.Bars.Bars = append(builder.doc.Bars.Bars, gpifBar{ID: barID, Voices: strings.Join(voiceIDs, " "), Clef: gp8BarClef(builder.song, trackIndex, measure)})
			}
		}
		builder.doc.MasterBars.MasterBars = append(builder.doc.MasterBars.MasterBars, gp8MasterBar(header, strings.Join(barIDs, " ")))
	}
	return nil
}

func (builder *gp8Builder) addReport(code, feature string, disposition ExportDisposition, location ScoreLocation, reason string) {
	if builder.report == nil {
		return
	}
	builder.report.Entries = append(builder.report.Entries, ExportReportEntry{
		Code: code, Feature: feature, Disposition: disposition, Location: location, Reason: reason,
	})
}

func (builder *gp8Builder) reportBeatConversion(beat *Beat, location ScoreLocation) {
	if beat.Duration.Dotted && beat.Duration.DoubleDotted {
		builder.addReport("gp8.normalize.duration-dot-flags", "rhythm", ExportDispositionNormalized, location, "GP8 emits the double-dot value when both legacy dot flags are set")
	}
	if beat.Duration.TupletEnters == 0 && beat.Duration.TupletTimes == 0 {
		builder.addReport("gp8.normalize.tuplet-default", "rhythm", ExportDispositionNormalized, location, "GP8 emits a missing tuplet as the canonical 1:1 legacy ratio")
	}
	if beat.Display != (BeatDisplay{}) {
		builder.addReport("gp8.omit.beat-display", "score-core", ExportDispositionOmitted, location, "GP8 writer does not emit beat beam and tuplet display overrides")
	}
	if beat.Status == BeatStatusEmpty {
		builder.addReport("gp8.normalize.empty-beat", "note-and-beat-semantics", ExportDispositionNormalized, location, "GP8 writer emits an explicit empty beat as a rest")
	}
	dynamic := gp8ConvertBeatDynamic(beat)
	if dynamic.authoredNormalized {
		builder.addReport("gp8.normalize.beat-dynamic", "note-and-beat-semantics", ExportDispositionNormalized, location, "GPIF stores the authored beat dynamic as one of eight canonical markings")
	}
	if dynamic.noteVelocitiesNormalized {
		builder.addReport("gp8.normalize.note-velocity", "note-and-beat-semantics", ExportDispositionNormalized, location, "GPIF stores one quantized dynamic for all notes in a beat")
	}
	if beat.Effect.MixTableChange != nil {
		builder.addReport("gp8.omit.beat-mix-table-change", "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 writer does not emit beat-local mix-table changes")
	}
	if chord := beat.Effect.Chord; chord != nil {
		if len(chord.Barres) != 0 {
			builder.addReport("gp8.omit.chord-barres", "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 writer does not emit explicit chord barre ranges")
		}
		if len(chord.Fingerings) != 0 {
			builder.addReport("gp8.omit.chord-fingerings", "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 writer does not emit authored chord finger assignments")
		}
		if len(chord.Omissions) != 0 {
			builder.addReport("gp8.omit.chord-omissions", "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 writer does not emit legacy chord interval-omission flags")
		}
		if chord.Root != nil || chord.Bass != nil || chord.Kind != nil || chord.Extension != nil || chord.Fifth != nil || chord.Ninth != nil || chord.Eleventh != nil || chord.Tonality != nil || chord.Add != nil || chord.Sharp != nil || chord.NewFormat != nil || chord.Show != nil {
			builder.addReport("gp8.omit.chord-legacy-details", "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 writer emits the diagram but not the authored legacy chord spelling, quality, alteration, format, or display fields")
		}
		if chord.FirstFret != nil && *chord.FirstFret == 0 {
			builder.addReport("gp8.normalize.chord-first-fret", "note-and-beat-semantics", ExportDispositionNormalized, location, "GPIF represents an explicit chord first fret 0 as first fret 1")
		}
		if chord.Length != 0 && len(chord.Strings) != 0 && int(chord.Length) != len(chord.Strings) {
			builder.addReport("gp8.normalize.chord-length", "note-and-beat-semantics", ExportDispositionNormalized, location, "GPIF derives chord diagram length from the number of string states")
		}
	}
	if beat.Effect.HasRasgueado {
		builder.addReport("gp8.omit.rasgueado", "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 writer does not emit rasgueado")
	}
	if beat.Effect.PickStroke != BeatStrokeDirectionNone {
		builder.addReport("gp8.omit.pick-stroke", "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 writer does not emit pick-stroke direction")
	}
	if beat.Effect.SlapEffect != SlapEffectNone {
		builder.addReport("gp8.omit.slap-effect", "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 writer does not emit slap, pop, or tap effects")
	}
	if beat.Effect.Vibrato {
		builder.addReport("gp8.omit.beat-vibrato", "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 writer does not emit beat-wide vibrato")
	}
	if beat.Effect.Stroke.Direction != BeatStrokeDirectionNone && beat.Effect.Stroke.Value != uint16(DurationEighth) {
		builder.addReport("gp8.normalize.stroke-duration", "note-and-beat-semantics", ExportDispositionNormalized, location, "GP8 writer emits the stroke with an eighth-note duration")
	}
	if whammy := beat.Effect.TremoloBar; whammy != nil {
		conversion := gp8ConvertWhammy(whammy)
		builder.reportCurveConversion(whammy, conversion.omitted, conversion.normalized, location, curveReportSpec{
			omittedCode: "gp8.omit.whammy-curve", normalizedCode: "gp8.normalize.whammy-curve",
			summaryCode: "gp8.omit.whammy-summary", vibratoCode: "gp8.omit.whammy-point-vibrato", name: "whammy",
		})
	}
	for noteIndex := range beat.Notes {
		noteLocation := location
		noteLocation.Note = noteIndex
		builder.reportNoteConversion(&beat.Notes[noteIndex], noteLocation)
	}
}

func (builder *gp8Builder) reportNoteConversion(note *Note, location ScoreLocation) {
	if note.SwapAccidentals {
		builder.addReport("gp8.omit.swap-accidentals", "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 writer does not emit the legacy accidental-swap preference")
	}
	if note.DurationPercent != 1 {
		builder.addReport("gp8.omit.note-duration-percent", "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 has no note duration-percent field")
	}
	if note.Effect.TremoloPicking != nil {
		builder.addReport("gp8.omit.tremolo-picking", "tremolo-picking", ExportDispositionOmitted, location, "GP8 writer does not emit tremolo picking")
	}
	if _, conflict := gp8ResolveNoteAccent(note.Effect); conflict {
		builder.addReport("gp8.normalize.note-accent-authority", "note-and-beat-semantics", ExportDispositionNormalized, location, "the typed note accent takes precedence over conflicting legacy accent booleans")
	}
	if note.Effect.HasLeftHandFinger || (note.Effect.LeftHandFinger != FingeringOpen && note.Effect.LeftHandFinger != FingeringThumb) {
		builder.addReport("gp8.omit.left-hand-fingering", "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 writer does not emit left-hand fingering")
	}
	if note.Effect.HasRightHandFinger || (note.Effect.RightHandFinger != FingeringOpen && note.Effect.RightHandFinger != FingeringThumb) {
		builder.addReport("gp8.omit.right-hand-fingering", "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 writer does not emit right-hand fingering")
	}
	if bend := note.Effect.Bend; bend != nil {
		conversion := gp8ConvertBend(bend)
		builder.reportCurveConversion(bend, conversion.omitted, conversion.normalized, location, curveReportSpec{
			omittedCode: "gp8.omit.bend-curve", normalizedCode: "gp8.normalize.bend-curve",
			summaryCode: "gp8.omit.bend-summary", vibratoCode: "gp8.omit.bend-point-vibrato", name: "bend",
		})
	}
	if trill := note.Effect.Trill; trill != nil {
		canonical := defaultDuration()
		canonical.Value = uint16(DurationSixteenth)
		if trill.Duration != canonical {
			builder.addReport("gp8.normalize.trill-duration", "note-and-beat-semantics", ExportDispositionNormalized, location, "GP8 writer emits the trill with a sixteenth-note duration")
		}
	}
	if harmonic := note.Effect.Harmonic; harmonic != nil {
		if harmonic.Pitch != nil || harmonic.Octave != nil {
			builder.addReport("gp8.omit.harmonic-pitch", "harmonics", ExportDispositionOmitted, location, "GP8 writer emits harmonic kind and fret but not pitch or octave fields")
		}
		if harmonic.Fret != nil && harmonic.FretFloat != nil && int8(*harmonic.FretFloat) != *harmonic.Fret {
			builder.addReport("gp8.normalize.harmonic-fret-authority", "harmonics", ExportDispositionNormalized, location, "the exact harmonic fret takes precedence over its conflicting legacy integer view")
		}
	}
	graceDurationChanged := false
	graceFretChanged := false
	graceRawFretOmitted := false
	graceVelocityChanged := false
	graceBendOmitted := false
	for _, grace := range note.Effect.Graces {
		if _, supported := gp8NoteValue(uint16(grace.Duration)); !supported {
			graceDurationChanged = true
		}
		if grace.ExactFret != nil && int64(*grace.ExactFret) != int64(grace.Fret) {
			graceFretChanged = true
		}
		if grace.RawFret != nil {
			graceRawFretOmitted = true
		}
		velocity := grace.Velocity
		if velocity == 0 {
			velocity = DefaultVelocity
		}
		if grace.Velocity == 0 || gpifDynamicToVelocity(gp8VelocityToDynamic(velocity)) != velocity {
			graceVelocityChanged = true
		}
		if grace.Transition == GraceEffectTransitionBend {
			graceBendOmitted = true
		}
	}
	if graceDurationChanged {
		builder.addReport("gp8.normalize.grace-duration", "grace-relationships", ExportDispositionNormalized, location, "GP8 uses a thirty-second note for an unsupported grace duration")
	}
	if graceFretChanged {
		builder.addReport("gp8.normalize.grace-fret-authority", "grace-relationships", ExportDispositionNormalized, location, "ExactFret is authoritative when it conflicts with the legacy grace Fret")
	}
	if graceRawFretOmitted {
		builder.addReport("gp8.omit.grace-raw-fret", "grace-relationships", ExportDispositionOmitted, location, "GP8 does not preserve the source-format raw grace fret byte")
	}
	if graceVelocityChanged {
		builder.addReport("gp8.normalize.grace-velocity", "grace-relationships", ExportDispositionNormalized, location, "GPIF stores grace velocity as one canonical dynamic")
	}
	if graceBendOmitted {
		builder.addReport("gp8.omit.grace-bend-transition", "grace-relationships", ExportDispositionOmitted, location, "GP8 writer does not emit a bend transition from a grace note")
	}
}

type curveReportSpec struct {
	omittedCode    string
	normalizedCode string
	summaryCode    string
	vibratoCode    string
	name           string
}

func (builder *gp8Builder) reportCurveConversion(effect *BendEffect, omitted, normalized bool, location ScoreLocation, spec curveReportSpec) {
	if omitted {
		builder.addReport(spec.omittedCode, "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 writer supports "+spec.name+" curves with two through four non-collinear points")
	}
	if normalized {
		builder.addReport(spec.normalizedCode, "note-and-beat-semantics", ExportDispositionNormalized, location, "GPIF uses one value for both middle "+spec.name+" points")
	}
	if effect.Kind != BendTypeNone || effect.Value != 0 {
		builder.addReport(spec.summaryCode, "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 writer derives the "+spec.name+" from points and omits the summary fields")
	}
	for _, point := range effect.Points {
		if point.Vibrato {
			builder.addReport(spec.vibratoCode, "note-and-beat-semantics", ExportDispositionOmitted, location, "GP8 writer does not emit "+spec.name+"-point vibrato")
			break
		}
	}
}

func gp8MasterBar(header *MeasureHeader, bars string) gpifMasterBar {
	mode := "Major"
	if header.KeySignature.IsMinor {
		mode = "Minor"
	}
	result := gpifMasterBar{
		Key:  gpifKey{Mode: mode, AccidentalCount: int(header.KeySignature.Key)},
		Time: fmt.Sprintf("%d/%d", header.TimeSignature.Numerator, header.TimeSignature.Denominator.Value),
		Bars: bars,
	}
	if header.Marker != nil {
		result.Section = &gpifSection{Text: header.Marker.Title}
	}
	if header.RepeatOpen || header.RepeatClose >= 0 {
		result.Repeat = &gpifRepeat{}
	}
	if header.RepeatOpen {
		result.Repeat.Start = "true"
	}
	if header.RepeatClose >= 0 {
		result.Repeat.End = "true"
		result.Repeat.Count = int(header.RepeatClose) + 1
	}
	if header.RepeatAlternative != 0 {
		endings := make([]string, 0, 8)
		for index := range 8 {
			if header.RepeatAlternative&(1<<index) != 0 {
				endings = append(endings, strconv.Itoa(index+1))
			}
		}
		result.AlternateEndings = strings.Join(endings, " ")
	}
	if header.DoubleBar {
		result.DoubleBar = &struct{}{}
	}
	switch header.TripletFeel {
	case TripletFeelEighth:
		result.TripletFeel = "Triplet8th"
	case TripletFeelSixteenth:
		result.TripletFeel = "Triplet16th"
	case TripletFeelDottedEighth:
		result.TripletFeel = "Dotted8th"
	case TripletFeelDottedSixteenth:
		result.TripletFeel = "Dotted16th"
	case TripletFeelScottishEighth:
		result.TripletFeel = "Scottish8th"
	case TripletFeelScottishSixteenth:
		result.TripletFeel = "Scottish16th"
	}
	return result
}

type gp8GraceGroup struct {
	duration Duration
	velocity int16
	onBeat   bool
	notes    []Note
}

func (builder *gp8Builder) addGraceBeats(trackIndex int, staffStrings []GuitarString, beat *Beat) ([]string, error) {
	var ids []string
	for _, sequence := range graceSequences(beat) {
		groups := builder.graceGroups(trackIndex, beat, sequence)
		groupIDs, err := builder.addGraceGroups(trackIndex, staffStrings, groups)
		if err != nil {
			return nil, err
		}
		ids = append(ids, groupIDs...)
	}
	return ids, nil
}

func graceSequences(beat *Beat) []uint8 {
	seen := make(map[uint8]struct{})
	for noteIndex := range beat.Notes {
		for _, grace := range beat.Notes[noteIndex].Effect.Graces {
			seen[grace.Sequence] = struct{}{}
		}
	}
	result := make([]uint8, 0, len(seen))
	for sequence := range seen {
		result = append(result, sequence)
	}
	slices.Sort(result)
	return result
}

func (builder *gp8Builder) graceGroups(trackIndex int, beat *Beat, sequence uint8) []gp8GraceGroup {
	var groups []gp8GraceGroup
	for noteIndex := range beat.Notes {
		note := &beat.Notes[noteIndex]
		for graceIndex := range note.Effect.Graces {
			grace := &note.Effect.Graces[graceIndex]
			if grace.Sequence != sequence {
				continue
			}
			duration := defaultDuration()
			duration.Value = uint16(grace.Duration)
			if _, supported := gp8NoteValue(duration.Value); !supported {
				duration.Value = uint16(DurationThirtySecond)
			}
			velocity := grace.Velocity
			if velocity == 0 {
				velocity = DefaultVelocity
			}
			groupIndex := -1
			for index := range groups {
				if groups[index].duration == duration && groups[index].velocity == velocity && groups[index].onBeat == grace.IsOnBeat {
					groupIndex = index
					break
				}
			}
			if groupIndex < 0 {
				groups = append(groups, gp8GraceGroup{duration: duration, velocity: velocity, onBeat: grace.IsOnBeat})
				groupIndex = len(groups) - 1
			}
			graceNote := *note
			graceNote.Value = int16(grace.Fret)
			if grace.ExactFret != nil {
				graceNote.Value = int16(*grace.ExactFret)
			}
			graceNote.PercussionArticulation = grace.PercussionArticulation
			graceNote.HasPercussionArticulation = grace.HasPercussionArticulation
			if builder.song.Tracks[trackIndex].PercussionTrack && (graceNote.Value < 27 || graceNote.Value > 87) {
				graceNote.Value = note.Value
			}
			graceNote.Velocity = velocity
			graceNote.Kind = NoteTypeNormal
			graceNote.Effect = defaultNoteEffect()
			if grace.IsDead {
				graceNote.Kind = NoteTypeDead
			}
			switch grace.Transition {
			case GraceEffectTransitionSlide:
				graceNote.Effect.Slides = []SlideType{SlideLegatoSlideTo}
			case GraceEffectTransitionHammer:
				graceNote.Effect.Hammer = true
			}
			groups[groupIndex].notes = append(groups[groupIndex].notes, graceNote)
		}
	}
	return groups
}

func (builder *gp8Builder) addGraceGroups(trackIndex int, staffStrings []GuitarString, groups []gp8GraceGroup) ([]string, error) {
	ids := make([]string, 0, len(groups))
	for _, group := range groups {
		rhythmID, err := builder.addRhythm(group.duration)
		if err != nil {
			return nil, err
		}
		id := strconv.Itoa(len(builder.doc.Beats.Beats))
		graceKind := "BeforeBeat"
		if group.onBeat {
			graceKind = "OnBeat"
		}
		result := gpifBeat{
			ID:         id,
			Rhythm:     gpifRhythmRef{Ref: rhythmID},
			Dynamic:    gp8VelocityToDynamic(group.velocity),
			GraceNotes: graceKind,
		}
		noteIDs := make([]string, 0, len(group.notes))
		for noteIndex := range group.notes {
			noteID, err := builder.addNote(trackIndex, staffStrings, &group.notes[noteIndex])
			if err != nil {
				return nil, err
			}
			noteIDs = append(noteIDs, noteID)
		}
		result.Notes = strings.Join(noteIDs, " ")
		builder.doc.Beats.Beats = append(builder.doc.Beats.Beats, result)
		ids = append(ids, id)
	}
	return ids, nil
}

func (builder *gp8Builder) addBeat(trackIndex int, staffStrings []GuitarString, beat *Beat) (string, error) {
	rhythmID, err := builder.addRhythm(beat.Duration)
	if err != nil {
		return "", err
	}
	beatID := strconv.Itoa(len(builder.doc.Beats.Beats))
	result := gpifBeat{ID: beatID, Rhythm: gpifRhythmRef{Ref: rhythmID}, FreeText: beat.Text}
	if beat.isGrace {
		result.GraceNotes = "BeforeBeat"
		if beat.graceOnBeat {
			result.GraceNotes = "OnBeat"
		}
	}
	result.Dynamic = gp8ConvertBeatDynamic(beat).marking
	if beat.Effect.Chord != nil {
		result.Chord = builder.chordIDs[trackIndex][beat.Effect.Chord]
	}
	if beat.Effect.FadeIn {
		result.Fadding = "FadeIn"
	}
	result.Whammy = gp8Whammy(beat.Effect.TremoloBar)
	switch beat.Effect.Hairpin {
	case HairpinCrescendo:
		result.Hairpin = "Crescendo"
	case HairpinDiminuendo:
		result.Hairpin = "Decrescendo"
	}
	switch beat.Effect.Stroke.Direction {
	case BeatStrokeDirectionUp:
		result.Arpeggio = "Up"
	case BeatStrokeDirectionDown:
		result.Arpeggio = "Down"
	}
	switch beat.Octave {
	case OctaveOttava:
		result.Ottavia = "8va"
	case OctaveOttavaBassa:
		result.Ottavia = "8vb"
	case OctaveQuindicesima:
		result.Ottavia = "15ma"
	case OctaveQuindicesimaBassa:
		result.Ottavia = "15mb"
	}

	noteIDs := make([]string, 0, len(beat.Notes))
	for noteIndex := range beat.Notes {
		noteID, err := builder.addNote(trackIndex, staffStrings, &beat.Notes[noteIndex])
		if err != nil {
			return "", err
		}
		noteIDs = append(noteIDs, noteID)
	}
	result.Notes = strings.Join(noteIDs, " ")
	builder.doc.Beats.Beats = append(builder.doc.Beats.Beats, result)
	return beatID, nil
}

func (builder *gp8Builder) addRhythm(duration Duration) (string, error) {
	if id, ok := builder.rhythmIDs[duration]; ok {
		return id, nil
	}
	noteValue, ok := gp8NoteValue(duration.Value)
	if !ok {
		return "", fmt.Errorf("unsupported duration value %d", duration.Value)
	}
	id := strconv.Itoa(len(builder.doc.Rhythms.Rhythms))
	rhythm := gpifRhythm{ID: id, NoteValue: noteValue}
	switch {
	case duration.DoubleDotted:
		rhythm.AugmentationDot = &gpifAugDot{Count: 2}
	case duration.Dotted:
		rhythm.AugmentationDot = &gpifAugDot{Count: 1}
	}
	if duration.TupletEnters > 0 && duration.TupletTimes > 0 && (duration.TupletEnters != 1 || duration.TupletTimes != 1) {
		rhythm.PrimaryTuplet = &gpifTuplet{Num: int(duration.TupletEnters), Den: int(duration.TupletTimes)}
	}
	builder.rhythmIDs[duration] = id
	builder.doc.Rhythms.Rhythms = append(builder.doc.Rhythms.Rhythms, rhythm)
	return id, nil
}

func (builder *gp8Builder) addNote(trackIndex int, staffStrings []GuitarString, note *Note) (string, error) {
	track := &builder.song.Tracks[trackIndex]
	noteID := strconv.Itoa(len(builder.doc.Notes.Notes))
	fret := int(note.Value)
	midi := gp8NoteMIDI(track, staffStrings, note)
	properties := []gpifProperty{
		{Name: "Fret", Fret: &fret},
		{Name: "Midi", Number: &midi},
	}
	if note.String > 0 {
		stringValue := float64(int(note.String) - 1)
		if !track.PercussionTrack && int(note.String) <= len(staffStrings) {
			stringValue = float64(len(staffStrings) - int(note.String))
		}
		properties = append(properties, gpifProperty{Name: "String", String: &stringValue})
	}
	if !track.PercussionTrack {
		pitch := gp8Pitch(midi)
		properties = append([]gpifProperty{{Name: "ConcertPitch", Pitch: &pitch}, {Name: "TransposedPitch", Pitch: &pitch}}, properties...)
	}
	articulation := 0
	result := gpifNote{ID: noteID, InstrumentArticulation: &articulation, Properties: gpifProperties{Properties: properties}}
	result.Properties.Properties = append(result.Properties.Properties, gp8ConvertBend(note.Effect.Bend).properties...)
	if note.Effect.Harmonic != nil {
		harmonicType := gp8HarmonicType(note.Effect.Harmonic.Kind)
		if harmonicType != "" {
			result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "HarmonicType", HType: &harmonicType})
		}
		var harmonicFret *float64
		if note.Effect.Harmonic.FretFloat != nil {
			harmonicFret = note.Effect.Harmonic.FretFloat
		} else if note.Effect.Harmonic.Fret != nil {
			value := float64(*note.Effect.Harmonic.Fret)
			harmonicFret = &value
		}
		if harmonicFret != nil {
			value := strconv.FormatFloat(*harmonicFret, 'f', -1, 64)
			result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "HarmonicFret", HFret: &value})
		}
	}
	if track.PercussionTrack {
		if index, ok := gp8PercussionArticulationIndex(track, note); ok {
			articulation = index
		} else {
			var ok bool
			articulation, ok = builder.articulationIDs[trackIndex][note.Value]
			if !ok {
				return "", fmt.Errorf("percussion MIDI value %d has no exported articulation resource", note.Value)
			}
		}
		result.InstrumentArticulation = &articulation
	}
	if note.TieOrigin || note.Kind == NoteTypeTie {
		result.Tie = &gpifTie{
			Origin:      strconv.FormatBool(note.TieOrigin),
			Destination: strconv.FormatBool(note.Kind == NoteTypeTie),
		}
	}
	if note.Kind == NoteTypeDead || note.Effect.DeadNote || note.Effect.GhostNote {
		if note.Kind == NoteTypeDead || note.Effect.DeadNote {
			enable := ""
			result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "Muted", Enable: &enable})
		}
		if note.Effect.GhostNote {
			result.AntiAccent = "Normal"
		}
	}
	if note.Effect.LetRing {
		value := ""
		result.LetRing = &value
	}
	if note.Effect.PalmMute {
		value := ""
		result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "PalmMuted", Enable: &value})
	}
	if note.Effect.Hammer {
		enable := ""
		result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "HopoOrigin", Enable: &enable})
	}
	if note.Effect.Tapped {
		enable := ""
		result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "Tapped", Enable: &enable})
	}
	if note.Effect.LeftHandTapped {
		enable := ""
		result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "LeftHandTapped", Enable: &enable})
	}
	if len(note.Effect.Slides) > 0 {
		flags := 0
		for _, slide := range note.Effect.Slides {
			switch slide {
			case SlideShiftSlideTo:
				flags |= 0x01
			case SlideLegatoSlideTo:
				flags |= 0x02
			case SlideOutDownwards:
				flags |= 0x04
			case SlideOutUpwards:
				flags |= 0x08
			case SlideIntoFromBelow:
				flags |= 0x10
			case SlideIntoFromAbove:
				flags |= 0x20
			case SlidePickSlideDown:
				flags |= 0x40
			case SlidePickSlideUp:
				flags |= 0x80
			}
		}
		value := strconv.Itoa(flags)
		result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "Slide", Flags: &value})
	}
	result.Vibrato = gp8ResolveNoteVibrato(note.Effect)
	if note.Effect.Trill != nil {
		result.Trill = &gpifTrill{Fret: int(note.Effect.Trill.Fret)}
	}
	if note.Effect.Staccato {
		result.Accent |= 0x01
	}
	accentFlags, _ := gp8ResolveNoteAccent(note.Effect)
	result.Accent |= accentFlags
	builder.doc.Notes.Notes = append(builder.doc.Notes.Notes, result)
	return noteID, nil
}

func gp8BarClef(song *Song, trackIndex int, measure *Measure) string {
	if song.Tracks[trackIndex].PercussionTrack {
		return "Neutral"
	}
	switch measure.Clef {
	case MeasureClefBass:
		return "F4"
	case MeasureClefTenor:
		return "C4"
	case MeasureClefAlto:
		return "C3"
	}
	track := &song.Tracks[trackIndex]
	if track.ChannelIndex >= 0 && track.ChannelIndex < len(song.Channels) {
		program := song.Channels[track.ChannelIndex].Instrument
		if program >= 32 && program <= 39 {
			return "F4"
		}
	}
	return "G2"
}

func gp8HarmonicType(kind HarmonicType) string {
	switch kind {
	case HarmonicTypeNatural:
		return "Natural"
	case HarmonicTypeArtificial:
		return "Artificial"
	case HarmonicTypePinch:
		return "Pinch"
	case HarmonicTypeTapped:
		return "Tap"
	case HarmonicTypeSemi:
		return "Semi"
	case HarmonicTypeFeedback:
		return "Feedback"
	default:
		return ""
	}
}

func gp8Whammy(bend *BendEffect) *gpifWhammy {
	return gp8ConvertWhammy(bend).whammy
}

type gp8WhammyConversion struct {
	whammy     *gpifWhammy
	omitted    bool
	normalized bool
}

func gp8ConvertWhammy(bend *BendEffect) gp8WhammyConversion {
	if bend == nil {
		return gp8WhammyConversion{}
	}
	points := simplifyBendPoints(slices.Clone(bend.Points))
	if len(points) < 2 || len(points) > 4 {
		return gp8WhammyConversion{omitted: len(points) != 0}
	}
	origin := points[0]
	destination := points[len(points)-1]
	middle1, middle2 := origin, destination
	switch len(points) {
	case 4:
		middle1, middle2 = points[1], points[2]
	case 3:
		middle1, middle2 = points[1], points[1]
	case 2:
		middle1 = BendPoint{
			Position: uint8((uint16(origin.Position) + uint16(destination.Position)) / 2),
			Value:    int8((int16(origin.Value) + int16(destination.Value)) / 2),
		}
		middle2 = middle1
	}
	whammy := &gpifWhammy{
		OriginValue:       gp8WhammyValue(origin.Value),
		MiddleValue:       gp8WhammyValue(middle1.Value),
		DestinationValue:  gp8WhammyValue(destination.Value),
		OriginOffset:      gp8WhammyOffset(origin.Position),
		MiddleOffset1:     gp8WhammyOffset(middle1.Position),
		MiddleOffset2:     gp8WhammyOffset(middle2.Position),
		DestinationOffset: gp8WhammyOffset(destination.Position),
	}
	encoded := []BendPoint{
		{Position: origin.Position, Value: origin.Value},
		{Position: middle1.Position, Value: middle1.Value},
		{Position: middle2.Position, Value: middle1.Value},
		{Position: destination.Position, Value: destination.Value},
	}
	return gp8WhammyConversion{
		whammy:     whammy,
		normalized: !slices.Equal(simplifyBendPoints(canonicalizeStandardWhammyPoints(simplifyBendPoints(encoded))), points),
	}
}

func gp8WhammyOffset(position uint8) string {
	return strconv.FormatFloat(float64(position)*100/float64(BendEffectMaxPosition), 'f', 6, 64)
}

func gp8WhammyValue(value int8) string {
	return strconv.FormatFloat(float64(value)*float64(GPBendSemitone), 'f', 6, 64)
}

type gp8BendConversion struct {
	properties []gpifProperty
	omitted    bool
	normalized bool
}

func gp8ConvertBend(bend *BendEffect) gp8BendConversion {
	if bend == nil {
		return gp8BendConversion{}
	}
	points := simplifyBendPoints(slices.Clone(bend.Points))
	if len(points) < 2 || len(points) > 4 {
		return gp8BendConversion{omitted: len(points) != 0}
	}
	origin := points[0]
	destination := points[len(points)-1]
	var middle1, middle2 BendPoint
	switch len(points) {
	case 4:
		if points[0].Value == points[1].Value && points[2].Value == points[3].Value {
			// GPIF keeps the destination value through the end of the note. Encode
			// an initial hold and release by ending the explicit curve where the
			// final value is reached.
			middle1 = points[1]
			middle2 = points[1]
			destination = points[2]
		} else {
			middle1 = points[1]
			middle2 = points[2]
		}
	case 3:
		middle1 = points[1]
		if points[1].Value == points[2].Value {
			// A destination before the end denotes a bend followed by a hold.
			destination = points[1]
			middle2 = points[2]
		} else {
			middle2 = points[1]
		}
	default:
		middle1 = BendPoint{
			Position: uint8((uint16(origin.Position) + uint16(destination.Position)) / 2),
			Value:    int8((int16(origin.Value) + int16(destination.Value)) / 2),
		}
		middle2 = middle1
	}
	enable := ""
	properties := []gpifProperty{
		{Name: "Bended", Enable: &enable},
		{Name: "BendDestinationOffset", Float: gp8BendOffset(destination.Position)},
		{Name: "BendDestinationValue", Float: gp8BendValue(destination.Value)},
		{Name: "BendMiddleOffset1", Float: gp8BendOffset(middle1.Position)},
		{Name: "BendMiddleOffset2", Float: gp8BendOffset(middle2.Position)},
		{Name: "BendMiddleValue", Float: gp8BendValue(middle1.Value)},
		{Name: "BendOriginOffset", Float: gp8BendOffset(origin.Position)},
		{Name: "BendOriginValue", Float: gp8BendValue(origin.Value)},
	}
	encoded := gpifBendProperties{
		enabled:                true,
		originPosition:         origin.Position,
		originValue:            origin.Value,
		middlePosition1:        middle1.Position,
		middlePosition2:        middle2.Position,
		middleValue:            middle1.Value,
		destinationPosition:    destination.Position,
		hasDestinationPosition: true,
		destinationValue:       destination.Value,
	}
	return gp8BendConversion{
		properties: properties,
		normalized: !slices.Equal(simplifyBendPoints(encoded.effect().Points), simplifyBendPoints(points)),
	}
}

func gp8BendOffset(position uint8) *string {
	value := strconv.FormatFloat(float64(position)*100/float64(BendEffectMaxPosition), 'f', 6, 64)
	return &value
}

func gp8BendValue(value int8) *string {
	encoded := strconv.FormatFloat(float64(value)*float64(GPBendSemitone), 'f', 6, 64)
	return &encoded
}

func gp8NoteMIDI(track *Track, staffStrings []GuitarString, note *Note) int {
	midi := int(note.Value)
	if !track.PercussionTrack && note.String > 0 && int(note.String) <= len(staffStrings) {
		midi += int(staffStrings[int(note.String)-1].Value)
	}
	return midi
}

func gp8Pitch(midi int) gpifPitch {
	steps := [...]string{"C", "C", "D", "D", "E", "F", "F", "G", "G", "A", "A", "B"}
	accidental := ""
	switch midi % 12 {
	case 1, 3, 6, 8, 10:
		accidental = "#"
	}
	return gpifPitch{Step: steps[midi%12], Accidental: accidental, Octave: midi/12 - 1}
}

func gp8NoteValue(value uint16) (string, bool) {
	switch value {
	case 1:
		return "Whole", true
	case 2:
		return "Half", true
	case 4:
		return "Quarter", true
	case 8:
		return "Eighth", true
	case 16:
		return "16th", true
	case 32:
		return "32nd", true
	case 64:
		return "64th", true
	case 128:
		return "128th", true
	default:
		return "", false
	}
}

func gp8VelocityToDynamic(velocity int16) string {
	index := int((velocity-MinVelocity+VelocityIncrement/2)/VelocityIncrement) + 1
	switch min(8, max(1, index)) {
	case 1:
		return "PPP"
	case 2:
		return "PP"
	case 3:
		return "P"
	case 4:
		return "MP"
	case 5:
		return "MF"
	case 6:
		return "F"
	case 7:
		return "FF"
	default:
		return "FFF"
	}
}

func gp8ResolveNoteVibrato(effect NoteEffect) string {
	strength := effect.VibratoStrength
	if strength == NoteVibratoNone && effect.Vibrato {
		strength = NoteVibratoSlight
	}
	switch strength {
	case NoteVibratoSlight:
		return "Slight"
	case NoteVibratoWide:
		return "Wide"
	default:
		return ""
	}
}

func gp8ResolveNoteAccent(effect NoteEffect) (int, bool) {
	legacyFlags := 0
	if effect.HeavyAccentuatedNote {
		legacyFlags |= 0x04
	}
	if effect.AccentuatedNote {
		legacyFlags |= 0x08
	}
	if effect.Accent == NoteAccentNone {
		return legacyFlags, false
	}
	typedFlags := 0
	switch effect.Accent {
	case NoteAccentNormal:
		typedFlags = 0x08
	case NoteAccentHeavy:
		typedFlags = 0x04
	case NoteAccentTenuto:
		typedFlags = 0x10
	}
	return typedFlags, legacyFlags != 0 && legacyFlags != typedFlags
}

type gp8BeatDynamicConversion struct {
	marking                  string
	velocity                 int16
	authoredNormalized       bool
	noteVelocitiesNormalized bool
}

func gp8ConvertBeatDynamic(beat *Beat) gp8BeatDynamicConversion {
	source := beat.Dynamics
	if source == 0 && len(beat.Notes) > 0 {
		source = beat.Notes[0].Velocity
	}
	if source == 0 {
		return gp8BeatDynamicConversion{}
	}
	marking := gp8VelocityToDynamic(source)
	target := gpifDynamicToVelocity(marking)
	conversion := gp8BeatDynamicConversion{
		marking:            marking,
		velocity:           target,
		authoredNormalized: beat.Dynamics != 0 && beat.Dynamics != target,
	}
	conversion.noteVelocitiesNormalized = slices.ContainsFunc(beat.Notes, func(note Note) bool {
		return note.Velocity != conversion.velocity
	})
	return conversion
}

func gp8DrumStaffLine(value int16) int {
	switch value {
	case 35:
		return 8
	case 36:
		return 7
	case 37, 38, 40:
		return 3
	case 41:
		return 5
	case 43:
		return 6
	case 45:
		return 5
	case 47:
		return 4
	case 48:
		return 2
	case 50:
		return 1
	case 42, 44, 46:
		return -1
	case 52:
		return -3
	case 55:
		return -2
	case 57:
		return -1
	case 92:
		return -1
	case 99:
		return 1
	case 102:
		return -1
	default:
		return 0
	}
}

func gp8PercussionArticulationIndex(track *Track, note *Note) (int, bool) {
	if note.HasPercussionArticulation {
		return note.PercussionArticulation, note.PercussionArticulation >= 0 && note.PercussionArticulation < len(track.PercussionArticulations)
	}
	for index, articulation := range track.PercussionArticulations {
		for _, input := range articulation.InputMIDINumbers {
			if input == int(note.Value) {
				return index, true
			}
		}
	}
	return 0, false
}

func gp8PercussionElements(track *Track, options GP8ExportOptions) []gpifElement {
	elements := make([]gpifElement, 0)
	for _, source := range track.PercussionArticulations {
		placement := source.TechniquePlacement
		if placement == "" {
			placement = "outside"
		}
		defaultHead := source.NoteheadDefault
		halfHead := source.NoteheadHalf
		if halfHead == "" {
			halfHead = defaultHead
		}
		wholeHead := source.NoteheadWhole
		if wholeHead == "" {
			wholeHead = defaultHead
		}
		noteheads := strings.TrimSpace(strings.Join([]string{defaultHead, halfHead, wholeHead}, " "))
		for _, input := range source.InputMIDINumbers {
			if override := options.PercussionNoteheads[int16(input)]; override != GP8PercussionNoteheadDefault {
				noteheads = gp8PercussionNoteheads(override)
				break
			}
		}
		inputs := make([]string, 0, len(source.InputMIDINumbers))
		for _, input := range source.InputMIDINumbers {
			inputs = append(inputs, strconv.Itoa(input))
		}
		articulation := gpifArticulation{
			Name:               source.Name,
			StaffLine:          source.StaffLine,
			Noteheads:          noteheads,
			TechniquePlacement: placement,
			TechniqueSymbol:    source.TechniqueSymbol,
			InputMIDINumbers:   strings.Join(inputs, " "),
			OutputRSESound:     source.OutputRSESound,
			OutputMIDINumber:   source.OutputMIDINumber,
		}
		if len(elements) == 0 || elements[len(elements)-1].Name != source.ElementName || elements[len(elements)-1].Type != source.ElementType || elements[len(elements)-1].SoundbankName != source.ElementSoundbankName {
			elements = append(elements, gpifElement{Name: source.ElementName, Type: source.ElementType, SoundbankName: source.ElementSoundbankName})
		}
		last := &elements[len(elements)-1]
		last.Articulations.Articulations = append(last.Articulations.Articulations, articulation)
	}

	values := make([]int16, 0)
	seen := make(map[int16]struct{})
	addFallback := func(value int16) {
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	for _, staff := range gp8ExportStaves(track) {
		for measureIndex := range staff.Measures {
			for voiceIndex := range staff.Measures[measureIndex].Voices {
				for beatIndex := range staff.Measures[measureIndex].Voices[voiceIndex].Beats {
					for noteIndex := range staff.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes {
						note := &staff.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes[noteIndex]
						if _, ok := gp8PercussionArticulationIndex(track, note); !ok {
							addFallback(note.Value)
						}
						for graceIndex := range note.Effect.Graces {
							grace := &note.Effect.Graces[graceIndex]
							graceNote := Note{Value: int16(grace.Fret), HasPercussionArticulation: grace.HasPercussionArticulation, PercussionArticulation: grace.PercussionArticulation}
							if grace.ExactFret != nil {
								graceNote.Value = int16(*grace.ExactFret)
							}
							if graceNote.Value < 27 || graceNote.Value > 87 {
								graceNote.Value = note.Value
							}
							if _, ok := gp8PercussionArticulationIndex(track, &graceNote); !ok {
								addFallback(graceNote.Value)
							}
						}
					}
				}
			}
		}
	}
	slices.Sort(values)
	return append(elements, gp8DrumElements(values, options)...)
}

func gp8DrumElements(values []int16, options GP8ExportOptions) []gpifElement {
	elements := make([]gpifElement, 0, len(values))
	type elementKey struct {
		name          string
		kind          string
		soundbankName string
	}
	elementIndices := make(map[elementKey]int, len(values))
	for _, value := range values {
		element := gp8DrumElement(value, options)
		key := elementKey{name: element.Name, kind: element.Type, soundbankName: element.SoundbankName}
		if index, exists := elementIndices[key]; exists {
			existing := &elements[index]
			existing.Articulations.Articulations = append(existing.Articulations.Articulations, element.Articulations.Articulations...)
			continue
		}
		elements = append(elements, element)
		elementIndices[key] = len(elements) - 1
	}
	return elements
}

func gp8DrumArticulationIDs(elements []gpifElement) map[int16]int {
	ids := make(map[int16]int)
	index := 0
	for _, element := range elements {
		for _, articulation := range element.Articulations.Articulations {
			for _, input := range strings.Fields(articulation.InputMIDINumbers) {
				midi, err := strconv.ParseInt(input, 10, 16)
				if err == nil {
					ids[int16(midi)] = index
				}
			}
			index++
		}
	}
	return ids
}

func gp8DrumElement(value int16, options GP8ExportOptions) gpifElement {
	midi := strconv.Itoa(int(value))
	element := gpifElement{
		Name: "MIDI " + midi,
		Type: "percussion",
	}
	articulation := gpifArticulation{
		Name:               element.Name,
		StaffLine:          gp8DrumStaffLine(value),
		Noteheads:          "noteheadBlack noteheadHalf noteheadWhole",
		TechniquePlacement: "outside",
		InputMIDINumbers:   midi,
		OutputMIDINumber:   int(value),
	}
	switch value {
	case 35:
		element.Name = "Acoustic Kick Drum"
		element.Type = "kickDrum"
		element.SoundbankName = "AcousticKick-Percu"
		articulation.Name = "Kick (hit)"
		articulation.OutputRSESound = "pedal.hit.hit"
	case 36:
		element.Name = "Kick Drum"
		element.Type = "kickDrum"
		element.SoundbankName = "Master-Kick"
		articulation.Name = "Kick (hit)"
		articulation.OutputRSESound = "pedal.hit.hit"
	case 37:
		element.Name = "Snare"
		element.Type = "snare"
		element.SoundbankName = "Master-Snare"
		articulation.Name = "Snare (side stick)"
		articulation.Noteheads = "noteheadXBlack noteheadXBlack noteheadXBlack"
		articulation.OutputRSESound = "stick.hit.sidestick"
	case 38:
		element.Name = "Snare"
		element.Type = "snare"
		element.SoundbankName = "Master-Snare"
		articulation.Name = "Snare (hit)"
		articulation.OutputRSESound = "stick.hit.hit"
	case 41:
		element.Name = "Very Low Floor Tom"
		element.Type = "tom"
		element.SoundbankName = "LowFloorTom-Percu"
		articulation.Name = "Low Floor Tom (hit)"
		articulation.OutputRSESound = "stick.hit.hit"
	case 42:
		element.Name = "Charley"
		element.Type = "hiHat"
		element.SoundbankName = "Master-Hihat"
		articulation.Name = "Hi-Hat (closed)"
		articulation.Noteheads = "noteheadXBlack noteheadXBlack noteheadXBlack"
		articulation.OutputRSESound = "stick.hit.closed"
	case 44:
		element.Name = "Charley"
		element.Type = "hiHat"
		element.SoundbankName = "Master-Hihat"
		articulation.Name = "Pedal Hi-Hat (hit)"
		articulation.StaffLine = 9
		articulation.Noteheads = "noteheadXBlack noteheadXBlack noteheadXBlack"
		articulation.OutputRSESound = "pedal.hit.pedal"
	case 45:
		element.Name = "Tom Low"
		element.Type = "tom"
		element.SoundbankName = "Master-Tom02"
		articulation.Name = "Low Tom (hit)"
		articulation.OutputRSESound = "stick.hit.hit"
	case 46:
		element.Name = "Charley"
		element.Type = "hiHat"
		element.SoundbankName = "Master-Hihat"
		articulation.Name = "Hi-Hat (open)"
		articulation.Noteheads = "noteheadCircleX noteheadCircleX noteheadCircleX"
		articulation.OutputRSESound = "stick.hit.open"
	case 43:
		element.Name = "Tom Very Low"
		element.Type = "tom"
		element.SoundbankName = "Master-Tom01"
		articulation.Name = "Very Low Tom (hit)"
		articulation.OutputRSESound = "stick.hit.hit"
	case 47:
		element.Name = "Tom Medium"
		element.Type = "tom"
		element.SoundbankName = "Master-Tom03"
		articulation.Name = "Mid Tom (hit)"
		articulation.OutputRSESound = "stick.hit.hit"
	case 48:
		element.Name = "Tom High"
		element.Type = "tom"
		element.SoundbankName = "Master-Tom04"
		articulation.Name = "High Tom (hit)"
		articulation.OutputRSESound = "stick.hit.hit"
	case 49:
		element.Name = "Crash High"
		element.Type = "crash"
		element.SoundbankName = "Master-Crash02"
		articulation.Name = "Crash high (hit)"
		articulation.StaffLine = -2
		articulation.Noteheads = "noteheadHeavyX noteheadHeavyX noteheadHeavyX"
		articulation.OutputRSESound = "stick.hit.hit"
	case 50:
		element.Name = "Tom Very High"
		element.Type = "tom"
		element.SoundbankName = "Master-Tom05"
		articulation.Name = "High Floor Tom (hit)"
		articulation.OutputRSESound = "stick.hit.hit"
	case 51:
		element.Name = "Ride"
		element.Type = "ride"
		element.SoundbankName = "Master-Ride"
		articulation.Name = "Ride (middle)"
		articulation.Noteheads = "noteheadXBlack noteheadXBlack noteheadXBlack"
		articulation.OutputRSESound = "stick.hit.mid"
	case 52:
		element.Name = "China"
		element.Type = "china"
		element.SoundbankName = "Master-China"
		articulation.Name = "China (hit)"
		articulation.Noteheads = "noteheadHeavyXHat noteheadHeavyXHat noteheadHeavyXHat"
		articulation.OutputRSESound = "stick.hit.hit"
	case 53:
		element.Name = "Ride"
		element.Type = "ride"
		element.SoundbankName = "Master-Ride"
		articulation.Name = "Ride (bell)"
		articulation.Noteheads = "noteheadDiamondWhite noteheadDiamondWhite noteheadDiamondWhite"
		articulation.OutputRSESound = "stick.hit.bell"
	case 55:
		element.Name = "Splash"
		element.Type = "splash"
		element.SoundbankName = "Master-Splash"
		articulation.Name = "Splash (hit)"
		articulation.Noteheads = "noteheadXBlack noteheadXBlack noteheadXBlack"
		articulation.OutputRSESound = "stick.hit.hit"
	case 57:
		element.Name = "Crash Medium"
		element.Type = "crash"
		element.SoundbankName = "Master-Crash01"
		articulation.Name = "Crash medium (hit)"
		articulation.StaffLine = -1
		articulation.Noteheads = "noteheadHeavyX noteheadHeavyX noteheadHeavyX"
		articulation.OutputRSESound = "stick.hit.hit"
	case 92:
		element.Name = "Charley"
		element.Type = "hiHat"
		element.SoundbankName = "Master-Hihat"
		articulation.Name = "Hi-Hat (half)"
		articulation.Noteheads = "noteheadCircleSlash noteheadCircleSlash noteheadCircleSlash"
		articulation.OutputMIDINumber = 46
		articulation.OutputRSESound = "stick.hit.half"
	case 99:
		element.Name = "Cowbell Low"
		element.Type = "cowbell"
		element.SoundbankName = "CowbellBig-Percu"
		articulation.Name = "Cowbell low (hit)"
		articulation.StaffLine = 1
		articulation.Noteheads = "noteheadTriangleUpBlack noteheadTriangleUpHalf noteheadTriangleUpWhole"
		articulation.OutputMIDINumber = 56
		articulation.OutputRSESound = "stick.hit.hit"
	case 102:
		element.Name = "Cowbell High"
		element.Type = "cowbell"
		element.SoundbankName = "CowbellSmall-Percu"
		articulation.Name = "Cowbell high (hit)"
		articulation.StaffLine = -1
		articulation.Noteheads = "noteheadTriangleUpBlack noteheadTriangleUpHalf noteheadTriangleUpWhole"
		articulation.OutputMIDINumber = 56
		articulation.OutputRSESound = "stick.hit.hit"
	}
	if notehead := options.PercussionNoteheads[value]; notehead != GP8PercussionNoteheadDefault {
		articulation.Noteheads = gp8PercussionNoteheads(notehead)
	}
	element.Articulations.Articulations = []gpifArticulation{articulation}
	return element
}

func gp8PercussionNoteheads(notehead GP8PercussionNotehead) string {
	switch notehead {
	case GP8PercussionNoteheadFilled:
		return "noteheadBlack noteheadHalf noteheadWhole"
	case GP8PercussionNoteheadX:
		return "noteheadXBlack noteheadXBlack noteheadXBlack"
	case GP8PercussionNoteheadCircleX:
		return "noteheadCircleX noteheadCircleX noteheadCircleX"
	case GP8PercussionNoteheadHeavyX:
		return "noteheadHeavyX noteheadHeavyX noteheadHeavyX"
	default:
		return ""
	}
}

func sequentialIDs(count int) string {
	ids := make([]string, count)
	for index := range count {
		ids[index] = strconv.Itoa(index)
	}
	return strings.Join(ids, " ")
}
