# Semantic score model

`Score` is an alias for `Song`. The alias gives the model a semantic name
without a second writable object tree. Existing callers can continue to use
`Song`.

The model covers Guitar Pro import and GP8 export. It does not cover AlphaTab
rendering, synthesis, or other file importers.

The semantic matrix traces each public field, GPIF wire field, source dispatch,
and public enum member to an executable case. Each case records its stage,
format, value shapes, evidence role, and evidence source. Structural schema
evidence proves XML shape only. It cannot close a semantic behavior obligation.
Enum discovery uses Go type information across the package, so inferred
constants and aliases cannot bypass the matrix. Every exported member name has
its own executable obligation, even when two members have the same value.

## Processing stages

The parser uses these stages:

1. The container reader extracts the format data.
2. The format reader decodes source records.
3. The importer creates independent score occurrences.
4. `FinalizeSong` derives exact and legacy timing values.
5. `ValidateSong` checks the public result without mutation.

`FinalizeSong` is deterministic and idempotent. The parser calls the same
finalizer before it returns a score. A caller can call it after an authored
timing change.

`ValidateSong` checks format-independent invariants. GP8 export uses a separate
capability check because a valid score can exceed one target format.

## Ownership and compatibility

The hierarchy is `Score -> Track -> Staff -> Measure -> Voice -> Beat -> Note`.
Each parsed beat and note is an independent occurrence. Reused GPIF definitions
do not share mutable effect data.

`Track.Staves` preserves all staff data. `Track.Measures` and `Track.Strings`
are compatibility views of the first staff. A non-nil compatibility slice is
the authority for staff 0. This rule applies to finalization, validation, and
GP8 export. Clear the matching compatibility slice before you replace staff 0
directly. Later staves are always authoritative and remain independent.

GPIF master-bar references list bars by track, then by staff. An interior `-1`
voice reference keeps an empty voice slot. A bar-level `-1` replaces one whole
track. The parser reports a short, long, or misplaced bar list as invalid data.

GP8 export preserves tuning and a common capo. It reports non-default legacy
fret counts, connection ports, twelve-string flags, and banjo flags as
omissions. It also reports a custom line count on a pitched staff. Percussion
staff line counts remain supported.

GP8 export preserves the selected MIDI program, primary and effect channels,
volume, balance, mute state, solo state, sound definitions, and sound changes.
It reports bank and effect controllers as one scoped MIDI omission. It also
reports each legacy master or track RSE record as one scoped omission. The RSE
tests assert every descendant covered by those parent reports.

GPIF `AudioEngineState` maps `RSE` to `Track.UseRse` and `MIDI` to false. An
unknown engine state produces an unknown-syntax diagnostic.

`InitialTempo` is the fractional authored value. `Tempo` is its legacy integer
projection. When an opening automation identifies the stale value, an edit to
either representation wins and produces a normalization report. GP8 preserves
the tempo name and ordered automation values. It cannot preserve `HideTempo`,
so it reports that omission.

A score must have a valid opening tempo from `Tempo`, `InitialTempo`, or an
automation at bar zero and position zero. A later automation does not supply
the missing opening value.

GP8 has one playback-state value and one MIDI port per track. Export reports a
normalization when mute and solo are both true or when the effect channel uses
a different port. It also reports a default binding for an unbound track.
String numbers are canonical positions in tuning order. Export reports any
authored number that differs from that position. The ZIP reader restores track
visibility from `LayoutConfiguration` when that record is present.

The existing integer timing fields remain compatibility projections.
`ExactStart` and `ScoreTime` preserve fractional score ticks. The exporter
quantizes values only at a target boundary.

## Authored values and loss

The model preserves ordered grace effects, staff ownership, chord scope,
percussion articulation identity, fractional tempo, and exact duration ratios.
Registered conformance fixtures cover the supported feature slices. Focused
regression tests cover chord scope, fractional tempo, and checked boundaries.
The generated conformance ledger identifies partial projections that remain.

`ParseWithOptions` reports source data that the model cannot preserve. Strict
parse mode rejects selected diagnostic kinds. The default `Parse` function
keeps its permissive behavior.

`PreflightExport` reports target changes before serialization. A strict export
policy rejects normalized or omitted values unless its code is in the explicit
allowlist.

