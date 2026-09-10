// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
	"strings"
)

func parseGPIF(data []byte) (*Song, error) {
	return parseGPIFWithContext(data, nil)
}

func parseGPIFWithContext(data []byte, context *parseContext) (*Song, error) {
	if err := gpifAuditXML(data, context); err != nil {
		return nil, fmt.Errorf("parsing GPIF XML: %w", err)
	}
	var doc gpifDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing GPIF XML: %w", err)
	}
	if context != nil && context.format != "GP6" {
		switch {
		case strings.HasPrefix(doc.GPVersion, "7"):
			context.setFormat("GP7")
		case strings.HasPrefix(doc.GPVersion, "8"):
			context.setFormat("GP8")
		}
	}
	gpifAuditDiagnostics(doc, context)

	song := &Song{
		Tempo:     120,
		TempoName: "Moderate",
	}
	version := doc.GPVersion
	if version == "" && context != nil && context.format == "GP6" {
		version = "6"
	}
	song.Version = gpifVersion(version)
	song.Anacrusis = doc.MasterTrack.Anacrusis != nil

	// Score info
	song.Name = doc.Score.Title
	song.Subtitle = doc.Score.SubTitle
	song.Artist = doc.Score.Artist
	song.Album = doc.Score.Album
	song.Words = doc.Score.Words
	song.Author = doc.Score.Music
	if doc.Score.WordsAndMusic != "" {
		if song.Words == "" {
			song.Words = doc.Score.WordsAndMusic
		}
		if song.Author == "" {
			song.Author = doc.Score.WordsAndMusic
		}
	}
	song.Copyright = doc.Score.Copyright
	song.Transcriber = doc.Score.Tabber
	song.Instructions = doc.Score.Instructions
	if doc.Score.Notices != "" {
		song.Notice = strings.Split(doc.Score.Notices, "\n")
	}

	gpifReadBackingTrack(doc, song)
	gpifAuditMasterAutomations(doc.MasterTrack.Automations.Automations, context)
	gpifReadSyncPoints(doc.MasterTrack.Automations.Automations, song)
	gpifReadTempoAutomations(doc.MasterTrack.Automations.Automations, song, context)

	// Build rhythm lookup
	rhythmMap := make(map[string]*gpifRhythm)
	for i := range doc.Rhythms.Rhythms {
		r := &doc.Rhythms.Rhythms[i]
		rhythmMap[r.ID] = r
	}

	// Build note lookup
	noteMap := make(map[string]*gpifNote)
	for i := range doc.Notes.Notes {
		n := &doc.Notes.Notes[i]
		noteMap[n.ID] = n
	}

	// Build beat lookup
	beatMap := make(map[string]*gpifBeat)
	for i := range doc.Beats.Beats {
		b := &doc.Beats.Beats[i]
		beatMap[b.ID] = b
	}

	// Build voice lookup
	voiceMap := make(map[string]*gpifVoice)
	for i := range doc.Voices.Voices {
		v := &doc.Voices.Voices[i]
		voiceMap[v.ID] = v
	}

	// Build bar lookup
	barMap := make(map[string]*gpifBar)
	for i := range doc.Bars.Bars {
		b := &doc.Bars.Bars[i]
		barMap[b.ID] = b
	}

	// Parse tracks
	trackIDs := splitIDs(doc.MasterTrack.Tracks)
	trackChordMaps := make([]gpifChordScope, 0, len(trackIDs))
	for _, trackID := range trackIDs {
		track := defaultTrack()
		track.Number = int32(len(song.Tracks))
		chordMap := gpifChordScope{}
		for _, t := range doc.Tracks.Tracks {
			if t.ID == trackID {
				gpifAuditTrackAutomations(t, context)
				track.Name = t.Name
				capo, capoErr := gpifResolveCapo(t)
				if capoErr != nil {
					return nil, fmt.Errorf("track %s capo: %w", trackID, capoErr)
				}
				track.Offset = capo.value
				track.PercussionTrack = t.isPercussionTrack()
				if track.PercussionTrack {
					track.PercussionArticulations = gpifReadPercussionArticulations(t.InstrumentSet, t.NotationPatch)
				}
				if t.Lyrics != nil {
					for _, line := range t.Lyrics.Lines {
						track.Lyrics = append(track.Lyrics, TrackLyricLine(line))
					}
				}
				for _, sound := range t.Sounds.Sounds {
					track.Sounds = append(track.Sounds, TrackSound{
						Name: sound.Name, Label: sound.Label, Path: sound.Path,
						Role: sound.Role, Program: int32(sound.Program),
					})
				}
				for _, automation := range t.Automations.Automations {
					if automation.Type != "Sound" {
						continue
					}
					for soundIndex, sound := range track.Sounds {
						if automation.Value.Text == sound.Path+";"+sound.Name+";"+sound.Role {
							track.SoundAutomations = append(track.SoundAutomations, SoundAutomation{
								Bar: automation.Bar, Position: automation.Position, Sound: soundIndex,
							})
							break
						}
					}
				}
				staffCount := max(1, len(t.Staves.Staff))
				if t.Instrument != nil && (strings.HasSuffix(t.Instrument.Ref, "-gs") || strings.HasSuffix(t.Instrument.Ref, "GrandStaff")) {
					staffCount = max(2, staffCount)
				}
				lineCount := 5
				if t.InstrumentSet != nil && t.InstrumentSet.LineCount > 0 {
					lineCount = t.InstrumentSet.LineCount
				}
				if t.NotationPatch != nil && t.NotationPatch.LineCount > 0 {
					lineCount = t.NotationPatch.LineCount
				}
				trackStrings := append([]GuitarString(nil), track.Strings...)
				if parsed := gpifReadStaffStrings(gpifStaff{Properties: t.Properties}); parsed != nil {
					trackStrings = parsed
				}
				track.Staves = make([]Staff, staffCount)
				for staffIndex := range track.Staves {
					strings := append([]GuitarString(nil), trackStrings...)
					if staffIndex < len(t.Staves.Staff) {
						if parsed := gpifReadStaffStrings(t.Staves.Staff[staffIndex]); parsed != nil {
							strings = parsed
						}
					}
					track.Staves[staffIndex] = Staff{
						Strings:                   strings,
						PercussionTrack:           track.PercussionTrack,
						StandardNotationLineCount: lineCount,
					}
				}
				track.Strings = track.Staves[0].Strings
				var chordErr error
				chordMap, chordErr = gpifReadChordMap(t)
				if chordErr != nil {
					return nil, fmt.Errorf("track %q chord collection: %w", t.ID, chordErr)
				}
				// Parse color
				if t.Color != "" {
					parts := splitIDs(t.Color)
					if len(parts) >= 3 {
						r, _ := strconv.Atoi(parts[0])
						g, _ := strconv.Atoi(parts[1])
						b, _ := strconv.Atoi(parts[2])
						track.Color = int32(r)*65536 + int32(g)*256 + int32(b)
					}
				}
				ch := defaultMidiChannel()
				if len(t.Sounds.Sounds) > 0 {
					ch.Instrument = int32(t.Sounds.Sounds[0].Program)
				}
				ch.Channel = gpifMIDIChannel(t.MidiConnection.Port, t.MidiConnection.PrimaryChannel)
				ch.EffectChannel = gpifMIDIChannel(t.MidiConnection.Port, t.MidiConnection.SecondaryChannel)
				if t.GeneralMidi != nil {
					ch.Channel = gpifMIDIChannel(t.GeneralMidi.Port, t.GeneralMidi.PrimaryChannel)
					ch.EffectChannel = gpifMIDIChannel(t.GeneralMidi.Port, t.GeneralMidi.SecondaryChannel)
					if t.GeneralMidi.Program != nil {
						ch.Instrument = int32(*t.GeneralMidi.Program)
					}
				}
				if t.RSE != nil {
					gpifApplyChannelStrip(t.RSE.ChannelStrip.Parameters, &ch)
					gpifAuditChannelStripAutomations(t.RSE.ChannelStrip.Automations.Automations, t.ID, context)
					gpifReadVolumeAutomations(
						t.RSE.ChannelStrip.Automations.Automations,
						len(song.Tracks),
						song,
					)
				}
				song.Channels = append(song.Channels, ch)
				track.ChannelIndex = len(song.Channels) - 1
				track.UseRse = t.AudioEngineState == "RSE"
				track.Mute = t.PlaybackState == "Mute"
				track.Solo = t.PlaybackState == "Solo"
				break
			}
		}
		if len(track.Staves) == 0 {
			track.populateSingleStaff()
		}
		song.Tracks = append(song.Tracks, track)
		trackChordMaps = append(trackChordMaps, chordMap)
	}
	song.consolidateTrackChannels()

	// Parse master bars → measure headers + measures
	fallbackArticulations := make([]gpifPercussionFallbacks, len(song.Tracks))
	for trackIndex := range song.Tracks {
		fallbackArticulations[trackIndex].tableLength = len(song.Tracks[trackIndex].PercussionArticulations)
	}
	for mbIdx, mb := range doc.MasterBars.MasterBars {
		if mbIdx >= math.MaxUint16 {
			return nil, fmt.Errorf("master-bar index %d exceeds legacy uint16 boundary", mbIdx)
		}
		mh := defaultMeasureHeader()
		mh.Number = uint16(mbIdx + 1)

		// Time signature
		if mb.Time != "" {
			parts := strings.Split(mb.Time, "/")
			if len(parts) != 2 {
				return nil, fmt.Errorf("master bar %d has invalid time signature %q", mbIdx, mb.Time)
			}
			numerator, numeratorErr := strconv.ParseInt(parts[0], 10, 64)
			if numeratorErr != nil || numerator <= 0 || numerator > math.MaxInt8 {
				return nil, fmt.Errorf("master bar %d time-signature numerator %q is outside 1..%d", mbIdx, parts[0], math.MaxInt8)
			}
			denominator, denominatorErr := strconv.ParseInt(parts[1], 10, 64)
			if denominatorErr != nil || denominator <= 0 || denominator > math.MaxUint16 {
				return nil, fmt.Errorf("master bar %d time-signature denominator %q is outside 1..%d", mbIdx, parts[1], math.MaxUint16)
			}
			mh.TimeSignature.Numerator = int8(numerator)
			mh.TimeSignature.Denominator.Value = uint16(denominator)
		}

		// Key signature
		if mb.Key.AccidentalCount < math.MinInt8 || mb.Key.AccidentalCount > math.MaxInt8 {
			return nil, fmt.Errorf("master bar %d key accidental count %d is outside %d..%d", mbIdx, mb.Key.AccidentalCount, math.MinInt8, math.MaxInt8)
		}
		mh.KeySignature.Key = int8(mb.Key.AccidentalCount)
		mh.KeySignature.IsMinor = mb.Key.Mode == "Minor"

		if mb.Repeat != nil {
			mh.RepeatOpen = mb.Repeat.Start == "true"
			if mb.Repeat.End == "true" && mb.Repeat.Count > 0 {
				if mb.Repeat.Count > math.MaxInt8+1 {
					return nil, fmt.Errorf("master bar %d repeat count %d is outside 1..%d", mbIdx, mb.Repeat.Count, math.MaxInt8+1)
				}
				mh.RepeatClose = int8(mb.Repeat.Count - 1)
			}
		}
		for _, ending := range splitIDs(mb.AlternateEndings) {
			number, err := strconv.Atoi(ending)
			if err != nil || number < 1 || number > 8 {
				return nil, fmt.Errorf("master bar %d alternate ending %q is outside 1..8", mbIdx, ending)
			}
			mh.RepeatAlternative |= 1 << (number - 1)
		}

		// Section marker
		if mb.Section != nil {
			title := mb.Section.Text
			if title == "" {
				title = mb.Section.Letter
			}
			mh.Marker = &Marker{Title: title}
		}

		// Double bar
		mh.DoubleBar = mb.DoubleBar != nil

		// Triplet feel
		switch mb.TripletFeel {
		case "Triplet8th":
			mh.TripletFeel = TripletFeelEighth
		case "Triplet16th":
			mh.TripletFeel = TripletFeelSixteenth
		case "Dotted8th":
			mh.TripletFeel = TripletFeelDottedEighth
		case "Dotted16th":
			mh.TripletFeel = TripletFeelDottedSixteenth
		case "Scottish8th":
			mh.TripletFeel = TripletFeelScottishEighth
		case "Scottish16th":
			mh.TripletFeel = TripletFeelScottishSixteenth
		}

		song.MeasureHeaders = append(song.MeasureHeaders, mh)

		// GPIF lists bars vertically by staff, advancing the track only after its final staff.
		barIDs := splitIDs(mb.Bars)
		barIndex := 0
		newMeasure := func(trackIndex, staffIndex int) Measure {
			measure := defaultMeasure()
			measure.Number = mbIdx + 1
			measure.TrackIndex = trackIndex
			measure.StaffIndex = staffIndex
			measure.HeaderIndex = mbIdx
			measure.TimeSignature = mh.TimeSignature
			measure.KeySignature = mh.KeySignature
			measure.HasDoubleBar = mh.DoubleBar
			return measure
		}
		for trackIdx := 0; trackIdx < len(song.Tracks) && barIndex < len(barIDs); {
			track := &song.Tracks[trackIdx]
			for staffIdx := 0; staffIdx < len(track.Staves) && barIndex < len(barIDs); staffIdx++ {
				barID := barIDs[barIndex]
				barIndex++
				if barID == "-1" {
					for skippedStaffIdx := staffIdx; skippedStaffIdx < len(track.Staves); skippedStaffIdx++ {
						measure := newMeasure(trackIdx, skippedStaffIdx)
						track.Staves[skippedStaffIdx].Measures = append(track.Staves[skippedStaffIdx].Measures, measure)
					}
					break
				}
				staff := &track.Staves[staffIdx]
				m := newMeasure(trackIdx, staffIdx)

				if bar, ok := barMap[barID]; ok {
					m.Clef = gpifMeasureClef(bar.Clef)
					voiceIDs := splitIDs(bar.Voices)
					pendingVoicePlaceholders := 0
					for _, voiceID := range voiceIDs {
						if voiceID == "-1" {
							pendingVoicePlaceholders++
							continue
						}
						for pendingVoicePlaceholders > 0 {
							m.Voices = append(m.Voices, Voice{})
							pendingVoicePlaceholders--
						}
						voice := Voice{}
						var pendingGrace []gpifPendingGrace
						if v, ok := voiceMap[voiceID]; ok {
							beatIDs := splitIDs(v.Beats)
							for _, beatID := range beatIDs {
								if beatID == "-1" {
									continue
								}
								beat := defaultBeat()
								isGrace := false
								graceOnBeat := false
								if b, ok := beatMap[beatID]; ok {
									if r, ok := rhythmMap[b.Rhythm.Ref]; ok {
										duration, durationErr := gpifRhythmToDuration(r)
										if durationErr != nil {
											return nil, fmt.Errorf("rhythm %q: %w", r.ID, durationErr)
										}
										beat.Duration = duration
									}

									beat.Effect.FadeIn = b.Fadding == "FadeIn"
									beat.Effect.Hairpin = gpifHairpin(b.Hairpin)
									gpifApplyBeatEffects(b, &beat)
									if chord, ok := trackChordMaps[trackIdx].resolve(staffIdx, b.Chord); ok {
										beat.Effect.Chord = &chord
									}
									switch b.GraceNotes {
									case "OnBeat":
										isGrace = true
										graceOnBeat = true
									case "BeforeBeat":
										isGrace = true
									}

									velocity := gpifDynamicToVelocity(b.Dynamic)
									beat.Dynamics = velocity
									for _, noteID := range splitIDs(b.Notes) {
										if noteID == "-1" {
											continue
										}
										if n, ok := noteMap[noteID]; ok {
											note, noteErr := gpifNoteToNote(n, len(staff.Strings), staff.PercussionTrack)
											if noteErr != nil {
												return nil, fmt.Errorf("note %q: %w", n.ID, noteErr)
											}
											gpifNormalizePercussionArticulation(track, &note, &fallbackArticulations[trackIdx])
											note.Velocity = velocity
											beat.Notes = append(beat.Notes, note)
										}
									}
									gpifApplyTremoloPicking(b.Tremolo, beat.Notes)
									if len(beat.Notes) == 0 {
										beat.Status = BeatStatusRest
									}
								}
								if isGrace {
									pendingGrace = append(pendingGrace, gpifPendingGrace{beat: beat, onBeat: graceOnBeat})
									continue
								}
								voice.Beats = append(
									voice.Beats,
									gpifApplyPendingGrace(&beat, pendingGrace, staff.PercussionTrack)...,
								)
								pendingGrace = nil
								voice.Beats = append(voice.Beats, beat)
							}
							voice.Beats = append(
								voice.Beats,
								gpifApplyPendingGrace(nil, pendingGrace, staff.PercussionTrack)...,
							)
						}
						m.Voices = append(m.Voices, voice)
					}
				}

				staff.Measures = append(staff.Measures, m)
			}
			trackIdx++
			track.Measures = track.Staves[0].Measures
			track.Strings = track.Staves[0].Strings
		}
	}

	if err := song.finalizeTiming(); err != nil {
		return nil, fmt.Errorf("finalizing GPIF timing: %w", err)
	}

	return song, nil
}
