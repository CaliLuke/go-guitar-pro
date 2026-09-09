// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
)

type m23CorpusSnapshot struct {
	SchemaVersion int                 `json:"schemaVersion"`
	Oracle        string              `json:"oracle"`
	Fixtures      []m23FixtureReceipt `json:"fixtures"`
}

type m23FixtureReceipt struct {
	Path        string                 `json:"path"`
	Features    []string               `json:"features"`
	Diagnostics []m23DiagnosticReceipt `json:"diagnostics"`
	Differences []m23DifferenceReceipt `json:"differences"`
}

type m23DiagnosticReceipt struct {
	Code         string              `json:"code"`
	Kind         ParseDiagnosticKind `json:"kind"`
	Count        int                 `json:"count"`
	SourcePath   string              `json:"sourcePath,omitempty"`
	ObjectID     string              `json:"objectId,omitempty"`
	BinaryOffset *int64              `json:"binaryOffset,omitempty"`
	LocationsSHA string              `json:"locationsSha256"`
}

type m23DifferenceReceipt struct {
	semanticDifference
	SemanticPath string `json:"semanticPath"`
	Reason       string `json:"reason"`
	Issues       []int  `json:"issues"`
}

type m23OracleResult struct {
	Fixture string `json:"fixture"`
	Score   any    `json:"score"`
}

var m23CorpusRun struct {
	sync.Once
	err error
}

func TestSemanticMatrixM23WholeCorpusAccounting(t *testing.T) {
	runSemanticMatrixM23WholeCorpusAccounting(newSemanticMatrixRun(t))
}

func runSemanticMatrixM23WholeCorpusAccounting(run *semanticMatrixRun) {
	requireAlphaTabConformance(run.t)
	m23CorpusRun.Do(func() {
		m23CorpusRun.err = verifyM23CorpusSnapshot(run.t)
	})
	if m23CorpusRun.err != nil {
		run.t.Fatal(m23CorpusRun.err)
	}
}

func verifyM23CorpusSnapshot(t *testing.T) error {
	t.Helper()
	path := "conformance/corpus-snapshot.json"
	classifications := readM23DifferenceClassifications(path)
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
	oracleScores, err := readM23OracleScores(paths)
	if err != nil {
		return err
	}
	actual := m23CorpusSnapshot{SchemaVersion: 2, Oracle: "@coderline/alphatab@1.8.4", Fixtures: make([]m23FixtureReceipt, 0, len(inventory.Fixtures))}
	for _, fixture := range inventory.Fixtures {
		data, readErr := os.ReadFile(fixture.Path)
		if readErr != nil {
			return readErr
		}
		result, parseErr := ParseWithOptions(data, ParseOptions{})
		if parseErr != nil {
			return fmt.Errorf("%s: %w", fixture.Path, parseErr)
		}
		receipt := m23FixtureReceipt{
			Path:        fixture.Path,
			Features:    fixture.Features,
			Diagnostics: m23Diagnostics(result.Diagnostics),
		}
		alphaScore, ok := oracleScores[fixture.Path]
		if !ok {
			return fmt.Errorf("%s has no AlphaTab batch result", fixture.Path)
		}
		goFeatures := selectConformanceFeatures(normalizeGoScore(result.Song), fixture.Features)
		alphaFeatures := selectConformanceFeatures(alphaScore, fixture.Features)
		differences := semanticDifferences(goFeatures, alphaFeatures)
		for _, difference := range differences {
			difference.Feature = conformanceFeature(difference.Path)
			if difference.Feature == "" {
				return fmt.Errorf("%s has unclassified semantic difference %s", fixture.Path, difference.Path)
			}
			receiptDifference := m23DifferenceReceipt{
				semanticDifference: difference,
				SemanticPath:       m23SemanticPath(difference.Path, goFeatures, alphaFeatures),
			}
			if classification, ok := classifications[m23DifferenceKey(fixture.Path, receiptDifference)]; ok {
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
	var expected m23CorpusSnapshot
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

func m23SemanticPath(differencePath string, scores ...any) string {
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

func readM23DifferenceClassifications(path string) map[string]m23DifferenceReceipt {
	classifications := make(map[string]m23DifferenceReceipt)
	data, err := os.ReadFile(path)
	if err != nil {
		return classifications
	}
	var snapshot m23CorpusSnapshot
	if json.Unmarshal(data, &snapshot) != nil {
		return classifications
	}
	for _, fixture := range snapshot.Fixtures {
		for _, difference := range fixture.Differences {
			if difference.Reason != "" && len(difference.Issues) > 0 {
				classifications[m23DifferenceKey(fixture.Path, difference)] = difference
			}
		}
	}
	return classifications
}

func m23DifferenceKey(fixture string, difference m23DifferenceReceipt) string {
	encoded, _ := json.Marshal(struct {
		semanticDifference
		SemanticPath string `json:"semanticPath"`
	}{semanticDifference: difference.semanticDifference, SemanticPath: difference.SemanticPath})
	return fixture + "\x00" + string(encoded)
}

func readM23OracleScores(paths []string) (map[string]any, error) {
	args := append([]string{"conformance/oracle.mjs", "--batch"}, paths...)
	output, err := exec.Command("node", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("AlphaTab batch oracle: %w\n%s", err, output)
	}
	var results []m23OracleResult
	if err := json.Unmarshal(output, &results); err != nil {
		return nil, fmt.Errorf("decode AlphaTab batch oracle: %w", err)
	}
	scores := make(map[string]any, len(results))
	for _, result := range results {
		if _, duplicate := scores[result.Fixture]; duplicate {
			return nil, fmt.Errorf("duplicate AlphaTab batch result for %s", result.Fixture)
		}
		scores[result.Fixture] = result.Score
	}
	return scores, nil
}

func m23Diagnostics(diagnostics []ParseDiagnostic) []m23DiagnosticReceipt {
	bySignature := make(map[string]*m23DiagnosticReceipt)
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
			receipt = &m23DiagnosticReceipt{
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
	receipts := make([]m23DiagnosticReceipt, 0, len(signatures))
	for _, signature := range signatures {
		sort.Strings(locationsBySignature[signature])
		digest := sha256.Sum256([]byte(strings.Join(locationsBySignature[signature], "\x00")))
		bySignature[signature].LocationsSHA = fmt.Sprintf("%x", digest)
		receipts = append(receipts, *bySignature[signature])
	}
	return receipts
}

func TestM23DiagnosticsAccountsForEveryLocation(t *testing.T) {
	first := []ParseDiagnostic{
		{Code: "probe", Kind: ParseDiagnosticInvalidData, SourcePath: "/a"},
		{Code: "probe", Kind: ParseDiagnosticInvalidData, SourcePath: "/b"},
	}
	second := []ParseDiagnostic{
		{Code: "probe", Kind: ParseDiagnosticInvalidData, SourcePath: "/a"},
		{Code: "probe", Kind: ParseDiagnosticInvalidData, SourcePath: "/c"},
	}
	if reflect.DeepEqual(m23Diagnostics(first), m23Diagnostics(second)) {
		t.Fatal("counted diagnostics hid a non-representative location change")
	}
}
