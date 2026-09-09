// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"slices"
	"strings"
	"testing"
)

func TestSemanticMatrixM16PercussionIdentity(t *testing.T) {
	runSemanticMatrixM16PercussionIdentity(newSemanticMatrixRun(t))
}

func runSemanticMatrixM16PercussionIdentity(run *semanticMatrixRun) {
	t := run.t
	song := m16PercussionSong(t)
	track := &song.Tracks[0]
	wantDefinitions := slices.Clone(track.PercussionArticulations)
	run.Field("Track.PercussionTrack", track.PercussionTrack, true)
	run.Field("Staff.PercussionTrack", track.Staves[0].PercussionTrack, true)
	run.Field("Staff.StandardNotationLineCount", track.Staves[0].StandardNotationLineCount, 7)
	run.Field("Track.PercussionArticulations", track.PercussionArticulations, wantDefinitions)
	for index, articulation := range track.PercussionArticulations {
		want := wantDefinitions[index]
		run.Field("PercussionArticulation.ElementName", articulation.ElementName, want.ElementName)
		run.Field("PercussionArticulation.ElementType", articulation.ElementType, want.ElementType)
		run.Field("PercussionArticulation.ElementSoundbankName", articulation.ElementSoundbankName, want.ElementSoundbankName)
		run.Field("PercussionArticulation.Name", articulation.Name, want.Name)
		run.Field("PercussionArticulation.StaffLine", articulation.StaffLine, want.StaffLine)
		run.Field("PercussionArticulation.NoteheadDefault", articulation.NoteheadDefault, want.NoteheadDefault)
		run.Field("PercussionArticulation.NoteheadHalf", articulation.NoteheadHalf, want.NoteheadHalf)
		run.Field("PercussionArticulation.NoteheadWhole", articulation.NoteheadWhole, want.NoteheadWhole)
		run.Field("PercussionArticulation.TechniquePlacement", articulation.TechniquePlacement, want.TechniquePlacement)
		run.Field("PercussionArticulation.TechniqueSymbol", articulation.TechniqueSymbol, want.TechniqueSymbol)
		run.Field("PercussionArticulation.InputMIDINumbers", articulation.InputMIDINumbers, want.InputMIDINumbers)
		run.Field("PercussionArticulation.OutputRSESound", articulation.OutputRSESound, want.OutputRSESound)
		run.Field("PercussionArticulation.OutputMIDINumber", articulation.OutputMIDINumber, want.OutputMIDINumber)
	}
	mainNotes := track.Measures[0].Voices[0].Beats[0].Notes
	run.Field("Note.HasPercussionArticulation", []bool{mainNotes[0].HasPercussionArticulation, mainNotes[1].HasPercussionArticulation, mainNotes[2].HasPercussionArticulation}, []bool{true, true, false})
	run.Field("Note.PercussionArticulation", []int{mainNotes[0].PercussionArticulation, mainNotes[1].PercussionArticulation}, []int{0, 2})
	run.Field("Note.Value", []int16{mainNotes[0].Value, mainNotes[1].Value, mainNotes[2].Value}, []int16{91, 46, 40})
	run.Field("GraceEffect.HasPercussionArticulation", []bool{mainNotes[0].Effect.Graces[0].HasPercussionArticulation, mainNotes[1].Effect.Graces[0].HasPercussionArticulation}, []bool{true, false})
	run.Field("GraceEffect.PercussionArticulation", mainNotes[0].Effect.Graces[0].PercussionArticulation, 0)

	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	wire := extractM16Wire(t, data)
	if len(wire.tracks) != 1 {
		t.Fatalf("wire tracks = %d, want 1", len(wire.tracks))
	}
	wireTrack := wire.tracks[0]
	run.Wire("gpifTrack.InstrumentSet", wireTrack.InstrumentSet != nil, true)
	run.Wire("gpifInstrumentSet.Name", wireTrack.InstrumentSet.Name, "Drums")
	run.Wire("gpifInstrumentSet.Type", wireTrack.InstrumentSet.Type, "drumKit")
	run.Wire("gpifInstrumentSet.LineCount", wireTrack.InstrumentSet.LineCount, 7)
	run.Wire("gpifInstrumentSet.Elements", len(wireTrack.InstrumentSet.Elements), 4)
	run.Wire("gpifElements.Elements", len(wireTrack.InstrumentSet.Elements), 4)
	flat := flattenM16WireArticulations(wireTrack.InstrumentSet.Elements)
	if len(flat) != 4 {
		t.Fatalf("wire articulations = %#v, want three custom and one builtin fallback", flat)
	}
	run.Wire("gpifElement.Name", m16ElementStrings(wireTrack.InstrumentSet.Elements, func(element m16WireElement) string { return element.Name }), []string{"Metal", "Skin", "Metal", "Charley"})
	run.Wire("gpifElement.Type", m16ElementStrings(wireTrack.InstrumentSet.Elements, func(element m16WireElement) string { return element.Type }), []string{"cymbal", "drum", "cymbal", "hiHat"})
	run.Wire("gpifElement.SoundbankName", m16ElementStrings(wireTrack.InstrumentSet.Elements, func(element m16WireElement) string { return element.SoundbankName }), []string{"SB-A", "SB-B", "SB-A", "Master-Hihat"})
	run.Wire("gpifElement.Articulations", len(flat), 4)
	run.Wire("gpifArticulations.Articulations", len(flat), 4)
	firstThree := flat[:3]
	run.Wire("gpifArticulation.Name", m16ArticulationStrings(firstThree, func(articulation m16WireArticulation) string { return articulation.Name }), []string{"Edge", "Center", "Bell"})
	run.Wire("gpifArticulation.StaffLine", m16ArticulationInts(firstThree, func(articulation m16WireArticulation) int { return articulation.StaffLine }), []int{-3, 3, -1})
	run.Wire("gpifArticulation.Noteheads", m16ArticulationStrings(firstThree, func(articulation m16WireArticulation) string { return articulation.Noteheads }), []string{"noteheadHeavyX noteheadHalf noteheadWhole", "noteheadBlack noteheadDiamondWhite noteheadCircleX", "noteheadXBlack noteheadXHalf noteheadXWhole"})
	run.Wire("gpifArticulation.TechniquePlacement", m16ArticulationStrings(firstThree, func(articulation m16WireArticulation) string { return articulation.TechniquePlacement }), []string{"above", "inside", "below"})
	run.Wire("gpifArticulation.TechniqueSymbol", m16ArticulationStrings(firstThree, func(articulation m16WireArticulation) string { return articulation.TechniqueSymbol }), []string{"pictEdgeOfCymbal", "articStaccatoAbove", "stringsDownBow"})
	run.Wire("gpifArticulation.InputMIDINumbers", m16ArticulationStrings(firstThree, func(articulation m16WireArticulation) string { return articulation.InputMIDINumbers }), []string{"91 92", "38 40", "93"})
	run.Wire("gpifArticulation.OutputRSESound", m16ArticulationStrings(firstThree, func(articulation m16WireArticulation) string { return articulation.OutputRSESound }), []string{"edge.hit", "center.hit", "bell.hit"})
	run.Wire("gpifArticulation.OutputMIDINumber", m16ArticulationInts(firstThree, func(articulation m16WireArticulation) int { return articulation.OutputMIDINumber }), []int{56, 38, 56})
	identities := m16WireNoteIdentities(wire.notes)
	run.Wire("gpifNote.InstrumentArticulation", identities, []int{0, 3, 0, 2, 1})
	run.Wire("gpifNote.Properties", len(wire.notes), 5)
	run.Wire("gpifProperty.Number", m16WireNoteMIDIs(wire.notes), []int{91, 46, 91, 46, 40})

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	gotTrack := roundTrip.Tracks[0]
	if len(gotTrack.PercussionArticulations) < 4 {
		t.Fatalf("round-trip definitions = %#v, want three custom and one builtin fallback", gotTrack.PercussionArticulations)
	}
	if !slices.EqualFunc(gotTrack.PercussionArticulations[:3], wantDefinitions, func(a, b PercussionArticulation) bool { return gpifSamePercussionArticulation(a, b) }) {
		t.Fatalf("round-trip custom definitions = %#v, want %#v", gotTrack.PercussionArticulations[:3], wantDefinitions)
	}
	gotNotes := gotTrack.Measures[0].Voices[0].Beats[0].Notes
	if gotNotes[0].PercussionArticulation != 0 || gotNotes[1].PercussionArticulation != 2 || gotNotes[2].PercussionArticulation != 1 {
		t.Fatalf("round-trip main identities = %d/%d/%d, want 0/2/1", gotNotes[0].PercussionArticulation, gotNotes[1].PercussionArticulation, gotNotes[2].PercussionArticulation)
	}
	if gotNotes[0].Effect.Graces[0].PercussionArticulation != 0 || gotNotes[1].Effect.Graces[0].PercussionArticulation != 3 {
		t.Fatalf("round-trip grace identities = %d/%d, want 0/3", gotNotes[0].Effect.Graces[0].PercussionArticulation, gotNotes[1].Effect.Graces[0].PercussionArticulation)
	}
	if gotTrack.PercussionArticulations[0].OutputMIDINumber != 56 || gotTrack.PercussionArticulations[2].OutputMIDINumber != 56 || gotTrack.PercussionArticulations[0].Name == gotTrack.PercussionArticulations[2].Name {
		t.Fatal("duplicate sounding MIDI collapsed distinct articulation identities")
	}
}

