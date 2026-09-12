// SPDX-License-Identifier: MIT

package integration_test

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestStaffNotationPublicOwnership(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp8/staff-notation-settings.gp")
	if err != nil {
		t.Fatal(err)
	}
	first := &song.Tracks[0]
	if len(first.Staves) != 2 || first.Staves[0].NotationSettings == nil || first.Staves[1].NotationSettings == nil {
		t.Fatal("missing independent staff settings")
	}
	standard := gp.StaffNotationSettings{Standard: true}
	alternate := gp.StaffNotationSettings{Tablature: true, Slash: true, Numbered: true}
	if *first.Staves[0].NotationSettings != standard || *first.Staves[1].NotationSettings != standard || *song.Tracks[1].Staves[0].NotationSettings != alternate {
		t.Fatal("source notation flags changed")
	}
	if song.Tracks[1].Visible || !first.Visible {
		t.Fatal("track visibility coupled to staff views")
	}
	if !first.Staves[0].Measures[0].Voices[0].Beats[0].Slashed || song.Tracks[1].Staves[0].Measures[0].Voices[0].Beats[0].Slashed {
		t.Fatal("beat and staff slash flags conflated")
	}
	first.Staves[1].NotationSettings.Numbered = true
	if first.Staves[0].NotationSettings.Numbered {
		t.Fatal("staff setting pointers are shared")
	}
	first.Staves[1].NotationSettings.Numbered = false
	// Separate-track flags are retained by the pinned track-wide configuration.
	for i := range song.Tracks {
		track := &song.Tracks[i]
		track.Settings = gp.TrackSettings{Notation: track.Settings.Notation, Tablature: track.Settings.Tablature}
	}
	song.Version = gp.Version{}
	first.Staves[0].NotationSettings.Slash = true
	first.Staves[1].NotationSettings.Slash = true
	first.Staves[0].Measures[0].Voices[0].Beats[0].Slashed = false
	secondBefore := song.Tracks[1].Staves[0].Measures[0].Voices[0].Beats[0].Notes[0]
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("export=%v %#v", err, report.Entries)
	}
	parsed, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if !parsed.Tracks[0].Staves[0].NotationSettings.Slash || !parsed.Tracks[0].Staves[1].NotationSettings.Slash || parsed.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Slashed {
		t.Fatal("edited beat and staff flags changed")
	}
	// The fixture's automatic A-sharp becomes a concrete native pitch record.
	secondBefore.AccidentalMode = gp.NoteAccidentalSharp
	wantPublic, _ := json.Marshal(secondBefore)
	gotPublic, _ := json.Marshal(parsed.Tracks[1].Staves[0].Measures[0].Voices[0].Beats[0].Notes[0])
	if !bytes.Equal(gotPublic, wantPublic) {
		t.Fatal("display edit changed note data")
	}
}

func TestStaffNotationLegacyEditAuthority(t *testing.T) {
	for _, edit := range []struct {
		name   string
		change func(*gp.Track)
		want   gp.StaffNotationSettings
	}{
		{"staff only", func(track *gp.Track) {
			for i := range track.Staves {
				track.Staves[i].NotationSettings.Tablature = true
			}
		}, gp.StaffNotationSettings{Standard: true, Tablature: true}},
		{"legacy standard", func(track *gp.Track) {
			track.Settings.Notation = false
			for i := range track.Staves {
				track.Staves[i].NotationSettings.Slash = true
			}
		}, gp.StaffNotationSettings{Slash: true}},
		{"legacy tab", func(track *gp.Track) { track.Settings.Tablature = true }, gp.StaffNotationSettings{Standard: true, Tablature: true}},
		{"conflict", func(track *gp.Track) {
			track.Settings.Notation = false
			for i := range track.Staves {
				track.Staves[i].NotationSettings.Standard = true
				track.Staves[i].NotationSettings.Numbered = true
			}
		}, gp.StaffNotationSettings{Numbered: true}},
	} {
		t.Run(edit.name, func(t *testing.T) {
			song, err := gp.ParseFile("../testdata/gp8/staff-notation-settings.gp")
			if err != nil {
				t.Fatal(err)
			}
			edit.change(&song.Tracks[0])
			settings := *song.Tracks[0].Staves[0].NotationSettings
			data, err := gp.Export(song, gp.ExportFormatGP8)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := gp.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			for _, staff := range parsed.Tracks[0].Staves {
				if *staff.NotationSettings != edit.want {
					t.Fatalf("reconciled settings=%#v, want %#v", staff.NotationSettings, edit.want)
				}
			}
			if *song.Tracks[0].Staves[0].NotationSettings != settings {
				t.Fatal("export mutated authored staff settings")
			}
		})
	}
}

func TestPartConfigurationPublicMalformedLengths(t *testing.T) {
	fixture := mustReadFixture(t, "../testdata/gp8/staff-notation-settings.gp")
	archive, err := zip.NewReader(bytes.NewReader(fixture), int64(len(fixture)))
	if err != nil {
		t.Fatal(err)
	}
	readMember := func(name string) []byte {
		file, openErr := archive.Open(name)
		if openErr != nil {
			t.Fatal(openErr)
		}
		data, readErr := io.ReadAll(file)
		closeErr := file.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("read member: %v/%v", readErr, closeErr)
		}
		return data
	}
	gpif, configuration := readMember("Content/score.gpif"), readMember("Content/PartConfiguration")
	for _, length := range []int{0, 1, 3, 4, 8, 9, len(configuration) - 4, len(configuration) - 1} {
		t.Run(fmt.Sprint(length), func(t *testing.T) {
			data := buildGP7Archive(t, string(gpif), "Content/PartConfiguration", configuration[:length])
			song, parseErr := gp.Parse(data)
			result, optionsErr := gp.ParseWithOptions(data, gp.ParseOptions{})
			for _, got := range []error{parseErr, optionsErr} {
				var typed *gp.ParseError
				if !errors.As(got, &typed) || !strings.Contains(got.Error(), "reading PartConfiguration:") {
					t.Fatalf("malformed configuration error=%v", got)
				}
			}
			if song != nil || result != nil {
				t.Fatal("malformed configuration returned score")
			}
		})
	}
}
