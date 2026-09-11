// SPDX-License-Identifier: MIT

package integration_test

import (
	"reflect"
	"slices"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestBeatTechniquesLegacyExport(t *testing.T) {
	for _, value := range []gp.SlapEffect{gp.SlapEffectTapping, gp.SlapEffectSlapping, gp.SlapEffectPopping} {
		song := dynamicPolicySong(t, 47, false)
		beat := &song.Tracks[0].Measures[0].Voices[0].Beats[1]
		beat.Notes[1].Velocity = 47
		beat.Effect.SlapEffect = value
		data, err := gp.Export(song, gp.ExportFormatGP8)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := gp.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		if parsed.Tracks[0].Measures[0].Voices[0].Beats[1].Effect.SlapEffect != value {
			t.Fatalf("legacy technique %d discarded", value)
		}
	}
}

func TestBeatTechniquesIndependentStatesAndPolicy(t *testing.T) {
	for mask := 0; mask < 32; mask++ {
		song := dynamicPolicySong(t, 47, false)
		beat := &song.Tracks[0].Measures[0].Voices[0].Beats[1]
		beat.Notes[1].Velocity = 47
		beat.Effect.Tap, beat.Effect.Slap, beat.Effect.Pop = mask&1 != 0, mask&2 != 0, mask&4 != 0
		beat.Notes[0].Effect.Tapped, beat.Notes[0].Effect.LeftHandTapped = mask&8 != 0, mask&16 != 0
		before := beat.Notes[0].Effect
		code := ""
		if beat.Effect.Tap && !before.Tapped {
			code = "gp8.normalize.beat-tap-note-flag"
		}
		if !beat.Effect.Tap && before.Tapped {
			code = "gp8.normalize.beat-tap-note-authority"
		}
		for _, allowed := range [][]string{nil, {"unrelated"}, {code}} {
			data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
			if code != "" {
				if len(report.Entries) != 1 || report.Entries[0].Code != code || report.Entries[0].Location.Beat != 1 {
					t.Fatalf("mask %d report %#v", mask, report)
				}
				if !slices.Contains(allowed, code) {
					if err == nil || len(data) != 0 {
						t.Fatal("strict accepted")
					}
					continue
				}
			}
			if err != nil {
				t.Fatalf("mask %d: %v %#v", mask, err, report)
			}
			parsed, err := gp.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			got := parsed.Tracks[0].Measures[0].Voices[0].Beats[1]
			if got.Effect.Tap != (beat.Effect.Tap || before.Tapped) || got.Effect.Slap != beat.Effect.Slap || got.Effect.Pop != beat.Effect.Pop || got.Notes[0].Effect.LeftHandTapped != before.LeftHandTapped || !reflect.DeepEqual(beat.Notes[0].Effect, before) {
				t.Fatalf("mask %d lost independence", mask)
			}
		}
	}
}

func TestBeatTechniquesImportedEdits(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*gp.BeatEffects)
		want [3]bool
		code string
	}{
		{"clear tap", func(e *gp.BeatEffects) { e.Tap = false }, [3]bool{true, true, true}, "gp8.normalize.beat-tap-note-authority"},
		{"clear slap", func(e *gp.BeatEffects) { e.Slap = false }, [3]bool{true, false, true}, ""},
		{"legacy tap", func(e *gp.BeatEffects) { e.SlapEffect = gp.SlapEffectTapping }, [3]bool{true, false, false}, ""},
		{"legacy clear", func(e *gp.BeatEffects) { e.SlapEffect = gp.SlapEffectNone }, [3]bool{true, false, false}, "gp8.normalize.beat-tap-note-authority"},
		{"conflict", func(e *gp.BeatEffects) { e.Slap = false; e.SlapEffect = gp.SlapEffectSlapping }, [3]bool{true, false, true}, "gp8.normalize.beat-technique-authority"},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := dynamicPolicySong(t, 47, false)
			b := &song.Tracks[0].Measures[0].Voices[0].Beats[1]
			b.Notes[1].Velocity = 47
			b.Effect.Tap, b.Effect.Slap, b.Effect.Pop = true, true, true
			b.Notes[0].Effect.Tapped = true
			data, err := gp.Export(song, gp.ExportFormatGP8)
			if err != nil {
				t.Fatal(err)
			}
			song, err = gp.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			song.Version = gp.Version{}
			for i := range song.Tracks {
				song.Tracks[i].Settings = gp.TrackSettings{Notation: true, Tablature: true}
			}
			b = &song.Tracks[0].Measures[0].Voices[0].Beats[1]
			test.edit(&b.Effect)
			data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if test.code != "" && !slices.ContainsFunc(report.Entries, func(e gp.ExportReportEntry) bool { return e.Code == test.code }) {
				t.Fatal(report)
			}
			for _, allowed := range [][]string{nil, {"unrelated"}, {test.code}} {
				strictData, strictReport, strictErr := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
				if !reflect.DeepEqual(strictReport, report) {
					t.Fatal("policy changed report")
				}
				if test.code != "" && !slices.Contains(allowed, test.code) {
					if strictErr == nil || len(strictData) != 0 {
						t.Fatal("unallowed collision accepted")
					}
				} else if strictErr != nil {
					t.Fatal(strictErr, report)
				}
			}
			parsed, err := gp.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			e := parsed.Tracks[0].Measures[0].Voices[0].Beats[1].Effect
			if [3]bool{e.Tap, e.Slap, e.Pop} != test.want {
				t.Fatalf("got %v", e)
			}
		})
	}
}

func TestBeatTechniquesRestAndInvalid(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp8/section-track-names.gp")
	if err != nil {
		t.Fatal(err)
	}
	song.Version = gp.Version{}
	for i := range song.Tracks {
		song.Tracks[i].Settings = gp.TrackSettings{Notation: true, Tablature: true}
	}
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	beat.Effect.Tap = true
	for _, allowed := range [][]string{nil, {"unrelated"}, {"gp8.omit.beat-tap"}} {
		data, report, e := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: allowed}})
		if len(report.Entries) != 1 || report.Entries[0].Code != "gp8.omit.beat-tap" {
			t.Fatal(report)
		}
		if !slices.Contains(allowed, "gp8.omit.beat-tap") {
			if e == nil || len(data) != 0 {
				t.Fatal("rest strict accepted")
			}
			continue
		}
		if e != nil {
			t.Fatal(e)
		}
		parsed, e := gp.Parse(data)
		if e != nil {
			t.Fatal(e)
		}
		got := parsed.Tracks[0].Measures[0].Voices[0].Beats[0]
		if got.Effect.Tap || len(got.Notes) != 0 {
			t.Fatal("rest fabricated")
		}
	}
	beat.Effect.SlapEffect = gp.SlapEffect(255)
	if !slices.ContainsFunc(gp.ValidateSong(song), func(d gp.ScoreDiagnostic) bool { return d.Code == "score.beat.slap-effect" }) {
		t.Fatal("invalid enum undiagnosed")
	}
	if data, _, e := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{}); e == nil || len(data) != 0 {
		t.Fatal("invalid enum exported")
	}
}
