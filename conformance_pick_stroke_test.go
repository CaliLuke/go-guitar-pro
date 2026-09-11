// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"reflect"
	"testing"
)

const conformancePickStrokeCase = "M10-PICK-STROKE"
const conformancePickStrokeValue = "up and down pick strokes independent from brush"

type conformancePickStrokeFact struct {
	Track      int    `json:"track"`
	Staff      int    `json:"staff"`
	Bar        int    `json:"bar"`
	Voice      int    `json:"voice"`
	Beat       int    `json:"beat"`
	PickStroke string `json:"pickStroke"`
	BrushType  string `json:"brushType"`
}

func TestConformancePickStroke(t *testing.T) {
	runConformancePickStroke(newConformanceRun(t))
}

func runConformancePickStroke(run *conformanceRun) {
	t := run.t
	source := parseTestFixture(t, "testdata/gp7/effects.gp")
	sourceBeats := source.Tracks[0].Measures[24].Voices[0].Beats
	run.Preserved("BeatEffects.PickStroke", []BeatStrokeDirection{sourceBeats[2].Effect.PickStroke, sourceBeats[3].Effect.PickStroke}, []BeatStrokeDirection{BeatStrokeDirectionUp, BeatStrokeDirectionDown})
	song := pickStrokeConformanceSong(t)
	before := append([]Beat(nil), song.Tracks[0].Measures[0].Voices[0].Beats...)
	options := ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}}
	preflight := PreflightExport(song, ExportFormatGP8, options)
	data, report, err := ExportWithReport(song, ExportFormatGP8, options)
	if err != nil || !reflect.DeepEqual(report, preflight) {
		t.Fatalf("strict export %v %#v", err, report)
	}
	run.ClaimReport(claimSite("pick-stroke", "export", conformancePickStrokeCase, conformancePickStrokeValue)).Report(conformancePickStrokeCase, reportCodes(report), []string{})
	wire := conformanceWireDocument(t, data).Beats.Beats
	if len(wire) != 3 {
		t.Fatalf("wire beats %d", len(wire))
	}
	var picks, brushes, names []string
	for _, beat := range wire {
		pick, brush, name := "", "", ""
		for _, property := range beat.Properties.Properties {
			if property.Name == "PickStroke" {
				if name != "" {
					t.Fatal("duplicate pick property")
				}
				name = property.Name
			}
			if property.Direction == nil {
				continue
			}
			if property.Name == "PickStroke" {
				pick = *property.Direction
			}
			if property.Name == "Brush" {
				brush = *property.Direction
			}
		}
		names = append(names, name)
		picks = append(picks, pick)
		brushes = append(brushes, brush)
	}
	run.ClaimSerialization(claimSite("pick-stroke", "export", conformancePickStrokeCase, conformancePickStrokeValue)).Wire("gpifProperty.Direction", picks, []string{"Up", "Down", ""})
	run.Wire("gpifProperty.Name", names, []string{"PickStroke", "PickStroke", ""})
	run.Wire("gpifProperty.Direction", brushes, []string{"Down", "Up", "Down"})
	parsed, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := parsed.Tracks[0].Measures[0].Voices[0].Beats
	var values []BeatStrokeDirection
	for i := range got {
		values = append(values, got[i].Effect.PickStroke)
		run.Field("BeatStroke.Direction", got[i].Effect.Stroke.Direction, before[i].Effect.Stroke.Direction)
	}
	run.ClaimPrimary(claimSite("pick-stroke", "export", conformancePickStrokeCase, conformancePickStrokeValue)).Preserved("BeatEffects.PickStroke", values, []BeatStrokeDirection{BeatStrokeDirectionUp, BeatStrokeDirectionDown, BeatStrokeDirectionNone})
	if !reflect.DeepEqual(song.Tracks[0].Measures[0].Voices[0].Beats, before) {
		t.Fatal("export mutated beats")
	}
}

