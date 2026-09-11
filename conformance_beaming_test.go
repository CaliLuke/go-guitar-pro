// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const conformanceBeamingCase = "M10-BEAMING"
const conformanceBeamingValue = "custom groups and independent beat overrides"

type conformanceBeamingRuleFact struct {
	Duration int   `json:"duration"`
	Groups   []int `json:"groups"`
}

type conformanceBeamingMasterBarFact struct {
	Bar   int                          `json:"bar"`
	Rules []conformanceBeamingRuleFact `json:"rules"`
}

type conformanceBeamingBeatFact struct {
	Track     int    `json:"track"`
	Staff     int    `json:"staff"`
	Bar       int    `json:"bar"`
	Voice     int    `json:"voice"`
	Beat      int    `json:"beat"`
	Mode      string `json:"mode"`
	Invert    bool   `json:"invert"`
	Direction string `json:"direction"`
}

type conformanceBeamingFacts struct {
	MasterBars []conformanceBeamingMasterBarFact `json:"masterBars"`
	Beats      []conformanceBeamingBeatFact      `json:"beats"`
}

func TestConformanceBeaming(t *testing.T) {
	runConformanceBeaming(newConformanceRun(t))
}

func runConformanceBeaming(run *conformanceRun) {
	t := run.t
	source := parseTestFixture(t, "testdata/gp7/tap.gp")
	rules := source.MeasureHeaders[0].BeamingRules
	wantRules := &BeamingRules{Duration: NoteValue(DurationEighth), Groups: []int{2, 2, 2}}
	run.ClaimPrimary(claimSite("beaming", "import", conformanceBeamingCase, conformanceBeamingValue)).Preserved("MeasureHeader.BeamingRules", rules, wantRules)
	run.Preserved("BeamingRules.Duration", rules.Duration, NoteValue(DurationEighth))
	run.Preserved("BeamingRules.Groups", rules.Groups, []int{2, 2, 2})
	for beatIndex, beat := range source.Tracks[0].Measures[0].Voices[0].Beats {
		run.Preserved("Beat.PreferredBeamDirection", beat.PreferredBeamDirection, VoiceDirectionUp)
		run.Preserved("Beat.BeamingMode", beat.BeamingMode, BeatBeamingAuto)
		run.Preserved("Beat.InvertBeamDirection", beat.InvertBeamDirection, false)
		if beatIndex == 0 {
			rules.Groups[0] = 3
			if source.MeasureHeaders[0].TimeSignature.Beams[0] != 2 {
				t.Fatal("editing custom beaming rules changed the meter compatibility grouping")
			}
		}
	}
	if rules.Groups[0] != 3 || !slices.Equal(rules.Groups[1:], []int{2, 2}) {
		t.Fatalf("edited source rules = %#v", rules)
	}
	owned := conformanceTwoBarBeamingSong(t)
	ownedData, ownedReport, err := ExportWithReport(owned, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(ownedReport.Entries) != 0 {
		t.Fatalf("owned beaming export = %v, %#v", err, ownedReport.Entries)
	}
	ownedWire := conformanceWireDocument(t, ownedData)
	if ownedWire.MasterBars.MasterBars[0].XProperties != nil || ownedWire.MasterBars.MasterBars[1].XProperties == nil {
		t.Fatalf("master-bar beaming ownership wire = %#v, want only bar 1", ownedWire.MasterBars.MasterBars)
	}
	ownedRoundTrip, err := Parse(ownedData)
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("MeasureHeader.BeamingRules", ownedRoundTrip.MeasureHeaders[0].BeamingRules, (*BeamingRules)(nil))
	run.Preserved("MeasureHeader.BeamingRules", ownedRoundTrip.MeasureHeaders[1].BeamingRules, owned.MeasureHeaders[1].BeamingRules)

	legacy := parseTestFixture(t, "testdata/gp5/beaming-mode.gp5")
	legacyCases := []struct {
		bar, beat int
		mode      BeatBeamingMode
	}{
		{1, 1, BeatBeamingForceMerge},
		{2, 0, BeatBeamingForceSplit},
		{2, 2, BeatBeamingForceSplit},
		{3, 0, BeatBeamingForceSplitSecondary},
	}
	for _, test := range legacyCases {
		got := legacy.Tracks[0].Measures[test.bar].Voices[0].Beats[test.beat].BeamingMode
		if got != test.mode {
			t.Errorf("GP5 bar %d beat %d beaming mode = %d, want %d", test.bar, test.beat, got, test.mode)
		}
	}
	for _, beat := range legacy.Tracks[0].Measures[4].Voices[0].Beats[:2] {
		if beat.PreferredBeamDirection != VoiceDirectionDown {
			t.Errorf("GP5 down-stem direction = %d", beat.PreferredBeamDirection)
		}
	}
	for _, beat := range legacy.Tracks[0].Measures[5].Voices[0].Beats[:2] {
		if beat.PreferredBeamDirection != VoiceDirectionUp {
			t.Errorf("GP5 up-stem direction = %d", beat.PreferredBeamDirection)
		}
	}

	song := semanticExportProbeSong(t)
	song.MeasureHeaders[0].BeamingRules = &BeamingRules{Duration: NoteValue(DurationEighth), Groups: []int{3, 2, 3}}
	voice := &song.Tracks[0].Measures[0].Voices[0]
	base := voice.Beats[0]
	voice.Beats = make([]Beat, 4)
	for index := range voice.Beats {
		voice.Beats[index] = base
		voice.Beats[index].Notes = slices.Clone(base.Notes)
	}
	voice.Beats[0].BeamingMode = BeatBeamingForceSplit
	voice.Beats[0].PreferredBeamDirection = VoiceDirectionUp
	voice.Beats[1].BeamingMode = BeatBeamingForceMerge
	voice.Beats[1].PreferredBeamDirection = VoiceDirectionDown
	voice.Beats[2].BeamingMode = BeatBeamingForceSplitSecondary
	voice.Beats[3].InvertBeamDirection = true
	if finalizeErr := FinalizeSong(song); finalizeErr != nil {
		t.Fatal(finalizeErr)
	}

	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("beaming export = %v, %#v", err, report.Entries)
	}
	run.ClaimReport(claimSite("beaming", "export", conformanceBeamingCase, conformanceBeamingValue)).Report(conformanceBeamingCase, reportCodes(report), []string{})
	wire := conformanceWireDocument(t, data)
	run.ClaimSerialization(claimSite("beaming", "export", conformanceBeamingCase, conformanceBeamingValue)).Wire("gpifMasterBar.XProperties", conformanceBeamingRuleWire(wire.MasterBars.MasterBars[0].XProperties), []string{
		"1124139010=8", "1124139264=3", "1124139265=2", "1124139266=3",
	})
	run.Wire("gpifXProperties.Properties", len(wire.MasterBars.MasterBars[0].XProperties.Properties), 4)
	run.Wire("gpifXProperty.ID", wire.MasterBars.MasterBars[0].XProperties.Properties[0].ID, gpifMasterBarBeamingDurationID)
	run.Wire("gpifXProperty.Int", *wire.MasterBars.MasterBars[0].XProperties.Properties[0].Int, "8")
	run.Wire("gpifBeat.XProperties", conformanceBeamingBeatXPropertyWire(wire.Beats.Beats), []string{
		"1124204546=2", "1124204546=1", "1124204552=1", "1124204545=1",
	})
	run.Wire("gpifBeat.TransposedPitchStemOrientation", conformanceBeamingStemWire(wire.Beats.Beats, false), []string{"Upward", "Downward", "", ""})
	run.Wire("gpifBeat.UserTransposedPitchStemOrientation", conformanceBeamingStemWire(wire.Beats.Beats, true), []string{"Upward", "Downward", "", ""})

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	roundRules := roundTrip.MeasureHeaders[0].BeamingRules
	run.ClaimPrimary(claimSite("beaming", "model", conformanceBeamingCase, conformanceBeamingValue), claimSite("beaming", "export", conformanceBeamingCase, conformanceBeamingValue)).Preserved("MeasureHeader.BeamingRules", roundRules, song.MeasureHeaders[0].BeamingRules)
	roundBeats := roundTrip.Tracks[0].Measures[0].Voices[0].Beats
	for index, want := range []BeatBeamingMode{BeatBeamingForceSplit, BeatBeamingForceMerge, BeatBeamingForceSplitSecondary, BeatBeamingAuto} {
		run.Preserved("Beat.BeamingMode", roundBeats[index].BeamingMode, want)
	}
	run.Preserved("Beat.InvertBeamDirection", []bool{roundBeats[0].InvertBeamDirection, roundBeats[3].InvertBeamDirection}, []bool{false, true})
	run.Preserved("Beat.PreferredBeamDirection", []VoiceDirection{roundBeats[0].PreferredBeamDirection, roundBeats[1].PreferredBeamDirection}, []VoiceDirection{VoiceDirectionUp, VoiceDirectionDown})

	for index, mode := range []BeatBeamingMode{BeatBeamingAuto, BeatBeamingForceSplit, BeatBeamingForceMerge, BeatBeamingForceSplitSecondary} {
		run.Enum([]string{
			"BeatBeamingMode.BeatBeamingAuto", "BeatBeamingMode.BeatBeamingForceSplit", "BeatBeamingMode.BeatBeamingForceMerge", "BeatBeamingMode.BeatBeamingForceSplitSecondary",
		}[index], mode, BeatBeamingMode(index))
	}
	if !reflect.DeepEqual(song.MeasureHeaders[0].BeamingRules.Groups, []int{3, 2, 3}) {
		t.Fatal("export mutated authored beaming groups")
	}
}

