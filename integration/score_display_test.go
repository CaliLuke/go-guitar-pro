// SPDX-License-Identifier: MIT

package integration_test

import (
	"encoding/json"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestScoreDisplayPublicPolicies(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp8/track-names.gp")
	if err != nil {
		t.Fatal(err)
	}
	style := song.Style
	if style == nil || style.SingleTrackNameMode == nil || *style.SingleTrackNameMode != gp.TrackNamesAllSystems || style.MultiTrackNamesVisible == nil || !*style.MultiTrackNamesVisible || style.HideDynamics == nil || !*style.HideDynamics {
		t.Fatal("authored score display policies missing")
	}
	yes, no := true, false
	brackets := gp.BracketSimilarInstruments
	style.Brackets = &brackets
	style.SystemSeparators = &yes
	style.HideDynamics = &no
	style.DisplayTuning = &no
	style.ChordDiagramsOnTop = &no
	style.ChordDiagramsInScore = &yes
	style.SingleTrackNamesVisible = &no
	style.MultiTrackNamesVisible = &yes
	style.FirstSystemShortNames = &no
	style.OtherSystemsShortNames = &yes
	style.FirstSystemHorizontalNames = &yes
	style.OtherSystemsHorizontalNames = &no
	song.Version = gp.Version{}
	song.PanAutomations = nil
	song.VolumeAutomations = nil
	for i := range song.Tracks {
		tr := &song.Tracks[i]
		tr.UseRse = false
		tr.Settings = gp.TrackSettings{Notation: tr.Settings.Notation, Tablature: tr.Settings.Tablature}
	}
	before, _ := json.Marshal(song)
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("strict display export=%v %#v", err, report)
	}
	after, _ := json.Marshal(song)
	if string(before) != string(after) {
		t.Fatal("display export mutated source")
	}
	parsed, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := parsed.Style
	if *got.Brackets != brackets || !*got.SystemSeparators || *got.HideDynamics || *got.DisplayTuning || *got.ChordDiagramsOnTop || !*got.ChordDiagramsInScore || *got.SingleTrackNamesVisible || !*got.MultiTrackNamesVisible || *got.FirstSystemShortNames || !*got.OtherSystemsShortNames || !*got.FirstSystemHorizontalNames || *got.OtherSystemsHorizontalNames {
		t.Fatal("independent display edits changed")
	}
	mode := gp.TrackNamesFirstSystemEachPage
	style.SingleTrackNameMode = &mode
	data, report, err = gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}})
	if err == nil || len(data) != 0 || len(report.Entries) != 1 || report.Entries[0].Code != "gp8.normalize.track-name-page-policy" {
		t.Fatalf("consumer page-mode limit missing: %v %#v", err, report)
	}
}
