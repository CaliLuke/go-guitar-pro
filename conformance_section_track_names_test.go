// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"errors"
	"reflect"
	"strings"
	"testing"
)

const sectionNamesCase = "M02-SECTION-TRACK-NAMES"
const sectionNamesValue = "independent section letter and text"
const shortNamesValue = "independent nonempty Unicode short names"
const sectionNamesFixture = "testdata/gp8/section-track-names.gp"

type sectionNameFact struct {
	Letter string `json:"letter"`
	Text   string `json:"text"`
}
type trackNameFact struct {
	Name         string `json:"name"`
	RawShortName string `json:"rawShortName"`
	ShortName    string `json:"shortName"`
}
type sectionTrackNameFacts struct {
	Tracks   []trackNameFact    `json:"tracks"`
	Sections []*sectionNameFact `json:"sections"`
}
type sectionTrackNameWire struct {
	Tracks []struct {
		Name      string  `xml:"Name"`
		ShortName *string `xml:"ShortName"`
	} `xml:"Tracks>Track"`
	MasterBars []struct {
		Section *struct {
			Letter string `xml:"Letter"`
			Text   string `xml:"Text"`
		} `xml:"Section"`
	} `xml:"MasterBars>MasterBar"`
}

func TestConformanceSectionTrackNames(t *testing.T) {
	runConformanceSectionTrackNames(newConformanceRun(t))
}

func runConformanceSectionTrackNames(run *conformanceRun) {
	t := run.t
	song := conformanceNamesSong(t)
	first := song.MeasureHeaders[0].Marker
	run.ClaimPrimary(claimAllStages("sections", sectionNamesCase, sectionNamesValue)...).Preserved("Marker.Letter", first.Letter, "A")
	run.Preserved("Marker.Text", first.Text, "Verse")
	run.Preserved("Marker.Title", first.Title, "Verse")
	run.ClaimPrimary(claimAllStages("short-name", sectionNamesCase, shortNamesValue)...).Preserved("Track.ShortName", song.Tracks[0].ShortName, stringPointer("Gtr Ω <&>"))
	before := *first
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil {
		t.Fatal(err, report.Entries)
	}
	run.ClaimReport(claimSite("sections", "export", sectionNamesCase, sectionNamesValue), claimSite("short-name", "export", sectionNamesCase, shortNamesValue)).Report(sectionNamesCase, reportCodes(report), []string{})
	var wire sectionTrackNameWire
	if decodeErr := xml.Unmarshal(conformanceBarreGPIF(t, data), &wire); decodeErr != nil {
		t.Fatal(decodeErr)
	}
	run.ClaimSerialization(claimSite("sections", "export", sectionNamesCase, sectionNamesValue)).Wire("gpifSection.Letter", wire.MasterBars[0].Section.Letter, "A")
	run.Wire("gpifSection.Text", wire.MasterBars[0].Section.Text, "Verse")
	run.ClaimSerialization(claimSite("short-name", "export", sectionNamesCase, shortNamesValue)).Wire("gpifTrack.ShortName", wire.Tracks[0].ShortName, stringPointer("Gtr Ω <&>"))
	if !reflect.DeepEqual(*first, before) {
		t.Fatal("section export mutated the authored marker")
	}
	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []*sectionNameFact{{"A", "Verse"}, {"", `Text Ω <&> " ]]>`}, {"B <&>", ""}, {"", ""}, nil, {"C", "Coda"}} {
		marker := parsed.MeasureHeaders[i].Marker
		if want == nil {
			if marker != nil {
				t.Fatal("absent section became present")
			}
			continue
		}
		if marker == nil || marker.Letter != want.Letter || marker.Text != want.Text {
			t.Fatalf("section %d = %#v, want %#v", i, marker, want)
		}
	}
	conformanceSectionTitleEdits(t)
	conformanceShortNamePresence(t)
	conformanceNamesConsumerPolicy(t, run)
	for _, name := range []string{"", "Full name"} {
		probe := conformanceNamesSong(t)
		probe.Tracks[0].ShortName = stringPointer("")
		probe.Tracks[0].Name = name
		report := PreflightExport(probe, ExportFormatGP8, ExportOptions{})
		run.Dispatch("buildTrack:track.Name", hasExportCode(report, "gp8.omit.short-name-consumer-empty"), name != "")
	}
}

