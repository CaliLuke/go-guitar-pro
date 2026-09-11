// SPDX-License-Identifier: MIT

package integration_test

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"reflect"
	"slices"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func textContractSong(t *testing.T) *gp.Song {
	t.Helper()
	song, err := gp.ParseFile("../testdata/gp8/section-track-names.gp")
	if err != nil {
		t.Fatal(err)
	}
	song.Version = gp.Version{}
	for i := range song.Tracks {
		song.Tracks[i].Settings = gp.TrackSettings{Notation: true, Tablature: true}
	}
	if r := gp.PreflightExport(song, gp.ExportFormatGP8, gp.ExportOptions{}); len(r.Entries) != 0 {
		t.Fatal(r)
	}
	return song
}

func TestMetadataTextPublicContract(t *testing.T) {
	for _, test := range []struct {
		text string
		loss bool
	}{
		{"line\nnext\t<&> 日本語", false}, {" \n\tboundary\t\n ", false}, {"A<&]]>B", false}, {"\n\tA<&]]>B\t\n", true}, {"carriage\rreturn", false}, {"\rboundary\r\n", true},
	} {
		t.Run(test.text, func(t *testing.T) {
			song := textContractSong(t)
			song.Author = "author:" + test.text
			song.Transcriber = "tabber:" + test.text
			song.Instructions = test.text
			before := *song
			data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
			if err != nil {
				t.Fatal(err)
			}
			wantCodes := []string(nil)
			if test.loss {
				wantCodes = []string{"gp8.normalize.metadata-text-consumer"}
			}
			if got := textReportCodes(report); !slices.Equal(got, wantCodes) {
				t.Fatalf("codes=%v want%v", got, wantCodes)
			}
			parsed, err := gp.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Author != song.Author || parsed.Transcriber != song.Transcriber || parsed.Instructions != test.text || !reflect.DeepEqual(*song, before) {
				t.Fatal("metadata changed or input mutated")
			}
			if test.loss {
				for _, allowed := range [][]string{nil, {"unrelated"}, wantCodes} {
					output, _, exportErr := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
					if slices.Equal(allowed, wantCodes) {
						if exportErr != nil || len(output) == 0 {
							t.Fatal(exportErr)
						}
					} else if exportErr == nil || len(output) != 0 {
						t.Fatal("strict policy accepted consumer loss")
					}
				}
			}
		})
	}
}

func TestAssignedScoreLyricsPublicContract(t *testing.T) {
	for _, mode := range []string{"empty track", "equivalent", "conflict", "unassigned"} {
		t.Run(mode, func(t *testing.T) {
			song := textContractSong(t)
			lines := []gp.LyricLine{{Text: "第一 second", StartMeasureIndex: 1}, {Text: "", StartMeasureIndex: 3}, {Text: "troisième", StartMeasureIndex: 2}}
			want := []gp.TrackLyricLine{{Text: "第一 second", Offset: 1}, {Text: "", Offset: 3}, {Text: "troisième", Offset: 2}}
			song.Lyrics = gp.Lyrics{TrackIndex: 1, Lines: lines}
			if mode == "equivalent" {
				song.Tracks[1].Lyrics = slices.Clone(want)
			}
			if mode == "conflict" {
				want = []gp.TrackLyricLine{{Text: "explicit", Offset: 0}}
				song.Tracks[1].Lyrics = want
			}
			if mode == "unassigned" {
				song.Lyrics.TrackIndex = -1
				want = nil
			}
			beat := &song.Tracks[1].Measures[0].Voices[0].Beats[0]
			beat.Text = "independent text"
			beat.Lyrics = []string{"independent beat"}
			snapshot, err := xml.Marshal(song.Lyrics)
			if err != nil {
				t.Fatal(err)
			}
			codes := []string(nil)
			if mode == "conflict" || mode == "unassigned" {
				codes = []string{"gp8.omit.score-lyrics"}
			}
			if mode != "unassigned" {
				codes = append(codes, "gp8.omit.track-lyrics-consumer-dispatch")
			}
			data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: codes}})
			if err != nil {
				t.Fatal(err, report)
			}
			if !slices.Equal(textReportCodes(report), codes) {
				t.Fatalf("report=%#v", report)
			}
			parsed, err := gp.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(parsed.Tracks[1].Lyrics, want) || len(parsed.Tracks[0].Lyrics) != 0 {
				t.Fatalf("lyrics=%#v", parsed.Tracks)
			}
			after, _ := xml.Marshal(song.Lyrics)
			if !bytes.Equal(snapshot, after) {
				t.Fatal("mutated score lyrics")
			}
			gotBeat := parsed.Tracks[1].Measures[0].Voices[0].Beats[0]
			if gotBeat.Text != "independent text" || !slices.Equal(gotBeat.Lyrics, beat.Lyrics) {
				t.Fatalf("beat scopes=%#v", gotBeat)
			}
			var wire struct {
				Tracks []struct {
					Lyrics struct {
						Lines []gp.TrackLyricLine `xml:"Line"`
					} `xml:"Lyrics"`
				} `xml:"Tracks>Track"`
			}
			if err = xml.Unmarshal(textContractGPIF(t, data), &wire); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(wire.Tracks[1].Lyrics.Lines, want) {
				t.Fatalf("wire lines=%#v", wire.Tracks[1])
			}
		})
	}
}

