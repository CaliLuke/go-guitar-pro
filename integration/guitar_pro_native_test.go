// SPDX-License-Identifier: MIT

package integration_test

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestGP8NativePitchRecords(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp3/mix-table-events.gp3")
	if err != nil {
		t.Fatal(err)
	}
	data, err := gp.Export(song, gp.ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Notes []struct {
			Properties []struct {
				Name  string `xml:"name,attr"`
				Pitch *struct {
					Step, Accidental string
					Octave           int
				}
			} `xml:"Properties>Property"`
		} `xml:"Notes>Note"`
	}
	if err := xml.Unmarshal(textContractGPIF(t, data), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Notes) != 12 {
		t.Fatalf("note count = %d, want 12", len(doc.Notes))
	}
	for i, note := range doc.Notes {
		found := map[string]bool{}
		for _, p := range note.Properties {
			if p.Name != "ConcertPitch" && p.Name != "TransposedPitch" {
				continue
			}
			if p.Pitch == nil || p.Pitch.Step != "C" || p.Pitch.Accidental != "#" || p.Pitch.Octave != 5 {
				t.Fatalf("note %d %s does not spell MIDI 61: %#v", i, p.Name, p.Pitch)
			}
			found[p.Name] = true
		}
		if len(found) != 2 {
			t.Fatalf("note %d lacks native-required pitch records: %v", i, found)
		}
	}
}

func TestGP8NativeSoundPositionUnits(t *testing.T) {
	native, err := gp.ParseFile("../conformance/capabilities/evidence/guitar-pro-native/original-native.gp")
	if err != nil {
		t.Fatal(err)
	}
	if got := native.Tracks[0].SoundAutomations[1].Position; got != .25 {
		t.Errorf("native quarter-note offset imports as bar ratio %v, want .25", got)
	}
	song, err := gp.ParseFile("../testdata/gp3/mix-table-events.gp3")
	if err != nil {
		t.Fatal(err)
	}
	data, err := gp.Export(song, gp.ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Events []struct {
			Type     string
			Position float64
		} `xml:"Tracks>Track>Automations>Automation"`
	}
	if err := xml.Unmarshal(textContractGPIF(t, data), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Events) != 2 || doc.Events[0].Type != "Sound" || doc.Events[0].Position != 1 {
		t.Fatalf("sound position must be one quarter note (native tick 481): %#v", doc.Events)
	}
}

func TestGuitarProNativeSoundNames(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp3/mix-table-events.gp3")
	if err != nil {
		t.Fatal(err)
	}
	song.Tracks[0].Sounds[1].Label = "Flute <&> Ω"
	data, err := gp.Export(song, gp.ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(textContractGPIF(t, data))
	for _, text := range []string{"<Name><![CDATA[Legacy program 73 bank 0]]></Name>", "<Label><![CDATA[Flute <&> Ω]]></Label>", "<Name><![CDATA[Track 1]]></Name>"} {
		if !strings.Contains(raw, text) {
			t.Fatalf("Guitar Pro requires text as CDATA: missing %s", text)
		}
	}
	parsed, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Tracks[0].Sounds[1].Name != song.Tracks[0].Sounds[1].Name || parsed.Tracks[0].Sounds[1].Label != song.Tracks[0].Sounds[1].Label {
		t.Fatal("sound identity changed")
	}
}

func TestLegacyVolumeNativeScale(t *testing.T) {
	source, err := os.ReadFile("../testdata/gp3/mix-table-events.gp3")
	if err != nil {
		t.Fatal(err)
	}
	// Exhaustive native GP3 conversions, with only the volume byte changed.
	want := []float64{0, .05, .10, .14, .20, .25, .31, .37, .43, .48, .50, .53, .56, .58, .61, .64, .66}
	if source[980] != 12 {
		t.Fatal("fixture mix record moved")
	}
	for value, gain := range want {
		t.Run(fmt.Sprint(value), func(t *testing.T) {
			input := append([]byte(nil), source...)
			input[980] = byte(value)
			song, err := gp.Parse(input)
			if err != nil {
				t.Fatal(err)
			}
			if len(song.VolumeAutomations) != 1 || song.VolumeAutomations[0].Value != gain || song.VolumeAutomations[0].Linear {
				t.Fatal(song.VolumeAutomations)
			}
			data, err := gp.Export(song, gp.ExportFormatGP8)
			if err != nil {
				t.Fatal(err)
			}
			out, err := gp.Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			if len(out.VolumeAutomations) != 1 || out.VolumeAutomations[0] != song.VolumeAutomations[0] {
				t.Fatal(out.VolumeAutomations)
			}
		})
	}
}

func TestGP8AutomaticPitchKeyAndBounds(t *testing.T) {
	for _, test := range []struct {
		key              int8
		step, accidental string
	}{{-2, "B", "b"}, {2, "A", "#"}} {
		song := accidentalPublicSong(t)
		song.Tracks[0].Staves[0].Measures[0].KeySignature.Key = test.key
		for bi := range song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats {
			for ni := range song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[bi].Notes {
				n := &song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[bi].Notes[ni]
				n.String = 1
				n.Value = 10
				n.AccidentalMode = gp.NoteAccidentalDefault
			}
		}
		data, err := gp.Export(song, gp.ExportFormatGP8)
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Notes []struct {
				Properties []struct {
					Name  string `xml:"name,attr"`
					Pitch *struct {
						Step, Accidental string
						Octave           int
					}
				} `xml:"Properties>Property"`
			} `xml:"Notes>Note"`
		}
		if err := xml.Unmarshal(textContractGPIF(t, data), &doc); err != nil {
			t.Fatal(err)
		}
		for _, n := range doc.Notes {
			for _, p := range n.Properties {
				if p.Pitch != nil && (p.Pitch.Step != test.step || p.Pitch.Accidental != test.accidental || p.Pitch.Octave != 5) {
					t.Fatalf("key%d: %#v", test.key, p.Pitch)
				}
			}
		}
	}
	song := accidentalPublicSong(t)
	song.Tracks[0].Staves[0].CapoFret = 2147483647
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
	if err == nil || len(data) != 0 || !strings.Contains(err.Error(), "native pitch records cannot represent") {
		t.Fatalf("unrepresentable pitch accepted: %v %#v", err, report)
	}
	found := false
	for _, entry := range report.Entries {
		found = found || (entry.Code == "gp8.reject.conversion" && entry.Disposition == gp.ExportDispositionRejected)
	}
	if !found {
		t.Fatal("pitch failure lacks preflight rejection", report)
	}
}
