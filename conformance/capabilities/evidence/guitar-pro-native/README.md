# Native Guitar Pro validation

Guitar Pro 8.1.5 build 31 opened each input through its normal file dialog.
Save As produced each native GPIF artifact. The parent JSON receipt records hashes and exact facts.

`baseline-export.gp` comes from the public Go API at the receipt's baseline commit.
`baseline-native.gp` is the file after Guitar Pro opened and saved it.
The export contains 12 note occurrences. The native save contains zero.

`original-native.gp` comes from opening the original GP3 fixture directly in Guitar Pro.
It retains all 12 note occurrences. Its sound dialog places the flute event at tick 481.
The display uses one-based positions and 480 ticks per quarter note.

`current-sound-ui.txt` records the Go export after the sound-text fix.
Its flute event appears at tick 121. The native sound position is measured in quarter notes.
Tempo and channel-strip positions use a different scale. The position-only probe changes only Sound.Position and restores tick 481.
Its Linear=true value stays unchanged. The earlier position probe also set Linear=false.

The pitch probe adds valid ConcertPitch and TransposedPitch records to each note.
Its native save retains all 12 note occurrences. Removing either record again loses all notes.
The probe does not define the complete default-spelling or pitch-context contract.
Issue #118 tracks that repair. The probe is not a production fix.

The absent-accidental probe retains notes but creates conflicting pitch values.
Guitar Pro saves C natural at octave 5 beside MIDI 61. Omitting accidentals is not a valid default-spelling solution.

The native original maps legacy volume 12 to channel-strip value 0.56.
The Go export writes 0.75. Native zero-duration changes use Linear=false; Go writes Linear=true.
Issue #85 remains open for these native discrepancies and sound-position units.

`sound-text-fixed-native.gp` preserves all three sound names after the CDATA fix.
`labeled-sound-native.gp` also preserves the label `Flute <&> Ω`.
These files still lose notes. The sound-text regression proves text preservation only.

`notation-ui.txt` shows one notation selection for the grand-staff track and an independently hidden second track.
It does not prove different notation types within one track or native note preservation.

Guitar Pro's MIDI export omitted program changes even from the native original.
Therefore, the MIDI file does not validate sound-event timing. The sound dialog provides that evidence.

## Reproduce an export

Run from the repository root:

```sh
mkdir -p /tmp/native-pitch-check
go run conformance/capabilities/probe.go <<'JSON'
{"Paths":["testdata/gp3/mix-table-events.gp3"],"Output":"/tmp/native-pitch-check"}
JSON
```

Open `/tmp/native-pitch-check/0.gp` in Guitar Pro. Save a separate copy.
Extract `Content/score.gpif` from both files. Follow Bars, Voices, Beats, and Notes references to count occurrences.
Do not compare definition counts: Guitar Pro deduplicates note definitions during save.
