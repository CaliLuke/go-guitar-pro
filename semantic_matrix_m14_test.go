// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestSemanticMatrixM14ChordDefinitions(t *testing.T) {
	runSemanticMatrixM14ChordDefinitions(newSemanticMatrixRun(t))
}

func runSemanticMatrixM14ChordDefinitions(run *semanticMatrixRun) {
	t := run.t
	firstFret := uint8(3)
	root := PitchClass{Note: "C#", Just: 0, Accidental: 1, Value: 1, Sharp: true}
	bass := PitchClass{Note: "Eb", Just: 4, Accidental: -1, Value: 3, Sharp: false}
	kind := ChordType(2)
	extension := ChordExtensionThirteenth
	fifth := ChordAlterationAugmented
	ninth := ChordAlterationDiminished
	eleventh := ChordAlterationPerfect
	tonality := ChordAlterationDiminished
	add, sharp, newFormat, show := true, false, false, false
	chord := &Chord{
		Name:       "C#13/Eb",
		Length:     6,
		Strings:    []int8{3, -1, 0, 5, 4, 3},
		FirstFret:  &firstFret,
		Barres:     []Barre{{Fret: 3, Start: 1, End: 3}, {Fret: 5, Start: 4, End: 6}},
		Fingerings: []Fingering{FingeringLittle, FingeringOpen, FingeringOpen, FingeringAnnular, FingeringIndex, FingeringMiddle},
		Omissions:  []bool{true, false, true, false, false, true, false},
		Root:       &root,
		Bass:       &bass,
		Kind:       &kind,
		Extension:  &extension,
		Fifth:      &fifth,
		Ninth:      &ninth,
		Eleventh:   &eleventh,
		Tonality:   &tonality,
		Add:        &add,
		Sharp:      &sharp,
		NewFormat:  &newFormat,
		Show:       &show,
	}
	run.Field("Chord.Name", chord.Name, "C#13/Eb")
	run.Field("Chord.Length", chord.Length, uint8(6))
	run.Field("Chord.Strings", chord.Strings, []int8{3, -1, 0, 5, 4, 3})
	run.Field("Chord.FirstFret", *chord.FirstFret, uint8(3))
	run.Field("Chord.Barres", chord.Barres, []Barre{{Fret: 3, Start: 1, End: 3}, {Fret: 5, Start: 4, End: 6}})
	run.Field("Barre.Fret", []int8{chord.Barres[0].Fret, chord.Barres[1].Fret}, []int8{3, 5})
	run.Field("Barre.Start", []int8{chord.Barres[0].Start, chord.Barres[1].Start}, []int8{1, 4})
	run.Field("Barre.End", []int8{chord.Barres[0].End, chord.Barres[1].End}, []int8{3, 6})
	run.Field("Chord.Fingerings", chord.Fingerings, []Fingering{FingeringLittle, FingeringOpen, FingeringOpen, FingeringAnnular, FingeringIndex, FingeringMiddle})
	run.Field("Chord.Omissions", chord.Omissions, []bool{true, false, true, false, false, true, false})
	run.Field("Chord.Root", *chord.Root, root)
	run.Field("Chord.Bass", *chord.Bass, bass)
	run.Field("Chord.Kind", *chord.Kind, ChordType(2))
	run.Field("Chord.Extension", *chord.Extension, ChordExtensionThirteenth)
	run.Field("Chord.Fifth", *chord.Fifth, ChordAlterationAugmented)
	run.Field("Chord.Ninth", *chord.Ninth, ChordAlterationDiminished)
	run.Field("Chord.Eleventh", *chord.Eleventh, ChordAlterationPerfect)
	run.Field("Chord.Tonality", *chord.Tonality, ChordAlterationDiminished)
	run.Field("Chord.Add", *chord.Add, true)
	run.Field("Chord.Sharp", *chord.Sharp, false)
	run.Field("Chord.NewFormat", *chord.NewFormat, false)
	run.Field("Chord.Show", *chord.Show, false)
	for prefix, pitch := range map[string]PitchClass{"root": root, "bass": bass} {
		run.Field("PitchClass.Note", prefix+":"+pitch.Note, prefix+":"+map[string]string{"root": "C#", "bass": "Eb"}[prefix])
		run.Field("PitchClass.Just", pitch.Just, map[string]int8{"root": 0, "bass": 4}[prefix])
		run.Field("PitchClass.Accidental", pitch.Accidental, map[string]int8{"root": 1, "bass": -1}[prefix])
		run.Field("PitchClass.Value", pitch.Value, map[string]int8{"root": 1, "bass": 3}[prefix])
		run.Field("PitchClass.Sharp", pitch.Sharp, prefix == "root")
	}

	song := semanticValidPitchedGP8Song(t)
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	beat.Effect.Chord = chord
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	wantCodes := []string{
		"gp8.omit.chord-barres",
		"gp8.omit.chord-fingerings",
		"gp8.omit.chord-legacy-details",
		"gp8.omit.chord-omissions",
	}
	for _, code := range wantCodes {
		if !hasExportCode(report, code) {
			t.Errorf("report = %#v, want %s", report.Entries, code)
		}
	}
	strictData, _, strictErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	var lossErr *ExportLossError
	if len(strictData) != 0 || !errors.As(strictErr, &lossErr) {
		t.Fatalf("strict chord export = %d bytes, %v", len(strictData), strictErr)
	}
	data, exported, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: reportCodes(report)}})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(reportCodes(exported), reportCodes(report)) {
		t.Fatalf("preflight codes = %v, export codes = %v", reportCodes(report), reportCodes(exported))
	}
	wire := extractM14ChordWire(t, data)
	run.Wire("gpifItem.ID", wire.id, "0")
	run.Wire("gpifItem.Name", wire.name, "C#13/Eb")
	run.Wire("gpifItem.Diagram", wire.hasDiagram, true)
	run.Wire("gpifItem.Chord", wire.hasChord, true)
	run.Wire("gpifDiagram.StringCount", wire.stringCount, 6)
	run.Wire("gpifDiagram.FretCount", wire.fretCount, 5)
	run.Wire("gpifDiagram.BaseFret", wire.baseFret, 2)
	run.Wire("gpifDiagram.Frets", wire.frets, []m14WireFret{{String: 5, Fret: 1}, {String: 3, Fret: -2}, {String: 2, Fret: 3}, {String: 1, Fret: 2}, {String: 0, Fret: 1}})
	run.Wire("gpifDiagramFret.String", m14WireFretStrings(wire.frets), []int{5, 3, 2, 1, 0})
	run.Wire("gpifDiagramFret.Fret", m14WireFretValues(wire.frets), []int{1, -2, 3, 2, 1})
	run.Wire("gpifDiagram.Fingering", wire.hasFingering, false)
	run.Wire("gpifDiagram.Properties", wire.propertyNames, []string(nil))
	if wire.chordChildCount != 1 || wire.chordDetailCount != 0 {
		t.Fatalf("wire chord children = %d details = %d, want marker only", wire.chordChildCount, wire.chordDetailCount)
	}
	run.Wire("gpifBeat.Chord", extractGPIFLeafText(t, data)["GPIF/Beats/Beat/Chord"], "01")

	sourceGPIF := strings.Replace(semanticM14ScopedGPIF, `</Diagram></Item>`, `<Fingering><Position fret="1" finger="Index"/><Position fret="2" finger="Thumb"/></Fingering><Property name="ShowName" type="bool" value="false"/></Diagram></Item>`, 1)
	sourceWire := decodeM14ChordWire(t, []byte(sourceGPIF))
	run.Wire("gpifDiagram.Fingering", sourceWire.hasFingering, true)
	run.Wire("gpifDiagramFingering.Positions", sourceWire.positions, []m14WirePosition{{Fret: 1, Finger: "Index"}, {Fret: 2, Finger: "Thumb"}})
	run.Wire("gpifDiagramPosition.Fret", []int{sourceWire.positions[0].Fret, sourceWire.positions[1].Fret}, []int{1, 2})
	run.Wire("gpifDiagramPosition.Finger", []string{sourceWire.positions[0].Finger, sourceWire.positions[1].Finger}, []string{"Index", "Thumb"})
	run.Wire("gpifDiagram.Properties", sourceWire.properties, []m14WireProperty{{Name: "ShowName", Type: "bool", Value: "false"}})
	run.Wire("gpifDiagramProperty.Name", sourceWire.properties[0].Name, "ShowName")
	run.Wire("gpifDiagramProperty.Type", sourceWire.properties[0].Type, "bool")
	run.Wire("gpifDiagramProperty.Value", sourceWire.properties[0].Value, "false")
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Chord
	if got == nil {
		t.Fatal("round-trip chord is nil")
	}
	run.Field("Chord.Name", got.Name, chord.Name)
	run.Field("Chord.Length", got.Length, chord.Length)
	run.Field("Chord.Strings", got.Strings, chord.Strings)
	run.Field("Chord.FirstFret", *got.FirstFret, *chord.FirstFret)
	if len(got.Barres) != 0 || len(got.Fingerings) != 0 || len(got.Omissions) != 0 || got.Root != nil || got.Bass != nil || got.Kind != nil || got.Show != nil {
		t.Fatalf("round-trip kept omitted chord details: %#v", got)
	}
}

