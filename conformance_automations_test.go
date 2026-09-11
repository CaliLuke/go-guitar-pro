// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

func TestConformanceAutomationSemantics(t *testing.T) {
	runConformanceAutomationSemantics(newConformanceRun(t))
	conformanceIndependentClaim(t, "field:TempoAutomation.Linear", claimSite("automation-detail", "model", "M17-AUTOMATION-SEMANTICS", "linear and step changes with the same value and position"))
	conformanceIndependentClaim(t, "field:Track.SoundAutomations", claimSite("sound-automation", "model", "M17-AUTOMATION-SEMANTICS", "similar sound names with distinct paths and roles"))
	conformanceIndependentClaim(t, "field:Song.VolumeAutomations", claimSite("volume-automation", "model", "M17-AUTOMATION-SEMANTICS", "ordered tempo, sound, and volume changes"))
}

func runConformanceAutomationSemantics(run *conformanceRun) {
	t := run.t
	song := semanticValidPitchedGP8Song(t)
	song.Tracks[0].Settings.Notation = true
	song.Tempo = 111
	song.InitialTempo = KnownSourceValue(BPM(111))
	song.TempoAutomations = []TempoAutomation{
		{Bar: 0, Position: 0, Tempo: 111, Text: "Bright"},
		{Bar: 0, Position: 0.5, Tempo: 145.5, Text: "step tempo"},
		{Bar: 0, Position: 0.5, Tempo: 145.5, Linear: true, Text: "linear tempo", Hidden: true},
	}
	track := &song.Tracks[0]
	track.Sounds = []TrackSound{
		{Name: "Lead", Label: "Lead A", Path: "factory/a", Role: "main", Program: 27},
		{Name: "Lead", Label: "Lead B", Path: "factory/b", Role: "solo", Program: 81},
	}
	track.SoundAutomations = []SoundAutomation{
		{Bar: 0, Position: 0, Sound: 0},
		{Bar: 0, Position: 0.5, Sound: 1, Text: "step sound"},
		{Bar: 0, Position: 0.5, Sound: 1, Linear: true, Text: "linear sound", Hidden: true},
	}
	song.VolumeAutomations = []VolumeAutomation{
		{Track: 0, Bar: 0, Position: 0, Value: 0.25, Linear: false},
		{Track: 0, Bar: 1, Position: 1, Value: 0.875, Linear: true},
	}

	wah := &WahEffect{Value: -23, Display: true}
	change := &MixTableChange{
		Tremolo:    &MixTableItem{Value: 11, Duration: 1, AllTracks: true},
		Volume:     &MixTableItem{Value: 12, Duration: 2, AllTracks: false},
		Balance:    &MixTableItem{Value: 5, Duration: 3, AllTracks: true},
		Chorus:     &MixTableItem{Value: 44, Duration: 4, AllTracks: false},
		Reverb:     &MixTableItem{Value: 55, Duration: 5, AllTracks: true},
		Instrument: &MixTableItem{Value: 66, Duration: 0, AllTracks: false},
		Phaser:     &MixTableItem{Value: 77, Duration: 6, AllTracks: true},
		Tempo:      &MixTableItem{Value: 303, Duration: 7, AllTracks: false},
		Wah:        wah,
		TempoName:  "M17 tempo",
		Rse:        RseInstrument{Instrument: 7, Unknown: 8, SoundBank: 9, EffectNumber: 10},
		HideTempo:  true,
		UseRse:     true,
	}
	track.Measures[0].Voices[0].Beats[0].Effect.MixTableChange = change
	track.Staves[0].Measures = track.Measures

	run.Field("Song.TempoAutomations", song.TempoAutomations, []TempoAutomation{{Bar: 0, Position: 0, Tempo: 111, Text: "Bright"}, {Bar: 0, Position: 0.5, Tempo: 145.5, Text: "step tempo"}, {Bar: 0, Position: 0.5, Tempo: 145.5, Linear: true, Text: "linear tempo", Hidden: true}})
	for index, automation := range song.TempoAutomations {
		run.Field("TempoAutomation.Bar", automation.Bar, 0)
		run.Field("TempoAutomation.Position", automation.Position, []float64{0, 0.5, 0.5}[index])
		run.Field("TempoAutomation.Tempo", automation.Tempo, []float64{111, 145.5, 145.5}[index])
		assertAutomationDetailFields(run, "TempoAutomation", automation.Linear, automation.Text, automation.Hidden, index, []string{"Bright", "step tempo", "linear tempo"})
	}
	run.ClaimPrimary(claimSite("automation-detail", "model", "M17-AUTOMATION-SEMANTICS", "linear and step changes with the same value and position")).Preserved("TempoAutomation.Linear", []bool{song.TempoAutomations[0].Linear, song.TempoAutomations[1].Linear, song.TempoAutomations[2].Linear}, []bool{false, false, true})
	run.Field("Track.Sounds", track.Sounds, []TrackSound{{Name: "Lead", Label: "Lead A", Path: "factory/a", Role: "main", Program: 27}, {Name: "Lead", Label: "Lead B", Path: "factory/b", Role: "solo", Program: 81}})
	run.Field("Track.SoundAutomations", track.SoundAutomations, []SoundAutomation{{Bar: 0, Position: 0, Sound: 0}, {Bar: 0, Position: 0.5, Sound: 1, Text: "step sound"}, {Bar: 0, Position: 0.5, Sound: 1, Linear: true, Text: "linear sound", Hidden: true}})
	for index, automation := range track.SoundAutomations {
		run.Field("SoundAutomation.Bar", automation.Bar, 0)
		run.Field("SoundAutomation.Position", automation.Position, []float64{0, 0.5, 0.5}[index])
		run.Field("SoundAutomation.Sound", automation.Sound, []int{0, 1, 1}[index])
		assertAutomationDetailFields(run, "SoundAutomation", automation.Linear, automation.Text, automation.Hidden, index, []string{"", "step sound", "linear sound"})
	}
	run.ClaimPrimary(claimSite("volume-automation", "model", "M17-AUTOMATION-SEMANTICS", "ordered tempo, sound, and volume changes")).Preserved("Song.VolumeAutomations", song.VolumeAutomations, []VolumeAutomation{{Track: 0, Bar: 0, Position: 0, Value: 0.25}, {Track: 0, Bar: 1, Position: 1, Value: 0.875, Linear: true}})
	for index, automation := range song.VolumeAutomations {
		run.Preserved("VolumeAutomation.Track", automation.Track, 0)
		run.Preserved("VolumeAutomation.Bar", automation.Bar, index)
		run.Preserved("VolumeAutomation.Position", automation.Position, []float64{0, 1}[index])
		run.Preserved("VolumeAutomation.Value", automation.Value, []float64{0.25, 0.875}[index])
		run.Preserved("VolumeAutomation.Linear", automation.Linear, index == 1)
	}
	assertAutomationMixTableFields(run, track.Measures[0].Voices[0].Beats[0].Effect.MixTableChange, change)

	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	for _, code := range []string{"gp8.omit.volume-automation-consumer", "gp8.omit.mix-table-chorus", "gp8.omit.sound-automation-visibility"} {
		if !hasExportCode(report, code) {
			t.Errorf("report = %#v, want %s", report.Entries, code)
		}
	}
	data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: reportCodes(report)}})
	if err != nil {
		t.Fatal(err)
	}
	wire := extractAutomationWireDocument(t, data)
	run.Wire("gpifAutomations.Automations", len(wire.masterAutomations), 3)
	run.Wire("gpifAutomation.Type", conformanceAutomationStrings(wire.masterAutomations, func(item conformanceAutomationWireAutomation) string { return item.Type }), []string{"Tempo", "Tempo", "Tempo"})
	run.Wire("gpifAutomation.Bar", conformanceAutomationInts(wire.masterAutomations, func(item conformanceAutomationWireAutomation) int { return item.Bar }), []int{0, 0, 0})
	run.Wire("gpifAutomation.Position", conformanceAutomationFloats(wire.masterAutomations, func(item conformanceAutomationWireAutomation) float64 { return item.Position }), []float64{0, 0.5, 0.5})
	run.Wire("gpifAutomation.Value", conformanceAutomationStrings(wire.masterAutomations, func(item conformanceAutomationWireAutomation) string { return item.Value }), []string{"111 2", "145.5 2", "145.5 2"})
	run.Wire("gpifAutomation.Linear", conformanceAutomationBools(wire.masterAutomations, func(item conformanceAutomationWireAutomation) bool { return item.Linear }), []bool{false, false, true})
	run.Wire("gpifAutomation.Visible", conformanceAutomationStrings(wire.masterAutomations, func(item conformanceAutomationWireAutomation) string { return item.Visible }), []string{"true", "true", "false"})
	run.Wire("gpifAutomation.Text", conformanceAutomationStrings(wire.masterAutomations, func(item conformanceAutomationWireAutomation) string { return item.Text }), []string{"Bright", "step tempo", "linear tempo"})
	run.Wire("gpifSounds.Sounds", len(wire.sounds), 2)
	run.Wire("gpifSound.Name", []string{wire.sounds[0].Name, wire.sounds[1].Name}, []string{"Lead", "Lead"})
	run.Wire("gpifSound.Label", []string{wire.sounds[0].Label, wire.sounds[1].Label}, []string{"Lead A", "Lead B"})
	run.Wire("gpifSound.Path", []string{wire.sounds[0].Path, wire.sounds[1].Path}, []string{"factory/a", "factory/b"})
	run.Wire("gpifSound.Role", []string{wire.sounds[0].Role, wire.sounds[1].Role}, []string{"main", "solo"})
	run.Wire("gpifSound.Program", []int{wire.sounds[0].Program, wire.sounds[1].Program}, []int{27, 81})
	run.Wire("gpifTrack.Automations", len(wire.trackAutomations), 3)
	run.Wire("gpifTrack.Sounds", len(wire.sounds), 2)
	run.Wire("gpifAutomation.Value", conformanceAutomationStrings(wire.trackAutomations, func(item conformanceAutomationWireAutomation) string { return item.Value }), []string{"factory/a;Lead;main", "factory/b;Lead;solo", "factory/b;Lead;solo"})
	run.Wire("gpifAutomation.Position", conformanceAutomationFloats(wire.trackAutomations, func(item conformanceAutomationWireAutomation) float64 { return item.Position }), []float64{0, 0.5, 0.5})
	run.Wire("gpifAutomation.Linear", conformanceAutomationBools(wire.trackAutomations, func(item conformanceAutomationWireAutomation) bool { return item.Linear }), []bool{false, false, true})
	run.Wire("gpifAutomation.Visible", conformanceAutomationStrings(wire.trackAutomations, func(item conformanceAutomationWireAutomation) string { return item.Visible }), []string{"true", "true", "false"})
	run.Wire("gpifAutomation.Text", conformanceAutomationStrings(wire.trackAutomations, func(item conformanceAutomationWireAutomation) string { return item.Text }), []string{"", "step sound", "linear sound"})

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Field("Song.TempoAutomations", roundTrip.TempoAutomations, song.TempoAutomations)
	run.ClaimPrimary(claimSite("sound-automation", "model", "M17-AUTOMATION-SEMANTICS", "similar sound names with distinct paths and roles")).Field("Track.SoundAutomations", roundTrip.Tracks[0].SoundAutomations, track.SoundAutomations)
	run.Field("Track.Sounds", roundTrip.Tracks[0].Sounds, track.Sounds)
	if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
		got := readAlphaTabAutomationFacts(t, writeConformanceFixture(t, data))
		want := map[string]any{
			"tempo": []any{
				map[string]any{"bar": float64(0), "position": float64(0), "type": "tempo", "value": float64(111), "linear": false, "text": "Bright", "visible": true},
				map[string]any{"bar": float64(0), "position": 0.5, "type": "tempo", "value": 145.5, "linear": false, "text": "step tempo", "visible": true},
				map[string]any{"bar": float64(0), "position": 0.5, "type": "tempo", "value": 145.5, "linear": true, "text": "linear tempo", "visible": false},
			},
			"sound": []any{
				map[string]any{"track": float64(0), "bar": float64(0), "position": float64(0), "type": "instrument", "value": float64(27), "linear": false, "text": "", "visible": true},
				map[string]any{"track": float64(0), "bar": float64(0), "position": 0.5, "type": "instrument", "value": float64(81), "linear": false, "text": "step sound", "visible": true},
				// AlphaTab does not copy hidden visibility from the Sound record to its instrument automation.
				map[string]any{"track": float64(0), "bar": float64(0), "position": 0.5, "type": "instrument", "value": float64(81), "linear": true, "text": "linear sound", "visible": true},
			},
		}
		if differences := semanticDifferences(got, want); len(differences) != 0 {
			t.Fatalf("AlphaTab automation facts differ: %#v", differences)
		}
	}

	reordered := semanticValidPitchedGP8Song(t)
	reordered.Tracks[0].Settings.Notation = true
	reordered.Tracks[0].Sounds = []TrackSound{track.Sounds[1], track.Sounds[0]}
	reordered.Tracks[0].SoundAutomations = []SoundAutomation{
		{Bar: 0, Position: 0, Sound: 1},
		{Bar: 0, Position: 0.5, Sound: 0, Text: "step sound"},
		{Bar: 0, Position: 0.5, Sound: 0, Linear: true, Text: "linear sound", Hidden: true},
	}
	reorderedData, err := Export(reordered, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	reorderedWire := extractAutomationWireDocument(t, reorderedData)
	if got, want := conformanceAutomationStrings(reorderedWire.trackAutomations, func(item conformanceAutomationWireAutomation) string { return item.Value }), conformanceAutomationStrings(wire.trackAutomations, func(item conformanceAutomationWireAutomation) string { return item.Value }); !slices.Equal(got, want) {
		t.Errorf("sound references after definition reorder = %#v, want %#v", got, want)
	}

	authority := semanticValidPitchedGP8Song(t)
	authority.TempoName = "edited opening"
	authority.TempoAutomations = []TempoAutomation{
		{Bar: 1, Position: 0.25, Tempo: 99, Text: "later"},
		{Bar: 0, Position: 0, Tempo: 120, Text: "source opening"},
	}
	authorityData, err := Export(authority, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	authorityWire := extractAutomationWireDocument(t, authorityData)
	run.Wire("gpifAutomation.Text", conformanceAutomationStrings(authorityWire.masterAutomations, func(item conformanceAutomationWireAutomation) string { return item.Text }), []string{"later", "edited opening"})
	authority.TempoName = ""
	authorityData, err = Export(authority, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	authorityWire = extractAutomationWireDocument(t, authorityData)
	run.Field("TempoAutomation.Text", authority.TempoAutomations[1].Text, "source opening")
	run.Wire("gpifAutomation.Text", conformanceAutomationStrings(authorityWire.masterAutomations, func(item conformanceAutomationWireAutomation) string { return item.Text }), []string{"later", "source opening"})
}

func assertAutomationDetailFields(run *conformanceRun, prefix string, linear bool, text string, hidden bool, index int, texts []string) {
	run.Preserved(prefix+".Linear", linear, index == 2)
	run.Preserved(prefix+".Text", text, texts[index])
	if prefix == "SoundAutomation" {
		run.Omitted(prefix+".Hidden", hidden, index == 2)
	} else {
		run.Preserved(prefix+".Hidden", hidden, index == 2)
	}
}

func assertAutomationMixTableFields(run *conformanceRun, got, want *MixTableChange) {
	run.Omitted("BeatEffects.MixTableChange", got, want)
	run.Preserved("MixTableChange.Tremolo", got.Tremolo, want.Tremolo)
	run.Preserved("MixTableChange.Volume", got.Volume, want.Volume)
	run.Preserved("MixTableChange.Balance", got.Balance, want.Balance)
	run.Preserved("MixTableChange.Chorus", got.Chorus, want.Chorus)
	run.Preserved("MixTableChange.Reverb", got.Reverb, want.Reverb)
	run.Preserved("MixTableChange.Instrument", got.Instrument, want.Instrument)
	run.Preserved("MixTableChange.Phaser", got.Phaser, want.Phaser)
	run.Preserved("MixTableChange.Tempo", got.Tempo, want.Tempo)
	run.Preserved("MixTableChange.Wah", got.Wah, want.Wah)
	run.Preserved("MixTableChange.TempoName", got.TempoName, want.TempoName)
	run.Preserved("MixTableChange.Rse", got.Rse, want.Rse)
	run.Preserved("MixTableChange.HideTempo", got.HideTempo, want.HideTempo)
	run.Preserved("MixTableChange.UseRse", got.UseRse, want.UseRse)
	items := []*MixTableItem{got.Tremolo, got.Volume, got.Balance, got.Chorus, got.Reverb, got.Instrument, got.Phaser, got.Tempo}
	wantItems := []*MixTableItem{want.Tremolo, want.Volume, want.Balance, want.Chorus, want.Reverb, want.Instrument, want.Phaser, want.Tempo}
	for index, item := range items {
		if item == nil || wantItems[index] == nil {
			continue
		}
		run.Preserved("MixTableItem.Value", item.Value, wantItems[index].Value)
		run.Preserved("MixTableItem.Duration", item.Duration, wantItems[index].Duration)
		run.Preserved("MixTableItem.AllTracks", item.AllTracks, wantItems[index].AllTracks)
	}
	if got.Wah != nil && want.Wah != nil {
		run.Preserved("WahEffect.Value", got.Wah.Value, want.Wah.Value)
		run.Preserved("WahEffect.Display", got.Wah.Display, want.Wah.Display)
	}
}

func TestConformanceSourceDispatchAndDiagnostics(t *testing.T) {
	runConformanceSourceDispatchAndDiagnostics(newConformanceRun(t))
	conformanceIndependentClaim(t, "dispatch:parseGPIFWithContext:automation.Type", claimSite("automation-detail", "import", "M17-SOURCE-DISPATCH", "linear and step changes with the same value and position"))
	conformanceIndependentClaim(t, "dispatch:gpifAuditTrackAutomations:automation.Type", claimSite("sound-automation", "import", "M17-SOURCE-DISPATCH", "similar sound names with distinct paths and roles"))
	conformanceIndependentClaim(t, "dispatch:gpifReadVolumeAutomations:automation.Type", claimSite("volume-automation", "import", "M17-SOURCE-DISPATCH", "ordered tempo, sound, and volume changes"))
}

func runConformanceSourceDispatchAndDiagnostics(run *conformanceRun) {
	t := run.t
	source := conformanceAutomationGPIFSource()
	result, err := ParseWithOptions(conformanceGPIFArchive(t, source), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{
		"GPIF.MasterTrack.Automation.Tempo.Invalid",
		"GPIF.MasterTrack.Automation.SyncPoint.Value.Invalid",
		"GPIF.MasterTrack.Automation.Type.Unknown",
		"GPIF.Track.Automation.Sound.Reference",
		"GPIF.Track.Automation.Type.Unknown",
		"GPIF.ChannelStrip.Automation.Volume.Value.Invalid",
		"GPIF.ChannelStrip.Automation.Unsupported",
		"GPIF.ChannelStrip.Automation.Type.Unknown",
	} {
		if !slices.ContainsFunc(result.Diagnostics, func(diagnostic ParseDiagnostic) bool { return diagnostic.Code == code }) {
			t.Errorf("diagnostics = %#v, want %s", result.Diagnostics, code)
		}
	}
	for code, want := range map[string]int{
		"GPIF.MasterTrack.Automation.Tempo.Invalid":           2,
		"GPIF.MasterTrack.Automation.SyncPoint.Value.Invalid": 2,
		"GPIF.ChannelStrip.Automation.Unsupported":            2,
	} {
		got := 0
		for _, diagnostic := range result.Diagnostics {
			if diagnostic.Code == code {
				got++
			}
		}
		if got != want {
			t.Errorf("%s diagnostic count = %d, want %d", code, got, want)
		}
	}
	if got, want := result.Song.TempoAutomations, []TempoAutomation{{Bar: 0, Position: 0, Tempo: 120, Linear: true, Text: "source tempo", Hidden: true}}; !slices.Equal(got, want) {
		t.Fatalf("tempo automations after invalid records = %#v, want %#v", got, want)
	}
	if len(result.Song.SyncPoints) != 0 {
		t.Fatalf("sync points after invalid records = %#v, want none", result.Song.SyncPoints)
	}
	track := result.Song.Tracks[0]
	if len(track.SoundAutomations) != 2 {
		t.Fatalf("sound automations = %#v, want two resolved records", track.SoundAutomations)
	}
	run.ClaimPrimary(claimSite("automation-detail", "import", "M17-SOURCE-DISPATCH", "linear and step changes with the same value and position")).Dispatch("parseGPIFWithContext:automation.Type", track.SoundAutomations, []SoundAutomation{{Bar: 0, Position: 0, Sound: 0}, {Bar: 0, Position: 1, Sound: 1, Linear: true, Text: "source sound", Hidden: true}})
	if len(result.Song.VolumeAutomations) != 1 {
		t.Fatalf("volume automations = %#v, want one valid record", result.Song.VolumeAutomations)
	}
	run.ClaimPrimary(claimSite("volume-automation", "import", "M17-SOURCE-DISPATCH", "ordered tempo, sound, and volume changes")).Dispatch("gpifReadVolumeAutomations:automation.Type", result.Song.VolumeAutomations[0], VolumeAutomation{Track: 0, Bar: 0, Position: 1, Value: 0.875, Linear: true})

	masterContext := &parseContext{format: "GP8"}
	gpifAuditMasterAutomations([]gpifAutomation{{Type: "Tempo", Value: gpifAutomationValue{Text: "120 2"}}, {Type: "SyncPoint", Value: gpifAutomationValue{FrameOffset: "0"}}}, masterContext)
	run.Dispatch("gpifAuditMasterAutomations:automation.Type", len(masterContext.diagnostics), 0)
	tempoSong := &Song{}
	gpifReadTempoAutomations([]gpifAutomation{{Type: "Tempo", Bar: 1, Position: 0.5, Value: gpifAutomationValue{Text: "81.5 3"}}}, tempoSong, nil)
	run.Dispatch("gpifReadTempoAutomations:auto.Type", tempoSong.TempoAutomations, []TempoAutomation{{Bar: 1, Position: 0.5, Tempo: 122.25}})
	labelSong := &Song{TempoName: "stale"}
	gpifReadTempoAutomations([]gpifAutomation{
		{Type: "Tempo", Bar: 1, Value: gpifAutomationValue{Text: "90 2"}, Text: "later"},
		{Type: "Tempo", Bar: 0, Value: gpifAutomationValue{Text: "120 2"}},
	}, labelSong, nil)
	if labelSong.TempoName != "" {
		t.Fatalf("out-of-order empty opening label left stale name %q", labelSong.TempoName)
	}
	syncSong := &Song{}
	gpifReadSyncPoints([]gpifAutomation{{Type: "SyncPoint", Bar: 1, Position: 1, Linear: true, Visible: "false", Value: gpifAutomationValue{FrameOffset: "44100", BarOccurrence: "2"}}}, syncSong)
	run.Dispatch("gpifReadSyncPoints:automation.Type", len(syncSong.SyncPoints), 1)

	trackContext := &parseContext{format: "GP8"}
	gpifAuditTrackAutomations(gpifTrack{ID: "t", Sounds: gpifSounds{Sounds: []gpifSound{{Name: "Lead", Path: "a", Role: "main"}}}, Automations: gpifAutomations{Automations: []gpifAutomation{{Type: "Sound", Value: gpifAutomationValue{Text: "a;Lead;main"}}, {Type: "SustainPedal", Value: gpifAutomationValue{Text: "0 1"}}}}}, 1, trackContext)
	run.ClaimPrimary(claimSite("sound-automation", "import", "M17-SOURCE-DISPATCH", "similar sound names with distinct paths and roles")).Dispatch("gpifAuditTrackAutomations:automation.Type", len(trackContext.diagnostics), 0)
	channelContext := &parseContext{format: "GP8"}
	gpifAuditChannelStripAutomations([]gpifAutomation{{Type: "DSPParam_12", Value: gpifAutomationValue{Text: "0.5"}}, {Type: "DSPParam_00"}, {Type: "DSPParam_01"}, {Type: "DSPParam_11"}}, "t", channelContext)
	run.Dispatch("gpifAuditChannelStripAutomations:automation.Type", len(channelContext.diagnostics), 3)
}

func TestConformanceAutomationValidation(t *testing.T) {
	runConformanceAutomationValidation(newConformanceRun(t))
}

func runConformanceAutomationValidation(run *conformanceRun) {
	t := run.t
	valid := semanticValidPitchedGP8Song(t)
	valid.Tracks[0].Sounds = []TrackSound{{Name: "A", Program: 1}}
	valid.Tracks[0].SoundAutomations = []SoundAutomation{{Bar: 0, Position: 0, Sound: 0}, {Bar: 1, Position: 1, Sound: 0}}
	valid.TempoAutomations = []TempoAutomation{{Bar: 0, Position: 0, Tempo: 90}, {Bar: 1, Position: 1, Tempo: 120}}
	valid.VolumeAutomations = []VolumeAutomation{{Track: 0, Bar: 0, Position: 0, Value: 0}, {Track: 0, Bar: 1, Position: 1, Value: 1, Linear: true}}
	if diagnostics := ValidateSong(valid); len(diagnostics) != 0 {
		t.Fatalf("boundary automations = %#v, want valid", diagnostics)
	}
	run.Field("TempoAutomation.Position", valid.TempoAutomations[1].Position, 1.0)
	run.Field("SoundAutomation.Position", valid.Tracks[0].SoundAutomations[0].Position, 0.0)
	run.Field("VolumeAutomation.Value", valid.VolumeAutomations[1].Value, 1.0)

	tests := []struct {
		name string
		code string
		set  func(*Song)
	}{
		{name: "tempo negative bar", code: "score.tempo-automation.value", set: func(song *Song) { song.TempoAutomations = []TempoAutomation{{Bar: -1, Tempo: 90}} }},
		{name: "tempo high bar", code: "score.tempo-automation.value", set: func(song *Song) {
			song.TempoAutomations = []TempoAutomation{{Bar: len(song.MeasureHeaders), Tempo: 90}}
		}},
		{name: "tempo negative position", code: "score.tempo-automation.value", set: func(song *Song) { song.TempoAutomations = []TempoAutomation{{Bar: 0, Position: -0.01, Tempo: 90}} }},
		{name: "tempo NaN position", code: "score.tempo-automation.value", set: func(song *Song) { song.TempoAutomations = []TempoAutomation{{Bar: 0, Position: math.NaN(), Tempo: 90}} }},
		{name: "tempo infinite value", code: "score.tempo-automation.value", set: func(song *Song) { song.TempoAutomations = []TempoAutomation{{Bar: 0, Tempo: math.Inf(1)}} }},
		{name: "tempo negative value", code: "score.tempo-automation.value", set: func(song *Song) { song.TempoAutomations = []TempoAutomation{{Bar: 0, Tempo: -1}} }},
		{name: "sound negative reference", code: "score.sound-automation.reference", set: func(song *Song) { song.Tracks[0].SoundAutomations = []SoundAutomation{{Bar: 0, Sound: -1}} }},
		{name: "sound high reference", code: "score.sound-automation.reference", set: func(song *Song) { song.Tracks[0].SoundAutomations = []SoundAutomation{{Bar: 0, Sound: 1}} }},
		{name: "sound high bar", code: "score.sound-automation.location", set: func(song *Song) {
			song.Tracks[0].SoundAutomations = []SoundAutomation{{Bar: len(song.MeasureHeaders), Sound: 0}}
		}},
		{name: "sound infinite position", code: "score.sound-automation.location", set: func(song *Song) {
			song.Tracks[0].SoundAutomations = []SoundAutomation{{Bar: 0, Position: math.Inf(-1), Sound: 0}}
		}},
		{name: "volume negative track", code: "score.volume-automation.value", set: func(song *Song) { song.VolumeAutomations = []VolumeAutomation{{Track: -1, Bar: 0, Value: 0.5}} }},
		{name: "volume high track", code: "score.volume-automation.value", set: func(song *Song) { song.VolumeAutomations = []VolumeAutomation{{Track: 1, Bar: 0, Value: 0.5}} }},
		{name: "volume high position", code: "score.volume-automation.value", set: func(song *Song) {
			song.VolumeAutomations = []VolumeAutomation{{Track: 0, Bar: 0, Position: 1.01, Value: 0.5}}
		}},
		{name: "volume NaN value", code: "score.volume-automation.value", set: func(song *Song) { song.VolumeAutomations = []VolumeAutomation{{Track: 0, Bar: 0, Value: math.NaN()}} }},
		{name: "volume negative value", code: "score.volume-automation.value", set: func(song *Song) { song.VolumeAutomations = []VolumeAutomation{{Track: 0, Bar: 0, Value: -0.01}} }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			song := semanticValidPitchedGP8Song(t)
			song.Tracks[0].Sounds = []TrackSound{{Name: "A", Program: 1}}
			test.set(song)
			diagnostics := ValidateSong(song)
			if !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == test.code }) {
				t.Fatalf("diagnostics = %#v, want %s", diagnostics, test.code)
			}
		})
	}
}

