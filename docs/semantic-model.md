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


`Marker.Letter` and `Marker.Text` preserve the independent GPIF section fields.
A nil `MeasureHeader.Marker` means no section. A non-nil empty marker preserves
an authored empty section. Import initializes `Marker.Title` from nonempty text,
then from the letter. While the parsed title is unchanged, letter and text edits
are authoritative. A changed parsed title overrides text and preserves the letter.
Conflicting simultaneous title and text edits receive `gp8.normalize.section-title`.
For a programmatic marker, nonempty letter or text is authoritative. A title-only
marker exports its title as text. GP3 through GP5 supply only this legacy caption.
Reconciliation does not mutate the marker.

`Track.ShortName` is the independent authored abbreviation. Nil means the GPIF
field was absent. A non-nil value preserves exact text, including an empty string.
Direct edits control GP8 output without changing the full name or truncating text.
Each parsed track occurrence owns its pointer. GP3 through GP5 have no authored
short-name field.

Pinned AlphaTab derives an empty short name from the full name during finalization.
When that derived name differs, GP8 reports `gp8.omit.short-name-consumer-empty`
and retains the authored empty text. Raw importer evidence does not establish final
consumer preservation. Section fields and nonempty short names preserve Unicode,
XML-sensitive characters, CDATA terminators, and internal carriage returns.
Carriage returns use decimal character references to prevent XML line-ending normalization.
When a CDATA terminator or carriage return requires ordinary XML text, the pinned consumer trims boundary whitespace.
The corresponding `gp8.omit.section-consumer-whitespace` or
`gp8.omit.short-name-consumer-whitespace` report identifies that narrow loss.
Strict preservation requires an explicit allowance for each applicable report.

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

GP3 through GP5 fret counts must fit the public `uint8` range, 0 through 255.
The readers reject unrepresentable source integers before conversion. This error
identifies the public model limit; it does not define Guitar Pro syntax.
Pinned AlphaTab skips the binary field and writes GPIF fret count 24.
No retained target value or standalone twelve-string/banjo flag supports a broader claim.
The three existing omission codes remain independent, including strict-policy allowances.

GP5 RSE instrument integers must fit the public signed `int16` range, -32768 through 32767.
The four GP5.1 integer fields and the first three GP5.0 fields receive checked conversion.
GP5.0 keeps its signed-short effect number and one padding byte.
Negative identifiers, including -1, remain opaque signed values.
GP8 retains the existing master-RSE, track-RSE, and UseRse omission reports.
Each nonzero leaf independently triggers its parent report; a present zero-valued EQ knob still has authored slice shape.
Zero scalars and absent or empty knob slices do not create an RSE loss.
Pinned AlphaTab skips binary RSE banks and projects GPIF channel strips to ordinary balance and volume.
These limits do not change MIDI, TrackSound, or volume-automation contracts.

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

`Track.Sounds` retains each authored definition independently of its MIDI
program. The GPIF reference key contains the path, name, and role. Distinct
definitions can share one program. `SoundAutomation.Sound` is an index into
that slice. When you reorder definitions, remap the indices to preserve each
selection.

The first definition supplies the base program and bank; the channel supplies
them only when no definitions exist. An explicit event at bar zero and position
zero supplies the opening selection. It does not rewrite the first definition
or channel mirror. Multiple opening events retain their authored order. A
conflicting channel mirror receives `gp8.normalize.sound-authority`.

Pinned AlphaTab exposes the base program and instrument events but does not
retain a public named sound table. Consumer evidence covers event count, order,
position, program, text, and interpolation. Exact wire references and Go
reimport establish the named identities. The existing Hidden consumer
limitation remains explicit.

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
Legacy beat-local changes use the mix-table projection contract below.

GP3–5 import promotes beat-local mix-table tempo, program, volume, and balance
changes into `TempoAutomations`, `Track.SoundAutomations`, `VolumeAutomations`,
and `PanAutomations`. These collections own later edits and clearing. Retained
raw records do not replay cleared events. Program changes retain the selected
MIDI bank and use explicit generated sound definitions. Volume and balance
values use the legacy 0..16 scale and become normalized values divided by 16.
All-tracks volume, balance, and program changes expand to each target track.
Tempo changes are score-wide.

