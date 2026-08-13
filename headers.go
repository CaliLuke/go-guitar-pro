// SPDX-License-Identifier: MIT

package goguitarpro

// Version holds file version information.
type Version struct {
	Data      string
	Number    [3]byte
	Clipboard bool
}

// Clipboard holds clipboard data for clipboard-format files.
type Clipboard struct {
	StartMeasure int32
	StopMeasure  int32
	StartTrack   int32
	StopTrack    int32
	StartBeat    int32
	StopBeat     int32
	SubBarCopy   bool
}

// MeasureHeader contains metadata for measures over multiple tracks.
type MeasureHeader struct {
	Marker            *Marker
	Direction         *DirectionSign
	Start             int64
	Tempo             int32
	TimeSignature     TimeSignature
	Number            uint16
	KeySignature      KeySignature
	RepeatOpen        bool
	RepeatAlternative uint8
	RepeatClose       int8
	TripletFeel       TripletFeel
	DoubleBar         bool
}

func defaultMeasureHeader() MeasureHeader {
	return MeasureHeader{
		Number:        1,
		Start:         DurationQuarterTime,
		RepeatClose:   -1,
		TimeSignature: defaultTimeSignature(),
	}
}

func (mh *MeasureHeader) length() int64 {
	return int64(mh.TimeSignature.Numerator) * int64(mh.TimeSignature.Denominator.time())
}

// Marker is a marker annotation for beats.
type Marker struct {
	Title string
	Color int32
}

func (s *Song) readClipboard(c *cursor) error {
	if !s.Version.Clipboard {
		return nil
	}
	cb := Clipboard{}
	var err error
	cb.StartMeasure, err = c.readInt()
	if err != nil {
		return err
	}
	cb.StopMeasure, err = c.readInt()
	if err != nil {
		return err
	}
	cb.StartTrack, err = c.readInt()
	if err != nil {
		return err
	}
	cb.StopTrack, err = c.readInt()
	if err != nil {
		return err
	}
	if s.Version.Number[0] == 5 {
		cb.StartBeat, err = c.readInt()
		if err != nil {
			return err
		}
		cb.StopBeat, err = c.readInt()
		if err != nil {
			return err
		}
		sbc, err := c.readInt()
		if err != nil {
			return err
		}
		cb.SubBarCopy = sbc != 0
	}
	s.Clipboard = &cb
	return nil
}

func (s *Song) readMeasureHeaders(c *cursor, measureCount int) error {
	var previous *MeasureHeader
	for i := 1; i <= measureCount; i++ {
		mh, err := s.readMeasureHeader(c, i, previous)
		if err != nil {
			return err
		}
		s.MeasureHeaders = append(s.MeasureHeaders, mh)
		prev := s.MeasureHeaders[len(s.MeasureHeaders)-1]
		previous = &prev
	}
	return nil
}

func (s *Song) readMeasureHeadersV5(c *cursor, measureCount int, signs, fromSigns map[DirectionSign]int16) error {
	var previous *MeasureHeader
	for i := 1; i <= measureCount; i++ {
		mh, err := s.readMeasureHeaderV5(c, i, previous)
		if err != nil {
			return err
		}
		s.MeasureHeaders = append(s.MeasureHeaders, mh)
		prev := s.MeasureHeaders[len(s.MeasureHeaders)-1]
		previous = &prev
	}
	// Apply directions
	for sign, measure := range signs {
		if measure > 0 && int(measure)-1 < len(s.MeasureHeaders) {
			d := sign
			s.MeasureHeaders[measure-1].Direction = &d
		}
	}
	for sign, measure := range fromSigns {
		if measure > 0 && int(measure)-1 < len(s.MeasureHeaders) {
			d := sign
			s.MeasureHeaders[measure-1].Direction = &d
		}
	}
	return nil
}

// readMeasureHeaderWithFlags reads a measure header and returns the flags byte.
func (s *Song) readMeasureHeaderWithFlags(c *cursor, number int, previous *MeasureHeader) (MeasureHeader, byte, error) {
	flag, err := c.readByte()
	if err != nil {
		return MeasureHeader{}, 0, err
	}
	mh, err := s.readMeasureHeaderBody(c, flag, number, previous)
	return mh, flag, err
}

func (s *Song) readMeasureHeader(c *cursor, number int, previous *MeasureHeader) (MeasureHeader, error) {
	flag, err := c.readByte()
	if err != nil {
		return MeasureHeader{}, err
	}
	return s.readMeasureHeaderBody(c, flag, number, previous)
}

