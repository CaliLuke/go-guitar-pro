// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestGPIFHammerAndTapOriginsRemainDistinct(t *testing.T) {
	for _, test := range []struct {
		name            string
		property        string
		wantHammer      bool
		wantTapped      bool
		wantLeftHandTap bool
	}{
		{name: "hammer-pull", property: "HopoOrigin", wantHammer: true},
		{name: "tap", property: "Tapped", wantTapped: true},
		{name: "left-hand tap", property: "LeftHandTapped", wantLeftHandTap: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			enable := ""
			note, err := gpifNoteToNote(&gpifNote{Properties: gpifProperties{Properties: []gpifProperty{{Name: test.property, Enable: &enable}}}}, 6, false)
			if err != nil {
				t.Fatal(err)
			}
			if note.Effect.Hammer != test.wantHammer || note.Effect.Tapped != test.wantTapped || note.Effect.LeftHandTapped != test.wantLeftHandTap {
				t.Errorf("%s effect = %#v", test.property, note.Effect)
			}
		})
	}
}

func TestGPIFPickSlidesRetainDirection(t *testing.T) {
	for _, test := range []struct {
		flags string
		want  SlideType
	}{{flags: "64", want: SlidePickSlideDown}, {flags: "128", want: SlidePickSlideUp}} {
		note, err := gpifNoteToNote(&gpifNote{Properties: gpifProperties{Properties: []gpifProperty{{Name: "Slide", Flags: &test.flags}}}}, 6, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(note.Effect.Slides) != 1 || note.Effect.Slides[0] != test.want {
			t.Errorf("slide flags %s = %v, want [%d]", test.flags, note.Effect.Slides, test.want)
		}
	}
}

func TestGPIFVibratoStrengthRemainsDistinct(t *testing.T) {
	for _, test := range []struct {
		wire string
		want NoteVibrato
	}{{wire: "", want: NoteVibratoNone}, {wire: "None", want: NoteVibratoNone}, {wire: "Slight", want: NoteVibratoSlight}, {wire: "Wide", want: NoteVibratoWide}} {
		note, err := gpifNoteToNote(&gpifNote{Vibrato: test.wire}, 6, false)
		if err != nil {
			t.Fatal(err)
		}
		if note.Effect.VibratoStrength != test.want || note.Effect.Vibrato != (test.want != NoteVibratoNone) {
			t.Errorf("vibrato %q = (%d,%t), want (%d,%t)", test.wire, note.Effect.VibratoStrength, note.Effect.Vibrato, test.want, test.want != NoteVibratoNone)
		}
	}
}

func TestGPIFAccentKindRemainsDistinct(t *testing.T) {
	for _, test := range []struct {
		wire int
		want NoteAccent
	}{{wire: 0, want: NoteAccentNone}, {wire: 0x08, want: NoteAccentNormal}, {wire: 0x04, want: NoteAccentHeavy}, {wire: 0x10, want: NoteAccentTenuto}, {wire: 0x1c, want: NoteAccentTenuto}} {
		note, err := gpifNoteToNote(&gpifNote{Accent: test.wire}, 6, false)
		if err != nil {
			t.Fatal(err)
		}
		if note.Effect.Accent != test.want {
			t.Errorf("accent %#x = %d, want %d", test.wire, note.Effect.Accent, test.want)
		}
	}
}

func TestGP8HammerAndTapOriginsRoundTrip(t *testing.T) {
	tests := []struct {
		name       string
		effect     NoteEffect
		properties []string
	}{
		{name: "none", effect: defaultNoteEffect()},
		{name: "hammer", effect: NoteEffect{Hammer: true}, properties: []string{"HopoOrigin"}},
		{name: "tap", effect: NoteEffect{Tapped: true}, properties: []string{"Tapped"}},
		{name: "left-hand tap", effect: NoteEffect{LeftHandTapped: true}, properties: []string{"LeftHandTapped"}},
		{name: "independent combination", effect: NoteEffect{Hammer: true, Tapped: true, LeftHandTapped: true}, properties: []string{"HopoOrigin", "Tapped", "LeftHandTapped"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			data := noteEffectsStrictEffectExport(t, test.effect)
			wire := extractTechniqueWireNote(t, data)
			if !wire.allEnabled(test.properties...) {
				t.Fatalf("enabled properties = %v, want %v", wire.propertyNames(), test.properties)
			}
			roundTrip, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect
			if got.Hammer != test.effect.Hammer || got.Tapped != test.effect.Tapped || got.LeftHandTapped != test.effect.LeftHandTapped {
				t.Fatalf("round-trip origins = %#v, want %#v", got, test.effect)
			}
		})
	}
}

func TestGP8PickSlideEquivalenceClass(t *testing.T) {
	for _, test := range []struct {
		name   string
		slides []SlideType
		flags  string
	}{
		{name: "none"},
		{name: "down", slides: []SlideType{SlidePickSlideDown}, flags: "64"},
		{name: "up", slides: []SlideType{SlidePickSlideUp}, flags: "128"},
		{name: "both directions", slides: []SlideType{SlidePickSlideDown, SlidePickSlideUp}, flags: "192"},
		{name: "pick and linked slide", slides: []SlideType{SlideShiftSlideTo, SlidePickSlideDown}, flags: "65"},
	} {
		t.Run(test.name, func(t *testing.T) {
			effect := defaultNoteEffect()
			effect.Slides = test.slides
			data := noteEffectsStrictEffectExport(t, effect)
			wire := extractTechniqueWireNote(t, data)
			if got := wire.propertyText("Slide", "flags"); got != test.flags {
				t.Fatalf("slide flags = %q, want %q", got, test.flags)
			}
			roundTrip, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Slides
			if !slices.Equal(got, test.slides) {
				t.Fatalf("round-trip slides = %v, want %v", got, test.slides)
			}
		})
	}
}

func TestGP8NoteVibratoEquivalenceClass(t *testing.T) {
	for _, test := range []struct {
		name       string
		strength   NoteVibrato
		legacy     bool
		wantWire   string
		wantParsed NoteVibrato
	}{
		{name: "none"},
		{name: "legacy presence", legacy: true, wantWire: "Slight", wantParsed: NoteVibratoSlight},
		{name: "slight", strength: NoteVibratoSlight, wantWire: "Slight", wantParsed: NoteVibratoSlight},
		{name: "wide", strength: NoteVibratoWide, wantWire: "Wide", wantParsed: NoteVibratoWide},
		{name: "wide with compatible legacy presence", strength: NoteVibratoWide, legacy: true, wantWire: "Wide", wantParsed: NoteVibratoWide},
	} {
		t.Run(test.name, func(t *testing.T) {
			effect := defaultNoteEffect()
			effect.VibratoStrength = test.strength
			effect.Vibrato = test.legacy
			data := noteEffectsStrictEffectExport(t, effect)
			if got := extractGPIFLeafText(t, data)["GPIF/Notes/Note/Vibrato"]; got != test.wantWire {
				t.Fatalf("vibrato wire = %q, want %q", got, test.wantWire)
			}
			roundTrip, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect
			if got.VibratoStrength != test.wantParsed || got.Vibrato != (test.wantParsed != NoteVibratoNone) {
				t.Fatalf("round-trip vibrato = (%d,%t)", got.VibratoStrength, got.Vibrato)
			}
		})
	}
}

func TestGP8NoteAccentEquivalenceClass(t *testing.T) {
	for _, test := range []struct {
		name       string
		accent     NoteAccent
		normal     bool
		heavy      bool
		wantWire   string
		wantParsed NoteAccent
	}{
		{name: "none"},
		{name: "legacy normal", normal: true, wantWire: "8", wantParsed: NoteAccentNormal},
		{name: "legacy heavy", heavy: true, wantWire: "4", wantParsed: NoteAccentHeavy},
		{name: "normal", accent: NoteAccentNormal, wantWire: "8", wantParsed: NoteAccentNormal},
		{name: "heavy", accent: NoteAccentHeavy, wantWire: "4", wantParsed: NoteAccentHeavy},
		{name: "tenuto", accent: NoteAccentTenuto, wantWire: "16", wantParsed: NoteAccentTenuto},
	} {
		t.Run(test.name, func(t *testing.T) {
			effect := defaultNoteEffect()
			effect.Accent = test.accent
			effect.AccentuatedNote = test.normal
			effect.HeavyAccentuatedNote = test.heavy
			data := noteEffectsStrictEffectExport(t, effect)
			if got := extractGPIFLeafText(t, data)["GPIF/Notes/Note/Accent"]; got != test.wantWire {
				t.Fatalf("accent wire = %q, want %q", got, test.wantWire)
			}
			roundTrip, err := Parse(data)
			if err != nil {
				t.Fatal(err)
			}
			got := roundTrip.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Accent
			if got != test.wantParsed {
				t.Fatalf("round-trip accent = %d, want %d", got, test.wantParsed)
			}
		})
	}
}

func TestNoteAccentAuthorityPolicy(t *testing.T) {
	effect := defaultNoteEffect()
	effect.Accent = NoteAccentTenuto
	effect.HeavyAccentuatedNote = true
	song := noteEffectsEffectSong(t, effect)
	songBefore := fmt.Sprintf("%#v", song)
	preflight := PreflightExport(song, ExportFormatGP8, ExportOptions{})
	entry := exportReportEntry(preflight, "gp8.normalize.note-accent-authority")
	if entry == nil || entry.Location != (ScoreLocation{}) {
		t.Fatalf("accent authority report = %#v", entry)
	}
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
	if err != nil || len(data) == 0 || !reflect.DeepEqual(report, preflight) {
		t.Fatalf("permissive export = %d bytes, %#v, %v", len(data), report.Entries, err)
	}
	strict, _, strictErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	var lossErr *ExportLossError
	if len(strict) != 0 || !errors.As(strictErr, &lossErr) {
		t.Fatalf("strict export = %d bytes, %v", len(strict), strictErr)
	}
	allowed, _, allowedErr := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{
		RequirePreservation: true, AllowedCodes: []string{"gp8.normalize.note-accent-authority"},
	}})
	if allowedErr != nil || len(allowed) == 0 {
		t.Fatalf("allowed export = %d bytes, %v", len(allowed), allowedErr)
	}
	if got := extractGPIFLeafText(t, allowed)["GPIF/Notes/Note/Accent"]; got != "16" {
		t.Fatalf("authority accent wire = %q, want 16", got)
	}
	if fmt.Sprintf("%#v", song) != songBefore {
		t.Fatal("preflight or export mutated the score")
	}
}

