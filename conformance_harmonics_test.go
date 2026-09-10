// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"math"
	"slices"
	"testing"
)

func TestConformanceHarmonicVariants(t *testing.T) {
	runConformanceHarmonicVariants(newConformanceRun(t))
}

func runConformanceHarmonicVariants(run *conformanceRun) {
	t := run.t
	for _, source := range []struct {
		value byte
		want  HarmonicType
	}{{1, HarmonicTypeNatural}, {3, HarmonicTypeTapped}, {4, HarmonicTypePinch}, {5, HarmonicTypeSemi}, {15, HarmonicTypeArtificial}, {17, HarmonicTypeArtificial}, {22, HarmonicTypeArtificial}} {
		effect, err := (&Song{}).readHarmonic(newCursor([]byte{source.value}), &Note{Value: 6})
		if err != nil {
			t.Fatal(err)
		}
		run.Dispatch("readHarmonic:kind", effect.Kind, source.want)
	}
	for _, source := range []struct {
		bytes []byte
		want  HarmonicType
	}{{[]byte{1}, HarmonicTypeNatural}, {[]byte{2, 1, 0, 0}, HarmonicTypeArtificial}, {[]byte{3, 12}, HarmonicTypeTapped}, {[]byte{4}, HarmonicTypePinch}, {[]byte{5}, HarmonicTypeSemi}} {
		effect, err := (&Song{}).readHarmonicV5(newCursor(source.bytes))
		if err != nil {
			t.Fatal(err)
		}
		run.Dispatch("readHarmonicV5ForNote:kind", effect.Kind, source.want)
	}
	for _, unknown := range []struct {
		code  string
		parse func(*cursor) error
	}{
		{"Binary.Note.Harmonic.Kind.Unsupported", func(cursor *cursor) error { _, err := (&Song{}).readHarmonic(cursor, &Note{}); return err }},
		{"Binary.Note.HarmonicV5.Kind.Unsupported", func(cursor *cursor) error { _, err := (&Song{}).readHarmonicV5(cursor); return err }},
	} {
		context := &parseContext{format: "GP5"}
		if err := unknown.parse(newCursorWithContext([]byte{99}, context)); err != nil {
			t.Fatal(err)
		}
		if diagnostic := conformanceSourceAuditDiagnosticByCode(context.diagnostics, unknown.code); diagnostic == nil || diagnostic.Kind != ParseDiagnosticUnsupportedFeature {
			t.Fatalf("%s diagnostics = %#v", unknown.code, context.diagnostics)
		}
	}
	legacyFret := int8(2)
	exactFret := 2.4
	for _, test := range []struct {
		name   string
		member string
		kind   HarmonicType
		wire   string
	}{
		{name: "natural", member: "HarmonicTypeNatural", kind: HarmonicTypeNatural, wire: "Natural"},
		{name: "artificial", member: "HarmonicTypeArtificial", kind: HarmonicTypeArtificial, wire: "Artificial"},
		{name: "pinch", member: "HarmonicTypePinch", kind: HarmonicTypePinch, wire: "Pinch"},
		{name: "tapped", member: "HarmonicTypeTapped", kind: HarmonicTypeTapped, wire: "Tap"},
		{name: "semi", member: "HarmonicTypeSemi", kind: HarmonicTypeSemi, wire: "Semi"},
		{name: "feedback", member: "HarmonicTypeFeedback", kind: HarmonicTypeFeedback, wire: "Feedback"},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := conformanceHarmonicSong(t)
			harmonic := &HarmonicEffect{Kind: test.kind, Fret: &legacyFret, FretFloat: &exactFret}
			conformanceHarmonicFirstNote(song).Effect.Harmonic = harmonic
			run.Preserved("HarmonicEffect.Kind", harmonic.Kind, test.kind)
			run.Normalized("HarmonicEffect.Fret", *harmonic.Fret, legacyFret)
			run.Preserved("HarmonicEffect.FretFloat", *harmonic.FretFloat, exactFret)
			if got := gp8HarmonicType(test.kind); got != test.wire {
				t.Errorf("gp8HarmonicType(%d) = %q, want %q", test.kind, got, test.wire)
			}
			data, err := Export(song, ExportFormatGP8)
			if err != nil {
				t.Fatal(err)
			}
			properties := extractHarmonicWire(t, data)
			run.Wire("gpifProperty.HType", conformanceHarmonicWireValue(properties, "HarmonicType", "HType"), test.wire)
			run.Wire("gpifProperty.HFret", conformanceHarmonicWireValue(properties, "HarmonicFret", "HFret"), "2.4")
			run.Wire("gpifProperty.Float", conformanceHarmonicWireValue(properties, "HarmonicFret", "Float"), "")
			run.Wire("gpifProperty.Name", conformanceHarmonicPropertyNames(properties), []string{"HarmonicType", "HarmonicFret"})
			parsed, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			run.Enum("HarmonicType."+test.member, conformanceHarmonicFirstNote(parsed).Effect.Harmonic.Kind, test.kind)
		})
	}
	if got := gp8HarmonicType(HarmonicType(99)); got != "" {
		t.Errorf("gp8HarmonicType(99) = %q, want empty", got)
	}

	t.Run("no harmonic", func(t *testing.T) {
		song := conformanceHarmonicSong(t)
		conformanceHarmonicFirstNote(song).Effect.Harmonic = nil
		data, err := Export(song, ExportFormatGP8)
		if err != nil {
			t.Fatal(err)
		}
		properties := extractHarmonicWire(t, data)
		run.Wire("gpifProperty.Name", conformanceHarmonicPropertyNames(properties), []string(nil))
	})

	t.Run("integer fret", func(t *testing.T) {
		song := conformanceHarmonicSong(t)
		fret := int8(12)
		conformanceHarmonicFirstNote(song).Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeNatural, Fret: &fret}
		data, err := Export(song, ExportFormatGP8)
		if err != nil {
			t.Fatal(err)
		}
		properties := extractHarmonicWire(t, data)
		run.Wire("gpifProperty.HFret", conformanceHarmonicWireValue(properties, "HarmonicFret", "HFret"), "12")
	})

	t.Run("pitch spelling loss is explicit", func(t *testing.T) {
		pitch := PitchClass{Note: "C#", Just: 0, Accidental: 1, Value: 1, Sharp: true}
		octave := OctaveOttava
		song := conformanceHarmonicSong(t)
		harmonic := &HarmonicEffect{Kind: HarmonicTypeArtificial, Pitch: &pitch, Octave: &octave, Fret: &legacyFret, FretFloat: &exactFret}
		conformanceHarmonicFirstNote(song).Effect.Harmonic = harmonic
		run.Omitted("HarmonicEffect.Pitch", *harmonic.Pitch, pitch)
		run.Omitted("HarmonicEffect.Octave", *harmonic.Octave, octave)
		run.Preserved("PitchClass.Note", harmonic.Pitch.Note, pitch.Note)
		run.Preserved("PitchClass.Just", harmonic.Pitch.Just, pitch.Just)
		run.Preserved("PitchClass.Accidental", harmonic.Pitch.Accidental, pitch.Accidental)
		run.Preserved("PitchClass.Value", harmonic.Pitch.Value, pitch.Value)
		run.Preserved("PitchClass.Sharp", harmonic.Pitch.Sharp, pitch.Sharp)
		report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
		if !hasExportCode(report, "gp8.omit.harmonic-pitch") {
			t.Fatalf("pitch report = %#v, want harmonic pitch omission", report.Entries)
		}
		data, strictReport, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
		var lossErr *ExportLossError
		if len(data) != 0 || !errors.As(err, &lossErr) || !hasExportCode(strictReport, "gp8.omit.harmonic-pitch") {
			t.Fatalf("strict pitch export = %d bytes, %#v, %v", len(data), strictReport.Entries, err)
		}
	})
	for _, field := range []struct {
		name string
		set  func(*HarmonicEffect)
	}{
		{"pitch", func(harmonic *HarmonicEffect) {
			value := PitchClass{Note: "C#", Value: 1, Sharp: true}
			harmonic.Pitch = &value
		}},
		{"octave", func(harmonic *HarmonicEffect) { value := OctaveOttava; harmonic.Octave = &value }},
	} {
		t.Run("isolated "+field.name+" loss", func(t *testing.T) {
			song := conformanceHarmonicSong(t)
			harmonic := &HarmonicEffect{Kind: HarmonicTypeNatural, Fret: &legacyFret, FretFloat: &exactFret}
			field.set(harmonic)
			conformanceHarmonicFirstNote(song).Effect.Harmonic = harmonic
			if report := PreflightExport(song, ExportFormatGP8, ExportOptions{}); !hasExportCode(report, "gp8.omit.harmonic-pitch") {
				t.Fatalf("isolated %s report = %#v, want gp8.omit.harmonic-pitch", field.name, report.Entries)
			}
			data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
			if len(data) != 0 || err == nil {
				t.Fatalf("isolated %s strict export = %d bytes, %v", field.name, len(data), err)
			}
		})
	}

	t.Run("binary pitch source", func(t *testing.T) {
		legacy, err := (&Song{}).readHarmonic(newCursor([]byte{15}), &Note{Value: 6})
		if err != nil {
			t.Fatal(err)
		}
		if legacy.Pitch == nil || legacy.Octave == nil {
			t.Fatalf("legacy harmonic = %#v, want pitch and octave", legacy)
		}
		run.Field("HarmonicEffect.Kind", legacy.Kind, HarmonicTypeArtificial)
		run.Field("HarmonicEffect.Pitch", *legacy.Pitch, PitchClass{Note: "C#", Just: 0, Accidental: 1, Value: 1, Sharp: true})
		run.Field("HarmonicEffect.Octave", *legacy.Octave, OctaveOttava)
		run.Field("PitchClass.Note", legacy.Pitch.Note, "C#")
		run.Field("PitchClass.Just", legacy.Pitch.Just, int8(0))
		run.Field("PitchClass.Accidental", legacy.Pitch.Accidental, int8(1))
		run.Field("PitchClass.Value", legacy.Pitch.Value, int8(1))
		run.Field("PitchClass.Sharp", legacy.Pitch.Sharp, true)

		gp5, err := (&Song{}).readHarmonicV5(newCursor([]byte{2, 1, 0xff, byte(OctaveQuindicesima)}))
		if err != nil {
			t.Fatal(err)
		}
		if gp5.Pitch == nil || gp5.Octave == nil {
			t.Fatalf("GP5 harmonic = %#v, want pitch and octave", gp5)
		}
		run.Field("PitchClass.Just", gp5.Pitch.Just, int8(1))
		run.Field("PitchClass.Accidental", gp5.Pitch.Accidental, int8(-1))
		run.Field("PitchClass.Value", gp5.Pitch.Value, int8(0))
		run.Field("PitchClass.Sharp", gp5.Pitch.Sharp, false)
		run.Field("HarmonicEffect.Octave", *gp5.Octave, OctaveQuindicesima)

		tapped, err := (&Song{}).readHarmonicV5(newCursor([]byte{3, 12}))
		if err != nil {
			t.Fatal(err)
		}
		if tapped.Fret == nil {
			t.Fatalf("tapped harmonic = %#v, want fret", tapped)
		}
		run.Field("HarmonicEffect.Fret", *tapped.Fret, int8(12))
	})
}

