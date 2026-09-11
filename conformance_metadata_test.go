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
	"reflect"
	"strings"
	"testing"
)

func TestConformanceMetadataImport(t *testing.T) {
	runConformanceMetadataImport(newConformanceRun(t))
}

func TestConformanceMetadataCorpus(t *testing.T) {
	runConformanceMetadataCorpus(newConformanceRun(t))
}

func runConformanceMetadataCorpus(run *conformanceRun) {
	tests := []struct {
		path        string
		words       string
		author      string
		writer      string
		transcriber string
		version     Version
	}{
		{path: "testdata/gp3/score-info.gp3", words: "Music", author: "Music", writer: "Tab", version: Version{Data: "FICHIER GUITAR PRO v3.00", Number: [3]byte{3}}},
		{path: "testdata/gp4/score-info.gp4", words: "Music", author: "Music", writer: "Tab", version: Version{Data: "FICHIER GUITAR PRO v4.06", Number: [3]byte{4, 0, 6}}},
		{path: "testdata/gp5/score-info.gp5", words: "Words", author: "Music", writer: "Tab", version: Version{Data: "FICHIER GUITAR PRO v5.10", Number: [3]byte{5, 1}}},
		{path: "testdata/gp6/score-info.gpx", words: "Words", author: "Music", transcriber: "Tab", version: Version{Data: "6", Number: [3]byte{6}}},
		{path: "testdata/gp7/score-info.gp", words: "Words", author: "Music", transcriber: "Tab", version: Version{Data: "7", Number: [3]byte{7}}},
	}
	for _, test := range tests {
		run.t.Run(test.path, func(t *testing.T) {
			song := parseTestFixture(t, test.path)
			run.Preserved("Song.Name", song.Name, "Title")
			run.Preserved("Song.Subtitle", song.Subtitle, "Subtitle")
			run.Preserved("Song.Artist", song.Artist, "Artist")
			run.Preserved("Song.Album", song.Album, "Album")
			run.Preserved("Song.Words", song.Words, test.words)
			run.Preserved("Song.Author", song.Author, test.author)
			run.Omitted("Song.Writer", song.Writer, test.writer)
			run.Preserved("Song.Transcriber", song.Transcriber, test.transcriber)
			run.Preserved("Song.Copyright", song.Copyright, "Copyright")
			run.Preserved("Song.Instructions", song.Instructions, "Instructions")
			run.Normalized("Song.Notice", song.Notice, []string{"Notice1", "Notice2"})
			run.Omitted("Song.Comments", song.Comments, "")
			run.Omitted("Song.Date", song.Date, "")
			run.Normalized("Song.Version", song.Version, test.version)
			run.Normalized("Version.Data", song.Version.Data, test.version.Data)
			run.Derived("Version.Number", song.Version.Number, test.version.Number)
			run.Omitted("Version.Clipboard", song.Version.Clipboard, false)
		})
	}
}

