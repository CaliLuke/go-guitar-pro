// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/binary"
	"reflect"
	"strings"
	"testing"
)

const headerFooterCase = "M02-HEADER-FOOTER"
const headerFooterValue = "distinct independent header and footer templates and visibility"

func TestConformanceHeaderFooter(t *testing.T) { runConformanceHeaderFooter(newConformanceRun(t)) }
func runConformanceHeaderFooter(run *conformanceRun) {
	t := run.t
	for _, format := range []string{"gp5", "gp8"} {
		song := headerFooterSong(t, format)
		run.Normalized("ScoreStyle.HeaderFooter", song.Style.HeaderFooter != nil, true)
		expected := []string{"Title: %TITLE%", "Subtitle: %SUBTITLE%", "Artist: %ARTIST%", "Album: %ALBUM%", "Words: %WORDS%", "Music: %MUSIC%", "Words & Music: %MUSIC%", "Transcriber: %TABBER%", "Copyright: %COPYRIGHT%", "Copyright2"}
		visible := []bool{false, true, false, true, false, true, false, true, true, false}
		if format == "gp5" {
			expected[6] = "Words & Music: %WORDSMUSIC%"
			visible[9] = true
		}
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.omit.page-setup", "gp8.normalize.empty-beat", "gp8.normalize.track-name-page-policy"}}})
		if err != nil {
			t.Fatal(err)
		}
		parsed, parseErr := Parse(data)
		if parseErr != nil {
			t.Fatal(parseErr)
		}
		wantPage := PageSetup{Title: expected[0], Subtitle: expected[1], Artist: expected[2], Album: expected[3], Words: expected[4], Music: expected[5], WordAndMusic: expected[6], Copyright: expected[8] + "\n" + expected[9], HeaderAndFooter: 170}
		run.Normalized("Song.PageSetup", parsed.PageSetup, wantPage)
		for index, field := range []string{"Title", "Subtitle", "Artist", "Album", "Words", "Music", "WordAndMusic"} {
			run.Normalized("PageSetup."+field, *pageTemplatePointers(&parsed.PageSetup)[index], *pageTemplatePointers(&wantPage)[index])
		}
		run.Normalized("PageSetup.Copyright", parsed.PageSetup.Copyright, wantPage.Copyright)
		run.Normalized("PageSetup.HeaderAndFooter", parsed.PageSetup.HeaderAndFooter, wantPage.HeaderAndFooter)
		records := scoreStyleWire(t, data)
		for index, slot := range headerFooterSlots(song.Style.HeaderFooter) {
			run.Normalized("HeaderFooterSettings."+headerFooterNames[index], *slot != nil, format != "gp5" || index != 7)
			if *slot == nil {
				continue
			}
			run.ClaimPrimary(claimAllStages("page-setup", headerFooterCase, headerFooterValue)...).Preserved("HeaderFooterStyle.Template", (*slot).Template, &expected[index])
			run.Preserved("HeaderFooterStyle.Visible", (*slot).Visible, &visible[index])
			template, visibility := findStyleRecord(t, records, headerFooterKey(index, false)), findStyleRecord(t, records, headerFooterKey(index, true))
			run.Wire("binaryStyleRecord.key", []string{template.key, visibility.key}, []string{headerFooterKey(index, false), headerFooterKey(index, true)})
			run.Wire("binaryStyleRecord.kind", []byte{template.kind, visibility.kind}, []byte{3, 0})
			run.ClaimSerialization(claimSite("page-setup", "export", headerFooterCase, headerFooterValue)).Wire("binaryStyleRecord.value", string(template.value[2:]), expected[index])
			run.Wire("binaryStyleRecord.value", int(binary.BigEndian.Uint16(template.value)), len(expected[index]))
			run.Wire("binaryStyleRecord.value", visibility.value[0] == 1, visible[index])
		}
		want := []string{"gp8.normalize.track-name-page-policy"}
		if format == "gp5" {
			want = []string{"gp8.omit.page-setup", "gp8.normalize.empty-beat", "gp8.normalize.empty-beat"}
		}
		run.ClaimReport(claimSite("page-setup", "export", headerFooterCase, headerFooterValue)).Report(headerFooterCase, reportCodes(report), want)
	}
	song := headerFooterSong(t, "gp8")
	song.Style.HeaderFooter.Title = &HeaderFooterStyle{Visible: ptrTo(false)}
	parsed, err := Parse(mustStringNumberExport(t, song))
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("HeaderFooterStyle.Template", parsed.Style.HeaderFooter.Title.Template, (*string)(nil))
	run.Preserved("HeaderFooterStyle.Visible", parsed.Style.HeaderFooter.Title.Visible, ptrTo(false))
}
func headerFooterSong(t *testing.T, format string) *Song {
	t.Helper()
	suffix := ".gp"
	if format == "gp5" {
		suffix = ".gp5"
	}
	song := parseTestFixture(t, "testdata/"+format+"/header-footer"+suffix)
	cleanNotationProvenance(song)
	song.Writer = ""
	song.Lyrics = Lyrics{}
	song.MasterEffect = RseMasterEffect{}
	song.PanAutomations = nil
	song.VolumeAutomations = nil
	for i := range song.Tracks {
		song.Tracks[i].UseRse = false
		song.Tracks[i].Rse = TrackRse{}
	}
	return song
}
func TestAlphaTabHeaderFooter(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, format := range []string{"gp5", "gp8"} {
		source := "testdata/" + format + "/header-footer.gp"
		if format == "gp5" {
			source += "5"
		}
		var original, got scoreStyleFacts
		readAlphaTabOracleFacts(t, "--score-style", source, &original)
		song := headerFooterSong(t, format)
		readAlphaTabOracleFacts(t, "--score-style", writeConformanceFixture(t, mustStringNumberExport(t, song)), &got)
		if !reflect.DeepEqual(got, original) {
			t.Fatalf("source headers/styles changed: got=%#v want=%#v", got, original)
		}
		// Every existing element gets a distinct exact Unicode/XML-sensitive/CR template.
		for i, slot := range headerFooterSlots(song.Style.HeaderFooter) {
			if *slot == nil {
				continue
			}
			value := headerFooterNames[i] + " é名 <&>\r\n%TITLE%  "
			(*slot).Template = &value
			(*slot).Visible = ptrTo(i%2 == 0)
		}
		readAlphaTabOracleFacts(t, "--score-style", writeConformanceFixture(t, mustStringNumberExport(t, song)), &got)
		if len(got.Headers) != len(original.Headers) {
			t.Fatal("edited header count changed")
		}
		for _, header := range got.Headers {
			index := headerFactIndex(t, header["element"].(string))
			if header["template"] != headerFooterNames[index]+" é名 <&>\r\n%TITLE%  " || header["visible"] != (index%2 == 0) {
				t.Fatalf("edited header changed: %#v", header)
			}
		}
		if !reflect.DeepEqual(got.Other, original.Other) || got.Extended != original.Extended || got.Numbers != original.Numbers {
			t.Fatal("header edit changed another stylesheet policy")
		}
		for i := range got.Headers {
			if got.Headers[i]["align"] != original.Headers[i]["align"] {
				t.Fatal("header edit changed source alignment")
			}
		}
	}
	conformanceIndependentClaim(t, "field:HeaderFooterStyle.Template", claimAllStages("page-setup", headerFooterCase, headerFooterValue)...)
}
func headerFactIndex(t *testing.T, name string) int {
	t.Helper()
	if name == "SubTitle" {
		name = "Subtitle"
	}
	if name == "Transcriber" {
		name = "Tabber"
	}
	if name == "CopyrightSecondLine" {
		name = "Copyright2"
	}
	for i, n := range headerFooterNames {
		if n == name {
			return i
		}
	}
	t.Fatalf("unknown header %s", name)
	return 0
}
func TestHeaderFooterEmptyAbsentAndBounds(t *testing.T) {
	song := headerFooterSong(t, "gp8")
	for _, template := range []*string{nil, ptrTo(""), ptrTo(strings.Repeat("é", 16383) + "a")} {
		song.Style.HeaderFooter.Title = &HeaderFooterStyle{Template: template}
		data := mustStringNumberExport(t, song)
		parsed, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		if template == nil {
			if parsed.Style.HeaderFooter.Title != nil {
				t.Fatal("absent template materialized")
			}
		} else if !reflect.DeepEqual(parsed.Style.HeaderFooter.Title.Template, template) || parsed.Style.HeaderFooter.Title.Visible != nil {
			t.Fatal("empty/boundary template changed")
		}
	}
	for _, template := range []string{strings.Repeat("a", 32768), string([]byte{255})} {
		song.Style.HeaderFooter.Title = &HeaderFooterStyle{Template: &template}
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{AllowedCodes: []string{"gp8.reject.score.style.header-template"}}})
		if err == nil || len(data) != 0 {
			t.Fatalf("invalid template exported: %#v", report)
		}
	}
	xml := string(conformanceBarreGPIF(t, mustStringNumberExport(t, headerFooterSong(t, "gp8"))))
	// A known template or visibility key with the wrong binary type is malformed.
	for _, record := range []binaryStyleRecord{{key: "Header/Title", kind: 0, value: []byte{1}}, {key: "Footer/drawCopyright2", kind: 3, value: []byte{0, 0}}} {
		data := []byte{0, 0, 0, 1, byte(len(record.key))}
		data = append(data, record.key...)
		data = append(data, record.kind)
		data = append(data, record.value...)
		archive := conformanceBackingArchive(t, xml, map[string][]byte{"Content/BinaryStylesheet": data})
		parsed, err := Parse(archive)
		result, optionsErr := ParseWithOptions(archive, ParseOptions{})
		if parsed != nil || result != nil || err == nil || optionsErr == nil || !strings.Contains(err.Error(), record.key) {
			t.Fatalf("wrong header record type accepted: %v/%v", err, optionsErr)
		}
	}
}
func TestHeaderFooterLegacyAuthority(t *testing.T) {
	song := headerFooterSong(t, "gp8")
	before := *song.Style.HeaderFooter.Title.Template
	song.PageSetup.Title = "Legacy edit é"
	song.PageSetup.HeaderAndFooter ^= 1
	song.PageSetup.Copyright = "first\nsecond\nthird"
	source := conformanceContractSnapshot(song, false)
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(reportCodes(report), []string{"gp8.normalize.page-copyright-lines", "gp8.normalize.track-name-page-policy"}) {
		t.Fatalf("copyright ambiguity=%#v", report)
	}
	if conformanceContractSnapshot(song, false) != source || *song.Style.HeaderFooter.Title.Template != before {
		t.Fatal("legacy reconciliation mutated source")
	}
	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if *parsed.Style.HeaderFooter.Title.Template != "Legacy edit é" || !*parsed.Style.HeaderFooter.Title.Visible || *parsed.Style.HeaderFooter.Copyright.Template != "first" || *parsed.Style.HeaderFooter.Copyright2.Template != "second\nthird" {
		t.Fatal("legacy authority or deterministic copyright split changed")
	}
	// GP5 keeps the original two source strings, even if the first contains a newline.
	song = headerFooterSong(t, "gp5")
	song.PageSetup.Copyright = "a\nb\nc"
	song.importLegacyHeaderFooter("a\nb", "c")
	out := song.resolvedHeaderFooter()
	if *out.Copyright.Template != "a\nb" || *out.Copyright2.Template != "c" {
		t.Fatal("known legacy source boundary discarded")
	}
}