func textReportCodes(report gp.ExportReport) []string {
	var codes []string
	for _, e := range report.Entries {
		codes = append(codes, e.Code)
	}
	return codes
}
func textContractGPIF(t *testing.T, data []byte) []byte {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range r.File {
		if f.Name == "Content/score.gpif" {
			reader, e := f.Open()
			if e != nil {
				t.Fatal(e)
			}
			b, e := io.ReadAll(reader)
			_ = reader.Close()
			if e != nil {
				t.Fatal(e)
			}
			return b
		}
	}
	t.Fatal("missing GPIF")
	return nil
}

func TestLegacyAssignedLyricsSource(t *testing.T) {
	for _, path := range []string{"../testdata/gp4/score-info.gp4", "../testdata/gp5/score-info.gp5"} {
		t.Run(path, func(t *testing.T) {
			song, err := gp.ParseFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if song.Lyrics.TrackIndex != 0 || len(song.Lyrics.Lines) != 5 {
				t.Fatalf("source lyrics=%#v", song.Lyrics)
			}
			want := make([]gp.TrackLyricLine, 5)
			for i, line := range song.Lyrics.Lines {
				want[i] = gp.TrackLyricLine{Text: fmt.Sprintf("Line%d", i+1), Offset: i}
				if line.Text != want[i].Text || line.StartMeasureIndex != i {
					t.Fatalf("source line%d=%#v", i, line)
				}
			}
			original, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			// This unique full source record stores one-based track 1 and measures 1..5.
			var record bytes.Buffer
			if err = binary.Write(&record, binary.LittleEndian, int32(1)); err != nil {
				t.Fatal(err)
			}
			for i := range 5 {
				for _, value := range []int32{int32(i + 1), 5} {
					if err = binary.Write(&record, binary.LittleEndian, value); err != nil {
						t.Fatal(err)
					}
				}
				if _, err = fmt.Fprintf(&record, "Line%d", i+1); err != nil {
					t.Fatal(err)
				}
			}
			offset := bytes.Index(original, record.Bytes())
			if offset < 0 || bytes.Count(original, record.Bytes()) != 1 {
				t.Fatal("complete source lyric record not located uniquely")
			}
			if len(song.Tracks) > 1 {
				changed := bytes.Clone(original)
				binary.LittleEndian.PutUint32(changed[offset:], 2)
				song, err = gp.Parse(changed)
				if err != nil {
					t.Fatal(err)
				}
				if song.Lyrics.TrackIndex != 1 {
					t.Fatal("one-based track choice 2 did not become public index 1")
				}
			}
			data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if slices.Contains(textReportCodes(report), "gp8.omit.score-lyrics") {
				t.Fatal("assigned source lyrics were omitted")
			}
			parsed, err := gp.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(parsed.Tracks[song.Lyrics.TrackIndex].Lyrics, want) {
				t.Fatalf("exported lyrics=%#v", parsed.Tracks[song.Lyrics.TrackIndex].Lyrics)
			}
		})
	}
}

func TestAssignedLyricsInvalidReferencesAndClearing(t *testing.T) {
	for _, index := range []int{-2, 2} {
		song := textContractSong(t)
		song.Lyrics = gp.Lyrics{TrackIndex: index, Lines: []gp.LyricLine{{Text: "invalid"}}}
		if !slices.ContainsFunc(gp.ValidateSong(song), func(d gp.ScoreDiagnostic) bool {
			return d.Code == "score.lyrics.track-reference" && d.Location.Track == index
		}) {
			t.Fatal("missing invalid reference diagnostic")
		}
		data, _, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
		if err == nil || len(data) != 0 {
			t.Fatal("invalid lyric reference exported")
		}
	}
	song := textContractSong(t)
	song.Lyrics = gp.Lyrics{TrackIndex: 1, Lines: []gp.LyricLine{{Text: "fallback", StartMeasureIndex: 1}}}
	song.Tracks[1].Lyrics = []gp.TrackLyricLine{{Text: "explicit", Offset: 2}}
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"unrelated"}}})
	if err == nil || len(data) != 0 || !slices.Equal(textReportCodes(report), []string{"gp8.omit.score-lyrics"}) || report.Entries[0].Location.Track != 1 {
		t.Fatal("conflict was not scoped", report, err)
	}
	// Clearing the explicit list re-enables the legacy fallback; clearing both removes it.
	song.Tracks[1].Lyrics = nil
	for _, lines := range [][]gp.LyricLine{song.Lyrics.Lines, nil} {
		song.Lyrics.Lines = lines
		data, report, err = gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}})
		if err != nil || len(report.Entries) != 0 {
			t.Fatal(report, err)
		}
		parsed, e := gp.Parse(data)
		if e != nil {
			t.Fatal(e)
		}
		if len(parsed.Tracks[1].Lyrics) != len(lines) {
			t.Fatal("clearing authority failed")
		}
	}
}
