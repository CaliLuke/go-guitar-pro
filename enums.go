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
	// TripletFeelDottedEighth alternates a dotted eighth note with a sixteenth note.
	TripletFeelDottedEighth TripletFeel = 3
	// TripletFeelDottedSixteenth alternates a dotted sixteenth note with a thirty-second note.
	TripletFeelDottedSixteenth TripletFeel = 4
	// TripletFeelScottishEighth applies the Scottish eighth-note rhythm.
	TripletFeelScottishEighth TripletFeel = 5
	// TripletFeelScottishSixteenth applies the Scottish sixteenth-note rhythm.
	TripletFeelScottishSixteenth TripletFeel = 6
)

// Hairpin describes a gradual dynamic change beginning on a beat.
type Hairpin int8

// Hairpin values enumerate the gradual dynamic changes supported by GPIF.
const (
	// HairpinNone indicates no gradual dynamic change.
	HairpinNone Hairpin = iota
	HairpinCrescendo
	HairpinDiminuendo
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

// SimileMark identifies a measure-repeat symbol attached to one bar.
type SimileMark int8

// Simile mark values preserve GPIF's one-bar and two-bar repeat symbols.
const (
	// SimileMarkNone indicates that the measure contains its authored notation.
	SimileMarkNone SimileMark = iota
	// SimileMarkSimple repeats the preceding measure.
	SimileMarkSimple
	// SimileMarkFirstOfDouble is the first half of a two-measure repeat symbol.
	SimileMarkFirstOfDouble
	// SimileMarkSecondOfDouble is the second half of a two-measure repeat symbol.
	SimileMarkSecondOfDouble
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
	SlidePickSlideDown SlideType = 5
	SlidePickSlideUp   SlideType = 6
)

// BeatVibrato identifies the authored strength of a beat-wide whammy-bar vibrato.
type BeatVibrato uint8

// Beat vibrato values preserve GPIF's explicit strength.
const (
	BeatVibratoNone   BeatVibrato = 0
	BeatVibratoSlight BeatVibrato = 1
	BeatVibratoWide   BeatVibrato = 2
)

// NoteVibrato identifies the authored strength of a note vibrato.
type NoteVibrato uint8

// Note vibrato values preserve GPIF's explicit strength.
const (
	NoteVibratoNone   NoteVibrato = 0
	NoteVibratoSlight NoteVibrato = 1
	NoteVibratoWide   NoteVibrato = 2
)

// TremoloPickingStyle identifies the notation style of a tremolo-picking effect.
type TremoloPickingStyle uint8

// Tremolo-picking styles include the Guitar Pro default and AlphaTab's
// model-only buzz-roll spelling.
const (
	// TremoloPickingStyleDefault uses diagonal marks across the note stem.
	TremoloPickingStyleDefault TremoloPickingStyle = iota
	// TremoloPickingStyleBuzzRoll uses AlphaTab's z-shaped buzz-roll glyph.
	TremoloPickingStyleBuzzRoll
)

// NoteAccent identifies the authored articulation accent on a note.
type NoteAccent uint8

// Note accent values preserve GPIF's normal, heavy, and tenuto distinctions.
const (
	NoteAccentNone   NoteAccent = 0
	NoteAccentNormal NoteAccent = 1
	NoteAccentHeavy  NoteAccent = 2
	NoteAccentTenuto NoteAccent = 3
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

// BarreShape describes the strings covered by a beat-level barre mark.
type BarreShape uint8

// Beat-level barre shapes distinguish no mark from full- and half-barre marks.
const (
	// BarreShapeNone indicates that the beat has no authored barre mark.
	BarreShapeNone BarreShape = iota
	// BarreShapeFull indicates a barre across all strings.
	BarreShapeFull
	// BarreShapeHalf indicates a barre across half of the strings.
	BarreShapeHalf
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

// BeatBeamingMode controls the beam connection from one beat to the next.
type BeatBeamingMode int8

// Beat beaming modes preserve explicit split and merge instructions.
const (
	// BeatBeamingAuto lets the master-bar grouping rules choose the connection.
	BeatBeamingAuto BeatBeamingMode = iota
	// BeatBeamingForceSplit forces the beam to end before the next beat.
	BeatBeamingForceSplit
	// BeatBeamingForceMerge forces the beam to continue to the next beat.
	BeatBeamingForceMerge
	// BeatBeamingForceSplitSecondary splits only the secondary beam before the next beat.
	BeatBeamingForceSplitSecondary
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
	// HarmonicTypeFeedback identifies a sustained feedback harmonic.
	HarmonicTypeFeedback HarmonicType = 6
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
