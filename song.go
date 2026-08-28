// SPDX-License-Identifier: MIT

package goguitarpro

import "fmt"

// GPIFBackingTrackSampleRate is the fixed frame rate Guitar Pro uses for audio synchronization.
const GPIFBackingTrackSampleRate = 44100

// Song is the root data structure for a parsed Guitar Pro file.
type Song struct {
	Clipboard      *Clipboard
	currentTrack   *int
	Name           string
	Artist         string
	Writer         string
	Album          string
	Words          string
	Author         string
	Date           string
	Copyright      string
	TempoName      string
	Transcriber    string
	Subtitle       string
	Comments       string
	Instructions   string
	Version        Version
	Notice         []string
	Channels       []MidiChannel
	MeasureHeaders []MeasureHeader
	// Anacrusis reports whether the first measure is a pickup measure.
	Anacrusis bool
	Tracks    []Track
	// BackingTrack contains the GPIF audio track and its embedded bytes when present.
	BackingTrack *BackingTrack
	// SyncPoints contains each score-to-backing-track anchor in a GPIF file.
	SyncPoints []SyncPoint
	// TempoAutomations contains each tempo change in a GPIF file.
	TempoAutomations []TempoAutomation
	// VolumeAutomations contains each track gain point in a GPIF file.
	VolumeAutomations []VolumeAutomation
	Lyrics            Lyrics
	MasterEffect      RseMasterEffect
	PageSetup         PageSetup
	Tempo             int16
	Key               KeySignature
	HideTempo         bool
	TripletFeel       TripletFeel
}

// BackingTrack describes the external audio attached to a GPIF score.
type BackingTrack struct {
	Name             string
	Source           string
	AssetID          string
	OriginalFilePath string
	OriginalFileSHA1 string
	EmbeddedFilePath string
	AudioData        []byte
	// FramePadding is the signed 44.1 kHz frame offset stored by Guitar Pro.
	FramePadding int64
	Enabled      bool
}

// SyncPoint describes one GPIF score-to-backing-track anchor.
type SyncPoint struct {
	// Bar is the zero-based master-bar index.
	Bar int
	// Position is the point position as a fraction of the bar length.
	Position float64
	// BarOccurrence identifies a particular playback occurrence when repeats are present.
	BarOccurrence int
	// FrameOffset is the raw 44.1 kHz project-frame position persisted for the point.
	FrameOffset int64
	// MediaTimeMS is the absolute backing-track position after subtracting FramePadding.
	MediaTimeMS float64
	// ModifiedTempo and OriginalTempo are Guitar Pro's persisted derived tempo metadata.
	ModifiedTempo float64
	OriginalTempo float64
	Linear        bool
	Visible       bool
}

// VolumeAutomation describes one track gain point in a GPIF file.
type VolumeAutomation struct {
	// Track is the zero-based track index.
	Track int
	// Bar is the zero-based bar index.
	Bar int
	// Position is the point position as a fraction of the bar length.
	Position float64
	// Value is the normalized channel-strip volume from silent (0) to full (1).
	Value float64
	// Linear means the segment from the preceding point to this point is linear.
	Linear bool
}

// TempoAutomation describes one tempo change in a GPIF file.
type TempoAutomation struct {
	// Bar is the zero-based bar index.
	Bar int
	// Position is the change position as a fraction of the bar length.
	Position float64
	// Tempo is the number of beats per minute after the change.
	Tempo float64
}

