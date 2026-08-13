// SPDX-License-Identifier: MIT

package goguitarpro

// RseEqualizer represents an equalizer.
type RseEqualizer struct {
	Knobs []float32
	Gain  float32
}

// RseMasterEffect represents the master effect.
type RseMasterEffect struct {
	Equalizer RseEqualizer
	Volume    float32
	Reverb    float32
}

// RseInstrument represents an RSE instrument.
type RseInstrument struct {
	EffectCategory string
	Effect         string
	Instrument     int16
	Unknown        int16
	SoundBank      int16
	EffectNumber   int16
}

// TrackRse represents track RSE settings.
type TrackRse struct {
	Instrument       RseInstrument
	Equalizer        RseEqualizer
	Humanize         uint8
	AutoAccentuation Accentuation
}

func (s *Song) readRseMasterEffect(c *cursor) (RseMasterEffect, error) {
	me := RseMasterEffect{Equalizer: RseEqualizer{Knobs: make([]float32, 10)}}
	if versionGreaterThan(s.Version.Number, [3]byte{5, 0, 0}) {
		vol, err := c.readInt()
		if err != nil {
			return me, err
		}
		me.Volume = float32(vol)
		// skip unknown int
		if _, readErr := c.readInt(); readErr != nil {
			return me, readErr
		}
		eq, err := s.readRseEqualizer(c, 11)
		if err != nil {
			return me, err
		}
		me.Equalizer = eq
	}
	return me, nil
}

func (s *Song) readRseEqualizer(c *cursor, knobs int) (RseEqualizer, error) {
	e := RseEqualizer{}
	for i := 0; i < knobs; i++ {
		b, err := c.readSignedByte()
		if err != nil {
			return e, err
		}
		e.Knobs = append(e.Knobs, -float32(b)/10.0)
	}
	return e, nil
}

func (s *Song) readTrackRse(c *cursor, track *Track) error {
	h, err := c.readByte()
	if err != nil {
		return err
	}
	track.Rse.Humanize = h
	// Skip 24 bytes (6 ints)
	if skipErr := c.skip(24); skipErr != nil {
		return skipErr
	}
	instr, err := s.readRseInstrument(c)
	if err != nil {
		return err
	}
	track.Rse.Instrument = instr
	if versionGreaterThan(s.Version.Number, [3]byte{5, 0, 0}) {
		eq, err := s.readRseEqualizer(c, 4)
		if err != nil {
			return err
		}
		track.Rse.Equalizer = eq
		if err := s.readRseInstrumentEffect(c, &track.Rse.Instrument); err != nil {
			return err
		}
	}
	return nil
}

func (s *Song) readRseInstrument(c *cursor) (RseInstrument, error) {
	var instr RseInstrument
	i, err := c.readInt()
	if err != nil {
		return instr, err
	}
	instr.Instrument = int16(i)
	u, err := c.readInt()
	if err != nil {
		return instr, err
	}
	instr.Unknown = int16(u)
	sb, err := c.readInt()
	if err != nil {
		return instr, err
	}
	instr.SoundBank = int16(sb)
	if s.Version.Number == [3]byte{5, 0, 0} {
		en, err := c.readShort()
		if err != nil {
			return instr, err
		}
		instr.EffectNumber = en
		if err := c.skip(1); err != nil {
			return instr, err
		}
	} else {
		en, err := c.readInt()
		if err != nil {
			return instr, err
		}
		instr.EffectNumber = int16(en)
	}
	return instr, nil
}

func (s *Song) readRseInstrumentEffect(c *cursor, instr *RseInstrument) error {
	if versionGreaterThan(s.Version.Number, [3]byte{5, 0, 0}) {
		eff, err := c.readIntByteSizeString()
		if err != nil {
			return err
		}
		instr.Effect = eff
		cat, err := c.readIntByteSizeString()
		if err != nil {
			return err
		}
		instr.EffectCategory = cat
	}
	return nil
}