func TestSemanticMatrixM14ChordDefaultsAndLength(t *testing.T) {
	runSemanticMatrixM14ChordDefaultsAndLength(newSemanticMatrixRun(t))
}

func runSemanticMatrixM14ChordDefaultsAndLength(run *semanticMatrixRun) {
	t := run.t
	gChord := Chord{
		Name:       "G",
		Length:     6,
		Strings:    []int8{3, 0, 0, 0, 2, 3},
		Fingerings: []Fingering{FingeringLittle, FingeringOpen, FingeringOpen, FingeringOpen, FingeringIndex, FingeringMiddle},
	}
	run.Field("Chord.Name", gChord.Name, "G")
	run.Field("Chord.Length", gChord.Length, uint8(6))
	run.Field("Chord.Strings", gChord.Strings, []int8{3, 0, 0, 0, 2, 3})
	run.Field("Chord.Fingerings", gChord.Fingerings, []Fingering{FingeringLittle, FingeringOpen, FingeringOpen, FingeringOpen, FingeringIndex, FingeringMiddle})
	allFingerings := Chord{Fingerings: []Fingering{FingeringOpen, FingeringThumb, FingeringIndex, FingeringMiddle, FingeringAnnular, FingeringLittle}}
	run.Field("Chord.Fingerings", allFingerings.Fingerings, []Fingering{-1, 0, 1, 2, 3, 4})
	empty := Chord{}
	run.Field("Chord.Add", empty.Add, (*bool)(nil))
	run.Field("Chord.Bass", empty.Bass, (*PitchClass)(nil))
	run.Field("Chord.Eleventh", empty.Eleventh, (*ChordAlteration)(nil))
	run.Field("Chord.Extension", empty.Extension, (*ChordExtension)(nil))
	run.Field("Chord.Fifth", empty.Fifth, (*ChordAlteration)(nil))
	run.Field("Chord.FirstFret", empty.FirstFret, (*uint8)(nil))
	run.Field("Chord.Kind", empty.Kind, (*ChordType)(nil))
	run.Field("Chord.NewFormat", empty.NewFormat, (*bool)(nil))
	run.Field("Chord.Ninth", empty.Ninth, (*ChordAlteration)(nil))
	run.Field("Chord.Root", empty.Root, (*PitchClass)(nil))
	run.Field("Chord.Sharp", empty.Sharp, (*bool)(nil))
	run.Field("Chord.Show", empty.Show, (*bool)(nil))
	run.Field("Chord.Tonality", empty.Tonality, (*ChordAlteration)(nil))
	for _, test := range []struct {
		name      string
		firstFret *uint8
		firstCode bool
	}{{name: "absent"}, {name: "explicit zero", firstFret: ptrTo(uint8(0)), firstCode: true}} {
		t.Run(test.name, func(t *testing.T) {
			song := semanticValidPitchedGP8Song(t)
			chord := &Chord{Name: test.name, Length: 7, Strings: []int8{-1, 0, 2, 2, 1, 0}, FirstFret: test.firstFret}
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Chord = chord
			report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
			wantCodes := []string{"gp8.normalize.chord-length"}
			if test.firstCode {
				wantCodes = append(wantCodes, "gp8.normalize.chord-first-fret")
			} else if hasExportCode(report, "gp8.normalize.chord-first-fret") {
				t.Errorf("absent first fret produced an authored-value normalization: %#v", report.Entries)
			}
			for _, code := range wantCodes {
				if !hasExportCode(report, code) {
					t.Errorf("report = %#v, want %s", report.Entries, code)
				}
			}
			run.Field("Chord.FirstFret", chord.FirstFret, test.firstFret)
			run.Field("Chord.Length", chord.Length, uint8(7))
		})
	}
}

