// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const (
	conformanceGolpeCase  = "M10-GOLPE"
	conformanceGolpeValue = "none, thumb, and finger variants"
)

type conformanceGolpeFact struct {
	Track     int    `json:"track"`
	Staff     int    `json:"staff"`
	Bar       int    `json:"bar"`
	Voice     int    `json:"voice"`
	Beat      int    `json:"beat"`
	Golpe     string `json:"golpe"`
	NoteCount int    `json:"noteCount"`
}

func TestConformanceGolpe(t *testing.T) {
	runConformanceGolpe(newConformanceRun(t))
}

func runConformanceGolpe(run *conformanceRun) {
	t := run.t
	values := []GolpeType{GolpeTypeNone, GolpeTypeThumb, GolpeTypeFinger}
	for index, name := range []string{"GolpeType.GolpeTypeNone", "GolpeType.GolpeTypeThumb", "GolpeType.GolpeTypeFinger"} {
		run.Enum(name, values[index], GolpeType(index))
	}

	fixture := parseTestFixture(t, "testdata/gp7/golpe.gp")
	golpes := conformanceGolpes(fixture)
	if len(golpes) != 24 || conformanceCountGolpe(golpes, GolpeTypeThumb) != 12 || conformanceCountGolpe(golpes, GolpeTypeFinger) != 12 {
		t.Fatalf("GP7 golpes = %#v, want 12 Thumb and 12 Finger values", golpes)
	}
	expectedFixture := append(slices.Repeat([]GolpeType{GolpeTypeFinger}, 8), slices.Repeat([]GolpeType{GolpeTypeThumb}, 8)...)
	expectedFixture = append(expectedFixture, slices.Repeat([]GolpeType{GolpeTypeFinger}, 4)...)
	expectedFixture = append(expectedFixture, slices.Repeat([]GolpeType{GolpeTypeThumb}, 4)...)
	run.ClaimPrimary(claimSite("golpe", "import", conformanceGolpeCase, conformanceGolpeValue)).Preserved("BeatEffects.Golpe", golpes, expectedFixture)

	for _, binaryFixture := range []string{"testdata/gp3/Effects.gp3", "testdata/gp4/Effects.gp4", "testdata/gp5/Effects.gp5"} {
		if _, err := os.Stat(binaryFixture); err != nil {
			t.Fatal(err)
		}
		if got := conformanceGolpes(parseTestFixture(t, binaryFixture)); len(got) != 0 {
			t.Fatalf("%s synthesized %d golpe values", binaryFixture, len(got))
		}
	}

	song := semanticExportProbeSong(t)
	base := song.Tracks[0].Measures[0].Voices[0].Beats[0]
	beats := make([]Beat, len(values))
	for index, value := range values {
		beats[index] = base
		beats[index].Notes = slices.Clone(base.Notes)
		beats[index].Effect.Golpe = value
	}
	beats[2].Notes = append(beats[2].Notes, Note{Value: 5, String: 2, DurationPercent: 1, Kind: NoteTypeNormal, Velocity: Forte})
	beats[2].Effect.Fade = BeatFadeOut
	song.Tracks[0].Measures[0].Voices[0].Beats = beats
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	before := slices.Clone(song.Tracks[0].Measures[0].Voices[0].Beats)
	for index := range before {
		before[index].Notes = slices.Clone(before[index].Notes)
	}
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("strict golpe export = %v, %#v", err, report.Entries)
	}
	run.ClaimReport(claimSite("golpe", "export", conformanceGolpeCase, conformanceGolpeValue)).Report(conformanceGolpeCase, reportCodes(report), []string{})
	wire := conformanceWireDocument(t, data).Beats.Beats
	if len(wire) != 3 {
		t.Fatalf("wire beats = %d, want 3", len(wire))
	}
	run.ClaimSerialization(claimSite("golpe", "export", conformanceGolpeCase, conformanceGolpeValue)).Wire("gpifBeat.Golpe", []string{wire[0].Golpe, wire[1].Golpe, wire[2].Golpe}, []string{"", "Thumb", "Finger"})
	if len(strings.Fields(wire[2].Notes)) != 2 || wire[2].Fadding != "FadeOut" {
		t.Fatalf("simultaneous-note golpe wire = notes %q, fade %q", wire[2].Notes, wire[2].Fadding)
	}
	for _, version := range []string{"6.0", "7.0", "8.0"} {
		gpif := strings.Replace(string(conformanceBarreGPIF(t, data)), "<GPVersion>8.0</GPVersion>", "<GPVersion>"+version+"</GPVersion>", 1)
		parsed, parseErr := ParseWithOptions(conformanceGPIFArchive(t, gpif), ParseOptions{Strict: true})
		if parseErr != nil {
			t.Fatalf("GP%s exact golpe parse: %v", version, parseErr)
		}
		if got := conformanceGolpesIncludingNone(parsed.Song); !slices.Equal(got, values) {
			t.Fatalf("GP%s golpes = %#v, want %#v", version, got, values)
		}
	}
	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	roundTripBeats := roundTrip.Tracks[0].Measures[0].Voices[0].Beats
	run.ClaimPrimary(claimSite("golpe", "model", conformanceGolpeCase, conformanceGolpeValue), claimSite("golpe", "export", conformanceGolpeCase, conformanceGolpeValue)).Field("BeatEffects.Golpe", conformanceGolpesIncludingNone(roundTrip), values)
	if len(roundTripBeats[2].Notes) != 2 || roundTripBeats[2].Effect.Fade != BeatFadeOut {
		t.Fatalf("round-trip simultaneous-note golpe = %d notes, fade %d", len(roundTripBeats[2].Notes), roundTripBeats[2].Effect.Fade)
	}
	if !reflect.DeepEqual(song.Tracks[0].Measures[0].Voices[0].Beats, before) {
		t.Fatal("golpe preflight or export mutated the public song")
	}

	conformanceGolpeOccurrenceIsolation(t)
	conformanceGolpeDiagnostics(t)
}

