// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
)

type gpifPendingGrace struct {
	beat   Beat
	onBeat bool
}

func gpifApplyPendingGrace(target *Beat, pending []gpifPendingGrace, percussion bool) []Beat {
	var orphans []Beat
	for pendingIndex, pendingBeat := range pending {
		orphan := pendingBeat.beat
		orphan.isGrace = true
		orphan.graceOnBeat = pendingBeat.onBeat
		orphan.Notes = nil
		for noteIndex := range pendingBeat.beat.Notes {
			graceNote := pendingBeat.beat.Notes[noteIndex]
			effect := gpifGraceEffect(&graceNote, &pendingBeat.beat.Duration, pendingBeat.onBeat, pendingIndex)
			targetIndex := gpifGraceTarget(target, &graceNote, percussion)
			if targetIndex >= 0 {
				target.Notes[targetIndex].Effect.Graces = append(target.Notes[targetIndex].Effect.Graces, effect)
				continue
			}
			orphan.Notes = append(orphan.Notes, graceNote)
		}
		if len(orphan.Notes) > 0 || len(pendingBeat.beat.Notes) == 0 {
			orphans = append(orphans, orphan)
		}
	}
	return orphans
}

func gpifGraceTarget(target *Beat, grace *Note, percussion bool) int {
	if target == nil {
		return -1
	}
	if percussion {
		for index := range target.Notes {
			if target.Notes[index].Value == grace.Value {
				return index
			}
		}
		return -1
	}
	for index := range target.Notes {
		if target.Notes[index].String == grace.String {
			return index
		}
	}
	return -1
}

func gpifGraceEffect(note *Note, duration *Duration, onBeat bool, sequence int) GraceEffect {
	transition := GraceEffectTransitionNone
	switch {
	case note.Effect.Hammer:
		transition = GraceEffectTransitionHammer
	case len(note.Effect.Slides) > 0:
		transition = GraceEffectTransitionSlide
	}
	var exactFret *Fret
	legacyFret := int8(0)
	if fret, err := NewFret(int64(note.Value)); err == nil {
		exactFret = &fret
		if fret <= math.MaxInt8 {
			legacyFret = int8(fret)
		}
	}
	return GraceEffect{
		Duration:                  uint8(min(uint16(math.MaxUint8), duration.Value)),
		Fret:                      legacyFret,
		ExactFret:                 exactFret,
		PercussionArticulation:    note.PercussionArticulation,
		HasPercussionArticulation: note.HasPercussionArticulation,
		IsDead:                    note.Kind == NoteTypeDead,
		IsOnBeat:                  onBeat,
		Sequence:                  uint8(min(sequence, math.MaxUint8)),
		Transition:                transition,
		Velocity:                  note.Velocity,
	}
}

func gpifVersion(value string) Version {
	version := Version{Data: value}
	parts := strings.Split(value, ".")
	for index := 0; index < len(parts) && index < len(version.Number); index++ {
		number, err := strconv.ParseUint(parts[index], 10, 8)
		if err == nil {
			version.Number[index] = byte(number)
		}
	}
	return version
}

func gpifReadStaffStrings(staff gpifStaff) []GuitarString {
	for _, property := range staff.Properties {
		if property.Name != "Tuning" || property.Pitches == "" {
			continue
		}
		values := splitIDs(property.Pitches)
		strings := make([]GuitarString, 0, len(values))
		for sourceIndex := len(values) - 1; sourceIndex >= 0; sourceIndex-- {
			value, err := strconv.ParseInt(values[sourceIndex], 10, 8)
			if err != nil {
				continue
			}
			strings = append(strings, GuitarString{
				Number: int8(len(strings) + 1),
				Value:  int8(value),
			})
		}
		return strings
	}
	return nil
}

func gpifReadTuningName(staff gpifStaff) (string, bool) {
	for _, property := range staff.Properties {
		if property.Name != "Tuning" {
			continue
		}
		if property.Label == nil {
			return "", false
		}
		return *property.Label, true
	}
	return "", false
}

