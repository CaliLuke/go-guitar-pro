// SPDX-License-Identifier: MIT

package integration_test

import (
	"reflect"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestGP8AuthoredAccidentals(t *testing.T) {
	for _, test := range []struct {
		mode gp.NoteAccidentalMode
		midi int16
	}{
		{gp.NoteAccidentalDefault, 61}, {gp.NoteAccidentalNatural, 60}, {gp.NoteAccidentalSharp, 61}, {gp.NoteAccidentalFlat, 61}, {gp.NoteAccidentalDoubleSharp, 62}, {gp.NoteAccidentalDoubleFlat, 60},
	} {
		s := accidentalPublicSong(t)
		notes := s.Tracks[0].Measures[0].Voices[0].Beats[1].Notes
		for i := range notes {
			notes[i].String = int8(i + 1)
			notes[i].Value = test.midi - 60
			notes[i].Velocity = 47
			notes[i].AccidentalMode = test.mode
		}
		if ds := gp.ValidateSong(s); len(ds) != 0 {
			t.Fatalf("mode %v rejected %#v", test.mode, ds)
		}
		before := append([]gp.Note(nil), notes...)
		data, report, err := gp.ExportWithReport(s, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}})
		if err != nil || len(report.Entries) != 0 {
			t.Fatalf("mode %v export %v %#v", test.mode, err, report)
		}
		parsed, err := gp.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		got := parsed.Tracks[0].Measures[0].Voices[0].Beats[1].Notes
		for _, note := range got {
			if note.AccidentalMode != test.mode || note.Value != test.midi-60 || note.String == 0 {
				t.Fatalf("mode %v changed %#v", test.mode, note)
			}
		}
		if !reflect.DeepEqual(before, notes) {
			t.Fatal("export mutated notes")
		}
	}
}

func TestGP8EnharmonicChordEdits(t *testing.T) {
	s := accidentalPublicSong(t)
	notes := s.Tracks[0].Measures[0].Voices[0].Beats[1].Notes
	for i := range notes {
		notes[i].String = int8(i + 1)
		notes[i].Value = 1
		notes[i].Velocity = 47
	}
	notes[0].AccidentalMode = gp.NoteAccidentalSharp
	notes[1].AccidentalMode = gp.NoteAccidentalFlat
	for _, clear := range []bool{false, true} {
		if clear {
			notes[0].AccidentalMode = gp.NoteAccidentalDefault
			notes[1].AccidentalMode = gp.NoteAccidentalDoubleFlat
			notes[1].Value = 0
		}
		data, report, err := gp.ExportWithReport(s, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}})
		if err != nil || len(report.Entries) != 0 {
			t.Fatalf("edited chord %v %#v", err, report)
		}
		got, err := gp.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		for i, n := range got.Tracks[0].Measures[0].Voices[0].Beats[1].Notes {
			if n.AccidentalMode != notes[i].AccidentalMode || n.Value != notes[i].Value {
				t.Fatalf("spelling or pitch changed %#v", n)
			}
		}
	}
}

func TestGP8AccidentalContradictions(t *testing.T) {
	for _, test := range []struct {
		mode gp.NoteAccidentalMode
		midi int16
	}{{gp.NoteAccidentalNatural, 61}, {gp.NoteAccidentalSharp, 62}, {gp.NoteAccidentalFlat, 60}, {gp.NoteAccidentalDoubleSharp, 60}, {gp.NoteAccidentalDoubleFlat, 61}, {gp.NoteAccidentalMode(255), 60}} {
		s := accidentalPublicSong(t)
		note := &s.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[0]
		note.String = 1
		note.Value = test.midi - 60
		note.AccidentalMode = test.mode
		if ds := gp.ValidateSong(s); len(ds) == 0 {
			t.Fatalf("mode%v pitch%d accepted", test.mode, test.midi)
		}
		if data, _, err := gp.ExportWithReport(s, gp.ExportFormatGP8, gp.ExportOptions{}); err == nil || len(data) != 0 {
			t.Fatalf("contradiction exported mode%v pitch%d", test.mode, test.midi)
		}
	}
}

func accidentalPublicSong(t *testing.T) *gp.Song {
	t.Helper()
	s := dynamicPolicySong(t, 47, false)
	for i := range s.Tracks[0].Staves[0].Strings {
		s.Tracks[0].Staves[0].Strings[i].Value = 60
	}
	return s
}

func TestGP8AccidentalSwapPolicy(t *testing.T) {
	s := accidentalPublicSong(t)
	notes := s.Tracks[0].Measures[0].Voices[0].Beats[1].Notes
	for i := range notes {
		notes[i].Velocity = 47
	}
	notes[0].Value = 1
	notes[0].AccidentalMode = gp.NoteAccidentalFlat
	notes[0].SwapAccidentals = true
	want := gp.PreflightExport(s, gp.ExportFormatGP8, gp.ExportOptions{})
	if len(want.Entries) != 1 || want.Entries[0].Code != "gp8.omit.swap-accidentals" || want.Entries[0].Location.Beat != 1 || want.Entries[0].Location.Note != 0 {
		t.Fatalf("swap report %#v", want)
	}
	for _, allow := range [][]string{nil, {"gp8.omit.harmonic-pitch"}, {"gp8.omit.swap-accidentals"}} {
		data, report, err := gp.ExportWithReport(s, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allow}})
		if !reflect.DeepEqual(report, want) {
			t.Fatal("preflight differs from export")
		}
		if len(allow) == 0 || allow[0] != "gp8.omit.swap-accidentals" {
			if err == nil || len(data) != 0 {
				t.Fatal("unapproved swap loss")
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := gp.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		note := parsed.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[0]
		if note.AccidentalMode != gp.NoteAccidentalFlat || note.Value != 1 || note.SwapAccidentals {
			t.Fatalf("typed and legacy spelling reconciliation %#v", note)
		}
	}
}