func TestSemanticMatrixM14ChordScopeAndIsolation(t *testing.T) {
	runSemanticMatrixM14ChordScopeAndIsolation(newSemanticMatrixRun(t))
}

func runSemanticMatrixM14ChordScopeAndIsolation(run *semanticMatrixRun) {
	t := run.t
	for _, name := range []string{"DiagramCollection", "ChordCollection"} {
		definitions := make(map[string]Chord)
		err := gpifReadChordProperties([]gpifStaffProperty{{Name: name, Items: &gpifItems{Items: []gpifItem{{ID: "definition", Name: name}}}}}, definitions)
		if err != nil {
			t.Fatal(err)
		}
		run.Dispatch("gpifReadChordProperties:property.Name", definitions["definition"].Name, name)
		context := &parseContext{format: "GP8"}
		gpifAuditChordIDs(context, make(map[string]struct{}), []gpifStaffProperty{{Name: name, Items: &gpifItems{Items: []gpifItem{{ID: "definition", Name: name}}}}}, "/GPIF/Properties")
		run.Dispatch("gpifAuditChordIDs:property.Name", len(context.diagnostics), 0)
	}
	result, err := ParseWithOptions(conformanceGPIFArchive(t, semanticM14ScopedGPIF), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	track := &result.Song.Tracks[0]
	upper := track.Staves[0].Measures[0].Voices[0].Beats
	lower := track.Staves[1].Measures[0].Voices[0].Beats
	if len(upper) != 5 || len(lower) != 2 {
		t.Fatalf("scoped beats = %d, %d; want 5, 2", len(upper), len(lower))
	}
	run.Field("BeatEffects.Chord", []string{upper[0].Effect.Chord.Name, upper[1].Effect.Chord.Name, lower[0].Effect.Chord.Name, lower[1].Effect.Chord.Name}, []string{"Upper", "Track", "Lower", "Track"})
	run.Field("Chord.Strings", []int8{upper[0].Effect.Chord.Strings[0], lower[0].Effect.Chord.Strings[0]}, []int8{1, 2})
	if upper[3].Effect.Chord == upper[4].Effect.Chord || !reflect.DeepEqual(*upper[3].Effect.Chord, *upper[4].Effect.Chord) {
		t.Fatalf("equal definitions = %#v, %#v; want equal content and independent occurrences", upper[3].Effect.Chord, upper[4].Effect.Chord)
	}
	if upper[0].Effect.Chord == upper[2].Effect.Chord || &upper[0].Effect.Chord.Strings[0] == &upper[2].Effect.Chord.Strings[0] || upper[0].Effect.Chord.FirstFret == upper[2].Effect.Chord.FirstFret {
		t.Fatal("repeated chord references share mutable occurrence data")
	}
	upper[0].Effect.Chord.Strings[0] = 7
	*upper[0].Effect.Chord.FirstFret = 5
	if upper[2].Effect.Chord.Strings[0] != 1 || *upper[2].Effect.Chord.FirstFret != 1 {
		t.Fatalf("second occurrence changed through first: %#v", upper[2].Effect.Chord)
	}
	data, err := Export(result.Song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	roundTripUpper := roundTrip.Tracks[0].Staves[0].Measures[0].Voices[0].Beats
	run.Field("Chord.Strings", []int8{roundTripUpper[0].Effect.Chord.Strings[0], roundTripUpper[2].Effect.Chord.Strings[0]}, []int8{7, 1})
	run.Field("Chord.FirstFret", []uint8{*roundTripUpper[0].Effect.Chord.FirstFret, *roundTripUpper[2].Effect.Chord.FirstFret}, []uint8{5, 1})

	reordered := strings.Replace(semanticM14ScopedGPIF,
		`<Item id="track" name="Track"><Diagram stringCount="1" fretCount="4" baseFret="0"><Fret string="0" fret="3"/></Diagram></Item><Item id="equal-a" name="Equal"><Diagram stringCount="1" fretCount="4" baseFret="0"><Fret string="0" fret="4"/></Diagram></Item><Item id="equal-b" name="Equal"><Diagram stringCount="1" fretCount="4" baseFret="0"><Fret string="0" fret="4"/></Diagram></Item>`,
		`<Item id="equal-b" name="Equal"><Diagram stringCount="1" fretCount="4" baseFret="0"><Fret string="0" fret="4"/></Diagram></Item><Item id="track" name="Track"><Diagram stringCount="1" fretCount="4" baseFret="0"><Fret string="0" fret="3"/></Diagram></Item><Item id="equal-a" name="Equal"><Diagram stringCount="1" fretCount="4" baseFret="0"><Fret string="0" fret="4"/></Diagram></Item>`, 1)
	reorderedSong, err := Parse(conformanceGPIFArchive(t, reordered))
	if err != nil {
		t.Fatal(err)
	}
	reorderedUpper := reorderedSong.Tracks[0].Staves[0].Measures[0].Voices[0].Beats
	if !reflect.DeepEqual(chordNamesAndFrets(upper[1:]), chordNamesAndFrets(reorderedUpper[1:])) {
		t.Fatalf("definition reorder changed referenced semantics: %v != %v", chordNamesAndFrets(upper[1:]), chordNamesAndFrets(reorderedUpper[1:]))
	}

	for _, test := range []struct {
		name string
		data string
		code string
	}{
		{name: "missing reference", data: strings.Replace(semanticM14ScopedGPIF, "<Chord>track</Chord>", "<Chord>missing</Chord>", 1), code: "GPIF.Beat.Chord.Reference"},
		{name: "empty id", data: strings.Replace(semanticM14ScopedGPIF, `id="track" name="Track"`, `id="" name="Track"`, 1), code: "GPIF.ChordDefinition.EmptyID"},
		{name: "duplicate id", data: strings.Replace(semanticM14ScopedGPIF, `id="equal-a"`, `id="track"`, 1), code: "GPIF.ChordDefinition.DuplicateID"},
		{name: "fingering", data: strings.Replace(semanticM14ScopedGPIF, `</Diagram></Item>`, `<Fingering><Position fret="1" finger="Index"/></Fingering></Diagram></Item>`, 1), code: "GPIF.Chord.Diagram.Fingering"},
		{name: "show name", data: strings.Replace(semanticM14ScopedGPIF, `</Diagram></Item>`, `<Property name="ShowName" type="bool" value="false"/></Diagram></Item>`, 1), code: "GPIF.Chord.Diagram.Property.ShowName"},
		{name: "show diagram", data: strings.Replace(semanticM14ScopedGPIF, `</Diagram></Item>`, `<Property name="ShowDiagram" type="bool" value="false"/></Diagram></Item>`, 1), code: "GPIF.Chord.Diagram.Property.ShowDiagram"},
		{name: "show fingering", data: strings.Replace(semanticM14ScopedGPIF, `</Diagram></Item>`, `<Property name="ShowFingering" type="bool" value="false"/></Diagram></Item>`, 1), code: "GPIF.Chord.Diagram.Property.ShowFingering"},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsed, parseErr := ParseWithOptions(conformanceGPIFArchive(t, test.data), ParseOptions{})
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			if !slices.ContainsFunc(parsed.Diagnostics, func(d ParseDiagnostic) bool { return d.Code == test.code }) {
				t.Fatalf("diagnostics = %#v, want %s", parsed.Diagnostics, test.code)
			}
		})
	}
}

