// SPDX-License-Identifier: MIT

package goguitarpro

import "testing"

func TestParseGPIFRetainsHairpins(t *testing.T) {
	song, err := ParseFile("testdata/gp7/notation-legend.gp")
	if err != nil {
		t.Fatal(err)
	}

	var crescendo int
	var decrescendo int
	for trackIndex := range song.Tracks {
		for staffIndex := range song.Tracks[trackIndex].Staves {
			staff := &song.Tracks[trackIndex].Staves[staffIndex]
			for measureIndex := range staff.Measures {
				for voiceIndex := range staff.Measures[measureIndex].Voices {
					for _, beat := range staff.Measures[measureIndex].Voices[voiceIndex].Beats {
						switch beat.Effect.Hairpin {
						case HairpinCrescendo:
							crescendo++
						case HairpinDiminuendo:
							decrescendo++
						}
					}
				}
			}
		}
	}

	if crescendo != 5 || decrescendo != 3 {
		t.Errorf("hairpins = %d crescendo/%d decrescendo, want 5/3", crescendo, decrescendo)
	}
}

func TestGPIFHairpinAcceptsLegacyDiminuendo(t *testing.T) {
	if got := gpifHairpin("Diminuendo"); got != HairpinDiminuendo {
		t.Errorf("legacy Diminuendo = %v, want HairpinDiminuendo", got)
	}
}
