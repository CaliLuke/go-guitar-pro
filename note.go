// SPDX-License-Identifier: MIT

package goguitarpro

// Note represents a note.
type Note struct {
	Effect          NoteEffect
	DurationPercent float32
	Value           int16
	Velocity        int16
	String          int8
	SwapAccidentals bool
	Kind            NoteType
}

func defaultNote() Note {
	return Note{
		Velocity:        DefaultVelocity,
		Effect:          defaultNoteEffect(),
		DurationPercent: 1.0,
		Kind:            NoteTypeRest,
	}
}

func (s *Song) readNotes(c *cursor, trackIndex int, beat *Beat, duration *Duration, noteEffect NoteEffect) error {
	flags, err := c.readByte()
	if err != nil {
		return err
	}
	for i := 0; i < len(s.Tracks[trackIndex].Strings); i++ {
		str := s.Tracks[trackIndex].Strings[i]
		if (flags & (1 << (7 - str.Number))) > 0 {
			note := defaultNote()
			note.Effect = noteEffect // Clone the shared note effect
			if versionLessThan(s.Version.Number, [3]byte{5, 0, 0}) {
				if err := s.readNote(c, &note, str, trackIndex); err != nil {
					return err
				}
			} else {
				if err := s.readNoteV5(c, &note, str, trackIndex); err != nil {
					return err
				}
			}
			beat.Notes = append(beat.Notes, note)
		}
		beat.Duration = *duration
	}
	return nil
}

func (s *Song) readNote(c *cursor, note *Note, guitarString GuitarString, trackIndex int) error {
	flags, err := c.readByte()
	if err != nil {
		return err
	}
	note.String = guitarString.Number
	note.Effect.GhostNote = (flags & 0x04) == 0x04

	if (flags & 0x20) == 0x20 {
		kind, kindErr := c.readByte()
		if kindErr != nil {
			return kindErr
		}
		note.Kind = NoteType(kind)
	}
	if (flags & 0x01) == 0x01 {
		// time-independent duration
		if _, err := c.readSignedByte(); err != nil {
			return err
		}
		if _, err := c.readSignedByte(); err != nil {
			return err
		}
	}
	if (flags & 0x10) == 0x10 {
		v, velocityErr := c.readSignedByte()
		if velocityErr != nil {
			return velocityErr
		}
		note.Velocity = unpackVelocity(int16(v))
	}
	if (flags & 0x20) == 0x20 {
		fret, fretErr := c.readSignedByte()
		if fretErr != nil {
			return fretErr
		}
		var value int16
		if note.Kind == NoteTypeTie {
			value = s.getTiedNoteValue(guitarString.Number, trackIndex)
		} else {
			value = int16(fret)
		}
		note.Value = clampI16(value, 0, 99)
	}
	if (flags & 0x80) == 0x80 {
		lf, err := c.readSignedByte()
		if err != nil {
			return err
		}
		note.Effect.LeftHandFinger = Fingering(lf)
		rf, err := c.readSignedByte()
		if err != nil {
			return err
		}
		note.Effect.RightHandFinger = Fingering(rf)
	}
	if (flags & 0x08) == 0x08 {
		if s.Version.Number == [3]byte{3, 0, 0} {
			if err := s.readNoteEffectsV3(c, note); err != nil {
				return err
			}
		} else if s.Version.Number[0] == 4 {
			if err := s.readNoteEffectsV4(c, note); err != nil {
				return err
			}
		}
		if note.Effect.Harmonic != nil && note.Effect.Harmonic.Kind == HarmonicTypeTapped {
			fret := int8(note.Value) + 12
			note.Effect.Harmonic.Fret = &fret
		}
	}
	return nil
}

