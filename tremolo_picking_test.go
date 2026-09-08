// SPDX-License-Identifier: MIT

package goguitarpro

import "testing"

func TestParseGPIFRetainsTremoloPicking(t *testing.T) {
	song, err := ParseFile("testdata/gp7/tremolo-picking.gp")
	if err != nil {
		t.Fatal(err)
	}

	var noteCount int
	var marked []uint16
	for trackIndex := range song.Tracks {
		for staffIndex := range song.Tracks[trackIndex].Staves {
			staff := &song.Tracks[trackIndex].Staves[staffIndex]
			for measureIndex := range staff.Measures {
				for voiceIndex := range staff.Measures[measureIndex].Voices {
					for beatIndex := range staff.Measures[measureIndex].Voices[voiceIndex].Beats {
						beat := &staff.Measures[measureIndex].Voices[voiceIndex].Beats[beatIndex]
						noteCount += len(beat.Notes)
						if len(beat.Notes) == 0 || beat.Notes[0].Effect.TremoloPicking == nil {
							continue
						}
						value := beat.Notes[0].Effect.TremoloPicking.Duration.Value
						marked = append(marked, value)
						for noteIndex := range beat.Notes {
							effect := beat.Notes[noteIndex].Effect.TremoloPicking
							if effect == nil || effect.Duration.Value != value {
								t.Errorf("marked beat note %d tremolo = %#v, want duration %d", noteIndex, effect, value)
							}
						}
					}
				}
			}
		}
	}

	if noteCount != 15 {
		t.Errorf("notes = %d, want 15", noteCount)
	}
	want := []uint16{32, 16, 8, 32, 16, 8}
	if len(marked) != len(want) {
		t.Fatalf("marked beats = %d, want %d", len(marked), len(want))
	}
	for index := range want {
		if marked[index] != want[index] {
			t.Errorf("marked beat %d duration = %d, want %d", index, marked[index], want[index])
		}
	}
}
