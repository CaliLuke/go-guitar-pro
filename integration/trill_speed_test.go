// SPDX-License-Identifier: MIT

package integration_test

import (
	"reflect"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestGP8TrillSpeedPolicy(t *testing.T) {
	for _, speed := range []uint16{16, 32, 64} {
		s := noteTargetPolicySong(t)
		note := &s.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[0]
		note.Effect.Trill = &gp.TrillEffect{Fret: 7, Duration: gp.Duration{Value: speed, TupletEnters: 1, TupletTimes: 1}}
		note.Effect.PalmMute = true
		note.Effect.Staccato = true
		code := "gp8.normalize.trill-duration"
		if speed == 16 {
			code = ""
		}
		parsed := assertNoteTargetPolicy(t, s, code, gp.ExportDispositionNormalized)
		got := parsed.Tracks[0].Measures[0].Voices[0].Beats[1]
		want := s.Tracks[0].Measures[0].Voices[0].Beats[1]
		n := got.Notes[0]
		if got.Duration != want.Duration || !reflect.DeepEqual(got.ExactStart, want.ExactStart) || n.Value != 3 || n.Effect.Trill == nil || n.Effect.Trill.Fret != 7 || n.Effect.Trill.Duration != (gp.Duration{Value: 16, TupletEnters: 1, TupletTimes: 1}) || !n.Effect.LetRing || !n.Effect.PalmMute || !n.Effect.Staccato {
			t.Fatalf("trill speed%d changed unrelated semantics %#v", speed, got)
		}
	}
}