func conformanceBeamingRuleWire(properties *gpifXProperties) []string {
	if properties == nil {
		return nil
	}
	result := make([]string, 0, len(properties.Properties))
	for _, property := range properties.Properties {
		result = append(result, property.ID+"="+gpifXPropertyText(property))
	}
	return result
}

func conformanceBeamingBeatXPropertyWire(beats []gpifBeat) []string {
	result := make([]string, len(beats))
	for index := range beats {
		result[index] = strings.Join(conformanceBeamingRuleWire(beats[index].XProperties), ",")
	}
	return result
}

func conformanceBeamingStemWire(beats []gpifBeat, user bool) []string {
	result := make([]string, len(beats))
	for index := range beats {
		result[index] = beats[index].TransposedPitchStemOrientation
		if user {
			result[index] = beats[index].UserTransposedPitchStemOrientation
		}
	}
	return result
}

func TestGPIFBeamingDiagnostics(t *testing.T) {
	archive, err := os.ReadFile("testdata/gp7/tap.gp")
	if err != nil {
		t.Fatal(err)
	}
	sourceArchive, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatal(err)
	}
	source := string(readZipMember(t, sourceArchive, "Content/score.gpif"))
	for _, test := range []struct {
		name, old, replacement, code string
	}{
		{"duration", "<Int>8</Int>", "<Int>3</Int>", "GPIF.MasterBar.Beaming.Duration.InvalidValue"},
		{"group zero", "<Int>2</Int>", "<Int>0</Int>", "GPIF.MasterBar.Beaming.Group.InvalidValue"},
		{"missing group payload", "<Int>2</Int>", "", "GPIF.MasterBar.Beaming.Group.InvalidValue"},
		{"invalid stem", "<TransposedPitchStemOrientation>Upward</TransposedPitchStemOrientation>", "<TransposedPitchStemOrientation>Sideways</TransposedPitchStemOrientation>", "GPIF.Beat.TransposedPitchStemOrientation.InvalidValue"},
		{"invalid beat property", "<Beat id=\"0\">", "<Beat id=\"0\"><XProperties><XProperty id=\"1124204546\"><Int>9</Int></XProperty></XProperties>", "GPIF.Beat.Beaming.XProperty.InvalidValue"},
	} {
		t.Run(test.name, func(t *testing.T) {
			mutated := strings.Replace(source, test.old, test.replacement, 1)
			result, parseErr := ParseWithOptions(conformanceGPIFArchive(t, mutated), ParseOptions{})
			if parseErr != nil || !slices.ContainsFunc(result.Diagnostics, func(diagnostic ParseDiagnostic) bool { return diagnostic.Code == test.code }) {
				t.Fatalf("diagnostics = %#v, %v; want %s", result.Diagnostics, parseErr, test.code)
			}
			_, strictErr := ParseWithOptions(conformanceGPIFArchive(t, mutated), ParseOptions{Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticInvalidData}})
			var strict *StrictParseError
			if !errors.As(strictErr, &strict) {
				t.Fatalf("strict parse = %v, want StrictParseError", strictErr)
			}
		})
	}

	for _, test := range []struct {
		name       string
		properties string
		want       BeatBeamingMode
	}{
		{"secondary then merge", `<XProperty id="1124204552"><Int>1</Int></XProperty><XProperty id="1124204546"><Int>1</Int></XProperty>`, BeatBeamingForceMerge},
		{"secondary then split", `<XProperty id="1124204552"><Int>1</Int></XProperty><XProperty id="1124204546"><Int>2</Int></XProperty>`, BeatBeamingForceSplit},
		{"merge then secondary", `<XProperty id="1124204546"><Int>1</Int></XProperty><XProperty id="1124204552"><Int>1</Int></XProperty>`, BeatBeamingForceSplitSecondary},
		{"split then secondary", `<XProperty id="1124204546"><Int>2</Int></XProperty><XProperty id="1124204552"><Int>1</Int></XProperty>`, BeatBeamingForceSplit},
	} {
		t.Run(test.name, func(t *testing.T) {
			mutated := strings.Replace(source, `<Beat id="0">`, `<Beat id="0"><XProperties>`+test.properties+`</XProperties>`, 1)
			song, parseErr := Parse(conformanceGPIFArchive(t, mutated))
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			if got := song.Tracks[0].Measures[0].Voices[0].Beats[0].BeamingMode; got != test.want {
				t.Fatalf("ordered XProperties mode = %d, want %d", got, test.want)
			}
		})
	}

	for _, test := range []struct {
		name string
		edit func(*Song)
		code string
	}{
		{"invalid duration", func(song *Song) { song.MeasureHeaders[0].BeamingRules = &BeamingRules{Duration: 3, Groups: []int{2}} }, "score.measure.beaming-rules"},
		{"empty groups", func(song *Song) { song.MeasureHeaders[0].BeamingRules = &BeamingRules{Duration: 8} }, "score.measure.beaming-rules"},
		{"zero group", func(song *Song) { song.MeasureHeaders[0].BeamingRules = &BeamingRules{Duration: 8, Groups: []int{0}} }, "score.measure.beaming-rules"},
		{"invalid mode", func(song *Song) { song.Tracks[0].Measures[0].Voices[0].Beats[0].BeamingMode = 4 }, "score.beat.beaming-mode"},
		{"invalid direction", func(song *Song) { song.Tracks[0].Measures[0].Voices[0].Beats[0].PreferredBeamDirection = 3 }, "score.beat.beam-direction"},
	} {
		t.Run(test.name, func(t *testing.T) {
			probe := semanticExportProbeSong(t)
			test.edit(probe)
			if !slices.ContainsFunc(ValidateSong(probe), func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == test.code }) {
				t.Fatalf("validation = %#v, want %s", ValidateSong(probe), test.code)
			}
			if data, _, exportErr := ExportWithReport(probe, ExportFormatGP8, ExportOptions{}); exportErr == nil || len(data) != 0 {
				t.Fatalf("invalid beaming export = %d bytes, %v", len(data), exportErr)
			}
		})
	}
}