func ptrTo[T any](value T) *T { return &value }

func chordNamesAndFrets(beats []Beat) []any {
	result := make([]any, 0, len(beats)*2)
	for index := range beats {
		if beats[index].Effect.Chord == nil {
			result = append(result, nil, nil)
			continue
		}
		result = append(result, beats[index].Effect.Chord.Name, slices.Clone(beats[index].Effect.Chord.Strings))
	}
	return result
}

type m14WireFret struct {
	String int
	Fret   int
}

func m14WireFretStrings(frets []m14WireFret) []int {
	result := make([]int, len(frets))
	for index := range frets {
		result[index] = frets[index].String
	}
	return result
}

func m14WireFretValues(frets []m14WireFret) []int {
	result := make([]int, len(frets))
	for index := range frets {
		result[index] = frets[index].Fret
	}
	return result
}

type m14WirePosition struct {
	Fret   int
	Finger string
}

type m14WireProperty struct {
	Name  string
	Type  string
	Value string
}

type m14ChordWire struct {
	name             string
	id               string
	hasDiagram       bool
	hasChord         bool
	stringCount      int
	fretCount        int
	baseFret         int
	frets            []m14WireFret
	hasFingering     bool
	propertyNames    []string
	positions        []m14WirePosition
	properties       []m14WireProperty
	chordChildCount  int
	chordDetailCount int
}

