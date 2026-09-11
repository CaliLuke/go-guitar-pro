// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestConformanceChordDefinitions(t *testing.T) {
	runConformanceChordDefinitions(newConformanceRun(t))
}

func runConformanceChordDefinitions(run *conformanceRun) {
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
		Length:     7,
		Strings:    []int8{3, 3, -1, 0, 5, 5, 7},
		FirstFret:  &firstFret,
		Barres:     []Barre{{Fret: 3, Start: 1, End: 2}, {Fret: 5, Start: 5, End: 6}},
		Fingerings: []Fingering{FingeringIndex, FingeringIndex, FingeringOpen, FingeringOpen, FingeringAnnular, FingeringAnnular, FingeringThumb},
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
	run.ClaimPrimary(claimSite("chord-name", "model", "M14-CHORD-DEFINITIONS", "C#13/Eb name")).Preserved("Chord.Name", chord.Name, "C#13/Eb")
	run.Preserved("Chord.Length", chord.Length, uint8(7))
	run.Preserved("Chord.Strings", chord.Strings, []int8{3, 3, -1, 0, 5, 5, 7})
	run.Preserved("Chord.FirstFret", *chord.FirstFret, uint8(3))
	run.ClaimPrimary(claimSite("chord-diagram", "model", "M14-CHORD-DEFINITIONS", "two barre ranges and seven strings")).Preserved("Chord.Barres", chord.Barres, []Barre{{Fret: 3, Start: 1, End: 2}, {Fret: 5, Start: 5, End: 6}})
	run.Preserved("Barre.Fret", []int8{chord.Barres[0].Fret, chord.Barres[1].Fret}, []int8{3, 5})
	run.Preserved("Barre.Start", []int8{chord.Barres[0].Start, chord.Barres[1].Start}, []int8{1, 5})
	run.Preserved("Barre.End", []int8{chord.Barres[0].End, chord.Barres[1].End}, []int8{2, 6})
	run.Preserved("Chord.Fingerings", chord.Fingerings, []Fingering{FingeringIndex, FingeringIndex, FingeringOpen, FingeringOpen, FingeringAnnular, FingeringAnnular, FingeringThumb})
	run.Enum("Fingering.FingeringUnknown", FingeringUnknown, Fingering(-2))
	run.Omitted("Chord.Omissions", chord.Omissions, []bool{true, false, true, false, false, true, false})
	run.Omitted("Chord.Root", *chord.Root, root)
	run.Omitted("Chord.Bass", *chord.Bass, bass)
	run.Omitted("Chord.Kind", *chord.Kind, ChordType(2))
	run.Omitted("Chord.Extension", *chord.Extension, ChordExtensionThirteenth)
	run.Omitted("Chord.Fifth", *chord.Fifth, ChordAlterationAugmented)
	run.Omitted("Chord.Ninth", *chord.Ninth, ChordAlterationDiminished)
	run.Omitted("Chord.Eleventh", *chord.Eleventh, ChordAlterationPerfect)
	run.Omitted("Chord.Tonality", *chord.Tonality, ChordAlterationDiminished)
	run.Omitted("Chord.Add", *chord.Add, true)
	run.Omitted("Chord.Sharp", *chord.Sharp, false)
	run.Omitted("Chord.NewFormat", *chord.NewFormat, false)
	run.Omitted("Chord.Show", *chord.Show, false)
	for prefix, pitch := range map[string]PitchClass{"root": root, "bass": bass} {
		run.Field("PitchClass.Note", prefix+":"+pitch.Note, prefix+":"+map[string]string{"root": "C#", "bass": "Eb"}[prefix])
		run.Field("PitchClass.Just", pitch.Just, map[string]int8{"root": 0, "bass": 4}[prefix])
		run.Field("PitchClass.Accidental", pitch.Accidental, map[string]int8{"root": 1, "bass": -1}[prefix])
		run.Field("PitchClass.Value", pitch.Value, map[string]int8{"root": 1, "bass": 3}[prefix])
		run.Field("PitchClass.Sharp", pitch.Sharp, prefix == "root")
	}
	for _, test := range []struct {
		member string
		apply  func(*Chord)
	}{
		{"ChordAlteration.ChordAlterationPerfect", func(chord *Chord) { value := ChordAlterationPerfect; chord.Fifth = &value }},
		{"ChordAlteration.ChordAlterationDiminished", func(chord *Chord) { value := ChordAlterationDiminished; chord.Fifth = &value }},
		{"ChordAlteration.ChordAlterationAugmented", func(chord *Chord) { value := ChordAlterationAugmented; chord.Fifth = &value }},
		{"ChordExtension.ChordExtensionNone", func(chord *Chord) { value := ChordExtensionNone; chord.Extension = &value }},
		{"ChordExtension.ChordExtensionNinth", func(chord *Chord) { value := ChordExtensionNinth; chord.Extension = &value }},
		{"ChordExtension.ChordExtensionEleventh", func(chord *Chord) { value := ChordExtensionEleventh; chord.Extension = &value }},
		{"ChordExtension.ChordExtensionThirteenth", func(chord *Chord) { value := ChordExtensionThirteenth; chord.Extension = &value }},
	} {
		probe := semanticValidPitchedGP8Song(t)
		probeChord := &Chord{Name: "C", Length: 6, Strings: []int8{0, 1, 0, 2, 3, -1}}
		test.apply(probeChord)
		probe.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Chord = probeChord
		probeReport := PreflightExport(probe, ExportFormatGP8, ExportOptions{})
		run.Enum(test.member, hasExportCode(probeReport, "gp8.omit.chord-legacy-details"), true)
	}
	for _, field := range []struct {
		name string
		set  func(*Chord)
	}{
		{"root", func(chord *Chord) { value := root; chord.Root = &value }},
		{"bass", func(chord *Chord) { value := bass; chord.Bass = &value }},
		{"kind", func(chord *Chord) { value := kind; chord.Kind = &value }},
		{"extension", func(chord *Chord) { value := extension; chord.Extension = &value }},
		{"fifth", func(chord *Chord) { value := fifth; chord.Fifth = &value }},
		{"ninth", func(chord *Chord) { value := ninth; chord.Ninth = &value }},
		{"eleventh", func(chord *Chord) { value := eleventh; chord.Eleventh = &value }},
		{"tonality", func(chord *Chord) { value := tonality; chord.Tonality = &value }},
		{"add", func(chord *Chord) { value := add; chord.Add = &value }},
		{"sharp", func(chord *Chord) { value := sharp; chord.Sharp = &value }},
		{"new format", func(chord *Chord) { value := newFormat; chord.NewFormat = &value }},
		{"show", func(chord *Chord) { value := show; chord.Show = &value }},
	} {
		t.Run("isolated "+field.name+" loss", func(t *testing.T) {
			probe := semanticValidPitchedGP8Song(t)
			probeChord := &Chord{Name: "C", Length: 6, Strings: []int8{0, 1, 0, 2, 3, -1}}
			field.set(probeChord)
			probe.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Chord = probeChord
			if report := PreflightExport(probe, ExportFormatGP8, ExportOptions{}); !hasExportCode(report, "gp8.omit.chord-legacy-details") {
				t.Fatalf("isolated %s report = %#v, want gp8.omit.chord-legacy-details", field.name, report.Entries)
			}
			data, _, err := ExportWithReport(probe, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
			if len(data) != 0 || err == nil {
				t.Fatalf("isolated %s strict export = %d bytes, %v", field.name, len(data), err)
			}
		})
	}

	song := semanticValidPitchedGP8Song(t)
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	beat.Effect.Chord = chord
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	wantCodes := []string{
		"gp8.omit.chord-legacy-details",
		"gp8.omit.chord-omissions",
	}
	for _, code := range wantCodes {
		if !hasExportCode(report, code) {
			t.Errorf("report = %#v, want %s", report.Entries, code)
		}
	}
	run.ClaimReport(claimSite("chord-name", "export", "M14-CHORD-DEFINITIONS", "C#13/Eb name")).Report("M14-CHORD-DEFINITIONS", reportCodes(report), []string{"gp8.normalize.track-view", "gp8.omit.chord-annular-consumer-barre", "gp8.omit.chord-omissions", "gp8.omit.chord-legacy-details"})
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
	wire := extractChordWire(t, data)
	run.Wire("gpifItem.ID", wire.id, "0")
	run.ClaimSerialization(claimSite("chord-name", "export", "M14-CHORD-DEFINITIONS", "C#13/Eb name")).Wire("gpifItem.Name", wire.name, "C#13/Eb")
	run.Wire("gpifItem.Diagram", wire.hasDiagram, true)
	run.Wire("gpifItem.Chord", wire.hasChord, true)
	run.Wire("gpifDiagram.StringCount", wire.stringCount, 7)
	run.Wire("gpifDiagram.FretCount", wire.fretCount, 5)
	run.Wire("gpifDiagram.BaseFret", wire.baseFret, 2)
	run.Wire("gpifDiagram.Frets", wire.frets, []conformanceChordWireFret{{String: 6, Fret: 1}, {String: 5, Fret: 1}, {String: 3, Fret: -2}, {String: 2, Fret: 3}, {String: 1, Fret: 3}, {String: 0, Fret: 5}})
	run.Wire("gpifDiagramFret.String", conformanceChordWireFretStrings(wire.frets), []int{6, 5, 3, 2, 1, 0})
	run.Wire("gpifDiagramFret.Fret", conformanceChordWireFretValues(wire.frets), []int{1, 1, -2, 3, 3, 5})
	run.Wire("gpifDiagram.Fingering", wire.hasFingering, true)
	run.Wire("gpifDiagramFingering.Positions", wire.positions, []conformanceChordWirePosition{{Fret: 1, Finger: "Index", String: 6}, {Fret: 1, Finger: "Index", String: 5}, {Fret: 4294967295, Finger: "None", String: 4}, {Fret: -2, Finger: "None", String: 3}, {Fret: 3, Finger: "Ring", String: 2}, {Fret: 3, Finger: "Ring", String: 1}, {Fret: 5, Finger: "Thumb", String: 0}})
	run.Wire("gpifDiagramPosition.Fret", []int{wire.positions[0].Fret, wire.positions[2].Fret, wire.positions[3].Fret, wire.positions[6].Fret}, []int{1, 4294967295, -2, 5})
	run.Wire("gpifDiagramPosition.Finger", []string{wire.positions[0].Finger, wire.positions[2].Finger, wire.positions[4].Finger, wire.positions[6].Finger}, []string{"Index", "None", "Ring", "Thumb"})
	run.Wire("gpifDiagramPosition.String", []int{wire.positions[0].String, wire.positions[2].String, wire.positions[4].String, wire.positions[6].String}, []int{6, 4, 2, 0})
	run.Wire("gpifDiagram.Properties", wire.propertyNames, []string(nil))
	if wire.chordChildCount != 1 || wire.chordDetailCount != 0 {
		t.Fatalf("wire chord children = %d details = %d, want marker only", wire.chordChildCount, wire.chordDetailCount)
	}
	run.Wire("gpifBeat.Chord", extractGPIFLeafText(t, data)["GPIF/Beats/Beat/Chord"], "01")

	sourceGPIF := strings.Replace(conformanceChordScopedGPIF, `</Diagram></Item>`, `<Property name="ShowName" type="bool" value="false"/></Diagram></Item>`, 1)
	sourceWire := decodeChordWire(t, []byte(sourceGPIF))
	run.Wire("gpifDiagram.Properties", sourceWire.properties, []conformanceChordWireProperty{{Name: "ShowName", Type: "bool", Value: "false"}})
	run.Wire("gpifDiagramProperty.Name", sourceWire.properties[0].Name, "ShowName")
	run.Wire("gpifDiagramProperty.Type", sourceWire.properties[0].Type, "bool")
	run.Wire("gpifDiagramProperty.Value", sourceWire.properties[0].Value, "false")
	detailedItem := `<Item id="track" name="Track"><Diagram stringCount="7" fretCount="5" baseFret="2"><Fret string="6" fret="1"/><Fret string="5" fret="1"/><Fret string="3" fret="-2"/><Fret string="2" fret="3"/><Fret string="1" fret="3"/><Fret string="0" fret="5"/><Fingering><Position finger="Index" fret="1" string="6"/><Position finger="Index" fret="1" string="5"/><Position finger="None" fret="4294967295" string="4"/><Position finger="None" fret="-2" string="3"/><Position finger="Ring" fret="3" string="2"/><Position finger="Ring" fret="3" string="1"/><Position finger="Thumb" fret="5" string="0"/></Fingering></Diagram></Item>`
	for _, ringToken := range []string{"Ring", "Rank"} {
		gpif := strings.Replace(conformanceChordScopedGPIF, `<Item id="track" name="Track"><Diagram stringCount="1" fretCount="4" baseFret="0"><Fret string="0" fret="3"/></Diagram></Item>`, strings.ReplaceAll(detailedItem, `finger="Ring"`, `finger="`+ringToken+`"`), 1)
		parsed, parseErr := ParseWithOptions(conformanceGPIFArchive(t, gpif), ParseOptions{})
		if parseErr != nil {
			t.Fatalf("parse %s fingering: %v", ringToken, parseErr)
		}
		imported := parsed.Song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[1].Effect.Chord
		run.Preserved("Chord.FirstFret", *imported.FirstFret, uint8(3))
		run.Preserved("Chord.Length", imported.Length, uint8(7))
		run.Preserved("Chord.Strings", imported.Strings, chord.Strings)
		run.Preserved("Chord.Fingerings", imported.Fingerings, chord.Fingerings)
		run.ClaimPrimary(claimSite("chord-diagram", "import", "M14-CHORD-DEFINITIONS", "two barre ranges and seven strings")).Preserved("Chord.Barres", imported.Barres, chord.Barres)
		if slices.ContainsFunc(parsed.Diagnostics, func(diagnostic ParseDiagnostic) bool { return diagnostic.Code == "GPIF.Chord.Diagram.Fingering" }) {
			t.Fatalf("%s fingering remained diagnosed: %#v", ringToken, parsed.Diagnostics)
		}
	}
	legacy := parseTestFixture(t, "testdata/gp5/Unknown Chord Extension.gp5")
	legacyChord := legacy.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Chord
	if legacyChord == nil {
		t.Fatal("legacy chord is nil")
	}
	run.Preserved("Chord.Length", legacyChord.Length, uint8(6))
	run.Preserved("Chord.Strings", legacyChord.Strings, []int8{-1, 3, 3, 1, 3, -1})
	run.Preserved("Chord.Fingerings", legacyChord.Fingerings, []Fingering{FingeringOpen, FingeringAnnular, FingeringAnnular, FingeringIndex, FingeringMiddle, FingeringOpen})
	run.Preserved("Chord.Barres", legacyChord.Barres, []Barre{{Fret: 3, Start: 2, End: 3}})
	legacyData, legacyReport, legacyErr := ExportWithReport(legacy, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: reportCodes(PreflightExport(legacy, ExportFormatGP8, ExportOptions{}))}})
	if legacyErr != nil {
		t.Fatalf("legacy chord export: %v, %#v", legacyErr, legacyReport.Entries)
	}
	if hasExportCode(legacyReport, "gp8.omit.chord-barres") || hasExportCode(legacyReport, "gp8.omit.chord-fingerings") {
		t.Fatalf("legacy representable chord report = %#v", legacyReport.Entries)
	}
	legacyRoundTrip, legacyErr := Parse(legacyData)
	if legacyErr != nil {
		t.Fatal(legacyErr)
	}
	legacyGot := legacyRoundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Chord
	run.Field("Chord.Barres", legacyGot.Barres, legacyChord.Barres)
	run.Field("Chord.Fingerings", legacyGot.Fingerings, legacyChord.Fingerings)
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Chord
	if got == nil {
		t.Fatal("round-trip chord is nil")
	}
	run.ClaimPrimary(claimSite("chord-name", "export", "M14-CHORD-DEFINITIONS", "C#13/Eb name")).Field("Chord.Name", got.Name, chord.Name)
	run.Field("Chord.Length", got.Length, chord.Length)
	run.Field("Chord.Strings", got.Strings, chord.Strings)
	run.Field("Chord.FirstFret", *got.FirstFret, *chord.FirstFret)
	run.Field("Chord.Barres", got.Barres, chord.Barres)
	run.Field("Chord.Fingerings", got.Fingerings, chord.Fingerings)
	if len(got.Omissions) != 0 || got.Root != nil || got.Bass != nil || got.Kind != nil || got.Show != nil {
		t.Fatalf("round-trip kept omitted chord details: %#v", got)
	}
}