func conformanceSectionTitleEdits(t *testing.T) {
	t.Helper()
	for _, test := range []struct {
		name         string
		edit         func(*Marker)
		letter, text string
		conflict     bool
	}{
		{"text edit", func(m *Marker) { m.Text = "New Ω <&>" }, "A", "New Ω <&>", false},
		{"letter edit", func(m *Marker) { m.Letter = "B" }, "B", "Verse", false},
		{"clear both", func(m *Marker) { m.Letter = ""; m.Text = "" }, "", "", false},
		{"legacy edit", func(m *Marker) { m.Title = "Legacy" }, "A", "Legacy", false},
		{"legacy clear", func(m *Marker) { m.Title = "" }, "A", "", false},
		{"conflicting edit", func(m *Marker) { m.Title = "Legacy"; m.Text = "New"; m.Letter = "B" }, "B", "Legacy", true},
		{"matching edit", func(m *Marker) { m.Title = "Same"; m.Text = "Same" }, "A", "Same", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := conformanceNamesSong(t)
			marker := song.MeasureHeaders[0].Marker
			test.edit(marker)
			before := *marker
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if hasExportCode(report, "gp8.normalize.section-title") != test.conflict {
				t.Fatalf("section conflict policy = %#v", report.Entries)
			}
			parsed, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			got := parsed.MeasureHeaders[0].Marker
			if got.Letter != test.letter || got.Text != test.text || *marker != before {
				t.Fatalf("edited section = %#v, source %#v", got, marker)
			}
			if test.conflict {
				assertSectionNamesStrictPolicy(t, song, "gp8.normalize.section-title")
			}
		})
	}
	for _, marker := range []*Marker{{Title: "Legacy only"}, {Letter: "L", Text: "Canonical", Title: "Ignored"}} {
		song := conformanceNamesSong(t)
		song.MeasureHeaders[0].Marker = marker
		data, err := Export(song, ExportFormatGP8)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		wantLetter, wantText := marker.Letter, marker.Text
		if wantText == "" {
			wantText = marker.Title
		}
		got := parsed.MeasureHeaders[0].Marker
		if got.Letter != wantLetter || got.Text != wantText {
			t.Fatalf("programmatic marker = %#v", got)
		}
	}
	// Binary markers have one legacy caption and no independent letter.
	legacy := parseTestFixture(t, "testdata/gp3/Measure Header.gp3")
	for _, header := range legacy.MeasureHeaders {
		if header.Marker != nil {
			letter, text := header.Marker.sectionValues()
			if letter != "" || text != header.Marker.Title {
				t.Fatal("binary caption changed")
			}
		}
	}
}

func conformanceShortNamePresence(t *testing.T) {
	t.Helper()
	for _, want := range []*string{nil, stringPointer(""), stringPointer("  Ω <&> \" 日本語  "), stringPointer("line\ntext ]]> Ω")} {
		song := conformanceNamesSong(t)
		song.Tracks[0].ShortName = want
		beforeName := song.Tracks[0].Name
		data, err := Export(song, ExportFormatGP8)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(parsed.Tracks[0].ShortName, want) || song.Tracks[0].Name != beforeName || *song.Tracks[1].ShortName != "Bass long label 日本語" {
			t.Fatal("short-name edit changed its identity, full name or adjacent track")
		}
	}
	source := string(conformanceBarreGPIF(t, mustExportNames(t, conformanceNamesSong(t))))
	source = strings.Replace(source, "<Tracks>0 1</Tracks>", "<Tracks>0 0</Tracks>", 1)
	parsed, err := Parse(conformanceGPIFArchive(t, source))
	if err != nil {
		t.Fatal(err)
	}
	*parsed.Tracks[0].ShortName = "only first"
	if *parsed.Tracks[1].ShortName != "Gtr Ω <&>" {
		t.Fatal("reused track short-name storage is shared")
	}
}

func conformanceNamesConsumerPolicy(t *testing.T, run *conformanceRun) {
	t.Helper()
	for _, test := range []struct {
		code string
		edit func(*Song)
	}{
		{"gp8.omit.short-name-consumer-empty", func(s *Song) { s.Tracks[0].ShortName = stringPointer("") }},
		{"gp8.omit.short-name-consumer-whitespace", func(s *Song) { s.Tracks[0].ShortName = stringPointer("  x ]]>  ") }},
		{"gp8.omit.section-consumer-whitespace", func(s *Song) { s.MeasureHeaders[0].Marker.Text = "  x ]]>  " }},
	} {
		song := conformanceNamesSong(t)
		test.edit(song)
		report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
		run.Report(sectionNamesCase, reportCodes(report), []string{test.code})
		assertSectionNamesStrictPolicy(t, song, test.code)
	}
}

