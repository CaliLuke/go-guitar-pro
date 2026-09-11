// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

type conformanceBeatVibratoFact struct {
	Track   int    `json:"track"`
	Staff   int    `json:"staff"`
	Bar     int    `json:"bar"`
	Voice   int    `json:"voice"`
	Beat    int    `json:"beat"`
	Vibrato string `json:"vibrato"`
}

func TestConformanceBeatVibrato(t *testing.T) {
	runConformanceBeatVibrato(newConformanceRun(t))
}

func runConformanceBeatVibrato(run *conformanceRun) {
	t := run.t
	for index, value := range []BeatVibrato{BeatVibratoNone, BeatVibratoSlight, BeatVibratoWide} {
		run.Enum([]string{"BeatVibrato.BeatVibratoNone", "BeatVibrato.BeatVibratoSlight", "BeatVibrato.BeatVibratoWide"}[index], value, BeatVibrato(index))
	}

	for _, test := range []struct {
		fixture string
		format  string
	}{
		{fixture: "testdata/gp3/Effects.gp3", format: "GP3"},
		{fixture: "testdata/gp4/Effects.gp4", format: "GP4"},
		{fixture: "testdata/gp5/Effects.gp5", format: "GP5"},
	} {
		song := parseTestFixture(t, test.fixture)
		strengths := conformanceBeatVibratoStrengths(song)
		if len(strengths) == 0 {
			t.Fatalf("%s beat vibrato fixture has no authored marks", test.format)
		}
		for _, strength := range strengths {
			if strength != BeatVibratoSlight {
				t.Fatalf("%s binary beat vibrato = %d, want Slight", test.format, strength)
			}
		}
	}
	gp7 := parseTestFixture(t, "testdata/gp7/tremolo-vibrato.gp")
	gp7Strengths := conformanceBeatVibratoStrengths(gp7)
	if conformanceCountBeatVibrato(gp7Strengths, BeatVibratoSlight) != 2 || conformanceCountBeatVibrato(gp7Strengths, BeatVibratoWide) != 2 {
		t.Fatalf("GP7 beat-vibrato strengths = %#v, want two Slight and two Wide", gp7Strengths)
	}
	if noteStrengths := conformanceNoteVibratoStrengths(gp7); conformanceCountNoteVibrato(noteStrengths, NoteVibratoSlight) != 4 || conformanceCountNoteVibrato(noteStrengths, NoteVibratoWide) != 2 {
		t.Fatalf("GP7 note-vibrato strengths = %#v, want independent note values", noteStrengths)
	}

	for _, test := range []struct {
		wire string
		want BeatVibrato
	}{{wire: "Slight", want: BeatVibratoSlight}, {wire: "Wide", want: BeatVibratoWide}} {
		beat := Beat{}
		gpifApplyBeatEffects(&gpifBeat{Properties: gpifProperties{Properties: []gpifProperty{{Name: "VibratoWTremBar", Strength: &test.wire}}}}, &beat)
		run.ClaimPrimary(claimSite("beat-vibrato", "import", "M10-BEAT-VIBRATO", "none, slight, and wide strengths")).Preserved("BeatEffects.VibratoStrength", beat.Effect.VibratoStrength, test.want)
		run.Dispatch("gpifApplyBeatEffects:p.Strength", beat.Effect.VibratoStrength, test.want)
		if !beat.Effect.Vibrato {
			t.Fatalf("GPIF %s did not initialize legacy presence", test.wire)
		}
	}

	song := semanticExportProbeSong(t)
	first := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
	first.Effect.VibratoStrength = BeatVibratoSlight
	first.Notes[0].Effect.VibratoStrength = NoteVibratoWide
	second := *first
	second.Notes = slices.Clone(first.Notes)
	second.Effect.VibratoStrength = BeatVibratoWide
	second.Effect.Vibrato = true
	song.Tracks[0].Measures[0].Voices[0].Beats = append(song.Tracks[0].Measures[0].Voices[0].Beats, second)
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	beforeEffects := []BeatEffects{song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect, song.Tracks[0].Measures[0].Voices[0].Beats[1].Effect}
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(report.Entries) != 0 {
		t.Fatalf("strict beat vibrato export = %v, %#v", err, report.Entries)
	}
	run.ClaimReport(claimSite("beat-vibrato", "export", "M10-BEAT-VIBRATO", "none, slight, and wide strengths")).Report("M10-BEAT-VIBRATO", reportCodes(report), []string{})
	wire := conformanceBeatVibratoWire(t, data)
	run.ClaimSerialization(claimSite("beat-vibrato", "export", "M10-BEAT-VIBRATO", "none, slight, and wide strengths")).Wire("gpifProperty.Strength", wire, []string{"Slight", "Wide"})

	roundTrip, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	roundTripBeats := roundTrip.Tracks[0].Measures[0].Voices[0].Beats
	run.ClaimPrimary(claimSite("beat-vibrato", "model", "M10-BEAT-VIBRATO", "none, slight, and wide strengths"), claimSite("beat-vibrato", "export", "M10-BEAT-VIBRATO", "none, slight, and wide strengths")).Field("BeatEffects.VibratoStrength", []BeatVibrato{roundTripBeats[0].Effect.VibratoStrength, roundTripBeats[1].Effect.VibratoStrength}, []BeatVibrato{BeatVibratoSlight, BeatVibratoWide})
	if roundTripBeats[0].Notes[0].Effect.VibratoStrength != NoteVibratoWide {
		t.Fatalf("note vibrato was conflated with beat vibrato: %#v", roundTripBeats[0].Notes[0].Effect)
	}
	afterEffects := []BeatEffects{song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect, song.Tracks[0].Measures[0].Voices[0].Beats[1].Effect}
	if !reflect.DeepEqual(afterEffects, beforeEffects) {
		t.Fatal("beat vibrato export mutated the public song")
	}

	conformanceBeatVibratoReconciliation(t)
	conformanceBeatVibratoDiagnostics(run)
}

