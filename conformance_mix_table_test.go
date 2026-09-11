// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

const mixTableCase = "M17-MIX-TABLE-PROJECTION"

type mixTempoFact struct {
	Bar             int
	Position, Value float64
	Linear, Visible bool
	Text            string
}
type mixBeatFact struct {
	Track, Staff, Bar, Voice, Beat int
	Position                       float64
	Type                           string
	Value                          float64
	Linear                         bool
}
type mixProgramFact struct{ Track, Tick, Channel, Program int }
type mixConsumerFacts struct {
	Tempo    []mixTempoFact
	Beats    []mixBeatFact
	Programs []mixProgramFact
}

func TestConformanceMixTableProjection(t *testing.T) {
	runConformanceMixTableProjection(newConformanceRun(t))
}
func runConformanceMixTableProjection(run *conformanceRun) {
	assertMixTableControllerReports(run)
	song := parseTestFixture(run.t, "testdata/gp3/mix-table-events.gp3")
	run.Preserved("Song.VolumeAutomations", song.VolumeAutomations, []VolumeAutomation{{Track: 0, Bar: 0, Position: .25, Value: .75, Linear: true}})
	run.Preserved("Song.PanAutomations", song.PanAutomations, []PanAutomation{{Track: 0, Bar: 0, Position: .25, Value: .3125, Linear: true}})
	before := conformanceContractSnapshot(song, false)
	output, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
	if err != nil {
		run.t.Fatal(err)
	}
	codes := []string{}
	for _, entry := range report.Entries {
		if strings.HasPrefix(entry.Code, "gp8.omit.mix-table") || strings.HasPrefix(entry.Code, "gp8.normalize.mix-table") || entry.Code == "gp8.omit.volume-automation-consumer" || entry.Code == "gp8.omit.pan-automation-consumer" || entry.Code == "gp8.normalize.sound-automation-consumer-position" {
			codes = append(codes, entry.Code)
		}
	}
	run.Report(mixTableCase, codes, []string{"gp8.omit.pan-automation-consumer", "gp8.omit.volume-automation-consumer", "gp8.normalize.sound-automation-consumer-position"})
	out, err := Parse(output)
	if err != nil {
		run.t.Fatal(err)
	}
	run.Preserved("TempoAutomation.Position", out.TempoAutomations[1].Position, .25)
	run.Preserved("TempoAutomation.Tempo", out.TempoAutomations[1].Tempo, float64(173))
	run.Preserved("TempoAutomation.Linear", out.TempoAutomations[1].Linear, true)
	a := out.Tracks[0].SoundAutomations[0]
	run.Preserved("SoundAutomation.Position", a.Position, .25)
	run.Preserved("SoundAutomation.Linear", a.Linear, true)
	run.Preserved("TrackSound.Program", out.Tracks[0].Sounds[a.Sound].Program, int32(73))
	run.Preserved("TrackSound.Bank", out.Tracks[0].Sounds[a.Sound].Bank, int32(0))
	run.Preserved("Song.VolumeAutomations", out.VolumeAutomations, song.VolumeAutomations)
	run.Preserved("Song.PanAutomations", out.PanAutomations, song.PanAutomations)
	run.Omitted("BeatEffects.MixTableChange", out.Tracks[0].Measures[0].Voices[0].Beats[1].Effect.MixTableChange, (*MixTableChange)(nil))
	doc := conformanceWireDocument(run.t, output)
	tempo := doc.MasterTrack.Automations.Automations[1]
	run.Wire("gpifAutomation.Type", tempo.Type, "Tempo")
	run.Wire("gpifAutomation.Position", tempo.Position, .25)
	run.Wire("gpifAutomation.Value", tempo.Value.Text, "173 2")
	run.Wire("gpifAutomation.Linear", tempo.Linear, true)
	track := doc.Tracks.Tracks[0]
	run.Wire("gpifTrack.Automations", track.Automations.Automations[0].Position, .25)
	run.Wire("gpifChannelStrip.Automations", track.RSE.ChannelStrip.Automations.Automations, []gpifAutomation{{Type: "DSPParam_12", Bar: 0, Position: .25, Linear: true, Value: gpifAutomationValue{Text: "0.75"}}, {Type: "DSPParam_11", Bar: 0, Position: .25, Linear: true, Value: gpifAutomationValue{Text: "0.3125"}}})
	if !reflect.DeepEqual(before, conformanceContractSnapshot(song, false)) {
		run.t.Fatal("input mutated")
	}
}

