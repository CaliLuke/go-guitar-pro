// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"bytes"
	"encoding/binary"
	"errors"
	"reflect"
	"slices"
	"testing"
)

const whammyExportWhammyCurveCode = "gp8.normalize.whammy-curve"

type whammyExportWhammyCase struct {
	name       string
	effect     BendEffect
	wantTarget []BendPoint
	wantCodes  []string
}

func TestGP8WhammyMiddleHoldReportsInterpretedLoss(t *testing.T) {
	for _, test := range whammyExportWhammyCases() {
		t.Run(test.name, func(t *testing.T) {
			whammyExportAssertExportBehavior(t, &test.effect, test.wantTarget, test.wantCodes)
		})
	}
}

func TestAlphaTabGP8WhammyTargetInterpretation(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, test := range whammyExportWhammyCases() {
		t.Run("programmatic/"+test.name, func(t *testing.T) {
			data := whammyExportAssertExportBehavior(t, &test.effect, test.wantTarget, test.wantCodes)
			if got := whammyExportAlphaTabWhammy(t, data); !slices.Equal(got, test.wantTarget) {
				t.Fatalf("AlphaTab interpreted target points = %#v, want %#v", got, test.wantTarget)
			}
		})
	}

	for _, test := range whammyExportParsedFixtureCases(t) {
		t.Run("parsed-binary/"+test.name, func(t *testing.T) {
			data := whammyExportAssertExportBehavior(t, test.effect, test.wantTarget, test.wantCodes)
			if got := whammyExportAlphaTabWhammy(t, data); !slices.Equal(got, test.wantTarget) {
				t.Fatalf("AlphaTab interpreted parsed-binary target points = %#v, want %#v", got, test.wantTarget)
			}
		})
	}
}

func TestGP8WhammyParsedBinaryPaths(t *testing.T) {
	for _, test := range whammyExportWhammyCases()[2:4] {
		t.Run("raw-record/"+test.name, func(t *testing.T) {
			effect := whammyExportReadBinaryWhammy(t, test.effect.Points)
			if !reflect.DeepEqual(effect, &test.effect) {
				t.Fatalf("binary whammy = %#v, want %#v", effect, &test.effect)
			}
			whammyExportAssertExportBehavior(t, effect, test.wantTarget, test.wantCodes)
		})
	}

	for _, test := range whammyExportParsedFixtureCases(t) {
		t.Run("public-fixture/"+test.name, func(t *testing.T) {
			before := conformanceContractSnapshot(test.owner, false)
			whammyExportAssertExportBehavior(t, test.effect, test.wantTarget, test.wantCodes)
			if got := conformanceContractSnapshot(test.owner, false); got != before {
				t.Fatal("export mutated the parsed binary source score")
			}
		})
	}
}

