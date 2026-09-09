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
	for trackIndex := range song.Tracks {
		track := &song.Tracks[trackIndex]
		if track.Offset < 0 {
			add("score.track.capo", ScoreDiagnosticValue, ScoreLocation{Track: trackIndex}, "capo fret %d is negative", track.Offset)
		}
		if track.ChannelIndex < -1 || track.ChannelIndex >= len(song.Channels) {
			add("score.track.channel-reference", ScoreDiagnosticStructural, ScoreLocation{Track: trackIndex}, "channel index %d is outside -1..%d", track.ChannelIndex, len(song.Channels)-1)
		}
		staves := gp8ExportStaves(track)
		for staffIndex := range staves {
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
				for voiceIndex := range measure.Voices {
					if int(measure.Voices[voiceIndex].MeasureIndex) != measureIndex {
						voiceLocation := location
						voiceLocation.Voice = voiceIndex
						add("score.voice.measure-ownership", ScoreDiagnosticStructural, voiceLocation, "measure index %d does not match owner %d", measure.Voices[voiceIndex].MeasureIndex, measureIndex)
					}
				}
				validateScoreVoices(track, measure, location, &diagnostics)
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

func validateScoreVoices(track *Track, measure *Measure, base ScoreLocation, diagnostics *[]ScoreDiagnostic) {
	for voiceIndex := range measure.Voices {
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
			for noteIndex, note := range beat.Notes {
				if track.PercussionTrack && note.HasPercussionArticulation && (note.PercussionArticulation < 0 || note.PercussionArticulation >= len(track.PercussionArticulations)) {
					noteLocation := location
					noteLocation.Note = noteIndex
					*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.percussion-reference", Kind: ScoreDiagnosticStructural, Location: noteLocation, Reason: fmt.Sprintf("articulation %d is outside 0..%d", note.PercussionArticulation, len(track.PercussionArticulations)-1)})
				}
			}
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