func TestGPIFXPropertyAuditClosure(t *testing.T) {
	brushData, err := os.ReadFile("testdata/gp7/brush.gp")
	if err != nil {
		t.Fatal(err)
	}
	brushResult, err := ParseWithOptions(brushData, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	const brushCode = "GPIF.Beat.XProperty.BrushDuration"
	if got := countParseDiagnostics(brushResult.Diagnostics, brushCode); got != 4 {
		t.Fatalf("brush duration XProperty diagnostics = %d, want 4", got)
	}
	for _, diagnostic := range brushResult.Diagnostics {
		if diagnostic.Code == brushCode && diagnostic.Feature != "brush" {
			t.Fatalf("brush duration diagnostic feature = %q, want brush", diagnostic.Feature)
		}
	}
	_, strictErr := ParseWithOptions(brushData, ParseOptions{Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticUnknownSyntax}})
	var strict *StrictParseError
	if !errors.As(strictErr, &strict) || countParseDiagnostics(strict.Diagnostics, brushCode) != 4 {
		t.Fatalf("strict brush XProperty diagnostics = %#v, %v; want 4 %s", strict, strictErr, brushCode)
	}

	tapData, err := os.ReadFile("testdata/gp7/tap.gp")
	if err != nil {
		t.Fatal(err)
	}
	tapArchive, err := zip.NewReader(bytes.NewReader(tapData), int64(len(tapData)))
	if err != nil {
		t.Fatal(err)
	}
	tapSource := string(readZipMember(t, tapArchive, "Content/score.gpif"))
	knownBeat := strings.Replace(tapSource, `<Beat id="0">`, `<Beat id="0"><XProperties><XProperty id="1124204546"><Int>1</Int></XProperty></XProperties>`, 1)
	knownResult, err := ParseWithOptions(conformanceGPIFArchive(t, knownBeat), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"GPIF.Beat.XProperty.Unknown", "GPIF.Beat.XProperty.BrushDuration", "GPIF.MasterBar.XProperty.Unknown"} {
		if countParseDiagnostics(knownResult.Diagnostics, code) != 0 {
			t.Fatalf("known beaming XProperties produced %s: %#v", code, knownResult.Diagnostics)
		}
	}

	for _, test := range []struct {
		name, source, code string
	}{
		{"future beat", strings.Replace(tapSource, `<Beat id="0">`, `<Beat id="0"><XProperties><XProperty id="999999999"><Int>1</Int></XProperty></XProperties>`, 1), "GPIF.Beat.XProperty.Unknown"},
		{"future master bar", strings.Replace(tapSource, `id="1124139010"`, `id="999999998"`, 1), "GPIF.MasterBar.XProperty.Unknown"},
	} {
		t.Run(test.name, func(t *testing.T) {
			data := conformanceGPIFArchive(t, test.source)
			result, parseErr := ParseWithOptions(data, ParseOptions{})
			if parseErr != nil || countParseDiagnostics(result.Diagnostics, test.code) != 1 {
				t.Fatalf("unknown XProperty diagnostics = %#v, %v; want one %s", result.Diagnostics, parseErr, test.code)
			}
			_, parseErr = ParseWithOptions(data, ParseOptions{Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticUnknownSyntax}})
			var strict *StrictParseError
			if !errors.As(parseErr, &strict) || countParseDiagnostics(strict.Diagnostics, test.code) != 1 {
				t.Fatalf("strict unknown XProperty = %#v, %v; want one %s", strict, parseErr, test.code)
			}
		})
	}
}

