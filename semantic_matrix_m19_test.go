// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestSemanticMatrixM19BackingAssetsAndSyncPoints(t *testing.T) {
	runSemanticMatrixM19BackingAssetsAndSyncPoints(newSemanticMatrixRun(t))
}

func runSemanticMatrixM19BackingAssetsAndSyncPoints(run *semanticMatrixRun) {
	t := run.t
	gpif := m19ValidGPIF()
	wire := decodeM19Wire(t, []byte(gpif))
	if wire.BackingTrack == nil {
		t.Fatal("wire backing track is nil")
	}
	run.Wire("gpifDocument.BackingTrack", wire.BackingTrack != nil, true)
	run.Wire("gpifBackingTrack.Name", wire.BackingTrack.Name, "Selected audio")
	run.Wire("gpifBackingTrack.Enabled", wire.BackingTrack.Enabled, true)
	run.Wire("gpifBackingTrack.Source", wire.BackingTrack.Source, "Local")
	run.Wire("gpifBackingTrack.AssetID", wire.BackingTrack.AssetID, "asset-b")
	run.Wire("gpifBackingTrack.FramePadding", wire.BackingTrack.FramePadding, "-22050")
	run.Wire("gpifDocument.Assets", len(wire.Assets.Items), 2)
	run.Wire("gpifAssets.Assets", len(wire.Assets.Items), 2)
	run.Wire("gpifAsset.ID", []string{wire.Assets.Items[0].ID, wire.Assets.Items[1].ID}, []string{"asset-a", "asset-b"})
	run.Wire("gpifAsset.OriginalFilePath", []string{wire.Assets.Items[0].OriginalFilePath, wire.Assets.Items[1].OriginalFilePath}, []string{"/unused/a.wav", "/selected/b.wav"})
	run.Wire("gpifAsset.OriginalFileSHA1", []string{wire.Assets.Items[0].OriginalFileSHA1, wire.Assets.Items[1].OriginalFileSHA1}, []string{"sha-a", "sha-b"})
	run.Wire("gpifAsset.EmbeddedFilePath", []string{wire.Assets.Items[0].EmbeddedFilePath, wire.Assets.Items[1].EmbeddedFilePath}, []string{"Content/Assets/a.bin", "Content/Assets/b.bin"})
	if len(wire.MasterTrack.Automations.Items) != 2 {
		t.Fatalf("wire sync points = %d, want 2", len(wire.MasterTrack.Automations.Items))
	}
	run.Wire("gpifMasterTrack.Automations", len(wire.MasterTrack.Automations.Items), 2)
	run.Wire("gpifAutomations.Automations", len(wire.MasterTrack.Automations.Items), 2)
	run.Wire("gpifAutomation.Type", []string{wire.MasterTrack.Automations.Items[0].Type, wire.MasterTrack.Automations.Items[1].Type}, []string{"SyncPoint", "SyncPoint"})
	run.Wire("gpifAutomation.Linear", []bool{wire.MasterTrack.Automations.Items[0].Linear, wire.MasterTrack.Automations.Items[1].Linear}, []bool{false, true})
	run.Wire("gpifAutomation.Bar", []int{wire.MasterTrack.Automations.Items[0].Bar, wire.MasterTrack.Automations.Items[1].Bar}, []int{0, 1})
	run.Wire("gpifAutomation.Position", []float64{wire.MasterTrack.Automations.Items[0].Position, wire.MasterTrack.Automations.Items[1].Position}, []float64{0, 1})
	run.Wire("gpifAutomation.Visible", []string{wire.MasterTrack.Automations.Items[0].Visible, wire.MasterTrack.Automations.Items[1].Visible}, []string{"true", "false"})
	run.Wire("gpifAutomation.Value", len(wire.MasterTrack.Automations.Items), 2)
	firstWireValue := wire.MasterTrack.Automations.Items[0].Value
	secondWireValue := wire.MasterTrack.Automations.Items[1].Value
	run.Wire("gpifAutomationValue.BarIndex", []string{firstWireValue.BarIndex, secondWireValue.BarIndex}, []string{"0", "1"})
	run.Wire("gpifAutomationValue.BarOccurrence", []string{firstWireValue.BarOccurrence, secondWireValue.BarOccurrence}, []string{"0", "2"})
	run.Wire("gpifAutomationValue.FrameOffset", []string{firstWireValue.FrameOffset, secondWireValue.FrameOffset}, []string{"0", "44100"})
	run.Wire("gpifAutomationValue.ModifiedTempo", []string{firstWireValue.ModifiedTempo, secondWireValue.ModifiedTempo}, []string{"120.5", "90.25"})
	run.Wire("gpifAutomationValue.OriginalTempo", []string{firstWireValue.OriginalTempo, secondWireValue.OriginalTempo}, []string{"100", "100.75"})

	selectedAudio := []byte{0x01, 0x02, 0x03, 0x04}
	data := m19Archive(t, gpif, map[string][]byte{
		"Content/Assets/a.bin": []byte("unused"),
		"Content/Assets/b.bin": selectedAudio,
	})
	result, err := ParseWithOptions(data, ParseOptions{Strict: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Song.BackingTrack == nil {
		t.Fatal("parsed backing track is nil")
	}
	backing := result.Song.BackingTrack
	run.Omitted("Song.BackingTrack", backing != nil, true)
	run.Preserved("BackingTrack.Name", backing.Name, "Selected audio")
	run.Preserved("BackingTrack.Source", backing.Source, "Local")
	run.Preserved("BackingTrack.AssetID", backing.AssetID, "asset-b")
	run.Preserved("BackingTrack.OriginalFilePath", backing.OriginalFilePath, "/selected/b.wav")
	run.Preserved("BackingTrack.OriginalFileSHA1", backing.OriginalFileSHA1, "sha-b")
	run.Preserved("BackingTrack.EmbeddedFilePath", backing.EmbeddedFilePath, "Content/Assets/b.bin")
	run.Preserved("BackingTrack.AudioData", backing.AudioData, selectedAudio)
	run.Preserved("BackingTrack.FramePadding", backing.FramePadding, int64(-22050))
	run.Preserved("BackingTrack.Enabled", backing.Enabled, true)
	if len(result.Song.SyncPoints) != 2 {
		t.Fatalf("sync points = %#v, want two", result.Song.SyncPoints)
	}
	run.Omitted("Song.SyncPoints", len(result.Song.SyncPoints), 2)
	zero, err := NewBarPosition(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	one, err := NewBarPosition(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("SyncPoint.Bar", []int{result.Song.SyncPoints[0].Bar, result.Song.SyncPoints[1].Bar}, []int{0, 1})
	run.Normalized("SyncPoint.Position", []float64{result.Song.SyncPoints[0].Position, result.Song.SyncPoints[1].Position}, []float64{0, 1})
	run.Derived("SyncPoint.BarPosition", []BarPosition{result.Song.SyncPoints[0].BarPosition, result.Song.SyncPoints[1].BarPosition}, []BarPosition{zero, one})
	run.Preserved("SyncPoint.BarOccurrence", []int{result.Song.SyncPoints[0].BarOccurrence, result.Song.SyncPoints[1].BarOccurrence}, []int{0, 2})
	run.Normalized("SyncPoint.FrameOffset", []int64{result.Song.SyncPoints[0].FrameOffset, result.Song.SyncPoints[1].FrameOffset}, []int64{0, 44100})
	run.Derived("SyncPoint.AudioFrame", []AudioFrame{result.Song.SyncPoints[0].AudioFrame, result.Song.SyncPoints[1].AudioFrame}, []AudioFrame{0, 44100})
	run.Derived("SyncPoint.MediaTimeMS", []float64{result.Song.SyncPoints[0].MediaTimeMS, result.Song.SyncPoints[1].MediaTimeMS}, []float64{500, 1500})
	run.Preserved("SyncPoint.ModifiedTempo", []float64{result.Song.SyncPoints[0].ModifiedTempo, result.Song.SyncPoints[1].ModifiedTempo}, []float64{120.5, 90.25})
	run.Preserved("SyncPoint.OriginalTempo", []float64{result.Song.SyncPoints[0].OriginalTempo, result.Song.SyncPoints[1].OriginalTempo}, []float64{100, 100.75})
	run.Preserved("SyncPoint.Linear", []bool{result.Song.SyncPoints[0].Linear, result.Song.SyncPoints[1].Linear}, []bool{false, true})
	run.Preserved("SyncPoint.Visible", []bool{result.Song.SyncPoints[0].Visible, result.Song.SyncPoints[1].Visible}, []bool{true, false})
	run.Dispatch("gpifReadSyncPoints:automation.Type", len(result.Song.SyncPoints), 2)

	t.Run("external metadata is retained without fetching", func(t *testing.T) {
		external := strings.ReplaceAll(m19ValidGPIF(), "<Source>Local</Source>", "<Source>External</Source>")
		external = strings.ReplaceAll(external, "<EmbeddedFilePath>Content/Assets/b.bin</EmbeddedFilePath>", "<EmbeddedFilePath></EmbeddedFilePath>")
		external = strings.ReplaceAll(external, "<OriginalFilePath>/selected/b.wav</OriginalFilePath>", "<OriginalFilePath>https://invalid.example/audio.wav</OriginalFilePath>")
		externalResult, err := ParseWithOptions(m19Archive(t, external, map[string][]byte{"Content/Assets/a.bin": []byte("unused")}), ParseOptions{Strict: true})
		if err != nil {
			t.Fatal(err)
		}
		got := externalResult.Song.BackingTrack
		if got == nil || got.Source != "External" || got.OriginalFilePath != "https://invalid.example/audio.wav" || got.OriginalFileSHA1 != "sha-b" || got.EmbeddedFilePath != "" || len(got.AudioData) != 0 {
			t.Fatalf("external backing track = %#v", got)
		}
	})
}

func TestSemanticMatrixM19BackingSourceDiagnostics(t *testing.T) {
	runSemanticMatrixM19BackingSourceDiagnostics(newSemanticMatrixRun(t))
}

func runSemanticMatrixM19BackingSourceDiagnostics(run *semanticMatrixRun) {
	t := run.t
	context := &parseContext{format: "GPIF"}
	gpifAuditMasterAutomations([]gpifAutomation{{Type: "SyncPoint", Value: gpifAutomationValue{FrameOffset: "0"}}}, context)
	run.Dispatch("gpifAuditMasterAutomations:automation.Type", len(context.diagnostics), 0)

	for _, test := range []struct {
		name    string
		gpif    string
		entries map[string][]byte
		code    string
	}{
		{
			name: "malformed frame padding",
			gpif: m19BackingGPIF("not-an-integer", "Content/Assets/audio.bin"),
			code: "GPIF.BackingTrack.FramePadding.Invalid",
		},
		{
			name: "missing embedded bytes",
			gpif: m19BackingGPIF("-22050", "Content/Assets/missing.bin"),
			code: "GPIF.BackingTrack.EmbeddedFile.Reference",
		},
		{
			name:    "empty embedded bytes",
			gpif:    m19BackingGPIF("-22050", "Content/Assets/empty.bin"),
			entries: map[string][]byte{"Content/Assets/empty.bin": {}},
			code:    "GPIF.BackingTrack.EmbeddedFile.Empty",
		},
		{
			name: "missing local embedded path",
			gpif: m19BackingGPIF("-22050", ""),
			code: "GPIF.BackingTrack.EmbeddedFile.Reference",
		},
		{
			name: "broken asset id",
			gpif: strings.Replace(m19BackingGPIF("0", "Content/Assets/audio.bin"), "<AssetId>asset-a</AssetId>", "<AssetId>missing</AssetId>", 1),
			code: "GPIF.BackingTrack.AssetId.Reference",
		},
		{
			name: "missing asset id",
			gpif: strings.Replace(m19BackingGPIF("0", "Content/Assets/audio.bin"), "<AssetId>asset-a</AssetId>", "<AssetId></AssetId>", 1),
			code: "GPIF.BackingTrack.AssetId.Reference",
		},
		{
			name: "duplicate asset id",
			gpif: strings.Replace(m19BackingGPIF("0", "Content/Assets/audio.bin"), "</Assets>", `<Asset id="asset-a"><EmbeddedFilePath>Content/Assets/other.bin</EmbeddedFilePath></Asset></Assets>`, 1),
			code: "GPIF.Asset.DuplicateID",
		},
		{
			name: "empty asset id",
			gpif: strings.Replace(m19BackingGPIF("0", "Content/Assets/audio.bin"), `id="asset-a"`, `id=""`, 1),
			code: "GPIF.Asset.EmptyID",
		},
		{
			name: "padding overflow",
			gpif: m19BackingGPIF("9223372036854775808", "Content/Assets/audio.bin"),
			code: "GPIF.BackingTrack.FramePadding.Invalid",
		},
		{
			name: "missing padding",
			gpif: strings.Replace(m19BackingGPIF("0", "Content/Assets/audio.bin"), "<FramePadding>0</FramePadding>", "", 1),
			code: "GPIF.BackingTrack.FramePadding.Invalid",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := ParseWithOptions(m19Archive(t, test.gpif, test.entries), ParseOptions{Strict: true})
			var strictErr *StrictParseError
			if !errors.As(err, &strictErr) {
				t.Fatalf("strict parse = %#v, %v, want StrictParseError", result, err)
			}
			diagnostic := m19FindParseCode(result, test.code)
			if diagnostic == nil || diagnostic.Kind != ParseDiagnosticInvalidData || diagnostic.Feature != "score-core" || diagnostic.SourcePath == "" {
				t.Fatalf("diagnostics = %#v, want %s", result, test.code)
			}
		})
	}

	disabled := strings.Replace(m19BackingGPIF("0", "Content/Assets/missing.bin"), "<Enabled>true</Enabled>", "<Enabled>false</Enabled>", 1)
	disabled = strings.Replace(disabled, "<AssetId>asset-a</AssetId>", "<AssetId></AssetId>", 1)
	disabledResult, disabledErr := ParseWithOptions(m19Archive(t, disabled, nil), ParseOptions{Strict: true})
	if disabledErr != nil {
		t.Fatalf("disabled backing track = %#v, %v, want no asset requirement", disabledResult, disabledErr)
	}
	for _, code := range []string{"GPIF.BackingTrack.AssetId.Reference", "GPIF.BackingTrack.EmbeddedFile.Reference"} {
		if diagnostic := m19FindParseCode(disabledResult, code); diagnostic != nil {
			t.Fatalf("disabled backing track diagnostic = %#v", diagnostic)
		}
	}

	validEntries := map[string][]byte{"Content/Assets/a.bin": []byte("unused"), "Content/Assets/b.bin": []byte("selected")}
	for _, test := range []struct {
		name string
		old  string
		new  string
		code string
	}{
		{name: "missing frame", old: "<FrameOffset>0</FrameOffset>", new: "<FrameOffset></FrameOffset>", code: "GPIF.MasterTrack.Automation.SyncPoint.Value.Invalid"},
		{name: "negative frame", old: "<FrameOffset>0</FrameOffset>", new: "<FrameOffset>-1</FrameOffset>", code: "GPIF.MasterTrack.Automation.SyncPoint.Value.Invalid"},
		{name: "frame overflow", old: "<FrameOffset>0</FrameOffset>", new: "<FrameOffset>9223372036854775808</FrameOffset>", code: "GPIF.MasterTrack.Automation.SyncPoint.Value.Invalid"},
		{name: "negative position", old: "<Position>0</Position>", new: "<Position>-0.1</Position>", code: "GPIF.MasterTrack.Automation.SyncPoint.Value.Invalid"},
		{name: "high position", old: "<Position>0</Position>", new: "<Position>1.1</Position>", code: "GPIF.MasterTrack.Automation.SyncPoint.Value.Invalid"},
		{name: "NaN position", old: "<Position>0</Position>", new: "<Position>NaN</Position>", code: "GPIF.MasterTrack.Automation.SyncPoint.Value.Invalid"},
		{name: "infinite position", old: "<Position>0</Position>", new: "<Position>+Inf</Position>", code: "GPIF.MasterTrack.Automation.SyncPoint.Value.Invalid"},
		{name: "negative occurrence", old: "<BarOccurrence>0</BarOccurrence>", new: "<BarOccurrence>-1</BarOccurrence>", code: "GPIF.MasterTrack.Automation.SyncPoint.Value.Invalid"},
		{name: "high bar", old: "<BarIndex>0</BarIndex>", new: "<BarIndex>2</BarIndex>", code: "score.sync-point.location"},
		{name: "negative modified tempo", old: "<ModifiedTempo>120.5</ModifiedTempo>", new: "<ModifiedTempo>-1</ModifiedTempo>", code: "score.sync-point.tempo"},
		{name: "NaN modified tempo", old: "<ModifiedTempo>120.5</ModifiedTempo>", new: "<ModifiedTempo>NaN</ModifiedTempo>", code: "GPIF.MasterTrack.Automation.SyncPoint.Value.Invalid"},
		{name: "negative original tempo", old: "<OriginalTempo>100</OriginalTempo>", new: "<OriginalTempo>-1</OriginalTempo>", code: "score.sync-point.tempo"},
		{name: "infinite original tempo", old: "<OriginalTempo>100</OriginalTempo>", new: "<OriginalTempo>+Inf</OriginalTempo>", code: "GPIF.MasterTrack.Automation.SyncPoint.Value.Invalid"},
	} {
		t.Run(test.name, func(t *testing.T) {
			gpif := m19ReplaceOnce(t, m19ValidGPIF(), test.old, test.new)
			result, err := ParseWithOptions(m19Archive(t, gpif, validEntries), ParseOptions{Strict: true})
			var strictErr *StrictParseError
			diagnostic := m19FindParseCode(result, test.code)
			if !errors.As(err, &strictErr) || diagnostic == nil || diagnostic.Kind != ParseDiagnosticInvalidData || diagnostic.Feature == "" {
				t.Fatalf("strict parse = %#v, %v, want %s", result, err, test.code)
			}
		})
	}
}

func TestSemanticMatrixM19ValidationAndExportPolicy(t *testing.T) {
	runSemanticMatrixM19ValidationAndExportPolicy(newSemanticMatrixRun(t))
}

func runSemanticMatrixM19ValidationAndExportPolicy(run *semanticMatrixRun) {
	t := run.t
	valid := m19ProgrammaticSong(t)
	report := PreflightExport(valid, ExportFormatGP8, ExportOptions{})
	if hasExportCode(report, "gp8.normalize.sync-point-frame-authority") || hasExportCode(report, "gp8.normalize.sync-point-position-authority") {
		t.Fatalf("agreeing sync points report = %#v", report.Entries)
	}
	backingEntry := m19ExportEntries(report, "gp8.omit.backing-track")
	if len(backingEntry) != 1 || backingEntry[0].Location != (ScoreLocation{}) {
		t.Fatalf("backing omission = %#v, want one score-scoped entry", backingEntry)
	}
	syncEntries := m19ExportEntries(report, "gp8.omit.sync-points")
	wantLocations := []ScoreLocation{{Measure: 0}, {Measure: 1}}
	gotLocations := make([]ScoreLocation, len(syncEntries))
	for index := range syncEntries {
		gotLocations[index] = syncEntries[index].Location
	}
	if !reflect.DeepEqual(gotLocations, wantLocations) {
		t.Fatalf("sync omission locations = %#v, want %#v", gotLocations, wantLocations)
	}
	data, strictReport, err := ExportWithReport(valid, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	var lossErr *ExportLossError
	if len(data) != 0 || !errors.As(err, &lossErr) || len(m19ExportEntries(strictReport, "gp8.omit.sync-points")) != 2 {
		t.Fatalf("strict omission export = %d bytes, %#v, %v", len(data), strictReport.Entries, err)
	}
	data, _, err = ExportWithReport(valid, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{
		RequirePreservation: true,
		AllowedCodes:        []string{"gp8.omit.backing-track", "gp8.omit.sync-points"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	outputWire := decodeM19Wire(t, readM19GPIF(t, data))
	hasSyncPoint := slices.ContainsFunc(outputWire.MasterTrack.Automations.Items, func(automation struct {
		Type     string  `xml:"Type"`
		Linear   bool    `xml:"Linear"`
		Bar      int     `xml:"Bar"`
		Position float64 `xml:"Position"`
		Visible  string  `xml:"Visible"`
		Value    struct {
			BarIndex      string `xml:"BarIndex"`
			BarOccurrence string `xml:"BarOccurrence"`
			ModifiedTempo string `xml:"ModifiedTempo"`
			OriginalTempo string `xml:"OriginalTempo"`
			FrameOffset   string `xml:"FrameOffset"`
		} `xml:"Value"`
	}) bool {
		return automation.Type == "SyncPoint"
	})
	run.Wire("gpifDocument.BackingTrack", outputWire.BackingTrack != nil, false)
	run.Wire("gpifDocument.Assets", len(outputWire.Assets.Items), 0)
	run.Wire("gpifAutomation.Type", hasSyncPoint, false)
	if outputWire.BackingTrack != nil || len(outputWire.Assets.Items) != 0 || hasSyncPoint {
		t.Fatalf("independent GPIF retained omitted backing data: %#v", outputWire)
	}

	conflict := m19ProgrammaticSong(t)
	conflict.SyncPoints[1].FrameOffset = 22050
	conflict.SyncPoints[1].Position = 0.25
	run.Field("SyncPoint.FrameOffset", conflict.SyncPoints[1].FrameOffset, int64(22050))
	run.Field("SyncPoint.AudioFrame", conflict.SyncPoints[1].AudioFrame, AudioFrame(44100))
	run.Field("SyncPoint.Position", conflict.SyncPoints[1].Position, 0.25)
	run.Field("SyncPoint.BarPosition", conflict.SyncPoints[1].BarPosition, valid.SyncPoints[1].BarPosition)
	conflictReport := PreflightExport(conflict, ExportFormatGP8, ExportOptions{})
	for _, code := range []string{"gp8.normalize.sync-point-frame-authority", "gp8.normalize.sync-point-position-authority"} {
		entries := m19ExportEntries(conflictReport, code)
		if len(entries) != 1 || entries[0].Location.Measure != 1 {
			t.Errorf("%s entries = %#v, want point location", code, entries)
		}
	}
	data, _, err = ExportWithReport(conflict, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{
		RequirePreservation: true,
		AllowedCodes:        []string{"gp8.omit.backing-track", "gp8.omit.sync-points"},
	}})
	if len(data) != 0 || !errors.As(err, &lossErr) {
		t.Fatalf("strict compatibility export = %d bytes, %v, want refusal", len(data), err)
	}

	for _, test := range []struct {
		name   string
		mutate func(*Song, *SyncPoint)
		code   string
	}{
		{name: "negative bar", mutate: func(_ *Song, point *SyncPoint) { point.Bar = -1 }, code: "score.sync-point.location"},
		{name: "high bar", mutate: func(song *Song, point *SyncPoint) { point.Bar = len(song.MeasureHeaders) }, code: "score.sync-point.location"},
		{name: "negative position", mutate: func(_ *Song, point *SyncPoint) { point.Position = -0.1 }, code: "score.sync-point.location"},
		{name: "high position", mutate: func(_ *Song, point *SyncPoint) { point.Position = 1.1 }, code: "score.sync-point.location"},
		{name: "NaN position", mutate: func(_ *Song, point *SyncPoint) { point.Position = math.NaN() }, code: "score.sync-point.location"},
		{name: "infinite position", mutate: func(_ *Song, point *SyncPoint) { point.Position = math.Inf(1) }, code: "score.sync-point.location"},
		{name: "negative occurrence", mutate: func(_ *Song, point *SyncPoint) { point.BarOccurrence = -1 }, code: "score.sync-point.location"},
		{name: "negative frame offset", mutate: func(_ *Song, point *SyncPoint) { point.FrameOffset = -1 }, code: "score.sync-point.frame"},
		{name: "negative audio frame", mutate: func(_ *Song, point *SyncPoint) { point.AudioFrame = AudioFrame(-1) }, code: "score.sync-point.frame"},
		{name: "NaN media time", mutate: func(_ *Song, point *SyncPoint) { point.MediaTimeMS = math.NaN() }, code: "score.sync-point.media-time"},
		{name: "infinite media time", mutate: func(_ *Song, point *SyncPoint) { point.MediaTimeMS = math.Inf(1) }, code: "score.sync-point.media-time"},
		{name: "stale media time", mutate: func(_ *Song, point *SyncPoint) { point.MediaTimeMS++ }, code: "score.sync-point.media-time"},
		{name: "negative modified tempo", mutate: func(_ *Song, point *SyncPoint) { point.ModifiedTempo = -1 }, code: "score.sync-point.tempo"},
		{name: "NaN modified tempo", mutate: func(_ *Song, point *SyncPoint) { point.ModifiedTempo = math.NaN() }, code: "score.sync-point.tempo"},
		{name: "negative original tempo", mutate: func(_ *Song, point *SyncPoint) { point.OriginalTempo = -1 }, code: "score.sync-point.tempo"},
		{name: "infinite original tempo", mutate: func(_ *Song, point *SyncPoint) { point.OriginalTempo = math.Inf(1) }, code: "score.sync-point.tempo"},
	} {
		t.Run(test.name, func(t *testing.T) {
			song := m19ProgrammaticSong(t)
			test.mutate(song, &song.SyncPoints[0])
			diagnostics := ValidateSong(song)
			if !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == test.code }) {
				t.Errorf("diagnostics = %#v, want %s", diagnostics, test.code)
			}
			preflight := PreflightExport(song, ExportFormatGP8, ExportOptions{})
			if !hasExportCode(preflight, "gp8.reject."+test.code) {
				t.Errorf("preflight = %#v, want %s rejection", preflight.Entries, test.code)
			}
		})
	}
}

func m19BackingGPIF(framePadding, embeddedPath string) string {
	return `<?xml version="1.0" encoding="utf-8"?>
<GPIF>
  <BackingTrack><Name>Audio</Name><Enabled>true</Enabled><Source>Local</Source><AssetId>asset-a</AssetId><FramePadding>` + framePadding + `</FramePadding></BackingTrack>
  <Assets><Asset id="asset-a"><OriginalFilePath>/audio/source.wav</OriginalFilePath><OriginalFileSha1>sha-a</OriginalFileSha1><EmbeddedFilePath>` + embeddedPath + `</EmbeddedFilePath></Asset></Assets>
</GPIF>`
}

func m19ValidGPIF() string {
	return `<?xml version="1.0" encoding="utf-8"?>
<GPIF>
  <MasterTrack><Automations>
    <Automation><Type>SyncPoint</Type><Linear>false</Linear><Bar>0</Bar><Position>0</Position><Visible>true</Visible><Value><BarIndex>0</BarIndex><BarOccurrence>0</BarOccurrence><ModifiedTempo>120.5</ModifiedTempo><OriginalTempo>100</OriginalTempo><FrameOffset>0</FrameOffset></Value></Automation>
    <Automation><Type>SyncPoint</Type><Linear>true</Linear><Bar>1</Bar><Position>1</Position><Visible>false</Visible><Value><BarIndex>1</BarIndex><BarOccurrence>2</BarOccurrence><ModifiedTempo>90.25</ModifiedTempo><OriginalTempo>100.75</OriginalTempo><FrameOffset>44100</FrameOffset></Value></Automation>
  </Automations></MasterTrack>
  <BackingTrack><Name>Selected audio</Name><Enabled>true</Enabled><Source>Local</Source><AssetId>asset-b</AssetId><FramePadding>-22050</FramePadding></BackingTrack>
  <Assets>
    <Asset id="asset-a"><OriginalFilePath>/unused/a.wav</OriginalFilePath><OriginalFileSha1>sha-a</OriginalFileSha1><EmbeddedFilePath>Content/Assets/a.bin</EmbeddedFilePath></Asset>
    <Asset id="asset-b"><OriginalFilePath>/selected/b.wav</OriginalFilePath><OriginalFileSha1>sha-b</OriginalFileSha1><EmbeddedFilePath>Content/Assets/b.bin</EmbeddedFilePath></Asset>
  </Assets>
  <MasterBars><MasterBar><Time>4/4</Time></MasterBar><MasterBar><Time>4/4</Time></MasterBar></MasterBars>
</GPIF>`
}

type m19WireDocument struct {
	BackingTrack *struct {
		Name         string `xml:"Name"`
		Enabled      bool   `xml:"Enabled"`
		Source       string `xml:"Source"`
		AssetID      string `xml:"AssetId"`
		FramePadding string `xml:"FramePadding"`
	} `xml:"BackingTrack"`
	Assets struct {
		Items []struct {
			ID               string `xml:"id,attr"`
			OriginalFilePath string `xml:"OriginalFilePath"`
			OriginalFileSHA1 string `xml:"OriginalFileSha1"`
			EmbeddedFilePath string `xml:"EmbeddedFilePath"`
		} `xml:"Asset"`
	} `xml:"Assets"`
	MasterTrack struct {
		Automations struct {
			Items []struct {
				Type     string  `xml:"Type"`
				Linear   bool    `xml:"Linear"`
				Bar      int     `xml:"Bar"`
				Position float64 `xml:"Position"`
				Visible  string  `xml:"Visible"`
				Value    struct {
					BarIndex      string `xml:"BarIndex"`
					BarOccurrence string `xml:"BarOccurrence"`
					ModifiedTempo string `xml:"ModifiedTempo"`
					OriginalTempo string `xml:"OriginalTempo"`
					FrameOffset   string `xml:"FrameOffset"`
				} `xml:"Value"`
			} `xml:"Automation"`
		} `xml:"Automations"`
	} `xml:"MasterTrack"`
}

func decodeM19Wire(t *testing.T, data []byte) m19WireDocument {
	t.Helper()
	var result m19WireDocument
	if err := xml.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func m19ProgrammaticSong(t *testing.T) *Song {
	t.Helper()
	zero, err := NewBarPosition(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	one, err := NewBarPosition(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	song := syntheticGP8Song()
	song.BackingTrack = &BackingTrack{
		Name: "Selected audio", Source: "Local", AssetID: "asset-b",
		OriginalFilePath: "/selected/b.wav", OriginalFileSHA1: "sha-b",
		EmbeddedFilePath: "Content/Assets/b.bin", AudioData: []byte{1, 2, 3, 4},
		FramePadding: -22050, Enabled: true,
	}
	song.SyncPoints = []SyncPoint{
		{Bar: 0, Position: 0, BarPosition: zero, BarOccurrence: 0, FrameOffset: 0, AudioFrame: 0, MediaTimeMS: 500, ModifiedTempo: 120.5, OriginalTempo: 100, Linear: false, Visible: true},
		{Bar: 1, Position: 1, BarPosition: one, BarOccurrence: 2, FrameOffset: 44100, AudioFrame: 44100, MediaTimeMS: 1500, ModifiedTempo: 90.25, OriginalTempo: 100.75, Linear: true, Visible: false},
	}
	return song
}

func m19ExportEntries(report ExportReport, code string) []ExportReportEntry {
	var entries []ExportReportEntry
	for _, entry := range report.Entries {
		if entry.Code == code {
			entries = append(entries, entry)
		}
	}
	return entries
}

func readM19GPIF(t *testing.T, data []byte) []byte {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range archive.File {
		if file.Name != "Content/score.gpif" {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		contents, err := io.ReadAll(reader)
		closeErr := reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		return contents
	}
	t.Fatal("GP8 archive has no score.gpif")
	return nil
}

func m19ReplaceOnce(t *testing.T, source, old, replacement string) string {
	t.Helper()
	result := strings.Replace(source, old, replacement, 1)
	if result == source {
		t.Fatalf("source has no %q", old)
	}
	return result
}

func m19Archive(t *testing.T, gpif string, entries map[string][]byte) []byte {
	t.Helper()
	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	writer, err := archive.Create("Content/score.gpif")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte(gpif)); err != nil {
		t.Fatal(err)
	}
	for name, data := range entries {
		writer, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := writer.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return output.Bytes()
}

func m19FindParseCode(result *ParseResult, code string) *ParseDiagnostic {
	if result == nil {
		return nil
	}
	for index := range result.Diagnostics {
		if result.Diagnostics[index].Code == code {
			return &result.Diagnostics[index]
		}
	}
	return nil
}
