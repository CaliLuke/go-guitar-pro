// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

type conformanceCases struct {
	SchemaVersion int `json:"schemaVersion"`
	InputCases    []struct {
		ID       string   `json:"id"`
		Fixture  string   `json:"fixture"`
		Features []string `json:"features"`
		Issues   []int    `json:"issues"`
	} `json:"inputCases"`
	ExportCases []struct {
		ID       string   `json:"id"`
		Features []string `json:"features"`
		Issues   []int    `json:"issues"`
	} `json:"exportCases"`
}

type semanticDifference struct {
	Path     string `json:"path"`
	Feature  string `json:"feature"`
	Go       any    `json:"go,omitempty"`
	AlphaTab any    `json:"alphaTab,omitempty"`
}

type differenceSnapshot struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Case          string               `json:"case"`
	Comparison    string               `json:"comparison"`
	Differences   []semanticDifference `json:"differences"`
}

type fixtureInventory struct {
	SchemaVersion int                      `json:"schemaVersion"`
	Fixtures      []fixtureInventoryRecord `json:"fixtures"`
}

type fixtureInventoryRecord struct {
	Path             string   `json:"path"`
	SHA256           string   `json:"sha256"`
	Format           string   `json:"format"`
	Disposition      string   `json:"disposition"`
	Reason           string   `json:"reason"`
	Features         []string `json:"features"`
	SemanticSnapshot bool     `json:"semanticSnapshot"`
}