func extractM14ChordWire(t *testing.T, data []byte) m14ChordWire {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	gpif := readZipMember(t, archive, "Content/score.gpif")
	return decodeM14ChordWire(t, gpif)
}

func decodeM14ChordWire(t *testing.T, gpif []byte) m14ChordWire {
	t.Helper()
	decoder := xml.NewDecoder(bytes.NewReader(gpif))
	var result m14ChordWire
	depth, chordDepth := 0, -1
	inFirstItem := false
	for {
		token, decodeErr := decoder.Token()
		if errors.Is(decodeErr, io.EOF) {
			break
		}
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		switch value := token.(type) {
		case xml.StartElement:
			depth++
			if value.Name.Local == "Item" && result.name == "" {
				inFirstItem = true
				result.id = m14XMLAttribute(value, "id")
				result.name = m14XMLAttribute(value, "name")
			}
			if !inFirstItem {
				continue
			}
			switch value.Name.Local {
			case "Diagram":
				result.hasDiagram = true
				result.stringCount = m14XMLIntAttribute(t, value, "stringCount")
				result.fretCount = m14XMLIntAttribute(t, value, "fretCount")
				result.baseFret = m14XMLIntAttribute(t, value, "baseFret")
			case "Fret":
				result.frets = append(result.frets, m14WireFret{String: m14XMLIntAttribute(t, value, "string"), Fret: m14XMLIntAttribute(t, value, "fret")})
			case "Fingering":
				result.hasFingering = true
			case "Position":
				result.positions = append(result.positions, m14WirePosition{Fret: m14XMLIntAttribute(t, value, "fret"), Finger: m14XMLAttribute(value, "finger")})
			case "Property":
				property := m14WireProperty{Name: m14XMLAttribute(value, "name"), Type: m14XMLAttribute(value, "type"), Value: m14XMLAttribute(value, "value")}
				result.propertyNames = append(result.propertyNames, property.Name)
				result.properties = append(result.properties, property)
			case "Chord":
				result.hasChord = true
				result.chordChildCount++
				chordDepth = depth
			default:
				if chordDepth >= 0 && depth > chordDepth {
					result.chordDetailCount++
				}
			}
		case xml.EndElement:
			if inFirstItem && value.Name.Local == "Item" {
				return result
			}
			if chordDepth == depth && value.Name.Local == "Chord" {
				chordDepth = -1
			}
			depth--
		}
	}
	t.Fatal("GP8 archive has no chord item")
	return m14ChordWire{}
}

