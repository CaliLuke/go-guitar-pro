// SPDX-License-Identifier: MIT

package goguitarpro

// TrackSettings contains settings of the track.
type TrackSettings struct {
	Tablature       bool
	Notation        bool
	DiagramAreBelow bool
	ShowRhythm      bool
	ForceHorizontal bool
	ForceChannels   bool
	DiagramList     bool
	DiagramInScore  bool
	AutoLetRing     bool
	AutoBrush       bool
	ExtendRhythmic  bool
}

// Track represents a track.
type Track struct {
	Name                      string
	Measures                  []Measure
	Strings                   []GuitarString
	Rse                       TrackRse
	ChannelIndex              int
	Offset                    int32
	Number                    int32
	Color                     int32
	Settings                  TrackSettings
	Mute                      bool
	TwelveStringedGuitarTrack bool
	BanjoTrack                bool
	Port                      uint8
	FretCount                 uint8
	IndicateTuning            bool
	UseRse                    bool
	PercussionTrack           bool
	Visible                   bool
	Solo                      bool
}

// GuitarString represents a guitar string with tuning.
type GuitarString struct {
	Number int8
	Value  int8
}

func defaultTrack() Track {
	return Track{
		Number:    1,
		Visible:   true,
		Name:      "Track 1",
		FretCount: 24,
		Color:     0xff0000,
		Port:      1,
		Strings: []GuitarString{
			{1, 64}, {2, 59}, {3, 55}, {4, 50}, {5, 45}, {6, 40},
		},
		Settings: TrackSettings{
			Tablature:   true,
			Notation:    true,
			DiagramList: true,
		},
	}
}

func (s *Song) readTracks(c *cursor, trackCount int) error {
	for i := 0; i < trackCount; i++ {
		if err := s.readTrack(c, i); err != nil {
			return err
		}
	}
	return nil
}

func (s *Song) readTrack(c *cursor, number int) error {
	track := defaultTrack()
	track.Number = int32(number)

	flags1, err := c.readByte()
	if err != nil {
		return err
	}
	track.PercussionTrack = (flags1 & 0x01) == 0x01
	track.TwelveStringedGuitarTrack = (flags1 & 0x02) == 0x02
	track.BanjoTrack = (flags1 & 0x04) == 0x04

	name, err := c.readByteSizeString(40)
	if err != nil {
		return err
	}
	track.Name = name

	stringCount, err := c.readInt()
	if err != nil {
		return err
	}
	track.Strings = nil
	for i := int8(0); i < 7; i++ {
		tuning, tuningErr := c.readInt()
		if tuningErr != nil {
			return tuningErr
		}
		if int8(stringCount) > i {
			track.Strings = append(track.Strings, GuitarString{Number: i + 1, Value: int8(tuning)})
		}
	}

	port, err := c.readInt()
	if err != nil {
		return err
	}
	track.Port = uint8(port)

	if channelErr := s.readChannel(c); channelErr != nil {
		return channelErr
	}
	if number < len(s.Channels) && s.Channels[number].Channel == 9 {
		track.PercussionTrack = true
	}

	fretCount, err := c.readInt()
	if err != nil {
		return err
	}
	track.FretCount = uint8(fretCount)

	offset, err := c.readInt()
	if err != nil {
		return err
	}
	track.Offset = offset

	color, err := c.readColor()
	if err != nil {
		return err
	}
	track.Color = color

	s.Tracks = append(s.Tracks, track)
	return nil
}

func (s *Song) readTracksV5(c *cursor, trackCount int) error {
	for i := 0; i < trackCount; i++ {
		if err := s.readTrackV5(c, i); err != nil {
			return err
		}
	}
	// Skip trailing bytes
	n := 1
	if s.Version.Number == [3]byte{5, 0, 0} {
		n = 2
	}
	return c.skip(n)
}

func (s *Song) readTrackV5(c *cursor, number int) error {
	track := defaultTrack()
	track.Number = int32(number)

	if number == 0 || s.Version.Number == [3]byte{5, 0, 0} {
		if err := c.skip(1); err != nil {
			return err
		}
	}
	flags1, err := c.readByte()
	if err != nil {
		return err
	}
	track.PercussionTrack = (flags1 & 0x01) == 0x01
	track.BanjoTrack = (flags1 & 0x02) == 0x02
	track.Visible = (flags1 & 0x04) == 0x04
	track.Solo = (flags1 & 0x10) == 0x10
	track.Mute = (flags1 & 0x20) == 0x20
	track.UseRse = (flags1 & 0x40) == 0x40
	track.IndicateTuning = (flags1 & 0x80) == 0x80

	name, err := c.readByteSizeString(40)
	if err != nil {
		return err
	}
	track.Name = name

	stringCount, err := c.readInt()
	if err != nil {
		return err
	}
	track.Strings = nil
	for i := int8(0); i < 7; i++ {
		tuning, tuningErr := c.readInt()
		if tuningErr != nil {
			return tuningErr
		}
		if int8(stringCount) > i {
			track.Strings = append(track.Strings, GuitarString{Number: i + 1, Value: int8(tuning)})
		}
	}

	port, err := c.readInt()
	if err != nil {
		return err
	}
	track.Port = uint8(port)

	if channelErr := s.readChannel(c); channelErr != nil {
		return channelErr
	}
	if number < len(s.Channels) && s.Channels[number].Channel == 9 {
		track.PercussionTrack = true
	}

	fretCount, err := c.readInt()
	if err != nil {
		return err
	}
	track.FretCount = uint8(fretCount)

	offset, err := c.readInt()
	if err != nil {
		return err
	}
	track.Offset = offset

	color, err := c.readColor()
	if err != nil {
		return err
	}
	track.Color = color

	flags2, err := c.readShort()
	if err != nil {
		return err
	}
	track.Settings.Tablature = (flags2 & 0x0001) == 0x0001
	track.Settings.Notation = (flags2 & 0x0002) == 0x0002
	track.Settings.DiagramAreBelow = (flags2 & 0x0004) == 0x0004
	track.Settings.ShowRhythm = (flags2 & 0x0008) == 0x0008
	track.Settings.ForceHorizontal = (flags2 & 0x0010) == 0x0010
	track.Settings.ForceChannels = (flags2 & 0x0020) == 0x0020
	track.Settings.DiagramList = (flags2 & 0x0040) == 0x0040
	track.Settings.DiagramInScore = (flags2 & 0x0080) == 0x0080
	track.Settings.AutoLetRing = (flags2 & 0x0200) == 0x0200
	track.Settings.AutoBrush = (flags2 & 0x0400) == 0x0400
	track.Settings.ExtendRhythmic = (flags2 & 0x0800) == 0x0800

	accentByte, err := c.readByte()
	if err != nil {
		return err
	}
	track.Rse.AutoAccentuation = Accentuation(accentByte)

	if number < len(s.Channels) {
		bankByte, err := c.readByte()
		if err != nil {
			return err
		}
		s.Channels[number].Bank = bankByte
	} else {
		if _, err := c.readByte(); err != nil {
			return err
		}
	}

	if err := s.readTrackRse(c, &track); err != nil {
		return err
	}

	s.Tracks = append(s.Tracks, track)
	return nil
}
