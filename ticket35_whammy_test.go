// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/binary"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestParseBinaryWhammyPreservesDipsAndHolds(t *testing.T) {
	fixtures, want := ticket35BinaryWhammyCases()
	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			song := parseTestFixture(t, fixture)
			barOffset := 0
			if strings.Contains(fixture, "Effects") {
				barOffset = 8
			}
			for index, expected := range want {
				beat := song.Tracks[0].Staves[0].Measures[barOffset+index].Voices[0].Beats[0]
				if beat.Effect.TremoloBar == nil {
					t.Fatalf("bar %d whammy is nil", barOffset+index)
				}
				if !reflect.DeepEqual(beat.Effect.TremoloBar.Points, expected) {
					t.Fatalf("bar %d whammy points = %#v, want %#v", barOffset+index, beat.Effect.TremoloBar.Points, expected)
				}
			}
		})
	}
}

func TestAlphaTabBinaryWhammyProjection(t *testing.T) {
	requireAlphaTabConformance(t)
	fixtures, want := ticket35BinaryWhammyCases()
	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			score := readAlphaTabScore(t, fixture)
			barOffset := 0
			if strings.Contains(fixture, "Effects") {
				barOffset = 8
			}
			for index, expected := range want {
				beat := ticket35NormalizedBeat(t, score, barOffset+index)
				if !reflect.DeepEqual(ticket35BendPoints(beat["whammy"]), expected) {
					t.Fatalf("bar %d AlphaTab whammy = %#v, want %#v", barOffset+index, beat["whammy"], expected)
				}
			}
		})
	}
}

func ticket35BinaryWhammyCases() ([]string, [][]BendPoint) {
	return []string{
		"testdata/gp4/tremolo.gp4",
		"testdata/gp5/tremolo.gp5",
		"testdata/gp4/Effects.gp4",
		"testdata/gp5/Effects.gp5",
	}, [][]BendPoint{
		{{Position: 0, Value: 0}, {Position: 6, Value: -4}, {Position: 12, Value: 0}},
		{{Position: 0, Value: -4}, {Position: 9, Value: -4}, {Position: 12, Value: 0}},
		{{Position: 0, Value: 0}, {Position: 9, Value: -4}, {Position: 12, Value: -4}},
	}
}

func TestParseGP3TremoloUsesItsSeparateEncoding(t *testing.T) {
	raw := make([]byte, 4)
	binary.LittleEndian.PutUint32(raw, 100)
	whammy, err := (&Song{}).readTremoloBar(newCursor(raw))
	if err != nil {
		t.Fatal(err)
	}
	want := []BendPoint{{Position: 0}, {Position: 6, Value: -4}, {Position: 12}}
	if !slices.Equal(whammy.Points, want) {
		t.Fatalf("GP3 tremolo points = %#v, want %#v", whammy.Points, want)
	}

	song := parseTestFixture(t, "testdata/gp3/Effects.gp3")
	for bar := 8; bar <= 11; bar++ {
		beatWhammy := song.Tracks[0].Staves[0].Measures[bar].Voices[0].Beats[0].Effect.TremoloBar
		if beatWhammy == nil || !slices.Equal(beatWhammy.Points, want) {
			t.Fatalf("public GP3 bar %d whammy = %#v, want %#v", bar, beatWhammy, want)
		}
	}
}

func TestAlphaTabGP3WhammyOracleLimitation(t *testing.T) {
	requireAlphaTabConformance(t)
	goSong := parseTestFixture(t, "testdata/gp3/Effects.gp3")
	alphaScore := readAlphaTabScore(t, "testdata/gp3/Effects.gp3")
	want := []BendPoint{{Position: 0}, {Position: 6, Value: -4}, {Position: 12}}
	for bar := 8; bar <= 11; bar++ {
		goWhammy := goSong.Tracks[0].Staves[0].Measures[bar].Voices[0].Beats[0].Effect.TremoloBar
		if goWhammy == nil || !slices.Equal(goWhammy.Points, want) {
			t.Fatalf("Go GP3 bar %d whammy = %#v, want %#v", bar, goWhammy, want)
		}
		if alphaWhammy := ticket35NormalizedBeat(t, alphaScore, bar)["whammy"]; alphaWhammy != nil {
			t.Fatalf("pinned AlphaTab GP3 limitation changed at bar %d: whammy = %#v", bar, alphaWhammy)
		}
	}
}

