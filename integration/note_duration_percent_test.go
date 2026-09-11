// SPDX-License-Identifier: MIT

package integration_test

import (
	"math"
	"reflect"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestGP8NoteDurationPercentagePolicy(t *testing.T) {
	for _, value := range []float32{.5, .75, 1} {
		s := noteTargetPolicySong(t)
		voice := &s.Tracks[0].Measures[0].Voices[0]
		origin := &voice.Beats[1].Notes[0]
		origin.DurationPercent = value
		origin.TieOrigin = true
		dest := *origin
		dest.Kind = gp.NoteTypeTie
		dest.TieOrigin = false
		dest.DurationPercent = 1
		voice.Beats = append(voice.Beats, gp.Beat{Duration: gp.Duration{Value: 4, TupletEnters: 1, TupletTimes: 1}, Status: gp.BeatStatusNormal, Dynamics: 95, Notes: []gp.Note{dest}})
		code := "gp8.omit.note-duration-percent"
		if value == 1 {
			code = ""
		}
		parsed := assertNoteTargetPolicy(t, s, code, gp.ExportDispositionOmitted)
		got := parsed.Tracks[0].Measures[0].Voices[0].Beats
		for bi := range got {
			want := voice.Beats[bi]
			if got[bi].Duration != want.Duration || !reflect.DeepEqual(got[bi].ExactStart, want.ExactStart) {
				t.Fatalf("rhythm changed at beat%d", bi)
			}
			for ni, n := range got[bi].Notes {
				src := want.Notes[ni]
				if n.DurationPercent != 1 || n.Value != src.Value || n.String != src.String || n.Kind != src.Kind || n.TieOrigin != src.TieOrigin || n.Effect.LetRing != src.Effect.LetRing {
					t.Fatalf("note semantics changed %#v", n)
				}
			}
		}
	}
}
func TestGP8RejectsInvalidDurationPercentages(t *testing.T) {
	for _, value := range []float32{-.01, float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1))} {
		s := noteTargetPolicySong(t)
		s.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[0].DurationPercent = value
		diagnostics := gp.ValidateSong(s)
		found := false
		for _, d := range diagnostics {
			if d.Code == "score.note.duration-percent" && d.Location == (gp.ScoreLocation{Beat: 1}) {
				found = true
			}
		}
		if !found {
			t.Fatalf("invalid fraction %v lacks scoped diagnostic %#v", value, diagnostics)
		}
		data, _, err := gp.ExportWithReport(s, gp.ExportFormatGP8, gp.ExportOptions{})
		if err == nil || len(data) != 0 {
			t.Fatalf("invalid fraction %v exported", value)
		}
	}
}
