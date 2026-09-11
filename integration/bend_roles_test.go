// SPDX-License-Identifier: MIT
package integration_test

import (
	"math"
	"reflect"
	"strings"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestGP8BendControlRoles(t *testing.T) {
	for _, offsets := range [][]float64{{0, 35, 65, 100}, {0, 50, 50, 35}, {0, 50, 65.5, 35.5}} {
		song := dynamicPolicySong(t, 47, false)
		beat := &song.Tracks[0].Measures[0].Voices[0].Beats[1]
		beat.Notes[1].Velocity = 47
		points := make([]gp.BendPoint, 4)
		for i, offset := range offsets {
			points[i] = gp.BendPoint{Position: uint8(math.Round(offset * 12 / 100)), ExactOffset: &offset}
			if i == 1 || i == 2 {
				points[i].Value = 2
			}
		}
		beat.Notes[0].Effect.Bend = &gp.BendEffect{Points: points}
		if ds := gp.ValidateSong(song); len(ds) != 0 {
			t.Fatalf("offsets %v diagnostics %#v", offsets, ds)
		}
		before := append([]gp.BendPoint(nil), points...)
		opts := gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}}
		preflight := gp.PreflightExport(song, gp.ExportFormatGP8, opts)
		data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, opts)
		if err != nil || len(report.Entries) != 0 || !reflect.DeepEqual(report, preflight) {
			t.Fatalf("export %v %#v", err, report)
		}
		parsed, err := gp.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		got := parsed.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[0].Effect.Bend.Points
		if len(got) != 4 {
			t.Fatalf("controls %#v", got)
		}
		for i, p := range got {
			offset := float64(p.Position) * 100 / 12
			if p.ExactOffset != nil {
				offset = *p.ExactOffset
			}
			if offset != offsets[i] || p.Value != points[i].Value {
				t.Fatalf("role %d got %#v want %v", i, p, points[i])
			}
		}
		if !reflect.DeepEqual(points, before) {
			t.Fatal("export mutated controls")
		}
	}
}
func TestGP8BendRoleFixture(t *testing.T) {
	for _, path := range []string{"../testdata/gp7/canon-audio-track.gp", "../testdata/gp8/canon-audio-track.gp"} {
		song, err := gp.ParseFile(path)
		if err != nil {
			t.Fatal(err)
		}
		data, _, originalErr := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
		if originalErr == nil || len(data) != 0 || !strings.Contains(originalErr.Error(), "sound automation 1 has bar 179 position 2") {
			t.Fatalf("original source must retain its separate sound-position rejection: %v", originalErr)
		}
		wantEvent := gp.SoundAutomation{Bar: 179, Position: 2, Sound: 1}
		if len(song.Tracks[2].SoundAutomations) != 3 || song.Tracks[2].SoundAutomations[1] != wantEvent {
			t.Fatalf("unexpected source sound events %#v", song.Tracks[2].SoundAutomations)
		}
		// Isolate only the independently invalid event, preserving the sound domain.
		events := song.Tracks[2].SoundAutomations
		song.Tracks[2].SoundAutomations = append(events[:1:1], events[2:]...)
		_, _, err = gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}
}

func TestGP8ControlRolesRejectArbitraryWhammyOrder(t *testing.T) {
	song := dynamicPolicySong(t, 47, false)
	song.Tracks[0].Measures[0].Voices[0].Beats[1].Effect.TremoloBar = &gp.BendEffect{Points: []gp.BendPoint{{}, {Position: 9, Value: 1}, {Position: 6, Value: 2}, {Position: 4}}}
	data, _, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
	if err == nil || len(data) != 0 || !strings.Contains(err.Error(), "precedes offset") {
		t.Fatalf("whammy order rejection changed: %v", err)
	}
}

func TestGP8BendRoleBoundaries(t *testing.T) {
	for _, points := range [][]gp.BendPoint{
		{{}, {Position: 9, Value: 2}, {Position: 3}},
		{{}, {Position: 9, Value: 1}, {Position: 9, Value: 2}, {Position: 3}},
		{{}, {Position: 9, Value: 2}, {Position: 9, Value: 2}, {Position: 3}, {Position: 12}},
		{{Position: 6}, {Position: 3, Value: 2}, {Position: 9, Value: 2}, {Position: 4}},
	} {
		song := dynamicPolicySong(t, 47, false)
		beat := &song.Tracks[0].Measures[0].Voices[0].Beats[1]
		beat.Notes[1].Velocity = 47
		beat.Notes[0].Effect.Bend = &gp.BendEffect{Points: points}
		data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
		if err == nil || len(data) != 0 {
			t.Fatalf("unordered arbitrary curve accepted %#v %#v", points, report)
		}
	}
	for _, offset := range []float64{-0.01, 100.01, math.NaN(), math.Inf(1)} {
		song := dynamicPolicySong(t, 47, false)
		song.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[0].Effect.Bend = &gp.BendEffect{Points: []gp.BendPoint{{}, {Position: 6, Value: 2}, {Position: 6, Value: 2}, {ExactOffset: &offset}}}
		data, _, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
		if err == nil || len(data) != 0 {
			t.Fatalf("invalid exact offset %v accepted", offset)
		}
	}
}

func TestGP8BendTerminalHoldDoesNotInventControlRoles(t *testing.T) {
	for _, test := range []struct{ points, want []gp.BendPoint }{
		{[]gp.BendPoint{{}, {Position: 3, Value: 1}, {Position: 12, Value: 1}}, []gp.BendPoint{{}, {Position: 3, Value: 1}}},
		{[]gp.BendPoint{{}, {Position: 3, Value: 1}}, []gp.BendPoint{{}, {Position: 3, Value: 1}}},
		{[]gp.BendPoint{{}, {Position: 3, Value: 1}, {Position: 12, Value: 1}, {Position: 3, Value: 1}}, []gp.BendPoint{{}, {Position: 3, Value: 1}, {Position: 12, Value: 1}, {Position: 3, Value: 1}}},
	} {
		song := dynamicPolicySong(t, 47, false)
		beat := &song.Tracks[0].Measures[0].Voices[0].Beats[1]
		beat.Notes[1].Velocity = 47
		beat.Notes[0].Effect.Bend = &gp.BendEffect{Points: test.points}
		before := append([]gp.BendPoint(nil), test.points...)
		data, _, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := gp.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		got := parsed.Tracks[0].Measures[0].Voices[0].Beats[1].Notes[0].Effect.Bend.Points
		if !reflect.DeepEqual(got, test.want) {
			t.Fatalf("authored%#v reparsed%#v want%#v", test.points, got, test.want)
		}
		if !reflect.DeepEqual(test.points, before) {
			t.Fatal("export mutated authored curve")
		}
	}
}
