// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"os"
	"slices"
	"testing"
)

func TestConformanceBeatSemantics(t *testing.T) {
	runConformanceBeatSemantics(newConformanceRun(t))
}

func runConformanceBeatSemantics(run *conformanceRun) {
	t := run.t
	song := semanticValidPitchedGP8Song(t)
	song.Tracks[0].Settings.Notation = true
	song.MeasureHeaders = slices.Clone(song.MeasureHeaders[:1])
	song.Tracks[0].Measures = slices.Clone(song.Tracks[0].Measures[:1])
	quarter := defaultDuration()
	beats := []Beat{
		{Duration: quarter, Status: BeatStatusNormal, Dynamics: MinVelocity + VelocityIncrement*7, Text: "normal <&>", Notes: []Note{{Value: 1, String: 1, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1}}},
		{Duration: quarter, Status: BeatStatusRest, Dynamics: MinVelocity, Text: "rest text"},
		{Duration: quarter, Status: BeatStatusEmpty, Dynamics: DefaultVelocity, Text: "empty metadata"},
	}
	song.Tracks[0].Measures[0].Voices = []Voice{{Beats: beats}}
	song.Tracks[0].Staves[0].Measures = song.Tracks[0].Measures
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	for index := range beats {
		got := song.Tracks[0].Measures[0].Voices[0].Beats[index]
		run.Normalized("Beat.Status", got.Status, beats[index].Status)
		run.Normalized("Beat.Dynamics", got.Dynamics, beats[index].Dynamics)
		run.Preserved("Beat.Text", got.Text, beats[index].Text)
	}
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	for _, code := range []string{"gp8.normalize.note-velocity", "gp8.normalize.empty-beat"} {
		if !hasExportCode(report, code) {
			t.Errorf("report = %#v, want %s", report.Entries, code)
		}
	}
	data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: []string{"gp8.normalize.note-velocity", "gp8.normalize.empty-beat"}}})
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats
	run.Field("Beat.Status", []BeatStatus{got[0].Status, got[1].Status, got[2].Status}, []BeatStatus{BeatStatusNormal, BeatStatusRest, BeatStatusRest})
	run.Enum("BeatStatus.BeatStatusNormal", got[0].Status, BeatStatusNormal)
	run.Enum("BeatStatus.BeatStatusRest", got[1].Status, BeatStatusRest)
	run.Enum("BeatStatus.BeatStatusEmpty", got[2].Status, BeatStatusRest)
	run.Field("Beat.Dynamics", []int16{got[0].Dynamics, got[1].Dynamics, got[2].Dynamics}, []int16{MinVelocity + VelocityIncrement*7, MinVelocity, DefaultVelocity})
	run.Field("Beat.Text", []string{got[0].Text, got[1].Text, got[2].Text}, []string{"normal <&>", "rest text", "empty metadata"})
	values := extractGPIFLeafText(t, data)
	run.Wire("gpifBeat.Dynamic", values["GPIF/Beats/Beat/Dynamic"], "FFFPPPF")
	run.Wire("gpifBeat.FreeText", values["GPIF/Beats/Beat/FreeText"], "normal <&>rest textempty metadata")
}

func TestConformanceBeatEffects(t *testing.T) {
	runConformanceBeatEffects(newConformanceRun(t))
}