func TestAlphaTabHeaderFooterDefaultsAndLegacyEdits(t *testing.T) {
	requireAlphaTabConformance(t)
	song := headerFooterSong(t, "gp8")
	for _, template := range []*string{nil, ptrTo(""), ptrTo(strings.Repeat("é", 16383) + "a")} {
		song.Style.HeaderFooter.Title = &HeaderFooterStyle{Template: template}
		var got scoreStyleFacts
		readAlphaTabOracleFacts(t, "--score-style", writeConformanceFixture(t, mustStringNumberExport(t, song)), &got)
		header := headerConsumerFact(t, got, "Title")
		want := "%TITLE%"
		if template != nil {
			want = *template
		}
		if header["template"] != want {
			t.Fatal("absent/empty/boundary consumer template changed")
		}
		if _, exists := header["visible"]; exists {
			t.Fatal("absent visibility became explicit")
		}
	}
	song = headerFooterSong(t, "gp8")
	song.PageSetup.Copyright = "first é\nsecond\nthird"
	song.PageSetup.Title = "legacy é title"
	var got scoreStyleFacts
	readAlphaTabOracleFacts(t, "--score-style", writeConformanceFixture(t, mustStringNumberExport(t, song)), &got)
	for element, want := range map[string]string{"Copyright": "first é", "CopyrightSecondLine": "second\nthird", "Title": "legacy é title"} {
		if headerConsumerFact(t, got, element)["template"] != want {
			t.Fatalf("legacy %s changed", element)
		}
	}
}