For programmatic raw changes, GP8 projects events in an isolated copy. Positions
come from authored durations, including tuplets, instead of cached beat starts.
Explicit collection events at the same position take precedence. Equal values
are not duplicated; conflicts report `gp8.normalize.mix-table-<controller>-authority`.
Explicit tempo and sound slice order remains intact; raw events follow score
traversal order. Gain and pan events stay chronological within each track, with
stable order at equal positions. The exporter never changes the input score.

GPIF retains event positions, values, and interpolation flags. The pinned
consumer retains tempo timing, ignores channel-strip gain and pan events, and
attaches positive-position sound events to the first beat of their bar.
`gp8.normalize.sound-automation-consumer-position` reports that timing change
for each affected sound event. The complete GP3 regression requests program 73
at tick 960; pinned GP8 playback applies it at tick 0. This remains a consumer
limitation, separate from the exact GPIF and Go position.

Chorus, reverb, phaser, and tremolo each receive a named mix-table omission with
the exact value, duration, and all-tracks flag. Nonzero transition durations for
supported controllers receive separate per-controller transition omissions.
RSE fields and engine switches also have separate reports. GP8 does not infer
proprietary RSE processing or synthesize audio. Empty raw records and orphaned
tempo labels have explicit omissions. These limits keep mix-table export partial.

`Song.PanAutomations` owns authored pan events independently from static channel
balance. `Value` uses 0 for left, 0.5 for center, and 1 for right. GPIF
`DSPParam_11` uses these units. Legacy mix-table balance values use 0 through
16; import divides them by 16 and marks interpolation as linear.

Events must use valid track and bar indexes and finite positions and values
within 0 through 1. Each track's events must be chronological. Equal positions
retain their authored order. GPIF groups events by track; cross-track slice
interleaving is not retained. Clearing or editing the public slice controls
export, without changing initial `MidiChannel.Balance`.

Legacy import expands an all-tracks balance event to each track. Its raw
mix-table record remains available with transition duration and ownership flags.
That record does not override later edits to `PanAutomations`. The remaining
per-controller mix-table reports cover unrepresented metadata and transitions.

GP8 retains pan events in GPIF and Go reimport. Pinned AlphaTab ignores these
channel-strip events. Each event receives `gp8.omit.pan-automation-consumer`;
strict export requires that allowance. Original legacy Balance events provide
a positive consumer control, but GP8 export remains partial.

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

GP8 has one playback-state value and one MIDI port per track. The primary
channel supplies the port; each channel keeps its local number modulo 16.
Differing ports receive `gp8.normalize.effect-channel-port`. Mute takes precedence
when mute and solo are both true, with `gp8.normalize.playback-state`.
Strict preservation rejects each conflict unless its exact code is allowed.
These decisions do not mutate the authored channel numbers or playback flags.

Export also reports a default binding for an unbound track.
String numbers are canonical positions in tuning order. Export reports any
authored number that differs from that position. The ZIP reader restores track
visibility from `LayoutConfiguration` when that record is present.

The existing integer timing fields remain compatibility projections.
`ExactStart` and `ScoreTime` preserve fractional score ticks. The exporter
quantizes values only at a target boundary.

`BendPoint.ExactOffset` preserves a bend or whammy's authored GPIF position as a
percentage from 0 through 100 when the legacy 0-through-12 `Position` cannot
represent it. A programmatic non-nil exact offset is authoritative; nil falls
back to `Position`. On an imported point, the exact offset remains authoritative
while `Position` is unchanged. Editing `Position` makes the legacy view
authoritative, including when both views changed, and export reports that
conflict. Binary bend and whammy offsets are checked on their native 0-through-60
scale before conversion. Bend heights retain their existing whole-semitone
projection. The same authority applies to beat whammy curves.

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

`Note.ShowStringNumber` requests an explicit string-number display on its exact note.
It is independent from string identity, fret, pitch, and tuning. Direct edits control GP8 output.
GPIF requires an `Enable` child for true; its text is not a boolean value.
An absent property or a property without `Enable` means false.
Reused note definitions produce independent display flags on each occurrence.

`Staff.NotationSettings` owns standard, tablature, slash, and numbered-notation requests.
Each parsed staff receives independent mutable settings when PartConfiguration supplies its track group.
Without a group, nil uses the legacy `Track.Settings` standard/tab flags and false for slash/numbered.
The implicit percussion fallback uses standard notation without tablature.
GP5 staff settings initialize these values from the authored binary display flags.
GP3 and GP4 retain their existing legacy defaults.

