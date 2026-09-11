// SPDX-License-Identifier: MIT

package goguitarpro

import "fmt"

var (
	gpifPitchInvalidSource   = diagnosticSource("GPIF.Note.Pitch.Invalid", "note-and-beat-semantics", ParseDiagnosticInvalidData)
	gpifPitchContextSource   = diagnosticSource("GPIF.Note.Pitch.Context", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature)
	gpifPitchAuthoritySource = diagnosticSource("GPIF.Note.Pitch.Authority", "note-and-beat-semantics", ParseDiagnosticLossyProjection)
)

func gpifPitchMIDI(pitch *gpifPitch) (int64, bool) {
	if pitch == nil || pitch.Octave < -1 || pitch.Octave > 11 {
		return 0, false
	}
	steps := map[string]int64{"C": 0, "D": 2, "E": 4, "F": 5, "G": 7, "A": 9, "B": 11}
	step, valid := steps[pitch.Step]
	if !valid {
		return 0, false
	}
	mode, valid := accidentalMode(pitch.Accidental)
	if !valid {
		return 0, false
	}
	return int64(pitch.Octave)*12 + step + accidentalSemitones(mode), true
}

func gpifAuditPitchPayload(context *parseContext, noteID, path string, property gpifProperty) {
	if _, valid := gpifPitchMIDI(property.Pitch); !valid {
		context.add(gpifPitchInvalidSource, ParseDiagnostic{SourcePath: path + "/Pitch", ObjectID: noteID, Location: ParseLocation{NoteID: noteID}, Reason: "pitch requires a step C through B, a supported accidental and an octave within -1..11"})
	}
}

func gpifAuditNoteSpelling(context *parseContext, source *gpifNote, note *Note, staff *Staff, measure *Measure, beat *Beat) {
	if staff.PercussionTrack {
		return
	} // GPIF percussion pitch records are placeholders.
	path := gpifObjectPath("Notes/Note", source.ID) + "/Properties"
	for _, property := range source.Properties.Properties {
		if property.Name == "Fret" && property.Fret != nil && *property.Fret < 0 {
			if mode, _ := gpifAuthoredAccidental(source.Properties.Properties); mode != NoteAccidentalDefault {
				context.add(gpifPitchContextSource, ParseDiagnostic{SourcePath: path, ObjectID: source.ID, Location: ParseLocation{NoteID: source.ID}, Reason: "negative derived fret uses an absolute-MIDI fallback whose retained string identity cannot preserve authored spelling"})
			}
			return
		}
	}
	if reason := accidentalContextLimit(staff, note); reason != "" && note.AccidentalMode != NoteAccidentalDefault {
		context.add(gpifPitchContextSource, ParseDiagnostic{SourcePath: path, ObjectID: source.ID, Location: ParseLocation{NoteID: source.ID}, Reason: reason})
		return
	}
	var concertPitch *gpifPitch
	for _, property := range source.Properties.Properties {
		if property.Name == "ConcertPitch" {
			if value, valid := gpifPitchMIDI(property.Pitch); valid && value == soundingNoteMIDI(staff, note) {
				concertPitch = property.Pitch
			}
		}
	}
	validSource := true
	alternateContext := false
	var concertMode, transposedMode NoteAccidentalMode
	var concert, transposed bool
	for _, property := range source.Properties.Properties {
		expected := soundingNoteMIDI(staff, note)
		switch property.Name {
		case "ConcertPitch":
			concert = true
		case "TransposedPitch":
			transposed = true
			expected = writtenNoteMIDI(staff, measure, beat, note)
		default:
			continue
		}
		actual, valid := gpifPitchMIDI(property.Pitch)
		if !valid {
			validSource = false
			continue
		}
		mode, _ := accidentalMode(property.Pitch.Accidental)
		if property.Name == "ConcertPitch" {
			concertMode = mode
		} else {
			transposedMode = mode
		}
		if actual != expected && property.Name == "TransposedPitch" && concertPitch != nil {
			// Original GPIF may spell a nominal-tuning pitch or omit octave
			// notation adjustments. Its consistent concert record establishes
			// the separate numeric pitch; retain the selected authored payload.
			alternateContext = true
			continue
		}
		if actual != expected {
			validSource = false
			context.add(gpifPitchInvalidSource, ParseDiagnostic{SourcePath: path + fmt.Sprintf("/Property[@name=%q]/Pitch", property.Name), ObjectID: source.ID, Location: ParseLocation{NoteID: source.ID}, Reason: fmt.Sprintf("authored pitch %d contradicts numeric %s pitch %d", actual, property.Name, expected)})
		}
	}
	if validSource && note.AccidentalMode != NoteAccidentalDefault {
		_, selected := gpifAuthoredAccidental(source.Properties.Properties)
		if selected != nil && selected.Pitch != nil {
			if !alternateContext {
				concertPitch = nil
			}
			note.rememberAccidentalSource(*selected, concertPitch, staff, measure, beat)
		}
	}
	if concert && transposed && concertMode != transposedMode {
		context.add(gpifPitchAuthoritySource, ParseDiagnostic{SourcePath: path, ObjectID: source.ID, Location: ParseLocation{NoteID: source.ID}, Reason: "TransposedPitch controls the public accidental mode; a distinct ConcertPitch spelling is not independently editable"})
	}
}
