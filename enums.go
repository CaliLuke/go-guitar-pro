// SPDX-License-Identifier: MIT

package goguitarpro

// TripletFeel represents an enumeration of different triplet feels.
type TripletFeel int8

// Triplet feel values describe the supported swing subdivisions.
const (
	// TripletFeelNone disables triplet-feel playback.
	TripletFeelNone      TripletFeel = 0
	TripletFeelEighth    TripletFeel = 1
	TripletFeelSixteenth TripletFeel = 2
)

// MeasureClef represents available clefs.
type MeasureClef int8

// Measure clef values enumerate the supported clefs.
const (
	// MeasureClefTreble selects the treble clef.
	MeasureClefTreble MeasureClef = iota
	MeasureClefBass
	MeasureClefTenor
	MeasureClefAlto
)

// LineBreak represents a line break directive.
type LineBreak int8

// Line break values describe score wrapping behavior.
const (
	// LineBreakNone leaves line wrapping unchanged.
	LineBreakNone    LineBreak = 0
	LineBreakBreak   LineBreak = 1
	LineBreakProtect LineBreak = 2
)

// SlideType represents all supported slide types.
type SlideType int8

// Slide type values describe how a slide enters or leaves a note.
const (
	// SlideIntoFromAbove slides into a note from above.
	SlideIntoFromAbove SlideType = -2
	SlideIntoFromBelow SlideType = -1
	SlideNone          SlideType = 0
	SlideShiftSlideTo  SlideType = 1
	SlideLegatoSlideTo SlideType = 2
	SlideOutDownwards  SlideType = 3
	SlideOutUpwards    SlideType = 4
)

// NoteType represents note types.
type NoteType uint8

// Note type values describe the semantic kind of a note.
const (
	// NoteTypeRest represents a rest rather than a sounded note.
	NoteTypeRest   NoteType = 0
	NoteTypeNormal NoteType = 1
	NoteTypeTie    NoteType = 2
	NoteTypeDead   NoteType = 3
)

// BeatStatus represents beat status.
type BeatStatus uint8

// Beat status values describe whether a beat sounds or rests.
const (
	// BeatStatusEmpty represents a beat with no content.
	BeatStatusEmpty  BeatStatus = 0
	BeatStatusNormal BeatStatus = 1
	BeatStatusRest   BeatStatus = 2
)

// VoiceDirection represents the direction of beams for a voice.
type VoiceDirection int8

// Voice direction values describe beam orientation.
const (
	// VoiceDirectionNone leaves beam direction unspecified.
	VoiceDirectionNone VoiceDirection = 0
	VoiceDirectionUp   VoiceDirection = 1
	VoiceDirectionDown VoiceDirection = 2
)

// TupletBracket describes where a tuplet bracket begins or ends.
type TupletBracket int8

// Tuplet bracket values mark bracket boundaries.
const (
	// TupletBracketNone indicates no tuplet bracket boundary.
	TupletBracketNone  TupletBracket = 0
	TupletBracketStart TupletBracket = 1
	TupletBracketEnd   TupletBracket = 2
)

// Octave describes an octave-transposition sign.
type Octave uint8

// Octave values enumerate supported octave-transposition signs.
const (
	// OctaveNone indicates no octave transposition.
	OctaveNone              Octave = 0
	OctaveOttava            Octave = 1
	OctaveQuindicesima      Octave = 2
	OctaveOttavaBassa       Octave = 3
	OctaveQuindicesimaBassa Octave = 4
)

// BeatStrokeDirection describes the direction of a beat stroke.
type BeatStrokeDirection int8

// The values for the beat-stroke direction describe the pick direction.
const (
	// BeatStrokeDirectionNone indicates no stroke direction.
	BeatStrokeDirectionNone BeatStrokeDirection = 0
	BeatStrokeDirectionUp   BeatStrokeDirection = 1
	BeatStrokeDirectionDown BeatStrokeDirection = 2
)

// SlapEffect describes a slap-style articulation.
type SlapEffect uint8

// Slap effect values enumerate supported slap articulations.
const (
	// SlapEffectNone indicates no slap articulation.
	SlapEffectNone     SlapEffect = 0
	SlapEffectTapping  SlapEffect = 1
	SlapEffectSlapping SlapEffect = 2
	SlapEffectPopping  SlapEffect = 3
)

// BendType identifies a bend gesture.
type BendType int8