func m14XMLAttribute(element xml.StartElement, name string) string {
	for _, attribute := range element.Attr {
		if attribute.Name.Local == name {
			return attribute.Value
		}
	}
	return ""
}

func m14XMLIntAttribute(t *testing.T, element xml.StartElement, name string) int {
	t.Helper()
	var value int
	if _, err := fmt.Sscan(m14XMLAttribute(element, name), &value); err != nil {
		t.Fatalf("%s %s: %v", element.Name.Local, name, err)
	}
	return value
}

const semanticM14ScopedGPIF = `<?xml version="1.0" encoding="utf-8"?>
<GPIF>
  <GPVersion>8.0</GPVersion>
  <Score><Title>Scoped chords</Title></Score>
  <MasterTrack><Tracks>0</Tracks></MasterTrack>
  <Tracks><Track id="0"><Name>Piano</Name><Properties><Property name="ChordCollection"><Items><Item id="track" name="Track"><Diagram stringCount="1" fretCount="4" baseFret="0"><Fret string="0" fret="3"/></Diagram></Item><Item id="equal-a" name="Equal"><Diagram stringCount="1" fretCount="4" baseFret="0"><Fret string="0" fret="4"/></Diagram></Item><Item id="equal-b" name="Equal"><Diagram stringCount="1" fretCount="4" baseFret="0"><Fret string="0" fret="4"/></Diagram></Item><Item id="shared" name="Track shared"><Diagram stringCount="1" fretCount="4" baseFret="0"><Fret string="0" fret="6"/></Diagram></Item></Items></Property></Properties><Staves>
    <Staff><Properties><Property name="DiagramCollection"><Items><Item id="shared" name="Upper"><Diagram stringCount="1" fretCount="4" baseFret="0"><Fret string="0" fret="1"/></Diagram></Item></Items></Property></Properties></Staff>
    <Staff><Properties><Property name="DiagramCollection"><Items><Item id="shared" name="Lower"><Diagram stringCount="1" fretCount="4" baseFret="0"><Fret string="0" fret="2"/></Diagram></Item></Items></Property></Properties></Staff>
  </Staves><MidiConnection><Port>0</Port><PrimaryChannel>0</PrimaryChannel><SecondaryChannel>1</SecondaryChannel></MidiConnection></Track></Tracks>
  <MasterBars><MasterBar><Time>4/4</Time><Bars>0 1</Bars></MasterBar></MasterBars>
  <Bars><Bar id="0"><Clef>G2</Clef><Voices>0</Voices></Bar><Bar id="1"><Clef>F4</Clef><Voices>1</Voices></Bar></Bars>
  <Voices><Voice id="0"><Beats>0 1 2 5 6</Beats></Voice><Voice id="1"><Beats>3 4</Beats></Voice></Voices>
  <Beats><Beat id="0"><Rhythm ref="0"/><Chord>shared</Chord></Beat><Beat id="1"><Rhythm ref="0"/><Chord>track</Chord></Beat><Beat id="2"><Rhythm ref="0"/><Chord>shared</Chord></Beat><Beat id="3"><Rhythm ref="0"/><Chord>shared</Chord></Beat><Beat id="4"><Rhythm ref="0"/><Chord>track</Chord></Beat><Beat id="5"><Rhythm ref="0"/><Chord>equal-a</Chord></Beat><Beat id="6"><Rhythm ref="0"/><Chord>equal-b</Chord></Beat></Beats>
  <Notes/><Rhythms><Rhythm id="0"><NoteValue>Quarter</NoteValue></Rhythm></Rhythms>
</GPIF>`
