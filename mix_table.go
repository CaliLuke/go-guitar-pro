// SPDX-License-Identifier: MIT

package goguitarpro

// MixTableItem describes a mix parameter change.
type MixTableItem struct {
	Value     uint8
	Duration  uint8
	AllTracks bool
}

// WahEffect represents a wah-wah effect.
type WahEffect struct {
	Value   int8
	Display bool
}

// MixTableChange describes a change in mix parameters.
type MixTableChange struct {
	Tremolo    *MixTableItem
	Volume     *MixTableItem
	Balance    *MixTableItem
	Chorus     *MixTableItem
	Reverb     *MixTableItem
	Instrument *MixTableItem
	Phaser     *MixTableItem
	Tempo      *MixTableItem
	Wah        *WahEffect
	TempoName  string
	Rse        RseInstrument
	HideTempo  bool
	UseRse     bool
}

func (s *Song) readMixTableChange(c *cursor) (MixTableChange, error) {
	var mtc MixTableChange
	mtc.HideTempo = true
	if err := s.readMixTableChangeValues(c, &mtc); err != nil {
		return mtc, err
	}
	if err := s.readMixTableChangeDurations(c, &mtc); err != nil {
		return mtc, err
	}
	if versionGTE(s.Version.Number, [3]byte{4, 0, 0}) {
		flags, err := s.readMixTableChangeFlags(c, &mtc)
		if err != nil {
			return mtc, err
		}
		if versionGTE(s.Version.Number, [3]byte{5, 0, 0}) {
			wah, err := s.readWahEffect(c, flags)
			if err != nil {
				return mtc, err
			}
			mtc.Wah = &wah
			if err := s.readRseInstrumentEffect(c, &mtc.Rse); err != nil {
				return mtc, err
			}
		}
	}
	return mtc, nil
}

func (s *Song) readMixTableChangeValues(c *cursor, mtc *MixTableChange) error {
	// instrument
	b, err := c.readSignedByte()
	if err != nil {
		return err
	}
	if b >= 0 {
		mtc.Instrument = &MixTableItem{Value: uint8(b)}
	}
	// RSE instrument GP5
	if s.Version.Number[0] == 5 {
		instr, instrumentErr := s.readRseInstrument(c)
		if instrumentErr != nil {
			return instrumentErr
		}
		mtc.Rse = instr
	}
	if s.Version.Number == [3]byte{5, 0, 0} {
		if skipErr := c.skip(1); skipErr != nil {
			return skipErr
		}
	}
	// volume
	b, err = c.readSignedByte()
	if err != nil {
		return err
	}
	if b >= 0 {
		mtc.Volume = &MixTableItem{Value: uint8(b)}
	}
	// balance
	b, err = c.readSignedByte()
	if err != nil {
		return err
	}
	if b >= 0 {
		mtc.Balance = &MixTableItem{Value: uint8(b)}
	}
	// chorus
	b, err = c.readSignedByte()
	if err != nil {
		return err
	}
	if b >= 0 {
		mtc.Chorus = &MixTableItem{Value: uint8(b)}
	}
	// reverb
	b, err = c.readSignedByte()
	if err != nil {
		return err
	}
	if b >= 0 {
		mtc.Reverb = &MixTableItem{Value: uint8(b)}
	}
	// phaser
	b, err = c.readSignedByte()
	if err != nil {
		return err
	}
	if b >= 0 {
		mtc.Phaser = &MixTableItem{Value: uint8(b)}
	}
	// tremolo
	b, err = c.readSignedByte()
	if err != nil {
		return err
	}
	if b >= 0 {
		mtc.Tremolo = &MixTableItem{Value: uint8(b)}
	}
	// tempo
	if versionGTE(s.Version.Number, [3]byte{5, 0, 0}) {
		tn, tempoNameErr := c.readIntByteSizeString()
		if tempoNameErr != nil {
			return tempoNameErr
		}
		mtc.TempoName = tn
	}
	tempo, err := c.readInt()
	if err != nil {
		return err
	}
	if tempo >= 0 {
		mtc.Tempo = &MixTableItem{Value: uint8(tempo)}
	}
	return nil
}

func (s *Song) readMixTableChangeDurations(c *cursor, mtc *MixTableChange) error {
	if mtc.Volume != nil {
		d, err := c.readSignedByte()
		if err != nil {
			return err
		}
		mtc.Volume.Duration = uint8(d)
	}
	if mtc.Balance != nil {
		d, err := c.readSignedByte()
		if err != nil {
			return err
		}
		mtc.Balance.Duration = uint8(d)
	}
	if mtc.Chorus != nil {
		d, err := c.readSignedByte()
		if err != nil {
			return err
		}
		mtc.Chorus.Duration = uint8(d)
	}
	if mtc.Reverb != nil {
		d, err := c.readSignedByte()
		if err != nil {
			return err
		}
		mtc.Reverb.Duration = uint8(d)
	}
	if mtc.Phaser != nil {
		d, err := c.readSignedByte()
		if err != nil {
			return err
		}
		mtc.Phaser.Duration = uint8(d)
	}
	if mtc.Tremolo != nil {
		d, err := c.readSignedByte()
		if err != nil {
			return err
		}
		mtc.Tremolo.Duration = uint8(d)
	}
	if mtc.Tempo != nil {
		d, err := c.readSignedByte()
		if err != nil {
			return err
		}
		mtc.Tempo.Duration = uint8(d)
		mtc.HideTempo = false
		if versionGreaterThan(s.Version.Number, [3]byte{5, 0, 0}) {
			ht, err := c.readBool()
			if err != nil {
				return err
			}
			mtc.HideTempo = ht
		}
	}
	return nil
}

func (s *Song) readMixTableChangeFlags(c *cursor, mtc *MixTableChange) (int8, error) {
	flags, err := c.readSignedByte()
	if err != nil {
		return 0, err
	}
	if mtc.Volume != nil {
		mtc.Volume.AllTracks = (flags & 0x01) == 0x01
	}
	if mtc.Balance != nil {
		mtc.Balance.AllTracks = (flags & 0x02) == 0x02
	}
	if mtc.Chorus != nil {
		mtc.Chorus.AllTracks = (flags & 0x04) == 0x04
	}
	if mtc.Reverb != nil {
		mtc.Reverb.AllTracks = (flags & 0x08) == 0x08
	}
	if mtc.Phaser != nil {
		mtc.Phaser.AllTracks = (flags & 0x10) == 0x10
	}
	if mtc.Tremolo != nil {
		mtc.Tremolo.AllTracks = (flags & 0x20) == 0x20
	}
	if versionGTE(s.Version.Number, [3]byte{5, 0, 0}) {
		mtc.UseRse = (flags & 0x40) == 0x40
	}
	return flags, nil
}

func (s *Song) readWahEffect(c *cursor, flags int8) (WahEffect, error) {
	val, err := c.readSignedByte()
	if err != nil {
		return WahEffect{}, err
	}
	return WahEffect{
		Value:   val,
		Display: (flags & -0x80) == -0x80,
	}, nil
}