`Song.Author` is the GPIF `Music` value. `Song.Writer` is the separate legacy
binary writer value. GP8 export preserves `Author` and reports a nonempty
`Writer` as omitted.

`Version` records source-format provenance. GP8 output uses the writer version.
Export reports this version change as a normalization. Clipboard ranges are
binary source data, so GP8 export reports them as omitted.

Measure headers own key and triplet-feel values during GP8 export. Non-default
legacy values on `Song` produce normalization reports. Embedded newlines in one
notice item also produce a report because GPIF stores one newline-separated
text value.

GP8 stores track visibility outside GPIF. It stores notation and tablature view
flags in the part configuration. The writer reports other non-default track and
beat display settings. It also reports page setup, marker color, voice direction,
and line-break data that it cannot emit.

Master bars own key, meter, and double-bar output. A non-default compatibility
value on `Measure` produces a normalization report when it conflicts with its
header.

GP8 export preserves master-bar key changes, meter values, section text,
repeats, alternate endings, triplet feel, and double bars. It preserves treble,
bass, alto, tenor, and percussion clefs. It does not write legacy navigation
directions, header-local tempo, or authored meter beam groups. Export reports
each of these omissions. Import and export reject an invalid meter or repeat
count before a value can wrap to a smaller integer type.

Duration arithmetic uses exact rational score ticks until it updates a legacy
integer field. The score starts at tick 960. Each voice starts at its measure
origin. A grace beat does not advance the regular beat timeline. A pickup uses
its longest staff content as its length, including an empty pickup. Repeated
finalization is idempotent.

GP8 preserves all supported note values, one or two dots, and tuplets through
the legacy 255 limit. If both legacy dot flags are true, GP8 emits two dots and
reports a normalization. The legacy 0:0 tuplet means 1:1. GP8 emits the
canonical representation and reports that normalization. Invalid durations
are rejected before export.

A pitched note uses `String` 1 through the staff string count for a fret. A
`String` value of 0 makes `Value` an absolute MIDI note. Validation rejects an
invalid string, note kind, MIDI result, or duration percentage. It also reports
a tie destination without an earlier matching note. Export can keep that tie
because its origin can be outside an imported excerpt.

GP8 preserves note pitch, string, dead-note state, and complete tie markers. It
does not preserve `SwapAccidentals` or `DurationPercent`, so export reports
these fields when they are non-default.

`Beat.Dynamics` is the authored beat-wide value. GP8 uses it when it is set,
including on a rest. If it is zero, GP8 uses the first note velocity. GPIF has
one quantized dynamic for the complete beat, so export reports note velocities
that differ from the selected target dynamic. Text remains valid on rests.
GP8 changes an explicit empty beat to a rest and reports that normalization.

GP8 preserves fade-in, hairpin, octave, and stroke direction. The target uses
an eighth-note stroke duration. Export reports a different source duration. It
also reports rasgueado, pick stroke, slap effects, and beat vibrato because the
writer does not emit them.

GP8 preserves accents, ghost notes, staccato, palm mute, dead notes, let ring,
boolean vibrato, hammer origin, slide flags, harmonic kind, and trill fret.
The writer reports fingerings and tremolo picking because GP8 export does not
emit them. This rule includes an authored thumb value.
Use `HasLeftHandFinger` or `HasRightHandFinger` to mark an authored `Thumb` or
`Open` value. Nonzero named fingers remain compatible without these markers.

GPIF can distinguish vibrato strength, tenuto, hammer endpoints, and tapping
variants. The public model stores less precise values. Parse diagnostics record
each collapsed distinction before import removes it. Validation rejects unknown
fingering and slide enum values. Parse diagnostics reject a missing technique
payload and classify unknown slide bits.

GP8 preserves bend and whammy curves when the target can keep their shape.
The writer can remove a collinear point without changing the curve. It reports
unequal middle values, point vibrato, and summary fields. Strict export rejects
each unapproved change. Parse diagnostics check GPIF curve numbers before they
enter narrow legacy fields.

GPIF consumers interpret some monotonic whammy controls as one standard
gesture. Export reports normalization when that interpretation removes an
interior hold or changes a noncollinear intermediate rate. The report is based
on the interpreted target curve, not only the emitted XML coordinates.

