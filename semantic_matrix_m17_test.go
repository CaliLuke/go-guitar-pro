// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"math"
	"slices"
	"strings"
	"testing"
)

func TestSemanticMatrixM17AutomationSemantics(t *testing.T) {
	runSemanticMatrixM17AutomationSemantics(newSemanticMatrixRun(t))
}

func runSemanticMatrixM17AutomationSemantics(run *semanticMatrixRun) {
	t := run.t
	song := semanticValidPitchedGP8Song(t)
	song.Tracks[0].Settings.Notation = true
	song.Tempo = 111
	song.InitialTempo = KnownSourceValue(BPM(111))
	song.TempoAutomations = []TempoAutomation{
		{Bar: 0, Position: 0, Tempo: 111},
		{Bar: 1, Position: 0.75, Tempo: 145.5},
		{Bar: 1, Position: 0.25, Tempo: 91.25},
	}
	track := &song.Tracks[0]
	track.Sounds = []TrackSound{
		{Name: "Lead", Label: "Lead A", Path: "factory/a", Role: "main", Program: 27},
		{Name: "Lead", Label: "Lead B", Path: "factory/b", Role: "solo", Program: 81},
	}
	track.SoundAutomations = []SoundAutomation{
		{Bar: 0, Position: 0, Sound: 0},
		{Bar: 1, Position: 0.75, Sound: 1},
		{Bar: 1, Position: 0.25, Sound: 0},
	}
	song.VolumeAutomations = []VolumeAutomation{
		{Track: 0, Bar: 0, Position: 0, Value: 0.25, Linear: false},
		{Track: 0, Bar: 1, Position: 1, Value: 0.875, Linear: true},
	}

	wah := &WahEffect{Value: -23, Display: true}
	change := &MixTableChange{
		Tremolo:    &MixTableItem{Value: 11, Duration: 1, AllTracks: true},
		Volume:     &MixTableItem{Value: 22, Duration: 2, AllTracks: false},
		Balance:    &MixTableItem{Value: 33, Duration: 3, AllTracks: true},
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

	run.Field("Song.TempoAutomations", song.TempoAutomations, []TempoAutomation{{Bar: 0, Position: 0, Tempo: 111}, {Bar: 1, Position: 0.75, Tempo: 145.5}, {Bar: 1, Position: 0.25, Tempo: 91.25}})
	for index, automation := range song.TempoAutomations {
		run.Field("TempoAutomation.Bar", automation.Bar, []int{0, 1, 1}[index])
		run.Field("TempoAutomation.Position", automation.Position, []float64{0, 0.75, 0.25}[index])
		run.Field("TempoAutomation.Tempo", automation.Tempo, []float64{111, 145.5, 91.25}[index])
	}
	run.Field("Track.Sounds", track.Sounds, []TrackSound{{Name: "Lead", Label: "Lead A", Path: "factory/a", Role: "main", Program: 27}, {Name: "Lead", Label: "Lead B", Path: "factory/b", Role: "solo", Program: 81}})
	run.Field("Track.SoundAutomations", track.SoundAutomations, []SoundAutomation{{Bar: 0, Position: 0, Sound: 0}, {Bar: 1, Position: 0.75, Sound: 1}, {Bar: 1, Position: 0.25, Sound: 0}})
	for index, automation := range track.SoundAutomations {
		run.Field("SoundAutomation.Bar", automation.Bar, []int{0, 1, 1}[index])
		run.Field("SoundAutomation.Position", automation.Position, []float64{0, 0.75, 0.25}[index])
		run.Field("SoundAutomation.Sound", automation.Sound, []int{0, 1, 0}[index])
	}
	run.Omitted("Song.VolumeAutomations", song.VolumeAutomations, []VolumeAutomation{{Track: 0, Bar: 0, Position: 0, Value: 0.25}, {Track: 0, Bar: 1, Position: 1, Value: 0.875, Linear: true}})
	for index, automation := range song.VolumeAutomations {
		run.Preserved("VolumeAutomation.Track", automation.Track, 0)
		run.Preserved("VolumeAutomation.Bar", automation.Bar, index)
		run.Preserved("VolumeAutomation.Position", automation.Position, []float64{0, 1}[index])
		run.Preserved("VolumeAutomation.Value", automation.Value, []float64{0.25, 0.875}[index])
		run.Preserved("VolumeAutomation.Linear", automation.Linear, index == 1)
	}
	assertM17MixTableFields(run, track.Measures[0].Voices[0].Beats[0].Effect.MixTableChange, change)

	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	for _, code := range []string{"gp8.omit.volume-automations", "gp8.omit.beat-mix-table-change"} {
		if !hasExportCode(report, code) {
			t.Errorf("report = %#v, want %s", report.Entries, code)
		}
	}
	data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: reportCodes(report)}})
	if err != nil {
		t.Fatal(err)
	}
	wire := extractM17WireDocument(t, data)
	run.Wire("gpifAutomations.Automations", len(wire.masterAutomations), 3)
	run.Wire("gpifAutomation.Type", m17AutomationStrings(wire.masterAutomations, func(item m17WireAutomation) string { return item.Type }), []string{"Tempo", "Tempo", "Tempo"})
	run.Wire("gpifAutomation.Bar", m17AutomationInts(wire.masterAutomations, func(item m17WireAutomation) int { return item.Bar }), []int{0, 1, 1})
	run.Wire("gpifAutomation.Position", m17AutomationFloats(wire.masterAutomations, func(item m17WireAutomation) float64 { return item.Position }), []float64{0, 0.75, 0.25})
	run.Wire("gpifAutomation.Value", m17AutomationStrings(wire.masterAutomations, func(item m17WireAutomation) string { return item.Value }), []string{"111 2", "145.5 2", "91.25 2"})
	run.Wire("gpifAutomation.Linear", m17AutomationBools(wire.masterAutomations, func(item m17WireAutomation) bool { return item.Linear }), []bool{false, false, false})
	run.Wire("gpifAutomation.Visible", m17AutomationStrings(wire.masterAutomations, func(item m17WireAutomation) string { return item.Visible }), []string{"true", "true", "true"})
	run.Wire("gpifSounds.Sounds", len(wire.sounds), 2)
	run.Wire("gpifSound.Name", []string{wire.sounds[0].Name, wire.sounds[1].Name}, []string{"Lead", "Lead"})
	run.Wire("gpifSound.Label", []string{wire.sounds[0].Label, wire.sounds[1].Label}, []string{"Lead A", "Lead B"})
	run.Wire("gpifSound.Path", []string{wire.sounds[0].Path, wire.sounds[1].Path}, []string{"factory/a", "factory/b"})
	run.Wire("gpifSound.Role", []string{wire.sounds[0].Role, wire.sounds[1].Role}, []string{"main", "solo"})
	run.Wire("gpifSound.Program", []int{wire.sounds[0].Program, wire.sounds[1].Program}, []int{27, 81})
	run.Wire("gpifTrack.Automations", len(wire.trackAutomations), 3)
	run.Wire("gpifTrack.Sounds", len(wire.sounds), 2)
	run.Wire("gpifAutomation.Value", m17AutomationStrings(wire.trackAutomations, func(item m17WireAutomation) string { return item.Value }), []string{"factory/a;Lead;main", "factory/b;Lead;solo", "factory/a;Lead;main"})
	run.Wire("gpifAutomation.Position", m17AutomationFloats(wire.trackAutomations, func(item m17WireAutomation) float64 { return item.Position }), []float64{0, 0.75, 0.25})
	run.Wire("gpifAutomation.Linear", m17AutomationBools(wire.trackAutomations, func(item m17WireAutomation) bool { return item.Linear }), []bool{false, false, false})
	run.Wire("gpifAutomation.Visible", m17AutomationStrings(wire.trackAutomations, func(item m17WireAutomation) string { return item.Visible }), []string{"true", "true", "true"})

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Field("Song.TempoAutomations", roundTrip.TempoAutomations, song.TempoAutomations)
	run.Field("Track.SoundAutomations", roundTrip.Tracks[0].SoundAutomations, track.SoundAutomations)
	run.Field("Track.Sounds", roundTrip.Tracks[0].Sounds, track.Sounds)

	reordered := semanticValidPitchedGP8Song(t)
	reordered.Tracks[0].Settings.Notation = true
	reordered.Tracks[0].Sounds = []TrackSound{track.Sounds[1], track.Sounds[0]}
	reordered.Tracks[0].SoundAutomations = []SoundAutomation{
		{Bar: 0, Position: 0, Sound: 1},
		{Bar: 1, Position: 0.75, Sound: 0},
		{Bar: 1, Position: 0.25, Sound: 1},
	}
	reorderedData, err := Export(reordered, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	reorderedWire := extractM17WireDocument(t, reorderedData)
	if got, want := m17AutomationStrings(reorderedWire.trackAutomations, func(item m17WireAutomation) string { return item.Value }), m17AutomationStrings(wire.trackAutomations, func(item m17WireAutomation) string { return item.Value }); !slices.Equal(got, want) {
		t.Errorf("sound references after definition reorder = %#v, want %#v", got, want)
	}
}

