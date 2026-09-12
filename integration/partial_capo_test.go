// SPDX-License-Identifier: MIT

package integration_test

import (
	"os"
	"reflect"
	"strings"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestNativePartialCapoPitchAudit(t *testing.T) {
	for _, name := range []string{"asymmetric", "fretted", "combined"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile("../testdata/gp8/partial-capo-" + name + ".gp")
			if err != nil {
				t.Fatal(err)
			}
			result, err := gp.ParseWithOptions(data, gp.ParseOptions{Strict: true, StrictKinds: []gp.ParseDiagnosticKind{gp.ParseDiagnosticInvalidData}})
			if err != nil {
				t.Fatal(err)
			}
			staff := &result.Song.Tracks[0].Staves[0]
			flags := []bool{true, false, false, true, false, false}
			if name == "fretted" {
				flags = []bool{true, true, true, true, true, true}
			}
			if staff.PartialCapo == nil || staff.PartialCapo.Offset != 3 || !reflect.DeepEqual(staff.PartialCapo.Strings, flags) {
				t.Fatalf("partial capo = %#v", staff.PartialCapo)
			}
			for i, beat := range staff.Measures[0].Voices[0].Beats {
				want := int16(0)
				if name == "fretted" {
					want = int16(i)
				}
				if note := beat.Notes[0]; note.String != int8(i+1) || note.Value != want {
					t.Fatalf("note %d changed: %#v", i, note)
				}
			}
			output, _, err := gp.ExportWithReport(result.Song, gp.ExportFormatGP8, gp.ExportOptions{})
			if err != nil {
				t.Fatal(err)
			}
			round, err := gp.ParseWithOptions(output, gp.ParseOptions{Strict: true, StrictKinds: []gp.ParseDiagnosticKind{gp.ParseDiagnosticInvalidData}})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(round.Song.Tracks[0].Staves[0].PartialCapo, staff.PartialCapo) {
				t.Fatal("partial capo lost in export")
			}
		})
	}
}

func TestPartialCapoMalformedSource(t *testing.T) {
	data, err := os.ReadFile("../testdata/gp8/partial-capo-asymmetric.gp")
	if err != nil {
		t.Fatal(err)
	}
	raw := string(textContractGPIF(t, data))
	for _, test := range []struct{ name, from, to string }{
		{"negative", "<Fret>3</Fret>", "<Fret>-3</Fret>"},
		{"overflow", "<Fret>3</Fret>", "<Fret>2147483648</Fret>"},
		{"short flags", "<Bitset>001001</Bitset>", "<Bitset>001</Bitset>"},
		{"invalid flag", "<Bitset>001001</Bitset>", "<Bitset>00100x</Bitset>"},
		{"missing flags", "<Bitset>001001</Bitset>", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			changed := strings.Replace(raw, test.from, test.to, 1)
			if changed == raw {
				t.Fatal("mutation missed")
			}
			_, err := gp.Parse(buildGP7Archive(t, changed, "", nil))
			if err == nil {
				t.Fatal("invalid partial capo accepted")
			}
		})
	}
	// A partial capo does not excuse corrupt pitch records on any string.
	for _, pair := range [][2]string{{"<Step>G</Step>", "<Step>A</Step>"}, {"<Step>B</Step>", "<Step>C</Step>"}} {
		changed := strings.Replace(raw, pair[0], pair[1], 1)
		if changed == raw {
			t.Fatal("pitch mutation missed")
		}
		if _, err := gp.ParseWithOptions(buildGP7Archive(t, changed, "", nil), gp.ParseOptions{Strict: true, StrictKinds: []gp.ParseDiagnosticKind{gp.ParseDiagnosticInvalidData}}); err == nil {
			t.Fatal("contradictory pitch accepted")
		}
	}
}