Standard bends use three or four wire control points to identify one gesture.
Import converts these controls to the canonical public gesture. It does not
change a custom curve or remove a point that has vibrato. A missing GPIF bend
destination offset means the end of the note. A middle value without middle
offsets means the midpoint. GP8 export reports an authored initial or final
hold when this standard gesture encoding cannot keep the hold boundary.

GP8 percussion resources cover every resolved staff, including articulations
used only by grace notes. The writer emits staff definitions before the shared
instrument set because the pinned consumer applies that set only to staves that
already exist. GPIF stores one percussion notation line count per track. Equal
staff values are preserved; a later conflicting value is reported and strict
export refuses it unless that exact normalization is allowed.

`HarmonicEffect.FretFloat` is authoritative when both harmonic fret fields are
present. GP8 reports a conflicting legacy `Fret` value as a normalization. It
also reports harmonic pitch spelling and octave because the target does not
emit them. Import rejects malformed, non-finite, and out-of-range harmonic
frets. A GPIF feedback harmonic remains distinct through import and GP8 export.

GP8 preserves a chord name, string pattern, and representable first fret. It
reports barres, fingerings, omissions, and legacy chord descriptions. Imported
chord occurrences own separate mutable slices and pointers. Track and staff
definition scopes remain separate when they use the same local identifier.

Grace notes keep their order, exact fret, articulation identity, dead state,
placement, transition, and supported duration. GP8 reports a raw source fret,
an unsupported duration, a conflicting legacy fret, a noncanonical velocity,
or a bend transition. Reused grace definitions produce independent occurrence
data.

Percussion articulations retain separate notation and playback identities even
when two definitions use the same output MIDI value. GP8 also creates builtin
fallback definitions for identity-free main and grace notes. Import and public
validation reject invalid articulation identities and MIDI ranges. Notehead
options change only the requested output notation.

GP8 keeps the authored order of tempo and sound automations. It does not sort
same-bar events by position. Parse diagnostics report missing values, broken
sound references, unknown automation types, and unsupported channel-strip
types. They also report linear tempo or sound interpolation because the public
records do not contain an interpolation field.

Score lyrics and track lyrics have separate scopes. GP8 preserves ordered track
lyrics and their offsets, but it does not emit binary score lyrics. Export
reports the score-level omission. Beat text remains separate and is preserved
on rests. An undispatched GPIF lyric source produces a loss diagnostic because
the public model does not retain that source state.

An enabled local backing track must refer to an archive entry with audio data.
Its frame padding must fit in a signed 64-bit frame count. Sync points preserve
their authored bar, position, tempo, visibility, frame, and media-time values.
GP8 export reports backing tracks and sync points because it does not emit them.

The GPIF source audit accounts for every known wire field and named dispatch.
It rejects conflicting duplicate properties. It also keeps an explicit source
state when zero and absence have different meanings. New wire fields or
dispatch cases fail the inventory until the ledger classifies them.

Validation, preflight, and export use the same format-independent invariants.
These read-only operations do not change the score. A failed strict file export
does not replace an existing file and does not leave a new partial file.

The whole-corpus receipt covers every inventoried fixture in one pinned
AlphaTab batch. It stores counted diagnostic signatures and exact semantic
differences. Each difference records its path, both values, a narrow reason,
and an open issue. A snapshot update does not accept a new difference. A new
or changed difference has no classification, and the strict test fails.
Each counted diagnostic also hashes every source and object location. This hash
detects location drift without storing every repeated diagnostic in full.

Test sufficiency uses several independent signals. The contract inventory
checks breadth. The AlphaTab oracle checks results through another importer.
Metamorphic tests check equivalent structures. Deterministic mutation tests
check that realistic faults make tests fail. Structural fuzz tests check
malformed input and valid nested score graphs. Statement coverage remains a
minimum execution measure. No one signal proves that the code has no defects.
Together, these signals make missing evidence visible and measurable.

Each public field has a disposition-bearing runtime assertion. The semantic
matrix compares that declaration with the complete public-field partition. A
new field cannot inherit a broad feature claim. The mutation gate swaps two
unrelated dispositions and requires the matrix to fail. Another mutation
removes represented-field serialization and requires its case to fail. These
checks do not replace stage-specific and value-specific behavior evidence.

This policy supports gradual migration. New code can use `Score`, exact value
types, staves, diagnostics, and preflight reports. Existing `Song` code remains
source compatible.