func (s *Song) readBinary(c *cursor) error {
	var err error
	s.Version, err = c.readVersionString()
	if err != nil {
		return fmt.Errorf("reading version: %w", err)
	}
	major := s.Version.Number[0]
	if major < 3 || major > 5 {
		return fmt.Errorf("unsupported binary GP version: %s", s.Version.Data)
	}

	if clipboardErr := s.readClipboard(c); clipboardErr != nil {
		return fmt.Errorf("reading clipboard: %w", clipboardErr)
	}
	if infoErr := s.readInfo(c); infoErr != nil {
		return fmt.Errorf("reading info: %w", infoErr)
	}

	// GP3/4: triplet feel stored as bool
	if major < 5 {
		tf, tripletErr := c.readBool()
		if tripletErr != nil {
			return fmt.Errorf("reading triplet feel: %w", tripletErr)
		}
		if tf {
			s.TripletFeel = TripletFeelEighth
		}
	}

	// GP4+: lyrics
	if major >= 4 {
		s.Lyrics, err = s.readLyrics(c)
		if err != nil {
			return fmt.Errorf("reading lyrics: %w", err)
		}
	}

	// GP5.1+: RSE master effect
	if major >= 5 {
		s.MasterEffect, err = s.readRseMasterEffect(c)
		if err != nil {
			return fmt.Errorf("reading RSE master effect: %w", err)
		}
	}

	// GP5+: page setup
	if major >= 5 {
		if pageSetupErr := s.readPageSetup(c); pageSetupErr != nil {
			return fmt.Errorf("reading page setup: %w", pageSetupErr)
		}
		s.TempoName, err = c.readIntSizeString()
		if err != nil {
			return fmt.Errorf("reading tempo name: %w", err)
		}
	}

	// Tempo
	tempo, err := c.readInt()
	if err != nil {
		return fmt.Errorf("reading tempo: %w", err)
	}
	s.Tempo = int16(tempo)

	// GP5.1+: hide tempo
	if versionGreaterThan(s.Version.Number, [3]byte{5, 0, 0}) {
		s.HideTempo, err = c.readBool()
		if err != nil {
			return fmt.Errorf("reading hide tempo: %w", err)
		}
	}

	// Key signature (stored as int32 for all versions)
	keyInt, err := c.readInt()
	if err != nil {
		return fmt.Errorf("reading key: %w", err)
	}
	s.Key.Key = int8(keyInt)

	// GP4+: octave byte
	if major >= 4 {
		if _, octaveErr := c.readByte(); octaveErr != nil {
			return fmt.Errorf("reading octave: %w", octaveErr)
		}
	}

	// MIDI channels
	if channelErr := s.readMidiChannels(c); channelErr != nil {
		return fmt.Errorf("reading MIDI channels: %w", channelErr)
	}

	// GP5: directions and reverb
	var signs, fromSigns map[DirectionSign]int16
	if major >= 5 {
		signs, fromSigns, err = s.readDirections(c)
		if err != nil {
			return fmt.Errorf("reading directions: %w", err)
		}
		reverb, reverbErr := c.readInt()
		if reverbErr != nil {
			return fmt.Errorf("reading reverb: %w", reverbErr)
		}
		s.MasterEffect.Reverb = float32(reverb)
	}

	minHeaderBytes := 1
	if major >= 5 {
		minHeaderBytes = 4
	}
	measureCount, err := c.readCount(minHeaderBytes, "measure count")
	if err != nil {
		return err
	}
	trackCount, err := c.readCount(98, "track count")
	if err != nil {
		return err
	}

	// Measure headers
	if major >= 5 {
		if err := s.readMeasureHeadersV5(c, measureCount, signs, fromSigns); err != nil {
			return fmt.Errorf("reading measure headers: %w", err)
		}
	} else {
		if err := s.readMeasureHeaders(c, measureCount); err != nil {
			return fmt.Errorf("reading measure headers: %w", err)
		}
	}

	// Tracks
	if major >= 5 {
		if err := s.readTracksV5(c, trackCount); err != nil {
			return fmt.Errorf("reading tracks: %w", err)
		}
	} else {
		if err := s.readTracks(c, trackCount); err != nil {
			return fmt.Errorf("reading tracks: %w", err)
		}
	}
	s.consolidateTrackChannels()

	// Measures (beats/notes)
	if err := s.readMeasures(c); err != nil {
		return fmt.Errorf("reading measures: %w", err)
	}
	return nil
}

func (s *Song) readInfo(c *cursor) error {
	var err error
	s.Name, err = c.readIntByteSizeString()
	if err != nil {
		return err
	}
	s.Subtitle, err = c.readIntByteSizeString()
	if err != nil {
		return err
	}
	s.Artist, err = c.readIntByteSizeString()
	if err != nil {
		return err
	}
	s.Album, err = c.readIntByteSizeString()
	if err != nil {
		return err
	}
	s.Words, err = c.readIntByteSizeString()
	if err != nil {
		return err
	}
	if s.Version.Number[0] >= 5 {
		s.Author, err = c.readIntByteSizeString()
		if err != nil {
			return err
		}
	} else {
		s.Author = s.Words
	}
	s.Copyright, err = c.readIntByteSizeString()
	if err != nil {
		return err
	}
	s.Writer, err = c.readIntByteSizeString()
	if err != nil {
		return err
	}
	s.Instructions, err = c.readIntByteSizeString()
	if err != nil {
		return err
	}
	noticeCount, err := c.readCount(5, "notice line count")
	if err != nil {
		return err
	}
	for i := 0; i < noticeCount; i++ {
		n, err := c.readIntByteSizeString()
		if err != nil {
			return err
		}
		s.Notice = append(s.Notice, n)
	}
	return nil
}

// Version comparison helpers.
func versionGreaterThan(a, b [3]byte) bool {
	if a[0] != b[0] {
		return a[0] > b[0]
	}
	if a[1] != b[1] {
		return a[1] > b[1]
	}
	return a[2] > b[2]
}

func versionGTE(a, b [3]byte) bool {
	return a == b || versionGreaterThan(a, b)
}

func versionLessThan(a, b [3]byte) bool {
	return !versionGTE(a, b)
}