func TestAlphaTabFixtureInventory(t *testing.T) {
	requireAlphaTabConformance(t)
	data, err := os.ReadFile("conformance/fixture-inventory.json")
	if err != nil {
		t.Fatal(err)
	}
	var inventory fixtureInventory
	if unmarshalErr := json.Unmarshal(data, &inventory); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	cases := readConformanceCases(t)
	caseByFixture := make(map[string][]string, len(cases.InputCases))
	for _, test := range cases.InputCases {
		caseByFixture[test.Fixture] = test.Features
	}
	actualPaths := make([]string, 0, len(inventory.Fixtures))
	err = filepath.WalkDir("testdata", func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !isGuitarProFixture(path) {
			return nil
		}
		actualPaths = append(actualPaths, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	inventoryPaths := make([]string, 0, len(inventory.Fixtures))
	for _, fixture := range inventory.Fixtures {
		inventoryPaths = append(inventoryPaths, fixture.Path)
		digest, digestErr := conformanceFixtureDigest(fixture.Path)
		if digestErr != nil {
			t.Fatal(digestErr)
		}
		if digest != fixture.SHA256 {
			t.Fatalf("%s hash = %s, want %s; review fixture drift manually", fixture.Path, digest, fixture.SHA256)
		}
		if fixture.Format != filepath.Base(filepath.Dir(fixture.Path)) {
			t.Fatalf("%s format = %s", fixture.Path, fixture.Format)
		}
		caseFeatures, isCase := caseByFixture[fixture.Path]
		if fixture.SemanticSnapshot != isCase {
			t.Fatalf("%s semanticSnapshot = %t, case declaration = %t", fixture.Path, fixture.SemanticSnapshot, isCase)
		}
		if isCase && !slices.Equal(fixture.Features, caseFeatures) {
			t.Fatalf("%s features = %v, case features = %v", fixture.Path, fixture.Features, caseFeatures)
		}
	}
	if !slices.Equal(actualPaths, inventoryPaths) {
		t.Fatal("fixture inventory paths changed; add or remove entries with an explicit disposition, feature mapping, and reason")
	}
}

func TestAlphaTabInputConformance(t *testing.T) {
	requireAlphaTabConformance(t)
	cases := readConformanceCases(t)
	for _, test := range cases.InputCases {
		t.Run(test.ID, func(t *testing.T) {
			verifyFixtureHash(t, test.Fixture)
			song := parseTestFixture(t, test.Fixture)
			goScore := selectConformanceFeatures(normalizeGoScore(song), test.Features)
			alphaScore := selectConformanceFeatures(readAlphaTabScore(t, test.Fixture), test.Features)
			assertDifferenceSnapshot(t, test.ID, "input-go-vs-alphatab", goScore, alphaScore)
		})
	}
}

func TestAlphaTabExportConformance(t *testing.T) {
	cases := readConformanceCases(t)
	for _, test := range cases.ExportCases {
		t.Run(test.ID, func(t *testing.T) {
			requireAlphaTabConformance(t)
			source := conformanceExportCase(t, test.ID)
			data, err := Export(source, ExportFormatGP8)
			if err != nil {
				t.Fatal(err)
			}
			path := writeConformanceFixture(t, data)
			assertDifferenceSnapshot(
				t,
				test.ID,
				"go-source-vs-export-alphatab",
				selectConformanceFeatures(normalizeGoScore(source), test.Features),
				selectConformanceFeatures(readAlphaTabScore(t, path), test.Features),
			)
		})
	}
}

func TestAlphaTabPreservesInspectedCapo(t *testing.T) {
	requireAlphaTabConformance(t)
	source := semanticExportProbeSong(t)
	source.Tracks[0].CapoFret = 2
	data, err := Export(source, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	score, ok := readAlphaTabScore(t, writeConformanceFixture(t, data)).(map[string]any)
	if !ok {
		t.Fatal("AlphaTab score is not an object")
	}
	tracks, ok := score["tracks"].([]any)
	if !ok || len(tracks) != 1 {
		t.Fatalf("AlphaTab tracks = %#v", score["tracks"])
	}
	track, ok := tracks[0].(map[string]any)
	if !ok {
		t.Fatalf("AlphaTab track = %#v", tracks[0])
	}
	staves, ok := track["staves"].([]any)
	if !ok || len(staves) != 1 {
		t.Fatalf("AlphaTab staves = %#v", track["staves"])
	}
	staff, ok := staves[0].(map[string]any)
	if !ok || staff["capo"] != float64(2) {
		t.Fatalf("AlphaTab capo = %#v, want 2", staff["capo"])
	}
}

func TestAlphaTabPreservesIndependentStaffCapos(t *testing.T) {
	requireAlphaTabConformance(t)
	source := conformanceStaffCapoSong(t, 2, 5)
	data, report, err := ExportWithReport(source, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true},
	})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("two-staff capo export = %v, %#v", err, report.Entries)
	}
	score, ok := readAlphaTabScore(t, writeConformanceFixture(t, data)).(map[string]any)
	if !ok {
		t.Fatal("AlphaTab score is not an object")
	}
	tracks, ok := score["tracks"].([]any)
	if !ok || len(tracks) != 1 {
		t.Fatalf("AlphaTab tracks = %#v", score["tracks"])
	}
	track, ok := tracks[0].(map[string]any)
	if !ok {
		t.Fatalf("AlphaTab track = %#v", tracks[0])
	}
	staves, ok := track["staves"].([]any)
	if !ok || len(staves) != 2 {
		t.Fatalf("AlphaTab staves = %#v", track["staves"])
	}
	for staffIndex, want := range []struct {
		capo float64
		midi float64
	}{{capo: 2, midi: 69}, {capo: 5, midi: 57}} {
		staff, staffOK := staves[staffIndex].(map[string]any)
		if !staffOK || staff["capo"] != want.capo {
			t.Fatalf("AlphaTab staff %d capo = %#v, want %v", staffIndex, staff["capo"], want.capo)
		}
		bars := staff["bars"].([]any)
		voices := bars[0].(map[string]any)["voices"].([]any)
		beats := voices[0].(map[string]any)["beats"].([]any)
		notes := beats[0].(map[string]any)["notes"].([]any)
		if midi := notes[0].(map[string]any)["midi"]; midi != want.midi {
			t.Fatalf("AlphaTab staff %d sounding MIDI = %#v, want %v", staffIndex, midi, want.midi)
		}
	}
	conformanceIndependentClaim(t, "field:Staff.CapoFret", claimSite("capo", "import", "M04-STAFF-CAPO", "two staff capos 2 and 5"), claimSite("capo", "model", "M04-STAFF-CAPO", "two staff capos 2 and 5"), claimSite("capo", "export", "M04-STAFF-CAPO", "two staff capos 2 and 5"))
}

func TestAlphaTabPreservesTuningLabels(t *testing.T) {
	requireAlphaTabConformance(t)
	source := conformanceTuningLabelExportSong(t)
	data, report, err := ExportWithReport(source, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true},
	})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("tuning label export = %v, %#v", err, report.Entries)
	}
	path := writeConformanceFixture(t, data)
	command := exec.Command("node", alphaTabOracleScript(), "--tuning-labels", path)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("AlphaTab tuning oracle: %v\n%s", err, output)
	}
	var staves []struct {
		Track  int       `json:"track"`
		Staff  int       `json:"staff"`
		Name   string    `json:"name"`
		Tuning []float64 `json:"tuning"`
		Capo   float64   `json:"capo"`
	}
	if err := json.Unmarshal(output, &staves); err != nil || len(staves) < 2 {
		t.Fatalf("AlphaTab tuning facts = %#v, %v", staves, err)
	}
	for staffIndex, wantLabel := range []string{"Author's tuning", ""} {
		staff := staves[staffIndex]
		if staff.Track != 0 || staff.Staff != staffIndex || staff.Name != wantLabel {
			t.Fatalf("AlphaTab staff %d label facts = %#v, want %q", staffIndex, staff, wantLabel)
		}
		if !slices.Equal(staff.Tuning, []float64{65, 60, 56, 51, 46, 41}) {
			t.Fatalf("AlphaTab staff %d tuning = %#v", staffIndex, staff.Tuning)
		}
		if staff.Capo != 0 {
			t.Fatalf("AlphaTab staff %d capo = %#v, want 0", staffIndex, staff.Capo)
		}
	}
	conformanceIndependentClaim(t, "field:Staff.TuningName", claimSite("tuning", "import", "M04-TUNING-LABELS", "two identical pitch arrays with distinct labels"), claimSite("tuning", "model", "M04-TUNING-LABELS", "two identical pitch arrays with distinct labels"), claimSite("tuning", "export", "M04-TUNING-LABELS", "two identical pitch arrays with distinct labels"))
}

