// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/binary"
	"errors"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
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
			name: "known unsupported transpose",
			mutate: func(gpif string) string {
				return strings.Replace(gpif, "<Staves>", "<Transpose><Chromatic>2</Chromatic><Octave>1</Octave></Transpose><Staves>", 1)
			},
			kind: ParseDiagnosticUnsupportedFeature, feature: "staff-ownership", pathContains: "Transpose",
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
				return insertFirstGPIFObjectChild(t, gpif, "<Beats>", "</Beat>", "<Wah>Open</Wah>")
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
			name: "recognized whammy property",
			mutate: func(gpif string) string {
				return insertFirstGPIFObjectChild(t, gpif, "<Beats>", "</Beat>", "<Properties><Property name=\"WhammyBar\"><Enable/></Property></Properties>")
			},
			kind: ParseDiagnosticUnsupportedFeature, feature: "note-and-beat-semantics", pathContains: "WhammyBar",
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
				return strings.Replace(gpif, "<MasterTrack>", "<BackingTrack><AssetId>missing-asset</AssetId></BackingTrack><MasterTrack>", 1)
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
		code         string
		pathContains string
	}{
		{
			name: "unsupported sustain pedal",
			mutate: func(gpif string) string {
				return strings.Replace(gpif, "<Staves>", `<Automations><Automation><Type>SustainPedal</Type><Bar>0</Bar><Position>0</Position><Value>1</Value></Automation></Automations><Staves>`, 1)
			},
			kind: ParseDiagnosticUnsupportedFeature, code: "GPIF.Track.Automation.SustainPedal", pathContains: "SustainPedal",
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
			diagnostic := findParseDiagnostic(result.Diagnostics, test.kind, "score-core")
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
	track.Strings = []GuitarString{{Number: 1, Value: 62}}
	for measureIndex := range track.Measures {
		track.Measures[measureIndex].Voices = []Voice{{Beats: []Beat{{
			Duration: defaultDuration(),
			Status:   BeatStatusNormal,
			Notes:    []Note{{Value: 5, String: 1, Kind: NoteTypeNormal, Velocity: Forte}},
		}}}}
	}
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseWithOptions(data, ParseOptions{Strict: true}); err != nil {
		t.Fatalf("strict parse rejected library-authored pitched export: %v", err)
	}
	contradictory := rewriteConformanceGPIF(t, data, func(gpif string) string {
		return strings.Replace(gpif, "<Step>G</Step>", "<Step>A</Step>", 1)
	})
	if _, err := ParseWithOptions(contradictory, ParseOptions{Strict: true}); err == nil {
		t.Fatal("strict parse accepted a concert pitch that contradicts the MIDI value")
	}
}

func TestStrictParseReportsUnmappedTenutoAccent(t *testing.T) {
	data := diagnosticGP8Fixture(t, func(gpif string) string {
		return insertFirstGPIFObjectChild(t, gpif, "<Notes>", "</Note>", "<Accent>16</Accent>")
	})
	result, err := ParseWithOptions(data, ParseOptions{Strict: true})
	var strictErr *StrictParseError
	if !errors.As(err, &strictErr) {
		t.Fatalf("error = %v, want StrictParseError", err)
	}
	if result == nil || findParseDiagnostic(result.Diagnostics, ParseDiagnosticUnsupportedFeature, "note-and-beat-semantics") == nil {
		t.Fatalf("diagnostics = %#v, want unsupported tenuto accent", result)
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

func TestDurationTime(t *testing.T) {
	tests := []struct {
		name string
		in   Duration
		want uint32
	}{
		{"quarter", Duration{Value: 4, TupletEnters: 1, TupletTimes: 1}, 960},
		{"dotted quarter", Duration{Value: 4, Dotted: true, TupletEnters: 1, TupletTimes: 1}, 1440},
		{"double dotted quarter", Duration{Value: 4, DoubleDotted: true, TupletEnters: 1, TupletTimes: 1}, 1680},
		{"simultaneous dot flags use double dot", Duration{Value: 4, Dotted: true, DoubleDotted: true, TupletEnters: 1, TupletTimes: 1}, 1680},
		{"double dotted eighth triplet", Duration{Value: 8, DoubleDotted: true, TupletEnters: 3, TupletTimes: 2}, 560},
		{"quarter triplet", Duration{Value: 4, TupletEnters: 3, TupletTimes: 2}, 640},
		{"unset value", Duration{TupletEnters: 1, TupletTimes: 1}, 0},
		{"unset tuplet", Duration{Value: 4}, 960},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.in.time(); got != test.want {
				t.Fatalf("time() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestGPIFDoubleDottedDurationAndTiming(t *testing.T) {
	duration, err := gpifRhythmToDuration(&gpifRhythm{
		NoteValue: "Eighth", AugmentationDot: &gpifAugDot{Count: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if duration.Dotted || !duration.DoubleDotted || duration.time() != 840 {
		t.Fatalf("duration = %#v, time = %d; want one double-dot flag and 840 ticks", duration, duration.time())
	}

	song := parseTestFixture(t, "testdata/gp7/colors.gp")
	beats := song.Tracks[0].Staves[0].Measures[1].Voices[0].Beats
	if len(beats) < 2 || beats[0].Start == nil || beats[1].Start == nil {
		t.Fatalf("fixture beats = %#v, want two finalized beat starts", beats)
	}
	if got := beats[0].Duration; got.Dotted || !got.DoubleDotted || got.time() != 840 {
		t.Fatalf("fixture duration = %#v, time = %d; want double-dotted eighth at 840 ticks", got, got.time())
	}
	if *beats[1].Start != 5640 || *beats[1].Start-*beats[0].Start != 840 {
		t.Fatalf("beat starts = %d, %d; want 4800, 5640", *beats[0].Start, *beats[1].Start)
	}
}

func TestGPIFConversionsRejectValuesThatWouldNarrow(t *testing.T) {
	if _, err := gpifRhythmToDuration(&gpifRhythm{
		NoteValue: "Quarter", PrimaryTuplet: &gpifTuplet{Num: 256, Den: 255},
	}); err == nil {
		t.Fatal("uint8-overflowing tuplet was accepted")
	}
	fret := math.MaxInt16 + 1
	if _, err := gpifNoteToNote(&gpifNote{Properties: gpifProperties{Properties: []gpifProperty{
		{Name: "Fret", Fret: &fret},
	}}}, 6, false); err == nil {
		t.Fatal("int16-overflowing fret was accepted")
	}
	midi := 128
	if _, err := gpifNoteToNote(&gpifNote{Properties: gpifProperties{Properties: []gpifProperty{
		{Name: "Midi", Number: &midi},
	}}}, 6, false); err == nil {
		t.Fatal("out-of-range MIDI note was accepted")
	}
}

func TestGPIFMasterBarValuesDoNotWrapAtLegacyBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(string) string
		wantError bool
		errorText string
		assert    func(*testing.T, *Song)
	}{
		{name: "maximum numerator", mutate: func(gpif string) string {
			return strings.Replace(gpif, "4/4", "127/4", 1)
		}, assert: func(t *testing.T, song *Song) {
			if song.MeasureHeaders[0].TimeSignature.Numerator != 127 {
				t.Fatalf("numerator = %d, want 127", song.MeasureHeaders[0].TimeSignature.Numerator)
			}
		}},
		{name: "numerator overflow", wantError: true, mutate: func(gpif string) string {
			return strings.Replace(gpif, "4/4", "128/4", 1)
		}},
		{name: "modulo numerator", wantError: true, mutate: func(gpif string) string {
			return strings.Replace(gpif, "4/4", "260/4", 1)
		}},
		{name: "denominator modulo overflow", wantError: true, errorText: "outside 1..65535", mutate: func(gpif string) string {
			return strings.Replace(gpif, "4/4", "4/65540", 1)
		}},
		{name: "key overflow", wantError: true, mutate: func(gpif string) string {
			return strings.Replace(gpif, "<MasterBar><Time>", "<MasterBar><Key><AccidentalCount>128</AccidentalCount></Key><Time>", 1)
		}},
		{name: "maximum repeat count", mutate: func(gpif string) string {
			return strings.Replace(gpif, "<Bars>0 1</Bars>", `<Repeat end="true" count="128"/><Bars>0 1</Bars>`, 1)
		}, assert: func(t *testing.T, song *Song) {
			if song.MeasureHeaders[0].RepeatClose != 127 {
				t.Fatalf("repeat close = %d, want 127", song.MeasureHeaders[0].RepeatClose)
			}
		}},
		{name: "repeat overflow", wantError: true, mutate: func(gpif string) string {
			return strings.Replace(gpif, "<Bars>0 1</Bars>", `<Repeat end="true" count="130"/><Bars>0 1</Bars>`, 1)
		}},
		{name: "alternate ending overflow", wantError: true, mutate: func(gpif string) string {
			return strings.Replace(gpif, "<Bars>0 1</Bars>", "<AlternateEndings>9</AlternateEndings><Bars>0 1</Bars>", 1)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ParseWithOptions(conformanceGPIFArchive(t, test.mutate(staffScopedChordGPIF)), ParseOptions{Strict: true})
			if test.wantError {
				if err == nil {
					t.Fatalf("strict parse succeeded with song %#v", result.Song.MeasureHeaders[0])
				}
				if test.errorText != "" && !strings.Contains(err.Error(), test.errorText) {
					t.Fatalf("error = %v, want %q", err, test.errorText)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if test.assert != nil {
				test.assert(t, result.Song)
			}
		})
	}
}

func TestGPIFPreservesFractionalInitialTempo(t *testing.T) {
	data := diagnosticGP8Fixture(t, func(gpif string) string {
		return strings.Replace(gpif, "132 2", "120.5 2", 1)
	})
	result, err := ParseWithOptions(data, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Song.InitialTempo.State != SourceValueKnown || result.Song.InitialTempo.Value != BPM(120.5) {
		t.Fatalf("initial tempo = %#v, want known 120.5 BPM", result.Song.InitialTempo)
	}
}

func TestGPIFTempoBoundaryDiagnostics(t *testing.T) {
	t.Run("non-finite is unknown", func(t *testing.T) {
		data := diagnosticGP8Fixture(t, func(gpif string) string {
			return strings.Replace(gpif, "132 2", "Inf 2", 1)
		})
		result, err := ParseWithOptions(data, ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if result.Song.InitialTempo.State != SourceValueUnknown || result.Song.InitialTempo.Raw != "Inf" {
			t.Fatalf("initial tempo = %#v, want unknown Inf", result.Song.InitialTempo)
		}
		if findParseDiagnostic(result.Diagnostics, ParseDiagnosticInvalidData, "tempo-automations") == nil {
			t.Fatalf("diagnostics = %#v, want invalid tempo", result.Diagnostics)
		}
	})

	t.Run("legacy overflow preserves semantic BPM", func(t *testing.T) {
		data := diagnosticGP8Fixture(t, func(gpif string) string {
			return strings.Replace(gpif, "132 2", "40000 2", 1)
		})
		result, err := ParseWithOptions(data, ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if result.Song.InitialTempo.State != SourceValueKnown || result.Song.InitialTempo.Value != BPM(40000) {
			t.Fatalf("initial tempo = %#v, want known 40000 BPM", result.Song.InitialTempo)
		}
		if result.Song.Tempo < 0 {
			t.Fatalf("legacy tempo silently wrapped to %d", result.Song.Tempo)
		}
		if findParseDiagnostic(result.Diagnostics, ParseDiagnosticLossyProjection, "tempo-automations") == nil {
			t.Fatalf("diagnostics = %#v, want legacy tempo projection warning", result.Diagnostics)
		}
	})
}

func TestTupletBeatsAdvanceByPlayedLength(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp5/tuplets.gp5")
	beats := song.Tracks[0].Measures[0].Voices[0].Beats
	if len(beats) < 3 {
		t.Fatalf("beats = %d, want at least 3", len(beats))
	}
	base := *beats[0].Start
	for index, beat := range beats[:3] {
		want := base + int64(index)*640
		if beat.Start == nil || *beat.Start != want {
			t.Errorf("beat %d starts at %v, want %d", index, beat.Start, want)
		}
	}
}

func TestMarkerTitlesExcludeLengthPrefix(t *testing.T) {
	markerCount := 0
	err := filepath.WalkDir("testdata", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !isGuitarProFixture(path) {
			return nil
		}
		extension := strings.ToLower(filepath.Ext(path))
		if extension != ".gp3" && extension != ".gp4" && extension != ".gp5" {
			return nil
		}
		song := parseTestFixture(t, path)
		for _, header := range song.MeasureHeaders {
			if header.Marker == nil || header.Marker.Title == "" {
				continue
			}
			markerCount++
			title := header.Marker.Title
			if strings.ContainsRune(title, 0) {
				t.Errorf("%s: invalid marker title %q", path, title)
			}
			if title != "" && int(title[0]) == len(title)-1 {
				t.Errorf("%s: marker title contains its length prefix: %q", path, title)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if markerCount == 0 {
		t.Fatal("no marker titles in the compatibility corpus")
	}
}

func TestPercussionUsesTrackChannel(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp5/nightwish.gp5")
	if len(song.Tracks) != 11 {
		t.Fatalf("tracks = %d, want 11", len(song.Tracks))
	}
	if song.Tracks[9].PercussionTrack {
		t.Error("track 9 uses a melodic channel")
	}
	drums := song.Tracks[10]
	if !drums.PercussionTrack {
		t.Error("track 10 uses the percussion channel")
	}
	if drums.ChannelIndex < 0 || song.Channels[drums.ChannelIndex].Channel != DefaultPercussionChannel {
		t.Errorf("track 10 channel index = %d, want the percussion channel", drums.ChannelIndex)
	}
}

func TestGP5PercussionGraceUsesDrumArticulation(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp5/motherload-percussion-grace.gp5")
	var track *Track
	for index := range song.Tracks {
		if song.Tracks[index].Name == "Percussion" {
			track = &song.Tracks[index]
			break
		}
	}
	if track == nil {
		t.Fatal("Percussion track not found")
	}
	if !track.PercussionTrack {
		t.Fatal("Percussion track is not marked as percussion")
	}

	var graceBars []int
	for measureIndex, measure := range track.Measures {
		for _, voice := range measure.Voices {
			for _, beat := range voice.Beats {
				for _, note := range beat.Notes {
					if len(note.Effect.Graces) == 0 {
						continue
					}
					if note.Value != 38 {
						t.Errorf("bar %d grace parent articulation = %d, want 38", measureIndex+1, note.Value)
					}
					for _, grace := range note.Effect.Graces {
						graceBars = append(graceBars, measureIndex+1)
						if grace.Fret != 38 {
							t.Errorf("bar %d grace articulation = %d, want 38", measureIndex+1, grace.Fret)
						}
						if grace.RawFret == nil || *grace.RawFret != 0 {
							t.Errorf("bar %d raw grace fret = %v, want 0", measureIndex+1, grace.RawFret)
						}
					}
				}
			}
		}
	}
	wantBars := []int{2, 2, 3, 26, 27, 51, 74, 75}
	if !slices.Equal(graceBars, wantBars) {
		t.Errorf("percussion grace bars = %v, want %v", graceBars, wantBars)
	}
}
func TestGPIFPercussionPreservesArticulations(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/percussion.gp")
	track := &song.Tracks[0]
	if len(track.PercussionArticulations) != 95 {
		t.Fatalf("percussion articulations = %d, want 95", len(track.PercussionArticulations))
	}
	if got := track.Staves[0].StandardNotationLineCount; got != 1 {
		t.Errorf("standard notation line count = %d, want 1", got)
	}

	hit := track.PercussionArticulations[68]
	returned := track.PercussionArticulations[69]
	if hit.ElementName != "Cabasa" || hit.ElementType != "cabasa" || hit.Name != "Cabasa (hit)" {
		t.Errorf("hit identity = %#v", hit)
	}
	if hit.StaffLine != 0 || hit.NoteheadDefault != "noteheadBlack" || hit.NoteheadHalf != "noteheadHalf" || hit.NoteheadWhole != "noteheadWhole" {
		t.Errorf("hit notation = %#v", hit)
	}
	if hit.TechniquePlacement != "outside" || hit.TechniqueSymbol != "" || !slices.Equal(hit.InputMIDINumbers, []int{69}) || hit.OutputRSESound != "hand.hit.hit" || hit.OutputMIDINumber != 69 {
		t.Errorf("hit playback = %#v", hit)
	}
	if returned.ElementName != "Cabasa" || returned.ElementType != "cabasa" || returned.Name != "Cabasa (return)" {
		t.Errorf("return identity = %#v", returned)
	}
	if returned.StaffLine != 0 || returned.TechniquePlacement != "outside" || returned.TechniqueSymbol != "stringsUpBow" || !slices.Equal(returned.InputMIDINumbers, []int{117}) || returned.OutputRSESound != "hand.hit.return" || returned.OutputMIDINumber != 69 {
		t.Errorf("return metadata = %#v", returned)
	}

	var notes []Note
	for _, measure := range track.Staves[0].Measures {
		for _, voice := range measure.Voices {
			for _, beat := range voice.Beats {
				notes = append(notes, beat.Notes...)
			}
		}
	}
	if len(notes) < 2 {
		t.Fatalf("notes = %d, want at least 2", len(notes))
	}
	if !notes[0].HasPercussionArticulation || notes[0].Value != 69 || notes[0].PercussionArticulation != 68 {
		t.Errorf("first note value/articulation/presence = %d/%d/%t, want 69/68/true", notes[0].Value, notes[0].PercussionArticulation, notes[0].HasPercussionArticulation)
	}
	if !notes[1].HasPercussionArticulation || notes[1].Value != 117 || notes[1].PercussionArticulation != 69 {
		t.Errorf("second note value/articulation/presence = %d/%d/%t, want 117/69/true", notes[1].Value, notes[1].PercussionArticulation, notes[1].HasPercussionArticulation)
	}
}

func TestGPIFPitchedTrackHasNoPercussionDefinitions(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/tuning.gp")
	for trackIndex, track := range song.Tracks {
		if !track.PercussionTrack && len(track.PercussionArticulations) != 0 {
			t.Fatalf("pitched track %d has %d percussion definitions", trackIndex, len(track.PercussionArticulations))
		}
	}
}

func TestGP5TrackMixerUsesNormalizedMidiRange(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp5/Demo v5.gp5")
	if len(song.Tracks) < 2 {
		t.Fatalf("tracks = %d, want at least 2", len(song.Tracks))
	}

	rhythm := song.Channels[song.Tracks[0].ChannelIndex]
	solo := song.Channels[song.Tracks[1].ChannelIndex]
	if rhythm.Volume != 87 || rhythm.Balance != 64 {
		t.Errorf("rhythm channel = %#v, want volume 87 and balance 64", rhythm)
	}
	if solo.Volume != 119 || solo.Balance != 64 {
		t.Errorf("solo channel = %#v, want volume 119 and balance 64", solo)
	}
}

func TestGP5PlaybackChannelsUseAuthoredPairs(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp5/Demo v5.gp5")
	want := [][2]uint8{{0, 1}, {4, 3}, {12, 7}, {8, 5}, {9, 9}}
	if len(song.Tracks) != len(want) {
		t.Fatalf("tracks = %d, want %d", len(song.Tracks), len(want))
	}
	for index, track := range song.Tracks {
		channel := song.Channels[track.ChannelIndex]
		if channel.Channel != want[index][0] || channel.EffectChannel != want[index][1] {
			t.Errorf("track %d channels = %d/%d, want %d/%d", index, channel.Channel, channel.EffectChannel, want[index][0], want[index][1])
		}
	}
}

func TestGP5LastExplicitNoteDynamicAppliesToBeat(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp5/Demo v5.gp5")
	beat := song.Tracks[0].Measures[0].Voices[0].Beats[0]
	if len(beat.Notes) != 3 {
		t.Fatalf("opening beat notes = %d, want 3", len(beat.Notes))
	}
	if beat.Notes[0].Velocity != Forte || beat.Dynamics != MinVelocity+VelocityIncrement*7 {
		t.Errorf("opening note/beat dynamics = %d/%d, want %d/%d", beat.Notes[0].Velocity, beat.Dynamics, Forte, MinVelocity+VelocityIncrement*7)
	}
}

func TestGP6PercussionTrack(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp6/full-song.gpx")
	if len(song.Tracks) != 11 {
		t.Fatalf("tracks = %d, want 11", len(song.Tracks))
	}
	drums := song.Tracks[10]
	if drums.Name != "Jukka" {
		t.Fatalf("track 10 name = %q, want %q", drums.Name, "Jukka")
	}
	if !drums.PercussionTrack {
		t.Error("GP6 drumkit track is not marked as percussion")
	}
}

func TestGPIFPercussionSignals(t *testing.T) {
	tests := []struct {
		name  string
		track gpifTrack
		want  bool
	}{
		{name: "instrument set", track: gpifTrack{InstrumentSet: &gpifInstrumentSet{Type: "drums"}}, want: true},
		{name: "GP6 instrument", track: gpifTrack{Instrument: &gpifInstrument{Ref: "drmkt"}}, want: true},
		{name: "percussion channel", track: gpifTrack{GeneralMidi: &gpifGeneralMidi{PrimaryChannel: 9}}, want: true},
		{name: "melodic channel", track: gpifTrack{GeneralMidi: &gpifGeneralMidi{PrimaryChannel: 8}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.track.isPercussionTrack(); got != test.want {
				t.Errorf("isPercussionTrack() = %t, want %t", got, test.want)
			}
		})
	}
}

func TestGPIFTempoAutomations(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp8/beat-tempo-change.gp")
	if song.Tempo != 120 {
		t.Errorf("tempo = %d, want 120", song.Tempo)
	}
	if len(song.TempoAutomations) != 9 {
		t.Fatalf("tempo automations = %d, want 9", len(song.TempoAutomations))
	}
	first := song.TempoAutomations[0]
	if first.Bar != 0 || first.Position != 0 || first.Tempo != 120 {
		t.Errorf("first tempo automation = %#v", first)
	}
}

func TestGPIFTrackMixerState(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/track-balance.gp")
	if len(song.Tracks) < 2 || len(song.Channels) < 2 {
		t.Fatalf("tracks/channels = %d/%d, want at least 2", len(song.Tracks), len(song.Channels))
	}
	first := song.Channels[song.Tracks[0].ChannelIndex]
	second := song.Channels[song.Tracks[1].ChannelIndex]
	if first.Channel != 0 || first.EffectChannel != 1 || second.Channel != 2 || second.EffectChannel != 3 {
		t.Errorf("GPIF MIDI channels = [%d/%d %d/%d], want [0/1 2/3]", first.Channel, first.EffectChannel, second.Channel, second.EffectChannel)
	}
	if first.Volume != 101 || first.Balance != 0 || second.Balance != 32 {
		t.Errorf("GPIF channel strip = %#v %#v", first, second)
	}
	if len(song.VolumeAutomations) < 2 {
		t.Fatalf("volume automations = %d, want at least 2", len(song.VolumeAutomations))
	}
	if got := song.VolumeAutomations[0]; got.Track != 0 || got.Bar != 0 || got.Position != 0 || got.Value != 0.72 || got.Linear {
		t.Errorf("first volume automation = %#v", got)
	}
}

func TestGPIFTrackTunings(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/multi-track.gp")
	if len(song.Tracks) < 5 {
		t.Fatalf("tracks = %d, want at least 5", len(song.Tracks))
	}
	want := []GuitarString{{1, 62}, {2, 57}, {3, 53}, {4, 48}, {5, 43}, {6, 36}}
	if got := song.Tracks[3].Strings; !slices.Equal(got, want) {
		t.Errorf("Drop C tuning = %#v, want %#v", got, want)
	}
}

func TestGPIFNormalizesStringNumbersAndOpenFrets(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/brush.gp")
	notes := song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes
	if len(notes) < 3 {
		t.Fatalf("opening chord notes = %d, want at least 3", len(notes))
	}
	if got := notes[0]; got.String != 1 || got.Value != 0 {
		t.Errorf("highest open string = string %d fret %d, want string 1 fret 0", got.String, got.Value)
	}
	if got := notes[1]; got.String != 2 || got.Value != 1 {
		t.Errorf("second string note = string %d fret %d, want string 2 fret 1", got.String, got.Value)
	}
	if got := notes[2]; got.String != 3 || got.Value != 0 {
		t.Errorf("third open string = string %d fret %d, want string 3 fret 0", got.String, got.Value)
	}
}

func TestGPIFTrackMuteState(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/dead-slap.gp")
	muted := 0
	for _, track := range song.Tracks {
		if track.Mute {
			muted++
		}
	}
	if muted != 1 {
		t.Errorf("muted tracks = %d, want 1", muted)
	}
}

func TestGPIFUsesOpeningTempoAutomation(t *testing.T) {
	song := &Song{Tempo: 120}
	gpifReadTempoAutomations([]gpifAutomation{
		{Type: "Tempo", Bar: 4, Value: gpifAutomationValue{Text: "180 2"}},
		{Type: "Tempo", Bar: 0, Position: 0.25, Value: gpifAutomationValue{Text: "90 2"}, Text: "Slow"},
		{Type: "Tempo", Bar: 0, Value: gpifAutomationValue{Text: "140 2"}, Text: "Allegro"},
		{Type: "Tempo", Bar: 2, Value: gpifAutomationValue{Text: "invalid"}},
	}, song, nil)
	if song.Tempo != 140 || song.TempoName != "Allegro" {
		t.Errorf("opening tempo = %d %q, want 140 %q", song.Tempo, song.TempoName, "Allegro")
	}
	if len(song.TempoAutomations) != 3 {
		t.Errorf("tempo automations = %d, want 3", len(song.TempoAutomations))
	}
}

func TestGPIFAppliesTempoReferenceUnits(t *testing.T) {
	values := []string{"100 1", "100 2", "100 3", "100 4", "100 5", "100", "100 invalid", "100 0", "100 6", "100 3x", "100 3.5", "81.5 3"}
	want := []float64{50, 100, 150, 200, 300, 50, 100, 100, 100, 150, 150, 122.25}
	automations := make([]gpifAutomation, 0, len(values))
	for index, value := range values {
		automations = append(automations, gpifAutomation{
			Type: "Tempo", Bar: index, Value: gpifAutomationValue{Text: value},
		})
	}
	song := &Song{}
	gpifReadTempoAutomations(automations, song, nil)
	if len(song.TempoAutomations) != len(want) {
		t.Fatalf("tempo automations = %d, want %d", len(song.TempoAutomations), len(want))
	}
	for index := range want {
		if got := song.TempoAutomations[index].Tempo; got != want[index] {
			t.Errorf("tempo %q = %v, want %v", values[index], got, want[index])
		}
	}
	if song.Tempo != 50 {
		t.Errorf("initial tempo = %d, want 50", song.Tempo)
	}
}

func TestMixTableTempoAboveByte(t *testing.T) {
	data := []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	data = binary.LittleEndian.AppendUint32(data, 300)
	data = append(data, 0, 0)
	song := &Song{Version: Version{Number: [3]byte{4, 0, 0}}}
	change, err := song.readMixTableChange(newCursor(data))
	if err != nil {
		t.Fatal(err)
	}
	if change.Tempo == nil || change.Tempo.Value != 300 {
		t.Errorf("tempo = %#v, want 300", change.Tempo)
	}
}

func TestReadCountRejectsInvalidCounts(t *testing.T) {
	for _, data := range [][]byte{
		{0xff, 0xff, 0xff, 0xff, 0, 0},
		{0xff, 0xff, 0xff, 0x7f, 0, 0},
		{1, 0},
	} {
		if _, err := newCursor(data).readCount(2, "beat count"); err == nil {
			t.Error("readCount() error = nil")
		}
	}
}

func TestCorruptFilesReturnParseError(t *testing.T) {
	original, err := os.ReadFile("testdata/gp5/serenade.gp5")
	if err != nil {
		t.Fatal(err)
	}
	for _, fraction := range []int{2, 3, 4, 8, 16, 32, 64} {
		t.Run("truncated-1/"+strconv.Itoa(fraction), func(t *testing.T) {
			_, err := Parse(original[:len(original)/fraction])
			var parseErr *ParseError
			if !errors.As(err, &parseErr) {
				t.Fatalf("error = %v, want *ParseError", err)
			}
		})
	}
}

func TestGPXRejectsAbsurdDecompressedLength(t *testing.T) {
	data := []byte{0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0}
	if _, err := gpxDecompress(data); err == nil {
		t.Error("gpxDecompress() error = nil")
	}
}

func parseTestFixture(t *testing.T, path string) *Song {
	t.Helper()
	song, err := ParseFile(path)
	if err != nil {
		t.Fatalf("ParseFile(%q): %v", path, err)
	}
	return song
}