For parsed settings, an unchanged `Track.Settings.Notation` or `Tablature` leaves that staff field authoritative.
Changing either legacy flag overrides only that field on every staff during export.
A simultaneous conflict uses the changed legacy flag. Slash and numbered requests remain independent.
For programmatic settings, a non-nil staff value controls all four flags.
Clearing a staff pointer restores the compatibility fallback. Export does not mutate either view.

GP6 and ZIP import read the bounded PartConfiguration view/group records and reject truncated lengths.
The first score view maps one group to each corresponding track; missing groups leave their tracks unchanged.
The pinned consumer applies that group's flags to every staff in its track.
Zero source flags select standard notation. Track visibility remains separate in LayoutConfiguration.

GP8 writes the first staff's resolved configuration for its track.
Distinct track configurations survive exact pinned consumption, including slash and numbered notation.
The consumer cannot retain different configurations on two staves within one track.
Each differing later-staff flag receives its own `gp8.omit.staff-*` report and exact staff location.
An all-false first-staff configuration receives `gp8.normalize.track-view` because the consumer enables standard notation.
Explicit percussion tablature receives `gp8.omit.percussion-tablature` because the consumer suppresses it.
Strict preservation requires every applicable allowance. These limits remain open under #86, #92, and #106.

`Beat.Slashed` is the independent authored beat-level slash mark.
GPIF element presence means true, including a literal false text value.
Direct edits preserve note count, notes, tuning, and beat status.
The mark neither enables staff slash notation nor inherits that preference.

GP8 stores track visibility outside GPIF. The writer reports other non-default track and
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

GP8 retains an authored final `DoubleBar` element. Pinned AlphaTab clears that
flag and resolves the terminal bar line to `LightHeavy`. A legacy explicit
`LightLight` line therefore changes after export. This is a consumer loss,
not an equivalent projection. Preflight reports
`gp8.omit.double-bar-consumer-terminal` at the final master bar. Strict export
requires that exact allowance. Nonfinal double bars retain their flag and
`LightLight` consumer line. The writer never deletes the authored final flag.

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

Legacy GP5 duration fractions 0.5 and 0.75 remain distinct from beat rhythm.
GP8 reports each nondefault fraction with `gp8.omit.note-duration-percent`.
The pinned GPIF consumer returns 1.0 without changing rhythm, ties or let-ring.
Strict export requires that exact omission; 1.0 remains a preservation control.
Negative and nonfinite fractions are invalid. Pinned AlphaTab reads legacy
GP5 fraction bytes with the wrong endianness. The existing oracle correction
reinterprets only affected subnormals; those defective readings are not authored
export inputs.

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
reports rasgueado and slap effects because the writer does not emit them.

`BeatEffects.PickStroke` is the authority for an authored pick direction.
GP8 preserves up, down, and absent marks independently from `BeatEffects.Stroke`.
Direct edits control output, and `BeatStrokeDirectionNone` removes the pick mark.
Validation and export reject undefined directions without changing the score.

`BeatEffects.RasgueadoPattern` retains each of the eighteen named GPIF finger patterns.
`HasRasgueado` remains its legacy presence view. On an imported beat, editing
only one view controls export, including clearing. Incompatible edits to both
views favor the named pattern and report `gp8.normalize.rasgueado-authority`.
A true boolean without a named pattern uses the pinned binary consumer's
`Ii` default and reports `gp8.normalize.rasgueado-unspecified`.
GP4 and GP5 source flags do not identify a finger pattern; GP3 has no such flag.
Unknown GPIF patterns receive a diagnostic and never acquire a substitute gesture.
Invalid public pattern values are rejected. Export does not mutate either view.

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

GPIF carries a trill target fret but no trill speed. Pinned AlphaTab sets
that speed to a sixteenth note. Authored sixteenth speed preserves without loss.
Thirty-second and sixty-fourth speeds require `gp8.normalize.trill-duration`.
Allowed export keeps the target fret and makes duration 16 explicit in Go and
the pinned consumer. Beat rhythm and other effects do not substitute for speed.
`NoteEffect.Accent` is authoritative when it is nonzero and emits the exact
normal, heavy, or tenuto GPIF flag; tenuto uses `0x10`. When `Accent` is
`NoteAccentNone`, `AccentuatedNote` and `HeavyAccentuatedNote` are compatibility
fallbacks. A conflicting typed and legacy accent produces a normalization
report, and the typed value wins. `NoteEffect.VibratoStrength` is likewise
authoritative when nonzero and preserves `Slight` and `Wide` exactly. A true
legacy `Vibrato` value falls back to `Slight` when the typed strength is absent.
Beat-level vibrato uses the separate `BeatEffects.VibratoStrength` contract.

