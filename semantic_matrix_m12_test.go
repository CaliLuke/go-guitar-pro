// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
)

func TestSemanticMatrixM12CurvePreservation(t *testing.T) {
	runSemanticMatrixM12CurvePreservation(newSemanticMatrixRun(t))
}

func runSemanticMatrixM12CurvePreservation(run *semanticMatrixRun) {
	t := run.t
	representable := []struct {
		name    string
		present bool
		points  []BendPoint
	}{
		{name: "nil"},
		{name: "empty", present: true, points: []BendPoint{}},
		{name: "two point rise", present: true, points: []BendPoint{{Position: 0}, {Position: 12, Value: 2}}},
		{name: "two point fall", present: true, points: []BendPoint{{Position: 0, Value: 2}, {Position: 12}}},
		{name: "three point up down", present: true, points: []BendPoint{{Position: 0}, {Position: 6, Value: 2}, {Position: 12}}},
		{name: "initial hold", present: true, points: []BendPoint{{Position: 0, Value: 2}, {Position: 3, Value: 2}, {Position: 12}}},
		{name: "final hold", present: true, points: []BendPoint{{Position: 0}, {Position: 9, Value: 2}, {Position: 12, Value: 2}}},
		{name: "four shared middle", present: true, points: []BendPoint{{Position: 0}, {Position: 3, Value: 2}, {Position: 9, Value: 2}, {Position: 12}}},
		{name: "coincident equal", present: true, points: []BendPoint{{Position: 0}, {Position: 6, Value: 2}, {Position: 6, Value: 2}, {Position: 12}}},
	}
	for _, target := range []string{"bend", "whammy"} {
		for _, test := range representable {
			t.Run(target+"/"+test.name, func(t *testing.T) {
				song := m12Song(t)
				if test.present {
					m12SetCurve(song, target, &BendEffect{Points: slices.Clone(test.points)})
				}
				data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
				if err != nil || len(data) == 0 {
					t.Fatalf("strict export = %d bytes, %#v, %v", len(data), report.Entries, err)
				}
			})
		}
	}
	t.Run("bend combined initial and final hold", func(t *testing.T) {
		song := m12Song(t)
		m12SetCurve(song, "bend", &BendEffect{Points: []BendPoint{{Position: 0, Value: 2}, {Position: 3, Value: 2}, {Position: 9}, {Position: 12}}})
		if _, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}}); err != nil {
			t.Fatalf("strict bend hold export: %#v, %v", report.Entries, err)
		}
	})

	collinear := []BendPoint{
		{Position: 0, Value: 0},
		{Position: 3, Value: 1},
		{Position: 6, Value: 2},
		{Position: 9, Value: 3},
		{Position: 12, Value: 4},
	}
	song := m12Song(t)
	song.Tracks[0].Settings.Notation = true
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	beat.Effect.TremoloBar = &BendEffect{Points: slices.Clone(collinear)}
	beat.Notes[0].Effect.Bend = &BendEffect{Points: slices.Clone(collinear)}

	run.Field("BeatEffects.TremoloBar", beat.Effect.TremoloBar, &BendEffect{Points: collinear})
	run.Field("NoteEffect.Bend", beat.Notes[0].Effect.Bend, &BendEffect{Points: collinear})
	run.Field("BendEffect.Points", beat.Notes[0].Effect.Bend.Points, collinear)
	for index, point := range collinear {
		run.Field("BendPoint.Position", beat.Notes[0].Effect.Bend.Points[index].Position, point.Position)
		run.Field("BendPoint.Value", beat.Notes[0].Effect.Bend.Points[index].Value, point.Value)
		run.Field("BendPoint.Vibrato", beat.Notes[0].Effect.Bend.Points[index].Vibrato, false)
	}

	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true},
	})
	if err != nil {
		t.Fatalf("strict collinear export: %#v, %v", report.Entries, err)
	}
	wire := readM12Wire(t, data)
	run.Wire("gpifProperty.Float", []string{
		wire.bend["BendOriginOffset"], wire.bend["BendMiddleOffset1"], wire.bend["BendMiddleValue"],
		wire.bend["BendDestinationOffset"], wire.bend["BendDestinationValue"],
	}, []string{"0.000000", "50.000000", "50.000000", "100.000000", "100.000000"})
	run.Wire("gpifBeat.Whammy", len(wire.whammy) != 0, true)
	run.Wire("gpifWhammy.OriginValue", wire.whammy["originValue"], "0.000000")
	run.Wire("gpifWhammy.MiddleValue", wire.whammy["middleValue"], "50.000000")
	run.Wire("gpifWhammy.DestinationValue", wire.whammy["destinationValue"], "100.000000")
	run.Wire("gpifWhammy.OriginOffset", wire.whammy["originOffset"], "0.000000")
	run.Wire("gpifWhammy.MiddleOffset1", wire.whammy["middleOffset1"], "50.000000")
	run.Wire("gpifWhammy.MiddleOffset2", wire.whammy["middleOffset2"], "50.000000")
	run.Wire("gpifWhammy.DestinationOffset", wire.whammy["destinationOffset"], "100.000000")
	for name, want := range map[string]string{
		"BendOriginOffset":      "0.000000",
		"BendMiddleOffset1":     "50.000000",
		"BendMiddleValue":       "50.000000",
		"BendDestinationOffset": "100.000000",
		"BendDestinationValue":  "100.000000",
	} {
		if got := wire.bend[name]; got != want {
			t.Errorf("GPIF %s = %q, want %q", name, got, want)
		}
	}
	for name, want := range map[string]string{
		"originOffset": "0.000000", "middleOffset1": "50.000000", "middleValue": "50.000000",
		"destinationOffset": "100.000000", "destinationValue": "100.000000",
	} {
		if got := wire.whammy[name]; got != want {
			t.Errorf("GPIF Whammy %s = %q, want %q", name, got, want)
		}
	}
	if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
		got := m12AlphaTabFirstBend(t, data)
		want := []BendPoint{{Position: 0, Value: 0}, {Position: 12, Value: 4}}
		if !slices.Equal(got, want) {
			t.Fatalf("AlphaTab bend controls = %#v, want %#v", got, want)
		}
	}
}

