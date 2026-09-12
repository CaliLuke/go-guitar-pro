// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"math"
)

// ScoreDiagnosticKind classifies a format-independent score invariant.
type ScoreDiagnosticKind string

const (
	// ScoreDiagnosticStructural reports invalid ownership, indexing, or references.
	ScoreDiagnosticStructural ScoreDiagnosticKind = "structural"
	// ScoreDiagnosticTiming reports an invalid duration or finalized start.
	ScoreDiagnosticTiming ScoreDiagnosticKind = "timing"
	// ScoreDiagnosticValue reports an invalid finite value or normalized location.
	ScoreDiagnosticValue ScoreDiagnosticKind = "value"
)

// ScoreLocation identifies one occurrence in the public score hierarchy.
type ScoreLocation struct {
	Track   int
	Staff   int
	Measure int
	Voice   int
	Beat    int
	Note    int
}

// ScoreDiagnostic describes one violation of the format-independent Song contract.
type ScoreDiagnostic struct {
	Code     string
	Kind     ScoreDiagnosticKind
	Location ScoreLocation
	Reason   string
}

// FinalizeSong deterministically derives exact and legacy timing fields.
// Repeating it without changing authored content is idempotent.
func FinalizeSong(song *Song) error {
	if song == nil {
		return fmt.Errorf("finalizing song: song is nil")
	}
	return song.finalizeTiming()
}