func TestAlphaTabPreservesClefOctaves(t *testing.T) {
	requireAlphaTabConformance(t)
	source := conformanceClefOctaveSong(t)
	data, report, err := ExportWithReport(source, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true},
	})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("clef octave export = %v, %#v", err, report.Entries)
	}
	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Tracks[0].Measures[1].ClefOctave = OctaveOttavaBassa
	parsed.Tracks[0].Staves[1].Measures[0].ClefOctave = OctaveQuindicesima
	edited, editedReport, err := ExportWithReport(parsed, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{
			"gp8.normalize.source-version", "gp8.omit.track-display-settings",
		}},
	})
	if err != nil {
		t.Fatalf("edited clef octave export = %v, %#v", err, editedReport.Entries)
	}
	if got := conformanceAlphaTabClefOctaves(t, edited); !slices.Equal(got, []string{"8va", "8vb", "none", "15ma", "15mb", "none"}) {
		t.Fatalf("AlphaTab clef octaves = %v", got)
	}
	if got := conformanceBeatAlphaTabFirstOctave(t, edited); got != "15mb" {
		t.Fatalf("AlphaTab beat octave = %q, want 15mb", got)
	}
	conformanceIndependentClaim(t, "field:Measure.ClefOctave", claimSite("clef-octave", "import", "M07-CLEF-OCTAVE", "8va"), claimSite("clef-octave", "model", "M07-CLEF-OCTAVE", "8va"), claimSite("clef-octave", "export", "M07-CLEF-OCTAVE", "8va"))
}

func TestAlphaTabPreservesDirections(t *testing.T) {
	requireAlphaTabConformance(t)

	for _, test := range []struct {
		name string
		path string
		want map[int][]DirectionSign
	}{
		{
			name: "GP5 complete marker set",
			path: "testdata/gp5/Directions.gp5",
			want: map[int][]DirectionSign{
				0: {DirectionSignCoda}, 1: {DirectionSignDoubleCoda}, 2: {DirectionSignSegno}, 3: {DirectionSignSegnoSegno}, 4: {DirectionSignFine},
				5: {DirectionSignDaCapo}, 6: {DirectionSignDaCapoAlCoda}, 7: {DirectionSignDaCapoAlDoubleCoda}, 8: {DirectionSignDaCapoAlFine},
				9: {DirectionSignDaSegno}, 10: {DirectionSignDaSegnoSegno}, 11: {DirectionSignDaSegnoAlCoda}, 12: {DirectionSignDaSegnoAlDoubleCoda},
				13: {DirectionSignDaSegnoSegnoAlCoda}, 14: {DirectionSignDaSegnoSegnoAlDoubleCoda}, 15: {DirectionSignDaSegnoAlFine},
				16: {DirectionSignDaSegnoSegnoAlFine}, 17: {DirectionSignDaCoda}, 18: {DirectionSignDaDoubleCoda},
			},
		},
		{name: "GPIF targets and jumps", path: "testdata/gp7/timer.gp", want: map[int][]DirectionSign{3: {DirectionSignFine}, 7: {DirectionSignDaCapoAlFine}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := parseTestFixture(t, test.path)
			data, err := Export(source, ExportFormatGP8)
			if err != nil {
				t.Fatal(err)
			}
			got := conformanceAlphaTabDirections(t, data)
			for index, want := range test.want {
				if !slices.Equal(got[index], conformanceDirectionNames(want)) {
					t.Fatalf("AlphaTab measure %d directions = %v, want %v", index, got[index], conformanceDirectionNames(want))
				}
			}
		})
	}

	programmatic := semanticValidPitchedGP8Song(t)
	programmatic.Tracks[0].Settings.Notation = true
	all := append(slices.Clone(directionSignOrder), directionJumpOrder...)
	programmatic.MeasureHeaders[0].Directions = all
	data, report, err := ExportWithReport(programmatic, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("simultaneous directions export = %v, %#v", err, report.Entries)
	}
	if got := conformanceAlphaTabDirections(t, data)[0]; !slices.Equal(got, conformanceDirectionNames(all)) {
		t.Fatalf("AlphaTab simultaneous directions = %v", got)
	}
	conformanceIndependentClaim(t, "field:MeasureHeader.Directions", claimSite("directions", "import", "M07-DIRECTIONS", "simultaneous target and jump markers"), claimSite("directions", "model", "M07-DIRECTIONS", "simultaneous target and jump markers"), claimSite("directions", "export", "M07-DIRECTIONS", "simultaneous target and jump markers"))
}

