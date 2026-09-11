// SPDX-License-Identifier: MIT

package goguitarpro

import (
	"fmt"
	"math"
)

// Velocity and bend constants define the effect scales for Guitar Pro.
const (
	MinVelocity       int16 = 15
	VelocityIncrement int16 = 16
	Forte             int16 = MinVelocity + VelocityIncrement*5
	DefaultVelocity   int16 = Forte

	BendEffectMaxPosition float32 = 12.0
	GPBendSemitone        float32 = 25.0
	GPBendPosition        float32 = 60.0
)

// BendPoint is a single point within a BendEffect.
type BendPoint struct {
	// Position is the normalized curve position from 0 through 12.
	Position uint8
	// ExactOffset preserves a bend or whammy's authored GPIF percentage from 0
	// through 100 when Position cannot represent it exactly. For a programmatic
	// point, a non-nil ExactOffset is authoritative. For an imported point,
	// ExactOffset remains authoritative while Position is unchanged; editing
	// Position makes the legacy field authoritative, including when both fields
	// are edited. Clearing ExactOffset always falls back to Position.
	ExactOffset *float64
	// Value is the signed pitch offset in semitones.
	Value   int8
	Vibrato bool

	importedPosition       uint8
	hasImportedExactOffset bool
}

// BendEffect describes string bends and tremolo bars.
type BendEffect struct {
	Points []BendPoint
	// Value preserves the legacy Guitar Pro summary in units of 1/25 semitone.
	// Use Points for the semantic curve.
	Value int16
	Kind  BendType
}

// GraceEffect represents a grace note effect.
type GraceEffect struct {
	// Duration is a note-value denominator such as 16 or 32.
	Duration uint8
	Fret     int8
	// ExactFret preserves a non-negative GPIF fret that does not fit Fret.
	ExactFret *Fret
	// RawFret preserves the source-format fret byte when Fret is normalized.
	RawFret *int8
	// PercussionArticulation is the grace note's track-local articulation index.
	// It is meaningful only when HasPercussionArticulation is true.
	PercussionArticulation int
	// HasPercussionArticulation reports whether the track-local identity is present.
	HasPercussionArticulation bool
	IsDead                    bool
	IsOnBeat                  bool
	// Sequence is the zero-based order of this grace note in its attached group.
	Sequence   uint8
	Transition GraceEffectTransition
	Velocity   int16
}

// HarmonicEffect represents a harmonic note effect.
type HarmonicEffect struct {
	Pitch  *PitchClass
	Octave *Octave
	// Fret is the legacy integer view of FretFloat. GPIF import truncates
	// a fractional authored fret toward zero after checking the int8 boundary.
	Fret *int8
	// FretFloat preserves the authored GPIF harmonic fret and takes precedence
	// over Fret during GP8 export.
	FretFloat *float64
	Kind      HarmonicType
}

// TremoloPickingEffect represents a tremolo picking effect.
type TremoloPickingEffect struct {
	// Duration is the authored note-value subdivision and remains the rate
	// authority for compatibility. Guitar Pro GPIF supports values 8, 16, and 32.
	Duration Duration
	// Style controls the notation glyph independently from the subdivision.
	Style TremoloPickingStyle
}

// TrillEffect represents a trill effect.
type TrillEffect struct {
	Fret     int8
	Duration Duration
}

// NoteEffect contains all effects which can be applied to one note.
type NoteEffect struct {
	Bend           *BendEffect
	Trill          *TrillEffect
	Graces         []GraceEffect
	TremoloPicking *TremoloPickingEffect
	Harmonic       *HarmonicEffect
	Slides         []SlideType
	LetRing        bool
	LeftHandFinger Fingering
	// HasLeftHandFinger distinguishes an authored zero-valued thumb from an absent fingering.
	HasLeftHandFinger bool
	// Accent is authoritative when nonzero. The legacy accent booleans are used
	// as a compatibility fallback when Accent is NoteAccentNone.
	Accent          NoteAccent
	AccentuatedNote bool
	PalmMute        bool
	RightHandFinger Fingering
	// HasRightHandFinger distinguishes an authored zero-valued thumb from an absent fingering.
	HasRightHandFinger   bool
	HeavyAccentuatedNote bool
	Staccato             bool
	// Hammer preserves the authored hammer/pull origin marker.
	Hammer bool
	// HammerDestination preserves the authored GPIF destination marker independently
	// from Hammer and consumer-derived links. Direct edits control GP8 emission.
	// Finalization does not add, remove, or infer either endpoint.
	HammerDestination bool
	// Tapped preserves the GPIF Tapped note property independently from a
	// hammer/pull origin.
	Tapped bool
	// LeftHandTapped preserves the GPIF left-hand tapping destination marker.
	LeftHandTapped bool
	GhostNote      bool
	DeadNote       bool
	// VibratoStrength is authoritative when nonzero. Vibrato remains the
	// compatibility fallback for callers that only model presence.
	VibratoStrength NoteVibrato
	Vibrato         bool
}

