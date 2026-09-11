// SPDX-License-Identifier: MIT

package integration_test

import (
	"reflect"
	"slices"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestRasgueadoSourcePatterns(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp7/rasgueado.gp")
	if err != nil {
		t.Fatal(err)
	}
	beats := song.Tracks[0].Measures[0].Voices[0].Beats
	if !beats[0].Effect.HasRasgueado || !beats[2].Effect.HasRasgueado {
		t.Fatal("distinct source rasgueado marks were discarded")
	}
}

func TestRasgueadoPublicEditAuthority(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*gp.BeatEffects)
		want gp.RasgueadoPattern
		code string
	}{
		{"named", func(e *gp.BeatEffects) { e.RasgueadoPattern = gp.RasgueadoPeami }, gp.RasgueadoPeami, ""},
		{"clear named", func(e *gp.BeatEffects) { e.RasgueadoPattern = gp.RasgueadoNone }, gp.RasgueadoNone, ""},
		{"clear legacy", func(e *gp.BeatEffects) { e.HasRasgueado = false }, gp.RasgueadoNone, ""},
		{"incompatible", func(e *gp.BeatEffects) { e.RasgueadoPattern = gp.RasgueadoMi; e.HasRasgueado = false }, gp.RasgueadoMi, "gp8.normalize.rasgueado-authority"},
	} {
		t.Run(test.name, func(t *testing.T) {
			song, err := gp.ParseFile("../testdata/gp7/rasgueado.gp")
			if err != nil {
				t.Fatal(err)
			}
			beats := song.Tracks[0].Measures[0].Voices[0].Beats
			if beats[0].Effect.RasgueadoPattern != gp.RasgueadoIi || beats[2].Effect.RasgueadoPattern != gp.RasgueadoPmpAnapaest {
				t.Fatal("distinct named source patterns lost")
			}
			test.edit(&beats[0].Effect)
			before := beats[0].Effect
			output, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if found := slices.ContainsFunc(report.Entries, func(e gp.ExportReportEntry) bool { return e.Code == "gp8.normalize.rasgueado-authority" }); found != (test.code != "") {
				t.Fatal(report)
			}
			parsed, err := gp.Parse(output)
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.RasgueadoPattern != test.want || !reflect.DeepEqual(before, beats[0].Effect) {
				t.Fatal("edit authority or read-only export failed")
			}
		})
	}
}

func TestRasgueadoUnspecifiedAndInvalidPublicValues(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp8/section-track-names.gp")
	if err != nil {
		t.Fatal(err)
	}
	song.Version = gp.Version{}
	for i := range song.Tracks {
		song.Tracks[i].Settings = gp.TrackSettings{Notation: true, Tablature: true}
	}
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	beat.Effect.HasRasgueado = true
	code := "gp8.normalize.rasgueado-unspecified"
	for _, allowed := range [][]string{nil, {"unrelated"}, {code}} {
		data, report, e := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
		if len(report.Entries) != 1 || report.Entries[0].Code != code {
			t.Fatal(report)
		}
		if slices.Contains(allowed, code) {
			if e != nil {
				t.Fatal(e)
			}
			parsed, pErr := gp.Parse(data)
			if pErr != nil {
				t.Fatal(pErr)
			}
			if parsed.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.RasgueadoPattern != gp.RasgueadoIi {
				t.Fatal("wrong documented default")
			}
		} else if e == nil || len(data) != 0 {
			t.Fatal("unallowed unspecified pattern exported")
		}
	}
	beat.Effect.RasgueadoPattern = gp.RasgueadoPattern(255)
	if !slices.ContainsFunc(gp.ValidateSong(song), func(d gp.ScoreDiagnostic) bool { return d.Code == "score.beat.rasgueado" }) {
		t.Fatal("invalid public pattern undiagnosed")
	}
	data, _, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
	if err == nil || len(data) != 0 {
		t.Fatal("invalid pattern exported")
	}
}
