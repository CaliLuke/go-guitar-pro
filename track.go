// SPDX-License-Identifier: MIT

package goguitarpro

import "fmt"

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

// Staff represents one notation staff owned by a track.
type Staff struct {
	// NotationSettings owns this staff's standard, tab, slash and numbered requests.
	// Nil uses Track.Settings for standard/tab and false for slash/numbered.
	// Changed parsed Track.Settings flags override only their corresponding staff flag.
	NotationSettings *StaffNotationSettings
	// Measures contains this staff's ordered measures.
	Measures []Measure
	// Strings contains this staff's tuning from highest string to lowest.
	Strings []GuitarString
	// TuningName contains the authored label for this staff's tuning.
	// It is independent from Strings, CapoFret, and string order. Changing the
	// label does not recalculate or otherwise change those values.
	TuningName string
	// PercussionTrack reports whether this staff uses percussion articulations.
	PercussionTrack bool
	// StandardNotationLineCount is the number of rendered lines in standard notation.
	StandardNotationLineCount int
	// CapoFret is the non-negative fret at which this staff's capo is placed.
	CapoFret int32
	// TranspositionPitch is the authored sounding offset in semitones.
	// A sounded MIDI pitch is the string-and-fret pitch minus this value.
	// Changing it does not rewrite the authored tuning, string, or fret.
	TranspositionPitch int32
	// DisplayTranspositionPitch is the authored notation offset in semitones.
	// It changes the effective displayed key without changing sounding pitch,
	// tuning, string, or fret.
	DisplayTranspositionPitch int32
}

// Track represents a track.
type Track struct {
	// SystemLayout preserves authored system counts. Nil leaves this scope unspecified.
	SystemLayout *SystemLayout
	// MultiRest requests combined rests when this track is viewed alone. Nil
	// uses false, independently of ScoreStyle.MultiRest. Measures remain intact.
	MultiRest *bool
	Name      string
	// ShortName is the independent authored abbreviation. Nil means absent.
	// A non-nil empty string preserves an explicitly empty short name.
	ShortName *string
	// Staves preserves every staff in GPIF track order. Binary GP3–5 tracks have one staff.
	Staves []Staff
	// Measures is the first staff's compatibility view. Use Staves for lossless multi-staff access.
	Measures []Measure
	// Strings is the first staff's compatibility view. Use Staves for staff-specific tuning.
	Strings []GuitarString
	Lyrics  []TrackLyricLine
	Sounds  []TrackSound
	// PercussionArticulations contains this track's ordered GPIF percussion definitions.
	PercussionArticulations []PercussionArticulation
	SoundAutomations        []SoundAutomation
	Rse                     TrackRse
	// ChannelIndex is the index of the track channel in Song.Channels.
	// A value of -1 means that the file does not bind the track to a channel.
	ChannelIndex int
	// CapoFret is the legacy first-staff capo view. A changed parsed value applies to every staff during export.
	CapoFret int32
	// Number is the one-based ordinal of the track in the score.
	Number int32
	// Color is the track color encoded as 0xRRGGBB.
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
	notationCompatibility     [2]bool
	notationCompatibilitySet  bool
	capoFretCompatibility     int32
	capoFretCompatibilitySet  bool
}

// PercussionArticulation describes one track-local GPIF percussion definition.
type PercussionArticulation struct {
	// ElementName identifies the parent instrument-set element used by notation patches.
	ElementName string
	// ElementType retains the authored instrument-set element type.
	ElementType string
	// ElementSoundbankName identifies the authored RSE soundbank for the parent element.
	ElementSoundbankName string
	// Name identifies the articulation within its parent element.
	Name string
	// StaffLine is the Guitar Pro staff step, including lines and spaces.
	StaffLine int
	// NoteheadDefault is the notehead used for durations other than half and whole notes.
	NoteheadDefault string
	// NoteheadHalf is the notehead used for half notes.
	NoteheadHalf string
	// NoteheadWhole is the notehead used for whole notes.
	NoteheadWhole string
	// TechniquePlacement identifies where the technique symbol is rendered.
	TechniquePlacement string
	// TechniqueSymbol identifies the optional technique glyph.
	TechniqueSymbol string
	// InputMIDINumbers contains every authored input MIDI value for this articulation.
	InputMIDINumbers []int
	// OutputRSESound identifies the authored RSE playback sound.
	OutputRSESound string
	// OutputMIDINumber is the MIDI value used for playback.
	OutputMIDINumber int
}

func (t *Track) populateSingleStaff() {
	t.Staves = []Staff{{
		Measures:                  t.Measures,
		Strings:                   t.Strings,
		PercussionTrack:           t.PercussionTrack,
		StandardNotationLineCount: 5,
		CapoFret:                  t.CapoFret,
	}}
}

