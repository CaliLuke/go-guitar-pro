// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/xml"
	"os"
	"strings"
	"testing"
)

// Expected GP6 input identities in element-major order (three variations each).
var gp6ExpectedArticulations = []int16{
	36, 36, 36, 38, 91, 37, 99, 36, 36, 56, 36, 36, 102, 36, 36,
	43, 36, 36, 45, 36, 36, 47, 36, 36, 48, 36, 36, 50, 36, 36,
	42, 92, 46, 44, 36, 36, 57, 36, 36, 49, 36, 36, 55, 36, 36,
	51, 93, 53, 52, 36, 36,
}

func TestGP6PercussionElementVariations(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp6/percussion-elements.gpx")
	track := &song.Tracks[0]
	for element := range 17 {
		variations := gp6ExpectedArticulations[element*3 : element*3+3]
		for variation, want := range variations {
			note := track.Measures[element].Voices[0].Beats[variation].Notes[0]
			if note.Value != want {
				t.Fatalf("element %d variation %d value=%d, want %d", element, variation, note.Value, want)
			}
			if !note.HasPercussionArticulation || note.PercussionArticulation >= len(track.PercussionArticulations) {
				t.Fatalf("unresolved articulation: %+v", note)
			}
		}
	}
}

func TestAlphaTabGP6PercussionElementVariations(t *testing.T) {
	requireAlphaTabConformance(t)
	source := "testdata/gp6/percussion-elements.gpx"
	song := parseTestFixture(t, source)
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	score := readAlphaTabScore(t, writeConformanceFixture(t, data)).(map[string]any)
	track := score["tracks"].([]any)[0].(map[string]any)
	bars := track["staves"].([]any)[0].(map[string]any)["bars"].([]any)
	for element := range 17 {
		variations := gp6ExpectedArticulations[element*3 : element*3+3]
		beats := bars[element].(map[string]any)["voices"].([]any)[0].(map[string]any)["beats"].([]any)
		for variation, input := range variations {
			note := beats[variation].(map[string]any)["notes"].([]any)[0].(map[string]any)
			want := int(input)
			if output, ok := map[int16]int{91: 38, 92: 46, 93: 51, 99: 56, 102: 56}[input]; ok {
				want = output
			}
			if note["midi"] != float64(want) {
				t.Errorf("element %d variation %d MIDI=%v want %d", element, variation, note["midi"], want)
			}
		}
	}
}

func TestGP6PercussionNativeConversion(t *testing.T) {
	source := parseTestFixture(t, "testdata/gp6/percussion-elements.gpx")
	native := parseTestFixture(t, "testdata/gp8/gp6-percussion-elements.gp")
	for element := range 17 {
		for variation := range 3 {
			got := source.Tracks[0].Measures[element].Voices[0].Beats[variation].Notes[0]
			want := native.Tracks[0].Measures[element].Voices[0].Beats[variation].Notes[0]
			if got.Value != want.Value {
				t.Errorf("element %d variation %d input=%d native=%d", element, variation, got.Value, want.Value)
			}
			gotDefinition := source.Tracks[0].PercussionArticulations[got.PercussionArticulation]
			wantDefinition := native.Tracks[0].PercussionArticulations[want.PercussionArticulation]
			if !gpifSamePercussionArticulation(gotDefinition, wantDefinition) {
				t.Errorf("element %d variation %d differs from native notation/playback", element, variation)
			}
		}
	}
}

func TestConformanceGP6Percussion(t *testing.T) { runConformanceGP6Percussion(newConformanceRun(t)) }

func runConformanceGP6Percussion(run *conformanceRun) {
	t := run.t
	source, err := os.ReadFile("testdata/gp6/percussion-elements.gpx")
	if err != nil {
		t.Fatal(err)
	}
	files, err := gpxReadFiles(source)
	if err != nil {
		t.Fatal(err)
	}
	var wire gpifDocument
	if unmarshalErr := xml.Unmarshal(files["score.gpif"], &wire); unmarshalErr != nil {
		t.Fatal(unmarshalErr)
	}
	song, err := Parse(source)
	if err != nil {
		t.Fatal(err)
	}
	for element := range 17 {
		variations := gp6ExpectedArticulations[element*3 : element*3+3]
		for variation, input := range variations {
			properties := wire.Notes.Notes[element*3+variation].Properties.Properties
			run.Wire("gpifProperty.Element", *properties[0].Element, element)
			run.Wire("gpifProperty.Variation", *properties[1].Variation, variation)
			note := song.Tracks[0].Measures[element].Voices[0].Beats[variation].Notes[0]
			run.Field("Note.Value", note.Value, input)
			run.Dispatch("gpifApplyPercussionElement:property.Name", note.Value, input)
		}
	}
	TestGP6PercussionElementVariations(t)
}