func TestConformanceChordDefaultsAndLength(t *testing.T) {
	runConformanceChordDefaultsAndLength(newConformanceRun(t))
}

func runConformanceChordDefaultsAndLength(run *conformanceRun) {
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
	allFingerings := Chord{Fingerings: []Fingering{FingeringUnknown, FingeringOpen, FingeringThumb, FingeringIndex, FingeringMiddle, FingeringAnnular, FingeringLittle}}
	run.Field("Chord.Fingerings", allFingerings.Fingerings, []Fingering{-2, -1, 0, 1, 2, 3, 4})
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

func TestChordDiagramRejectsMalformedMappings(t *testing.T) {
	wireString := 0
	tests := []struct {
		name    string
		diagram gpifDiagram
	}{
		{name: "negative base fret", diagram: gpifDiagram{StringCount: 1, BaseFret: -1}},
		{name: "first fret overflow", diagram: gpifDiagram{StringCount: 1, BaseFret: math.MaxUint8}},
		{name: "negative string count", diagram: gpifDiagram{StringCount: -1}},
		{name: "string count overflow", diagram: gpifDiagram{StringCount: math.MaxUint8 + 1}},
		{name: "fret string below range", diagram: gpifDiagram{StringCount: 1, Frets: []gpifDiagramFret{{String: -1}}}},
		{name: "fret string above range", diagram: gpifDiagram{StringCount: 1, Frets: []gpifDiagramFret{{String: 1}}}},
		{name: "duplicate fret string", diagram: gpifDiagram{StringCount: 1, Frets: []gpifDiagramFret{{String: 0}, {String: 0}}}},
		{name: "fret addition overflow", diagram: gpifDiagram{StringCount: 1, BaseFret: 1, Frets: []gpifDiagramFret{{String: 0, Fret: math.MaxInt}}}},
		{name: "absolute fret overflow", diagram: gpifDiagram{StringCount: 1, BaseFret: 127, Frets: []gpifDiagramFret{{String: 0, Fret: 1}}}},
		{name: "missing fingering string", diagram: gpifDiagram{StringCount: 1, Fingering: &gpifDiagramFingering{Positions: []gpifDiagramPosition{{Finger: "None"}}}}},
		{name: "fingering string above range", diagram: gpifDiagram{StringCount: 1, Fingering: &gpifDiagramFingering{Positions: []gpifDiagramPosition{{Finger: "None", String: ptrTo(1)}}}}},
		{name: "duplicate fingering string", diagram: gpifDiagram{StringCount: 1, Frets: []gpifDiagramFret{{String: 0}}, Fingering: &gpifDiagramFingering{Positions: []gpifDiagramPosition{{Finger: "None", String: &wireString}, {Finger: "None", String: &wireString}}}}},
		{name: "unknown finger", diagram: gpifDiagram{StringCount: 1, Frets: []gpifDiagramFret{{String: 0}}, Fingering: &gpifDiagramFingering{Positions: []gpifDiagramPosition{{Finger: "Unknown", String: &wireString}}}}},
		{name: "fingering fret mismatch", diagram: gpifDiagram{StringCount: 1, Frets: []gpifDiagramFret{{String: 0, Fret: 1}}, Fingering: &gpifDiagramFingering{Positions: []gpifDiagramPosition{{Finger: "Index", Fret: 2, String: &wireString}}}}},
		{name: "fingering addition overflow", diagram: gpifDiagram{StringCount: 1, BaseFret: 1, Fingering: &gpifDiagramFingering{Positions: []gpifDiagramPosition{{Finger: "Index", Fret: math.MaxInt, String: &wireString}}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definitions := make(map[string]Chord)
			err := gpifReadChordProperties([]gpifStaffProperty{{Name: "DiagramCollection", Items: &gpifItems{Items: []gpifItem{{ID: "bad", Diagram: &test.diagram}}}}}, definitions)
			if err == nil {
				t.Fatalf("malformed diagram unexpectedly parsed: %#v", definitions)
			}
		})
	}
}

func TestChordDiagramValidationAndNonmutation(t *testing.T) {
	tests := []struct {
		name       string
		chord      Chord
		targetOnly bool
	}{
		{name: "fingering count", chord: Chord{Strings: []int8{1}, Fingerings: []Fingering{}}},
		{name: "unknown fingering value", chord: Chord{Strings: []int8{1}, Fingerings: []Fingering{Fingering(5)}}},
		{name: "None on fretted string", targetOnly: true, chord: Chord{Strings: []int8{1}, Fingerings: []Fingering{FingeringOpen}}},
		{name: "finger on open string", targetOnly: true, chord: Chord{Strings: []int8{0}, Fingerings: []Fingering{FingeringIndex}}},
		{name: "zero barre fret", chord: Chord{Strings: []int8{0, 0}, Barres: []Barre{{Start: 1, End: 2}}}},
		{name: "barre endpoint below range", chord: Chord{Strings: []int8{1, 1}, Barres: []Barre{{Fret: 1, Start: 0, End: 2}}}},
		{name: "barre endpoint above range", chord: Chord{Strings: []int8{1, 1}, Barres: []Barre{{Fret: 1, Start: 1, End: 3}}}},
		{name: "barre endpoint order", chord: Chord{Strings: []int8{1, 1}, Barres: []Barre{{Fret: 1, Start: 2, End: 1}}}},
		{name: "barre fret mismatch", targetOnly: true, chord: Chord{Strings: []int8{1, 2}, Barres: []Barre{{Fret: 1, Start: 1, End: 2}}}},
		{name: "too many synthesized fingers", targetOnly: true, chord: Chord{Strings: []int8{1, 1}, Barres: []Barre{{1, 1, 2}, {1, 1, 2}, {1, 1, 2}, {1, 1, 2}, {1, 1, 2}, {1, 1, 2}}}},
		{name: "shared synthesized endpoint", targetOnly: true, chord: Chord{Strings: []int8{1, 1, 2}, Barres: []Barre{{1, 1, 2}, {2, 2, 3}}}},
		{name: "barre fingering conflict", targetOnly: true, chord: Chord{Strings: []int8{1, 1}, Fingerings: []Fingering{FingeringIndex, FingeringMiddle}, Barres: []Barre{{1, 1, 2}}}},
		{name: "undeclared repeated finger", targetOnly: true, chord: Chord{Strings: []int8{1, 1}, Fingerings: []Fingering{FingeringIndex, FingeringIndex}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			song := semanticValidPitchedGP8Song(t)
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Chord = &test.chord
			before := fmt.Sprintf("%#v", song)
			diagnostics := ValidateSong(song)
			if test.targetOnly {
				if slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == "score.chord.diagram" }) {
					t.Fatalf("target-only conflict entered score diagnostics: %#v", diagnostics)
				}
				data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
				if err != nil || len(data) == 0 || !hasExportCode(report, "gp8.omit.chord-diagram-conflict") {
					t.Fatalf("target conflict export = %d bytes, %#v, %v", len(data), report.Entries, err)
				}
				strictData, strictReport, strictErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
				var lossErr *ExportLossError
				if len(strictData) != 0 || !errors.As(strictErr, &lossErr) || !hasExportCode(strictReport, "gp8.omit.chord-diagram-conflict") {
					t.Fatalf("strict target conflict = %d bytes, %#v, %v", len(strictData), strictReport.Entries, strictErr)
				}
			} else {
				if !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == "score.chord.diagram" }) {
					t.Fatalf("diagnostics = %#v, want score.chord.diagram", diagnostics)
				}
				data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
				if len(data) != 0 || err == nil || !hasExportCode(report, "gp8.reject.score.chord.diagram") {
					t.Fatalf("invalid chord export = %d bytes, %#v, %v", len(data), report.Entries, err)
				}
			}
			if fmt.Sprintf("%#v", song) != before {
				t.Fatal("chord validation or export mutated the song")
			}
		})
	}
}