func TestSemanticMatrixM12CurveLossPolicy(t *testing.T) {
	runSemanticMatrixM12CurveLossPolicy(newSemanticMatrixRun(t))
}

func runSemanticMatrixM12CurveLossPolicy(run *semanticMatrixRun) {
	t := run.t
	alternatingFive := []BendPoint{{Position: 0}, {Position: 3, Value: 2}, {Position: 6}, {Position: 9, Value: 2}, {Position: 12}}
	cases := []struct {
		name   string
		points []BendPoint
		code   string
	}{
		{name: "one point", points: []BendPoint{{Position: 0, Value: 2}}, code: "gp8.omit.%s-curve"},
		{name: "five point alternating", points: alternatingFive, code: "gp8.omit.%s-curve"},
		{name: "unequal middle values", points: []BendPoint{{Position: 0}, {Position: 3, Value: 1}, {Position: 9, Value: 2}, {Position: 12}}, code: "gp8.normalize.%s-curve"},
		{name: "coincident unequal values", points: []BendPoint{{Position: 0}, {Position: 6, Value: 1}, {Position: 6, Value: 2}, {Position: 12}}, code: "gp8.normalize.%s-curve"},
		{name: "odd midpoint sum", points: []BendPoint{{Position: 0}, {Position: 12, Value: 1}}, code: "gp8.normalize.%s-curve"},
	}
	for _, target := range []string{"bend", "whammy"} {
		for _, test := range cases {
			t.Run(target+"/"+test.name, func(t *testing.T) {
				song := m12Song(t)
				song.Tracks[0].Settings.Notation = true
				effect := &BendEffect{Points: slices.Clone(test.points)}
				beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
				if target == "bend" {
					beat.Notes[0].Effect.Bend = effect
				} else {
					beat.Effect.TremoloBar = effect
				}
				wantCode := strings.Replace(test.code, "%s", target, 1)
				report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
				if !hasExportCode(report, wantCode) {
					t.Fatalf("report = %#v, want %s", report.Entries, wantCode)
				}
				data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
				var lossErr *ExportLossError
				if len(data) != 0 || !errors.As(err, &lossErr) {
					t.Fatalf("strict export = %d bytes, %v", len(data), err)
				}
			})
		}
	}
	t.Run("whammy combined initial and final hold", func(t *testing.T) {
		song := m12Song(t)
		m12SetCurve(song, "whammy", &BendEffect{Points: []BendPoint{{Position: 0, Value: 2}, {Position: 3, Value: 2}, {Position: 9}, {Position: 12}}})
		report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
		if !hasExportCode(report, "gp8.normalize.whammy-curve") {
			t.Fatalf("report = %#v, want whammy normalization", report.Entries)
		}
		if data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}}); len(data) != 0 || err == nil {
			t.Fatalf("strict whammy hold export = %d bytes, %v", len(data), err)
		}
	})

	summary := m12Song(t)
	summary.Tracks[0].Settings.Notation = true
	beat := &summary.Tracks[0].Measures[0].Voices[0].Beats[0]
	beat.Effect.TremoloBar = &BendEffect{Kind: BendTypeDive, Value: 25, Points: []BendPoint{{Position: 0, Vibrato: true}, {Position: 12, Value: -2}}}
	beat.Notes[0].Effect.Bend = &BendEffect{Kind: BendTypePrebend, Value: 50, Points: []BendPoint{{Position: 0, Vibrato: true}, {Position: 12, Value: 2}}}
	run.Field("BendEffect.Kind", beat.Notes[0].Effect.Bend.Kind, BendTypePrebend)
	run.Field("BendEffect.Value", beat.Notes[0].Effect.Bend.Value, int16(50))
	run.Field("BendPoint.Vibrato", beat.Notes[0].Effect.Bend.Points[0].Vibrato, true)
	report := PreflightExport(summary, ExportFormatGP8, ExportOptions{})
	for _, code := range []string{"gp8.omit.bend-summary", "gp8.omit.whammy-summary", "gp8.omit.bend-point-vibrato", "gp8.omit.whammy-point-vibrato"} {
		if !hasExportCode(report, code) {
			t.Errorf("report = %#v, want %s", report.Entries, code)
		}
	}
	if data, _, err := ExportWithReport(summary, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}}); len(data) != 0 || err == nil {
		t.Fatalf("strict summary export = %d bytes, %v", len(data), err)
	}
}