// ValidateSong checks format-independent structure and derived timing without mutation.
// Target serialization limits are checked separately by the exporter.
func ValidateSong(song *Song) []ScoreDiagnostic {
	if song == nil {
		return []ScoreDiagnostic{{Code: "score.nil", Kind: ScoreDiagnosticStructural, Reason: "song is nil"}}
	}
	diagnostics := make([]ScoreDiagnostic, 0)
	validateScoreStyle(song.Style, &diagnostics)
	validateScoreDisplay(song.Style, &diagnostics)
	validateHeaderFooter(song, &diagnostics)
	add := func(code string, kind ScoreDiagnosticKind, location ScoreLocation, format string, args ...any) {
		diagnostics = append(diagnostics, ScoreDiagnostic{Code: code, Kind: kind, Location: location, Reason: fmt.Sprintf(format, args...)})
	}
	validateSystemLayout(song.SystemLayout, "score.system-layout", ScoreLocation{}, len(song.MeasureHeaders), &diagnostics)
	validTempoAuthority := song.Tempo > 0
	tempoDiagnostic := false
	if song.Tempo < 0 {
		add("score.tempo", ScoreDiagnosticValue, ScoreLocation{}, "legacy tempo %d must be non-negative", song.Tempo)
		tempoDiagnostic = true
	}
	if song.InitialTempo.State == SourceValueKnown {
		if _, err := NewBPM(float64(song.InitialTempo.Value)); err != nil {
			add("score.tempo", ScoreDiagnosticValue, ScoreLocation{}, "initial tempo: %v", err)
			tempoDiagnostic = true
		} else {
			validTempoAuthority = true
		}
	}
	if !validTempoAuthority {
		for _, automation := range song.TempoAutomations {
			if automation.Bar == 0 && automation.Position == 0 {
				if _, err := NewBPM(automation.Tempo); err == nil {
					validTempoAuthority = true
				}
				break
			}
		}
	}
	if !validTempoAuthority && !tempoDiagnostic {
		add("score.tempo", ScoreDiagnosticValue, ScoreLocation{}, "song has no valid opening tempo")
	}
	for channelIndex, channel := range song.Channels {
		if channel.Bank < 0 || channel.Bank > 16383 {
			add("score.channel.bank", ScoreDiagnosticValue, ScoreLocation{}, "channel %d bank %d is outside 0..16383", channelIndex, channel.Bank)
		}
		for name, value := range map[string]int8{"volume": channel.Volume, "balance": channel.Balance, "chorus": channel.Chorus, "reverb": channel.Reverb, "phaser": channel.Phaser, "tremolo": channel.Tremolo} {
			if value < 0 {
				add("score.channel."+name, ScoreDiagnosticValue, ScoreLocation{}, "channel %d %s %d is outside 0..127", channelIndex, name, value)
			}
		}
	}
	for measureIndex := range song.MeasureHeaders {
		header := &song.MeasureHeaders[measureIndex]
		location := ScoreLocation{Measure: measureIndex}
		validateDisplayScale(header.DisplayScale, "score.master-bar.display-scale", location, &diagnostics)
		if header.TimeSignature.Numerator <= 0 {
			add("score.measure.time-signature", ScoreDiagnosticValue, location, "time-signature numerator %d must be positive", header.TimeSignature.Numerator)
		}
		if _, err := header.TimeSignature.Denominator.MusicalDuration(); err != nil {
			add("score.measure.time-signature", ScoreDiagnosticValue, location, "time-signature denominator: %v", err)
		}
		if header.RepeatCount > maxRepeatCount {
			add("score.measure.repeat-count", ScoreDiagnosticValue, location, "repeat count %d is outside 0..%d", header.RepeatCount, maxRepeatCount)
		}
		for _, direction := range header.resolvedDirections() {
			if direction < DirectionSignCoda || direction > DirectionSignDaDoubleCoda {
				add("score.measure.direction", ScoreDiagnosticValue, location, "direction %d is not defined", direction)
			}
		}
		if header.TripletFeel < TripletFeelNone || header.TripletFeel > TripletFeelScottishSixteenth {
			add("score.measure.triplet-feel", ScoreDiagnosticValue, location, "triplet feel %d is not defined", header.TripletFeel)
		}
		if err := validateBeamingRules(header.BeamingRules); err != nil {
			add("score.measure.beaming-rules", ScoreDiagnosticValue, location, "beaming rules: %v", err)
		}
		measureLength, measureLengthErr := header.ExactLength()
		for fermataIndex, fermata := range header.Fermatas {
			if fermata.Type < FermataTypeShort || fermata.Type > FermataTypeLong {
				add("score.measure.fermata-type", ScoreDiagnosticValue, location, "fermata %d type %d is not defined", fermataIndex, fermata.Type)
			}
			if math.IsNaN(fermata.Length) || math.IsInf(fermata.Length, 0) || fermata.Length < 0 {
				add("score.measure.fermata-length", ScoreDiagnosticValue, location, "fermata %d length %v must be finite and non-negative", fermataIndex, fermata.Length)
			}
			if fermata.Offset.Numerator() < 0 || fermata.Offset.Denominator() <= 0 || measureLengthErr == nil && fermata.Offset.Compare(measureLength) >= 0 {
				add("score.measure.fermata-offset", ScoreDiagnosticTiming, location, "fermata %d offset %d/%d must be within the measure", fermataIndex, fermata.Offset.Numerator(), fermata.Offset.Denominator())
			}
			for earlierIndex := 0; earlierIndex < fermataIndex; earlierIndex++ {
				if fermata.Offset.Compare(header.Fermatas[earlierIndex].Offset) == 0 {
					add("score.measure.fermata-offset", ScoreDiagnosticTiming, location, "fermata %d duplicates offset %d/%d", fermataIndex, fermata.Offset.Numerator(), fermata.Offset.Denominator())
					break
				}
			}
		}
	}
	if len(song.Lyrics.Lines) != 0 && (song.Lyrics.TrackIndex < -1 || song.Lyrics.TrackIndex >= len(song.Tracks)) {
		add("score.lyrics.track-reference", ScoreDiagnosticStructural, ScoreLocation{Track: song.Lyrics.TrackIndex}, "lyrics track index %d is outside -1..%d", song.Lyrics.TrackIndex, len(song.Tracks)-1)
	}
	for lineIndex, line := range song.Lyrics.Lines {
		if line.Text != "" && (line.StartMeasureIndex < -1 || line.StartMeasureIndex >= len(song.MeasureHeaders)) {
			add("score.lyrics.measure-reference", ScoreDiagnosticStructural, ScoreLocation{Measure: line.StartMeasureIndex}, "lyric line %d start measure index %d is outside -1..%d", lineIndex, line.StartMeasureIndex, len(song.MeasureHeaders)-1)
		}
	}
	for trackIndex := range song.Tracks {
		track := &song.Tracks[trackIndex]
		validateSystemLayout(track.SystemLayout, "score.track.system-layout", ScoreLocation{Track: trackIndex}, len(song.MeasureHeaders), &diagnostics)
		if track.CapoFret < 0 {
			add("score.track.capo", ScoreDiagnosticValue, ScoreLocation{Track: trackIndex}, "capo fret %d is negative", track.CapoFret)
		}
		for staffIndex := range track.Staves {
			validatePartialCapo(&track.Staves[staffIndex], ScoreLocation{Track: trackIndex, Staff: staffIndex}, &diagnostics)
			if track.Staves[staffIndex].CapoFret < 0 {
				add("score.staff.capo", ScoreDiagnosticValue, ScoreLocation{Track: trackIndex, Staff: staffIndex}, "capo fret %d is negative", track.Staves[staffIndex].CapoFret)
			}
		}
		if track.ChannelIndex < -1 || track.ChannelIndex >= len(song.Channels) {
			add("score.track.channel-reference", ScoreDiagnosticStructural, ScoreLocation{Track: trackIndex}, "channel index %d is outside -1..%d", track.ChannelIndex, len(song.Channels)-1)
		} else if track.ChannelIndex >= 0 {
			channel := song.Channels[track.ChannelIndex]
			if channel.Instrument < 0 || channel.Instrument > 127 {
				add("score.channel.instrument", ScoreDiagnosticValue, ScoreLocation{Track: trackIndex}, "selected channel %d instrument %d is outside 0..127", track.ChannelIndex, channel.Instrument)
			}
		}
		if track.PercussionTrack {
			for articulationIndex, articulation := range track.PercussionArticulations {
				location := ScoreLocation{Track: trackIndex}
				if articulation.OutputMIDINumber < 0 || articulation.OutputMIDINumber > 127 {
					add("score.percussion-articulation.output-midi", ScoreDiagnosticValue, location, "percussion articulation %d output MIDI value %d is outside 0..127", articulationIndex, articulation.OutputMIDINumber)
				}
				for inputIndex, input := range articulation.InputMIDINumbers {
					if input < 0 || input > 127 {
						add("score.percussion-articulation.input-midi", ScoreDiagnosticValue, location, "percussion articulation %d input MIDI value %d at index %d is outside 0..127", articulationIndex, input, inputIndex)
					}
				}
			}
		}
		staves := gp8ExportStaves(track)
		for staffIndex := range staves {
			tiedNotes := make(map[int]map[int8]int16)
			pedalDown := false
			if len(staves[staffIndex].Measures) != len(song.MeasureHeaders) {
				add("score.staff.measure-count", ScoreDiagnosticStructural, ScoreLocation{Track: trackIndex, Staff: staffIndex}, "measure count %d does not match header count %d", len(staves[staffIndex].Measures), len(song.MeasureHeaders))
			}
			for measureIndex := range staves[staffIndex].Measures {
				measure := &staves[staffIndex].Measures[measureIndex]
				location := ScoreLocation{Track: trackIndex, Staff: staffIndex, Measure: measureIndex}
				validateDisplayScale(measure.DisplayScale, "score.bar.display-scale", location, &diagnostics)
				if measure.HeaderIndex < 0 || measure.HeaderIndex >= len(song.MeasureHeaders) {
					add("score.measure.header-reference", ScoreDiagnosticStructural, location, "header index %d is outside 0..%d", measure.HeaderIndex, len(song.MeasureHeaders)-1)
				} else {
					header := &song.MeasureHeaders[measure.HeaderIndex]
					if measure.HeaderIndex != measureIndex {
						add("score.measure.header-alignment", ScoreDiagnosticStructural, location, "header index %d does not match measure position %d", measure.HeaderIndex, measureIndex)
					}
					if measure.Start != header.Start {
						add("score.measure.start", ScoreDiagnosticTiming, location, "start %d does not match header start %d", measure.Start, header.Start)
					}
					measureStart := scoreTimeOrLegacy(measure.ExactStart, measure.Start)
					headerStart := scoreTimeOrLegacy(header.ExactStart, header.Start)
					if measureStart.Compare(headerStart) != 0 {
						add("score.measure.exact-start", ScoreDiagnosticTiming, location, "exact start %d/%d does not match header exact start %d/%d", measureStart.Numerator(), measureStart.Denominator(), headerStart.Numerator(), headerStart.Denominator())
					}
				}
				if measure.TrackIndex != trackIndex {
					add("score.measure.track-ownership", ScoreDiagnosticStructural, location, "track index %d does not match owner %d", measure.TrackIndex, trackIndex)
				}
				if measure.StaffIndex != staffIndex {
					add("score.measure.staff-ownership", ScoreDiagnosticStructural, location, "staff index %d does not match owner %d", measure.StaffIndex, staffIndex)
				}
				if measure.SimileMark < SimileMarkNone || measure.SimileMark > SimileMarkSecondOfDouble {
					add("score.measure.simile-mark", ScoreDiagnosticValue, location, "simile mark %d is not defined", measure.SimileMark)
				}
				if measure.ClefOctave < OctaveNone || measure.ClefOctave > OctaveQuindicesimaBassa {
					add("score.measure.clef-octave", ScoreDiagnosticValue, location, "clef octave %d is not defined", measure.ClefOctave)
				}
				if staffIndex > 0 && len(measure.SustainPedals) != 0 {
					add("score.measure.sustain-pedal.staff", ScoreDiagnosticStructural, location, "sustain-pedal markers must belong to staff 0")
				}
				enteringDown := pedalDown
				for markerIndex, marker := range measure.SustainPedals {
					if marker.Type < SustainPedalTypeDown || marker.Type > SustainPedalTypeRelease {
						add("score.measure.sustain-pedal.type", ScoreDiagnosticValue, location, "sustain-pedal marker %d type %d is not defined", markerIndex, marker.Type)
					}
					if math.IsNaN(marker.Position) || math.IsInf(marker.Position, 0) || marker.Position < 0 || marker.Position > 1 {
						add("score.measure.sustain-pedal.position", ScoreDiagnosticValue, location, "sustain-pedal marker %d position %v must be finite and within 0..1", markerIndex, marker.Position)
					}
					if markerIndex > 0 && marker.Position <= measure.SustainPedals[markerIndex-1].Position {
						add("score.measure.sustain-pedal.order", ScoreDiagnosticValue, location, "sustain-pedal marker %d position %v must be after %v", markerIndex, marker.Position, measure.SustainPedals[markerIndex-1].Position)
					}
					switch marker.Type {
					case SustainPedalTypeDown:
						if enteringDown {
							add("score.measure.sustain-pedal.down", ScoreDiagnosticValue, location, "down marker %d cannot occur in a bar entered with the pedal down because GPIF consumers reinterpret it as hold", markerIndex)
						}
						pedalDown = true
					case SustainPedalTypeHold:
						if len(measure.SustainPedals) != 1 || marker.Position != 0 || !enteringDown {
							add("score.measure.sustain-pedal.hold", ScoreDiagnosticValue, location, "hold must be the sole position-0 marker in a bar entered with the pedal down")
						}
						pedalDown = true
					case SustainPedalTypeRelease:
						pedalDown = false
					}
				}
				for voiceIndex := range measure.Voices {
					if int(measure.Voices[voiceIndex].MeasureIndex) != measureIndex {
						voiceLocation := location
						voiceLocation.Voice = voiceIndex
						add("score.voice.measure-ownership", ScoreDiagnosticStructural, voiceLocation, "measure index %d does not match owner %d", measure.Voices[voiceIndex].MeasureIndex, measureIndex)
					}
				}
				validateScoreVoices(track, &staves[staffIndex], measure, location, tiedNotes, &diagnostics)
			}
		}
		for automationIndex, automation := range track.SoundAutomations {
			if automation.Sound < 0 || automation.Sound >= len(track.Sounds) {
				add("score.sound-automation.reference", ScoreDiagnosticStructural, ScoreLocation{Track: trackIndex}, "sound automation %d references sound %d", automationIndex, automation.Sound)
			}
			if automation.Bar < 0 || automation.Bar >= len(song.MeasureHeaders) || !validSoundAutomationPosition(automation.Bar, automation.Position) {
				add("score.sound-automation.location", ScoreDiagnosticValue, ScoreLocation{Track: trackIndex}, "sound automation %d has bar %d position %v; bar must reference a measure and position must be finite within 0..1, or -0.125..0 in bar 0", automationIndex, automation.Bar, automation.Position)
			}
		}
		soundReferences := make(map[string]int, len(track.Sounds))
		for soundIndex, sound := range track.Sounds {
			reference := sound.Path + ";" + sound.Name + ";" + sound.Role
			if previous, exists := soundReferences[reference]; exists {
				add("score.track-sound.reference-collision", ScoreDiagnosticStructural, ScoreLocation{Track: trackIndex}, "sound %d and sound %d have the same GPIF reference %q", soundIndex, previous, reference)
			} else {
				soundReferences[reference] = soundIndex
			}
			if sound.Bank < 0 || sound.Bank > 16383 {
				add("score.track-sound.bank", ScoreDiagnosticValue, ScoreLocation{Track: trackIndex}, "sound %d bank %d is outside 0..16383", soundIndex, sound.Bank)
			}
		}
	}
	framePadding := int64(0)
	if song.BackingTrack != nil {
		framePadding = song.BackingTrack.FramePadding
	}
	for index, point := range song.SyncPoints {
		location := ScoreLocation{Measure: point.Bar}
		if point.Bar < 0 || point.Bar >= len(song.MeasureHeaders) || math.IsNaN(point.Position) || math.IsInf(point.Position, 0) || point.Position < 0 || point.Position > 1 || point.BarOccurrence < 0 {
			add("score.sync-point.location", ScoreDiagnosticValue, location, "sync point %d has bar %d position %v occurrence %d", index, point.Bar, point.Position, point.BarOccurrence)
		}
		if point.FrameOffset < 0 || point.AudioFrame < 0 {
			add("score.sync-point.frame", ScoreDiagnosticValue, location, "sync point %d has frame offset %d and audio frame %d", index, point.FrameOffset, point.AudioFrame)
		}
		expectedMediaTime := (float64(point.AudioFrame) - float64(framePadding)) / GPIFBackingTrackSampleRate * 1000
		if math.IsNaN(point.MediaTimeMS) || math.IsInf(point.MediaTimeMS, 0) || point.MediaTimeMS != expectedMediaTime {
			add("score.sync-point.media-time", ScoreDiagnosticTiming, location, "sync point %d media time %v does not match derived value %v", index, point.MediaTimeMS, expectedMediaTime)
		}
		if math.IsNaN(point.ModifiedTempo) || math.IsInf(point.ModifiedTempo, 0) || point.ModifiedTempo < 0 || math.IsNaN(point.OriginalTempo) || math.IsInf(point.OriginalTempo, 0) || point.OriginalTempo < 0 {
			add("score.sync-point.tempo", ScoreDiagnosticValue, location, "sync point %d has modified tempo %v and original tempo %v", index, point.ModifiedTempo, point.OriginalTempo)
		}
	}
	for index, automation := range song.TempoAutomations {
		if automation.Bar < 0 || automation.Bar >= len(song.MeasureHeaders) || math.IsNaN(automation.Position) || math.IsInf(automation.Position, 0) || automation.Position < 0 || automation.Position > 1 || math.IsNaN(automation.Tempo) || math.IsInf(automation.Tempo, 0) || automation.Tempo <= 0 {
			add("score.tempo-automation.value", ScoreDiagnosticValue, ScoreLocation{}, "tempo automation %d has bar %d position %v tempo %v", index, automation.Bar, automation.Position, automation.Tempo)
		}
	}
	validatePanAutomations(song, &diagnostics)
	for index, automation := range song.VolumeAutomations {
		if automation.Track < 0 || automation.Track >= len(song.Tracks) || automation.Bar < 0 || automation.Bar >= len(song.MeasureHeaders) || math.IsNaN(automation.Position) || math.IsInf(automation.Position, 0) || automation.Position < 0 || automation.Position > 1 || math.IsNaN(automation.Value) || math.IsInf(automation.Value, 0) || automation.Value < 0 || automation.Value > 1 {
			add("score.volume-automation.value", ScoreDiagnosticValue, ScoreLocation{Track: automation.Track, Measure: automation.Bar}, "volume automation %d has track %d bar %d position %v value %v", index, automation.Track, automation.Bar, automation.Position, automation.Value)
		}
	}
	return diagnostics
}

