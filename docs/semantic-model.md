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

A `supported` capability stage has one closure claim in the semantic matrix.
The claim selects one typed source, one non-default value, one stage, and one executed obligation.
The case emits the claim identity at the exact assertion site.
An export receipt binds its primary, wire, and report assertions in one claim context.
Independent evidence IDs run AlphaTab-gated callbacks that emit matching claim identities.
A structured limit names the pinned oracle, exact obligation, and source test.
The inventory rejects missing claims, duplicate claims, and claims for stages that are not supported.

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

`Beat.Lyrics` is the ordered, beat-scoped authored lyric sequence. It is
independent from `Beat.Text`, `Track.Lyrics`, and `Song.Lyrics`. Nil means that
the GPIF `Lyrics` element was absent; a non-nil empty slice preserves an
authored empty element. Direct public edits are authoritative for GP8 export.
Each parsed occurrence owns independent lyric storage, even when multiple
voices reference one GPIF beat definition.

`Beat.DeadSlapped` is the independent authored beat-level dead-slap marker.
GPIF presence is authoritative regardless of element text. A note-free marked
beat imports with `BeatStatusNormal`; an ordinary note-free unmarked beat
imports as `BeatStatusRest`. Direct edits to the marker control GP8 output and
do not rewrite `Beat.Status`, add notes, or change note kinds and tap/slap/pop
effects. GP3 through GP5 do not supply this marker. Explicit empty beats retain
their existing GP8 normalization to rests.

`BeatEffects.Golpe` is the sole authority for an authored beat-level golpe.
`GolpeTypeThumb` and `GolpeTypeFinger` preserve the two GPIF spellings, while
`GolpeTypeNone` represents source absence. Direct edits control GP8 output and
do not change the beat's notes or its other techniques. Each parsed occurrence
owns its value even when several score positions reuse one GPIF beat definition.
GP3 through GP5 have no golpe source record.

`Track.Staves` preserves all staff data. `Track.Measures` and `Track.Strings`
are compatibility views of the first staff. A non-nil compatibility slice is
the authority for staff 0. This rule applies to finalization, validation, and
GP8 export. Clear the matching compatibility slice before you replace staff 0
directly. Later staves are always authoritative and remain independent.

`Track.Number`, `MeasureHeader.Number`, and `Measure.Number` are one-based
ordinals. Fields whose names end in `Index` are zero-based references.
`Staff.CapoFret` is the non-negative authored capo fret and is the authority for
each staff. `Track.CapoFret` is the legacy first-staff compatibility scalar. On
a parsed score, direct staff edits are authoritative while the track scalar is
unchanged. Changing the parsed track scalar applies that value to every staff
at export; if both views changed, the legacy track edit wins. For a
programmatic score, any nonzero staff capo makes the staff values authoritative.
When every staff capo is zero, a nonzero legacy scalar applies to every staff.
Export computes this reconciliation without mutating the authored score. Tuning
values are absolute MIDI note numbers for open strings. `Staff.TuningName` is
the independently authored tuning label. `Staff.Strings` remains the sole
authority for pitches and string order, and `Staff.CapoFret` remains the capo
authority. Editing a label never recalculates or changes those values. GPIF
track-level tuning labels initialize every staff. A staff-local `Label` element,
including an explicitly empty element, overrides that inherited label; a
staff-local tuning property without `Label` keeps the inherited name. GP3
through GP5 do not author a tuning label, so their imported value is empty.

GPIF master-bar references list bars by track, then by staff. An interior `-1`
voice reference keeps an empty voice slot. A bar-level `-1` replaces one whole
track. The parser reports a short, long, or misplaced bar list as invalid data.

GP8 export preserves each staff's tuning and capo. It reports non-default legacy
fret counts, connection ports, twelve-string flags, and banjo flags as
omissions. It also reports a custom line count on a pitched staff. Percussion
staff line counts remain supported.

