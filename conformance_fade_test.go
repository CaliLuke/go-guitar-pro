// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const conformanceFadeCase = "M10-FADE"

type conformanceBeatFadeFact struct {
	Track int    `json:"track"`
	Staff int    `json:"staff"`
	Bar   int    `json:"bar"`
	Voice int    `json:"voice"`
	Beat  int    `json:"beat"`
	Fade  string `json:"fade"`
}

func TestConformanceBeatFade(t *testing.T) {
	runConformanceBeatFade(newConformanceRun(t))
}

func runConformanceBeatFade(run *conformanceRun) {
	t := run.t
	values := []BeatFade{BeatFadeNone, BeatFadeIn, BeatFadeOut, BeatFadeVolumeSwell}
	names := []string{"BeatFade.BeatFadeNone", "BeatFade.BeatFadeIn", "BeatFade.BeatFadeOut", "BeatFade.BeatFadeVolumeSwell"}
	for index, value := range values {
		run.Enum(names[index], value, BeatFade(index))
	}

	for _, fixture := range []string{"testdata/gp3/Effects.gp3", "testdata/gp4/Effects.gp4", "testdata/gp5/Effects.gp5"} {
		fades := conformanceBeatFades(parseTestFixture(t, fixture))
		if len(fades) == 0 || slices.ContainsFunc(fades, func(fade BeatFade) bool { return fade != BeatFadeIn }) {
			t.Fatalf("%s binary fades = %#v, want nonempty FadeIn-only values", fixture, fades)
		}
	}

	gp7 := parseTestFixture(t, "testdata/gp7/fade.gp")
	gp7Fades := conformanceBeatFades(gp7)
	if conformanceCountBeatFade(gp7Fades, BeatFadeVolumeSwell) != 2 || conformanceCountBeatFade(gp7Fades, BeatFadeOut) != 1 || conformanceCountBeatFade(gp7Fades, BeatFadeIn) != 1 {
		t.Fatalf("GP7 fades = %#v, want two VolumeSwell, one FadeOut, and one FadeIn", gp7Fades)
	}
	run.ClaimPrimary(claimSite("fade-other", "import", conformanceFadeCase, "none, fade-in, fade-out, and volume-swell values")).Preserved("BeatEffects.Fade", gp7Fades, []BeatFade{BeatFadeVolumeSwell, BeatFadeVolumeSwell, BeatFadeIn, BeatFadeOut})
	for _, beat := range conformanceAllBeats(gp7) {
		fade, _ := beat.Effect.resolvedFade()
		if beat.Effect.FadeIn != (fade == BeatFadeIn) {
			t.Fatalf("imported fade compatibility = (%d, %t)", fade, beat.Effect.FadeIn)
		}
	}

	song := semanticExportProbeSong(t)
	first := song.Tracks[0].Measures[0].Voices[0].Beats[0]
	beats := make([]Beat, len(values))
	for index, fade := range values {
		beats[index] = first
		beats[index].Notes = slices.Clone(first.Notes)
		beats[index].Effect.Fade = fade
		beats[index].Effect.FadeIn = fade == BeatFadeIn
	}
	song.Tracks[0].Measures[0].Voices[0].Beats = beats
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	before := append([]Beat(nil), song.Tracks[0].Measures[0].Voices[0].Beats...)
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("strict fade export = %v, %#v", err, report.Entries)
	}
	run.ClaimReport(claimSite("fade-other", "export", conformanceFadeCase, "none, fade-in, fade-out, and volume-swell values")).Report(conformanceFadeCase, reportCodes(report), []string{})
	run.ClaimSerialization(claimSite("fade-other", "export", conformanceFadeCase, "none, fade-in, fade-out, and volume-swell values")).Wire("gpifBeat.Fadding", conformanceFadeWire(t, data), []string{"FadeIn", "FadeOut", "VolumeSwell"})
	gpif := string(conformanceBarreGPIF(t, data))
	for _, version := range []string{"6.0", "7.0", "8.0"} {
		versioned := strings.Replace(gpif, "<GPVersion>8.0</GPVersion>", "<GPVersion>"+version+"</GPVersion>", 1)
		parsed, parseErr := ParseWithOptions(conformanceGPIFArchive(t, versioned), ParseOptions{Strict: true})
		if parseErr != nil {
			t.Fatalf("GP%s exact fade parse: %v", version, parseErr)
		}
		if got := conformanceBeatFadesIncludingNone(parsed.Song); !slices.Equal(got, values) {
			t.Fatalf("GP%s fades = %#v, want %#v", version, got, values)
		}
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	run.ClaimPrimary(claimSite("fade-other", "model", conformanceFadeCase, "none, fade-in, fade-out, and volume-swell values"), claimSite("fade-other", "export", conformanceFadeCase, "none, fade-in, fade-out, and volume-swell values")).Field("BeatEffects.Fade", conformanceBeatFadesIncludingNone(roundTrip), values)
	if !reflect.DeepEqual(song.Tracks[0].Measures[0].Voices[0].Beats, before) {
		t.Fatal("fade export mutated the public song")
	}

	conformanceBeatFadeReconciliation(t)
	conformanceBeatFadeDiagnostics(run)
}