func TestConformanceBinaryMixTable(t *testing.T) {
	runConformanceBinaryMixTable(newConformanceRun(t))
}

func runConformanceBinaryMixTable(run *conformanceRun) {
	data := []byte{11, 22, 33, 44, 55, 66, 77}
	data = binary.LittleEndian.AppendUint32(data, 303)
	data = append(data, 2, 3, 4, 5, 6, 1, 7, 0x3f)
	song := &Song{Version: Version{Number: [3]byte{4, 0, 0}}}
	change, err := song.readMixTableChange(newCursor(data))
	if err != nil {
		run.t.Fatal(err)
	}
	want := &MixTableChange{
		Tremolo:    &MixTableItem{Value: 77, Duration: 1, AllTracks: true},
		Volume:     &MixTableItem{Value: 22, Duration: 2, AllTracks: true},
		Balance:    &MixTableItem{Value: 33, Duration: 3, AllTracks: true},
		Chorus:     &MixTableItem{Value: 44, Duration: 4, AllTracks: true},
		Reverb:     &MixTableItem{Value: 55, Duration: 5, AllTracks: true},
		Instrument: &MixTableItem{Value: 11},
		Phaser:     &MixTableItem{Value: 66, Duration: 6, AllTracks: true},
		Tempo:      &MixTableItem{Value: 303, Duration: 7},
	}
	assertAutomationMixTableFields(run, &change, want)
	run.ClaimPrimary(
		claimSite("mix-table", "import", "M17-BINARY-MIX-TABLE", "every mix-table field"),
		claimSiteAtStage("mix-table", "model", "M17-BINARY-MIX-TABLE", "every mix-table field", "import"),
	).Omitted("BeatEffects.MixTableChange", &change, want)
}