func TestAlphaTabPreservesSimileMarks(t *testing.T) {
	requireAlphaTabConformance(t)
	source := conformanceMeasureSong(t)
	data, err := Export(source, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	score, ok := readAlphaTabScore(t, writeConformanceFixture(t, data)).(map[string]any)
	if !ok {
		t.Fatal("AlphaTab score is not an object")
	}
	tracks, ok := score["tracks"].([]any)
	if !ok || len(tracks) != 1 {
		t.Fatalf("AlphaTab tracks = %#v", score["tracks"])
	}
	track := tracks[0].(map[string]any)
	staff := track["staves"].([]any)[0].(map[string]any)
	bars := staff["bars"].([]any)
	got := make([]string, len(bars))
	for index, item := range bars {
		got[index] = item.(map[string]any)["simileMark"].(string)
	}
	want := []string{"none", "simple", "first-of-double", "second-of-double"}
	if !slices.Equal(got, want) {
		t.Fatalf("AlphaTab simile marks = %v, want %v", got, want)
	}
	conformanceIndependentClaim(t, "field:Measure.SimileMark", claimSite("simile", "import", "M07-MASTER-BARS", "all simile marks"), claimSite("simile", "model", "M07-MASTER-BARS", "all simile marks"), claimSite("simile", "export", "M07-MASTER-BARS", "all simile marks"))
}

func TestAlphaTabPreservesNativePercussionFallbacks(t *testing.T) {
	requireAlphaTabConformance(t)
	source := conformanceNativePercussionSong(t)
	data, report, err := ExportWithReport(source, ExportFormatGP8, ExportOptions{
		LossPolicy: ExportLossPolicy{RequirePreservation: true},
	})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("native percussion export = %v, %#v", err, report.Entries)
	}
	score := readAlphaTabScore(t, writeConformanceFixture(t, data)).(map[string]any)
	track := score["tracks"].([]any)[0].(map[string]any)
	definitions := track["percussionArticulations"].([]any)
	if len(definitions) != len(conformanceNativePercussionCases()) {
		t.Fatalf("AlphaTab percussion definitions = %d, want %d", len(definitions), len(conformanceNativePercussionCases()))
	}
	byInput := make(map[int]map[string]any, len(definitions))
	for _, item := range definitions {
		definition := item.(map[string]any)
		byInput[int(definition["inputMidiNumber"].(float64))] = definition
	}
	for _, want := range conformanceNativePercussionCases() {
		definition, ok := byInput[int(want.midi)]
		if !ok {
			t.Errorf("AlphaTab has no definition for input MIDI %d", want.midi)
			continue
		}
		wantHeads := strings.Fields(strings.ToLower(want.noteheads))
		got := []any{
			definition["elementName"], definition["staffLine"], definition["noteheadDefault"], definition["noteheadHalf"], definition["noteheadWhole"],
			definition["techniquePlacement"], definition["techniqueSymbol"], definition["outputMidiNumber"],
		}
		wantDefinition := []any{
			want.element, float64(want.staffLine), wantHeads[0], wantHeads[1], wantHeads[2],
			want.placement, map[bool]string{true: strings.ToLower(want.symbol), false: "none"}[want.symbol != ""], float64(want.outputMIDI),
		}
		if !slices.Equal(got, wantDefinition) {
			t.Errorf("AlphaTab definition %d = %#v, want %#v", want.midi, got, wantDefinition)
		}
	}

	staff := track["staves"].([]any)[0].(map[string]any)
	bar := staff["bars"].([]any)[0].(map[string]any)
	voice := bar["voices"].([]any)[0].(map[string]any)
	beat := voice["beats"].([]any)[0].(map[string]any)
	notes := beat["notes"].([]any)
	if len(notes) != len(conformanceNativePercussionCases()) {
		t.Fatalf("AlphaTab percussion notes = %d, want %d", len(notes), len(conformanceNativePercussionCases()))
	}
	for index, want := range conformanceNativePercussionCases() {
		note := notes[index].(map[string]any)
		if note["percussionInput"] != float64(want.midi) || note["midi"] != float64(want.outputMIDI) {
			t.Errorf("AlphaTab note %d input/output = %#v/%#v, want %d/%d", index, note["percussionInput"], note["midi"], want.midi, want.outputMIDI)
		}
	}
}

