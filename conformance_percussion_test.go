// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func TestConformancePercussionIdentity(t *testing.T) {
	runConformancePercussionIdentity(newConformanceRun(t))
}

func runConformancePercussionIdentity(run *conformanceRun) {
	t := run.t
	song := conformancePercussionSong(t)
	track := &song.Tracks[0]
	wantDefinitions := slices.Clone(track.PercussionArticulations)
	run.Field("Track.PercussionTrack", track.PercussionTrack, true)
	run.Field("Staff.PercussionTrack", track.Staves[0].PercussionTrack, true)
	run.Field("Staff.StandardNotationLineCount", track.Staves[0].StandardNotationLineCount, 7)
	run.Preserved("Track.PercussionArticulations", track.PercussionArticulations, wantDefinitions)
	for index, articulation := range track.PercussionArticulations {
		want := wantDefinitions[index]
		run.Preserved("PercussionArticulation.ElementName", articulation.ElementName, want.ElementName)
		run.Preserved("PercussionArticulation.ElementType", articulation.ElementType, want.ElementType)
		run.Preserved("PercussionArticulation.ElementSoundbankName", articulation.ElementSoundbankName, want.ElementSoundbankName)
		run.Preserved("PercussionArticulation.Name", articulation.Name, want.Name)
		run.Preserved("PercussionArticulation.StaffLine", articulation.StaffLine, want.StaffLine)
		run.Preserved("PercussionArticulation.NoteheadDefault", articulation.NoteheadDefault, want.NoteheadDefault)
		run.Preserved("PercussionArticulation.NoteheadHalf", articulation.NoteheadHalf, want.NoteheadHalf)
		run.Preserved("PercussionArticulation.NoteheadWhole", articulation.NoteheadWhole, want.NoteheadWhole)
		run.Preserved("PercussionArticulation.TechniquePlacement", articulation.TechniquePlacement, want.TechniquePlacement)
		run.Preserved("PercussionArticulation.TechniqueSymbol", articulation.TechniqueSymbol, want.TechniqueSymbol)
		run.Preserved("PercussionArticulation.InputMIDINumbers", articulation.InputMIDINumbers, want.InputMIDINumbers)
		run.Preserved("PercussionArticulation.OutputRSESound", articulation.OutputRSESound, want.OutputRSESound)
		run.Preserved("PercussionArticulation.OutputMIDINumber", articulation.OutputMIDINumber, want.OutputMIDINumber)
	}
	mainNotes := track.Measures[0].Voices[0].Beats[0].Notes
	run.Field("Note.HasPercussionArticulation", []bool{mainNotes[0].HasPercussionArticulation, mainNotes[1].HasPercussionArticulation, mainNotes[2].HasPercussionArticulation}, []bool{true, true, false})
	run.ClaimPrimary(claimSite("percussion", "model", "M16-PERCUSSION-IDENTITY", "every articulation field")).Field("Note.PercussionArticulation", []int{mainNotes[0].PercussionArticulation, mainNotes[1].PercussionArticulation}, []int{0, 2})
	run.Field("Note.Value", []int16{mainNotes[0].Value, mainNotes[1].Value, mainNotes[2].Value}, []int16{91, 46, 40})
	run.Field("GraceEffect.HasPercussionArticulation", []bool{mainNotes[0].Effect.Graces[0].HasPercussionArticulation, mainNotes[1].Effect.Graces[0].HasPercussionArticulation}, []bool{true, false})
	run.Field("GraceEffect.PercussionArticulation", mainNotes[0].Effect.Graces[0].PercussionArticulation, 0)

	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	wire := extractPercussionWire(t, data)
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
	flat := flattenPercussionWireArticulations(wireTrack.InstrumentSet.Elements)
	if len(flat) != 4 {
		t.Fatalf("wire articulations = %#v, want three custom and one builtin fallback", flat)
	}
	run.Wire("gpifElement.Name", conformancePercussionElementStrings(wireTrack.InstrumentSet.Elements, func(element conformancePercussionWireElement) string { return element.Name }), []string{"Metal", "Skin", "Metal", "Charley"})
	run.Wire("gpifElement.Type", conformancePercussionElementStrings(wireTrack.InstrumentSet.Elements, func(element conformancePercussionWireElement) string { return element.Type }), []string{"cymbal", "drum", "cymbal", "hiHat"})
	run.Wire("gpifElement.SoundbankName", conformancePercussionElementStrings(wireTrack.InstrumentSet.Elements, func(element conformancePercussionWireElement) string { return element.SoundbankName }), []string{"SB-A", "SB-B", "SB-A", "Master-Hihat"})
	run.Wire("gpifElement.Articulations", len(flat), 4)
	run.Wire("gpifArticulations.Articulations", len(flat), 4)
	firstThree := flat[:3]
	run.Wire("gpifArticulation.Name", conformancePercussionArticulationStrings(firstThree, func(articulation conformancePercussionWireArticulation) string { return articulation.Name }), []string{"Edge", "Center", "Bell"})
	run.Wire("gpifArticulation.StaffLine", conformancePercussionArticulationInts(firstThree, func(articulation conformancePercussionWireArticulation) int { return articulation.StaffLine }), []int{-3, 3, -1})
	run.Wire("gpifArticulation.Noteheads", conformancePercussionArticulationStrings(firstThree, func(articulation conformancePercussionWireArticulation) string { return articulation.Noteheads }), []string{"noteheadHeavyX noteheadHalf noteheadWhole", "noteheadBlack noteheadDiamondWhite noteheadCircleX", "noteheadXBlack noteheadXHalf noteheadXWhole"})
	run.Wire("gpifArticulation.TechniquePlacement", conformancePercussionArticulationStrings(firstThree, func(articulation conformancePercussionWireArticulation) string {
		return articulation.TechniquePlacement
	}), []string{"above", "inside", "below"})
	run.Wire("gpifArticulation.TechniqueSymbol", conformancePercussionArticulationStrings(firstThree, func(articulation conformancePercussionWireArticulation) string { return articulation.TechniqueSymbol }), []string{"pictEdgeOfCymbal", "articStaccatoAbove", "stringsDownBow"})
	run.Wire("gpifArticulation.InputMIDINumbers", conformancePercussionArticulationStrings(firstThree, func(articulation conformancePercussionWireArticulation) string { return articulation.InputMIDINumbers }), []string{"91 92", "38 40", "93"})
	run.Wire("gpifArticulation.OutputRSESound", conformancePercussionArticulationStrings(firstThree, func(articulation conformancePercussionWireArticulation) string { return articulation.OutputRSESound }), []string{"edge.hit", "center.hit", "bell.hit"})
	run.Wire("gpifArticulation.OutputMIDINumber", conformancePercussionArticulationInts(firstThree, func(articulation conformancePercussionWireArticulation) int { return articulation.OutputMIDINumber }), []int{56, 38, 56})
	identities := conformancePercussionWireNoteIdentities(wire.notes)
	run.Wire("gpifNote.InstrumentArticulation", identities, []int{0, 3, 0, 2, 1})
	run.Wire("gpifNote.Properties", len(wire.notes), 5)
	run.Wire("gpifProperty.Number", conformancePercussionWireNoteMIDIs(wire.notes), []int{91, 46, 91, 46, 40})

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

func TestConformanceSourceAndValidation(t *testing.T) {
	runConformanceSourceAndValidation(newConformanceRun(t))
}

func runConformanceSourceAndValidation(run *conformanceRun) {
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
				run.ClaimPrimary(claimSite("percussion", "import", "M16-SOURCE-VALIDATION", "every articulation field")).Dispatch("isPercussionTrack:t.InstrumentSet.Type", got, true)
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

	negative := conformancePercussionSong(t)
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
			song := conformancePercussionSong(t)
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
	unsupported.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Value = 27
	unsupported.Tracks[0].Staves[0].Measures = unsupported.Tracks[0].Measures
	report := PreflightExport(unsupported, ExportFormatGP8, ExportOptions{})
	run.Dispatch("validateGP8Staff:element.Type", slices.ContainsFunc(report.Entries, func(entry ExportReportEntry) bool { return entry.Disposition == ExportDispositionRejected }), true)
}

func TestConformanceNativePercussionFallbacks(t *testing.T) {
	runConformanceNativePercussionFallbacks(newConformanceRun(t))
}

func runConformanceNativePercussionFallbacks(run *conformanceRun) {
	t := run.t
	for _, test := range conformanceNativePercussionCases() {
		t.Run(test.name, func(t *testing.T) {
			song := conformancePercussionBuiltinPercussionSong(t, test.midi)
			exact, err := NewFret(int64(test.midi))
			if err != nil {
				t.Fatal(err)
			}
			note := &song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0]
			note.Effect.Graces = []GraceEffect{{
				Fret:       int8(test.midi),
				ExactFret:  &exact,
				Duration:   DurationThirtySecond,
				Velocity:   Forte,
				Transition: GraceEffectTransitionNone,
			}}
			song.Tracks[0].Staves[0].Measures = song.Tracks[0].Measures

			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{
				LossPolicy: ExportLossPolicy{RequirePreservation: true},
			})
			if err != nil || len(report.Entries) != 0 {
				t.Fatalf("native percussion export = %v, %#v", err, report.Entries)
			}
			wire := extractPercussionWire(t, data)
			flat := flattenPercussionWireArticulations(wire.tracks[0].InstrumentSet.Elements)
			if len(flat) != 1 {
				t.Fatalf("wire articulations = %#v, want one", flat)
			}
			articulation := flat[0]
			run.Wire("gpifArticulation.InputMIDINumbers", articulation.InputMIDINumbers, test.input)
			run.Wire("gpifArticulation.Name", articulation.Name, test.articulation)
			run.Wire("gpifArticulation.StaffLine", articulation.StaffLine, test.staffLine)
			run.Wire("gpifArticulation.Noteheads", articulation.Noteheads, test.noteheads)
			run.Wire("gpifArticulation.TechniquePlacement", articulation.TechniquePlacement, test.placement)
			run.Wire("gpifArticulation.TechniqueSymbol", articulation.TechniqueSymbol, test.symbol)
			run.Wire("gpifArticulation.OutputRSESound", articulation.OutputRSESound, test.rse)
			run.Wire("gpifArticulation.OutputMIDINumber", articulation.OutputMIDINumber, test.outputMIDI)
			element := wire.tracks[0].InstrumentSet.Elements[0]
			run.Wire("gpifElement.Name", element.Name, test.element)
			run.Wire("gpifElement.Type", element.Type, test.kind)
			run.Wire("gpifElement.SoundbankName", element.SoundbankName, test.soundbank)
			run.Wire("gpifNote.Properties", conformancePercussionWireNoteMIDIs(wire.notes), []int{int(test.midi), int(test.midi)})

			roundTrip, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			gotTrack := &roundTrip.Tracks[0]
			gotNote := gotTrack.Staves[0].Measures[0].Voices[0].Beats[0].Notes[0]
			run.Preserved("Note.Value", gotNote.Value, test.midi)
			if len(gotNote.Effect.Graces) != 1 {
				t.Fatalf("round-trip graces = %#v, want one", gotNote.Effect.Graces)
			}
			run.Preserved("GraceEffect.ExactFret", *gotNote.Effect.Graces[0].ExactFret, exact)
			if len(gotTrack.PercussionArticulations) != 1 {
				t.Fatalf("round-trip articulations = %#v, want one", gotTrack.PercussionArticulations)
			}
			run.Preserved("Track.PercussionArticulations", gotTrack.PercussionArticulations[0].InputMIDINumbers, []int{int(test.midi)})
		})
	}

	for _, midi := range []int16{27, 28, 32} {
		t.Run("unsupported "+strconv.Itoa(int(midi)), func(t *testing.T) {
			song := conformancePercussionBuiltinPercussionSong(t, midi)
			_, err := Export(song, ExportFormatGP8)
			want := "percussion MIDI value " + strconv.Itoa(int(midi)) + " without a native Guitar Pro drum-kit articulation"
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Fatalf("unsupported main-note export = %v, want %q", err, want)
			}

			graceSong := conformancePercussionBuiltinPercussionSong(t, 38)
			exact, newErr := NewFret(int64(midi))
			if newErr != nil {
				t.Fatal(newErr)
			}
			graceSong.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces = []GraceEffect{{Fret: int8(midi), ExactFret: &exact, Duration: DurationThirtySecond, Velocity: Forte}}
			graceSong.Tracks[0].Staves[0].Measures = graceSong.Tracks[0].Measures
			_, err = Export(graceSong, ExportFormatGP8)
			if err == nil || !strings.Contains(err.Error(), "grace 0 uses "+want) {
				t.Fatalf("unsupported grace export = %v, want grace-scoped %q", err, want)
			}
		})
	}

	for _, test := range []struct {
		name  string
		exact Fret
		want  string
	}{
		{name: "exact below native table", exact: 26, want: "grace 0 uses percussion MIDI value 26 without a native Guitar Pro drum-kit articulation"},
		{name: "exact above native table", exact: 88, want: "grace 0 uses percussion MIDI value 88 without a native Guitar Pro drum-kit articulation"},
		{name: "exact above MIDI range", exact: 128, want: "grace 0 has MIDI value 128 outside 0..127"},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := conformancePercussionBuiltinPercussionSong(t, 38)
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces = []GraceEffect{{
				Fret: 0, ExactFret: &test.exact, Duration: DurationThirtySecond, Velocity: Forte,
			}}
			song.Tracks[0].Staves[0].Measures = song.Tracks[0].Measures
			if _, err := Export(song, ExportFormatGP8); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("unsupported exact grace export = %v, want %q", err, test.want)
			}
		})
	}

	t.Run("custom exact grace input remains authoritative", func(t *testing.T) {
		song := conformancePercussionBuiltinPercussionSong(t, 38)
		track := &song.Tracks[0]
		track.PercussionArticulations = []PercussionArticulation{{
			ElementName: "Custom", ElementType: "percussion", Name: "Exact 26", StaffLine: 4,
			NoteheadDefault: "noteheadDiamondWhite", InputMIDINumbers: []int{26}, OutputMIDINumber: 26,
		}}
		exact := Fret(26)
		track.Measures[0].Voices[0].Beats[0].Notes[0].Effect.Graces = []GraceEffect{{
			Fret: 26, ExactFret: &exact, Duration: DurationThirtySecond, Velocity: Forte,
		}}
		track.Staves[0].Measures = track.Measures
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
		if err != nil || len(report.Entries) != 0 {
			t.Fatalf("custom exact grace export = %v, %#v", err, report.Entries)
		}
		wire := extractPercussionWire(t, data)
		if got := conformancePercussionWireNoteMIDIs(wire.notes); !slices.Equal(got, []int{26, 38}) {
			t.Fatalf("custom exact grace wire MIDI = %v, want [26 38]", got)
		}
		if got := flattenPercussionWireArticulations(wire.tracks[0].InstrumentSet.Elements); len(got) != 2 || got[0].InputMIDINumbers != "26" || got[0].Name != "Exact 26" {
			t.Fatalf("custom exact grace definitions = %#v", got)
		}
	})
}