func TestGP6PercussionRejectsInvalidPairs(t *testing.T) {
	for _, values := range [][2]int{{-1, 0}, {17, 0}, {0, -1}, {0, 3}} {
		note := gpifNote{Properties: gpifProperties{Properties: []gpifProperty{{Name: "Element", Element: &values[0]}, {Name: "Variation", Variation: &values[1]}}}}
		if _, err := gpifNoteToNote(&note, 0, true); err == nil {
			t.Errorf("accepted invalid pair %v", values)
		}
	}
	for _, name := range []string{"Element", "Variation"} {
		value := 1
		property := gpifProperty{Name: name}
		if name == "Element" {
			property.Element = &value
		} else {
			property.Variation = &value
		}
		source := gpifNote{Properties: gpifProperties{Properties: []gpifProperty{property}}}
		if _, err := gpifNoteToNote(&source, 0, true); err == nil {
			t.Errorf("accepted incomplete %s", name)
		}
	}
}

func TestGP6ExtendedPercussionDefinitions(t *testing.T) {
	native := parseTestFixture(t, "testdata/gp7/percussion.gp")
	for _, input := range []int{91, 93} {
		var want *PercussionArticulation
		for index := range native.Tracks[0].PercussionArticulations {
			a := &native.Tracks[0].PercussionArticulations[index]
			if len(a.InputMIDINumbers) > 0 && a.InputMIDINumbers[0] == input {
				want = a
				break
			}
		}
		if want == nil {
			t.Fatalf("native input %d missing", input)
		}
		wire := gp8DrumElement(int16(input), GP8ExportOptions{})
		got := gpifReadPercussionArticulations(&gpifInstrumentSet{Elements: gpifElements{Elements: []gpifElement{wire}}}, nil)
		if len(got) != 1 || !gpifSamePercussionArticulation(got[0], *want) {
			t.Errorf("input %d definition differs from native GP7", input)
		}
	}
}

func TestGP6PercussionBuiltinDoesNotAliasCustomTable(t *testing.T) {
	track := Track{PercussionArticulations: make([]PercussionArticulation, 128)}
	note := Note{Value: 35, PercussionArticulation: 35, HasPercussionArticulation: true}
	fallbacks := gpifPercussionFallbacks{tableLength: 128}
	gpifNormalizePercussionArticulation(&track, &note, &fallbacks, true)
	if note.PercussionArticulation != 128 || track.PercussionArticulations[128].OutputMIDINumber != 35 {
		t.Fatal("builtin kick resolved as a custom articulation")
	}
}

func TestGP6PercussionGraceAndAuthority(t *testing.T) {
	source, err := os.ReadFile("testdata/gp6/percussion-elements.gpx")
	if err != nil {
		t.Fatal(err)
	}
	files, err := gpxReadFiles(source)
	if err != nil {
		t.Fatal(err)
	}
	wire := strings.Replace(string(files["score.gpif"]), `<Beat id="0">`, `<Beat id="0"><GraceNotes>BeforeBeat</GraceNotes>`, 1)
	song, err := Parse(conformanceGPIFArchive(t, wire))
	if err != nil {
		t.Fatal(err)
	}
	graces := song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces
	if len(graces) != 1 || graces[0].Fret != 36 || !graces[0].HasPercussionArticulation {
		t.Fatalf("lost GP6 grace identity: %+v", graces)
	}
	element, variation, midi, explicit := 10, 1, 0, 2
	properties := []gpifProperty{{Name: "Element", Element: &element}, {Name: "Variation", Variation: &variation}, {Name: "Midi", Number: &midi}}
	note := gpifNote{Properties: gpifProperties{Properties: properties}}
	converted, err := gpifNoteToNote(&note, 0, true)
	if err != nil || converted.Value != 92 {
		t.Fatalf("GP6 pair did not own input identity: %+v, %v", converted, err)
	}
	note.InstrumentArticulation = &explicit
	converted, err = gpifNoteToNote(&note, 0, true)
	if err != nil || converted.PercussionArticulation != explicit {
		t.Fatalf("explicit articulation lost authority: %+v, %v", converted, err)
	}
}
