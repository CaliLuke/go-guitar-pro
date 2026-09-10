# AlphaTab capability audit

Review date: September 10, 2026.
Initial library review: `0c3890b`. Refreshed after simile support in `339113c`.
Oracle: AlphaTab 1.8.4, revision `022a45c8e42370f9e12e68949d11eada370da83d`.

The library has substantial remaining capability gaps. A complete existing semantic matrix does not establish AlphaTab parity.
That matrix accounts for the library's declared fields and their dispositions, including explicit omissions.
The corpus oracle compares a selected projection of the score.
Neither mechanism detects an upstream feature that is absent from both the public model and the comparison.

The [SQLite database](capabilities.sqlite) and [generated capability list](CAPABILITIES.md) now expose that larger scope.
The [maintenance guide](README.md) explains the queries and update process.

## Inventory scope

The inventory contains 107 capability rows: 101 within the Guitar Pro audit and six excluded service categories.
The excluded categories cover repeat traversal, other importers, other exporters, rendering, synthesis, and AlphaTab-specific graph utilities.
Authored notation and playback data remain within scope, even when the library does not render or play them.

The database maps all 405 feature rows from the four public Guitar Pro format tables.
Those tables contain 157 distinct feature labels.
The source inventory contains 4,365 constructs from 119 pinned source and test files.
Of these, 1,507 constructs belong to model files.
Every discovered public model field has a capability association.
Associations identify where to review a field. They do not prove its behavior.
Other unclassified constructs remain queryable through `unreviewed_constructs`.

The database also imports the existing field, enum, wire, dispatch, test, fixture, and diagnostic inventories.
This preserves the connection between capability planning and executable contracts.

| Stage | Supported | Partial | Missing | Unverified |
| --- | ---: | ---: | ---: | ---: |
| Import | 48 | 27 | 24 | 2 |
| Public model | 48 | 27 | 25 | 1 |
| GP8 export | 32 | 22 | 45 | 2 |

Only 29 rows have a supported rating across all three stages.
These are checklist counts with different feature sizes. They are not a percentage of all musical behavior.
Ratings apply to each row's stated scope, bounds, and format qualifications.

## Highest-priority findings

| Capability | Finding | Reproduction or evidence |
| --- | --- | --- |
| Navigation | GPIF targets and jumps disappear. GP5 allows only one public direction per header. GP8 omits it. | Eight inputs lose directions. See `directions`. |
| Fermatas | Hold type, duration, and position are absent from the public model. | Four inputs lose fermatas. See `fermata`. |
| Free time | The free-time flag disappears. Fixed meter alone cannot preserve it. | `testdata/gp7/free-time.gp`, master bars 1, 2, and 6. |
| Key mode | Valid lowercase `minor` becomes major. Go compares the source with `Minor` exactly. | Generated `synthetic/key-mode-lowercase.gp`. AlphaTab sees minor before export and major afterward. |
| Transposition | Independent sounding and display transposition are absent. Effective keys can also change. | `upstream/guitarpro8/e.gp` has chromatic transposition 4. Its effective key changes from four flats to zero. |
| Clef octave | Bar-level octave shifts disappear, despite support for beat-level octave shifts. | Two inputs lose `Bar.clefOttava`. See `clef-octave`. |
| MIDI bank | Initial bank selection and bank-change events are not retained through export. | Upstream GP8 fixtures lose banks 77 and 256. `bank-change.gp` loses a bank 256 event. |
| Sustain pedal | Pedal down, hold, and release markers have no public destination. | `testdata/gp7/sustain.gp`. |
| Tremolo picking | Import stores the technique, but GP8 omits it. | Thirteen inputs show this loss. See `tremolo`. |
| Audio and automation export | Backing audio, sync points, volume events, and legacy mix changes have explicit export omissions. | Public preflight reports and the existing conformance cases. |
| Ornaments and slurs | Turns, mordents, and authored legato/slur state lack complete destinations. | `testdata/gp7/ornaments.gp` and `testdata/gp7/numbered.gp`. |

These findings need separate, bounded fixes. The database contains completion criteria and exact probe selectors for the confirmed examples.
The audit does not create broad waivers in the existing differential oracle.

## Other gaps

The complete list also covers these areas:

- Separate section letters and text, track short names, tuning labels, and per-staff capo state.
- Fade-out, volume swell, beat vibrato strength, wah state, rasgueado patterns, and brush timing.
- Pick strokes, fingerings, trill duration, chord barres, chord visibility, and pitch spelling.
- Beat lyrics, slashed beats, dead slaps, golpe marks, barre shapes, timers, and explicit string numbers.
- Numbered notation, multiple-bar rest preferences, staff visibility, beaming, system layout, and stylesheet data.
- Legacy RSE data, proprietary effects, partial capos, password protection, and independent display-duration overrides.

Some gaps concern richer source data. Others concern a public value that the exporter omits.
The database records those stages separately.

## Repeat support and recent corrections

The reported repeat-count defect is fixed in this checkout.
`MeasureHeader.RepeatCount` stores total passes: zero means no repeat end, two means two passes, and six means six passes.
The public maximum is 128. Binary offsets remain inside the codecs.
The public API test and measure conformance cases cover the current contract.
The broad probe found no repeat-count differences across 28 inputs with non-default repeat values.

