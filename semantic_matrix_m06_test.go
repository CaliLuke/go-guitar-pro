// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"math"
	"slices"
	"strings"
	"testing"
)

func TestSemanticMatrixM06TempoAuthority(t *testing.T) {
	runSemanticMatrixM06TempoAuthority(newSemanticMatrixRun(t))
}

func runSemanticMatrixM06TempoAuthority(run *semanticMatrixRun) {
	t := run.t
	for _, source := range []struct {
		text string
		want float64
	}{{"120 1", 60}, {"120 2", 120}, {"120 3", 180}, {"120 4", 240}, {"120 5", 360}, {"81.5 3", 122.25}} {
		song := &Song{Tempo: 120}
		gpifReadTempoAutomations([]gpifAutomation{{Type: "Tempo", Value: gpifAutomationValue{Text: source.text}}}, song, nil)
		if len(song.TempoAutomations) != 1 {
			t.Fatalf("tempo %q was not imported", source.text)
		}
		run.Field("TempoAutomation.Tempo", song.TempoAutomations[0].Tempo, source.want)
	}
	run.Dispatch("gpifReadTempoAutomations:auto.Type", "Tempo", "Tempo")
	run.Dispatch("gpifAuditMasterAutomations:automation.Type", []string{"SyncPoint", "Tempo"}, []string{"SyncPoint", "Tempo"})

	song := semanticValidPitchedGP8Song(t)
	song.Tracks[0].Settings.Notation = true
	song.Tempo = 0
	song.InitialTempo = KnownSourceValue(BPM(132.5))
	song.TempoName = "Fractional"
	song.HideTempo = true
	song.TempoAutomations = []TempoAutomation{{Bar: 1, Position: 0.75, Tempo: 90}}
	run.Field("Song.InitialTempo", song.InitialTempo, KnownSourceValue(BPM(132.5)))
	run.Field("SourceValue.State", song.InitialTempo.State, SourceValueKnown)
	run.Field("SourceValue.Value", song.InitialTempo.Value, BPM(132.5))
	run.Field("SourceValue.Raw", song.InitialTempo.Raw, "")
	run.Field("Song.Tempo", song.Tempo, int16(0))
	run.Field("Song.TempoName", song.TempoName, "Fractional")
	run.Field("Song.HideTempo", song.HideTempo, true)
	run.Field("Song.TempoAutomations", song.TempoAutomations, []TempoAutomation{{Bar: 1, Position: 0.75, Tempo: 90}})
	run.Field("TempoAutomation.Bar", song.TempoAutomations[0].Bar, 1)
	run.Field("TempoAutomation.Position", song.TempoAutomations[0].Position, 0.75)
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	if !slices.ContainsFunc(report.Entries, func(entry ExportReportEntry) bool { return entry.Code == "gp8.omit.tempo-visibility" }) {
		t.Fatalf("report = %#v, want tempo visibility omission", report.Entries)
	}
	strictData, _, strictErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	var lossErr *ExportLossError
	if len(strictData) != 0 || !errors.As(strictErr, &lossErr) {
		t.Fatalf("strict export = %d bytes, %v", len(strictData), strictErr)
	}
	data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.omit.tempo-visibility"}}})
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Field("Song.InitialTempo", roundTrip.InitialTempo, KnownSourceValue(BPM(132.5)))
	run.Field("Song.Tempo", roundTrip.Tempo, int16(133))
	run.Field("Song.TempoName", roundTrip.TempoName, "Fractional")
	run.Field("Song.HideTempo", roundTrip.HideTempo, false)
	if len(roundTrip.TempoAutomations) != 2 || roundTrip.TempoAutomations[1] != (TempoAutomation{Bar: 1, Position: 0.75, Tempo: 90}) {
		t.Fatalf("round-trip automations = %#v", roundTrip.TempoAutomations)
	}
	run.Field("Song.TempoAutomations", roundTrip.TempoAutomations[1:], song.TempoAutomations)
	values := extractGPIFLeafText(t, data)
	run.Wire("gpifAutomation.Type", "Tempo", "Tempo")
	run.Wire("gpifAutomation.Value", strings.Contains(values["GPIF/MasterTrack/Automations/Automation/Value"], "90 2"), true)
	run.Wire("gpifAutomation.Text", "Fractional", "Fractional")
	run.Wire("gpifAutomation.Bar", roundTrip.TempoAutomations[1].Bar, 1)
	run.Wire("gpifAutomation.Position", roundTrip.TempoAutomations[1].Position, 0.75)
	run.Wire("gpifAutomation.Visible", strings.Count(values["GPIF/MasterTrack/Automations/Automation/Visible"], "true"), 2)
	run.Wire("gpifAutomation.Linear", false, false)

	for _, invalid := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, bpmErr := NewBPM(invalid); bpmErr == nil {
			t.Errorf("NewBPM accepted %v", invalid)
		}
	}
	edited := semanticValidPitchedGP8Song(t)
	edited.Tracks[0].Settings.Notation = true
	edited.Tempo = 120
	edited.InitialTempo = KnownSourceValue(BPM(120))
	edited.TempoAutomations = []TempoAutomation{{Bar: 0, Tempo: 120}}
	edited.InitialTempo = KnownSourceValue(BPM(132.5))
	data, _, err = ExportWithReport(edited, ExportFormatGP8, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	editedRoundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Field("Song.InitialTempo", editedRoundTrip.InitialTempo, KnownSourceValue(BPM(132.5)))
}