func conformanceGolpeOccurrenceIsolation(t *testing.T) {
	t.Helper()
	song := semanticExportProbeSong(t)
	song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Golpe = GolpeTypeThumb
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	gpif := strings.Replace(string(conformanceBarreGPIF(t, data)), "<Beats>0</Beats>", "<Beats>0 0</Beats>", 1)
	if !strings.Contains(gpif, "<Beats>0 0</Beats>") {
		t.Fatal("could not reuse the exported beat definition")
	}
	parsed, err := Parse(conformanceGPIFArchive(t, gpif))
	if err != nil {
		t.Fatal(err)
	}
	beats := parsed.Tracks[0].Measures[0].Voices[0].Beats
	if len(beats) != 2 || beats[0].Effect.Golpe != GolpeTypeThumb || beats[1].Effect.Golpe != GolpeTypeThumb {
		t.Fatalf("reused golpe occurrences = %#v", beats)
	}
	beats[0].Effect.Golpe = GolpeTypeFinger
	if beats[1].Effect.Golpe != GolpeTypeThumb {
		t.Fatal("editing one reused occurrence changed another")
	}
}

func conformanceGolpeDiagnostics(t *testing.T) {
	t.Helper()
	invalid := semanticExportProbeSong(t)
	invalid.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Golpe = GolpeType(3)
	if diagnostics := ValidateSong(invalid); !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == "score.beat.golpe" }) {
		t.Fatalf("invalid golpe diagnostics = %#v", diagnostics)
	}
	data, report, err := ExportWithReport(invalid, ExportFormatGP8, ExportOptions{})
	if len(data) != 0 || err == nil || !hasExportCode(report, "gp8.reject.score.beat.golpe") {
		t.Fatalf("invalid golpe export = %d bytes, %#v, %v", len(data), report.Entries, err)
	}

	valid := semanticExportProbeSong(t)
	valid.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Golpe = GolpeTypeFinger
	validData, err := Export(valid, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(string(conformanceBarreGPIF(t, validData)), "<Golpe>Finger</Golpe>", "<Golpe>FutureGolpe</Golpe>", 1)
	result, parseErr := ParseWithOptions(conformanceGPIFArchive(t, mutated), ParseOptions{})
	if parseErr != nil || !slices.ContainsFunc(result.Diagnostics, func(diagnostic ParseDiagnostic) bool {
		return diagnostic.Code == "GPIF.Beat.Golpe.InvalidValue" && diagnostic.Feature == "golpe" && strings.HasSuffix(diagnostic.SourcePath, "/Golpe")
	}) {
		t.Fatalf("source golpe diagnostics = %#v, %v", result.Diagnostics, parseErr)
	}
	if got := result.Song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.Golpe; got != GolpeTypeNone {
		t.Fatalf("unknown golpe imported as %d, want None", got)
	}
	_, strictErr := ParseWithOptions(conformanceGPIFArchive(t, mutated), ParseOptions{Strict: true})
	var strict *StrictParseError
	if !errors.As(strictErr, &strict) {
		t.Fatalf("strict source parse = %v, want StrictParseError", strictErr)
	}
}