func TestPartialCapoPublicEditsAndBounds(t *testing.T) {
	for _, test := range []struct {
		name    string
		edit    func(*gp.Staff)
		invalid bool
	}{
		{"absent", func(s *gp.Staff) { s.PartialCapo = nil }, false},
		{"inactive", func(s *gp.Staff) { s.PartialCapo = &gp.PartialCapo{Strings: make([]bool, len(s.Strings))} }, false},
		{"offset", func(s *gp.Staff) { s.PartialCapo.Offset = 1 }, false},
		{"selection", func(s *gp.Staff) { s.PartialCapo.Strings = []bool{false, true, false, false, false, true} }, false},
		{"negative", func(s *gp.Staff) { s.PartialCapo.Offset = -1 }, true},
		{"short", func(s *gp.Staff) { s.PartialCapo.Strings = s.PartialCapo.Strings[:5] }, true},
		{"long", func(s *gp.Staff) { s.PartialCapo.Strings = append(s.PartialCapo.Strings, true) }, true},
		{"unrepresentable pitch", func(s *gp.Staff) { s.PartialCapo.Offset = 2147483647 }, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			song, err := gp.ParseFile("../testdata/gp8/partial-capo-asymmetric.gp")
			if err != nil {
				t.Fatal(err)
			}
			staff := &song.Tracks[0].Staves[0]
			test.edit(staff)
			for bi := range staff.Measures[0].Voices[0].Beats {
				staff.Measures[0].Voices[0].Beats[bi].Notes[0].AccidentalMode = gp.NoteAccidentalDefault
			}
			before := staff.PartialCapo
			data, _, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
			if test.invalid {
				if err == nil || len(data) != 0 {
					t.Fatal("invalid authoring accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			round, err := gp.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before, round.Tracks[0].Staves[0].PartialCapo) || staff.PartialCapo != before {
				t.Fatal("edit lost or source mutated")
			}
		})
	}
}

func TestPartialCapoInactivePlaceholder(t *testing.T) {
	for _, format := range []string{"gp7", "gp8"} {
		data, err := os.ReadFile("../testdata/" + format + "/faulty.gp")
		if err != nil {
			t.Fatal(err)
		}
		result, err := gp.ParseWithOptions(data, gp.ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		staff := result.Song.Tracks[3].Staves[0]
		if staff.PartialCapo == nil || staff.PartialCapo.Offset != 0 || !reflect.DeepEqual(staff.PartialCapo.Strings, make([]bool, len(staff.Strings))) {
			t.Fatalf("inactive placeholder: %#v", staff.PartialCapo)
		}
		count := 0
		for _, d := range result.Diagnostics {
			if d.Code == "GPIF.Staff.PartialCapo.InactiveFlags.Normalized" {
				count++
			}
		}
		if count == 0 {
			t.Fatal("missing inactive selection normalization")
		}
	}
}

func TestPartialCapoStaffOverride(t *testing.T) {
	raw := `<GPIF><GPVersion>8.1.5</GPVersion><Score><Title>Partial override</Title></Score><MasterTrack><Tracks>0</Tracks></MasterTrack><Tracks><Track id="0"><Name>Guitar</Name><Properties><Property name="Tuning"><Pitches>40 45 50 55 59 64</Pitches></Property><Property name="PartialCapoFret"><Fret>3</Fret></Property><Property name="PartialCapoStringFlags"><Bitset>001001</Bitset></Property></Properties><Staves><Staff><Properties><Property name="Tuning"><Pitches>40 45 50 55</Pitches></Property><Property name="PartialCapoFret"><Fret>2</Fret></Property><Property name="PartialCapoStringFlags"><Bitset>0001</Bitset></Property></Properties></Staff></Staves></Track></Tracks><MasterBars><MasterBar><Time>4/4</Time><Bars>0</Bars></MasterBar></MasterBars><Bars><Bar id="0"><Clef>G2</Clef><Voices>0</Voices></Bar></Bars><Voices><Voice id="0"><Beats>0</Beats></Voice></Voices><Beats><Beat id="0"><Rhythm ref="0"/></Beat></Beats><Notes/><Rhythms><Rhythm id="0"><NoteValue>Whole</NoteValue></Rhythm></Rhythms></GPIF>`
	song, err := gp.Parse(buildGP7Archive(t, raw, "", nil))
	if err != nil {
		t.Fatal(err)
	}
	want := &gp.PartialCapo{Offset: 2, Strings: []bool{true, false, false, false}}
	if !reflect.DeepEqual(song.Tracks[0].Staves[0].PartialCapo, want) {
		t.Fatalf("staff override lost: %#v", song.Tracks[0].Staves[0].PartialCapo)
	}
	inherited := strings.ReplaceAll(raw, `<Property name="PartialCapoFret"><Fret>2</Fret></Property><Property name="PartialCapoStringFlags"><Bitset>0001</Bitset></Property>`, "")
	if inherited == raw {
		t.Fatal("inherited source mutation missed")
	}
	t.Run("active inherited mismatch", func(t *testing.T) {
		if _, parseErr := gp.Parse(buildGP7Archive(t, inherited, "", nil)); parseErr == nil {
			t.Fatal("active inherited mask mismatch accepted")
		}
	})
	t.Run("inactive inherited placeholder", func(t *testing.T) {
		inactive := strings.ReplaceAll(strings.ReplaceAll(inherited, "<Fret>3</Fret>", "<Fret>0</Fret>"), "<Bitset>001001</Bitset>", "<Bitset>000000</Bitset>")
		result, parseErr := gp.ParseWithOptions(buildGP7Archive(t, inactive, "", nil), gp.ParseOptions{})
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		got := result.Song.Tracks[0].Staves[0].PartialCapo
		if got == nil || got.Offset != 0 || !reflect.DeepEqual(got.Strings, make([]bool, 4)) {
			t.Fatalf("inactive inherited mask = %#v", got)
		}
		count := 0
		for _, d := range result.Diagnostics {
			if d.Code == "GPIF.Staff.PartialCapo.InactiveFlags.Normalized" {
				count++
			}
		}
		if count != 1 {
			t.Fatalf("inactive inherited normalizations = %d", count)
		}
	})
}