func TestParseGPIFWhammyProperties(t *testing.T) {
	want := [][]BendPoint{
		{{Position: 0, Value: 0}, {Position: 6, Value: -4}, {Position: 12, Value: 0}},
		{{Position: 0, Value: -4}, {Position: 12, Value: 0}},
		{{Position: 0, Value: 0}, {Position: 6, Value: -4}, {Position: 12, Value: -4}},
		{{Position: 0, Value: -4}, {Position: 3, Value: -12}, {Position: 6, Value: -12}, {Position: 9, Value: 0}},
	}
	song := parseTestFixture(t, "testdata/gp6/tremolo.gpx")
	for bar, expected := range want {
		beat := song.Tracks[0].Staves[0].Measures[bar].Voices[0].Beats[0]
		if beat.Effect.TremoloBar == nil {
			t.Fatalf("bar %d whammy is nil", bar)
		}
		if !reflect.DeepEqual(beat.Effect.TremoloBar.Points, expected) {
			t.Fatalf("bar %d whammy points = %#v, want %#v", bar, beat.Effect.TremoloBar.Points, expected)
		}
	}
	result, err := ParseFileWithOptions("testdata/gp6/tremolo.gpx", ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range result.Diagnostics {
		if strings.Contains(diagnostic.SourcePath, "WhammyBar") && diagnostic.Kind == ParseDiagnosticUnsupportedFeature {
			t.Fatalf("valid GP6 whammy property diagnosed as unsupported: %#v", diagnostic)
		}
	}
}

func TestParseGPIFWhammyPropertiesPreservesExplicitZeroMiddle(t *testing.T) {
	properties := `<Properties>` +
		`<Property name="WhammyBar"><Enable/></Property>` +
		`<Property name="WhammyBarOriginValue"><Float>100</Float></Property>` +
		`<Property name="WhammyBarMiddleValue"><Float>0</Float></Property>` +
		`<Property name="WhammyBarMiddleOffset1"><Float>50</Float></Property>` +
		`<Property name="WhammyBarDestinationValue"><Float>100</Float></Property>` +
		`</Properties>`
	source := strings.Replace(staffScopedChordGPIF,
		`<Beat id="0"><Rhythm ref="0"/><Chord>0</Chord></Beat>`,
		`<Beat id="0"><Rhythm ref="0"/>`+properties+`</Beat>`, 1)
	result, err := ParseWithOptions(conformanceGPIFArchive(t, source), ParseOptions{Strict: true})
	if err != nil {
		t.Fatalf("strict GPIF whammy parse: diagnostics = %#v, error = %v", result.Diagnostics, err)
	}
	points := result.Song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Effect.TremoloBar.Points
	want := []BendPoint{{Position: 0, Value: 4}, {Position: 6, Value: 0}, {Position: 12, Value: 4}}
	if !reflect.DeepEqual(points, want) {
		t.Fatalf("zero-middle whammy points = %#v, want %#v", points, want)
	}

	malformed := strings.Replace(source,
		`<Property name="WhammyBarMiddleValue"><Float>0</Float></Property>`,
		`<Property name="WhammyBarMiddleValue"><Float>NaN</Float></Property>`, 1)
	malformedResult, strictErr := ParseWithOptions(conformanceGPIFArchive(t, malformed), ParseOptions{Strict: true})
	var rejected *StrictParseError
	if !errors.As(strictErr, &rejected) || !slices.ContainsFunc(malformedResult.Diagnostics, func(diagnostic ParseDiagnostic) bool {
		return diagnostic.Code == "GPIF.Beat.Whammy.Invalid" && strings.Contains(diagnostic.SourcePath, "WhammyBarMiddleValue")
	}) {
		t.Fatalf("malformed named whammy = diagnostics %#v, error %v; want strict GPIF.Beat.Whammy.Invalid", malformedResult.Diagnostics, strictErr)
	}
}

func TestAlphaTabWhammyCorpusConformance(t *testing.T) {
	requireAlphaTabConformance(t)
	fixtures := []string{
		"testdata/gp4/Effects.gp4",
		"testdata/gp4/tremolo.gp4",
		"testdata/gp5/Effects.gp5",
		"testdata/gp5/tremolo.gp5",
		"testdata/gp6/tremolo.gpx",
		"testdata/gp7/tremolo-bar.gp",
		"testdata/gp7/tremolo-vibrato.gp",
		"testdata/gp7/tremolo.gp",
		"testdata/gp7/whammy-advanced.gp",
	}
	alphaScores, err := readM23OracleScores(fixtures)
	if err != nil {
		t.Fatal(err)
	}
	var differences []string
	for _, fixture := range fixtures {
		goFacts := collectConformanceFacts(normalizeGoScore(parseTestFixture(t, fixture)), map[string]bool{"whammy": true}, nonNilConformanceFact)
		alphaFacts := collectConformanceFacts(alphaScores[fixture], map[string]bool{"whammy": true}, nonNilConformanceFact)
		fixtureDifferences := semanticDifferences(goFacts, alphaFacts)
		if len(fixtureDifferences) != 0 {
			t.Logf("%s Go whammy facts = %#v; AlphaTab = %#v", fixture, goFacts, alphaFacts)
		}
		for _, difference := range fixtureDifferences {
			differences = append(differences, fixture+":"+difference.Path)
		}
	}
	if len(differences) != 0 {
		t.Fatalf("found %d whammy projection differences:\n%s", len(differences), strings.Join(differences, "\n"))
	}
}

func TestGP8WhammyExportPreservesSupportedCurves(t *testing.T) {
	curves := map[string][]BendPoint{
		"negative dip":      {{Position: 0}, {Position: 6, Value: -4}, {Position: 12}},
		"positive peak":     {{Position: 0}, {Position: 6, Value: 4}, {Position: 12}},
		"monotonic rise":    {{Position: 0, Value: -4}, {Position: 12, Value: 4}},
		"monotonic fall":    {{Position: 0, Value: 4}, {Position: 12, Value: -4}},
		"initial hold":      {{Position: 0, Value: -4}, {Position: 9, Value: -4}, {Position: 12}},
		"final hold":        {{Position: 0}, {Position: 9, Value: -4}, {Position: 12, Value: -4}},
		"asymmetric trough": {{Position: 0, Value: -4}, {Position: 3, Value: -12}, {Position: 6, Value: -12}, {Position: 9}},
	}
	for name, points := range curves {
		t.Run(name, func(t *testing.T) {
			song := m12Song(t)
			m12SetCurve(song, "whammy", &BendEffect{Points: slices.Clone(points)})
			options := ExportOptions{
				LossPolicy: ExportLossPolicy{RequirePreservation: true},
			}
			songBefore := m21Snapshot(song, false)
			optionsBefore := m21Snapshot(options, false)
			data, report, err := ExportWithReport(song, ExportFormatGP8, options)
			if err != nil || len(data) == 0 {
				t.Fatalf("strict GP8 export = %d bytes, %#v, %v", len(data), report.Entries, err)
			}
			assertTicket35ReadOnly(t, song, songBefore, options, optionsBefore)
			roundTrip, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			got := roundTrip.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Effect.TremoloBar
			if got == nil || !slices.Equal(simplifyBendPoints(got.Points), simplifyBendPoints(points)) {
				t.Fatalf("GP8 round-trip whammy = %#v, want %#v", got, points)
			}
		})
	}
}

func TestAlphaTabGP8WhammyExportProjection(t *testing.T) {
	requireAlphaTabConformance(t)
	source := parseTestFixture(t, "testdata/gp4/tremolo.gp4")
	cases := []struct {
		name   string
		points []BendPoint
	}{
		{name: "programmatic asymmetric", points: []BendPoint{{Position: 0, Value: -4}, {Position: 3, Value: -12}, {Position: 6, Value: -12}, {Position: 9}}},
		{name: "programmatic positive peak", points: []BendPoint{{Position: 0}, {Position: 6, Value: 4}, {Position: 12}}},
		{name: "programmatic monotonic rise", points: []BendPoint{{Position: 0, Value: -4}, {Position: 12, Value: 4}}},
		{name: "programmatic monotonic fall", points: []BendPoint{{Position: 0, Value: 4}, {Position: 12, Value: -4}}},
	}
	for bar := 0; bar < 3; bar++ {
		cases = append(cases, struct {
			name   string
			points []BendPoint
		}{name: "parsed binary " + string(rune('0'+bar)), points: source.Tracks[0].Staves[0].Measures[bar].Voices[0].Beats[0].Effect.TremoloBar.Points})
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			points := test.points
			song := m12Song(t)
			m12SetCurve(song, "whammy", &BendEffect{Points: slices.Clone(points)})
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{
				LossPolicy: ExportLossPolicy{RequirePreservation: true},
			})
			if err != nil {
				t.Fatalf("strict GP8 export: %#v, %v", report.Entries, err)
			}
			score := readAlphaTabScore(t, writeConformanceFixture(t, data))
			got := simplifyBendPoints(ticket35BendPoints(ticket35NormalizedBeat(t, score, 0)["whammy"]))
			if !slices.Equal(got, simplifyBendPoints(points)) {
				t.Fatalf("AlphaTab exported whammy = %#v, want %#v", got, points)
			}
		})
	}
}