func conformanceBeatFadeReconciliation(t *testing.T) {
	t.Helper()
	for _, test := range []struct {
		name       string
		imported   BeatFade
		edit       func(*BeatEffects)
		want       BeatFade
		wantReport bool
	}{
		{name: "typed-only edit", imported: BeatFadeOut, edit: func(effect *BeatEffects) { effect.Fade = BeatFadeVolumeSwell }, want: BeatFadeVolumeSwell},
		{name: "typed-only clear", imported: BeatFadeOut, edit: func(effect *BeatEffects) { effect.Fade = BeatFadeNone }, want: BeatFadeNone},
		{name: "legacy-only edit", imported: BeatFadeOut, edit: func(effect *BeatEffects) { effect.FadeIn = true }, want: BeatFadeIn},
		{name: "legacy-only clear", imported: BeatFadeIn, edit: func(effect *BeatEffects) { effect.FadeIn = false }, want: BeatFadeNone},
		{name: "compatible double edit", imported: BeatFadeOut, edit: func(effect *BeatEffects) { effect.Fade = BeatFadeIn; effect.FadeIn = true }, want: BeatFadeIn},
		{name: "conflicting double edit", imported: BeatFadeIn, edit: func(effect *BeatEffects) { effect.Fade = BeatFadeVolumeSwell; effect.FadeIn = false }, want: BeatFadeVolumeSwell, wantReport: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := semanticExportProbeSong(t)
			effect := &source.Tracks[0].Measures[0].Voices[0].Beats[0].Effect
			effect.Fade = test.imported
			effect.FadeIn = test.imported == BeatFadeIn
			data, err := Export(source, ExportFormatGP8)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			parsedEffect := &parsed.Tracks[0].Measures[0].Voices[0].Beats[0].Effect
			test.edit(parsedEffect)
			before := *parsedEffect
			output, report, err := ExportWithReport(parsed, ExportFormatGP8, ExportOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if hasExportCode(report, "gp8.normalize.fade-authority") != test.wantReport {
				t.Fatalf("authority report = %#v, want report %t", report.Entries, test.wantReport)
			}
			if got := conformanceFadeWire(t, output); !slices.Equal(got, conformanceBeatFadeExpectedWire(test.want)) {
				t.Fatalf("resolved fade wire = %v, want %d", got, test.want)
			}
			if *parsedEffect != before {
				t.Fatal("fade reconciliation mutated the public effect")
			}
			if test.wantReport {
				strictData, _, strictErr := ExportWithReport(parsed, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
				var loss *ExportLossError
				if len(strictData) != 0 || !errors.As(strictErr, &loss) {
					t.Fatalf("strict conflicting edit export = %d bytes, %v", len(strictData), strictErr)
				}
			}
		})
	}

	programmatic := BeatEffects{Fade: BeatFadeOut}
	if got, conflict := programmatic.resolvedFade(); got != BeatFadeOut || conflict {
		t.Fatalf("programmatic typed authority = (%d, %t)", got, conflict)
	}
	programmatic = BeatEffects{FadeIn: true}
	if got, conflict := programmatic.resolvedFade(); got != BeatFadeIn || conflict {
		t.Fatalf("programmatic legacy fallback = (%d, %t)", got, conflict)
	}
}

func conformanceBeatFadeDiagnostics(run *conformanceRun) {
	t := run.t
	context := &parseContext{format: "GP8"}
	gpifAuditDiagnostics(gpifDocument{Beats: gpifBeats{Beats: []gpifBeat{{ID: "beat", Fadding: "FutureFade"}}}}, context)
	run.Dispatch("gpifAuditDiagnostics:beat.Fadding", slices.ContainsFunc(context.diagnostics, func(diagnostic ParseDiagnostic) bool { return diagnostic.Code == "GPIF.Beat.Fadding.InvalidValue" }), true)

	invalid := semanticExportProbeSong(t)
	invalid.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Fade = BeatFade(4)
	if diagnostics := ValidateSong(invalid); !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == "score.beat.fade" }) {
		t.Fatalf("invalid beat fade diagnostics = %#v", diagnostics)
	}
	data, report, err := ExportWithReport(invalid, ExportFormatGP8, ExportOptions{})
	if len(data) != 0 || err == nil || !hasExportCode(report, "gp8.reject.score.beat.fade") {
		t.Fatalf("invalid beat fade export = %d bytes, %#v, %v", len(data), report.Entries, err)
	}

	valid := semanticExportProbeSong(t)
	valid.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Fade = BeatFadeOut
	validData, err := Export(valid, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(string(conformanceBarreGPIF(t, validData)), "<Fadding>FadeOut</Fadding>", "<Fadding>FutureFade</Fadding>", 1)
	result, parseErr := ParseWithOptions(conformanceGPIFArchive(t, mutated), ParseOptions{})
	if parseErr != nil || !slices.ContainsFunc(result.Diagnostics, func(diagnostic ParseDiagnostic) bool { return diagnostic.Code == "GPIF.Beat.Fadding.InvalidValue" }) {
		t.Fatalf("source diagnostics = %#v, %v", result.Diagnostics, parseErr)
	}
	_, strictErr := ParseWithOptions(conformanceGPIFArchive(t, mutated), ParseOptions{Strict: true})
	var strict *StrictParseError
	if !errors.As(strictErr, &strict) {
		t.Fatalf("strict source parse = %v, want StrictParseError", strictErr)
	}
}

