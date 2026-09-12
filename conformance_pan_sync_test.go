// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"math"
	"os"
	"reflect"
	"strconv"
	"testing"
)

func TestConformancePanAutomations(t *testing.T) { runConformancePanAutomations(newConformanceRun(t)) }
func TestAlphaTabPanAutomations(t *testing.T) {
	requireAlphaTabConformance(t)
	runConformancePanAutomations(newConformanceRun(t))
}
func runConformancePanAutomations(run *conformanceRun) {
	t := run.t
	song := consumerLimitSong(t)
	song.Channels[0].Balance = 32
	song.Tracks = append(song.Tracks, song.Tracks[0])
	song.PanAutomations = []PanAutomation{{Track: 0, Bar: 0, Position: 0.25, Value: 0}, {Track: 0, Bar: 0, Position: 0.75, Value: 1, Linear: true}, {Track: 0, Bar: 0, Position: 0.75, Value: 0.5}, {Track: 1, Bar: 1, Position: 1, Value: 0.25, Linear: true}}
	audit := &parseContext{format: "GP8"}
	gpifAuditChannelStripAutomations([]gpifAutomation{{Type: "DSPParam_11", Value: gpifAutomationValue{Text: "not-a-number"}}}, "track-a", audit)
	if len(audit.diagnostics) != 1 || audit.diagnostics[0].Code != "GPIF.ChannelStrip.Automation.Pan.Value.Invalid" || audit.diagnostics[0].ObjectID != "track-a" {
		t.Fatal(audit.diagnostics)
	}
	before := append([]PanAutomation(nil), song.PanAutomations...)
	data, report := assertConsumerLossPolicy(t, song, []string{"gp8.omit.pan-automation-consumer", "gp8.omit.pan-automation-consumer", "gp8.omit.pan-automation-consumer", "gp8.omit.pan-automation-consumer"})
	run.Report("M17-PAN-EVENTS", reportCodes(report), []string{"gp8.omit.pan-automation-consumer", "gp8.omit.pan-automation-consumer", "gp8.omit.pan-automation-consumer", "gp8.omit.pan-automation-consumer"})
	got, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("Song.PanAutomations", got.PanAutomations, before)
	for i, a := range got.PanAutomations {
		want := before[i]
		run.Preserved("PanAutomation.Track", a.Track, want.Track)
		run.Preserved("PanAutomation.Bar", a.Bar, want.Bar)
		run.Preserved("PanAutomation.Position", a.Position, want.Position)
		run.Preserved("PanAutomation.Value", a.Value, want.Value)
		run.Preserved("PanAutomation.Linear", a.Linear, want.Linear)
		if report.Entries[i].Location != (ScoreLocation{Track: a.Track, Measure: a.Bar}) {
			t.Fatal(report.Entries[i])
		}
	}
	run.Dispatch("gpifReadPanAutomations:a.Type", got.PanAutomations, before)
	doc := conformanceWireDocument(t, data)
	for ti, track := range doc.Tracks.Tracks {
		var values []gpifAutomation
		for _, a := range track.RSE.ChannelStrip.Automations.Automations {
			if a.Type == "DSPParam_11" {
				values = append(values, a)
			}
		}
		want := [][]gpifAutomation{
			{{Type: "DSPParam_11", Position: 0.25, Value: gpifAutomationValue{Text: "0"}}, {Type: "DSPParam_11", Position: 0.75, Linear: true, Value: gpifAutomationValue{Text: "1"}}, {Type: "DSPParam_11", Position: 0.75, Value: gpifAutomationValue{Text: "0.5"}}},
			{{Type: "DSPParam_11", Bar: 1, Position: 1, Linear: true, Value: gpifAutomationValue{Text: "0.25"}}},
		}[ti]
		run.Wire("gpifChannelStrip.Automations", values, want)
	}
	source := parseTestFixture(t, "testdata/gp5/motherload-percussion-grace.gp5")
	var legacy []PanAutomation
	for _, a := range source.PanAutomations {
		if (a.Track == 2 || a.Track == 3) && a.Bar == 143 {
			legacy = append(legacy, a)
		}
	}
	run.Field("Song.PanAutomations", legacy, []PanAutomation{{Track: 2, Bar: 143, Value: 0, Linear: false}, {Track: 3, Bar: 143, Value: 1, Linear: false}})
	if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
		var balances []int
		readAlphaTabOracleFacts(t, "--pan-initial-balances", writeConformanceFixture(t, data), &balances)
		if !reflect.DeepEqual(balances, []int{4, 4}) {
			t.Fatal(balances)
		}
		var facts []struct {
			Track, Bar, Beat int
			Value            float64
			Linear           bool
		}
		readAlphaTabOracleFacts(t, "--pan-automations", writeConformanceFixture(t, data), &facts)
		if len(facts) != 0 {
			t.Fatal(facts)
		}
		readAlphaTabOracleFacts(t, "--pan-automations", "testdata/gp5/motherload-percussion-grace.gp5", &facts)
		if !reflect.DeepEqual(facts, []struct {
			Track, Bar, Beat int
			Value            float64
			Linear           bool
		}{{2, 143, 0, 0, true}, {3, 143, 0, 16, true}}) {
			t.Fatal(facts)
		}
		legacyData, _, legacyErr := ExportWithReport(source, ExportFormatGP8, ExportOptions{})
		if legacyErr != nil {
			t.Fatal(legacyErr)
		}
		readAlphaTabOracleFacts(t, "--pan-automations", writeConformanceFixture(t, legacyData), &facts)
		if len(facts) != 0 {
			t.Fatal(facts)
		}
	}
}

