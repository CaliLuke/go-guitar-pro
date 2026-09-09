# Guitar Pro semantic support

Generated from [`conformance/feature-ledger.json`](../conformance/feature-ledger.json). Do not edit these tables by hand.

AlphaTab oracle: `@coderline/alphatab@1.8.4`, source `022a45c8e42370f9e12e68949d11eada370da83d`.

| Feature | Formats | Status | Issues | Reason |
| --- | --- | --- | --- | --- |
| `score-core` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12), [#13](https://github.com/CaliLuke/go-guitar-pro/issues/13) | The full corpus parses, while richer metadata and loss reporting remain tracked by issues 12 and 13. |
| `rhythm` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12), [#15](https://github.com/CaliLuke/go-guitar-pro/issues/15) | Selected duration, dot, tuplet, time-signature, and rest cases are compared; exact score time and the remaining authored rhythm model are tracked by issues 12 and 15. |
| `timing` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#14](https://github.com/CaliLuke/go-guitar-pro/issues/14), [#15](https://github.com/CaliLuke/go-guitar-pro/issues/15) | Selected absolute display starts agree after zero-origin normalization; exact score time and shared finalized-graph validation are tracked by issues 14 and 15. |
| `staff-ownership` | gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | The grand-staff contract compares exact track, staff, bar, voice, beat, and note hierarchy plus authored contents; broader model ownership and legacy projection are tracked by issue 12. |
| `grace-relationships` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | Selected authored ownership and source-fret cases agree after excluding display duration and playback pitch; complete cross-format grace coverage is tracked by issue 12. |
| `note-and-beat-semantics` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | The canonical contract inventories these fields; a lossless semantic model is tracked by issue 12. |
| `tremolo-picking` | gp3, gp4, gp5, gp6, gp7, gp8 | supported | none | Binary and GPIF tremolo-picking subdivisions are retained on notes; all six GPIF fixture beats agree with AlphaTab. |
| `harmonics` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | GPIF HFret values and common harmonic kinds are compared on import and GP8 export; distinctions that the legacy model collapses are tracked by issue 12. |
| `hairpins` | gp6, gp7, gp8 | supported | none | All eight GPIF hairpins and GP8 Decrescendo output agree with AlphaTab; legacy Diminuendo input remains accepted. |
| `tempo-automations` | gp3, gp4, gp5, gp6, gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | GPIF references 1–5 and absent or invalid defaults agree after BPM normalization; authored units, text, visibility, and interpolation remain part of issue 12. |
| `percussion-articulations` | gp7, gp8 | partial | [#12](https://github.com/CaliLuke/go-guitar-pro/issues/12) | GP7–8 GPIF articulation identity, per-track definitions, notation metadata, input values, and output MIDI are compared; cross-format semantic integration is tracked by issue 12. |

## Parse diagnostics

The diagnostic disposition is separate from semantic feature support. Strict parsing rejects only dispositions marked Yes.

| Disposition | Strict rejection | Meaning |
| --- | --- | --- |
| `invalid-data` | yes | The source contains a malformed value, duplicate object ID, or broken object reference. |
| `unknown-syntax` | yes | The source contains syntax that the GPIF audit does not recognize. |
| `unsupported-feature` | yes | The parser recognizes the source content but cannot represent it in Song. |
| `lossy-projection` | yes | The parser maps the source content to a less precise Song value. |
| `deliberate-default` | no | The parser applies a documented default when the source omits a value. |
| `deliberate-ignore` | no | The parser intentionally excludes source metadata that is outside the Song contract. |

Each diagnostic receipt names one source construct. Its feature value uses an ID from the semantic support table.

| Source construct | Feature | Disposition | Reason |
| --- | --- | --- | --- |
| `GPIF.Chord.Diagram.Fingering` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Chord.Diagram.Property.ShowDiagram` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Chord.Diagram.Property.ShowFingering` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Chord.Diagram.Property.ShowName` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Chord.Diagram.Property.Unknown` | `note-and-beat-semantics` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.Staff.Property.CapoFret` | `staff-ownership` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Staff.Property.Tuning.Label` | `staff-ownership` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Staff.Property.Tuning.MissingPitches` | `staff-ownership` | `invalid-data` | The recognized source construct is missing its required payload. |
| `GPIF.Staff.Property.Unknown` | `staff-ownership` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.Track.Property.CapoFret` | `staff-ownership` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Track.Property.Tuning.Label` | `staff-ownership` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Track.Property.Tuning.MissingPitches` | `staff-ownership` | `invalid-data` | The recognized source construct is missing its required payload. |
| `GPIF.Track.Property.Unknown` | `staff-ownership` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownAttribute.ScoreCore` | `score-core` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownElement.ScoreCore` | `score-core` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `Binary.Note.TimeIndependentDuration` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `Binary.RSE.MasterMetadata` | `score-core` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `Binary.RSE.TrackMetadata` | `score-core` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Asset.DuplicateID` | `score-core` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Asset.EmptyID` | `score-core` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.BackingTrack.AssetId.Reference` | `score-core` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Bar.DuplicateID` | `staff-ownership` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Bar.EmptyID` | `staff-ownership` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Bar.Voices.Reference` | `note-and-beat-semantics` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Beat.Arpeggio.InvalidValue` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Chord.Reference` | `note-and-beat-semantics` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Beat.DuplicateID` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Beat.Dynamic.InvalidValue` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.EmptyID` | `note-and-beat-semantics` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Beat.Fadding.InvalidValue` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Fadding.Lossy` | `note-and-beat-semantics` | `lossy-projection` | Song retains a less precise value than this source construct. |
| `GPIF.Beat.GraceNotes.InvalidValue` | `grace-relationships` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Hairpin.InvalidValue` | `hairpins` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Notes.Reference` | `note-and-beat-semantics` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Beat.Ottavia.InvalidValue` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.BarreFret` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.BarreString` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.Brush.InvalidDirection` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.Brush.MissingDirection` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Beat.Property.PickStroke.InvalidDirection` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.PickStroke.MissingDirection` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Beat.Property.Popped.MissingEnable` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Beat.Property.PrimaryPickupTone` | `note-and-beat-semantics` | `deliberate-ignore` | The notation-focused Song model intentionally excludes this playback metadata. |
| `GPIF.Beat.Property.PrimaryPickupVolume` | `note-and-beat-semantics` | `deliberate-ignore` | The notation-focused Song model intentionally excludes this playback metadata. |
| `GPIF.Beat.Property.Rasgueado` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.Slapped.MissingEnable` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Beat.Property.Unknown` | `note-and-beat-semantics` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.Beat.Property.VibratoWTremBar` | `note-and-beat-semantics` | `lossy-projection` | Song retains a less precise value than this source construct. |
| `GPIF.Beat.Property.WhammyBar` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.WhammyBarDestinationOffset` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.WhammyBarDestinationValue` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.WhammyBarExtend` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.WhammyBarMiddleOffset1` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.WhammyBarMiddleOffset2` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.WhammyBarMiddleValue` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.WhammyBarOriginOffset` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Property.WhammyBarOriginValue` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Rhythm.Reference` | `rhythm` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Beat.Tremolo.InvalidValue` | `tremolo-picking` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Beat.Wah` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.ChordDefinition.DuplicateID` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.ChordDefinition.EmptyID` | `note-and-beat-semantics` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.MasterBar.Bars.Reference` | `staff-ownership` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.MasterBar.TripletFeel.InvalidValue` | `rhythm` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.MasterTrack.Tracks.Reference` | `staff-ownership` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Note.DuplicateID` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Note.EmptyID` | `note-and-beat-semantics` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Note.Property.BendDestinationOffset.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.BendDestinationValue.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.BendMiddleOffset1.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.BendMiddleOffset2.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.BendMiddleValue.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.BendOriginOffset.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.BendOriginValue.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.ConcertPitch` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Property.Element` | `percussion-articulations` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Property.Fret.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.HarmonicFret.MissingPayload` | `harmonics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.HarmonicType.Feedback` | `harmonics` | `lossy-projection` | Song retains a less precise value than this source construct. |
| `GPIF.Note.Property.HarmonicType.MissingPayload` | `harmonics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.HarmonicType.Unsupported` | `harmonics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Property.HopoDestination` | `note-and-beat-semantics` | `lossy-projection` | Song retains a less precise value than this source construct. |
| `GPIF.Note.Property.HopoOrigin` | `note-and-beat-semantics` | `lossy-projection` | Song retains a less precise value than this source construct. |
| `GPIF.Note.Property.LeftHandTapped` | `note-and-beat-semantics` | `lossy-projection` | Song retains a less precise value than this source construct. |
| `GPIF.Note.Property.Midi.MissingPayload` | `percussion-articulations` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.Octave` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Property.Slide.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.String.MissingPayload` | `note-and-beat-semantics` | `invalid-data` | The property must contain its required typed payload. |
| `GPIF.Note.Property.Tapped` | `note-and-beat-semantics` | `lossy-projection` | Song retains a less precise value than this source construct. |
| `GPIF.Note.Property.Tone` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Property.TransposedPitch` | `note-and-beat-semantics` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Property.Unknown` | `note-and-beat-semantics` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.Note.Property.Variation` | `percussion-articulations` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Note.Vibrato` | `note-and-beat-semantics` | `lossy-projection` | Song retains a less precise value than this source construct. |
| `GPIF.Rhythm.DuplicateID` | `rhythm` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Rhythm.EmptyID` | `rhythm` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Rhythm.NoteValue.InvalidValue` | `rhythm` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.Track.DuplicateID` | `staff-ownership` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Track.EmptyID` | `staff-ownership` | `invalid-data` | The source object must have a non-empty ID. |
| `GPIF.Track.Transpose` | `staff-ownership` | `unsupported-feature` | Song has no lossless destination for this recognized source construct. |
| `GPIF.UnknownAttribute.NoteAndBeat` | `note-and-beat-semantics` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownAttribute.Rhythm` | `rhythm` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownAttribute.StaffOwnership` | `staff-ownership` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownElement.NoteAndBeat` | `note-and-beat-semantics` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownElement.Rhythm` | `rhythm` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.UnknownElement.StaffOwnership` | `staff-ownership` | `unknown-syntax` | The GPIF audit does not recognize this source construct. |
| `GPIF.Voice.Beats.Reference` | `note-and-beat-semantics` | `invalid-data` | The source reference must resolve to an object of the requested type. |
| `GPIF.Voice.DuplicateID` | `note-and-beat-semantics` | `invalid-data` | The source object ID must be unique within its collection. |
| `GPIF.Voice.EmptyID` | `note-and-beat-semantics` | `invalid-data` | The source object must have a non-empty ID. |
