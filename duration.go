// SPDX-License-Identifier: MIT

package goguitarpro

import "math"

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

func (d *Duration) convertTime(time uint32) uint32 {
	if d.TupletEnters == 0 || d.TupletTimes == 0 {
		return time
	}
	return time * uint32(d.TupletTimes) / uint32(d.TupletEnters)
}

func (d *Duration) time() uint32 {
	if d.Value == 0 {
		return 0
	}
	result := math.Trunc(float64(DurationQuarterTime) * 4.0 / float64(d.Value))
	switch {
	case d.Dotted:
		result += math.Trunc(result / 2.0)
	case d.DoubleDotted:
		result += math.Trunc(result/4.0) * 3.0
	}
	return d.convertTime(uint32(result))
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
		}
	}
	return d, nil
}
