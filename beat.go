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
	// Kind distinguishes GPIF Brush properties from Arpeggio elements. For
	// compatibility, KindNone with a non-none Direction exports as an arpeggio.
	Kind      BeatStrokeKind
	Direction BeatStrokeDirection
	// Duration is the note-value denominator used to spread the stroke. Zero
	// retains the historical target-default behavior for programmatic strokes.
	Duration NoteValue
	// ExactDuration is the authored stroke duration in 960-PPQ score ticks.
	// A nil value preserves the absence of GPIF's timing XProperty. On import,
	// it remains authoritative until Duration is edited; clearing it falls back
	// to Duration. If both views are edited incompatibly, Duration wins.
	ExactDuration *ScoreTime

	importedDuration NoteValue
	hasImported      bool
	importedExact    bool
}

// BeatLegato preserves the authored endpoints of a beat-level legato phrase.
// A destination can be present without an origin when the phrase begins before
// an imported excerpt, and an origin can be present without a destination when
// it continues beyond the excerpt.
type BeatLegato struct {
	Origin      bool
	Destination bool
}

// Voice contains multiple beats.
type Voice struct {
	Beats []Beat
	// MeasureIndex is the zero-based index of the owning measure in its staff.
	MeasureIndex int16
	Direction    VoiceDirection
}

// BeatEffects contains all beat effects.
type BeatEffects struct {
	// WahPedal is a beat-local event. GP5 imports derive it from MixTableChange.Wah:
	// -1 means absent, 0..99 Open, and 100..127 Closed. Imported edits to either
	// view control export; incompatible edits to both favor WahPedal. A nonzero
	// programmatic state wins, otherwise the legacy value is the fallback.
	// GPIF imports do not fabricate a legacy mix-table record.
	WahPedal   WahPedal
	Chord      *Chord
	TremoloBar *BendEffect
	// TremoloPicking is the beat-wide authored authority. The legacy
	// NoteEffect.TremoloPicking field is used only when this pointer is nil.
	TremoloPicking *TremoloPickingEffect
	MixTableChange *MixTableChange
	Stroke         BeatStroke
	// RasgueadoPattern is the named beat pattern. On imported beats, an edit to
	// either this field or HasRasgueado controls export; incompatible edits to
	// both favor the named pattern. A true unspecified boolean defaults to Ii
	// with an explicit export normalization.
	RasgueadoPattern RasgueadoPattern
	HasRasgueado     bool
	// PickStroke is the authored pick direction, independent from Stroke.
	// Direct edits control GP8 output; BeatStrokeDirectionNone removes the mark.
	PickStroke BeatStrokeDirection
	// Fade is the typed authored fade view. For a programmatic score, a
	// nonzero value is authoritative and FadeIn=true is the legacy fallback.
	// On an imported beat, editing only one view makes that view authoritative;
	// when both are edited incompatibly, Fade wins and GP8 export reports the
	// conflict. Export and validation do not mutate either view.
	Fade   BeatFade
	FadeIn bool
	// Golpe is the authored beat-level golpe variant. It is independent from
	// notes and other beat techniques, and direct edits control GP8 output.
	Golpe   GolpeType
	Hairpin Hairpin
	// Tap, Slap, and Pop are independent beat techniques. On imported beats,
	// edits to these states take precedence over incompatible SlapEffect edits.
	// An edit only to SlapEffect replaces all three states. Programmatic nonzero
	// states take precedence; otherwise SlapEffect is the fallback. GPIF imports
	// derive Tap from note Tapped properties without changing either note flag.
	Tap, Slap, Pop bool
	// SlapEffect is the legacy single-technique view. Imported combinations use
	// Pop, then Slap, then Tap priority, independent of source property order.
	SlapEffect SlapEffect
	// VibratoStrength is the typed beat-wide vibrato view. For a programmatic
	// score, a nonzero strength is authoritative and Vibrato=true is the legacy
	// Slight fallback. On an imported beat, editing only one view makes that view
	// authoritative; when both are edited incompatibly, the typed view wins and
	// GP8 export reports the conflict. Export and validation do not mutate either
	// view.
	VibratoStrength BeatVibrato
	Vibrato         bool

	importedWah             WahPedal
	importedWahValue        int8
	importedWahPresent      bool
	hasImportedWah          bool
	importedTechniques      beatTechniques
	importedSlapEffect      SlapEffect
	hasImportedTechniques   bool
	importedRasgueado       RasgueadoPattern
	importedHasRasgueado    bool
	hasImportedRasgueado    bool
	importedVibratoStrength BeatVibrato
	importedVibrato         bool
	hasImportedVibrato      bool
	importedFade            BeatFade
	importedFadeIn          bool
	hasImportedFade         bool
}

