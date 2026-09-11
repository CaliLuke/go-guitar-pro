// SPDX-License-Identifier: MIT
package integration_test

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestLegacyGraceBendSource(t *testing.T) {
	data, err := os.ReadFile("../testdata/gp4/fade-to-black.gp4")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data[70913:70917], []byte{0x16, 0x06, 0x02, 0x02}) {
		t.Fatal("source grace record changed")
	}
	song, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	g := song.Tracks[6].Measures[201].Voices[0].Beats[4].Notes[0].Effect.Graces[0]
	if g.Transition != gp.GraceEffectTransitionBend || g.Fret != 22 || g.RawFret == nil || *g.RawFret != 22 || g.Duration != 32 || g.Velocity != 95 || g.IsOnBeat {
		t.Fatalf("binary bend grace %#v", g)
	}
}
func TestGP8GraceBendLossPolicy(t *testing.T) {
	for _, onBeat := range []bool{false, true} {
		song := dynamicPolicySong(t, 47, false)
		beat := &song.Tracks[0].Measures[0].Voices[0].Beats[1]
		beat.Notes[1].Velocity = 47
		fret := gp.Fret(2)
		grace := gp.GraceEffect{Fret: 2, ExactFret: &fret, Duration: 32, Velocity: gp.Forte, IsOnBeat: onBeat, Transition: gp.GraceEffectTransitionBend}
		beat.Notes[1].Effect.Graces = []gp.GraceEffect{grace}
		if ds := gp.ValidateSong(song); len(ds) != 0 {
			t.Fatalf("invalid grace score %#v", ds)
		}
		for _, allowed := range [][]string{nil, {"gp8.normalize.grace-velocity"}, {"gp8.omit.grace-bend-transition"}} {
			options := gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}}
			preflight := gp.PreflightExport(song, gp.ExportFormatGP8, options)
			data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, options)
			if !reflect.DeepEqual(report, preflight) || len(report.Entries) != 1 || report.Entries[0].Code != "gp8.omit.grace-bend-transition" || report.Entries[0].Location != (gp.ScoreLocation{Beat: 1, Note: 1}) {
				t.Fatalf("grace report %#v", report)
			}
			if len(allowed) == 0 || allowed[0] != "gp8.omit.grace-bend-transition" {
				var loss *gp.ExportLossError
				if len(data) != 0 || !errors.As(err, &loss) {
					t.Fatalf("unallowed export %d %v", len(data), err)
				}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := gp.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			got := parsed.Tracks[0].Measures[0].Voices[0].Beats[1].Notes
			if len(got) != 2 || got[1].String != 2 || got[1].Value != 5 || len(got[0].Effect.Graces) != 0 || len(got[1].Effect.Graces) != 1 {
				t.Fatalf("grace ownership %#v", got)
			}
			actual := got[1].Effect.Graces[0]
			if actual.Transition != gp.GraceEffectTransitionNone || actual.ExactFret == nil || *actual.ExactFret != 2 || actual.IsOnBeat != onBeat || actual.Duration != 32 || actual.Velocity != 95 || actual.RawFret != nil || got[1].Effect.Bend != nil {
				t.Fatalf("allowed grace %#v", actual)
			}
		}
		if !reflect.DeepEqual(beat.Notes[1].Effect.Graces, []gp.GraceEffect{grace}) {
			t.Fatal("export mutated grace")
		}
	}
}