func runConformanceMetadataImport(run *conformanceRun) {
	const source = `<?xml version="1.0" encoding="utf-8"?>
<GPIF>
  <GPVersion>8.0</GPVersion>
  <Score>
    <Title>Title &amp; 名</Title><SubTitle>Subtitle</SubTitle><Artist>Artist</Artist>
    <Album>Album</Album><Words>words sentinel</Words><Music>author sentinel</Music><WordsAndMusic>shared sentinel</WordsAndMusic>
    <Copyright>Copyright</Copyright><Tabber>Transcriber</Tabber>
    <Instructions>Line 1&#10;Line 2 &lt;ok&gt;</Instructions><Notices>First&#10;Second</Notices>
  </Score>
  <MasterTrack><Tracks>0</Tracks></MasterTrack>
  <Tracks><Track id="0"><Name>Guitar</Name><Staves><Staff/></Staves>
    <MidiConnection><Port>0</Port><PrimaryChannel>0</PrimaryChannel><SecondaryChannel>1</SecondaryChannel></MidiConnection>
  </Track></Tracks>
  <MasterBars><MasterBar><Key><AccidentalCount>-2</AccidentalCount><Mode>Minor</Mode></Key><Time>4/4</Time><TripletFeel>Triplet16th</TripletFeel><Bars>0</Bars></MasterBar></MasterBars>
  <Bars><Bar id="0"><Clef>G2</Clef><Voices>-1</Voices></Bar></Bars>
  <Voices/><Beats/><Notes/><Rhythms/>
</GPIF>`

	archive := conformanceGPIFArchive(run.t, source)
	result, err := ParseWithOptions(archive, ParseOptions{})
	if err != nil {
		run.t.Fatal(err)
	}
	song := result.Song
	for _, options := range []ParseOptions{
		{Strict: true},
		{Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticUnknownSyntax}},
	} {
		strictResult, strictErr := ParseWithOptions(archive, options)
		if strictErr != nil {
			run.t.Fatalf("supported metadata strict parse failed: %v, diagnostics %#v", strictErr, strictResult.Diagnostics)
		}
		if strictResult.Song.Name != song.Name || strictResult.Song.Author != song.Author {
			run.t.Fatalf("strict metadata parse changed the score: %#v", strictResult.Song)
		}
	}
	run.ClaimPrimary(claimSite("metadata", "import", "M01-GPIF-METADATA-IMPORT", "Unicode")).Field("Song.Name", song.Name, "Title & 名")
	run.Field("Song.Subtitle", song.Subtitle, "Subtitle")
	run.Field("Song.Artist", song.Artist, "Artist")
	run.Field("Song.Album", song.Album, "Album")
	run.Field("Song.Words", song.Words, "words sentinel")
	run.Field("Song.Author", song.Author, "author sentinel")
	run.Field("Song.Writer", song.Writer, "")
	run.Field("Song.Transcriber", song.Transcriber, "Transcriber")
	run.Field("Song.Copyright", song.Copyright, "Copyright")
	run.Field("Song.Instructions", song.Instructions, "Line 1\nLine 2 <ok>")
	run.Field("Song.Notice", song.Notice, []string{"First", "Second"})
	run.Field("Song.Comments", song.Comments, "")
	run.Field("Song.Date", song.Date, "")
	run.Field("Song.Version", song.Version, Version{Data: "8.0", Number: [3]byte{8, 0, 0}})
	run.Field("Version.Data", song.Version.Data, "8.0")
	run.Field("Version.Number", song.Version.Number, [3]byte{8, 0, 0})
	run.Field("Version.Clipboard", song.Version.Clipboard, false)
	run.Normalized("Song.Key", song.Key, KeySignature{})
	run.Normalized("Song.TripletFeel", song.TripletFeel, TripletFeelNone)

	values := decodeGPIFLeafText(run.t, []byte(source))
	run.Wire("gpifDocument.GPVersion", values["GPIF/GPVersion"], "8.0")
	run.Wire("gpifScore.Title", values["GPIF/Score/Title"], "Title & 名")
	run.Wire("gpifScore.SubTitle", values["GPIF/Score/SubTitle"], "Subtitle")
	run.Wire("gpifScore.Artist", values["GPIF/Score/Artist"], "Artist")
	run.Wire("gpifScore.Album", values["GPIF/Score/Album"], "Album")
	run.Wire("gpifScore.Words", values["GPIF/Score/Words"], "words sentinel")
	run.Wire("gpifScore.Music", values["GPIF/Score/Music"], "author sentinel")
	run.Wire("gpifScore.WordsAndMusic", values["GPIF/Score/WordsAndMusic"], "shared sentinel")
	run.Wire("gpifScore.Copyright", values["GPIF/Score/Copyright"], "Copyright")
	run.Wire("gpifScore.Tabber", values["GPIF/Score/Tabber"], "Transcriber")
	run.Wire("gpifScore.Instructions", values["GPIF/Score/Instructions"], "Line 1\nLine 2 <ok>")
	run.Wire("gpifScore.Notices", values["GPIF/Score/Notices"], "First\nSecond")
}

func TestConformanceOracleKeepsAuthorAndWriterDistinct(t *testing.T) {
	runConformanceOracleKeepsAuthorAndWriterDistinct(newConformanceRun(t))
}

func runConformanceOracleKeepsAuthorAndWriterDistinct(run *conformanceRun) {
	t := run.t
	song := syntheticGP8SongWithoutTerminalDoubleBar()
	song.Author = "author sentinel"
	song.Writer = "writer sentinel"
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	normalized, ok := readAlphaTabScore(t, writeConformanceFixture(t, data)).(map[string]any)
	if !ok {
		t.Fatalf("AlphaTab score has type %T", normalized)
	}
	metadata, ok := normalized["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("normalized metadata = %#v", normalized["metadata"])
	}
	if got := metadata["music"]; got != song.Author {
		t.Fatalf("normalized music = %q, want author %q", got, song.Author)
	}
	run.Field("Song.Author", metadata["music"], song.Author)
}

