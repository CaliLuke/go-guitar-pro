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
	beat    Beat
	onBeat  bool
	noteIDs []string
}

func gpifApplyPendingGrace(target *Beat, pending []gpifPendingGrace, percussion bool, context *parseContext) []Beat {
	if gpifRetainTimedGraceGroup(target, pending, percussion) {
		beats := make([]Beat, len(pending))
		for i, grace := range pending {
			beats[i] = grace.beat
			beats[i].isGrace = true
			beats[i].graceOnBeat = grace.onBeat
		}
		return beats
	}
	var orphans []Beat
	for pendingIndex, pendingBeat := range pending {
		orphan := pendingBeat.beat
		orphan.isGrace = true
		orphan.graceOnBeat = pendingBeat.onBeat
		orphan.Notes = nil
		for noteIndex := range pendingBeat.beat.Notes {
			graceNote := pendingBeat.beat.Notes[noteIndex]
			effect := gpifGraceEffect(&graceNote, &pendingBeat.beat.Duration, pendingBeat.onBeat, pendingIndex)
			effect.Timer = cloneBeatTimer(pendingBeat.beat.Timer)
			targetIndex := gpifGraceTarget(target, &graceNote, percussion)
			if targetIndex >= 0 {
				noteID := ""
				if noteIndex < len(pendingBeat.noteIDs) {
					noteID = pendingBeat.noteIDs[noteIndex]
				}
				if graceNote.AccidentalMode != NoteAccidentalDefault {
					context.add(gpifPitchContextSource, ParseDiagnostic{SourcePath: gpifObjectPath("Notes/Note", noteID) + "/Properties", ObjectID: noteID, Location: ParseLocation{NoteID: noteID}, Reason: "ordered GraceEffect has no independent authored accidental mode"})
				}
				if graceNote.Effect.HammerDestination {
					context.add(diagnosticSource("GPIF.Note.HammerDestination.Grace", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), ParseDiagnostic{SourcePath: gpifObjectPath("Notes/Note", noteID) + "/Properties/Property[@name=\"HopoDestination\"]", ObjectID: noteID, Location: ParseLocation{NoteID: noteID}, Reason: "ordered GraceEffect has no independent authored hammer/pull destination marker"})
				}
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

// Keep a timed source group intact when attachment would split a chord or
// move an orphan ahead of earlier attached graces, or exceed the sequence range.
// A timer belongs to its source beat.
func gpifRetainTimedGraceGroup(target *Beat, pending []gpifPendingGrace, percussion bool) bool {
	hasTimer, needsStandalone := false, len(pending) > math.MaxUint8+1
	for _, grace := range pending {
		hasTimer = hasTimer || grace.beat.Timer != nil
		needsStandalone = needsStandalone || len(grace.beat.Notes) == 0
		for i := range grace.beat.Notes {
			if gpifGraceTarget(target, &grace.beat.Notes[i], percussion) < 0 {
				needsStandalone = true
			}
		}
	}
	return hasTimer && needsStandalone
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
	clone.ShowName = cloneSemanticPointer(source.ShowName)
	clone.ShowDiagram = cloneSemanticPointer(source.ShowDiagram)
	clone.ShowFingering = cloneSemanticPointer(source.ShowFingering)
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
			gpifReadChordDisplay(&chord, item.Diagram)
			if item.Diagram != nil {
				if item.Diagram.StringCount < 0 || item.Diagram.StringCount > math.MaxUint8 {
					return fmt.Errorf("diagram %q string count %d is outside 0..255", item.ID, item.Diagram.StringCount)
				}
				if item.Diagram.BaseFret < 0 || item.Diagram.BaseFret >= math.MaxUint8 {
					return fmt.Errorf("diagram %q base fret %d cannot map to a one-based chord fret", item.ID, item.Diagram.BaseFret)
				}
				firstFret := uint8(item.Diagram.BaseFret + 1)
				chord.FirstFret = &firstFret
				chord.Length = uint8(item.Diagram.StringCount)
				chord.Strings = make([]int8, item.Diagram.StringCount)
				for index := range chord.Strings {
					chord.Strings[index] = -1
				}
				seenStrings := make(map[int]struct{}, len(item.Diagram.Frets))
				for _, fret := range item.Diagram.Frets {
					if fret.String < 0 || fret.String >= item.Diagram.StringCount {
						return fmt.Errorf("diagram %q string %d is outside 0..%d", item.ID, fret.String, item.Diagram.StringCount-1)
					}
					if _, exists := seenStrings[fret.String]; exists {
						return fmt.Errorf("diagram %q repeats string %d", item.ID, fret.String)
					}
					seenStrings[fret.String] = struct{}{}
					if fret.Fret > math.MaxInt-item.Diagram.BaseFret {
						return fmt.Errorf("diagram %q fret %d plus base fret %d overflows int", item.ID, fret.Fret, item.Diagram.BaseFret)
					}
					absoluteFret := item.Diagram.BaseFret + fret.Fret
					if absoluteFret < 0 || absoluteFret > math.MaxInt8 {
						return fmt.Errorf("diagram %q fret %d plus base fret %d is outside 0..%d", item.ID, fret.Fret, item.Diagram.BaseFret, math.MaxInt8)
					}
					index := item.Diagram.StringCount - fret.String - 1
					chord.Strings[index] = int8(absoluteFret)
				}
				if item.Diagram.Fingering != nil {
					fingerings, barres, err := gpifChordFingerings(item.ID, item.Diagram, chord.Strings)
					if err != nil {
						return err
					}
					chord.Fingerings = fingerings
					chord.Barres = barres
				}
			}
			chords[item.ID] = chord
		}
	}
	return nil
}

type gpifChordBarreKey struct {
	finger Fingering
	fret   int8
}

func gpifChordFingerings(id string, diagram *gpifDiagram, strings []int8) ([]Fingering, []Barre, error) {
	fingerings := make([]Fingering, len(strings))
	for index := range fingerings {
		fingerings[index] = FingeringUnknown
	}
	groups := make(map[gpifChordBarreKey][]int)
	order := make([]gpifChordBarreKey, 0)
	for positionIndex, position := range diagram.Fingering.Positions {
		if position.String == nil {
			return nil, nil, fmt.Errorf("diagram %q fingering position %d has no string", id, positionIndex)
		}
		if *position.String < 0 || *position.String >= diagram.StringCount {
			return nil, nil, fmt.Errorf("diagram %q fingering string %d is outside 0..%d", id, *position.String, diagram.StringCount-1)
		}
		publicIndex := diagram.StringCount - *position.String - 1
		if fingerings[publicIndex] != FingeringUnknown {
			return nil, nil, fmt.Errorf("diagram %q repeats fingering for string %d", id, *position.String)
		}
		if position.Fret > math.MaxInt-diagram.BaseFret {
			return nil, nil, fmt.Errorf("diagram %q fingering fret %d plus base fret %d overflows int", id, position.Fret, diagram.BaseFret)
		}
		absoluteFret := diagram.BaseFret + position.Fret
		if position.Finger == "None" && position.Fret == math.MaxUint32 {
			absoluteFret = -1
		}
		if absoluteFret < -1 || absoluteFret > math.MaxInt8 {
			return nil, nil, fmt.Errorf("diagram %q fingering fret %d plus base fret %d is outside 0..%d", id, position.Fret, diagram.BaseFret, math.MaxInt8)
		}
		if strings[publicIndex] != int8(absoluteFret) {
			return nil, nil, fmt.Errorf("diagram %q fingering string %d fret %d does not match chord fret %d", id, *position.String, absoluteFret, strings[publicIndex])
		}
		finger, err := gpifChordFinger(position.Finger)
		if err != nil {
			return nil, nil, fmt.Errorf("diagram %q fingering position %d: %w", id, positionIndex, err)
		}
		fingerings[publicIndex] = finger
		if finger == FingeringOpen {
			continue
		}
		key := gpifChordBarreKey{finger: finger, fret: int8(absoluteFret)}
		if _, exists := groups[key]; !exists {
			order = append(order, key)
		}
		groups[key] = append(groups[key], publicIndex+1)
	}
	barres := make([]Barre, 0)
	for _, key := range order {
		positions := groups[key]
		if len(positions) < 2 {
			continue
		}
		start, end := positions[0], positions[0]
		for _, position := range positions[1:] {
			start = min(start, position)
			end = max(end, position)
		}
		barres = append(barres, Barre{Fret: key.fret, Start: int8(start), End: int8(end)})
	}
	return fingerings, barres, nil
}

func gpifChordFinger(token string) (Fingering, error) {
	switch token {
	case "None":
		return FingeringOpen, nil
	case "Thumb":
		return FingeringThumb, nil
	case "Index":
		return FingeringIndex, nil
	case "Middle":
		return FingeringMiddle, nil
	case "Ring", "Rank":
		return FingeringAnnular, nil
	case "Pinky":
		return FingeringLittle, nil
	default:
		return FingeringUnknown, fmt.Errorf("finger %q is not defined", token)
	}
}

func gpifReadChordDisplay(chord *Chord, diagram *gpifDiagram) {
	showName, showDiagram, showFingering := true, diagram != nil, diagram != nil
	if diagram != nil {
		values := map[string]*bool{"ShowName": &showName, "ShowDiagram": &showDiagram, "ShowFingering": &showFingering}
		for _, property := range diagram.Properties {
			if value, known := values[property.Name]; known {
				*value = property.Value == "true"
			}
		}
	}
	chord.ShowName, chord.ShowDiagram, chord.ShowFingering = &showName, &showDiagram, &showFingering
}