`BeatEffects.WahPedal` preserves a beat-local None, Open, or Closed event.
None means no event; it does not reset an earlier pedal event. GPIF writes
Open and Closed on the owning beat. GP5 maps legacy values 0..99 to Open and
100..127 to Closed. Value -1 means no event. Values below -1 remain in
`MixTableChange.Wah.Value` with `Binary.Beat.Wah.Unsupported`; GPIF export
reports `gp8.omit.wah-legacy-state`. GP3 and GP4 have no wah event byte.

Imported edits to either WahPedal or the legacy value control export.
Clearing the legacy pointer removes the event. Incompatible edits to both
favor WahPedal and report `gp8.normalize.wah-authority`. A nonzero programmatic
state takes precedence; otherwise the legacy value supplies the event.
GPIF import leaves the legacy mix-table pointer absent. Export never mutates
these views. Intermediate legacy values preserve the consumer state but report
`gp8.normalize.wah-legacy-value`. GPIF has no legacy wah display field:
`gp8.omit.wah-display` reports the exact boolean for a present event, unsupported
state, or nondefault flag. Static sound settings do not supply pedal events.

`BeatEffects.Tap`, `Slap`, and `Pop` preserve independent beat techniques.
GP3–5 imports map the legacy enum to one state. GPIF imports combine enabled
Slapped/Popped properties and derive Tap from actual note Tapped properties.
The legacy `SlapEffect` projection uses Pop, then Slap, then Tap priority.
On imported beats, edits only to the legacy enum replace all three states;
edits to the independent states control those states, including clearing.
Incompatible edits to both views favor the independent states and report
`gp8.normalize.beat-technique-authority`. Programmatic nonzero states take
precedence; otherwise the legacy enum supplies the states.

GPIF retains slap and pop as independent beat properties. Its pinned consumer
requires a note Tapped property for beat tap. When needed, export adds that
property to the first output note and reports `gp8.normalize.beat-tap-note-flag`.
Export does not change the authored note. Tap on a rest reports
`gp8.omit.beat-tap`; export does not add a note. Authored note Tapped flags remain
independent and retained. If these flags conflict with a cleared beat Tap,
`gp8.normalize.beat-tap-note-authority` reports the consumer's enabled beat Tap.
LeftHandTapped remains independent in every combination.

`NoteEffect.Hammer`, `Tapped`, and `LeftHandTapped` retain the GPIF
`HopoOrigin`, `Tapped`, and `LeftHandTapped` properties independently, including
when more than one is set. GP8 export writes each property independently. `NoteEffect.HammerDestination` retains the independent authored destination
marker. The consumer derives links separately, as described below.

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

Standard bends use three or four wire control points to identify one gesture. A final
hold uses its early destination for redundant middle controls; it does not
create a nonmonotonic tuple from the terminal hold endpoint.
Import converts these controls to the canonical public gesture. It does not
change a custom curve or remove a point that has vibrato. A missing GPIF bend
destination offset means the end of the note. A middle value without middle
offsets means the midpoint. GP8 export reports an authored initial or final
hold when this standard gesture encoding cannot keep the hold boundary.

Four-point note bends and whammy curves also represent named GPIF roles: origin, middle1,
middle2, and destination. A bounded tuple can place the destination before
middle2 when both middle values match, origin precedes both middles, and the
destination does not precede origin. Import and export retain all four roles
and exact offsets without sorting. Other unordered curves remain invalid.
Finite offset bounds, point vibrato, and unsupported shapes retain their
existing checks and loss reports. Whammy consumer interpretation still reports
a removed hold or changed rate. Distinct middle offsets such as 35 and 35.5
remain distinct despite sharing one legacy Position. GP6 named properties and
the GP7/GP8 Whammy element use this same percentage authority.

The GP7 and GP8 canon sources author note177 at offsets 0, 50, 50, 35.
Go preserves those controls. Pinned AlphaTab keeps only the hold endpoints
at offsets 0 and 21 on its 0-through-60 scale, for both source and output.
The complete source still fails export because track2 sound event1 has
bar179 and position2. The isolated bend regression removes only that event;
it does not expand the sound-event domain.

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