func TestHeaderFooterPhysicalPageLimits(t *testing.T) {
	for _, edit := range []func(*PageSetup){func(p *PageSetup) { p.PageWidth = 1 }, func(p *PageSetup) { p.PageHeight = 1 }, func(p *PageSetup) { p.MarginLeft = 1 }, func(p *PageSetup) { p.MarginRight = 1 }, func(p *PageSetup) { p.MarginTop = 1 }, func(p *PageSetup) { p.MarginBottom = 1 }, func(p *PageSetup) { p.ScoreSizeProportion = 1 }, func(p *PageSetup) { p.PageNumber = "page" }, func(p *PageSetup) { p.HeaderAndFooter |= 1 << 8 }} {
		song := headerFooterSong(t, "gp8")
		edit(&song.PageSetup)
		before := conformanceContractSnapshot(song, false)
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.normalize.track-name-page-policy"}}})
		if err == nil || len(data) != 0 || !reflect.DeepEqual(reportCodes(report), []string{"gp8.omit.page-setup", "gp8.normalize.track-name-page-policy"}) {
			t.Fatalf("page limit missing: %v %#v", err, report)
		}
		if conformanceContractSnapshot(song, false) != before {
			t.Fatal("page limit validation mutated source")
		}
	}
}

func headerConsumerFact(t *testing.T, facts scoreStyleFacts, element string) map[string]any {
	t.Helper()
	for _, row := range facts.Headers {
		if row["element"] == element {
			return row
		}
	}
	t.Fatalf("consumer header %s is absent", element)
	return nil
}
