// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
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
	source.Tracks[0].Offset = 2
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
	if result.Song.Tracks[0].Offset != 4 {
		t.Fatalf("Go effective capo = %d, want 4", result.Song.Tracks[0].Offset)
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
}

func TestAlphaTabGPIFTiming(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, test := range []struct {
		name                  string
		fixture               string
		old                   string
		new                   string
		exactTupletDifference bool
	}{
		{name: "notes", fixture: "testdata/gp7/notes.gp"},
		{name: "time signatures", fixture: "testdata/gp7/time-signatures.gp"},
		{name: "pickup", fixture: "testdata/gp7/anacrusis.gp"},
		{name: "empty pickup", fixture: "testdata/gp7/anacrusis.gp", old: "<Beats>0 1</Beats>", new: "<Beats>-1</Beats>"},
		{name: "multiple voices", fixture: "testdata/gp7/multi-voice.gp"},
		{name: "tuplets", fixture: "testdata/gp7/tuplets.gp", exactTupletDifference: true},
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
			if test.exactTupletDifference && reflect.DeepEqual(differences, []semanticDifference{{
				Path: "/timing/24/value", Go: float64(2560), AlphaTab: float64(2559),
			}}) {
				return
			}
			if len(differences) != 0 {
				formatted, marshalErr := json.MarshalIndent(differences, "", "  ")
				if marshalErr != nil {
					t.Fatal(marshalErr)
				}
				t.Fatalf("timing differs from AlphaTab:\n%s", formatted)
			}
		})
	}
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
	command := exec.Command("node", "conformance/oracle.mjs", fixture)
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

func semanticDifferences(goValue, alphaValue any) []semanticDifference {
	goJSON, _ := json.Marshal(goValue)
	alphaJSON, _ := json.Marshal(alphaValue)
	var left, right any
	_ = json.Unmarshal(goJSON, &left)
	_ = json.Unmarshal(alphaJSON, &right)
	differences := make([]semanticDifference, 0)
	collectSemanticDifferences("", left, right, &differences)
	return differences
}

