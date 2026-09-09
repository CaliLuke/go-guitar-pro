// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestSemanticMatrixM07MasterBars(t *testing.T) {
	runSemanticMatrixM07MasterBars(newSemanticMatrixRun(t))
}

func runSemanticMatrixM07MasterBars(run *semanticMatrixRun) {
	t := run.t
	song := semanticM07Song(t)
	headers := song.MeasureHeaders

	run.Preserved("Song.MeasureHeaders", len(headers), 4)
	run.Preserved("MeasureHeader.Number", []uint16{headers[0].Number, headers[1].Number, headers[2].Number, headers[3].Number}, []uint16{1, 2, 3, 4})
	run.Preserved("MeasureHeader.KeySignature", []KeySignature{headers[0].KeySignature, headers[2].KeySignature}, []KeySignature{{Key: -7}, {Key: 7, IsMinor: true}})
	run.Preserved("KeySignature.Key", []int8{headers[0].KeySignature.Key, headers[2].KeySignature.Key}, []int8{-7, 7})
	run.Preserved("KeySignature.IsMinor", []bool{headers[0].KeySignature.IsMinor, headers[2].KeySignature.IsMinor}, []bool{false, true})
	run.Preserved("MeasureHeader.TimeSignature", []TimeSignature{headers[0].TimeSignature, headers[2].TimeSignature}, []TimeSignature{
		{Numerator: 1, Denominator: Duration{Value: 1, TupletEnters: 1, TupletTimes: 1}, Beams: [4]uint8{1}},
		{Numerator: 7, Denominator: Duration{Value: 8, TupletEnters: 1, TupletTimes: 1}, Beams: [4]uint8{3, 2, 2}},
	})
	run.Preserved("TimeSignature.Numerator", []int8{headers[0].TimeSignature.Numerator, headers[2].TimeSignature.Numerator, headers[3].TimeSignature.Numerator}, []int8{1, 7, 127})
	run.Preserved("TimeSignature.Denominator", []uint16{headers[0].TimeSignature.Denominator.Value, headers[2].TimeSignature.Denominator.Value, headers[3].TimeSignature.Denominator.Value}, []uint16{1, 8, 128})
	run.Omitted("TimeSignature.Beams", []([4]uint8){headers[0].TimeSignature.Beams, headers[2].TimeSignature.Beams}, []([4]uint8){{1, 0, 0, 0}, {3, 2, 2, 0}})
	run.Field("MeasureHeader.Marker", []string{headers[0].Marker.Title, headers[2].Marker.Title}, []string{"A <&>", "Coda Ω"})
	run.Field("Marker.Title", headers[2].Marker.Title, "Coda Ω")
	run.Preserved("MeasureHeader.RepeatOpen", headers[0].RepeatOpen, true)
	run.Preserved("MeasureHeader.RepeatClose", []int8{headers[1].RepeatClose, headers[2].RepeatClose, headers[3].RepeatClose}, []int8{0, 5, 127})
	run.Preserved("MeasureHeader.RepeatAlternative", []uint8{headers[0].RepeatAlternative, headers[2].RepeatAlternative}, []uint8{1, 0b10000001})
	run.Preserved("MeasureHeader.TripletFeel", []TripletFeel{headers[0].TripletFeel, headers[2].TripletFeel}, []TripletFeel{TripletFeelEighth, TripletFeelSixteenth})
	run.Preserved("MeasureHeader.DoubleBar", headers[2].DoubleBar, true)
	run.Omitted("MeasureHeader.Tempo", []int32{headers[0].Tempo, headers[2].Tempo}, []int32{90, 120})
	run.Normalized("Measure.KeySignature", []KeySignature{song.Tracks[0].Measures[0].KeySignature, song.Tracks[0].Measures[2].KeySignature}, []KeySignature{{Key: -7}, {Key: 7, IsMinor: true}})
	run.Normalized("Measure.TimeSignature", []int8{song.Tracks[0].Measures[0].TimeSignature.Numerator, song.Tracks[0].Measures[2].TimeSignature.Numerator}, []int8{1, 7})
	run.Field("Measure.HasDoubleBar", song.Tracks[0].Measures[2].HasDoubleBar, true)

	for value := DirectionSignCoda; value <= DirectionSignDaDoubleCoda; value++ {
		probe := semanticValidPitchedGP8Song(t)
		probe.Tracks[0].Settings.Notation = true
		direction := value
		probe.MeasureHeaders[0].Direction = &direction
		run.Omitted("MeasureHeader.Direction", *probe.MeasureHeaders[0].Direction, value)
		report := PreflightExport(probe, ExportFormatGP8, ExportOptions{})
		if !hasExportCode(report, "gp8.omit.measure-direction") {
			t.Errorf("direction %d report = %#v, want omission", value, report.Entries)
		}
		run.Enum(m07DirectionSignMember(value), hasExportCode(report, "gp8.omit.measure-direction"), true)
	}
	var directionBytes bytes.Buffer
	for value := int16(1); value <= int16(DirectionSignDaDoubleCoda)+1; value++ {
		if err := binary.Write(&directionBytes, binary.LittleEndian, value); err != nil {
			t.Fatal(err)
		}
	}
	signs, fromSigns, err := (&Song{}).readDirections(newCursor(directionBytes.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	for value := DirectionSignCoda; value <= DirectionSignDaDoubleCoda; value++ {
		got := signs[value]
		if value >= DirectionSignDaCapo {
			got = fromSigns[value]
		}
		run.Field("MeasureHeader.Direction", got, int16(value)+1)
	}
	if _, _, truncatedErr := (&Song{}).readDirections(newCursor(directionBytes.Bytes()[:len(directionBytes.Bytes())-1])); truncatedErr == nil {
		t.Fatal("truncated GP5 directions were accepted")
	}

	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	for _, code := range []string{"gp8.omit.measure-direction", "gp8.omit.measure-tempo", "gp8.omit.time-signature-beams"} {
		if !hasExportCode(report, code) {
			t.Errorf("report = %#v, want %s", report.Entries, code)
		}
	}
	strictData, _, strictErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	var lossErr *ExportLossError
	if len(strictData) != 0 || !errors.As(strictErr, &lossErr) {
		t.Fatalf("strict export = %d bytes, %v", len(strictData), strictErr)
	}
	allowed := []string{"gp8.omit.measure-direction", "gp8.omit.measure-tempo", "gp8.omit.time-signature-beams"}
	data, allowedReport, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(reportCodes(report), reportCodes(allowedReport)) {
		t.Fatalf("preflight codes = %v, export codes = %v", reportCodes(report), reportCodes(allowedReport))
	}
	wires := extractM07Wire(t, data)
	if len(wires.masterBars) != 4 || len(wires.clefs) != 4 {
		t.Fatalf("wire master bars/clefs = %d/%d, want 4/4", len(wires.masterBars), len(wires.clefs))
	}
	run.Wire("gpifMasterBar.Key", []m07WireKey{wires.masterBars[0].key, wires.masterBars[2].key}, []m07WireKey{{"Major", -7}, {"Minor", 7}})
	run.Wire("gpifKey.Mode", []string{wires.masterBars[0].key.mode, wires.masterBars[2].key.mode}, []string{"Major", "Minor"})
	run.Wire("gpifKey.AccidentalCount", []int{wires.masterBars[0].key.count, wires.masterBars[2].key.count}, []int{-7, 7})
	run.Wire("gpifMasterBar.Time", []string{wires.masterBars[0].time, wires.masterBars[2].time, wires.masterBars[3].time}, []string{"1/1", "7/8", "127/128"})
	run.Wire("gpifMasterBar.Section", []string{wires.masterBars[0].sectionText, wires.masterBars[2].sectionText}, []string{"A <&>", "Coda Ω"})
	run.Wire("gpifSection.Text", wires.masterBars[2].sectionText, "Coda Ω")
	run.Wire("gpifSection.Letter", wires.masterBars[2].sectionLetter, "")
	run.Wire("gpifMasterBar.Repeat", []m07WireRepeat{wires.masterBars[0].repeat, wires.masterBars[1].repeat, wires.masterBars[2].repeat, wires.masterBars[3].repeat}, []m07WireRepeat{
		{start: "true"}, {end: "true", count: 1}, {end: "true", count: 6}, {end: "true", count: 128},
	})
	run.Wire("gpifRepeat.Start", wires.masterBars[0].repeat.start, "true")
	run.Wire("gpifRepeat.End", wires.masterBars[2].repeat.end, "true")
	run.Wire("gpifRepeat.Count", []int{wires.masterBars[1].repeat.count, wires.masterBars[2].repeat.count, wires.masterBars[3].repeat.count}, []int{1, 6, 128})
	run.Wire("gpifMasterBar.AlternateEndings", []string{wires.masterBars[0].alternateEndings, wires.masterBars[2].alternateEndings}, []string{"1", "1 8"})
	run.Wire("gpifMasterBar.DoubleBar", wires.masterBars[2].doubleBar, true)
	run.Wire("gpifMasterBar.TripletFeel", []string{wires.masterBars[0].tripletFeel, wires.masterBars[2].tripletFeel}, []string{"Triplet8th", "Triplet16th"})
	run.Wire("gpifBar.Clef", wires.clefs, []string{"G2", "F4", "C3", "C4"})

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	variantSong := parseTestFixture(t, "testdata/gp7/triplet-feel.gp")
	tripletFeels := []TripletFeel{
		roundTrip.MeasureHeaders[0].TripletFeel,
		roundTrip.MeasureHeaders[2].TripletFeel,
		variantSong.MeasureHeaders[2].TripletFeel,
		variantSong.MeasureHeaders[3].TripletFeel,
		variantSong.MeasureHeaders[4].TripletFeel,
		variantSong.MeasureHeaders[5].TripletFeel,
	}
	wantTripletFeels := []TripletFeel{
		TripletFeelEighth,
		TripletFeelSixteenth,
		TripletFeelDottedEighth,
		TripletFeelDottedSixteenth,
		TripletFeelScottishEighth,
		TripletFeelScottishSixteenth,
	}
	run.Dispatch("parseGPIFWithContext:mb.TripletFeel", tripletFeels, wantTripletFeels)
	run.Enum("TripletFeel.TripletFeelNone", roundTrip.MeasureHeaders[1].TripletFeel, TripletFeelNone)
	for index, member := range []string{
		"TripletFeel.TripletFeelEighth",
		"TripletFeel.TripletFeelSixteenth",
		"TripletFeel.TripletFeelDottedEighth",
		"TripletFeel.TripletFeelDottedSixteenth",
		"TripletFeel.TripletFeelScottishEighth",
		"TripletFeel.TripletFeelScottishSixteenth",
	} {
		run.Enum(member, tripletFeels[index], wantTripletFeels[index])
	}
	run.Preserved("MeasureHeader.TripletFeel", tripletFeels, wantTripletFeels)
	clefs := []MeasureClef{roundTrip.Tracks[0].Measures[0].Clef, roundTrip.Tracks[0].Measures[1].Clef, roundTrip.Tracks[0].Measures[2].Clef, roundTrip.Tracks[0].Measures[3].Clef}
	run.Preserved("Measure.Clef", clefs, []MeasureClef{MeasureClefTreble, MeasureClefBass, MeasureClefAlto, MeasureClefTenor})
	run.Enum("MeasureClef.MeasureClefTreble", clefs[0], MeasureClefTreble)
	run.Enum("MeasureClef.MeasureClefBass", clefs[1], MeasureClefBass)
	run.Enum("MeasureClef.MeasureClefAlto", clefs[2], MeasureClefAlto)
	run.Enum("MeasureClef.MeasureClefTenor", clefs[3], MeasureClefTenor)
	run.Field("MeasureHeader.Direction", roundTrip.MeasureHeaders[0].Direction, (*DirectionSign)(nil))
	run.Field("MeasureHeader.Tempo", roundTrip.MeasureHeaders[0].Tempo, int32(0))
	run.Field("TimeSignature.Beams", roundTrip.MeasureHeaders[0].TimeSignature.Beams, [4]uint8{2, 2, 2, 2})

	percussion := semanticValidGP8Song(t)
	percussionData, _, err := ExportWithReport(percussion, ExportFormatGP8, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	neutral := extractM07Wire(t, percussionData)
	run.Wire("gpifBar.Clef", neutral.clefs[0], "Neutral")
}

func m07DirectionSignMember(value DirectionSign) string {
	return []string{
		"DirectionSign.DirectionSignCoda",
		"DirectionSign.DirectionSignDoubleCoda",
		"DirectionSign.DirectionSignSegno",
		"DirectionSign.DirectionSignSegnoSegno",
		"DirectionSign.DirectionSignFine",
		"DirectionSign.DirectionSignDaCapo",
		"DirectionSign.DirectionSignDaCapoAlCoda",
		"DirectionSign.DirectionSignDaCapoAlDoubleCoda",
		"DirectionSign.DirectionSignDaCapoAlFine",
		"DirectionSign.DirectionSignDaSegno",
		"DirectionSign.DirectionSignDaSegnoAlCoda",
		"DirectionSign.DirectionSignDaSegnoAlDoubleCoda",
		"DirectionSign.DirectionSignDaSegnoAlFine",
		"DirectionSign.DirectionSignDaSegnoSegno",
		"DirectionSign.DirectionSignDaSegnoSegnoAlCoda",
		"DirectionSign.DirectionSignDaSegnoSegnoAlDoubleCoda",
		"DirectionSign.DirectionSignDaSegnoSegnoAlFine",
		"DirectionSign.DirectionSignDaCoda",
		"DirectionSign.DirectionSignDaDoubleCoda",
	}[int(value)]
}

func TestSemanticMatrixM07AuthorityAndBoundaries(t *testing.T) {
	runSemanticMatrixM07AuthorityAndBoundaries(newSemanticMatrixRun(t))
}

func runSemanticMatrixM07AuthorityAndBoundaries(run *semanticMatrixRun) {
	t := run.t
	conflict := semanticValidPitchedGP8Song(t)
	conflict.Tracks[0].Settings.Notation = true
	conflict.MeasureHeaders[0].KeySignature = KeySignature{Key: -3, IsMinor: true}
	conflict.MeasureHeaders[0].TimeSignature.Numerator = 7
	conflict.MeasureHeaders[0].TimeSignature.Denominator.Value = 8
	conflict.Tracks[0].Measures[0].KeySignature = KeySignature{Key: 4}
	conflict.Tracks[0].Measures[0].TimeSignature.Numerator = 3
	conflict.Tracks[0].Measures[0].TimeSignature.Denominator.Value = 4
	run.Field("Measure.KeySignature", conflict.Tracks[0].Measures[0].KeySignature, KeySignature{Key: 4})
	run.Field("Measure.TimeSignature", []uint16{uint16(conflict.Tracks[0].Measures[0].TimeSignature.Numerator), conflict.Tracks[0].Measures[0].TimeSignature.Denominator.Value}, []uint16{3, 4})
	report := PreflightExport(conflict, ExportFormatGP8, ExportOptions{})
	for _, code := range []string{"gp8.normalize.measure-key-authority", "gp8.normalize.measure-time-authority"} {
		if !hasExportCode(report, code) {
			t.Errorf("conflict report = %#v, want %s", report.Entries, code)
		}
	}
	strictData, _, strictErr := ExportWithReport(conflict, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	var lossErr *ExportLossError
	if len(strictData) != 0 || !errors.As(strictErr, &lossErr) {
		t.Fatalf("strict conflict export = %d bytes, %v", len(strictData), strictErr)
	}
	data, _, err := ExportWithReport(conflict, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.normalize.measure-key-authority", "gp8.normalize.measure-time-authority"}}})
	if err != nil {
		t.Fatal(err)
	}
	conflictWire := extractM07Wire(t, data)
	run.Wire("gpifMasterBar.Key", conflictWire.masterBars[0].key, m07WireKey{mode: "Minor", count: -3})
	run.Wire("gpifMasterBar.Time", conflictWire.masterBars[0].time, "7/8")

	for _, test := range []struct {
		name            string
		time            string
		repeat          string
		ending          string
		wantError       string
		wantNumerator   int8
		wantDenominator uint16
		wantRepeat      int8
		wantEnding      uint8
	}{
		{name: "one", time: "1/1", repeat: "1", ending: "1", wantNumerator: 1, wantDenominator: 1, wantRepeat: 0, wantEnding: 1},
		{name: "four", time: "4/4", repeat: "2", ending: "1 8", wantNumerator: 4, wantDenominator: 4, wantRepeat: 1, wantEnding: 0b10000001},
		{name: "seven", time: "7/8", repeat: "6", wantNumerator: 7, wantDenominator: 8, wantRepeat: 5},
		{name: "maximum", time: "127/128", repeat: "128", wantNumerator: 127, wantDenominator: 128, wantRepeat: 127},
		{name: "zero numerator", time: "0/4", repeat: "2", wantError: "outside 1..127"},
		{name: "negative numerator", time: "-1/4", repeat: "2", wantError: "outside 1..127"},
		{name: "numerator overflow", time: "128/4", repeat: "2", wantError: "outside 1..127"},
		{name: "numerator modulo", time: "260/4", repeat: "2", wantError: "outside 1..127"},
		{name: "zero denominator", time: "4/0", repeat: "2", wantError: "outside 1..65535"},
		{name: "non-note denominator", time: "4/3", repeat: "2", wantError: "unsupported note value 3"},
		{name: "denominator modulo", time: "4/65540", repeat: "2", wantError: "outside 1..65535"},
		{name: "repeat overflow 129", time: "4/4", repeat: "129", wantError: "outside 1..128"},
		{name: "repeat overflow 130", time: "4/4", repeat: "130", wantError: "outside 1..128"},
		{name: "ending zero", time: "4/4", repeat: "2", ending: "0", wantError: "outside 1..8"},
		{name: "ending nine", time: "4/4", repeat: "2", ending: "9", wantError: "outside 1..8"},
		{name: "ending text", time: "4/4", repeat: "2", ending: "x", wantError: "outside 1..8"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := m07GPIF(test.time, test.repeat, test.ending)
			result, err := ParseWithOptions(conformanceGPIFArchive(t, source), ParseOptions{Strict: true})
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) {
					t.Fatalf("parse error = %v, want %q", err, test.wantError)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			header := result.Song.MeasureHeaders[0]
			run.Field("TimeSignature.Numerator", header.TimeSignature.Numerator, test.wantNumerator)
			run.Field("TimeSignature.Denominator", header.TimeSignature.Denominator.Value, test.wantDenominator)
			run.Field("MeasureHeader.RepeatClose", header.RepeatClose, test.wantRepeat)
			run.Field("MeasureHeader.RepeatAlternative", header.RepeatAlternative, test.wantEnding)
		})
	}

	for _, invalid := range []struct {
		name   string
		mutate func(*Song)
		want   string
	}{
		{name: "negative repeat", mutate: func(song *Song) { song.MeasureHeaders[0].RepeatClose = -2 }, want: "repeat-close"},
		{name: "denominator three", mutate: func(song *Song) { song.MeasureHeaders[0].TimeSignature.Denominator.Value = 3 }, want: "unsupported note value 3"},
		{name: "zero numerator", mutate: func(song *Song) { song.MeasureHeaders[0].TimeSignature.Numerator = 0 }, want: "numerator 0 must be positive"},
		{name: "undefined direction", mutate: func(song *Song) { value := DirectionSign(99); song.MeasureHeaders[0].Direction = &value }, want: "direction 99 is not defined"},
		{name: "undefined triplet feel", mutate: func(song *Song) { song.MeasureHeaders[0].TripletFeel = TripletFeel(9) }, want: "triplet feel 9 is not defined"},
	} {
		t.Run(invalid.name, func(t *testing.T) {
			song := semanticValidPitchedGP8Song(t)
			song.Tracks[0].Settings.Notation = true
			invalid.mutate(song)
			report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
			if !slices.ContainsFunc(report.Entries, func(entry ExportReportEntry) bool {
				return entry.Disposition == ExportDispositionRejected && strings.Contains(entry.Reason, invalid.want)
			}) {
				t.Fatalf("preflight report = %#v, want rejected %q", report.Entries, invalid.want)
			}
			_, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.reject.score"}}})
			if err == nil || !strings.Contains(err.Error(), invalid.want) {
				t.Fatalf("export error = %v, want %q", err, invalid.want)
			}
		})
	}
}

func semanticM07Song(t *testing.T) *Song {
	t.Helper()
	song := semanticValidPitchedGP8Song(t)
	song.Tracks[0].Settings.Notation = true
	song.MeasureHeaders = append(song.MeasureHeaders, defaultMeasureHeader())
	track := &song.Tracks[0]
	track.Measures = append(track.Measures, Measure{Voices: []Voice{{Beats: []Beat{{Duration: defaultDuration(), Status: BeatStatusRest}}}}})
	track.Staves[0].Measures = track.Measures
	for index := range song.MeasureHeaders {
		header := &song.MeasureHeaders[index]
		header.Number = uint16(index + 1)
		measure := &track.Measures[index]
		measure.Number = index + 1
		measure.HeaderIndex = index
		measure.TrackIndex = 0
		measure.StaffIndex = 0
		for voiceIndex := range measure.Voices {
			measure.Voices[voiceIndex].MeasureIndex = int16(index)
		}
	}
	directions := []DirectionSign{DirectionSignCoda, DirectionSignDaCapo, DirectionSignDaDoubleCoda, DirectionSignSegno}
	values := []struct {
		time        TimeSignature
		key         KeySignature
		marker      string
		repeatOpen  bool
		repeatClose int8
		ending      uint8
		triplet     TripletFeel
		doubleBar   bool
		tempo       int32
		clef        MeasureClef
	}{
		{TimeSignature{Numerator: 1, Denominator: Duration{Value: 1, TupletEnters: 1, TupletTimes: 1}, Beams: [4]uint8{1}}, KeySignature{Key: -7}, "A <&>", true, -1, 1, TripletFeelEighth, false, 90, MeasureClefTreble},
		{defaultTimeSignature(), KeySignature{}, "", false, 0, 0, TripletFeelNone, false, 0, MeasureClefBass},
		{TimeSignature{Numerator: 7, Denominator: Duration{Value: 8, TupletEnters: 1, TupletTimes: 1}, Beams: [4]uint8{3, 2, 2}}, KeySignature{Key: 7, IsMinor: true}, "Coda Ω", false, 5, 0b10000001, TripletFeelSixteenth, true, 120, MeasureClefAlto},
		{TimeSignature{Numerator: 127, Denominator: Duration{Value: 128, TupletEnters: 1, TupletTimes: 1}, Beams: [4]uint8{2, 2, 2, 2}}, KeySignature{Key: -1, IsMinor: true}, "", false, 127, 0, TripletFeelNone, false, 0, MeasureClefTenor},
	}
	for index, value := range values {
		header := &song.MeasureHeaders[index]
		header.TimeSignature = value.time
		header.KeySignature = value.key
		if value.marker != "" {
			header.Marker = &Marker{Title: value.marker}
		} else {
			header.Marker = nil
		}
		header.Direction = &directions[index]
		header.RepeatOpen = value.repeatOpen
		header.RepeatClose = value.repeatClose
		header.RepeatAlternative = value.ending
		header.TripletFeel = value.triplet
		header.DoubleBar = value.doubleBar
		header.Tempo = value.tempo
		measure := &track.Measures[index]
		measure.TimeSignature = value.time
		measure.KeySignature = value.key
		measure.HasDoubleBar = value.doubleBar
		measure.Clef = value.clef
	}
	track.Staves[0].Measures = track.Measures
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("M07 baseline diagnostics = %#v", diagnostics)
	}
	return song
}

func hasExportCode(report ExportReport, code string) bool {
	return slices.ContainsFunc(report.Entries, func(entry ExportReportEntry) bool { return entry.Code == code })
}

func reportCodes(report ExportReport) []string {
	codes := make([]string, len(report.Entries))
	for index, entry := range report.Entries {
		codes[index] = entry.Code
	}
	return codes
}

type m07WireKey struct {
	mode  string
	count int
}

type m07WireRepeat struct {
	start string
	end   string
	count int
}

type m07WireMasterBar struct {
	key              m07WireKey
	time             string
	sectionLetter    string
	sectionText      string
	repeat           m07WireRepeat
	alternateEndings string
	doubleBar        bool
	tripletFeel      string
}

type m07WireScore struct {
	masterBars []m07WireMasterBar
	clefs      []string
}

func extractM07Wire(t *testing.T, data []byte) m07WireScore {
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
	var result m07WireScore
	var path []string
	for {
		token, decodeErr := decoder.Token()
		if errors.Is(decodeErr, io.EOF) {
			break
		}
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		switch value := token.(type) {
		case xml.StartElement:
			path = append(path, value.Name.Local)
			if strings.Join(path, "/") == "GPIF/MasterBars/MasterBar" {
				result.masterBars = append(result.masterBars, m07WireMasterBar{})
			}
			if value.Name.Local == "Repeat" && len(result.masterBars) != 0 {
				repeat := &result.masterBars[len(result.masterBars)-1].repeat
				for _, attribute := range value.Attr {
					switch attribute.Name.Local {
					case "start":
						repeat.start = attribute.Value
					case "end":
						repeat.end = attribute.Value
					case "count":
						repeat.count, _ = strconv.Atoi(attribute.Value)
					}
				}
			}
			if value.Name.Local == "DoubleBar" && len(result.masterBars) != 0 {
				result.masterBars[len(result.masterBars)-1].doubleBar = true
			}
		case xml.CharData:
			text := strings.TrimSpace(string(value))
			if text == "" {
				continue
			}
			joined := strings.Join(path, "/")
			if joined == "GPIF/Bars/Bar/Clef" {
				result.clefs = append(result.clefs, text)
				continue
			}
			if len(result.masterBars) == 0 {
				continue
			}
			bar := &result.masterBars[len(result.masterBars)-1]
			switch joined {
			case "GPIF/MasterBars/MasterBar/Key/Mode":
				bar.key.mode = text
			case "GPIF/MasterBars/MasterBar/Key/AccidentalCount":
				bar.key.count, _ = strconv.Atoi(text)
			case "GPIF/MasterBars/MasterBar/Time":
				bar.time = text
			case "GPIF/MasterBars/MasterBar/Section/Letter":
				bar.sectionLetter = text
			case "GPIF/MasterBars/MasterBar/Section/Text":
				bar.sectionText = text
			case "GPIF/MasterBars/MasterBar/AlternateEndings":
				bar.alternateEndings = text
			case "GPIF/MasterBars/MasterBar/TripletFeel":
				bar.tripletFeel = text
			}
		case xml.EndElement:
			path = path[:len(path)-1]
		}
	}
	return result
}

func m07GPIF(time, repeat, endings string) string {
	repeatXML := ""
	if repeat != "" {
		repeatXML = fmt.Sprintf(`<Repeat end="true" count="%s"/>`, repeat)
	}
	endingXML := ""
	if endings != "" {
		endingXML = "<AlternateEndings>" + endings + "</AlternateEndings>"
	}
	masterBar := fmt.Sprintf(`<MasterBars><MasterBar><Key><Mode>Major</Mode><AccidentalCount>-3</AccidentalCount></Key><Time>%s</Time>%s%s<Bars>0 1</Bars><TripletFeel>Triplet8th</TripletFeel></MasterBar></MasterBars>`, time, repeatXML, endingXML)
	return strings.Replace(staffScopedChordGPIF, `<MasterBars><MasterBar><Time>4/4</Time><Bars>0 1</Bars></MasterBar></MasterBars>`, masterBar, 1)
}
