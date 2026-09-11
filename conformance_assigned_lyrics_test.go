// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"reflect"
	"slices"
	"testing"
)

const assignedLyricsCase = "M18-ASSIGNED-LYRICS"

type assignedLyricTrackFact struct {
	Track int              `json:"track"`
	Lines []TrackLyricLine `json:"lines"`
}
type assignedLyricFacts struct {
	Lines []assignedLyricTrackFact   `json:"lines"`
	Beats []conformanceBeatLyricFact `json:"beats"`
}

func assignedLyricSong(t *testing.T) *Song {
	song := conformanceNamesSong(t)
	template := consumerLimitSong(t).Tracks[0].Measures[0].Voices[0].Beats[0]
	for ti := range song.Tracks {
		for mi := range song.Tracks[ti].Measures {
			song.Tracks[ti].Measures[mi].Voices = []Voice{{Beats: []Beat{template}}}
		}
		song.Tracks[ti].Staves[0].Measures = song.Tracks[ti].Measures
	}
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	song.Lyrics = Lyrics{TrackIndex: 1, Lines: []LyricLine{{Text: "第一 second", StartMeasureIndex: 1}, {Text: "", StartMeasureIndex: 3}, {Text: "troisième", StartMeasureIndex: 2}}}
	beat := &song.Tracks[1].Measures[0].Voices[0].Beats[0]
	beat.Text = "independent text"

	return song
}
func assignedLyricLines() []TrackLyricLine {
	return []TrackLyricLine{{Text: "第一 second", Offset: 1}, {Text: "", Offset: 3}, {Text: "troisième", Offset: 2}}
}