func whammyExportWhammyCases() []whammyExportWhammyCase {
	normalized := []string{whammyExportWhammyCurveCode}
	return []whammyExportWhammyCase{
		{
			name: "zero-start fall early hold 1-4",
			effect: BendEffect{Points: []BendPoint{
				{Position: 0, Value: 0}, {Position: 1, Value: -4},
				{Position: 4, Value: -4}, {Position: 12, Value: -8},
			}},
			wantTarget: []BendPoint{{Position: 0, Value: 0}, {Position: 12, Value: -8}},
			wantCodes:  normalized,
		},
		{
			name: "zero-start rise early hold 1-4",
			effect: BendEffect{Points: []BendPoint{
				{Position: 0, Value: 0}, {Position: 1, Value: 4},
				{Position: 4, Value: 4}, {Position: 12, Value: 8},
			}},
			wantTarget: []BendPoint{{Position: 0, Value: 0}, {Position: 12, Value: 8}},
			wantCodes:  normalized,
		},
		{
			name: "zero-start fall middle hold 3-9",
			effect: BendEffect{Points: []BendPoint{
				{Position: 0, Value: 0}, {Position: 3, Value: -4},
				{Position: 9, Value: -4}, {Position: 12, Value: -8},
			}},
			wantTarget: []BendPoint{{Position: 0, Value: 0}, {Position: 12, Value: -8}},
			wantCodes:  normalized,
		},
		{
			name: "zero-start rise middle hold 3-9",
			effect: BendEffect{Points: []BendPoint{
				{Position: 0, Value: 0}, {Position: 3, Value: 4},
				{Position: 9, Value: 4}, {Position: 12, Value: 8},
			}},
			wantTarget: []BendPoint{{Position: 0, Value: 0}, {Position: 12, Value: 8}},
			wantCodes:  normalized,
		},
		{
			name: "zero-start fall late hold 8-11",
			effect: BendEffect{Points: []BendPoint{
				{Position: 0, Value: 0}, {Position: 8, Value: -4},
				{Position: 11, Value: -4}, {Position: 12, Value: -8},
			}},
			wantTarget: []BendPoint{{Position: 0, Value: 0}, {Position: 12, Value: -8}},
			wantCodes:  normalized,
		},
		{
			name: "zero-start rise late hold 8-11",
			effect: BendEffect{Points: []BendPoint{
				{Position: 0, Value: 0}, {Position: 8, Value: 4},
				{Position: 11, Value: 4}, {Position: 12, Value: 8},
			}},
			wantTarget: []BendPoint{{Position: 0, Value: 0}, {Position: 12, Value: 8}},
			wantCodes:  normalized,
		},
		{
			name: "nonzero-start fall early hold 1-4",
			effect: BendEffect{Points: []BendPoint{
				{Position: 0, Value: 4}, {Position: 1, Value: 0},
				{Position: 4, Value: 0}, {Position: 12, Value: -4},
			}},
			wantTarget: []BendPoint{{Position: 0, Value: 4}, {Position: 12, Value: -4}},
			wantCodes:  normalized,
		},
		{
			name: "nonzero-start rise early hold 1-4",
			effect: BendEffect{Points: []BendPoint{
				{Position: 0, Value: -4}, {Position: 1, Value: 0},
				{Position: 4, Value: 0}, {Position: 12, Value: 4},
			}},
			wantTarget: []BendPoint{{Position: 0, Value: -4}, {Position: 12, Value: 4}},
			wantCodes:  normalized,
		},
		{
			name: "nonzero-start fall middle hold 3-9",
			effect: BendEffect{Points: []BendPoint{
				{Position: 0, Value: 4}, {Position: 3, Value: 0},
				{Position: 9, Value: 0}, {Position: 12, Value: -4},
			}},
			wantTarget: []BendPoint{{Position: 0, Value: 4}, {Position: 12, Value: -4}},
			wantCodes:  normalized,
		},
		{
			name: "nonzero-start rise middle hold 3-9",
			effect: BendEffect{Points: []BendPoint{
				{Position: 0, Value: -4}, {Position: 3, Value: 0},
				{Position: 9, Value: 0}, {Position: 12, Value: 4},
			}},
			wantTarget: []BendPoint{{Position: 0, Value: -4}, {Position: 12, Value: 4}},
			wantCodes:  normalized,
		},
		{
			name: "nonzero-start fall late hold 8-11",
			effect: BendEffect{Points: []BendPoint{
				{Position: 0, Value: 4}, {Position: 8, Value: 0},
				{Position: 11, Value: 0}, {Position: 12, Value: -4},
			}},
			wantTarget: []BendPoint{{Position: 0, Value: 4}, {Position: 12, Value: -4}},
			wantCodes:  normalized,
		},
		{
			name: "nonzero-start rise late hold 8-11",
			effect: BendEffect{Points: []BendPoint{
				{Position: 0, Value: -4}, {Position: 8, Value: 0},
				{Position: 11, Value: 0}, {Position: 12, Value: 4},
			}},
			wantTarget: []BendPoint{{Position: 0, Value: -4}, {Position: 12, Value: 4}},
			wantCodes:  normalized,
		},
		{
			name:       "three-point noncollinear rise at non-midpoint",
			effect:     BendEffect{Points: []BendPoint{{Position: 0}, {Position: 3, Value: 4}, {Position: 12, Value: 8}}},
			wantTarget: []BendPoint{{Position: 0}, {Position: 12, Value: 8}},
			wantCodes:  normalized,
		},
		{
			name:       "three-point noncollinear fall at non-midpoint",
			effect:     BendEffect{Points: []BendPoint{{Position: 0}, {Position: 3, Value: -4}, {Position: 12, Value: -8}}},
			wantTarget: []BendPoint{{Position: 0}, {Position: 12, Value: -8}},
			wantCodes:  normalized,
		},
		{
			name:       "three-point collinear rise control",
			effect:     BendEffect{Points: []BendPoint{{Position: 0}, {Position: 3, Value: 2}, {Position: 12, Value: 8}}},
			wantTarget: []BendPoint{{Position: 0}, {Position: 12, Value: 8}},
		},
		{
			name:       "three-point collinear fall control",
			effect:     BendEffect{Points: []BendPoint{{Position: 0}, {Position: 3, Value: -2}, {Position: 12, Value: -8}}},
			wantTarget: []BendPoint{{Position: 0}, {Position: 12, Value: -8}},
		},
		{
			name:       "two-point monotonic rise control",
			effect:     BendEffect{Points: []BendPoint{{Position: 0, Value: -4}, {Position: 12, Value: 4}}},
			wantTarget: []BendPoint{{Position: 0, Value: -4}, {Position: 12, Value: 4}},
		},
		{
			name:       "two-point monotonic fall control",
			effect:     BendEffect{Points: []BendPoint{{Position: 0, Value: 4}, {Position: 12, Value: -4}}},
			wantTarget: []BendPoint{{Position: 0, Value: 4}, {Position: 12, Value: -4}},
		},
		{
			name:       "negative dip control",
			effect:     BendEffect{Points: []BendPoint{{Position: 0}, {Position: 6, Value: -4}, {Position: 12}}},
			wantTarget: []BendPoint{{Position: 0}, {Position: 6, Value: -4}, {Position: 12}},
		},
		{
			name:       "positive peak control",
			effect:     BendEffect{Points: []BendPoint{{Position: 0}, {Position: 6, Value: 4}, {Position: 12}}},
			wantTarget: []BendPoint{{Position: 0}, {Position: 6, Value: 4}, {Position: 12}},
		},
		{
			name: "asymmetric trough control",
			effect: BendEffect{Points: []BendPoint{
				{Position: 0, Value: -4}, {Position: 3, Value: -12},
				{Position: 6, Value: -12}, {Position: 9, Value: 0},
			}},
			wantTarget: []BendPoint{
				{Position: 0, Value: -4}, {Position: 3, Value: -12},
				{Position: 6, Value: -12}, {Position: 9, Value: 0},
			},
		},
		{
			name: "asymmetric peak control",
			effect: BendEffect{Points: []BendPoint{
				{Position: 0, Value: 4}, {Position: 3, Value: 12},
				{Position: 6, Value: 12}, {Position: 9, Value: 0},
			}},
			wantTarget: []BendPoint{
				{Position: 0, Value: 4}, {Position: 3, Value: 12},
				{Position: 6, Value: 12}, {Position: 9, Value: 0},
			},
		},
		{
			name:       "initial hold control",
			effect:     BendEffect{Points: []BendPoint{{Position: 0, Value: -4}, {Position: 9, Value: -4}, {Position: 12}}},
			wantTarget: []BendPoint{{Position: 0, Value: -4}, {Position: 9, Value: -4}, {Position: 9, Value: -4}, {Position: 12}},
		},
		{
			name:       "final hold control",
			effect:     BendEffect{Points: []BendPoint{{Position: 0}, {Position: 9, Value: -4}, {Position: 12, Value: -4}}},
			wantTarget: []BendPoint{{Position: 0}, {Position: 9, Value: -4}, {Position: 9, Value: -4}, {Position: 12, Value: -4}},
		},
		{
			name: "combined unequal middles summary and vibrato",
			effect: BendEffect{
				Kind: BendTypeDive, Value: 200,
				Points: []BendPoint{
					{Position: 0}, {Position: 3, Value: -4, Vibrato: true},
					{Position: 9, Value: -6}, {Position: 12, Value: -8},
				},
			},
			wantTarget: []BendPoint{{Position: 0}, {Position: 12, Value: -8}},
			wantCodes: []string{
				whammyExportWhammyCurveCode,
				"gp8.omit.whammy-summary",
				"gp8.omit.whammy-point-vibrato",
			},
		},
	}
}