`Chord.ShowName`, `ShowDiagram`, and `ShowFingering` are independent display requests.
Each imported chord occurrence owns its pointers. Direct edits control GP8 output.
Nil pointers retain the existing target default true. A GPIF Diagram defaults all three flags to true.
Without a Diagram, the name defaults to true and diagram/fingering display defaults to false.
Explicit true and false properties override each default independently. Invalid property spellings produce a scoped invalid-data diagnostic.
The legacy binary `Chord.Show` field remains a separate reported omission.
Display flags do not discard string or finger data. The existing GP8 writer supplies a muted fallback diagram for an empty string slice.
The independent display receipt records that fallback separately from the retained flags.

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

The GP4 fade-to-black grace record at byte70913 is `16 06 02 02`.
Public Parse retains transition Bend at track6/measure201/voice0/beat4/note0/grace0,
with fret22, raw fret22, duration32, velocity95, and before-beat placement.
Pinned AlphaTab ignores this transition byte. Its raw grace beat4 precedes
owner beat5 on string6 and has no bend, slide, or hammer effect.
Its finalized grace duration8 does not replace the authored duration32.

GP8 reports `gp8.omit.grace-bend-transition` at the owning note. Strict export
refuses output unless that exact omission is allowed. Allowed output keeps
the grace placement and owning string; it does not fabricate a bend curve.
Matching absence in the pinned consumer proves this omission policy only.
Future preservation requires independently verified curve semantics, not a
curve inferred from adjacent frets.

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

Score lyrics and track lyrics have separate public scopes. Binary score lyrics expose
`Lyrics.TrackIndex` and `LyricLine.StartMeasureIndex` as zero-based references;
`-1` means that the source stored its unassigned zero value. Line order is the
slice order. GP8 projects assigned score lyrics onto the selected track when
its explicit `Track.Lyrics` list is empty. Equivalent lists produce one copy.
If the lists conflict, explicit track lyrics win and `gp8.omit.score-lyrics`
reports the omitted score representation. Unassigned score lyrics keep that loss.
Invalid track references are rejected. Export does not modify either list.
Clear explicit track lyrics to restore the score fallback; clear both to remove it.
GP8 import exposes projected lines as `Track.Lyrics`, including empty text and exact offsets.

Beat lyrics and `Beat.Text` remain separate from these ordered track lines.
Pinned AlphaTab suppresses every track lyric dispatch when any beat lyric element exists.
Export reports `gp8.omit.track-lyrics-consumer-dispatch` for each affected track.
Both scopes remain intact in GPIF and Go. Without beat lyrics, the consumer
assigns track syllables to first-staff beats, starting at each line's measure offset.
It skips rests and empty beats, interprets lyric punctuation, and drops unplaced syllables.
It does not expose the raw line table after dispatch. Empty lines retain wire
shape but contribute no syllables. An undispatched GPIF lyric source still produces
a loss diagnostic because the public model does not retain that source state.

Metadata and track lyric text use the shared GPIF text encoder. CDATA preserves
line breaks, tabs, Unicode, XML characters, and boundary whitespace.
Text containing a CDATA terminator or carriage return uses decimal character references.
This retains the full XML and Go value. Pinned AlphaTab trims its boundary whitespace.
`gp8.normalize.metadata-text-consumer` identifies affected metadata fields;
`gp8.normalize.track-lyrics-text-consumer` identifies affected track lyric lines.
Strict preservation refuses these normalizations without the exact allowance.

Notices join with literal line separators. Leading and trailing empty entries remain
representable, but an embedded newline loses the original slice boundary.
That case retains `gp8.normalize.notice-lines`. One empty notice retains
`gp8.omit.empty-notice`. Writer, Comments, Date, and clipboard ranges keep their
independent omissions; they do not replace Music, Tabber, or other supported credits.

An enabled local backing track must refer to an archive entry with audio data.
Its frame padding must fit in a signed 64-bit frame count. Sync points preserve
their authored bar, position, tempo, visibility, frame, and media-time values.
GP8 export embeds enabled `Local` backing tracks, using the public record as the
edit authority for asset metadata, path, bytes, and padding. The GP8 target uses
a signed 32-bit frame-padding field, so export rejects values outside that range.
Asset paths that collide with fixed GP8 archive members are also rejected.
Disabled and non-local backing-track records remain explicit omissions.

GP8 emits `Song.SyncPoints` in authored order. `AudioFrame` and `BarPosition`
are authoritative; conflicting `FrameOffset` and `Position` compatibility views
receive their existing normalization reports. Set both views when editing them
together. `MediaTimeMS` must match `(AudioFrame - FramePadding) / 44100 * 1000`;
validation rejects a stale projection. Export does not mutate these fields.

