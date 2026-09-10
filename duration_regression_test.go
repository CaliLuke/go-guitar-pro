// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"math"
	"strings"
	"testing"
)

func TestDurationTime(t *testing.T) {
	tests := []struct {
		name string
		in   Duration
		want uint32
	}{
		{"quarter", Duration{Value: 4, TupletEnters: 1, TupletTimes: 1}, 960},
		{"dotted quarter", Duration{Value: 4, Dotted: true, TupletEnters: 1, TupletTimes: 1}, 1440},
		{"double dotted quarter", Duration{Value: 4, DoubleDotted: true, TupletEnters: 1, TupletTimes: 1}, 1680},
		{"simultaneous dot flags use double dot", Duration{Value: 4, Dotted: true, DoubleDotted: true, TupletEnters: 1, TupletTimes: 1}, 1680},
		{"double dotted eighth triplet", Duration{Value: 8, DoubleDotted: true, TupletEnters: 3, TupletTimes: 2}, 560},
		{"quarter triplet", Duration{Value: 4, TupletEnters: 3, TupletTimes: 2}, 640},
		{"unset value", Duration{TupletEnters: 1, TupletTimes: 1}, 0},
		{"unset tuplet", Duration{Value: 4}, 960},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.in.time(); got != test.want {
				t.Fatalf("time() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestGPIFDoubleDottedDurationAndTiming(t *testing.T) {
	duration, err := gpifRhythmToDuration(&gpifRhythm{
		NoteValue: "Eighth", AugmentationDot: &gpifAugDot{Count: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if duration.Dotted || !duration.DoubleDotted || duration.time() != 840 {
		t.Fatalf("duration = %#v, time = %d; want one double-dot flag and 840 ticks", duration, duration.time())
	}

	song := parseTestFixture(t, "testdata/gp7/colors.gp")
	beats := song.Tracks[0].Staves[0].Measures[1].Voices[0].Beats
	if len(beats) < 2 || beats[0].Start == nil || beats[1].Start == nil {
		t.Fatalf("fixture beats = %#v, want two finalized beat starts", beats)
	}
	if got := beats[0].Duration; got.Dotted || !got.DoubleDotted || got.time() != 840 {
		t.Fatalf("fixture duration = %#v, time = %d; want double-dotted eighth at 840 ticks", got, got.time())
	}
	if *beats[1].Start != 5640 || *beats[1].Start-*beats[0].Start != 840 {
		t.Fatalf("beat starts = %d, %d; want 4800, 5640", *beats[0].Start, *beats[1].Start)
	}
}

func TestGPIFConversionsRejectValuesThatWouldNarrow(t *testing.T) {
	if _, err := gpifRhythmToDuration(&gpifRhythm{
		NoteValue: "Quarter", PrimaryTuplet: &gpifTuplet{Num: 256, Den: 255},
	}); err == nil {
		t.Fatal("uint8-overflowing tuplet was accepted")
	}
	fret := math.MaxInt16 + 1
	if _, err := gpifNoteToNote(&gpifNote{Properties: gpifProperties{Properties: []gpifProperty{
		{Name: "Fret", Fret: &fret},
	}}}, 6, false); err == nil {
		t.Fatal("int16-overflowing fret was accepted")
	}
	midi := 128
	if _, err := gpifNoteToNote(&gpifNote{Properties: gpifProperties{Properties: []gpifProperty{
		{Name: "Midi", Number: &midi},
	}}}, 6, false); err == nil {
		t.Fatal("out-of-range MIDI note was accepted")
	}
}

func TestGPIFMasterBarValuesDoNotWrapAtLegacyBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(string) string
		wantError bool
		errorText string
		assert    func(*testing.T, *Song)
	}{
		{name: "maximum numerator", mutate: func(gpif string) string {
			return strings.Replace(gpif, "4/4", "127/4", 1)
		}, assert: func(t *testing.T, song *Song) {
			if song.MeasureHeaders[0].TimeSignature.Numerator != 127 {
				t.Fatalf("numerator = %d, want 127", song.MeasureHeaders[0].TimeSignature.Numerator)
			}
		}},
		{name: "numerator overflow", wantError: true, mutate: func(gpif string) string {
			return strings.Replace(gpif, "4/4", "128/4", 1)
		}},
		{name: "modulo numerator", wantError: true, mutate: func(gpif string) string {
			return strings.Replace(gpif, "4/4", "260/4", 1)
		}},
		{name: "denominator modulo overflow", wantError: true, errorText: "outside 1..65535", mutate: func(gpif string) string {
			return strings.Replace(gpif, "4/4", "4/65540", 1)
		}},
		{name: "key overflow", wantError: true, mutate: func(gpif string) string {
			return strings.Replace(gpif, "<MasterBar><Time>", "<MasterBar><Key><AccidentalCount>128</AccidentalCount></Key><Time>", 1)
		}},
		{name: "maximum repeat count", mutate: func(gpif string) string {
			return strings.Replace(gpif, "<Bars>0 1</Bars>", `<Repeat end="true" count="128"/><Bars>0 1</Bars>`, 1)
		}, assert: func(t *testing.T, song *Song) {
			if song.MeasureHeaders[0].RepeatClose != 127 {
				t.Fatalf("repeat close = %d, want 127", song.MeasureHeaders[0].RepeatClose)
			}
		}},
		{name: "repeat overflow", wantError: true, mutate: func(gpif string) string {
			return strings.Replace(gpif, "<Bars>0 1</Bars>", `<Repeat end="true" count="130"/><Bars>0 1</Bars>`, 1)
		}},
		{name: "alternate ending overflow", wantError: true, mutate: func(gpif string) string {
			return strings.Replace(gpif, "<Bars>0 1</Bars>", "<AlternateEndings>9</AlternateEndings><Bars>0 1</Bars>", 1)
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := ParseWithOptions(conformanceGPIFArchive(t, test.mutate(staffScopedChordGPIF)), ParseOptions{Strict: true})
			if test.wantError {
				if err == nil {
					t.Fatalf("strict parse succeeded with song %#v", result.Song.MeasureHeaders[0])
				}
				if test.errorText != "" && !strings.Contains(err.Error(), test.errorText) {
					t.Fatalf("error = %v, want %q", err, test.errorText)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if test.assert != nil {
				test.assert(t, result.Song)
			}
		})
	}
}

func TestGPIFPreservesFractionalInitialTempo(t *testing.T) {
	data := diagnosticGP8Fixture(t, func(gpif string) string {
		return strings.Replace(gpif, "132 2", "120.5 2", 1)
	})
	result, err := ParseWithOptions(data, ParseOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Song.InitialTempo.State != SourceValueKnown || result.Song.InitialTempo.Value != BPM(120.5) {
		t.Fatalf("initial tempo = %#v, want known 120.5 BPM", result.Song.InitialTempo)
	}
}

func TestGPIFTempoBoundaryDiagnostics(t *testing.T) {
	t.Run("non-finite is unknown", func(t *testing.T) {
		data := diagnosticGP8Fixture(t, func(gpif string) string {
			return strings.Replace(gpif, "132 2", "Inf 2", 1)
		})
		result, err := ParseWithOptions(data, ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if result.Song.InitialTempo.State != SourceValueUnknown || result.Song.InitialTempo.Raw != "Inf" {
			t.Fatalf("initial tempo = %#v, want unknown Inf", result.Song.InitialTempo)
		}
		if findParseDiagnostic(result.Diagnostics, ParseDiagnosticInvalidData, "tempo-automations") == nil {
			t.Fatalf("diagnostics = %#v, want invalid tempo", result.Diagnostics)
		}
	})

	t.Run("legacy overflow preserves semantic BPM", func(t *testing.T) {
		data := diagnosticGP8Fixture(t, func(gpif string) string {
			return strings.Replace(gpif, "132 2", "40000 2", 1)
		})
		result, err := ParseWithOptions(data, ParseOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if result.Song.InitialTempo.State != SourceValueKnown || result.Song.InitialTempo.Value != BPM(40000) {
			t.Fatalf("initial tempo = %#v, want known 40000 BPM", result.Song.InitialTempo)
		}
		if result.Song.Tempo < 0 {
			t.Fatalf("legacy tempo silently wrapped to %d", result.Song.Tempo)
		}
		if findParseDiagnostic(result.Diagnostics, ParseDiagnosticLossyProjection, "tempo-automations") == nil {
			t.Fatalf("diagnostics = %#v, want legacy tempo projection warning", result.Diagnostics)
		}
	})
}

func TestTupletBeatsAdvanceByPlayedLength(t *testing.T) {
	song := parseTestFixture(t, "testdata/gp5/tuplets.gp5")
	beats := song.Tracks[0].Measures[0].Voices[0].Beats
	if len(beats) < 3 {
		t.Fatalf("beats = %d, want at least 3", len(beats))
	}
	base := *beats[0].Start
	for index, beat := range beats[:3] {
		want := base + int64(index)*640
		if beat.Start == nil || *beat.Start != want {
			t.Errorf("beat %d starts at %v, want %d", index, beat.Start, want)
		}
	}
}