func TestConformanceGPIFCompatibilityAndDiagnostics(t *testing.T) {
	runConformanceGPIFCompatibilityAndDiagnostics(newConformanceRun(t))
}

func runConformanceGPIFCompatibilityAndDiagnostics(run *conformanceRun) {
	t := run.t
	for _, test := range []struct {
		source string
		want   HarmonicType
	}{
		{source: "NoHarmonic"},
		{source: "Natural", want: HarmonicTypeNatural},
		{source: "Artificial", want: HarmonicTypeArtificial},
		{source: "Pinch", want: HarmonicTypePinch},
		{source: "Tap", want: HarmonicTypeTapped},
		{source: "Semi", want: HarmonicTypeSemi},
		{source: "Feedback", want: HarmonicTypeFeedback},
	} {
		t.Run(test.source, func(t *testing.T) {
			context := &parseContext{format: "GPIF"}
			gpifAuditNoteProperty(context, "n1", `/GPIF/Notes/Note[@id="n1"]`, gpifProperty{Name: "HarmonicType", HType: &test.source}, nil)
			gotDisposition := "preserved"
			wantDisposition := "preserved"
			if len(context.diagnostics) != 0 {
				gotDisposition = string(context.diagnostics[0].Kind)
			}
			run.Dispatch("gpifAuditNoteProperty:property.HType", gotDisposition, wantDisposition)

			harmonic := conformanceHarmonicFromProperties(t, []gpifProperty{{Name: "HarmonicType", HType: &test.source}})
			var got HarmonicType
			if harmonic != nil {
				got = harmonic.Kind
			}
			run.Dispatch("gpifNoteToNote:p.HType", got, test.want)
		})
	}

	for _, test := range []struct {
		name       string
		properties []gpifProperty
		want       float64
	}{
		{name: "HFret native", properties: conformanceHarmonicProperties("HFret", "2.4", false), want: 2.4},
		{name: "Float compatibility", properties: conformanceHarmonicProperties("Float", "2.4", false), want: 2.4},
		{name: "zero boundary", properties: conformanceHarmonicProperties("HFret", "0", false), want: 0},
		{name: "integer", properties: conformanceHarmonicProperties("HFret", "12", false), want: 12},
		{name: "legacy maximum", properties: conformanceHarmonicProperties("HFret", "127", false), want: 127},
		{name: "fret before type", properties: conformanceHarmonicProperties("HFret", "2.4", true), want: 2.4},
		{name: "HFret takes precedence", properties: []gpifProperty{{Name: "HarmonicType", HType: conformanceHarmonicPtr("Artificial")}, {Name: "HarmonicFret", HFret: conformanceHarmonicPtr("2.4"), Float: conformanceHarmonicPtr("12")}}, want: 2.4},
	} {
		t.Run(test.name, func(t *testing.T) {
			harmonic := conformanceHarmonicFromProperties(t, test.properties)
			if harmonic == nil || harmonic.Fret == nil || harmonic.FretFloat == nil {
				t.Fatalf("harmonic = %#v, want both fret views", harmonic)
			}
			run.Field("HarmonicEffect.FretFloat", *harmonic.FretFloat, test.want)
			run.Field("HarmonicEffect.Fret", *harmonic.Fret, int8(test.want))
		})
	}

	for _, reverse := range []bool{false, true} {
		name := "type before fret"
		if reverse {
			name = "fret before type"
		}
		t.Run("XML "+name, func(t *testing.T) {
			typeXML := `<Property name="HarmonicType"><HType>Artificial</HType></Property>`
			fretXML := `<Property name="HarmonicFret"><HFret>2.4</HFret></Property>`
			properties := typeXML + fretXML
			if reverse {
				properties = fretXML + typeXML
			}
			result, err := ParseWithOptions(conformanceHarmonicDiagnosticFixture(t, func(gpif string) string {
				return insertFirstNoteProperty(t, gpif, properties)
			}), ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			harmonic := conformanceHarmonicFirstNote(result.Song).Effect.Harmonic
			if harmonic == nil || harmonic.FretFloat == nil {
				t.Fatalf("harmonic = %#v, want artificial fret", harmonic)
			}
			run.Dispatch("gpifNoteToNote:p.Name", harmonic.Kind, HarmonicTypeArtificial)
			run.Field("HarmonicEffect.FretFloat", *harmonic.FretFloat, 2.4)
		})
	}

	for _, test := range []struct {
		name string
		xml  string
		code string
		kind ParseDiagnosticKind
	}{
		{name: "missing type payload", xml: `<Property name="HarmonicType"/>`, code: "GPIF.Note.Property.HarmonicType.MissingPayload", kind: ParseDiagnosticInvalidData},
		{name: "missing fret payload", xml: `<Property name="HarmonicFret"/>`, code: "GPIF.Note.Property.HarmonicFret.MissingPayload", kind: ParseDiagnosticInvalidData},
		{name: "unknown type", xml: `<Property name="HarmonicType"><HType>Crystal</HType></Property>`, code: "GPIF.Note.Property.HarmonicType.Unsupported", kind: ParseDiagnosticUnsupportedFeature},
		{name: "empty type", xml: `<Property name="HarmonicType"><HType/></Property>`, code: "GPIF.Note.Property.HarmonicType.Unsupported", kind: ParseDiagnosticUnsupportedFeature},
		{name: "malformed HFret", xml: `<Property name="HarmonicFret"><HFret>not-a-number</HFret></Property>`, code: "GPIF.Note.Property.HarmonicFret.Invalid", kind: ParseDiagnosticInvalidData},
		{name: "non-finite HFret", xml: `<Property name="HarmonicFret"><HFret>NaN</HFret></Property>`, code: "GPIF.Note.Property.HarmonicFret.Invalid", kind: ParseDiagnosticInvalidData},
		{name: "malformed Float", xml: `<Property name="HarmonicFret"><Float>+Inf</Float></Property>`, code: "GPIF.Note.Property.HarmonicFret.Invalid", kind: ParseDiagnosticInvalidData},
		{name: "negative fret", xml: `<Property name="HarmonicFret"><HFret>-0.5</HFret></Property>`, code: "GPIF.Note.Property.HarmonicFret.Invalid", kind: ParseDiagnosticInvalidData},
		{name: "fret narrowing boundary", xml: `<Property name="HarmonicFret"><HFret>128</HFret></Property>`, code: "GPIF.Note.Property.HarmonicFret.Invalid", kind: ParseDiagnosticInvalidData},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := assertHarmonicStrictDiagnostic(t, test.xml, test.code, test.kind)
			if test.code == "GPIF.Note.Property.HarmonicFret.Invalid" {
				harmonic := conformanceHarmonicFirstNote(result.Song).Effect.Harmonic
				if harmonic != nil && (harmonic.Fret != nil || harmonic.FretFloat != nil) {
					t.Fatalf("malformed fret populated harmonic = %#v", harmonic)
				}
			}
		})
	}

	t.Run("feedback remains distinct", func(t *testing.T) {
		data := conformanceHarmonicDiagnosticFixture(t, func(gpif string) string {
			return insertFirstNoteProperty(t, gpif,
				`<Property name="HarmonicType"><HType>Feedback</HType></Property><Property name="HarmonicFret"><HFret>12</HFret></Property>`)
		})
		result, err := ParseWithOptions(data, ParseOptions{Strict: true})
		if err != nil {
			t.Fatal(err)
		}
		harmonic := conformanceHarmonicFirstNote(result.Song).Effect.Harmonic
		if harmonic == nil {
			t.Fatal("feedback harmonic was dropped entirely")
		}
		run.Dispatch("gpifNoteToNote:p.HType", harmonic.Kind, HarmonicTypeFeedback)
	})
}