func TestRejectsInvalidNoteEffectEnums(t *testing.T) {
	tests := []struct {
		name string
		set  func(*NoteEffect)
		code string
	}{
		{name: "slide below", set: func(effect *NoteEffect) { effect.Slides = []SlideType{-3} }, code: "score.note.slide"},
		{name: "slide above", set: func(effect *NoteEffect) { effect.Slides = []SlideType{7} }, code: "score.note.slide"},
		{name: "vibrato", set: func(effect *NoteEffect) { effect.VibratoStrength = NoteVibrato(3) }, code: "score.note.vibrato"},
		{name: "accent", set: func(effect *NoteEffect) { effect.Accent = NoteAccent(4) }, code: "score.note.accent"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			effect := defaultNoteEffect()
			test.set(&effect)
			song := noteEffectsEffectSong(t, effect)
			if !slices.ContainsFunc(ValidateSong(song), func(diagnostic ScoreDiagnostic) bool { return diagnostic.Code == test.code }) {
				t.Fatalf("diagnostics = %#v, want %s", ValidateSong(song), test.code)
			}
			data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{})
			if len(data) != 0 || err == nil || !hasExportCode(report, "gp8.reject."+test.code) {
				t.Fatalf("invalid export = %d bytes, %#v, %v", len(data), report.Entries, err)
			}
		})
	}
}