// Beat contains multiple notes.
type Beat struct {
	// Start is the absolute display-time start in ticks. Voices begin independently
	// at their measure start. Playback transformations are not applied.
	Start *int64
	// ExactStart preserves fractional score ticks before Start is quantized.
	ExactStart *ScoreTime
	Effect     BeatEffects
	// Legato is nil when the source has no authored beat-level legato marker.
	// Each parsed beat owns its own record, including when GPIF reuses a beat
	// definition in more than one score occurrence.
	Legato *BeatLegato
	// BarreFret is the authored non-negative fret for a beat-level barre mark.
	// Nil means that no barre was authored. A present fret must be paired with
	// BarreShapeFull or BarreShapeHalf.
	BarreFret *Fret
	// BarreShape is the authored full- or half-barre shape. BarreShapeNone must
	// be paired with a nil BarreFret.
	BarreShape BarreShape
	// BeamingMode controls the beam connection from this beat to the next beat.
	// It is the export authority. Binary GP5 flags are moved from the following
	// beat's legacy Display fields into this field during import.
	BeamingMode BeatBeamingMode
	// InvertBeamDirection reverses the automatically selected stem direction. It
	// is independent from an explicit PreferredBeamDirection.
	InvertBeamDirection bool
	// PreferredBeamDirection is the authored up/down stem override. None means
	// no explicit direction and is independent from InvertBeamDirection.
	PreferredBeamDirection VoiceDirection
	// Lyrics preserves the ordered lines authored directly on this beat. Nil
	// means that the GPIF Lyrics element was absent. A non-nil empty slice means
	// that an empty Lyrics element was authored. The slice is independent from
	// Beat.Text, Track.Lyrics, and Song.Lyrics, and direct edits are authoritative.
	Lyrics []string
	// DeadSlapped preserves an authored beat-level dead-slap mark. It is
	// independent from beat status, note kind, and tap/slap/pop effects. Direct
	// edits are authoritative for GP8 export; false means that no mark is written.
	DeadSlapped bool
	Text        string
	Notes       []Note
	Duration    Duration
	// Dynamics is the beat-wide MIDI velocity authored by Guitar Pro. Zero means
	// absent and lets export use the first note velocity. A nonzero value must be
	// within 1..127. GP3-5 stores the value on notes, but the last explicit value
	// applies to the whole beat during playback.
	Dynamics    int16
	Display     BeatDisplay
	Octave      Octave
	Status      BeatStatus
	isGrace     bool
	graceOnBeat bool
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
	beat.promoteLegacyTremoloPicking()
	beat.Effect.rememberRasgueado()
	beat.importLegacyTechniques()
	if value, present := beat.Effect.legacyWahValue(); present {
		beat.Effect.WahPedal = legacyWah(value)
	}
	beat.Effect.rememberWah()
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
	voice.Beats[b].PreferredBeamDirection = voice.Beats[b].Display.BeamDirection
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
	if b > 0 {
		previous := &voice.Beats[b-1]
		switch {
		case voice.Beats[b].Display.BreakSecondary != 0:
			previous.BeamingMode = BeatBeamingForceSplitSecondary
		case voice.Beats[b].Display.ForceBeam:
			previous.BeamingMode = BeatBeamingForceMerge
		case voice.Beats[b].Display.BreakBeam:
			previous.BeamingMode = BeatBeamingForceSplit
		}
	}

	return dur, nil
}

func (s *Song) readBeatEffectsV3(c *cursor, noteEffect *NoteEffect) (BeatEffects, NoteEffect, error) {
	var be BeatEffects
	flags, err := c.readByte()
	if err != nil {
		return be, *noteEffect, err
	}
	if flags&0x03 != 0 {
		be.setImportedVibrato(BeatVibratoSlight)
	}
	be.setImportedFade(BeatFadeNone)
	if (flags & 0x10) == 0x10 {
		be.setImportedFade(BeatFadeIn)
	}
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
	if (flags1 & 0x02) == 0x02 {
		be.setImportedVibrato(BeatVibratoSlight)
	}
	be.setImportedFade(BeatFadeNone)
	if (flags1 & 0x10) == 0x10 {
		be.setImportedFade(BeatFadeIn)
	}
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
	if down < 0 || down > 6 || up < 0 || up > 6 {
		return bs, fmt.Errorf("stroke codes down=%d up=%d must be within 0..6", down, up)
	}
	if down > 0 && up > 0 {
		return bs, fmt.Errorf("stroke codes contain conflicting up and down values")
	}
	if up > 0 {
		bs.Kind = BeatStrokeKindBrush
		bs.Direction = BeatStrokeDirectionUp
		bs.Duration = NoteValue(strokeValue(up))
		bs.ExactDuration = strokeScoreTime(up)
	}
	if down > 0 {
		bs.Kind = BeatStrokeKindBrush
		bs.Direction = BeatStrokeDirectionDown
		bs.Duration = NoteValue(strokeValue(down))
		bs.ExactDuration = strokeScoreTime(down)
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
	if bs.Kind != BeatStrokeKindNone {
		bs.importedDuration = bs.Duration
		bs.hasImported = true
	}
	return bs, nil
}

func strokeScoreTime(value int8) *ScoreTime {
	ticks := int64(30)
	if value >= 3 && value <= 6 {
		ticks = int64(60 << (value - 3))
	}
	exact, _ := NewScoreTime(ticks, 1)
	return &exact
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
