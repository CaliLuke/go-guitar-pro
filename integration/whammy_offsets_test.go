// SPDX-License-Identifier: MIT

package integration_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestGP8ExactWhammyOffsets(t *testing.T) {
	for _, offsets := range [][]float64{{0, 35, 35.5, 100}, {0, 35, 35.5, 25}} {
		s := dynamicPolicySong(t, 47, false)
		b := &s.Tracks[0].Measures[0].Voices[0].Beats[1]
		b.Notes[1].Velocity = 47
		points := make([]gp.BendPoint, 4)
		for i, offset := range offsets {
			points[i] = gp.BendPoint{Position: uint8(math.Round(offset * 12 / 100)), ExactOffset: &offset}
			if i == 1 || i == 2 {
				points[i].Value = -2
			}
		}
		b.Effect.TremoloBar = &gp.BendEffect{Points: points}
		if ds := gp.ValidateSong(s); len(ds) != 0 {
			t.Fatalf("valid whammy rejected %#v", ds)
		}
		data, report, err := gp.ExportWithReport(s, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}})
		if err != nil || len(report.Entries) != 0 {
			t.Fatalf("exact whammy export %v %#v", err, report)
		}
		parsed, err := gp.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		got := parsed.Tracks[0].Measures[0].Voices[0].Beats[1].Effect.TremoloBar.Points
		if len(got) != 4 {
			t.Fatalf("lost control roles %#v", got)
		}
		actual := make([]float64, 4)
		for i, p := range got {
			actual[i] = float64(p.Position) * 100 / 12
			if p.ExactOffset != nil {
				actual[i] = *p.ExactOffset
			}
			if p.Value != points[i].Value {
				t.Fatal("changed signed height")
			}
		}
		if !reflect.DeepEqual(actual, offsets) {
			t.Fatalf("offsets %#v want %#v", actual, offsets)
		}
	}
}

func TestGP8WhammyOffsetEdits(t *testing.T) {
	for _, edit := range []string{"exact", "legacy", "both", "clear", "programmatic"} {
		s := dynamicPolicySong(t, 47, false)
		b := &s.Tracks[0].Measures[0].Voices[0].Beats[1]
		b.Notes[1].Velocity = 47
		offset := 35.0
		b.Effect.TremoloBar = &gp.BendEffect{Points: []gp.BendPoint{{}, {Position: 4, ExactOffset: &offset, Value: -2}, {Position: 9, Value: -2}, {Position: 12}}}
		data, err := gp.Export(s, gp.ExportFormatGP8)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := gp.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		points := parsed.Tracks[0].Measures[0].Voices[0].Beats[1].Effect.TremoloBar.Points
		changed := 36.25
		want := changed
		conflict := false
		switch edit {
		case "exact":
			points[1].ExactOffset = &changed
		case "legacy":
			points[1].Position = 6
			want = 50
			conflict = true
		case "both":
			points[1].Position = 6
			points[1].ExactOffset = &changed
			want = 50
			conflict = true
		case "clear":
			points[1].ExactOffset = nil
			want = 100.0 / 3
		case "programmatic":
			points[1] = gp.BendPoint{Position: 6, ExactOffset: &changed, Value: -2}
		}
		// Put the imported controls in an otherwise lossless programmatic score.
		b.Effect.TremoloBar.Points = points
		before, _ := json.Marshal(s)
		preflight := gp.PreflightExport(s, gp.ExportFormatGP8, gp.ExportOptions{})
		opts := gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}}
		data, report, err := gp.ExportWithReport(s, gp.ExportFormatGP8, opts)
		if !reflect.DeepEqual(report, preflight) {
			t.Fatal("preflight differs")
		}
		if conflict {
			var loss *gp.ExportLossError
			if !errors.As(err, &loss) || len(data) != 0 || len(report.Entries) != 1 || report.Entries[0].Code != "gp8.normalize.whammy-offset-authority" || report.Entries[0].Location.Beat != 1 {
				t.Fatalf("conflict %v %#v", err, report)
			}
			opts.LossPolicy.AllowedCodes = []string{"gp8.normalize.bend-offset-authority"}
			if rejected, _, e := gp.ExportWithReport(s, gp.ExportFormatGP8, opts); e == nil || len(rejected) != 0 {
				t.Fatal("unrelated allowance bypassed policy")
			}
			opts.LossPolicy.AllowedCodes = []string{"gp8.normalize.whammy-offset-authority"}
			data, _, err = gp.ExportWithReport(s, gp.ExportFormatGP8, opts)
		}
		if err != nil {
			t.Fatal(err)
		}
		after, _ := json.Marshal(s)
		if !bytes.Equal(before, after) {
			t.Fatal("mutated public model")
		}
		output, err := gp.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		point := output.Tracks[0].Measures[0].Voices[0].Beats[1].Effect.TremoloBar.Points[1]
		actual := float64(point.Position) * 100 / 12
		if point.ExactOffset != nil {
			actual = *point.ExactOffset
		}
		if actual != want {
			t.Fatalf("%s got %v want %v", edit, actual, want)
		}
	}
}

func TestGP8WhammyOffsetBounds(t *testing.T) {
	for _, offset := range []float64{-0.01, 100.01, math.NaN(), math.Inf(1), math.Inf(-1)} {
		s := dynamicPolicySong(t, 47, false)
		s.Tracks[0].Measures[0].Voices[0].Beats[1].Effect.TremoloBar = &gp.BendEffect{Points: []gp.BendPoint{{}, {Position: 4, ExactOffset: &offset, Value: -2}, {Position: 12}}}
		if ds := gp.ValidateSong(s); len(ds) == 0 {
			t.Fatalf("invalid %v accepted", offset)
		}
		if data, _, err := gp.ExportWithReport(s, gp.ExportFormatGP8, gp.ExportOptions{}); err == nil || len(data) != 0 {
			t.Fatalf("invalid %v exported", offset)
		}
	}
}
