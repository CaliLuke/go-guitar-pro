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
	add := func(code string, kind ScoreDiagnosticKind, location ScoreLocation, format string, args ...any) {
		diagnostics = append(diagnostics, ScoreDiagnostic{Code: code, Kind: kind, Location: location, Reason: fmt.Sprintf(format, args...)})
	}
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
		if channel.Instrument < 0 || channel.Instrument > 127 {
			add("score.channel.instrument", ScoreDiagnosticValue, ScoreLocation{}, "channel %d instrument %d is outside 0..127", channelIndex, channel.Instrument)
		}
		if channel.Bank > 127 {
			add("score.channel.bank", ScoreDiagnosticValue, ScoreLocation{}, "channel %d bank %d is outside 0..127", channelIndex, channel.Bank)
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
		if header.TimeSignature.Numerator <= 0 {
			add("score.measure.time-signature", ScoreDiagnosticValue, location, "time-signature numerator %d must be positive", header.TimeSignature.Numerator)
		}
		if _, err := header.TimeSignature.Denominator.MusicalDuration(); err != nil {
			add("score.measure.time-signature", ScoreDiagnosticValue, location, "time-signature denominator: %v", err)
		}
		if header.RepeatCount > maxRepeatCount {
			add("score.measure.repeat-count", ScoreDiagnosticValue, location, "repeat count %d is outside 0..%d", header.RepeatCount, maxRepeatCount)
		}
		if header.Direction != nil && (*header.Direction < DirectionSignCoda || *header.Direction > DirectionSignDaDoubleCoda) {
			add("score.measure.direction", ScoreDiagnosticValue, location, "direction %d is not defined", *header.Direction)
		}
		if header.TripletFeel < TripletFeelNone || header.TripletFeel > TripletFeelScottishSixteenth {
			add("score.measure.triplet-feel", ScoreDiagnosticValue, location, "triplet feel %d is not defined", header.TripletFeel)
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
		if track.CapoFret < 0 {
			add("score.track.capo", ScoreDiagnosticValue, ScoreLocation{Track: trackIndex}, "capo fret %d is negative", track.CapoFret)
		}
		for staffIndex := range track.Staves {
			if track.Staves[staffIndex].CapoFret < 0 {
				add("score.staff.capo", ScoreDiagnosticValue, ScoreLocation{Track: trackIndex, Staff: staffIndex}, "capo fret %d is negative", track.Staves[staffIndex].CapoFret)
			}
		}
		if track.ChannelIndex < -1 || track.ChannelIndex >= len(song.Channels) {
			add("score.track.channel-reference", ScoreDiagnosticStructural, ScoreLocation{Track: trackIndex}, "channel index %d is outside -1..%d", track.ChannelIndex, len(song.Channels)-1)
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
			if len(staves[staffIndex].Measures) != len(song.MeasureHeaders) {
				add("score.staff.measure-count", ScoreDiagnosticStructural, ScoreLocation{Track: trackIndex, Staff: staffIndex}, "measure count %d does not match header count %d", len(staves[staffIndex].Measures), len(song.MeasureHeaders))
			}
			for measureIndex := range staves[staffIndex].Measures {
				measure := &staves[staffIndex].Measures[measureIndex]
				location := ScoreLocation{Track: trackIndex, Staff: staffIndex, Measure: measureIndex}
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
			if automation.Bar < 0 || automation.Bar >= len(song.MeasureHeaders) || math.IsNaN(automation.Position) || math.IsInf(automation.Position, 0) || automation.Position < 0 || automation.Position > 1 {
				add("score.sound-automation.location", ScoreDiagnosticValue, ScoreLocation{Track: trackIndex}, "sound automation %d has bar %d position %v", automationIndex, automation.Bar, automation.Position)
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
			validateBendEffect(beat.Effect.TremoloBar, "score.beat.whammy", location, diagnostics)
			for noteIndex, note := range beat.Notes {
				noteLocation := location
				noteLocation.Note = noteIndex
				percussion := track.PercussionTrack || staff.PercussionTrack
				validateBendEffect(note.Effect.Bend, "score.note.bend", noteLocation, diagnostics)
				validateHarmonicEffect(note.Effect.Harmonic, noteLocation, diagnostics)
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
	for index, point := range effect.Points {
		if point.Position > uint8(BendEffectMaxPosition) {
			*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: codePrefix + ".position", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("bend point %d position %d is outside 0..%d", index, point.Position, uint8(BendEffectMaxPosition))})
		}
		if index > 0 && point.Position < effect.Points[index-1].Position {
			*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: codePrefix + ".order", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("bend point %d position %d precedes position %d", index, point.Position, effect.Points[index-1].Position)})
		}
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