func conformanceAutomationGPIFSource() string {
	master := `<MasterTrack><Tracks>0</Tracks><Automations>
    <Automation><Type>Tempo</Type><Linear>true</Linear><Bar>0</Bar><Position>0</Position><Visible>false</Visible><Value>120 2</Value><Text>source tempo</Text></Automation>
    <Automation><Type>Tempo</Type><Bar>0</Bar><Position>0</Position></Automation>
    <Automation><Type>Tempo</Type><Bar>0</Bar><Position>0.5</Position><Value>invalid</Value></Automation>
    <Automation><Type>SyncPoint</Type><Bar>0</Bar><Position>0</Position><Value/></Automation>
    <Automation><Type>SyncPoint</Type><Bar>0</Bar><Position>0</Position><Value><FrameOffset>bad</FrameOffset></Value></Automation>
    <Automation><Bar>0</Bar><Position>0</Position><Value>120 2</Value></Automation>
    <Automation><Type>FutureMaster</Type><Linear>true</Linear><Bar>0</Bar><Position>1</Position><Visible>true</Visible><Value>120 2</Value></Automation>
  </Automations></MasterTrack>`
	trackData := `<Sounds>
    <Sound><Name>Lead</Name><Label>Lead A</Label><Path>factory/a</Path><Role>main</Role><MIDI><Program>27</Program><PrimaryChannel>0</PrimaryChannel></MIDI></Sound>
    <Sound><Name>Lead</Name><Label>Lead B</Label><Path>factory/b</Path><Role>solo</Role><MIDI><Program>81</Program><PrimaryChannel>0</PrimaryChannel></MIDI></Sound>
  </Sounds><Automations>
    <Automation><Type>Sound</Type><Bar>0</Bar><Position>0</Position><Value>factory/a;Lead;main</Value></Automation>
    <Automation><Type>Sound</Type><Linear>true</Linear><Bar>0</Bar><Position>1</Position><Visible>false</Visible><Value>factory/b;Lead;solo</Value><Text>source sound</Text></Automation>
    <Automation><Type>Sound</Type><Bar>0</Bar><Position>0.5</Position><Value>factory/a;Lead;solo</Value></Automation>
    <Automation><Type>Sound</Type><Bar>0</Bar><Position>0.5</Position></Automation>
    <Automation><Type>SustainPedal</Type><Bar>0</Bar><Position>0.25</Position><Value>0 1</Value></Automation>
    <Automation><Bar>0</Bar><Position>0</Position><Value>factory/a;Lead;main</Value></Automation>
    <Automation><Type>FutureTrack</Type><Linear>true</Linear><Bar>0</Bar><Position>1</Position><Visible>true</Visible><Value>factory/a;Lead;main</Value></Automation>
  </Automations><RSE><ChannelStrip><Parameters>0 0 0 0 0 0 0 0 0 0 0.5 0.5 0.5</Parameters><Automations>
    <Automation><Type>DSPParam_12</Type><Linear>true</Linear><Bar>0</Bar><Position>1</Position><Value>0.875</Value></Automation>
    <Automation><Type>DSPParam_12</Type><Bar>0</Bar><Position>0</Position></Automation>
    <Automation><Type>DSPParam_00</Type><Bar>0</Bar><Position>0</Position><Value>0.1</Value></Automation>
    <Automation><Type>DSPParam_01</Type><Bar>0</Bar><Position>0</Position><Value>0.2</Value></Automation>
    <Automation><Type>DSPParam_11</Type><Bar>0</Bar><Position>0</Position><Value>0.3</Value></Automation>
    <Automation><Bar>0</Bar><Position>0</Position><Value>0.4</Value></Automation>
    <Automation><Type>FutureDSP</Type><Linear>true</Linear><Bar>0</Bar><Position>1</Position><Visible>true</Visible><Value>0.5</Value></Automation>
  </Automations></ChannelStrip></RSE>`
	result := strings.Replace(staffScopedChordGPIF, `<MasterTrack><Tracks>0</Tracks></MasterTrack>`, master, 1)
	return strings.Replace(result, `<Staves>`, trackData+`<Staves>`, 1)
}

