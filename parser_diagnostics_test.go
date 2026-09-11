// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestParseWithOptionsReportsGPIFContentLoss(t *testing.T) {
	tests := []struct {
		name         string
		mutate       func(string) string
		kind         ParseDiagnosticKind
		feature      string
		pathContains string
	}{
		{
			name: "unknown note property",
			mutate: func(gpif string) string {
				return insertFirstNoteProperty(t, gpif, `<Property name="FutureTechnique"><Enable/></Property>`)
			},
			kind: ParseDiagnosticUnknownSyntax, feature: "note-and-beat-semantics", pathContains: "FutureTechnique",
		},
		{
			name: "invalid transposition nominal key",
			mutate: func(gpif string) string {
				return strings.Replace(gpif, "<Staves>", "<PartSounding><NominalKey>H</NominalKey><TranspositionPitch>0</TranspositionPitch></PartSounding><Staves>", 1)
			},
			kind: ParseDiagnosticUnsupportedFeature, feature: "transposition", pathContains: "NominalKey",
		},
		{
			name: "unsupported harmonic enum",
			mutate: func(gpif string) string {
				return insertFirstNoteProperty(t, gpif, `<Property name="HarmonicType"><HType>FutureHarmonic</HType></Property>`)
			},
			kind: ParseDiagnosticUnsupportedFeature, feature: "harmonics", pathContains: "HarmonicType",
		},
		{
			name: "invalid rhythm reference",
			mutate: func(gpif string) string {
				return replaceFirstGPIFReference(t, gpif, "<Rhythm ref=\"", "missing-rhythm")
			},
			kind: ParseDiagnosticInvalidData, feature: "rhythm", pathContains: "Rhythm",
		},
		{
			name: "unknown note element",
			mutate: func(gpif string) string {
				return insertFirstGPIFObjectChild(t, gpif, "<Notes>", "</Note>", "<FutureTechnique/>")
			},
			kind: ParseDiagnosticUnknownSyntax, feature: "note-and-beat-semantics", pathContains: "FutureTechnique",
		},
		{
			name: "unknown note attribute",
			mutate: func(gpif string) string {
				start := strings.LastIndex(gpif, "<Notes>")
				if start < 0 {
					t.Fatal("GPIF has no Notes element")
				}
				return gpif[:start] + strings.Replace(gpif[start:], `<Note id="`, `<Note future="x" id="`, 1)
			},
			kind: ParseDiagnosticUnknownSyntax, feature: "note-and-beat-semantics", pathContains: "@future",
		},
		{
			name: "unsupported beat wah",
			mutate: func(gpif string) string {
				return insertFirstGPIFObjectChild(t, gpif, "<Beats>", "</Beat>", "<Wah>UnknownPedal</Wah>")
			},
			kind: ParseDiagnosticUnsupportedFeature, feature: "note-and-beat-semantics", pathContains: "Wah",
		},
		{
			name: "invalid beat dynamic",
			mutate: func(gpif string) string {
				return insertFirstGPIFObjectChild(t, gpif, "<Beats>", "</Beat>", "<Dynamic>FutureDynamic</Dynamic>")
			},
			kind: ParseDiagnosticUnsupportedFeature, feature: "note-and-beat-semantics", pathContains: "Dynamic",
		},
		{
			name: "invalid pick stroke payload",
			mutate: func(gpif string) string {
				return insertFirstGPIFObjectChild(t, gpif, "<Beats>", "</Beat>", "<Properties><Property name=\"PickStroke\"/></Properties>")
			},
			kind: ParseDiagnosticInvalidData, feature: "note-and-beat-semantics", pathContains: "PickStroke",
		},
		{
			name: "malformed whammy property",
			mutate: func(gpif string) string {
				return insertFirstGPIFObjectChild(t, gpif, "<Beats>", "</Beat>", "<Properties><Property name=\"WhammyBar\"><Enable/></Property><Property name=\"WhammyBarMiddleValue\"/></Properties>")
			},
			kind: ParseDiagnosticInvalidData, feature: "note-and-beat-semantics", pathContains: "WhammyBarMiddleValue",
		},
		{
			name: "invalid chord reference",
			mutate: func(gpif string) string {
				return insertFirstGPIFObjectChild(t, gpif, "<Beats>", "</Beat>", "<Chord>missing-chord</Chord>")
			},
			kind: ParseDiagnosticInvalidData, feature: "note-and-beat-semantics", pathContains: "Chord",
		},
		{
			name: "invalid backing asset reference",
			mutate: func(gpif string) string {
				return strings.Replace(gpif, "<MasterTrack>", "<BackingTrack><Enabled>true</Enabled><Source>Local</Source><AssetId>missing-asset</AssetId><FramePadding>0</FramePadding></BackingTrack><MasterTrack>", 1)
			},
			kind: ParseDiagnosticInvalidData, feature: "score-core", pathContains: "AssetId",
		},
		{
			name: "unknown score element",
			mutate: func(gpif string) string {
				return strings.Replace(gpif, "<Score>", "<Score><FutureScoreField/>", 1)
			},
			kind: ParseDiagnosticUnknownSyntax, feature: "score-core", pathContains: "FutureScoreField",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := diagnosticGP8Fixture(t, test.mutate)
			result, err := ParseWithOptions(data, ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if result.Song == nil {
				t.Fatal("song = nil")
			}
			diagnostic := findParseDiagnostic(result.Diagnostics, test.kind, test.feature)
			if diagnostic == nil {
				t.Fatalf("diagnostics = %#v, want %s for %s", result.Diagnostics, test.kind, test.feature)
			}
			if diagnostic.Format != "GP8" {
				t.Errorf("diagnostic format = %q, want GP8", diagnostic.Format)
			}
			if !strings.Contains(diagnostic.SourcePath, test.pathContains) || diagnostic.Reason == "" {
				t.Errorf("diagnostic = %#v, want source path containing %q and a reason", diagnostic, test.pathContains)
			}
			if diagnostic.Code == "" {
				t.Errorf("diagnostic = %#v, want stable source code", diagnostic)
			}
			if _, compatibilityErr := Parse(data); compatibilityErr != nil {
				t.Fatalf("legacy Parse rejected permissive input: %v", compatibilityErr)
			}
		})
	}
}

