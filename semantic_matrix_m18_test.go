// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestSemanticMatrixM18LyricScopes(t *testing.T) {
	runSemanticMatrixM18LyricScopes(newSemanticMatrixRun(t))
}

func runSemanticMatrixM18LyricScopes(run *semanticMatrixRun) {
	t := run.t
	song := semanticValidPitchedGP8Song(t)
	song.Lyrics = Lyrics{
		TrackChoice: 1,
		Lines: []LyricLine{
			{Number: 0, StartingMeasure: 0, Text: "score-only one"},
			{Number: 1, StartingMeasure: 2, Text: "score punctuation: [a], don't!"},
			{Number: 2, StartingMeasure: 4, Text: "score 日本語"},
		},
	}
	song.Tracks[0].Lyrics = []TrackLyricLine{
		{Text: "track first", Offset: 0},
		{Text: "", Offset: 2},
		{Text: "track punctuation + 日本語", Offset: 4},
	}
	rest := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	rest.Status = BeatStatusRest
	rest.Notes = nil
	rest.Text = "rest-only text"
	song.Tracks[0].Staves[0].Measures = song.Tracks[0].Measures

	run.Omitted("Song.Lyrics", song.Lyrics, Lyrics{TrackChoice: 1, Lines: []LyricLine{{Number: 0, StartingMeasure: 0, Text: "score-only one"}, {Number: 1, StartingMeasure: 2, Text: "score punctuation: [a], don't!"}, {Number: 2, StartingMeasure: 4, Text: "score 日本語"}}})
	run.Preserved("Lyrics.TrackChoice", song.Lyrics.TrackChoice, uint8(1))
	run.Preserved("Lyrics.Lines", song.Lyrics.Lines, []LyricLine{{Number: 0, StartingMeasure: 0, Text: "score-only one"}, {Number: 1, StartingMeasure: 2, Text: "score punctuation: [a], don't!"}, {Number: 2, StartingMeasure: 4, Text: "score 日本語"}})
	for index, line := range song.Lyrics.Lines {
		run.Preserved("LyricLine.Number", line.Number, uint8(index))
		run.Preserved("LyricLine.StartingMeasure", line.StartingMeasure, []uint16{0, 2, 4}[index])
		run.Preserved("LyricLine.Text", line.Text, []string{"score-only one", "score punctuation: [a], don't!", "score 日本語"}[index])
	}
	run.Preserved("Track.Lyrics", song.Tracks[0].Lyrics, []TrackLyricLine{{Text: "track first", Offset: 0}, {Text: "", Offset: 2}, {Text: "track punctuation + 日本語", Offset: 4}})
	for index, line := range song.Tracks[0].Lyrics {
		run.Preserved("TrackLyricLine.Text", line.Text, []string{"track first", "", "track punctuation + 日本語"}[index])
		run.Preserved("TrackLyricLine.Offset", line.Offset, []int{0, 2, 4}[index])
	}
	run.Field("Beat.Text", rest.Text, "rest-only text")

	snapshot := cloneM18Song(*song)
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	if !reflect.DeepEqual(*song, snapshot) {
		t.Fatal("preflight mutated lyric-bearing song")
	}
	if !hasExportCode(report, "gp8.omit.score-lyrics") {
		t.Fatalf("preflight report = %#v, want score lyric omission", report.Entries)
	}
	for _, entry := range report.Entries {
		if strings.Contains(entry.Code, "track-lyrics") {
			t.Fatalf("track lyrics were reported as a loss: %#v", entry)
		}
	}
	strictData, _, strictErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"unrelated"}}})
	var lossErr *ExportLossError
	if len(strictData) != 0 || !errors.As(strictErr, &lossErr) {
		t.Fatalf("strict score lyric export = %d bytes, %v", len(strictData), strictErr)
	}
	data, exported, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: reportCodes(report)}})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(reportCodes(exported), reportCodes(report)) {
		t.Fatalf("preflight codes = %v, export codes = %v", reportCodes(report), reportCodes(exported))
	}
	wire := extractM18Wire(t, data)
	if len(wire.Tracks) != 1 || wire.Tracks[0].Lyrics == nil {
		t.Fatalf("wire tracks = %#v, want one lyric-bearing track", wire.Tracks)
	}
	run.Wire("gpifTrack.Lyrics", wire.Tracks[0].Lyrics != nil, true)
	run.Wire("gpifLyrics.Dispatched", wire.Tracks[0].Lyrics.Dispatched, true)
	run.Wire("gpifLyrics.Lines", wire.Tracks[0].Lyrics.Lines, []m18WireLyricLine{{Text: "track first", Offset: 0}, {Text: "", Offset: 2}, {Text: "track punctuation + 日本語", Offset: 4}})
	for index, line := range wire.Tracks[0].Lyrics.Lines {
		run.Wire("gpifLyricLine.Text", line.Text, []string{"track first", "", "track punctuation + 日本語"}[index])
		run.Wire("gpifLyricLine.Offset", line.Offset, []int{0, 2, 4}[index])
	}
	wireRestText := ""
	for _, beat := range wire.Beats {
		if beat.FreeText == "rest-only text" {
			wireRestText = beat.FreeText
			break
		}
	}
	if wireRestText == "" {
		t.Fatalf("wire beats = %#v, want rest text", wire.Beats)
	}
	run.Wire("gpifBeat.FreeText", wireRestText, "rest-only text")

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Field("Track.Lyrics", roundTrip.Tracks[0].Lyrics, song.Tracks[0].Lyrics)
	if len(roundTrip.Lyrics.Lines) != 0 || roundTrip.Lyrics.TrackChoice != 0 {
		t.Fatalf("round-trip score lyrics = %#v, want reported omission", roundTrip.Lyrics)
	}
	gotRest := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0]
	run.Field("Beat.Text", gotRest.Text, "rest-only text")
}