// Bend type values enumerate supported bend gestures.
const (
	// BendTypeNone indicates no bend gesture.
	BendTypeNone            BendType = 0
	BendTypeBend            BendType = 1
	BendTypeBendRelease     BendType = 2
	BendTypeBendReleaseBend BendType = 3
	BendTypePrebend         BendType = 4
	BendTypePrebendRelease  BendType = 5
	BendTypeDip             BendType = 6
	BendTypeDive            BendType = 7
	BendTypeReleaseUp       BendType = 8
	BendTypeInvertedDip     BendType = 9
	BendTypeReturn          BendType = 10
	BendTypeReleaseDown     BendType = 11
)

// GraceEffectTransition identifies the transition from a grace note.
type GraceEffectTransition int8

// The values for grace-effect transitions identify transitions from grace notes.
const (
	// GraceEffectTransitionNone indicates no transition effect.
	GraceEffectTransitionNone   GraceEffectTransition = 0
	GraceEffectTransitionSlide  GraceEffectTransition = 1
	GraceEffectTransitionBend   GraceEffectTransition = 2
	GraceEffectTransitionHammer GraceEffectTransition = 3
)

// HarmonicType identifies a harmonic technique.
type HarmonicType int8

// Harmonic type values enumerate supported harmonic techniques.
const (
	// HarmonicTypeNatural identifies a natural harmonic.
	HarmonicTypeNatural    HarmonicType = 1
	HarmonicTypeArtificial HarmonicType = 2
	HarmonicTypeTapped     HarmonicType = 3
	HarmonicTypePinch      HarmonicType = 4
	HarmonicTypeSemi       HarmonicType = 5
)

// Accentuation identifies a dynamic accent strength.
type Accentuation uint8

// Accentuation values enumerate dynamic accent strengths.
const (
	// AccentuationNone indicates no accentuation.
	AccentuationNone       Accentuation = 0
	AccentuationVerySoft   Accentuation = 1
	AccentuationSoft       Accentuation = 2
	AccentuationMedium     Accentuation = 3
	AccentuationStrong     Accentuation = 4
	AccentuationVeryStrong Accentuation = 5
)

// Fingering identifies the finger used to play a note.
type Fingering int8

// Fingering values enumerate open strings and finger choices.
const (
	// FingeringOpen indicates an open string.
	FingeringOpen    Fingering = -1
	FingeringThumb   Fingering = 0
	FingeringIndex   Fingering = 1
	FingeringMiddle  Fingering = 2
	FingeringAnnular Fingering = 3
	FingeringLittle  Fingering = 4
)

// ChordType identifies a chord quality.
type ChordType uint8

// ChordAlteration identifies an interval alteration.
type ChordAlteration uint8

// Chord alteration values enumerate supported interval alterations.
const (
	// ChordAlterationPerfect leaves the interval unaltered.
	ChordAlterationPerfect    ChordAlteration = 0
	ChordAlterationDiminished ChordAlteration = 1
	ChordAlterationAugmented  ChordAlteration = 2
)

// ChordExtension identifies an added chord extension.
type ChordExtension uint8

// Chord extension values enumerate supported added tones.
const (
	// ChordExtensionNone indicates no extension.
	ChordExtensionNone       ChordExtension = 0
	ChordExtensionNinth      ChordExtension = 1
	ChordExtensionEleventh   ChordExtension = 2
	ChordExtensionThirteenth ChordExtension = 3
)

// DirectionSign identifies a score navigation marker.
type DirectionSign int

// Direction sign values enumerate score navigation markers.
const (
	// DirectionSignCoda identifies a coda marker.
	DirectionSignCoda DirectionSign = iota
	DirectionSignDoubleCoda
	DirectionSignSegno
	DirectionSignSegnoSegno
	DirectionSignFine
	DirectionSignDaCapo
	DirectionSignDaCapoAlCoda
	DirectionSignDaCapoAlDoubleCoda
	DirectionSignDaCapoAlFine
	DirectionSignDaSegno
	DirectionSignDaSegnoAlCoda
	DirectionSignDaSegnoAlDoubleCoda
	DirectionSignDaSegnoAlFine
	DirectionSignDaSegnoSegno
	DirectionSignDaSegnoSegnoAlCoda
	DirectionSignDaSegnoSegnoAlDoubleCoda
	DirectionSignDaSegnoSegnoAlFine
	DirectionSignDaCoda
	DirectionSignDaDoubleCoda
)
