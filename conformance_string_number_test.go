// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"reflect"
	"strings"
	"testing"
)

const stringNumberCase = "M02-STRING-NUMBER-DISPLAY"
const stringNumberValue = "explicit display on non-default string with reused note definition"
const stringNumberFixture = "testdata/gp8/string-number-display.gp"

func TestConformanceStringNumberDisplay(t *testing.T) {
	runConformanceStringNumberDisplay(newConformanceRun(t))
}

func runConformanceStringNumberDisplay(run *conformanceRun) {
	t := run.t
	song := stringNumberSong(t)
	beats := song.Tracks[0].Measures[0].Voices[0].Beats
	run.ClaimPrimary(claimAllStages("show-string", stringNumberCase, stringNumberValue)...).Preserved("Note.ShowStringNumber", beats[0].Notes[0].ShowStringNumber, true)
	for i, want := range []bool{true, false, true, false} {
		run.Preserved("Note.ShowStringNumber", beats[i].Notes[0].ShowStringNumber, want)
	}
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil {
		t.Fatal(err)
	}
	run.ClaimReport(claimSite("show-string", "export", stringNumberCase, stringNumberValue)).Report(stringNumberCase, reportCodes(report), []string{})
	doc := conformanceWireDocument(t, data)
	for i, want := range []bool{true, false, true, false} {
		enabled := false
		for _, property := range doc.Notes.Notes[i].Properties.Properties {
			if property.Name == "ShowStringNumber" {
				enabled = property.Enable != nil
			}
		}
		run.ClaimSerialization(claimSite("show-string", "export", stringNumberCase, stringNumberValue)).Wire("gpifProperty.Enable", enabled, want)
	}
	// A present Enable child means true regardless of its text; no Enable is false.
	for _, enable := range []*string{nil, stringPointer(""), stringPointer("false")} {
		note, readErr := gpifNoteToNote(&gpifNote{Properties: gpifProperties{Properties: []gpifProperty{{Name: "ShowStringNumber", Enable: enable}}}}, 6, false)
		if readErr != nil {
			t.Fatal(readErr)
		}
		run.Preserved("Note.ShowStringNumber", note.ShowStringNumber, enable != nil)
	}
	context := &parseContext{format: "GP8"}
	gpifAuditNoteProperty(context, "n0", "/GPIF/Notes/Note", gpifProperty{Name: "ShowStringNumber", Enable: stringPointer("")})
	run.Dispatch("gpifAuditNoteProperty:property.Name", len(context.diagnostics), 0)
	beats[0].Notes[0].ShowStringNumber = false
	if !beats[2].Notes[0].ShowStringNumber {
		t.Fatal("reused definition shares display state")
	}
}

func stringNumberSong(t *testing.T) *Song {
	t.Helper()
	song := parseTestFixture(t, stringNumberFixture)
	song.Version = Version{}
	song.Tracks[0].Settings = TrackSettings{Notation: true}
	return song
}

type stringNumberFact struct {
	Track, Staff, Bar, Voice, Beat, Note int
	String, Fret, MIDI                   int
	Show                                 bool
}

func TestAlphaTabStringNumberDisplay(t *testing.T) {
	requireAlphaTabConformance(t)
	want := []stringNumberFact{{0, 0, 0, 0, 0, 0, 4, 9, 64, true}, {0, 0, 0, 0, 1, 0, 5, 2, 61, false}, {0, 0, 0, 0, 2, 0, 4, 9, 64, true}, {0, 0, 0, 0, 3, 0, 3, 8, 58, false}}
	var got []stringNumberFact
	readAlphaTabOracleFacts(t, "--string-number-display", stringNumberFixture, &got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("source facts=%#v", got)
	}
	song := stringNumberSong(t)
	for _, edited := range []bool{false, true} {
		if edited {
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].ShowStringNumber = false
			song.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[0].ShowStringNumber = true
			want[0].Show = false
			want[1].Show = true
		}
		data, err := Export(song, ExportFormatGP8)
		if err != nil {
			t.Fatal(err)
		}
		readAlphaTabOracleFacts(t, "--string-number-display", writeConformanceFixture(t, data), &got)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("edited=%t consumer=%#v", edited, got)
		}
	}
	conformanceIndependentClaim(t, "field:Note.ShowStringNumber", claimAllStages("show-string", stringNumberCase, stringNumberValue)...)
	// GPIF enables display by child presence even for literal false text.
	raw := string(conformanceBarreGPIF(t, mustStringNumberExport(t, stringNumberSong(t))))
	raw = strings.ReplaceAll(raw, "<Enable></Enable>", "<Enable>false</Enable>")
	readAlphaTabOracleFacts(t, "--string-number-display", writeConformanceFixture(t, conformanceGPIFArchive(t, raw)), &got)
	if !got[0].Show || !got[2].Show {
		t.Fatal("consumer changed Enable presence semantics")
	}
}

func mustStringNumberExport(t *testing.T, song *Song) []byte {
	t.Helper()
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
