// SPDX-License-Identifier: MIT
package goguitarpro

// NoteOrnament identifies the authored turn or mordent on a note.
type NoteOrnament uint8

const (
	// NoteOrnamentNone means no authored ornament.
	NoteOrnamentNone NoteOrnament = iota
	// NoteOrnamentInvertedTurn requests an inverted turn.
	NoteOrnamentInvertedTurn
	// NoteOrnamentTurn requests a turn.
	NoteOrnamentTurn
	// NoteOrnamentUpperMordent requests an upper mordent.
	NoteOrnamentUpperMordent
	// NoteOrnamentLowerMordent requests a lower mordent.
	NoteOrnamentLowerMordent
)

func gpifNoteOrnament(n gpifNote) NoteOrnament {
	switch n.Ornament {
	case "InvertedTurn":
		return NoteOrnamentInvertedTurn
	case "Turn":
		return NoteOrnamentTurn
	case "UpperMordent":
		return NoteOrnamentUpperMordent
	case "LowerMordent":
		return NoteOrnamentLowerMordent
	default:
		return NoteOrnamentNone
	}
}

func gp8NoteOrnament(value NoteOrnament) string {
	switch value {
	case NoteOrnamentInvertedTurn:
		return "InvertedTurn"
	case NoteOrnamentTurn:
		return "Turn"
	case NoteOrnamentUpperMordent:
		return "UpperMordent"
	case NoteOrnamentLowerMordent:
		return "LowerMordent"
	default:
		return ""
	}
}
