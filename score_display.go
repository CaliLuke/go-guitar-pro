// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/binary"
	"fmt"
)

// BracketMode selects how the score groups staves with brackets or braces.
type BracketMode uint8

const (
	// BracketNone draws no brackets.
	BracketNone BracketMode = iota
	// BracketStaves groups the staves of a track.
	BracketStaves
	// BracketSimilarInstruments groups tracks with the same MIDI program.
	BracketSimilarInstruments
)

// TrackNameSystemMode selects the authored systems that display track names.
// A false visibility request takes precedence over this mode in the consumer.
type TrackNameSystemMode uint8

const (
	// TrackNamesFirstSystem requests names on the first system.
	TrackNamesFirstSystem TrackNameSystemMode = iota
	// TrackNamesFirstSystemEachPage requests the first system of each page.
	// GP8 retains wire value 1, but the pinned consumer normalizes it to FirstSystem.
	TrackNamesFirstSystemEachPage
	// TrackNamesAllSystems requests names on all systems.
	TrackNamesAllSystems
)

type scoreDisplayBoolean struct {
	key   string
	field **bool
}

func scoreDisplayBooleans(style *ScoreStyle) []scoreDisplayBoolean {
	return []scoreDisplayBoolean{
		{"StandardNotation/hideDynamics", &style.HideDynamics},
		{"Global/useSystemSignSeparator", &style.SystemSeparators},
		{"Global/DisplayTuning", &style.DisplayTuning},
		{"Global/DrawChords", &style.ChordDiagramsOnTop},
		{"System/drawChordInScore", &style.ChordDiagramsInScore},
		{"System/showTrackNameSingle", &style.SingleTrackNamesVisible},
		{"System/showTrackNameMulti", &style.MultiTrackNamesVisible},
		{"System/shortTrackNameOnFirstSystem", &style.FirstSystemShortNames},
		{"System/shortTrackNameOnOtherSystems", &style.OtherSystemsShortNames},
		{"System/horizontalTrackNameOnFirstSystem", &style.FirstSystemHorizontalNames},
		{"System/horizontalTrackNameOnOtherSystems", &style.OtherSystemsHorizontalNames},
	}
}
func scoreDisplayOwnedKey(key string) bool {
	var style ScoreStyle
	for _, binding := range scoreDisplayBooleans(&style) {
		if binding.key == key {
			return true
		}
	}
	return key == "System/bracketExtendMode" || key == "System/trackNameModeSingle" || key == "System/trackNameModeMulti"
}
func applyScoreDisplayRecord(style *ScoreStyle, record binaryStyleRecord) error {
	for _, binding := range scoreDisplayBooleans(style) {
		if binding.key == record.key {
			if record.kind != 0 {
				return fmt.Errorf("reading BinaryStylesheet: key %q requires boolean type 0, got %d", record.key, record.kind)
			}
			value := record.value[0] == 1
			*binding.field = &value
			return nil
		}
	}
	if !scoreDisplayOwnedKey(record.key) {
		return nil
	}
	if record.kind != 1 {
		return fmt.Errorf("reading BinaryStylesheet: key %q requires integer type 1, got %d", record.key, record.kind)
	}
	value := int32(binary.BigEndian.Uint32(record.value))
	if value < 0 || value > 2 {
		return fmt.Errorf("reading BinaryStylesheet: key %q value %d is outside 0..2", record.key, value)
	}
	switch record.key {
	case "System/bracketExtendMode":
		v := BracketMode(value)
		style.Brackets = &v
	case "System/trackNameModeSingle":
		v := TrackNameSystemMode(value)
		style.SingleTrackNameMode = &v
	case "System/trackNameModeMulti":
		v := TrackNameSystemMode(value)
		style.MultiTrackNameMode = &v
	}
	return nil
}
func scoreDisplayRecords(style *ScoreStyle) []binaryStyleRecord {
	if style == nil {
		return nil
	}
	var records []binaryStyleRecord
	for _, binding := range scoreDisplayBooleans(style) {
		if *binding.field != nil {
			value := byte(0)
			if **binding.field {
				value = 1
			}
			records = append(records, binaryStyleRecord{key: binding.key, kind: 0, value: []byte{value}})
		}
	}
	if style.Brackets != nil {
		records = append(records, scoreDisplayInteger("System/bracketExtendMode", uint32(*style.Brackets)))
	}
	if style.SingleTrackNameMode != nil {
		records = append(records, scoreDisplayInteger("System/trackNameModeSingle", uint32(*style.SingleTrackNameMode)))
	}
	if style.MultiTrackNameMode != nil {
		records = append(records, scoreDisplayInteger("System/trackNameModeMulti", uint32(*style.MultiTrackNameMode)))
	}
	return records
}
func scoreDisplayInteger(key string, value uint32) binaryStyleRecord {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, value)
	return binaryStyleRecord{key: key, kind: 1, value: data}
}
func validateScoreDisplay(style *ScoreStyle, diagnostics *[]ScoreDiagnostic) {
	if style == nil {
		return
	}
	if style.Brackets != nil && *style.Brackets > BracketSimilarInstruments {
		*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.style.bracket-mode", Kind: ScoreDiagnosticValue, Reason: fmt.Sprintf("bracket mode %d is undefined", *style.Brackets)})
	}
	for _, mode := range []*TrackNameSystemMode{style.SingleTrackNameMode, style.MultiTrackNameMode} {
		if mode != nil && *mode > TrackNamesAllSystems {
			*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.style.track-name-mode", Kind: ScoreDiagnosticValue, Reason: fmt.Sprintf("track-name mode %d is undefined", *mode)})
		}
	}
}
func (builder *gp8Builder) reportScoreDisplay() {
	style := builder.song.Style
	if style == nil {
		return
	}
	for _, view := range []struct {
		name string
		mode *TrackNameSystemMode
	}{{"single-track", style.SingleTrackNameMode}, {"multi-track", style.MultiTrackNameMode}} {
		if view.mode != nil && *view.mode == TrackNamesFirstSystemEachPage {
			builder.addReport("gp8.normalize.track-name-page-policy", "score-core", ExportDispositionNormalized, ScoreLocation{}, "the pinned consumer has no separate policy for names on the first system of each page in the "+view.name+" view; wire value 1 remains exact")
		}
	}
}