func TestAlphaTabPreservesCrossInstrumentPercussionGrace(t *testing.T) {
	requireAlphaTabConformance(t)
	source := conformancePercussionBuiltinPercussionSong(t, 46)
	exact, err := NewFret(38)
	if err != nil {
		t.Fatal(err)
	}
	note := &source.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
	note.Effect.Graces = []GraceEffect{{
		Duration:  DurationThirtySecond,
		ExactFret: &exact,
		Fret:      38,
		Velocity:  Forte,
	}}
	source.Tracks[0].Staves[0].Measures = source.Tracks[0].Measures
	if finalizeErr := FinalizeSong(source); finalizeErr != nil {
		t.Fatal(finalizeErr)
	}
	data, err := Export(source, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	gotGrace := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces
	if len(gotGrace) != 1 || gotGrace[0].Fret != 38 {
		t.Fatalf("Go cross-instrument percussion grace = %#v, want fret 38", gotGrace)
	}
	score := readAlphaTabScore(t, writeConformanceFixture(t, data)).(map[string]any)
	track := score["tracks"].([]any)[0].(map[string]any)
	staff := track["staves"].([]any)[0].(map[string]any)
	bar := staff["bars"].([]any)[0].(map[string]any)
	voice := bar["voices"].([]any)[0].(map[string]any)
	beat := voice["beats"].([]any)[0].(map[string]any)
	alphaNote := beat["notes"].([]any)[0].(map[string]any)
	if graces := alphaNote["graces"].([]any); len(graces) != 1 {
		t.Fatalf("AlphaTab cross-instrument percussion graces = %#v, want one", graces)
	}
}

func TestAlphaTabGPIFCapoPrecedence(t *testing.T) {
	requireAlphaTabConformance(t)
	gpif := strings.Replace(multiStaffFollowedByTrackGPIF, "<Name>Piano</Name>",
		`<Name>Piano</Name><Properties><Property name="CapoFret"><Fret>2</Fret></Property></Properties>`, 1)
	gpif = strings.Replace(gpif,
		`<Staff><Properties><Property name="Tuning"><Pitches>40 45</Pitches></Property></Properties></Staff>`,
		`<Staff><Properties><Property name="Tuning"><Pitches>40 45</Pitches></Property><Property name="CapoFret"><Fret>4</Fret></Property></Properties></Staff>`, 1)
	gpif = strings.Replace(gpif,
		`<Staff><Properties><Property name="Tuning"><Pitches>36 43</Pitches></Property></Properties></Staff>`,
		`<Staff><Properties><Property name="Tuning"><Pitches>36 43</Pitches></Property><Property name="CapoFret"><Fret>4</Fret></Property></Properties></Staff>`, 1)
	data := conformanceGPIFArchive(t, gpif)
	result, err := ParseWithOptions(data, ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Song.Tracks[0].CapoFret != 4 {
		t.Fatalf("Go effective capo = %d, want 4", result.Song.Tracks[0].CapoFret)
	}
	score, ok := readAlphaTabScore(t, writeConformanceFixture(t, data)).(map[string]any)
	if !ok {
		t.Fatal("AlphaTab score is not an object")
	}
	tracks, ok := score["tracks"].([]any)
	if !ok || len(tracks) < 1 {
		t.Fatalf("AlphaTab tracks = %#v", score["tracks"])
	}
	track, ok := tracks[0].(map[string]any)
	if !ok {
		t.Fatalf("AlphaTab track = %#v", tracks[0])
	}
	staves, ok := track["staves"].([]any)
	if !ok || len(staves) != 2 {
		t.Fatalf("AlphaTab staves = %#v", track["staves"])
	}
	for staffIndex, item := range staves {
		staff, staffOK := item.(map[string]any)
		if !staffOK || staff["capo"] != float64(4) {
			t.Fatalf("AlphaTab staff %d capo = %#v, want 4", staffIndex, staff["capo"])
		}
	}
}

func TestAlphaTabComparatorDetectsWireMutations(t *testing.T) {
	requireAlphaTabConformance(t)
	source := conformanceExportSong()
	data, err := Export(source, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	corrected := rewriteConformanceGPIF(t, data, func(gpif string) string {
		harmonic := regexp.MustCompile(`(<Property name="HarmonicFret">\s*)<Float>([^<]+)</Float>`)
		gpif = harmonic.ReplaceAllString(gpif, `${1}<HFret>${2}</HFret>`)
		return strings.ReplaceAll(gpif, "<Hairpin>Diminuendo</Hairpin>", "<Hairpin>Decrescendo</Hairpin>")
	})
	mutated := rewriteConformanceGPIF(t, corrected, func(gpif string) string {
		harmonic := regexp.MustCompile(`(<Property name="HarmonicFret">\s*)<HFret>([^<]+)</HFret>`)
		gpif = harmonic.ReplaceAllString(gpif, `${1}<Float>${2}</Float>`)
		return strings.ReplaceAll(gpif, "<Hairpin>Decrescendo</Hairpin>", "<Hairpin>Diminuendo</Hairpin>")
	})

	correctedScore := readAlphaTabScore(t, writeConformanceFixture(t, corrected))
	mutatedScore := readAlphaTabScore(t, writeConformanceFixture(t, mutated))
	differences := semanticDifferences(correctedScore, mutatedScore)
	if !differenceContains(differences, "/effects/harmonic/fret") {
		t.Error("HFret -> Float mutation did not change normalized harmonic semantics")
	}
	if !differenceContains(differences, "/hairpin") {
		t.Error("Decrescendo -> Diminuendo mutation did not change normalized hairpin semantics")
	}
}

func TestStaffOwnershipProjectionDetectsContentMutations(t *testing.T) {
	baselineSong := parseTestFixture(t, "testdata/gp7/grand-staff.gp")
	baseline := selectConformanceFeatures(normalizeGoScore(baselineSong), []string{"staff-ownership"})

	pitchSong := parseTestFixture(t, "testdata/gp7/grand-staff.gp")
	pitchBeats := conformanceStaffBeatsWithNotes(&pitchSong.Tracks[0].Staves[1])
	if len(pitchBeats) == 0 {
		t.Fatal("second staff has no notes")
	}
	pitchBeats[0].Notes[0].Value++
	if differences := semanticDifferences(baseline, selectConformanceFeatures(normalizeGoScore(pitchSong), []string{"staff-ownership"})); len(differences) == 0 {
		t.Fatal("staff-ownership projection did not detect a second-staff pitch mutation")
	}

	moveSong := parseTestFixture(t, "testdata/gp7/grand-staff.gp")
	moveBeats := conformanceStaffBeatsWithNotes(&moveSong.Tracks[0].Staves[1])
	if len(moveBeats) < 2 {
		t.Fatal("second staff has fewer than two populated beats")
	}
	note := moveBeats[0].Notes[0]
	moveBeats[0].Notes = moveBeats[0].Notes[1:]
	moveBeats[1].Notes = append(moveBeats[1].Notes, note)
	if differences := semanticDifferences(baseline, selectConformanceFeatures(normalizeGoScore(moveSong), []string{"staff-ownership"})); len(differences) == 0 {
		t.Fatal("staff-ownership projection did not detect moving a note between beats")
	}
}

func conformanceStaffBeatsWithNotes(staff *Staff) []*Beat {
	var beats []*Beat
	for measureIndex := range staff.Measures {
		for voiceIndex := range staff.Measures[measureIndex].Voices {
			voice := &staff.Measures[measureIndex].Voices[voiceIndex]
			for beatIndex := range voice.Beats {
				if len(voice.Beats[beatIndex].Notes) > 0 {
					beats = append(beats, &voice.Beats[beatIndex])
				}
			}
		}
	}
	return beats
}

func TestAlphaTabMultiStaffTrackOrdering(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, test := range []struct {
		name string
		gpif string
	}{
		{name: "all staves present", gpif: multiStaffFollowedByTrackGPIF},
		{
			name: "missing multi-staff track",
			gpif: strings.Replace(multiStaffFollowedByTrackGPIF, "<Bars>0 1 2</Bars>", "<Bars>-1 2</Bars>", 1),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			song, err := parseGPIF([]byte(test.gpif))
			if err != nil {
				t.Fatal(err)
			}
			archive := conformanceGPIFArchive(t, test.gpif)
			goScore := selectConformanceFeatures(normalizeGoScore(song), []string{"staff-ownership"})
			alphaScore := selectConformanceFeatures(
				readAlphaTabScore(t, writeConformanceFixture(t, archive)),
				[]string{"staff-ownership"},
			)
			if differences := semanticDifferences(goScore, alphaScore); len(differences) != 0 {
				formatted, marshalErr := json.MarshalIndent(differences, "", "  ")
				if marshalErr != nil {
					t.Fatal(marshalErr)
				}
				t.Fatalf("multi-staff track ordering differs from AlphaTab:\n%s", formatted)
			}
		})
	}
	conformanceIndependentClaim(t, "field:Track.Staves", claimSite("ownership", "import", "M03-OWNERSHIP-IMPORT", "two-staff track followed by one-staff track"), claimSite("ownership", "model", "M22-MULTI-STAFF-CONTEXT", "two-staff track followed by one-staff track"))
}

func TestAlphaTabTempoReferences(t *testing.T) {
	requireAlphaTabConformance(t)
	fixture, err := os.ReadFile("testdata/gp7/notes.gp")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		value string
		want  float64
	}{
		{value: "120 1", want: 60},
		{value: "120 2", want: 120},
		{value: "120 3", want: 180},
		{value: "120 4", want: 240},
		{value: "120 5", want: 360},
		{value: "120", want: 60},
		{value: "120 invalid", want: 120},
		{value: "120 0", want: 120},
		{value: "120 6", want: 120},
		{value: "120 3x", want: 180},
		{value: "120 3.5", want: 180},
		{value: "81.5 3", want: 122.25},
	} {
		t.Run(test.value, func(t *testing.T) {
			data := rewriteConformanceGPIF(t, fixture, func(gpif string) string {
				return strings.Replace(gpif, "<Value>120 2</Value>", "<Value>"+test.value+"</Value>", 1)
			})
			song, parseErr := Parse(data)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			if len(song.TempoAutomations) == 0 || song.TempoAutomations[0].Tempo != test.want {
				t.Fatalf("Go tempo for %q = %#v, want %v", test.value, song.TempoAutomations, test.want)
			}
			goScore := selectConformanceFeatures(normalizeGoScore(song), []string{"tempo-automations"})
			alphaScore := selectConformanceFeatures(
				readAlphaTabScore(t, writeConformanceFixture(t, data)),
				[]string{"tempo-automations"},
			)
			if differences := semanticDifferences(goScore, alphaScore); len(differences) != 0 {
				formatted, marshalErr := json.MarshalIndent(differences, "", "  ")
				if marshalErr != nil {
					t.Fatal(marshalErr)
				}
				t.Fatalf("tempo reference %q differs from AlphaTab:\n%s", test.value, formatted)
			}
		})
	}
	conformanceIndependentClaim(t, "field:Song.TempoAutomations", claimSite("tempo", "import", "M06-TEMPO-AUTHORITY", "fractional opening tempo"), claimSite("tempo", "model", "M06-TEMPO-AUTHORITY", "fractional opening tempo"))
}

func TestAlphaTabGPIFTiming(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, test := range []struct {
		name    string
		fixture string
		old     string
		new     string
	}{
		{name: "notes", fixture: "testdata/gp7/notes.gp"},
		{name: "time signatures", fixture: "testdata/gp7/time-signatures.gp"},
		{name: "pickup", fixture: "testdata/gp7/anacrusis.gp"},
		{name: "empty pickup", fixture: "testdata/gp7/anacrusis.gp", old: "<Beats>0 1</Beats>", new: "<Beats>-1</Beats>"},
		{name: "multiple voices", fixture: "testdata/gp7/multi-voice.gp"},
		{name: "tuplets", fixture: "testdata/gp7/tuplets.gp"},
		{name: "grace", fixture: "testdata/gp7/grace.gp"},
		{name: "unmatched grace", fixture: "testdata/gp7/grace.gp", old: "<Beats>0 1 2 3 4</Beats>", new: "<Beats>1 2 4</Beats>"},
	} {
		t.Run(test.name, func(t *testing.T) {
			data, err := os.ReadFile(test.fixture)
			if err != nil {
				t.Fatal(err)
			}
			if test.old != "" {
				data = rewriteConformanceGPIF(t, data, func(gpif string) string {
					return strings.Replace(gpif, test.old, test.new, 1)
				})
			}
			song, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			goScore := selectConformanceFeatures(normalizeGoScore(song), []string{"timing"})
			alphaScore := selectConformanceFeatures(
				readAlphaTabScore(t, writeConformanceFixture(t, data)),
				[]string{"timing"},
			)
			differences := semanticDifferences(goScore, alphaScore)
			if len(differences) != 0 {
				formatted, marshalErr := json.MarshalIndent(differences, "", "  ")
				if marshalErr != nil {
					t.Fatal(marshalErr)
				}
				t.Fatalf("timing differs from AlphaTab:\n%s", formatted)
			}
		})
	}
	conformanceIndependentClaim(t, "field:Song.Anacrusis", claimSite("pickup", "import", "M22-PICKUP-TUPLET-TEMPO", "pickup"), claimSite("pickup", "model", "M22-PICKUP-TUPLET-TEMPO", "pickup"))
	conformanceIndependentClaim(t, "field:Beat.Duration", claimSite("duration", "import", "M08-DURATIONS", "all note values"), claimSite("duration", "model", "M08-DURATIONS", "all note values"))
	conformanceIndependentClaim(t, "field:Duration.TupletEnters", claimSite("tuplets", "import", "M08-DURATIONS", "1:1, 3:2, 5:4, 7:4, and 255:254 tuplets"), claimSite("tuplets", "model", "M08-DURATIONS", "1:1, 3:2, 5:4, 7:4, and 255:254 tuplets"))
}

func TestAlphaTabGP5DurationPercentWireValues(t *testing.T) {
	requireAlphaTabConformance(t)
	tests := []struct {
		fixture       string
		threeQuarters int
		halves        int
	}{
		{fixture: "testdata/gp5/Effects.gp5", threeQuarters: 6, halves: 1},
		{fixture: "testdata/gp5/other-effects.gp5", threeQuarters: 1, halves: 1},
	}
	for _, test := range tests {
		t.Run(test.fixture, func(t *testing.T) {
			data, err := os.ReadFile(test.fixture)
			if err != nil {
				t.Fatal(err)
			}
			littleThreeQuarters := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xe8, 0x3f}
			littleHalf := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xe0, 0x3f}
			if got := bytes.Count(data, littleThreeQuarters); got != test.threeQuarters {
				t.Fatalf("little-endian 0.75 values = %d, want %d", got, test.threeQuarters)
			}
			if got := bytes.Count(data, littleHalf); got != test.halves {
				t.Fatalf("little-endian 0.5 values = %d, want %d", got, test.halves)
			}

			song, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			goFacts := durationPercentFacts(normalizeGoScore(song))
			alphaFacts := durationPercentFacts(readAlphaTabScore(t, test.fixture))
			if differences := semanticDifferences(goFacts, alphaFacts); len(differences) != 0 {
				formatted, marshalErr := json.MarshalIndent(differences, "", "  ")
				if marshalErr != nil {
					t.Fatal(marshalErr)
				}
				t.Fatalf("duration percentages differ from little-endian wire values:\n%s", formatted)
			}
		})
	}
}

