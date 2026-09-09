// SPDX-License-Identifier: MIT

package goguitarpro

// Duration constants use note values from Guitar Pro and internal tick timing.
const (
	DurationQuarterTime         int64 = 960
	DurationQuarter             uint8 = 4
	DurationEighth              uint8 = 8
	DurationSixteenth           uint8 = 16
	DurationThirtySecond        uint8 = 32
	DurationSixtyFourth         uint8 = 64
	DurationHundredTwentyEighth uint8 = 128
)

// Duration represents a beat duration.
type Duration struct {
	Value        uint16
	Dotted       bool
	DoubleDotted bool
	TupletEnters uint8
	TupletTimes  uint8
}

func defaultDuration() Duration {
	return Duration{
		Value:        uint16(DurationQuarter),
		TupletEnters: 1,
		TupletTimes:  1,
	}
}

func (d *Duration) time() uint32 {
	exact, err := d.ExactScoreTime()
	if err != nil || exact.FloorTicks() > int64(^uint32(0)) {
		return 0
	}
	return uint32(exact.FloorTicks())
}

// MusicalDuration converts the legacy duration fields to an unambiguous semantic value.
// If both legacy dot flags are set, the double-dot flag takes precedence.
func (d Duration) MusicalDuration() (MusicalDuration, error) {
	dots := DotCount(0)
	if d.DoubleDotted {
		dots = 2
	} else if d.Dotted {
		dots = 1
	}
	return NewMusicalDuration(NoteValue(d.Value), dots, TupletRatio{
		Enters: uint16(d.TupletEnters), Times: uint16(d.TupletTimes),
	})
}

// ExactScoreTime evaluates the legacy duration without intermediate tick truncation.
func (d Duration) ExactScoreTime() (ScoreTime, error) {
	semantic, err := d.MusicalDuration()
	if err != nil {
		return ScoreTime{}, err
	}
	return semantic.ExactScoreTime()
}

// TimeSignature represents a time signature.
type TimeSignature struct {
	Numerator   int8
	Denominator Duration
	Beams       [4]uint8
}

func defaultTimeSignature() TimeSignature {
	return TimeSignature{
		Numerator:   4,
		Denominator: defaultDuration(),
		Beams:       [4]uint8{2, 2, 2, 2},
	}
}

// KeySignature represents a key signature.
type KeySignature struct {
	Key     int8
	IsMinor bool
}

// readDuration reads beat duration from cursor.
func readDuration(c *cursor, flags byte) (Duration, error) {
	b, err := c.readSignedByte()
	if err != nil {
		return Duration{}, err
	}
	d := defaultDuration()
	d.Value = 1 << (b + 2)
	d.Dotted = (flags & 0x01) == 0x01
	if (flags & 0x20) == 0x20 {
		iTuplet, err := c.readInt()
		if err != nil {
			return Duration{}, err
		}
		switch iTuplet {
		case 3:
			d.TupletEnters, d.TupletTimes = 3, 2
		case 5:
			d.TupletEnters, d.TupletTimes = 5, 4
		case 6:
			d.TupletEnters, d.TupletTimes = 6, 4
		case 7:
			d.TupletEnters, d.TupletTimes = 7, 4
		case 9:
			d.TupletEnters, d.TupletTimes = 9, 8
		case 10:
			d.TupletEnters, d.TupletTimes = 10, 8
		case 11:
			d.TupletEnters, d.TupletTimes = 11, 8
		case 12:
			d.TupletEnters, d.TupletTimes = 12, 8
		case 13:
			d.TupletEnters, d.TupletTimes = 13, 8
		default:
			c.report(diagnosticSource("Binary.Duration.Tuplet.Unsupported", "rhythm", ParseDiagnosticUnsupportedFeature), "binary tuplet discriminator is not supported")
		}
	}
	return d, nil
}
