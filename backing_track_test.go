// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"math"
	"path/filepath"
	"testing"
)

func TestGP7BackingTrackAndSyncPoints(t *testing.T) {
	audio := []byte("synthetic audio")
	data := buildGP7Archive(t, `<?xml version="1.0" encoding="utf-8"?>
<GPIF>
  <Score><Title>Synced score</Title></Score>
  <MasterTrack>
    <Automations>
      <Automation>
        <Type>SyncPoint</Type><Bar>0</Bar><Position>0</Position><Visible>true</Visible>
        <Value>
          <BarIndex>0</BarIndex><BarOccurrence>0</BarOccurrence>
          <ModifiedTempo>133.63637</ModifiedTempo><OriginalTempo>163</OriginalTempo>
          <FrameOffset>0</FrameOffset>
        </Value>
      </Automation>
      <Automation>
        <Type>SyncPoint</Type><Bar>1</Bar><Position>0</Position><Visible>true</Visible>
        <Value>
          <BarIndex>1</BarIndex><BarOccurrence>0</BarOccurrence>
          <ModifiedTempo>162.85582</ModifiedTempo><OriginalTempo>163</OriginalTempo>
          <FrameOffset>79200</FrameOffset>
        </Value>
      </Automation>
    </Automations>
  </MasterTrack>
  <BackingTrack>
    <Name>Audio Track</Name><Enabled>true</Enabled><Source>Local</Source>
    <AssetId>0</AssetId><FramePadding>-50700</FramePadding>
  </BackingTrack>
  <Assets>
    <Asset id="0">
      <OriginalFilePath>/source/song.mp3</OriginalFilePath>
      <OriginalFileSha1>asset-key</OriginalFileSha1>
      <EmbeddedFilePath>Content/Assets/asset-key.mp3</EmbeddedFilePath>
    </Asset>
  </Assets>
</GPIF>`, "Content/Assets/asset-key.mp3", audio)

	song, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if song.BackingTrack == nil {
		t.Fatal("backing track = nil")
	}
	if got := song.BackingTrack; got.Name != "Audio Track" || !got.Enabled || got.Source != "Local" || got.AssetID != "0" {
		t.Errorf("backing track = %#v", got)
	}
	if got := song.BackingTrack.FramePadding; got != -50700 {
		t.Errorf("frame padding = %d, want -50700", got)
	}
	if got := song.BackingTrack.EmbeddedFilePath; got != "Content/Assets/asset-key.mp3" {
		t.Errorf("embedded path = %q", got)
	}
	if !bytes.Equal(song.BackingTrack.AudioData, audio) {
		t.Errorf("audio data = %q, want %q", song.BackingTrack.AudioData, audio)
	}

	if len(song.SyncPoints) != 2 {
		t.Fatalf("sync points = %d, want 2", len(song.SyncPoints))
	}
	first := song.SyncPoints[0]
	if first.Bar != 0 || first.Position != 0 || first.BarOccurrence != 0 || first.FrameOffset != 0 {
		t.Errorf("first sync point = %#v", first)
	}
	if math.Abs(first.MediaTimeMS-1149.659863945578) > 1e-9 {
		t.Errorf("first media time = %.12f, want 1149.659863945578", first.MediaTimeMS)
	}
	if first.ModifiedTempo != 133.63637 || first.OriginalTempo != 163 || !first.Visible {
		t.Errorf("first sync tempo metadata = %#v", first)
	}

	second := song.SyncPoints[1]
	if second.Bar != 1 || second.FrameOffset != 79200 {
		t.Errorf("second sync point = %#v", second)
	}
	if math.Abs(second.MediaTimeMS-2945.5782312925166) > 1e-9 {
		t.Errorf("second media time = %.12f, want 2945.578231292517", second.MediaTimeMS)
	}
}

func TestGP8AudioTrackFixturePreservesSyncData(t *testing.T) {
	song, err := ParseFile(filepath.Join("testdata", "gp8", "canon-audio-track.gp"))
	if err != nil {
		t.Fatal(err)
	}
	if song.BackingTrack == nil {
		t.Fatal("backing track = nil")
	}
	if got := song.BackingTrack; got.FramePadding != -72900 || got.AssetID != "0" || !got.Enabled {
		t.Errorf("backing track = %#v", got)
	}
	if got := len(song.BackingTrack.AudioData); got != 936701 {
		t.Errorf("embedded audio bytes = %d, want 936701", got)
	}
	if len(song.SyncPoints) != 16 {
		t.Fatalf("sync points = %d, want 16", len(song.SyncPoints))
	}
	first := song.SyncPoints[0]
	if first.Bar != 0 || first.Position != 0 || first.FrameOffset != 0 {
		t.Errorf("first sync point = %#v", first)
	}
	if math.Abs(first.MediaTimeMS-1653.061224489796) > 1e-9 {
		t.Errorf("first media time = %.12f, want 1653.061224489796", first.MediaTimeMS)
	}
	second := song.SyncPoints[1]
	if second.Bar != 0 || second.Position != 0.5 || second.FrameOffset != 73620 {
		t.Errorf("second sync point = %#v", second)
	}
	if math.Abs(second.MediaTimeMS-3322.448979591837) > 1e-9 {
		t.Errorf("second media time = %.12f, want 3322.448979591837", second.MediaTimeMS)
	}
}

func buildGP7Archive(t *testing.T, gpif, assetPath string, assetData []byte) []byte {
	t.Helper()
	var data bytes.Buffer
	archive := zip.NewWriter(&data)

	gpifFile, err := archive.Create("Content/score.gpif")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gpifFile.Write([]byte(gpif)); err != nil {
		t.Fatal(err)
	}
	assetFile, err := archive.Create(assetPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := assetFile.Write(assetData); err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return data.Bytes()
}
