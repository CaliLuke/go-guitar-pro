// SPDX-License-Identifier: MIT

package goguitarpro

import "slices"

// NoteAccidentalMode selects the authored enharmonic spelling of a note.
// It does not change the numeric sounding pitch.
type NoteAccidentalMode uint8

const (
	// NoteAccidentalDefault leaves spelling to the notation consumer.
	NoteAccidentalDefault NoteAccidentalMode = iota
	// NoteAccidentalNatural requests a natural note name.
	NoteAccidentalNatural
	// NoteAccidentalSharp requests a note name with one sharp.
	NoteAccidentalSharp
	// NoteAccidentalDoubleSharp requests a note name with two sharps.
	NoteAccidentalDoubleSharp
	// NoteAccidentalFlat requests a note name with one flat.
	NoteAccidentalFlat
	// NoteAccidentalDoubleFlat requests a note name with two flats.
	NoteAccidentalDoubleFlat
)

func accidentalToken(mode NoteAccidentalMode) (string, bool) {
	switch mode {
	case NoteAccidentalDefault, NoteAccidentalNatural:
		return "", true
	case NoteAccidentalSharp:
		return "#", true
	case NoteAccidentalDoubleSharp:
		return "x", true
	case NoteAccidentalFlat:
		return "b", true
	case NoteAccidentalDoubleFlat:
		return "bb", true
	default:
		return "", false
	}
}

func accidentalMode(token *string) (NoteAccidentalMode, bool) {
	if token == nil {
		return NoteAccidentalDefault, true
	}
	switch *token {
	case "":
		return NoteAccidentalNatural, true
	case "#":
		return NoteAccidentalSharp, true
	case "x":
		return NoteAccidentalDoubleSharp, true
	case "b":
		return NoteAccidentalFlat, true
	case "bb":
		return NoteAccidentalDoubleFlat, true
	default:
		return NoteAccidentalDefault, false
	}
}

func accidentalSemitones(mode NoteAccidentalMode) int64 {
	switch mode {
	case NoteAccidentalSharp:
		return 1
	case NoteAccidentalDoubleSharp:
		return 2
	case NoteAccidentalFlat:
		return -1
	case NoteAccidentalDoubleFlat:
		return -2
	default:
		return 0
	}
}

func spelledPitch(midi int64, mode NoteAccidentalMode) (gpifPitch, bool) {
	token, valid := accidentalToken(mode)
	if !valid {
		return gpifPitch{}, false
	}
	natural := midi - accidentalSemitones(mode)
	chroma := (natural%12 + 12) % 12
	steps := map[int64]string{0: "C", 2: "D", 4: "E", 5: "F", 7: "G", 9: "A", 11: "B"}
	step, valid := steps[chroma]
	if !valid {
		return gpifPitch{}, false
	}
	return gpifPitch{Step: step, Accidental: &token, Octave: int((natural - chroma) / 12)}, true
}

func octavePitch(octave Octave) int64 {
	switch octave {
	case OctaveOttava:
		return 12
	case OctaveOttavaBassa:
		return -12
	case OctaveQuindicesima:
		return 24
	case OctaveQuindicesimaBassa:
		return -24
	default:
		return 0
	}
}

func writtenNoteMIDI(staff *Staff, measure *Measure, beat *Beat, note *Note) int64 {
	return soundingNoteMIDI(staff, note) - int64(staff.DisplayTranspositionPitch) - octavePitch(measure.ClefOctave) - octavePitch(beat.Octave)
}

func accidentalContextLimit(staff *Staff, note *Note) string {
	if staff.PercussionTrack {
		return "percussion notation does not retain an authored pitched accidental mode"
	}
	if note.String == 0 {
		return "the GP8 consumer does not retain absolute note pitch from Midi alone; accidental mode requires string and fret context"
	}
	if staff.TranspositionPitch != 0 {
		return "GP8 omits the sounding transposition needed to realize the authored accidental spelling"
	}
	if note.Effect.Harmonic != nil && note.Effect.Harmonic.Kind == HarmonicTypeNatural {
		return "natural-harmonic written pitch spelling is not represented by the note accidental contract"
	}
	return ""
}

