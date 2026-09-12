// SPDX-License-Identifier: MIT

package goguitarpro

import "fmt"

func validatePartialCapo(staff *Staff, location ScoreLocation, diagnostics *[]ScoreDiagnostic) {
	partial := staff.PartialCapo
	if partial == nil {
		return
	}
	if partial.Offset < 0 {
		*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.staff.partial-capo.offset", Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("partial capo offset %d is negative", partial.Offset)})
	}
	if len(partial.Strings) != len(staff.Strings) {
		*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: "score.staff.partial-capo.strings", Kind: ScoreDiagnosticStructural, Location: location, Reason: fmt.Sprintf("partial capo has %d flags for %d strings", len(partial.Strings), len(staff.Strings))})
	}
}

func validatePartialCapoPitches(staff *Staff, note *Note, location ScoreLocation, diagnostics *[]ScoreDiagnostic) {
	check := func(candidate *Note, code, label string) {
		if candidate.String <= 0 || int(candidate.String) > len(staff.Strings) || partialCapoOffset(staff, candidate) == 0 {
			return
		}
		midi := soundingNoteMIDI(staff, candidate)
		if midi < 0 || midi > 127 {
			*diagnostics = append(*diagnostics, ScoreDiagnostic{Code: code, Kind: ScoreDiagnosticValue, Location: location, Reason: fmt.Sprintf("%s sounding MIDI value %d after partial capo is outside 0..127", label, midi)})
		}
	}
	check(note, "score.note.partial-capo-midi", "note")
	for i, grace := range note.Effect.Graces {
		open := grace.Fret == 0
		if grace.ExactFret != nil {
			open = *grace.ExactFret == 0
		}
		if open {
			check(&Note{String: note.String}, "score.grace.partial-capo-midi", fmt.Sprintf("grace %d", i))
		}
	}
}
