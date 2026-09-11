// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
)

type conformanceCorpusSnapshot struct {
	SchemaVersion int                               `json:"schemaVersion"`
	Oracle        string                            `json:"oracle"`
	Fixtures      []conformanceCorpusFixtureReceipt `json:"fixtures"`
}

type conformanceCorpusFixtureReceipt struct {
	Path          string                               `json:"path"`
	Features      []string                             `json:"features"`
	Diagnostics   []conformanceCorpusDiagnosticReceipt `json:"diagnostics"`
	Differences   []conformanceCorpusDifferenceReceipt `json:"differences"`
	ImportFailure *conformanceCorpusImportFailure      `json:"importFailure,omitempty"`
}

type conformanceCorpusErrorReceipt struct {
	Type       string `json:"type"`
	Message    string `json:"message"`
	MessageHex string `json:"messageHex,omitempty"`
}

type conformanceCorpusImportFailure struct {
	GoError       *conformanceCorpusErrorReceipt `json:"goError"`
	AlphaTabError *conformanceCorpusErrorReceipt `json:"alphaTabError"`
}

type conformanceCorpusDiagnosticReceipt struct {
	Code         string              `json:"code"`
	Kind         ParseDiagnosticKind `json:"kind"`
	Count        int                 `json:"count"`
	SourcePath   string              `json:"sourcePath,omitempty"`
	ObjectID     string              `json:"objectId,omitempty"`
	BinaryOffset *int64              `json:"binaryOffset,omitempty"`
	LocationsSHA string              `json:"locationsSha256"`
}

type conformanceCorpusDifferenceReceipt struct {
	semanticDifference
	SemanticPath string `json:"semanticPath"`
	Reason       string `json:"reason"`
	Issues       []int  `json:"issues"`
}

type conformanceCorpusOracleResult struct {
	Fixture string                         `json:"fixture"`
	Score   any                            `json:"score"`
	Error   *conformanceCorpusErrorReceipt `json:"error"`
}

var conformanceCorpusRun struct {
	sync.Once
	err error
}

func TestConformanceWholeCorpusAccounting(t *testing.T) {
	runConformanceWholeCorpusAccounting(newConformanceRun(t))
}

func runConformanceWholeCorpusAccounting(run *conformanceRun) {
	requireAlphaTabConformance(run.t)
	conformanceCorpusRun.Do(func() {
		conformanceCorpusRun.err = verifyCorpusSnapshot(run.t)
	})
	if conformanceCorpusRun.err != nil {
		run.t.Fatal(conformanceCorpusRun.err)
	}
}