`Staff.DisplayTranspositionPitch` is the staff-authoritative notation offset in
semitones. For legacy GPIF `PartSounding`, `TranspositionPitch` initializes the
display offset while `NominalKey` independently derives each staff's effective
key; `Neutral` means no key offset. GPIF `Transpose` takes precedence over both
legacy values when present. Conflicting legacy values receive scoped lossy
diagnostics. `MeasureHeader.KeySignature` remains the concert and wire
authority, while each staff's `Measure.KeySignature` retains its effective
displayed key. Editing a staff offset changes export without rewriting tuning,
string, fret, or the concert key. GP8 stores one coupled display/key offset per
track, so export reports an independent effective-key combination and a
differing later-staff display value. The independent key uses the existing
`gp8.normalize.measure-key-authority` report. Percussion consumers reset the
display offset, which is also reported.

`Staff.TranspositionPitch` is the independent sounding offset. Sounding MIDI is
the open-string, capo, and fret pitch minus this value. The field never rewrites
those authored values. GP8 GPIF has no sounding-offset destination, so every
nonzero value is reported as omitted. Source octave and chromatic components are
combined with checked signed 32-bit arithmetic before they enter the model.

GP8 export preserves the selected MIDI program, primary and effect channels,
volume, balance, mute state, solo state, sound definitions, and sound changes.
`Song.Channels` retains all authored legacy channel-table slots, including
invalid sentinel program values. Program validation is reference-aware: an
unused slot does not invalidate the score or block export, while every track
that selects a slot requires its program to be within 0 through 127. Export and
validation do not rewrite either selected or unused program values. Percussion
channel selection keeps its existing import normalization to program zero.
`MidiChannel.Bank` and `TrackSound.Bank` use the combined MIDI bank range 0
through 16383. GPIF sound `MSB` and `LSB` values must each fit 0 through 127.
The first explicit `TrackSound` owns the initial program and bank; import mirrors
those values into the channel. When a track has no explicit sound table, the
channel supplies the GP8 sound fallback. Export reports a normalization if an
explicit first sound conflicts with its channel mirror. Sound automations keep
their authored order, including multiple bank-and-program changes at one score
position. GP8 reports the remaining legacy effect controllers as one scoped
MIDI omission. It also reports each legacy master or track RSE record as one
scoped omission. The RSE tests assert every descendant covered by those parent
reports.

`SoundAutomation.Position` supports finite values from 0 through 1 in every
bar. Bar zero also supports opening preroll from -0.125 up to zero. This
bounded contract follows the GP7 grace fixture; it does not define a universal
GPIF limit. Validation and GP8 preflight reject earlier preroll, negative
positions in later bars, and non-finite positions.

Direct edits to the position and sound reference control GP8 output. Export
retains each value and the slice order, including equal positions and preroll
events after position-zero events. It does not move events to zero or modify
the authored records.

GPIF `AudioEngineState` maps `RSE` to `Track.UseRse` and `MIDI` to false. An
unknown engine state produces an unknown-syntax diagnostic.

`InitialTempo` is the fractional authored value. `Tempo` is its legacy integer
projection. When an opening automation identifies the stale value, an edit to
either representation wins and produces a normalization report. GPIF tempo and
sound automations preserve authored order, interpolation, text, and per-event
visibility. `Hidden` uses an inverse flag so the zero value keeps the historical
visible default for programmatic callers. `Song.TempoName` remains the
compatibility authority for the opening tempo label; clear it to edit that label
through `TempoAutomation.Text`. GP8 preserves the automation wire values. The
pinned consumer does not retain hidden visibility on the instrument automation
it derives from a sound record, so export reports that specific omission.

`Song.HideTempo` supplies visibility when GP8 synthesizes an opening tempo
at bar zero and position zero. The first explicit opening event, wherever it
appears in authored order, owns visibility. A disagreement with `Song.HideTempo`
produces `gp8.normalize.tempo-visibility-authority`; strict preservation requires
an explicit allowance. GPIF import initializes the compatibility flag from that
opening event. To edit visibility on a parsed GPIF score, update its opening
`TempoAutomation.Hidden` and matching `Song.HideTempo`. Export does not mutate
either view.