Frame offsets remain exact decimal integers in GPIF. Pinned AlphaTab converts
them to binary64 media time. Nonrepresentable integers receive
`gp8.normalize.sync-point-consumer-integer`. Exact bar positions that cannot
survive the numeric GPIF representation receive
`gp8.reject.sync-point-position-precision`. Neither path silently narrows authored
values. Occurrence, position, interpolation, and visibility survive independent
consumption for representable values.

GPIF and Go retain `ModifiedTempo` and `OriginalTempo`. Pinned AlphaTab ignores
these metadata fields. Nonzero values receive `gp8.omit.sync-point-consumer-tempo`.
If an omitted backing track has nonzero padding, each point also receives
`gp8.normalize.sync-point-consumer-padding`: its frame survives, but target media
time changes. Strict export requires each applicable allowance. Export remains
partial for these documented consumer limits.

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

Timed `BeatStatusEmpty` occurrences export as rests, with
`gp8.normalize.empty-beat` at the authored beat. Their rhythm and text remain
unchanged. An absent voice has no authored beat and produces no such report.
GP8 records four voice slots with `-1` for absence. Go preserves interior absent
slots and trims trailing absent slots. Pinned AlphaTab creates an empty quarter-note
placeholder in each absent slot. These placeholders are separate from timed rests.

Harmonic kind and fractional fret are independent of legacy harmonic spelling.
All six kinds retain fret `2.4` in GPIF, Go, and the pinned consumer.
Authored harmonic pitch or octave produces one `gp8.omit.harmonic-pitch` report.
A conflicting legacy fret produces a separate authority report. Strict export
requires each applicable code; allowing spelling loss does not allow fret conflict.

`Note.Ornament` is the authority for each note’s turn or mordent.
GPIF stores Turn, InvertedTurn, UpperMordent, or LowerMordent; an absent element means None.
Each parsed occurrence owns its value, including occurrences that share a source definition.
Direct edits preserve string, fret, and adjacent notes. Unknown source strings produce an unsupported-feature diagnostic.
Export rejects undefined enum values before writing the score.

`Beat.Timer` owns one beat's timer mark. Nil means no timer.
A present `BeatTimer` with nil `Milliseconds` requests a derived playback timer.
A non-nil millisecond value is explicit, including zero. Direct edits control export.
Each parsed beat occurrence owns independent timer pointers, even when definitions are reused.
Finalization does not calculate playback time. GP8 emits -1 for a derived request.
The pinned importer retains that request as `showTimer=true` and `timer=null`.
MIDI generation can later replace a consumer timer with its calculated playback time.

Explicit milliseconds must be integers from zero through 9007199254740991.
This safe-integer boundary is the library's exact consumer contract, not a universal GPIF limit.
Empty and -1 source values request derivation. Other negative or malformed values produce `GPIF.Beat.Timer.InvalidValue`.
Permissive import retains a derived request for invalid text; strict import rejects it.
Invalid authored millisecond values produce `score.beat.timer` and hard export rejection.

GPIF grace beats become `GraceEffect` records, which have no beat-timer destination.
A timer on such a source beat produces `GPIF.Beat.Timer.GraceUnsupported`.
Strict source import rejects this loss; permissive import retains the grace note without its timer.
The raw grace source and exported artifact record that remaining limit under #108.

`Note.AccidentalMode` owns an authored accidental choice independently of numeric
pitch. Its zero value, `NoteAccidentalDefault`, leaves spelling to the consumer.
The other modes request natural, sharp, double sharp, flat, or double flat.
For a new or edited pitched string note, GP8 derives the compatible step and
octave from its written numeric pitch. It emits one `TransposedPitch` property. An
explicit natural emits an empty `Accidental` child. Default emits no pitch
property.

GPIF octave zero starts at MIDI zero. Contradictory modes and undefined
enum values fail validation without changing pitch or mutating the song.

On GPIF import, `TransposedPitch` wins over `ConcertPitch` regardless of property
order. An absent accidental child means default. An empty child means natural.
A distinct concert accidental produces `GPIF.Note.Pitch.Authority`. This reports
the concert spelling that the public mode cannot edit independently.

Numeric fret, tuning, capo, display transposition,
beat octave, and clef octave remain the pitch context. The `AccidentalMode` field
controls spelling. A default value restores automatic spelling.

After a numeric pitch edit, update or clear an incompatible accidental mode.