func authoredScoreDiagnostics(song *Song) []ScoreDiagnostic {
	derivedCodes := map[string]struct{}{
		"score.measure.header-alignment": {},
		"score.measure.track-ownership":  {},
		"score.measure.staff-ownership":  {},
		"score.voice.measure-ownership":  {},
		"score.measure.start":            {},
		"score.measure.exact-start":      {},
		"score.beat.start":               {},
		"score.beat.exact-start":         {},
		// A tie can begin outside an imported excerpt. Report it to callers, but
		// do not prevent GP8 from preserving the destination marker.
		"score.note.tie-destination": {},
	}
	diagnostics := ValidateSong(song)
	authored := make([]ScoreDiagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		if _, derived := derivedCodes[diagnostic.Code]; !derived {
			authored = append(authored, diagnostic)
		}
	}
	return authored
}

func validateScoreVoices(track *Track, staff *Staff, measure *Measure, base ScoreLocation, tiedNotes map[int]map[int8]int16, diagnostics *[]ScoreDiagnostic) {
	for voiceIndex := range measure.Voices {
		if tiedNotes[voiceIndex] == nil {
			tiedNotes[voiceIndex] = make(map[int8]int16)
		}
		expected := scoreTimeOrLegacy(measure.ExactStart, measure.Start)
		for beatIndex := range measure.Voices[voiceIndex].Beats {
			beat := &measure.Voices[voiceIndex].Beats[beatIndex]
			location := base
			location.Voice = voiceIndex
			location.Beat = beatIndex
			if beat.Start != nil && *beat.Start != expected.FloorTicks() {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.start", Kind: ScoreDiagnosticTiming, Location: location, Reason: fmt.Sprintf("start %d does not match finalized start %d", *beat.Start, expected.FloorTicks())})
			}
			if beat.ExactStart != nil && beat.ExactStart.Compare(expected) != 0 {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.exact-start", Kind: ScoreDiagnosticTiming, Location: location, Reason: fmt.Sprintf("exact start %d/%d does not match finalized start %d/%d", beat.ExactStart.Numerator(), beat.ExactStart.Denominator(), expected.Numerator(), expected.Denominator())})
			}
			if beat.Dynamics < 0 || beat.Dynamics > 127 {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.dynamics", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("beat dynamics %d must be absent (0) or within MIDI velocity 1..127", beat.Dynamics)})
			}
			if beat.BarreFret == nil {
				if beat.BarreShape != BarreShapeNone {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.barre-pair", Kind: ScoreDiagnosticValue, Location: location, Reason: "barre shape requires a barre fret"})
				}
			} else {
				if *beat.BarreFret < 0 {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.barre-fret", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("barre fret %d must be non-negative", *beat.BarreFret)})
				}
				if beat.BarreShape == BarreShapeNone {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.barre-pair", Kind: ScoreDiagnosticValue, Location: location, Reason: "barre fret requires a full or half shape"})
				}
			}
			if beat.BarreShape > BarreShapeHalf {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.barre-shape", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("barre shape %d is not defined", beat.BarreShape)})
			}
			if beat.BeamingMode < BeatBeamingAuto || beat.BeamingMode > BeatBeamingForceSplitSecondary {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.beaming-mode", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("beaming mode %d is not defined", beat.BeamingMode)})
			}
			if beat.PreferredBeamDirection < VoiceDirectionNone || beat.PreferredBeamDirection > VoiceDirectionDown {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.beam-direction", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("preferred beam direction %d is not defined", beat.PreferredBeamDirection)})
			}
			if beat.Effect.Chord != nil {
				if err := validateChordDiagramStructure(beat.Effect.Chord); err != nil {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.chord.diagram", Kind: ScoreDiagnosticValue, Location: location, Reason: err.Error()})
				}
			}
			if pick := beat.Effect.PickStroke; pick < BeatStrokeDirectionNone || pick > BeatStrokeDirectionDown {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.pick-stroke", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("pick-stroke direction %d is not defined", pick)})
			}
			stroke := beat.Effect.Stroke
			if stroke.Kind > BeatStrokeKindArpeggio {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.stroke-kind", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("stroke kind %d is not defined", stroke.Kind)})
			}
			if stroke.Direction < BeatStrokeDirectionNone || stroke.Direction > BeatStrokeDirectionDown {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.stroke-direction", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("stroke direction %d is not defined", stroke.Direction)})
			}
			if stroke.Kind != BeatStrokeKindNone && stroke.Direction == BeatStrokeDirectionNone {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.stroke-pair", Kind: ScoreDiagnosticValue, Location: location, Reason: "stroke kind requires a direction"})
			}
			if stroke.Direction == BeatStrokeDirectionNone && (stroke.Duration != 0 || stroke.ExactDuration != nil) {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.stroke-pair", Kind: ScoreDiagnosticValue, Location: location, Reason: "stroke timing requires a direction"})
			}
			if stroke.Direction != BeatStrokeDirectionNone && stroke.Duration != 0 {
				if _, ok := beatStrokeDurationTicks(stroke.Duration); !ok {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.stroke-duration", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("stroke duration %d is not a supported note value", stroke.Duration)})
				}
			}
			duration, err := beat.Duration.ExactScoreTime()
			if err != nil {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.duration", Kind: ScoreDiagnosticTiming, Location: location, Reason: err.Error()})
				continue
			}
			if !beat.isGrace {
				if next, addErr := expected.Add(duration); addErr == nil {
					expected = next
				} else {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.duration", Kind: ScoreDiagnosticTiming, Location: location, Reason: addErr.Error()})
				}
			}
			validateMixTable(beat.Effect.MixTableChange, location, diagnostics)
			validateBendEffect(beat.Effect.TremoloBar, "score.beat.whammy", location, diagnostics)
			if beat.Effect.VibratoStrength > BeatVibratoWide {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.vibrato", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("beat vibrato %d is not defined", beat.Effect.VibratoStrength)})
			}
			if beat.Effect.WahPedal > WahPedalClosed {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.wah", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("wah pedal state %d is not defined", beat.Effect.WahPedal)})
			}
			if beat.Effect.SlapEffect > SlapEffectPopping {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.slap-effect", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("slap effect %d is not defined", beat.Effect.SlapEffect)})
			}
			if beat.Effect.RasgueadoPattern > RasgueadoPeami {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.rasgueado", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("rasgueado pattern %d is not defined", beat.Effect.RasgueadoPattern)})
			}
			if beat.Effect.Fade > BeatFadeVolumeSwell {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.fade", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("beat fade %d is not defined", beat.Effect.Fade)})
			}
			if beat.Timer != nil && beat.Timer.Milliseconds != nil && (*beat.Timer.Milliseconds < 0 || *beat.Timer.Milliseconds > maxBeatTimerMilliseconds) {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.timer", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("timer milliseconds %d is outside 0..%d", *beat.Timer.Milliseconds, maxBeatTimerMilliseconds)})
			}
			validateGraceTimers(beat, location, diagnostics)
			if beat.Effect.Golpe > GolpeTypeFinger {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.golpe", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("beat golpe %d is not defined", beat.Effect.Golpe)})
			}
			if tremolo, _ := beat.resolvedTremoloPicking(); tremolo != nil {
				if _, err := tremolo.Duration.MusicalDuration(); err != nil {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.tremolo-picking-duration", Kind: ScoreDiagnosticValue, Location: location, Reason: err.Error()})
				}
				if tremolo.Style > TremoloPickingStyleBuzzRoll {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.beat.tremolo-picking-style", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("tremolo-picking style %d is not defined", tremolo.Style)})
				}
			}
			for noteIndex, note := range beat.Notes {
				noteLocation := location
				noteLocation.Note = noteIndex
				percussion := track.PercussionTrack || staff.PercussionTrack
				validateBendEffect(note.Effect.Bend, "score.note.bend", noteLocation, diagnostics)
				validateHarmonicEffect(note.Effect.Harmonic, noteLocation, diagnostics)
				validateNoteAccidental(staff, measure, beat, &note, percussion, noteLocation, diagnostics)
				if note.Effect.LeftHandFinger < FingeringOpen || note.Effect.LeftHandFinger > FingeringLittle {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.left-hand-fingering", Kind: ScoreDiagnosticValue, Location: noteLocation, Reason: fmt.Sprintf("left-hand fingering %d is not defined", note.Effect.LeftHandFinger)})
				}
				if note.Effect.RightHandFinger < FingeringOpen || note.Effect.RightHandFinger > FingeringLittle {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.right-hand-fingering", Kind: ScoreDiagnosticValue, Location: noteLocation, Reason: fmt.Sprintf("right-hand fingering %d is not defined", note.Effect.RightHandFinger)})
				}
				for _, slide := range note.Effect.Slides {
					if slide < SlideIntoFromAbove || slide > SlidePickSlideUp {
						*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.slide", Kind: ScoreDiagnosticValue, Location: noteLocation, Reason: fmt.Sprintf("slide %d is not defined", slide)})
					}
				}
				if note.Ornament > NoteOrnamentLowerMordent {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.ornament", Kind: ScoreDiagnosticValue, Location: noteLocation, Reason: fmt.Sprintf("note ornament %d is not defined", note.Ornament)})
				}
				if note.Effect.VibratoStrength > NoteVibratoWide {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.vibrato", Kind: ScoreDiagnosticValue, Location: noteLocation, Reason: fmt.Sprintf("note vibrato %d is not defined", note.Effect.VibratoStrength)})
				}
				if note.Effect.Accent > NoteAccentTenuto {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.accent", Kind: ScoreDiagnosticValue, Location: noteLocation, Reason: fmt.Sprintf("note accent %d is not defined", note.Effect.Accent)})
				}
				if percussion && note.HasPercussionArticulation && (note.PercussionArticulation < 0 || note.PercussionArticulation >= len(track.PercussionArticulations)) {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.percussion-reference", Kind: ScoreDiagnosticStructural, Location: noteLocation, Reason: fmt.Sprintf("articulation %d is outside 0..%d", note.PercussionArticulation, len(track.PercussionArticulations)-1)})
				}
				if percussion {
					for graceIndex, grace := range note.Effect.Graces {
						if grace.HasPercussionArticulation && (grace.PercussionArticulation < 0 || grace.PercussionArticulation >= len(track.PercussionArticulations)) {
							*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.grace.percussion-reference", Kind: ScoreDiagnosticStructural, Location: noteLocation, Reason: fmt.Sprintf("grace %d uses percussion articulation %d with %d definitions", graceIndex, grace.PercussionArticulation, len(track.PercussionArticulations))})
						}
					}
				}
				if math.IsNaN(float64(note.DurationPercent)) || math.IsInf(float64(note.DurationPercent), 0) || note.DurationPercent < 0 {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.duration-percent", Kind: ScoreDiagnosticValue, Location: noteLocation, Reason: fmt.Sprintf("duration percent %v must be finite and non-negative", note.DurationPercent)})
				}
				if note.Kind < NoteTypeNormal || note.Kind > NoteTypeDead {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.kind", Kind: ScoreDiagnosticValue, Location: noteLocation, Reason: fmt.Sprintf("note kind %d is not valid for a sounded note", note.Kind)})
				}
				if !percussion && (note.String < 0 || int(note.String) > len(staff.Strings)) {
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.string", Kind: ScoreDiagnosticValue, Location: noteLocation, Reason: fmt.Sprintf("string %d is outside 0..%d", note.String, len(staff.Strings))})
				}
				if !percussion {
					validatePartialCapoPitches(staff, &note, noteLocation, diagnostics)
				}
				if !percussion && staff.TranspositionPitch != 0 && note.String >= 0 && int(note.String) <= len(staff.Strings) {
					midi := soundingNoteMIDI(staff, &note)
					if midi < 0 || midi > 127 {
						*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.sounding-midi", Kind: ScoreDiagnosticValue, Location: noteLocation, Reason: fmt.Sprintf("sounding MIDI value %d after staff transposition %d is outside 0..127", midi, staff.TranspositionPitch)})
					}
				}
				if !percussion && note.Kind == NoteTypeTie {
					previous, ok := tiedNotes[voiceIndex][note.String]
					if !ok || previous != note.Value {
						*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.tie-destination", Kind: ScoreDiagnosticStructural, Location: noteLocation, Reason: fmt.Sprintf("tie destination on string %d with value %d has no matching earlier note", note.String, note.Value)})
					}
				}
				tiedNotes[voiceIndex][note.String] = note.Value
			}
		}
	}
}

