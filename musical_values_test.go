// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"math"
	"testing"
)

func TestMusicalValueBoundaries(t *testing.T) {
	tests := []struct {
		name    string
		valid   func() error
		invalid func() error
	}{
		{"fret", func() error { _, err := NewFret(math.MaxInt16); return err }, func() error { _, err := NewFret(-1); return err }},
		{"MIDI note", func() error { _, err := NewMIDINote(127); return err }, func() error { _, err := NewMIDINote(128); return err }},
		{"articulation ID", func() error { _, err := NewPercussionArticulationID(math.MaxInt32); return err }, func() error { _, err := NewPercussionArticulationID(-1); return err }},
		{"audio frame", func() error { _, err := NewAudioFrame(math.MaxInt64); return err }, func() error { _, err := NewAudioFrame(-1); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.valid(); err != nil {
				t.Fatalf("valid boundary: %v", err)
			}
			if err := test.invalid(); err == nil {
				t.Fatal("invalid boundary was accepted")
			}
		})
	}
}

func TestBPMPreservesFractionalTempo(t *testing.T) {
	tempo, err := NewBPM(120.5)
	if err != nil {
		t.Fatal(err)
	}
	if float64(tempo) != 120.5 {
		t.Fatalf("tempo = %v, want 120.5", tempo)
	}
	for _, invalid := range []float64{0, -1, math.NaN(), math.Inf(1)} {
		if _, err := NewBPM(invalid); err == nil {
			t.Errorf("NewBPM(%v) accepted invalid tempo", invalid)
		}
	}
}

func TestSourceValueDistinguishesMissingZeroAndUnknown(t *testing.T) {
	var missing SourceValue[int]
	zero := KnownSourceValue(0)
	unknown := UnknownSourceValue[int]("future-value")
	if missing.State != SourceValueMissing {
		t.Fatalf("zero value state = %v, want missing", missing.State)
	}
	if zero.State != SourceValueKnown || zero.Value != 0 {
		t.Fatalf("known zero = %#v", zero)
	}
	if unknown.State != SourceValueUnknown || unknown.Raw != "future-value" {
		t.Fatalf("unknown value = %#v", unknown)
	}
}

func TestScoreTimeArithmeticAndBoundaries(t *testing.T) {
	zero := ScoreTime{}
	if zero.Numerator() != 0 || zero.Denominator() != 1 {
		t.Fatalf("zero score time = %d/%d, want 0/1", zero.Numerator(), zero.Denominator())
	}
	oneThird, err := NewScoreTime(1, 3)
	if err != nil {
		t.Fatal(err)
	}
	twoThirds, err := oneThird.Add(oneThird)
	if err != nil {
		t.Fatal(err)
	}
	if twoThirds.Numerator() != 2 || twoThirds.Denominator() != 3 {
		t.Fatalf("sum = %d/%d, want 2/3", twoThirds.Numerator(), twoThirds.Denominator())
	}
	maximum, _ := NewScoreTime(math.MaxInt64, 1)
	one, _ := NewScoreTime(1, 1)
	if _, overflowErr := maximum.Add(one); overflowErr == nil {
		t.Fatal("overflowing score-time addition succeeded")
	}
	if _, overflowErr := maximum.Multiply(2); overflowErr == nil {
		t.Fatal("overflowing score-time multiplication succeeded")
	}
	if _, multiplyErr := one.Multiply(-1); multiplyErr == nil {
		t.Fatal("negative score-time multiplier was accepted")
	}
	if _, subtractErr := oneThird.Subtract(one); subtractErr == nil {
		t.Fatal("negative score-time difference was accepted")
	}
	if _, constructionErr := NewScoreTime(-1, 1); constructionErr == nil {
		t.Fatal("negative score time was accepted")
	}
	if _, constructionErr := NewScoreTime(1, 0); constructionErr == nil {
		t.Fatal("zero score-time denominator was accepted")
	}
	halfMaximum, _ := NewScoreTime(math.MaxInt64, 2)
	product, err := halfMaximum.Multiply(2)
	if err != nil || product.Numerator() != math.MaxInt64 || product.Denominator() != 1 {
		t.Fatalf("cancelling multiplication = %d/%d, %v; want %d/1", product.Numerator(), product.Denominator(), err, int64(math.MaxInt64))
	}
	sum, err := halfMaximum.Add(halfMaximum)
	if err != nil || sum.Numerator() != math.MaxInt64 || sum.Denominator() != 1 {
		t.Fatalf("cancelling addition = %d/%d, %v; want %d/1", sum.Numerator(), sum.Denominator(), err, int64(math.MaxInt64))
	}
	largeDenominator, _ := NewScoreTime(math.MaxInt64-2, math.MaxInt64-1)
	if _, err := halfMaximum.Add(largeDenominator); err == nil {
		t.Fatal("unrepresentable large cross-product sum unexpectedly succeeded")
	}
}