func TestConformanceChordScopeAndIsolation(t *testing.T) {
	runConformanceChordScopeAndIsolation(newConformanceRun(t))
}

func runConformanceChordScopeAndIsolation(run *conformanceRun) {
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
	result, err := ParseWithOptions(conformanceGPIFArchive(t, conformanceChordScopedGPIF), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	track := &result.Song.Tracks[0]
	upper := track.Staves[0].Measures[0].Voices[0].Beats
	lower := track.Staves[1].Measures[0].Voices[0].Beats
	if len(upper) != 5 || len(lower) != 2 {
		t.Fatalf("scoped beats = %d, %d; want 5, 2", len(upper), len(lower))
	}
	run.ClaimPrimary(claimSite("chord-name", "import", "M14-CHORD-SCOPE-ISOLATION", "C#13/Eb name")).Preserved("BeatEffects.Chord", []string{upper[0].Effect.Chord.Name, upper[1].Effect.Chord.Name, lower[0].Effect.Chord.Name, lower[1].Effect.Chord.Name}, []string{"Upper", "Track", "Lower", "Track"})
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

	reordered := strings.Replace(conformanceChordScopedGPIF,
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
		{name: "missing reference", data: strings.Replace(conformanceChordScopedGPIF, "<Chord>track</Chord>", "<Chord>missing</Chord>", 1), code: "GPIF.Beat.Chord.Reference"},
		{name: "empty id", data: strings.Replace(conformanceChordScopedGPIF, `id="track" name="Track"`, `id="" name="Track"`, 1), code: "GPIF.ChordDefinition.EmptyID"},
		{name: "duplicate id", data: strings.Replace(conformanceChordScopedGPIF, `id="equal-a"`, `id="track"`, 1), code: "GPIF.ChordDefinition.DuplicateID"},
		{name: "show name", data: strings.Replace(conformanceChordScopedGPIF, `</Diagram></Item>`, `<Property name="ShowName" type="bool" value="invalid"/></Diagram></Item>`, 1), code: "GPIF.Chord.Diagram.Property.ShowName"},
		{name: "show diagram", data: strings.Replace(conformanceChordScopedGPIF, `</Diagram></Item>`, `<Property name="ShowDiagram" type="bool" value="invalid"/></Diagram></Item>`, 1), code: "GPIF.Chord.Diagram.Property.ShowDiagram"},
		{name: "show fingering", data: strings.Replace(conformanceChordScopedGPIF, `</Diagram></Item>`, `<Property name="ShowFingering" type="bool" value="invalid"/></Diagram></Item>`, 1), code: "GPIF.Chord.Diagram.Property.ShowFingering"},
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

type conformanceChordWireFret struct {
	String int
	Fret   int
}

func conformanceChordWireFretStrings(frets []conformanceChordWireFret) []int {
	result := make([]int, len(frets))
	for index := range frets {
		result[index] = frets[index].String
	}
	return result
}

func conformanceChordWireFretValues(frets []conformanceChordWireFret) []int {
	result := make([]int, len(frets))
	for index := range frets {
		result[index] = frets[index].Fret
	}
	return result
}

type conformanceChordWirePosition struct {
	Fret   int
	Finger string
	String int
}

type conformanceChordWireProperty struct {
	Name  string
	Type  string
	Value string
}

type conformanceAlphaTabChordDiagramFact struct {
	Track      int    `json:"track"`
	Staff      int    `json:"staff"`
	Bar        int    `json:"bar"`
	Voice      int    `json:"voice"`
	Beat       int    `json:"beat"`
	Name       string `json:"name"`
	FirstFret  int    `json:"firstFret"`
	Strings    []int  `json:"strings"`
	BarreFrets []int  `json:"barreFrets"`
}

func TestAlphaTabPreservesChordDiagrams(t *testing.T) {
	requireAlphaTabConformance(t)
	song := semanticValidPitchedGP8Song(t)
	firstFret := uint8(3)
	song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Chord = &Chord{
		Name: "Two barres", FirstFret: &firstFret, Length: 7,
		Strings: []int8{3, 3, -1, 0, 5, 5, 7},
		Barres:  []Barre{{Fret: 3, Start: 1, End: 2}, {Fret: 5, Start: 5, End: 6}},
	}
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.normalize.track-view"}}})
	if err != nil || !reflect.DeepEqual(reportCodes(report), []string{"gp8.normalize.track-view"}) {
		t.Fatalf("chord diagram export = %v, %#v", err, report.Entries)
	}
	var facts []conformanceAlphaTabChordDiagramFact
	readAlphaTabOracleFacts(t, "--chord-diagrams", writeConformanceFixture(t, data), &facts)
	facts = slices.DeleteFunc(facts, func(fact conformanceAlphaTabChordDiagramFact) bool { return fact.Name != "Two barres" })
	want := []conformanceAlphaTabChordDiagramFact{{Track: 0, Staff: 0, Bar: 0, Voice: 0, Beat: 1, Name: "Two barres", FirstFret: 3, Strings: []int{3, 3, -1, 0, 5, 5, 7}, BarreFrets: []int{3, 5}}}
	if !reflect.DeepEqual(facts, want) {
		t.Fatalf("AlphaTab chord diagram facts = %#v, want %#v", facts, want)
	}
	sites := []conformanceClaimSite{
		claimSite("chord-diagram", "import", "M14-CHORD-DEFINITIONS", "two barre ranges and seven strings"),
		claimSite("chord-diagram", "model", "M14-CHORD-DEFINITIONS", "two barre ranges and seven strings"),
	}
	conformanceIndependentClaim(t, "field:Chord.Barres", sites...)
}

type conformanceChordWire struct {
	name             string
	id               string
	hasDiagram       bool
	hasChord         bool
	stringCount      int
	fretCount        int
	baseFret         int
	frets            []conformanceChordWireFret
	hasFingering     bool
	propertyNames    []string
	positions        []conformanceChordWirePosition
	properties       []conformanceChordWireProperty
	chordChildCount  int
	chordDetailCount int
}

func extractChordWire(t *testing.T, data []byte) conformanceChordWire {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	gpif := readZipMember(t, archive, "Content/score.gpif")
	return decodeChordWire(t, gpif)
}

func decodeChordWire(t *testing.T, gpif []byte) conformanceChordWire {
	t.Helper()
	decoder := xml.NewDecoder(bytes.NewReader(gpif))
	var result conformanceChordWire
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
				result.id = conformanceChordXMLAttribute(value, "id")
				result.name = conformanceChordXMLAttribute(value, "name")
			}
			if !inFirstItem {
				continue
			}
			switch value.Name.Local {
			case "Diagram":
				result.hasDiagram = true
				result.stringCount = conformanceChordXMLIntAttribute(t, value, "stringCount")
				result.fretCount = conformanceChordXMLIntAttribute(t, value, "fretCount")
				result.baseFret = conformanceChordXMLIntAttribute(t, value, "baseFret")
			case "Fret":
				result.frets = append(result.frets, conformanceChordWireFret{String: conformanceChordXMLIntAttribute(t, value, "string"), Fret: conformanceChordXMLIntAttribute(t, value, "fret")})
			case "Fingering":
				result.hasFingering = true
			case "Position":
				result.positions = append(result.positions, conformanceChordWirePosition{Fret: conformanceChordXMLIntAttribute(t, value, "fret"), Finger: conformanceChordXMLAttribute(value, "finger"), String: conformanceChordXMLIntAttribute(t, value, "string")})
			case "Property":
				property := conformanceChordWireProperty{Name: conformanceChordXMLAttribute(value, "name"), Type: conformanceChordXMLAttribute(value, "type"), Value: conformanceChordXMLAttribute(value, "value")}
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
	return conformanceChordWire{}
}

func conformanceChordXMLAttribute(element xml.StartElement, name string) string {
	for _, attribute := range element.Attr {
		if attribute.Name.Local == name {
			return attribute.Value
		}
	}
	return ""
}

func conformanceChordXMLIntAttribute(t *testing.T, element xml.StartElement, name string) int {
	t.Helper()
	var value int
	if _, err := fmt.Sscan(conformanceChordXMLAttribute(element, name), &value); err != nil {
		t.Fatalf("%s %s: %v", element.Name.Local, name, err)
	}
	return value
}

const conformanceChordScopedGPIF = `<?xml version="1.0" encoding="utf-8"?>
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

func TestGP8ChordConsumerLimits(t *testing.T) { runChordConsumerLimits(newConformanceRun(t), false) }
func TestAlphaTabChordConsumerLimits(t *testing.T) {
	requireAlphaTabConformance(t)
	runChordConsumerLimits(newConformanceRun(t), true)
}

func runChordConsumerLimits(run *conformanceRun, oracle bool) {
	t := run.t
	for _, count := range []int{0, 3, 5} {
		authored := count == 0
		t.Run(fmt.Sprintf("authored=%t barres=%d", authored, count), func(t *testing.T) {
			song := consumerLimitSong(t)
			first := uint8(3)
			chord := &Chord{Name: "Annular", Length: 7, FirstFret: &first, Strings: []int8{3, 3, -1, 0, 5, 5, 7}, Barres: []Barre{{Fret: 3, Start: 1, End: 2}, {Fret: 5, Start: 5, End: 6}}, Fingerings: []Fingering{FingeringIndex, FingeringIndex, FingeringOpen, FingeringOpen, FingeringAnnular, FingeringAnnular, FingeringThumb}}
			want := []int{3}
			if !authored {
				chord.Strings = []int8{3, 3, 5, 5, 7, 7, 0}
				chord.Fingerings = nil
				chord.Barres = []Barre{{Fret: 3, Start: 1, End: 2}, {Fret: 5, Start: 3, End: 4}, {Fret: 7, Start: 5, End: 6}}
				want = []int{3, 5}
				if count == 5 {
					chord.Length = 10
					chord.Strings = []int8{3, 3, 5, 5, 7, 7, 9, 9, 11, 11}
					chord.Barres = append(chord.Barres, Barre{Fret: 9, Start: 7, End: 8}, Barre{Fret: 11, Start: 9, End: 10})
					want = []int{3, 5, 9, 11}
				}
			}
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Chord = chord
			data, report := assertConsumerLossPolicy(t, song, []string{"gp8.omit.chord-annular-consumer-barre"})
			run.Report("M14-CHORD-CONSUMER", reportCodes(report), []string{"gp8.omit.chord-annular-consumer-barre"})
			expectedFret := 5
			if !authored {
				expectedFret = 7
			}
			if report.Entries[0].Location != (ScoreLocation{}) || report.Entries[0].Reason != fmt.Sprintf("pinned AlphaTab ignores the Ring finger token and loses the annular barre at fret %d; GPIF retains the authored finger meaning", expectedFret) {
				t.Fatalf("imprecise chord report: %#v", report)
			}
			wire := extractChordWire(t, data)
			ring := 0
			for _, position := range wire.positions {
				if position.Finger == "Ring" {
					ring++
				}
			}
			if ring != 2 {
				t.Fatalf("Ring wire positions = %d", ring)
			}
			roundTrip, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Chord
			run.Preserved("Chord.Barres", got.Barres, chord.Barres)
			if authored {
				run.Preserved("Chord.Fingerings", got.Fingerings, chord.Fingerings)
			}
			if oracle {
				var facts []conformanceAlphaTabChordDiagramFact
				readAlphaTabOracleFacts(t, "--chord-diagrams", writeConformanceFixture(t, data), &facts)
				facts = slices.DeleteFunc(facts, func(f conformanceAlphaTabChordDiagramFact) bool { return f.Name != "Annular" })
				if len(facts) != 1 || !slices.Equal(facts[0].BarreFrets, want) || facts[0].FirstFret != 3 || facts[0].Track != 0 || facts[0].Staff != 0 || facts[0].Bar != 0 || facts[0].Voice != 0 || facts[0].Beat != 0 {
					t.Fatalf("consumer chord = %#v", facts)
				}
			}
		})
	}
	for _, sameFret := range []bool{false, true} {
		t.Run(fmt.Sprintf("annular positions share fret=%t", sameFret), func(t *testing.T) {
			song := consumerLimitSong(t)
			chord := &Chord{Name: "Annular positions", Length: 3, Strings: []int8{3, 5, 7}, Fingerings: []Fingering{FingeringAnnular, FingeringAnnular, FingeringAnnular}}
			codes := []string{}
			if sameFret {
				chord.Strings = []int8{3, 3, 3}
				chord.Barres = []Barre{{Fret: 3, Start: 1, End: 3}}
				codes = []string{"gp8.omit.chord-annular-consumer-barre"}
			}
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Chord = chord
			data, report := assertConsumerLossPolicy(t, song, codes)
			run.Report("M14-CHORD-CONSUMER", reportCodes(report), codes)
			if oracle {
				var facts []conformanceAlphaTabChordDiagramFact
				readAlphaTabOracleFacts(t, "--chord-diagrams", writeConformanceFixture(t, data), &facts)
				if len(facts) == 0 || facts[0].Name != chord.Name || len(facts[0].BarreFrets) != 0 {
					t.Fatalf("annular positions: %#v", facts)
				}
			}
		})
	}

}