func TestStrictGPIFParsingRejectsMalformedCapo(t *testing.T) {
	tests := []struct {
		name string
		code string
		body string
	}{
		{name: "missing fret", code: "GPIF.Track.Property.CapoFret.MissingFret", body: `<Property name="CapoFret"/>`},
		{name: "negative fret", code: "GPIF.Track.Property.CapoFret.Negative", body: `<Property name="CapoFret"><Fret>-1</Fret></Property>`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := diagnosticGP8Fixture(t, func(gpif string) string {
				return strings.Replace(gpif, "<Staves>", "<Properties>"+test.body+"</Properties><Staves>", 1)
			})
			result, err := ParseWithOptions(data, ParseOptions{Strict: true})
			var strictErr *StrictParseError
			if !errors.As(err, &strictErr) || result == nil {
				t.Fatalf("strict parse = %#v, %v, want StrictParseError", result, err)
			}
			if !slices.ContainsFunc(result.Diagnostics, func(diagnostic ParseDiagnostic) bool {
				return diagnostic.Code == test.code && diagnostic.Kind == ParseDiagnosticInvalidData
			}) {
				t.Fatalf("diagnostics = %#v, want %s", result.Diagnostics, test.code)
			}
		})
	}
}

func TestParseWithOptionsStrictRejectsSelectedLoss(t *testing.T) {
	data := diagnosticGP8Fixture(t, func(gpif string) string {
		return insertFirstNoteProperty(t, gpif, `<Property name="FutureTechnique"><Enable/></Property>`)
	})
	result, err := ParseWithOptions(data, ParseOptions{Strict: true})
	var strictErr *StrictParseError
	if !errors.As(err, &strictErr) {
		t.Fatalf("error = %v, want StrictParseError", err)
	}
	if result == nil || result.Song == nil || len(result.Diagnostics) == 0 {
		t.Fatalf("strict result = %#v, want parsed song and diagnostics", result)
	}
	selected, selectedErr := ParseWithOptions(data, ParseOptions{
		Strict: true, StrictKinds: []ParseDiagnosticKind{ParseDiagnosticInvalidData},
	})
	if selectedErr != nil {
		t.Fatalf("strict invalid-data filter rejected unknown syntax: %v", selectedErr)
	}
	if selected == nil || len(selected.Diagnostics) == 0 {
		t.Fatal("selected strict result omitted permissive diagnostics")
	}
}
func TestParseWithOptionsStrictAllowsDeliberateIgnore(t *testing.T) {
	data := diagnosticGP8Fixture(t, func(gpif string) string {
		return insertFirstGPIFObjectChild(t, gpif, "<Beats>", "</Beat>", `<Properties><Property name="PrimaryPickupVolume"><Float>0.5</Float></Property></Properties>`)
	})
	result, err := ParseWithOptions(data, ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	diagnostic := findParseDiagnostic(result.Diagnostics, ParseDiagnosticDeliberateIgnore, "note-and-beat-semantics")
	if diagnostic == nil || diagnostic.Code == "" {
		t.Fatalf("diagnostics = %#v, want coded deliberate ignore", result.Diagnostics)
	}
}

func TestGPIFAutomationDispatchDiagnostics(t *testing.T) {
	tests := []struct {
		name         string
		mutate       func(string) string
		kind         ParseDiagnosticKind
		feature      string
		code         string
		pathContains string
	}{
		{
			name: "invalid sustain pedal reference",
			mutate: func(gpif string) string {
				return strings.Replace(gpif, "<Staves>", `<Automations><Automation><Type>SustainPedal</Type><Bar>0</Bar><Position>0</Position><Value>0 2</Value></Automation></Automations><Staves>`, 1)
			},
			kind: ParseDiagnosticInvalidData, feature: "sustain-pedal", code: "GPIF.Track.Automation.SustainPedal.Value.Invalid", pathContains: "Value",
		},
		{
			name: "unknown master automation",
			mutate: func(gpif string) string {
				return strings.Replace(gpif, "</MasterTrack>", `<Automations><Automation><Type>FutureFeature</Type><Bar>0</Bar><Position>0</Position><Value>42</Value></Automation></Automations></MasterTrack>`, 1)
			},
			kind: ParseDiagnosticUnknownSyntax, code: "GPIF.MasterTrack.Automation.Type.Unknown", pathContains: "MasterTrack",
		},
		{
			name: "unresolved sound reference",
			mutate: func(gpif string) string {
				return strings.Replace(gpif, "<Staves>", `<Automations><Automation><Type>Sound</Type><Bar>0</Bar><Position>0</Position><Value>missing;sound;Default</Value></Automation></Automations><Staves>`, 1)
			},
			kind: ParseDiagnosticInvalidData, code: "GPIF.Track.Automation.Sound.Reference", pathContains: "Automations",
		},
		{
			name: "unknown channel strip automation",
			mutate: func(gpif string) string {
				return strings.Replace(gpif, "<Staves>", `<RSE><ChannelStrip><Parameters>0 0 0 0 0 0 0 0 0 0 0.5 0.5 0.5</Parameters><Automations><Automation><Type>FutureDSP</Type><Bar>0</Bar><Position>0</Position><Value>0.5</Value></Automation></Automations></ChannelStrip></RSE><Staves>`, 1)
			},
			kind: ParseDiagnosticUnknownSyntax, code: "GPIF.ChannelStrip.Automation.Type.Unknown", pathContains: "ChannelStrip",
		},
		{
			name: "invalid volume automation value",
			mutate: func(gpif string) string {
				return strings.Replace(gpif, "<Staves>", `<RSE><ChannelStrip><Parameters>0 0 0 0 0 0 0 0 0 0 0.5 0.5 0.5</Parameters><Automations><Automation><Type>DSPParam_12</Type><Bar>0</Bar><Position>0</Position><Value>not-a-number</Value></Automation></Automations></ChannelStrip></RSE><Staves>`, 1)
			},
			kind: ParseDiagnosticInvalidData, code: "GPIF.ChannelStrip.Automation.Volume.Value.Invalid", pathContains: "Value",
		},
		{
			name: "out of range volume automation",
			mutate: func(gpif string) string {
				return strings.Replace(gpif, "<Staves>", `<RSE><ChannelStrip><Parameters>0 0 0 0 0 0 0 0 0 0 0.5 0.5 0.5</Parameters><Automations><Automation><Type>DSPParam_12</Type><Bar>0</Bar><Position>2</Position><Value>-1</Value></Automation></Automations></ChannelStrip></RSE><Staves>`, 1)
			},
			kind: ParseDiagnosticInvalidData, code: "GPIF.ChannelStrip.Automation.Volume.Range.Invalid", pathContains: "Automation",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := conformanceGPIFArchive(t, test.mutate(staffScopedChordGPIF))
			result, err := ParseWithOptions(data, ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			feature := test.feature
			if feature == "" {
				feature = "score-core"
			}
			diagnostic := findParseDiagnostic(result.Diagnostics, test.kind, feature)
			if diagnostic == nil || diagnostic.Code != test.code || !strings.Contains(diagnostic.SourcePath, test.pathContains) {
				t.Fatalf("diagnostics = %#v, want %s at path containing %q", result.Diagnostics, test.kind, test.pathContains)
			}
			if _, compatibilityErr := Parse(data); compatibilityErr != nil {
				t.Fatalf("legacy Parse rejected permissive input: %v", compatibilityErr)
			}
			strictResult, strictErr := ParseWithOptions(data, ParseOptions{Strict: true})
			var policyErr *StrictParseError
			if !errors.As(strictErr, &policyErr) || strictResult == nil {
				t.Fatalf("strict result = %#v, error = %v, want StrictParseError", strictResult, strictErr)
			}
		})
	}
}

func TestParseWithOptionsKeepsSupportedGPIFClean(t *testing.T) {
	result, err := ParseWithOptions(diagnosticGP8Fixture(t, func(gpif string) string { return gpif }), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Diagnostics) != 0 {
		t.Fatalf("diagnostics = %#v, want none", result.Diagnostics)
	}
}

func TestStrictParseAcceptsPitchedGP8Export(t *testing.T) {
	song := syntheticGP8Song()
	track := &song.Tracks[0]
	track.PercussionTrack = false
	song.Channels[0].Channel = 0
	song.Channels[0].EffectChannel = 0
	track.Strings = []GuitarString{{Number: 1, Value: 62}}
	for measureIndex := range track.Measures {
		track.Measures[measureIndex].Voices = []Voice{{Beats: []Beat{{
			Duration: defaultDuration(),
			Status:   BeatStatusNormal,
			Notes:    []Note{{Value: 5, String: 1, Kind: NoteTypeNormal, Velocity: Forte, AccidentalMode: NoteAccidentalNatural}},
		}}}}
	}
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseWithOptions(data, ParseOptions{Strict: true})
	if err != nil {
		t.Fatalf("strict parse rejected library-authored pitched export: %v", err)
	}
	if parsed.Song.Tracks[0].PercussionTrack || parsed.Song.Tracks[0].Staves[0].PercussionTrack {
		t.Fatal("pitched regression fixture exported as percussion")
	}
	contradictory := rewriteConformanceGPIF(t, data, func(gpif string) string {
		changed := strings.Replace(gpif, "<Step>G</Step>", "<Step>A</Step>", 1)
		if changed == gpif {
			t.Fatal("pitched regression did not mutate an authored spelling")
		}
		return changed
	})
	if _, err := ParseWithOptions(contradictory, ParseOptions{Strict: true}); err == nil {
		t.Fatal("strict parse accepted an authored pitch that contradicts the numeric value")
	}
}

func TestStrictParsePreservesTenutoAccent(t *testing.T) {
	data := diagnosticGP8Fixture(t, func(gpif string) string {
		return strings.Replace(gpif, "<Accent>8</Accent>", "<Accent>16</Accent>", 1)
	})
	result, err := ParseWithOptions(data, ParseOptions{Strict: true})
	if err != nil {
		t.Fatalf("strict parse rejected tenuto: %v", err)
	}
	note := result.Song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	if note.Effect.Accent != NoteAccentTenuto {
		t.Fatalf("accent = %d, want tenuto", note.Effect.Accent)
	}
}

func TestParseWithOptionsReportsBinaryOffsets(t *testing.T) {
	result, err := ParseFileWithOptions("testdata/gp5/other-effects.gp5", ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.BinaryOffset == nil {
			continue
		}
		if diagnostic.Format != "GP5" || diagnostic.SourcePath != "" || *diagnostic.BinaryOffset < 0 {
			t.Fatalf("binary diagnostic = %#v, want GP5 offset without XML path", diagnostic)
		}
		if diagnostic.Code == "" {
			t.Fatalf("binary diagnostic = %#v, want stable source code", diagnostic)
		}
		return
	}
	t.Fatalf("diagnostics = %#v, want binary offset", result.Diagnostics)
}
func diagnosticGP8Fixture(t *testing.T, mutate func(string) string) []byte {
	t.Helper()
	data, err := Export(syntheticGP8Song(), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	return rewriteConformanceGPIF(t, data, mutate)
}

func insertFirstNoteProperty(t *testing.T, gpif, property string) string {
	t.Helper()
	start := strings.LastIndex(gpif, "<Notes>")
	if start < 0 {
		t.Fatal("GPIF has no Notes element")
	}
	suffix := strings.Replace(gpif[start:], "<Properties>", "<Properties>"+property, 1)
	if suffix == gpif[start:] {
		t.Fatal("first GPIF note has no Properties element")
	}
	return gpif[:start] + suffix
}

func insertFirstGPIFObjectChild(t *testing.T, gpif, collection, closing, child string) string {
	t.Helper()
	start := strings.LastIndex(gpif, collection)
	if start < 0 {
		t.Fatalf("GPIF has no %s element", collection)
	}
	suffix := strings.Replace(gpif[start:], closing, child+closing, 1)
	if suffix == gpif[start:] {
		t.Fatalf("GPIF %s has no %s element", collection, closing)
	}
	return gpif[:start] + suffix
}
func replaceFirstGPIFReference(t *testing.T, gpif, prefix, replacement string) string {
	t.Helper()
	start := strings.Index(gpif, prefix)
	if start < 0 {
		t.Fatalf("GPIF has no %q reference", prefix)
	}
	valueStart := start + len(prefix)
	valueEnd := strings.IndexByte(gpif[valueStart:], '"')
	if valueEnd < 0 {
		t.Fatalf("GPIF %q reference has no closing quote", prefix)
	}
	valueEnd += valueStart
	return gpif[:valueStart] + replacement + gpif[valueEnd:]
}

func findParseDiagnostic(diagnostics []ParseDiagnostic, kind ParseDiagnosticKind, feature string) *ParseDiagnostic {
	for index := range diagnostics {
		if diagnostics[index].Kind == kind && diagnostics[index].Feature == feature {
			return &diagnostics[index]
		}
	}
	return nil
}