func conformanceGolpes(song *Song) []GolpeType {
	var result []GolpeType
	for _, beat := range conformanceAllBeats(song) {
		if beat.Effect.Golpe != GolpeTypeNone {
			result = append(result, beat.Effect.Golpe)
		}
	}
	return result
}

func conformanceGolpesIncludingNone(song *Song) []GolpeType {
	var result []GolpeType
	for _, beat := range conformanceAllBeats(song) {
		result = append(result, beat.Effect.Golpe)
	}
	return result
}

func conformanceCountGolpe(values []GolpeType, want GolpeType) int {
	return len(slices.DeleteFunc(slices.Clone(values), func(value GolpeType) bool { return value != want }))
}

func TestAlphaTabPreservesGolpe(t *testing.T) {
	requireAlphaTabConformance(t)
	fixture := "testdata/gp7/golpe.gp"
	var source []conformanceGolpeFact
	readAlphaTabOracleFacts(t, "--golpe", fixture, &source)
	if len(source) != 24 || conformanceCountGolpeFacts(source, "finger") != 12 || conformanceCountGolpeFacts(source, "thumb") != 12 {
		t.Fatalf("AlphaTab source golpe facts = %#v", source)
	}
	song := parseTestFixture(t, fixture)
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var output []conformanceGolpeFact
	readAlphaTabOracleFacts(t, "--golpe", writeConformanceFixture(t, data), &output)
	if !reflect.DeepEqual(output, source) {
		t.Fatalf("AlphaTab output golpe facts = %#v, want %#v", output, source)
	}

	editedBeat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	if editedBeat.Effect.Golpe != GolpeTypeFinger {
		t.Fatalf("edit source golpe = %d, want Finger", editedBeat.Effect.Golpe)
	}
	editedBeat.Effect.Golpe = GolpeTypeThumb
	edited, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var editedFacts []conformanceGolpeFact
	readAlphaTabOracleFacts(t, "--golpe", writeConformanceFixture(t, edited), &editedFacts)
	if conformanceCountGolpeFacts(editedFacts, "finger") != 11 || conformanceCountGolpeFacts(editedFacts, "thumb") != 13 {
		t.Fatalf("AlphaTab edited golpe facts = %#v", editedFacts)
	}
	if editedFacts[0].Track != source[0].Track || editedFacts[0].Staff != source[0].Staff || editedFacts[0].Bar != source[0].Bar || editedFacts[0].Voice != source[0].Voice || editedFacts[0].Beat != source[0].Beat {
		t.Fatalf("edited golpe location = %#v, want %#v", editedFacts[0], source[0])
	}
	conformanceIndependentClaim(t, "field:BeatEffects.Golpe", claimAllStages("golpe", conformanceGolpeCase, conformanceGolpeValue)...)
}

func conformanceCountGolpeFacts(facts []conformanceGolpeFact, want string) int {
	return len(slices.DeleteFunc(slices.Clone(facts), func(fact conformanceGolpeFact) bool { return fact.Golpe != want }))
}
