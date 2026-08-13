// SPDX-License-Identifier: MIT

package goguitarpro

import "math"

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
	Position uint8
	Value    int8
	Vibrato  bool
}

// BendEffect describes string bends and tremolo bars.
type BendEffect struct {
	Points []BendPoint
	Value  int16
	Kind   BendType
}

// GraceEffect represents a grace note effect.
type GraceEffect struct {
	Duration   uint8
	Fret       int8
	IsDead     bool
	IsOnBeat   bool
	Transition GraceEffectTransition
	Velocity   int16
}

// HarmonicEffect represents a harmonic note effect.
type HarmonicEffect struct {
	Pitch  *PitchClass
	Octave *Octave
	Fret   *int8
	Kind   HarmonicType
}

// TremoloPickingEffect represents a tremolo picking effect.
type TremoloPickingEffect struct {
	Duration Duration
}

// TrillEffect represents a trill effect.
type TrillEffect struct {
	Fret     int8
	Duration Duration
}

// NoteEffect contains all effects which can be applied to one note.
type NoteEffect struct {
	Bend                 *BendEffect
	Trill                *TrillEffect
	Grace                *GraceEffect
	TremoloPicking       *TremoloPickingEffect
	Harmonic             *HarmonicEffect
	Slides               []SlideType
	LetRing              bool
	LeftHandFinger       Fingering
	AccentuatedNote      bool
	PalmMute             bool
	RightHandFinger      Fingering
	HeavyAccentuatedNote bool
	Staccato             bool
	Hammer               bool
	GhostNote            bool
	Vibrato              bool
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

// readBendEffect reads a bend effect.
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
		bp := BendPoint{
			Position: uint8(math.Round(float64(int16(posRaw)) * float64(BendEffectMaxPosition) / float64(GPBendPosition))),
			Value:    int8(math.Round(float64(int16(valRaw)) / float64(GPBendSemitone))),
			Vibrato:  vibrato,
		}
		be.Points = append(be.Points, bp)
	}
	if count > 0 {
		return be, nil
	}
	return nil, nil
}

// readGraceEffect reads a grace note effect (GP3/GP4).
func (s *Song) readGraceEffect(c *cursor) (GraceEffect, error) {
	fret, err := c.readSignedByte()
	if err != nil {
		return GraceEffect{}, err
	}
	g := GraceEffect{Fret: fret, Velocity: DefaultVelocity, Duration: 1}
	velByte, err := c.readByte()
	if err != nil {
		return g, err
	}
	g.Velocity = unpackVelocity(int16(velByte))
	durByte, err := c.readByte()
	if err != nil {
		return g, err
	}
	g.Duration = 1 << (7 - durByte)
	g.IsDead = g.Fret == -1
	transByte, err := c.readSignedByte()
	if err != nil {
		return g, err
	}
	g.Transition = GraceEffectTransition(transByte)
	return g, nil
}

// readGraceEffectV5 reads a grace note effect (GP5).
func (s *Song) readGraceEffectV5(c *cursor) (GraceEffect, error) {
	fretByte, err := c.readByte()
	if err != nil {
		return GraceEffect{}, err
	}
	g := GraceEffect{Fret: int8(fretByte), Velocity: DefaultVelocity, Duration: 1}
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
	case 3:
		tp.Duration.Value = uint16(DurationSixteenth)
	case 2:
		tp.Duration.Value = uint16(DurationThirtySecond)
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
	kind, err := c.readSignedByte()
	if err != nil {
		return HarmonicEffect{}, err
	}
	he := HarmonicEffect{}
	switch kind {
	case 1:
		he.Kind = HarmonicTypeNatural
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
	case 3:
		he.Kind = HarmonicTypeTapped
		fretByte, err := c.readByte()
		if err != nil {
			return he, err
		}
		fret := int8(fretByte)
		he.Fret = &fret
	case 4:
		he.Kind = HarmonicTypePinch
	case 5:
		he.Kind = HarmonicTypeSemi
	}
	return he, nil
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
	}
	return t, nil
}