func validateChordDiagramStructure(chord *Chord) error {
	stringCount := len(chord.Strings)
	if chord.Fingerings != nil && len(chord.Fingerings) != stringCount {
		return fmt.Errorf("chord fingering count %d does not match string count %d", len(chord.Fingerings), stringCount)
	}
	for index, finger := range chord.Fingerings {
		if finger < FingeringUnknown || finger > FingeringLittle {
			return fmt.Errorf("chord fingering %d at string position %d is not defined", finger, index+1)
		}
	}
	for index, barre := range chord.Barres {
		if barre.Fret <= 0 {
			return fmt.Errorf("chord barre %d fret %d must be positive", index, barre.Fret)
		}
		if barre.Start < 1 || int(barre.Start) > stringCount || barre.End < 1 || int(barre.End) > stringCount {
			return fmt.Errorf("chord barre %d endpoints %d..%d are outside 1..%d", index, barre.Start, barre.End, stringCount)
		}
		if barre.Start >= barre.End {
			return fmt.Errorf("chord barre %d endpoints %d..%d must identify two strings in highest-to-lowest order", index, barre.Start, barre.End)
		}
	}
	return nil
}

func validateGP8ChordDiagram(chord *Chord) error {
	if err := validateChordDiagramStructure(chord); err != nil {
		return err
	}
	for index, finger := range chord.Fingerings {
		if finger == FingeringUnknown {
			continue
		}
		if finger == FingeringOpen && chord.Strings[index] > 0 {
			return fmt.Errorf("none fingering at string position %d requires a muted or open string", index+1)
		}
		if finger != FingeringOpen && chord.Strings[index] <= 0 {
			return fmt.Errorf("muted or open string position %d cannot use finger %d", index+1, finger)
		}
	}
	if chord.Fingerings == nil && len(chord.Barres) > 5 {
		return fmt.Errorf("%d chord barres need more than five distinct GPIF fingers", len(chord.Barres))
	}
	barresByKey := make(map[gpifChordBarreKey]Barre, len(chord.Barres))
	usedSyntheticEndpoints := make(map[int]struct{}, len(chord.Barres)*2)
	for index, barre := range chord.Barres {
		startIndex, endIndex := int(barre.Start)-1, int(barre.End)-1
		if chord.Strings[startIndex] != barre.Fret || chord.Strings[endIndex] != barre.Fret {
			return fmt.Errorf("chord barre %d fret %d does not match endpoint frets %d and %d", index, barre.Fret, chord.Strings[startIndex], chord.Strings[endIndex])
		}
		if chord.Fingerings == nil {
			for _, endpoint := range []int{startIndex, endIndex} {
				if _, exists := usedSyntheticEndpoints[endpoint]; exists {
					return fmt.Errorf("chord barre %d shares endpoint position %d with another barre", index, endpoint+1)
				}
				usedSyntheticEndpoints[endpoint] = struct{}{}
			}
			continue
		}
		finger := chord.Fingerings[startIndex]
		if finger == FingeringUnknown || finger == FingeringOpen || chord.Fingerings[endIndex] != finger {
			return fmt.Errorf("chord barre %d endpoints must share one known fretting finger", index)
		}
		key := gpifChordBarreKey{finger: finger, fret: barre.Fret}
		if _, exists := barresByKey[key]; exists {
			return fmt.Errorf("chord barre %d duplicates finger %d at fret %d", index, finger, barre.Fret)
		}
		barresByKey[key] = barre
	}
	if chord.Fingerings != nil {
		groups := make(map[gpifChordBarreKey][]int)
		for index, finger := range chord.Fingerings {
			if finger == FingeringUnknown || finger == FingeringOpen {
				continue
			}
			key := gpifChordBarreKey{finger: finger, fret: chord.Strings[index]}
			groups[key] = append(groups[key], index+1)
		}
		for key, positions := range groups {
			if len(positions) < 2 {
				continue
			}
			start, end := positions[0], positions[0]
			for _, position := range positions[1:] {
				start = min(start, position)
				end = max(end, position)
			}
			barre, exists := barresByKey[key]
			if !exists || barre.Start != int8(start) || barre.End != int8(end) {
				return fmt.Errorf("repeated finger %d at fret %d requires an exact matching barre range", key.finger, key.fret)
			}
		}
	}
	return nil
}

