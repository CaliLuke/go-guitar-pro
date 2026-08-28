// SPDX-License-Identifier: MIT

package goguitarpro

import "fmt"

// BeatDisplay contains parameters of beat display.
type BeatDisplay struct {
	BreakBeam            bool
	ForceBeam            bool
	BeamDirection        VoiceDirection
	TupletBracket        TupletBracket
	BreakSecondary       uint8
	BreakSecondaryTuplet bool
	ForceBracket         bool
}

// BeatStroke represents a stroke effect for beats.
type BeatStroke struct {
	Direction BeatStrokeDirection
	Value     uint16
}

// Voice contains multiple beats.
type Voice struct {
	Beats        []Beat
	MeasureIndex int16
	Direction    VoiceDirection
}

// BeatEffects contains all beat effects.
type BeatEffects struct {
	Chord          *Chord
	TremoloBar     *BendEffect
	MixTableChange *MixTableChange
	Stroke         BeatStroke
	HasRasgueado   bool
	PickStroke     BeatStrokeDirection
	FadeIn         bool
	SlapEffect     SlapEffect
	Vibrato        bool
}

// Beat contains multiple notes.
type Beat struct {
	Start    *int64
	Effect   BeatEffects
	Text     string
	Notes    []Note
	Duration Duration
	// Dynamics is the beat-wide MIDI velocity authored by Guitar Pro. GP3-5
	// stores the value on notes, but the last explicit value applies to the
	// whole beat during playback.
	Dynamics int16
	Display  BeatDisplay
	Octave   Octave
	Status   BeatStatus
}

func defaultBeat() Beat {
	return Beat{
		Duration: defaultDuration(),
		Dynamics: DefaultVelocity,
		Status:   BeatStatusNormal,
	}
}

var debugBeatRange [2]int // set in tests

func (s *Song) readBeat(c *cursor, voice *Voice, start int64, trackIndex int) (int64, error) {
	beatStartPos := c.pos
	dbg := beatStartPos >= debugBeatRange[0] && beatStartPos <= debugBeatRange[1]
	flags, err := c.readByte()
	if err != nil {
		return 0, fmt.Errorf("readBeat flags at %d: %w", beatStartPos, err)
	}
	if dbg {
		fmt.Printf("  readBeat pos=%d flags=0x%02x\n", beatStartPos, flags)
	}

	beat := defaultBeat()
	startCopy := start
	beat.Start = &startCopy

	if (flags & 0x40) == 0x40 {
		status, statusErr := c.readByte()
		if statusErr != nil {
			return 0, statusErr
		}
		beat.Status = BeatStatus(status)
	}

	duration, err := readDuration(c, flags)
	if err != nil {
		return 0, fmt.Errorf("readBeat duration (flags=0x%02x pos=%d): %w", flags, c.pos, err)
	}
	if dbg {
		fmt.Printf("    after duration pos=%d\n", c.pos)
	}

	noteEffect := defaultNoteEffect()

	if (flags & 0x02) == 0x02 {
		if dbg {
			fmt.Printf("    reading chord at pos=%d\n", c.pos)
		}
		chord, err := s.readChord(c, uint8(len(s.Tracks[trackIndex].Strings)))
		if err != nil {
			return 0, fmt.Errorf("readBeat chord at %d: %w", c.pos, err)
		}
		beat.Effect.Chord = &chord
		if dbg {
			fmt.Printf("    after chord pos=%d\n", c.pos)
		}
	}
	if (flags & 0x04) == 0x04 {
		if dbg {
			fmt.Printf("    reading text at pos=%d\n", c.pos)
		}
		text, err := c.readIntByteSizeString()
		if err != nil {
			return 0, fmt.Errorf("readBeat text at %d (flags=0x%02x): %w", c.pos, flags, err)
		}
		beat.Text = text
		if dbg {
			fmt.Printf("    after text pos=%d text=%q\n", c.pos, text)
		}
	}
	if (flags & 0x08) == 0x08 {
		savedChord := beat.Effect.Chord
		if s.Version.Number[0] == 3 {
			be, ne, err := s.readBeatEffectsV3(c, &noteEffect)
			if err != nil {
				return 0, fmt.Errorf("readBeat effectsV3 at %d: %w", c.pos, err)
			}
			beat.Effect = be
			noteEffect = ne
		} else {
			be, err := s.readBeatEffectsV4(c)
			if err != nil {
				return 0, fmt.Errorf("readBeat effectsV4 at %d: %w", c.pos, err)
			}
			beat.Effect = be
		}
		beat.Effect.Chord = savedChord
	}
	if (flags & 0x10) == 0x10 {
		mtc, err := s.readMixTableChange(c)
		if err != nil {
			return 0, fmt.Errorf("readBeat mixTable at %d: %w", c.pos, err)
		}
		beat.Effect.MixTableChange = &mtc
	}

	if dbg {
		fmt.Printf("    reading notes at pos=%d\n", c.pos)
	}
	if err := s.readNotes(c, trackIndex, &beat, &duration, noteEffect); err != nil {
		return 0, fmt.Errorf("readBeat notes at %d (flags=0x%02x): %w", c.pos, flags, err)
	}
	if dbg {
		fmt.Printf("    after notes pos=%d noteCount=%d\n", c.pos, len(beat.Notes))
	}

	voice.Beats = append(voice.Beats, beat)

	if beat.Status == BeatStatusEmpty {
		return 0, nil
	}
	return int64(duration.time()), nil
}

