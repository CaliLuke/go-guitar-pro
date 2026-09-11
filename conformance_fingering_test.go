// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

const conformanceFingeringCase = "M11-FINGERING"

func TestConformanceFingering(t *testing.T) {
	runConformanceFingering(newConformanceRun(t))
}

func runConformanceFingering(run *conformanceRun) {
	t := run.t
	song := conformanceTechniqueSong(t)
	song.Tracks[0].Settings.Notation = true
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	left := []Fingering{FingeringThumb, FingeringIndex, FingeringMiddle, FingeringAnnular, FingeringLittle}
	right := []Fingering{FingeringLittle, FingeringAnnular, FingeringMiddle, FingeringIndex, FingeringThumb}
	beat.Notes = make([]Note, len(left)+1)
	for index := range left {
		beat.Notes[index] = Note{Value: int16(index + 1), String: int8(index%5 + 1), Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1, Effect: defaultNoteEffect()}
		beat.Notes[index].Effect.LeftHandFinger = left[index]
		beat.Notes[index].Effect.RightHandFinger = right[index]
		beat.Notes[index].Effect.HasLeftHandFinger = index == 0
		beat.Notes[index].Effect.HasRightHandFinger = index == len(right)-1
	}
	beat.Notes[len(left)] = Note{Value: 6, String: 6, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1, Effect: defaultNoteEffect()}

	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil {
		t.Fatal(err)
	}
	run.ClaimReport(claimSite("fingering", "export", conformanceFingeringCase, "different left and right thumb-through-little fingerings")).Report(conformanceFingeringCase, reportCodes(report), []string{})
	document := conformanceWireDocument(t, data)
	if len(document.Notes.Notes) != len(beat.Notes) {
		t.Fatalf("wire notes = %d, want %d", len(document.Notes.Notes), len(beat.Notes))
	}
	wantLeft := []string{"P", "I", "M", "A", "C"}
	wantRight := []string{"C", "A", "M", "I", "P"}
	for index := range left {
		wire := document.Notes.Notes[index]
		if wire.LeftFingering == nil || wire.RightFingering == nil {
			t.Fatalf("wire note %d fingerings = %#v, %#v", index, wire.LeftFingering, wire.RightFingering)
		}
		run.ClaimSerialization(claimSite("fingering", "export", conformanceFingeringCase, "different left and right thumb-through-little fingerings")).Wire("gpifNote.LeftFingering", *wire.LeftFingering, wantLeft[index])
		run.Wire("gpifNote.RightFingering", *wire.RightFingering, wantRight[index])
	}
	absentWire := document.Notes.Notes[len(left)]
	if absentWire.LeftFingering != nil || absentWire.RightFingering != nil {
		t.Fatalf("absent wire fingerings = %#v, %#v", absentWire.LeftFingering, absentWire.RightFingering)
	}

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	gotNotes := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Notes
	for index := range left {
		got := gotNotes[index].Effect
		run.ClaimPrimary(claimSite("fingering", "export", conformanceFingeringCase, "different left and right thumb-through-little fingerings")).Preserved("NoteEffect.LeftHandFinger", got.LeftHandFinger, left[index])
		run.Preserved("NoteEffect.RightHandFinger", got.RightHandFinger, right[index])
		run.Preserved("NoteEffect.HasLeftHandFinger", got.HasLeftHandFinger, true)
		run.Preserved("NoteEffect.HasRightHandFinger", got.HasRightHandFinger, true)
	}
	absent := gotNotes[len(left)].Effect
	if absent.HasLeftHandFinger || absent.HasRightHandFinger {
		t.Fatalf("absent round-trip fingering markers = %#v", absent)
	}

	openSong := conformanceTechniqueSong(t)
	openSong.Tracks[0].Settings.Notation = true
	openNote := &openSong.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	openNote.Effect.LeftHandFinger = FingeringOpen
	openNote.Effect.RightHandFinger = FingeringOpen
	openNote.Effect.HasLeftHandFinger = true
	openNote.Effect.HasRightHandFinger = true
	openData, openReport, openErr := ExportWithReport(openSong, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	var loss *ExportLossError
	if len(openData) != 0 || !errors.As(openErr, &loss) {
		t.Fatalf("strict open fingering export = %d bytes, %#v, %v", len(openData), openReport.Entries, openErr)
	}
	if got := reportCodes(openReport); !reflect.DeepEqual(got, []string{"gp8.omit.left-hand-fingering", "gp8.omit.right-hand-fingering"}) {
		t.Fatalf("open fingering report = %#v", got)
	}

	leftInvalid := "Future"
	rightInvalid := ""
	context := &parseContext{format: "GP8"}
	gpifAuditDiagnostics(gpifDocument{Notes: gpifNotes{Notes: []gpifNote{{ID: "note", LeftFingering: &leftInvalid, RightFingering: &rightInvalid}}}}, context)
	for _, code := range []string{"GPIF.Note.LeftFingering.InvalidValue", "GPIF.Note.RightFingering.InvalidValue"} {
		if !slices.ContainsFunc(context.diagnostics, func(diagnostic ParseDiagnostic) bool { return diagnostic.Code == code }) {
			t.Errorf("source diagnostics = %#v, want %s", context.diagnostics, code)
		}
	}
}

type conformanceFingeringFact struct {
	Track int    `json:"track"`
	Staff int    `json:"staff"`
	Bar   int    `json:"bar"`
	Voice int    `json:"voice"`
	Beat  int    `json:"beat"`
	Note  int    `json:"note"`
	Left  string `json:"left"`
	Right string `json:"right"`
}

func TestAlphaTabPreservesFingering(t *testing.T) {
	requireAlphaTabConformance(t)
	fixture := "testdata/gp7/effects.gp"
	var source []conformanceFingeringFact
	readAlphaTabOracleFacts(t, "--fingering", fixture, &source)
	if len(source) == 0 {
		t.Fatal("AlphaTab source contains no fingering facts")
	}
	song := parseTestFixture(t, fixture)
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var output []conformanceFingeringFact
	readAlphaTabOracleFacts(t, "--fingering", writeConformanceFixture(t, data), &output)
	if !reflect.DeepEqual(output, source) {
		t.Fatalf("AlphaTab output fingering facts = %#v, want %#v", output, source)
	}
	conformanceIndependentClaim(t, "field:NoteEffect.LeftHandFinger", claimSite("fingering", "export", conformanceFingeringCase, "different left and right thumb-through-little fingerings"))
}