Simile marks are a separate feature. Commit `339113c` now preserves all four variants through `Measure.SimileMark` and GP8 export.
The refreshed probe and independent consumer assertion confirm the resolved slice; the existing [issue 39](https://github.com/CaliLuke/go-guitar-pro/issues/39) is closed and no duplicate implementation ticket is filed.
Expanded repeat and jump traversal remains outside the documented authored-score scope.

Note vibrato strength, tenuto, and separate tapping flags also exist in the current implementation.
The probe found no differences in the exercised cases: 21 vibrato inputs, one tenuto input, and one tapping input.
Those counts do not prove all combinations.

Some paragraphs in `docs/semantic-model.md` still describe their previous lossy representations.
Some diagnostic-source declarations also retain old disposition labels even when the current dispatch no longer emits that loss.
Future work must review executable behavior and current fields alongside the prose.

## Runtime audit

The probe used 364 inputs:

- All 313 local corpus files.
- Fifty pinned upstream files without a byte-identical local counterpart.
- One generated lowercase-key-mode fixture.

There are 219 Guitar Pro fixture files in the pinned upstream directories.
Matching by content avoids treating a renamed local fixture as missing.

Both consumer projections completed for 338 inputs.
The receipt contains 19,656 capability comparisons, including 2,544 comparisons with non-default source values.
It records 1,706 raw differences across 42 capability groups.
These are capability-per-input differences, not counts of independent defects.

The remaining inputs had explicit blockers:

- One upstream file failed Go parsing because it contains a fractional negative string index, `-2.5`.
- Ten parsed files failed Go export. Causes include invalid MIDI programs, unsupported percussion mappings, curve ordering, and a negative sound-event position.
- Fifteen Go-exported target archives failed the pinned AlphaTab inflater with `Invalid huffman`. The original upstream files loaded successfully. Repackaging unchanged member bytes using ZIP Store made all fifteen targets load. See issue #68 and the investigation review.

All local corpus files parsed successfully. Eight of them failed GP8 export.
The database preserves every failure and does not score a blocked comparison as equal.

Many raw differences involve display defaults or consumer derivation.
For example, imported guitar display transposition often defaults to minus twelve semitones.
That contributes to the transposition difference count. It does not establish that every affected file has incorrect sounding pitch.

The pinned GPIF importer also clears the final bar's double-bar flag.
That explains twelve binary-to-GP8 double-bar differences despite an emitted XML flag.
The capability entry records this target-consumer limitation.

The probe compares selected authored fields through import plus export.
It does not replace the existing normalized oracle or prove every capability.
Its exact values and paths make those additional differences available for review.

## Documentation and ownership gaps

The public [format tables](https://alphatab.net/docs/category/formats) provide a useful inventory of user-visible capabilities.
They distinguish model, reading, rendering, audio, and AlphaTex support.
The database preserves those claims separately from this library's stage ratings.

The [GP8 table](https://alphatab.net/docs/formats/guitar-pro-8) says audio files are unsupported.
Pinned source and tests support backing-track data, and this library already imports it.
Therefore website ratings cannot replace pinned source evidence.

The old upstream scanner covers nine primary model files.
It also attributes helper-class fields to the filename's main class.
For example, `BeamingRules.groups` becomes `MasterBar.groups`, and `SyncPointData.barOccurence` becomes `Automation.barOccurence`.
The new inventory keeps the declaring class and includes every discovered model enum member.

Several broad feature-ledger rows still point to [issue 34](https://github.com/CaliLuke/go-guitar-pro/issues/34), which is closed.
That ticket is not an open owner for the remaining gaps.
[Issue 38](https://github.com/CaliLuke/go-guitar-pro/issues/38) remains open, although the current checkout contains its repeat API change.
The initial audit left issue states unchanged. The follow-up [backlog](BACKLOG.md) gives each remaining gap an individual owner ticket.

## Suggested implementation order

1. Fix the lowercase key-mode bug and add the missing upstream fixture cases to focused tests.
2. Add navigation collections, fermatas, free time, and clef octave values.
3. Preserve transposition, bank changes, sustain pedal data, and per-staff instrument context.
4. Close export-only technique and automation gaps.
5. Add the remaining notation and stylesheet fields.
6. Resolve unverified source variants and expand the independent comparisons.

Each step needs valid non-default public API assertions and independent consumer evidence.
The existing semantic workflow remains the acceptance process for implementation changes.
The database makes the remaining scope visible and keeps the evidence connected.

## Agent readiness

The follow-up review defines 77 bounded work items for the 72 incomplete capability rows.
The initial backlog had 46 implementation tasks and 31 investigations. The [enriched investigation review](INVESTIGATION-REVIEW.md) converts 15 of those investigations into bounded implementation work.
Five additional tasks separate export validation failures, an oracle inflater failure, and stale documentation from broader capability work.
Each task has acceptance criteria, source evidence, a reproduction path, and explicit dependencies.
The database exposes ready work, blocked work, issue-state drift, and potential shared-file conflicts.
A completed investigation records a decision; it does not promote a capability rating.
See [the agent workflow](README.md#work-through-the-backlog) before dispatching work.
