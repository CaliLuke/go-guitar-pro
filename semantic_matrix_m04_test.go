// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"slices"
	"testing"
)

func TestSemanticMatrixM04InstrumentContext(t *testing.T) {
	runSemanticMatrixM04InstrumentContext(newSemanticMatrixRun(t))
}

func runSemanticMatrixM04InstrumentContext(run *semanticMatrixRun) {
	t := run.t
	for _, pitches := range []string{"64", "40 45 50 55 59 64", "35 40 45 50 55 59 64"} {
		strings := gpifReadStaffStrings(gpifStaff{Properties: []gpifStaffProperty{{Name: "Tuning", Pitches: pitches}}})
		run.Field("GuitarString.Number", strings[0].Number, int8(1))
		run.Field("GuitarString.Value", strings[0].Value, int8(64))
		run.Field("Track.Strings", len(strings), len(splitIDs(pitches)))
		track := Track{Strings: strings}
		for _, note := range []Note{{String: 1}, {String: int8(len(strings)), Value: 3}} {
			got := gp8NoteMIDI(&track, strings, &note)
			want := int(strings[int(note.String)-1].Value) + int(note.Value)
			if got != want {
				t.Errorf("sounding MIDI for string %d = %d, want %d", note.String, got, want)
			}
		}
	}
	run.Dispatch("gpifReadStaffStrings:property.Name", "Tuning", "Tuning")
	run.Dispatch("gpifReadCapo:property.Name", "CapoFret", "CapoFret")
	run.Dispatch("gpifAuditOwnedStaffProperty:property.Name", []string{"CapoFret", "ChordCollection", "DiagramCollection", "Tuning"}, []string{"CapoFret", "ChordCollection", "DiagramCollection", "Tuning"})

	capo := func(value int) gpifStaffProperty { return gpifStaffProperty{Name: "CapoFret", Fret: &value} }
	tests := []struct {
		track  gpifTrack
		value  int32
		common bool
	}{
		{track: gpifTrack{Staves: gpifStaves{Staff: []gpifStaff{{}}}}, common: true},
		{track: gpifTrack{Properties: []gpifStaffProperty{capo(2)}, Staves: gpifStaves{Staff: []gpifStaff{{}}}}, value: 2, common: true},
		{track: gpifTrack{Staves: gpifStaves{Staff: []gpifStaff{{Properties: []gpifStaffProperty{capo(4)}}}}}, value: 4, common: true},
		{track: gpifTrack{Properties: []gpifStaffProperty{capo(2)}, Staves: gpifStaves{Staff: []gpifStaff{{Properties: []gpifStaffProperty{capo(4)}}, {Properties: []gpifStaffProperty{capo(4)}}}}}, value: 4, common: true},
		{track: gpifTrack{Staves: gpifStaves{Staff: []gpifStaff{{Properties: []gpifStaffProperty{capo(2)}}, {Properties: []gpifStaffProperty{capo(4)}}}}}, value: 2},
		{track: gpifTrack{Properties: []gpifStaffProperty{capo(2)}, Staves: gpifStaves{Staff: []gpifStaff{{Properties: []gpifStaffProperty{capo(0)}}, {}}}}, common: false},
	}
	for _, test := range tests {
		got, err := gpifResolveCapo(test.track)
		if err != nil || got.value != test.value || got.common != test.common {
			t.Errorf("capo resolution = %#v, %v, want %d common %t", got, err, test.value, test.common)
		}
		run.Field("Track.Offset", got.value, test.value)
	}

	song := semanticValidPitchedGP8Song(t)
	track := &song.Tracks[0]
	track.Offset = 4
	track.FretCount = 31
	track.Port = 3
	track.TwelveStringedGuitarTrack = true
	track.BanjoTrack = true
	track.Settings.Notation = true
	track.Staves[0].StandardNotationLineCount = 7
	run.Field("Track.FretCount", track.FretCount, uint8(31))
	run.Field("Track.Port", track.Port, uint8(3))
	run.Field("Track.TwelveStringedGuitarTrack", track.TwelveStringedGuitarTrack, true)
	run.Field("Track.BanjoTrack", track.BanjoTrack, true)
	run.Field("Staff.StandardNotationLineCount", track.Staves[0].StandardNotationLineCount, 7)
	run.Field("Track.PercussionTrack", track.PercussionTrack, false)
	run.Field("Staff.PercussionTrack", track.Staves[0].PercussionTrack, false)

	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	wantCodes := []string{
		"gp8.omit.track-fret-count",
		"gp8.omit.track-port",
		"gp8.omit.track-twelve-stringed",
		"gp8.omit.track-banjo",
		"gp8.omit.staff-line-count",
	}
	for _, code := range wantCodes {
		if !slices.ContainsFunc(report.Entries, func(entry ExportReportEntry) bool { return entry.Code == code }) {
			t.Errorf("report = %#v, want %s", report.Entries, code)
		}
	}
	strictData, _, strictErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	var lossErr *ExportLossError
	if len(strictData) != 0 || !errors.As(strictErr, &lossErr) {
		t.Fatalf("strict export = %d bytes, %v", len(strictData), strictErr)
	}

	allowed := ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: wantCodes}}
	data, exported, err := ExportWithReport(song, ExportFormatGP8, allowed)
	if err != nil || len(data) == 0 || len(exported.Entries) != len(wantCodes) {
		t.Fatalf("allowlisted export = %d bytes, %#v, %v", len(data), exported, err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Field("Track.Offset", roundTrip.Tracks[0].Offset, int32(4))
	run.Wire("gpifStaffProperty.Name", "CapoFret", "CapoFret")
	run.Wire("gpifStaffProperty.Fret", roundTrip.Tracks[0].Offset, int32(4))
	run.Wire("gpifStaffProperty.Pitches", roundTrip.Tracks[0].Strings, track.Strings)
}
