# Capability work backlog

Generated from `work-items/*.json`. Regenerate with `manage.py refresh`.

77 work items: 61 implementation tasks and 16 investigations.
Investigation closure records a decision; it does not certify capability support.

Use `manage.py ready` for work without unfinished dependencies. Check shared files before dispatching concurrent work.

GitHub tracking issue: [#40](https://github.com/CaliLuke/go-guitar-pro/issues/40).

| Work item | Kind | P | State | Prerequisites | Ticket |
| --- | --- | ---: | --- | --- | --- |
| [Preserve authored tempo and sound automation details](work-items/automation-detail.json) | implementation | 1 | in_progress | — | [#41](https://github.com/CaliLuke/go-guitar-pro/issues/41) |
| [Export embedded audio backing-track assets](work-items/backing-track.json) | implementation | 1 | in_progress | — | [#42](https://github.com/CaliLuke/go-guitar-pro/issues/42) |
| [Preserve exact authored bend offsets](work-items/bends.json) | implementation | 1 | todo | — | [#60](https://github.com/CaliLuke/go-guitar-pro/issues/60) |
| [Preserve independent capo values on multiple staves](work-items/capo.json) | implementation | 1 | in_progress | — | [#43](https://github.com/CaliLuke/go-guitar-pro/issues/43) |
| [Preserve bar-level clef octave shifts](work-items/clef-octave.json) | implementation | 1 | todo | — | [#44](https://github.com/CaliLuke/go-guitar-pro/issues/44) |
| [Preserve navigation targets and jumps on each measure](work-items/directions.json) | implementation | 1 | todo | — | [#45](https://github.com/CaliLuke/go-guitar-pro/issues/45) |
| [Correct stale semantic documentation and diagnostic labels](work-items/documentation-contract-drift.json) | implementation | 1 | todo | — | [#46](https://github.com/CaliLuke/go-guitar-pro/issues/46) |
| [Preserve GPIF bend control roles with nonmonotonic offsets](work-items/export-curve-order.json) | implementation | 1 | blocked | [#60](https://github.com/CaliLuke/go-guitar-pro/issues/60) | [#62](https://github.com/CaliLuke/go-guitar-pro/issues/62) |
| [Exclude unused legacy MIDI program slots from export rejection](work-items/export-invalid-program.json) | implementation | 1 | todo | — | [#63](https://github.com/CaliLuke/go-guitar-pro/issues/63) |
| [Preserve authored opening sound-event preroll](work-items/export-negative-sound-position.json) | implementation | 1 | blocked | [#41](https://github.com/CaliLuke/go-guitar-pro/issues/41) | [#64](https://github.com/CaliLuke/go-guitar-pro/issues/64) |
| [Preserve fermata position, type and length](work-items/fermata.json) | implementation | 1 | todo | — | [#47](https://github.com/CaliLuke/go-guitar-pro/issues/47) |
| [Preserve free-time bars without inventing a meter](work-items/free-time.json) | implementation | 1 | todo | — | [#48](https://github.com/CaliLuke/go-guitar-pro/issues/48) |
| [Preserve lowercase GPIF minor key modes](work-items/key.json) | implementation | 1 | todo | — | [#49](https://github.com/CaliLuke/go-guitar-pro/issues/49) |
| [Preserve authored legato and slur endpoints](work-items/legato-slurs.json) | implementation | 1 | todo | — | [#50](https://github.com/CaliLuke/go-guitar-pro/issues/50) |
| [Preserve MIDI bank selection and bank changes](work-items/midi-bank.json) | implementation | 1 | todo | — | [#51](https://github.com/CaliLuke/go-guitar-pro/issues/51) |
| [Require executable evidence for supported capability claims](work-items/oracle-breadth.json) | implementation | 1 | todo | — | [#67](https://github.com/CaliLuke/go-guitar-pro/issues/67) |
| [Emit GP8 archives readable by the pinned AlphaTab inflater](work-items/oracle-inflater.json) | implementation | 1 | todo | — | [#68](https://github.com/CaliLuke/go-guitar-pro/issues/68) |
| [Preserve pan automation independently from static balance](work-items/pan-automation.json) | implementation | 1 | blocked | [#41](https://github.com/CaliLuke/go-guitar-pro/issues/41) | [#52](https://github.com/CaliLuke/go-guitar-pro/issues/52) |
| [Export valid percussion notes currently rejected by GP8](work-items/percussion.json) | implementation | 1 | todo | — | [#53](https://github.com/CaliLuke/go-guitar-pro/issues/53) |
| [Prove sound identity independently of MIDI program](work-items/sound-automation.json) | implementation | 1 | blocked | [#51](https://github.com/CaliLuke/go-guitar-pro/issues/51), [#41](https://github.com/CaliLuke/go-guitar-pro/issues/41) | [#69](https://github.com/CaliLuke/go-guitar-pro/issues/69) |
| [Preserve sustain pedal down, hold and release markers](work-items/sustain-pedal.json) | implementation | 1 | todo | — | [#54](https://github.com/CaliLuke/go-guitar-pro/issues/54) |
| [Export authored backing-track synchronization points](work-items/sync-points.json) | implementation | 1 | blocked | [#42](https://github.com/CaliLuke/go-guitar-pro/issues/42) | [#55](https://github.com/CaliLuke/go-guitar-pro/issues/55) |
| [Apply authored HideTempo to the opening GP8 tempo event](work-items/tempo.json) | implementation | 1 | blocked | [#41](https://github.com/CaliLuke/go-guitar-pro/issues/41) | [#70](https://github.com/CaliLuke/go-guitar-pro/issues/70) |
| [Preserve sounding and display transposition separately](work-items/transposition.json) | implementation | 1 | todo | — | [#56](https://github.com/CaliLuke/go-guitar-pro/issues/56) |
| [Export tremolo picking and preserve supported stroke variants](work-items/tremolo.json) | implementation | 1 | todo | — | [#57](https://github.com/CaliLuke/go-guitar-pro/issues/57) |
| [Preserve tuning labels independently from pitches](work-items/tuning.json) | implementation | 1 | todo | — | [#58](https://github.com/CaliLuke/go-guitar-pro/issues/58) |
| [Classify authored GP constructs with importer-backed ownership](work-items/upstream-discovery.json) | implementation | 1 | todo | — | [#71](https://github.com/CaliLuke/go-guitar-pro/issues/71) |
| [Export authored channel-strip volume automation](work-items/volume-automation.json) | implementation | 1 | todo | — | [#59](https://github.com/CaliLuke/go-guitar-pro/issues/59) |
| [Apply exact authored offset preservation to whammy curves](work-items/whammy.json) | implementation | 1 | blocked | [#60](https://github.com/CaliLuke/go-guitar-pro/issues/60), [#62](https://github.com/CaliLuke/go-guitar-pro/issues/62) | [#72](https://github.com/CaliLuke/go-guitar-pro/issues/72) |
| [Establish target limits for per-note dynamics and velocities](work-items/dynamics.json) | investigation | 1 | todo | — | [#61](https://github.com/CaliLuke/go-guitar-pro/issues/61) |
| [Bound remaining grace chord and transition semantics](work-items/grace.json) | investigation | 1 | todo | — | [#65](https://github.com/CaliLuke/go-guitar-pro/issues/65) |
| [Classify conflicting MIDI port and playback-state limits](work-items/midi-routing.json) | investigation | 1 | todo | — | [#66](https://github.com/CaliLuke/go-guitar-pro/issues/66) |
| [Preserve authored pitch spelling and accidental choices](work-items/accidentals.json) | implementation | 2 | blocked | [#56](https://github.com/CaliLuke/go-guitar-pro/issues/56) | [#73](https://github.com/CaliLuke/go-guitar-pro/issues/73) |
| [Preserve beat barre fret and shape](work-items/barre.json) | implementation | 2 | todo | — | [#74](https://github.com/CaliLuke/go-guitar-pro/issues/74) |
| [Preserve authored beam groups and stem overrides](work-items/beaming.json) | implementation | 2 | todo | — | [#75](https://github.com/CaliLuke/go-guitar-pro/issues/75) |
| [Preserve lyrics authored directly on beats](work-items/beat-lyrics.json) | implementation | 2 | todo | — | [#76](https://github.com/CaliLuke/go-guitar-pro/issues/76) |
| [Preserve and export beat vibrato strength](work-items/beat-vibrato.json) | implementation | 2 | todo | — | [#77](https://github.com/CaliLuke/go-guitar-pro/issues/77) |
| [Preserve authored brush and arpeggio timing](work-items/brush.json) | implementation | 2 | todo | — | [#78](https://github.com/CaliLuke/go-guitar-pro/issues/78) |
| [Preserve representable chord diagram barres and fingerings](work-items/chord-diagram.json) | implementation | 2 | todo | — | [#79](https://github.com/CaliLuke/go-guitar-pro/issues/79) |
| [Preserve dead-slapped beats without reducing them to rests](work-items/dead-slap.json) | implementation | 2 | todo | — | [#80](https://github.com/CaliLuke/go-guitar-pro/issues/80) |
| [Preserve fade-out and volume-swell beat effects](work-items/fade-other.json) | implementation | 2 | todo | — | [#81](https://github.com/CaliLuke/go-guitar-pro/issues/81) |
| [Export existing left and right hand fingerings](work-items/fingering.json) | implementation | 2 | todo | — | [#82](https://github.com/CaliLuke/go-guitar-pro/issues/82) |
| [Preserve thumb and finger golpe marks](work-items/golpe.json) | implementation | 2 | todo | — | [#83](https://github.com/CaliLuke/go-guitar-pro/issues/83) |
| [Preserve explicit hammer and pull-off destination markers](work-items/hammer.json) | implementation | 2 | todo | — | [#84](https://github.com/CaliLuke/go-guitar-pro/issues/84) |
| [Export assigned legacy score lyrics as GPIF track lyrics](work-items/lyrics.json) | implementation | 2 | todo | — | [#98](https://github.com/CaliLuke/go-guitar-pro/issues/98) |
| [Project supported legacy mix-table events into GP8 automation](work-items/mix-table.json) | implementation | 2 | blocked | [#59](https://github.com/CaliLuke/go-guitar-pro/issues/59), [#52](https://github.com/CaliLuke/go-guitar-pro/issues/52), [#51](https://github.com/CaliLuke/go-guitar-pro/issues/51), [#41](https://github.com/CaliLuke/go-guitar-pro/issues/41) | [#85](https://github.com/CaliLuke/go-guitar-pro/issues/85) |
| [Read and preserve per-staff part notation settings](work-items/notation-visibility.json) | implementation | 2 | todo | — | [#86](https://github.com/CaliLuke/go-guitar-pro/issues/86) |
| [Preserve turns and mordents through GPIF and GP8](work-items/ornaments.json) | implementation | 2 | todo | — | [#87](https://github.com/CaliLuke/go-guitar-pro/issues/87) |
| [Export existing pick stroke directions](work-items/pick-stroke.json) | implementation | 2 | todo | — | [#88](https://github.com/CaliLuke/go-guitar-pro/issues/88) |
| [Preserve named rasgueado finger patterns](work-items/rasgueado.json) | implementation | 2 | todo | — | [#89](https://github.com/CaliLuke/go-guitar-pro/issues/89) |
| [Keep section letters separate from section text](work-items/sections.json) | implementation | 2 | todo | — | [#90](https://github.com/CaliLuke/go-guitar-pro/issues/90) |
| [Preserve authored track short names](work-items/short-name.json) | implementation | 2 | todo | — | [#91](https://github.com/CaliLuke/go-guitar-pro/issues/91) |
| [Preserve slashed beats and slash staff notation separately](work-items/slash.json) | implementation | 2 | blocked | [#86](https://github.com/CaliLuke/go-guitar-pro/issues/86) | [#92](https://github.com/CaliLuke/go-guitar-pro/issues/92) |
| [Preserve independent tap, slap and pop beat techniques](work-items/tap-slap-pop.json) | implementation | 2 | todo | — | [#93](https://github.com/CaliLuke/go-guitar-pro/issues/93) |
| [Preserve wah pedal state through GPIF and GP8](work-items/wah.json) | implementation | 2 | todo | — | [#94](https://github.com/CaliLuke/go-guitar-pro/issues/94) |
| [Resolve final-bar double-line consumer differences](work-items/double-bar.json) | investigation | 2 | todo | — | [#95](https://github.com/CaliLuke/go-guitar-pro/issues/95) |
| [Resolve harmonic spelling and octave target limits](work-items/harmonics.json) | investigation | 2 | todo | — | [#96](https://github.com/CaliLuke/go-guitar-pro/issues/96) |
| [Resolve GP8 support for legacy instrument flags](work-items/legacy-instrument.json) | investigation | 2 | todo | — | [#97](https://github.com/CaliLuke/go-guitar-pro/issues/97) |
| [Classify GP8 limits for binary-only score metadata](work-items/metadata.json) | investigation | 2 | todo | — | [#99](https://github.com/CaliLuke/go-guitar-pro/issues/99) |
| [Establish whether GP8 can preserve explicit empty beats](work-items/rests.json) | investigation | 2 | todo | — | [#100](https://github.com/CaliLuke/go-guitar-pro/issues/100) |
| [Bound RSE data preservation against real GP8 consumers](work-items/rse.json) | investigation | 2 | todo | — | [#101](https://github.com/CaliLuke/go-guitar-pro/issues/101) |
| [Determine GP8 support for authored note-duration percentages](work-items/sound-duration.json) | investigation | 2 | todo | — | [#102](https://github.com/CaliLuke/go-guitar-pro/issues/102) |
| [Determine supported GP8 trill-speed encoding](work-items/trill.json) | investigation | 2 | todo | — | [#103](https://github.com/CaliLuke/go-guitar-pro/issues/103) |
| [Preserve global extended barlines and bar-number policy](work-items/barlines.json) | implementation | 3 | todo | — | [#109](https://github.com/CaliLuke/go-guitar-pro/issues/109) |
| [Preserve independent chord display flags](work-items/chord-display.json) | implementation | 3 | todo | — | [#104](https://github.com/CaliLuke/go-guitar-pro/issues/104) |
| [Preserve authored system layouts and bar display scales](work-items/layout.json) | implementation | 3 | todo | — | [#112](https://github.com/CaliLuke/go-guitar-pro/issues/112) |
| [Preserve score and track multirest preferences](work-items/multi-rest.json) | implementation | 3 | todo | — | [#105](https://github.com/CaliLuke/go-guitar-pro/issues/105) |
| [Preserve numbered notation per staff](work-items/numbered.json) | implementation | 3 | blocked | [#86](https://github.com/CaliLuke/go-guitar-pro/issues/86) | [#106](https://github.com/CaliLuke/go-guitar-pro/issues/106) |
| [Preserve supported header and footer templates and visibility](work-items/page-setup.json) | implementation | 3 | blocked | [#109](https://github.com/CaliLuke/go-guitar-pro/issues/109) | [#114](https://github.com/CaliLuke/go-guitar-pro/issues/114) |
| [Preserve explicit string-number display requests](work-items/show-string.json) | implementation | 3 | todo | — | [#107](https://github.com/CaliLuke/go-guitar-pro/issues/107) |
| [Preserve supported score bracket, track-name and display policies](work-items/stylesheet.json) | implementation | 3 | blocked | [#109](https://github.com/CaliLuke/go-guitar-pro/issues/109) | [#117](https://github.com/CaliLuke/go-guitar-pro/issues/117) |
| [Preserve authored timer marks and visibility](work-items/timer.json) | implementation | 3 | todo | — | [#108](https://github.com/CaliLuke/go-guitar-pro/issues/108) |
| [Determine Guitar Pro support for common and cut time](work-items/common-time.json) | investigation | 3 | todo | — | [#110](https://github.com/CaliLuke/go-guitar-pro/issues/110) |
| [Determine whether GP input authors independent display duration](work-items/display-duration-override.json) | investigation | 3 | todo | — | [#111](https://github.com/CaliLuke/go-guitar-pro/issues/111) |
| [Determine GP source support for note visibility and noteheads](work-items/note-display.json) | investigation | 3 | todo | — | [#113](https://github.com/CaliLuke/go-guitar-pro/issues/113) |
| [Decide the supported scope for partial capos](work-items/partial-capo.json) | investigation | 3 | blocked | [#43](https://github.com/CaliLuke/go-guitar-pro/issues/43) | [#115](https://github.com/CaliLuke/go-guitar-pro/issues/115) |
| [Decide scope and oracle for password-protected Guitar Pro files](work-items/password.json) | investigation | 3 | todo | — | [#116](https://github.com/CaliLuke/go-guitar-pro/issues/116) |
