// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"os"
	"slices"
	"strings"
	"testing"
)

func TestParseGPIFPreservesSimileMarks(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/simile-mark.gp")
	got := make([]SimileMark, len(song.Tracks[0].Measures))
	for index := range song.Tracks[0].Measures {
		got[index] = song.Tracks[0].Measures[index].SimileMark
	}
	want := []SimileMark{
		SimileMarkNone,
		SimileMarkSimple,
		SimileMarkNone,
		SimileMarkNone,
		SimileMarkFirstOfDouble,
		SimileMarkSecondOfDouble,
		SimileMarkNone,
		SimileMarkSimple,
		SimileMarkFirstOfDouble,
		SimileMarkSecondOfDouble,
		SimileMarkNone,
		SimileMarkSimple,
		SimileMarkNone,
		SimileMarkNone,
		SimileMarkNone,
		SimileMarkFirstOfDouble,
		SimileMarkSecondOfDouble,
		SimileMarkNone,
		SimileMarkNone,
	}
	if !slices.Equal(got, want) {
		t.Fatalf("simile marks = %v, want %v", got, want)
	}
}

func TestParseGPIFRejectsUnknownSimileMarkInStrictMode(t *testing.T) {
	source, err := os.ReadFile("testdata/gp7/simile-mark.gp")
	if err != nil {
		t.Fatal(err)
	}
	data := rewriteConformanceGPIF(t, source, func(gpif string) string {
		return strings.Replace(gpif, "<SimileMark>Simple</SimileMark>", "<SimileMark>Triple</SimileMark>", 1)
	})
	result, err := ParseWithOptions(data, ParseOptions{Strict: true})
	if err == nil {
		t.Fatal("strict parse accepted an unknown simile mark")
	}
	if result == nil || !slices.ContainsFunc(result.Diagnostics, func(diagnostic ParseDiagnostic) bool {
		return diagnostic.Code == "GPIF.Bar.SimileMark.InvalidValue" && diagnostic.SourcePath == `/GPIF/Bars/Bar[@id="1"]/SimileMark`
	}) {
		t.Fatalf("diagnostics = %#v, want invalid simile-mark value", result)
	}
}