GP5.1's hide-tempo flag survives through the synthesized event;
pinned AlphaTab discards that legacy source flag but retains GPIF visibility.

`Song.VolumeAutomations` owns the channel-strip volume events. Direct edits
control GP8 output. The writer retains track ownership, bar, position, value,
linear flag, and event order within each track, including equal positions.
GPIF groups events by track, so Go reimport groups the public slice by track.
Cross-track interleaving has no playback meaning and is not retained.

Pinned AlphaTab ignores these channel-strip events. Export reports each event
with `gp8.omit.volume-automation-consumer`; strict preservation requires an
explicit allowance. This contract follows the accepted revision of issue #59.
Legacy beat-local mix-table volume changes remain a separate capability.

`Measure.SustainPedals` is the ordered sustain-pedal sequence for that track's
first staff and measure. GPIF reference 1 maps to `SustainPedalTypeDown`, and
reference 3 maps to `SustainPedalTypeRelease`. If a down state continues into a
bar with no explicit marker, import derives one `SustainPedalTypeHold` marker at
position 0. A hold is valid only in that canonical form. Marker positions must
be finite, within 0 through 1, and strictly increasing inside a measure. GP8
writes down and release records in public order and skips derived holds; the
consumer reconstructs the hold from the surrounding pedal state. A bar entered
with the pedal down cannot contain a Down marker, even after an earlier Release
in that bar. GPIF consumers determine Down-versus-Hold from the state at bar
entry and would otherwise silently reinterpret that Down as Hold. Validation
and preflight reject this non-representable sequence. Pedal playback links and
GP3 through GP5 storage are outside this contract.

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

`BendPoint.ExactOffset` preserves a note bend's authored GPIF position as a
percentage from 0 through 100 when the legacy 0-through-12 `Position` cannot
represent it. A programmatic non-nil exact offset is authoritative; nil falls
back to `Position`. On an imported point, the exact offset remains authoritative
while `Position` is unchanged. Editing `Position` makes the legacy view
authoritative, including when both views changed, and export reports that
conflict. Binary note-bend offsets are checked on their native 0-through-60
scale before conversion. Bend heights retain their existing whole-semitone
projection. This exact-offset contract does not apply to beat whammy curves.

`Beat.BarreFret` and `Beat.BarreShape` are the paired authority for an authored
beat-level barre mark. A nil fret must use `BarreShapeNone`; a present checked
fret must use `BarreShapeFull` or `BarreShapeHalf`. Direct edits to either field
are authoritative, and validation rejects incomplete pairs before GP8 export.
These fields are independent from `Chord.Barres`, which describes fingering
ranges inside a chord diagram.

`BeatEffects.Stroke.Kind` distinguishes a GPIF `Brush` property from an
`Arpeggio` element. `KindNone` with a non-none direction retains the historical
programmatic behavior and exports as an arpeggio. `Duration` remains the
note-value denominator compatibility view; the supported values 1 through 128
convert to `3840 / Duration` ticks, while zero requests the target default.
`ExactDuration` preserves the authored
960-PPQ tick count, including GPIF XProperty `687935489`. A nil exact value
preserves source absence. On an imported stroke, exact timing owns export while
the compatibility duration is unchanged. Editing `Duration` makes that view
authoritative, including when both views changed; clearing `ExactDuration`
also falls back to `Duration`. Export computes this reconciliation without
mutating the score. GP8 can store integral values from 0 through 2147483647;
fractional or larger exact values receive a brush-specific omission report.
GP3 through GP5 stroke codes 1 and 2 map to 30 ticks, followed by 60, 120, 240,
and 480 ticks for codes 3 through 6.

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

GPIF key modes accept the exact `Major`, `major`, `Minor`, and `minor`
spellings; an absent or empty mode defaults to major. An unknown spelling keeps
permissive parsing compatible by projecting to major and produces an explicit
unsupported-feature diagnostic, so strict parsing can reject it. The accidental
count and mode remain independent on every adjacent measure header. GP8 writes
the canonical `Major` or `Minor` spelling.