func TestResolvedHammerAndLinkedSlideEquivalence(t *testing.T) {
	staff := Staff{
		Strings: []GuitarString{{Number: 1}, {Number: 2}},
		Measures: []Measure{
			{Voices: []Voice{
				{Beats: []Beat{
					{Notes: []Note{{String: 1, Effect: NoteEffect{Hammer: true, Slides: []SlideType{SlideShiftSlideTo}}}}},
					{Notes: []Note{{String: 1}}},
					{Notes: []Note{{String: 2, Effect: NoteEffect{Hammer: true}}}},
					{Notes: []Note{{String: 1, Effect: NoteEffect{LeftHandTapped: true}}}},
					{Notes: []Note{{String: 1, Effect: NoteEffect{Hammer: true, Slides: []SlideType{SlideLegatoSlideTo, SlidePickSlideUp}}}}},
				}},
			}},
		},
	}
	links := goNoteLinkProjections(&staff)
	beats := staff.Measures[0].Voices[0].Beats
	if !links[&beats[0].Notes[0]].hammerOrigin || !slices.Equal(links[&beats[0].Notes[0]].slides, []SlideType{SlideShiftSlideTo}) {
		t.Fatalf("same-string destination projection = %#v", links[&beats[0].Notes[0]])
	}
	if !links[&beats[2].Notes[0]].hammerOrigin {
		t.Fatal("left-hand-tapped different-string destination was not resolved")
	}
	dangling := links[&beats[4].Notes[0]]
	if dangling.hammerOrigin || !slices.Equal(dangling.slides, []SlideType{SlidePickSlideUp}) {
		t.Fatalf("dangling linked effects = %#v, want hammer false and independent pick slide", dangling)
	}
}