func (t *Track) reconcileFirstStaffCompatibility() {
	if len(t.Staves) == 0 {
		t.populateSingleStaff()
		return
	}
	if t.Measures != nil {
		t.Staves[0].Measures = t.Measures
	} else {
		t.Measures = t.Staves[0].Measures
	}
	if t.Strings != nil {
		t.Staves[0].Strings = t.Strings
	} else {
		t.Strings = t.Staves[0].Strings
	}
	t.Staves[0].PercussionTrack = t.PercussionTrack
	t.Measures = t.Staves[0].Measures
	t.Strings = t.Staves[0].Strings
}

func (t *Track) markCapoFretCompatibility() {
	t.capoFretCompatibility = t.CapoFret
	t.capoFretCompatibilitySet = true
}

func (t *Track) resolvedStaffCapos() []int32 {
	if len(t.Staves) == 0 {
		return []int32{t.CapoFret}
	}
	capos := make([]int32, len(t.Staves))
	staffHasNonZero := false
	for index := range t.Staves {
		capos[index] = t.Staves[index].CapoFret
		staffHasNonZero = staffHasNonZero || capos[index] != 0
	}
	legacyEdit := t.capoFretCompatibilitySet && t.CapoFret != t.capoFretCompatibility
	legacyOnlyProgrammaticValue := !t.capoFretCompatibilitySet && !staffHasNonZero && t.CapoFret != 0
	if legacyEdit || legacyOnlyProgrammaticValue {
		for index := range capos {
			capos[index] = t.CapoFret
		}
	}
	return capos
}

// TrackSound describes one selectable GPIF playback sound.
type TrackSound struct {
	Name    string
	Label   string
	Path    string
	Role    string
	Program int32
	// Bank is the combined MIDI bank-select value from 0 through 16383.
	Bank int32
}

// SoundAutomation selects a track sound at a score position. GP3-5 mix-table
// program changes are promoted into this collection on import. Direct edits and
// clearing override those retained raw records. GP8 reports the pinned consumer's
// first-beat relocation of positive-position sound events.
type SoundAutomation struct {
	// Bar is a zero-based measure-header index.
	Bar int
	// Position is a ratio from 0 at the bar start through 1 at the bar end.
	// Bar 0 also supports opening preroll from -0.125 up to 0.
	// Direct edits are authoritative. GPIF uses quarter-note offsets; import and
	// export convert those units while preserving the public ratio and slice order.
	Position float64
	// Sound is a zero-based index in Track.Sounds.
	// Remap this index when reordering definitions to preserve its selection.
	Sound int
	// Linear means the sound change is applied linearly.
	Linear bool
	// Text is the authored annotation for this event.
	Text string
	// Hidden reports whether the authored event is suppressed.
	// The zero value keeps the historically visible programmatic default.
	Hidden bool
}

// GuitarString represents a guitar string with tuning.
type GuitarString struct {
	// Number is the one-based string number in tuning order, highest to lowest.
	Number int8
	// Value is the absolute MIDI note number of the open string, from 0 through 127.
	Value int8
}

func defaultTrack() Track {
	return Track{
		Number:       1,
		Visible:      true,
		Name:         "Track 1",
		FretCount:    24,
		Color:        0xff0000,
		Port:         1,
		ChannelIndex: -1,
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
	track.Number = int32(number + 1)

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

	if channelErr := s.readChannel(c, &track); channelErr != nil {
		return channelErr
	}

	fretCount, err := readTrackFretCount(c, number+1)
	if err != nil {
		return err
	}
	track.FretCount = fretCount

	offset, err := c.readInt()
	if err != nil {
		return err
	}
	track.CapoFret = offset

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
	track.Number = int32(number + 1)

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

	if channelErr := s.readChannel(c, &track); channelErr != nil {
		return channelErr
	}

	fretCount, err := readTrackFretCount(c, number+1)
	if err != nil {
		return err
	}
	track.FretCount = fretCount

	offset, err := c.readInt()
	if err != nil {
		return err
	}
	track.CapoFret = offset

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

	bankByte, err := c.readByte()
	if err != nil {
		return err
	}
	if track.ChannelIndex >= 0 && track.ChannelIndex < len(s.Channels) {
		s.Channels[track.ChannelIndex].Bank = int32(bankByte)
	}

	if err := s.readTrackRse(c, &track); err != nil {
		return err
	}

	s.Tracks = append(s.Tracks, track)
	return nil
}

func readTrackFretCount(c *cursor, number int) (uint8, error) {
	value, err := c.readInt()
	if err != nil {
		return 0, fmt.Errorf("reading track %d fret count: %w", number, err)
	}
	if value < 0 || value > 255 {
		return 0, fmt.Errorf("reading track %d fret count: value %d is outside public uint8 range 0..255", number, value)
	}
	return uint8(value), nil
}
