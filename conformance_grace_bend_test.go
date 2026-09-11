// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"os"
	"reflect"
	"slices"
	"testing"
)

func TestConformanceGraceBendPolicy(t *testing.T) {
	runConformanceGraceBendPolicy(newConformanceRun(t))
}
func runConformanceGraceBendPolicy(run *conformanceRun) {
	t := run.t
	source, err := os.ReadFile("testdata/gp4/fade-to-black.gp4")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprintf("%x", sha256.Sum256(source)) != "51712a4e61b017da00b4cb37d6c6382cf6ef97ff0ed9f477de61756c20cb05f6" || !bytes.Equal(source[70913:70917], []byte{0x16, 0x06, 0x02, 0x02}) {
		t.Fatal("legacy grace source evidence changed")
	}
	parsed, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	g := parsed.Tracks[6].Measures[201].Voices[0].Beats[4].Notes[0].Effect.Graces[0]
	run.Normalized("GraceEffect.Transition", g.Transition, GraceEffectTransitionBend)
	run.Normalized("GraceEffect.Fret", g.Fret, int8(22))
	run.Preserved("GraceEffect.RawFret", *g.RawFret, int8(22))
	run.Preserved("GraceEffect.Duration", g.Duration, uint8(32))
	run.Preserved("GraceEffect.Velocity", g.Velocity, int16(95))
	run.Preserved("GraceEffect.IsOnBeat", g.IsOnBeat, false)
	for _, onBeat := range []bool{false, true} {
		song := conformanceGraceBendSong(t, onBeat)
		data, report := assertConsumerLossPolicy(t, song, []string{"gp8.omit.grace-bend-transition"})
		run.Report("M15-GRACE-BEND-POLICY", reportCodes(report), []string{"gp8.omit.grace-bend-transition"})
		if len(report.Entries) != 1 || report.Entries[0].Disposition != ExportDispositionOmitted || report.Entries[0].Location != (ScoreLocation{Note: 1}) {
			t.Fatalf("grace omission report %#v", report)
		}
		wire := extractGraceWire(t, data)
		wantPlacement := "BeforeBeat"
		if onBeat {
			wantPlacement = "OnBeat"
		}
		run.Wire("gpifBeat.GraceNotes", wire.graceKinds, []string{wantPlacement})
		run.Wire("gpifBeat.Notes", wire.graceNoteCounts, []int{1})
		run.Wire("gpifRhythm.NoteValue", wire.graceNoteValues, []string{"32nd"})
		got, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		notes := got.Tracks[0].Measures[0].Voices[0].Beats[0].Notes
		if len(notes) != 2 || notes[1].String != 2 || notes[1].Value != 5 || len(notes[0].Effect.Graces) != 0 || len(notes[1].Effect.Graces) != 1 || notes[1].Effect.Bend != nil {
			t.Fatalf("grace owner %#v", notes)
		}
		grace := notes[1].Effect.Graces[0]
		run.Field("GraceEffect.Transition", grace.Transition, GraceEffectTransitionNone)
		run.Preserved("GraceEffect.ExactFret", *grace.ExactFret, Fret(2))
		run.Field("GraceEffect.Duration", grace.Duration, uint8(32))
		run.Field("GraceEffect.IsOnBeat", grace.IsOnBeat, onBeat)
		run.Field("GraceEffect.Velocity", grace.Velocity, Forte)
	}
}
func conformanceGraceBendSong(t *testing.T, onBeat bool) *Song {
	t.Helper()
	song := semanticExportProbeSong(t)
	song.Tracks[0].Measures[0].Voices = []Voice{{Beats: []Beat{{Duration: defaultDuration(), Status: BeatStatusNormal, Dynamics: Forte, Notes: []Note{
		conformanceGraceNote(7, 1, nil), conformanceGraceNote(5, 2, []GraceEffect{conformanceGrace(2, DurationThirtySecond, Forte, onBeat, 0, GraceEffectTransitionBend, false)}),
	}}}}}
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if ds := ValidateSong(song); len(ds) != 0 {
		t.Fatalf("invalid grace score %#v", ds)
	}
	return song
}
func TestAlphaTabGraceBendPolicy(t *testing.T) {
	var source []curveGraceFact
	readAlphaTabOracleFacts(t, "--curve-grace", "testdata/gp4/fade-to-black.gp4", &source)
	source = slices.DeleteFunc(source, func(f curveGraceFact) bool {
		return f.Track != 6 || f.Bar != 201 || f.Voice != 0 || f.Beat < 4 || f.Beat > 5
	})
	want := []curveGraceFact{
		{Track: 6, Bar: 201, Beat: 4, GraceType: 2, String: 6, Fret: 22, Duration: 8, Dynamic: 5, Bend: []retainedBendControl{}},
		{Track: 6, Bar: 201, Beat: 5, String: 6, Fret: 22, Duration: 8, Dynamic: 5, Bend: []retainedBendControl{}},
	}
	if !reflect.DeepEqual(source, want) {
		t.Fatalf("legacy consumer importer limit %#v want %#v", source, want)
	}
	for _, onBeat := range []bool{false, true} {
		data, _ := assertConsumerLossPolicy(t, conformanceGraceBendSong(t, onBeat), []string{"gp8.omit.grace-bend-transition"})
		var output []curveGraceFact
		readAlphaTabOracleFacts(t, "--curve-grace", writeConformanceFixture(t, data), &output)
		graceType := 2
		if onBeat {
			graceType = 1
		}
		want = []curveGraceFact{
			{GraceType: graceType, String: 5, Fret: 2, Duration: 8, Dynamic: 5, Bend: []retainedBendControl{}},
			{Beat: 1, String: 6, Fret: 7, Duration: 4, Dynamic: 5, Bend: []retainedBendControl{}},
			{Beat: 1, Note: 1, String: 5, Fret: 5, Duration: 4, Dynamic: 5, Bend: []retainedBendControl{}},
		}
		if !reflect.DeepEqual(output, want) {
			t.Fatalf("allowed consumer grace %#v want %#v", output, want)
		}
	}
}