func TestConformanceSyncPointExport(t *testing.T) {
	runConformanceSyncPointExport(newConformanceRun(t))
}
func TestAlphaTabSyncPointExport(t *testing.T) {
	requireAlphaTabConformance(t)
	runConformanceSyncPointExport(newConformanceRun(t))
}

func TestConformanceDenseSyncTempoExport(t *testing.T) {
	runConformanceDenseSyncTempoExport(newConformanceRun(t))
}

func TestAlphaTabDenseSyncTempoExport(t *testing.T) {
	requireAlphaTabConformance(t)
	runConformanceDenseSyncTempoExport(newConformanceRun(t))
}

func runConformanceDenseSyncTempoExport(run *conformanceRun) {
	t := run.t
	song := conformanceBackingProgrammaticSong(t)
	song.MeasureHeaders[0].RepeatStart = true
	song.MeasureHeaders[1].RepeatCount = 68
	song.SyncPoints = nil
	for i := range 136 {
		position := float64(i%4) / 4
		barPosition, err := NewBarPositionFromFloat64(position)
		if err != nil {
			t.Fatal(err)
		}
		frame := int64(i * 44100)
		song.SyncPoints = append(song.SyncPoints, SyncPoint{
			Bar: i % 2, BarOccurrence: i / 2, Position: position, BarPosition: barPosition,
			FrameOffset: frame, AudioFrame: AudioFrame(frame), MediaTimeMS: float64(i*1000 + 500),
			ModifiedTempo: 90.25 + float64(i), OriginalTempo: 120.5 + float64(i),
			Linear: i%2 == 0, Visible: i%3 == 0,
		})
	}
	// Exercise each metadata field independently as well as together.
	song.SyncPoints[0].ModifiedTempo = 0
	song.SyncPoints[1].OriginalTempo = 0
	before := append([]SyncPoint(nil), song.SyncPoints...)
	data, report := assertConsumerLossPolicy(t, song, []string{})
	run.Report("M19-SYNC-TEMPO-PRESERVATION", reportCodes(report), []string{})
	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("Song.SyncPoints", parsed.SyncPoints, before)
	if !reflect.DeepEqual(song.SyncPoints, before) {
		t.Fatal("export mutated sync points")
	}
	index := 0
	for _, automation := range conformanceWireDocument(t, data).MasterTrack.Automations.Automations {
		if automation.Type != "SyncPoint" {
			continue
		}
		if index >= len(before) {
			t.Fatal("export added sync points")
		}
		want := before[index]
		run.Wire("gpifAutomationValue.ModifiedTempo", automation.Value.ModifiedTempo, strconv.FormatFloat(want.ModifiedTempo, 'g', -1, 64))
		run.Wire("gpifAutomationValue.OriginalTempo", automation.Value.OriginalTempo, strconv.FormatFloat(want.OriginalTempo, 'g', -1, 64))
		index++
	}
	if index != len(before) {
		t.Fatalf("export retained %d of %d sync points", index, len(before))
	}
	if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
		var facts []struct{ ModifiedTempoPresent, OriginalTempoPresent bool }
		readAlphaTabOracleFacts(t, "--sync-points", writeConformanceFixture(t, data), &facts)
		if len(facts) != len(before) {
			t.Fatalf("consumer retained %d of %d sync points", len(facts), len(before))
		}
		for _, fact := range facts {
			if fact.ModifiedTempoPresent || fact.OriginalTempoPresent {
				t.Fatal("recheck the pinned consumer tempo limitation", fact)
			}
		}
	}
}

