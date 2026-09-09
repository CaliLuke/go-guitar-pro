// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"math"
	"reflect"
	"slices"
	"strconv"
	"testing"
)

func TestSemanticMatrixM08Durations(t *testing.T) {
	runSemanticMatrixM08Durations(newSemanticMatrixRun(t))
}

func runSemanticMatrixM08Durations(run *semanticMatrixRun) {
	t := run.t
	for _, tuplet := range []struct {
		value         byte
		enters, times uint8
	}{
		{3, 3, 2}, {5, 5, 4}, {6, 6, 4}, {7, 7, 4}, {9, 9, 8}, {10, 10, 8}, {11, 11, 8}, {12, 12, 8}, {13, 13, 8},
	} {
		duration, err := readDuration(newCursor([]byte{2, tuplet.value, 0, 0, 0}), 0x20)
		if err != nil {
			t.Fatal(err)
		}
		run.Dispatch("readDuration:iTuplet", [2]uint8{duration.TupletEnters, duration.TupletTimes}, [2]uint8{tuplet.enters, tuplet.times})
	}
	for dots := 0; dots <= 2; dots++ {
		duration, err := gpifRhythmToDuration(&gpifRhythm{NoteValue: "Quarter", AugmentationDot: &gpifAugDot{Count: dots}})
		if err != nil {
			t.Fatal(err)
		}
		run.Dispatch("gpifRhythmToDuration:r.AugmentationDot.Count", [2]bool{duration.Dotted, duration.DoubleDotted}, [2]bool{dots == 1, dots == 2})
		exact, err := (MusicalDuration{Value: 4, Dots: DotCount(dots), Tuplet: TupletRatio{Enters: 1, Times: 1}}).ExactScoreTime()
		if err != nil {
			t.Fatal(err)
		}
		run.Dispatch("ExactScoreTime:validated.Dots", exact.Compare(ScoreTime{}) > 0, true)
	}
	unknownTupletContext := &parseContext{format: "GP5"}
	if _, err := readDuration(newCursorWithContext([]byte{2, 4, 0, 0, 0}, unknownTupletContext), 0x20); err != nil {
		t.Fatal(err)
	}
	if diagnostic := m20DiagnosticByCode(unknownTupletContext.diagnostics, "Binary.Duration.Tuplet.Unsupported"); diagnostic == nil || diagnostic.Kind != ParseDiagnosticUnsupportedFeature {
		t.Fatalf("unknown binary tuplet diagnostics = %#v", unknownTupletContext.diagnostics)
	}
	noteValues := []struct {
		value uint16
		wire  string
	}{{1, "Whole"}, {2, "Half"}, {4, "Quarter"}, {8, "Eighth"}, {16, "16th"}, {32, "32nd"}, {64, "64th"}, {128, "128th"}}
	for _, noteValue := range noteValues {
		decoded, err := gpifRhythmToDuration(&gpifRhythm{NoteValue: noteValue.wire})
		if err != nil {
			t.Fatal(err)
		}
		run.Dispatch("gpifRhythmToDuration:r.NoteValue", decoded.Value, noteValue.value)
		for dots := DotCount(0); dots <= 2; dots++ {
			semantic, err := NewMusicalDuration(NoteValue(noteValue.value), dots, TupletRatio{Enters: 1, Times: 1})
			if err != nil {
				t.Fatal(err)
			}
			duration, err := semantic.LegacyDuration()
			if err != nil {
				t.Fatal(err)
			}
			run.Preserved("Duration.Value", duration.Value, noteValue.value)
			run.Normalized("Duration.Dotted", duration.Dotted, dots == 1)
			run.Normalized("Duration.DoubleDotted", duration.DoubleDotted, dots == 2)
			run.Normalized("Duration.TupletEnters", duration.TupletEnters, uint8(1))
			run.Normalized("Duration.TupletTimes", duration.TupletTimes, uint8(1))

			song := semanticM08SingleBeatSong(t, duration)
			data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
			if err != nil {
				t.Fatal(err)
			}
			wire := extractM08Rhythm(t, data)
			run.Wire("gpifRhythm.NoteValue", wire.noteValue, noteValue.wire)
			wantDots := 0
			if dots != 0 {
				wantDots = int(dots)
			}
			run.Wire("gpifRhythm.AugmentationDot", wire.dotCount, wantDots)
			run.Wire("gpifAugDot.Count", wire.dotCount, wantDots)
			roundTrip, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Duration
			run.Preserved("Beat.Duration", got, duration)
		}
	}

	for _, ratio := range []TupletRatio{{Enters: 1, Times: 1}, {Enters: 3, Times: 2}, {Enters: 5, Times: 4}, {Enters: 7, Times: 4}, {Enters: 255, Times: 254}} {
		semantic, err := NewMusicalDuration(4, 0, ratio)
		if err != nil {
			t.Fatal(err)
		}
		duration, err := semantic.LegacyDuration()
		if err != nil {
			t.Fatal(err)
		}
		song := semanticM08SingleBeatSong(t, duration)
		data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
		if err != nil {
			t.Fatal(err)
		}
		wire := extractM08Rhythm(t, data)
		wantNum, wantDen := 0, 0
		if ratio != (TupletRatio{Enters: 1, Times: 1}) {
			wantNum = int(ratio.Enters)
			wantDen = int(ratio.Times)
		}
		run.Wire("gpifRhythm.PrimaryTuplet", [2]int{wire.tupletNum, wire.tupletDen}, [2]int{wantNum, wantDen})
		run.Wire("gpifTuplet.Num", wire.tupletNum, wantNum)
		run.Wire("gpifTuplet.Den", wire.tupletDen, wantDen)
	}

	for _, invalid := range []MusicalDuration{
		{Value: 3, Tuplet: TupletRatio{Enters: 1, Times: 1}},
		{Value: 4, Dots: 3, Tuplet: TupletRatio{Enters: 1, Times: 1}},
		{Value: 4, Tuplet: TupletRatio{Enters: 3, Times: 0}},
		{Value: 4, Tuplet: TupletRatio{Enters: 0, Times: 2}},
	} {
		if _, err := invalid.ExactScoreTime(); err == nil {
			t.Errorf("invalid duration accepted: %#v", invalid)
		}
	}
	if _, err := (MusicalDuration{Value: 4, Tuplet: TupletRatio{Enters: 256, Times: 255}}).LegacyDuration(); err == nil {
		t.Fatal("legacy duration accepted a ratio outside uint8")
	}

	for _, normalized := range []struct {
		name     string
		duration Duration
		code     string
	}{
		{name: "simultaneous dots", duration: Duration{Value: 4, Dotted: true, DoubleDotted: true, TupletEnters: 1, TupletTimes: 1}, code: "gp8.normalize.duration-dot-flags"},
		{name: "zero default tuplet", duration: Duration{Value: 4}, code: "gp8.normalize.tuplet-default"},
	} {
		t.Run(normalized.name, func(t *testing.T) {
			song := semanticM08SingleBeatSong(t, normalized.duration)
			report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
			if !hasExportCode(report, normalized.code) {
				t.Fatalf("report = %#v, want %s", report.Entries, normalized.code)
			}
			data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
			var lossErr *ExportLossError
			if len(data) != 0 || !errors.As(err, &lossErr) {
				t.Fatalf("strict export = %d bytes, %v", len(data), err)
			}
		})
	}
}