func TestGP5NativePercussionFallbackFixtures(t *testing.T) {
	for _, test := range []struct {
		path   string
		values []int16
	}{
		{path: "testdata/gp5/canon.gp5", values: []int16{31, 59}},
		{path: "testdata/gp5/full-song.gp5", values: []int16{54}},
		{path: "testdata/gp5/nightwish.gp5", values: []int16{40, 54}},
	} {
		t.Run(test.path, func(t *testing.T) {
			song := parseTestFixture(t, test.path)
			if _, err := Export(song, ExportFormatGP8); err != nil {
				t.Fatalf("GP8 export still rejects native values %v: %v", test.values, err)
			}
			got := conformancePercussionValues(song)
			for _, want := range test.values {
				if !slices.Contains(got, want) {
					t.Errorf("source percussion values = %v, want %d", got, want)
				}
			}
		})
	}

	const upstream = "references/alphaTab/packages/alphatab/test-data/guitarpro5/percussion-all.gp5"
	if _, err := os.Stat(upstream); err != nil {
		t.Skip("pinned AlphaTab source checkout is unavailable")
	}
	song := parseTestFixture(t, upstream)
	got := conformancePercussionValues(song)
	want := make([]int16, 0, 61)
	for value := int16(27); value <= 87; value++ {
		want = append(want, value)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("upstream percussion values = %v, want %v", got, want)
	}
	if _, err := Export(song, ExportFormatGP8); err == nil || !strings.Contains(err.Error(), "percussion MIDI value 27 without a native Guitar Pro drum-kit articulation") {
		t.Fatalf("upstream unsupported-value rejection = %v", err)
	}
}

