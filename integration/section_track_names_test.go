// SPDX-License-Identifier: MIT

package integration_test

import (
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestSectionAndTrackNames(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp8/section-track-names.gp")
	if err != nil {
		t.Fatal(err)
	}
	if song.MeasureHeaders[0].Marker.Letter != "A" || song.MeasureHeaders[0].Marker.Text != "Verse" || song.MeasureHeaders[0].Marker.Title != "Verse" {
		t.Fatalf("section lost its independent fields: %#v", song.MeasureHeaders[0].Marker)
	}
	for i, want := range []string{"Gtr Ω <&>", "Bass long label 日本語"} {
		if song.Tracks[i].ShortName == nil || *song.Tracks[i].ShortName != want {
			t.Fatalf("track %d short name = %#v", i, song.Tracks[i].ShortName)
		}
	}
}

func TestSectionTitlePublicEdits(t *testing.T) {
	for _, test := range []struct {
		name         string
		edit         func(*gp.Marker)
		letter, text string
	}{
		{"independent", func(m *gp.Marker) { m.Letter = "Ω <&>"; m.Text = "新 text" }, "Ω <&>", "新 text"},
		{"clear", func(m *gp.Marker) { m.Letter = ""; m.Text = "" }, "", ""},
		{"legacy", func(m *gp.Marker) { m.Title = "Legacy edit" }, "A", "Legacy edit"},
		{"legacy clear", func(m *gp.Marker) { m.Title = "" }, "A", ""},
		{"conflict", func(m *gp.Marker) { m.Title = "Legacy"; m.Text = "Other"; m.Letter = "B" }, "B", "Legacy"},
	} {
		t.Run(test.name, func(t *testing.T) {
			song, err := gp.ParseFile("../testdata/gp8/section-track-names.gp")
			if err != nil {
				t.Fatal(err)
			}
			marker := song.MeasureHeaders[0].Marker
			test.edit(marker)
			before := *marker
			data, err := gp.Export(song, gp.ExportFormatGP8)
			if err != nil {
				t.Fatal(err)
			}
			got, err := gp.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			if got.MeasureHeaders[0].Marker.Letter != test.letter || got.MeasureHeaders[0].Marker.Text != test.text || *marker != before {
				t.Fatalf("section edit changed: %#v", got.MeasureHeaders[0].Marker)
			}
			if got.MeasureHeaders[4].Marker != nil || got.MeasureHeaders[3].Marker == nil {
				t.Fatal("empty and absent sections were conflated")
			}
		})
	}
}

func TestShortNamePublicPresenceAndPolicy(t *testing.T) {
	empty, long := "", "  Long 日本語 <&> abbreviation  "
	for _, want := range []*string{nil, &empty, &long} {
		song, err := gp.ParseFile("../testdata/gp8/section-track-names.gp")
		if err != nil {
			t.Fatal(err)
		}
		song.Tracks[0].ShortName = want
		song.Version = gp.Version{}
		for i := range song.Tracks {
			song.Tracks[i].Settings = gp.TrackSettings{Notation: true, Tablature: true}
		}
		data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
		if err != nil {
			t.Fatal(err)
		}
		wantEmpty := want != nil && *want == ""
		if wantEmpty {
			if len(report.Entries) != 1 || report.Entries[0].Code != "gp8.omit.short-name-consumer-empty" || report.Entries[0].Disposition != gp.ExportDispositionOmitted {
				t.Fatalf("empty name policy = %#v", report.Entries)
			}
		} else if len(report.Entries) != 0 {
			t.Fatalf("unexpected name report = %#v", report.Entries)
		}
		parsed, err := gp.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		got := parsed.Tracks[0].ShortName
		if (got == nil) != (want == nil) || (got != nil && *got != *want) || parsed.Tracks[0].Name != "Lead guitar" || *parsed.Tracks[1].ShortName != "Bass long label 日本語" {
			t.Fatal("authored short name or adjacent identity changed")
		}
		if wantEmpty {
			output, _, strictErr := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}})
			if strictErr == nil || len(output) != 0 {
				t.Fatal("strict export accepted an unallowed empty-name consumer loss")
			}
		}
	}
}
