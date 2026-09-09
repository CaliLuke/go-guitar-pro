// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"math"
	"math/big"
	"math/bits"
)

// Fret is a non-negative authored fret number independent of a wire-format width.
type Fret int16

// NewFret validates and converts an authored fret number.
func NewFret(value int64) (Fret, error) {
	if value < 0 || value > math.MaxInt16 {
		return 0, fmt.Errorf("fret %d is outside 0..%d", value, math.MaxInt16)
	}
	return Fret(value), nil
}

// MIDINote is an absolute MIDI note number.
type MIDINote uint8

// NewMIDINote validates and converts an absolute MIDI note number.
func NewMIDINote(value int64) (MIDINote, error) {
	if value < 0 || value > 127 {
		return 0, fmt.Errorf("MIDI note %d is outside 0..127", value)
	}
	return MIDINote(value), nil
}

// PercussionArticulationID identifies an entry in one track's articulation table.
type PercussionArticulationID int32

// NewPercussionArticulationID validates a present track-local articulation ID.
func NewPercussionArticulationID(value int64) (PercussionArticulationID, error) {
	if value < 0 || value > math.MaxInt32 {
		return 0, fmt.Errorf("percussion articulation ID %d is outside 0..%d", value, math.MaxInt32)
	}
	return PercussionArticulationID(value), nil
}

// BPM is a finite positive tempo in beats per minute.
type BPM float64

// NewBPM validates and converts a tempo without discarding its fractional part.
func NewBPM(value float64) (BPM, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return 0, fmt.Errorf("BPM %v must be finite and positive", value)
	}
	return BPM(value), nil
}

// LegacyTempo rounds BPM to the existing int16 tempo field without wrapping or clamping.
func (b BPM) LegacyTempo() (int16, error) {
	validated, err := NewBPM(float64(b))
	if err != nil {
		return 0, err
	}
	rounded := math.Round(float64(validated))
	if rounded > math.MaxInt16 {
		return 0, fmt.Errorf("BPM %v exceeds the legacy int16 boundary", b)
	}
	return int16(rounded), nil
}

// AudioFrame is a non-negative absolute audio sample-frame position.
type AudioFrame int64

// NewAudioFrame validates an absolute audio sample-frame position.
func NewAudioFrame(value int64) (AudioFrame, error) {
	if value < 0 {
		return 0, fmt.Errorf("audio frame %d must be non-negative", value)
	}
	return AudioFrame(value), nil
}

// SourceValueState distinguishes absent, recognized, and unknown source values.
type SourceValueState uint8

const (
	// SourceValueMissing means the source omitted the value.
	SourceValueMissing SourceValueState = iota
	// SourceValueKnown means Value contains a recognized value, including zero.
	SourceValueKnown
	// SourceValueUnknown means Raw contains an unrecognized source value.
	SourceValueUnknown
)

// SourceValue preserves whether a source value was absent, recognized, or unknown.
type SourceValue[T any] struct {
	Value T
	Raw   string
	State SourceValueState
}

// KnownSourceValue returns a present recognized source value.
func KnownSourceValue[T any](value T) SourceValue[T] {
	return SourceValue[T]{Value: value, State: SourceValueKnown}
}

// UnknownSourceValue returns a present but unrecognized source value.
func UnknownSourceValue[T any](raw string) SourceValue[T] {
	return SourceValue[T]{Raw: raw, State: SourceValueUnknown}
}

// ScoreTime is an exact non-negative rational count of score ticks at 960 PPQ.
// Use [ScoreTime.FloorTicks] only at an integer-tick compatibility boundary.
type ScoreTime struct {
	numerator   int64
	denominator int64
}

func (t ScoreTime) terms() (int64, int64) {
	if t.numerator == 0 && t.denominator == 0 {
		return 0, 1
	}
	return t.numerator, t.denominator
}

// NewScoreTime constructs an exact score time and reduces it to lowest terms.
func NewScoreTime(numerator, denominator int64) (ScoreTime, error) {
	if numerator < 0 {
		return ScoreTime{}, fmt.Errorf("score-time numerator %d must be non-negative", numerator)
	}
	if denominator <= 0 {
		return ScoreTime{}, fmt.Errorf("score-time denominator %d must be positive", denominator)
	}
	divisor := greatestCommonDivisor(numerator, denominator)
	return ScoreTime{numerator: numerator / divisor, denominator: denominator / divisor}, nil
}

// Numerator returns the reduced tick numerator.
func (t ScoreTime) Numerator() int64 {
	numerator, _ := t.terms()
	return numerator
}

// Denominator returns the reduced tick denominator.
func (t ScoreTime) Denominator() int64 {
	_, denominator := t.terms()
	return denominator
}

// FloorTicks quantizes score time toward zero for legacy integer-tick fields.
func (t ScoreTime) FloorTicks() int64 {
	numerator, denominator := t.terms()
	if denominator <= 0 {
		return 0
	}
	return numerator / denominator
}

// Add returns the exact sum or an error if the small score-time domain overflows int64.
func (t ScoreTime) Add(other ScoreTime) (ScoreTime, error) {
	return combineScoreTimes(t, other, false)
}

