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
			a := TempoAutomation{Bar: location.Measure, Position: event.position, Tempo: float64(mt.Tempo.Value), Linear: false, Text: mt.TempoName, Hidden: mt.HideTempo}
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
			conflicts = append(conflicts, projectMixTableLevels(song, result, target, event)...)
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
				equal = equal || (source.Tracks[target].Sounds[a.Sound].Program == event.change.Instrument.Value && !a.Linear && a.Text == "" && !a.Hidden)
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
	track.SoundAutomations = append(track.SoundAutomations, SoundAutomation{Bar: event.location.Measure, Position: event.position, Sound: index, Linear: false})
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

// appendMixTableControl reports a conflict when authored events at the same
// position contain no equivalent value. Authored order and values remain intact.
func appendMixTableControl[T VolumeAutomation | PanAutomation](authored []T, output *[]T, value T) bool {
	position := VolumeAutomation(value)
	found, equal := false, false
	for _, existing := range authored {
		candidate := VolumeAutomation(existing)
		if candidate.Track == position.Track && candidate.Bar == position.Bar && candidate.Position == position.Position {
			found = true
			equal = equal || existing == value
		}
	}
	if !found {
		*output = append(*output, value)
	}
	return found && !equal
}

func projectMixTableLevels(song, result *Song, target int, event legacyMixEvent) []legacyMixConflict {
	var conflicts []legacyMixConflict
	for _, control := range []struct {
		name string
		item *MixTableItem
	}{{"volume", event.change.Volume}, {"balance", event.change.Balance}} {
		item := control.item
		if item == nil || item.Value < 0 || item.Value > 16 || (target != event.location.Track && !item.AllTracks) {
			continue
		}
		value := VolumeAutomation{Track: target, Bar: event.location.Measure, Position: event.position, Value: float64(item.Value) / 16, Linear: false}
		var conflict bool
		if control.name == "volume" {
			value.Value = legacyVolumeGain[item.Value]
			conflict = appendMixTableControl(song.VolumeAutomations, &result.VolumeAutomations, value)
		} else {
			conflict = appendMixTableControl(song.PanAutomations, &result.PanAutomations, PanAutomation(value))
		}
		if conflict {
			conflicts = append(conflicts, legacyMixConflict{control.name, event.location, target})
		}
	}
	return conflicts
}

// legacyVolumeGain is Guitar Pro 8's conversion from legacy 0..16 mix-table
// volume to channel-strip gain. Each value was measured by changing only the
// volume byte in the GP3 regression and saving it with native Guitar Pro.
// See conformance/capabilities/evidence/guitar-pro-repair.json.
var legacyVolumeGain = [...]float64{0, .05, .10, .14, .20, .25, .31, .37, .43, .48, .50, .53, .56, .58, .61, .64, .66}