func TestConformanceAssignedLyrics(t *testing.T) { runConformanceAssignedLyrics(newConformanceRun(t)) }
func runConformanceAssignedLyrics(run *conformanceRun) {
	t := run.t
	song := assignedLyricSong(t)
	run.Normalized("Song.Lyrics", song.Lyrics, Lyrics{TrackIndex: 1, Lines: []LyricLine{{Text: "第一 second", StartMeasureIndex: 1}, {Text: "", StartMeasureIndex: 3}, {Text: "troisième", StartMeasureIndex: 2}}})
	run.Preserved("Lyrics.TrackIndex", song.Lyrics.TrackIndex, 1)
	run.Preserved("Lyrics.Lines", len(song.Lyrics.Lines), 3)
	for i, line := range song.Lyrics.Lines {
		run.Preserved("LyricLine.Text", line.Text, assignedLyricLines()[i].Text)
		run.Preserved("LyricLine.StartMeasureIndex", line.StartMeasureIndex, assignedLyricLines()[i].Offset)
	}
	before := conformanceContractSnapshot(song, false)
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil {
		t.Fatal(err, report)
	}
	run.Report(assignedLyricsCase, reportCodes(report), []string{})
	var wire struct {
		Tracks []struct {
			Lyrics *gpifLyrics `xml:"Lyrics"`
		} `xml:"Tracks>Track"`
	}
	if err = xml.Unmarshal([]byte(conformanceBrushGPIF(t, data)), &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Tracks[0].Lyrics != nil || wire.Tracks[1].Lyrics == nil {
		t.Fatal("wrong lyric track", wire)
	}
	got := wire.Tracks[1].Lyrics
	run.Wire("gpifTrack.Lyrics", got != nil, true)
	run.Wire("gpifLyrics.Dispatched", got.Dispatched, true)
	run.Wire("gpifLyrics.Lines", len(got.Lines), 3)
	for i, line := range got.Lines {
		run.Wire("gpifLyricLine.Text", line.Text, assignedLyricLines()[i].Text)
		run.Wire("gpifLyricLine.Offset", line.Offset, assignedLyricLines()[i].Offset)
	}
	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("Track.Lyrics", parsed.Tracks[1].Lyrics, assignedLyricLines())
	for i, line := range parsed.Tracks[1].Lyrics {
		run.Preserved("TrackLyricLine.Text", line.Text, assignedLyricLines()[i].Text)
		run.Preserved("TrackLyricLine.Offset", line.Offset, assignedLyricLines()[i].Offset)
	}
	if !reflect.DeepEqual(before, conformanceContractSnapshot(song, false)) {
		t.Fatal("mutated input")
	}

	for _, lyrics := range [][]string{{}, {"independent"}} {
		mixed := assignedLyricSong(t)
		mixed.Tracks[0].Measures[0].Voices[0].Beats[0].Lyrics = lyrics
		_, lossReport := assertMetadataTextPolicy(t, mixed, []string{"gp8.omit.track-lyrics-consumer-dispatch"})
		run.Report(assignedLyricsCase, reportCodes(lossReport), []string{"gp8.omit.track-lyrics-consumer-dispatch"})
		if lossReport.Entries[0].Location != (ScoreLocation{Track: 1}) {
			t.Fatal(lossReport)
		}
	}
	textLimit := assignedLyricSong(t)
	textLimit.Lyrics.Lines[0].Text = "\n\tA<&]]>B\t\n"
	_, textReport := assertMetadataTextPolicy(t, textLimit, []string{"gp8.normalize.track-lyrics-text-consumer"})
	run.Report(assignedLyricsCase, reportCodes(textReport), []string{"gp8.normalize.track-lyrics-text-consumer"})
	for _, path := range []string{"testdata/gp4/score-info.gp4", "testdata/gp5/score-info.gp5"} {
		source := parseTestFixture(t, path)
		output, exportReport, exportErr := ExportWithReport(source, ExportFormatGP8, ExportOptions{})
		if exportErr != nil || hasExportCode(exportReport, "gp8.omit.score-lyrics") {
			t.Fatal(exportErr, exportReport)
		}
		imported, parseErr := Parse(output)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		for i, line := range source.Lyrics.Lines {
			run.Preserved("Lyrics.TrackIndex", source.Lyrics.TrackIndex, 0)
			run.Preserved("LyricLine.StartMeasureIndex", line.StartMeasureIndex, i)
			run.Preserved("TrackLyricLine.Offset", imported.Tracks[0].Lyrics[i].Offset, i)
		}
	}
}

func TestAlphaTabAssignedLyrics(t *testing.T) {
	requireAlphaTabConformance(t)
	song := assignedLyricSong(t)
	data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil {
		t.Fatal(err)
	}
	var facts assignedLyricFacts
	readAlphaTabOracleFacts(t, "--assigned-lyrics", writeConformanceFixture(t, data), &facts)
	if !reflect.DeepEqual(facts.Lines, []assignedLyricTrackFact{{Track: 1, Lines: assignedLyricLines()}}) {
		t.Fatalf("consumer lines=%#v", facts.Lines)
	}
	wantBeats := []conformanceBeatLyricFact{
		{Track: 1, Bar: 1, Lyrics: []string{"第一", "", ""}},
		{Track: 1, Bar: 2, Lyrics: []string{"second", "", "troisième"}},
	}
	if !reflect.DeepEqual(facts.Beats, wantBeats) {
		t.Fatalf("dispatched=%#v want%#v", facts.Beats, wantBeats)
	}
	for _, path := range []string{"testdata/gp4/score-info.gp4", "testdata/gp5/score-info.gp5"} {
		var source assignedLyricFacts
		readAlphaTabOracleFacts(t, "--assigned-lyrics", path, &source)
		want := []TrackLyricLine{{Text: "Line1", Offset: 0}, {Text: "Line2", Offset: 1}, {Text: "Line3", Offset: 2}, {Text: "Line4", Offset: 3}, {Text: "Line5", Offset: 4}}
		if len(source.Lines) != 1 || source.Lines[0].Track != 0 || !slices.Equal(source.Lines[0].Lines, want) {
			t.Fatalf("%s source=%#v", path, source)
		}
		score := parseTestFixture(t, path)
		output, exportErr := Export(score, ExportFormatGP8)
		if exportErr != nil {
			t.Fatal(exportErr)
		}
		var target assignedLyricFacts
		readAlphaTabOracleFacts(t, "--assigned-lyrics", writeConformanceFixture(t, output), &target)
		if !reflect.DeepEqual(source, target) {
			t.Fatalf("%s consumer source=%#v target=%#v", path, source, target)
		}
	}
}

func TestAlphaTabAssignedLyricsWithBeatLyrics(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, lyrics := range [][]string{{}, {"independent beat", "日本語"}} {
		song := assignedLyricSong(t)
		beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
		beat.Lyrics = lyrics
		beat.Text = "beat text"
		data, report := assertMetadataTextPolicy(t, song, []string{"gp8.omit.track-lyrics-consumer-dispatch"})
		if report.Entries[0].Location != (ScoreLocation{Track: 1}) {
			t.Fatal(report)
		}
		parsed, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(parsed.Tracks[1].Lyrics, assignedLyricLines()) || !reflect.DeepEqual(parsed.Tracks[0].Measures[0].Voices[0].Beats[0].Lyrics, lyrics) {
			t.Fatal("Go conflated track and beat lyrics")
		}
		var facts assignedLyricFacts
		readAlphaTabOracleFacts(t, "--assigned-lyrics", writeConformanceFixture(t, data), &facts)
		want := []conformanceBeatLyricFact{{Lyrics: lyrics, Text: "beat text"}}
		if len(facts.Lines) != 0 || !reflect.DeepEqual(facts.Beats, want) {
			t.Fatalf("suppressed dispatch=%#v", facts)
		}
	}
}

func TestAlphaTabAssignedLyricsTextLimit(t *testing.T) {
	requireAlphaTabConformance(t)
	song := assignedLyricSong(t)
	song.Lyrics.Lines = []LyricLine{{Text: "\n\tA<&]]>B\t\n", StartMeasureIndex: 1}}
	data, report := assertMetadataTextPolicy(t, song, []string{"gp8.normalize.track-lyrics-text-consumer"})
	if report.Entries[0].Location.Track != 1 {
		t.Fatal(report)
	}
	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Tracks[1].Lyrics[0].Text != song.Lyrics.Lines[0].Text {
		t.Fatal("trimmed authored Go text")
	}
	var facts assignedLyricFacts
	readAlphaTabOracleFacts(t, "--assigned-lyrics", writeConformanceFixture(t, data), &facts)
	if !reflect.DeepEqual(facts.Lines, []assignedLyricTrackFact{{Track: 1, Lines: []TrackLyricLine{{Text: "A<&]]>B", Offset: 1}}}}) {
		t.Fatalf("consumer text=%#v", facts.Lines)
	}
}
