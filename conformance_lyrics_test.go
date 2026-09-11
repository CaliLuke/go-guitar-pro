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
	run.ClaimPrimary(claimSite("lyrics", "model", "M18-LYRIC-SCOPES", "distinct score, track, and rest text")).Preserved("Lyrics.Lines", song.Lyrics.Lines, []LyricLine{{StartMeasureIndex: 0, Text: "score-only one"}, {StartMeasureIndex: 0, Text: "score punctuation: [a], don't!"}, {StartMeasureIndex: 0, Text: "score 日本語"}})
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
	run.ClaimPrimary(claimSite("lyrics", "import", "M18-BINARY-SCORE-LYRICS", "distinct score, track, and rest text")).Field("Lyrics.Lines", len(lyrics.Lines), 5)
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

const conformanceBeatLyricsCase = "M18-BEAT-LYRICS"
const conformanceBeatLyricsValue = "ordered empty and Unicode beat lines"

type conformanceBeatLyricFact struct {
	Track  int      `json:"track"`
	Staff  int      `json:"staff"`
	Bar    int      `json:"bar"`
	Voice  int      `json:"voice"`
	Beat   int      `json:"beat"`
	Lyrics []string `json:"lyrics"`
	Text   string   `json:"text"`
}

func TestConformanceBeatLyrics(t *testing.T) {
	runConformanceBeatLyrics(newConformanceRun(t))
}