func assertM17MixTableFields(run *semanticMatrixRun, got, want *MixTableChange) {
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

func TestSemanticMatrixM17SourceDispatchAndDiagnostics(t *testing.T) {
	runSemanticMatrixM17SourceDispatchAndDiagnostics(newSemanticMatrixRun(t))
}

func runSemanticMatrixM17SourceDispatchAndDiagnostics(run *semanticMatrixRun) {
	t := run.t
	source := m17AutomationGPIFSource()
	result, err := ParseWithOptions(conformanceGPIFArchive(t, source), ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{
		"GPIF.MasterTrack.Automation.Tempo.Invalid",
		"GPIF.MasterTrack.Automation.Tempo.Linear",
		"GPIF.MasterTrack.Automation.SyncPoint.Value.Invalid",
		"GPIF.MasterTrack.Automation.Type.Unknown",
		"GPIF.Track.Automation.Sound.Reference",
		"GPIF.Track.Automation.Sound.Linear",
		"GPIF.Track.Automation.SustainPedal",
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
		"GPIF.ChannelStrip.Automation.Unsupported":            3,
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
	if got := result.Song.TempoAutomations; len(got) != 1 || got[0].Tempo != 120 {
		t.Fatalf("tempo automations after invalid records = %#v, want one valid value", got)
	}
	if len(result.Song.SyncPoints) != 0 {
		t.Fatalf("sync points after invalid records = %#v, want none", result.Song.SyncPoints)
	}
	track := result.Song.Tracks[0]
	if len(track.SoundAutomations) != 2 {
		t.Fatalf("sound automations = %#v, want two resolved records", track.SoundAutomations)
	}
	run.Dispatch("parseGPIFWithContext:automation.Type", track.SoundAutomations, []SoundAutomation{{Bar: 0, Position: 0, Sound: 0}, {Bar: 0, Position: 1, Sound: 1}})
	if len(result.Song.VolumeAutomations) != 1 {
		t.Fatalf("volume automations = %#v, want one valid record", result.Song.VolumeAutomations)
	}
	run.Dispatch("gpifReadVolumeAutomations:automation.Type", result.Song.VolumeAutomations[0], VolumeAutomation{Track: 0, Bar: 0, Position: 1, Value: 0.875, Linear: true})

	masterContext := &parseContext{format: "GP8"}
	gpifAuditMasterAutomations([]gpifAutomation{{Type: "Tempo", Value: gpifAutomationValue{Text: "120 2"}}, {Type: "SyncPoint", Value: gpifAutomationValue{FrameOffset: "0"}}}, masterContext)
	run.Dispatch("gpifAuditMasterAutomations:automation.Type", len(masterContext.diagnostics), 0)
	tempoSong := &Song{}
	gpifReadTempoAutomations([]gpifAutomation{{Type: "Tempo", Bar: 1, Position: 0.5, Value: gpifAutomationValue{Text: "81.5 3"}}}, tempoSong, nil)
	run.Dispatch("gpifReadTempoAutomations:auto.Type", tempoSong.TempoAutomations, []TempoAutomation{{Bar: 1, Position: 0.5, Tempo: 122.25}})
	syncSong := &Song{}
	gpifReadSyncPoints([]gpifAutomation{{Type: "SyncPoint", Bar: 1, Position: 1, Linear: true, Visible: "false", Value: gpifAutomationValue{FrameOffset: "44100", BarOccurrence: "2"}}}, syncSong)
	run.Dispatch("gpifReadSyncPoints:automation.Type", len(syncSong.SyncPoints), 1)

	trackContext := &parseContext{format: "GP8"}
	gpifAuditTrackAutomations(gpifTrack{ID: "t", Sounds: gpifSounds{Sounds: []gpifSound{{Name: "Lead", Path: "a", Role: "main"}}}, Automations: gpifAutomations{Automations: []gpifAutomation{{Type: "Sound", Value: gpifAutomationValue{Text: "a;Lead;main"}}, {Type: "SustainPedal"}}}}, trackContext)
	run.Dispatch("gpifAuditTrackAutomations:automation.Type", len(trackContext.diagnostics), 1)
	channelContext := &parseContext{format: "GP8"}
	gpifAuditChannelStripAutomations([]gpifAutomation{{Type: "DSPParam_12", Value: gpifAutomationValue{Text: "0.5"}}, {Type: "DSPParam_00"}, {Type: "DSPParam_01"}, {Type: "DSPParam_11"}}, "t", channelContext)
	run.Dispatch("gpifAuditChannelStripAutomations:automation.Type", len(channelContext.diagnostics), 3)
}

func TestSemanticMatrixM17AutomationValidation(t *testing.T) {
	runSemanticMatrixM17AutomationValidation(newSemanticMatrixRun(t))
}

func runSemanticMatrixM17AutomationValidation(run *semanticMatrixRun) {
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

func TestSemanticMatrixM17BinaryMixTable(t *testing.T) {
	runSemanticMatrixM17BinaryMixTable(newSemanticMatrixRun(t))
}

func runSemanticMatrixM17BinaryMixTable(run *semanticMatrixRun) {
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
	assertM17MixTableFields(run, &change, want)
}

func m17AutomationGPIFSource() string {
	master := `<MasterTrack><Tracks>0</Tracks><Automations>
    <Automation><Type>Tempo</Type><Linear>true</Linear><Bar>0</Bar><Position>0</Position><Value>120 2</Value></Automation>
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
    <Automation><Type>Sound</Type><Linear>true</Linear><Bar>0</Bar><Position>1</Position><Value>factory/b;Lead;solo</Value></Automation>
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

type m17WireAutomation struct {
	Type     string  `xml:"Type"`
	Linear   bool    `xml:"Linear"`
	Value    string  `xml:"Value"`
	Visible  string  `xml:"Visible"`
	Bar      int     `xml:"Bar"`
	Position float64 `xml:"Position"`
}

type m17WireSound struct {
	Name    string `xml:"Name"`
	Label   string `xml:"Label"`
	Path    string `xml:"Path"`
	Role    string `xml:"Role"`
	Program int    `xml:"MIDI>Program"`
}

type m17WireDocument struct {
	masterAutomations []m17WireAutomation
	trackAutomations  []m17WireAutomation
	sounds            []m17WireSound
}

func extractM17WireDocument(t *testing.T, data []byte) m17WireDocument {
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
		gpif, readErr = readM17WireBytes(reader)
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
				Items []m17WireAutomation `xml:"Automation"`
			} `xml:"Automations"`
		} `xml:"MasterTrack"`
		Tracks struct {
			Items []struct {
				Sounds struct {
					Items []m17WireSound `xml:"Sound"`
				} `xml:"Sounds"`
				Automations struct {
					Items []m17WireAutomation `xml:"Automation"`
				} `xml:"Automations"`
			} `xml:"Track"`
		} `xml:"Tracks"`
	}
	if err := xml.Unmarshal(gpif, &raw); err != nil {
		t.Fatal(err)
	}
	result := m17WireDocument{masterAutomations: raw.MasterTrack.Automations.Items}
	if len(raw.Tracks.Items) > 0 {
		result.trackAutomations = raw.Tracks.Items[0].Automations.Items
		result.sounds = raw.Tracks.Items[0].Sounds.Items
	}
	return result
}

func readM17WireBytes(reader interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var output bytes.Buffer
	_, err := output.ReadFrom(reader)
	return output.Bytes(), err
}

func m17AutomationStrings(items []m17WireAutomation, get func(m17WireAutomation) string) []string {
	result := make([]string, len(items))
	for index, item := range items {
		result[index] = get(item)
	}
	return result
}

func m17AutomationInts(items []m17WireAutomation, get func(m17WireAutomation) int) []int {
	result := make([]int, len(items))
	for index, item := range items {
		result[index] = get(item)
	}
	return result
}

func m17AutomationFloats(items []m17WireAutomation, get func(m17WireAutomation) float64) []float64 {
	result := make([]float64, len(items))
	for index, item := range items {
		result[index] = get(item)
	}
	return result
}

func m17AutomationBools(items []m17WireAutomation, get func(m17WireAutomation) bool) []bool {
	result := make([]bool, len(items))
	for index, item := range items {
		result[index] = get(item)
	}
	return result
}
