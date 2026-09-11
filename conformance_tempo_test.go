// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func TestConformanceTempoAuthority(t *testing.T) {
	runConformanceTempoAuthority(newConformanceRun(t))
}

func runConformanceTempoAuthority(run *conformanceRun) {
	t := run.t
	run.Enum("SourceValueState.SourceValueMissing", (SourceValue[BPM]{}).State, SourceValueMissing)
	run.Enum("SourceValueState.SourceValueKnown", KnownSourceValue(BPM(120)).State, SourceValueKnown)
	run.Enum("SourceValueState.SourceValueUnknown", UnknownSourceValue[BPM]("future").State, SourceValueUnknown)
	for _, source := range []struct {
		text string
		want float64
	}{{"120 1", 60}, {"120 2", 120}, {"120 3", 180}, {"120 4", 240}, {"120 5", 360}, {"81.5 3", 122.25}} {
		song := &Song{Tempo: 120}
		gpifReadTempoAutomations([]gpifAutomation{{Type: "Tempo", Value: gpifAutomationValue{Text: source.text}}}, song, nil)
		if len(song.TempoAutomations) != 1 {
			t.Fatalf("tempo %q was not imported", source.text)
		}
		run.Preserved("TempoAutomation.Tempo", song.TempoAutomations[0].Tempo, source.want)
	}
	dispatchSong := &Song{Tempo: 120}
	gpifReadTempoAutomations([]gpifAutomation{{Type: "Tempo", Value: gpifAutomationValue{Text: "90 2"}}}, dispatchSong, nil)
	run.Dispatch("gpifReadTempoAutomations:auto.Type", dispatchSong.TempoAutomations, []TempoAutomation{{Tempo: 90}})
	auditContext := &parseContext{format: "GP8"}
	gpifAuditMasterAutomations([]gpifAutomation{{Type: "Tempo", Value: gpifAutomationValue{Text: "90 2"}}, {Type: "SyncPoint", Value: gpifAutomationValue{Text: "0"}}}, auditContext)
	run.Dispatch("gpifAuditMasterAutomations:automation.Type", len(auditContext.diagnostics), 0)

	song := semanticValidPitchedGP8Song(t)
	song.Tracks[0].Settings.Notation = true
	song.Tempo = 0
	song.InitialTempo = KnownSourceValue(BPM(132.5))
	song.TempoName = "Fractional"
	song.HideTempo = true
	song.TempoAutomations = []TempoAutomation{{Bar: 1, Position: 0.75, Tempo: 90}}
	run.Preserved("Song.InitialTempo", song.InitialTempo, KnownSourceValue(BPM(132.5)))
	run.Preserved("SourceValue.State", song.InitialTempo.State, SourceValueKnown)
	run.Preserved("SourceValue.Value", song.InitialTempo.Value, BPM(132.5))
	run.Preserved("SourceValue.Raw", song.InitialTempo.Raw, "")
	run.Normalized("Song.Tempo", song.Tempo, int16(0))
	run.Preserved("Song.TempoName", song.TempoName, "Fractional")
	run.Preserved("Song.HideTempo", song.HideTempo, true)
	run.ClaimPrimary(claimSite("tempo", "import", "M06-TEMPO-AUTHORITY", "fractional opening tempo")).Preserved("Song.TempoAutomations", song.TempoAutomations, []TempoAutomation{{Bar: 1, Position: 0.75, Tempo: 90}})
	run.Preserved("TempoAutomation.Bar", song.TempoAutomations[0].Bar, 1)
	run.Preserved("TempoAutomation.Position", song.TempoAutomations[0].Position, 0.75)
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	run.ClaimReport(claimSite("tempo", "export", "M06-TEMPO-AUTHORITY", "hidden fractional opening tempo")).Report("M06-TEMPO-AUTHORITY", reportCodes(report), []string{})
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
	run.ClaimPrimary(claimSite("tempo", "export", "M06-TEMPO-AUTHORITY", "hidden fractional opening tempo")).Field("Song.HideTempo", roundTrip.HideTempo, true)
	if len(roundTrip.TempoAutomations) != 2 || roundTrip.TempoAutomations[1] != (TempoAutomation{Bar: 1, Position: 0.75, Tempo: 90}) {
		t.Fatalf("round-trip automations = %#v", roundTrip.TempoAutomations)
	}
	run.ClaimPrimary(claimSite("tempo", "model", "M06-TEMPO-AUTHORITY", "fractional opening tempo")).Field("Song.TempoAutomations", roundTrip.TempoAutomations[1:], song.TempoAutomations)
	values := extractGPIFLeafText(t, data)
	run.Wire("gpifAutomation.Type", strings.Count(values["GPIF/MasterTrack/Automations/Automation/Type"], "Tempo"), 2)
	run.Wire("gpifAutomation.Value", strings.Contains(values["GPIF/MasterTrack/Automations/Automation/Value"], "90 2"), true)
	run.Wire("gpifAutomation.Text", values["GPIF/MasterTrack/Automations/Automation/Text"], "Fractional")
	run.Wire("gpifAutomation.Bar", roundTrip.TempoAutomations[1].Bar, 1)
	run.Wire("gpifAutomation.Position", roundTrip.TempoAutomations[1].Position, 0.75)
	run.ClaimSerialization(claimSite("tempo", "export", "M06-TEMPO-AUTHORITY", "hidden fractional opening tempo")).Wire("gpifAutomation.Visible", values["GPIF/MasterTrack/Automations/Automation/Visible"], "falsetrue")
	run.Wire("gpifAutomation.Linear", strings.Count(values["GPIF/MasterTrack/Automations/Automation/Linear"], "false"), 2)

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

func TestAlphaTabPreservesOpeningTempoVisibility(t *testing.T) {
	requireAlphaTabConformance(t)
	for _, hidden := range []bool{false, true} {
		song := semanticValidPitchedGP8Song(t)
		song.Tempo = 0
		song.InitialTempo = KnownSourceValue(BPM(132.5))
		song.TempoName = "Fractional"
		song.HideTempo = hidden
		song.TempoAutomations = nil
		assertOpeningTempoConsumer(t, song, 132.5, "Fractional", hidden)
		if hidden {
			conformanceIndependentClaim(t, "field:Song.HideTempo", claimSite("tempo", "export", "M06-TEMPO-AUTHORITY", "hidden fractional opening tempo"))
		}
	}
	legacy, err := ParseFile("testdata/gp5/nightwish.gp5")
	if err != nil {
		t.Fatal(err)
	}
	if !legacy.HideTempo || legacy.Version.Number != [3]byte{5, 1, 0} {
		t.Fatalf("GP5 hidden source = %#v, hidden %v, automations %#v", legacy.Version, legacy.HideTempo, legacy.TempoAutomations)
	}
	wantChanges := []TempoAutomation{
		{Bar: 88, Position: 0.625, Tempo: 85, Linear: true, Hidden: true},
		{Bar: 89, Tempo: 90, Linear: true, Hidden: true},
		{Bar: 92, Tempo: 80, Linear: true, Hidden: true},
		{Bar: 93, Tempo: 80, Linear: true, Hidden: true},
		{Bar: 93, Position: 0.875, Tempo: 60, Linear: true, Hidden: true},
		{Bar: 94, Tempo: 80, Linear: true, Hidden: true},
		{Bar: 85, Tempo: 90, Linear: true},
		{Bar: 93, Position: 0.75, Tempo: 70, Linear: true},
	}
	if !reflect.DeepEqual(legacy.TempoAutomations, wantChanges) {
		t.Fatalf("legacy tempo events = %#v, want %#v", legacy.TempoAutomations, wantChanges)
	}
	// This corpus label contains non-UTF-8 bytes. Author a valid label while
	// retaining the imported GP5 tempo and visibility.
	legacy.TempoName = "Legacy hidden"
	assertOpeningTempoConsumer(t, legacy, float64(legacy.InitialTempo.Value), legacy.TempoName, true)
}

func assertOpeningTempoConsumer(t *testing.T, song *Song, bpm float64, text string, hidden bool) {
	t.Helper()
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if hasExportCode(report, "gp8.omit.tempo-visibility") || hasExportCode(report, "gp8.normalize.tempo-visibility-authority") {
		t.Fatalf("unexpected visibility report %#v", report.Entries)
	}
	wire := extractAutomationWireDocument(t, data)
	if len(wire.masterAutomations) != 1+len(song.TempoAutomations) || wire.masterAutomations[0].Visible != strconv.FormatBool(!hidden) {
		t.Fatalf("opening visibility wire %#v", wire.masterAutomations)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	want := append([]TempoAutomation{{Tempo: bpm, Text: text, Hidden: hidden}}, song.TempoAutomations...)
	if !reflect.DeepEqual(roundTrip.TempoAutomations, want) || roundTrip.HideTempo != hidden {
		t.Fatalf("round trip = %#v hidden %v", roundTrip.TempoAutomations, roundTrip.HideTempo)
	}
	facts := readAlphaTabAutomationFacts(t, writeConformanceFixture(t, data)).(map[string]any)
	expected := []any{map[string]any{"bar": float64(0), "position": float64(0), "type": "tempo", "value": bpm, "linear": false, "text": text, "visible": !hidden}}
	for _, automation := range song.TempoAutomations {
		expected = append(expected, map[string]any{"bar": float64(automation.Bar), "position": automation.Position, "type": "tempo", "value": automation.Tempo, "linear": automation.Linear, "text": automation.Text, "visible": !automation.Hidden})
	}
	// The consumer enumerates master bars, retaining event order within each bar.
	sort.SliceStable(expected, func(i, j int) bool {
		return expected[i].(map[string]any)["bar"].(float64) < expected[j].(map[string]any)["bar"].(float64)
	})
	if differences := semanticDifferences(facts["tempo"], expected); len(differences) != 0 {
		t.Fatalf("consumer tempo differs %#v", differences)
	}
}
