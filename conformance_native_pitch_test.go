// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"os"
	"testing"
)

const nativeRepairPath = "conformance/capabilities/evidence/guitar-pro-repair/"

func TestConformanceNativePitch(t *testing.T) { runConformanceNativePitch(newConformanceRun(t)) }

func runConformanceNativePitch(run *conformanceRun) {
	runConformanceAutomaticPitchKeys(run)
	runConformanceNativeKeyAuthority(run)
	for _, test := range []struct {
		name                            string
		concert, written, nativeWritten []int64
	}{
		{"default", []int64{61}, []int64{61}, []int64{61}},
		{"flat", []int64{61}, []int64{61}, []int64{61}},
		{"capo", []int64{64}, []int64{64}, []int64{64}},
		{"display", []int64{61}, []int64{49}, []int64{49}},
		{"clef", []int64{61}, []int64{73}, []int64{61}},
		{"octave", []int64{61}, []int64{49}, []int64{61}},
		// Harmonic GPIF records retain the fret pitch; native MIDI proves overtone 84.
		{"harmonic", []int64{65}, []int64{65}, []int64{65}},
		{"absolute", []int64{61}, []int64{61}, []int64{61}},
		{"grace", []int64{63, 61}, []int64{63, 61}, []int64{63, 61}},
		{"sounding", []int64{61}, []int64{61}, []int64{61}},
		{"independent-capo", []int64{61, 64}, []int64{61, 64}, []int64{61, 64}},
	} {
		source := parseTestFixture(run.t, nativeRepairPath+"contexts/"+test.name+".gp")
		for si := range source.Tracks[0].Staves {
			for mi := range source.Tracks[0].Staves[si].Measures {
				for vi := range source.Tracks[0].Staves[si].Measures[mi].Voices {
					for bi := range source.Tracks[0].Staves[si].Measures[mi].Voices[vi].Beats {
						for ni := range source.Tracks[0].Staves[si].Measures[mi].Voices[vi].Beats[bi].Notes {
							note := &source.Tracks[0].Staves[si].Measures[mi].Voices[vi].Beats[bi].Notes[ni]
							if test.name != "flat" {
								note.AccidentalMode = NoteAccidentalDefault
							}
						}
					}
				}
			}
		}
		if test.name == "sounding" {
			source.Tracks[0].Staves[0].TranspositionPitch = 12
		}
		data, report, err := ExportWithReport(source, ExportFormatGP8, ExportOptions{})
		if err != nil {
			run.t.Fatalf("%s: %v", test.name, err)
		}
		if test.name == "sounding" && !hasExportCode(report, "gp8.omit.staff-sounding-transposition") {
			run.t.Fatal("sounding transposition limit unreported")
		}
		doc := conformanceWireDocument(run.t, data)
		assertNativePitchRecords(run, doc, test.concert, test.written)
		nativeData, err := os.ReadFile(nativeRepairPath + "contexts/" + test.name + "-native.gp")
		if err != nil {
			run.t.Fatal(err)
		}
		native := conformanceWireDocument(run.t, nativeData)
		assertNativePitchRecords(run, native, test.concert, test.nativeWritten)
		for i, note := range doc.Notes.Notes {
			// Native saves reorder properties. Compare their exact named values.
			for _, name := range []string{"Fret", "Midi"} {
				field := "gpifProperty.Number"
				if name == "Fret" {
					field = "gpifProperty.Fret"
				}
				run.Wire(field, nativeNoteScalar(native.Notes.Notes[i], name), nativeNoteScalar(note, name))
			}
			if test.name != "absolute" {
				run.Wire("gpifProperty.String", nativeNoteScalar(native.Notes.Notes[i], "String"), nativeNoteScalar(note, "String"))
			}
		}
	}
}

func assertNativePitchRecords(run *conformanceRun, doc gpifDocument, concert, written []int64) {
	run.Wire("gpifNote.Properties", len(doc.Notes.Notes), len(concert))
	for i, note := range doc.Notes.Notes {
		found := map[string]bool{}
		for _, p := range note.Properties.Properties {
			if p.Name != "ConcertPitch" && p.Name != "TransposedPitch" {
				continue
			}
			actual, valid := gpifPitchMIDI(p.Pitch)
			run.Wire("gpifProperty.Pitch", valid, true)
			want := concert[i]
			if p.Name == "TransposedPitch" {
				want = written[i]
			}
			run.Wire("gpifProperty.Pitch", actual, want)
			run.Wire("gpifPitch.Octave", p.Pitch.Octave, int(want/12))
			found[p.Name] = true
		}
		run.Wire("gpifProperty.Name", found, map[string]bool{"ConcertPitch": true, "TransposedPitch": true})
	}
}

func nativeNoteScalar(note gpifNote, name string) any {
	for _, p := range note.Properties.Properties {
		if p.Name == name {
			switch name {
			case "Fret":
				return *p.Fret
			case "String":
				return *p.String
			default:
				return *p.Number
			}
		}
	}
	return -1
}

