// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"slices"
	"testing"
)

func TestConformanceStaffCapo(t *testing.T) {
	runConformanceStaffCapo(newConformanceRun(t))
}

func runConformanceStaffCapo(run *conformanceRun) {
	t := run.t
	song := conformanceStaffCapoSong(t, 2, 5)
	track := &song.Tracks[0]
	wantCapos := []int32{2, 5}
	wantPitches := []int{69, 57}
	run.Preserved("Track.CapoFret", track.CapoFret, int32(2))
	run.Preserved("Staff.CapoFret", staffCapos(track), wantCapos)
	if got := staffCapoPitches(track); !slices.Equal(got, wantPitches) {
		t.Errorf("sounding pitches = %v, want %v", got, wantPitches)
	}

	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true},
	})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("two-staff capo export = %v, %#v", err, report.Entries)
	}
	wireTrack := conformanceCapoWireTrack(t, data)
	if _, found, readErr := gpifReadCapo(wireTrack.Properties); readErr != nil || found {
		t.Fatalf("track-level capo = found %t, error %v; want omitted", found, readErr)
	}
	wireCapos := make([]int32, len(wireTrack.Staves.Staff))
	for staffIndex, staff := range wireTrack.Staves.Staff {
		value, found, readErr := gpifReadCapo(staff.Properties)
		if readErr != nil || !found {
			t.Fatalf("staff %d wire capo = %d, found %t, error %v", staffIndex, value, found, readErr)
		}
		wireCapos[staffIndex] = value
	}
	run.Wire("gpifStaffProperty.Fret", wireCapos, wantCapos)

	roundTrip := conformanceParseCapoExport(t, data)
	run.Preserved("Staff.CapoFret", staffCapos(&roundTrip.Tracks[0]), wantCapos)
	if got := staffCapoPitches(&roundTrip.Tracks[0]); !slices.Equal(got, wantPitches) {
		t.Errorf("round-trip sounding pitches = %v, want %v", got, wantPitches)
	}

	// Direct staff edits are authoritative while the parsed legacy scalar is unchanged.
	directStaffEdit := &roundTrip.Tracks[0]
	directStaffEdit.Staves[1].CapoFret = 7
	directData, err := Export(roundTrip, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	if got := staffCapos(&conformanceParseCapoExport(t, directData).Tracks[0]); !slices.Equal(got, []int32{2, 7}) {
		t.Fatalf("direct staff edit capos = %v, want [2 7]", got)
	}
	if got := staffCapos(directStaffEdit); !slices.Equal(got, []int32{2, 7}) {
		t.Fatalf("export mutated authored staff capos to %v", got)
	}

	// A changed legacy scalar applies uniformly. If both views change, the legacy edit wins.
	legacySource := conformanceParseCapoExport(t, data)
	legacyTrack := &legacySource.Tracks[0]
	legacyTrack.CapoFret = 4
	legacyTrack.Staves[1].CapoFret = 7
	legacyData, err := Export(legacySource, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	if got := staffCapos(&conformanceParseCapoExport(t, legacyData).Tracks[0]); !slices.Equal(got, []int32{4, 4}) {
		t.Fatalf("legacy track edit capos = %v, want [4 4]", got)
	}

	// A programmatic legacy scalar remains usable when no staff has an authored capo.
	legacyOnly := conformanceStaffCapoSong(t, 0, 0)
	legacyOnly.Tracks[0].CapoFret = 3
	legacyOnlyData, err := Export(legacyOnly, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	if got := staffCapos(&conformanceParseCapoExport(t, legacyOnlyData).Tracks[0]); !slices.Equal(got, []int32{3, 3}) {
		t.Fatalf("programmatic legacy capo = %v, want [3 3]", got)
	}

	maximum := int32(1<<31 - 1)
	boundary := conformanceStaffCapoSong(t, maximum, maximum)
	boundary.Tracks[0].CapoFret = maximum
	boundaryData, err := Export(boundary, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	if got := staffCapos(&conformanceParseCapoExport(t, boundaryData).Tracks[0]); !slices.Equal(got, []int32{maximum, maximum}) {
		t.Fatalf("maximum capo round trip = %v, want [%d %d]", got, maximum, maximum)
	}

	invalid := conformanceStaffCapoSong(t, 2, -1)
	invalidData, invalidReport, invalidErr := ExportWithReport(invalid, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true},
	})
	entry := exportReportEntry(invalidReport, "gp8.reject.score.staff.capo")
	if len(invalidData) != 0 || invalidErr == nil || entry == nil {
		t.Fatalf("negative staff capo export = %d bytes, %#v, %v", len(invalidData), invalidReport.Entries, invalidErr)
	}
	if entry.Location.Track != 0 || entry.Location.Staff != 1 {
		t.Fatalf("negative staff capo location = %#v, want track 0 staff 1", entry.Location)
	}

	combinedInvalid := conformanceParseCapoExport(t, data)
	combinedInvalid.Tracks[0].CapoFret = 4
	combinedInvalid.Tracks[0].Staves[1].CapoFret = -1
	combinedData, combinedReport, combinedErr := ExportWithReport(combinedInvalid, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true},
	})
	combinedEntry := exportReportEntry(combinedReport, "gp8.reject.score.staff.capo")
	if len(combinedData) != 0 || combinedErr == nil || combinedEntry == nil {
		t.Fatalf("combined legacy and negative staff edit = %d bytes, %#v, %v", len(combinedData), combinedReport.Entries, combinedErr)
	}
	if combinedEntry.Location.Track != 0 || combinedEntry.Location.Staff != 1 {
		t.Fatalf("combined negative staff capo location = %#v, want track 0 staff 1", combinedEntry.Location)
	}
}