func countParseDiagnostics(diagnostics []ParseDiagnostic, code string) int {
	count := 0
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			count++
		}
	}
	return count
}

func TestBeamingRuleBoundaries(t *testing.T) {
	for _, test := range []struct {
		name     string
		duration NoteValue
		groups   []int
	}{
		{"minimum", 1, []int{1}},
		{"maximum", 256, slices.Repeat([]int{int(^uint32(0) >> 1)}, 32)},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := semanticExportProbeSong(t)
			song.MeasureHeaders[0].BeamingRules = &BeamingRules{Duration: test.duration, Groups: test.groups}
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
			if err != nil || len(report.Entries) != 0 {
				t.Fatalf("boundary export = %v, %#v", err, report.Entries)
			}
			roundTrip, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(roundTrip.MeasureHeaders[0].BeamingRules, song.MeasureHeaders[0].BeamingRules) {
				t.Fatalf("boundary rules = %#v, want %#v", roundTrip.MeasureHeaders[0].BeamingRules, song.MeasureHeaders[0].BeamingRules)
			}
		})
	}
}

func TestAlphaTabPreservesBeaming(t *testing.T) {
	requireAlphaTabConformance(t)
	fixture := "testdata/gp7/tap.gp"
	var source conformanceBeamingFacts
	readAlphaTabOracleFacts(t, "--beaming", fixture, &source)
	if !reflect.DeepEqual(source.MasterBars, []conformanceBeamingMasterBarFact{{Bar: 0, Rules: []conformanceBeamingRuleFact{{Duration: 8, Groups: []int{2, 2, 2}}}}}) {
		t.Fatalf("AlphaTab source master-bar beaming = %#v", source.MasterBars)
	}
	fixtureData, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		properties string
		want       string
	}{
		{"secondary then merge", `<XProperty id="1124204552"><Int>1</Int></XProperty><XProperty id="1124204546"><Int>1</Int></XProperty>`, "forcemergewithnext"},
		{"secondary then split", `<XProperty id="1124204552"><Int>1</Int></XProperty><XProperty id="1124204546"><Int>2</Int></XProperty>`, "forcesplittonext"},
		{"merge then secondary", `<XProperty id="1124204546"><Int>1</Int></XProperty><XProperty id="1124204552"><Int>1</Int></XProperty>`, "forcesplitonsecondarytonext"},
		{"split then secondary", `<XProperty id="1124204546"><Int>2</Int></XProperty><XProperty id="1124204552"><Int>1</Int></XProperty>`, "forcesplittonext"},
	} {
		t.Run(test.name, func(t *testing.T) {
			mutated := rewriteConformanceGPIF(t, fixtureData, func(gpif string) string {
				return strings.Replace(gpif, `<Beat id="0">`, `<Beat id="0"><XProperties>`+test.properties+`</XProperties>`, 1)
			})
			var facts conformanceBeamingFacts
			readAlphaTabOracleFacts(t, "--beaming", writeConformanceFixture(t, mutated), &facts)
			if len(facts.Beats) == 0 || facts.Beats[0].Mode != test.want {
				t.Fatalf("AlphaTab ordered XProperties = %#v, want first mode %s", facts.Beats, test.want)
			}
		})
	}
	song := parseTestFixture(t, fixture)
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var output conformanceBeamingFacts
	readAlphaTabOracleFacts(t, "--beaming", writeConformanceFixture(t, data), &output)
	if !reflect.DeepEqual(output, source) {
		t.Fatalf("AlphaTab output beaming = %#v, want %#v", output, source)
	}

	rules := song.MeasureHeaders[0].BeamingRules
	rules.Groups = []int{3, 3}
	beats := song.Tracks[0].Measures[0].Voices[0].Beats
	beats[0].BeamingMode = BeatBeamingForceSplit
	beats[0].InvertBeamDirection = true
	beats[1].BeamingMode = BeatBeamingForceMerge
	beats[1].PreferredBeamDirection = VoiceDirectionDown
	beats[2].BeamingMode = BeatBeamingForceSplitSecondary
	edited, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	readAlphaTabOracleFacts(t, "--beaming", writeConformanceFixture(t, edited), &output)
	want := conformanceBeamingFacts{
		MasterBars: []conformanceBeamingMasterBarFact{{Bar: 0, Rules: []conformanceBeamingRuleFact{{Duration: 8, Groups: []int{3, 3}}}}},
		Beats: []conformanceBeamingBeatFact{
			{Track: 0, Staff: 0, Bar: 0, Voice: 0, Beat: 0, Mode: "forcesplittonext", Invert: true, Direction: "up"},
			{Track: 0, Staff: 0, Bar: 0, Voice: 0, Beat: 1, Mode: "forcemergewithnext", Direction: "down"},
			{Track: 0, Staff: 0, Bar: 0, Voice: 0, Beat: 2, Mode: "forcesplitonsecondarytonext", Direction: "up"},
		},
	}
	if !reflect.DeepEqual(output, want) {
		t.Fatalf("AlphaTab edited beaming = %#v, want %#v", output, want)
	}

	invertOnly := semanticExportProbeSong(t)
	invertOnly.Tracks[0].Measures[0].Voices[0].Beats[0].InvertBeamDirection = true
	invertData, err := Export(invertOnly, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	readAlphaTabOracleFacts(t, "--beaming", writeConformanceFixture(t, invertData), &output)
	invertWant := conformanceBeamingFacts{
		MasterBars: []conformanceBeamingMasterBarFact{},
		Beats: []conformanceBeamingBeatFact{{
			Track: 0, Staff: 0, Bar: 0, Voice: 0, Beat: 0,
			Mode: "auto", Invert: true, Direction: "none",
		}},
	}
	if !reflect.DeepEqual(output, invertWant) {
		t.Fatalf("AlphaTab invert-only beaming = %#v, want %#v", output, invertWant)
	}

	owned := conformanceTwoBarBeamingSong(t)
	ownedData, err := Export(owned, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	readAlphaTabOracleFacts(t, "--beaming", writeConformanceFixture(t, ownedData), &output)
	ownedWant := conformanceBeamingFacts{
		MasterBars: []conformanceBeamingMasterBarFact{{Bar: 1, Rules: []conformanceBeamingRuleFact{{Duration: 8, Groups: []int{3, 2, 3}}}}},
		Beats:      []conformanceBeamingBeatFact{},
	}
	if !reflect.DeepEqual(output, ownedWant) {
		t.Fatalf("AlphaTab exact master-bar ownership = %#v, want %#v", output, ownedWant)
	}
	conformanceIndependentClaim(t, "field:MeasureHeader.BeamingRules", claimAllStages("beaming", conformanceBeamingCase, conformanceBeamingValue)...)
}

func conformanceTwoBarBeamingSong(t *testing.T) *Song {
	t.Helper()
	song := semanticExportProbeSong(t)
	secondHeader := song.MeasureHeaders[0]
	secondHeader.Number = 2
	secondHeader.BeamingRules = &BeamingRules{Duration: NoteValue(DurationEighth), Groups: []int{3, 2, 3}}
	song.MeasureHeaders = append(song.MeasureHeaders, secondHeader)
	secondMeasure := song.Tracks[0].Measures[0]
	secondMeasure.Number = 2
	secondMeasure.HeaderIndex = 1
	secondMeasure.Voices = slices.Clone(secondMeasure.Voices)
	secondMeasure.Voices[0].Beats = slices.Clone(secondMeasure.Voices[0].Beats)
	secondMeasure.Voices[0].Beats[0].Notes = slices.Clone(secondMeasure.Voices[0].Beats[0].Notes)
	song.Tracks[0].Measures = append(song.Tracks[0].Measures, secondMeasure)
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("two-bar beaming diagnostics = %#v", diagnostics)
	}
	return song
}