func (s *Song) readNoteV5(c *cursor, note *Note, guitarString GuitarString, trackIndex int) error {
	flags, err := c.readByte()
	if err != nil {
		return err
	}
	note.String = guitarString.Number
	note.Effect.HeavyAccentuatedNote = (flags & 0x02) == 0x02
	note.Effect.GhostNote = (flags & 0x04) == 0x04
	note.Effect.AccentuatedNote = (flags & 0x40) == 0x40

	if (flags & 0x20) == 0x20 {
		kind, kindErr := c.readByte()
		if kindErr != nil {
			return kindErr
		}
		note.Kind = NoteType(kind)
	}
	if (flags & 0x10) == 0x10 {
		v, velocityErr := c.readSignedByte()
		if velocityErr != nil {
			return velocityErr
		}
		note.Velocity = unpackVelocity(int16(v))
	}
	if (flags & 0x20) == 0x20 {
		fret, fretErr := c.readSignedByte()
		if fretErr != nil {
			return fretErr
		}
		var value int16
		if note.Kind == NoteTypeTie {
			value = s.getTiedNoteValue(guitarString.Number, trackIndex)
		} else {
			value = int16(fret)
		}
		note.Value = clampI16(value, 0, 99)
	}
	if (flags & 0x80) == 0x80 {
		lf, leftFingerErr := c.readSignedByte()
		if leftFingerErr != nil {
			return leftFingerErr
		}
		note.Effect.LeftHandFinger = Fingering(lf)
		rf, rightFingerErr := c.readSignedByte()
		if rightFingerErr != nil {
			return rightFingerErr
		}
		note.Effect.RightHandFinger = Fingering(rf)
	}
	if (flags & 0x01) == 0x01 {
		dp, durationPercentErr := c.readDouble()
		if durationPercentErr != nil {
			return durationPercentErr
		}
		note.DurationPercent = float32(dp)
	}
	// Second set of flags
	flags2, err := c.readByte()
	if err != nil {
		return err
	}
	note.SwapAccidentals = (flags2 & 0x02) == 0x02

	if (flags & 0x08) == 0x08 {
		if effectErr := s.readNoteEffectsV4(c, note); effectErr != nil {
			return effectErr
		}
	}
	// GP5 stores this as a fret even for drums; expose the semantic articulation.
	if s.Tracks[trackIndex].PercussionTrack && note.Effect.Grace != nil {
		note.Effect.Grace.Fret = int8(note.Value)
	}
	return nil
}

func (s *Song) readNoteEffectsV3(c *cursor, note *Note) error {
	flags, err := c.readByte()
	if err != nil {
		return err
	}
	note.Effect.Hammer = (flags & 0x02) == 0x02
	note.Effect.LetRing = (flags & 0x08) == 0x08
	if (flags & 0x01) == 0x01 {
		bend, err := s.readBendEffect(c)
		if err != nil {
			return err
		}
		note.Effect.Bend = bend
	}
	if (flags & 0x10) == 0x10 {
		grace, err := s.readGraceEffect(c)
		if err != nil {
			return err
		}
		note.Effect.Grace = &grace
	}
	if (flags & 0x04) == 0x04 {
		note.Effect.Slides = append(note.Effect.Slides, SlideShiftSlideTo)
	}
	return nil
}

