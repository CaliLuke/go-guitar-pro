// SPDX-License-Identifier: MIT

package goguitarpro

// DefaultPercussionChannel is the zero-based percussion channel for General MIDI.
const DefaultPercussionChannel uint8 = 9

// MidiChannel describes MIDI playback data for a track.
type MidiChannel struct {
	Channel       uint8
	EffectChannel uint8
	Instrument    int32
	// Volume and Balance use the normalized MIDI controller range 0 through 127.
	Volume  int8
	Balance int8
	Chorus  int8
	Reverb  int8
	Phaser  int8
	Tremolo int8
	Bank    uint8
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
		ch, err := s.readMidiChannel(c, i*2)
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
	mc.EffectChannel = channel + 1
	rawVolume, err := c.readSignedByte()
	if err != nil {
		return mc, err
	}
	mc.Volume = normalizeBinaryMixerValue(rawVolume)
	rawBalance, err := c.readSignedByte()
	if err != nil {
		return mc, err
	}
	mc.Balance = normalizeBinaryMixerValue(rawBalance)
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
	mc.Instrument = instrument
	// Skip 2 bytes (backward compatibility with v3.0)
	if err := c.skip(2); err != nil {
		return mc, err
	}
	return mc, nil
}

// normalizeBinaryMixerValue converts the 0-16 mixer scale stored by GP3-5
// into the same 0-127 controller range exposed for GPIF files.
func normalizeBinaryMixerValue(value int8) int8 {
	if value <= 0 {
		return 0
	}
	if value >= 16 {
		return 127
	}
	return int8((int(value)*127 + 8) / 16)
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
		if track.PercussionTrack || s.Channels[idx].isPercussionChannel() {
			track.PercussionTrack = true
			s.Channels[idx].Channel = DefaultPercussionChannel
			s.Channels[idx].EffectChannel = DefaultPercussionChannel
			s.Channels[idx].Instrument = 0
		} else {
			if s.Channels[idx].Instrument < 0 {
				s.Channels[idx].Instrument = 0
			}
			s.Channels[idx].EffectChannel = uint8(effectChannel)
		}
	}
	return nil
}

// consolidateTrackChannels applies the same playable-channel guarantees as
// Guitar Pro importers: channel 10 is reserved for percussion and every
// melodic primary/effect channel is unique. GP3-5 playback records describe
// channel pairs, while tracks reference those records independently.
func (s *Song) consolidateTrackChannels() {
	used := map[uint8]bool{DefaultPercussionChannel: true}
	for index := range s.Tracks {
		track := &s.Tracks[index]
		if track.ChannelIndex < 0 || track.ChannelIndex >= len(s.Channels) {
			continue
		}
		channel := &s.Channels[track.ChannelIndex]
		if track.PercussionTrack {
			channel.Channel = DefaultPercussionChannel
			channel.EffectChannel = DefaultPercussionChannel
			channel.Instrument = 0
			continue
		}

		channel.Channel = nextAvailableChannel(channel.Channel, used)
		used[channel.Channel] = true
		channel.EffectChannel = nextAvailableChannel(channel.EffectChannel, used)
		used[channel.EffectChannel] = true
	}
}

func nextAvailableChannel(channel uint8, used map[uint8]bool) uint8 {
	for channel == DefaultPercussionChannel || used[channel] {
		channel++
	}
	return channel
}