func TestAlphaTabConsumesExportedNoteEffectKinds(t *testing.T) {
	requireAlphaTabConformance(t)
	effect := defaultNoteEffect()
	effect.Hammer = true
	effect.Slides = []SlideType{SlidePickSlideDown}
	effect.VibratoStrength = NoteVibratoWide
	effect.Accent = NoteAccentTenuto
	song := noteEffectsEffectSong(t, effect)
	voice := &song.Tracks[0].Measures[0].Voices[0]
	voice.Beats = append(voice.Beats, Beat{Duration: defaultDuration(), Status: BeatStatusNormal, Dynamics: Forte, Notes: []Note{{String: 1, Value: 5, Kind: NoteTypeNormal, Velocity: Forte, DurationPercent: 1, Effect: defaultNoteEffect()}}})
	if err := FinalizeSong(song); err != nil {
		t.Fatal(err)
	}
	data := noteEffectsStrictSongExport(t, song)
	alpha := readAlphaTabScore(t, writeConformanceFixture(t, data))
	facts := collectConformanceFacts(alpha, map[string]bool{"effects": true}, nil)
	if len(facts) < 1 {
		t.Fatal("AlphaTab returned no note effects")
	}
	value := facts[0].(map[string]any)["value"].(map[string]any)
	if value["hammerOrigin"] != true || value["vibrato"] != "wide" || value["accent"] != "tenuto" || !reflect.DeepEqual(value["slides"], []any{"pick-slide-down"}) {
		t.Fatalf("AlphaTab effects = %#v", value)
	}
}

func noteEffectsStrictEffectExport(t *testing.T, effect NoteEffect) []byte {
	t.Helper()
	return noteEffectsStrictSongExport(t, noteEffectsEffectSong(t, effect))
}

func noteEffectsStrictSongExport(t *testing.T, song *Song) []byte {
	t.Helper()
	data, report, err := ExportWithReport(song, ExportFormatGP8, ExportOptions{LossPolicy: ExportLossPolicy{RequirePreservation: true}})
	if err != nil || len(data) == 0 {
		t.Fatalf("strict effect export = %d bytes, %#v, %v", len(data), report.Entries, err)
	}
	return data
}

func noteEffectsEffectSong(t *testing.T, effect NoteEffect) *Song {
	t.Helper()
	song := conformanceTechniqueSong(t)
	song.Tracks[0].Settings.Notation = true
	song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect = effect
	return song
}