`MeasureHeader.FreeTime` preserves the GPIF marker on its exact master bar.
The marker is present or absent independently on every bar; it does not inherit
and it does not replace the numeric `TimeSignature`. Finalization therefore uses
the exact notated meter length for every non-pickup bar, even when a free-time
bar has empty, shorter, or longer voice content. Only the existing opening
`Song.Anacrusis` rule uses content length. GP8 writes `<FreeTime>` only when the
flag is true. GPIF defines this as element presence, so even a source spelling
such as `<FreeTime>false</FreeTime>` means true.

GP8 stores track visibility outside GPIF. It stores notation and tablature view
flags in the part configuration. The writer reports other non-default track and
beat display settings. It also reports page setup, marker color, voice direction,
and line-break data that it cannot emit.

`MeasureHeader.BeamingRules` is the optional authored custom grouping on one
exact master bar. It owns its duration denominator and one through 32 positive
group sizes; it does not inherit from adjacent bars or derive from
`TimeSignature.Beams`. Supported denominators are 1, 2, 4, 8, 16, 32, 64, 128,
and 256. A group size is limited to 2,147,483,647. GPIF trailing zero group
properties are padding and are removed during import; a zero before a later
positive group is invalid.

`Beat.BeamingMode` is the canonical connection from that beat to the next.
`Beat.InvertBeamDirection` and `Beat.PreferredBeamDirection` are independent
authored overrides. GP5 stores connection flags on the following beat, so import
moves them to the previous beat while retaining the raw `Beat.Display` fields.
GPIF applies primary and secondary beaming XProperties in source order. A later
primary split or merge replaces an earlier secondary split; a later secondary
split replaces merge but does not replace primary split.
Known beaming and brush-duration XProperty IDs are accepted and preserved.
Malformed or orphan brush timing receives scoped invalid-data diagnostics.
Only future or otherwise unhandled IDs remain explicit unknown syntax.
After import, edits to the canonical fields are authoritative; the legacy fields
do not reconcile back into them. GP8 writes the canonical fields and reports
each non-default legacy-only display field separately. A GPIF grace-beat stem
orientation remains outside this contract because grace occurrences project to
`GraceEffect`, not to a public `Beat`.

Master bars own key, meter, and double-bar output. A non-default compatibility
value on `Measure` produces a normalization report when it conflicts with its
header.

`MeasureHeader.Directions` is the complete authored set of navigation targets
and jumps. GP5 and GPIF import canonicalize it in `DirectionSign` order and
remove duplicates. `MeasureHeader.Direction` remains the legacy single-marker
view and receives the final marker in canonical order after import. On a parsed
score, an unchanged legacy pointer leaves `Directions`
authoritative. Changing the pointer replaces the set with that singleton, and
clearing it removes all directions; this rule also wins when both views change.
For a programmatic score, a non-nil `Directions` slice is authoritative, and a
nil slice falls back to `Direction`. Validation and GP8 export reconcile and
canonicalize these views without modifying the score. Navigation playback and
repeat traversal remain outside the library contract.

`Measure.ClefOctave` is the authored octave shift attached to that staff's clef.
It is independent from `Beat.Octave`, which applies only to its beat. The normal
first-staff `Track.Measures` compatibility rule applies to post-parse edits;
later staff measures remain independently authoritative.

`MeasureHeader.Fermatas` is the authoritative set of authored holds for one
master bar. Each `Fermata` keeps an exact measure-relative `ScoreTime` offset,
the Short, Medium, or Long symbol, and a finite non-negative length. Offsets
must be unique and earlier than the exact notated measure length. GPIF rational
offsets remain exact, including sub-tick values, and GP8 export writes the
reduced rational quarter-note position. A consumer can associate a fermata with
a beat at the same offset, but `Beat` does not contain a second mutable copy.
GP3 through GP5 fermatas and playback-duration stretching are outside this
contract.

