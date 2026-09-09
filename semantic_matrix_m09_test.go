// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"math"
	"slices"
	"strconv"
	"testing"
)

func TestSemanticMatrixM09NoteRepresentation(t *testing.T) {
	runSemanticMatrixM09NoteRepresentation(newSemanticMatrixRun(t))
}

func runSemanticMatrixM09NoteRepresentation(run *semanticMatrixRun) {
	t := run.t
	song := semanticValidPitchedGP8Song(t)
	song.Tracks[0].Settings.Notation = true
	song.MeasureHeaders = slices.Clone(song.MeasureHeaders[:1])
	song.Tracks[0].Measures = slices.Clone(song.Tracks[0].Measures[:1])
	quarter := defaultDuration()
	notes := []Note{
		{Value: 0, String: 1, Kind: NoteTypeNormal, TieOrigin: true, Velocity: Forte, DurationPercent: 1},
		{Value: 0, String: 1, Kind: NoteTypeTie, TieOrigin: true, Velocity: Forte, DurationPercent: 1},
		{Value: 0, String: 1, Kind: NoteTypeTie, Velocity: Forte, DurationPercent: 1, Effect: NoteEffect{DeadNote: true}},
		{Value: 0, String: 0, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1},
		{Value: 127, String: 0, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1, SwapAccidentals: true},
		{Value: 5, String: 6, Kind: NoteTypeDead, Velocity: Forte, DurationPercent: 0.5},
	}
	beats := make([]Beat, len(notes))
	for index := range notes {
		beats[index] = Beat{Duration: quarter, Status: BeatStatusNormal, Notes: []Note{notes[index]}}
	}
	song.Tracks[0].Measures[0].Voices = []Voice{{Beats: beats}}
	song.Tracks[0].Staves[0].Measures = song.Tracks[0].Measures
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	for index := range notes {
		note := song.Tracks[0].Measures[0].Voices[0].Beats[index].Notes[0]
		run.Field("Note.Value", note.Value, notes[index].Value)
		run.Field("Note.String", note.String, notes[index].String)
		run.Field("Note.Kind", note.Kind, notes[index].Kind)
		run.Field("Note.TieOrigin", note.TieOrigin, notes[index].TieOrigin)
		run.Field("Note.SwapAccidentals", note.SwapAccidentals, notes[index].SwapAccidentals)
		run.Field("Note.DurationPercent", note.DurationPercent, notes[index].DurationPercent)
		run.Field("Note.HasPercussionArticulation", note.HasPercussionArticulation, false)
		run.Field("Note.PercussionArticulation", note.PercussionArticulation, 0)
	}
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	for _, code := range []string{"gp8.omit.swap-accidentals", "gp8.omit.note-duration-percent"} {
		if !hasExportCode(report, code) {
			t.Errorf("report = %#v, want %s", report.Entries, code)
		}
	}
	strictData, _, strictErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	var lossErr *ExportLossError
	if len(strictData) != 0 || !errors.As(strictErr, &lossErr) {
		t.Fatalf("strict export = %d bytes, %v", len(strictData), strictErr)
	}
	data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.omit.swap-accidentals", "gp8.omit.note-duration-percent"}}})
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	gotBeats := roundTrip.Tracks[0].Measures[0].Voices[0].Beats
	for index := range notes {
		got := gotBeats[index].Notes[0]
		run.Field("Note.Value", got.Value, notes[index].Value)
		run.Field("Note.String", got.String, notes[index].String)
		run.Field("Note.Kind", got.Kind, notes[index].Kind)
		run.Field("Note.TieOrigin", got.TieOrigin, notes[index].TieOrigin)
	}
	run.Field("Note.SwapAccidentals", gotBeats[4].Notes[0].SwapAccidentals, false)
	run.Field("Note.DurationPercent", gotBeats[5].Notes[0].DurationPercent, float32(1))

	values := extractGPIFLeafText(t, data)
	wireNotes := extractM09Notes(t, data)
	run.Wire("gpifNotes.Notes", len(wireNotes.ids), 6)
	run.Wire("gpifNote.ID", wireNotes.ids, []string{"0", "1", "2", "3", "4", "5"})
	run.Wire("gpifNote.Properties", len(wireNotes.propertyNames) > 0, true)
	run.Wire("gpifProperties.Properties", len(wireNotes.propertyNames) >= 25, true)
	run.Wire("gpifProperty.Name", slices.Contains(wireNotes.propertyNames, "Muted"), true)
	run.Wire("gpifProperty.Fret", values["GPIF/Notes/Note/Properties/Property/Fret"], "00001275")
	run.Wire("gpifProperty.Number", values["GPIF/Notes/Note/Properties/Property/Number"], "646464012745")
	run.Wire("gpifProperty.String", values["GPIF/Notes/Note/Properties/Property/String"], "5550")
	run.Wire("gpifProperty.Pitch", values["GPIF/Notes/Note/Properties/Property/Pitch/Step"], "EEEEEECCGGAA")
	run.Wire("gpifPitch.Step", values["GPIF/Notes/Note/Properties/Property/Pitch/Step"], "EEEEEECCGGAA")
	run.Wire("gpifPitch.Accidental", values["GPIF/Notes/Note/Properties/Property/Pitch/Accidental"], "")
	run.Wire("gpifPitch.Octave", values["GPIF/Notes/Note/Properties/Property/Pitch/Octave"], "444444-1-19922")
	run.Wire("gpifBeat.Notes", values["GPIF/Beats/Beat/Notes"], "012345")
	run.Wire("gpifNote.Tie", len(wireNotes.ties), 3)
	run.Wire("gpifTie.Origin", []bool{wireNotes.ties[0][0], wireNotes.ties[1][0], wireNotes.ties[2][0]}, []bool{true, true, false})
	run.Wire("gpifTie.Destination", []bool{wireNotes.ties[0][1], wireNotes.ties[1][1], wireNotes.ties[2][1]}, []bool{false, true, true})
}

