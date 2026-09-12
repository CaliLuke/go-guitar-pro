// SPDX-License-Identifier: MIT

package integration_test

import (
	"encoding/xml"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestGraceNotesDoNotInheritOwnerMarkers(t *testing.T) {
	song, err := gp.ParseFile("../conformance/capabilities/evidence/guitar-pro-repair/contexts/grace.gp")
	if err != nil {
		t.Fatal(err)
	}
	note := &song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Notes[0]
	if len(note.Effect.Graces) != 1 {
		t.Fatal("missing grace")
	}
	note.Ornament = gp.NoteOrnamentTurn
	note.ShowStringNumber = true
	note.TieOrigin = true
	data, err := gp.Export(song, gp.ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Notes []struct {
			Ornament string
			Tie      struct {
				Origin bool `xml:"origin,attr"`
			}
			Properties []struct {
				Name string `xml:"name,attr"`
			} `xml:"Properties>Property"`
		} `xml:"Notes>Note"`
	}
	if err := xml.Unmarshal(textContractGPIF(t, data), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Notes) != 2 {
		t.Fatal(doc.Notes)
	}
	for i, n := range doc.Notes {
		showString := false
		for _, p := range n.Properties {
			showString = showString || p.Name == "ShowStringNumber"
		}
		if i == 0 && (n.Ornament != "" || n.Tie.Origin || showString) {
			t.Fatalf("grace inherited owner markers: %#v", n)
		}
		if i == 1 && (n.Ornament != "Turn" || !n.Tie.Origin || !showString) {
			t.Fatalf("owner markers lost: %#v", n)
		}
	}
}