GPIF retains each exact fermata offset. Pinned AlphaTab converts the rational
offset to a signed 32-bit tick through binary64 arithmetic. This conversion
can move fractional offsets and some whole-tick offsets. Export reports each
movement with `gp8.normalize.fermata-consumer-offset`. If two offsets collide,
`gp8.omit.fermata-consumer-collision` identifies the overwritten and replacing
fermata indices and the consumer tick. Strict preservation refuses these
reports unless the caller allows their codes.

GP8 export preserves master-bar key changes, meter values, section text,
repeats, alternate endings, triplet feel, and double bars. It preserves treble,
bass, alto, tenor, and percussion clefs, including 8va, 8vb, 15ma, and 15mb
clef shifts. It also preserves every navigation target and jump on a master
bar. It writes authored `MeasureHeader.BeamingRules`, but it does not write the
legacy `TimeSignature.Beams` compatibility array or header-local tempo. Export
reports each of these omissions. Import and export reject an invalid meter,
repeat count, or beaming boundary before a value can wrap to a smaller integer
type.

`MeasureHeader.RepeatStart` marks the start of a repeat section.
`MeasureHeader.RepeatCount` is the total number of passes displayed at the
repeat end, so a repeat displayed as `6x` has a count of 6. Zero means that the
measure is not a repeat end. Positive counts from 1 through 128 are supported.
GP3 and GP4 count offsets are decoded at the binary reader boundary; the public
model, GP5, GPIF, GP8 export, and AlphaTab comparisons all use total passes.

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

`Measure.SimileMark` preserves GPIF's one-measure repeat symbol and both halves
of its two-measure repeat symbol. The value belongs to each staff measure, not
the shared measure header, because different staves may repeat independently.
Unknown source text and undefined programmatic enum values are rejected.

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
that differ from the selected target dynamic. Direct edits to either field remain
authoritative. Export does not change their authored values.

Canonical velocity 47 is P, while 48 quantizes to P. Export reports a
noncanonical explicit beat value with `gp8.normalize.beat-dynamic`. It separately reports differing note
velocities with `gp8.normalize.note-velocity`, once per beat. Strict export
requires an allowance for each applicable code and returns no output otherwise.

GPIF has no independent per-note velocity destination. Text remains valid on rests.
GP8 changes an explicit empty beat to a rest and reports that normalization.

`Beat.Legato` preserves the authored GPIF beat-level origin and destination
flags. The pointer is absent when the source has no `Legato` element. Every
parsed occurrence owns an independent record, including occurrences created
from one reused GPIF beat definition. Origin-only and destination-only records
are valid excerpt boundaries. The library does not derive missing endpoints or
reuse hammer and slide fields for this relationship.

GP8 writes both authored
attributes. The pinned consumer retains origins and derives each destination
from the preceding origin. Export reports each destination mismatch, including
excerpt boundaries and edits, with `gp8.omit.legato-consumer-destination`.
Strict preservation refuses that loss unless the caller allows the report code.
The same rule applies across adjacent bars within one staff and voice.
An exported grace beat precedes its owner and has no legato origin.

GP8 preserves every beat fade, hairpin, octave, stroke kind and direction, and integral
exact stroke timing from 0 through 2147483647 ticks. `BeatStroke.Duration` is a
note-value-denominator compatibility view, while `ExactDuration` retains the
authored tick value and source absence. Imported exact timing owns export until
`Duration` is edited; clearing exact timing also falls back to `Duration`.
Fractional or larger exact values receive a scoped omission report. GP8 also
reports rasgueado, pick stroke, and slap effects because the writer does not
emit them.