func whammyExportAssertExportBehavior(t *testing.T, effect *BendEffect, wantTarget []BendPoint, wantCodes []string) []byte {
	t.Helper()
	song := conformanceCurveSong(t)
	conformanceCurveSetCurve(song, "whammy", effect)
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("valid source diagnostics = %#v", diagnostics)
	}
	songBefore := conformanceContractSnapshot(song, false)

	preflightOptions := ExportOptions{}
	preflightOptionsBefore := conformanceContractSnapshot(preflightOptions, false)
	preflight := PreflightExport(song, ExportFormatGP8, preflightOptions)
	whammyExportAssertReport(t, preflight, wantCodes)
	assertWhammyConformanceReadOnly(t, song, songBefore, preflightOptions, preflightOptionsBefore)

	if len(wantCodes) != 0 {
		refusedAllowlists := [][]string{nil, {"unrelated"}}
		if len(wantCodes) > 1 && slices.Contains(wantCodes, whammyExportWhammyCurveCode) {
			refusedAllowlists = append(refusedAllowlists, []string{whammyExportWhammyCurveCode})
		}
		for _, allowed := range refusedAllowlists {
			options := ExportOptions{LossPolicy: ExportLossPolicy{
				RequirePreservation: true,
				AllowedCodes:        slices.Clone(allowed),
			}}
			optionsBefore := conformanceContractSnapshot(options, false)
			data, report, err := ExportWithReport(song, ExportFormatGP8, options)
			var lossErr *ExportLossError
			if len(data) != 0 || !errors.As(err, &lossErr) || !reflect.DeepEqual(report, preflight) {
				t.Fatalf("strict export with allowlist %v = %d bytes, report %#v, error %v; want zero bytes, loss error, and preflight report", allowed, len(data), report.Entries, err)
			}
			wantRefused := slices.DeleteFunc(slices.Clone(wantCodes), func(code string) bool {
				return slices.Contains(allowed, code)
			})
			gotRefused := make([]string, len(lossErr.Entries))
			for index, entry := range lossErr.Entries {
				gotRefused[index] = entry.Code
			}
			if !slices.Equal(gotRefused, wantRefused) {
				t.Fatalf("refused codes with allowlist %v = %v, want %v", allowed, gotRefused, wantRefused)
			}
			assertWhammyConformanceReadOnly(t, song, songBefore, options, optionsBefore)
		}
	}

	allowedOptions := ExportOptions{LossPolicy: ExportLossPolicy{
		RequirePreservation: true,
		AllowedCodes:        slices.Clone(wantCodes),
	}}
	allowedOptionsBefore := conformanceContractSnapshot(allowedOptions, false)
	allowedData, allowedReport, allowedErr := ExportWithReport(song, ExportFormatGP8, allowedOptions)
	if allowedErr != nil || len(allowedData) == 0 || !reflect.DeepEqual(allowedReport, preflight) {
		t.Fatalf("exactly allowed export = %d bytes, report %#v, error %v; want bytes and preflight report", len(allowedData), allowedReport.Entries, allowedErr)
	}
	assertWhammyConformanceReadOnly(t, song, songBefore, allowedOptions, allowedOptionsBefore)
	if got := whammyExportRoundTripWhammy(t, allowedData); !slices.Equal(got, wantTarget) {
		t.Fatalf("allowed interpreted target points = %#v, want %#v", got, wantTarget)
	}

	permissiveOptions := ExportOptions{}
	permissiveOptionsBefore := conformanceContractSnapshot(permissiveOptions, false)
	permissiveData, permissiveReport, permissiveErr := ExportWithReport(song, ExportFormatGP8, permissiveOptions)
	if permissiveErr != nil || len(permissiveData) == 0 || !reflect.DeepEqual(permissiveReport, preflight) {
		t.Fatalf("permissive export = %d bytes, report %#v, error %v; want bytes and preflight report", len(permissiveData), permissiveReport.Entries, permissiveErr)
	}
	assertWhammyConformanceReadOnly(t, song, songBefore, permissiveOptions, permissiveOptionsBefore)
	if got := whammyExportRoundTripWhammy(t, permissiveData); !slices.Equal(got, wantTarget) {
		t.Fatalf("permissive interpreted target points = %#v, want %#v", got, wantTarget)
	}
	return permissiveData
}