func readAlphaTabAutomationFacts(t *testing.T, fixture string) any {
	t.Helper()
	var facts any
	readAlphaTabOracleFacts(t, "--automations", fixture, &facts)
	return facts
}

func readAlphaTabOracleFacts(t *testing.T, mode, fixture string, destination any) {
	t.Helper()
	command := exec.Command("node", alphaTabOracleScript(), mode, fixture)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("AlphaTab %s oracle: %v\n%s", mode, err, output)
	}
	if err := json.Unmarshal(output, destination); err != nil {
		t.Fatalf("decoding AlphaTab %s facts: %v\n%s", mode, err, output)
	}
}

type conformanceAutomationWireAutomation struct {
	Type     string  `xml:"Type"`
	Linear   bool    `xml:"Linear"`
	Value    string  `xml:"Value"`
	Visible  string  `xml:"Visible"`
	Text     string  `xml:"Text"`
	Bar      int     `xml:"Bar"`
	Position float64 `xml:"Position"`
}

type conformanceAutomationWireSound struct {
	Name    string `xml:"Name"`
	Label   string `xml:"Label"`
	Path    string `xml:"Path"`
	Role    string `xml:"Role"`
	Program int    `xml:"MIDI>Program"`
}

type conformanceAutomationWireDocument struct {
	masterAutomations []conformanceAutomationWireAutomation
	trackAutomations  []conformanceAutomationWireAutomation
	sounds            []conformanceAutomationWireSound
}