func runConformanceBeatEffects(run *conformanceRun) {
	t := run.t
	for hairpinIndex, hairpin := range []Hairpin{HairpinNone, HairpinCrescendo, HairpinDiminuendo} {
		wireHairpin := []string{"", "Crescendo", "Diminuendo"}[hairpinIndex]
		run.Enum([]string{"Hairpin.HairpinNone", "Hairpin.HairpinCrescendo", "Hairpin.HairpinDiminuendo"}[hairpinIndex], gpifHairpin(wireHairpin), hairpin)
		for strokeIndex, stroke := range []BeatStrokeDirection{BeatStrokeDirectionNone, BeatStrokeDirectionUp, BeatStrokeDirectionDown} {
			wireStroke := []string{"", "Up", "Down"}[strokeIndex]
			parsedStroke := Beat{}
			gpifApplyBeatEffects(&gpifBeat{Arpeggio: wireStroke}, &parsedStroke)
			run.Enum([]string{"BeatStrokeDirection.BeatStrokeDirectionNone", "BeatStrokeDirection.BeatStrokeDirectionUp", "BeatStrokeDirection.BeatStrokeDirectionDown"}[strokeIndex], parsedStroke.Effect.Stroke.Direction, stroke)
			beat := defaultBeat()
			beat.Effect.Hairpin = hairpin
			beat.Effect.Stroke = BeatStroke{Direction: stroke, Value: uint16(DurationEighth)}
			run.Preserved("Beat.Effect", beat.Effect, BeatEffects{Hairpin: hairpin, Stroke: BeatStroke{Direction: stroke, Value: uint16(DurationEighth)}})
			run.Preserved("BeatEffects.Hairpin", beat.Effect.Hairpin, hairpin)
			run.Normalized("BeatEffects.Stroke", beat.Effect.Stroke, BeatStroke{Direction: stroke, Value: uint16(DurationEighth)})
			run.Preserved("BeatStroke.Direction", beat.Effect.Stroke.Direction, stroke)
			run.Preserved("BeatStroke.Value", beat.Effect.Stroke.Value, uint16(DurationEighth))
		}
	}
	for index, value := range []SlapEffect{SlapEffectNone, SlapEffectTapping, SlapEffectSlapping, SlapEffectPopping} {
		probe := semanticValidPitchedGP8Song(t)
		probe.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.SlapEffect = value
		probeReport := PreflightExport(probe, ExportFormatGP8, ExportOptions{})
		run.Enum([]string{"SlapEffect.SlapEffectNone", "SlapEffect.SlapEffectTapping", "SlapEffect.SlapEffectSlapping", "SlapEffect.SlapEffectPopping"}[index], hasExportCode(probeReport, "gp8.omit.slap-effect"), value != SlapEffectNone)
		run.Omitted("BeatEffects.SlapEffect", BeatEffects{SlapEffect: value}.SlapEffect, value)
	}
	for _, value := range []BeatStrokeDirection{BeatStrokeDirectionNone, BeatStrokeDirectionUp, BeatStrokeDirectionDown} {
		run.Omitted("BeatEffects.PickStroke", BeatEffects{PickStroke: value}.PickStroke, value)
	}
	for _, source := range []struct {
		name  string
		value string
		want  Octave
	}{
		{"OctaveNone", "", OctaveNone},
		{"OctaveOttava", "8va", OctaveOttava},
		{"OctaveOttavaBassa", "8vb", OctaveOttavaBassa},
		{"OctaveQuindicesima", "15ma", OctaveQuindicesima},
		{"OctaveQuindicesimaBassa", "15mb", OctaveQuindicesimaBassa},
	} {
		beat := Beat{}
		gpifApplyBeatEffects(&gpifBeat{Ottavia: source.value}, &beat)
		run.Preserved("Beat.Octave", beat.Octave, source.want)
		run.Dispatch("gpifApplyBeatEffects:b.Ottavia", beat.Octave, source.want)

		song := semanticExportProbeSong(t)
		song.Tracks[0].Measures[0].Voices[0].Beats[0].Octave = source.want
		data, err := Export(song, ExportFormatGP8)
		if err != nil {
			t.Fatal(err)
		}
		wire := extractGPIFLeafText(t, data)["GPIF/Beats/Beat/Ottavia"]
		run.Wire("gpifBeat.Ottavia", wire, source.value)
		roundTrip, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Octave
		run.Enum("Octave."+source.name, got, source.want)
		if os.Getenv("ALPHATAB_CONFORMANCE") == "1" {
			run.Enum("Octave."+source.name, conformanceBeatAlphaTabFirstOctave(t, data), goOctave(source.want))
		}
	}
	for _, source := range []struct {
		value string
		want  BeatStrokeDirection
	}{{"Up", BeatStrokeDirectionUp}, {"Down", BeatStrokeDirectionDown}} {
		beat := Beat{}
		gpifApplyBeatEffects(&gpifBeat{Arpeggio: source.value}, &beat)
		run.Dispatch("gpifApplyBeatEffects:b.Arpeggio", beat.Effect.Stroke.Direction, source.want)
	}
	for _, source := range []struct{ name, want string }{{"Brush", "stroke-down"}, {"PickStroke", "pick-down"}, {"Slapped", "slap"}, {"Popped", "pop"}, {"VibratoWTremBar", "vibrato"}} {
		direction, strength := "Down", "Wide"
		property := gpifProperty{Name: source.name, Direction: &direction, Strength: &strength}
		beat := Beat{}
		gpifApplyBeatEffects(&gpifBeat{Properties: gpifProperties{Properties: []gpifProperty{property}}}, &beat)
		got := ""
		switch {
		case beat.Effect.Stroke.Direction == BeatStrokeDirectionDown:
			got = "stroke-down"
		case beat.Effect.PickStroke == BeatStrokeDirectionDown:
			got = "pick-down"
		case beat.Effect.SlapEffect == SlapEffectSlapping:
			got = "slap"
		case beat.Effect.SlapEffect == SlapEffectPopping:
			got = "pop"
		case beat.Effect.Vibrato:
			got = "vibrato"
		}
		run.Dispatch("gpifApplyBeatEffects:p.Name", got, source.want)
	}
	for _, direction := range []string{"Up", "Down"} {
		beat := Beat{}
		gpifApplyBeatEffects(&gpifBeat{Properties: gpifProperties{Properties: []gpifProperty{{Name: "Brush", Direction: &direction}}}}, &beat)
		run.Dispatch("gpifApplyBeatEffects:p.Direction", beat.Effect.Stroke.Direction, map[string]BeatStrokeDirection{"Up": BeatStrokeDirectionUp, "Down": BeatStrokeDirectionDown}[direction])
	}
	song := semanticValidPitchedGP8Song(t)
	song.Tracks[0].Settings.Notation = true
	beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	beat.Effect.FadeIn = true
	beat.Effect.Hairpin = HairpinCrescendo
	beat.Effect.HasRasgueado = true
	beat.Effect.PickStroke = BeatStrokeDirectionUp
	beat.Effect.SlapEffect = SlapEffectPopping
	beat.Effect.Vibrato = true
	beat.Effect.Stroke = BeatStroke{Direction: BeatStrokeDirectionDown, Value: uint16(DurationSixteenth)}
	run.Preserved("BeatEffects.FadeIn", beat.Effect.FadeIn, true)
	run.Omitted("BeatEffects.HasRasgueado", beat.Effect.HasRasgueado, true)
	run.Field("BeatEffects.PickStroke", beat.Effect.PickStroke, BeatStrokeDirectionUp)
	run.Field("BeatEffects.SlapEffect", beat.Effect.SlapEffect, SlapEffectPopping)
	run.Omitted("BeatEffects.Vibrato", beat.Effect.Vibrato, true)
	report := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	for _, code := range []string{"gp8.omit.rasgueado", "gp8.omit.pick-stroke", "gp8.omit.slap-effect", "gp8.omit.beat-vibrato", "gp8.normalize.stroke-duration"} {
		if !hasExportCode(report, code) {
			t.Errorf("report = %#v, want %s", report.Entries, code)
		}
	}
	data, _, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true, AllowedCodes: reportCodes(report)}})
	if err != nil {
		t.Fatal(err)
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Effect
	run.Field("BeatEffects.FadeIn", got.FadeIn, true)
	run.Field("BeatEffects.Hairpin", got.Hairpin, HairpinCrescendo)
	run.Field("BeatEffects.Stroke", got.Stroke, BeatStroke{Direction: BeatStrokeDirectionDown, Value: uint16(DurationEighth)})

	var lossErr *ExportLossError
	strict, _, strictErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if len(strict) != 0 || !errors.As(strictErr, &lossErr) {
		t.Fatalf("strict effect export = %d bytes, %v", len(strict), strictErr)
	}
	values := extractGPIFLeafText(t, data)
	run.Wire("gpifBeat.Fadding", values["GPIF/Beats/Beat/Fadding"], "FadeIn")
	run.Wire("gpifBeat.Hairpin", values["GPIF/Beats/Beat/Hairpin"], "Crescendo")
	run.Wire("gpifBeat.Arpeggio", values["GPIF/Beats/Beat/Arpeggio"], "Down")
}

func conformanceBeatAlphaTabFirstOctave(t *testing.T, data []byte) string {
	t.Helper()
	root := readAlphaTabScore(t, writeConformanceFixture(t, data)).(map[string]any)
	tracks := root["tracks"].([]any)
	staves := tracks[0].(map[string]any)["staves"].([]any)
	bars := staves[0].(map[string]any)["bars"].([]any)
	voices := bars[0].(map[string]any)["voices"].([]any)
	beats := voices[0].(map[string]any)["beats"].([]any)
	return beats[0].(map[string]any)["octave"].(string)
}