func TestSemanticMatrixM16SourceAndValidation(t *testing.T) {
	runSemanticMatrixM16SourceAndValidation(newSemanticMatrixRun(t))
}

func runSemanticMatrixM16SourceAndValidation(run *semanticMatrixRun) {
	t := run.t
	detection := []struct {
		name  string
		track gpifTrack
		want  bool
	}{
		{name: "drums type", track: gpifTrack{InstrumentSet: &gpifInstrumentSet{Type: "drums"}}, want: true},
		{name: "percussion type", track: gpifTrack{InstrumentSet: &gpifInstrumentSet{Type: "percussion"}}, want: true},
		{name: "drum kit type", track: gpifTrack{InstrumentSet: &gpifInstrumentSet{Type: "drumKit"}}, want: true},
		{name: "GP6 instrument", track: gpifTrack{Instrument: &gpifInstrument{Ref: "drmkt"}}, want: true},
		{name: "general MIDI channel", track: gpifTrack{GeneralMidi: &gpifGeneralMidi{PrimaryChannel: 9}}, want: true},
		{name: "connection channel", track: gpifTrack{MidiConnection: gpifMidiConnection{PrimaryChannel: 9}}, want: true},
		{name: "pitched", track: gpifTrack{GeneralMidi: &gpifGeneralMidi{PrimaryChannel: 8}}, want: false},
	}
	for _, test := range detection {
		t.Run(test.name, func(t *testing.T) {
			got := test.track.isPercussionTrack()
			if got != test.want {
				t.Errorf("isPercussionTrack() = %t, want %t", got, test.want)
			}
			if test.track.InstrumentSet != nil {
				run.Dispatch("isPercussionTrack:t.InstrumentSet.Type", got, true)
			}
		})
	}

	instrumentSet := &gpifInstrumentSet{Elements: gpifElements{Elements: []gpifElement{{
		Name: "Ride", Type: "cymbal", SoundbankName: "Bank",
		Articulations: gpifArticulations{Articulations: []gpifArticulation{{
			Name: "Bell", StaffLine: -2, Noteheads: "noteheadXBlack noteheadXHalf noteheadXWhole",
			TechniquePlacement: "outside", TechniqueSymbol: "pictEdgeOfCymbal",
			InputMIDINumbers: "53 93", OutputRSESound: "bell.hit", OutputMIDINumber: 53,
		}}},
	}}}}
	patch := &gpifInstrumentSet{Elements: gpifElements{Elements: []gpifElement{{Name: "Ride", Articulations: gpifArticulations{Articulations: []gpifArticulation{{Name: "Bell", StaffLine: 6}}}}}}}
	definitions := gpifReadPercussionArticulations(instrumentSet, patch)
	if len(definitions) != 1 {
		t.Fatalf("patched definitions = %#v", definitions)
	}
	wantPatched := PercussionArticulation{
		ElementName: "Ride", ElementType: "cymbal", ElementSoundbankName: "Bank",
		Name: "Bell", StaffLine: 6, NoteheadDefault: "noteheadXBlack", NoteheadHalf: "noteheadXHalf", NoteheadWhole: "noteheadXWhole",
		TechniquePlacement: "outside", TechniqueSymbol: "pictEdgeOfCymbal", InputMIDINumbers: []int{53, 93}, OutputRSESound: "bell.hit", OutputMIDINumber: 53,
	}
	if !gpifSamePercussionArticulation(definitions[0], wantPatched) {
		t.Fatalf("patched definition = %#v, want only the staff-line override in %#v", definitions[0], wantPatched)
	}

	negative := m16PercussionSong(t)
	negativeData, err := Export(negative, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	negativeData = rewriteConformanceGPIF(t, negativeData, func(gpif string) string {
		return strings.Replace(gpif, "<InstrumentArticulation>2</InstrumentArticulation>", "<InstrumentArticulation>-1</InstrumentArticulation>", 1)
	})
	result, err := ParseWithOptions(negativeData, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(result.Diagnostics, func(diagnostic ParseDiagnostic) bool {
		return diagnostic.Code == "GPIF.Note.InstrumentArticulation.Invalid"
	}) {
		t.Fatalf("negative identity diagnostics = %#v", result.Diagnostics)
	}
	if result.Song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[1].HasPercussionArticulation {
		t.Fatal("negative source identity unexpectedly survived in the public note")
	}

	highData := rewriteConformanceGPIF(t, negativeData, func(gpif string) string {
		return strings.Replace(gpif, "<InstrumentArticulation>-1</InstrumentArticulation>", "<InstrumentArticulation>128</InstrumentArticulation>", 1)
	})
	highResult, err := ParseWithOptions(highData, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	highNote := highResult.Song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[1]
	if !highNote.HasPercussionArticulation || highNote.PercussionArticulation != 128 {
		t.Fatalf("high source identity = %t/%d, want retained 128", highNote.HasPercussionArticulation, highNote.PercussionArticulation)
	}
	if !slices.ContainsFunc(ValidateSong(highResult.Song), func(diagnostic ScoreDiagnostic) bool {
		return diagnostic.Code == "score.note.percussion-reference"
	}) {
		t.Fatal("retained high source identity has no shared-model diagnostic")
	}

	for _, test := range []struct {
		name string
		code string
		set  func(*Song)
	}{
		{name: "negative note identity", code: "score.note.percussion-reference", set: func(song *Song) {
			note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
			note.HasPercussionArticulation = true
			note.PercussionArticulation = -1
		}},
		{name: "high note identity", code: "score.note.percussion-reference", set: func(song *Song) {
			note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
			note.HasPercussionArticulation = true
			note.PercussionArticulation = len(song.Tracks[0].PercussionArticulations)
		}},
		{name: "negative grace identity", code: "score.grace.percussion-reference", set: func(song *Song) {
			grace := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces[0]
			grace.HasPercussionArticulation = true
			grace.PercussionArticulation = -1
		}},
		{name: "high grace identity", code: "score.grace.percussion-reference", set: func(song *Song) {
			grace := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces[0]
			grace.HasPercussionArticulation = true
			grace.PercussionArticulation = len(song.Tracks[0].PercussionArticulations)
		}},
		{name: "negative output MIDI", code: "score.percussion-articulation.output-midi", set: func(song *Song) { song.Tracks[0].PercussionArticulations[0].OutputMIDINumber = -1 }},
		{name: "high output MIDI", code: "score.percussion-articulation.output-midi", set: func(song *Song) { song.Tracks[0].PercussionArticulations[0].OutputMIDINumber = 128 }},
		{name: "negative input MIDI", code: "score.percussion-articulation.input-midi", set: func(song *Song) { song.Tracks[0].PercussionArticulations[0].InputMIDINumbers[0] = -1 }},
		{name: "high input MIDI", code: "score.percussion-articulation.input-midi", set: func(song *Song) { song.Tracks[0].PercussionArticulations[0].InputMIDINumbers[0] = 128 }},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := m16PercussionSong(t)
			test.set(song)
			diagnostics := ValidateSong(song)
			if !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == test.code }) {
				t.Fatalf("diagnostics = %#v, want %s", diagnostics, test.code)
			}
		})
	}

	track := Track{PercussionArticulations: make([]PercussionArticulation, 34)}
	track.PercussionArticulations[0] = PercussionArticulation{ElementName: "Acoustic Kick Drum", ElementType: "kickDrum", Name: "Custom", InputMIDINumbers: []int{35}, OutputMIDINumber: 42}
	fallbacks := gpifPercussionFallbacks{tableLength: len(track.PercussionArticulations)}
	note := Note{PercussionArticulation: 35, HasPercussionArticulation: true}
	gpifNormalizePercussionArticulation(&track, &note, &fallbacks)
	run.Dispatch("gpifNormalizePercussionArticulation:element.Type", note.PercussionArticulation, 34)
	if track.PercussionArticulations[34].OutputMIDINumber != 35 {
		t.Fatal("builtin fallback collided with the custom definition")
	}

	unsupported := semanticValidGP8Song(t)
	unsupported.Tracks[0].PercussionArticulations = nil
	unsupported.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Value = 40
	unsupported.Tracks[0].Staves[0].Measures = unsupported.Tracks[0].Measures
	report := PreflightExport(unsupported, ExportFormatGP8, ExportOptions{})
	run.Dispatch("validateGP8Staff:element.Type", slices.ContainsFunc(report.Entries, func(entry ExportReportEntry) bool { return entry.Disposition == ExportDispositionRejected }), true)
}

