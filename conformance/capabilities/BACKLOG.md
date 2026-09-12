# Capability work backlog

Generated from `work-items/*.json`. Regenerate with `manage.py refresh`.

78 work items: 76 implementation tasks and 2 investigations.
Investigation closure records a decision; it does not certify capability support.

Use `manage.py ready` for work without unfinished dependencies or missing external inputs. Check shared files before dispatching concurrent work.

GitHub tracking issue: [#40](https://github.com/CaliLuke/go-guitar-pro/issues/40).

| Work item | Kind | P | State | Prerequisites | Ticket |
| --- | --- | ---: | --- | --- | --- |
| [Preserve authored tempo and sound automation details](work-items/automation-detail.json) | implementation | 1 | done | — | [#41](https://github.com/CaliLuke/go-guitar-pro/issues/41) |
| [Export embedded audio backing-track assets](work-items/backing-track.json) | implementation | 1 | done | — | [#42](https://github.com/CaliLuke/go-guitar-pro/issues/42) |
| [Preserve exact authored bend offsets](work-items/bends.json) | implementation | 1 | done | — | [#60](https://github.com/CaliLuke/go-guitar-pro/issues/60) |
| [Preserve independent capo values on multiple staves](work-items/capo.json) | implementation | 1 | done | — | [#43](https://github.com/CaliLuke/go-guitar-pro/issues/43) |
| [Preserve bar-level clef octave shifts](work-items/clef-octave.json) | implementation | 1 | done | — | [#44](https://github.com/CaliLuke/go-guitar-pro/issues/44) |
| [Preserve navigation targets and jumps on each measure](work-items/directions.json) | implementation | 1 | done | — | [#45](https://github.com/CaliLuke/go-guitar-pro/issues/45) |
| [Correct stale semantic documentation and diagnostic labels](work-items/documentation-contract-drift.json) | implementation | 1 | done | — | [#46](https://github.com/CaliLuke/go-guitar-pro/issues/46) |
| [Lock down chord-velocity and beat-dynamic export loss policy](work-items/dynamics.json) | implementation | 1 | done | — | [#61](https://github.com/CaliLuke/go-guitar-pro/issues/61) |
| [Preserve GPIF bend control roles with nonmonotonic offsets](work-items/export-curve-order.json) | implementation | 1 | done | [#60](https://github.com/CaliLuke/go-guitar-pro/issues/60) | [#62](https://github.com/CaliLuke/go-guitar-pro/issues/62) |
| [Exclude unused legacy MIDI program slots from export rejection](work-items/export-invalid-program.json) | implementation | 1 | done | — | [#63](https://github.com/CaliLuke/go-guitar-pro/issues/63) |
| [Preserve authored opening sound-event preroll](work-items/export-negative-sound-position.json) | implementation | 1 | done | [#41](https://github.com/CaliLuke/go-guitar-pro/issues/41) | [#64](https://github.com/CaliLuke/go-guitar-pro/issues/64) |
| [Preserve fermata position, type and length](work-items/fermata.json) | implementation | 1 | done | — | [#47](https://github.com/CaliLuke/go-guitar-pro/issues/47) |
| [Preserve free-time bars without inventing a meter](work-items/free-time.json) | implementation | 1 | done | — | [#48](https://github.com/CaliLuke/go-guitar-pro/issues/48) |
| [Verify legacy grace-bend preservation and explicit export omission](work-items/grace.json) | implementation | 1 | done | — | [#65](https://github.com/CaliLuke/go-guitar-pro/issues/65) |
| [Preserve lowercase GPIF minor key modes](work-items/key.json) | implementation | 1 | done | — | [#49](https://github.com/CaliLuke/go-guitar-pro/issues/49) |
| [Preserve authored legato and slur endpoints](work-items/legato-slurs.json) | implementation | 1 | done | — | [#50](https://github.com/CaliLuke/go-guitar-pro/issues/50) |
| [Preserve MIDI bank selection and bank changes](work-items/midi-bank.json) | implementation | 1 | done | — | [#51](https://github.com/CaliLuke/go-guitar-pro/issues/51) |
| [Add independent regression coverage for conflicting MIDI routes and playback states](work-items/midi-routing.json) | implementation | 1 | done | — | [#66](https://github.com/CaliLuke/go-guitar-pro/issues/66) |
| [Preserve pitched notes when GP8 exports reopen in Guitar Pro](work-items/native-pitched-notes.json) | implementation | 1 | done | — | [#118](https://github.com/CaliLuke/go-guitar-pro/issues/118) |
| [Require executable evidence for supported capability claims](work-items/oracle-breadth.json) | implementation | 1 | done | — | [#67](https://github.com/CaliLuke/go-guitar-pro/issues/67) |
| [Emit GP8 archives readable by the pinned AlphaTab inflater](work-items/oracle-inflater.json) | implementation | 1 | done | — | [#68](https://github.com/CaliLuke/go-guitar-pro/issues/68) |
| [Preserve pan automation independently from static balance](work-items/pan-automation.json) | implementation | 1 | done | [#41](https://github.com/CaliLuke/go-guitar-pro/issues/41) | [#52](https://github.com/CaliLuke/go-guitar-pro/issues/52) |
| [Export valid percussion notes currently rejected by GP8](work-items/percussion.json) | implementation | 1 | done | — | [#53](https://github.com/CaliLuke/go-guitar-pro/issues/53) |
| [Prove sound identity independently of MIDI program](work-items/sound-automation.json) | implementation | 1 | done | [#51](https://github.com/CaliLuke/go-guitar-pro/issues/51), [#41](https://github.com/CaliLuke/go-guitar-pro/issues/41) | [#69](https://github.com/CaliLuke/go-guitar-pro/issues/69) |
| [Preserve sustain pedal down, hold and release markers](work-items/sustain-pedal.json) | implementation | 1 | done | — | [#54](https://github.com/CaliLuke/go-guitar-pro/issues/54) |
| [Export authored backing-track synchronization points](work-items/sync-points.json) | implementation | 1 | done | [#42](https://github.com/CaliLuke/go-guitar-pro/issues/42) | [#55](https://github.com/CaliLuke/go-guitar-pro/issues/55) |
| [Apply authored HideTempo to the opening GP8 tempo event](work-items/tempo.json) | implementation | 1 | done | [#41](https://github.com/CaliLuke/go-guitar-pro/issues/41) | [#70](https://github.com/CaliLuke/go-guitar-pro/issues/70) |
| [Preserve sounding and display transposition separately](work-items/transposition.json) | implementation | 1 | done | — | [#56](https://github.com/CaliLuke/go-guitar-pro/issues/56) |
| [Export tremolo picking and preserve supported stroke variants](work-items/tremolo.json) | implementation | 1 | done | — | [#57](https://github.com/CaliLuke/go-guitar-pro/issues/57) |
| [Preserve tuning labels independently from pitches](work-items/tuning.json) | implementation | 1 | done | — | [#58](https://github.com/CaliLuke/go-guitar-pro/issues/58) |
| [Classify authored GP constructs with importer-backed ownership](work-items/upstream-discovery.json) | implementation | 1 | done | — | [#71](https://github.com/CaliLuke/go-guitar-pro/issues/71) |
| [Export authored channel-strip volume automation](work-items/volume-automation.json) | implementation | 1 | done | — | [#59](https://github.com/CaliLuke/go-guitar-pro/issues/59) |
| [Apply exact authored offset preservation to whammy curves](work-items/whammy.json) | implementation | 1 | done | [#60](https://github.com/CaliLuke/go-guitar-pro/issues/60), [#62](https://github.com/CaliLuke/go-guitar-pro/issues/62) | [#72](https://github.com/CaliLuke/go-guitar-pro/issues/72) |
| [Preserve authored pitch spelling and accidental choices](work-items/accidentals.json) | implementation | 2 | done | [#56](https://github.com/CaliLuke/go-guitar-pro/issues/56) | [#73](https://github.com/CaliLuke/go-guitar-pro/issues/73) |
| [Preserve beat barre fret and shape](work-items/barre.json) | implementation | 2 | done | — | [#74](https://github.com/CaliLuke/go-guitar-pro/issues/74) |
| [Preserve authored beam groups and stem overrides](work-items/beaming.json) | implementation | 2 | done | — | [#75](https://github.com/CaliLuke/go-guitar-pro/issues/75) |
| [Preserve lyrics authored directly on beats](work-items/beat-lyrics.json) | implementation | 2 | done | — | [#76](https://github.com/CaliLuke/go-guitar-pro/issues/76) |
| [Preserve and export beat vibrato strength](work-items/beat-vibrato.json) | implementation | 2 | done | — | [#77](https://github.com/CaliLuke/go-guitar-pro/issues/77) |
| [Preserve authored brush and arpeggio timing](work-items/brush.json) | implementation | 2 | done | — | [#78](https://github.com/CaliLuke/go-guitar-pro/issues/78) |
| [Preserve representable chord diagram barres and fingerings](work-items/chord-diagram.json) | implementation | 2 | done | — | [#79](https://github.com/CaliLuke/go-guitar-pro/issues/79) |
| [Preserve dead-slapped beats without reducing them to rests](work-items/dead-slap.json) | implementation | 2 | done | — | [#80](https://github.com/CaliLuke/go-guitar-pro/issues/80) |
| [Report terminal double-bar consumer loss without stripping authored XML](work-items/double-bar.json) | implementation | 2 | done | — | [#95](https://github.com/CaliLuke/go-guitar-pro/issues/95) |
| [Preserve fade-out and volume-swell beat effects](work-items/fade-other.json) | implementation | 2 | done | — | [#81](https://github.com/CaliLuke/go-guitar-pro/issues/81) |
| [Export existing left and right hand fingerings](work-items/fingering.json) | implementation | 2 | done | — | [#82](https://github.com/CaliLuke/go-guitar-pro/issues/82) |
| [Preserve thumb and finger golpe marks](work-items/golpe.json) | implementation | 2 | done | — | [#83](https://github.com/CaliLuke/go-guitar-pro/issues/83) |
| [Preserve explicit hammer and pull-off destination markers](work-items/hammer.json) | implementation | 2 | done | — | [#84](https://github.com/CaliLuke/go-guitar-pro/issues/84) |
| [Add independent harmonic-kind coverage and selective spelling-loss policy](work-items/harmonics.json) | implementation | 2 | done | — | [#96](https://github.com/CaliLuke/go-guitar-pro/issues/96) |
| [Reject wrapped legacy fret counts and verify isolated instrument losses](work-items/legacy-instrument.json) | implementation | 2 | done | — | [#97](https://github.com/CaliLuke/go-guitar-pro/issues/97) |
| [Export assigned legacy score lyrics as GPIF track lyrics](work-items/lyrics.json) | implementation | 2 | done | — | [#98](https://github.com/CaliLuke/go-guitar-pro/issues/98) |
| [Preserve metadata text in AlphaTab and verify legacy metadata losses](work-items/metadata.json) | implementation | 2 | done | — | [#99](https://github.com/CaliLuke/go-guitar-pro/issues/99) |
| [Project supported legacy mix-table events into GP8 automation](work-items/mix-table.json) | implementation | 2 | done | [#59](https://github.com/CaliLuke/go-guitar-pro/issues/59), [#52](https://github.com/CaliLuke/go-guitar-pro/issues/52), [#51](https://github.com/CaliLuke/go-guitar-pro/issues/51), [#41](https://github.com/CaliLuke/go-guitar-pro/issues/41), [#118](https://github.com/CaliLuke/go-guitar-pro/issues/118) | [#85](https://github.com/CaliLuke/go-guitar-pro/issues/85) |
| [Read and preserve per-staff part notation settings](work-items/notation-visibility.json) | implementation | 2 | done | [#118](https://github.com/CaliLuke/go-guitar-pro/issues/118) | [#86](https://github.com/CaliLuke/go-guitar-pro/issues/86) |
| [Preserve turns and mordents through GPIF and GP8](work-items/ornaments.json) | implementation | 2 | done | — | [#87](https://github.com/CaliLuke/go-guitar-pro/issues/87) |
| [Export existing pick stroke directions](work-items/pick-stroke.json) | implementation | 2 | done | — | [#88](https://github.com/CaliLuke/go-guitar-pro/issues/88) |
| [Preserve named rasgueado finger patterns](work-items/rasgueado.json) | implementation | 2 | done | — | [#89](https://github.com/CaliLuke/go-guitar-pro/issues/89) |
| [Add independent GP8 regressions for empty beats and absent voice slots](work-items/rests.json) | implementation | 2 | done | — | [#100](https://github.com/CaliLuke/go-guitar-pro/issues/100) |
| [Reject wrapped RSE identifiers and verify each legacy omission](work-items/rse.json) | implementation | 2 | done | — | [#101](https://github.com/CaliLuke/go-guitar-pro/issues/101) |
| [Keep section letters separate from section text](work-items/sections.json) | implementation | 2 | done | — | [#90](https://github.com/CaliLuke/go-guitar-pro/issues/90) |
| [Preserve authored track short names](work-items/short-name.json) | implementation | 2 | done | — | [#91](https://github.com/CaliLuke/go-guitar-pro/issues/91) |
| [Preserve slashed beats and slash staff notation separately](work-items/slash.json) | implementation | 2 | done | [#86](https://github.com/CaliLuke/go-guitar-pro/issues/86), [#118](https://github.com/CaliLuke/go-guitar-pro/issues/118) | [#92](https://github.com/CaliLuke/go-guitar-pro/issues/92) |
| [Verify duration-percentage export losses against the pinned consumer](work-items/sound-duration.json) | implementation | 2 | done | — | [#102](https://github.com/CaliLuke/go-guitar-pro/issues/102) |
| [Preserve independent tap, slap and pop beat techniques](work-items/tap-slap-pop.json) | implementation | 2 | done | — | [#93](https://github.com/CaliLuke/go-guitar-pro/issues/93) |
| [Verify fixed GP8 trill speed and selective normalization policy](work-items/trill.json) | implementation | 2 | done | — | [#103](https://github.com/CaliLuke/go-guitar-pro/issues/103) |
| [Preserve wah pedal state through GPIF and GP8](work-items/wah.json) | implementation | 2 | done | — | [#94](https://github.com/CaliLuke/go-guitar-pro/issues/94) |
| [Preserve global extended barlines and bar-number policy](work-items/barlines.json) | implementation | 3 | done | — | [#109](https://github.com/CaliLuke/go-guitar-pro/issues/109) |
| [Preserve independent chord display flags](work-items/chord-display.json) | implementation | 3 | done | — | [#104](https://github.com/CaliLuke/go-guitar-pro/issues/104) |
| [Exclude other-format common-time glyphs from the Guitar Pro gap inventory](work-items/common-time.json) | implementation | 3 | done | — | [#110](https://github.com/CaliLuke/go-guitar-pro/issues/110) |
| [Exclude MusicXML display-duration placeholders from Guitar Pro gaps](work-items/display-duration-override.json) | implementation | 3 | done | — | [#111](https://github.com/CaliLuke/go-guitar-pro/issues/111) |
| [Preserve authored system layouts and bar display scales](work-items/layout.json) | implementation | 3 | done | — | [#112](https://github.com/CaliLuke/go-guitar-pro/issues/112) |
| [Preserve score and track multirest preferences](work-items/multi-rest.json) | implementation | 3 | done | — | [#105](https://github.com/CaliLuke/go-guitar-pro/issues/105) |
| [Correct pitched note-display scope and protect percussion ownership](work-items/note-display.json) | implementation | 3 | done | — | [#113](https://github.com/CaliLuke/go-guitar-pro/issues/113) |
| [Preserve numbered notation per staff](work-items/numbered.json) | implementation | 3 | done | [#86](https://github.com/CaliLuke/go-guitar-pro/issues/86), [#118](https://github.com/CaliLuke/go-guitar-pro/issues/118) | [#106](https://github.com/CaliLuke/go-guitar-pro/issues/106) |
| [Preserve supported header and footer templates and visibility](work-items/page-setup.json) | implementation | 3 | done | [#109](https://github.com/CaliLuke/go-guitar-pro/issues/109) | [#114](https://github.com/CaliLuke/go-guitar-pro/issues/114) |
| [Preserve explicit string-number display requests](work-items/show-string.json) | implementation | 3 | done | — | [#107](https://github.com/CaliLuke/go-guitar-pro/issues/107) |
| [Preserve supported score bracket, track-name and display policies](work-items/stylesheet.json) | implementation | 3 | done | [#109](https://github.com/CaliLuke/go-guitar-pro/issues/109) | [#117](https://github.com/CaliLuke/go-guitar-pro/issues/117) |
| [Preserve authored timer marks and visibility](work-items/timer.json) | implementation | 3 | done | — | [#108](https://github.com/CaliLuke/go-guitar-pro/issues/108) |
| [Acquire partial-capo fixtures with verified string ordering](work-items/partial-capo.json) | investigation | 3 | needs-input | [#43](https://github.com/CaliLuke/go-guitar-pro/issues/43) | [#115](https://github.com/CaliLuke/go-guitar-pro/issues/115) |
| [Acquire owned GP8 protection-mode fixtures and error receipts](work-items/password.json) | investigation | 3 | done | — | [#116](https://github.com/CaliLuke/go-guitar-pro/issues/116) |
