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
are compatibility views of the first staff. GP8 export honors replacement of
non-nil compatibility slices. `FinalizeSong` reconnects those slices to the
first staff. New multi-staff code must use `Track.Staves` for other staves.

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

Master bars own double-bar output. A non-default `Measure.HasDoubleBar` value
that conflicts with its header produces a normalization report.

This policy supports gradual migration. New code can use `Score`, exact value
types, staves, diagnostics, and preflight reports. Existing `Song` code remains
source compatible.