type gpifChordScope struct {
	track  map[string]Chord
	staves []map[string]Chord
}

func (s gpifChordScope) resolve(staffIndex int, id string) (Chord, bool) {
	if staffIndex >= 0 && staffIndex < len(s.staves) {
		if chord, ok := s.staves[staffIndex][id]; ok {
			return cloneChordOccurrence(chord), true
		}
	}
	chord, ok := s.track[id]
	return cloneChordOccurrence(chord), ok
}

func cloneChordOccurrence(source Chord) Chord {
	clone := source
	clone.FirstFret = cloneSemanticPointer(source.FirstFret)
	clone.Ninth = cloneSemanticPointer(source.Ninth)
	clone.Root = cloneSemanticPointer(source.Root)
	clone.Fifth = cloneSemanticPointer(source.Fifth)
	clone.Extension = cloneSemanticPointer(source.Extension)
	clone.Bass = cloneSemanticPointer(source.Bass)
	clone.Tonality = cloneSemanticPointer(source.Tonality)
	clone.Add = cloneSemanticPointer(source.Add)
	clone.Sharp = cloneSemanticPointer(source.Sharp)
	clone.NewFormat = cloneSemanticPointer(source.NewFormat)
	clone.Kind = cloneSemanticPointer(source.Kind)
	clone.Eleventh = cloneSemanticPointer(source.Eleventh)
	clone.Show = cloneSemanticPointer(source.Show)
	clone.Strings = slices.Clone(source.Strings)
	clone.Barres = slices.Clone(source.Barres)
	clone.Omissions = slices.Clone(source.Omissions)
	clone.Fingerings = slices.Clone(source.Fingerings)
	return clone
}

func cloneSemanticPointer[T any](source *T) *T {
	if source == nil {
		return nil
	}
	clone := *source
	return &clone
}

func gpifReadChordMap(track gpifTrack) (gpifChordScope, error) {
	scope := gpifChordScope{track: make(map[string]Chord), staves: make([]map[string]Chord, len(track.Staves.Staff))}
	if err := gpifReadChordProperties(track.Properties, scope.track); err != nil {
		return gpifChordScope{}, err
	}
	for staffIndex, staff := range track.Staves.Staff {
		scope.staves[staffIndex] = make(map[string]Chord)
		if err := gpifReadChordProperties(staff.Properties, scope.staves[staffIndex]); err != nil {
			return gpifChordScope{}, fmt.Errorf("staff %d: %w", staffIndex, err)
		}
	}
	return scope, nil
}

func gpifReadChordProperties(properties []gpifStaffProperty, chords map[string]Chord) error {
	for _, property := range properties {
		if (property.Name != "DiagramCollection" && property.Name != "ChordCollection") || property.Items == nil {
			continue
		}
		for _, item := range property.Items.Items {
			if item.ID == "" {
				continue
			}
			chord := Chord{Name: item.Name}
			if item.Diagram != nil {
				if item.Diagram.StringCount < 0 || item.Diagram.StringCount > math.MaxUint8 {
					return fmt.Errorf("diagram %q string count %d is outside 0..255", item.ID, item.Diagram.StringCount)
				}
				firstFret := uint8(min(math.MaxUint8, max(0, item.Diagram.BaseFret+1)))
				chord.FirstFret = &firstFret
				chord.Length = uint8(item.Diagram.StringCount)
				chord.Strings = make([]int8, item.Diagram.StringCount)
				for index := range chord.Strings {
					chord.Strings[index] = -1
				}
				for _, fret := range item.Diagram.Frets {
					index := item.Diagram.StringCount - fret.String - 1
					if index >= 0 && index < len(chord.Strings) {
						chord.Strings[index] = int8(min(math.MaxInt8, max(0, item.Diagram.BaseFret+fret.Fret)))
					}
				}
			}
			chords[item.ID] = chord
		}
	}
	return nil
}