func TestConformanceBinaryClipboard(t *testing.T) {
	runConformanceBinaryClipboard(newConformanceRun(t))
}

func runConformanceBinaryClipboard(run *conformanceRun) {
	absent := Song{Version: Version{Number: [3]byte{5, 1, 0}}}
	if err := absent.readClipboard(newCursor(nil)); err != nil {
		run.t.Fatal(err)
	}
	run.Omitted("Song.Clipboard", absent.Clipboard, (*Clipboard)(nil))

	var data bytes.Buffer
	for _, value := range []int32{2, 4, 1, 3, 5, 7, 1} {
		if err := binary.Write(&data, binary.LittleEndian, value); err != nil {
			run.t.Fatal(err)
		}
	}
	song := Song{Version: Version{Number: [3]byte{5, 1, 0}, Clipboard: true}}
	if err := song.readClipboard(newCursor(data.Bytes())); err != nil {
		run.t.Fatal(err)
	}
	want := &Clipboard{StartMeasure: 2, StopMeasure: 4, StartTrack: 1, StopTrack: 3, StartBeat: 5, StopBeat: 7, SubBarCopy: true}
	run.Field("Song.Clipboard", song.Clipboard, want)
	run.Omitted("Clipboard.StartMeasure", song.Clipboard.StartMeasure, want.StartMeasure)
	run.Omitted("Clipboard.StopMeasure", song.Clipboard.StopMeasure, want.StopMeasure)
	run.Omitted("Clipboard.StartTrack", song.Clipboard.StartTrack, want.StartTrack)
	run.Omitted("Clipboard.StopTrack", song.Clipboard.StopTrack, want.StopTrack)
	run.Omitted("Clipboard.StartBeat", song.Clipboard.StartBeat, want.StartBeat)
	run.Omitted("Clipboard.StopBeat", song.Clipboard.StopBeat, want.StopBeat)
	run.Omitted("Clipboard.SubBarCopy", song.Clipboard.SubBarCopy, want.SubBarCopy)
}

func TestConformanceMetadataExportPolicy(t *testing.T) {
	runConformanceMetadataExportPolicy(newConformanceRun(t))
}