func assertSectionNamesStrictPolicy(t *testing.T, song *Song, code string) {
	t.Helper()
	for _, allowed := range [][]string{nil, {code}} {
		data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
		if len(allowed) == 0 {
			var loss *ExportLossError
			if !errors.As(err, &loss) || len(data) != 0 {
				t.Fatalf("strict names export = %v", err)
			}
		} else if err != nil || len(data) == 0 {
			t.Fatal(err)
		}
	}
}

func mustExportNames(t *testing.T, song *Song) []byte {
	t.Helper()
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestAlphaTabSectionTrackNames(t *testing.T) {
	requireAlphaTabConformance(t)
	var source sectionTrackNameFacts
	readAlphaTabOracleFacts(t, "--section-track-names", sectionNamesFixture, &source)
	want := sectionTrackNameFacts{Tracks: []trackNameFact{{"Lead guitar", "Gtr Ω <&>", "Gtr Ω <&>"}, {"Bass guitar", "Bass long label 日本語", "Bass long label 日本語"}}, Sections: []*sectionNameFact{{"A", "Verse"}, {"", `Text Ω <&> " ]]>`}, {"B <&>", ""}, {"", ""}, nil, {"C", "Coda"}}}
	if !reflect.DeepEqual(source, want) {
		t.Fatalf("source facts = %#v", source)
	}
	song := conformanceNamesSong(t)
	var got sectionTrackNameFacts
	readAlphaTabOracleFacts(t, "--section-track-names", writeConformanceFixture(t, mustExportNames(t, song)), &got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("exported facts = %#v, want %#v", got, want)
	}
	song.MeasureHeaders[0].Marker.Letter = "新 <&>"
	song.MeasureHeaders[0].Marker.Text = "Edited Ω <&>"
	song.Tracks[0].ShortName = stringPointer("Edited Ω <&> \" long")
	want.Sections[0] = &sectionNameFact{"新 <&>", "Edited Ω <&>"}
	want.Tracks[0].RawShortName = *song.Tracks[0].ShortName
	want.Tracks[0].ShortName = *song.Tracks[0].ShortName
	readAlphaTabOracleFacts(t, "--section-track-names", writeConformanceFixture(t, mustExportNames(t, song)), &got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("edited consumer fields = %#v, want %#v", got, want)
	}
	conformanceIndependentClaim(t, "field:Marker.Letter", claimAllStages("sections", sectionNamesCase, sectionNamesValue)...)
	conformanceIndependentClaim(t, "field:Track.ShortName", claimAllStages("short-name", sectionNamesCase, shortNamesValue)...)
	for _, value := range []string{"", "  Unicode Ω <&>  ", "  x ]]>  ", "line\ntext ]]> Ω"} {
		song.Tracks[0].ShortName = stringPointer(value)
		song.MeasureHeaders[0].Marker.Text = value
		readAlphaTabOracleFacts(t, "--section-track-names", writeConformanceFixture(t, mustExportNames(t, song)), &got)
		if value == "" {
			if got.Tracks[0].RawShortName != "" || got.Tracks[0].ShortName != "Lead guita" {
				t.Fatalf("empty-name derived limit = %#v", got.Tracks[0])
			}
		} else {
			expected := value
			if gpifTextConsumerTrims(value) {
				expected = strings.TrimSpace(value)
			}
			if got.Tracks[0].RawShortName != expected || got.Tracks[0].ShortName != expected || got.Sections[0].Text != expected {
				t.Fatalf("consumer text %q = %#v", value, got)
			}
		}
	}
}

func conformanceNamesSong(t *testing.T) *Song {
	t.Helper()
	song := parseTestFixture(t, sectionNamesFixture)
	// Exclude unrelated provenance and display settings from the names policy.
	song.Version = Version{}
	for i := range song.Tracks {
		song.Tracks[i].Settings = TrackSettings{Notation: true, Tablature: true}
	}
	return song
}