func TestAlphaTabNoteEffectCorpusClasses(t *testing.T) {
	requireAlphaTabConformance(t)
	fixtures := []string{
		"testdata/gp3/Effects.gp3", "testdata/gp3/hammer.gp3", "testdata/gp3/slides.gp3", "testdata/gp3/vibrato.gp3",
		"testdata/gp4/Effects.gp4", "testdata/gp4/accentuations.gp4", "testdata/gp4/hammer.gp4",
		"testdata/gp5/Effects.gp5", "testdata/gp5/hammer.gp5",
		"testdata/gp6/effects.gpx", "testdata/gp6/hammer.gpx", "testdata/gp6/other-effects.gpx",
		"testdata/gp7/accentuations.gp", "testdata/gp7/effects.gp", "testdata/gp7/hammer.gp", "testdata/gp7/left-hand-tap.gp",
		"testdata/gp7/other-effects.gp", "testdata/gp7/pick-slide.gp", "testdata/gp7/tap.gp", "testdata/gp7/tremolo-vibrato.gp",
	}
	oracleScores, err := readCorpusOracleScores(fixtures)
	if err != nil {
		t.Fatal(err)
	}
	var remaining []string
	for _, fixture := range fixtures {
		result, parseErr := ParseWithOptions(noteEffectsReadFile(t, fixture), ParseOptions{})
		if parseErr != nil {
			t.Fatalf("%s: %v", fixture, parseErr)
		}
		features := []string{"note-and-beat-semantics"}
		goScore := selectConformanceFeatures(normalizeGoScore(result.Song), features)
		alphaScore := selectConformanceFeatures(oracleScores[fixture], features)
		for _, difference := range semanticDifferences(goScore, alphaScore) {
			semanticPath := conformanceCorpusSemanticPath(difference.Path, goScore, alphaScore)
			if noteEffectsNoteEffectDifferencePath(semanticPath) {
				remaining = append(remaining, fmt.Sprintf("%s%s", fixture, semanticPath))
			}
		}
	}
	if len(remaining) != 0 {
		t.Fatalf("target note-effect differences remain (%d):\n%s", len(remaining), strings.Join(remaining, "\n"))
	}
}

func noteEffectsNoteEffectDifferencePath(path string) bool {
	return strings.HasSuffix(path, "/effects/hammerOrigin") ||
		strings.Contains(path, "/effects/slides/") ||
		strings.HasSuffix(path, "/effects/vibrato") ||
		strings.HasSuffix(path, "/effects/accent")
}

func noteEffectsReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestLegacyHarmonicFretDerivation(t *testing.T) {
	tests := []struct {
		path    string
		measure int
		want    []float64
	}{
		{path: "testdata/gp3/Effects.gp3", measure: 3, want: []float64{2.4, 2.4, 2.4, 2.4, 2.4}},
		{path: "testdata/gp4/Effects.gp4", measure: 3, want: []float64{2.4, 12, 12, 12, 12}},
		{path: "testdata/gp5/Effects.gp5", measure: 3, want: []float64{2.4, 3.2, 3.2, 12, 12}},
		{path: "testdata/gp5/Harmonics.gp5", measure: 0, want: []float64{2.4, 12, 14.7, 12, 12}},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			song := parseTestFixture(t, test.path)
			beats := song.Tracks[0].Measures[test.measure].Voices[0].Beats
			for index, want := range test.want {
				harmonic := beats[index].Notes[0].Effect.Harmonic
				if harmonic == nil || harmonic.FretFloat == nil {
					t.Errorf("beat %d harmonic fret = %#v, want %v", index, harmonic, want)
					continue
				}
				if got := *harmonic.FretFloat; got != want {
					t.Errorf("beat %d harmonic fret = %v, want %v", index, got, want)
				}
			}
		})
	}
}

func TestLegacyTrillFretUsesResolvedMIDIPitch(t *testing.T) {
	for _, path := range []string{
		"testdata/gp4/trills.gp4",
		"testdata/gp5/trills.gp5",
	} {
		t.Run(path, func(t *testing.T) {
			song := parseTestFixture(t, path)
			trill := song.Tracks[0].Measures[0].Voices[0].Beats[0].Notes[0].Effect.Trill
			if trill == nil {
				t.Fatal("trill is nil")
			}
			if trill.Fret != 61 {
				t.Errorf("trill MIDI pitch = %d, want 61", trill.Fret)
			}
		})
	}
}

func TestGPIFFeedbackHarmonicRemainsDistinct(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp7/notation-legend.gp")
	harmonic := song.Tracks[0].Measures[67].Voices[0].Beats[0].Notes[0].Effect.Harmonic
	if harmonic == nil {
		t.Fatal("feedback harmonic is nil")
	}
	if harmonic.Kind != HarmonicType(6) {
		t.Errorf("feedback harmonic kind = %d, want distinct kind 6", harmonic.Kind)
	}
}