func conformanceStaffCapoSong(t *testing.T, upperCapo, lowerCapo int32) *Song {
	t.Helper()
	song := semanticValidPitchedGP8Song(t)
	song.MeasureHeaders = slices.Clone(song.MeasureHeaders[:1])
	track := &song.Tracks[0]
	track.Name = "Independent staff capos"
	track.CapoFret = upperCapo
	track.Settings.Notation = true
	track.Measures = slices.Clone(track.Measures[:1])
	track.Strings = []GuitarString{{Number: 1, Value: 64}}
	track.Measures[0].Voices = []Voice{{Beats: []Beat{{
		Duration: defaultDuration(), Status: BeatStatusNormal, Dynamics: Forte,
		Notes: []Note{{Value: 3, String: 1, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1}},
	}}}}
	upper := Staff{Measures: track.Measures, Strings: track.Strings, StandardNotationLineCount: 5, CapoFret: upperCapo}
	lowerMeasures := slices.Clone(track.Measures)
	lowerMeasures[0].Clef = MeasureClefBass
	lowerMeasures[0].Voices = []Voice{{Beats: []Beat{{
		Duration: defaultDuration(), Status: BeatStatusNormal, Dynamics: Forte,
		Notes: []Note{{Value: 4, String: 1, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1}},
	}}}}
	lower := Staff{
		Measures: lowerMeasures, Strings: []GuitarString{{Number: 1, Value: 48}},
		StandardNotationLineCount: 5, CapoFret: lowerCapo,
	}
	track.Staves = []Staff{upper, lower}
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	return song
}

func staffCapos(track *Track) []int32 {
	capos := make([]int32, len(track.Staves))
	for index := range track.Staves {
		capos[index] = track.Staves[index].CapoFret
	}
	return capos
}

func staffCapoPitches(track *Track) []int {
	pitches := make([]int, len(track.Staves))
	for staffIndex := range track.Staves {
		staff := &track.Staves[staffIndex]
		note := &staff.Measures[0].Voices[0].Beats[0].Notes[0]
		pitches[staffIndex] = gp8NoteMIDI(track, staff.Strings, note) + int(staff.CapoFret)
	}
	return pitches
}

func conformanceCapoWireTrack(t *testing.T, data []byte) gpifTrack {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var document gpifDocument
	if err := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &document); err != nil {
		t.Fatal(err)
	}
	if len(document.Tracks.Tracks) != 1 {
		t.Fatalf("wire tracks = %d, want 1", len(document.Tracks.Tracks))
	}
	return document.Tracks.Tracks[0]
}

func conformanceParseCapoExport(t *testing.T, data []byte) *Song {
	t.Helper()
	song, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	return song
}
