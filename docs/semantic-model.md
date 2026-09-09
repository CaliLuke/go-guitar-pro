# Semantic score model

`Score` is an alias for `Song`. The alias gives the model a semantic name
without a second writable object tree. Existing callers can continue to use
`Song`.

The model covers Guitar Pro import and GP8 export. It does not cover AlphaTab
rendering, synthesis, or other file importers.

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

`InitialTempo` is the fractional authored value. `Tempo` is its legacy integer
projection. When an opening automation identifies the stale value, an edit to
either representation wins and produces a normalization report. GP8 preserves
the tempo name and ordered automation values. It cannot preserve `HideTempo`,
so it reports that omission.

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

This policy supports gradual migration. New code can use `Score`, exact value
types, staves, diagnostics, and preflight reports. Existing `Song` code remains
source compatible.