func conformancePercussionValues(song *Song) []int16 {
	seen := make(map[int16]bool)
	for trackIndex := range song.Tracks {
		track := &song.Tracks[trackIndex]
		if !track.PercussionTrack {
			continue
		}
		for _, staff := range gp8ExportStaves(track) {
			for _, measure := range staff.Measures {
				for _, voice := range measure.Voices {
					for _, beat := range voice.Beats {
						for _, note := range beat.Notes {
							seen[note.Value] = true
							for index := range note.Effect.Graces {
								seen[gp8PercussionGraceNote(&note, &note.Effect.Graces[index]).Value] = true
							}
						}
					}
				}
			}
		}
	}
	values := make([]int16, 0, len(seen))
	for value := range seen {
		values = append(values, value)
	}
	slices.Sort(values)
	return values
}

type conformanceNativePercussionCase struct {
	name, input, element, kind, soundbank, articulation string
	midi                                                int16
	staffLine, outputMIDI                               int
	noteheads, placement, symbol, rse                   string
}

func conformanceNativePercussionCases() []conformanceNativePercussionCase {
	black := "noteheadBlack noteheadHalf noteheadWhole"
	x := "noteheadXBlack noteheadXBlack noteheadXBlack"
	triangle := "noteheadTriangleUpBlack noteheadTriangleUpBlack noteheadTriangleUpBlack"
	makeCase := func(midi int16, element, kind, soundbank, articulation string, staffLine int, noteheads, rse string) conformanceNativePercussionCase {
		return conformanceNativePercussionCase{name: strconv.Itoa(int(midi)) + " " + element, input: strconv.Itoa(int(midi)), midi: midi, element: element, kind: kind, soundbank: soundbank, articulation: articulation, staffLine: staffLine, noteheads: noteheads, placement: "outside", outputMIDI: int(midi), rse: rse}
	}
	cases := []conformanceNativePercussionCase{
		makeCase(29, "Ride Cymbal 2", "ride", "Ride-Percu", "Ride (choke)", 2, x, "stick.hit.choke"),
		makeCase(30, "Reverse Cymbal", "crash", "Reverse-Cymbal", "Reverse Cymbal (hit)", -3, x, "stick.hit.hit"),
		makeCase(31, "Sticks", "snare", "Stick-Percu", "Snare (side stick)", 3, "noteheadSlashedBlack2 noteheadSlashedBlack2 noteheadSlashedBlack2", "stick.hit.sidestick"),
		makeCase(33, "Metronome", "snare", "Metronome-Percu", "Metronome (hit)", 3, x, "stick.hit.sidestick"),
		makeCase(34, "Metronome", "snare", "Metronome-Percu", "Metronome (bell)", 3, "noteheadBlack noteheadBlack noteheadBlack", "stick.hit.hit"),
		makeCase(39, "Hand Clap", "handClap", "GroupHandClap-Percu", "Hand Clap (hit)", 3, black, "hand.hit.hit"),
		makeCase(40, "Electric Snare", "snare", "ElectricSnare-Percu", "Electric Snare (hit)", 3, black, "stick.hit.hit"),
		makeCase(54, "Tambourine", "tambourine", "Tambourine-Percu", "Tambourine (hit)", 3, triangle, "hand.hit.hit"),
		makeCase(56, "Cowbell Medium", "cowbell", "CowbellMid-Percu", "Cowbell medium (hit)", 0, "noteheadTriangleUpBlack noteheadTriangleUpHalf noteheadTriangleUpWhole", "stick.hit.hit"),
		makeCase(58, "Vibraslap", "vibraslap", "Vibraslap-Percu", "Vibraslap (hit)", 28, black, "hand.hit.hit"),
		makeCase(59, "Ride Cymbal 2", "ride", "Ride-Percu", "Ride (edge)", 2, x, "stick.hit.edge"),
		makeCase(60, "Bongo High", "bongo", "BongoHigh-Percu", "Bongo High (hit)", -4, black, "hand.hit.hit"),
		makeCase(61, "Bongo Low", "bongo", "BongoLow-Percu", "Bongo Low (hit)", -7, black, "hand.hit.hit"),
		makeCase(62, "Conga High", "conga", "CongaHigh-Percu", "Conga high (mute)", 19, black, "hand.hit.mute"),
		makeCase(63, "Conga High", "conga", "CongaHigh-Percu", "Conga high (hit)", 14, black, "hand.hit.hit"),
		makeCase(64, "Conga Low", "conga", "CongaLow-Percu", "Conga low (hit)", 17, black, "hand.hit.hit"),
		makeCase(65, "Timbale High", "timbale", "TimbaleHigh-Percu", "Timbale high (hit)", 9, black, "stick.hit.hit"),
		makeCase(66, "Timbale Low", "timbale", "TimbaleLow-Percu", "Timbale low (hit)", 10, black, "stick.hit.hit"),
		makeCase(67, "Agogo High", "agogo", "AgogoHigh-Percu", "Agogo high (hit)", 11, black, "stick.hit.hit"),
		makeCase(68, "Agogo Low", "agogo", "AgogoLow-Percu", "Agogo low (hit)", 12, black, "stick.hit.hit"),
		makeCase(69, "Cabasa", "cabasa", "Cabasa-Percu", "Cabasa (hit)", 23, black, "hand.hit.hit"),
		makeCase(70, "Left Maraca", "maraca", "Maracas-Percu", "Left Maraca (hit)", -12, black, "hand.hit.hit"),
		makeCase(71, "Whistle High", "whistle", "WhistleHigh-Percu", "Whistle high (hit)", -17, black, "blow.hit.hit"),
		makeCase(72, "Whistle Low", "whistle", "WhistleLow-Percu", "Whistle low (hit)", -11, black, "blow.hit.hit"),
		makeCase(73, "Guiro", "guiro", "Guiro-Percu", "Guiro (hit)", 38, black, "stick.hit.hit"),
		makeCase(74, "Guiro", "guiro", "Guiro-Percu", "Guiro (scrap-return)", 37, black, "stick.scrape.return"),
		makeCase(75, "Claves", "claves", "Claves-Percu", "Claves (hit)", 20, black, "stick.hit.hit"),
		makeCase(76, "Woodblock High", "woodblock", "WoodblockHigh-Percu", "Woodblock high (hit)", -10, triangle, "stick.hit.hit"),
		makeCase(77, "Woodblock Low", "woodblock", "WoodblockLow-Percu", "Woodblock low (hit)", -9, triangle, "stick.hit.hit"),
		makeCase(78, "Cuica", "cuica", "Cuica-Percu", "Cuica (mute)", 29, x, "hand.hit.mute"),
		makeCase(79, "Cuica", "cuica", "Cuica-Percu", "Cuica (open)", 30, black, "hand.hit.hit"),
		makeCase(80, "Triangle", "triangle", "Triangle-Percu", "Triangle (mute)", 26, x, "stick.hit.mute"),
		makeCase(81, "Triangle", "triangle", "Triangle-Percu", "Triangle (hit)", 27, black, "stick.hit.hit"),
		makeCase(82, "Shaker", "shaker", "ShakerStudio-Percu", "Shaker (hit)", -23, black, "hand.hit.hit"),
		makeCase(83, "Tinkle Bell", "jingleBell", "JingleBell-Percu", "Tinkle Bell (hit)", -20, black, "stick.hit.hit"),
		makeCase(84, "Bell Tree", "bellTree", "BellTree-Percu", "Bell Tree (hit)", -18, black, "stick.hit.hit"),
		makeCase(85, "Castanets", "castanets", "Castanets-Percu", "Castanets (hit)", 21, black, "hand.hit.hit"),
		makeCase(86, "Surdo", "surdo", "Surdo-Percu", "Surdo (hit)", 36, black, "brush.hit.hit"),
		makeCase(87, "Surdo", "surdo", "Surdo-Percu", "Surdo (mute)", 35, x, "brush.hit.mute"),
	}
	for index := range cases {
		switch cases[index].midi {
		case 29:
			cases[index].outputMIDI = 59
			cases[index].symbol = "articStaccatoAbove"
		case 30:
			cases[index].outputMIDI = 49
		case 31:
			cases[index].outputMIDI = 40
		case 33:
			cases[index].outputMIDI = 37
		case 34:
			cases[index].outputMIDI = 38
		case 59:
			cases[index].placement = "above"
			cases[index].symbol = "pictEdgeOfCymbal"
		case 62, 80, 87:
			cases[index].placement = "inside"
			cases[index].symbol = "noteheadParenthesis"
		case 83, 84:
			cases[index].outputMIDI = 53
		}
	}
	return cases
}