func TestWhammyProjectionAdaptersExposeBeatCurves(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp4/tremolo.gp4")
	beat := ticket35NormalizedBeat(t, normalizeGoScore(song), 0)
	want := []BendPoint{{Position: 0}, {Position: 6, Value: -4}, {Position: 12}}
	if got := ticket35BendPoints(beat["whammy"]); !slices.Equal(got, want) {
		t.Fatalf("Go corpus whammy projection = %#v, want %#v", got, want)
	}
}

func TestGP8WhammyExportLossPolicyIsExact(t *testing.T) {
	cases := []struct {
		name   string
		points []BendPoint
		codes  []string
	}{
		{name: "five points", points: []BendPoint{{Position: 0}, {Position: 3, Value: -4}, {Position: 6}, {Position: 9, Value: -4}, {Position: 12}}, codes: []string{"gp8.omit.whammy-curve"}},
		{name: "middle point vibrato", points: []BendPoint{{Position: 0}, {Position: 6, Value: -4, Vibrato: true}, {Position: 12}}, codes: []string{"gp8.normalize.whammy-curve", "gp8.omit.whammy-point-vibrato"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			song := m12Song(t)
			m12SetCurve(song, "whammy", &BendEffect{Points: slices.Clone(test.points)})
			songBefore := m21Snapshot(song, false)
			preflight := PreflightExport(song, ExportFormatGP8, ExportOptions{})
			if got := reportCodes(preflight); !slices.Equal(got, test.codes) {
				t.Fatalf("preflight codes = %v, want %v", got, test.codes)
			}
			for _, allowed := range [][]string{nil, {"unrelated"}} {
				options := ExportOptions{LossPolicy: ExportLossPolicy{
					RequirePreservation: true, AllowedCodes: allowed,
				}}
				optionsBefore := m21Snapshot(options, false)
				data, report, err := ExportWithReport(song, ExportFormatGP8, options)
				var lossErr *ExportLossError
				if len(data) != 0 || !errors.As(err, &lossErr) || !reflect.DeepEqual(report, preflight) {
					t.Fatalf("strict export = %d bytes, %#v, %v; preflight %#v", len(data), report.Entries, err, preflight.Entries)
				}
				assertTicket35ReadOnly(t, song, songBefore, options, optionsBefore)
			}
			exact := ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: slices.Clone(test.codes)}}
			exactBefore := m21Snapshot(exact, false)
			data, report, err := ExportWithReport(song, ExportFormatGP8, exact)
			if err != nil || len(data) == 0 || !reflect.DeepEqual(report, preflight) {
				t.Fatalf("allowed export = %d bytes, %#v, %v; preflight %#v", len(data), report.Entries, err, preflight.Entries)
			}
			assertTicket35ReadOnly(t, song, songBefore, exact, exactBefore)
		})
	}
}

