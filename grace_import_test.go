// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"os"
	"strings"
	"testing"
)

func TestParseLegacyGraceTransitionWireOrder(t *testing.T) {
	for _, fixture := range []string{"testdata/gp3/grace.gp3", "testdata/gp4/grace.gp4"} {
		t.Run(fixture, func(t *testing.T) {
			song := parseTestFixture(t, fixture)
			beats := song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats
			if len(beats) < 2 || len(beats[0].Notes) == 0 || len(beats[1].Notes) == 0 {
				t.Fatalf("opening beats = %#v", beats)
			}
			first := beats[0].Notes[0].Effect.Graces
			second := beats[1].Notes[0].Effect.Graces
			if len(first) != 1 || first[0].Transition != GraceEffectTransitionNone || first[0].Duration != DurationThirtySecond {
				t.Fatalf("first grace = %#v, want transition none and thirty-second duration", first)
			}
			if len(second) != 1 || second[0].Transition != GraceEffectTransitionSlide || second[0].Duration != DurationSixteenth {
				t.Fatalf("second grace = %#v, want slide transition and sixteenth duration", second)
			}
		})
	}
}

func TestGPIFKeepsOrphanGraceAtBeatLevel(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/grace-notes-advanced.gp")
	orphanCount := 0
	for trackIndex := range song.Tracks {
		for staffIndex := range song.Tracks[trackIndex].Staves {
			for measureIndex := range song.Tracks[trackIndex].Staves[staffIndex].Measures {
				for voiceIndex := range song.Tracks[trackIndex].Staves[staffIndex].Measures[measureIndex].Voices {
					for beatIndex := range song.Tracks[trackIndex].Staves[staffIndex].Measures[measureIndex].Voices[voiceIndex].Beats {
						beat := &song.Tracks[trackIndex].Staves[staffIndex].Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex]
						if !beat.isGrace {
							continue
						}
						orphanCount++
						for noteIndex := range beat.Notes {
							if len(beat.Notes[noteIndex].Effect.Graces) != 0 {
								t.Fatalf("orphan beat note contains an attached grace: track %d staff %d measure %d voice %d beat %d note %d = %#v", trackIndex, staffIndex, measureIndex, voiceIndex, beatIndex, noteIndex, beat.Notes[noteIndex].Effect.Graces)
							}
						}
					}
				}
			}
		}
	}
	if orphanCount != 5 {
		t.Fatalf("orphan grace beats = %d, want 5", orphanCount)
	}
}

func TestGP8ReexportsOrphanGraceAtBeatLevel(t *testing.T) {
	song := conformanceTechniqueSong(t)
	song.Tracks[0].Settings.Notation = true
	graceBeat := defaultBeat()
	graceBeat.Duration.Value = uint16(DurationThirtySecond)
	graceBeat.Notes = []Note{{Value: 2, String: 1, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1, Effect: defaultNoteEffect()}}
	orphans := gpifApplyPendingGrace(nil, []gpifPendingGrace{{beat: graceBeat}}, false, nil)
	if len(orphans) != 1 {
		t.Fatalf("orphan construction = %#v", orphans)
	}
	song.Tracks[0].Measures[0].Voices[0].Beats = orphans
	song.Tracks[0].Staves[0].Measures = song.Tracks[0].Measures
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 || len(data) == 0 {
		t.Fatalf("strict orphan export = %d bytes, %#v, %v", len(data), report.Entries, err)
	}
	gpif := string(readCurveGPIF(t, data))
	if count := strings.Count(gpif, "<GraceNotes>BeforeBeat</GraceNotes>"); count != 1 {
		t.Fatalf("exported orphan grace markers = %d, want 1", count)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip.Tracks[0].Staves[0].Measures[0].Voices[0].Beats
	if len(got) != 1 || !got[0].isGrace || len(got[0].Notes) != 1 || len(got[0].Notes[0].Effect.Graces) != 0 {
		t.Fatalf("round-trip orphan beats = %#v", got)
	}
	if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
		alpha := readAlphaTabScore(t, writeConformanceFixture(t, data))
		facts := collectConformanceFacts(alpha, map[string]bool{"graceRole": true}, func(value any) bool { return value == "orphan" })
		if len(facts) != 1 {
			t.Fatalf("AlphaTab orphan grace roles = %#v, want one", facts)
		}
	}
}

func TestAlphaTabGraceStatusCorpusConformance(t *testing.T) {
	requireAlphaTabConformance(t)
	tests := []struct {
		fixture  string
		features []string
	}{
		{fixture: "testdata/gp3/grace.gp3", features: []string{"grace-relationships"}},
		{fixture: "testdata/gp4/grace.gp4", features: []string{"grace-relationships"}},
		{fixture: "testdata/gp7/colors.gp", features: []string{"rhythm"}},
		{fixture: "testdata/gp7/dead-slap.gp", features: []string{"note-and-beat-semantics"}},
		{fixture: "testdata/gp7/grace-notes-advanced.gp", features: []string{"grace-relationships", "note-and-beat-semantics"}},
	}
	paths := make([]string, len(tests))
	for index := range tests {
		paths[index] = tests[index].fixture
	}
	alphaScores, err := readCorpusOracleScores(paths)
	if err != nil {
		t.Fatal(err)
	}
	var differences []string
	for _, test := range tests {
		goFeatures := selectConformanceFeatures(normalizeGoScore(parseTestFixture(t, test.fixture)), test.features)
		alphaFeatures := selectConformanceFeatures(alphaScores[test.fixture], test.features)
		for _, difference := range semanticDifferences(goFeatures, alphaFeatures) {
			semanticPath := conformanceCorpusSemanticPath(difference.Path, goFeatures, alphaFeatures)
			if strings.Contains(semanticPath, "/graces") || strings.HasSuffix(semanticPath, "/status") {
				differences = append(differences, test.fixture+":"+semanticPath)
			}
		}
	}
	if len(differences) > 0 {
		t.Fatalf("found %d grace/status differences:\n%s", len(differences), strings.Join(differences, "\n"))
	}
}