func conformanceAllBeats(song *Song) []Beat {
	var result []Beat
	for trackIndex := range song.Tracks {
		for staffIndex := range song.Tracks[trackIndex].Staves {
			for measureIndex := range song.Tracks[trackIndex].Staves[staffIndex].Measures {
				for voiceIndex := range song.Tracks[trackIndex].Staves[staffIndex].Measures[measureIndex].Voices {
					result = append(result, song.Tracks[trackIndex].Staves[staffIndex].Measures[measureIndex].Voices[voiceIndex].Beats...)
				}
			}
		}
	}
	return result
}

func conformanceBeatFades(song *Song) []BeatFade {
	var result []BeatFade
	for _, beat := range conformanceAllBeats(song) {
		if fade, _ := beat.Effect.resolvedFade(); fade != BeatFadeNone {
			result = append(result, fade)
		}
	}
	return result
}

func conformanceBeatFadesIncludingNone(song *Song) []BeatFade {
	var result []BeatFade
	for _, beat := range conformanceAllBeats(song) {
		fade, _ := beat.Effect.resolvedFade()
		result = append(result, fade)
	}
	return result
}

func conformanceCountBeatFade(values []BeatFade, want BeatFade) int {
	return len(slices.DeleteFunc(slices.Clone(values), func(value BeatFade) bool { return value != want }))
}

func conformanceFadeWire(t *testing.T, data []byte) []string {
	t.Helper()
	document := conformanceWireDocument(t, data)
	var result []string
	for _, beat := range document.Beats.Beats {
		if beat.Fadding != "" {
			result = append(result, beat.Fadding)
		}
	}
	return result
}

func conformanceBeatFadeExpectedWire(value BeatFade) []string {
	if wire := gp8BeatFade(value); wire != "" {
		return []string{wire}
	}
	return nil
}

func TestAlphaTabPreservesBeatFade(t *testing.T) {
	requireAlphaTabConformance(t)
	fixture := "testdata/gp7/fade.gp"
	var source []conformanceBeatFadeFact
	readAlphaTabOracleFacts(t, "--beat-fade", fixture, &source)
	if len(source) != 4 || conformanceCountFadeFacts(source, "volumeswell") != 2 || conformanceCountFadeFacts(source, "fadeout") != 1 || conformanceCountFadeFacts(source, "fadein") != 1 {
		t.Fatalf("AlphaTab source fade facts = %#v", source)
	}
	song := parseTestFixture(t, fixture)
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var output []conformanceBeatFadeFact
	readAlphaTabOracleFacts(t, "--beat-fade", writeConformanceFixture(t, data), &output)
	if !reflect.DeepEqual(output, source) {
		t.Fatalf("AlphaTab output fade facts = %#v, want %#v", output, source)
	}

	editedBeat := &song.Tracks[0].Measures[1].Voices[0].Beats[1]
	if editedBeat.Effect.Fade != BeatFadeOut {
		t.Fatalf("edit source fade = %d, want FadeOut", editedBeat.Effect.Fade)
	}
	editedBeat.Effect.Fade = BeatFadeVolumeSwell
	edited, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var editedFacts []conformanceBeatFadeFact
	readAlphaTabOracleFacts(t, "--beat-fade", writeConformanceFixture(t, edited), &editedFacts)
	if conformanceCountFadeFacts(editedFacts, "volumeswell") != 3 || conformanceCountFadeFacts(editedFacts, "fadeout") != 0 || conformanceCountFadeFacts(editedFacts, "fadein") != 1 {
		t.Fatalf("AlphaTab edited fade facts = %#v", editedFacts)
	}
	conformanceIndependentClaim(t, "field:BeatEffects.Fade", claimAllStages("fade-other", conformanceFadeCase, "none, fade-in, fade-out, and volume-swell values")...)
}

func conformanceCountFadeFacts(facts []conformanceBeatFadeFact, want string) int {
	return len(slices.DeleteFunc(slices.Clone(facts), func(fact conformanceBeatFadeFact) bool { return fact.Fade != want }))
}