func TestAlphaTabMixTableProjection(t *testing.T) {
	requireAlphaTabConformance(t)
	var source mixConsumerFacts
	readAlphaTabOracleFacts(t, "--mix-table", "testdata/gp3/mix-table-events.gp3", &source)
	wantSource := mixConsumerFacts{
		Tempo:    []mixTempoFact{{Bar: 0, Value: 120, Visible: true}, {Bar: 0, Position: .25, Value: 173, Linear: true, Visible: true}, {Bar: 0, Value: 120, Visible: true}, {Bar: 4, Value: 120, Linear: true, Visible: true}},
		Beats:    []mixBeatFact{{Type: "Instrument", Value: 25}, {Beat: 1, Position: .25, Type: "Volume", Value: 12, Linear: true}, {Beat: 1, Position: .25, Type: "Balance", Value: 5, Linear: true}, {Beat: 1, Position: .25, Type: "Instrument", Value: 73, Linear: true}, {Bar: 4, Type: "Instrument", Value: 25, Linear: true}},
		Programs: []mixProgramFact{{Program: 25}, {Channel: 1, Program: 25}, {Tick: 960, Program: 73}, {Tick: 960, Channel: 1, Program: 73}, {Tick: 15360, Program: 25}, {Tick: 15360, Channel: 1, Program: 25}},
	}
	if !reflect.DeepEqual(source, wantSource) {
		t.Fatalf("source=%#v want=%#v", source, wantSource)
	}
	song := parseTestFixture(t, "testdata/gp3/mix-table-events.gp3")
	output, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var got mixConsumerFacts
	readAlphaTabOracleFacts(t, "--mix-table", writeConformanceFixture(t, output), &got)
	// The raw consumer moves nonzero-position program changes to beat zero and
	// ignores channel-strip events. These exact losses have separate policy codes.
	want := mixConsumerFacts{
		Tempo:    []mixTempoFact{{Value: 120, Visible: true, Text: "Moderate"}, {Position: .25, Value: 173, Linear: true, Visible: true}, {Bar: 4, Value: 120, Linear: true, Visible: true}},
		Beats:    []mixBeatFact{{Position: .25, Type: "Instrument", Value: 73, Linear: true}, {Bar: 4, Type: "Bank"}, {Bar: 4, Type: "Instrument", Value: 25, Linear: true}},
		Programs: slices.Clone(wantSource.Programs),
	}
	want.Programs[2].Tick = 0
	want.Programs[3].Tick = 0
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("target=%#v want=%#v", got, want)
	}
}