func TestSemanticMatrixM16NoteheadOptions(t *testing.T) {
	runSemanticMatrixM16NoteheadOptions(newSemanticMatrixRun(t))
}

func runSemanticMatrixM16NoteheadOptions(run *semanticMatrixRun) {
	for _, test := range []struct {
		name string
		kind GP8PercussionNotehead
		want string
	}{
		{name: "native", kind: GP8PercussionNoteheadDefault, want: "noteheadCircleX noteheadCircleX noteheadCircleX"},
		{name: "filled", kind: GP8PercussionNoteheadFilled, want: "noteheadBlack noteheadHalf noteheadWhole"},
		{name: "x", kind: GP8PercussionNoteheadX, want: "noteheadXBlack noteheadXBlack noteheadXBlack"},
		{name: "circle x", kind: GP8PercussionNoteheadCircleX, want: "noteheadCircleX noteheadCircleX noteheadCircleX"},
		{name: "heavy x", kind: GP8PercussionNoteheadHeavyX, want: "noteheadHeavyX noteheadHeavyX noteheadHeavyX"},
	} {
		run.t.Run(test.name, func(t *testing.T) {
			song := m16BuiltinPercussionSong(t, 46)
			options := ExportOptions{GP8: GP8ExportOptions{PercussionNoteheads: map[int16]GP8PercussionNotehead{46: test.kind}}}
			data, err := ExportWithOptions(song, ExportFormatGP8, options)
			if err != nil {
				t.Fatal(err)
			}
			wire := extractM16Wire(t, data)
			flat := flattenM16WireArticulations(wire.tracks[0].InstrumentSet.Elements)
			if len(flat) != 1 {
				t.Fatalf("articulations = %#v, want one", flat)
			}
			run.Wire("gpifArticulation.Noteheads", flat[0].Noteheads, test.want)
		})
	}

	invalid := m16BuiltinPercussionSong(run.t, 46)
	for _, options := range []GP8ExportOptions{
		{PercussionNoteheads: map[int16]GP8PercussionNotehead{-1: GP8PercussionNoteheadX}},
		{PercussionNoteheads: map[int16]GP8PercussionNotehead{128: GP8PercussionNoteheadX}},
		{PercussionNoteheads: map[int16]GP8PercussionNotehead{46: GP8PercussionNotehead(99)}},
	} {
		if _, err := ExportWithOptions(invalid, ExportFormatGP8, ExportOptions{GP8: options}); err == nil {
			run.t.Fatalf("invalid percussion notehead option %#v was accepted", options)
		}
	}
}