func extractAutomationWireDocument(t *testing.T, data []byte) conformanceAutomationWireDocument {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var gpif []byte
	for _, file := range archive.File {
		if file.Name != "Content/score.gpif" {
			continue
		}
		reader, openErr := file.Open()
		if openErr != nil {
			t.Fatal(openErr)
		}
		var readErr error
		gpif, readErr = readAutomationWireBytes(reader)
		closeErr := reader.Close()
		if readErr != nil {
			t.Fatal(readErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	if gpif == nil {
		t.Fatal("GP8 archive has no score.gpif")
	}
	var raw struct {
		MasterTrack struct {
			Automations struct {
				Items []conformanceAutomationWireAutomation `xml:"Automation"`
			} `xml:"Automations"`
		} `xml:"MasterTrack"`
		Tracks struct {
			Items []struct {
				Sounds struct {
					Items []conformanceAutomationWireSound `xml:"Sound"`
				} `xml:"Sounds"`
				Automations struct {
					Items []conformanceAutomationWireAutomation `xml:"Automation"`
				} `xml:"Automations"`
			} `xml:"Track"`
		} `xml:"Tracks"`
	}
	if err := xml.Unmarshal(gpif, &raw); err != nil {
		t.Fatal(err)
	}
	result := conformanceAutomationWireDocument{masterAutomations: raw.MasterTrack.Automations.Items}
	if len(raw.Tracks.Items) > 0 {
		result.trackAutomations = raw.Tracks.Items[0].Automations.Items
		result.sounds = raw.Tracks.Items[0].Sounds.Items
	}
	return result
}

func readAutomationWireBytes(reader interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var output bytes.Buffer
	_, err := output.ReadFrom(reader)
	return output.Bytes(), err
}

func conformanceAutomationStrings(items []conformanceAutomationWireAutomation, get func(conformanceAutomationWireAutomation) string) []string {
	result := make([]string, len(items))
	for index, item := range items {
		result[index] = get(item)
	}
	return result
}

func conformanceAutomationInts(items []conformanceAutomationWireAutomation, get func(conformanceAutomationWireAutomation) int) []int {
	result := make([]int, len(items))
	for index, item := range items {
		result[index] = get(item)
	}
	return result
}

func conformanceAutomationFloats(items []conformanceAutomationWireAutomation, get func(conformanceAutomationWireAutomation) float64) []float64 {
	result := make([]float64, len(items))
	for index, item := range items {
		result[index] = get(item)
	}
	return result
}

func conformanceAutomationBools(items []conformanceAutomationWireAutomation, get func(conformanceAutomationWireAutomation) bool) []bool {
	result := make([]bool, len(items))
	for index, item := range items {
		result[index] = get(item)
	}
	return result
}

func TestGP8VolumeConsumerLimits(t *testing.T) { runVolumeConsumerLimits(newConformanceRun(t), false) }
func TestAlphaTabVolumeConsumerLimits(t *testing.T) {
	requireAlphaTabConformance(t)
	runVolumeConsumerLimits(newConformanceRun(t), true)
}

func runVolumeConsumerLimits(run *conformanceRun, oracle bool) {
	t := run.t
	song := consumerLimitSong(t)
	song.Tracks = append(song.Tracks, song.Tracks[0])
	song.Tracks[1].Number = 2
	song.VolumeAutomations = []VolumeAutomation{{Track: 0, Bar: 0, Position: 0.25, Value: 0.25}, {Track: 0, Bar: 0, Position: 0.75, Value: 0.875, Linear: true}, {Track: 0, Bar: 0, Position: 0.75, Value: 0.375}, {Track: 1, Bar: 1, Position: 0.5, Value: 0.125, Linear: true}}
	codes := []string{"gp8.omit.volume-automation-consumer", "gp8.omit.volume-automation-consumer", "gp8.omit.volume-automation-consumer", "gp8.omit.volume-automation-consumer"}
	data, report := assertConsumerLossPolicy(t, song, codes)
	run.Report("M17-VOLUME-CONSUMER", reportCodes(report), codes)
	for i, entry := range report.Entries {
		event := song.VolumeAutomations[i]
		if entry.Location != (ScoreLocation{Track: event.Track, Measure: event.Bar}) || entry.Disposition != ExportDispositionOmitted || entry.Reason != fmt.Sprintf("pinned AlphaTab ignores channel-strip volume automation[%d] at position %g with value %g and linear=%t; GPIF retains the event", i, event.Position, event.Value, event.Linear) {
			t.Fatalf("unscoped volume report: %#v", entry)
		}
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("Song.VolumeAutomations", roundTrip.VolumeAutomations, song.VolumeAutomations)
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	var doc gpifDocument
	if err := xml.Unmarshal(readZipMember(t, archive, "Content/score.gpif"), &doc); err != nil {
		t.Fatal(err)
	}
	for ti, track := range doc.Tracks.Tracks {
		var want []gpifAutomation
		for _, a := range song.VolumeAutomations {
			if a.Track == ti {
				want = append(want, gpifAutomation{Type: "DSPParam_12", Bar: a.Bar, Position: a.Position, Linear: a.Linear, Value: gpifAutomationValue{Text: map[float64]string{0.25: "0.25", 0.875: "0.875", 0.375: "0.375", 0.125: "0.125"}[a.Value]}})
			}
		}
		run.Wire("gpifChannelStrip.Automations", track.RSE.ChannelStrip.Automations.Automations, want)
	}
	if oracle {
		var facts []map[string]any
		readAlphaTabOracleFacts(t, "--volume-automations", writeConformanceFixture(t, data), &facts)
		if len(facts) != 0 {
			t.Fatalf("pinned consumer unexpectedly retains channel-strip volume: %#v", facts)
		}
		// A legacy mix-table event is a positive control for the same adapter.
		// It is not evidence for the separate channel-strip contract.
		readAlphaTabOracleFacts(t, "--volume-automations", "testdata/gp5/RSE.gp5", &facts)
		if len(facts) == 0 {
			t.Fatal("volume adapter missed the legacy positive control")
		}

	}
}