func gpifAuthoredAccidental(properties []gpifProperty) (NoteAccidentalMode, *gpifProperty) {
	var selected *gpifProperty
	transposed := false
	for _, p := range properties {
		if p.Name == "TransposedPitch" {
			selected = &p
			transposed = true
		} else if p.Name == "ConcertPitch" && !transposed {
			selected = &p
		}
	}
	if selected == nil || selected.Pitch == nil {
		return NoteAccidentalDefault, selected
	}
	mode, _ := accidentalMode(selected.Pitch.Accidental)
	return mode, selected
}

// This immutable source receipt keeps an authored notation coordinate system
// without adding a second public pitch authority. Any relevant public edit
// invalidates it, so the writer then derives spelling from the edited model.
type noteAccidentalSource struct {
	mode                    NoteAccidentalMode
	propertyName            string
	pitch, concert          *noteAccidentalPitch
	value                   int16
	stringNumber            int8
	tuning                  []GuitarString
	capo, sounding, display int32
	clef, octave            Octave
}

func (note *Note) sourceAccidental(staff *Staff, measure *Measure, beat *Beat) *noteAccidentalSource {
	source := note.accidentalSource
	if source == nil || source.mode != note.AccidentalMode || source.value != note.Value || source.stringNumber != note.String ||
		source.capo != staff.CapoFret || source.sounding != staff.TranspositionPitch || source.display != staff.DisplayTranspositionPitch ||
		source.clef != measure.ClefOctave || source.octave != beat.Octave || !slices.Equal(source.tuning, staff.Strings) {
		return nil
	}
	return source
}

// noteAccidentalPitch is immutable source spelling, not a public pitch authority.
type noteAccidentalPitch struct {
	step       string
	accidental *string
	octave     int
}

func rememberAccidentalPitch(pitch *gpifPitch) *noteAccidentalPitch {
	if pitch == nil {
		return nil
	}
	copied := &noteAccidentalPitch{step: pitch.Step, octave: pitch.Octave}
	if pitch.Accidental != nil {
		value := *pitch.Accidental
		copied.accidental = &value
	}
	return copied
}

func (pitch *noteAccidentalPitch) gpif() *gpifPitch {
	if pitch == nil {
		return nil
	}
	return &gpifPitch{Step: pitch.step, Accidental: pitch.accidental, Octave: pitch.octave}
}

func (note *Note) rememberAccidentalSource(property gpifProperty, concert *gpifPitch, staff *Staff, measure *Measure, beat *Beat) {
	note.accidentalSource = &noteAccidentalSource{mode: note.AccidentalMode, propertyName: property.Name,
		pitch: rememberAccidentalPitch(property.Pitch), concert: rememberAccidentalPitch(concert),
		value: note.Value, stringNumber: note.String, tuning: slices.Clone(staff.Strings), capo: staff.CapoFret, sounding: staff.TranspositionPitch,
		display: staff.DisplayTranspositionPitch, clef: measure.ClefOctave, octave: beat.Octave}
}

func validateNoteAccidental(staff *Staff, measure *Measure, beat *Beat, note *Note, percussion bool, location ScoreLocation, diagnostics *[]ScoreDiagnostic) {
	if _, valid := accidentalToken(note.AccidentalMode); !valid {
		*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.accidental-mode", Kind: ScoreDiagnosticValue, Location: location, Reason: "note accidental mode is not defined"})
		return
	}
	if percussion || note.AccidentalMode == NoteAccidentalDefault || accidentalContextLimit(staff, note) != "" {
		return
	}
	if note.sourceAccidental(staff, measure, beat) != nil {
		return
	}
	if _, valid := spelledPitch(writtenNoteMIDI(staff, measure, beat, note), note.AccidentalMode); !valid {
		*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.note.accidental-pitch", Kind: ScoreDiagnosticValue, Location: location, Reason: "authored accidental contradicts the written numeric pitch"})
	}
}
