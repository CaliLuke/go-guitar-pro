// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

const chordDisplayCase = "M14-CHORD-DISPLAY"
const chordDisplayValue = "independent name diagram and fingering visibility"
const chordDisplayFixture = "testdata/gp8/chord-display.gp"

func TestConformanceChordDisplay(t *testing.T) { runConformanceChordDisplay(newConformanceRun(t)) }
func runConformanceChordDisplay(run *conformanceRun) {
	t := run.t
	song := chordDisplaySong(t)
	for st := range song.Tracks[0].Staves {
		for n := 0; n < 8; n++ {
			chord := song.Tracks[0].Staves[st].Measures[n/4].Voices[0].Beats[n%4].Effect.Chord
			flags := n
			if st == 1 {
				flags = 7 - n
			}
			for k, field := range chordDisplayFields(chord) {
				if field.name == "ShowName" {
					run.ClaimPrimary(claimAllStages("chord-display", chordDisplayCase, chordDisplayValue)...)
				}
				run.Preserved("Chord."+field.name, *field.value, flags&(1<<k) != 0)
			}
		}
	}
	before := conformanceContractSnapshot(song, false)
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil {
		t.Fatal(err)
	}
	doc := conformanceWireDocument(t, data)

	wireCount := 0
	for _, property := range doc.Tracks.Tracks[0].Properties {
		if property.Items == nil {
			continue
		}
		for _, item := range property.Items.Items {
			id, parseErr := strconv.Atoi(item.ID)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			n, st := id%12, id/12
			if n >= 8 {
				continue
			}
			flags := n
			if st == 1 {
				flags = 7 - n
			}
			if len(item.Diagram.Properties) != 3 {
				t.Fatalf("chord %s property count", item.ID)
			}
			for k, p := range item.Diagram.Properties {
				run.Wire("gpifDiagramProperty.Name", p.Name, []string{"ShowName", "ShowDiagram", "ShowFingering"}[k])
				run.Wire("gpifDiagramProperty.Type", p.Type, "bool")
				run.ClaimSerialization(claimSite("chord-display", "export", chordDisplayCase, chordDisplayValue)).Wire("gpifDiagramProperty.Value", p.Value, fmt.Sprint(flags&(1<<k) != 0))
				wireCount++
			}
		}
	}
	if wireCount != 48 {
		t.Fatalf("asserted %d properties, want 48", wireCount)
	}

	{
		run.ClaimReport(claimSite("chord-display", "export", chordDisplayCase, chordDisplayValue)).Report(chordDisplayCase, reportCodes(report), []string{})
	}
	if conformanceContractSnapshot(song, false) != before {
		t.Fatal("chord export mutated source")
	}
}
func chordDisplayFields(chord *Chord) []struct {
	name  string
	value *bool
} {
	return []struct {
		name  string
		value *bool
	}{{"ShowName", chord.ShowName}, {"ShowDiagram", chord.ShowDiagram}, {"ShowFingering", chord.ShowFingering}}
}
func chordDisplaySong(t *testing.T) *Song {
	t.Helper()
	song := parseTestFixture(t, chordDisplayFixture)
	song.Version = Version{}
	song.Tracks[0].Settings = TrackSettings{Notation: true}
	return song
}

type chordDisplayFact struct {
	Track, Staff, Bar, Voice, Beat       int
	Name                                 string
	ShowName, ShowDiagram, ShowFingering bool
	Strings                              []int
}

func TestAlphaTabChordDisplay(t *testing.T) {
	requireAlphaTabConformance(t)
	want := []chordDisplayFact{}
	for st := 0; st < 2; st++ {
		for n := 0; n < 12; n++ {
			flags := n % 8
			if st == 1 {
				flags = 7 - flags
			}
			strings := []int{-1, -1, 3, -1, -1, -1}
			if n == 8 {
				flags = 7
			}
			if n == 9 {
				flags = 1
				strings = []int{}
			}
			if n == 10 {
				flags = 0
				if st == 1 {
					flags = 7
				}
			}
			if n == 11 {
				flags = 1
				if st == 1 {
					flags = 6
				}
			}
			want = append(want, chordDisplayFact{0, st, n / 4, 0, n % 4, "Same", flags&1 != 0, flags&2 != 0, flags&4 != 0, strings})
		}
	}
	var got []chordDisplayFact
	readAlphaTabOracleFacts(t, "--chord-display", chordDisplayFixture, &got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("source facts=%#v", got)
	}
	song := chordDisplaySong(t)
	for _, edited := range []bool{false, true} {
		if edited {
			first := song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Effect.Chord
			*first.ShowName = true
			*first.ShowDiagram = true
			*first.ShowFingering = true
			want[0].ShowName = true
			want[0].ShowDiagram = true
			want[0].ShowFingering = true
		}
		readAlphaTabOracleFacts(t, "--chord-display", writeConformanceFixture(t, mustStringNumberExport(t, song)), &got)
		// An absent source Diagram has no string entries; the existing writer emits an empty six-string diagram.
		for _, index := range []int{9, 21} {
			want[index].Strings = []int{-1, -1, -1, -1, -1, -1}
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("edited=%t consumer=%#v", edited, got)
		}
	}
	conformanceIndependentClaim(t, "field:Chord.ShowName", claimAllStages("chord-display", chordDisplayCase, chordDisplayValue)...)
}
