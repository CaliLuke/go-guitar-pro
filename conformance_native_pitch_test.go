// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"os"
	"testing"
)

const nativeRepairPath = "conformance/capabilities/evidence/guitar-pro-repair/"

func TestConformanceNativePitch(t *testing.T) { runConformanceNativePitch(newConformanceRun(t)) }

func runConformanceNativePitch(run *conformanceRun) {
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
		found := 0
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
			found++
		}
		run.Wire("gpifProperty.Name", found, 2)
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