func defaultNoteEffect() NoteEffect {
	return NoteEffect{
		LeftHandFinger:  FingeringOpen,
		RightHandFinger: FingeringOpen,
	}
}

func unpackVelocity(v int16) int16 {
	return MinVelocity + VelocityIncrement*v - VelocityIncrement
}

// readBendEffect reads the shared binary bend-point record without applying
// note- or beat-specific gesture semantics.
func (s *Song) readBendEffect(c *cursor) (*BendEffect, error) {
	kindByte, err := c.readSignedByte()
	if err != nil {
		return nil, err
	}
	be := &BendEffect{Kind: BendType(kindByte)}
	val, err := c.readInt()
	if err != nil {
		return nil, err
	}
	be.Value = int16(val)
	count, err := c.readInt()
	if err != nil {
		return nil, err
	}
	for i := int32(0); i < count; i++ {
		posRaw, err := c.readInt()
		if err != nil {
			return nil, err
		}
		valRaw, err := c.readInt()
		if err != nil {
			return nil, err
		}
		vibrato, err := c.readBool()
		if err != nil {
			return nil, err
		}
		if posRaw < 0 || posRaw > int32(GPBendPosition) {
			return nil, fmt.Errorf("bend point %d position %d is outside 0..%d", i, posRaw, int32(GPBendPosition))
		}
		bp := importedBendPoint(float64(posRaw)*100/float64(GPBendPosition), int8(math.Round(float64(int16(valRaw))/float64(GPBendSemitone))), vibrato)
		be.Points = append(be.Points, bp)
	}
	if count > 0 {
		return be, nil
	}
	return nil, nil
}

func (s *Song) readNoteBendEffect(c *cursor) (*BendEffect, error) {
	bend, err := s.readBendEffect(c)
	if err != nil || bend == nil {
		return bend, err
	}
	bend.Points = canonicalizeStandardBendPoints(bend.Points)
	return bend, nil
}

// canonicalizeStandardBendPoints removes the control points that Guitar Pro
// uses to identify a standard bend gesture. It keeps custom curves unchanged.
func canonicalizeStandardBendPoints(points []BendPoint) []BendPoint {
	if nonmonotonicBendControlRoles(points) {
		return points
	}
	if len(points) == 4 {
		origin, middle1, middle2, destination := points[0], points[1], points[2], points[3]
		if middle1.Vibrato || middle2.Vibrato || middle1.Value != middle2.Value {
			return points
		}
		if destination.Value > origin.Value && middle1.Value > destination.Value ||
			destination.Value == origin.Value && middle1.Value > origin.Value {
			return points
		}
		return []BendPoint{origin, destination}
	}
	if len(points) == 3 {
		origin, middle, destination := points[0], points[1], points[2]
		if middle.Vibrato {
			return points
		}
		if destination.Value > origin.Value && middle.Value > destination.Value ||
			destination.Value == origin.Value && middle.Value > origin.Value {
			return []BendPoint{origin, cloneBendPoint(middle), cloneBendPoint(middle), destination}
		}
		return []BendPoint{origin, destination}
	}
	return points
}