func (s *Song) readMeasureHeaderBody(c *cursor, flag byte, number int, previous *MeasureHeader) (MeasureHeader, error) {
	mh := defaultMeasureHeader()
	mh.Number = uint16(number)
	mh.Start = 0
	mh.TripletFeel = s.TripletFeel

	// Numerator
	if (flag & 0x01) == 0x01 {
		num, err := c.readSignedByte()
		if err != nil {
			return mh, err
		}
		mh.TimeSignature.Numerator = num
	} else if number > 1 && previous != nil {
		mh.TimeSignature.Numerator = previous.TimeSignature.Numerator
	}
	// Denominator
	if (flag & 0x02) == 0x02 {
		den, err := c.readSignedByte()
		if err != nil {
			return mh, err
		}
		mh.TimeSignature.Denominator.Value = uint16(den)
	} else if number > 1 && previous != nil {
		mh.TimeSignature.Denominator = previous.TimeSignature.Denominator
	}

	mh.RepeatOpen = (flag & 0x04) == 0x04
	if (flag & 0x08) == 0x08 {
		rc, err := c.readSignedByte()
		if err != nil {
			return mh, err
		}
		mh.RepeatClose = rc
	}
	if (flag & 0x10) == 0x10 {
		if s.Version.Number[0] == 5 {
			ra, err := c.readByte()
			if err != nil {
				return mh, err
			}
			mh.RepeatAlternative = ra
		} else {
			ra, err := s.readRepeatAlternative(c)
			if err != nil {
				return mh, err
			}
			mh.RepeatAlternative = ra
		}
	}
	if (flag & 0x20) == 0x20 {
		m, err := readMarker(c)
		if err != nil {
			return mh, err
		}
		mh.Marker = &m
	}
	if (flag & 0x40) == 0x40 {
		key, err := c.readSignedByte()
		if err != nil {
			return mh, err
		}
		minor, err := c.readSignedByte()
		if err != nil {
			return mh, err
		}
		mh.KeySignature.Key = key
		mh.KeySignature.IsMinor = minor != 0
	} else if mh.Number > 1 && previous != nil {
		mh.KeySignature = previous.KeySignature
	}
	mh.DoubleBar = (flag & 0x80) == 0x80
	return mh, nil
}

func (s *Song) readMeasureHeaderV5(c *cursor, number int, previous *MeasureHeader) (MeasureHeader, error) {
	if previous != nil {
		if err := c.skip(1); err != nil {
			return MeasureHeader{}, err
		}
	}
	mh, flags, err := s.readMeasureHeaderWithFlags(c, number, previous)
	if err != nil {
		return mh, err
	}
	if mh.RepeatClose > -1 {
		mh.RepeatClose--
	}
	if (flags & 0x03) == 0x03 {
		for i := 0; i < 4; i++ {
			b, beamErr := c.readByte()
			if beamErr != nil {
				return mh, beamErr
			}
			mh.TimeSignature.Beams[i] = b
		}
	} else if previous != nil {
		mh.TimeSignature.Beams = previous.TimeSignature.Beams
	}
	if (flags & 0x10) == 0 {
		if skipErr := c.skip(1); skipErr != nil {
			return mh, skipErr
		}
	}
	tfByte, err := c.readByte()
	if err != nil {
		return mh, err
	}
	mh.TripletFeel = TripletFeel(int8(tfByte))
	return mh, nil
}

func readMarker(c *cursor) (Marker, error) {
	title, err := c.readIntSizeString()
	if err != nil {
		return Marker{}, err
	}
	color, err := c.readColor()
	if err != nil {
		return Marker{}, err
	}
	return Marker{Title: title, Color: color}, nil
}

func (s *Song) readRepeatAlternative(c *cursor) (uint8, error) {
	value, err := c.readByte()
	if err != nil {
		return 0, err
	}
	var existingAlternative uint16
	for i := len(s.MeasureHeaders) - 1; i >= 0; i-- {
		if s.MeasureHeaders[i].RepeatOpen {
			break
		}
		existingAlternative |= uint16(s.MeasureHeaders[i].RepeatAlternative)
	}
	return uint8(((1 << value) - 1) ^ existingAlternative), nil
}

func (s *Song) readDirections(c *cursor) (map[DirectionSign]int16, map[DirectionSign]int16, error) {
	signs := make(map[DirectionSign]int16)
	fromSigns := make(map[DirectionSign]int16)

	signOrder := []DirectionSign{
		DirectionSignCoda, DirectionSignDoubleCoda, DirectionSignSegno, DirectionSignSegnoSegno, DirectionSignFine,
	}
	fromSignOrder := []DirectionSign{
		DirectionSignDaCapo, DirectionSignDaCapoAlCoda, DirectionSignDaCapoAlDoubleCoda, DirectionSignDaCapoAlFine,
		DirectionSignDaSegno, DirectionSignDaSegnoAlCoda, DirectionSignDaSegnoAlDoubleCoda, DirectionSignDaSegnoAlFine,
		DirectionSignDaSegnoSegno, DirectionSignDaSegnoSegnoAlCoda, DirectionSignDaSegnoSegnoAlDoubleCoda, DirectionSignDaSegnoSegnoAlFine,
		DirectionSignDaCoda, DirectionSignDaDoubleCoda,
	}

	for _, sign := range signOrder {
		val, err := c.readShort()
		if err != nil {
			return nil, nil, err
		}
		signs[sign] = val
	}
	for _, sign := range fromSignOrder {
		val, err := c.readShort()
		if err != nil {
			return nil, nil, err
		}
		fromSigns[sign] = val
	}
	return signs, fromSigns, nil
}