func m16PercussionSong(t *testing.T) *Song {
	t.Helper()
	song := semanticValidGP8Song(t)
	song.MeasureHeaders = slices.Clone(song.MeasureHeaders[:1])
	track := &song.Tracks[0]
	track.Measures = slices.Clone(track.Measures[:1])
	track.PercussionTrack = true
	track.Staves[0].PercussionTrack = true
	track.Staves[0].StandardNotationLineCount = 7
	track.PercussionArticulations = []PercussionArticulation{
		{ElementName: "Metal", ElementType: "cymbal", ElementSoundbankName: "SB-A", Name: "Edge", StaffLine: -3, NoteheadDefault: "noteheadHeavyX", NoteheadHalf: "noteheadHalf", NoteheadWhole: "noteheadWhole", TechniquePlacement: "above", TechniqueSymbol: "pictEdgeOfCymbal", InputMIDINumbers: []int{91, 92}, OutputRSESound: "edge.hit", OutputMIDINumber: 56},
		{ElementName: "Skin", ElementType: "drum", ElementSoundbankName: "SB-B", Name: "Center", StaffLine: 3, NoteheadDefault: "noteheadBlack", NoteheadHalf: "noteheadDiamondWhite", NoteheadWhole: "noteheadCircleX", TechniquePlacement: "inside", TechniqueSymbol: "articStaccatoAbove", InputMIDINumbers: []int{38, 40}, OutputRSESound: "center.hit", OutputMIDINumber: 38},
		{ElementName: "Metal", ElementType: "cymbal", ElementSoundbankName: "SB-A", Name: "Bell", StaffLine: -1, NoteheadDefault: "noteheadXBlack", NoteheadHalf: "noteheadXHalf", NoteheadWhole: "noteheadXWhole", TechniquePlacement: "below", TechniqueSymbol: "stringsDownBow", InputMIDINumbers: []int{93}, OutputRSESound: "bell.hit", OutputMIDINumber: 56},
	}
	exact91, _ := NewFret(91)
	exact46, _ := NewFret(46)
	track.Measures[0].Voices = []Voice{{Beats: []Beat{{
		Duration: defaultDuration(), Status: BeatStatusNormal, Dynamics: Forte,
		Notes: []Note{
			{Value: 91, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1, HasPercussionArticulation: true, PercussionArticulation: 0, Effect: NoteEffect{Graces: []GraceEffect{{Fret: 91, ExactFret: &exact91, Duration: DurationThirtySecond, Velocity: Forte, HasPercussionArticulation: true, PercussionArticulation: 0}}}},
			{Value: 46, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1, HasPercussionArticulation: true, PercussionArticulation: 2, Effect: NoteEffect{Graces: []GraceEffect{{Fret: 46, ExactFret: &exact46, Duration: DurationThirtySecond, Velocity: Forte}}}},
			{Value: 40, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1},
		},
	}}}}
	track.Staves[0].Measures = track.Measures
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	return song
}