func runConformanceBeatLyrics(run *conformanceRun) {
	t := run.t
	source := conformanceBeatLyricsGPIF()
	parsed, err := ParseWithOptions(conformanceGPIFArchive(t, source), ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	upper := parsed.Song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats
	lower := parsed.Song.Tracks[0].Staves[1].Measures[0].Voices[0].Beats
	wantLines := []string{"", "日本語 café"}
	if len(upper) != 2 || len(lower) != 3 {
		t.Fatalf("beat lyric occurrence counts = %d, %d; want 2, 3", len(upper), len(lower))
	}
	run.ClaimPrimary(claimSite("beat-lyrics", "import", conformanceBeatLyricsCase, conformanceBeatLyricsValue)).Preserved("Beat.Lyrics", upper[0].Lyrics, wantLines)
	run.Preserved("Beat.Lyrics", upper[1].Lyrics, wantLines)
	run.Preserved("Beat.Lyrics", lower[0].Lyrics, wantLines)
	if lower[1].Lyrics == nil || len(lower[1].Lyrics) != 0 {
		t.Fatalf("authored empty beat lyrics = %#v, want non-nil empty", lower[1].Lyrics)
	}
	if lower[2].Lyrics != nil {
		t.Fatalf("absent beat lyrics = %#v, want nil", lower[2].Lyrics)
	}
	for _, beat := range []Beat{upper[0], upper[1], lower[0]} {
		if beat.Text != "separate free text" {
			t.Fatalf("beat text = %q, want separate free text", beat.Text)
		}
	}

	// Reused GPIF beat definitions create independently editable public slices.
	upper[0].Lyrics[0] = "edited first"
	if upper[1].Lyrics[0] != "" || lower[0].Lyrics[0] != "" {
		t.Fatalf("editing one reused lyric occurrence changed another: %#v, %#v", upper[1].Lyrics, lower[0].Lyrics)
	}

	allowed := []string{"gp8.normalize.source-version", "gp8.omit.track-display-settings"}
	data, report, err := ExportWithReport(parsed.Song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
	if err != nil || !slices.Equal(reportCodes(report), allowed) {
		t.Fatalf("beat lyric export = %v, %#v", err, report.Entries)
	}
	run.ClaimReport(claimSite("beat-lyrics", "export", conformanceBeatLyricsCase, conformanceBeatLyricsValue)).Report(conformanceBeatLyricsCase, reportCodes(report), allowed)
	wire := extractLyricWire(t, data)
	if len(wire.Beats) != 5 {
		t.Fatalf("wire beats = %d, want 5", len(wire.Beats))
	}
	wantWire := [][]string{{"edited first", "日本語 café"}, wantLines, wantLines, nil, nil}
	gotWire := make([][]string, len(wire.Beats))
	gotPresent := make([]bool, len(wire.Beats))
	for index, beat := range wire.Beats {
		if beat.Lyrics != nil {
			gotPresent[index] = true
			gotWire[index] = beat.Lyrics.Lines
		}
	}
	run.ClaimSerialization(claimSite("beat-lyrics", "export", conformanceBeatLyricsCase, conformanceBeatLyricsValue)).Wire("gpifBeat.Lyrics", gotPresent, []bool{true, true, true, true, false})
	run.Wire("gpifBeatLyrics.Lines", gotWire, wantWire)
	if wire.Beats[3].Lyrics == nil || wire.Beats[4].Lyrics != nil {
		t.Fatalf("wire empty/absent lyric elements = %#v, %#v", wire.Beats[3].Lyrics, wire.Beats[4].Lyrics)
	}
	for index := range 3 {
		if wire.Beats[index].FreeText != "separate free text" {
			t.Fatalf("wire beat %d FreeText = %q", index, wire.Beats[index].FreeText)
		}
	}

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	roundUpper := roundTrip.Tracks[0].Staves[0].Measures[0].Voices[0].Beats
	roundLower := roundTrip.Tracks[0].Staves[1].Measures[0].Voices[0].Beats
	run.ClaimPrimary(
		claimSite("beat-lyrics", "model", conformanceBeatLyricsCase, conformanceBeatLyricsValue),
		claimSite("beat-lyrics", "export", conformanceBeatLyricsCase, conformanceBeatLyricsValue),
	).Preserved("Beat.Lyrics", roundUpper[0].Lyrics, []string{"edited first", "日本語 café"})
	run.Preserved("Beat.Lyrics", roundUpper[1].Lyrics, wantLines)
	run.Preserved("Beat.Lyrics", roundLower[0].Lyrics, wantLines)
	if roundLower[1].Lyrics == nil || len(roundLower[1].Lyrics) != 0 || roundLower[2].Lyrics != nil {
		t.Fatalf("round-trip empty/absent beat lyrics = %#v, %#v", roundLower[1].Lyrics, roundLower[2].Lyrics)
	}
	if parsed.Song.Tracks[0].Lyrics != nil || len(parsed.Song.Lyrics.Lines) != 0 {
		t.Fatal("beat lyrics leaked into track- or score-scoped lyrics")
	}
}

func TestGPIFBeatLyricsMalformedChildrenAreDiagnosed(t *testing.T) {
	source := strings.Replace(conformanceBeatLyricsGPIF(), "</Lyrics>", "<Future>lost</Future></Lyrics>", 1)
	result, err := ParseWithOptions(conformanceGPIFArchive(t, source), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(result.Diagnostics, func(diagnostic ParseDiagnostic) bool {
		return diagnostic.Code == "GPIF.UnknownElement.NoteAndBeat" &&
			diagnostic.Kind == ParseDiagnosticUnknownSyntax &&
			diagnostic.ObjectID == "0" && strings.HasSuffix(diagnostic.SourcePath, "/Lyrics/Future")
	}) {
		t.Fatalf("malformed beat lyric diagnostics = %#v", result.Diagnostics)
	}
	if _, strictErr := ParseWithOptions(conformanceGPIFArchive(t, source), ParseOptions{Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticUnknownSyntax}}); strictErr == nil {
		t.Fatal("strict parsing accepted an unknown beat lyric child")
	}
}

func TestAlphaTabPreservesBeatLyrics(t *testing.T) {
	requireAlphaTabConformance(t)
	sourceData := conformanceGPIFArchive(t, conformanceBeatLyricsGPIF())
	sourcePath := writeConformanceFixture(t, sourceData)
	var source []conformanceBeatLyricFact
	readAlphaTabOracleFacts(t, "--beat-lyrics", sourcePath, &source)
	wantLines := []string{"", "日本語 café"}
	want := []conformanceBeatLyricFact{
		{Track: 0, Staff: 0, Bar: 0, Voice: 0, Beat: 0, Lyrics: wantLines, Text: "separate free text"},
		{Track: 0, Staff: 0, Bar: 0, Voice: 0, Beat: 1, Lyrics: wantLines, Text: "separate free text"},
		{Track: 0, Staff: 1, Bar: 0, Voice: 0, Beat: 0, Lyrics: wantLines, Text: "separate free text"},
		{Track: 0, Staff: 1, Bar: 0, Voice: 0, Beat: 1, Lyrics: []string{}, Text: ""},
	}
	if !reflect.DeepEqual(source, want) {
		t.Fatalf("AlphaTab source beat lyrics = %#v, want %#v", source, want)
	}

	song, err := Parse(sourceData)
	if err != nil {
		t.Fatal(err)
	}
	song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Lyrics[0] = "edited first"
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var output []conformanceBeatLyricFact
	readAlphaTabOracleFacts(t, "--beat-lyrics", writeConformanceFixture(t, data), &output)
	want[0].Lyrics = []string{"edited first", "日本語 café"}
	if !reflect.DeepEqual(output, want) {
		t.Fatalf("AlphaTab output beat lyrics = %#v, want %#v", output, want)
	}
	conformanceIndependentClaim(t, "field:Beat.Lyrics", claimAllStages("beat-lyrics", conformanceBeatLyricsCase, conformanceBeatLyricsValue)...)
}

func conformanceBeatLyricsGPIF() string {
	source := strings.Replace(staffScopedChordGPIF, `<Beats>0</Beats>`, `<Beats>0 0</Beats>`, 1)
	source = strings.Replace(source, `<Beats>1</Beats>`, `<Beats>0 1 2</Beats>`, 1)
	source = strings.Replace(source, `<Beat id="0"><Rhythm ref="0"/><Chord>0</Chord></Beat>`, `<Beat id="0"><Rhythm ref="0"/><FreeText>separate free text</FreeText><Lyrics><Line></Line><Line>日本語 café</Line></Lyrics></Beat>`, 1)
	source = strings.Replace(source, `<Beat id="1"><Rhythm ref="0"/><Chord>0</Chord></Beat>`, `<Beat id="1"><Rhythm ref="0"/><Lyrics></Lyrics></Beat><Beat id="2"><Rhythm ref="0"/></Beat>`, 1)
	return source
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
	Lyrics   *gpifBeatLyrics `xml:"Lyrics"`
	FreeText string          `xml:"FreeText"`
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