func TestMusicalDurationRejectsInvalidTerms(t *testing.T) {
	for _, duration := range []MusicalDuration{
		{Value: 3, Tuplet: TupletRatio{Enters: 1, Times: 1}},
		{Value: 4, Dots: 3, Tuplet: TupletRatio{Enters: 1, Times: 1}},
		{Value: 4, Tuplet: TupletRatio{Enters: 3}},
	} {
		if _, err := duration.ExactScoreTime(); err == nil {
			t.Errorf("invalid duration accepted: %#v", duration)
		}
	}
}

func TestTimingRejectsInvalidDurationInsteadOfFreezingTimeline(t *testing.T) {
	invalid := defaultBeat()
	invalid.Duration.TupletEnters = 0
	song := Song{
		MeasureHeaders: []MeasureHeader{defaultMeasureHeader()},
		Tracks: []Track{{Measures: []Measure{{
			HeaderIndex: 0,
			Voices:      []Voice{{Beats: []Beat{invalid}}},
		}}}},
	}
	if err := song.finalizeTiming(); err == nil {
		t.Fatal("invalid duration silently finalized")
	}
}

func TestGPIFGracePreservesFretBeyondLegacyWidth(t *testing.T) {
	note := Note{Value: 200}
	duration := defaultDuration()
	grace := gpifGraceEffect(&note, &duration, false, 0)
	if grace.ExactFret == nil || *grace.ExactFret != Fret(200) {
		t.Fatalf("exact grace fret = %v, want 200", grace.ExactFret)
	}
	if grace.Fret != 0 {
		t.Fatalf("legacy grace fret = %d, want unavailable zero rather than a clamp", grace.Fret)
	}
}

func TestBarPositionRange(t *testing.T) {
	for _, terms := range [][2]int64{{0, 1}, {1, 2}, {1, 1}} {
		if _, err := NewBarPosition(terms[0], terms[1]); err != nil {
			t.Errorf("NewBarPosition(%d, %d): %v", terms[0], terms[1], err)
		}
	}
	if _, err := NewBarPosition(3, 2); err == nil {
		t.Fatal("bar position greater than one was accepted")
	}
	half, err := NewBarPositionFromFloat64(0.5)
	if err != nil || half.Ratio().Numerator() != 1 || half.Ratio().Denominator() != 2 {
		t.Fatalf("float bar position = %d/%d, %v; want 1/2", half.Ratio().Numerator(), half.Ratio().Denominator(), err)
	}
	if _, err := NewBarPositionFromFloat64(math.NaN()); err == nil {
		t.Fatal("NaN bar position was accepted")
	}
}

func TestMusicalDurationExactConversion(t *testing.T) {
	duration, err := NewMusicalDuration(4, 2, TupletRatio{Enters: 3, Times: 2})
	if err != nil {
		t.Fatal(err)
	}
	exact, err := duration.ExactScoreTime()
	if err != nil {
		t.Fatal(err)
	}
	if exact.Numerator() != 1120 || exact.Denominator() != 1 {
		t.Fatalf("double-dotted quarter triplet = %d/%d, want 1120/1", exact.Numerator(), exact.Denominator())
	}

	legacy, err := (MusicalDuration{Value: 4, Tuplet: TupletRatio{Enters: 256, Times: 1}}).LegacyDuration()
	if err == nil {
		t.Fatalf("legacy conversion unexpectedly succeeded: %#v", legacy)
	}

	ambiguous := defaultDuration()
	ambiguous.Dotted = true
	ambiguous.DoubleDotted = true
	semantic, err := ambiguous.MusicalDuration()
	if err != nil || semantic.Dots != 2 {
		t.Fatalf("ambiguous legacy dots converted to %#v, %v; want two dots", semantic, err)
	}
}

func TestTimingAccumulatesNonIntegralTupletsExactly(t *testing.T) {
	septuplet := defaultBeat()
	septuplet.Duration.TupletEnters = 7
	septuplet.Duration.TupletTimes = 4
	beats := make([]Beat, 7)
	for index := range beats {
		beats[index] = septuplet
	}
	song := Song{
		Anacrusis:      true,
		MeasureHeaders: []MeasureHeader{defaultMeasureHeader(), defaultMeasureHeader()},
		Tracks: []Track{{Measures: []Measure{
			{HeaderIndex: 0, Voices: []Voice{{Beats: beats}}},
			{HeaderIndex: 1, Voices: []Voice{{Beats: []Beat{defaultBeat()}}}},
		}}},
	}
	if err := song.finalizeTiming(); err != nil {
		t.Fatal(err)
	}

	wantStarts := []int64{960, 1508, 2057, 2605, 3154, 3702, 4251}
	gotBeats := song.Tracks[0].Measures[0].Voices[0].Beats
	for index, want := range wantStarts {
		if gotBeats[index].Start == nil || *gotBeats[index].Start != want {
			t.Errorf("beat %d start = %v, want %d", index, gotBeats[index].Start, want)
		}
		if gotBeats[index].ExactStart == nil {
			t.Errorf("beat %d exact start is missing", index)
		}
	}
	second := song.MeasureHeaders[1]
	if second.Start != 4800 || second.ExactStart.Numerator() != 4800 || second.ExactStart.Denominator() != 1 {
		t.Fatalf("second measure start = %d (%d/%d), want 4800 (4800/1)", second.Start, second.ExactStart.Numerator(), second.ExactStart.Denominator())
	}
}
