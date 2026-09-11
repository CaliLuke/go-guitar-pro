// SPDX-License-Identifier: MIT

package integration_test

import (
	"encoding/json"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestMultiRestPublicPreferences(t *testing.T) {
	result, err := gp.ParseFileWithOptions("../testdata/gp8/multi-rest-preferences.gp", gp.ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	song := result.Song
	if song.Style == nil || song.Style.MultiRest == nil || !*song.Style.MultiRest || song.Tracks[0].MultiRest == nil || *song.Tracks[0].MultiRest || song.Tracks[1].MultiRest == nil || !*song.Tracks[1].MultiRest {
		t.Fatal("global and conflicting track preferences lost")
	}
	if song.Style.MultiRest == song.Tracks[1].MultiRest || song.Tracks[0].MultiRest == song.Tracks[1].MultiRest {
		t.Fatal("preferences share mutable storage")
	}
	song.Version = gp.Version{}
	for i := range song.Tracks {
		track := &song.Tracks[i]
		track.Settings = gp.TrackSettings{Notation: track.Settings.Notation, Tablature: track.Settings.Tablature}
	}
	*song.Tracks[0].MultiRest = true
	*song.Tracks[1].MultiRest = false
	song.Style.MultiRest = nil
	before, _ := json.Marshal(song)
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("strict export=%v %#v", err, report)
	}
	after, _ := json.Marshal(song)
	if string(before) != string(after) {
		t.Fatal("export mutated source")
	}
	parsed, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Style.MultiRest == nil || *parsed.Style.MultiRest || !*parsed.Tracks[0].MultiRest || *parsed.Tracks[1].MultiRest {
		t.Fatal("independent edits/default changed")
	}
	for ti, track := range song.Tracks {
		for si, staff := range track.Staves {
			if len(staff.Measures) != len(parsed.Tracks[ti].Staves[si].Measures) {
				t.Fatal("multi-rest preference changed measure count")
			}
		}
	}
}
