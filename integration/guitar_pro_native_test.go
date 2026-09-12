// SPDX-License-Identifier: MIT

package integration_test

import (
	"strings"
	"testing"

	gp "github.com/CaliLuke/go-guitar-pro"
)

func TestGuitarProNativeSoundNames(t *testing.T) {
	song, err := gp.ParseFile("../testdata/gp3/mix-table-events.gp3")
	if err != nil {
		t.Fatal(err)
	}
	song.Tracks[0].Sounds[1].Label = "Flute <&> Ω"
	data, err := gp.Export(song, gp.ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	raw := string(textContractGPIF(t, data))
	for _, text := range []string{"<Name><![CDATA[Legacy program 73 bank 0]]></Name>", "<Label><![CDATA[Flute <&> Ω]]></Label>", "<Name><![CDATA[Track 1]]></Name>"} {
		if !strings.Contains(raw, text) {
			t.Fatalf("Guitar Pro requires text as CDATA: missing %s", text)
		}
	}
	parsed, err := gp.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Tracks[0].Sounds[1].Name != song.Tracks[0].Sounds[1].Name || parsed.Tracks[0].Sounds[1].Label != song.Tracks[0].Sounds[1].Label {
		t.Fatal("sound identity changed")
	}
}