`BeatEffects.Fade` preserves `None`, `FadeIn`, `FadeOut`, and `VolumeSwell`.
Binary GP3 through GP5 presence maps to `FadeIn`. Those formats cannot author
the other variants. For a programmatic score, a nonzero typed fade is
authoritative and a true legacy `BeatEffects.FadeIn` falls back to `FadeIn`.
For an imported beat, editing only one view makes that view authoritative,
including clearing it. If both views are edited incompatibly, the typed value
wins and GP8 export reports the conflict. Reconciliation does not mutate the
public model. Fade values are notation marks only; dynamics, hairpins,
mix-table volume, playback automation, and envelope timing are independent.

`BeatEffects.VibratoStrength` preserves the beat-wide whammy-bar vibrato as
`Slight` or `Wide`, independently from note vibrato. Binary GP3 through GP5
presence maps to `Slight`. For a programmatic score, a nonzero typed strength is
authoritative and a true legacy `BeatEffects.Vibrato` falls back to `Slight`.
For an imported beat, editing only one view makes that view authoritative,
including clearing it. If both views are edited incompatibly, the typed view
wins and GP8 export reports the conflict. Reconciliation does not mutate the
public model.

`BeatEffects.TremoloPicking` is the beat-wide authored authority, matching the
Guitar Pro and AlphaTab models. `TremoloPickingEffect.Duration` is the authored
subdivision: eighth, sixteenth, and thirty-second values map to GPIF `1/2`,
`1/4`, and `1/8`. These target values must be plain: a dot, double dot, or
non-default tuplet changes the authored duration and receives the scoped rate
omission instead of being flattened to its base value. Binary marks 1, 2, and
3 use the same order. The legacy
`NoteEffect.TremoloPicking` field remains a compatibility fallback when the
beat-wide pointer is nil; the final note-local value wins when legacy notes
conflict. Equality includes the complete authored duration, not only its note
value. A non-nil beat-wide value wins over all note-local values, and export
reports that conflict without mutating the score. Clear the beat-wide pointer
before editing through the legacy field.

Guitar Pro GPIF supports only the default notation style and marks 1 through 3.
AlphaTab's general model also admits marks 0, 4, and 5 plus the `BuzzRoll` style,
but its Guitar Pro writer does not serialize those extensions. GP8 export
therefore reports an unsupported subdivision separately from an unsupported
style. A supported rate is still emitted when only the style is omitted. GP3
has no tremolo-picking source record.

GP8 preserves accents, ghost notes, staccato, palm mute, dead notes, let ring,
hammer origin, tapping flags, slide flags, harmonic kind, and trill fret.
`NoteEffect.Accent` is authoritative when it is nonzero and emits the exact
normal, heavy, or tenuto GPIF flag; tenuto uses `0x10`. When `Accent` is
`NoteAccentNone`, `AccentuatedNote` and `HeavyAccentuatedNote` are compatibility
fallbacks. A conflicting typed and legacy accent produces a normalization
report, and the typed value wins. `NoteEffect.VibratoStrength` is likewise
authoritative when nonzero and preserves `Slight` and `Wide` exactly. A true
legacy `Vibrato` value falls back to `Slight` when the typed strength is absent.
Beat-level vibrato uses the separate `BeatEffects.VibratoStrength` contract.

`NoteEffect.Hammer`, `Tapped`, and `LeftHandTapped` retain the GPIF
`HopoOrigin`, `Tapped`, and `LeftHandTapped` properties independently, including
when more than one is set. GP8 export writes each property independently. GPIF
`HopoDestination` remains a lossy derived destination because `Song` has no
authored hammer-destination field.

`NoteEffect.LeftHandFinger` and `RightHandFinger` preserve the independently
authored note fingerings. Use `HasLeftHandFinger` or `HasRightHandFinger` to
distinguish an authored `Thumb` or `Open` value from absence. Nonzero named
fingers remain compatible without these markers. GPIF preserves Thumb, Index,
Middle, Annular, and Little as `P`, `I`, `M`, `A`, and `C`. It has no note-level
spelling for Open, so GP8 export reports only an explicitly authored Open value
as omitted. Export computes this presence rule without modifying the score.

Validation rejects unknown fingering, accent, vibrato-strength, and slide enum
values. Parse diagnostics reject a missing technique payload and classify
unknown slide bits.

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