func collectSemanticDifferences(path string, goValue, alphaValue any, differences *[]semanticDifference) {
	if reflect.DeepEqual(goValue, alphaValue) {
		return
	}
	leftMap, leftIsMap := goValue.(map[string]any)
	rightMap, rightIsMap := alphaValue.(map[string]any)
	if leftIsMap && rightIsMap {
		keys := make([]string, 0, len(leftMap)+len(rightMap))
		seen := make(map[string]bool)
		for key := range leftMap {
			keys = append(keys, key)
			seen[key] = true
		}
		for key := range rightMap {
			if !seen[key] {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)
		for _, key := range keys {
			collectSemanticDifferences(path+"/"+key, leftMap[key], rightMap[key], differences)
		}
		return
	}
	leftSlice, leftIsSlice := goValue.([]any)
	rightSlice, rightIsSlice := alphaValue.([]any)
	if leftIsSlice && rightIsSlice {
		shared := min(len(leftSlice), len(rightSlice))
		for index := range shared {
			collectSemanticDifferences(path+"/"+strconv.Itoa(index), leftSlice[index], rightSlice[index], differences)
		}
		for index := shared; index < len(leftSlice); index++ {
			*differences = append(*differences, semanticDifference{Path: path + "/" + strconv.Itoa(index), Go: leftSlice[index]})
		}
		for index := shared; index < len(rightSlice); index++ {
			*differences = append(*differences, semanticDifference{Path: path + "/" + strconv.Itoa(index), AlphaTab: rightSlice[index]})
		}
		return
	}
	*differences = append(*differences, semanticDifference{Path: path, Go: goValue, AlphaTab: alphaValue})
}

func conformanceFeature(path string) string {
	for _, feature := range []string{
		"rhythm", "timing", "grace-relationships", "tremolo-picking", "harmonics", "hairpins",
		"tempo-automations", "staff-ownership", "percussion-articulations", "note-and-beat-semantics", "score-core",
	} {
		if strings.HasPrefix(path, "/"+feature) {
			return feature
		}
	}
	classifications := []struct {
		fragments []string
		feature   string
	}{
		{[]string{"/graces"}, "grace-relationships"},
		{[]string{"/tremoloPicking"}, "tremolo-picking"},
		{[]string{"/harmonic"}, "harmonics"},
		{[]string{"/hairpin"}, "hairpins"},
		{[]string{"/tempoAutomations"}, "tempo-automations"},
		{[]string{"/staff-ownership", "/staffCount", "/noteCount", "/clefs", "/tuning"}, "staff-ownership"},
		{[]string{"/timing", "/start", "/durationTicks"}, "timing"},
		{[]string{"/rhythm", "/duration", "/tuplet", "/timeSignature", "/pickup"}, "rhythm"},
		{[]string{"/percussion", "/percussionArticulation", "/percussionInput", "/midi"}, "percussion-articulations"},
		{[]string{"/effects", "/kind", "/dynamic", "/status", "/notes", "/voices", "/bars"}, "note-and-beat-semantics"},
		{[]string{"/metadata", "/program", "/primaryChannel", "/name", "/index", "/repeat", "/alternateEndings", "/tripletFeel", "/text", "/schemaVersion", "/string", "/fret"}, "score-core"},
	}
	for _, classification := range classifications {
		for _, fragment := range classification.fragments {
			if strings.Contains(path, fragment) {
				return classification.feature
			}
		}
	}
	return ""
}

func selectConformanceFeatures(score any, features []string) any {
	data, _ := json.Marshal(score)
	var canonical any
	_ = json.Unmarshal(data, &canonical)
	selected := make(map[string]any, len(features))
	for _, feature := range features {
		switch feature {
		case "rhythm":
			selected[feature] = collectConformanceFacts(canonical, map[string]bool{
				"timeSignature": true, "pickup": true, "status": true, "duration": true, "durationTicks": true,
				"dots": true, "tuplet": true,
			}, nil)
		case "timing":
			selected[feature] = collectConformanceFacts(canonical, map[string]bool{"start": true}, nil)
		case "grace-relationships":
			selected[feature] = collectConformanceFacts(canonical, map[string]bool{"graces": true}, func(value any) bool {
				items, ok := value.([]any)
				return ok && len(items) > 0
			})
		case "tremolo-picking":
			selected[feature] = collectConformanceFacts(canonical, map[string]bool{"tremoloPicking": true}, nonNilConformanceFact)
		case "harmonics":
			selected[feature] = collectConformanceFacts(canonical, map[string]bool{"harmonic": true}, nonNilConformanceFact)
		case "hairpins":
			selected[feature] = collectConformanceFacts(canonical, map[string]bool{"hairpin": true}, func(value any) bool {
				return value != nil && value != "none"
			})
		case "tempo-automations":
			selected[feature] = collectConformanceFacts(canonical, map[string]bool{"tempoAutomations": true}, nil)
		case "staff-ownership":
			selected[feature] = conformanceStaffOwnership(canonical)
		case "percussion-articulations":
			selected[feature] = conformancePercussionFacts(canonical)
		case "score-core":
			selected[feature] = conformanceScoreCore(canonical)
		case "note-and-beat-semantics":
			selected[feature] = collectConformanceFacts(canonical, map[string]bool{
				"status": true, "dynamic": true, "text": true, "string": true, "fret": true,
				"kind": true, "durationPercent": true, "tieOrigin": true, "tieDestination": true, "effects": true,
				"octave": true,
			}, nil)
		default:
			panic("unhandled conformance feature " + feature)
		}
	}
	return selected
}

func conformanceScoreCore(score any) any {
	root, _ := score.(map[string]any)
	masterBars, _ := root["masterBars"].([]any)
	barFacts := make([]any, 0, len(masterBars))
	for _, item := range masterBars {
		bar, _ := item.(map[string]any)
		barFacts = append(barFacts, map[string]any{
			"index": bar["index"], "repeatStart": bar["repeatStart"], "repeatCount": bar["repeatCount"],
			"alternateEndings": bar["alternateEndings"], "tripletFeel": bar["tripletFeel"],
		})
	}
	tracks, _ := root["tracks"].([]any)
	trackFacts := make([]any, 0, len(tracks))
	for _, item := range tracks {
		track, _ := item.(map[string]any)
		trackFacts = append(trackFacts, map[string]any{
			"index": track["index"], "name": track["name"], "program": track["program"],
			"primaryChannel": track["primaryChannel"],
		})
	}
	return map[string]any{"metadata": root["metadata"], "masterBars": barFacts, "tracks": trackFacts}
}

func nonNilConformanceFact(value any) bool {
	return value != nil
}

func collectConformanceFacts(value any, keys map[string]bool, include func(any) bool) []any {
	var facts []any
	collectConformanceFactsAt("", value, keys, include, &facts)
	return facts
}

func collectConformanceFactsAt(path string, value any, keys map[string]bool, include func(any) bool, facts *[]any) {
	switch typed := value.(type) {
	case map[string]any:
		mapKeys := make([]string, 0, len(typed))
		for key := range typed {
			mapKeys = append(mapKeys, key)
		}
		sort.Strings(mapKeys)
		for _, key := range mapKeys {
			child := typed[key]
			childPath := path + "/" + key
			if key == "graces" && !keys[key] {
				continue
			}
			if keys[key] && (include == nil || include(child)) {
				*facts = append(*facts, map[string]any{"path": childPath, "value": child})
				continue
			}
			collectConformanceFactsAt(childPath, child, keys, include, facts)
		}
	case []any:
		for index, child := range typed {
			collectConformanceFactsAt(path+"/"+strconv.Itoa(index), child, keys, include, facts)
		}
	}
}

func conformanceStaffOwnership(score any) []any {
	root, _ := score.(map[string]any)
	tracks, _ := root["tracks"].([]any)
	result := make([]any, 0, len(tracks))
	for _, item := range tracks {
		track, _ := item.(map[string]any)
		result = append(result, map[string]any{
			"index": track["index"], "name": track["name"], "staves": track["staves"],
		})
	}
	return result
}

func conformancePercussionFacts(score any) []any {
	root, _ := score.(map[string]any)
	tracks, _ := root["tracks"].([]any)
	var facts []any
	for trackIndex, trackItem := range tracks {
		track, _ := trackItem.(map[string]any)
		if articulations, ok := track["percussionArticulations"].([]any); ok && len(articulations) > 0 {
			facts = append(facts, map[string]any{
				"path": fmt.Sprintf("/tracks/%d/percussionArticulations", trackIndex), "value": articulations,
			})
		}
		staves, _ := track["staves"].([]any)
		for staffIndex, staffItem := range staves {
			staff, _ := staffItem.(map[string]any)
			if staff["percussion"] != true {
				continue
			}
			facts = append(facts, map[string]any{
				"path": fmt.Sprintf("/tracks/%d/staves/%d/percussion", trackIndex, staffIndex), "value": true,
			})
			collectConformanceFactsAt(
				fmt.Sprintf("/tracks/%d/staves/%d", trackIndex, staffIndex),
				staff,
				map[string]bool{"standardNotationLineCount": true, "percussionArticulation": true, "percussionInput": true, "midi": true},
				nil,
				&facts,
			)
		}
	}
	return facts
}

func differenceContains(differences []semanticDifference, suffix string) bool {
	return slices.ContainsFunc(differences, func(difference semanticDifference) bool {
		return strings.HasSuffix(difference.Path, suffix)
	})
}

func normalizeGoScore(song *Song) any {
	masterBars := make([]any, 0, len(song.MeasureHeaders))
	for index, header := range song.MeasureHeaders {
		masterBars = append(masterBars, map[string]any{
			"index":            index,
			"start":            header.Start - DurationQuarterTime,
			"timeSignature":    []any{header.TimeSignature.Numerator, header.TimeSignature.Denominator.Value},
			"repeatStart":      header.RepeatOpen,
			"repeatCount":      normalizeGoRepeatCount(header.RepeatClose),
			"alternateEndings": header.RepeatAlternative,
			"tripletFeel":      goTripletFeel(header.TripletFeel),
			"pickup":           index == 0 && song.Anacrusis,
		})
	}
	tempoAutomations := make([]any, 0, len(song.TempoAutomations)+1)
	if len(song.TempoAutomations) == 0 && song.Tempo > 0 {
		tempoAutomations = append(tempoAutomations, map[string]any{
			"bar": 0, "position": float64(0), "type": "tempo", "value": float64(song.Tempo), "linear": false,
		})
	} else {
		for _, automation := range song.TempoAutomations {
			tempoAutomations = append(tempoAutomations, map[string]any{
				"bar": automation.Bar, "position": automation.Position, "type": "tempo", "value": automation.Tempo, "linear": false,
			})
		}
	}
	tracks := make([]any, 0, len(song.Tracks))
	for trackIndex := range song.Tracks {
		track := &song.Tracks[trackIndex]
		program := int32(0)
		primaryChannel := uint8(0)
		if track.ChannelIndex >= 0 && track.ChannelIndex < len(song.Channels) {
			program = song.Channels[track.ChannelIndex].Instrument
			primaryChannel = song.Channels[track.ChannelIndex].Channel
		}
		tracks = append(tracks, map[string]any{
			"index":                   trackIndex,
			"name":                    track.Name,
			"program":                 program,
			"primaryChannel":          primaryChannel,
			"percussionArticulations": normalizeGoPercussionArticulations(track.PercussionArticulations),
			"staves":                  normalizeGoStaves(song, trackIndex),
		})
	}
	return map[string]any{
		"schemaVersion": 1,
		"metadata": map[string]any{
			"title": song.Name, "subtitle": song.Subtitle, "artist": song.Artist, "album": song.Album,
			"words": song.Words, "music": song.Author, "copyright": song.Copyright, "instructions": song.Instructions,
		},
		"masterBars":       masterBars,
		"tempoAutomations": tempoAutomations,
		"tracks":           tracks,
	}
}

func normalizeGoTuning(strings []GuitarString) []any {
	result := make([]any, len(strings))
	for index := range strings {
		result[index] = strings[index].Value
	}
	return result
}

func normalizeGoPercussionArticulations(articulations []PercussionArticulation) []any {
	result := make([]any, 0, len(articulations))
	for _, articulation := range articulations {
		techniqueSymbol := "none"
		if articulation.TechniqueSymbol != "" {
			techniqueSymbol = strings.ToLower(articulation.TechniqueSymbol)
		}
		var inputMIDINumber any
		if len(articulation.InputMIDINumbers) > 0 {
			inputMIDINumber = articulation.InputMIDINumbers[0]
		}
		result = append(result, map[string]any{
			"elementName":        articulation.ElementName,
			"staffLine":          articulation.StaffLine,
			"noteheadDefault":    strings.ToLower(articulation.NoteheadDefault),
			"noteheadHalf":       strings.ToLower(articulation.NoteheadHalf),
			"noteheadWhole":      strings.ToLower(articulation.NoteheadWhole),
			"techniquePlacement": strings.ToLower(articulation.TechniquePlacement),
			"techniqueSymbol":    techniqueSymbol,
			"inputMidiNumber":    inputMIDINumber,
			"outputMidiNumber":   articulation.OutputMIDINumber,
		})
	}
	return result
}

func normalizeGoStaves(song *Song, trackIndex int) []any {
	track := &song.Tracks[trackIndex]
	staves := track.Staves
	if len(staves) == 0 {
		staves = []Staff{{
			Measures:                  track.Measures,
			Strings:                   track.Strings,
			PercussionTrack:           track.PercussionTrack,
			StandardNotationLineCount: 5,
		}}
	}
	result := make([]any, 0, len(staves))
	for staffIndex := range staves {
		staff := staves[staffIndex]
		staff.PercussionTrack = staff.PercussionTrack || track.PercussionTrack
		tuning := normalizeGoTuning(staff.Strings)
		if staff.PercussionTrack {
			tuning = []any{}
		}
		result = append(result, map[string]any{
			"index":                     staffIndex,
			"capo":                      track.Offset,
			"percussion":                staff.PercussionTrack,
			"standardNotationLineCount": staff.StandardNotationLineCount,
			"tuning":                    tuning,
			"bars":                      normalizeGoBars(song, track, &staff),
		})
	}
	return result
}

func normalizeGoBars(song *Song, track *Track, staff *Staff) []any {
	result := make([]any, 0, len(staff.Measures))
	for measureIndex := range staff.Measures {
		measure := &staff.Measures[measureIndex]
		voices := make([]any, 0, len(measure.Voices))
		for voiceIndex := range measure.Voices {
			voice := &measure.Voices[voiceIndex]
			if !goVoiceHasContent(voice) {
				continue
			}
			beats := make([]any, 0, len(voice.Beats))
			pendingGrace := make([]*Beat, 0)
			for beatIndex := range voice.Beats {
				beat := &voice.Beats[beatIndex]
				if beat.isGrace {
					pendingGrace = append(pendingGrace, beat)
					continue
				}
				pendingGrace = pendingGrace[:0]
				beats = append(beats, normalizeGoBeat(song, measureIndex, track, staff, beat))
			}
			for _, grace := range pendingGrace {
				beats = append(beats, normalizeGoBeat(song, measureIndex, track, staff, grace))
			}
			voices = append(voices, map[string]any{"index": voiceIndex, "beats": beats})
		}
		clef := goClef(measure.Clef)
		if staff.PercussionTrack {
			clef = "neutral"
		}
		result = append(result, map[string]any{"index": measureIndex, "clef": clef, "voices": voices})
	}
	return result
}

func goVoiceHasContent(voice *Voice) bool {
	return slices.ContainsFunc(voice.Beats, func(beat Beat) bool {
		return beat.Status != BeatStatusEmpty
	})
}

func normalizeGoBeat(song *Song, measureIndex int, track *Track, staff *Staff, beat *Beat) any {
	start := any(nil)
	if beat.Start != nil {
		start = *beat.Start - song.MeasureHeaders[measureIndex].Start
	}
	notes := make([]any, 0, len(beat.Notes))
	for noteIndex := range beat.Notes {
		notes = append(notes, normalizeGoNote(track, staff, &beat.Notes[noteIndex]))
	}
	return map[string]any{
		"start":          start,
		"status":         goBeatStatus(beat.Status),
		"duration":       beat.Duration.Value,
		"durationTicks":  beat.Duration.time(),
		"dots":           goDurationDots(beat.Duration),
		"tuplet":         []any{normalizedTuplet(beat.Duration.TupletEnters), normalizedTuplet(beat.Duration.TupletTimes)},
		"dynamic":        goDynamic(beat.Dynamics),
		"text":           beat.Text,
		"octave":         goOctave(beat.Octave),
		"hairpin":        goHairpin(beat.Effect.Hairpin),
		"tremoloPicking": goTremoloPicking(beat.Notes),
		"notes":          notes,
	}
}

func normalizeGoNote(track *Track, staff *Staff, note *Note) any {
	fret := any(note.Value)
	stringNumber := note.String
	articulation := any(nil)
	midi := int(note.Value)
	if staff.PercussionTrack {
		fret = nil
		stringNumber = -1
		if note.HasPercussionArticulation {
			articulation = note.PercussionArticulation
			if note.PercussionArticulation >= 0 && note.PercussionArticulation < len(track.PercussionArticulations) {
				midi = track.PercussionArticulations[note.PercussionArticulation].OutputMIDINumber
			}
		}
	} else if note.String == 0 {
		fret = nil
	} else if note.String > 0 && int(note.String) <= len(staff.Strings) {
		midi += int(staff.Strings[note.String-1].Value) + int(track.Offset)
	}
	graces := make([]any, 0, len(note.Effect.Graces))
	for _, grace := range note.Effect.Graces {
		rawFret := grace.Fret
		if grace.RawFret != nil {
			rawFret = *grace.RawFret
		}
		graces = append(graces, map[string]any{
			"rawFret": rawFret,
			"dead":    grace.IsDead, "onBeat": grace.IsOnBeat, "dynamic": goDynamic(grace.Velocity),
			"transition": goGraceTransition(grace.Transition), "staffPercussion": staff.PercussionTrack,
		})
	}
	percussionInput := any(nil)
	if staff.PercussionTrack {
		percussionInput = note.Value
	}
	return map[string]any{
		"string": stringNumber, "fret": fret, "percussionArticulation": articulation, "percussionInput": percussionInput, "midi": midi,
		"kind": goNoteKind(note.Kind), "dynamic": goDynamic(note.Velocity), "tieOrigin": note.TieOrigin,
		"durationPercent": note.DurationPercent,
		"tieDestination":  note.Kind == NoteTypeTie,
		"effects": map[string]any{
			"accent": goAccent(note.Effect), "ghost": note.Effect.GhostNote, "hammerOrigin": note.Effect.Hammer,
			"letRing": note.Effect.LetRing, "palmMute": note.Effect.PalmMute, "staccato": note.Effect.Staccato,
			"vibrato": goVibrato(note.Effect.Vibrato), "harmonic": normalizeGoHarmonic(note.Effect.Harmonic),
			"bend": normalizeGoBend(note.Effect.Bend), "trill": normalizeGoTrill(note.Effect.Trill),
			"slides": normalizeGoSlides(note.Effect.Slides),
		},
		"graces": graces,
	}
}

func normalizeGoHarmonic(harmonic *HarmonicEffect) any {
	if harmonic == nil {
		return nil
	}
	fret := any(nil)
	if harmonic.FretFloat != nil {
		fret = *harmonic.FretFloat
	} else if harmonic.Fret != nil {
		fret = *harmonic.Fret
	}
	return map[string]any{"kind": goHarmonicKind(harmonic.Kind), "fret": fret}
}

func normalizeGoBend(bend *BendEffect) any {
	if bend == nil || len(bend.Points) == 0 {
		return nil
	}
	points := make([]any, 0, len(bend.Points))
	for _, point := range bend.Points {
		points = append(points, map[string]any{"position": point.Position, "value": point.Value})
	}
	return points
}

func normalizeGoTrill(trill *TrillEffect) any {
	if trill == nil {
		return nil
	}
	return map[string]any{"fret": trill.Fret, "duration": trill.Duration.Value}
}

func normalizeGoSlides(slides []SlideType) []any {
	result := make([]any, 0, len(slides))
	for _, slide := range slides {
		result = append(result, goSlide(slide))
	}
	return result
}

func normalizedTuplet(value uint8) uint8 {
	if value == 0 {
		return 1
	}
	return value
}

func goDurationDots(duration Duration) int {
	if duration.DoubleDotted {
		return 2
	}
	if duration.Dotted {
		return 1
	}
	return 0
}

func goTremoloPicking(notes []Note) any {
	for _, note := range notes {
		if note.Effect.TremoloPicking != nil {
			return note.Effect.TremoloPicking.Duration.Value
		}
	}
	return nil
}

func goDynamic(velocity int16) string {
	delta := velocity - MinVelocity
	if delta < 0 || delta%VelocityIncrement != 0 {
		return strconv.Itoa(int(velocity))
	}
	index := delta / VelocityIncrement
	names := []string{"ppp", "pp", "p", "mp", "mf", "f", "ff", "fff"}
	if index < 0 || int(index) >= len(names) {
		return strconv.Itoa(int(velocity))
	}
	return names[index]
}

func goBeatStatus(status BeatStatus) string {
	switch status {
	case BeatStatusEmpty:
		return "empty"
	case BeatStatusRest:
		return "rest"
	case BeatStatusNormal:
		return "normal"
	default:
		return fmt.Sprintf("unknown:%d", status)
	}
}

func goNoteKind(kind NoteType) string {
	switch kind {
	case NoteTypeDead:
		return "dead"
	case NoteTypeTie:
		return "tie"
	case NoteTypeRest:
		return "rest"
	case NoteTypeNormal:
		return "normal"
	default:
		return fmt.Sprintf("unknown:%d", kind)
	}
}

func goHairpin(hairpin Hairpin) string {
	switch hairpin {
	case HairpinCrescendo:
		return "crescendo"
	case HairpinDiminuendo:
		return "decrescendo"
	case HairpinNone:
		return "none"
	default:
		return fmt.Sprintf("unknown:%d", hairpin)
	}
}

func goTripletFeel(feel TripletFeel) string {
	switch feel {
	case TripletFeelEighth:
		return "triplet-8th"
	case TripletFeelSixteenth:
		return "triplet-16th"
	case TripletFeelNone:
		return "none"
	default:
		return fmt.Sprintf("unknown:%d", feel)
	}
}

func goClef(clef MeasureClef) string {
	switch clef {
	case MeasureClefBass:
		return "bass"
	case MeasureClefTenor:
		return "tenor"
	case MeasureClefAlto:
		return "alto"
	case MeasureClefTreble:
		return "treble"
	default:
		return fmt.Sprintf("unknown:%d", clef)
	}
}

func goOctave(octave Octave) string {
	switch octave {
	case OctaveNone:
		return "none"
	case OctaveOttava:
		return "8va"
	case OctaveQuindicesima:
		return "15ma"
	case OctaveOttavaBassa:
		return "8vb"
	case OctaveQuindicesimaBassa:
		return "15mb"
	default:
		return fmt.Sprintf("unknown:%d", octave)
	}
}

func TestGoClefNormalizationContract(t *testing.T) {
	tests := []struct {
		clef MeasureClef
		want string
	}{
		{MeasureClefTreble, "treble"},
		{MeasureClefBass, "bass"},
		{MeasureClefTenor, "tenor"},
		{MeasureClefAlto, "alto"},
	}
	for _, test := range tests {
		if got := goClef(test.clef); got != test.want {
			t.Errorf("goClef(%d) = %q, want %q", test.clef, got, test.want)
		}
	}
}

func goAccent(effect NoteEffect) string {
	if effect.HeavyAccentuatedNote {
		return "heavy"
	}
	if effect.AccentuatedNote {
		return "normal"
	}
	return "none"
}

func goVibrato(vibrato bool) string {
	if vibrato {
		return "slight"
	}
	return "none"
}

func goHarmonicKind(kind HarmonicType) string {
	switch kind {
	case HarmonicTypeNatural:
		return "natural"
	case HarmonicTypeArtificial:
		return "artificial"
	case HarmonicTypeTapped:
		return "tap"
	case HarmonicTypePinch:
		return "pinch"
	case HarmonicTypeSemi:
		return "semi"
	case 0:
		return "none"
	default:
		return fmt.Sprintf("unknown:%d", kind)
	}
}

func goGraceTransition(transition GraceEffectTransition) string {
	switch transition {
	case GraceEffectTransitionSlide:
		return "slide"
	case GraceEffectTransitionBend:
		return "bend"
	case GraceEffectTransitionHammer:
		return "hammer"
	case GraceEffectTransitionNone:
		return "none"
	default:
		return fmt.Sprintf("unknown:%d", transition)
	}
}

func goSlide(slide SlideType) string {
	switch slide {
	case SlideIntoFromAbove:
		return "into-from-above"
	case SlideIntoFromBelow:
		return "into-from-below"
	case SlideShiftSlideTo:
		return "shift"
	case SlideLegatoSlideTo:
		return "legato"
	case SlideOutDownwards:
		return "out-down"
	case SlideOutUpwards:
		return "out-up"
	case SlideNone:
		return "none"
	default:
		return fmt.Sprintf("unknown:%d", slide)
	}
}

func normalizeGoRepeatCount(repeatClose int8) int {
	if repeatClose < 0 {
		return 0
	}
	return int(repeatClose) + 1
}

func conformanceExportSong() *Song {
	song := syntheticGP8Song()
	track := &song.Tracks[0]
	track.Name = "Guitar"
	track.Offset = 2
	track.PercussionTrack = false
	track.Strings = []GuitarString{{Number: 1, Value: 64}, {Number: 2, Value: 59}, {Number: 3, Value: 55}, {Number: 4, Value: 50}, {Number: 5, Value: 45}, {Number: 6, Value: 40}}
	song.Channels[0].Channel = 0
	song.Channels[0].EffectChannel = 1
	song.Channels[0].Instrument = 25
	for measureIndex := range track.Measures {
		for voiceIndex := range track.Measures[measureIndex].Voices {
			for beatIndex := range track.Measures[measureIndex].Voices[voiceIndex].Beats {
				beat := &track.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex]
				for noteIndex := range beat.Notes {
					note := &beat.Notes[noteIndex]
					note.String = int8(noteIndex + 1)
					note.Value = int16(3 + noteIndex*2)
					for graceIndex := range note.Effect.Graces {
						note.Effect.Graces[graceIndex].Fret = 2
					}
				}
			}
		}
	}
	firstBeat := &track.Measures[0].Voices[0].Beats[0]
	firstBeat.Effect.Hairpin = HairpinDiminuendo
	harmonicFret := 2.4
	firstBeat.Notes[0].Effect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeArtificial, FretFloat: &harmonicFret}
	return song
}