func TestConformanceHarmonicAuthorityAndValidation(t *testing.T) {
	runConformanceHarmonicAuthorityAndValidation(newConformanceRun(t))
}

func runConformanceHarmonicAuthorityAndValidation(run *conformanceRun) {
	t := run.t
	legacyFret := int8(3)
	exactFret := 2.4
	song := conformanceHarmonicSong(t)
	conformanceHarmonicFirstNote(song).Effect.Harmonic = &HarmonicEffect{
		Kind:      HarmonicTypeArtificial,
		Fret:      &legacyFret,
		FretFloat: &exactFret,
	}
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	run.Field("HarmonicEffect.Fret", *conformanceHarmonicFirstNote(song).Effect.Harmonic.Fret, legacyFret)
	run.Field("HarmonicEffect.FretFloat", *conformanceHarmonicFirstNote(song).Effect.Harmonic.FretFloat, exactFret)
	if !hasExportCode(report, "gp8.normalize.harmonic-fret-authority") {
		t.Fatalf("conflicting harmonic frets report = %#v, want authority normalization", report.Entries)
	}
	data, strictReport, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	var lossErr *ExportLossError
	if len(data) != 0 || !errors.As(err, &lossErr) || !hasExportCode(strictReport, "gp8.normalize.harmonic-fret-authority") {
		t.Fatalf("strict conflicting fret export = %d bytes, %#v, %v", len(data), strictReport.Entries, err)
	}
	allowed, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: reportCodes(report)}})
	if err != nil {
		t.Fatal(err)
	}
	run.Wire("gpifProperty.HFret", conformanceHarmonicWireValue(extractHarmonicWire(t, allowed), "HarmonicFret", "HFret"), "2.4")

	agreeing := conformanceHarmonicSong(t)
	legacyFret = 2
	conformanceHarmonicFirstNote(agreeing).Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeArtificial, Fret: &legacyFret, FretFloat: &exactFret}
	if agreeingReport := PreflightExport(agreeing, ExportFormatGP8, ExportOptions{}); hasExportCode(agreeingReport, "gp8.normalize.harmonic-fret-authority") {
		t.Errorf("agreeing harmonic frets report = %#v", agreeingReport.Entries)
	}

	for _, test := range []struct {
		name     string
		harmonic HarmonicEffect
		wantCode string
	}{
		{name: "unknown kind", harmonic: HarmonicEffect{Kind: HarmonicType(99)}, wantCode: "score.note.harmonic.kind"},
		{name: "negative legacy fret", harmonic: HarmonicEffect{Kind: HarmonicTypeNatural, Fret: conformanceHarmonicPtr(int8(-1))}, wantCode: "score.note.harmonic.fret"},
		{name: "negative exact fret", harmonic: HarmonicEffect{Kind: HarmonicTypeNatural, FretFloat: conformanceHarmonicPtr(-0.5)}, wantCode: "score.note.harmonic.fret-float"},
		{name: "not a number", harmonic: HarmonicEffect{Kind: HarmonicTypeNatural, FretFloat: conformanceHarmonicPtr(math.NaN())}, wantCode: "score.note.harmonic.fret-float"},
		{name: "infinity", harmonic: HarmonicEffect{Kind: HarmonicTypeNatural, FretFloat: conformanceHarmonicPtr(math.Inf(1))}, wantCode: "score.note.harmonic.fret-float"},
		{name: "legacy boundary", harmonic: HarmonicEffect{Kind: HarmonicTypeNatural, FretFloat: conformanceHarmonicPtr(128.0)}, wantCode: "score.note.harmonic.fret-float"},
	} {
		t.Run(test.name, func(t *testing.T) {
			caseSong := conformanceHarmonicSong(t)
			conformanceHarmonicFirstNote(caseSong).Effect.Harmonic = &test.harmonic
			diagnostics := ValidateSong(caseSong)
			if !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == test.wantCode }) {
				t.Errorf("diagnostics = %#v, want %s", diagnostics, test.wantCode)
			}
			report := PreflightExport(caseSong, ExportFormatGP8, ExportOptions{})
			if !hasExportCode(report, "gp8.reject."+test.wantCode) {
				t.Errorf("preflight = %#v, want rejection for %s", report.Entries, test.wantCode)
			}
			data, _, err := ExportWithReport(caseSong, ExportFormatGP8, ExportOptions{})
			if len(data) != 0 || err == nil {
				t.Errorf("invalid harmonic export = %d bytes, %v, want rejection", len(data), err)
			}
		})
	}
}

