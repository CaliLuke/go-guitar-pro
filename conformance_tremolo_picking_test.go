// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"encoding/json"
	"errors"
	"os/exec"
	"reflect"
	"slices"
	"testing"
)

type conformanceTremoloPickingFact struct {
	Track int    `json:"track"`
	Staff int    `json:"staff"`
	Bar   int    `json:"bar"`
	Voice int    `json:"voice"`
	Beat  int    `json:"beat"`
	Marks int    `json:"marks"`
	Style string `json:"style"`
}

func TestConformanceTremoloPicking(t *testing.T) {
	runConformanceTremoloPicking(newConformanceRun(t))
}

func runConformanceTremoloPicking(run *conformanceRun) {
	t := run.t

	// GP4 stores 1, 2, and 3 as the exact mark counts. This fixture catches
	// the historical swap of the 2- and 3-mark binary values.
	binary := parseTestFixture(t, "testdata/gp4/Effects.gp4")
	gotBinary := conformanceTremoloPickingDurations(binary)
	wantBinary := []uint16{uint16(DurationThirtySecond), uint16(DurationSixteenth), uint16(DurationEighth)}
	if !slices.Equal(gotBinary, wantBinary) {
		t.Fatalf("GP4 tremolo-picking durations = %v, want %v", gotBinary, wantBinary)
	}
	for _, source := range []struct {
		raw  byte
		want uint16
	}{{1, uint16(DurationEighth)}, {2, uint16(DurationSixteenth)}, {3, uint16(DurationThirtySecond)}} {
		effect, err := (&Song{}).readTremoloPicking(newCursor([]byte{source.raw}))
		if err != nil {
			t.Fatal(err)
		}
		run.Dispatch("readTremoloPicking:val", effect.Duration.Value, source.want)
	}
	for _, fixture := range []string{"testdata/gp5/Effects.gp5", "testdata/gp6/effects.gpx"} {
		if got := conformanceTremoloPickingDurations(parseTestFixture(t, fixture)); !slices.Equal(got, wantBinary) {
			t.Fatalf("%s tremolo-picking durations = %v, want %v", fixture, got, wantBinary)
		}
	}
	if got := conformanceTremoloPickingDurations(parseTestFixture(t, "testdata/gp3/Effects.gp3")); len(got) != 0 {
		t.Fatalf("GP3 tremolo-picking source records = %v, want none", got)
	}

	gpifSong := parseTestFixture(t, "testdata/gp7/tremolo-picking.gp")
	gotGPIF := conformanceTremoloPickingDurations(gpifSong)
	wantGPIF := []uint16{32, 16, 8, 32, 16, 8}
	if !slices.Equal(gotGPIF, wantGPIF) {
		t.Fatalf("GPIF tremolo-picking durations = %v, want %v", gotGPIF, wantGPIF)
	}
	first := &gpifSong.Tracks[0].Measures[0].Voices[0].Beats[0]
	second := &gpifSong.Tracks[0].Measures[0].Voices[0].Beats[1]
	if first.Effect.TremoloPicking == nil || len(first.Notes) == 0 || first.Notes[0].Effect.TremoloPicking == nil {
		t.Fatal("parsed GPIF tremolo-picking authority or compatibility field is absent")
	}
	first.Effect.TremoloPicking.Duration.Value = uint16(DurationEighth)
	if second.Effect.TremoloPicking.Duration.Value != uint16(DurationSixteenth) {
		t.Fatalf("editing one parsed occurrence changed the next to %d", second.Effect.TremoloPicking.Duration.Value)
	}

	for index, value := range []uint16{uint16(DurationEighth), uint16(DurationSixteenth), uint16(DurationThirtySecond)} {
		song := semanticExportProbeSong(t)
		beat := &song.Tracks[0].Measures[0].Voices[0].Beats[0]
		duration := defaultDuration()
		duration.Value = value
		beat.Effect.TremoloPicking = &TremoloPickingEffect{Duration: duration, Style: TremoloPickingStyleDefault}
		data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
		if err != nil || len(report.Entries) != 0 {
			t.Fatalf("supported tremolo duration %d export = %v, %#v", value, err, report.Entries)
		}
		wantWire := []string{"1/2", "1/4", "1/8"}[index]
		gotWire := extractGPIFLeafText(t, data)["GPIF/Beats/Beat/Tremolo"]
		if gotWire != wantWire {
			t.Fatalf("tremolo duration %d wire = %q, want %q", value, gotWire, wantWire)
		}
		run.Wire("gpifBeat.Tremolo", gotWire, wantWire)
		roundTrip, err := Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		gotBeat := &roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0]
		if gotBeat.Effect.TremoloPicking == nil || gotBeat.Effect.TremoloPicking.Duration.Value != value {
			t.Fatalf("tremolo duration %d round trip = %#v", value, gotBeat.Effect.TremoloPicking)
		}
		if gotBeat.Notes[0].Effect.TremoloPicking == nil || gotBeat.Notes[0].Effect.TremoloPicking.Duration.Value != value {
			t.Fatalf("legacy note tremolo duration %d round trip = %#v", value, gotBeat.Notes[0].Effect.TremoloPicking)
		}
		run.ClaimPrimary(claimSite("tremolo", "import", "M11-TREMOLO-PICKING", "binary marks 1, 2, and 3")).Preserved("BeatEffects.TremoloPicking", gotBeat.Effect.TremoloPicking.Duration.Value, value)
		run.Preserved("TremoloPickingEffect.Duration", gotBeat.Effect.TremoloPicking.Duration, duration)
		run.Omitted("TremoloPickingEffect.Style", beat.Effect.TremoloPicking.Style, TremoloPickingStyleDefault)
	}
	run.Enum("TremoloPickingStyle.TremoloPickingStyleDefault", TremoloPickingStyleDefault, TremoloPickingStyle(0))

	legacy := semanticExportProbeSong(t)
	legacyBeat := &legacy.Tracks[0].Measures[0].Voices[0].Beats[0]
	legacyDuration := defaultDuration()
	legacyDuration.Value = uint16(DurationSixteenth)
	legacyBeat.Notes[0].Effect.TremoloPicking = &TremoloPickingEffect{Duration: legacyDuration}
	legacyData, legacyReport, legacyErr := ExportWithReport(legacy, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if legacyErr != nil || len(legacyReport.Entries) != 0 || extractGPIFLeafText(t, legacyData)["GPIF/Beats/Beat/Tremolo"] != "1/4" {
		t.Fatalf("legacy tremolo fallback export = %v, %#v", legacyErr, legacyReport.Entries)
	}
	run.Normalized("NoteEffect.TremoloPicking", legacyBeat.Notes[0].Effect.TremoloPicking.Duration.Value, uint16(DurationSixteenth))

	conflict := semanticExportProbeSong(t)
	conflictBeat := &conflict.Tracks[0].Measures[0].Voices[0].Beats[0]
	conflictBeat.Effect.TremoloPicking = &TremoloPickingEffect{Duration: conformanceTremoloDuration(DurationEighth)}
	legacyConflictDuration := conformanceTremoloDuration(DurationEighth)
	legacyConflictDuration.Dotted = true
	conflictBeat.Notes[0].Effect.TremoloPicking = &TremoloPickingEffect{Duration: legacyConflictDuration}
	conflictData, conflictReport, conflictErr := ExportWithReport(conflict, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	var conflictLoss *ExportLossError
	if len(conflictData) != 0 || !errors.As(conflictErr, &conflictLoss) || !hasExportCode(conflictReport, "gp8.normalize.tremolo-picking-authority") {
		t.Fatalf("conflicting tremolo authority = %d bytes, %#v, %v", len(conflictData), conflictReport.Entries, conflictErr)
	}

	for _, value := range []uint8{DurationQuarter, DurationSixtyFourth, DurationHundredTwentyEighth} {
		unsupportedRate := semanticExportProbeSong(t)
		unsupportedRate.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.TremoloPicking = &TremoloPickingEffect{Duration: conformanceTremoloDuration(value)}
		rateData, rateReport, rateErr := ExportWithReport(unsupportedRate, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
		var rateLoss *ExportLossError
		if len(rateData) != 0 || !errors.As(rateErr, &rateLoss) || !hasExportCode(rateReport, "gp8.omit.tremolo-picking-rate") {
			t.Fatalf("unsupported tremolo rate %d = %d bytes, %#v, %v", value, len(rateData), rateReport.Entries, rateErr)
		}
	}

	for _, decorated := range []struct {
		name     string
		duration Duration
	}{
		{name: "dotted", duration: func() Duration {
			value := conformanceTremoloDuration(DurationEighth)
			value.Dotted = true
			return value
		}()},
		{name: "double-dotted", duration: func() Duration {
			value := conformanceTremoloDuration(DurationSixteenth)
			value.DoubleDotted = true
			return value
		}()},
		{name: "tuplet", duration: func() Duration {
			value := conformanceTremoloDuration(DurationThirtySecond)
			value.TupletEnters, value.TupletTimes = 3, 2
			return value
		}()},
	} {
		t.Run("decorated "+decorated.name, func(t *testing.T) {
			song := semanticExportProbeSong(t)
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.TremoloPicking = &TremoloPickingEffect{Duration: decorated.duration}
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
			if err != nil || !hasExportCode(report, "gp8.omit.tremolo-picking-rate") {
				t.Fatalf("best-effort decorated tremolo = %#v, %v", report.Entries, err)
			}
			if got := extractGPIFLeafText(t, data)["GPIF/Beats/Beat/Tremolo"]; got != "" {
				t.Fatalf("decorated tremolo wire = %q, want omitted", got)
			}
			strictData, strictReport, strictErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
			var loss *ExportLossError
			if len(strictData) != 0 || !errors.As(strictErr, &loss) || !hasExportCode(strictReport, "gp8.omit.tremolo-picking-rate") {
				t.Fatalf("strict decorated tremolo = %d bytes, %#v, %v", len(strictData), strictReport.Entries, strictErr)
			}
		})
	}

	unsupportedStyle := semanticExportProbeSong(t)
	styleBeat := &unsupportedStyle.Tracks[0].Measures[0].Voices[0].Beats[0]
	styleBeat.Effect.TremoloPicking = &TremoloPickingEffect{Duration: conformanceTremoloDuration(DurationEighth), Style: TremoloPickingStyleBuzzRoll}
	styleData, styleReport, styleErr := ExportWithReport(unsupportedStyle, ExportFormatGP8, ExportOptions{})
	if styleErr != nil || !hasExportCode(styleReport, "gp8.omit.tremolo-picking-style") {
		t.Fatalf("buzz-roll tremolo report = %#v, %v", styleReport.Entries, styleErr)
	}
	if got := extractGPIFLeafText(t, styleData)["GPIF/Beats/Beat/Tremolo"]; got != "1/2" {
		t.Fatalf("buzz-roll rate wire = %q, want 1/2", got)
	}
	strictStyleData, strictStyleReport, strictStyleErr := ExportWithReport(unsupportedStyle, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	var styleLoss *ExportLossError
	if len(strictStyleData) != 0 || !errors.As(strictStyleErr, &styleLoss) || !hasExportCode(strictStyleReport, "gp8.omit.tremolo-picking-style") {
		t.Fatalf("strict buzz-roll export = %d bytes, %#v, %v", len(strictStyleData), strictStyleReport.Entries, strictStyleErr)
	}
	run.Enum("TremoloPickingStyle.TremoloPickingStyleBuzzRoll", styleBeat.Effect.TremoloPicking.Style, TremoloPickingStyleBuzzRoll)
	run.Omitted("TremoloPickingEffect.Style", styleBeat.Effect.TremoloPicking.Style, TremoloPickingStyleBuzzRoll)

	for _, invalid := range []struct {
		name string
		set  func(*TremoloPickingEffect)
		code string
	}{
		{name: "duration", set: func(effect *TremoloPickingEffect) { effect.Duration = Duration{} }, code: "score.beat.tremolo-picking-duration"},
		{name: "style", set: func(effect *TremoloPickingEffect) { effect.Style = TremoloPickingStyle(99) }, code: "score.beat.tremolo-picking-style"},
	} {
		t.Run("invalid "+invalid.name, func(t *testing.T) {
			song := semanticExportProbeSong(t)
			effect := &TremoloPickingEffect{Duration: conformanceTremoloDuration(DurationEighth)}
			invalid.set(effect)
			song.Tracks[0].Measures[0].Voices[0].Beats[0].Effect.TremoloPicking = effect
			diagnostics := ValidateSong(song)
			if !slices.ContainsFunc(diagnostics, func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == invalid.code }) {
				t.Fatalf("invalid tremolo diagnostics = %#v, want %s", diagnostics, invalid.code)
			}
		})
	}
}

func conformanceTremoloDuration(value uint8) Duration {
	duration := defaultDuration()
	duration.Value = uint16(value)
	return duration
}

func conformanceTremoloPickingDurations(song *Song) []uint16 {
	var result []uint16
	for trackIndex := range song.Tracks {
		for staffIndex := range song.Tracks[trackIndex].Staves {
			for measureIndex := range song.Tracks[trackIndex].Staves[staffIndex].Measures {
				measure := &song.Tracks[trackIndex].Staves[staffIndex].Measures[measureIndex]
				for voiceIndex := range measure.Voices {
					for beatIndex := range measure.Voices[voiceIndex].Beats {
						effect, _ := measure.Voices[voiceIndex].Beats[beatIndex].resolvedTremoloPicking()
						if effect != nil {
							result = append(result, effect.Duration.Value)
						}
					}
				}
			}
		}
	}
	return result
}

func TestAlphaTabPreservesTremoloPicking(t *testing.T) {
	requireAlphaTabConformance(t)
	var source []conformanceTremoloPickingFact
	readAlphaTabOracleFacts(t, "--tremolo-picking", "testdata/gp4/Effects.gp4", &source)
	want := []conformanceTremoloPickingFact{
		{Track: 0, Staff: 0, Bar: 16, Voice: 0, Beat: 1, Marks: 3, Style: "default"},
		{Track: 0, Staff: 0, Bar: 16, Voice: 0, Beat: 2, Marks: 2, Style: "default"},
		{Track: 0, Staff: 0, Bar: 16, Voice: 0, Beat: 3, Marks: 1, Style: "default"},
	}
	if !reflect.DeepEqual(source, want) {
		t.Fatalf("AlphaTab GP4 tremolo facts = %#v, want %#v", source, want)
	}

	song := parseTestFixture(t, "testdata/gp4/Effects.gp4")
	data, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	var output []conformanceTremoloPickingFact
	readAlphaTabOracleFacts(t, "--tremolo-picking", writeConformanceFixture(t, data), &output)
	if !reflect.DeepEqual(output, source) {
		t.Fatalf("AlphaTab Go-output tremolo facts = %#v, want source %#v", output, source)
	}

	// A post-parse edit to the beat-wide authority controls the independent
	// consumer even while the compatibility note retains its parsed value.
	editedBeat := &song.Tracks[0].Measures[16].Voices[0].Beats[1]
	editedBeat.Effect.TremoloPicking.Duration = conformanceTremoloDuration(DurationEighth)
	edited, err := Export(song, ExportFormatGP8)
	if err != nil {
		t.Fatal(err)
	}
	readAlphaTabOracleFacts(t, "--tremolo-picking", writeConformanceFixture(t, edited), &output)
	if len(output) != len(source) || output[0].Marks != 1 || !reflect.DeepEqual(output[1:], source[1:]) {
		t.Fatalf("AlphaTab edited tremolo facts = %#v, want first mark 1 and unchanged remainder", output)
	}

	var model struct {
		MinMarks     int    `json:"minMarks"`
		MaxMarks     int    `json:"maxMarks"`
		DefaultMarks int    `json:"defaultMarks"`
		DefaultStyle string `json:"defaultStyle"`
		BuzzRoll     string `json:"buzzRollStyle"`
	}
	command := exec.Command("node", alphaTabOracleScript(), "--tremolo-picking-model")
	modelData, commandErr := command.CombinedOutput()
	if commandErr != nil {
		t.Fatalf("AlphaTab tremolo model oracle: %v\n%s", commandErr, modelData)
	}
	if err := json.Unmarshal(modelData, &model); err != nil {
		t.Fatal(err)
	}
	if model.MinMarks != 0 || model.MaxMarks != 5 || model.DefaultMarks != 0 || model.DefaultStyle != "default" || model.BuzzRoll != "buzzroll" {
		t.Fatalf("AlphaTab tremolo model facts = %#v", model)
	}
	conformanceIndependentClaim(t, "field:BeatEffects.TremoloPicking", claimSite("tremolo", "import", "M11-TREMOLO-PICKING", "binary marks 1, 2, and 3"))
}
