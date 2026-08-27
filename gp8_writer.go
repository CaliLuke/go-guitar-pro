// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"cmp"
	"encoding/binary"
	"encoding/xml"
	"fmt"
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

// Export serializes song in target format and returns the complete file bytes.
// Export currently supports [ExportFormatGP8]. It does not mutate song.
func Export(song *Song, target ExportFormat) ([]byte, error) {
	if song == nil {
		return nil, fmt.Errorf("exporting Guitar Pro file: song is nil")
	}
	if target != ExportFormatGP8 {
		return nil, fmt.Errorf("exporting Guitar Pro file: unsupported target %d", target)
	}
	if err := validateGP8Song(song); err != nil {
		return nil, fmt.Errorf("exporting Guitar Pro 8 file: %w", err)
	}

	doc, err := buildGP8Document(song)
	if err != nil {
		return nil, fmt.Errorf("exporting Guitar Pro 8 file: %w", err)
	}
	gpif, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling GPIF XML: %w", err)
	}
	gpif = append(append([]byte(xml.Header), gpif...), '\n')

	var output bytes.Buffer
	archive := zip.NewWriter(&output)
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
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: entry.method}
		header.SetMode(0o644)
		writer, createErr := archive.CreateHeader(header)
		if createErr != nil {
			_ = archive.Close()
			return nil, fmt.Errorf("creating archive member %q: %w", entry.name, createErr)
		}
		if _, writeErr := writer.Write(entry.data); writeErr != nil {
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
	data, err := Export(song, target)
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
	hasPositiveTempo := song.Tempo > 0
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
		if len(track.Measures) != len(song.MeasureHeaders) {
			return fmt.Errorf("track %d has %d measures, want %d", trackIndex, len(track.Measures), len(song.MeasureHeaders))
		}
		for measureIndex := range track.Measures {
			if len(track.Measures[measureIndex].Voices) > 4 {
				return fmt.Errorf("track %d measure %d has %d voices, Guitar Pro 8 supports at most 4", trackIndex, measureIndex, len(track.Measures[measureIndex].Voices))
			}
			for voiceIndex := range track.Measures[measureIndex].Voices {
				for beatIndex := range track.Measures[measureIndex].Voices[voiceIndex].Beats {
					for noteIndex, note := range track.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes {
						midi := gp8NoteMIDI(track, &note)
						if midi < 0 || midi > 127 {
							return fmt.Errorf("track %d measure %d voice %d beat %d note %d has MIDI value %d outside 0..127", trackIndex, measureIndex, voiceIndex, beatIndex, noteIndex, midi)
						}
					}
				}
			}
		}
	}
	return nil
}

type gp8Builder struct {
	song            *Song
	doc             gpifDocument
	rhythmIDs       map[Duration]string
	chordIDs        []map[*Chord]string
	articulationIDs []map[int16]int
}