func assertTicket35ReadOnly(t *testing.T, song *Song, wantSong string, options ExportOptions, wantOptions string) {
	t.Helper()
	if m21Snapshot(song, false) != wantSong || m21Snapshot(options, false) != wantOptions {
		t.Fatal("whammy preflight/export mutated its Song or ExportOptions input")
	}
}

func ticket35NormalizedBeat(t *testing.T, score any, bar int) map[string]any {
	t.Helper()
	track := score.(map[string]any)["tracks"].([]any)[0].(map[string]any)
	staff := track["staves"].([]any)[0].(map[string]any)
	barValue := staff["bars"].([]any)[bar].(map[string]any)
	voice := barValue["voices"].([]any)[0].(map[string]any)
	return voice["beats"].([]any)[0].(map[string]any)
}

func ticket35BendPoints(value any) []BendPoint {
	items, _ := value.([]any)
	result := make([]BendPoint, 0, len(items))
	for _, item := range items {
		point := item.(map[string]any)
		result = append(result, BendPoint{Position: uint8(ticket35Integer(point["position"])), Value: int8(ticket35Integer(point["value"]))})
	}
	return result
}

func ticket35Integer(value any) int64 {
	switch value := value.(type) {
	case float64:
		return int64(value)
	case uint8:
		return int64(value)
	case int8:
		return int64(value)
	case int:
		return int64(value)
	default:
		panic("unexpected whammy projection number")
	}
}
