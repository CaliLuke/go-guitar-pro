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

func TestSemanticMatrixM13HarmonicVariants(t *testing.T) {
	runSemanticMatrixM13HarmonicVariants(newSemanticMatrixRun(t))
}

func runSemanticMatrixM13HarmonicVariants(run *semanticMatrixRun) {
	t := run.t
	legacyFret := int8(2)
	exactFret := 2.4
	for _, test := range []struct {
		name string
		kind HarmonicType
		wire string
	}{
		{name: "natural", kind: HarmonicTypeNatural, wire: "Natural"},
		{name: "artificial", kind: HarmonicTypeArtificial, wire: "Artificial"},
		{name: "pinch", kind: HarmonicTypePinch, wire: "Pinch"},
		{name: "tapped", kind: HarmonicTypeTapped, wire: "Tap"},
		{name: "semi", kind: HarmonicTypeSemi, wire: "Semi"},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := m13Song(t)
			harmonic := &HarmonicEffect{Kind: test.kind, Fret: &legacyFret, FretFloat: &exactFret}
			m13FirstNote(song).Effect.Harmonic = harmonic
			run.Field("HarmonicEffect.Kind", harmonic.Kind, test.kind)
			run.Field("HarmonicEffect.Fret", *harmonic.Fret, legacyFret)
			run.Field("HarmonicEffect.FretFloat", *harmonic.FretFloat, exactFret)
			if got := gp8HarmonicType(test.kind); got != test.wire {
				t.Errorf("gp8HarmonicType(%d) = %q, want %q", test.kind, got, test.wire)
			}
			data, err := Export(song, ExportFormatGP8)
			if err != nil {
				t.Fatal(err)
			}
			properties := extractM13HarmonicWire(t, data)
			run.Wire("gpifProperty.HType", m13WireValue(properties, "HarmonicType", "HType"), test.wire)
			run.Wire("gpifProperty.HFret", m13WireValue(properties, "HarmonicFret", "HFret"), "2.4")
			run.Wire("gpifProperty.Float", m13WireValue(properties, "HarmonicFret", "Float"), "")
			run.Wire("gpifProperty.Name", m13HarmonicPropertyNames(properties), []string{"HarmonicType", "HarmonicFret"})
		})
	}
	if got := gp8HarmonicType(HarmonicType(99)); got != "" {
		t.Errorf("gp8HarmonicType(99) = %q, want empty", got)
	}

	t.Run("no harmonic", func(t *testing.T) {
		song := m13Song(t)
		m13FirstNote(song).Effect.Harmonic = nil
		data, err := Export(song, ExportFormatGP8)
		if err != nil {
			t.Fatal(err)
		}
		properties := extractM13HarmonicWire(t, data)
		run.Wire("gpifProperty.Name", m13HarmonicPropertyNames(properties), []string(nil))
	})

	t.Run("integer fret", func(t *testing.T) {
		song := m13Song(t)
		fret := int8(12)
		m13FirstNote(song).Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeNatural, Fret: &fret}
		data, err := Export(song, ExportFormatGP8)
		if err != nil {
			t.Fatal(err)
		}
		properties := extractM13HarmonicWire(t, data)
		run.Wire("gpifProperty.HFret", m13WireValue(properties, "HarmonicFret", "HFret"), "12")
	})

	t.Run("pitch spelling loss is explicit", func(t *testing.T) {
		pitch := PitchClass{Note: "C#", Just: 0, Accidental: 1, Value: 1, Sharp: true}
		octave := OctaveOttava
		song := m13Song(t)
		harmonic := &HarmonicEffect{Kind: HarmonicTypeArtificial, Pitch: &pitch, Octave: &octave, Fret: &legacyFret, FretFloat: &exactFret}
		m13FirstNote(song).Effect.Harmonic = harmonic
		run.Field("HarmonicEffect.Pitch", *harmonic.Pitch, pitch)
		run.Field("HarmonicEffect.Octave", *harmonic.Octave, octave)
		run.Field("PitchClass.Note", harmonic.Pitch.Note, pitch.Note)
		run.Field("PitchClass.Just", harmonic.Pitch.Just, pitch.Just)
		run.Field("PitchClass.Accidental", harmonic.Pitch.Accidental, pitch.Accidental)
		run.Field("PitchClass.Value", harmonic.Pitch.Value, pitch.Value)
		run.Field("PitchClass.Sharp", harmonic.Pitch.Sharp, pitch.Sharp)
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

func TestSemanticMatrixM13GPIFCompatibilityAndDiagnostics(t *testing.T) {
	runSemanticMatrixM13GPIFCompatibilityAndDiagnostics(newSemanticMatrixRun(t))
}

func runSemanticMatrixM13GPIFCompatibilityAndDiagnostics(run *semanticMatrixRun) {
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
		{source: "Feedback", want: HarmonicTypeSemi},
	} {
		t.Run(test.source, func(t *testing.T) {
			context := &parseContext{format: "GPIF"}
			gpifAuditNoteProperty(context, "n1", `/GPIF/Notes/Note[@id="n1"]`, gpifProperty{Name: "HarmonicType", HType: &test.source}, nil)
			gotDisposition := "preserved"
			wantDisposition := "preserved"
			if test.source == "Feedback" {
				wantDisposition = string(ParseDiagnosticLossyProjection)
			}
			if len(context.diagnostics) != 0 {
				gotDisposition = string(context.diagnostics[0].Kind)
			}
			run.Dispatch("gpifAuditNoteProperty:property.HType", gotDisposition, wantDisposition)

			harmonic := m13HarmonicFromProperties(t, []gpifProperty{{Name: "HarmonicType", HType: &test.source}})
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
		{name: "HFret native", properties: m13HarmonicProperties("HFret", "2.4", false), want: 2.4},
		{name: "Float compatibility", properties: m13HarmonicProperties("Float", "2.4", false), want: 2.4},
		{name: "zero boundary", properties: m13HarmonicProperties("HFret", "0", false), want: 0},
		{name: "integer", properties: m13HarmonicProperties("HFret", "12", false), want: 12},
		{name: "legacy maximum", properties: m13HarmonicProperties("HFret", "127", false), want: 127},
		{name: "fret before type", properties: m13HarmonicProperties("HFret", "2.4", true), want: 2.4},
		{name: "HFret takes precedence", properties: []gpifProperty{{Name: "HarmonicType", HType: m13Ptr("Artificial")}, {Name: "HarmonicFret", HFret: m13Ptr("2.4"), Float: m13Ptr("12")}}, want: 2.4},
	} {
		t.Run(test.name, func(t *testing.T) {
			harmonic := m13HarmonicFromProperties(t, test.properties)
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
			result, err := ParseWithOptions(m13DiagnosticFixture(t, func(gpif string) string {
				return insertFirstNoteProperty(t, gpif, properties)
			}), ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			harmonic := m13FirstNote(result.Song).Effect.Harmonic
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
			result := assertM13StrictDiagnostic(t, test.xml, test.code, test.kind)
			if test.code == "GPIF.Note.Property.HarmonicFret.Invalid" {
				harmonic := m13FirstNote(result.Song).Effect.Harmonic
				if harmonic != nil && (harmonic.Fret != nil || harmonic.FretFloat != nil) {
					t.Fatalf("malformed fret populated harmonic = %#v", harmonic)
				}
			}
		})
	}

	t.Run("feedback loss stays visible", func(t *testing.T) {
		result := assertM13StrictDiagnostic(t,
			`<Property name="HarmonicType"><HType>Feedback</HType></Property><Property name="HarmonicFret"><HFret>12</HFret></Property>`,
			"GPIF.Note.Property.HarmonicType.Feedback", ParseDiagnosticLossyProjection)
		harmonic := m13FirstNote(result.Song).Effect.Harmonic
		if harmonic == nil {
			t.Fatal("feedback harmonic was dropped entirely")
		}
		run.Dispatch("gpifNoteToNote:p.HType", harmonic.Kind, HarmonicTypeSemi)
	})
}

func TestSemanticMatrixM13HarmonicAuthorityAndValidation(t *testing.T) {
	runSemanticMatrixM13HarmonicAuthorityAndValidation(newSemanticMatrixRun(t))
}

func runSemanticMatrixM13HarmonicAuthorityAndValidation(run *semanticMatrixRun) {
	t := run.t
	legacyFret := int8(3)
	exactFret := 2.4
	song := m13Song(t)
	m13FirstNote(song).Effect.Harmonic = &HarmonicEffect{
		Kind:      HarmonicTypeArtificial,
		Fret:      &legacyFret,
		FretFloat: &exactFret,
	}
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	run.Field("HarmonicEffect.Fret", *m13FirstNote(song).Effect.Harmonic.Fret, legacyFret)
	run.Field("HarmonicEffect.FretFloat", *m13FirstNote(song).Effect.Harmonic.FretFloat, exactFret)
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
	run.Wire("gpifProperty.HFret", m13WireValue(extractM13HarmonicWire(t, allowed), "HarmonicFret", "HFret"), "2.4")

	agreeing := m13Song(t)
	legacyFret = 2
	m13FirstNote(agreeing).Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeArtificial, Fret: &legacyFret, FretFloat: &exactFret}
	if agreeingReport := PreflightExport(agreeing, ExportFormatGP8, ExportOptions{}); hasExportCode(agreeingReport, "gp8.normalize.harmonic-fret-authority") {
		t.Errorf("agreeing harmonic frets report = %#v", agreeingReport.Entries)
	}

	for _, test := range []struct {
		name     string
		harmonic HarmonicEffect
		wantCode string
	}{
		{name: "unknown kind", harmonic: HarmonicEffect{Kind: HarmonicType(99)}, wantCode: "score.note.harmonic.kind"},
		{name: "negative legacy fret", harmonic: HarmonicEffect{Kind: HarmonicTypeNatural, Fret: m13Ptr(int8(-1))}, wantCode: "score.note.harmonic.fret"},
		{name: "negative exact fret", harmonic: HarmonicEffect{Kind: HarmonicTypeNatural, FretFloat: m13Ptr(-0.5)}, wantCode: "score.note.harmonic.fret-float"},
		{name: "not a number", harmonic: HarmonicEffect{Kind: HarmonicTypeNatural, FretFloat: m13Ptr(math.NaN())}, wantCode: "score.note.harmonic.fret-float"},
		{name: "infinity", harmonic: HarmonicEffect{Kind: HarmonicTypeNatural, FretFloat: m13Ptr(math.Inf(1))}, wantCode: "score.note.harmonic.fret-float"},
		{name: "legacy boundary", harmonic: HarmonicEffect{Kind: HarmonicTypeNatural, FretFloat: m13Ptr(128.0)}, wantCode: "score.note.harmonic.fret-float"},
	} {
		t.Run(test.name, func(t *testing.T) {
			caseSong := m13Song(t)
			m13FirstNote(caseSong).Effect.Harmonic = &test.harmonic
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

func m13Song(t *testing.T) *Song {
	t.Helper()
	song := m11Song(t)
	song.Tracks[0].Settings.Notation = true
	return song
}

func m13FirstNote(song *Song) *Note {
	return &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
}

func m13Ptr[T any](value T) *T {
	return &value
}

func m13HarmonicProperties(element, value string, reverse bool) []gpifProperty {
	harmonicType := gpifProperty{Name: "HarmonicType", HType: m13Ptr("Artificial")}
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

func m13HarmonicFromProperties(t *testing.T, properties []gpifProperty) *HarmonicEffect {
	t.Helper()
	note, err := gpifNoteToNote(&gpifNote{Properties: gpifProperties{Properties: properties}}, 6, false)
	if err != nil {
		t.Fatal(err)
	}
	return note.Effect.Harmonic
}

type m13WireProperty struct {
	Name  string  `xml:"name,attr"`
	HType *string `xml:"HType"`
	HFret *string `xml:"HFret"`
	Float *string `xml:"Float"`
}

func extractM13HarmonicWire(t *testing.T, data []byte) []m13WireProperty {
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
						Items []m13WireProperty `xml:"Property"`
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

func m13WireValue(properties []m13WireProperty, name, element string) string {
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

func m13HarmonicPropertyNames(properties []m13WireProperty) []string {
	var names []string
	for _, property := range properties {
		if property.Name == "HarmonicType" || property.Name == "HarmonicFret" {
			names = append(names, property.Name)
		}
	}
	return names
}

func assertM13StrictDiagnostic(t *testing.T, propertyXML, code string, kind ParseDiagnosticKind) *ParseResult {
	t.Helper()
	data := m13DiagnosticFixture(t, func(gpif string) string {
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

func m13DiagnosticFixture(t *testing.T, mutate func(string) string) []byte {
	t.Helper()
	data, err := Export(m13Song(t), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	return rewriteConformanceGPIF(t, data, mutate)
}