func buildGP8Document(song *Song) (gpifDocument, error) {
	builder := gp8Builder{
		song:            song,
		rhythmIDs:       make(map[Duration]string),
		chordIDs:        make([]map[*Chord]string, len(song.Tracks)),
		articulationIDs: make([]map[int16]int, len(song.Tracks)),
	}
	builder.doc = gpifDocument{
		GPVersion: gp8DocumentVersion,
		GPRevision: gpifRevision{
			Required:    gp8RevisionRequired,
			Recommended: gp8RevisionRecommended,
			Value:       gp8Revision,
		},
		Encoding: gpifEncoding{Description: "GP8"},
		Score: gpifScore{
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
		},
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

func buildGP8TempoAutomations(song *Song) gpifAutomations {
	tempos := slices.Clone(song.TempoAutomations)
	hasInitial := false
	for _, tempo := range tempos {
		if tempo.Bar == 0 && tempo.Position == 0 {
			hasInitial = true
			break
		}
	}
	if !hasInitial && song.Tempo > 0 {
		tempos = append(tempos, TempoAutomation{Tempo: float64(song.Tempo)})
	}
	slices.SortFunc(tempos, func(a, b TempoAutomation) int {
		if order := cmp.Compare(a.Bar, b.Bar); order != 0 {
			return order
		}
		return cmp.Compare(a.Position, b.Position)
	})

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

func (builder *gp8Builder) prepareTrack(trackIndex int) {
	track := &builder.song.Tracks[trackIndex]
	chords := make(map[*Chord]string)
	for measureIndex := range track.Measures {
		for voiceIndex := range track.Measures[measureIndex].Voices {
			for beatIndex := range track.Measures[measureIndex].Voices[voiceIndex].Beats {
				chord := track.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Effect.Chord
				if chord != nil {
					if _, exists := chords[chord]; !exists {
						chords[chord] = strconv.Itoa(len(chords))
					}
				}
			}
		}
	}
	builder.chordIDs[trackIndex] = chords

	articulationIDs := make(map[int16]int)
	if track.PercussionTrack {
		values := make([]int16, 0)
		seen := make(map[int16]struct{})
		for measureIndex := range track.Measures {
			for voiceIndex := range track.Measures[measureIndex].Voices {
				for beatIndex := range track.Measures[measureIndex].Voices[voiceIndex].Beats {
					for _, note := range track.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes {
						if _, exists := seen[note.Value]; !exists {
							seen[note.Value] = struct{}{}
							values = append(values, note.Value)
						}
					}
				}
			}
		}
		slices.Sort(values)
		articulationIDs = gp8DrumArticulationIDs(gp8DrumElements(values))
	}
	builder.articulationIDs[trackIndex] = articulationIDs
}

func (builder *gp8Builder) buildTrack(trackIndex int) gpifTrack {
	track := &builder.song.Tracks[trackIndex]
	channel := defaultMidiChannel()
	if track.ChannelIndex >= 0 && track.ChannelIndex < len(builder.song.Channels) {
		channel = builder.song.Channels[track.ChannelIndex]
	} else if track.PercussionTrack {
		channel.Channel = DefaultPercussionChannel
		channel.EffectChannel = DefaultPercussionChannel
	}

	red := (uint32(track.Color) >> 16) & 0xff
	green := (uint32(track.Color) >> 8) & 0xff
	blue := uint32(track.Color) & 0xff
	result := gpifTrack{
		ID:    strconv.Itoa(trackIndex),
		Name:  track.Name,
		Color: fmt.Sprintf("%d %d %d", red, green, blue),
		Sounds: gpifSounds{Sounds: []gpifSound{{
			Name:    track.Name,
			Program: int(channel.Instrument),
			Channel: int(channel.Channel % 16),
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
	switch {
	case track.Mute:
		result.PlaybackState = "Mute"
	case track.Solo:
		result.PlaybackState = "Solo"
	}

	properties := make([]gpifStaffProperty, 0, 2)
	if len(track.Strings) > 0 {
		pitches := make([]string, 0, len(track.Strings))
		for index := len(track.Strings) - 1; index >= 0; index-- {
			pitches = append(pitches, strconv.Itoa(int(track.Strings[index].Value)))
		}
		properties = append(properties, gpifStaffProperty{Name: "Tuning", Pitches: strings.Join(pitches, " ")})
	}
	if len(builder.chordIDs[trackIndex]) > 0 {
		items := make([]gpifItem, 0, len(builder.chordIDs[trackIndex]))
		for chord, id := range builder.chordIDs[trackIndex] {
			items = append(items, gp8ChordItem(id, chord, len(track.Strings)))
		}
		slices.SortFunc(items, func(a, b gpifItem) int { return cmp.Compare(a.ID, b.ID) })
		properties = append(properties, gpifStaffProperty{Name: "DiagramCollection", Items: &gpifItems{Items: items}})
	}
	result.Staves = gpifStaves{Staff: []gpifStaff{{Properties: properties}}}

	if track.PercussionTrack {
		result.InstrumentSet = &gpifInstrumentSet{Name: "Drums", Type: "drumKit", LineCount: 5}
		values := make([]int16, 0, len(builder.articulationIDs[trackIndex]))
		for value := range builder.articulationIDs[trackIndex] {
			values = append(values, value)
		}
		slices.Sort(values)
		result.InstrumentSet.Elements.Elements = gp8DrumElements(values)
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
	diagram := &gpifDiagram{StringCount: stringCount, FretCount: 5}
	if chord.FirstFret != nil {
		diagram.BaseFret = int(*chord.FirstFret)
	}
	for index, fret := range chord.Strings {
		if fret >= 0 {
			diagram.Frets = append(diagram.Frets, gpifDiagramFret{String: stringCount - index - 1, Fret: int(fret)})
		}
	}
	return gpifItem{ID: id, Name: chord.Name, Diagram: diagram, Chord: &struct{}{}}
}

func (builder *gp8Builder) buildScoreGraph() error {
	for measureIndex := range builder.song.MeasureHeaders {
		header := &builder.song.MeasureHeaders[measureIndex]
		barIDs := make([]string, 0, len(builder.song.Tracks))
		for trackIndex := range builder.song.Tracks {
			barID := strconv.Itoa(len(builder.doc.Bars.Bars))
			barIDs = append(barIDs, barID)
			measure := &builder.song.Tracks[trackIndex].Measures[measureIndex]
			voiceIDs := make([]string, 0, 4)
			for voiceIndex := range measure.Voices {
				voice := &measure.Voices[voiceIndex]
				if len(voice.Beats) == 0 {
					continue
				}
				voiceID := strconv.Itoa(len(builder.doc.Voices.Voices))
				voiceIDs = append(voiceIDs, voiceID)
				beatIDs := make([]string, 0, len(voice.Beats))
				for beatIndex := range voice.Beats {
					graceIDs, err := builder.addGraceBeats(trackIndex, &voice.Beats[beatIndex])
					if err != nil {
						return fmt.Errorf("track %d measure %d voice %d beat %d grace notes: %w", trackIndex, measureIndex, voiceIndex, beatIndex, err)
					}
					beatIDs = append(beatIDs, graceIDs...)
					beatID, err := builder.addBeat(trackIndex, &voice.Beats[beatIndex])
					if err != nil {
						return fmt.Errorf("track %d measure %d voice %d beat %d: %w", trackIndex, measureIndex, voiceIndex, beatIndex, err)
					}
					beatIDs = append(beatIDs, beatID)
				}
				builder.doc.Voices.Voices = append(builder.doc.Voices.Voices, gpifVoice{ID: voiceID, Beats: strings.Join(beatIDs, " ")})
			}
			for len(voiceIDs) < 4 {
				voiceIDs = append(voiceIDs, "-1")
			}
			builder.doc.Bars.Bars = append(builder.doc.Bars.Bars, gpifBar{ID: barID, Voices: strings.Join(voiceIDs, " "), Clef: "G2"})
		}
		builder.doc.MasterBars.MasterBars = append(builder.doc.MasterBars.MasterBars, gp8MasterBar(header, strings.Join(barIDs, " ")))
	}
	return nil
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
	}
	return result
}

type gp8GraceGroup struct {
	duration Duration
	velocity int16
	onBeat   bool
	notes    []Note
}

func (builder *gp8Builder) addGraceBeats(trackIndex int, beat *Beat) ([]string, error) {
	var groups []gp8GraceGroup
	for noteIndex := range beat.Notes {
		note := &beat.Notes[noteIndex]
		grace := note.Effect.Grace
		if grace == nil {
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
			noteIDs = append(noteIDs, builder.addNote(trackIndex, &group.notes[noteIndex]))
		}
		result.Notes = strings.Join(noteIDs, " ")
		builder.doc.Beats.Beats = append(builder.doc.Beats.Beats, result)
		ids = append(ids, id)
	}
	return ids, nil
}

func (builder *gp8Builder) addBeat(trackIndex int, beat *Beat) (string, error) {
	rhythmID, err := builder.addRhythm(beat.Duration)
	if err != nil {
		return "", err
	}
	beatID := strconv.Itoa(len(builder.doc.Beats.Beats))
	result := gpifBeat{ID: beatID, Rhythm: gpifRhythmRef{Ref: rhythmID}, FreeText: beat.Text}
	if len(beat.Notes) > 0 {
		result.Dynamic = gp8VelocityToDynamic(beat.Notes[0].Velocity)
	}
	if beat.Effect.Chord != nil {
		result.Chord = builder.chordIDs[trackIndex][beat.Effect.Chord]
	}
	if beat.Effect.FadeIn {
		result.Fadding = "FadeIn"
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
		noteID := builder.addNote(trackIndex, &beat.Notes[noteIndex])
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

func (builder *gp8Builder) addNote(trackIndex int, note *Note) string {
	track := &builder.song.Tracks[trackIndex]
	noteID := strconv.Itoa(len(builder.doc.Notes.Notes))
	fret := int(note.Value)
	midi := gp8NoteMIDI(track, note)
	properties := []gpifProperty{
		{Name: "Fret", Fret: &fret},
		{Name: "Midi", Number: &midi},
	}
	if note.String > 0 {
		stringValue := float64(int(note.String) - 1)
		if !track.PercussionTrack && int(note.String) <= len(track.Strings) {
			stringValue = float64(len(track.Strings) - int(note.String))
		}
		properties = append(properties, gpifProperty{Name: "String", String: &stringValue})
	}
	if !track.PercussionTrack {
		pitch := gp8Pitch(midi)
		properties = append([]gpifProperty{{Name: "ConcertPitch", Pitch: &pitch}, {Name: "TransposedPitch", Pitch: &pitch}}, properties...)
	}
	articulation := 0
	result := gpifNote{ID: noteID, InstrumentArticulation: &articulation, Properties: gpifProperties{Properties: properties}}
	if track.PercussionTrack {
		articulation = builder.articulationIDs[trackIndex][note.Value]
		result.InstrumentArticulation = &articulation
	}
	if note.TieOrigin || note.Kind == NoteTypeTie {
		result.Tie = &gpifTie{
			Origin:      strconv.FormatBool(note.TieOrigin),
			Destination: strconv.FormatBool(note.Kind == NoteTypeTie),
		}
	}
	if note.Kind == NoteTypeDead || note.Effect.GhostNote {
		if note.Kind == NoteTypeDead {
			result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "Muted"})
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
		result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "HopoOrigin"})
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
			}
		}
		value := strconv.Itoa(flags)
		result.Properties.Properties = append(result.Properties.Properties, gpifProperty{Name: "Slide", Flags: &value})
	}
	if note.Effect.Vibrato {
		result.Vibrato = "Slight"
	}
	if note.Effect.Trill != nil {
		result.Trill = &gpifTrill{Fret: int(note.Effect.Trill.Fret)}
	}
	if note.Effect.Staccato {
		result.Accent |= 0x01
	}
	if note.Effect.HeavyAccentuatedNote {
		result.Accent |= 0x04
	}
	if note.Effect.AccentuatedNote {
		result.Accent |= 0x08
	}
	builder.doc.Notes.Notes = append(builder.doc.Notes.Notes, result)
	return noteID
}

func gp8NoteMIDI(track *Track, note *Note) int {
	midi := int(note.Value)
	if !track.PercussionTrack && note.String > 0 && int(note.String) <= len(track.Strings) {
		midi += int(track.Strings[int(note.String)-1].Value)
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

func gp8DrumStaffLine(value int16) int {
	switch value {
	case 35:
		return 8
	case 36:
		return 7
	case 38, 40:
		return 3
	case 41, 43:
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
	default:
		return 0
	}
}

func gp8DrumElements(values []int16) []gpifElement {
	elements := make([]gpifElement, 0, len(values))
	hiHatIndex := -1
	for _, value := range values {
		element := gp8DrumElement(value)
		if element.Type == "hiHat" && hiHatIndex >= 0 {
			hiHat := &elements[hiHatIndex]
			hiHat.Articulations.Articulations = append(hiHat.Articulations.Articulations, element.Articulations.Articulations...)
			continue
		}
		elements = append(elements, element)
		if element.Type == "hiHat" {
			hiHatIndex = len(elements) - 1
		}
	}
	return elements
}

func gp8DrumArticulationIDs(elements []gpifElement) map[int16]int {
	ids := make(map[int16]int)
	index := 0
	for _, element := range elements {
		for _, articulation := range element.Articulations.Articulations {
			ids[int16(articulation.OutputMIDINumber)] = index
			index++
		}
	}
	return ids
}

func gp8DrumElement(value int16) gpifElement {
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
	case 36:
		element.Name = "Kick Drum"
		element.Type = "kickDrum"
		element.SoundbankName = "Master-Kick"
		articulation.Name = "Kick (hit)"
		articulation.OutputRSESound = "pedal.hit.hit"
	case 38:
		element.Name = "Snare"
		element.Type = "snare"
		element.SoundbankName = "Master-Snare"
		articulation.Name = "Snare (hit)"
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
	case 46:
		element.Name = "Charley"
		element.Type = "hiHat"
		element.SoundbankName = "Master-Hihat"
		articulation.Name = "Hi-Hat (open)"
		articulation.Noteheads = "noteheadBlack noteheadHalf noteheadWhole"
		articulation.OutputRSESound = "stick.hit.open"
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
	}
	element.Articulations.Articulations = []gpifArticulation{articulation}
	return element
}

func sequentialIDs(count int) string {
	ids := make([]string, count)
	for index := range count {
		ids[index] = strconv.Itoa(index)
	}
	return strings.Join(ids, " ")
}
