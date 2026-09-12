// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"os"
	"reflect"
	"slices"
	"testing"
)

func TestConformancePartialCapo(t *testing.T) { runConformancePartialCapo(newConformanceRun(t)) }
func runConformancePartialCapo(run *conformanceRun) {
	assertPartialCapoCompatibilityTuning(run)
	t := run.t
	for _, test := range []struct {
		name        string
		flags       []bool
		midi, alpha []int64
	}{
		{"asymmetric", []bool{true, false, false, true, false, false}, []int64{67, 59, 55, 53, 45, 40}, []int64{64, 59, 55, 50, 45, 40}},
		{"combined", []bool{true, false, false, true, false, false}, []int64{69, 61, 57, 55, 47, 42}, []int64{66, 61, 57, 52, 47, 42}},
		{"fretted", []bool{true, true, true, true, true, true}, []int64{67, 60, 57, 53, 49, 45}, []int64{64, 60, 57, 53, 49, 45}},
	} {
		song := parseTestFixture(t, "testdata/gp8/partial-capo-"+test.name+".gp")
		staff := &song.Tracks[0].Staves[0]
		want := &PartialCapo{Offset: 3, Strings: test.flags}
		run.Preserved("Staff.PartialCapo", staff.PartialCapo, want)
		run.Preserved("PartialCapo.Offset", staff.PartialCapo.Offset, int32(3))
		run.Preserved("PartialCapo.Strings", staff.PartialCapo.Strings, test.flags)
		var midi []int64
		for _, beat := range staff.Measures[0].Voices[0].Beats {
			midi = append(midi, soundingNoteMIDI(staff, &beat.Notes[0]))
		}
		if !slices.Equal(midi, test.midi) {
			t.Fatalf("%s MIDI=%v want%v", test.name, midi, test.midi)
		}
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if hasExportCode(report, "gp8.omit.partial-capo-consumer") {
			t.Fatalf("reference defect incorrectly reported as export loss: %#v", report)
		}
		run.Report("M04-PARTIAL-CAPO", hasExportCode(report, "gp8.omit.partial-capo-consumer"), false)
		var allowed []string
		for _, entry := range report.Entries {
			allowed = append(allowed, entry.Code)
		}
		if _, _, strictErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}}); strictErr != nil {
			t.Fatalf("partial capo strict preservation: %v", strictErr)
		}
		wire := conformanceSingleWireTrack(t, data).Staves.Staff[0]
		var names []string
		for _, p := range wire.Properties {
			if p.Name == "PartialCapoFret" {
				names = append(names, p.Name)
				run.Wire("gpifStaffProperty.Fret", *p.Fret, 3)
			}
			if p.Name == "PartialCapoStringFlags" {
				names = append(names, p.Name)
				wantFlags := "001001"
				if test.name == "fretted" {
					wantFlags = "111111"
				}
				run.Wire("gpifStaffProperty.Bitset", *p.Bitset, wantFlags)
			}
		}
		run.Wire("gpifStaffProperty.Name", names, []string{"PartialCapoFret", "PartialCapoStringFlags"})
		round, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		run.Preserved("Staff.PartialCapo", round.Tracks[0].Staves[0].PartialCapo, want)
		parsed, err := gpifReadPartialCapo(wire.Properties, 6)
		if err != nil {
			t.Fatal(err)
		}
		run.Dispatch("gpifReadPartialCapo:property.Name", parsed, want)
		context := &parseContext{}
		for _, p := range wire.Properties {
			gpifAuditStaffProperty(context, "0", "staff", p)
		}
		run.Dispatch("gpifAuditOwnedStaffProperty:property.Name", context.diagnostics, []ParseDiagnostic(nil))
		if os.Getenv("ALPHATAB_CONFORMANCE") != "" {
			var facts []struct{ MIDI int64 }
			readAlphaTabOracleFacts(t, "--string-number-display", writeConformanceFixture(t, data), &facts)
			var got []int64
			for _, f := range facts {
				got = append(got, f.MIDI)
			}
			if !slices.Equal(got, test.alpha) {
				t.Fatalf("reference defect changed: %v want%v", got, test.alpha)
			}
		}
	}
	assertPartialCapoOwnership(t)
	assertPartialCapoInactiveFlags(run)
	flags := "0"
	fret := 0
	legacy, err := gpifReadPartialCapo([]gpifStaffProperty{{Name: "PartialCapoFret", Fret: &fret}, {Name: "PartialCapoStringFlags", Flags: &flags}}, 6)
	if err != nil {
		t.Fatal(err)
	}
	run.Wire("gpifStaffProperty.Flags", flags, "0")
	run.Preserved("PartialCapo.Strings", legacy.Strings, make([]bool, 6))
}