Neither edit rewrites the other authority. Invalid source step, token, or octave
produces `GPIF.Note.Pitch.Invalid`. Numeric source checks account for the original
notation coordinate system described below.

The legacy `Note.SwapAccidentals` field remains a separate contextual preference.
It does not overwrite `AccidentalMode` after parsing or editing. GP8 still reports
`gp8.omit.swap-accidentals`. The writer has no faithful target mapping.
The API does not expose the consumer's forced-hidden accidental mode because
these GPIF pitch properties cannot encode it distinctly.

GP8 reports `gp8.omit.note-accidental-context` for an authored mode on percussion
or an absolute note without string context. The same report covers natural
harmonics and staves with omitted sounding transposition. Strict export rejects that omission unless
the export policy explicitly allows it. These limits do not change the existing
numeric pitch contract. In particular, the pinned consumer does not read absolute pitch from
the writer's `Midi` property alone. Harmonic pitch and octave metadata remain
separate fields with their existing omission reports.

GPIF import reports `GPIF.Note.Pitch.Context` when context prevents supported
spelling interpretation. A negative source fret falls back to the existing
numeric representation and clears the accidental mode. A matched grace becomes
an ordered `GraceEffect`, which has no independent accidental field. Its source
note ID identifies that loss. An orphan grace remains a `Note` and retains its
mode. Generated grace notes do not inherit their owner's accidental choice.

An unchanged imported note can retain a GPIF spelling in a different notation
coordinate system. For example, `serenade.gp` uses nominal-tuning transposed
spelling, and `ottavia.gp` excludes octave marks from that source pitch. The
numeric concert record establishes sounding pitch. An immutable private receipt
retains the selected source pitch and its concert context where required.
This receipt adds no second public pitch authority.

GP8 emits these original pitch records while the mode, fret, string, tuning,
capo, transpositions, and octave marks remain unchanged. A change to any of
those values invalidates the receipt. Export then derives spelling from the
edited numeric context and validates the requested accidental. Clearing the
mode removes the preference. Reimport retains the original context across
successive exports. Derived spellings must use GPIF octaves -1 through 11.
Validation rejects an out-of-range written octave before serialization.
Malformed pitch syntax, contradictory concert pitch, and
unsubstantiated transposed-only pitch contradictions remain invalid source data.

`Song.SystemLayout` and `Track.SystemLayout` retain independent authored system counts.
A non-nil track layout controls that track. Export copies score counts to an unspecified track and reports `gp8.normalize.track-layout-inheritance`.
The pinned consumer uses track counts for one displayed track and score counts for multiple displayed tracks. It does not inherit missing track counts.
Export never overwrites the authored arrays. Reimport exposes inherited counts as explicit track values.
Within a layout, a zero default count means the source element is absent. A nil array is absent; an empty array remains explicit.
If both fields are empty, export emits an empty array to retain the explicit scope and reports `gp8.normalize.empty-layout-array`.
Reimport then exposes an empty array instead of nil. An explicit empty track scope continues to override score counts.
Positive counts must fit a signed 32-bit integer. The consumer default for an absent count is three.
`MeasureHeader.DisplayScale` and `Measure.DisplayScale` retain independent positive finite scales. Nil means absent and resolves to one.
GPIF uses master XProperty 1124073984 and bar XProperty 1124139520. Both export as Double; bar import also accepts Float, with Double taking precedence.
Each parsed bar occurrence owns its scale. Legacy line-break and display-width policies remain separate.

Six original GP6 fixtures contain the unreachable final remainder [4, -3] for a one-bar score.
A negative final entry is retained only when the sum of all counts equals the score length and the preceding counts exceed it.
All reachable system counts remain positive. Other zero or negative counts are rejected.

`Song.Style` owns the optional global `ExtendedBarLines` and `BarNumbers`
settings from GP6 through GP8 BinaryStylesheet records. Their values apply to the
whole score, independently of measure barline fields. Direct public edits control
export. A nil leaf removes that key and lets the consumer use its default:
false for extended barlines and AllBars for numbering. A nil `Song.Style` exports
an empty stylesheet; reimport creates a style container. PartConfiguration may
then populate independent view preferences.

The checked binary codec retains other validated records, including their keys,
types, payload bytes and order. This transport does not establish public semantic
support for those settings. Editing either owned field preserves all other
records. Setting `Song.Style` to nil discards the imported stylesheet. Duplicate
owned keys use the last source value and export as one record. The codec checks
all eight value types, UTF-8 strings, lengths, boolean bytes and supported policy
values. It rejects a stylesheet larger than one MiB. Authored undefined numbering
policies are validation errors and cannot be allowed as export losses.