func conformanceHarmonicSong(t *testing.T) *Song {
	t.Helper()
	song := conformanceTechniqueSong(t)
	song.Tracks[0].Settings.Notation = true
	return song
}

func conformanceHarmonicFirstNote(song *Song) *Note {
	return &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
}

func conformanceHarmonicPtr[T any](value T) *T {
	return &value
}

func conformanceHarmonicProperties(element, value string, reverse bool) []gpifProperty {
	harmonicType := gpifProperty{Name: "HarmonicType", HType: conformanceHarmonicPtr("Artificial")}
	harmonicFret := gpifProperty{Name: "HarmonicFret"}
	if element == "HFret" {
		harmonicFret.HFret = &value
	} else {
		harmonicFret.Float = &value
	}
	if reverse {
		return []gpifProperty{harmonicFret, harmonicType}
	}
	return []gpifProperty{harmonicType, harmonicFret}
}

func conformanceHarmonicFromProperties(t *testing.T, properties []gpifProperty) *HarmonicEffect {
	t.Helper()
	note, err := gpifNoteToNote(&gpifNote{Properties: gpifProperties{Properties: properties}}, 6, false)
	if err != nil {
		t.Fatal(err)
	}
	return note.Effect.Harmonic
}

type conformanceHarmonicWireProperty struct {
	Name  string  `xml:"name,attr"`
	HType *string `xml:"HType"`
	HFret *string `xml:"HFret"`
	Float *string `xml:"Float"`
}

