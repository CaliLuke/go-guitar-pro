// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"errors"
	"slices"
	"strings"
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
	run.Dispatch("gpifReadStaffStrings:property.Name", gpifReadStaffStrings(gpifStaff{Properties: []gpifStaffProperty{{Name: "Tuning", Pitches: "40 64"}}}), []GuitarString{{Number: 1, Value: 64}, {Number: 2, Value: 40}})
	capoValue := 4
	readCapo, foundCapo, readCapoErr := gpifReadCapo([]gpifStaffProperty{{Name: "CapoFret", Fret: &capoValue}})
	run.Dispatch("gpifReadCapo:property.Name", []any{readCapo, foundCapo, readCapoErr}, []any{int32(4), true, error(nil)})
	auditContext := &parseContext{format: "GP8"}
	for _, property := range []gpifStaffProperty{{Name: "Tuning", Pitches: "40 64"}, {Name: "CapoFret", Fret: &capoValue}, {Name: "ChordCollection"}, {Name: "DiagramCollection"}} {
		gpifAuditTrackProperty(auditContext, "t0", "/GPIF/Tracks/Track", property)
	}
	run.Dispatch("gpifAuditOwnedStaffProperty:property.Name", len(auditContext.diagnostics), 0)

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
	capoStrings := []GuitarString{{Number: 1, Value: 64}}
	capoTrack := Track{Offset: 4, Strings: capoStrings}
	capoNote := Note{String: 1}
	run.Field("Track.Offset", capoTrack.Offset, int32(4))
	if sounding := gp8NoteMIDI(&capoTrack, capoStrings, &capoNote) + int(capoTrack.Offset); sounding != 68 {
		t.Errorf("capo sounding MIDI = %d, want 68", sounding)
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
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	run.Wire("gpifStaffProperty.Name", strings.Contains(string(readZipMember(t, archive, "Content/score.gpif")), `name="CapoFret"`), true)
	run.Wire("gpifStaffProperty.Fret", roundTrip.Tracks[0].Offset, int32(4))
	run.Wire("gpifStaffProperty.Pitches", roundTrip.Tracks[0].Strings, track.Strings)
}