func whammyExportAssertReport(t *testing.T, report ExportReport, wantCodes []string) {
	t.Helper()
	if report.Target != ExportFormatGP8 || !slices.Equal(reportCodes(report), wantCodes) {
		t.Fatalf("preflight = target %d entries %#v, want target GP8 and codes %v", report.Target, report.Entries, wantCodes)
	}
	wantLocation := ScoreLocation{Track: 0, Staff: 0, Measure: 0, Voice: 0, Beat: 0}
	for _, entry := range report.Entries {
		wantDisposition := ExportDispositionOmitted
		if entry.Code == whammyExportWhammyCurveCode {
			wantDisposition = ExportDispositionNormalized
		}
		if entry.Feature != "note-and-beat-semantics" || entry.Disposition != wantDisposition || entry.Location != wantLocation {
			t.Fatalf("entry %s = %#v, want %s at %#v", entry.Code, entry, wantDisposition, wantLocation)
		}
	}
}

func whammyExportRoundTripWhammy(t *testing.T, data []byte) []BendPoint {
	t.Helper()
	song, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	whammy := song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Effect.TremoloBar
	if whammy == nil {
		t.Fatal("round-trip whammy is nil")
	}
	return whammy.Points
}

func whammyExportAlphaTabWhammy(t *testing.T, data []byte) []BendPoint {
	t.Helper()
	score := readAlphaTabScore(t, writeConformanceFixture(t, data))
	return whammyConformanceBendPoints(whammyConformanceNormalizedBeat(t, score, 0)["whammy"])
}

