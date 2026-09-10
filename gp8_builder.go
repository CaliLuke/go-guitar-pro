// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

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
	buildGP8BackingTrack(song.BackingTrack, &builder.doc)

	for trackIndex := range song.Tracks {
		builder.prepareTrack(trackIndex)
		builder.doc.Tracks.Tracks = append(builder.doc.Tracks.Tracks, builder.buildTrack(trackIndex))
	}
	if err := builder.buildScoreGraph(); err != nil {
		return gpifDocument{}, err
	}
	return builder.doc, nil
}

func gp8EmbedsBackingTrack(backingTrack *BackingTrack) bool {
	return backingTrack != nil && backingTrack.Enabled && backingTrack.Source == "Local"
}

func buildGP8BackingTrack(backingTrack *BackingTrack, document *gpifDocument) {
	if !gp8EmbedsBackingTrack(backingTrack) {
		return
	}
	document.BackingTrack = &gpifBackingTrack{
		Name:         backingTrack.Name,
		Enabled:      backingTrack.Enabled,
		Source:       backingTrack.Source,
		AssetID:      backingTrack.AssetID,
		FramePadding: strconv.FormatInt(backingTrack.FramePadding, 10),
	}
	document.Assets = gpifAssets{Assets: []gpifAsset{{
		ID:               backingTrack.AssetID,
		OriginalFilePath: backingTrack.OriginalFilePath,
		OriginalFileSHA1: backingTrack.OriginalFileSHA1,
		EmbeddedFilePath: backingTrack.EmbeddedFilePath,
	}}}
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
	openingTextUsed := false
	for _, tempo := range tempos {
		if tempo.Tempo <= 0 {
			continue
		}
		text := tempo.Text
		if !openingTextUsed && tempo.Bar == 0 && tempo.Position == 0 && song.TempoName != "" {
			text = song.TempoName
			openingTextUsed = true
		}
		automations.Automations = append(automations.Automations, gpifAutomation{
			Type:     "Tempo",
			Linear:   tempo.Linear,
			Value:    gpifAutomationValue{Text: strconv.FormatFloat(tempo.Tempo, 'f', -1, 64) + " 2"},
			Visible:  strconv.FormatBool(!tempo.Hidden),
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
			Type: "Sound", Value: gpifAutomationValue{Text: sound.Path + ";" + sound.Name + ";" + sound.Role, cdata: true},
			Linear: automation.Linear, Text: automation.Text, Visible: strconv.FormatBool(!automation.Hidden),
			Bar: automation.Bar, Position: automation.Position,
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
		capo := int(staff.CapoFret)
		result.Staves.Staff[staffIndex].Properties = append(
			result.Staves.Staff[staffIndex].Properties,
			gpifStaffProperty{Name: "CapoFret", Fret: &capo},
		)
		if len(staff.Strings) != 0 {
			pitches := make([]string, 0, len(staff.Strings))
			for index := len(staff.Strings) - 1; index >= 0; index-- {
				pitches = append(pitches, strconv.Itoa(int(staff.Strings[index].Value)))
			}
			result.Staves.Staff[staffIndex].Properties = append(
				result.Staves.Staff[staffIndex].Properties,
				gpifStaffProperty{Name: "Tuning", Pitches: strings.Join(pitches, " ")},
			)
		}
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
				builder.doc.Bars.Bars = append(builder.doc.Bars.Bars, gpifBar{
					ID: barID, Voices: strings.Join(voiceIDs, " "), Clef: gp8BarClef(builder.song, trackIndex, measure), Ottavia: gp8Octave(measure.ClefOctave), SimileMark: gp8SimileMark(measure.SimileMark),
				})
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
	if beat.Effect.Stroke.Direction != BeatStrokeDirectionNone && beat.Effect.Stroke.Duration != NoteValue(DurationEighth) {
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