// canonicalizeStandardWhammyPoints removes only the GPIF control points that
// identify standard whammy gestures. Binary whammy curves do not use this
// context and must retain their authored extrema and hold boundaries.
func canonicalizeStandardWhammyPoints(points []BendPoint) []BendPoint {
	if len(points) == 4 {
		origin, middle1, middle2, destination := points[0], points[1], points[2], points[3]
		if middle1.Vibrato || middle2.Vibrato || middle1.Value != middle2.Value {
			return points
		}
		switch {
		case origin.Value < middle1.Value && middle1.Value < destination.Value,
			origin.Value > middle1.Value && middle1.Value > destination.Value,
			origin.Value == middle1.Value && middle1.Value == destination.Value:
			return []BendPoint{origin, destination}
		case (origin.Value > middle1.Value && middle1.Value < destination.Value) ||
			(origin.Value < middle1.Value && middle1.Value > destination.Value):
			if resolvedBendOffset(middle1) == resolvedBendOffset(middle2) {
				return []BendPoint{origin, middle1, destination}
			}
		}
		return points
	}
	if len(points) == 3 {
		origin, middle, destination := points[0], points[1], points[2]
		if middle.Vibrato {
			return points
		}
		if origin.Value < middle.Value && middle.Value < destination.Value ||
			origin.Value > middle.Value && middle.Value > destination.Value ||
			origin.Value == middle.Value && middle.Value == destination.Value {
			return []BendPoint{origin, destination}
		}
	}
	return points
}

// readGraceEffect reads a grace note effect (GP3/GP4).
func (s *Song) readGraceEffect(c *cursor) (GraceEffect, error) {
	fret, err := c.readSignedByte()
	if err != nil {
		return GraceEffect{}, err
	}
	g := GraceEffect{Fret: fret, RawFret: &fret, Velocity: DefaultVelocity, Duration: 1}
	velByte, err := c.readByte()
	if err != nil {
		return g, err
	}
	g.Velocity = unpackVelocity(int16(velByte))
	transByte, err := c.readSignedByte()
	if err != nil {
		return g, err
	}
	durByte, err := c.readByte()
	if err != nil {
		return g, err
	}
	g.Duration = 1 << (7 - durByte)
	g.IsDead = g.Fret == -1
	g.Transition = GraceEffectTransition(transByte)
	return g, nil
}

// readGraceEffectV5 reads a grace note effect (GP5).
func (s *Song) readGraceEffectV5(c *cursor) (GraceEffect, error) {
	fretByte, err := c.readByte()
	if err != nil {
		return GraceEffect{}, err
	}
	rawFret := int8(fretByte)
	g := GraceEffect{Fret: rawFret, RawFret: &rawFret, Velocity: DefaultVelocity, Duration: 1}
	velByte, err := c.readByte()
	if err != nil {
		return g, err
	}
	g.Velocity = unpackVelocity(int16(velByte))
	transByte, err := c.readByte()
	if err != nil {
		return g, err
	}
	g.Transition = GraceEffectTransition(int8(transByte))
	durByte, err := c.readByte()
	if err != nil {
		return g, err
	}
	g.Duration = 1 << (7 - durByte)
	flags, err := c.readByte()
	if err != nil {
		return g, err
	}
	g.IsDead = (flags & 0x01) == 0x01
	g.IsOnBeat = (flags & 0x02) == 0x02
	return g, nil
}

// readTremoloPicking reads tremolo picking effect.
func (s *Song) readTremoloPicking(c *cursor) (TremoloPickingEffect, error) {
	val, err := c.readSignedByte()
	if err != nil {
		return TremoloPickingEffect{}, err
	}
	tp := TremoloPickingEffect{Duration: defaultDuration()}
	switch val {
	case 1:
		tp.Duration.Value = uint16(DurationEighth)
	case 2:
		tp.Duration.Value = uint16(DurationSixteenth)
	case 3:
		tp.Duration.Value = uint16(DurationThirtySecond)
	default:
		c.report(diagnosticSource("Binary.Note.TremoloPicking.Subdivision.Unsupported", "tremolo-picking", ParseDiagnosticUnsupportedFeature), "binary tremolo-picking subdivision is not supported")
	}
	return tp, nil
}

// readSlidesV5 reads slide types for GP5.
func (s *Song) readSlidesV5(c *cursor) ([]SlideType, error) {
	t, err := c.readByte()
	if err != nil {
		return nil, err
	}
	var slides []SlideType
	if (t & 0x01) == 0x01 {
		slides = append(slides, SlideShiftSlideTo)
	}
	if (t & 0x02) == 0x02 {
		slides = append(slides, SlideLegatoSlideTo)
	}
	if (t & 0x04) == 0x04 {
		slides = append(slides, SlideOutDownwards)
	}
	if (t & 0x08) == 0x08 {
		slides = append(slides, SlideOutUpwards)
	}
	if (t & 0x10) == 0x10 {
		slides = append(slides, SlideIntoFromBelow)
	}
	if (t & 0x20) == 0x20 {
		slides = append(slides, SlideIntoFromAbove)
	}
	return slides, nil
}