func whammyExportReadBinaryWhammy(t *testing.T, points []BendPoint) *BendEffect {
	t.Helper()
	var raw bytes.Buffer
	raw.WriteByte(byte(BendTypeNone))
	whammyExportWriteInt32(t, &raw, 0)
	whammyExportWriteInt32(t, &raw, int32(len(points)))
	for _, point := range points {
		whammyExportWriteInt32(t, &raw, int32(point.Position)*5)
		whammyExportWriteInt32(t, &raw, int32(point.Value)*int32(GPBendSemitone))
		if point.Vibrato {
			raw.WriteByte(1)
		} else {
			raw.WriteByte(0)
		}
	}
	effect, err := (&Song{}).readBendEffect(newCursor(raw.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if effect == nil {
		t.Fatal("binary whammy is nil")
	}
	return effect
}

func whammyExportWriteInt32(t *testing.T, output *bytes.Buffer, value int32) {
	t.Helper()
	if err := binary.Write(output, binary.LittleEndian, value); err != nil {
		t.Fatal(err)
	}
}

type whammyExportParsedFixtureCase struct {
	name       string
	owner      *Song
	effect     *BendEffect
	wantTarget []BendPoint
	wantCodes  []string
}

func whammyExportParsedFixtureCases(t *testing.T) []whammyExportParsedFixtureCase {
	t.Helper()
	var tests []whammyExportParsedFixtureCase
	for _, fixture := range []string{"testdata/gp4/tremolo.gp4", "testdata/gp5/tremolo.gp5"} {
		song := parseTestFixture(t, fixture)
		wantTargets := [][]BendPoint{
			{{Position: 0}, {Position: 6, Value: -4}, {Position: 12}},
			{{Position: 0, Value: -4}, {Position: 9, Value: -4}, {Position: 9, Value: -4}, {Position: 12}},
			{{Position: 0}, {Position: 9, Value: -4}, {Position: 9, Value: -4}, {Position: 12, Value: -4}},
		}
		for measure, want := range wantTargets {
			effect := song.Tracks[0].Staves[0].Measures[measure].Voices[0].Beats[0].Effect.TremoloBar
			if effect == nil {
				t.Fatalf("%s measure %d parsed whammy is nil", fixture, measure)
			}
			tests = append(tests, whammyExportParsedFixtureCase{
				name: fixture + "/measure-" + string(rune('0'+measure)), owner: song,
				effect: effect, wantTarget: want, wantCodes: []string{"gp8.omit.whammy-summary"},
			})
		}
	}
	return tests
}