func TestSemanticMatrixM18BinaryScoreLyrics(t *testing.T) {
	runSemanticMatrixM18BinaryScoreLyrics(newSemanticMatrixRun(t))
}

func runSemanticMatrixM18BinaryScoreLyrics(run *semanticMatrixRun) {
	t := run.t
	texts := []string{"first", "", "punctuation: [a], don't!", "fourth", "fifth"}
	starts := []int32{0, 2, 4, 6, 8}
	var data bytes.Buffer
	writeM18Int32(t, &data, 2)
	for index, text := range texts {
		writeM18Int32(t, &data, starts[index])
		writeM18Int32(t, &data, int32(len(text)))
		if _, err := data.WriteString(text); err != nil {
			t.Fatal(err)
		}
	}
	lyrics, err := (&Song{}).readLyrics(newCursor(data.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	run.Field("Lyrics.TrackChoice", lyrics.TrackChoice, uint8(2))
	run.Field("Lyrics.Lines", len(lyrics.Lines), 5)
	for index, line := range lyrics.Lines {
		run.Field("LyricLine.Number", line.Number, uint8(index))
		run.Field("LyricLine.StartingMeasure", line.StartingMeasure, uint16(starts[index]))
		run.Field("LyricLine.Text", line.Text, texts[index])
	}
}

func TestSemanticMatrixM18SourceLyricsDispatch(t *testing.T) {
	runSemanticMatrixM18SourceLyricsDispatch(newSemanticMatrixRun(t))
}

func runSemanticMatrixM18SourceLyricsDispatch(run *semanticMatrixRun) {
	for _, dispatched := range []bool{false, true} {
		attribute := "false"
		if dispatched {
			attribute = "true"
		}
		source := strings.Replace(semanticM14ScopedGPIF, `<Name>Piano</Name>`, `<Name>Piano</Name><Lyrics dispatched="`+attribute+`"><Line><Text>source-only</Text><Offset>3</Offset></Line></Lyrics>`, 1)
		wire := decodeM18Wire(run.t, []byte(source))
		run.Wire("gpifLyrics.Dispatched", wire.Tracks[0].Lyrics.Dispatched, dispatched)
		result, err := ParseWithOptions(conformanceGPIFArchive(run.t, source), ParseOptions{})
		if err != nil {
			run.t.Fatal(err)
		}
		run.Field("Track.Lyrics", result.Song.Tracks[0].Lyrics, []TrackLyricLine{{Text: "source-only", Offset: 3}})
		got := slices.ContainsFunc(result.Diagnostics, func(diagnostic ParseDiagnostic) bool {
			return diagnostic.Code == "GPIF.Track.Lyrics.Undispatched" &&
				diagnostic.Kind == ParseDiagnosticLossyProjection &&
				diagnostic.ObjectID == "0" && strings.HasSuffix(diagnostic.SourcePath, "/Lyrics/@dispatched")
		})
		if got != !dispatched {
			run.t.Errorf("undispatched diagnostic present = %v, want %v", got, !dispatched)
		}
		_, strictErr := ParseWithOptions(conformanceGPIFArchive(run.t, source), ParseOptions{Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticLossyProjection}})
		if (strictErr != nil) != !dispatched {
			run.t.Fatalf("strict dispatched=%v error = %v", dispatched, strictErr)
		}
	}
}

type m18WireDocument struct {
	Tracks []m18WireTrack `xml:"Tracks>Track"`
	Beats  []m18WireBeat  `xml:"Beats>Beat"`
}

type m18WireTrack struct {
	Lyrics *m18WireLyrics `xml:"Lyrics"`
}

type m18WireLyrics struct {
	Dispatched bool               `xml:"dispatched,attr"`
	Lines      []m18WireLyricLine `xml:"Line"`
}

type m18WireLyricLine struct {
	Text   string `xml:"Text"`
	Offset int    `xml:"Offset"`
}

type m18WireBeat struct {
	FreeText string `xml:"FreeText"`
}

func extractM18Wire(t *testing.T, data []byte) m18WireDocument {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	return decodeM18Wire(t, readZipMember(t, archive, "Content/score.gpif"))
}

func decodeM18Wire(t *testing.T, data []byte) m18WireDocument {
	t.Helper()
	var document m18WireDocument
	if err := xml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func writeM18Int32(t *testing.T, target *bytes.Buffer, value int32) {
	t.Helper()
	if err := binary.Write(target, binary.LittleEndian, value); err != nil {
		t.Fatal(err)
	}
}

func cloneM18Song(source Song) Song {
	clone := cloneM08Song(source)
	clone.Lyrics.Lines = slices.Clone(source.Lyrics.Lines)
	for trackIndex := range clone.Tracks {
		clone.Tracks[trackIndex].Lyrics = slices.Clone(source.Tracks[trackIndex].Lyrics)
	}
	return clone
}