func conformanceBeatVibratoReconciliation(t *testing.T) {
	t.Helper()
	for _, test := range []struct {
		name       string
		imported   BeatVibrato
		edit       func(*BeatEffects)
		want       BeatVibrato
		wantReport bool
	}{
		{name: "typed-only clear", imported: BeatVibratoWide, edit: func(effect *BeatEffects) { effect.VibratoStrength = BeatVibratoNone }, want: BeatVibratoNone},
		{name: "legacy-only clear", imported: BeatVibratoWide, edit: func(effect *BeatEffects) { effect.Vibrato = false }, want: BeatVibratoNone},
		{name: "compatible double clear", imported: BeatVibratoWide, edit: func(effect *BeatEffects) { effect.VibratoStrength = BeatVibratoNone; effect.Vibrato = false }, want: BeatVibratoNone},
		{name: "conflicting double edit", imported: BeatVibratoSlight, edit: func(effect *BeatEffects) { effect.VibratoStrength = BeatVibratoWide; effect.Vibrato = false }, want: BeatVibratoWide, wantReport: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := semanticExportProbeSong(t)
			effect := &source.Tracks[0].Measures[0].Voices[0].Beats[0].Effect
			effect.VibratoStrength = test.imported
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
			if hasExportCode(report, "gp8.normalize.beat-vibrato-authority") != test.wantReport {
				t.Fatalf("authority report = %#v, want report %t", report.Entries, test.wantReport)
			}
			if got := conformanceBeatVibratoWire(t, output); !slices.Equal(got, conformanceBeatVibratoExpectedWire(test.want)) {
				t.Fatalf("resolved beat vibrato wire = %v, want %d", got, test.want)
			}
			if *parsedEffect != before {
				t.Fatal("reconciliation mutated the public effect")
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

	programmatic := BeatEffects{VibratoStrength: BeatVibratoWide, Vibrato: false}
	if got, conflict := programmatic.resolvedVibrato(); got != BeatVibratoWide || conflict {
		t.Fatalf("programmatic typed authority = (%d, %t)", got, conflict)
	}
	programmatic = BeatEffects{Vibrato: true}
	if got, conflict := programmatic.resolvedVibrato(); got != BeatVibratoSlight || conflict {
		t.Fatalf("programmatic legacy fallback = (%d, %t)", got, conflict)
	}
}

func conformanceBeatVibratoDiagnostics(run *conformanceRun) {
	t := run.t
	for _, test := range []struct {
		name     string
		property gpifProperty
		code     string
	}{
		{name: "missing strength", property: gpifProperty{Name: "VibratoWTremBar"}, code: "GPIF.Beat.Property.VibratoWTremBar.MissingStrength"},
		{name: "unknown strength", property: func() gpifProperty {
			value := "Extreme"
			return gpifProperty{Name: "VibratoWTremBar", Strength: &value}
		}(), code: "GPIF.Beat.Property.VibratoWTremBar.InvalidStrength"},
	} {
		t.Run(test.name, func(t *testing.T) {
			context := &parseContext{format: "GP8"}
			gpifAuditBeatProperty(context, "beat", "/GPIF/Beats/Beat", test.property)
			run.Dispatch("gpifAuditBeatProperty:property.Name", slices.ContainsFunc(context.diagnostics, func(diagnostic ParseDiagnostic) bool { return diagnostic.Code == test.code }), true)
		})
	}

	invalid := semanticExportProbeSong(t)
	invalid.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.VibratoStrength = BeatVibrato(3)
	if diagnostics := ValidateSong(invalid); !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == "score.beat.vibrato" }) {
		t.Fatalf("invalid beat vibrato diagnostics = %#v", diagnostics)
	}
	data, report, err := ExportWithReport(invalid, ExportFormatGP8, ExportOptions{})
	if len(data) != 0 || err == nil || !hasExportCode(report, "gp8.reject.score.beat.vibrato") {
		t.Fatalf("invalid beat vibrato export = %d bytes, %#v, %v", len(data), report.Entries, err)
	}

	wide := semanticExportProbeSong(t)
	wide.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.VibratoStrength = BeatVibratoWide
	validData, err := Export(wide, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	gpif := string(conformanceBarreGPIF(t, validData))
	for _, test := range []struct{ old, replacement, code string }{
		{old: "<Strength>Wide</Strength>", replacement: "", code: "GPIF.Beat.Property.VibratoWTremBar.MissingStrength"},
		{old: "<Strength>Wide</Strength>", replacement: "<Strength>Extreme</Strength>", code: "GPIF.Beat.Property.VibratoWTremBar.InvalidStrength"},
	} {
		mutated := strings.Replace(gpif, test.old, test.replacement, 1)
		result, parseErr := ParseWithOptions(conformanceGPIFArchive(t, mutated), ParseOptions{})
		if parseErr != nil || !slices.ContainsFunc(result.Diagnostics, func(diagnostic ParseDiagnostic) bool { return diagnostic.Code == test.code }) {
			t.Fatalf("source diagnostics = %#v, %v; want %s", result.Diagnostics, parseErr, test.code)
		}
		_, strictErr := ParseWithOptions(conformanceGPIFArchive(t, mutated), ParseOptions{Strict: true})
		var strict *StrictParseError
		if !errors.As(strictErr, &strict) {
			t.Fatalf("strict source parse = %v, want StrictParseError", strictErr)
		}
	}
}

func conformanceBeatVibratoStrengths(song *Song) []BeatVibrato {
	var result []BeatVibrato
	for trackIndex := range song.Tracks {
		for staffIndex := range song.Tracks[trackIndex].Staves {
			for measureIndex := range song.Tracks[trackIndex].Staves[staffIndex].Measures {
				for voiceIndex := range song.Tracks[trackIndex].Staves[staffIndex].Measures[measureIndex].Voices {
					for _, beat := range song.Tracks[trackIndex].Staves[staffIndex].Measures[measureIndex].Voices[voiceIndex].Beats {
						if strength, _ := beat.Effect.resolvedVibrato(); strength != BeatVibratoNone {
							result = append(result, strength)
						}
					}
				}
			}
		}
	}
	return result
}

func conformanceNoteVibratoStrengths(song *Song) []NoteVibrato {
	var result []NoteVibrato
	for trackIndex := range song.Tracks {
		for staffIndex := range song.Tracks[trackIndex].Staves {
			for measureIndex := range song.Tracks[trackIndex].Staves[staffIndex].Measures {
				for voiceIndex := range song.Tracks[trackIndex].Staves[staffIndex].Measures[measureIndex].Voices {
					for _, beat := range song.Tracks[trackIndex].Staves[staffIndex].Measures[measureIndex].Voices[voiceIndex].Beats {
						for _, note := range beat.Notes {
							if note.Effect.VibratoStrength != NoteVibratoNone {
								result = append(result, note.Effect.VibratoStrength)
							}
						}
					}
				}
			}
		}
	}
	return result
}

func conformanceCountBeatVibrato(values []BeatVibrato, want BeatVibrato) int {
	count := 0
	for _, value := range values {
		if value == want {
			count++
		}
	}
	return count
}

func conformanceCountNoteVibrato(values []NoteVibrato, want NoteVibrato) int {
	count := 0
	for _, value := range values {
		if value == want {
			count++
		}
	}
	return count
}

func conformanceBeatVibratoWire(t *testing.T, data []byte) []string {
	t.Helper()
	document := conformanceWireDocument(t, data)
	var result []string
	for _, beat := range document.Beats.Beats {
		for _, property := range beat.Properties.Properties {
			if property.Name == "VibratoWTremBar" && property.Strength != nil {
				result = append(result, *property.Strength)
			}
		}
	}
	return result
}

func conformanceBeatVibratoExpectedWire(value BeatVibrato) []string {
	if wire := gp8BeatVibratoStrength(value); wire != "" {
		return []string{wire}
	}
	return nil
}

func TestAlphaTabPreservesBeatVibrato(t *testing.T) {
	requireAlphaTabConformance(t)
	mixedFixture := "testdata/gp7/tremolo-vibrato.gp"
	var mixedSource []conformanceBeatVibratoFact
	readAlphaTabOracleFacts(t, "--beat-vibrato", mixedFixture, &mixedSource)
	if len(mixedSource) != 4 || conformanceCountFactVibrato(mixedSource, "slight") != 2 || conformanceCountFactVibrato(mixedSource, "wide") != 2 {
		t.Fatalf("AlphaTab mixed source beat-vibrato facts = %#v", mixedSource)
	}
	mixedSong := parseTestFixture(t, mixedFixture)
	mixedData, err := Export(mixedSong, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var mixedOutput []conformanceBeatVibratoFact
	readAlphaTabOracleFacts(t, "--beat-vibrato", writeConformanceFixture(t, mixedData), &mixedOutput)
	if !reflect.DeepEqual(mixedOutput, mixedSource) {
		t.Fatalf("AlphaTab mixed output beat-vibrato facts = %#v, want %#v", mixedOutput, mixedSource)
	}

	fixture := "testdata/gp7/faulty.gp"
	var source []conformanceBeatVibratoFact
	readAlphaTabOracleFacts(t, "--beat-vibrato", fixture, &source)
	if len(source) != 50 {
		t.Fatalf("AlphaTab source wide beat-vibrato facts = %d, want 50", len(source))
	}
	if slices.ContainsFunc(source, func(fact conformanceBeatVibratoFact) bool { return fact.Vibrato != "wide" }) {
		t.Fatalf("AlphaTab source beat-vibrato facts include a non-wide value: %#v", source)
	}
	song := parseTestFixture(t, fixture)
	if got := conformanceBeatVibratoStrengths(song); len(got) != 50 || slices.ContainsFunc(got, func(value BeatVibrato) bool { return value != BeatVibratoWide }) {
		t.Fatalf("Go source beat-vibrato strengths = %#v", got)
	}
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var output []conformanceBeatVibratoFact
	readAlphaTabOracleFacts(t, "--beat-vibrato", writeConformanceFixture(t, data), &output)
	if !reflect.DeepEqual(output, source) {
		t.Fatalf("AlphaTab output beat-vibrato facts = %#v, want %#v", output, source)
	}
	conformanceIndependentClaim(t, "field:BeatEffects.VibratoStrength",
		claimSite("beat-vibrato", "import", "M10-BEAT-VIBRATO", "none, slight, and wide strengths"),
		claimSite("beat-vibrato", "model", "M10-BEAT-VIBRATO", "none, slight, and wide strengths"),
		claimSite("beat-vibrato", "export", "M10-BEAT-VIBRATO", "none, slight, and wide strengths"))
}

func conformanceCountFactVibrato(facts []conformanceBeatVibratoFact, want string) int {
	count := 0
	for _, fact := range facts {
		if fact.Vibrato == want {
			count++
		}
	}
	return count
}
