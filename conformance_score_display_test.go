// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/binary"
	"maps"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const scoreDisplayCase = "M02-SCORE-DISPLAY"
const scoreDisplayValue = "independent brackets names and global display requests"

type displayBooleanCase struct {
	field, key, consumer string
	fallback             bool
	selectField          func(*ScoreStyle) **bool
}

func displayBooleanCases() []displayBooleanCase {
	return []displayBooleanCase{
		{"HideDynamics", "StandardNotation/hideDynamics", "hideDynamics", false, func(s *ScoreStyle) **bool { return &s.HideDynamics }},
		{"SystemSeparators", "Global/useSystemSignSeparator", "useSystemSignSeparator", false, func(s *ScoreStyle) **bool { return &s.SystemSeparators }},
		{"DisplayTuning", "Global/DisplayTuning", "globalDisplayTuning", true, func(s *ScoreStyle) **bool { return &s.DisplayTuning }},
		{"ChordDiagramsOnTop", "Global/DrawChords", "globalDisplayChordDiagramsOnTop", true, func(s *ScoreStyle) **bool { return &s.ChordDiagramsOnTop }},
		{"ChordDiagramsInScore", "System/drawChordInScore", "globalDisplayChordDiagramsInScore", false, func(s *ScoreStyle) **bool { return &s.ChordDiagramsInScore }},
		{"SingleTrackNamesVisible", "System/showTrackNameSingle", "singleTrackTrackNamePolicy", true, func(s *ScoreStyle) **bool { return &s.SingleTrackNamesVisible }},
		{"MultiTrackNamesVisible", "System/showTrackNameMulti", "multiTrackTrackNamePolicy", true, func(s *ScoreStyle) **bool { return &s.MultiTrackNamesVisible }},
		{"FirstSystemShortNames", "System/shortTrackNameOnFirstSystem", "firstSystemTrackNameMode", true, func(s *ScoreStyle) **bool { return &s.FirstSystemShortNames }},
		{"OtherSystemsShortNames", "System/shortTrackNameOnOtherSystems", "otherSystemsTrackNameMode", true, func(s *ScoreStyle) **bool { return &s.OtherSystemsShortNames }},
		{"FirstSystemHorizontalNames", "System/horizontalTrackNameOnFirstSystem", "firstSystemTrackNameOrientation", false, func(s *ScoreStyle) **bool { return &s.FirstSystemHorizontalNames }},
		{"OtherSystemsHorizontalNames", "System/horizontalTrackNameOnOtherSystems", "otherSystemsTrackNameOrientation", false, func(s *ScoreStyle) **bool { return &s.OtherSystemsHorizontalNames }},
	}
}
func scoreDisplaySong(t *testing.T) *Song {
	t.Helper()
	s := parseTestFixture(t, "testdata/gp8/track-names.gp")
	cleanNotationProvenance(s)
	s.PanAutomations = nil
	s.VolumeAutomations = nil
	for i := range s.Tracks {
		s.Tracks[i].UseRse = false
	}
	return s
}
func TestConformanceScoreDisplay(t *testing.T) { runConformanceScoreDisplay(newConformanceRun(t)) }
func runConformanceScoreDisplay(run *conformanceRun) {
	t := run.t
	for _, c := range displayBooleanCases() {
		for _, value := range []*bool{nil, ptrTo(false), ptrTo(true)} {
			song := scoreDisplaySong(t)
			*c.selectField(song.Style) = value
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			run.Preserved("ScoreStyle."+c.field, *c.selectField(parsed.Style), value)
			records := scoreStyleWire(t, data)
			found := false
			for _, record := range records {
				if record.key == c.key {
					found = true
					run.Wire("binaryStyleRecord.key", record.key, c.key)
					run.Wire("binaryStyleRecord.kind", record.kind, byte(0))
					run.Wire("binaryStyleRecord.value", record.value, []byte{map[bool]byte{false: 0, true: 1}[*value]})
				}
			}
			run.Wire("binaryStyleRecord.key", found, value != nil)
			run.Report(scoreDisplayCase, reportCodes(report), []string{})
		}
	}
	for _, value := range []BracketMode{BracketNone, BracketStaves, BracketSimilarInstruments} {
		song := scoreDisplaySong(t)
		song.Style.Brackets = &value
		data := mustStringNumberExport(t, song)
		parsed, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		run.ClaimPrimary(claimImportModel("stylesheet", scoreDisplayCase, scoreDisplayValue)...).Preserved("ScoreStyle.Brackets", parsed.Style.Brackets, &value)
		run.Enum("BracketMode."+[]string{"BracketNone", "BracketStaves", "BracketSimilarInstruments"}[value], *parsed.Style.Brackets, value)
		record := findStyleRecord(t, scoreStyleWire(t, data), "System/bracketExtendMode")
		run.Wire("binaryStyleRecord.kind", record.kind, byte(1))
		run.Wire("binaryStyleRecord.value", binary.BigEndian.Uint32(record.value), uint32(value))
	}
	for _, multi := range []bool{false, true} {
		for _, value := range []*TrackNameSystemMode{nil, ptrTo(TrackNamesFirstSystem), ptrTo(TrackNamesFirstSystemEachPage), ptrTo(TrackNamesAllSystems)} {
			song := scoreDisplaySong(t)
			field, key := "SingleTrackNameMode", "System/trackNameModeSingle"
			song.Style.SingleTrackNameMode = value
			if multi {
				song.Style.SingleTrackNameMode = ptrTo(TrackNamesAllSystems)
				song.Style.MultiTrackNameMode = value
				field, key = "MultiTrackNameMode", "System/trackNameModeMulti"
			}
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.normalize.track-name-page-policy"}}})
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			got := parsed.Style.SingleTrackNameMode
			if multi {
				got = parsed.Style.MultiTrackNameMode
			}
			run.Preserved("ScoreStyle."+field, got, value)
			wantReports := []string{}
			if value != nil && *value == TrackNamesFirstSystemEachPage {
				wantReports = append(wantReports, "gp8.normalize.track-name-page-policy")
			}
			run.Report(scoreDisplayCase, reportCodes(report), wantReports)
			if value != nil {
				run.Enum("TrackNameSystemMode."+[]string{"TrackNamesFirstSystem", "TrackNamesFirstSystemEachPage", "TrackNamesAllSystems"}[*value], *got, *value)
				record := findStyleRecord(t, scoreStyleWire(t, data), key)
				run.Wire("binaryStyleRecord.value", binary.BigEndian.Uint32(record.value), uint32(*value))
			}
		}
	}
	song := scoreDisplaySong(t)
	song.Style.Brackets = nil
	parsed, err := Parse(mustStringNumberExport(t, song))
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("ScoreStyle.Brackets", parsed.Style.Brackets, (*BracketMode)(nil))
	dispatch := []string{}
	for _, record := range scoreDisplayWireSource(t) {
		if record.key == "System/bracketExtendMode" || record.key == "System/trackNameModeSingle" || record.key == "System/trackNameModeMulti" {
			dispatch = append(dispatch, record.key)
		}
	}
	slices.Sort(dispatch)
	run.Dispatch("applyScoreDisplayRecord:record.key", dispatch, []string{"System/bracketExtendMode", "System/trackNameModeMulti", "System/trackNameModeSingle"})
}
func scoreDisplayWireSource(t *testing.T) []binaryStyleRecord {
	t.Helper()
	return parseTestFixture(t, "testdata/gp8/track-names.gp").Style.records
}
func TestAlphaTabScoreDisplay(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, fixture := range []string{"testdata/gp7/brackets-braces-none.gp", "testdata/gp7/brackets-braces-staves.gp", "testdata/gp7/brackets-braces-similar.gp", "testdata/gp7/system-divider.gp", "testdata/gp8/track-names.gp", "testdata/gp8/track-names-hidden.gp", "testdata/gp8/hide-tuning.gp", "testdata/gp8/hide-diagrams.gp", "testdata/gp8/show-diagrams-in-score.gp", "testdata/gp8/directions.gp"} {
		var original, got scoreStyleFacts
		readAlphaTabOracleFacts(t, "--score-style", fixture, &original)
		song := parseTestFixture(t, fixture)
		readAlphaTabOracleFacts(t, "--score-style", writeConformanceFixture(t, mustStringNumberExport(t, song)), &got)
		if !reflect.DeepEqual(got, original) {
			t.Fatalf("source score styles changed for %s", fixture)
		}
	}
	var original scoreStyleFacts
	readAlphaTabOracleFacts(t, "--score-style", "testdata/gp8/track-names.gp", &original)
	for _, c := range displayBooleanCases() {
		for _, value := range []*bool{nil, ptrTo(false), ptrTo(true)} {
			song := scoreDisplaySong(t)
			*c.selectField(song.Style) = value
			want := original
			want.Other = maps.Clone(original.Other)
			v := c.fallback
			if value != nil {
				v = *value
			}
			switch c.field {
			case "SingleTrackNamesVisible", "MultiTrackNamesVisible":
				want.Other[c.consumer] = float64(2)
				if !v {
					want.Other[c.consumer] = float64(0)
				}
			case "FirstSystemShortNames", "OtherSystemsShortNames":
				want.Other[c.consumer] = float64(0)
				if v {
					want.Other[c.consumer] = float64(1)
				}
			case "FirstSystemHorizontalNames", "OtherSystemsHorizontalNames":
				want.Other[c.consumer] = float64(1)
				if v {
					want.Other[c.consumer] = float64(0)
				}
			default:
				want.Other[c.consumer] = v
			}
			var got scoreStyleFacts
			readAlphaTabOracleFacts(t, "--score-style", writeConformanceFixture(t, mustStringNumberExport(t, song)), &got)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%s value %v changed another policy", c.field, value)
			}
		}
	}
	for _, value := range []*BracketMode{nil, ptrTo(BracketNone), ptrTo(BracketStaves), ptrTo(BracketSimilarInstruments)} {
		song := scoreDisplaySong(t)
		song.Style.Brackets = value
		want := original
		want.Other = maps.Clone(original.Other)
		want.Other["bracketExtendMode"] = float64(1)
		if value != nil {
			want.Other["bracketExtendMode"] = float64(*value)
		}
		var got scoreStyleFacts
		readAlphaTabOracleFacts(t, "--score-style", writeConformanceFixture(t, mustStringNumberExport(t, song)), &got)
		if !reflect.DeepEqual(got, want) {
			t.Fatal("bracket mode changed another policy")
		}
	}
	conformanceIndependentClaim(t, "field:ScoreStyle.Brackets", claimImportModel("stylesheet", scoreDisplayCase, scoreDisplayValue)...)
}
func TestScoreDisplayMalformedAndAuthoredBounds(t *testing.T) {
	song := scoreDisplaySong(t)
	xml := string(conformanceBarreGPIF(t, mustStringNumberExport(t, song)))
	var records []binaryStyleRecord
	for _, c := range displayBooleanCases() {
		records = append(records, scoreDisplayInteger(c.key, 1))
	}
	for _, key := range []string{"System/bracketExtendMode", "System/trackNameModeSingle", "System/trackNameModeMulti"} {
		records = append(records, binaryStyleRecord{key: key, kind: 0, value: []byte{1}})
		for _, value := range []uint32{3, 255, 0x7fffffff, 0xffffffff} {
			records = append(records, scoreDisplayInteger(key, value))
		}
	}
	for _, record := range records {
		data := encodeDisplayTestRecords([]binaryStyleRecord{record})
		archive := conformanceBackingArchive(t, xml, map[string][]byte{"Content/BinaryStylesheet": data})
		parsed, err := Parse(archive)
		result, optionsErr := ParseWithOptions(archive, ParseOptions{})
		if parsed != nil || result != nil || err == nil || optionsErr == nil || !strings.Contains(err.Error(), record.key) {
			t.Fatalf("invalid display record accepted: %v/%v", err, optionsErr)
		}
	}
	for _, value := range []uint8{3, 255} {
		for _, which := range []int{0, 1, 2} {
			song := scoreDisplaySong(t)
			code := "score.style.track-name-mode"
			switch which {
			case 0:
				song.Style.Brackets = ptrTo(BracketMode(value))
				code = "score.style.bracket-mode"
			case 1:
				song.Style.SingleTrackNameMode = ptrTo(TrackNameSystemMode(value))
			default:
				song.Style.MultiTrackNameMode = ptrTo(TrackNameSystemMode(value))
			}
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{AllowedCodes: []string{"gp8.reject." + code}}})
			if err == nil || len(data) != 0 || len(report.Entries) != 1 || report.Entries[0].Code != "gp8.reject."+code {
				t.Fatalf("undefined style enum accepted: %v %#v", err, report)
			}
		}
	}
}