func runConformanceMetadataExportPolicy(run *conformanceRun) {
	t := run.t
	song := conformanceMetadataSong()
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("M01 baseline is invalid: %#v", diagnostics)
	}
	run.ClaimPrimary(claimSite("metadata", "model", "M01-GP8-METADATA-EXPORT", "Unicode")).Field("Song.Name", song.Name, "Title & 名")
	run.Field("Song.Subtitle", song.Subtitle, "Subtitle <one>")
	run.Field("Song.Artist", song.Artist, "Artist")
	run.Field("Song.Album", song.Album, "Album")
	run.Field("Song.Words", song.Words, "words sentinel")
	run.Field("Song.Author", song.Author, "author sentinel")
	run.Field("Song.Writer", song.Writer, "writer sentinel")
	run.Field("Song.Transcriber", song.Transcriber, "transcriber sentinel")
	run.Field("Song.Copyright", song.Copyright, "Copyright ©")
	run.Field("Song.Instructions", song.Instructions, "Line 1\nLine 2 & <ok>")
	run.Field("Song.Notice", song.Notice, []string{"First\nSecond", "Third"})
	run.Field("Song.Comments", song.Comments, "comments sentinel")
	run.Field("Song.Date", song.Date, "2026-09-09")
	run.Field("Song.Version", song.Version, Version{Data: "FICHIER GUITAR PRO v5.10", Number: [3]byte{5, 1, 0}, Clipboard: true})
	run.Field("Version.Data", song.Version.Data, "FICHIER GUITAR PRO v5.10")
	run.Field("Version.Number", song.Version.Number, [3]byte{5, 1, 0})
	run.Field("Version.Clipboard", song.Version.Clipboard, true)
	run.Field("Song.Clipboard", song.Clipboard, &Clipboard{StartMeasure: 2, StopMeasure: 4, StartTrack: 1, StopTrack: 3, StartBeat: 5, StopBeat: 7, SubBarCopy: true})
	run.Field("Song.Key", song.Key, KeySignature{Key: 3, IsMinor: true})
	run.Field("Song.TripletFeel", song.TripletFeel, TripletFeelEighth)
	before := fmt.Sprintf("%#v", song)
	options := ExportOptions{}
	optionsBefore := fmt.Sprintf("%#v", options)

	report := PreflightExport(song, ExportFormatGP8, options)
	wantDispositions := map[string]ExportDisposition{
		"gp8.omit.clipboard-range":                  ExportDispositionOmitted,
		"gp8.omit.comments":                         ExportDispositionOmitted,
		"gp8.omit.date":                             ExportDispositionOmitted,
		"gp8.omit.writer":                           ExportDispositionOmitted,
		"gp8.normalize.notice-lines":                ExportDispositionNormalized,
		"gp8.normalize.song-key-authority":          ExportDispositionNormalized,
		"gp8.normalize.song-triplet-feel-authority": ExportDispositionNormalized,
		"gp8.normalize.source-version":              ExportDispositionNormalized,
	}
	wantCodes := make([]string, 0, len(wantDispositions))
	for code, disposition := range wantDispositions {
		wantCodes = append(wantCodes, code)
		entry := exportReportEntry(report, code)
		if entry == nil || entry.Disposition != disposition || entry.Location != (ScoreLocation{}) {
			t.Errorf("preflight report = %#v, want %s at score root with disposition %s", report.Entries, code, disposition)
		}
	}
	if len(report.Entries) != len(wantDispositions) {
		t.Errorf("preflight report has %d entries, want %d: %#v", len(report.Entries), len(wantDispositions), report.Entries)
	}
	if fmt.Sprintf("%#v", song) != before || fmt.Sprintf("%#v", options) != optionsBefore {
		t.Fatal("preflight mutated its input")
	}

	data, exportReport, err := ExportWithReport(song, ExportFormatGP8, options)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(exportReport, report) {
		t.Fatalf("export report = %#v, want preflight report %#v", exportReport, report)
	}
	if fmt.Sprintf("%#v", song) != before || fmt.Sprintf("%#v", options) != optionsBefore {
		t.Fatal("export mutated its input")
	}

	values := extractGPIFLeafText(t, data)
	run.Wire("gpifDocument.GPVersion", values["GPIF/GPVersion"], gp8DocumentVersion)
	run.Wire("gpifScore.Title", values["GPIF/Score/Title"], song.Name)
	run.Wire("gpifScore.SubTitle", values["GPIF/Score/SubTitle"], song.Subtitle)
	run.Wire("gpifScore.Artist", values["GPIF/Score/Artist"], song.Artist)
	run.Wire("gpifScore.Album", values["GPIF/Score/Album"], song.Album)
	run.Wire("gpifScore.Words", values["GPIF/Score/Words"], song.Words)
	run.Wire("gpifScore.Music", values["GPIF/Score/Music"], song.Author)
	run.Wire("gpifScore.Copyright", values["GPIF/Score/Copyright"], song.Copyright)
	run.Wire("gpifScore.Tabber", values["GPIF/Score/Tabber"], song.Transcriber)
	run.Wire("gpifScore.Instructions", values["GPIF/Score/Instructions"], song.Instructions)
	run.Wire("gpifScore.Notices", values["GPIF/Score/Notices"], strings.Join(song.Notice, "\n"))
	for _, absent := range []string{"Writer", "Comments", "Date", "Clipboard"} {
		if path := findGPIFElement(values, absent); path != "" {
			t.Errorf("omitted %s appeared at %s", absent, path)
		}
	}

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if roundTrip.Author != song.Author || roundTrip.Writer != "" {
		t.Errorf("author/writer round trip = %q/%q, want %q/empty", roundTrip.Author, roundTrip.Writer, song.Author)
	}
	if !reflect.DeepEqual(roundTrip.Notice, []string{"First", "Second", "Third"}) {
		t.Errorf("normalized notices = %#v", roundTrip.Notice)
	}
	if roundTrip.MeasureHeaders[0].KeySignature != song.MeasureHeaders[0].KeySignature {
		t.Errorf("header key = %#v, want %#v", roundTrip.MeasureHeaders[0].KeySignature, song.MeasureHeaders[0].KeySignature)
	}
	if roundTrip.MeasureHeaders[0].TripletFeel != song.MeasureHeaders[0].TripletFeel {
		t.Errorf("header triplet feel = %v, want %v", roundTrip.MeasureHeaders[0].TripletFeel, song.MeasureHeaders[0].TripletFeel)
	}

	strict := ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}}
	strictData, strictReport, strictErr := ExportWithReport(song, ExportFormatGP8, strict)
	var lossErr *ExportLossError
	if len(strictData) != 0 || !errors.As(strictErr, &lossErr) || !reflect.DeepEqual(strictReport, report) {
		t.Fatalf("strict export = %d bytes, %#v, %v", len(strictData), strictReport, strictErr)
	}
	strict.LossPolicy.AllowedCodes = []string{"gp8.omit.backing-track"}
	if unrelatedData, _, unrelatedErr := ExportWithReport(song, ExportFormatGP8, strict); len(unrelatedData) != 0 || !errors.As(unrelatedErr, &lossErr) {
		t.Fatalf("unrelated allowlist export = %d bytes, %v", len(unrelatedData), unrelatedErr)
	}
	strict.LossPolicy.AllowedCodes = wantCodes
	allowedData, allowedReport, allowedErr := ExportWithReport(song, ExportFormatGP8, strict)
	if allowedErr != nil || len(allowedData) == 0 || !reflect.DeepEqual(allowedReport, report) {
		t.Fatalf("allowed export = %d bytes, %#v, %v", len(allowedData), allowedReport, allowedErr)
	}
}

