// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"slices"
	"strings"
	"testing"
)

const staffScopedChordGPIF = `<?xml version="1.0" encoding="utf-8"?>
<GPIF>
  <GPVersion>8.0</GPVersion>
  <Score><Title>Staff chords</Title></Score>
  <MasterTrack><Tracks>0</Tracks></MasterTrack>
  <Tracks><Track id="0"><Name>Piano</Name><Staves>
    <Staff><Properties><Property name="DiagramCollection"><Items><Item id="0" name="Upper"><Diagram stringCount="1" fretCount="4" baseFret="0"><Fret string="0" fret="1"/></Diagram></Item></Items></Property></Properties></Staff>
    <Staff><Properties><Property name="DiagramCollection"><Items><Item id="0" name="Lower"><Diagram stringCount="1" fretCount="4" baseFret="0"><Fret string="0" fret="2"/></Diagram></Item></Items></Property></Properties></Staff>
  </Staves><MidiConnection><Port>0</Port><PrimaryChannel>0</PrimaryChannel><SecondaryChannel>1</SecondaryChannel></MidiConnection></Track></Tracks>
  <MasterBars><MasterBar><Time>4/4</Time><Bars>0 1</Bars></MasterBar></MasterBars>
  <Bars><Bar id="0"><Clef>G2</Clef><Voices>0</Voices></Bar><Bar id="1"><Clef>F4</Clef><Voices>1</Voices></Bar></Bars>
  <Voices><Voice id="0"><Beats>0</Beats></Voice><Voice id="1"><Beats>1</Beats></Voice></Voices>
  <Beats><Beat id="0"><Rhythm ref="0"/><Chord>0</Chord></Beat><Beat id="1"><Rhythm ref="0"/><Chord>0</Chord></Beat></Beats>
  <Notes/><Rhythms><Rhythm id="0"><NoteValue>Quarter</NoteValue></Rhythm></Rhythms>
</GPIF>`

func TestGPIFChordIDsAreScopedPerStaff(t *testing.T) {
	context := &parseContext{format: "GPIF"}
	song, err := parseGPIFWithContext([]byte(staffScopedChordGPIF), context)
	if err != nil {
		t.Fatal(err)
	}
	track := &song.Tracks[0]
	upper := track.Staves[0].Measures[0].Voices[0].Beats[0].Effect.Chord
	lower := track.Staves[1].Measures[0].Voices[0].Beats[0].Effect.Chord
	if upper == nil || lower == nil || upper.Name != "Upper" || lower.Name != "Lower" {
		t.Fatalf("staff chords = %#v, %#v; want Upper, Lower", upper, lower)
	}
	for _, diagnostic := range context.diagnostics {
		if diagnostic.Code == "GPIF.ChordDefinition.DuplicateID" {
			t.Fatalf("staff-local IDs reported as duplicates: %#v", diagnostic)
		}
	}
}

func TestRepeatedGPIFChordOccurrencesOwnMutablePayloads(t *testing.T) {
	gpif := strings.Replace(staffScopedChordGPIF, `<Beats>0</Beats>`, `<Beats>0 0</Beats>`, 1)
	result, err := ParseWithOptions(conformanceGPIFArchive(t, gpif), ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	beats := result.Song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats
	if len(beats) != 2 {
		t.Fatalf("beats = %d, want 2", len(beats))
	}
	first := beats[0].Effect.Chord
	second := beats[1].Effect.Chord
	if first == nil || second == nil || first == second || first.FirstFret == nil || second.FirstFret == nil {
		t.Fatalf("chord occurrences = %#v, %#v", first, second)
	}
	first.Strings[0] = 7
	*first.FirstFret = 5
	if second.Strings[0] != 1 || *second.FirstFret != 1 {
		t.Fatalf("second chord changed through first occurrence: %#v", second)
	}

	data, err := Export(result.Song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	roundTripBeats := roundTrip.Tracks[0].Staves[0].Measures[0].Voices[0].Beats
	if roundTripBeats[0].Effect.Chord.Strings[0] != 7 || roundTripBeats[1].Effect.Chord.Strings[0] != 1 {
		t.Fatalf("round-trip chord strings = %v, %v", roundTripBeats[0].Effect.Chord.Strings, roundTripBeats[1].Effect.Chord.Strings)
	}
}

func TestGPIFRejectsOversizedChordDiagramBeforeAllocation(t *testing.T) {
	data := strings.Replace(staffScopedChordGPIF, `stringCount="1"`, `stringCount="1000"`, 1)
	if _, err := parseGPIF([]byte(data)); err == nil {
		t.Fatal("oversized chord diagram was accepted")
	}
}

func TestGPIFChordReferencesUseStaffScope(t *testing.T) {
	data := strings.Replace(staffScopedChordGPIF, `id="0" name="Lower"`, `id="2" name="Lower"`, 1)
	context := &parseContext{format: "GPIF"}
	if _, err := parseGPIFWithContext([]byte(data), context); err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range context.diagnostics {
		if diagnostic.Code == "GPIF.Beat.Chord.Reference" {
			return
		}
	}
	t.Fatalf("diagnostics = %#v, want wrong-staff chord reference", context.diagnostics)
}

func TestGPIFChordCollectionAndBaseFretRoundTrip(t *testing.T) {
	firstFret := uint8(5)
	song := syntheticGP8Song()
	song.Tracks[0].Measures[0].Voices[0].Beats[1].Effect.Chord = &Chord{
		Name: "Base-fret chord", FirstFret: &firstFret, Strings: []int8{5}, Length: 1,
	}
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	data = rewriteConformanceGPIF(t, data, func(gpif string) string {
		return moveFirstChordCollectionToTrack(t, gpif)
	})

	result, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	assertFirstChordBaseFret(t, result, firstFret, 5)

	reexported, err := Export(result, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	roundTripped, err := Parse(reexported)
	if err != nil {
		t.Fatal(err)
	}
	assertFirstChordBaseFret(t, roundTripped, firstFret, 5)
}

func moveFirstChordCollectionToTrack(t *testing.T, gpif string) string {
	t.Helper()
	start := strings.Index(gpif, `<Property name="DiagramCollection">`)
	if start < 0 {
		t.Fatal("GPIF has no diagram collection")
	}
	end := strings.Index(gpif[start:], "</Property>")
	if end < 0 {
		t.Fatal("GPIF diagram collection is not closed")
	}
	end += start + len("</Property>")
	collection := strings.Replace(gpif[start:end], `name="DiagramCollection"`, `name="ChordCollection"`, 1)
	gpif = gpif[:start] + gpif[end:]
	staves := strings.Index(gpif, "<Staves>")
	if staves < 0 {
		t.Fatal("GPIF track has no staves")
	}
	return gpif[:staves] + "<Properties>" + collection + "</Properties>" + gpif[staves:]
}

func assertFirstChordBaseFret(t *testing.T, song *Song, firstFret uint8, fret int8) {
	t.Helper()
	chord := song.Tracks[0].Measures[0].Voices[0].Beats[1].Effect.Chord
	if chord == nil || chord.FirstFret == nil || *chord.FirstFret != firstFret || !slices.Equal(chord.Strings, []int8{fret}) {
		t.Fatalf("chord = %#v, want first fret %d and strings [%d]", chord, firstFret, fret)
	}
}
