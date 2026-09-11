// SPDX-License-Identifier: MIT
package integration_test

import (
	"reflect"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestPublicNoteOrnaments(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp7/ornaments.gp")
	if err != nil {
		t.Fatal(err)
	}
	counts := map[gp.NoteOrnament]int{}
	for _, track := range song.Tracks {
		for _, measure := range track.Measures {
			for _, voice := range measure.Voices {
				for _, beat := range voice.Beats {
					for _, note := range beat.Notes {
						if note.Ornament != gp.NoteOrnamentNone {
							counts[note.Ornament]++
						}
					}
				}
			}
		}
	}
	want := map[gp.NoteOrnament]int{gp.NoteOrnamentTurn: 3, gp.NoteOrnamentInvertedTurn: 3, gp.NoteOrnamentUpperMordent: 3, gp.NoteOrnamentLowerMordent: 3}
	if !reflect.DeepEqual(counts, want) {
		t.Fatalf("ornament counts %#v want %#v", counts, want)
	}
}

func TestPublicOrnamentEditOwnership(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp7/ornaments.gp")
	if err != nil {
		t.Fatal(err)
	}
	notes := publicOrnamentNotes(song)
	if len(notes) != 12 {
		t.Fatal(len(notes))
	}
	before := *notes[1]
	notes[0].Ornament = gp.NoteOrnamentTurn
	if !reflect.DeepEqual(*notes[1], before) {
		t.Fatal("shared note definition changed")
	}
	data, _, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	out := publicOrnamentNotes(parsed)
	if len(out) != len(notes) {
		t.Fatal("note count changed")
	}
	for i, n := range notes {
		if out[i].Ornament != n.Ornament || out[i].Value != n.Value || out[i].String != n.String {
			t.Fatalf("note %d changed", i)
		}
	}
	notes[0].Ornament = gp.NoteOrnament(255)
	data, report, err := gp.ExportWithReport(song, gp.ExportFormatGP8, gp.ExportOptions{LossPolicy: gp.ExportLossPolicy{AllowedCodes: []string{"gp8.reject.score.note.ornament"}}})
	if err == nil || len(data) != 0 {
		t.Fatalf("undefined ornament exported %#v", report)
	}
}
func publicOrnamentNotes(song *gp.Song) []*gp.Note {
	var notes []*gp.Note
	for ti := range song.Tracks {
		for mi := range song.Tracks[ti].Measures {
			for vi := range song.Tracks[ti].Measures[mi].Voices {
				for bi := range song.Tracks[ti].Measures[mi].Voices[vi].Beats {
					beat := &song.Tracks[ti].Measures[mi].Voices[vi].Beats[bi]
					for ni := range beat.Notes {
						notes = append(notes, &beat.Notes[ni])
					}
				}
			}
		}
	}
	return notes
}