func TestAlphaTabTrackNamePolicyPrecedence(t *testing.T) {
	requireAlphaTabConformance(t)
	xml := string(conformanceBarreGPIF(t, mustStringNumberExport(t, scoreDisplaySong(t))))
	for _, multi := range []bool{false, true} {
		for _, visible := range []*bool{nil, ptrTo(false), ptrTo(true)} {
			for _, mode := range []*TrackNameSystemMode{nil, ptrTo(TrackNamesFirstSystem), ptrTo(TrackNamesFirstSystemEachPage), ptrTo(TrackNamesAllSystems)} {
				for _, reverse := range []bool{false, true} {
					suffix, consumer := "Single", "singleTrackTrackNamePolicy"
					if multi {
						suffix, consumer = "Multi", "multiTrackTrackNamePolicy"
					}
					var records []binaryStyleRecord
					if visible != nil {
						b := byte(0)
						if *visible {
							b = 1
						}
						records = append(records, binaryStyleRecord{key: "System/showTrackName" + suffix, kind: 0, value: []byte{b}})
					}
					if mode != nil {
						records = append(records, scoreDisplayInteger("System/trackNameMode"+suffix, uint32(*mode)))
					}
					if reverse {
						slices.Reverse(records)
					}
					source := conformanceBackingArchive(t, xml, map[string][]byte{"Content/BinaryStylesheet": encodeDisplayTestRecords(records)})
					song, err := Parse(source)
					if err != nil {
						t.Fatal(err)
					}
					gotMode, gotVisible := song.Style.SingleTrackNameMode, song.Style.SingleTrackNamesVisible
					if multi {
						gotMode, gotVisible = song.Style.MultiTrackNameMode, song.Style.MultiTrackNamesVisible
					}
					if !reflect.DeepEqual(gotMode, mode) || !reflect.DeepEqual(gotVisible, visible) {
						t.Fatal("source precedence erased authored fields")
					}
					wantPolicy := float64(1)
					if mode != nil && *mode == TrackNamesAllSystems {
						wantPolicy = 2
					}
					if visible != nil && !*visible {
						wantPolicy = 0
					}
					var input, output scoreStyleFacts
					readAlphaTabOracleFacts(t, "--score-style", writeConformanceFixture(t, source), &input)
					if input.Other[consumer] != wantPolicy {
						t.Fatalf("raw precedence %s=%v want %v", consumer, input.Other[consumer], wantPolicy)
					}
					cleanNotationProvenance(song)
					song.PanAutomations = nil
					song.VolumeAutomations = nil
					for i := range song.Tracks {
						song.Tracks[i].UseRse = false
					}
					data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.normalize.track-name-page-policy"}}})
					if err != nil {
						t.Fatal(err)
					}
					wantCodes := []string{}
					if mode != nil && *mode == TrackNamesFirstSystemEachPage {
						wantCodes = append(wantCodes, "gp8.normalize.track-name-page-policy")
					}
					if !reflect.DeepEqual(reportCodes(report), wantCodes) {
						t.Fatalf("scoped policy report=%#v", report)
					}
					readAlphaTabOracleFacts(t, "--score-style", writeConformanceFixture(t, data), &output)
					if !reflect.DeepEqual(output, input) {
						t.Fatal("source/output precedence or another policy changed")
					}
				}
			}
		}
	}
	// The pinned raw map gives a duplicate key its last value before applying policies.
	records := []binaryStyleRecord{{key: "System/showTrackNameSingle", kind: 0, value: []byte{0}}, scoreDisplayInteger("System/trackNameModeSingle", 2), {key: "System/showTrackNameSingle", kind: 0, value: []byte{1}}}
	source := conformanceBackingArchive(t, xml, map[string][]byte{"Content/BinaryStylesheet": encodeDisplayTestRecords(records)})
	song, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	if !*song.Style.SingleTrackNamesVisible {
		t.Fatal("last duplicate visibility did not win")
	}
	var input, output scoreStyleFacts
	readAlphaTabOracleFacts(t, "--score-style", writeConformanceFixture(t, source), &input)
	readAlphaTabOracleFacts(t, "--score-style", writeConformanceFixture(t, mustStringNumberExport(t, song)), &output)
	if input.Other["singleTrackTrackNamePolicy"] != float64(2) || !reflect.DeepEqual(input, output) {
		t.Fatal("duplicate raw-key precedence changed")
	}
}
func encodeDisplayTestRecords(records []binaryStyleRecord) []byte {
	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, uint32(len(records)))
	for _, record := range records {
		data = append(data, byte(len(record.key)))
		data = append(data, record.key...)
		data = append(data, record.kind)
		data = append(data, record.value...)
	}
	return data
}

func TestScoreDisplayParsedOwnership(t *testing.T) {
	song := scoreDisplaySong(t)
	if song.Style.SingleTrackNamesVisible == song.Style.MultiTrackNamesVisible || song.Style.SingleTrackNameMode == song.Style.MultiTrackNameMode {
		t.Fatal("parsed view requests share storage")
	}
	multi := *song.Style.MultiTrackNamesVisible
	mode := *song.Style.MultiTrackNameMode
	*song.Style.SingleTrackNamesVisible = !*song.Style.SingleTrackNamesVisible
	*song.Style.SingleTrackNameMode = TrackNamesFirstSystem
	if *song.Style.MultiTrackNamesVisible != multi || *song.Style.MultiTrackNameMode != mode {
		t.Fatal("single-view edit changed multi-view request")
	}
}