func TestSemanticMatrixM12CurveValidation(t *testing.T) {
	runSemanticMatrixM12CurveValidation(newSemanticMatrixRun(t))
}

func runSemanticMatrixM12CurveValidation(run *semanticMatrixRun) {
	t := run.t
	for _, test := range []struct {
		name   string
		points []BendPoint
	}{
		{name: "position overflow", points: []BendPoint{{Position: 0}, {Position: 13}}},
		{name: "unsorted", points: []BendPoint{{Position: 0}, {Position: 9, Value: 1}, {Position: 3, Value: 2}}},
	} {
		for _, target := range []string{"bend", "whammy"} {
			t.Run(target+"/"+test.name, func(t *testing.T) {
				song := m12Song(t)
				song.Tracks[0].Settings.Notation = true
				beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
				if target == "bend" {
					beat.Notes[0].Effect.Bend = &BendEffect{Points: test.points}
				} else {
					beat.Effect.TremoloBar = &BendEffect{Points: test.points}
				}
				report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
				if !slices.ContainsFunc(report.Entries, func(entry ExportReportEntry) bool { return entry.Disposition == ExportDispositionRejected }) {
					t.Fatalf("report = %#v, want rejected curve", report.Entries)
				}
			})
		}
	}
	for _, target := range []string{"bend", "whammy"} {
		t.Run(target+" invalid summary", func(t *testing.T) {
			song := m12Song(t)
			m12SetCurve(song, target, &BendEffect{Kind: BendType(99), Value: 32767, Points: []BendPoint{{Position: 0}, {Position: 12, Value: 2}}})
			report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
			if count := len(slices.DeleteFunc(slices.Clone(report.Entries), func(entry ExportReportEntry) bool { return entry.Disposition != ExportDispositionRejected })); count != 1 {
				t.Fatalf("report = %#v, want one kind rejection", report.Entries)
			}
		})
	}

	for _, test := range []struct{ property, value string }{
		{property: "BendOriginOffset", value: "NaN"},
		{property: "BendOriginOffset", value: "Inf"},
		{property: "BendOriginOffset", value: "-Inf"},
		{property: "BendOriginOffset", value: "-1"},
		{property: "BendDestinationOffset", value: "101"},
		{property: "BendOriginValue", value: "-3201"},
		{property: "BendDestinationValue", value: "3176"},
	} {
		t.Run("GPIF "+test.property+" "+test.value, func(t *testing.T) {
			data := diagnosticGP8Fixture(t, func(gpif string) string {
				return insertFirstNoteProperty(t, gpif, `<Property name="Bended"><Enable/></Property><Property name="`+test.property+`"><Float>`+test.value+`</Float></Property>`)
			})
			result, err := ParseWithOptions(data, ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if !slices.ContainsFunc(result.Diagnostics, func(d ParseDiagnostic) bool {
				return d.Code == "GPIF.Note.Property.BendNumber.Invalid" && d.Kind == ParseDiagnosticInvalidData
			}) {
				t.Fatalf("diagnostics = %#v, want invalid bend number", result.Diagnostics)
			}
			_, strictErr := ParseWithOptions(data, ParseOptions{Strict: true})
			var rejected *StrictParseError
			if !errors.As(strictErr, &rejected) {
				t.Fatalf("strict error = %v, want StrictParseError", strictErr)
			}
		})
	}
	for _, test := range []struct{ attribute, value string }{
		{attribute: "originOffset", value: "NaN"},
		{attribute: "middleOffset1", value: "-1"},
		{attribute: "destinationOffset", value: "101"},
		{attribute: "originValue", value: "-3201"},
		{attribute: "destinationValue", value: "3176"},
	} {
		t.Run("GPIF whammy "+test.attribute+" "+test.value, func(t *testing.T) {
			data := diagnosticGP8Fixture(t, func(gpif string) string {
				whammy := m12WhammyXML(test.attribute, test.value)
				return insertFirstGPIFObjectChild(t, gpif, "<Beats>", "</Beat>", whammy)
			})
			result, err := ParseWithOptions(data, ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if !slices.ContainsFunc(result.Diagnostics, func(d ParseDiagnostic) bool {
				return d.Code == "GPIF.Beat.Whammy.Invalid" && d.Kind == ParseDiagnosticInvalidData
			}) {
				t.Fatalf("diagnostics = %#v, want invalid whammy number", result.Diagnostics)
			}
		})
	}

	for _, test := range []struct {
		name  string
		value string
	}{
		{name: "offset", value: "12.5"},
		{name: "endpoint offset", value: "95.833333"},
		{name: "value", value: "37.5"},
	} {
		t.Run("GPIF quantized "+test.name, func(t *testing.T) {
			property := "BendMiddleOffset1"
			if test.name == "endpoint offset" {
				property = "BendDestinationOffset"
			}
			if test.name == "value" {
				property = "BendOriginValue"
			}
			data := diagnosticGP8Fixture(t, func(gpif string) string {
				return insertFirstNoteProperty(t, gpif, `<Property name="Bended"><Enable/></Property><Property name="`+property+`"><Float>`+test.value+`</Float></Property>`)
			})
			result, err := ParseWithOptions(data, ParseOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if !slices.ContainsFunc(result.Diagnostics, func(d ParseDiagnostic) bool {
				return d.Code == "GPIF.Note.Property.BendNumber.Quantized" && d.Kind == ParseDiagnosticLossyProjection
			}) {
				t.Fatalf("diagnostics = %#v, want bend quantization", result.Diagnostics)
			}
		})
	}
	t.Run("GPIF quantized whammy offset", func(t *testing.T) {
		data := diagnosticGP8Fixture(t, func(gpif string) string {
			return insertFirstGPIFObjectChild(t, gpif, "<Beats>", "</Beat>", m12WhammyXML("middleOffset1", "12.5"))
		})
		result, err := ParseWithOptions(data, ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.ContainsFunc(result.Diagnostics, func(d ParseDiagnostic) bool {
			return d.Code == "GPIF.Beat.Whammy.Quantized" && d.Kind == ParseDiagnosticLossyProjection
		}) {
			t.Fatalf("diagnostics = %#v, want whammy quantization", result.Diagnostics)
		}
	})

}

func readM12GPIF(t *testing.T, data []byte) []byte {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	return readZipMember(t, archive, "Content/score.gpif")
}

type m12Wire struct {
	bend   map[string]string
	whammy map[string]string
}

func readM12Wire(t *testing.T, data []byte) m12Wire {
	t.Helper()
	var document struct {
		Notes struct {
			Notes []struct {
				Properties struct {
					Properties []struct {
						Name  string `xml:"name,attr"`
						Float string `xml:"Float"`
					} `xml:"Property"`
				} `xml:"Properties"`
			} `xml:"Note"`
		} `xml:"Notes"`
		Beats struct {
			Beats []struct {
				Whammy *struct {
					OriginValue       string `xml:"originValue,attr"`
					MiddleValue       string `xml:"middleValue,attr"`
					DestinationValue  string `xml:"destinationValue,attr"`
					OriginOffset      string `xml:"originOffset,attr"`
					MiddleOffset1     string `xml:"middleOffset1,attr"`
					MiddleOffset2     string `xml:"middleOffset2,attr"`
					DestinationOffset string `xml:"destinationOffset,attr"`
				} `xml:"Whammy"`
			} `xml:"Beat"`
		} `xml:"Beats"`
	}
	if err := xml.Unmarshal(readM12GPIF(t, data), &document); err != nil {
		t.Fatal(err)
	}
	result := m12Wire{bend: make(map[string]string), whammy: make(map[string]string)}
	for _, note := range document.Notes.Notes {
		for _, property := range note.Properties.Properties {
			if strings.HasPrefix(property.Name, "Bend") {
				result.bend[property.Name] = property.Float
			}
		}
	}
	for _, beat := range document.Beats.Beats {
		if beat.Whammy == nil {
			continue
		}
		result.whammy = map[string]string{
			"originValue": beat.Whammy.OriginValue, "middleValue": beat.Whammy.MiddleValue,
			"destinationValue": beat.Whammy.DestinationValue, "originOffset": beat.Whammy.OriginOffset,
			"middleOffset1": beat.Whammy.MiddleOffset1, "middleOffset2": beat.Whammy.MiddleOffset2,
			"destinationOffset": beat.Whammy.DestinationOffset,
		}
		break
	}
	return result
}

func m12Song(t *testing.T) *Song {
	t.Helper()
	song := semanticExportProbeSong(t)
	for trackIndex := range song.Tracks {
		for measureIndex := range song.Tracks[trackIndex].Measures {
			for voiceIndex := range song.Tracks[trackIndex].Measures[measureIndex].Voices {
				for beatIndex := range song.Tracks[trackIndex].Measures[measureIndex].Voices[voiceIndex].Beats {
					for noteIndex := range song.Tracks[trackIndex].Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes {
						effect := &song.Tracks[trackIndex].Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes[noteIndex].Effect
						effect.LeftHandFinger = FingeringOpen
						effect.RightHandFinger = FingeringOpen
					}
				}
			}
		}
	}
	return song
}

func m12SetCurve(song *Song, target string, effect *BendEffect) {
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	if target == "bend" {
		beat.Notes[0].Effect.Bend = effect
		return
	}
	beat.Effect.TremoloBar = effect
}

func m12WhammyXML(attribute, value string) string {
	values := map[string]string{
		"originValue": "0", "middleValue": "0", "destinationValue": "0",
		"originOffset": "0", "middleOffset1": "50", "middleOffset2": "50", "destinationOffset": "100",
	}
	values[attribute] = value
	return `<Whammy originValue="` + values["originValue"] + `" middleValue="` + values["middleValue"] +
		`" destinationValue="` + values["destinationValue"] + `" originOffset="` + values["originOffset"] +
		`" middleOffset1="` + values["middleOffset1"] + `" middleOffset2="` + values["middleOffset2"] +
		`" destinationOffset="` + values["destinationOffset"] + `"/>`
}

func m12AlphaTabFirstBend(t *testing.T, data []byte) []BendPoint {
	t.Helper()
	root, ok := readAlphaTabScore(t, writeConformanceFixture(t, data)).(map[string]any)
	if !ok {
		t.Fatalf("AlphaTab root = %T", root)
	}
	tracks := root["tracks"].([]any)
	staves := tracks[0].(map[string]any)["staves"].([]any)
	bars := staves[0].(map[string]any)["bars"].([]any)
	voices := bars[0].(map[string]any)["voices"].([]any)
	beats := voices[0].(map[string]any)["beats"].([]any)
	notes := beats[0].(map[string]any)["notes"].([]any)
	effects := notes[0].(map[string]any)["effects"].(map[string]any)
	points := effects["bend"].([]any)
	result := make([]BendPoint, 0, len(points))
	for _, raw := range points {
		point := raw.(map[string]any)
		result = append(result, BendPoint{Position: uint8(point["position"].(float64)), Value: int8(point["value"].(float64))})
	}
	return result
}