func conformanceExportCase(t *testing.T, id string) *Song {
	t.Helper()
	switch id {
	case "gp8-semantic-export":
		return conformanceExportSong()
	case "gp8-percussion-export":
		return parseTestFixture(t, "testdata/gp7/percussion.gp")
	case "gp8-multi-staff-export":
		return parseTestFixture(t, "testdata/gp7/grand-staff.gp")
	default:
		t.Fatalf("export conformance case %q has no source builder", id)
		return nil
	}
}

func writeConformanceFixture(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "score.gp")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func conformanceGPIFArchive(t *testing.T, gpif string) []byte {
	t.Helper()
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	entry, err := writer.Create("Content/score.gpif")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write([]byte(gpif)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func rewriteConformanceGPIF(t interface {
	Helper()
	Fatal(args ...any)
}, data []byte, mutate func(string) string) []byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	writer := zip.NewWriter(&output)
	for _, file := range reader.File {
		input, openErr := file.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		contents, readErr := io.ReadAll(input)
		closeErr := input.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		if file.Name == "Content/score.gpif" {
			contents = []byte(mutate(string(contents)))
		}
		header := file.FileHeader
		header.Method = zip.Deflate
		entry, createErr := writer.CreateHeader(&header)
		if createErr != nil {
			t.Fatal(createErr)
		}
		if _, writeErr := entry.Write(contents); writeErr != nil {
			t.Fatal(writeErr)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func conformanceFixtureDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}