func TestConformanceNativeVolume(t *testing.T) { runConformanceNativeVolume(newConformanceRun(t)) }
func runConformanceNativeVolume(run *conformanceRun) {
	source, err := os.ReadFile("testdata/gp3/mix-table-events.gp3")
	if err != nil {
		run.t.Fatal(err)
	}
	for i, want := range []float64{0, .05, .10, .14, .20, .25, .31, .37, .43, .48, .50, .53, .56, .58, .61, .64, .66} {
		input := append([]byte(nil), source...)
		input[980] = byte(i)
		song, err := Parse(input)
		if err != nil {
			run.t.Fatal(err)
		}
		a := song.VolumeAutomations[0]
		run.Preserved("VolumeAutomation.Value", a.Value, want)
		run.Preserved("VolumeAutomation.Linear", a.Linear, false)
		run.Preserved("VolumeAutomation.Position", a.Position, .25)
		data, err := Export(song, ExportFormatGP8)
		if err != nil {
			run.t.Fatal(err)
		}
		doc := conformanceWireDocument(run.t, data)
		run.Wire("gpifChannelStrip.Automations", doc.Tracks.Tracks[0].RSE.ChannelStrip.Automations.Automations[0].Linear, false)
	}
}

func automaticKeySong(t *testing.T, master, compatibility int8, display int32, midi int16) *Song {
	t.Helper()
	song := accidentalSong(t, NoteAccidentalDefault, midi)
	song.MeasureHeaders[0].KeySignature.Key = master
	staff := &song.Tracks[0].Staves[0]
	staff.DisplayTranspositionPitch = display
	staff.Measures[0].KeySignature.Key = compatibility
	return song
}

func runConformanceAutomaticPitchKeys(run *conformanceRun) {
	for _, test := range []struct {
		master, compatibility      int8
		display                    int32
		midi                       int16
		concert, written           string
		concertToken, writtenToken string
	}{
		{-2, 0, 0, 70, "B", "B", "b", "b"},
		{2, -2, 0, 70, "A", "A", "#", "#"},
		{0, 0, -3, 60, "C", "E", "", "b"},
	} {
		song := automaticKeySong(run.t, test.master, test.compatibility, test.display, test.midi)
		data, err := Export(song, ExportFormatGP8)
		if err != nil {
			run.t.Fatal(err)
		}
		doc := conformanceWireDocument(run.t, data)
		for _, property := range doc.Notes.Notes[0].Properties.Properties {
			if property.Pitch == nil {
				continue
			}
			step, token := test.concert, test.concertToken
			if property.Name == "TransposedPitch" {
				step, token = test.written, test.writtenToken
			}
			run.Wire("gpifPitch.Step", property.Pitch.Step, step)
			run.Wire("gpifPitch.Accidental", *property.Pitch.Accidental, token)
		}
	}
}

func TestAlphaTabAutomaticPitchKeyAuthority(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, test := range []struct {
		master, compatibility int8
		display               int32
		midi                  int16
		mode, written         int
	}{
		{-2, 0, 0, 70, 5, 70},
		{2, -2, 0, 70, 3, 70},
		{0, 0, -3, 60, 5, 63},
	} {
		data, err := Export(automaticKeySong(t, test.master, test.compatibility, test.display, test.midi), ExportFormatGP8)
		if err != nil {
			t.Fatal(err)
		}
		var facts []accidentalFact
		readAlphaTabOracleFacts(t, "--accidental-facts", writeConformanceFixture(t, data), &facts)
		if len(facts) != 1 || facts[0].Mode != test.mode || facts[0].MIDI != int(test.midi) || facts[0].Display != test.written {
			t.Fatalf("master %d display %d: %#v", test.master, test.display, facts)
		}
	}
}

func runConformanceNativeKeyAuthority(run *conformanceRun) {
	for _, test := range []struct {
		name    string
		written int64
	}{{"minus-one", 61}, {"plus-eleven", 49}} {
		path := "conformance/capabilities/evidence/guitar-pro-key-authority/" + test.name
		song := parseTestFixture(run.t, path+"-export.gp")
		note := &song.Tracks[0].Staves[0].Measures[0].Voices[0].Beats[0].Notes[0]
		note.AccidentalMode = NoteAccidentalDefault
		data, err := Export(song, ExportFormatGP8)
		if err != nil {
			run.t.Fatal(err)
		}
		assertNativePitchRecords(run, conformanceWireDocument(run.t, data), []int64{60}, []int64{test.written})
		nativeData, err := os.ReadFile(path + "-native.gp")
		if err != nil {
			run.t.Fatal(err)
		}
		doc := conformanceWireDocument(run.t, nativeData)
		assertNativePitchRecords(run, doc, []int64{60}, []int64{test.written})
		for _, property := range doc.Notes.Notes[0].Properties.Properties {
			if property.Name == "TransposedPitch" {
				run.Wire("gpifPitch.Step", property.Pitch.Step, "D")
				run.Wire("gpifPitch.Accidental", *property.Pitch.Accidental, "b")
			}
		}
	}
}