func conformanceMetadataSong() *Song {
	song := syntheticGP8SongWithoutTerminalDoubleBar()
	song.Name = "Title & 名"
	song.Subtitle = "Subtitle <one>"
	song.Artist = "Artist"
	song.Album = "Album"
	song.Words = "words sentinel"
	song.Author = "author sentinel"
	song.Writer = "writer sentinel"
	song.Transcriber = "transcriber sentinel"
	song.Copyright = "Copyright ©"
	song.Instructions = "Line 1\nLine 2 & <ok>"
	song.Notice = []string{"First\nSecond", "Third"}
	song.Comments = "comments sentinel"
	song.Date = "2026-09-09"
	song.Version = Version{Data: "FICHIER GUITAR PRO v5.10", Number: [3]byte{5, 1, 0}, Clipboard: true}
	song.Clipboard = &Clipboard{StartMeasure: 2, StopMeasure: 4, StartTrack: 1, StopTrack: 3, StartBeat: 5, StopBeat: 7, SubBarCopy: true}
	song.Key = KeySignature{Key: 3, IsMinor: true}
	song.TripletFeel = TripletFeelEighth
	song.MeasureHeaders[0].KeySignature = KeySignature{Key: -2}
	song.MeasureHeaders[0].TripletFeel = TripletFeelSixteenth
	for index := range song.Tracks[0].Measures {
		song.Tracks[0].Measures[index].HeaderIndex = index
	}
	return song
}

func extractGPIFLeafText(t *testing.T, data []byte) map[string]string {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var gpif []byte
	for _, file := range archive.File {
		if file.Name != "Content/score.gpif" {
			continue
		}
		reader, openErr := file.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		gpif, err = io.ReadAll(reader)
		closeErr := reader.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			t.Fatal(err)
		}
		break
	}
	if gpif == nil {
		t.Fatal("GP8 archive has no score.gpif")
	}
	return decodeGPIFLeafText(t, gpif)
}

func decodeGPIFLeafText(t *testing.T, gpif []byte) map[string]string {
	t.Helper()
	decoder := xml.NewDecoder(bytes.NewReader(gpif))
	var stack []string
	values := make(map[string]string)
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
			stack = append(stack, value.Name.Local)
		case xml.CharData:
			text := string(value)
			if len(stack) != 0 && strings.TrimSpace(text) != "" {
				values[strings.Join(stack, "/")] += text
			}
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		}
	}
	return values
}

func findGPIFElement(values map[string]string, name string) string {
	for path := range values {
		parts := strings.Split(path, "/")
		if parts[len(parts)-1] == name {
			return path
		}
	}
	return ""
}
