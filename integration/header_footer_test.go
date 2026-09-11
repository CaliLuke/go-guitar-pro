// SPDX-License-Identifier: MIT

package integration_test

import (
	"encoding/json"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestHeaderFooterPublicValuesAndEdits(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp8/header-footer.gp")
	if err != nil {
		t.Fatal(err)
	}
	if song.Style == nil || song.Style.HeaderFooter == nil {
		t.Fatal("header/footer styles absent")
	}
	styles := song.Style.HeaderFooter
	if *styles.Title.Template != "Title: %TITLE%" || *styles.Title.Visible || *styles.Copyright2.Template != "Copyright2" || *styles.Copyright2.Visible {
		t.Fatal("template or visibility changed")
	}
	song.Version = gp.Version{}
	for i := range song.Tracks {
		tr := &song.Tracks[i]
		tr.UseRse = false
		tr.Settings = gp.TrackSettings{Notation: tr.Settings.Notation, Tablature: tr.Settings.Tablature}
	}
	value := "  é名 <&>\r\n%TITLE%  "
	styles.Title.Template = &value
	shown := true
	styles.Title.Visible = &shown
	styles.Copyright2.Template = &value
	song.PageSetup.Subtitle = "Legacy subtitle edit"
	before, _ := json.Marshal(song)
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.normalize.track-name-page-policy"}}})
	if err != nil || len(report.Entries) != 1 || report.Entries[0].Code != "gp8.normalize.track-name-page-policy" {
		t.Fatalf("strict header export=%v %#v", err, report)
	}
	after, _ := json.Marshal(song)
	if string(before) != string(after) {
		t.Fatal("export mutated style/compatibility views")
	}
	parsed, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if *parsed.Style.HeaderFooter.Title.Template != value || !*parsed.Style.HeaderFooter.Title.Visible || *parsed.Style.HeaderFooter.Copyright2.Template != value || *parsed.Style.HeaderFooter.Subtitle.Template != "Legacy subtitle edit" {
		t.Fatal("public header edits changed")
	}
	if *styles.Title.Template != value || *styles.Subtitle.Template != "Subtitle: %SUBTITLE%" {
		t.Fatal("reconciliation mutated rich source fields")
	}
	styles.Title.Template = nil
	styles.Title.Visible = nil
	data, err = gp.Export(song, gp.ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err = gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Style.HeaderFooter.Title != nil {
		t.Fatal("cleared authored template/visibility reappeared")
	}
}

func TestHeaderFooterGP5LegacyProjection(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp5/header-footer.gp5")
	if err != nil {
		t.Fatal(err)
	}
	if song.Style == nil || song.Style.HeaderFooter == nil || *song.Style.HeaderFooter.Copyright.Template != "Copyright: %COPYRIGHT%" || *song.Style.HeaderFooter.Copyright2.Template != "Copyright2" || !*song.Style.HeaderFooter.Copyright2.Visible {
		t.Fatal("GP5 source footer boundary lost")
	}
	song.PageSetup.Title = "Legacy edited title é名"
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	geometryReported := false
	for _, entry := range report.Entries {
		if entry.Code == "gp8.omit.page-setup" {
			geometryReported = true
		}
	}
	if !geometryReported {
		t.Fatal("physical page limit missing")
	}
	parsed, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if *parsed.Style.HeaderFooter.Title.Template != song.PageSetup.Title || *parsed.Style.HeaderFooter.Copyright2.Template != "Copyright2" {
		t.Fatal("GP5 edit or second footer changed")
	}
}