func (s *Song) readBeatV5(c *cursor, voice *Voice, start *int64, trackIndex int) (int64, error) {
	v5Start := c.pos
	dur, err := s.readBeat(c, voice, *start, trackIndex)
	if err != nil {
		return 0, err
	}

	b := len(voice.Beats) - 1
	dbg := v5Start >= debugBeatRange[0] && v5Start <= debugBeatRange[1]
	if dbg {
		fmt.Printf("  readBeatV5 reading flags2 at pos=%d\n", c.pos)
	}
	flags2, err := c.readShort()
	if err != nil {
		return 0, err
	}

	if (flags2 & 0x0010) == 0x0010 {
		voice.Beats[b].Octave = OctaveOttava
	}
	if (flags2 & 0x0020) == 0x0020 {
		voice.Beats[b].Octave = OctaveOttavaBassa
	}
	if (flags2 & 0x0040) == 0x0040 {
		voice.Beats[b].Octave = OctaveQuindicesima
	}
	if (flags2 & 0x0100) == 0x0100 {
		voice.Beats[b].Octave = OctaveQuindicesimaBassa
	}

	voice.Beats[b].Display.BreakBeam = (flags2 & 0x0001) == 0x0001
	voice.Beats[b].Display.ForceBeam = (flags2 & 0x0004) == 0x0004
	voice.Beats[b].Display.ForceBracket = (flags2 & 0x2000) == 0x2000
	voice.Beats[b].Display.BreakSecondaryTuplet = (flags2 & 0x1000) == 0x1000
	if (flags2 & 0x0002) == 0x0002 {
		voice.Beats[b].Display.BeamDirection = VoiceDirectionDown
	}
	if (flags2 & 0x0008) == 0x0008 {
		voice.Beats[b].Display.BeamDirection = VoiceDirectionUp
	}
	if (flags2 & 0x0200) == 0x0200 {
		voice.Beats[b].Display.TupletBracket = TupletBracketStart
	}
	if (flags2 & 0x0400) == 0x0400 {
		voice.Beats[b].Display.TupletBracket = TupletBracketEnd
	}
	if (flags2 & 0x0800) == 0x0800 {
		bs, err := c.readByte()
		if err != nil {
			return 0, err
		}
		voice.Beats[b].Display.BreakSecondary = bs
	}

	return dur, nil
}

