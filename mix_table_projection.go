// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"slices"
)

type legacyMixEvent struct {
	change   *MixTableChange
	location ScoreLocation
	position float64
}
type legacyMixConflict struct {
	controller string
	location   ScoreLocation
	target     int
}

func legacyMixEvents(song *Song) []legacyMixEvent {
	var result []legacyMixEvent
	for ti := range song.Tracks {
		for si, staff := range gp8ExportStaves(&song.Tracks[ti]) {
			for mi, measure := range staff.Measures {
				if mi >= len(song.MeasureHeaders) {
					continue
				}
				signature := song.MeasureHeaders[mi].TimeSignature
				length := float64(signature.Numerator) * float64(DurationQuarterTime) * 4 / float64(signature.Denominator.Value)
				for vi, voice := range measure.Voices {
					offset := ScoreTime{}
					for bi, beat := range voice.Beats {
						if beat.Effect.MixTableChange != nil {
							result = append(result, legacyMixEvent{beat.Effect.MixTableChange, ScoreLocation{Track: ti, Staff: si, Measure: mi, Voice: vi, Beat: bi}, float64(offset.Numerator()) / float64(offset.Denominator()) / length})
						}
						if !beat.isGrace {
							if duration, err := beat.Duration.ExactScoreTime(); err == nil {
								if next, addErr := offset.Add(duration); addErr == nil {
									offset = next
								}
							}
						}
					}
				}
			}
		}
	}
	return result
}

func copyMixTableSong(song *Song) *Song {
	result := *song
	result.Tracks = slices.Clone(song.Tracks)
	for i := range result.Tracks {
		result.Tracks[i].Sounds = slices.Clone(song.Tracks[i].Sounds)
		result.Tracks[i].SoundAutomations = slices.Clone(song.Tracks[i].SoundAutomations)
	}
	result.TempoAutomations = slices.Clone(song.TempoAutomations)
	result.VolumeAutomations = slices.Clone(song.VolumeAutomations)
	result.PanAutomations = slices.Clone(song.PanAutomations)
	return &result
}

func projectMixTableAutomations(song *Song) (*Song, []legacyMixConflict) {
	if song.importedMixTableAutomations {
		return song, nil
	}
	events := legacyMixEvents(song)
	if len(events) == 0 {
		return song, nil
	}
	result := copyMixTableSong(song)
	var conflicts []legacyMixConflict
	for _, event := range events {
		mt := event.change
		location := event.location
		if mt.Tempo != nil && mt.Tempo.Value > 0 {
			a := TempoAutomation{Bar: location.Measure, Position: event.position, Tempo: float64(mt.Tempo.Value), Linear: true, Text: mt.TempoName, Hidden: mt.HideTempo}
			found := false
			equal := false
			for _, existing := range song.TempoAutomations {
				if existing.Bar == a.Bar && existing.Position == a.Position {
					found = true
					equal = equal || existing == a
				}
			}
			if !found {
				result.TempoAutomations = append(result.TempoAutomations, a)
			} else if !equal {
				conflicts = append(conflicts, legacyMixConflict{"tempo", location, -1})
			}
		}
		for target := range song.Tracks {
			if mt.Volume != nil && mt.Volume.Value >= 0 && mt.Volume.Value <= 16 && (target == location.Track || mt.Volume.AllTracks) {
				a := VolumeAutomation{Track: target, Bar: location.Measure, Position: event.position, Value: float64(mt.Volume.Value) / 16, Linear: true}
				found, equal := false, false
				for _, existing := range song.VolumeAutomations {
					if existing.Track == target && existing.Bar == a.Bar && existing.Position == a.Position {
						found = true
						equal = equal || existing == a
					}
				}
				if !found {
					result.VolumeAutomations = append(result.VolumeAutomations, a)
				} else if !equal {
					conflicts = append(conflicts, legacyMixConflict{"volume", location, target})
				}
			}
			if mt.Balance != nil && mt.Balance.Value >= 0 && mt.Balance.Value <= 16 && (target == location.Track || mt.Balance.AllTracks) {
				a := PanAutomation{Track: target, Bar: location.Measure, Position: event.position, Value: float64(mt.Balance.Value) / 16, Linear: true}
				found, equal := false, false
				for _, existing := range song.PanAutomations {
					if existing.Track == target && existing.Bar == a.Bar && existing.Position == a.Position {
						found = true
						equal = equal || existing == a
					}
				}
				if !found {
					result.PanAutomations = append(result.PanAutomations, a)
				} else if !equal {
					conflicts = append(conflicts, legacyMixConflict{"balance", location, target})
				}
			}
			if mt.Instrument != nil && mt.Instrument.Value >= 0 && mt.Instrument.Value <= 127 && (target == location.Track || mt.Instrument.AllTracks) {
				if projectMixTableInstrument(song, result, target, event) {
					conflicts = append(conflicts, legacyMixConflict{"instrument", location, target})
				}
			}
		}
	}
	slices.SortStableFunc(result.VolumeAutomations, func(a, b VolumeAutomation) int {
		return compareMixPosition(a.Track, a.Bar, a.Position, b.Track, b.Bar, b.Position)
	})
	slices.SortStableFunc(result.PanAutomations, func(a, b PanAutomation) int {
		return compareMixPosition(a.Track, a.Bar, a.Position, b.Track, b.Bar, b.Position)
	})
	return result, conflicts
}

