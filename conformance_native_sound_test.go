// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestConformanceNativeSoundText(t *testing.T) {
	runConformanceNativeSoundText(newConformanceRun(t))
}

func runConformanceNativeSoundText(run *conformanceRun) {
	t := run.t
	song := parseTestFixture(t, "testdata/gp3/mix-table-events.gp3")
	song.Tracks[0].Sounds[1].Label = "Flute <&> Ω"
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	doc := conformanceWireDocument(t, data)
	run.Wire("gpifSound.Name", doc.Tracks.Tracks[0].Sounds.Sounds[1].Name, "Legacy program 73 bank 0")
	run.Wire("gpifSound.Label", doc.Tracks.Tracks[0].Sounds.Sounds[1].Label, "Flute <&> Ω")
	nativeData, err := os.ReadFile("conformance/capabilities/evidence/guitar-pro-native/labeled-sound-native.gp")
	if err != nil {
		t.Fatal(err)
	}
	native := conformanceWireDocument(t, nativeData)
	run.Field("TrackSound.Name", native.Tracks.Tracks[0].Sounds.Sounds[1].Name, song.Tracks[0].Sounds[1].Name)
	run.Field("TrackSound.Label", native.Tracks.Tracks[0].Sounds.Sounds[1].Label, song.Tracks[0].Sounds[1].Label)
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	raw := string(readZipMember(t, archive, "Content/score.gpif"))
	if !strings.Contains(raw, "<Name><![CDATA[Legacy program 73 bank 0]]></Name>") || !strings.Contains(raw, "<Label><![CDATA[Flute <&> Ω]]></Label>") {
		t.Fatal("Guitar Pro sound text must use CDATA")
	}
}

func TestGuitarProNativeReceiptIntegrity(t *testing.T) {
	data, err := os.ReadFile("conformance/capabilities/evidence/guitar-pro-native.json")
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Artifacts []struct {
			Path   string `json:"path"`
			SHA256 string `json:"sha256"`
		} `json:"artifacts"`
	}
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatal(err)
	}
	for _, artifact := range receipt.Artifacts {
		data, err := os.ReadFile(artifact.Path)
		if err != nil {
			t.Fatal(err)
		}
		digest := sha256.Sum256(data)
		if hex.EncodeToString(digest[:]) != artifact.SHA256 {
			t.Errorf("native evidence changed: %s", artifact.Path)
		}
	}
}
