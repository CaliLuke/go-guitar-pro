// SPDX-License-Identifier: MIT

package integration_test

import (
	"encoding/xml"
	"slices"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestPartialCapoEditsInvalidatePitchSpelling(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*gp.Staff)
		midi int
	}{
		{"offset", func(s *gp.Staff) { s.PartialCapo.Offset = 5 }, 69},
		{"selection", func(s *gp.Staff) { s.PartialCapo.Strings[0] = false }, 64},
		{"clear", func(s *gp.Staff) { s.PartialCapo = nil }, 64},
		{"fretted", func(s *gp.Staff) { s.Measures[0].Voices[0].Beats[0].Notes[0].Value = 1 }, 65},
	} {
		t.Run(test.name, func(t *testing.T) {
			song, err := gp.ParseFile("../testdata/gp8/partial-capo-asymmetric.gp")
			if err != nil {
				t.Fatal(err)
			}
			test.edit(&song.Tracks[0].Staves[0])
			data, err := gp.Export(song, gp.ExportFormatGP8)
			if err != nil {
				t.Fatal(err)
			}
			assertPartialExportPitches(t, data, []int{test.midi})
			if _, err := gp.ParseWithOptions(data, gp.ParseOptions{Strict: true, StrictKinds: []gp.ParseDiagnosticKind{gp.ParseDiagnosticInvalidData}}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPartialCapoAppliesToOpenGraceOnly(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp8/partial-capo-asymmetric.gp")
	if err != nil {
		t.Fatal(err)
	}
	note := &song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Notes[0]
	note.Value = 1
	note.Effect.Graces = []gp.GraceEffect{{Fret: 0, Duration: 32, Velocity: note.Velocity}}
	data, err := gp.Export(song, gp.ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	assertPartialExportPitches(t, data, []int{67, 65})
}

func TestPartialCapoValidation(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*gp.Staff)
		code string
	}{
		{"negative offset", func(s *gp.Staff) { s.PartialCapo.Offset = -1 }, "score.staff.partial-capo.offset"},
		{"short selection", func(s *gp.Staff) { s.PartialCapo.Strings = []bool{true} }, "score.staff.partial-capo.strings"},
		{"nil selection", func(s *gp.Staff) { s.PartialCapo.Strings = nil }, "score.staff.partial-capo.strings"},
		{"note pitch", func(s *gp.Staff) { s.PartialCapo.Offset = 100 }, "score.note.partial-capo-midi"},
		{"grace pitch", func(s *gp.Staff) {
			s.PartialCapo.Offset = 100
			n := &s.Measures[0].Voices[0].Beats[0].Notes[0]
			n.Value = 1
			n.Effect.Graces = []gp.GraceEffect{{Fret: 0, Duration: 32, Velocity: n.Velocity}}
		}, "score.grace.partial-capo-midi"},
	} {
		t.Run(test.name, func(t *testing.T) {
			song, err := gp.ParseFile("../testdata/gp8/partial-capo-asymmetric.gp")
			if err != nil {
				t.Fatal(err)
			}
			test.edit(&song.Tracks[0].Staves[0])
			if !slices.ContainsFunc(gp.ValidateSong(song), func(d gp.ScoreDiagnostic) bool {
				return d.Code == test.code && d.Location.Track == 0 && d.Location.Staff == 0
			}) {
				t.Fatal("missing scoped diagnostic", gp.ValidateSong(song))
			}
			data, _, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{AllowedCodes: []string{"gp8.reject." + test.code}}})
			if err == nil || len(data) != 0 {
				t.Fatal("invalid partial capo exported")
			}
		})
	}
}

func assertPartialExportPitches(t *testing.T, data []byte, want []int) {
	t.Helper()
	var doc struct {
		Notes []struct {
			Properties []struct {
				Name   string `xml:"name,attr"`
				Number int
				Pitch  *struct {
					Step, Accidental string
					Octave           int
				}
			} `xml:"Properties>Property"`
		} `xml:"Notes>Note"`
	}
	if err := xml.Unmarshal(textContractGPIF(t, data), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Notes) < len(want) {
		t.Fatal("missing output notes")
	}
	for i, midi := range want {
		for _, p := range doc.Notes[i].Properties {
			if p.Name == "Midi" && p.Number != midi {
				t.Fatalf("note %d Midi=%d want%d", i, p.Number, midi)
			}
			if p.Name == "ConcertPitch" {
				steps := map[string]int{"C": 0, "D": 2, "E": 4, "F": 5, "G": 7, "A": 9, "B": 11}
				accidental := map[string]int{"": 0, "#": 1, "b": -1, "x": 2, "bb": -2}
				if got := p.Pitch.Octave*12 + steps[p.Pitch.Step] + accidental[p.Pitch.Accidental]; got != midi {
					t.Fatalf("note %d concert=%d want%d", i, got, midi)
				}
			}
		}
	}
}

func TestInactivePartialCapoRetainsImportedSpelling(t *testing.T) {
	source, err := gp.ParseFile("../testdata/gp7/serenade.gp")
	if err != nil {
		t.Fatal(err)
	}
	original := source.Tracks[0].Measures[0].Voices[0].Beats[4].Notes[0]
	if original.Value != 1 {
		t.Fatal("expected fretted source sentinel")
	}
	for _, test := range []struct {
		name    string
		partial *gp.PartialCapo
	}{
		{"nil", nil},
		{"explicit zero", &gp.PartialCapo{Strings: make([]bool, 6)}},
		{"selected zero", &gp.PartialCapo{Strings: []bool{true, false, false, false, false, false}}},
		{"unselected offset", &gp.PartialCapo{Offset: 3, Strings: make([]bool, 6)}},
		{"fretted selected", &gp.PartialCapo{Offset: 3, Strings: []bool{true, false, false, false, false, false}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := accidentalPublicSong(t)
			staff := &song.Tracks[0].Staves[0]
			staff.Strings = slices.Clone(source.Tracks[0].Staves[0].Strings)
			song.Tracks[0].Strings = staff.Strings
			staff.DisplayTranspositionPitch = -12
			staff.PartialCapo = test.partial
			staff.Measures[0].Voices[0].Beats[1].Notes = []gp.Note{original}
			data, err := gp.Export(song, gp.ExportFormatGP8)
			if err != nil {
				t.Fatal(err)
			}
			var doc struct {
				Notes []struct {
					Properties []struct {
						Name  string `xml:"name,attr"`
						Pitch *struct {
							Step, Accidental string
							Octave           int
						}
					} `xml:"Properties>Property"`
				} `xml:"Notes>Note"`
			}
			if err := xml.Unmarshal(textContractGPIF(t, data), &doc); err != nil {
				t.Fatal(err)
			}
			if len(doc.Notes) != 1 {
				t.Fatal("unexpected note count")
			}
			found := false
			for _, p := range doc.Notes[0].Properties {
				if p.Name == "TransposedPitch" {
					found = true
					if p.Pitch == nil || p.Pitch.Step != "F" || p.Pitch.Accidental != "" || p.Pitch.Octave != 6 {
						t.Fatalf("unchanged source spelling lost: %#v", p.Pitch)
					}
				}
			}
			if !found {
				t.Fatal("missing source pitch")
			}
		})
	}
}

func TestPartialCapoCompatibilityTuningValidation(t *testing.T) {
	for _, resizeMask := range []bool{false, true} {
		song, err := gp.ParseFile("../testdata/gp8/partial-capo-asymmetric.gp")
		if err != nil {
			t.Fatal(err)
		}
		track := &song.Tracks[0]
		track.Strings = track.Strings[:4]
		track.Measures[0].Voices[0].Beats = track.Measures[0].Voices[0].Beats[:4]
		for i := range track.Measures[0].Voices[0].Beats {
			track.Measures[0].Voices[0].Beats[i].Notes[0].AccidentalMode = gp.NoteAccidentalDefault
		}
		if resizeMask {
			track.Staves[0].PartialCapo.Strings = []bool{true, false, false, true}
		}
		code := "score.staff.partial-capo.strings"
		invalid := slices.ContainsFunc(gp.ValidateSong(song), func(d gp.ScoreDiagnostic) bool {
			return d.Code == code && d.Location.Track == 0 && d.Location.Staff == 0
		})
		if invalid == resizeMask {
			t.Fatalf("resizeMask=%t: invalid=%t; validation must use the four-string compatibility tuning", resizeMask, invalid)
		}
		options := gp.ExportOptions{}
		if !resizeMask {
			options.LossPolicy = gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.reject." + code}}
		}
		data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, options)
		if !resizeMask {
			if err == nil || len(data) != 0 || !slices.ContainsFunc(report.Entries, func(e gp.ExportReportEntry) bool { return e.Code == "gp8.reject."+code }) {
				t.Fatalf("mismatched resolved mask exported: %v %#v", err, report)
			}
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		round, err := gp.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		got := round.Tracks[0].Staves[0]
		if len(got.Strings) != 4 || got.PartialCapo == nil || !slices.Equal(got.PartialCapo.Strings, []bool{true, false, false, true}) {
			t.Fatalf("edited tuning/mask not retained: %#v", got.PartialCapo)
		}
		if len(track.Staves[0].Strings) != 6 || len(track.Strings) != 4 {
			t.Fatal("validation/export mutated tuning views")
		}
	}
}