func validateHarmonicEffect(effect *HarmonicEffect, location ScoreLocation, diagnostics *[]ScoreDiagnostic) {
	if effect == nil {
		return
	}
	if effect.Kind < HarmonicTypeNatural || effect.Kind > HarmonicTypeFeedback {
		*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.harmonic.kind", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("harmonic kind %d is not defined", effect.Kind)})
	}
	if effect.Fret != nil && *effect.Fret < 0 {
		*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.harmonic.fret", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("harmonic fret %d must be non-negative", *effect.Fret)})
	}
	if effect.FretFloat != nil {
		fret := *effect.FretFloat
		if math.IsNaN(fret) || math.IsInf(fret, 0) || fret < 0 || fret > math.MaxInt8 {
			*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.harmonic.fret-float", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("exact harmonic fret %v must be finite and within 0..%d", fret, math.MaxInt8)})
		}
	}
}

func validateBendEffect(effect *BendEffect, codePrefix string, location ScoreLocation, diagnostics *[]ScoreDiagnostic) {
	if effect == nil {
		return
	}
	if effect.Kind < BendTypeNone || effect.Kind > BendTypeReleaseDown {
		*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: codePrefix + ".kind", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("bend kind %d is not defined", effect.Kind)})
	}
	previousOffset := float64(0)
	previousOffsetValid := false
	roleTuple := nonmonotonicBendControlRoles(effect.Points)
	for index, point := range effect.Points {
		currentOffsetValid := true
		if point.ExactOffset != nil {
			offset := *point.ExactOffset
			if math.IsNaN(offset) || math.IsInf(offset, 0) || offset < 0 || offset > 100 {
				*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: codePrefix + ".exact-offset", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("bend point %d exact offset %v must be finite and within 0..100", index, offset)})
				currentOffsetValid = false
			}
		}
		if point.Position > uint8(BendEffectMaxPosition) {
			*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: codePrefix + ".position", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("bend point %d position %d is outside 0..%d", index, point.Position, uint8(BendEffectMaxPosition))})
			currentOffsetValid = false
		}
		currentOffset := resolvedBendOffset(point)
		if index > 0 && previousOffsetValid && currentOffsetValid && currentOffset < previousOffset && !roleTuple {
			*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: codePrefix + ".order", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("bend point %d offset %v precedes offset %v", index, currentOffset, previousOffset)})
		}
		previousOffset = currentOffset
		previousOffsetValid = currentOffsetValid
	}
}

func scoreTimeOrLegacy(exact ScoreTime, legacy int64) ScoreTime {
	if exact != (ScoreTime{}) {
		return exact
	}
	value, err := NewScoreTime(legacy, 1)
	if err != nil {
		return ScoreTime{}
	}
	return value
}

// validSoundAutomationPosition applies the evidenced opening-preroll contract.
// Ordered comparisons also reject NaN and both infinities.
func validSoundAutomationPosition(bar int, position float64) bool {
	return position <= 1 && (position >= 0 || bar == 0 && position >= -0.125)
}