func m16BuiltinPercussionSong(t *testing.T, midi int16) *Song {
	t.Helper()
	song := semanticValidGP8Song(t)
	song.MeasureHeaders = slices.Clone(song.MeasureHeaders[:1])
	track := &song.Tracks[0]
	track.Measures = slices.Clone(track.Measures[:1])
	track.PercussionArticulations = nil
	track.Measures[0].Voices = []Voice{{Beats: []Beat{{Duration: defaultDuration(), Status: BeatStatusNormal, Dynamics: Forte, Notes: []Note{{Value: midi, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1}}}}}}
	track.Staves[0].Measures = track.Measures
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	return song
}

type m16WireDocument struct {
	Tracks struct {
		Tracks []m16WireTrack `xml:"Track"`
	} `xml:"Tracks"`
	Notes struct {
		Notes []m16WireNote `xml:"Note"`
	} `xml:"Notes"`
	tracks []m16WireTrack
	notes  []m16WireNote
}

type m16WireTrack struct {
	InstrumentSet *m16WireInstrumentSet `xml:"InstrumentSet"`
}

type m16WireInstrumentSet struct {
	Name      string           `xml:"Name"`
	Type      string           `xml:"Type"`
	LineCount int              `xml:"LineCount"`
	Elements  []m16WireElement `xml:"Elements>Element"`
}