type m09WireNotes struct {
	ids           []string
	propertyNames []string
	ties          [][2]bool
}

func extractM09Notes(t *testing.T, data []byte) m09WireNotes {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var source io.ReadCloser
	for _, file := range archive.File {
		if file.Name == "Content/score.gpif" {
			source, err = file.Open()
			if err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	if source == nil {
		t.Fatal("GP8 archive has no score.gpif")
	}
	defer source.Close()
	decoder := xml.NewDecoder(source)
	var result m09WireNotes
	inNotes := false
	for {
		token, decodeErr := decoder.Token()
		if errors.Is(decodeErr, io.EOF) {
			return result
		}
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		switch value := token.(type) {
		case xml.StartElement:
			switch value.Name.Local {
			case "Notes":
				inNotes = true
			case "Note":
				if inNotes {
					for _, attribute := range value.Attr {
						if attribute.Name.Local == "id" {
							result.ids = append(result.ids, attribute.Value)
						}
					}
				}
			case "Property":
				if inNotes {
					for _, attribute := range value.Attr {
						if attribute.Name.Local == "name" {
							result.propertyNames = append(result.propertyNames, attribute.Value)
						}
					}
				}
			case "Tie":
				if inNotes {
					tie := [2]bool{}
					for _, attribute := range value.Attr {
						parsed, _ := strconv.ParseBool(attribute.Value)
						switch attribute.Name.Local {
						case "origin":
							tie[0] = parsed
						case "destination":
							tie[1] = parsed
						}
					}
					result.ties = append(result.ties, tie)
				}
			}
		case xml.EndElement:
			if value.Name.Local == "Notes" {
				inNotes = false
			}
		}
	}
}

func TestSemanticMatrixM09NoteValidation(t *testing.T) {
	runSemanticMatrixM09NoteValidation(newSemanticMatrixRun(t))
}

func runSemanticMatrixM09NoteValidation(run *semanticMatrixRun) {
	t := run.t
	for _, test := range []struct {
		name   string
		mutate func(*Note)
		want   string
	}{
		{name: "negative duration percent", mutate: func(note *Note) { note.DurationPercent = -0.1 }, want: "duration percent"},
		{name: "NaN duration percent", mutate: func(note *Note) { note.DurationPercent = float32(math.NaN()) }, want: "duration percent"},
		{name: "infinite duration percent", mutate: func(note *Note) { note.DurationPercent = float32(math.Inf(1)) }, want: "duration percent"},
		{name: "negative string", mutate: func(note *Note) { note.String = -1 }, want: "string"},
		{name: "string overflow", mutate: func(note *Note) { note.String = 7 }, want: "string"},
		{name: "rest note", mutate: func(note *Note) { note.Kind = NoteTypeRest }, want: "note kind"},
		{name: "unknown note kind", mutate: func(note *Note) { note.Kind = NoteType(9) }, want: "note kind"},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := semanticValidPitchedGP8Song(t)
			song.Tracks[0].Settings.Notation = true
			note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
			test.mutate(note)
			report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
			if !slices.ContainsFunc(report.Entries, func(entry ExportReportEntry) bool { return entry.Disposition == ExportDispositionRejected }) {
				t.Fatalf("report = %#v, want rejected %s", report.Entries, test.want)
			}
		})
	}

	unbound := semanticValidPitchedGP8Song(t)
	unbound.Tracks[0].Settings.Notation = true
	note := &unbound.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	note.Kind = NoteTypeTie
	note.TieOrigin = false
	if diagnostics := ValidateSong(unbound); !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == "score.note.tie-destination" }) {
		t.Fatalf("unbound tie diagnostics = %#v, want tie destination diagnostic", diagnostics)
	}
	if _, _, err := ExportWithReport(unbound, ExportFormatGP8, ExportOptions{}); err != nil {
		t.Fatalf("encoding an excerpt tie: %v", err)
	}
}
