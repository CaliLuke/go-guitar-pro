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

const ticket37WhammyCurveCode = "gp8.normalize.whammy-curve"

type ticket37WhammyCase struct {
	name       string
	effect     BendEffect
	wantTarget []BendPoint
	wantCodes  []string
}

func TestGP8WhammyMiddleHoldReportsInterpretedLoss(t *testing.T) {
	for _, test := range ticket37WhammyCases() {
		t.Run(test.name, func(t *testing.T) {
			ticket37AssertExportBehavior(t, &test.effect, test.wantTarget, test.wantCodes)
		})
	}
}

func TestAlphaTabGP8WhammyTargetInterpretation(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, test := range ticket37WhammyCases() {
		t.Run("programmatic/"+test.name, func(t *testing.T) {
			data := ticket37AssertExportBehavior(t, &test.effect, test.wantTarget, test.wantCodes)
			if got := ticket37AlphaTabWhammy(t, data); !slices.Equal(got, test.wantTarget) {
				t.Fatalf("AlphaTab interpreted target points = %#v, want %#v", got, test.wantTarget)
			}
		})
	}

	for _, test := range ticket37ParsedFixtureCases(t) {
		t.Run("parsed-binary/"+test.name, func(t *testing.T) {
			data := ticket37AssertExportBehavior(t, test.effect, test.wantTarget, test.wantCodes)
			if got := ticket37AlphaTabWhammy(t, data); !slices.Equal(got, test.wantTarget) {
				t.Fatalf("AlphaTab interpreted parsed-binary target points = %#v, want %#v", got, test.wantTarget)
			}
		})
	}
}

func TestGP8WhammyParsedBinaryPaths(t *testing.T) {
	for _, test := range ticket37WhammyCases()[2:4] {
		t.Run("raw-record/"+test.name, func(t *testing.T) {
			effect := ticket37ReadBinaryWhammy(t, test.effect.Points)
			if !reflect.DeepEqual(effect, &test.effect) {
				t.Fatalf("binary whammy = %#v, want %#v", effect, &test.effect)
			}
			ticket37AssertExportBehavior(t, effect, test.wantTarget, test.wantCodes)
		})
	}

	for _, test := range ticket37ParsedFixtureCases(t) {
		t.Run("public-fixture/"+test.name, func(t *testing.T) {
			before := m21Snapshot(test.owner, false)
			ticket37AssertExportBehavior(t, test.effect, test.wantTarget, test.wantCodes)
			if got := m21Snapshot(test.owner, false); got != before {
				t.Fatal("export mutated the parsed binary source score")
			}
		})
	}
}

func ticket37WhammyCases() []ticket37WhammyCase {
	normalized := []string{ticket37WhammyCurveCode}
	return []ticket37WhammyCase{
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
				ticket37WhammyCurveCode,
				"gp8.omit.whammy-summary",
				"gp8.omit.whammy-point-vibrato",
			},
		},
	}
}