func TestMixTableAllTracksAndEqualPositionOrder(t *testing.T) {
	song := consumerLimitSong(t)
	other := song.Tracks[0]
	other.Staves = nil
	other.Measures = make([]Measure, len(song.MeasureHeaders))
	for i := range other.Measures {
		other.Measures[i] = Measure{HeaderIndex: i, Voices: []Voice{{Beats: []Beat{{Duration: defaultDuration(), Status: BeatStatusRest}}}}}
	}
	song.Tracks = append(song.Tracks, other)
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	beat.Effect.MixTableChange = &MixTableChange{Volume: &MixTableItem{Value: 12, AllTracks: true}, Balance: &MixTableItem{Value: 5, AllTracks: true}, Instrument: &MixTableItem{Value: 73, AllTracks: true}}
	song.VolumeAutomations = []VolumeAutomation{{Track: 0, Value: .1}, {Track: 0, Value: .2, Linear: true}}
	projected, conflicts := projectMixTableAutomations(song)
	if len(conflicts) != 1 || conflicts[0].controller != "volume" {
		t.Fatal(conflicts)
	}
	if !reflect.DeepEqual(projected.VolumeAutomations, []VolumeAutomation{{Track: 0, Value: .1}, {Track: 0, Value: .2, Linear: true}, {Track: 1, Value: .75, Linear: true}}) {
		t.Fatal(projected.VolumeAutomations)
	}
	if !reflect.DeepEqual(projected.PanAutomations, []PanAutomation{{Track: 0, Value: .3125, Linear: true}, {Track: 1, Value: .3125, Linear: true}}) {
		t.Fatal(projected.PanAutomations)
	}
	for i := range projected.Tracks {
		a := projected.Tracks[i].SoundAutomations
		if len(a) != 1 || projected.Tracks[i].Sounds[a[0].Sound].Program != 73 {
			t.Fatal(a)
		}
	}
	output, e := Export(song, ExportFormatGP8)
	if e != nil {
		t.Fatal(e)
	}
	out, e := Parse(output)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(out.VolumeAutomations, projected.VolumeAutomations) || !reflect.DeepEqual(out.PanAutomations, projected.PanAutomations) {
		t.Fatal("ownership/order changed")
	}
}

func assertMixTableControllerReports(run *conformanceRun) {
	for _, test := range []struct {
		code   string
		change MixTableChange
	}{
		{"chorus", MixTableChange{Chorus: &MixTableItem{Value: 7, Duration: 2, AllTracks: true}}},
		{"reverb", MixTableChange{Reverb: &MixTableItem{Value: 8, Duration: 3}}},
		{"phaser", MixTableChange{Phaser: &MixTableItem{Value: 9, Duration: 4, AllTracks: true}}},
		{"tremolo", MixTableChange{Tremolo: &MixTableItem{Value: 10, Duration: 5}}},
		{"tempo-transition", MixTableChange{Tempo: &MixTableItem{Value: 173, Duration: 2}}},
		{"instrument-transition", MixTableChange{Instrument: &MixTableItem{Value: 73, Duration: 3}}},
		{"volume-transition", MixTableChange{Volume: &MixTableItem{Value: 12, Duration: 4}}},
		{"balance-transition", MixTableChange{Balance: &MixTableItem{Value: 5, Duration: 5}}},
		{"rse-instrument", MixTableChange{Rse: RseInstrument{Instrument: 3}}},
		{"rse-unknown", MixTableChange{Rse: RseInstrument{Unknown: -1}}},
		{"rse-sound-bank", MixTableChange{Rse: RseInstrument{SoundBank: 5}}},
		{"rse-effect-number", MixTableChange{Rse: RseInstrument{EffectNumber: 6}}},
		{"rse-effect-category", MixTableChange{Rse: RseInstrument{EffectCategory: "category"}}},
		{"rse-effect", MixTableChange{Rse: RseInstrument{Effect: "effect"}}},
		{"use-rse", MixTableChange{UseRse: true}},
		{"tempo-label", MixTableChange{TempoName: "orphan"}},
	} {
		song := consumerLimitSong(run.t)
		song.Tracks[0].Measures[1].Voices[0].Beats[0].Effect.MixTableChange = &test.change
		report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
		codes := []string{}
		for _, entry := range report.Entries {
			if strings.HasPrefix(entry.Code, "gp8.omit.mix-table-") {
				codes = append(codes, entry.Code)
				if entry.Location != (ScoreLocation{Measure: 1}) {
					run.t.Fatal(entry)
				}
			}
		}
		run.Report(mixTableCase, codes, []string{"gp8.omit.mix-table-" + test.code})
	}
}