type m16WireElement struct {
	Name          string                `xml:"Name"`
	Type          string                `xml:"Type"`
	SoundbankName string                `xml:"SoundbankName"`
	Articulations []m16WireArticulation `xml:"Articulations>Articulation"`
}

type m16WireArticulation struct {
	Name               string `xml:"Name"`
	StaffLine          int    `xml:"StaffLine"`
	Noteheads          string `xml:"Noteheads"`
	TechniquePlacement string `xml:"TechniquePlacement"`
	TechniqueSymbol    string `xml:"TechniqueSymbol"`
	InputMIDINumbers   string `xml:"InputMidiNumbers"`
	OutputRSESound     string `xml:"OutputRSESound"`
	OutputMIDINumber   int    `xml:"OutputMidiNumber"`
}

type m16WireNote struct {
	InstrumentArticulation *int `xml:"InstrumentArticulation"`
	Properties             struct {
		Properties []struct {
			Name   string `xml:"name,attr"`
			Number *int   `xml:"Number"`
		} `xml:"Property"`
	} `xml:"Properties"`
}

func extractM16Wire(t *testing.T, data []byte) m16WireDocument {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var contents []byte
	for _, file := range archive.File {
		if file.Name != "Content/score.gpif" {
			continue
		}
		reader, openErr := file.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		contents, err = io.ReadAll(reader)
		closeErr := reader.Close()
		if err == nil {
			err = closeErr
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if contents == nil {
		t.Fatal("GP8 archive has no score.gpif")
	}
	var wire m16WireDocument
	if err := xml.Unmarshal(contents, &wire); err != nil {
		t.Fatal(err)
	}
	wire.tracks = wire.Tracks.Tracks
	wire.notes = wire.Notes.Notes
	return wire
}

func flattenM16WireArticulations(elements []m16WireElement) []m16WireArticulation {
	var result []m16WireArticulation
	for _, element := range elements {
		result = append(result, element.Articulations...)
	}
	return result
}

func m16ElementStrings(elements []m16WireElement, get func(m16WireElement) string) []string {
	result := make([]string, len(elements))
	for index, element := range elements {
		result[index] = get(element)
	}
	return result
}

func m16ArticulationStrings(articulations []m16WireArticulation, get func(m16WireArticulation) string) []string {
	result := make([]string, len(articulations))
	for index, articulation := range articulations {
		result[index] = get(articulation)
	}
	return result
}

func m16ArticulationInts(articulations []m16WireArticulation, get func(m16WireArticulation) int) []int {
	result := make([]int, len(articulations))
	for index, articulation := range articulations {
		result[index] = get(articulation)
	}
	return result
}

func m16WireNoteIdentities(notes []m16WireNote) []int {
	result := make([]int, len(notes))
	for index, note := range notes {
		if note.InstrumentArticulation != nil {
			result[index] = *note.InstrumentArticulation
		}
	}
	return result
}

func m16WireNoteMIDIs(notes []m16WireNote) []int {
	result := make([]int, len(notes))
	for noteIndex, note := range notes {
		for _, property := range note.Properties.Properties {
			if property.Name == "Midi" && property.Number != nil {
				result[noteIndex] = *property.Number
			}
		}
	}
	return result
}