func ticket37AssertExportBehavior(t *testing.T, effect *BendEffect, wantTarget []BendPoint, wantCodes []string) []byte {
	t.Helper()
	song := m12Song(t)
	m12SetCurve(song, "whammy", effect)
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("valid source diagnostics = %#v", diagnostics)
	}
	songBefore := m21Snapshot(song, false)

	preflightOptions := ExportOptions{}
	preflightOptionsBefore := m21Snapshot(preflightOptions, false)
	preflight := PreflightExport(song, ExportFormatGP8, preflightOptions)
	ticket37AssertReport(t, preflight, wantCodes)
	assertTicket35ReadOnly(t, song, songBefore, preflightOptions, preflightOptionsBefore)

	if len(wantCodes) != 0 {
		refusedAllowlists := [][]string{nil, {"unrelated"}}
		if len(wantCodes) > 1 && slices.Contains(wantCodes, ticket37WhammyCurveCode) {
			refusedAllowlists = append(refusedAllowlists, []string{ticket37WhammyCurveCode})
		}
		for _, allowed := range refusedAllowlists {
			options := ExportOptions{LossPolicy: ExportLossPolicy{
				RequirePreservation: true,
				AllowedCodes:        slices.Clone(allowed),
			}}
			optionsBefore := m21Snapshot(options, false)
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
			assertTicket35ReadOnly(t, song, songBefore, options, optionsBefore)
		}
	}

	allowedOptions := ExportOptions{LossPolicy: ExportLossPolicy{
		RequirePreservation: true,
		AllowedCodes:        slices.Clone(wantCodes),
	}}
	allowedOptionsBefore := m21Snapshot(allowedOptions, false)
	allowedData, allowedReport, allowedErr := ExportWithReport(song, ExportFormatGP8, allowedOptions)
	if allowedErr != nil || len(allowedData) == 0 || !reflect.DeepEqual(allowedReport, preflight) {
		t.Fatalf("exactly allowed export = %d bytes, report %#v, error %v; want bytes and preflight report", len(allowedData), allowedReport.Entries, allowedErr)
	}
	assertTicket35ReadOnly(t, song, songBefore, allowedOptions, allowedOptionsBefore)
	if got := ticket37RoundTripWhammy(t, allowedData); !slices.Equal(got, wantTarget) {
		t.Fatalf("allowed interpreted target points = %#v, want %#v", got, wantTarget)
	}

	permissiveOptions := ExportOptions{}
	permissiveOptionsBefore := m21Snapshot(permissiveOptions, false)
	permissiveData, permissiveReport, permissiveErr := ExportWithReport(song, ExportFormatGP8, permissiveOptions)
	if permissiveErr != nil || len(permissiveData) == 0 || !reflect.DeepEqual(permissiveReport, preflight) {
		t.Fatalf("permissive export = %d bytes, report %#v, error %v; want bytes and preflight report", len(permissiveData), permissiveReport.Entries, permissiveErr)
	}
	assertTicket35ReadOnly(t, song, songBefore, permissiveOptions, permissiveOptionsBefore)
	if got := ticket37RoundTripWhammy(t, permissiveData); !slices.Equal(got, wantTarget) {
		t.Fatalf("permissive interpreted target points = %#v, want %#v", got, wantTarget)
	}
	return permissiveData
}

func ticket37AssertReport(t *testing.T, report ExportReport, wantCodes []string) {
	t.Helper()
	if report.Target != ExportFormatGP8 || !slices.Equal(reportCodes(report), wantCodes) {
		t.Fatalf("preflight = target %d entries %#v, want target GP8 and codes %v", report.Target, report.Entries, wantCodes)
	}
	wantLocation := ScoreLocation{Track: 0, Staff: 0, Measure: 0, Voice: 0, Beat: 0}
	for _, entry := range report.Entries {
		wantDisposition := ExportDispositionOmitted
		if entry.Code == ticket37WhammyCurveCode {
			wantDisposition = ExportDispositionNormalized
		}
		if entry.Feature != "note-and-beat-semantics" || entry.Disposition != wantDisposition || entry.Location != wantLocation {
			t.Fatalf("entry %s = %#v, want %s at %#v", entry.Code, entry, wantDisposition, wantLocation)
		}
	}
}

func ticket37RoundTripWhammy(t *testing.T, data []byte) []BendPoint {
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

func ticket37AlphaTabWhammy(t *testing.T, data []byte) []BendPoint {
	t.Helper()
	score := readAlphaTabScore(t, writeConformanceFixture(t, data))
	return ticket35BendPoints(ticket35NormalizedBeat(t, score, 0)["whammy"])
}

func ticket37ReadBinaryWhammy(t *testing.T, points []BendPoint) *BendEffect {
	t.Helper()
	var raw bytes.Buffer
	raw.WriteByte(byte(BendTypeNone))
	ticket37WriteInt32(t, &raw, 0)
	ticket37WriteInt32(t, &raw, int32(len(points)))
	for _, point := range points {
		ticket37WriteInt32(t, &raw, int32(point.Position)*5)
		ticket37WriteInt32(t, &raw, int32(point.Value)*int32(GPBendSemitone))
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

func ticket37WriteInt32(t *testing.T, output *bytes.Buffer, value int32) {
	t.Helper()
	if err := binary.Write(output, binary.LittleEndian, value); err != nil {
		t.Fatal(err)
	}
}

type ticket37ParsedFixtureCase struct {
	name       string
	owner      *Song
	effect     *BendEffect
	wantTarget []BendPoint
	wantCodes  []string
}

func ticket37ParsedFixtureCases(t *testing.T) []ticket37ParsedFixtureCase {
	t.Helper()
	var tests []ticket37ParsedFixtureCase
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
			tests = append(tests, ticket37ParsedFixtureCase{
				name: fixture + "/measure-" + string(rune('0'+measure)), owner: song,
				effect: effect, wantTarget: want, wantCodes: []string{"gp8.omit.whammy-summary"},
			})
		}
	}
	return tests
}