func runConformanceSyncPointExport(run *conformanceRun) {
	t := run.t
	song := conformanceBackingProgrammaticSong(t)
	for i := range song.SyncPoints {
		point := &song.SyncPoints[i]
		point.Position = []float64{0.25, 0.75}[i]
		point.BarPosition, _ = NewBarPositionFromFloat64(point.Position)
		point.BarOccurrence = 1 + 2*i
		point.FrameOffset = int64(44100 * (i + 1))
		point.AudioFrame = AudioFrame(point.FrameOffset)
		point.MediaTimeMS = float64(1500 + 1000*i)
	}
	data, report := assertConsumerLossPolicy(t, song, []string{})
	run.Report("M19-SYNC-EXPORT", reportCodes(report), []string{})
	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.Preserved("Song.SyncPoints", parsed.SyncPoints, song.SyncPoints)
	var points []gpifAutomation
	for _, a := range conformanceWireDocument(t, data).MasterTrack.Automations.Automations {
		if a.Type == "SyncPoint" {
			points = append(points, a)
		}
	}
	if len(points) != 2 {
		t.Fatal(points)
	}
	run.Wire("gpifAutomation.Type", []string{points[0].Type, points[1].Type}, []string{"SyncPoint", "SyncPoint"})
	run.Wire("gpifAutomation.Position", []float64{points[0].Position, points[1].Position}, []float64{0.25, 0.75})
	run.Wire("gpifAutomationValue.FrameOffset", []string{points[0].Value.FrameOffset, points[1].Value.FrameOffset}, []string{"44100", "88200"})
	run.Wire("gpifAutomationValue.BarOccurrence", []string{points[0].Value.BarOccurrence, points[1].Value.BarOccurrence}, []string{"1", "3"})
	run.Wire("gpifAutomationValue.ModifiedTempo", []string{points[0].Value.ModifiedTempo, points[1].Value.ModifiedTempo}, []string{"120.5", "90.25"})
	run.Wire("gpifAutomationValue.OriginalTempo", []string{points[0].Value.OriginalTempo, points[1].Value.OriginalTempo}, []string{"100", "100.75"})
	run.Wire("gpifAutomation.Visible", []string{points[0].Visible, points[1].Visible}, []string{"true", "false"})
	if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
		var facts []struct {
			Bar                                                         int
			Position                                                    float64
			Occurrence                                                  int
			Milliseconds                                                float64
			Linear, Visible, ModifiedTempoPresent, OriginalTempoPresent bool
		}
		readAlphaTabOracleFacts(t, "--sync-points", writeConformanceFixture(t, data), &facts)
		want := []struct {
			Bar                                                         int
			Position                                                    float64
			Occurrence                                                  int
			Milliseconds                                                float64
			Linear, Visible, ModifiedTempoPresent, OriginalTempoPresent bool
		}{{0, 0.25, 1, 1500, false, true, false, false}, {1, 0.75, 3, 2500, true, false, false, false}}
		if !reflect.DeepEqual(facts, want) {
			t.Fatalf("consumer %#v want %#v", facts, want)
		}
	}
	// Safe integer boundaries must not hide exact GPIF frame values.
	song.SyncPoints = song.SyncPoints[:1]
	song.SyncPoints[0].AudioFrame = AudioFrame(1<<53 + 1)
	song.SyncPoints[0].FrameOffset = int64(song.SyncPoints[0].AudioFrame)
	song.SyncPoints[0].MediaTimeMS = (float64(song.SyncPoints[0].AudioFrame) - float64(song.BackingTrack.FramePadding)) / GPIFBackingTrackSampleRate * 1000
	data, report, err = ExportWithReport(song, ExportFormatGP8, ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !hasExportCode(report, "gp8.normalize.sync-point-consumer-integer") {
		t.Fatal(report)
	}
	parsed, err = Parse(data)
	if err != nil || parsed.SyncPoints[0].AudioFrame != song.SyncPoints[0].AudioFrame {
		t.Fatal("frame narrowed", err)
	}
	for _, a := range conformanceWireDocument(t, data).MasterTrack.Automations.Automations {
		if a.Type == "SyncPoint" {
			run.Wire("gpifAutomationValue.FrameOffset", a.Value.FrameOffset, "9007199254740993")
		}
	}
	if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
		var facts []struct{ Milliseconds float64 }
		readAlphaTabOracleFacts(t, "--sync-points", writeConformanceFixture(t, data), &facts)
		frame, padding := float64(song.SyncPoints[0].AudioFrame), float64(song.BackingTrack.FramePadding)
		want := frame/44100*1000 - padding/44100*1000
		if len(facts) != 1 || facts[0].Milliseconds != want {
			t.Fatal(facts, want)
		}
	}
	if !gp8SyncIntegerExact(1<<53) || gp8SyncIntegerExact(1<<53+1) || !gp8SyncIntegerExact(1<<53+2) || gp8SyncIntegerExact(math.MaxInt64) {
		t.Fatal("integer boundary")
	}
}

func TestLegacyPanAllTracksUsesExactTime(t *testing.T) {
	start := int64(0)
	exact, _ := NewScoreTime(1, 2)
	song := &Song{MeasureHeaders: []MeasureHeader{{TimeSignature: TimeSignature{Numerator: 4, Denominator: Duration{Value: 4}}}}, Tracks: []Track{{Measures: []Measure{{Voices: []Voice{{Beats: []Beat{{Start: &start, ExactStart: &exact, Effect: BeatEffects{MixTableChange: &MixTableChange{Balance: &MixTableItem{Value: 8, AllTracks: true, Duration: 3}}}}}}}}}}, {Measures: []Measure{{}}}}}
	readBinaryPanAutomations(song)
	want := []PanAutomation{{Track: 0, Position: 1.0 / 7680, Value: 0.5, Linear: true}, {Track: 1, Position: 1.0 / 7680, Value: 0.5, Linear: true}}
	if !reflect.DeepEqual(song.PanAutomations, want) {
		t.Fatalf("events %#v want %#v", song.PanAutomations, want)
	}
	if song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.MixTableChange.Balance.Duration != 3 {
		t.Fatal("legacy transition metadata changed")
	}
}