// Subtract returns the exact non-negative difference or an error for a negative result or overflow.
func (t ScoreTime) Subtract(other ScoreTime) (ScoreTime, error) {
	return combineScoreTimes(t, other, true)
}

// Multiply returns the exact product with a non-negative integer or an overflow error.
func (t ScoreTime) Multiply(multiplier int64) (ScoreTime, error) {
	numerator, denominator := t.terms()
	if numerator < 0 || denominator <= 0 {
		return ScoreTime{}, fmt.Errorf("multiplying invalid score time")
	}
	if multiplier < 0 {
		return ScoreTime{}, fmt.Errorf("score-time multiplier %d must be non-negative", multiplier)
	}
	divisor := greatestCommonDivisor(multiplier, denominator)
	multiplier /= divisor
	denominator /= divisor
	product, ok := checkedPositiveMultiply(numerator, multiplier)
	if !ok {
		return ScoreTime{}, fmt.Errorf("score-time multiplication overflows int64")
	}
	return NewScoreTime(product, denominator)
}

func combineScoreTimes(left, right ScoreTime, subtract bool) (ScoreTime, error) {
	leftNumerator, leftDenominator := left.terms()
	rightNumerator, rightDenominator := right.terms()
	if leftNumerator < 0 || leftDenominator <= 0 || rightNumerator < 0 || rightDenominator <= 0 {
		return ScoreTime{}, fmt.Errorf("combining invalid score time")
	}
	commonDivisor := greatestCommonDivisor(leftDenominator, rightDenominator)
	leftHigh, leftLow := bits.Mul64(uint64(leftNumerator), uint64(rightDenominator/commonDivisor))
	rightHigh, rightLow := bits.Mul64(uint64(rightNumerator), uint64(leftDenominator/commonDivisor))

	var numeratorHigh, numeratorLow uint64
	if subtract {
		if leftHigh < rightHigh || (leftHigh == rightHigh && leftLow < rightLow) {
			return ScoreTime{}, fmt.Errorf("score-time difference must be non-negative")
		}
		var borrow uint64
		numeratorLow, borrow = bits.Sub64(leftLow, rightLow, 0)
		numeratorHigh, _ = bits.Sub64(leftHigh, rightHigh, borrow)
	} else {
		var carry uint64
		numeratorLow, carry = bits.Add64(leftLow, rightLow, 0)
		numeratorHigh, carry = bits.Add64(leftHigh, rightHigh, carry)
		if carry != 0 {
			return ScoreTime{}, fmt.Errorf("score-time addition exceeds 128-bit intermediate range")
		}
	}

	reduction := greatestCommonDivisorUint64(scoreTimeUint128Mod(numeratorHigh, numeratorLow, uint64(commonDivisor)), uint64(commonDivisor))
	numeratorHigh, numeratorLow = scoreTimeUint128Divide(numeratorHigh, numeratorLow, reduction)
	if numeratorHigh != 0 || numeratorLow > math.MaxInt64 {
		return ScoreTime{}, fmt.Errorf("score-time result overflows int64")
	}
	denominator, ok := checkedPositiveMultiply(leftDenominator/commonDivisor, rightDenominator/int64(reduction))
	if !ok {
		return ScoreTime{}, fmt.Errorf("score-time denominator overflows int64")
	}
	return NewScoreTime(int64(numeratorLow), denominator)
}

func scoreTimeUint128Mod(high, low, divisor uint64) uint64 {
	if divisor == 1 {
		return 0
	}
	_, remainder := bits.Div64(0, high, divisor)
	_, remainder = bits.Div64(remainder, low, divisor)
	return remainder
}

func scoreTimeUint128Divide(high, low, divisor uint64) (uint64, uint64) {
	quotientHigh, remainder := bits.Div64(0, high, divisor)
	quotientLow, _ := bits.Div64(remainder, low, divisor)
	return quotientHigh, quotientLow
}

func greatestCommonDivisorUint64(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	if a == 0 {
		return 1
	}
	return a
}

// Compare returns -1, 0, or 1 according to the exact ordering of two score times.
func (t ScoreTime) Compare(other ScoreTime) int {
	tNumerator, tDenominator := t.terms()
	otherNumerator, otherDenominator := other.terms()
	leftHigh, leftLow := bits.Mul64(uint64(tNumerator), uint64(otherDenominator))
	rightHigh, rightLow := bits.Mul64(uint64(otherNumerator), uint64(tDenominator))
	if leftHigh != rightHigh {
		if leftHigh < rightHigh {
			return -1
		}
		return 1
	}
	if leftLow < rightLow {
		return -1
	}
	if leftLow > rightLow {
		return 1
	}
	return 0
}

// BarPosition is an exact normalized position in the inclusive range zero through one.
type BarPosition struct {
	value ScoreTime
}