func (s *Song) readNoteEffectsV4(c *cursor, note *Note) error {
	flags1, err := c.readSignedByte()
	if err != nil {
		return err
	}
	flags2, err := c.readSignedByte()
	if err != nil {
		return err
	}
	note.Effect.Hammer = (flags1 & 0x02) == 0x02
	note.Effect.LetRing = (flags1 & 0x08) == 0x08
	note.Effect.Staccato = (flags2 & 0x01) == 0x01
	note.Effect.PalmMute = (flags2 & 0x02) == 0x02
	note.Effect.Vibrato = (flags2&0x40) == 0x40 || note.Effect.Vibrato

	if (flags1 & 0x01) == 0x01 {
		bend, err := s.readBendEffect(c)
		if err != nil {
			return err
		}
		note.Effect.Bend = bend
	}
	if (flags1 & 0x10) == 0x10 {
		if versionGTE(s.Version.Number, [3]byte{5, 0, 0}) {
			grace, err := s.readGraceEffectV5(c)
			if err != nil {
				return err
			}
			note.Effect.Grace = &grace
		} else {
			grace, err := s.readGraceEffect(c)
			if err != nil {
				return err
			}
			note.Effect.Grace = &grace
		}
	}
	if (flags2 & 0x04) == 0x04 {
		tp, err := s.readTremoloPicking(c)
		if err != nil {
			return err
		}
		note.Effect.TremoloPicking = &tp
	}
	if (flags2 & 0x08) == 0x08 {
		if versionGTE(s.Version.Number, [3]byte{5, 0, 0}) {
			slides, err := s.readSlidesV5(c)
			if err != nil {
				return err
			}
			note.Effect.Slides = append(note.Effect.Slides, slides...)
		} else {
			st, err := c.readSignedByte()
			if err != nil {
				return err
			}
			note.Effect.Slides = append(note.Effect.Slides, SlideType(st))
		}
	}
	if (flags2 & 0x10) == 0x10 {
		if versionGTE(s.Version.Number, [3]byte{5, 0, 0}) {
			h, err := s.readHarmonicV5(c)
			if err != nil {
				return err
			}
			note.Effect.Harmonic = &h
		} else {
			h, err := s.readHarmonic(c, note)
			if err != nil {
				return err
			}
			note.Effect.Harmonic = &h
		}
	}
	if (flags2 & 0x20) == 0x20 {
		trill, err := s.readTrill(c)
		if err != nil {
			return err
		}
		note.Effect.Trill = &trill
	}
	return nil
}

func (s *Song) readHarmonic(c *cursor, note *Note) (HarmonicEffect, error) {
	he := HarmonicEffect{}
	kind, err := c.readSignedByte()
	if err != nil {
		return he, err
	}
	switch kind {
	case 1:
		he.Kind = HarmonicTypeNatural
	case 3:
		he.Kind = HarmonicTypeTapped
	case 4:
		he.Kind = HarmonicTypePinch
	case 5:
		he.Kind = HarmonicTypeSemi
	case 15:
		he.Kind = HarmonicTypeArtificial
		pitch := pitchClassFrom(int8((note.Value+7)%12), nil, nil)
		he.Pitch = &pitch
		oct := OctaveOttava
		he.Octave = &oct
	case 17:
		he.Kind = HarmonicTypeArtificial
		if s.currentTrack != nil {
			rv := s.realNoteValue(note, *s.currentTrack)
			pitch := pitchClassFrom(rv, nil, nil)
			he.Pitch = &pitch
		}
		oct := OctaveQuindicesima
		he.Octave = &oct
	case 22:
		he.Kind = HarmonicTypeArtificial
		if s.currentTrack != nil {
			rv := s.realNoteValue(note, *s.currentTrack)
			pitch := pitchClassFrom(rv, nil, nil)
			he.Pitch = &pitch
		}
		oct := OctaveOttava
		he.Octave = &oct
	}
	return he, nil
}

func (s *Song) realNoteValue(note *Note, trackIndex int) int8 {
	if note.String > 0 && trackIndex < len(s.Tracks) {
		idx := int(note.String) - 1
		if idx < len(s.Tracks[trackIndex].Strings) {
			return int8(note.Value) + s.Tracks[trackIndex].Strings[idx].Value
		}
	}
	return int8(note.Value)
}

func (s *Song) getTiedNoteValue(stringIndex int8, trackIndex int) int16 {
	for m := len(s.Tracks[trackIndex].Measures) - 1; m >= 0; m-- {
		for v := len(s.Tracks[trackIndex].Measures[m].Voices) - 1; v >= 0; v-- {
			for b := 0; b < len(s.Tracks[trackIndex].Measures[m].Voices[v].Beats); b++ {
				beat := &s.Tracks[trackIndex].Measures[m].Voices[v].Beats[b]
				if beat.Status != BeatStatusEmpty {
					for n := 0; n < len(beat.Notes); n++ {
						if beat.Notes[n].String == stringIndex {
							return beat.Notes[n].Value
						}
					}
				}
			}
		}
	}
	return -1
}

func clampI16(v, lo, hi int16) int16 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