func verifyCorpusSnapshot(t *testing.T) error {
	t.Helper()
	path := "conformance/corpus-snapshot.json"
	classifications := readCorpusDifferenceClassifications(path)
	inventoryData, err := os.ReadFile("conformance/fixture-inventory.json")
	if err != nil {
		return err
	}
	var inventory fixtureInventory
	if unmarshalErr := json.Unmarshal(inventoryData, &inventory); unmarshalErr != nil {
		return unmarshalErr
	}
	paths := make([]string, 0, len(inventory.Fixtures))
	for _, fixture := range inventory.Fixtures {
		paths = append(paths, fixture.Path)
	}
	oracleScores, err := readCorpusOracleReceipts(paths)
	if err != nil {
		return err
	}
	actual := conformanceCorpusSnapshot{SchemaVersion: 2, Oracle: "@coderline/alphatab@1.8.4", Fixtures: make([]conformanceCorpusFixtureReceipt, 0, len(inventory.Fixtures))}
	for _, fixture := range inventory.Fixtures {
		data, readErr := os.ReadFile(fixture.Path)
		if readErr != nil {
			return readErr
		}
		result, parseErr := ParseWithOptions(data, ParseOptions{})
		alphaResult, ok := oracleScores[fixture.Path]
		if !ok {
			return fmt.Errorf("%s has no AlphaTab batch result", fixture.Path)
		}
		receipt := conformanceCorpusFixtureReceipt{Path: fixture.Path, Features: fixture.Features}
		if parseErr != nil || alphaResult.Error != nil {
			if fixture.Disposition != "unsupported" {
				return fmt.Errorf("%s has an unclassified import failure: Go=%w AlphaTab=%v", fixture.Path, parseErr, alphaResult.Error)
			}
			if result != nil || alphaResult.Score != nil {
				return fmt.Errorf("%s import failure unexpectedly returned a score", fixture.Path)
			}
			receipt.Diagnostics = []conformanceCorpusDiagnosticReceipt{}
			receipt.ImportFailure = &conformanceCorpusImportFailure{AlphaTabError: alphaResult.Error}
			if parseErr != nil {
				receipt.ImportFailure.GoError = &conformanceCorpusErrorReceipt{Type: fmt.Sprintf("%T", parseErr), Message: string([]rune(parseErr.Error())), MessageHex: fmt.Sprintf("%x", parseErr.Error())}
			}
			actual.Fixtures = append(actual.Fixtures, receipt)
			continue
		}
		receipt.Diagnostics = conformanceCorpusDiagnostics(result.Diagnostics)
		alphaScore := alphaResult.Score
		goFeatures := selectConformanceFeatures(normalizeGoScore(result.Song), fixture.Features)
		alphaFeatures := selectConformanceFeatures(alphaScore, fixture.Features)
		goFeatures = conformanceCorpusApplyOracleLimitations(fixture.Path, goFeatures)
		alphaFeatures = conformanceCorpusApplyOracleLimitations(fixture.Path, alphaFeatures)
		differences := semanticDifferences(goFeatures, alphaFeatures)
		for _, difference := range differences {
			difference.Feature = conformanceFeature(difference.Path)
			if difference.Feature == "" {
				return fmt.Errorf("%s has unclassified semantic difference %s", fixture.Path, difference.Path)
			}
			receiptDifference := conformanceCorpusDifferenceReceipt{
				semanticDifference: difference,
				SemanticPath:       conformanceCorpusSemanticPath(difference.Path, goFeatures, alphaFeatures),
			}
			if classification, ok := classifications[conformanceCorpusDifferenceKey(fixture.Path, receiptDifference)]; ok {
				receiptDifference.Reason = classification.Reason
				receiptDifference.Issues = classification.Issues
			}
			receipt.Differences = append(receipt.Differences, receiptDifference)
		}
		actual.Fixtures = append(actual.Fixtures, receipt)
	}

	if os.Getenv("ALPHATAB_CORPUS_UPDATE") == "1" {
		encoded, marshalErr := json.MarshalIndent(actual, "", "  ")
		if marshalErr != nil {
			return marshalErr
		}
		return os.WriteFile(path, append(encoded, '\n'), 0o644)
	}
	expectedData, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read whole-corpus semantic snapshot: %w", err)
	}
	var expected conformanceCorpusSnapshot
	if err := json.Unmarshal(expectedData, &expected); err != nil {
		return err
	}
	if expected.SchemaVersion != actual.SchemaVersion || expected.Oracle != actual.Oracle || len(expected.Fixtures) != len(actual.Fixtures) {
		return fmt.Errorf("whole-corpus snapshot header or fixture count changed")
	}
	for index := range actual.Fixtures {
		want, got := expected.Fixtures[index], actual.Fixtures[index]
		if len(want.Differences) != len(got.Differences) {
			return fmt.Errorf("whole-corpus semantic difference count changed for %s", got.Path)
		}
		for differenceIndex := range want.Differences {
			if want.Differences[differenceIndex].Reason == "" || len(want.Differences[differenceIndex].Issues) == 0 {
				return fmt.Errorf("%s difference %s lacks a narrow reason and open issue", want.Path, want.Differences[differenceIndex].Path)
			}
			got.Differences[differenceIndex].Reason = want.Differences[differenceIndex].Reason
			got.Differences[differenceIndex].Issues = want.Differences[differenceIndex].Issues
		}
		if !reflect.DeepEqual(got, want) {
			return fmt.Errorf("whole-corpus semantic receipt changed for %s", got.Path)
		}
	}
	return nil
}