func TestSemanticMatrixM08ExactTiming(t *testing.T) {
	runSemanticMatrixM08ExactTiming(newSemanticMatrixRun(t))
}

func runSemanticMatrixM08ExactTiming(run *semanticMatrixRun) {
	t := run.t
	septuplet := defaultBeat()
	septuplet.Status = BeatStatusRest
	septuplet.Duration.TupletEnters = 7
	septuplet.Duration.TupletTimes = 4
	beats := make([]Beat, 7)
	for index := range beats {
		beats[index] = septuplet
	}
	secondVoice := defaultBeat()
	secondVoice.Status = BeatStatusRest
	wholeRest := defaultBeat()
	wholeRest.Status = BeatStatusRest
	wholeRest.Duration.Value = 1
	song := Song{
		Anacrusis:      true,
		MeasureHeaders: []MeasureHeader{defaultMeasureHeader(), defaultMeasureHeader(), defaultMeasureHeader()},
		Tracks: []Track{{ChannelIndex: -1, Measures: []Measure{
			{HeaderIndex: 0, Voices: []Voice{{Beats: beats}, {Beats: []Beat{secondVoice}}}},
			{HeaderIndex: 1, Voices: []Voice{{}}},
			{HeaderIndex: 2, Voices: []Voice{{Beats: []Beat{wholeRest}}}},
		}}},
	}
	if err := FinalizeSong(&song); err != nil {
		t.Fatal(err)
	}
	run.Preserved("Song.Anacrusis", song.Anacrusis, true)
	run.Derived("MeasureHeader.Start", []int64{song.MeasureHeaders[0].Start, song.MeasureHeaders[1].Start, song.MeasureHeaders[2].Start}, []int64{960, 4800, 8640})
	run.Derived("MeasureHeader.ExactStart", m08Times(song.MeasureHeaders), [][2]int64{{960, 1}, {4800, 1}, {8640, 1}})
	measures := song.Tracks[0].Measures
	run.Derived("Measure.Start", []int64{measures[0].Start, measures[1].Start, measures[2].Start}, []int64{960, 4800, 8640})
	run.Derived("Measure.ExactStart", m08MeasureTimes(measures), [][2]int64{{960, 1}, {4800, 1}, {8640, 1}})
	gotBeats := measures[0].Voices[0].Beats
	wantFloor := []int64{960, 1508, 2057, 2605, 3154, 3702, 4251}
	wantExact := [][2]int64{{960, 1}, {10560, 7}, {14400, 7}, {18240, 7}, {22080, 7}, {25920, 7}, {29760, 7}}
	for index := range gotBeats {
		run.Derived("Beat.Start", *gotBeats[index].Start, wantFloor[index])
		run.Derived("Beat.ExactStart", [2]int64{gotBeats[index].ExactStart.Numerator(), gotBeats[index].ExactStart.Denominator()}, wantExact[index])
	}
	if *measures[0].Voices[1].Beats[0].Start != measures[0].Start {
		t.Fatal("second voice did not start at the measure origin")
	}

	snapshot := cloneM08Song(song)
	if err := FinalizeSong(&song); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(song, snapshot) {
		t.Fatal("second finalization changed the deep score snapshot")
	}

	zero := ScoreTime{}
	oneThird, err := NewScoreTime(1, 3)
	if err != nil {
		t.Fatal(err)
	}
	twoThirds, err := oneThird.Add(oneThird)
	if err != nil || twoThirds.Numerator() != 2 || twoThirds.Denominator() != 3 || zero.Compare(oneThird) >= 0 {
		t.Fatalf("score-time arithmetic = %d/%d, %v", twoThirds.Numerator(), twoThirds.Denominator(), err)
	}
	if _, err := oneThird.Subtract(twoThirds); err == nil {
		t.Fatal("negative score-time result was accepted")
	}
	maximum, _ := NewScoreTime(math.MaxInt64, 1)
	if _, err := maximum.Add(mustScoreTime(t, 1, 1)); err == nil {
		t.Fatal("overflowing score-time sum was accepted")
	}
	if _, err := maximum.Multiply(2); err == nil {
		t.Fatal("overflowing score-time product was accepted")
	}
	for _, terms := range [][2]int64{{0, 1}, {1, 2}, {1, 1}} {
		if _, err := NewBarPosition(terms[0], terms[1]); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range []float64{-1, 1.1, math.NaN(), math.Inf(1)} {
		if _, err := NewBarPositionFromFloat64(value); err == nil {
			t.Errorf("bar position accepted %v", value)
		}
	}
}

func semanticM08SingleBeatSong(t *testing.T, duration Duration) *Song {
	t.Helper()
	song := semanticValidPitchedGP8Song(t)
	song.Tracks[0].Settings.Notation = true
	song.MeasureHeaders = slices.Clone(song.MeasureHeaders[:1])
	song.Tracks[0].Measures = slices.Clone(song.Tracks[0].Measures[:1])
	beat := Beat{Duration: duration, Status: BeatStatusRest}
	song.Tracks[0].Measures[0].Voices = []Voice{{Beats: []Beat{beat}}}
	song.Tracks[0].Staves[0].Measures = song.Tracks[0].Measures
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	return song
}

func m08Times(headers []MeasureHeader) [][2]int64 {
	result := make([][2]int64, len(headers))
	for index := range headers {
		result[index] = [2]int64{headers[index].ExactStart.Numerator(), headers[index].ExactStart.Denominator()}
	}
	return result
}

func m08MeasureTimes(measures []Measure) [][2]int64 {
	result := make([][2]int64, len(measures))
	for index := range measures {
		result[index] = [2]int64{measures[index].ExactStart.Numerator(), measures[index].ExactStart.Denominator()}
	}
	return result
}

func cloneM08Song(source Song) Song {
	clone := source
	clone.MeasureHeaders = slices.Clone(source.MeasureHeaders)
	clone.Tracks = slices.Clone(source.Tracks)
	for trackIndex := range clone.Tracks {
		clone.Tracks[trackIndex].Measures = cloneM08Measures(source.Tracks[trackIndex].Measures)
		clone.Tracks[trackIndex].Staves = slices.Clone(source.Tracks[trackIndex].Staves)
		for staffIndex := range clone.Tracks[trackIndex].Staves {
			clone.Tracks[trackIndex].Staves[staffIndex].Measures = cloneM08Measures(source.Tracks[trackIndex].Staves[staffIndex].Measures)
			clone.Tracks[trackIndex].Staves[staffIndex].Strings = slices.Clone(source.Tracks[trackIndex].Staves[staffIndex].Strings)
		}
	}
	return clone
}

func cloneM08Measures(source []Measure) []Measure {
	clone := slices.Clone(source)
	for measureIndex := range clone {
		measure := &clone[measureIndex]
		measure.Voices = slices.Clone(source[measureIndex].Voices)
		for voiceIndex := range measure.Voices {
			measure.Voices[voiceIndex].Beats = slices.Clone(source[measureIndex].Voices[voiceIndex].Beats)
			for beatIndex := range measure.Voices[voiceIndex].Beats {
				beat := &measure.Voices[voiceIndex].Beats[beatIndex]
				if beat.Start != nil {
					value := *beat.Start
					beat.Start = &value
				}
				if beat.ExactStart != nil {
					value := *beat.ExactStart
					beat.ExactStart = &value
				}
			}
		}
	}
	return clone
}

type m08WireRhythm struct {
	noteValue string
	dotCount  int
	tupletNum int
	tupletDen int
}

func extractM08Rhythm(t *testing.T, data []byte) m08WireRhythm {
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
	var result m08WireRhythm
	inRhythms := false
	inFirstRhythm := false
	for {
		token, decodeErr := decoder.Token()
		if errors.Is(decodeErr, io.EOF) {
			return result
		}
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		switch value := token.(type) {
		case xml.StartElement:
			if value.Name.Local == "Rhythms" {
				inRhythms = true
			}
			if value.Name.Local == "Rhythm" && inRhythms && result.noteValue == "" {
				inFirstRhythm = true
			}
			if !inFirstRhythm {
				continue
			}
			for _, attribute := range value.Attr {
				number, _ := strconv.Atoi(attribute.Value)
				switch {
				case value.Name.Local == "AugmentationDot" && attribute.Name.Local == "count":
					result.dotCount = number
				case value.Name.Local == "PrimaryTuplet" && attribute.Name.Local == "num":
					result.tupletNum = number
				case value.Name.Local == "PrimaryTuplet" && attribute.Name.Local == "den":
					result.tupletDen = number
				}
			}
			if value.Name.Local == "NoteValue" {
				if decodeErr := decoder.DecodeElement(&result.noteValue, &value); decodeErr != nil {
					t.Fatal(decodeErr)
				}
			}
		case xml.EndElement:
			if value.Name.Local == "Rhythm" && inFirstRhythm {
				return result
			}
			if value.Name.Local == "Rhythms" {
				inRhythms = false
			}
		}
	}
}