func compareMixPosition(at, ab int, ap float64, bt, bb int, bp float64) int {
	if at != bt {
		return at - bt
	}
	if ab != bb {
		return ab - bb
	}
	if ap < bp {
		return -1
	}
	if ap > bp {
		return 1
	}
	return 0
}

func projectMixTableInstrument(source, result *Song, target int, event legacyMixEvent) bool {
	track := &result.Tracks[target]
	found, equal := false, false
	for _, a := range source.Tracks[target].SoundAutomations {
		if a.Bar == event.location.Measure && a.Position == event.position {
			found = true
			if a.Sound >= 0 && a.Sound < len(source.Tracks[target].Sounds) {
				equal = equal || (source.Tracks[target].Sounds[a.Sound].Program == event.change.Instrument.Value && a.Linear && a.Text == "" && !a.Hidden)
			}
		}
	}
	if found {
		return !equal
	}
	if len(track.Sounds) == 0 {
		channel := defaultMidiChannel()
		if track.ChannelIndex >= 0 && track.ChannelIndex < len(result.Channels) {
			channel = result.Channels[track.ChannelIndex]
		}
		track.Sounds = append(track.Sounds, TrackSound{Name: track.Name, Program: channel.Instrument, Bank: channel.Bank})
	}
	bank := track.Sounds[0].Bank
	// Program changes retain the last selected bank at or before the event.
	lastBar, lastPosition := -1, float64(-1)
	for _, a := range track.SoundAutomations {
		if a.Sound >= 0 && a.Sound < len(track.Sounds) && compareMixPosition(0, a.Bar, a.Position, 0, event.location.Measure, event.position) <= 0 && compareMixPosition(0, a.Bar, a.Position, 0, lastBar, lastPosition) >= 0 {
			bank = track.Sounds[a.Sound].Bank
			lastBar, lastPosition = a.Bar, a.Position
		}
	}
	sound := TrackSound{Name: fmt.Sprintf("Legacy program %d bank %d", event.change.Instrument.Value, bank), Path: "go-guitar-pro/mix-table", Role: "program", Program: event.change.Instrument.Value, Bank: bank}
	index := slices.Index(track.Sounds, sound)
	if index < 0 {
		index = len(track.Sounds)
		track.Sounds = append(track.Sounds, sound)
	}
	track.SoundAutomations = append(track.SoundAutomations, SoundAutomation{Bar: event.location.Measure, Position: event.position, Sound: index, Linear: true})
	return false
}

func readBinaryMixTableAutomations(song *Song) {
	projected, _ := projectMixTableAutomations(song)
	song.TempoAutomations = projected.TempoAutomations
	song.VolumeAutomations = projected.VolumeAutomations
	song.PanAutomations = projected.PanAutomations
	for i := range song.Tracks {
		song.Tracks[i].Sounds = projected.Tracks[i].Sounds
		song.Tracks[i].SoundAutomations = projected.Tracks[i].SoundAutomations
	}
	song.importedMixTableAutomations = true
}

func validateMixTable(change *MixTableChange, location ScoreLocation, diagnostics *[]ScoreDiagnostic) {
	if change == nil {
		return
	}
	for _, check := range []struct {
		name             string
		item             *MixTableItem
		minimum, maximum int32
	}{{"tempo", change.Tempo, 1, 2147483647}, {"instrument", change.Instrument, 0, 127}, {"volume", change.Volume, 0, 16}, {"balance", change.Balance, 0, 16}} {
		if check.item != nil && (check.item.Value < check.minimum || check.item.Value > check.maximum) {
			*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.mix-table." + check.name, Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("legacy %s value %d is outside %d..%d", check.name, check.item.Value, check.minimum, check.maximum)})
		}
	}
}
