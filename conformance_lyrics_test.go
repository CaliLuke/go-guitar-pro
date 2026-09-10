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

func TestConformanceLyricScopes(t *testing.T) {
	runConformanceLyricScopes(newConformanceRun(t))
}

func runConformanceLyricScopes(run *conformanceRun) {
	t := run.t
	song := semanticValidPitchedGP8Song(t)
	song.Lyrics = Lyrics{
		TrackIndex: 0,
		Lines: []LyricLine{
			{StartMeasureIndex: 0, Text: "score-only one"},
			{StartMeasureIndex: 0, Text: "score punctuation: [a], don't!"},
			{StartMeasureIndex: 0, Text: "score 日本語"},
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

	run.Omitted("Song.Lyrics", song.Lyrics, Lyrics{TrackIndex: 0, Lines: []LyricLine{{StartMeasureIndex: 0, Text: "score-only one"}, {StartMeasureIndex: 0, Text: "score punctuation: [a], don't!"}, {StartMeasureIndex: 0, Text: "score 日本語"}}})
	run.Preserved("Lyrics.TrackIndex", song.Lyrics.TrackIndex, 0)
	run.Preserved("Lyrics.Lines", song.Lyrics.Lines, []LyricLine{{StartMeasureIndex: 0, Text: "score-only one"}, {StartMeasureIndex: 0, Text: "score punctuation: [a], don't!"}, {StartMeasureIndex: 0, Text: "score 日本語"}})
	for index, line := range song.Lyrics.Lines {
		run.Preserved("LyricLine.StartMeasureIndex", line.StartMeasureIndex, 0)
		run.Preserved("LyricLine.Text", line.Text, []string{"score-only one", "score punctuation: [a], don't!", "score 日本語"}[index])
	}
	run.Preserved("Track.Lyrics", song.Tracks[0].Lyrics, []TrackLyricLine{{Text: "track first", Offset: 0}, {Text: "", Offset: 2}, {Text: "track punctuation + 日本語", Offset: 4}})
	for index, line := range song.Tracks[0].Lyrics {
		run.Preserved("TrackLyricLine.Text", line.Text, []string{"track first", "", "track punctuation + 日本語"}[index])
		run.Preserved("TrackLyricLine.Offset", line.Offset, []int{0, 2, 4}[index])
	}
	run.Field("Beat.Text", rest.Text, "rest-only text")

	snapshot := cloneLyricSong(*song)
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
	wire := extractLyricWire(t, data)
	if len(wire.Tracks) != 1 || wire.Tracks[0].Lyrics == nil {
		t.Fatalf("wire tracks = %#v, want one lyric-bearing track", wire.Tracks)
	}
	run.Wire("gpifTrack.Lyrics", wire.Tracks[0].Lyrics != nil, true)
	run.Wire("gpifLyrics.Dispatched", wire.Tracks[0].Lyrics.Dispatched, true)
	run.Wire("gpifLyrics.Lines", wire.Tracks[0].Lyrics.Lines, []conformanceLyricWireLyricLine{{Text: "track first", Offset: 0}, {Text: "", Offset: 2}, {Text: "track punctuation + 日本語", Offset: 4}})
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
	if len(roundTrip.Lyrics.Lines) != 0 {
		t.Fatalf("round-trip score lyrics = %#v, want reported omission", roundTrip.Lyrics)
	}
	gotRest := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0]
	run.Field("Beat.Text", gotRest.Text, "rest-only text")
}

func TestConformanceBinaryScoreLyrics(t *testing.T) {
	runConformanceBinaryScoreLyrics(newConformanceRun(t))
}

func runConformanceBinaryScoreLyrics(run *conformanceRun) {
	t := run.t
	texts := []string{"first", "", "punctuation: [a], don't!", "fourth", "fifth"}
	wireStarts := []int32{1, 3, 5, 7, 9}
	wantStarts := []int{0, 2, 4, 6, 8}
	var data bytes.Buffer
	writeLyricInt32(t, &data, 2)
	for index, text := range texts {
		writeLyricInt32(t, &data, wireStarts[index])
		writeLyricInt32(t, &data, int32(len(text)))
		if _, err := data.WriteString(text); err != nil {
			t.Fatal(err)
		}
	}
	lyrics, err := (&Song{}).readLyrics(newCursor(data.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	run.Field("Lyrics.TrackIndex", lyrics.TrackIndex, 1)
	run.Field("Lyrics.Lines", len(lyrics.Lines), 5)
	for index, line := range lyrics.Lines {
		run.Field("LyricLine.StartMeasureIndex", line.StartMeasureIndex, wantStarts[index])
		run.Field("LyricLine.Text", line.Text, texts[index])
	}
}

func TestBinaryScoreLyricsPreserveUnassignedSentinels(t *testing.T) {
	var data bytes.Buffer
	writeLyricInt32(t, &data, 0)
	for range 5 {
		writeLyricInt32(t, &data, 0)
		writeLyricInt32(t, &data, 0)
	}
	lyrics, err := (&Song{}).readLyrics(newCursor(data.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if lyrics.TrackIndex != -1 {
		t.Fatalf("track index = %d, want -1", lyrics.TrackIndex)
	}
	for index, line := range lyrics.Lines {
		if line.StartMeasureIndex != -1 {
			t.Errorf("line %d start measure index = %d, want -1", index, line.StartMeasureIndex)
		}
	}
}

func TestBinaryScoreLyricsRejectNegativeWireIndexes(t *testing.T) {
	var negativeTrack bytes.Buffer
	writeLyricInt32(t, &negativeTrack, -1)
	if _, err := (&Song{}).readLyrics(newCursor(negativeTrack.Bytes())); err == nil || !strings.Contains(err.Error(), "track choice -1") {
		t.Fatalf("negative track error = %v", err)
	}

	var negativeMeasure bytes.Buffer
	writeLyricInt32(t, &negativeMeasure, 1)
	writeLyricInt32(t, &negativeMeasure, -1)
	if _, err := (&Song{}).readLyrics(newCursor(negativeMeasure.Bytes())); err == nil || !strings.Contains(err.Error(), "starting measure -1") {
		t.Fatalf("negative measure error = %v", err)
	}
}

func TestConformanceSourceLyricsDispatch(t *testing.T) {
	runConformanceSourceLyricsDispatch(newConformanceRun(t))
}

func runConformanceSourceLyricsDispatch(run *conformanceRun) {
	for _, dispatched := range []bool{false, true} {
		attribute := "false"
		if dispatched {
			attribute = "true"
		}
		source := strings.Replace(conformanceChordScopedGPIF, `<Name>Piano</Name>`, `<Name>Piano</Name><Lyrics dispatched="`+attribute+`"><Line><Text>source-only</Text><Offset>3</Offset></Line></Lyrics>`, 1)
		wire := decodeLyricWire(run.t, []byte(source))
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

type conformanceLyricWireDocument struct {
	Tracks []conformanceLyricWireTrack `xml:"Tracks>Track"`
	Beats  []conformanceLyricWireBeat  `xml:"Beats>Beat"`
}

type conformanceLyricWireTrack struct {
	Lyrics *conformanceLyricWireLyrics `xml:"Lyrics"`
}

type conformanceLyricWireLyrics struct {
	Dispatched bool                            `xml:"dispatched,attr"`
	Lines      []conformanceLyricWireLyricLine `xml:"Line"`
}

type conformanceLyricWireLyricLine struct {
	Text   string `xml:"Text"`
	Offset int    `xml:"Offset"`
}

type conformanceLyricWireBeat struct {
	FreeText string `xml:"FreeText"`
}

func extractLyricWire(t *testing.T, data []byte) conformanceLyricWireDocument {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	return decodeLyricWire(t, readZipMember(t, archive, "Content/score.gpif"))
}

func decodeLyricWire(t *testing.T, data []byte) conformanceLyricWireDocument {
	t.Helper()
	var document conformanceLyricWireDocument
	if err := xml.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	return document
}

func writeLyricInt32(t *testing.T, target *bytes.Buffer, value int32) {
	t.Helper()
	if err := binary.Write(target, binary.LittleEndian, value); err != nil {
		t.Fatal(err)
	}
}

func cloneLyricSong(source Song) Song {
	clone := cloneTimingSong(source)
	clone.Lyrics.Lines = slices.Clone(source.Lyrics.Lines)
	for trackIndex := range clone.Tracks {
		clone.Tracks[trackIndex].Lyrics = slices.Clone(source.Tracks[trackIndex].Lyrics)
	}
	return clone
}