func pickStrokeConformanceSong(t *testing.T) *Song {
	t.Helper()
	song := semanticValidPitchedGP8Song(t)
	song.Tracks[0].Settings.Notation = true
	song.MeasureHeaders = song.MeasureHeaders[:1]
	song.Tracks[0].Measures = song.Tracks[0].Measures[:1]
	song.Tracks[0].Measures[0].Voices = song.Tracks[0].Measures[0].Voices[:1]
	base := song.Tracks[0].Measures[0].Voices[0].Beats[0]
	base.Notes = []Note{{Value: 3, String: 1, Kind: NoteTypeNormal, Velocity: 47, DurationPercent: 1}}
	base.Dynamics = 47
	beats := make([]Beat, 3)
	for index, pick := range []BeatStrokeDirection{BeatStrokeDirectionUp, BeatStrokeDirectionDown, BeatStrokeDirectionNone} {
		beat := base
		beat.Notes = append([]Note(nil), base.Notes...)
		brush := BeatStrokeDirectionDown
		if index == 1 {
			brush = BeatStrokeDirectionUp
		}
		beat.Effect = BeatEffects{PickStroke: pick, Stroke: BeatStroke{Kind: BeatStrokeKindBrush, Direction: brush, Duration: NoteValue(DurationEighth)}}
		beats[index] = beat
	}
	song.Tracks[0].Measures[0].Voices[0].Beats = beats
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	if diagnostics := ValidateSong(song); len(diagnostics) != 0 {
		t.Fatalf("invalid public pick score %#v", diagnostics)
	}
	return song
}

func TestAlphaTabPreservesPickStroke(t *testing.T) {
	requireAlphaTabConformance(t)
	data, report, err := ExportWithReport(pickStrokeConformanceSong(t), ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("strict export %v %#v", err, report)
	}
	var facts []conformancePickStrokeFact
	readAlphaTabOracleFacts(t, "--pick-stroke", writeConformanceFixture(t, data), &facts)
	expected := []conformancePickStrokeFact{{PickStroke: "up", BrushType: "brushdown"}, {Beat: 1, PickStroke: "down", BrushType: "brushup"}, {Beat: 2, PickStroke: "none", BrushType: "brushdown"}}
	var authored []conformancePickStrokeFact
	for _, fact := range facts {
		if fact.Voice == 0 {
			authored = append(authored, fact)
		} else if fact.PickStroke != "none" {
			t.Fatalf("extra pick: %#v", fact)
		}
	}
	if !reflect.DeepEqual(authored, expected) {
		t.Fatalf("consumer picks %#v want %#v", facts, expected)
	}
	conformanceIndependentClaim(t, "field:BeatEffects.PickStroke", claimSite("pick-stroke", "export", conformancePickStrokeCase, conformancePickStrokeValue))
	fixture := "testdata/gp7/effects.gp"
	var source []conformancePickStrokeFact
	readAlphaTabOracleFacts(t, "--pick-stroke", fixture, &source)
	data, err = Export(parseTestFixture(t, fixture), ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	readAlphaTabOracleFacts(t, "--pick-stroke", writeConformanceFixture(t, data), &facts)
	wantSource := []conformancePickStrokeFact{{Bar: 24, Beat: 2, PickStroke: "up", BrushType: "none"}, {Bar: 24, Beat: 3, PickStroke: "down", BrushType: "none"}}
	if !reflect.DeepEqual(nonzeroPickStrokeFacts(source), wantSource) || !reflect.DeepEqual(nonzeroPickStrokeFacts(facts), wantSource) {
		t.Fatalf("source or exported picks differ: source %#v target %#v", nonzeroPickStrokeFacts(source), nonzeroPickStrokeFacts(facts))
	}
}

func nonzeroPickStrokeFacts(facts []conformancePickStrokeFact) []conformancePickStrokeFact {
	var result []conformancePickStrokeFact
	for _, fact := range facts {
		if fact.PickStroke != "none" {
			result = append(result, fact)
		}
	}
	return result
}
