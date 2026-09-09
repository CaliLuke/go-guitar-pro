// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"reflect"
	"sort"
	"testing"
)

type semanticMatrixRun struct {
	t          *testing.T
	assertions map[string]int
}

func newSemanticMatrixRun(t *testing.T) *semanticMatrixRun {
	t.Helper()
	return &semanticMatrixRun{t: t, assertions: make(map[string]int)}
}

func (run *semanticMatrixRun) Field(path string, got, want any) {
	run.equal("field", path, got, want)
}

func (run *semanticMatrixRun) Wire(path string, got, want any) {
	run.equal("wire", path, got, want)
}

func (run *semanticMatrixRun) Dispatch(path string, got, want any) {
	run.equal("dispatch", path, got, want)
}

func (run *semanticMatrixRun) equal(kind, path string, got, want any) {
	run.t.Helper()
	key := kind + ":" + path
	run.assertions[key]++
	if !reflect.DeepEqual(got, want) {
		run.t.Errorf("%s = %#v, want %#v", key, got, want)
	}
}

func (run *semanticMatrixRun) constructs() []string {
	result := make([]string, 0, len(run.assertions))
	for construct, count := range run.assertions {
		if count == 0 {
			panic(fmt.Sprintf("semantic assertion %s did not execute", construct))
		}
		result = append(result, construct)
	}
	sort.Strings(result)
	return result
}

var semanticMatrixExecutors = map[string]func(*semanticMatrixRun){
	"TestSemanticMatrixM01MetadataImport":                     runSemanticMatrixM01MetadataImport,
	"TestSemanticMatrixM01MetadataCorpus":                     runSemanticMatrixM01MetadataCorpus,
	"TestSemanticMatrixM01MetadataExportPolicy":               runSemanticMatrixM01MetadataExportPolicy,
	"TestSemanticMatrixM01BinaryClipboard":                    runSemanticMatrixM01BinaryClipboard,
	"TestSemanticMatrixM01OracleKeepsAuthorAndWriterDistinct": runSemanticMatrixM01OracleKeepsAuthorAndWriterDistinct,
}