func TestConformanceNoteheadOptions(t *testing.T) {
	runConformanceNoteheadOptions(newConformanceRun(t))
}

func runConformanceNoteheadOptions(run *conformanceRun) {
	for _, test := range []struct {
		name   string
		member string
		kind   GP8PercussionNotehead
		want   string
	}{
		{name: "native", member: "GP8PercussionNoteheadDefault", kind: GP8PercussionNoteheadDefault, want: "noteheadCircleX noteheadCircleX noteheadCircleX"},
		{name: "filled", member: "GP8PercussionNoteheadFilled", kind: GP8PercussionNoteheadFilled, want: "noteheadBlack noteheadHalf noteheadWhole"},
		{name: "x", member: "GP8PercussionNoteheadX", kind: GP8PercussionNoteheadX, want: "noteheadXBlack noteheadXBlack noteheadXBlack"},
		{name: "circle x", member: "GP8PercussionNoteheadCircleX", kind: GP8PercussionNoteheadCircleX, want: "noteheadCircleX noteheadCircleX noteheadCircleX"},
		{name: "heavy x", member: "GP8PercussionNoteheadHeavyX", kind: GP8PercussionNoteheadHeavyX, want: "noteheadHeavyX noteheadHeavyX noteheadHeavyX"},
	} {
		run.t.Run(test.name, func(t *testing.T) {
			song := conformancePercussionBuiltinPercussionSong(t, 46)
			options := ExportOptions{GP8: GP8ExportOptions{PercussionNoteheads: map[int16]GP8PercussionNotehead{46: test.kind}}}
			data, err := ExportWithOptions(song, ExportFormatGP8, options)
			if err != nil {
				t.Fatal(err)
			}
			wire := extractPercussionWire(t, data)
			flat := flattenPercussionWireArticulations(wire.tracks[0].InstrumentSet.Elements)
			if len(flat) != 1 {
				t.Fatalf("articulations = %#v, want one", flat)
			}
			run.Wire("gpifArticulation.Noteheads", flat[0].Noteheads, test.want)
			run.Enum("GP8PercussionNotehead."+test.member, flat[0].Noteheads, test.want)
		})
	}

	invalid := conformancePercussionBuiltinPercussionSong(run.t, 46)
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

func conformancePercussionSong(t *testing.T) *Song {
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

func conformancePercussionBuiltinPercussionSong(t *testing.T, midi int16) *Song {
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

func conformanceNativePercussionSong(t *testing.T) *Song {
	t.Helper()
	song := conformancePercussionBuiltinPercussionSong(t, 38)
	notes := make([]Note, 0, len(conformanceNativePercussionCases()))
	for _, percussion := range conformanceNativePercussionCases() {
		notes = append(notes, Note{Value: percussion.midi, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1})
	}
	song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes = notes
	song.Tracks[0].Staves[0].Measures = song.Tracks[0].Measures
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	return song
}

type conformancePercussionWireDocument struct {
	Tracks struct {
		Tracks []conformancePercussionWireTrack `xml:"Track"`
	} `xml:"Tracks"`
	Notes struct {
		Notes []conformancePercussionWireNote `xml:"Note"`
	} `xml:"Notes"`
	tracks []conformancePercussionWireTrack
	notes  []conformancePercussionWireNote
}

type conformancePercussionWireTrack struct {
	InstrumentSet *conformancePercussionWireInstrumentSet `xml:"InstrumentSet"`
}

type conformancePercussionWireInstrumentSet struct {
	Name      string                             `xml:"Name"`
	Type      string                             `xml:"Type"`
	LineCount int                                `xml:"LineCount"`
	Elements  []conformancePercussionWireElement `xml:"Elements>Element"`
}

type conformancePercussionWireElement struct {
	Name          string                                  `xml:"Name"`
	Type          string                                  `xml:"Type"`
	SoundbankName string                                  `xml:"SoundbankName"`
	Articulations []conformancePercussionWireArticulation `xml:"Articulations>Articulation"`
}

type conformancePercussionWireArticulation struct {
	Name               string `xml:"Name"`
	StaffLine          int    `xml:"StaffLine"`
	Noteheads          string `xml:"Noteheads"`
	TechniquePlacement string `xml:"TechniquePlacement"`
	TechniqueSymbol    string `xml:"TechniqueSymbol"`
	InputMIDINumbers   string `xml:"InputMidiNumbers"`
	OutputRSESound     string `xml:"OutputRSESound"`
	OutputMIDINumber   int    `xml:"OutputMidiNumber"`
}

type conformancePercussionWireNote struct {
	InstrumentArticulation *int `xml:"InstrumentArticulation"`
	Properties             struct {
		Properties []struct {
			Name   string `xml:"name,attr"`
			Number *int   `xml:"Number"`
		} `xml:"Property"`
	} `xml:"Properties"`
}

func extractPercussionWire(t *testing.T, data []byte) conformancePercussionWireDocument {
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
	var wire conformancePercussionWireDocument
	if err := xml.Unmarshal(contents, &wire); err != nil {
		t.Fatal(err)
	}
	wire.tracks = wire.Tracks.Tracks
	wire.notes = wire.Notes.Notes
	return wire
}

func flattenPercussionWireArticulations(elements []conformancePercussionWireElement) []conformancePercussionWireArticulation {
	var result []conformancePercussionWireArticulation
	for _, element := range elements {
		result = append(result, element.Articulations...)
	}
	return result
}

func conformancePercussionElementStrings(elements []conformancePercussionWireElement, get func(conformancePercussionWireElement) string) []string {
	result := make([]string, len(elements))
	for index, element := range elements {
		result[index] = get(element)
	}
	return result
}

func conformancePercussionArticulationStrings(articulations []conformancePercussionWireArticulation, get func(conformancePercussionWireArticulation) string) []string {
	result := make([]string, len(articulations))
	for index, articulation := range articulations {
		result[index] = get(articulation)
	}
	return result
}

func conformancePercussionArticulationInts(articulations []conformancePercussionWireArticulation, get func(conformancePercussionWireArticulation) int) []int {
	result := make([]int, len(articulations))
	for index, articulation := range articulations {
		result[index] = get(articulation)
	}
	return result
}

func conformancePercussionWireNoteIdentities(notes []conformancePercussionWireNote) []int {
	result := make([]int, len(notes))
	for index, note := range notes {
		if note.InstrumentArticulation != nil {
			result[index] = *note.InstrumentArticulation
		}
	}
	return result
}

func conformancePercussionWireNoteMIDIs(notes []conformancePercussionWireNote) []int {
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