`ScoreStyle.MultiRest` applies to the multi-track view. `Track.MultiRest` applies
when that track is viewed alone. PartConfiguration stores these preferences in
separate ordered score views. Direct edits control their corresponding view.
There is no inheritance between the global and single-track settings. Parsed
values use independent pointers. Missing views leave the corresponding field nil.
Nil requests default to false on export; the generated views contain explicit
false bytes. The pinned consumer retains the global boolean and a set of enabled
single-track indices. It leaves that set null when no track is enabled. This
absence/default normalization does not discard an authored true or false value.
Changing these requests does not remove measures or change their order.
`NoteEffect.Hammer` owns the authored hammer/pull origin marker.
`NoteEffect.HammerDestination` owns the separate GPIF destination marker.
Each boolean controls its own GP8 property. Changing or clearing one does not
change the other. Finalization preserves both values and does not invent an
endpoint or a chain. GP3 through GP5 provide the legacy origin flag only.

A GPIF `HopoDestination` with an `Enable` child sets the destination marker.
The child text does not control presence. A missing child produces an invalid
data diagnostic. Broken note references retain their existing diagnostics.
Repeated source definitions produce independent note occurrences.

The pinned consumer ignores the authored destination property. It derives links
from an origin to a later note on the same string. It also searches nearby
strings for left-hand tapping. An intervening ordinary note blocks that search
direction. In public Go coordinates, the search prefers the same string, then
higher string numbers, then lower string numbers. These derived links do not overwrite the public markers.

On fresh GPIF import, the consumer searches the remaining beats in the current
voice and the first beat of the next bar. Later beat links do not yet exist
during note finalization. The source method's nominal three-bar bound does not
extend this effective search range. Generated grace beats participate in the
same search, so a grace transition can supply an incoming link.

GP8 retains each authored property in the wire. If the consumer clears an
unlinked origin, export reports `gp8.omit.hammer-origin-consumer`. If no derived
incoming link retains a destination, export reports
`gp8.omit.hammer-destination-consumer`. Strict preservation requires a separate
allowance for each applicable code. An origin can create a consumer link without
an authored destination marker. This derived state does not add a public marker.

A matched source grace becomes an ordered `GraceEffect`, which has no independent
destination marker. `GPIF.Note.HammerDestination.Grace` reports that loss with the
source note ID. An orphan remains a `Note` and retains its authored marker.

`ScoreStyle.HeaderFooter` owns optional template and visibility fields for Title,
Subtitle, Artist, Album, Words, Music, WordsAndMusic, Tabber and two Copyright
entries. Nil leaves omit their record and use the consumer default. An empty
string is an explicit empty template. An element with both leaves nil becomes
absent after export. Templates remain literal, including whitespace, Unicode,
CR and LF. Valid UTF-8 templates can contain at most 32767 bytes.

GP5 imports its two copyright strings into separate entries before joining them
for `PageSetup.Copyright`. GP6 through GP8 import the supported BinaryStylesheet
records and populate the legacy PageSetup view. The legacy copyright visibility
bit reflects the first footer entry; the second entry retains its independent
visibility in HeaderFooter. Geometry and page-number values have no retaining
consumer mapping and still produce `gp8.omit.page-setup`.

After parsing, each changed PageSetup template overrides only its matching rich
template. A changed visibility bit overrides only the matching visibility field;
the copyright bit applies to both footer entries. Unchanged compatibility fields
leave rich edits, including absence, authoritative. For a new programmatic score,
a non-nil HeaderFooter container owns these values; PageSetup supplies legacy
values when that container is absent. Reconciliation never mutates either view.

An edited legacy copyright string splits at its first LF. Remaining text stays
in the second footer. Multiple separators produce
`gp8.normalize.page-copyright-lines`, which strict export rejects unless allowed.
An unchanged parsed GP5 source retains the original two string boundaries even
when either string contains LF.

No alignment API is introduced. Imported GP8 alignment records retain their exact
bytes and final consumer values, including non-default alignments. Projected
legacy templates omit alignment overrides and use the pinned defaults: centered
Title, Subtitle, Artist, Album and Copyright; left Words; right Music,
WordsAndMusic and Tabber. Clearing template and visibility leaves an imported
alignment record intact, so the consumer can still create its default element.