func durationPercentFacts(score any) []any {
	data, _ := json.Marshal(score)
	var canonical any
	_ = json.Unmarshal(data, &canonical)
	return collectConformanceFacts(canonical, map[string]bool{"durationPercent": true}, nil)
}

func requireAlphaTabConformance(t *testing.T) {
	t.Helper()
	if os.Getenv("ALPHATAB_CONFORMANCE") != "1" {
		t.Skip("run through ./conformance/check.sh")
	}
}

func readConformanceCases(t *testing.T) conformanceCases {
	t.Helper()
	data, err := os.ReadFile("conformance/cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases conformanceCases
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if cases.SchemaVersion != 1 {
		t.Fatalf("conformance schema version = %d, want 1", cases.SchemaVersion)
	}
	return cases
}

func verifyFixtureHash(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile("conformance/fixture-inventory.json")
	if err != nil {
		if os.IsNotExist(err) && os.Getenv("ALPHATAB_CONFORMANCE_UPDATE") == "1" {
			return
		}
		t.Fatal(err)
	}
	var inventory struct {
		Fixtures []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"fixtures"`
	}
	if err := json.Unmarshal(data, &inventory); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range inventory.Fixtures {
		if fixture.Path != path {
			continue
		}
		contents, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		digest := sha256.Sum256(contents)
		if got := hex.EncodeToString(digest[:]); got != fixture.SHA256 {
			t.Fatalf("fixture hash = %s, want %s; review fixture drift before refreshing snapshots", got, fixture.SHA256)
		}
		return
	}
	t.Fatalf("fixture %q is absent from conformance/fixture-inventory.json", path)
}

func readAlphaTabScore(t *testing.T, fixture string) any {
	t.Helper()
	command := exec.Command("node", alphaTabOracleScript(), fixture)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("AlphaTab oracle: %v\n%s", err, output)
	}
	var score any
	if err := json.Unmarshal(output, &score); err != nil {
		t.Fatalf("decoding AlphaTab snapshot: %v\n%s", err, output)
	}
	return score
}

func alphaTabOracleScript() string {
	if overlay := os.Getenv("ALPHATAB_ORACLE_OVERLAY"); overlay != "" {
		return overlay
	}
	return "conformance/oracle.mjs"
}

func assertDifferenceSnapshot(t *testing.T, id, comparison string, goScore, alphaScore any) {
	t.Helper()
	differences := semanticDifferences(goScore, alphaScore)
	for index := range differences {
		differences[index].Feature = conformanceFeature(differences[index].Path)
		if differences[index].Feature == "" {
			t.Fatalf("unclassified semantic difference at %s", differences[index].Path)
		}
	}
	snapshot := differenceSnapshot{
		SchemaVersion: 1,
		Case:          id,
		Comparison:    comparison,
		Differences:   differences,
	}
	path := filepath.Join("conformance", "snapshots", id+".json")
	actual, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	actual = append(actual, '\n')
	if os.Getenv("ALPHATAB_CONFORMANCE_UPDATE") == "1" {
		if mkdirErr := os.MkdirAll(filepath.Dir(path), 0o755); mkdirErr != nil {
			t.Fatal(mkdirErr)
		}
		if writeErr := os.WriteFile(path, actual, 0o644); writeErr != nil {
			t.Fatal(writeErr)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read semantic baseline: %v; regenerate only after reviewing and classifying the diff", err)
	}
	if !bytes.Equal(actual, want) {
		t.Fatalf("semantic differences changed for %s; inspect before running the explicit update command", id)
	}
}
