# Capability work backlog

Generated from `work-items/*.json`. Regenerate with `manage.py refresh`.

77 work items: 46 implementation tasks and 31 investigations.
Investigation closure records a decision; it does not certify capability support.

Use `manage.py ready` for work without unfinished dependencies. Check shared files before dispatching concurrent work.

| Work item | Kind | P | State | Prerequisites | Ticket |
| --- | --- | ---: | --- | --- | --- |
| [Preserve authored tempo and sound automation details](work-items/automation-detail.json) | implementation | 1 | todo | — | `automation-detail` |
| [Export embedded audio backing-track assets](work-items/backing-track.json) | implementation | 1 | todo | — | `backing-track` |
| [Preserve independent capo values on multiple staves](work-items/capo.json) | implementation | 1 | todo | — | `capo` |
| [Preserve bar-level clef octave shifts](work-items/clef-octave.json) | implementation | 1 | todo | — | `clef-octave` |
| [Preserve navigation targets and jumps on each measure](work-items/directions.json) | implementation | 1 | todo | — | `directions` |
| [Correct stale semantic documentation and diagnostic labels](work-items/documentation-contract-drift.json) | implementation | 1 | todo | — | `documentation-contract-drift` |
| [Preserve fermata position, type and length](work-items/fermata.json) | implementation | 1 | todo | — | `fermata` |
| [Preserve free-time bars without inventing a meter](work-items/free-time.json) | implementation | 1 | todo | — | `free-time` |
| [Preserve lowercase GPIF minor key modes](work-items/key.json) | implementation | 1 | todo | — | `key` |
| [Preserve authored legato and slur endpoints](work-items/legato-slurs.json) | implementation | 1 | todo | — | `legato-slurs` |
| [Preserve MIDI bank selection and bank changes](work-items/midi-bank.json) | implementation | 1 | todo | — | `midi-bank` |
| [Preserve pan automation independently from static balance](work-items/pan-automation.json) | implementation | 1 | blocked | `automation-detail` | `pan-automation` |
| [Export valid percussion notes currently rejected by GP8](work-items/percussion.json) | implementation | 1 | todo | — | `percussion` |
| [Preserve sustain pedal down, hold and release markers](work-items/sustain-pedal.json) | implementation | 1 | todo | — | `sustain-pedal` |
| [Export authored backing-track synchronization points](work-items/sync-points.json) | implementation | 1 | blocked | `backing-track` | `sync-points` |
| [Preserve sounding and display transposition separately](work-items/transposition.json) | implementation | 1 | todo | — | `transposition` |
| [Export tremolo picking and preserve supported stroke variants](work-items/tremolo.json) | implementation | 1 | todo | — | `tremolo` |
| [Preserve tuning labels independently from pitches](work-items/tuning.json) | implementation | 1 | todo | — | `tuning` |
| [Export authored channel-strip volume automation](work-items/volume-automation.json) | implementation | 1 | todo | — | `volume-automation` |
| [Define exact bend precision and style boundaries](work-items/bends.json) | investigation | 1 | todo | — | `bends` |
| [Establish target limits for per-note dynamics and velocities](work-items/dynamics.json) | investigation | 1 | todo | — | `dynamics` |
| [Investigate parsed bend point order rejected by export](work-items/export-curve-order.json) | investigation | 1 | todo | — | `export-curve-order` |
| [Classify parsed invalid MIDI programs that block export](work-items/export-invalid-program.json) | investigation | 1 | todo | — | `export-invalid-program` |
| [Resolve negative sound automation positions in grace fixtures](work-items/export-negative-sound-position.json) | investigation | 1 | todo | — | `export-negative-sound-position` |
| [Bound remaining grace chord and transition semantics](work-items/grace.json) | investigation | 1 | todo | — | `grace` |
| [Classify conflicting MIDI port and playback-state limits](work-items/midi-routing.json) | investigation | 1 | todo | — | `midi-routing` |
| [Make broad capability comparisons reliable acceptance evidence](work-items/oracle-breadth.json) | investigation | 1 | todo | — | `oracle-breadth` |
| [Isolate pinned AlphaTab inflater failures in transposition fixtures](work-items/oracle-inflater.json) | investigation | 1 | todo | — | `oracle-inflater` |
| [Verify sound-definition selection and ordered event fidelity](work-items/sound-automation.json) | investigation | 1 | blocked | `midi-bank`, `automation-detail` | `sound-automation` |
| [Preserve authored tempo visibility where GP8 supports it](work-items/tempo.json) | investigation | 1 | todo | — | `tempo` |
| [Reconcile expanded source inventory with executable obligations](work-items/upstream-discovery.json) | investigation | 1 | todo | — | `upstream-discovery` |
| [Classify remaining whammy curve and style losses](work-items/whammy.json) | investigation | 1 | blocked | `bends` | `whammy` |
| [Preserve authored pitch spelling and accidental choices](work-items/accidentals.json) | implementation | 2 | blocked | `transposition` | `accidentals` |
| [Preserve beat barre fret and shape](work-items/barre.json) | implementation | 2 | todo | — | `barre` |
| [Preserve authored beam groups and stem overrides](work-items/beaming.json) | implementation | 2 | todo | — | `beaming` |
| [Preserve lyrics authored directly on beats](work-items/beat-lyrics.json) | implementation | 2 | todo | — | `beat-lyrics` |
| [Preserve and export beat vibrato strength](work-items/beat-vibrato.json) | implementation | 2 | todo | — | `beat-vibrato` |
| [Preserve authored brush and arpeggio timing](work-items/brush.json) | implementation | 2 | todo | — | `brush` |
| [Preserve representable chord diagram barres and fingerings](work-items/chord-diagram.json) | implementation | 2 | todo | — | `chord-diagram` |
| [Preserve dead-slapped beats without reducing them to rests](work-items/dead-slap.json) | implementation | 2 | todo | — | `dead-slap` |
| [Preserve fade-out and volume-swell beat effects](work-items/fade-other.json) | implementation | 2 | todo | — | `fade-other` |
| [Export existing left and right hand fingerings](work-items/fingering.json) | implementation | 2 | todo | — | `fingering` |
| [Preserve thumb and finger golpe marks](work-items/golpe.json) | implementation | 2 | todo | — | `golpe` |
| [Preserve explicit hammer and pull-off destination markers](work-items/hammer.json) | implementation | 2 | todo | — | `hammer` |
| [Project supported legacy mix-table events into GP8 automation](work-items/mix-table.json) | implementation | 2 | blocked | `volume-automation`, `pan-automation`, `midi-bank`, `automation-detail` | `mix-table` |
| [Read and preserve per-staff part notation settings](work-items/notation-visibility.json) | implementation | 2 | todo | — | `notation-visibility` |
| [Preserve turns and mordents through GPIF and GP8](work-items/ornaments.json) | implementation | 2 | todo | — | `ornaments` |
| [Export existing pick stroke directions](work-items/pick-stroke.json) | implementation | 2 | todo | — | `pick-stroke` |
| [Preserve named rasgueado finger patterns](work-items/rasgueado.json) | implementation | 2 | todo | — | `rasgueado` |
| [Keep section letters separate from section text](work-items/sections.json) | implementation | 2 | todo | — | `sections` |
| [Preserve authored track short names](work-items/short-name.json) | implementation | 2 | todo | — | `short-name` |
| [Preserve slashed beats and slash staff notation separately](work-items/slash.json) | implementation | 2 | blocked | `notation-visibility` | `slash` |
| [Preserve independent tap, slap and pop beat techniques](work-items/tap-slap-pop.json) | implementation | 2 | todo | — | `tap-slap-pop` |
| [Preserve wah pedal state through GPIF and GP8](work-items/wah.json) | implementation | 2 | todo | — | `wah` |
| [Resolve final-bar double-line consumer differences](work-items/double-bar.json) | investigation | 2 | todo | — | `double-bar` |
| [Resolve harmonic spelling and octave target limits](work-items/harmonics.json) | investigation | 2 | todo | — | `harmonics` |
| [Resolve GP8 support for legacy instrument flags](work-items/legacy-instrument.json) | investigation | 2 | todo | — | `legacy-instrument` |
| [Resolve GP8 export of legacy score-level lyrics](work-items/lyrics.json) | investigation | 2 | todo | — | `lyrics` |
| [Classify GP8 limits for binary-only score metadata](work-items/metadata.json) | investigation | 2 | todo | — | `metadata` |
| [Establish whether GP8 can preserve explicit empty beats](work-items/rests.json) | investigation | 2 | todo | — | `rests` |
| [Bound RSE data preservation against real GP8 consumers](work-items/rse.json) | investigation | 2 | todo | — | `rse` |
| [Determine GP8 support for authored note-duration percentages](work-items/sound-duration.json) | investigation | 2 | todo | — | `sound-duration` |
| [Determine supported GP8 trill-speed encoding](work-items/trill.json) | investigation | 2 | todo | — | `trill` |
| [Preserve independent chord display flags](work-items/chord-display.json) | implementation | 3 | todo | — | `chord-display` |
| [Preserve score and track multirest preferences](work-items/multi-rest.json) | implementation | 3 | todo | — | `multi-rest` |
| [Preserve numbered notation per staff](work-items/numbered.json) | implementation | 3 | blocked | `notation-visibility` | `numbered` |
| [Preserve explicit string-number display requests](work-items/show-string.json) | implementation | 3 | todo | — | `show-string` |
| [Preserve authored timer marks and visibility](work-items/timer.json) | implementation | 3 | todo | — | `timer` |
| [Determine GPIF representations for extended barlines and numbering](work-items/barlines.json) | investigation | 3 | blocked | `double-bar` | `barlines` |
| [Determine Guitar Pro support for common and cut time](work-items/common-time.json) | investigation | 3 | todo | — | `common-time` |
| [Determine whether GP input authors independent display duration](work-items/display-duration-override.json) | investigation | 3 | todo | — | `display-duration-override` |
| [Bound system layout, scale and line-break preservation](work-items/layout.json) | investigation | 3 | todo | — | `layout` |
| [Determine GP source support for note visibility and noteheads](work-items/note-display.json) | investigation | 3 | todo | — | `note-display` |
| [Resolve page and header/footer preservation scope](work-items/page-setup.json) | investigation | 3 | todo | — | `page-setup` |
| [Decide the supported scope for partial capos](work-items/partial-capo.json) | investigation | 3 | blocked | `capo` | `partial-capo` |
| [Decide scope and oracle for password-protected Guitar Pro files](work-items/password.json) | investigation | 3 | todo | — | `password` |
| [Split remaining authored stylesheet gaps into scoped contracts](work-items/stylesheet.json) | investigation | 3 | todo | — | `stylesheet` |
