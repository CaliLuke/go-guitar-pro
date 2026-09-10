# Enriched investigation review

Reviewed 31 investigation tickets against library commit `f5a5ee962d086c369d2e523ba2de98ca1ddd481b` and pinned AlphaTab `022a45c8e42370f9e12e68949d11eada370da83d`.

Fifteen tickets have bounded implementation scope. Sixteen retain investigation classification: eight target limits, three other-format exclusions, and five evidence gaps.
No capability support rating is promoted by this review. No investigation is closed automatically. The original enrichment comments remain attached to their GitHub tickets.

## Converted tickets

| Ticket | Implementation scope | Review decision |
| --- | --- | --- |
| [#109](https://github.com/CaliLuke/go-guitar-pro/issues/109) | Preserve global extended barlines and bar-number policy | Pinned BinaryStylesheet apply/writeForScore supports both global keys. The current Go writer returns a zero-entry stylesheet, and the ZIP reader does not import these values. |
| [#60](https://github.com/CaliLuke/go-guitar-pro/issues/60) | Preserve exact authored bend offsets | Confirmed 0..12 rounding in gpifBendPosition and binary conversion. Pinned _toBendOffset multiplies by 0.6 without integer truncation, so fractional offsets also matter. |
| [#62](https://github.com/CaliLuke/go-guitar-pro/issues/62) | Preserve GPIF bend control roles with nonmonotonic offsets | The fixture authors destination 35 after middle offsets 50. Generic monotonic validation rejects a standard GPIF role tuple; the precision prerequisite avoids preserving only rounded coordinates. |
| [#63](https://github.com/CaliLuke/go-guitar-pro/issues/63) | Exclude unused legacy MIDI program slots from export rejection | Public-API inspection confirmed every -1 slot in both fixtures is unused, including channels 25/41/57 and 9/25/41/57. ValidateSong currently checks the entire table before target export. |
| [#64](https://github.com/CaliLuke/go-guitar-pro/issues/64) | Preserve authored opening sound-event preroll | The source contains bar 0 Position=-0.125; both parsers accept it and the current GP8 range check rejects it. The implementation must state a bounded preroll contract. |
| [#112](https://github.com/CaliLuke/go-guitar-pro/issues/112) | Preserve authored system layouts and bar display scales | Both system arrays and scale XProperties have explicit pinned GPIF parser assignments and non-default fixtures. Two closely related layout slices can share one bounded public contract. |
| [#98](https://github.com/CaliLuke/go-guitar-pro/issues/98) | Export assigned legacy score lyrics as GPIF track lyrics | The legacy selected-track lines have a GPIF Track.Lyrics destination already supported by both parsers; export currently omits Song.Lyrics unconditionally. |
| [#67](https://github.com/CaliLuke/go-guitar-pro/issues/67) | Require executable evidence for supported capability claims | Current capabilities link source/tests but do not enforce all five closure evidence roles; existing ledger and sensitivity machinery can supply the enforceable contract. |
| [#68](https://github.com/CaliLuke/go-guitar-pro/issues/68) | Emit GP8 archives readable by the pinned AlphaTab inflater | Reproduced on current code: pinned AlphaTab loads all 15 sources, rejects all 15 Go targets, and loads all 15 targets repackaged as Store with identical member bytes. The old source-side diagnosis was incorrect. |
| [#114](https://github.com/CaliLuke/go-guitar-pro/issues/114) | Preserve supported header and footer templates and visibility | Pinned BinaryStylesheet applies and writes header/footer templates and visibility, unlike physical geometry. The shared stylesheet infrastructure is owned by #109. |
| [#69](https://github.com/CaliLuke/go-guitar-pro/issues/69) | Prove sound identity independently of MIDI program | Current automation tests already cover ordered definitions/events and detail fields. The same-program identity case and opening selection authority are the remaining bounded work. |
| [#117](https://github.com/CaliLuke/go-guitar-pro/issues/117) | Preserve supported score bracket, track-name and display policies | Each listed group has explicit pinned BinaryStylesheet mappings and non-default GP8 fixtures. Scope is limited to those groups, with shared codec work supplied by #109. |
| [#70](https://github.com/CaliLuke/go-guitar-pro/issues/70) | Apply authored HideTempo to the opening GP8 tempo event | Current synthesized TempoAutomation lacks Hidden even when Song.HideTempo is true. Explicit TempoAutomation.Hidden already maps to GPIF Visible after #41. |
| [#71](https://github.com/CaliLuke/go-guitar-pro/issues/71) | Classify authored GP constructs with importer-backed ownership | Expanded discover() already records actual declaring classes, with regression tests for BeamingRules and SyncPointData. Importer-backed applicability and complete owner/disposition enforcement remain missing. |
| [#72](https://github.com/CaliLuke/go-guitar-pro/issues/72) | Apply exact authored offset preservation to whammy curves | Whammy uses the same 0..12 rounding and writer scale as bends; its precision work is concrete after the shared representation and role-aware validation changes. |

## Retained investigations

| Ticket | Disposition | Remaining boundary |
| --- | --- | --- |
| [#110](https://github.com/CaliLuke/go-guitar-pro/issues/110) | out-of-scope | Pinned Guitar Pro importers do not author timeSignatureCommon; model assignments come from other formats. |
| [#111](https://github.com/CaliLuke/go-guitar-pro/issues/111) | out-of-scope | overrideDisplayDuration is populated by MusicXML rather than GP3-GP8; no independent GP record is evidenced. |
| [#95](https://github.com/CaliLuke/go-guitar-pro/issues/95) | target-limit | Go emits and reparses DoubleBar; the final-bar difference is pinned-consumer finalization. No writer change is justified. |
| [#61](https://github.com/CaliLuke/go-guitar-pro/issues/61) | target-limit | One GPIF beat Dynamic token cannot represent arbitrary per-note velocities; retain normalization and strict rejection. |
| [#65](https://github.com/CaliLuke/go-guitar-pro/issues/65) | needs-evidence | Ordered grace/chord behavior is covered. The remaining bend-transition mapping is unproven; pinned legacy readGrace explicitly ignores transition 2. A faithful fixture/oracle is still required. |
| [#96](https://github.com/CaliLuke/go-guitar-pro/issues/96) | target-limit | Harmonic kind/fret are supported; legacy harmonic-specific pitch/octave spelling has no retaining target destination. |
| [#97](https://github.com/CaliLuke/go-guitar-pro/issues/97) | needs-evidence | FretCount has a plausible raw field but no retaining pinned consumer/non-default corpus case; twelve-string and independent banjo flags remain limits. |
| [#99](https://github.com/CaliLuke/go-guitar-pro/issues/99) | target-limit | Distinct Writer, Comments, Date, clipboard provenance and arbitrary notice boundaries cannot use other metadata fields without changing their meaning. |
| [#66](https://github.com/CaliLuke/go-guitar-pro/issues/66) | target-limit | GPIF has one port and one exclusive Default/Mute/Solo token; conflicting public values have no additional target field. |
| [#113](https://github.com/CaliLuke/go-guitar-pro/issues/113) | out-of-scope | Pitched visibility/custom noteheads belong to other-format or rendering paths; percussion noteheads are separately modeled. |
| [#115](https://github.com/CaliLuke/go-guitar-pro/issues/115) | needs-evidence | Partial-capo properties exist, but nonzero fixture semantics, string-flag order and a retaining oracle are missing. |
| [#116](https://github.com/CaliLuke/go-guitar-pro/issues/116) | needs-evidence | No known-credential protected fixture or characterized container/oracle exists; there is no bounded implementation contract. |
| [#100](https://github.com/CaliLuke/go-guitar-pro/issues/100) | target-limit | The target cannot distinguish an authored empty timed beat from a rest inside a populated voice. Whole-empty-voice encoding is a different scope. |
| [#101](https://github.com/CaliLuke/go-guitar-pro/issues/101) | needs-evidence | Detailed proprietary RSE semantics have no independent retaining target oracle. Opaque archive preservation would be a different contract. |
| [#102](https://github.com/CaliLuke/go-guitar-pro/issues/102) | target-limit | No GPIF note-duration-percentage property is retained by the pinned parser/writer; preserve explicit export loss. |
| [#103](https://github.com/CaliLuke/go-guitar-pro/issues/103) | target-limit | GPIF carries target fret but pinned parsing fixes duration to sixteenth. Do not repurpose rhythm or tremolo as speed. |

## Evidence checked during review

- Archive interoperability: all 15 original transposition-tonality sources loaded in pinned AlphaTab; all 15 Go exports failed with target-side Invalid huffman. Repackaging identical uncompressed members using ZIP Store made all 15 load successfully. Ordinary ZIP integrity checks passed. This corrects the original source-side diagnosis.
- Invalid programs: public parsing found unused -1 slots 25/41/57 in fade-to-black.gp4 and 9/25/41/57 in upstream beat-harmonics.gp3. None was selected by a track. The conversion excludes unreferenced slots from target rejection and keeps referenced invalid programs rejected.
- Offset precision: gpifBendPosition rounds to integer 0..12. Pinned GpifParser._toBendOffset multiplies by 0.6 without rounding. An integer 0..60 replacement alone cannot preserve every fractional GPIF percentage.
- Source discovery: capabilities/manage.py already keeps declaring-class ownership and tests BeamingRules.groups and SyncPointData.barOccurence. The enriched comment conflates it with the older scanner; the converted task keeps the existing fix.
- Grace transition: pinned Gp3To5Importer.readGrace ignores transition 2. Existing grace chord/order coverage does not prove a faithful bend-transition mapping.
- Styles, layout and visibility: checked the exact pinned GpifParser and BinaryStylesheet apply/writeForScore branches, existing Go omission reports, ZIP member handling and current automation/lyric tests.

## Query the review decisions

Each reviewed work item stores its decision and source-comment link in `evidence_json`:

```sql
SELECT w.id,w.kind,json_extract(e.value,'$.decision') AS decision,
       json_extract(e.value,'$.reference') AS source_comment
FROM work_item w,json_each(w.evidence_json) e
WHERE json_extract(e.value,'$.kind')='investigation-review';
```

## Follow-up directions

Each retained investigation has a comment specifying its next evidence or verification step:

- [#61: next steps](https://github.com/CaliLuke/go-guitar-pro/issues/61#issuecomment-5625487101)
- [#65: next steps](https://github.com/CaliLuke/go-guitar-pro/issues/65#issuecomment-5625487547)
- [#66: next steps](https://github.com/CaliLuke/go-guitar-pro/issues/66#issuecomment-5625488053)
- [#95: next steps](https://github.com/CaliLuke/go-guitar-pro/issues/95#issuecomment-5625488545)
- [#96: next steps](https://github.com/CaliLuke/go-guitar-pro/issues/96#issuecomment-5625489007)
- [#97: next steps](https://github.com/CaliLuke/go-guitar-pro/issues/97#issuecomment-5625489523)
- [#99: next steps](https://github.com/CaliLuke/go-guitar-pro/issues/99#issuecomment-5625490016)
- [#100: next steps](https://github.com/CaliLuke/go-guitar-pro/issues/100#issuecomment-5625490534)
- [#101: next steps](https://github.com/CaliLuke/go-guitar-pro/issues/101#issuecomment-5625491158)
- [#102: next steps](https://github.com/CaliLuke/go-guitar-pro/issues/102#issuecomment-5625491591)
- [#103: next steps](https://github.com/CaliLuke/go-guitar-pro/issues/103#issuecomment-5625492081)
- [#110: next steps](https://github.com/CaliLuke/go-guitar-pro/issues/110#issuecomment-5625492569)
- [#111: next steps](https://github.com/CaliLuke/go-guitar-pro/issues/111#issuecomment-5625493105)
- [#113: next steps](https://github.com/CaliLuke/go-guitar-pro/issues/113#issuecomment-5625493608)
- [#115: next steps](https://github.com/CaliLuke/go-guitar-pro/issues/115#issuecomment-5625494153)
- [#116: next steps](https://github.com/CaliLuke/go-guitar-pro/issues/116#issuecomment-5625494709)