// NewBarPosition constructs a normalized exact bar position.
func NewBarPosition(numerator, denominator int64) (BarPosition, error) {
	value, err := NewScoreTime(numerator, denominator)
	if err != nil {
		return BarPosition{}, err
	}
	one, _ := NewScoreTime(1, 1)
	if value.Compare(one) > 0 {
		return BarPosition{}, fmt.Errorf("bar position %d/%d is outside 0..1", numerator, denominator)
	}
	return BarPosition{value: value}, nil
}

// NewBarPositionFromFloat64 checks a legacy normalized position and preserves
// the exact rational value represented by that float.
func NewBarPositionFromFloat64(value float64) (BarPosition, error) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1 {
		return BarPosition{}, fmt.Errorf("bar position %v is outside finite range 0..1", value)
	}
	ratio := new(big.Rat).SetFloat64(value)
	if ratio == nil || !ratio.Num().IsInt64() || !ratio.Denom().IsInt64() {
		return BarPosition{}, fmt.Errorf("bar position %v exceeds the score-time rational boundary", value)
	}
	return NewBarPosition(ratio.Num().Int64(), ratio.Denom().Int64())
}

// Ratio returns the exact normalized position.
func (p BarPosition) Ratio() ScoreTime {
	return p.value
}

// NoteValue is a conventional whole, half, quarter, or smaller note denominator.
type NoteValue uint16

// DotCount is an unambiguous augmentation-dot count.
type DotCount uint8

// TupletRatio stores written notes entering the time of played notes.
type TupletRatio struct {
	Enters uint16
	Times  uint16
}

// MusicalDuration is the unambiguous semantic duration adapter for legacy Duration.
type MusicalDuration struct {
	Value  NoteValue
	Dots   DotCount
	Tuplet TupletRatio
}

// NewMusicalDuration validates an unambiguous note value, dot count, and tuplet ratio.
func NewMusicalDuration(value NoteValue, dots DotCount, tuplet TupletRatio) (MusicalDuration, error) {
	switch value {
	case 1, 2, 4, 8, 16, 32, 64, 128:
	default:
		return MusicalDuration{}, fmt.Errorf("unsupported note value %d", value)
	}
	if dots > 2 {
		return MusicalDuration{}, fmt.Errorf("dot count %d is outside 0..2", dots)
	}
	if tuplet.Enters == 0 && tuplet.Times == 0 {
		tuplet = TupletRatio{Enters: 1, Times: 1}
	} else if tuplet.Enters == 0 || tuplet.Times == 0 {
		return MusicalDuration{}, fmt.Errorf("tuplet ratio %d:%d must have two positive terms", tuplet.Enters, tuplet.Times)
	}
	return MusicalDuration{Value: value, Dots: dots, Tuplet: tuplet}, nil
}

// ExactScoreTime evaluates the duration exactly in 960-PPQ score ticks.
func (d MusicalDuration) ExactScoreTime() (ScoreTime, error) {
	validated, err := NewMusicalDuration(d.Value, d.Dots, d.Tuplet)
	if err != nil {
		return ScoreTime{}, err
	}
	dotNumerator, dotDenominator := int64(1), int64(1)
	switch validated.Dots {
	case 1:
		dotNumerator, dotDenominator = 3, 2
	case 2:
		dotNumerator, dotDenominator = 7, 4
	}
	numerator, ok := checkedPositiveMultiply(DurationQuarterTime*4, dotNumerator)
	if !ok {
		return ScoreTime{}, fmt.Errorf("duration numerator overflows int64")
	}
	numerator, ok = checkedPositiveMultiply(numerator, int64(validated.Tuplet.Times))
	if !ok {
		return ScoreTime{}, fmt.Errorf("duration numerator overflows int64")
	}
	denominator, ok := checkedPositiveMultiply(int64(validated.Value)*dotDenominator, int64(validated.Tuplet.Enters))
	if !ok {
		return ScoreTime{}, fmt.Errorf("duration denominator overflows int64")
	}
	return NewScoreTime(numerator, denominator)
}

// LegacyDuration converts the semantic duration to the existing public Duration API.
func (d MusicalDuration) LegacyDuration() (Duration, error) {
	validated, err := NewMusicalDuration(d.Value, d.Dots, d.Tuplet)
	if err != nil {
		return Duration{}, err
	}
	if validated.Tuplet.Enters > math.MaxUint8 || validated.Tuplet.Times > math.MaxUint8 {
		return Duration{}, fmt.Errorf("tuplet ratio %d:%d exceeds the legacy uint8 boundary", validated.Tuplet.Enters, validated.Tuplet.Times)
	}
	return Duration{
		Value: uint16(validated.Value), Dotted: validated.Dots == 1, DoubleDotted: validated.Dots == 2,
		TupletEnters: uint8(validated.Tuplet.Enters), TupletTimes: uint8(validated.Tuplet.Times),
	}, nil
}

func greatestCommonDivisor(a, b int64) int64 {
	for b != 0 {
		a, b = b, a%b
	}
	if a == 0 {
		return 1
	}
	return a
}

func checkedPositiveMultiply(a, b int64) (int64, bool) {
	if a < 0 || b < 0 || (a != 0 && b > math.MaxInt64/a) {
		return 0, false
	}
	return a * b, true
}