// readHarmonicV5 reads harmonic for GP5.
func (s *Song) readHarmonicV5(c *cursor) (HarmonicEffect, error) {
	return s.readHarmonicV5ForNote(c, &Note{})
}

func (s *Song) readHarmonicV5ForNote(c *cursor, note *Note) (HarmonicEffect, error) {
	kind, err := c.readSignedByte()
	if err != nil {
		return HarmonicEffect{}, err
	}
	he := HarmonicEffect{}
	switch kind {
	case 1:
		he.Kind = HarmonicTypeNatural
		setLegacyHarmonicFret(&he, int(note.Value))
	case 2:
		he.Kind = HarmonicTypeArtificial
		semitone, err := c.readByte()
		if err != nil {
			return he, err
		}
		accidental, err := c.readSignedByte()
		if err != nil {
			return he, err
		}
		pitch := pitchClassFrom(int8(semitone), &accidental, nil)
		he.Pitch = &pitch
		octByte, err := c.readByte()
		if err != nil {
			return he, err
		}
		oct := Octave(octByte)
		he.Octave = &oct
		playedPitch := int(s.realNoteValue(note, s.currentTrackOrZero())) % 12
		targetHarmonic := int(semitone) + int(accidental) + int(octByte)*12
		if targetHarmonic < playedPitch {
			targetHarmonic += 12
		}
		setLegacyHarmonicFret(&he, targetHarmonic-playedPitch)
	case 3:
		he.Kind = HarmonicTypeTapped
		fretByte, err := c.readByte()
		if err != nil {
			return he, err
		}
		fret := int8(fretByte)
		he.Fret = &fret
		value := legacyHarmonicFret(int(fretByte))
		he.FretFloat = &value
	case 4:
		he.Kind = HarmonicTypePinch
		setLegacyHarmonicFret(&he, 12)
	case 5:
		he.Kind = HarmonicTypeSemi
		setLegacyHarmonicFret(&he, 12)
	default:
		c.report(diagnosticSource("Binary.Note.HarmonicV5.Kind.Unsupported", "harmonics", ParseDiagnosticUnsupportedFeature), "binary GP5 harmonic kind is not supported")
	}
	return he, nil
}

func (s *Song) currentTrackOrZero() int {
	if s.currentTrack == nil {
		return 0
	}
	return *s.currentTrack
}

func legacyHarmonicFret(delta int) float64 {
	switch delta {
	case 2:
		return 2.4
	case 3:
		return 3.2
	case 8:
		return 8.2
	case 10:
		return 9.6
	case 14, 15:
		return 14.7
	case 21, 22:
		return 21.7
	case 4, 5, 7, 9, 12, 16, 17, 19, 24:
		return float64(delta)
	default:
		return 12
	}
}

func setLegacyHarmonicFret(harmonic *HarmonicEffect, delta int) {
	value := legacyHarmonicFret(delta)
	legacy := int8(value)
	harmonic.Fret = &legacy
	harmonic.FretFloat = &value
}

// readTrill reads a trill effect.
func (s *Song) readTrill(c *cursor) (TrillEffect, error) {
	fret, err := c.readSignedByte()
	if err != nil {
		return TrillEffect{}, err
	}
	period, err := c.readSignedByte()
	if err != nil {
		return TrillEffect{}, err
	}
	t := TrillEffect{Fret: fret, Duration: defaultDuration()}
	switch period {
	case 1:
		t.Duration.Value = uint16(DurationSixteenth)
	case 2:
		t.Duration.Value = uint16(DurationThirtySecond)
	case 3:
		t.Duration.Value = uint16(DurationSixtyFourth)
	default:
		c.report(diagnosticSource("Binary.Note.Trill.Period.Unsupported", "note-and-beat-semantics", ParseDiagnosticUnsupportedFeature), "binary trill period is not supported")
	}
	return t, nil
}