func (s *Song) readBeatEffectsV3(c *cursor, noteEffect *NoteEffect) (BeatEffects, NoteEffect, error) {
	var be BeatEffects
	flags, err := c.readByte()
	if err != nil {
		return be, *noteEffect, err
	}
	noteEffect.Vibrato = (flags&0x01) == 0x01 || noteEffect.Vibrato
	be.Vibrato = (flags & 0x02) == 0x02
	be.FadeIn = (flags & 0x10) == 0x10
	if (flags & 0x20) == 0x20 {
		slapByte, err := c.readByte()
		if err != nil {
			return be, *noteEffect, err
		}
		be.SlapEffect = SlapEffect(slapByte)
		if be.SlapEffect == SlapEffectNone {
			tb, err := s.readTremoloBar(c)
			if err != nil {
				return be, *noteEffect, err
			}
			be.TremoloBar = &tb
		} else {
			if _, err := c.readInt(); err != nil {
				return be, *noteEffect, err
			}
		}
	}
	if (flags & 0x40) == 0x40 {
		stroke, err := s.readBeatStroke(c)
		if err != nil {
			return be, *noteEffect, err
		}
		be.Stroke = stroke
	}
	if (flags & 0x04) == 0x04 {
		noteEffect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeNatural}
	}
	if (flags & 0x08) == 0x08 {
		noteEffect.Harmonic = &HarmonicEffect{Kind: HarmonicTypeArtificial}
	}
	return be, *noteEffect, nil
}

func (s *Song) readBeatEffectsV4(c *cursor) (BeatEffects, error) {
	var be BeatEffects
	flags1, err := c.readSignedByte()
	if err != nil {
		return be, err
	}
	flags2, err := c.readSignedByte()
	if err != nil {
		return be, err
	}
	be.Vibrato = (flags1 & 0x02) == 0x02
	be.FadeIn = (flags1 & 0x10) == 0x10
	if (flags1 & 0x20) == 0x20 {
		slapByte, err := c.readSignedByte()
		if err != nil {
			return be, err
		}
		be.SlapEffect = SlapEffect(uint8(slapByte))
	}
	if (flags2 & 0x04) == 0x04 {
		bend, err := s.readBendEffect(c)
		if err != nil {
			return be, err
		}
		be.TremoloBar = bend
	}
	if (flags1 & 0x40) == 0x40 {
		stroke, err := s.readBeatStroke(c)
		if err != nil {
			return be, err
		}
		be.Stroke = stroke
	}
	be.HasRasgueado = (flags2 & 0x01) == 0x01
	if (flags2 & 0x02) == 0x02 {
		ps, err := c.readSignedByte()
		if err != nil {
			return be, err
		}
		be.PickStroke = BeatStrokeDirection(ps)
	}
	return be, nil
}

func (s *Song) readBeatStroke(c *cursor) (BeatStroke, error) {
	var bs BeatStroke
	down, err := c.readSignedByte()
	if err != nil {
		return bs, err
	}
	up, err := c.readSignedByte()
	if err != nil {
		return bs, err
	}
	if up > 0 {
		bs.Direction = BeatStrokeDirectionUp
		bs.Value = uint16(strokeValue(up))
	}
	if down > 0 {
		bs.Direction = BeatStrokeDirectionDown
		bs.Value = uint16(strokeValue(down))
	}
	if versionGTE(s.Version.Number, [3]byte{5, 0, 0}) {
		// Swap direction
		switch bs.Direction {
		case BeatStrokeDirectionUp:
			bs.Direction = BeatStrokeDirectionDown
		case BeatStrokeDirectionDown:
			bs.Direction = BeatStrokeDirectionUp
		}
	}
	return bs, nil
}

func strokeValue(value int8) uint8 {
	switch value {
	case 1:
		return DurationHundredTwentyEighth
	case 2:
		return DurationSixtyFourth
	case 3:
		return DurationThirtySecond
	case 4:
		return DurationSixteenth
	case 5:
		return DurationEighth
	case 6:
		return DurationQuarter
	default:
		return DurationSixtyFourth
	}
}

func (s *Song) readTremoloBar(c *cursor) (BendEffect, error) {
	val, err := c.readInt()
	if err != nil {
		return BendEffect{}, err
	}
	be := BendEffect{Kind: BendTypeDip, Value: int16(val)}
	be.Points = []BendPoint{
		{Position: 0, Value: 0},
		{Position: uint8(BendEffectMaxPosition) / 2, Value: int8(-float32(be.Value) / GPBendSemitone)},
		{Position: uint8(BendEffectMaxPosition), Value: 0},
	}
	return be, nil
}