GP8 preserves a chord name, string pattern, representable first fret, barre
ranges, and finger assignments. `Chord.Strings` is the diagram width and order
authority from highest string to lowest string. `Chord.Fingerings`, when
non-nil, parallels that slice; `FingeringUnknown` means no authored assignment,
while `FingeringOpen` preserves a GPIF `None` position whose string state
distinguishes muted from open. `Barre.Start` and `Barre.End` are one-based
positions in `Chord.Strings`. Direct public edits are authoritative and export
does not modify them. When fingerings are absent, GP8 export assigns a distinct
finger to each representable barre only for the target encoding. Conflicting
range endpoints, repeated finger/fret groups, invalid indices, and unchecked
numeric mappings are rejected. GPIF `Ring` and legacy `Rank` both import as
`FingeringAnnular`; export emits `Ring`. GP4 and GP5 seven-slot storage is
trimmed to the actual chord string count. Interval omissions and legacy chord
descriptions remain separately reported losses. Imported chord occurrences own
separate mutable slices and pointers. Staff-local definitions take precedence
over track definitions with the same identifier.

GP8 emits the canonical `Ring` token for an annular chord finger. Pinned
AlphaTab recognizes the legacy `Rank` spelling but ignores `Ring`. Export
reports each lost annular barre with `gp8.omit.chord-annular-consumer-barre`.
The report applies to explicit fingerings and synthesized third barres.
The writer retains the finger meaning and exact GPIF positions. Strict
preservation refuses the consumer loss unless the caller allows its code.

Grace notes keep their order, exact fret, articulation identity, dead state,
placement, transition, and supported duration. GP8 reports a raw source fret,
an unsupported duration, a conflicting legacy fret, a noncanonical velocity,
or a bend transition. Reused grace definitions produce independent occurrence
data.

Percussion articulations retain separate notation and playback identities even
when two definitions use the same output MIDI value. GP8 also creates builtin
fallback definitions for identity-free main and grace notes. Import and public
validation reject invalid articulation identities and MIDI ranges. Notehead
options change only the requested output notation. The builtin table uses exact
native Guitar Pro definitions for inputs 29-31, 33-34, 39-40, 54, 56, and
58-87, including distinct input and output MIDI values and notation metadata.
Inputs 27, 28, and 32 remain rejected because the pinned native table has no
matching definition; a nearby convenience alias is not treated as equivalent.
An explicit percussion grace `ExactFret` is its input identity, including
values outside the native table. Only a legacy zero sentinel without an exact
value inherits the parent note identity.

GP8 keeps the authored order of tempo and sound automations. It does not sort
same-bar events by position. Parse diagnostics report missing values, broken
sound references, unknown automation types, and unsupported channel-strip
types. They also report linear tempo or sound interpolation because the public
records do not contain an interpolation field.

Score lyrics and track lyrics have separate scopes. Binary score lyrics expose
`Lyrics.TrackIndex` and `LyricLine.StartMeasureIndex` as zero-based references;
`-1` means that the source stored its unassigned zero value. Line order is the
slice order and is not duplicated in a second public number field. GP8
preserves ordered track lyrics and their offsets, but it does not emit binary
score lyrics. Export reports the score-level omission. Beat text remains
separate and is preserved on rests. An undispatched GPIF lyric source produces
a loss diagnostic because the public model does not retain that source state.

An enabled local backing track must refer to an archive entry with audio data.
Its frame padding must fit in a signed 64-bit frame count. Sync points preserve
their authored bar, position, tempo, visibility, frame, and media-time values.
GP8 export embeds enabled `Local` backing tracks, using the public record as the
edit authority for asset metadata, path, bytes, and padding. The GP8 target uses
a signed 32-bit frame-padding field, so export rejects values outside that range.
Asset paths that collide with fixed GP8 archive members are also rejected.
Disabled and non-local backing-track records remain explicit omissions. Sync
points remain a separate export omission.

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
