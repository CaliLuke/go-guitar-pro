// SPDX-License-Identifier: MIT

package goguitarpro

import "testing"

func TestParseGPIFRetainsHarmonicFrets(t *testing.T) {
	song, err := ParseFile("testdata/gp7/harmonics.gp")
	if err != nil {
		t.Fatal(err)
	}

	var got []float64
	for trackIndex := range song.Tracks {
		for staffIndex := range song.Tracks[trackIndex].Staves {
			staff := &song.Tracks[trackIndex].Staves[staffIndex]
			for measureIndex := range staff.Measures {
				for voiceIndex := range staff.Measures[measureIndex].Voices {
					for beatIndex := range staff.Measures[measureIndex].Voices[voiceIndex].Beats {
						for _, note := range staff.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex].Notes {
							if note.Effect.Harmonic != nil && note.Effect.Harmonic.FretFloat != nil {
								got = append(got, *note.Effect.Harmonic.FretFloat)
							}
						}
					}
				}
			}
		}
	}

	want := []float64{2.4, 12, 12, 12, 12}
	if len(got) != len(want) {
		t.Fatalf("harmonic frets = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("harmonic fret %d = %v, want %v", index, got[index], want[index])
		}
	}
}

func TestParseGPIFHarmonicFretBeforeType(t *testing.T) {
	harmonicFret := "2.4"
	harmonicType := "Artificial"
	note := gpifNoteToNote(&gpifNote{Properties: gpifProperties{Properties: []gpifProperty{
		{Name: "HarmonicFret", HFret: &harmonicFret},
		{Name: "HarmonicType", HType: &harmonicType},
	}}}, 6, false)

	if note.Effect.Harmonic == nil || note.Effect.Harmonic.FretFloat == nil || *note.Effect.Harmonic.FretFloat != 2.4 {
		t.Fatalf("harmonic = %#v, want artificial at fret 2.4", note.Effect.Harmonic)
	}
	if note.Effect.Harmonic.Kind != HarmonicTypeArtificial {
		t.Errorf("harmonic kind = %v, want artificial", note.Effect.Harmonic.Kind)
	}
}
