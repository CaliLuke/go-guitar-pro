// SPDX-License-Identifier: MIT

package goguitarpro

import "fmt"

// automaticPitch resolves an unspecified spelling against the key signature.
// The resulting records are required by Guitar Pro even for automatic notes.
func automaticPitch(midi int64, key KeySignature) (gpifPitch, bool) {
	steps := [...]string{"C", "D", "E", "F", "G", "A", "B"}
	naturals := [...]int64{0, 2, 4, 5, 7, 9, 11}
	order := "FCGDAEB"
	alteration := int64(1)
	count := int(key.Key)
	if count < 0 {
		order, alteration, count = "BEADGCF", -1, -count
	}
	for i, step := range steps {
		delta := int64(0)
		for j := 0; j < count && j < len(order); j++ {
			if string(order[j]) == step {
				delta = alteration
			}
		}
		if (midi-naturals[i]-delta)%12 != 0 {
			continue
		}
		octave := (midi - naturals[i] - delta) / 12
		if octave < -1 || octave > 11 {
			continue
		}
		token := ""
		if delta < 0 {
			token = "b"
		} else if delta > 0 {
			token = "#"
		}
		return gpifPitch{Step: step, Accidental: &token, Octave: int(octave)}, true
	}
	preferred := NoteAccidentalSharp
	if key.Key < 0 {
		preferred = NoteAccidentalFlat
	}
	for _, mode := range []NoteAccidentalMode{NoteAccidentalNatural, preferred} {
		if pitch, ok := spelledPitch(midi, mode); ok {
			return pitch, true
		}
	}
	return gpifPitch{}, false
}

func (builder *gp8Builder) nativeNotePitch(note *Note) ([]gpifProperty, int, error) {
	staff, measure, beat := builder.spellingStaff, builder.spellingMeasure, builder.spellingBeat
	// Sounding transposition has an existing explicit omission. Derive metadata
	// from the actual exported tuning, capo and fret, never from an omitted offset.
	concertMIDI := soundingNoteMIDI(staff, note) + int64(staff.TranspositionPitch)
	writtenMIDI := concertMIDI - int64(staff.DisplayTranspositionPitch) - octavePitch(measure.ClefOctave) - octavePitch(beat.Octave)
	concert, concertOK := automaticPitch(concertMIDI, measure.KeySignature)
	written, writtenOK := automaticPitch(writtenMIDI, measure.KeySignature)
	if note.AccidentalMode != NoteAccidentalDefault && accidentalContextLimit(staff, note) == "" {
		if source := note.sourceAccidental(staff, measure, beat); source != nil {
			if source.propertyName == "ConcertPitch" {
				concert, concertOK = *source.pitch.gpif(), true
				written, writtenOK = spelledPitch(writtenMIDI, note.AccidentalMode)
			} else {
				written, writtenOK = *source.pitch.gpif(), true
				if source.concert != nil {
					concert, concertOK = *source.concert.gpif(), true
				}
			}
		} else {
			written, writtenOK = spelledPitch(writtenMIDI, note.AccidentalMode)
		}
		// Reuse the selected spelling where both coordinates can express it.
		if source := note.sourceAccidental(staff, measure, beat); source == nil || (source.concert == nil && source.propertyName != "ConcertPitch") {
			if pitch, ok := spelledPitch(concertMIDI, note.AccidentalMode); ok {
				concert, concertOK = pitch, true
			}
		}
	}
	if !concertOK || !writtenOK {
		return nil, 0, fmt.Errorf("native pitch records cannot represent concert pitch %d and written pitch %d within GPIF octave bounds", concertMIDI, writtenMIDI)
	}
	return []gpifProperty{{Name: "ConcertPitch", Pitch: &concert}, {Name: "TransposedPitch", Pitch: &written}}, int(concertMIDI), nil
}