// conformanceCorpusApplyOracleLimitations removes only facts that the pinned independent
// consumer cannot expose. AlphaTab 1.8.4 does not expose GP3 beat whammy data;
// TestAlphaTabGP3WhammyOracleLimitation pins that behavior independently.
func conformanceCorpusApplyOracleLimitations(fixture string, features any) any {
	if fixture != "testdata/gp3/Effects.gp3" {
		return features
	}
	featureMap, ok := features.(map[string]any)
	if !ok {
		return features
	}
	facts, ok := featureMap["note-and-beat-semantics"].([]any)
	if !ok {
		return features
	}
	filtered := slices.DeleteFunc(slices.Clone(facts), func(value any) bool {
		fact, ok := value.(map[string]any)
		path, pathOK := fact["path"].(string)
		return ok && pathOK && conformanceCorpusLimitedGP3WhammyPath(path)
	})
	result := maps.Clone(featureMap)
	result["note-and-beat-semantics"] = filtered
	return result
}

func conformanceCorpusLimitedGP3WhammyPath(path string) bool {
	switch path {
	case "/tracks/0/staves/0/bars/8/voices/0/beats/0/whammy",
		"/tracks/0/staves/0/bars/9/voices/0/beats/0/whammy",
		"/tracks/0/staves/0/bars/10/voices/0/beats/0/whammy",
		"/tracks/0/staves/0/bars/11/voices/0/beats/0/whammy":
		return true
	default:
		return false
	}
}

func TestCorpusGP3WhammyOracleLimitationIsNarrow(t *testing.T) {
	probe := map[string]any{"note-and-beat-semantics": []any{
		map[string]any{"path": "/tracks/0/staves/0/bars/8/voices/0/beats/0/whammy", "value": []any{}},
		map[string]any{"path": "/tracks/0/staves/0/bars/12/voices/0/beats/0/whammy", "value": []any{}},
		map[string]any{"path": "/tracks/0/staves/0/bars/8/voices/0/beats/0/effects", "value": map[string]any{}},
	}}
	limited := conformanceCorpusApplyOracleLimitations("testdata/gp3/Effects.gp3", probe).(map[string]any)
	if got := limited["note-and-beat-semantics"].([]any); len(got) != 2 || !strings.Contains(got[0].(map[string]any)["path"].(string), "/bars/12/") || !strings.HasSuffix(got[1].(map[string]any)["path"].(string), "/effects") {
		t.Fatalf("GP3 oracle limitation = %#v, want only the four named whammy paths removed", got)
	}
	if got := conformanceCorpusApplyOracleLimitations("testdata/gp4/Effects.gp4", probe); !reflect.DeepEqual(got, probe) {
		t.Fatalf("GP4 facts changed by GP3-only limitation: %#v", got)
	}
	if facts := probe["note-and-beat-semantics"].([]any); len(facts) != 3 {
		t.Fatalf("oracle limitation mutated its input: %#v", facts)
	}
}

func conformanceCorpusSemanticPath(differencePath string, scores ...any) string {
	parts := strings.Split(strings.TrimPrefix(differencePath, "/"), "/")
	if len(parts) < 2 {
		return differencePath
	}
	index, err := strconv.Atoi(parts[1])
	if err != nil {
		return differencePath
	}
	for _, score := range scores {
		features, ok := score.(map[string]any)
		if !ok {
			continue
		}
		facts, ok := features[parts[0]].([]any)
		if !ok || index < 0 || index >= len(facts) {
			continue
		}
		fact, ok := facts[index].(map[string]any)
		if !ok {
			continue
		}
		if path, ok := fact["path"].(string); ok {
			if len(parts) > 3 {
				return path + "/" + strings.Join(parts[3:], "/")
			}
			return path
		}
	}
	return differencePath
}

func readCorpusDifferenceClassifications(path string) map[string]conformanceCorpusDifferenceReceipt {
	classifications := make(map[string]conformanceCorpusDifferenceReceipt)
	data, err := os.ReadFile(path)
	if err != nil {
		return classifications
	}
	var snapshot conformanceCorpusSnapshot
	if json.Unmarshal(data, &snapshot) != nil {
		return classifications
	}
	for _, fixture := range snapshot.Fixtures {
		for _, difference := range fixture.Differences {
			if difference.Reason != "" && len(difference.Issues) > 0 {
				classifications[conformanceCorpusDifferenceKey(fixture.Path, difference)] = difference
			}
		}
	}
	return classifications
}