func extractHarmonicWire(t *testing.T, data []byte) []conformanceHarmonicWireProperty {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range archive.File {
		if file.Name != "Content/score.gpif" {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		contents, err := io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		var document struct {
			Notes struct {
				Items []struct {
					Properties struct {
						Items []conformanceHarmonicWireProperty `xml:"Property"`
					} `xml:"Properties"`
				} `xml:"Note"`
			} `xml:"Notes"`
		}
		if err := xml.Unmarshal(contents, &document); err != nil {
			t.Fatal(err)
		}
		if len(document.Notes.Items) == 0 {
			t.Fatal("GPIF has no notes")
		}
		return document.Notes.Items[0].Properties.Items
	}
	t.Fatal("GP8 archive has no score.gpif")
	return nil
}

func conformanceHarmonicWireValue(properties []conformanceHarmonicWireProperty, name, element string) string {
	for _, property := range properties {
		if property.Name != name {
			continue
		}
		var value *string
		switch element {
		case "HType":
			value = property.HType
		case "HFret":
			value = property.HFret
		case "Float":
			value = property.Float
		}
		if value != nil {
			return *value
		}
	}
	return ""
}

func conformanceHarmonicPropertyNames(properties []conformanceHarmonicWireProperty) []string {
	var names []string
	for _, property := range properties {
		if property.Name == "HarmonicType" || property.Name == "HarmonicFret" {
			names = append(names, property.Name)
		}
	}
	return names
}

func assertHarmonicStrictDiagnostic(t *testing.T, propertyXML, code string, kind ParseDiagnosticKind) *ParseResult {
	t.Helper()
	data := conformanceHarmonicDiagnosticFixture(t, func(gpif string) string {
		return insertFirstNoteProperty(t, gpif, propertyXML)
	})
	result, err := ParseWithOptions(data, ParseOptions{Strict: true})
	var strictErr *StrictParseError
	if !errors.As(err, &strictErr) {
		t.Fatalf("strict parse error = %v, want StrictParseError", err)
	}
	if result == nil || !slices.ContainsFunc(result.Diagnostics, func(diagnostic ParseDiagnostic) bool {
		return diagnostic.Code == code && diagnostic.Kind == kind && diagnostic.Feature == "harmonics" && diagnostic.SourcePath != "" && diagnostic.ObjectID != ""
	}) {
		t.Fatalf("diagnostics = %#v, want %s (%s) receipt", result, code, kind)
	}
	return result
}

func conformanceHarmonicDiagnosticFixture(t *testing.T, mutate func(string) string) []byte {
	t.Helper()
	data, err := Export(conformanceHarmonicSong(t), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	return rewriteConformanceGPIF(t, data, mutate)
}