func assertPartialCapoOwnership(t *testing.T) {
	data, err := os.ReadFile("testdata/gp8/partial-capo-asymmetric.gp")
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	if decodeErr := xml.Unmarshal(readBackingGPIF(t, data), &document); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	document.Tracks.Tracks[0].Staves.Staff = append(document.Tracks.Tracks[0].Staves.Staff, document.Tracks.Tracks[0].Staves.Staff[0])
	document.MasterBars.MasterBars[0].Bars += " " + document.MasterBars.MasterBars[0].Bars
	raw, err := xml.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	song, err := Parse(conformanceBackingArchive(t, string(raw), nil))
	if err != nil {
		t.Fatal(err)
	}
	a, b := song.Tracks[0].Staves[0].PartialCapo, song.Tracks[0].Staves[1].PartialCapo
	if !reflect.DeepEqual(a, b) || a == b {
		t.Fatal("partial capo occurrence ownership")
	}
	a.Strings[0] = false
	a.Offset = 7
	if !b.Strings[0] || b.Offset != 3 {
		t.Fatal("partial capo mutable data aliases")
	}
}

func assertPartialCapoInactiveFlags(run *conformanceRun) {
	fret := 0
	flags := "000000"
	context := &parseContext{}
	partial, err := gpifReadPartialCapoWithContext([]gpifStaffProperty{{Name: "PartialCapoFret", Fret: &fret}, {Name: "PartialCapoStringFlags", Bitset: &flags}}, 4, context, "0", 0)
	if err != nil {
		run.t.Fatal(err)
	}
	run.Preserved("PartialCapo.Strings", partial.Strings, make([]bool, 4))
	var codes []string
	for _, diagnostic := range context.diagnostics {
		codes = append(codes, diagnostic.Code)
	}
	run.Report("M04-PARTIAL-CAPO", codes, []string{"GPIF.Staff.PartialCapo.InactiveFlags.Normalized"})
	run.Dispatch("gpifReadPartialCapoWithContext:property.Name", codes, []string{"GPIF.Staff.PartialCapo.InactiveFlags.Normalized"})
}

func assertPartialCapoCompatibilityTuning(run *conformanceRun) {
	song := parseTestFixture(run.t, "testdata/gp8/partial-capo-asymmetric.gp")
	track := &song.Tracks[0]
	track.Strings = track.Strings[:4]
	track.Measures[0].Voices[0].Beats = track.Measures[0].Voices[0].Beats[:4]
	for i := range track.Measures[0].Voices[0].Beats {
		track.Measures[0].Voices[0].Beats[i].Notes[0].AccidentalMode = NoteAccidentalDefault
	}
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
	if err == nil || len(data) != 0 {
		run.t.Fatal("invalid resolved partial mask exported")
	}
	run.Report("M04-PARTIAL-CAPO", reportCodes(report), []string{"gp8.reject.score.staff.partial-capo.strings"})
	flags := []bool{true, false, false, true}
	track.Staves[0].PartialCapo.Strings = flags
	data, err = Export(song, ExportFormatGP8)
	if err != nil {
		run.t.Fatal(err)
	}
	round, err := Parse(data)
	if err != nil {
		run.t.Fatal(err)
	}
	if len(round.Tracks[0].Staves[0].Strings) != 4 {
		run.t.Fatal("compatibility tuning lost")
	}
	run.Preserved("PartialCapo.Strings", round.Tracks[0].Staves[0].PartialCapo.Strings, flags)
}