func conformanceCorpusDifferenceKey(fixture string, difference conformanceCorpusDifferenceReceipt) string {
	encoded, _ := json.Marshal(struct {
		semanticDifference
		SemanticPath string `json:"semanticPath"`
	}{semanticDifference: difference.semanticDifference, SemanticPath: difference.SemanticPath})
	return fixture + "\x00" + string(encoded)
}

func readCorpusOracleReceipts(paths []string) (map[string]conformanceCorpusOracleResult, error) {
	args := append([]string{alphaTabOracleScript(), "--batch-receipts"}, paths...)
	output, err := exec.Command("node", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("AlphaTab batch oracle: %w\n%s", err, output)
	}
	var results []conformanceCorpusOracleResult
	if err := json.Unmarshal(output, &results); err != nil {
		return nil, fmt.Errorf("decode AlphaTab batch oracle: %w", err)
	}
	scores := make(map[string]conformanceCorpusOracleResult, len(results))
	for _, result := range results {
		if _, duplicate := scores[result.Fixture]; duplicate {
			return nil, fmt.Errorf("duplicate AlphaTab batch result for %s", result.Fixture)
		}
		scores[result.Fixture] = result
	}
	return scores, nil
}

func conformanceCorpusDiagnostics(diagnostics []ParseDiagnostic) []conformanceCorpusDiagnosticReceipt {
	bySignature := make(map[string]*conformanceCorpusDiagnosticReceipt)
	locationsBySignature := make(map[string][]string)
	for _, diagnostic := range diagnostics {
		signature := diagnostic.Code + "\x00" + string(diagnostic.Kind)
		location, _ := json.Marshal(struct {
			SourcePath   string        `json:"sourcePath,omitempty"`
			ObjectID     string        `json:"objectId,omitempty"`
			Location     ParseLocation `json:"location"`
			BinaryOffset *int64        `json:"binaryOffset,omitempty"`
		}{diagnostic.SourcePath, diagnostic.ObjectID, diagnostic.Location, diagnostic.BinaryOffset})
		locationsBySignature[signature] = append(locationsBySignature[signature], string(location))
		receipt := bySignature[signature]
		if receipt == nil {
			receipt = &conformanceCorpusDiagnosticReceipt{
				Code: diagnostic.Code, Kind: diagnostic.Kind, SourcePath: diagnostic.SourcePath,
				ObjectID: diagnostic.ObjectID, BinaryOffset: diagnostic.BinaryOffset,
			}
			bySignature[signature] = receipt
		}
		receipt.Count++
	}
	signatures := make([]string, 0, len(bySignature))
	for signature := range bySignature {
		signatures = append(signatures, signature)
	}
	sort.Strings(signatures)
	receipts := make([]conformanceCorpusDiagnosticReceipt, 0, len(signatures))
	for _, signature := range signatures {
		sort.Strings(locationsBySignature[signature])
		digest := sha256.Sum256([]byte(strings.Join(locationsBySignature[signature], "\x00")))
		bySignature[signature].LocationsSHA = fmt.Sprintf("%x", digest)
		receipts = append(receipts, *bySignature[signature])
	}
	return receipts
}

func TestCorpusDiagnosticsAccountsForEveryLocation(t *testing.T) {
	first := []ParseDiagnostic{
		{Code: "probe", Kind: ParseDiagnosticInvalidData, SourcePath: "/a"},
		{Code: "probe", Kind: ParseDiagnosticInvalidData, SourcePath: "/b"},
	}
	second := []ParseDiagnostic{
		{Code: "probe", Kind: ParseDiagnosticInvalidData, SourcePath: "/a"},
		{Code: "probe", Kind: ParseDiagnosticInvalidData, SourcePath: "/c"},
	}
	if reflect.DeepEqual(conformanceCorpusDiagnostics(first), conformanceCorpusDiagnostics(second)) {
		t.Fatal("counted diagnostics hid a non-representative location change")
	}
}

func readCorpusOracleScores(paths []string) (map[string]any, error) {
	receipts, err := readCorpusOracleReceipts(paths)
	if err != nil {
		return nil, err
	}
	scores := make(map[string]any, len(receipts))
	for path, receipt := range receipts {
		if receipt.Error != nil || receipt.Score == nil {
			return nil, fmt.Errorf("%s: expected an AlphaTab score, got import error %v", path, receipt.Error)
		}
		scores[path] = receipt.Score
	}
	return scores, nil
}
