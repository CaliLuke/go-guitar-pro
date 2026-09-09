// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"slices"
	"strings"
	"testing"
)

func TestParseCanonicalizesStandardBendCurves(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		bar     int
		beat    int
		want    []BendPoint
	}{
		{
			name:    "legacy three point bend",
			fixture: "testdata/gp3/bends.gp3",
			want:    []BendPoint{{Position: 0, Value: 0}, {Position: 12, Value: 4}},
		},
		{
			name:    "GPIF omitted destination offset",
			fixture: "testdata/gp6/bends.gpx",
			want:    []BendPoint{{Position: 0, Value: 0}, {Position: 12, Value: 4}},
		},
		{
			name:    "GPIF omitted middle offsets",
			fixture: "testdata/gp6/bends.gpx",
			bar:     1,
			want: []BendPoint{
				{Position: 0, Value: 0},
				{Position: 6, Value: 12},
				{Position: 6, Value: 12},
				{Position: 12, Value: 6},
			},
		},
		{
			name:    "GPIF four point simple bend",
			fixture: "testdata/gp7/bends.gp",
			want:    []BendPoint{{Position: 0, Value: 0}, {Position: 3, Value: 1}},
		},
		{
			name:    "GPIF four point bend release",
			fixture: "testdata/gp7/bends-advanced.gp",
			beat:    1,
			want: []BendPoint{
				{Position: 0, Value: 0},
				{Position: 2, Value: 4},
				{Position: 4, Value: 4},
				{Position: 6, Value: 0},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			song := parseTestFixture(t, test.fixture)
			bend := song.Tracks[0].Staves[0].Measures[test.bar].Voices[0].Beats[test.beat].Notes[0].Effect.Bend
			if bend == nil {
				t.Fatal("bend is nil")
			}
			if !slices.Equal(bend.Points, test.want) {
				t.Fatalf("bend points = %#v, want %#v", bend.Points, test.want)
			}
		})
	}
}

func TestAlphaTabBendCorpusConformance(t *testing.T) {
	requireAlphaTabConformance(t)
	fixtures := []string{
		"testdata/gp3/Effects.gp3",
		"testdata/gp3/bends.gp3",
		"testdata/gp4/Effects.gp4",
		"testdata/gp4/bends.gp4",
		"testdata/gp5/Effects.gp5",
		"testdata/gp5/bends.gp5",
		"testdata/gp6/bends.gpx",
		"testdata/gp6/effects.gpx",
		"testdata/gp7/bend-vibrato.gp",
		"testdata/gp7/bends-advanced.gp",
		"testdata/gp7/bends.gp",
		"testdata/gp7/effects.gp",
		"testdata/gp7/tied-bend-hammer.gp",
		"testdata/gp7/tremolo-vibrato.gp",
	}
	alphaScores, err := readM23OracleScores(fixtures)
	if err != nil {
		t.Fatal(err)
	}

	var bendDifferences []string
	for _, fixture := range fixtures {
		goFeatures := selectConformanceFeatures(normalizeGoScore(parseTestFixture(t, fixture)), []string{"note-and-beat-semantics"})
		alphaFeatures := selectConformanceFeatures(alphaScores[fixture], []string{"note-and-beat-semantics"})
		for _, difference := range semanticDifferences(goFeatures, alphaFeatures) {
			semanticPath := m23SemanticPath(difference.Path, goFeatures, alphaFeatures)
			if strings.Contains(semanticPath, "/effects/bend") {
				bendDifferences = append(bendDifferences, fixture+":"+semanticPath)
			}
		}
	}
	if len(bendDifferences) > 0 {
		t.Fatalf("found %d bend differences:\n%s", len(bendDifferences), strings.Join(bendDifferences, "\n"))
	}
}
