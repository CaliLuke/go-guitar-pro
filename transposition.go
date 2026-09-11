// SPDX-License-Identifier: MIT

package goguitarpro

import "fmt"

// Guitar Pro chooses enharmonic key spellings from this table when a display
// transposition is active. Columns run from seven flats through seven sharps.
var transposedKeySignatures = [12][15]int8{
	{-7, -6, -5, -4, -3, -2, -1, 0, 1, 2, 3, 4, 5, 6, 7},
	{-2, -1, 0, 1, 2, 3, 4, 5, 6, 7, -4, -3, -2, -1, 0},
	{3, 4, -7, -6, -5, -4, -3, -2, -1, 0, 1, 2, 3, 4, 5},
	{-4, -3, -2, -1, 0, 1, 2, 3, 4, 5, 6, 7, -4, -3, -2},
	{1, 2, 3, 4, -7, -6, -5, -4, -3, -2, -1, 0, 1, 2, 3},
	{-6, -5, -4, -3, -2, -1, 0, 1, 2, 3, 4, 5, 6, 7, -4},
	{-1, 0, 1, 2, 3, 4, -7, 6, 7, -4, -3, -2, -1, 0, 1},
	{4, -7, -6, -5, -4, -3, -2, -1, 0, 1, 2, 3, 4, 5, 6},
	{-3, -2, -1, 0, 1, 2, 3, 4, 5, 6, 7, -4, -3, -2, -1},
	{2, 3, 4, -7, -6, -5, -4, -3, -2, -1, 0, 1, 2, 3, 4},
	{-5, -4, -3, -2, -1, 0, 1, 2, 3, 4, 5, 6, 7, -4, -3},
	{0, 1, 2, 3, 4, -7, -6, 6, -4, -3, -2, -1, 0, 1, 2},
}

func transposeKeySignature(key KeySignature, pitch int32) KeySignature {
	if key.Key < -7 || key.Key > 7 {
		return key
	}
	semitone := pitch % 12
	if semitone < 0 {
		semitone += 12
	}
	key.Key = transposedKeySignatures[semitone][int(key.Key)+7]
	return key
}

func soundingNoteMIDI(staff *Staff, note *Note) int64 {
	midi := int64(note.Value)
	if note.String > 0 && int(note.String) <= len(staff.Strings) {
		midi += int64(staff.Strings[int(note.String)-1].Value) + int64(staff.CapoFret)
	}
	return midi - int64(staff.TranspositionPitch)
}

func gp8Transpose(pitch int32) gpifTranspose {
	value := int64(pitch)
	octave := value / 12
	if value < 0 && value%12 != 0 {
		octave--
	}
	return gpifTranspose{Chromatic: value - octave*12, Octave: octave}
}

func gpifNominalKeyPitch(name string) (int32, bool) {
	if name == "Neutral" {
		return 0, true
	}
	names := [...]string{"C", "Db", "D", "Eb", "E", "F", "Gb", "G", "Ab", "A", "Bb", "B"}
	for index, candidate := range names {
		if name == candidate {
			return int32(index), true
		}
	}
	return 0, false
}

func gpifAuditTrackTransposition(context *parseContext, track gpifTrack, path string) {
	if context == nil || track.PartSounding == nil {
		return
	}
	location := ParseLocation{TrackID: track.ID}
	displayPitch := track.PartSounding.TranspositionPitch
	if track.Transpose != nil {
		transpose, err := checkedGPIFTransposition(track.Transpose.Chromatic, track.Transpose.Octave)
		if err == nil && int64(transpose) != displayPitch {
			context.add(diagnosticSource("GPIF.Track.Transposition.ConflictingValues", "transposition", ParseDiagnosticLossyProjection), ParseDiagnostic{
				SourcePath: path + "/PartSounding/TranspositionPitch", ObjectID: track.ID, Location: location,
				Reason: fmt.Sprintf("PartSounding pitch %d conflicts with authoritative Transpose pitch %d", displayPitch, transpose),
			})
		}
	}
	if track.PartSounding.NominalKey == "" {
		return
	}
	nominalPitch, ok := gpifNominalKeyPitch(track.PartSounding.NominalKey)
	if !ok {
		context.add(diagnosticSource("GPIF.Track.PartSounding.NominalKey.InvalidValue", "transposition", ParseDiagnosticUnsupportedFeature), ParseDiagnostic{
			SourcePath: path + "/PartSounding/NominalKey", ObjectID: track.ID, Location: location,
			Reason: fmt.Sprintf("nominal key %q is not recognized", track.PartSounding.NominalKey),
		})
		return
	}
	if track.Transpose == nil {
		return
	}
	transpose, err := checkedGPIFTransposition(track.Transpose.Chromatic, track.Transpose.Octave)
	if err != nil {
		return
	}
	semitone := int64(transpose % 12)
	if semitone < 0 {
		semitone += 12
	}
	if int64(nominalPitch) != semitone {
		context.add(diagnosticSource("GPIF.Track.PartSounding.NominalKey.Conflict", "transposition", ParseDiagnosticLossyProjection), ParseDiagnostic{
			SourcePath: path + "/PartSounding/NominalKey", ObjectID: track.ID, Location: location,
			Reason: fmt.Sprintf("PartSounding nominal key %q conflicts with authoritative Transpose pitch %d", track.PartSounding.NominalKey, transpose),
		})
	}
}
