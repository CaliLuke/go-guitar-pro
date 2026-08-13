// SPDX-License-Identifier: MIT

package goguitarpro

// DefaultPercussionChannel is the zero-based percussion channel for General MIDI.
const DefaultPercussionChannel uint8 = 9

// MidiChannel describes MIDI playback data for a track.
type MidiChannel struct {
	Channel       uint8
	EffectChannel uint8
	Instrument    int32
	Volume        int8
	Balance       int8
	Chorus        int8
	Reverb        int8
	Phaser        int8
	Tremolo       int8
	Bank          uint8
}

func defaultMidiChannel() MidiChannel {
	return MidiChannel{
		Instrument: 25,
		Volume:     104,
		Balance:    64,
	}
}

func (mc *MidiChannel) isPercussionChannel() bool {
	return (mc.Channel % 16) == DefaultPercussionChannel
}

// readMidiChannels reads all 64 MIDI channels.
func (s *Song) readMidiChannels(c *cursor) error {
	for i := uint8(0); i < 64; i++ {
		ch, err := s.readMidiChannel(c, i)
		if err != nil {
			return err
		}
		s.Channels = append(s.Channels, ch)
	}
	return nil
}

func (s *Song) readMidiChannel(c *cursor, channel uint8) (MidiChannel, error) {
	instrument, err := c.readInt()
	if err != nil {
		return MidiChannel{}, err
	}
	mc := defaultMidiChannel()
	mc.Channel = channel
	mc.EffectChannel = channel
	mc.Volume, err = c.readSignedByte()
	if err != nil {
		return mc, err
	}
	mc.Balance, err = c.readSignedByte()
	if err != nil {
		return mc, err
	}
	mc.Chorus, err = c.readSignedByte()
	if err != nil {
		return mc, err
	}
	mc.Reverb, err = c.readSignedByte()
	if err != nil {
		return mc, err
	}
	mc.Phaser, err = c.readSignedByte()
	if err != nil {
		return mc, err
	}
	mc.Tremolo, err = c.readSignedByte()
	if err != nil {
		return mc, err
	}
	// Set instrument
	if instrument == -1 && mc.isPercussionChannel() {
		mc.Instrument = 0
	} else {
		mc.Instrument = instrument
	}
	// Skip 2 bytes (backward compatibility with v3.0)
	if err := c.skip(2); err != nil {
		return mc, err
	}
	return mc, nil
}

// readChannel reads a MIDI channel reference from a track.
func (s *Song) readChannel(c *cursor, track *Track) error {
	index, err := c.readInt()
	if err != nil {
		return err
	}
	index--
	effectChannel, err := c.readInt()
	if err != nil {
		return err
	}
	effectChannel--
	idx := int(index)
	if index >= 0 && idx < len(s.Channels) {
		track.ChannelIndex = idx
		if s.Channels[idx].Instrument < 0 {
			s.Channels[idx].Instrument = 0
		}
		if s.Channels[idx].isPercussionChannel() {
			track.PercussionTrack = true
		} else {
			s.Channels[idx].EffectChannel = uint8(effectChannel)
		}
	}
	return nil
}
