// SPDX-License-Identifier: MIT
package integration_test

import (
	"encoding/json"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestScoreStyleBarlinePublicValues(t *testing.T) {
	for _, test := range []struct {
		name   string
		extend bool
		policy gp.BarNumberPolicy
	}{{"extended-barlines", true, gp.BarNumberAllBars}, {"barnumbers-all", false, gp.BarNumberAllBars}, {"barnumbers-first", false, gp.BarNumberFirstOfSystem}, {"barnumbers-hide", false, gp.BarNumberHide}} {
		t.Run(test.name, func(t *testing.T) {
			song, err := gp.ParseFile("../testdata/gp8/" + test.name + ".gp")
			if err != nil {
				t.Fatal(err)
			}
			if song.Style == nil || song.Style.ExtendedBarLines == nil || song.Style.BarNumbers == nil || *song.Style.ExtendedBarLines != test.extend || *song.Style.BarNumbers != test.policy {
				t.Fatal("authored score style missing or changed")
			}
			song.Version = gp.Version{}
			song.PanAutomations = nil
			song.VolumeAutomations = nil
			for i := range song.Tracks {
				track := &song.Tracks[i]
				track.UseRse = false
				track.Settings = gp.TrackSettings{Notation: track.Settings.Notation, Tablature: track.Settings.Tablature}
			}
			*song.Style.ExtendedBarLines = !test.extend
			*song.Style.BarNumbers = gp.BarNumberHide
			before, _ := json.Marshal(song)
			data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.normalize.track-name-page-policy"}}})
			if err != nil || len(report.Entries) != 1 || report.Entries[0].Code != "gp8.normalize.track-name-page-policy" {
				t.Fatalf("strict style export=%v %#v", err, report)
			}
			after, _ := json.Marshal(song)
			if string(before) != string(after) {
				t.Fatal("style export mutated source")
			}
			parsed, err := gp.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			if *parsed.Style.ExtendedBarLines == test.extend || *parsed.Style.BarNumbers != gp.BarNumberHide {
				t.Fatal("style edit changed")
			}
			song.Style.ExtendedBarLines = nil
			song.Style.BarNumbers = nil
			data, err = gp.Export(song, gp.ExportFormatGP8)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err = gp.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			if parsed.Style.ExtendedBarLines != nil || parsed.Style.BarNumbers != nil {
				t.Fatal("cleared authored style reappeared")
			}
			invalid := gp.BarNumberPolicy(3)
			song.Style.BarNumbers = &invalid
			data, report, err = gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{AllowedCodes: []string{"gp8.reject.score.style.bar-number-policy"}}})
			if err == nil || len(data) != 0 {
				t.Fatalf("undefined style policy exported: %#v", report)
			}
		})
	}
}
